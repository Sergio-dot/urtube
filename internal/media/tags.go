package media

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// maxTagLength bounds each tag, in characters.
const maxTagLength = 200

// Tags describe a song: they are written into the file and decide where it
// is saved in the library.
type Tags struct {
	Artist string
	Title  string
	Album  string
	// Year is the release year, or 0 when unknown.
	Year int
}

// Validate checks tags supplied by a client: each must be at most
// maxTagLength characters without control characters, and the year must be
// plausible.
func (t Tags) Validate() error {
	for name, v := range map[string]string{"artist": t.Artist, "title": t.Title, "album": t.Album} {
		if utf8.RuneCountInString(v) > maxTagLength {
			return InvalidInput(fmt.Sprintf("tags.%s must be at most %d characters", name, maxTagLength))
		}
		if strings.ContainsFunc(v, unicode.IsControl) {
			return InvalidInput(fmt.Sprintf("tags.%s must not contain control characters", name))
		}
	}
	if t.Year != 0 && (t.Year < 1000 || t.Year > 9999) {
		return InvalidInput("tags.year must be a four-digit year")
	}
	return nil
}

// Or returns t with its empty fields taken from fallback.
func (t Tags) Or(fallback Tags) Tags {
	pick := func(v, f string) string {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
		return f
	}
	out := Tags{
		Artist: pick(t.Artist, fallback.Artist),
		Title:  pick(t.Title, fallback.Title),
		Album:  pick(t.Album, fallback.Album),
		Year:   t.Year,
	}
	if out.Year == 0 {
		out.Year = fallback.Year
	}
	return out
}

// VideoInfo is what a source knows about a video before downloading it.
type VideoInfo struct {
	ID       string
	Title    string
	Channel  string
	Uploader string
	// Track, Artists, Album and ReleaseYear are set for music the source
	// knows as such, e.g. YouTube Music songs.
	Track       string
	Artists     []string
	Album       string
	ReleaseYear int
	// Raw is the source's own representation of the video, handed back to
	// the Fetcher so it doesn't have to look the video up again.
	Raw []byte
}

// UnknownArtist is used when no artist can be found.
const UnknownArtist = "Unknown Artist"

var (
	// featuring splits "Artist feat. Other" into the main and featured
	// artists. "&", "," and " x " are not separators: they appear in band
	// names such as "Simon & Garfunkel".
	featuring = regexp.MustCompile(`(?i)\s+(?:feat\.?|ft\.?|featuring)\s+`)
	// featInTitle detects a featuring credit already present in a title.
	featInTitle = regexp.MustCompile(`(?i)\b(?:feat\.?|ft\.?|featuring)\s`)
	// bracketed matches one (...) or [...] group, with the space before it.
	bracketed = regexp.MustCompile(`\s*[(\[][^()\[\]]*[)\]]`)
	// videoNoise marks a bracketed group that describes the upload rather
	// than the recording, e.g. "(Official Video)" or "[4K]". Groups such as
	// "(Live at Wembley)" or "(Remastered 2011)" are kept.
	videoNoise = regexp.MustCompile(`(?i)\b(?:official|video|audio|lyrics?|visuali[sz]er|hd|hq|4k|mv|m/v)\b`)
	// artistTitle splits "Artist - Title" on a spaced hyphen, en or em dash.
	artistTitle = regexp.MustCompile(`^(.+?)\s+[-–—]\s+(.+)$`)
	// channelNoise is removed from a channel name used as the artist, e.g.
	// "Queen - Topic", "QueenVEVO" or "Queen Official".
	channelNoise = regexp.MustCompile(`(?i)(?:\s*-\s*topic|\s*vevo|\s+official)$`)
)

// GuessTags derives tags from what the source knows. In order of trust:
// music metadata (YouTube Music), an "Artist - Title" video title, and
// finally the channel name as the artist.
func GuessTags(info VideoInfo) Tags {
	artists := UniqueArtists(info.Artists)
	if info.Track != "" && len(artists) > 0 {
		t := Tags{
			Artist: artists[0],
			Title:  strings.TrimSpace(info.Track),
			Album:  strings.TrimSpace(info.Album),
			Year:   info.ReleaseYear,
		}
		// A long list is songwriting credits rather than featured artists.
		if others := artists[1:]; len(others) <= maxFeatured {
			t.Title = withFeaturing(t.Title, strings.Join(others, ", "))
		}
		return t
	}

	title := cleanTitle(info.Title)
	var t Tags
	if m := artistTitle.FindStringSubmatch(title); m != nil {
		t.Artist, t.Title = m[1], m[2]
	} else {
		t.Title = title
		t.Artist = cleanChannel(info.Channel)
		if t.Artist == "" {
			t.Artist = cleanChannel(info.Uploader)
		}
	}

	if parts := featuring.Split(t.Artist, 2); len(parts) == 2 {
		t.Artist = parts[0]
		t.Title = withFeaturing(t.Title, parts[1])
	}

	t.Artist = strings.TrimSpace(t.Artist)
	t.Title = strings.TrimSpace(t.Title)
	if t.Artist == "" {
		t.Artist = UnknownArtist
	}
	if t.Title == "" {
		t.Title = strings.TrimSpace(info.Title)
	}
	return t
}

// maxFeatured is the most artists credited as featured in a title.
const maxFeatured = 3

// UniqueArtists returns artists trimmed, without blanks or repeats (ignoring
// case), in their original order. YouTube Music sometimes lists every
// songwriter, some twice.
func UniqueArtists(artists []string) []string {
	var out []string
	for _, a := range artists {
		a = strings.TrimSpace(a)
		if a != "" && !slices.ContainsFunc(out, func(o string) bool { return strings.EqualFold(o, a) }) {
			out = append(out, a)
		}
	}
	return out
}

// ArtistCredit joins artists for display, shortening long lists:
// "A, B, C and 9 more".
func ArtistCredit(artists []string) string {
	artists = UniqueArtists(artists)
	if len(artists) <= maxFeatured+1 {
		return strings.Join(artists, ", ")
	}
	return fmt.Sprintf("%s and %d more", strings.Join(artists[:3], ", "), len(artists)-3)
}

// cleanTitle removes bracketed groups that describe the upload rather than
// the song, and collapses whitespace.
func cleanTitle(title string) string {
	title = bracketed.ReplaceAllStringFunc(title, func(group string) string {
		if videoNoise.MatchString(group) {
			return ""
		}
		return group
	})
	return strings.Join(strings.Fields(title), " ")
}

func cleanChannel(name string) string {
	return strings.TrimSpace(channelNoise.ReplaceAllString(strings.TrimSpace(name), ""))
}

// withFeaturing appends "(feat. others)" to title unless it already credits
// someone.
func withFeaturing(title, others string) string {
	others = strings.TrimSpace(others)
	if others == "" || featInTitle.MatchString(title) {
		return title
	}
	return title + " (feat. " + others + ")"
}
