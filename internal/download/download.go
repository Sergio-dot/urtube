package download

import (
	"context"
	"errors"

	"github.com/Sergio-dot/urtube/pkg/strutils"
	"github.com/lrstanley/go-ytdlp"
)

// Downloader is an interface for downloading videos
type Downloader interface {
	Download(ctx context.Context, body *DownloadRequest) error
}

// YtdlpDownloader is a downloader that uses ytdlp
type YtdlpDownloader struct {
	OutputDir string
}

// DownloadRequest is the request for downloading a video
type DownloadRequest struct {
	URL         string `json:"url"`
	PresetAlias string `json:"preset_alias"`
}

func (r *DownloadRequest) Validate() error {
	if strutils.IsEmpty(r.URL) {
		return errors.New("url is required")
	}
	if strutils.IsEmpty(r.PresetAlias) {
		return errors.New("preset_alias is required")
	}
	return nil
}

// Download downloads a video using ytdlp
func (d *YtdlpDownloader) Download(ctx context.Context, body *DownloadRequest) error {
	outputDir := d.OutputDir

	_, err := ytdlp.New().
		RemoteComponents("ejs:github").
		PresetAlias(body.PresetAlias).
		Output(outputDir+"/%(title)s.%(ext)s").
		Run(ctx, body.URL)
	if err != nil {
		return err
	}

	return nil
}
