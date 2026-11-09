// Package musicbrainz finds the album a song was released on, using the
// MusicBrainz database. YouTube often names a song's single as its album;
// MusicBrainz records what kind of release each one is.
package musicbrainz

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/Sergio-dot/urtube/internal/media"
)

const (
	defaultBaseURL = "https://musicbrainz.org/ws/2"
	// MusicBrainz asks clients to identify themselves and blocks those that
	// don't.
	userAgent = "urtube/1.0 ( https://github.com/Sergio-dot/urtube )"
	// minInterval spaces requests. MusicBrainz limits requests per IP
	// address, and other software on this machine may use it too: Jellyfin's
	// scans do.
	minInterval = 2 * time.Second
	// requestTimeout bounds one request.
	requestTimeout = 8 * time.Second
	// maxRetries is how many times a throttled request is sent again.
	maxRetries = 1
	// defaultRetryWait is how long to wait before sending a throttled request
	// again. A longer Retry-After isn't worth waiting for: lookups hold up the
	// download dialog or a download slot.
	defaultRetryWait = 5 * time.Second
	// defaultCooldown is how long all lookups stop after MusicBrainz keeps
	// throttling.
	defaultCooldown = 5 * time.Minute
	// maxCached bounds the answers remembered.
	maxCached = 2000
	// searchLimit is how many recordings to consider, the most MusicBrainz
	// returns at once. Popular songs have dozens, each listing the releases
	// it appears on, and a smaller limit can miss the original album.
	searchLimit = 100
	// maxResponseSize bounds the response read.
	maxResponseSize = 8 << 20
)

var (
	// ErrThrottled means MusicBrainz kept asking to slow down, so lookups
	// are paused.
	ErrThrottled = errors.New("MusicBrainz is throttling requests")
	// ErrBusy means the request's turn wouldn't come before its deadline.
	ErrBusy = errors.New("too many MusicBrainz lookups waiting")
)

// Client looks songs up in MusicBrainz. One Client paces all its requests to
// respect the rate limit, so share it. It implements media.AlbumFinder.
type Client struct {
	baseURL  string
	http     *http.Client
	interval time.Duration
	// retryWait and cooldown are defaultRetryWait and defaultCooldown, except
	// in tests.
	retryWait time.Duration
	cooldown  time.Duration

	mu sync.Mutex
	// next is the earliest time the next request may be sent.
	next time.Time
	// pausedUntil is when lookups may resume after repeated throttling.
	pausedUntil time.Time
	// cache remembers answers until restart, so that looking at the same
	// song twice sends nothing.
	cache map[string]media.Album
}

// New returns a Client for the public MusicBrainz server.
func New() *Client {
	return &Client{
		baseURL:   defaultBaseURL,
		http:      &http.Client{Timeout: requestTimeout},
		interval:  minInterval,
		retryWait: defaultRetryWait,
		cooldown:  defaultCooldown,
		cache:     make(map[string]media.Album),
	}
}

type searchResponse struct {
	Recordings []recording `json:"recordings"`
}

type recording struct {
	Title        string    `json:"title"`
	ArtistCredit []credit  `json:"artist-credit"`
	Releases     []release `json:"releases"`
}

type credit struct {
	Name       string `json:"name"`
	JoinPhrase string `json:"joinphrase"`
	Artist     struct {
		Name string `json:"name"`
	} `json:"artist"`
}

type release struct {
	Date         string       `json:"date"`
	Status       string       `json:"status"`
	ReleaseGroup releaseGroup `json:"release-group"`
}

type releaseGroup struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	PrimaryType    string   `json:"primary-type"`
	SecondaryTypes []string `json:"secondary-types"`
	// FirstReleaseDate is only set when looking up the release group itself.
	FirstReleaseDate string `json:"first-release-date"`
}

// FindAlbum returns the studio album tags' song first appeared on. If
// tags.Album is already one of the song's studio albums, it returns that. It
// returns a zero Album when the song matches no studio album, such as a
// song only released as a single.
func (c *Client) FindAlbum(ctx context.Context, tags media.Tags) (media.Album, error) {
	key := normalize(tags.Artist) + "\x00" + normalize(tags.Title) + "\x00" + normalize(tags.Album)
	if album, ok := c.cached(key); ok {
		return album, nil
	}

	target := c.baseURL + "/recording?" + url.Values{
		"query": {fmt.Sprintf("recording:%s AND artist:%s AND status:official", phrase(tags.Title), phrase(tags.Artist))},
		"fmt":   {"json"},
		"limit": {strconv.Itoa(searchLimit)},
	}.Encode()

	var found searchResponse
	if err := c.request(ctx, target, &found); err != nil {
		return media.Album{}, err
	}
	album, groupID := pickAlbum(found.Recordings, tags)
	if groupID == "" {
		c.remember(key, album)
		return album, nil
	}

	// The search can't settle the year: popular albums have more releases
	// than it returns, so their oldest may be missing. The album's own record
	// has its first release date; without it the album is still right.
	var group releaseGroup
	target = c.baseURL + "/release-group/" + url.PathEscape(groupID) + "?fmt=json"
	if err := c.request(ctx, target, &group); err == nil {
		album.Year = year(group.FirstReleaseDate)
		c.remember(key, album)
	}
	return album, nil
}

func (c *Client) cached(key string) (media.Album, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	album, ok := c.cache[key]
	return album, ok
}

func (c *Client) remember(key string, album media.Album) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.cache) >= maxCached {
		clear(c.cache)
	}
	c.cache[key] = album
}

