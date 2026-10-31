package ytdlp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Sergio-dot/urtube/internal/media"
)

// musicLookups bounds how many songs are looked up at once for a music
// search: each lookup is a yt-dlp process taking a few seconds.
const musicLookups = 5

// musicFields is the subset of a song's info printed by a lookup.
const musicFields = "%(.{id,title,track,artist,artists,album,release_year,release_date,duration,thumbnail,channel,uploader})j"

// SearchMusic searches YouTube Music songs. The search itself only yields
// IDs, so each song is then looked up (in parallel) for its artist, album,
// duration and thumbnail. Songs whose lookup fails are left out; if every
// lookup fails, the first error is returned.
func (s *Searcher) SearchMusic(ctx context.Context, query string, limit int) ([]media.Video, error) {
	ids, err := s.musicIDs(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	songs := make([]*media.Video, len(ids))
	errs := make([]error, len(ids))
	sem := make(chan struct{}, musicLookups)
	var wg sync.WaitGroup
	for i, id := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			v, err := s.lookupSong(ctx, id)
			if err != nil {
				errs[i] = err
				if ctx.Err() == nil {
					slog.Warn("music search: skipping song", "id", id, "error", err)
				}
				return
			}
			songs[i] = &v
		}()
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	videos := make([]media.Video, 0, len(songs))
	for _, v := range songs {
		if v != nil {
			videos = append(videos, *v)
		}
	}
	if len(videos) == 0 && len(ids) > 0 {
		return nil, fmt.Errorf("%w: %s", media.ErrSearchFailed, errorMessage(errors.Join(errs...)))
	}
	return videos, nil
}

// musicIDs returns the IDs of the first limit songs matching query.
func (s *Searcher) musicIDs(ctx context.Context, query string, limit int) ([]string, error) {
	searchURL := "https://music.youtube.com/search?q=" + url.QueryEscape(query) + "#songs"
	result, err := newCommand(s.Executable).
		FlatPlaylist().
		SkipDownload().
		NoWarnings().
		PlaylistItems(fmt.Sprintf("1:%d", limit)).
		Print("id").
		Run(ctx, searchURL)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("%w: %s", media.ErrSearchFailed, errorMessage(err))
	}
	var ids []string
	for line := range strings.Lines(result.Stdout) {
		if id := strings.TrimSpace(line); id != "" {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// songFields extends infoFields with what a search result card shows.
type songFields struct {
	infoFields
	Duration  float64 `json:"duration"`
	Thumbnail string  `json:"thumbnail"`
}

// lookupSong fetches one song's details. It skips the player script, which
// is only needed to download, to keep searches quick.
func (s *Searcher) lookupSong(ctx context.Context, id string) (media.Video, error) {
	songURL := "https://music.youtube.com/watch?v=" + url.QueryEscape(id)
	result, err := newCommand(s.Executable).
		SkipDownload().
		NoWarnings().
		ExtractorArgs("youtube:player_skip=js").
		Print(musicFields).
		Run(ctx, songURL)
	if err != nil {
		return media.Video{}, err
	}
	return toSong(songURL, []byte(result.Stdout))
}

func toSong(songURL string, raw []byte) (media.Video, error) {
	info, err := parseInfo(raw)
	if err != nil {
		return media.Video{}, err
	}
	var f songFields
	if err := json.Unmarshal(bytes.TrimSpace(raw), &f); err != nil {
		return media.Video{}, fmt.Errorf("parse song: %w", err)
	}

	tags := media.GuessTags(info)
	v := media.Video{
		ID:       info.ID,
		Title:    tags.Title,
		Uploader: media.ArtistCredit(info.Artists),
		Duration: time.Duration(f.Duration * float64(time.Second)),
		URL:      songURL,
		Album:    tags.Album,
		Tags:     &tags,
	}
	if v.Uploader == "" {
		v.Uploader = tags.Artist
	}
	if f.Thumbnail != "" {
		v.Thumbnails = []media.Thumbnail{{URL: f.Thumbnail}}
	}
	return v, nil
}

// videoIDPrefix matches the "BSTsnWoslP4: " yt-dlp puts before a message.
var videoIDPrefix = regexp.MustCompile(`^[A-Za-z0-9_-]{11}: `)

// errorMessage returns the first sentence of yt-dlp's last "ERROR:" line in
// err, without the "[youtube] id:" prefix or links, for showing to users:
// "Sign in to confirm you’re not a bot." Errors without one are returned
// whole.
func errorMessage(err error) string {
	text := err.Error()
	i := strings.LastIndex(text, "ERROR: ")
	if i < 0 {
		return text
	}
	msg, _, _ := strings.Cut(text[i+len("ERROR: "):], "\n")
	if _, rest, ok := strings.Cut(msg, "]"); ok && strings.HasPrefix(msg, "[") {
		msg = videoIDPrefix.ReplaceAllString(strings.TrimSpace(rest), "")
	}
	if j := strings.Index(msg, ". "); j >= 0 {
		msg = msg[:j+1]
	}
	return strings.TrimSpace(msg)
}
