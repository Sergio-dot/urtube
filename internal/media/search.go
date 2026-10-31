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
)

// SearchService finds videos.
type SearchService struct {
	source VideoSource
}

// NewSearchService creates a SearchService backed by source.
func NewSearchService(source VideoSource) *SearchService {
	return &SearchService{source: source}
}

// Search returns videos matching query. A non-positive limit uses
// DefaultSearchLimit and limits above MaxSearchLimit are capped.
func (s *SearchService) Search(ctx context.Context, query string, limit int, includeLive bool) ([]Video, error) {
	if strings.TrimSpace(query) == "" {
		return nil, ErrQueryRequired
	}
	if limit <= 0 {
		limit = DefaultSearchLimit
	}
	limit = min(limit, MaxSearchLimit)

	videos, err := s.source.Search(ctx, query, limit, includeLive)
	if err != nil {
		return nil, err
	}
	if len(videos) == 0 {
		return nil, ErrNoResults
	}
	return videos, nil
}
