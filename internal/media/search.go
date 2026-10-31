package media

import (
	"context"
	"strings"
)

const (
	// DefaultSearchLimit is used when no positive limit is requested.
	DefaultSearchLimit = 5
	// MaxSearchLimit caps how many results a single search may return.
	MaxSearchLimit = 50
	// MaxMusicSearchLimit caps music searches, which look each result up
	// to get its artist and album.
	MaxMusicSearchLimit = 20
)

// SearchSource is where a search looks.
type SearchSource string

const (
	// SourceVideos searches regular YouTube videos.
	SourceVideos SearchSource = "videos"
	// SourceMusic searches YouTube Music songs, which come with artist,
	// album and year.
	SourceMusic SearchSource = "music"
)

// SearchQuery is a search request.
type SearchQuery struct {
	Text string
	// Limit is the maximum number of results; non-positive means
	// DefaultSearchLimit.
	Limit int
	// IncludeLive keeps live streams in video results.
	IncludeLive bool
	// Source is where to search; empty means SourceVideos.
	Source SearchSource
}

// SearchService finds videos.
type SearchService struct {
	source VideoSource
}

// NewSearchService creates a SearchService backed by source.
func NewSearchService(source VideoSource) *SearchService {
	return &SearchService{source: source}
}

// Search returns videos or songs matching q. Limits are defaulted and capped
// per source.
func (s *SearchService) Search(ctx context.Context, q SearchQuery) ([]Video, error) {
	if strings.TrimSpace(q.Text) == "" {
		return nil, ErrQueryRequired
	}
	if q.Limit <= 0 {
		q.Limit = DefaultSearchLimit
	}

	var (
		videos []Video
		err    error
	)
	switch q.Source {
	case SourceVideos, "":
		videos, err = s.source.Search(ctx, q.Text, min(q.Limit, MaxSearchLimit), q.IncludeLive)
	case SourceMusic:
		videos, err = s.source.SearchMusic(ctx, q.Text, min(q.Limit, MaxMusicSearchLimit))
	default:
		return nil, ErrInvalidSource
	}
	if err != nil {
		return nil, err
	}
	if len(videos) == 0 {
		return nil, ErrNoResults
	}
	return videos, nil
}
