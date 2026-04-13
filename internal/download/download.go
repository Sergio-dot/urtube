package download

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/Sergio-dot/urtube/pkg/strutils"
	"github.com/lrstanley/go-ytdlp"
)

// Downloader is an interface for downloading videos
type Downloader interface {
	Download(ctx context.Context, body *DownloadRequest) error
}

// YtdlpDownloader is a downloader that uses ytdlp
type YtdlpDownloader struct {
	DownloadDir string
}

// DownloadRequest is the request for downloading a video
type DownloadRequest struct {
	URL   string            `json:"url"`
	Env   map[string]string `json:"env,omitempty"`
	Flags *ytdlp.FlagConfig `json:"flags,omitempty"`
}

func (r *DownloadRequest) Validate() error {
	if strutils.IsEmpty(r.URL) {
		return errors.New("url is required")
	}
	if r.Flags != nil {
		if err := r.Flags.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Download downloads a video using ytdlp
func (d *YtdlpDownloader) Download(ctx context.Context, body *DownloadRequest) error {
	cmd := ytdlp.New().
		RemoteComponents("ejs:github")

	if body.Flags != nil {
		cmd.SetFlagConfig(body.Flags)
	}

	if body.Flags != nil && body.Flags.Filesystem.Output != nil {
		cmd.Output(filepath.Join(d.DownloadDir, *body.Flags.Filesystem.Output))
	} else {
		cmd.Output(filepath.Join(d.DownloadDir, "%(title)s.%(ext)s"))
	}

	if body.Env != nil {
		for k, v := range body.Env {
			cmd.SetEnvVar(k, v)
		}
	}

	_, err := cmd.Run(ctx, body.URL)
	if err != nil {
		return err
	}

	return nil
}