// request fetches target into out, waiting its turn under the rate limit and
// sending the request again if MusicBrainz throttles it.
func (c *Client) request(ctx context.Context, target string, out any) error {
	for attempt := 0; ; attempt++ {
		if err := c.wait(ctx); err != nil {
			return err
		}
		retryAfter, err := c.get(ctx, target, out)
		if retryAfter == 0 {
			return err
		}
		if attempt == maxRetries || retryAfter > c.retryWait {
			c.pause(retryAfter)
			return ErrThrottled
		}
		// Holds back every lookup, not just this one: the limit is per IP.
		c.backOff(c.retryWait)
	}
}

// phrase quotes s as a phrase in MusicBrainz's Lucene query syntax.
func phrase(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

// wait blocks until this request may be sent. Slots are handed out in order
// of arrival, however many lookups are waiting. A lookup whose turn would
// come after its deadline gives up at once rather than using up a slot, and
// nothing is sent while lookups are paused.
func (c *Client) wait(ctx context.Context) error {
	c.mu.Lock()
	now := time.Now()
	if now.Before(c.pausedUntil) {
		c.mu.Unlock()
		return ErrThrottled
	}
	at := c.next
	if at.Before(now) {
		at = now
	}
	if deadline, ok := ctx.Deadline(); ok && at.After(deadline) {
		c.mu.Unlock()
		return ErrBusy
	}
	c.next = at.Add(c.interval)
	c.mu.Unlock()

	timer := time.NewTimer(time.Until(at))
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// backOff holds back all requests for d.
func (c *Client) backOff(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if until := time.Now().Add(d); until.After(c.next) {
		c.next = until
	}
}

// pause stops all lookups for the cooldown, or for d if that's longer.
func (c *Client) pause(d time.Duration) {
	d = max(d, c.cooldown)
	slog.Warn("MusicBrainz keeps asking to slow down, pausing album lookups", "for", d.String())

	c.mu.Lock()
	defer c.mu.Unlock()
	if until := time.Now().Add(d); until.After(c.pausedUntil) {
		c.pausedUntil = until
	}
}

// get sends one request and decodes the response into out. A positive
// duration means MusicBrainz throttled it and asked to wait that long.
func (c *Client) get(ctx context.Context, target string, out any) (time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return 0, fmt.Errorf("build MusicBrainz request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("query MusicBrainz: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusServiceUnavailable, http.StatusTooManyRequests:
		return c.retryAfter(resp), nil
	default:
		return 0, fmt.Errorf("MusicBrainz answered %s", resp.Status)
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseSize)).Decode(out); err != nil {
		return 0, fmt.Errorf("parse MusicBrainz response: %w", err)
	}
	return 0, nil
}

// retryAfter returns how long resp asks to wait, at least retryWait.
func (c *Client) retryAfter(resp *http.Response) time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(resp.Header.Get("Retry-After")))
	if err != nil {
		return c.retryWait
	}
	return max(time.Duration(seconds)*time.Second, c.retryWait)
}

// candidate is a studio album the song appears on.
type candidate struct {
	id       string
	title    string
	date     string // earliest in the search results
	releases int
}

// pickAlbum chooses among the studio albums of the recordings that really
// are tags' song: search results also include near matches. It returns the
// album and its release group ID, which is empty when the album isn't new
// to tags. Albums with no release date are ignored: they are mostly badly
// catalogued entries.
func pickAlbum(recordings []recording, tags media.Tags) (media.Album, string) {
	title := normalize(tags.Title)
	if title == "" {
		return media.Album{}, ""
	}
	artist := normalize(tags.Artist)
	current := normalize(tags.Album)

	albums := make(map[string]*candidate)
	for _, rec := range recordings {
		if normalize(rec.Title) != title || !creditedTo(rec, artist) {
			continue
		}
		for _, rel := range rec.Releases {
			group := rel.ReleaseGroup
			if rel.Status != "Official" || group.PrimaryType != "Album" || len(group.SecondaryTypes) > 0 {
				continue
			}
			c := albums[group.ID]
			if c == nil {
				c = &candidate{id: group.ID, title: group.Title}
				albums[group.ID] = c
			}
			c.releases++
			if rel.Date != "" && (c.date == "" || rel.Date < c.date) {
				c.date = rel.Date
			}
		}
	}

	var list []*candidate
	for _, c := range albums {
		if current != "" && normalize(c.title) == current {
			return media.Album{Title: tags.Album}, ""
		}
		if c.date != "" {
			list = append(list, c)
		}
	}
	if len(list) == 0 {
		return media.Album{}, ""
	}

	slices.SortFunc(list, func(a, b *candidate) int {
		return cmp.Or(
			cmp.Compare(a.date, b.date),
			cmp.Compare(b.releases, a.releases),
			cmp.Compare(a.title, b.title),
		)
	})
	return media.Album{Title: list[0].title}, list[0].id
}

// creditedTo reports whether artist, normalized, is the recording's main
// artist or its whole credit.
func creditedTo(rec recording, artist string) bool {
	if len(rec.ArtistCredit) == 0 || artist == "" {
		return false
	}
	first := rec.ArtistCredit[0]
	if normalize(first.Name) == artist || normalize(first.Artist.Name) == artist {
		return true
	}
	var whole strings.Builder
	for _, c := range rec.ArtistCredit {
		whole.WriteString(c.Name + c.JoinPhrase)
	}
	return normalize(whole.String()) == artist
}

// normalize reduces s to lowercase letters and digits, so differences in
// case, spacing and punctuation don't stop a match.
func normalize(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, s)
}

// year returns the year of a YYYY[-MM[-DD]] date, or 0.
func year(date string) int {
	if len(date) < 4 {
		return 0
	}
	y, err := strconv.Atoi(date[:4])
	if err != nil {
		return 0
	}
	return y
}
