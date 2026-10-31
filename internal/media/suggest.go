package media

import (
	"context"
	"fmt"
	"os"
)

// maxKnownArtists bounds the artist suggestions returned to clients.
const maxKnownArtists = 1000

// Suggestion is what a client shows before a download starts: the tags
// urtube would use and the artists already in the library.
type Suggestion struct {
	Tags         Tags
	KnownArtists []string
}

// SuggestService suggests tags for a video before it is downloaded, so
// users can correct them.
type SuggestService struct {
	inspector   Inspector
	destination Destination
}

// NewSuggestService creates a SuggestService that looks videos up with
// inspector and lists artists in destination's folder.
func NewSuggestService(inspector Inspector, destination Destination) *SuggestService {
	return &SuggestService{inspector: inspector, destination: destination}
}

// Suggest returns the guessed tags for the video at url and the artists in
// the library. An unusable download folder only leaves the artist list
// empty.
func (s *SuggestService) Suggest(ctx context.Context, url string) (Suggestion, error) {
	if err := validateURL(url); err != nil {
		return Suggestion{}, err
	}
	info, err := s.inspector.Inspect(ctx, url)
	if err != nil {
		if ctx.Err() != nil {
			return Suggestion{}, ctx.Err()
		}
		return Suggestion{}, fmt.Errorf("%w: %w", ErrVideoUnavailable, err)
	}

	sug := Suggestion{Tags: GuessTags(info)}
	if dir, err := s.destination.DownloadDir(); err == nil {
		sug.KnownArtists = KnownArtists(os.DirFS(dir), maxKnownArtists)
	}
	return sug, nil
}
