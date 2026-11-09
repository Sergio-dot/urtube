package media

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

// Album is a release a song appears on.
type Album struct {
	Title string
	// Year is the release year, or 0 when unknown.
	Year int
}

// AlbumFinder finds the album a song belongs to, for when the source only
// knows, say, its single.
type AlbumFinder interface {
	// FindAlbum returns the album for tags' artist and title. A zero Album
	// means it doesn't know one.
	FindAlbum(ctx context.Context, tags Tags) (Album, error)
}

// albumLookupTimeout bounds one lookup, including waiting for its turn under
// the finder's rate limit.
const albumLookupTimeout = 10 * time.Second

// correctAlbum returns tags with the album finder gives for the song. The
// album is a nicety: when finder has no answer or fails, tags are returned
// unchanged.
func correctAlbum(ctx context.Context, finder AlbumFinder, tags Tags) Tags {
	if finder == nil || tags.Artist == UnknownArtist || tags.Title == "" {
		return tags
	}
	lookupCtx, cancel := context.WithTimeout(ctx, albumLookupTimeout)
	defer cancel()

	song := tags
	song.Title = withoutFeaturing(tags.Title)
	album, err := finder.FindAlbum(lookupCtx, song)
	if err != nil {
		if ctx.Err() == nil {
			slog.Warn("album lookup failed, keeping the source's album", "error", err)
		}
		return tags
	}
	if album.Title == "" || album.Title == tags.Album {
		return tags
	}
	tags.Album = album.Title
	if album.Year != 0 {
		tags.Year = album.Year
	}
	return tags
}

// withoutFeaturing removes bracketed featuring credits from title, which
// catalogues keep in the artist credit instead.
func withoutFeaturing(title string) string {
	title = bracketed.ReplaceAllStringFunc(title, func(group string) string {
		if featInTitle.MatchString(group) {
			return ""
		}
		return group
	})
	return strings.TrimSpace(title)
}
