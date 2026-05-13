package search

import (
	"context"
	"errors"
	"fmt"

	"github.com/lrstanley/go-ytdlp"
)

// Searcher is an interface for searching videos.
type Searcher interface {
	Search(ctx context.Context, param string, limit int, wantLiveStream bool) ([]*ytdlp.ExtractedInfo, error)
}

// YtdlpSearcher is a searcher that uses ytdlp.
type YtdlpSearcher struct{}

var (
	ErrSearchFailed  = errors.New("failed to run search")
	ErrExtractFailed = errors.New("failed to extract video info")
	ErrNoResults     = errors.New("no results found")
)

// Search searches for videos using ytdlp.
func (s *YtdlpSearcher) Search(ctx context.Context, param string, limit int, wantLiveStream bool) ([]*ytdlp.ExtractedInfo, error) {
	if limit <= 0 {
		limit = 5
	}

	cmd := ytdlp.New().
		SetExecutable("yt-dlp").
		NoUpdate().
		PrintJSON().
		IgnoreErrors().
		NoWarnings().
		SkipDownload().
		FlatPlaylist()

	if !wantLiveStream {
		cmd.MatchFilters("live_status != 'is_live'")
	}

	// When filtering out live streams, over-fetch to compensate for results
	// that may be removed by the live_status filter.
	fetchLimit := limit
	if !wantLiveStream {
		fetchLimit = limit * 4
	}

	searchStr := fmt.Sprintf("ytsearch%d:%s", fetchLimit, param)
	result, err := cmd.Run(ctx, searchStr)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSearchFailed, err)
	}

	searchResults, err := result.GetExtractedInfo()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrExtractFailed, err)
	}
	if len(searchResults) == 0 {
		return nil, ErrNoResults
	}

	videos := make([]*ytdlp.ExtractedInfo, 0, limit)
	for _, r := range searchResults {
		if r == nil || r.Title == nil || r.URL == nil {
			continue
		}
		videos = append(videos, r)
		if len(videos) == limit {
			break
		}
	}

	if len(videos) == 0 {
		return nil, ErrNoResults
	}

	return videos, nil
}
