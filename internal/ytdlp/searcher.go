package ytdlp

import (
	"context"
	"fmt"
	"time"

	"github.com/Sergio-dot/urtube/internal/media"
	"github.com/lrstanley/go-ytdlp"
)

// Searcher searches YouTube with yt-dlp.
type Searcher struct {
	// Executable is the name or path of yt-dlp. Empty uses "yt-dlp".
	Executable string
}

// Search returns up to limit videos matching query. It returns an empty slice,
// not an error, when nothing matches.
func (s *Searcher) Search(ctx context.Context, query string, limit int, includeLive bool) ([]media.Video, error) {
	cmd := newCommand(s.Executable).
		PrintJSON().
		IgnoreErrors().
		NoWarnings().
		SkipDownload().
		FlatPlaylist()

	// When filtering out live streams, over-fetch to compensate for results
	// that may be removed by the live_status filter.
	fetchLimit := limit
	if !includeLive {
		cmd.MatchFilters("live_status != 'is_live'")
		fetchLimit = limit * 4
	}

	result, err := cmd.Run(ctx, fmt.Sprintf("ytsearch%d:%s", fetchLimit, query))
	if err != nil {
		return nil, fmt.Errorf("run yt-dlp search: %w", err)
	}

	infos, err := result.GetExtractedInfo()
	if err != nil {
		return nil, fmt.Errorf("parse yt-dlp search results: %w", err)
	}

	videos := make([]media.Video, 0, limit)
	for _, info := range infos {
		v, ok := toVideo(info)
		if !ok {
			continue
		}
		videos = append(videos, v)
		if len(videos) == limit {
			break
		}
	}
	return videos, nil
}

// toVideo maps a yt-dlp search result to a media.Video, reporting false for
// results without a title or URL.
func toVideo(info *ytdlp.ExtractedInfo) (media.Video, bool) {
	if info == nil || info.Title == nil || info.URL == nil {
		return media.Video{}, false
	}

	v := media.Video{
		ID:    info.ID,
		Title: *info.Title,
		URL:   *info.URL,
		IsLive: (info.LiveStatus != nil && *info.LiveStatus == ytdlp.ExtractedLiveStatusIsLive) ||
			(info.IsLive != nil && *info.IsLive),
	}
	switch {
	case info.Uploader != nil:
		v.Uploader = *info.Uploader
	case info.Channel != nil:
		v.Uploader = *info.Channel
	}
	if info.Duration != nil && !v.IsLive {
		v.Duration = time.Duration(*info.Duration * float64(time.Second))
	}
	for _, th := range info.Thumbnails {
		if th == nil || th.URL == "" {
			continue
		}
		t := media.Thumbnail{URL: th.URL}
		if th.Width != nil {
			t.Width = *th.Width
		}
		if th.Height != nil {
			t.Height = *th.Height
		}
		v.Thumbnails = append(v.Thumbnails, t)
	}
	return v, true
}
