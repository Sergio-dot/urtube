package search

import (
	"context"
	"errors"
	"fmt"

	"github.com/lrstanley/go-ytdlp"
)

// Searcher is an interface for searching videos
type Searcher interface {
	Search(ctx context.Context, param string) ([]*ytdlp.ExtractedInfo, error)
}

// YtdlpSearcher is a searcher that uses ytdlp
type YtdlpSearcher struct{}

var (
	ErrSearchFailed  = errors.New("failed to run search")
	ErrExtractFailed = errors.New("failed to extract video info")
	ErrNoResults     = errors.New("no results found")
)

// Search searches for videos using ytdlp
func (s *YtdlpSearcher) Search(ctx context.Context, param string) ([]*ytdlp.ExtractedInfo, error) {
	searchStr := fmt.Sprintf("ytsearch5:%s", param)

	command, err := ytdlp.New().
		PrintJSON().
		SkipDownload().
		FlatPlaylist().
		Run(ctx, searchStr)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSearchFailed, err)
	}

	searchResults, err := command.GetExtractedInfo()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrExtractFailed, err)
	}
	if len(searchResults) == 0 {
		return nil, ErrNoResults
	}

	videos := make([]*ytdlp.ExtractedInfo, 0, len(searchResults))
	for _, result := range searchResults {
		if result == nil || result.Title == nil || result.URL == nil {
			continue
		}

		videos = append(videos, result)
	}
	if len(videos) == 0 {
		return nil, ErrNoResults
	}

	return videos, nil
}
