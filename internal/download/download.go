package download

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/Sergio-dot/urtube/pkg/strutils"
	"github.com/lrstanley/go-ytdlp"
)

// Downloader is an interface for downloading videos.
type Downloader interface {
	// Download starts a video download and provides progress updates.
	Download(ctx context.Context, body *DownloadRequest, onProgress func(ProgressUpdate)) error
}

// ProgressUpdate represents a progress update for a download.
type ProgressUpdate struct {
	// UUID is the unique identifier for the download.
	UUID string `json:"uuid"`
	// VideoID is the identifier of the video.
	VideoID string `json:"videoId"`
	// Title is the title of the video.
	Title string `json:"title"`
	// Status is the current status of the download (e.g., downloading, finished, error).
	Status string `json:"status"`
	// ErrorMessage contains the error message if the status is error.
	ErrorMessage string `json:"errorMessage,omitempty"`
	// Percent is the percentage of the download completed.
	Percent string `json:"percent"`
	// Speed is the current download speed.
	Speed string `json:"speed"`
	// ETA is the estimated time of arrival.
	ETA string `json:"eta"`
	// Downloaded is the amount of data downloaded so far.
	Downloaded string `json:"downloaded"`
	// Total is the total size of the download.
	Total string `json:"total"`
}

// YtdlpDownloader is a downloader that uses ytdlp.
type YtdlpDownloader struct {
	// DownloadDir is the directory where downloads will be saved.
	DownloadDir string
}

// DownloadRequest is the request for downloading a video.
type DownloadRequest struct {
	// URL is the URL of the video to download.
	URL string `json:"url"`
	// VideoID is the identifier of the video.
	VideoID string `json:"videoId,omitempty"`
	// Title is the title of the video.
	Title string `json:"title,omitempty"`
	// Env contains environment variables for the download command.
	Env map[string]string `json:"env,omitempty"`
	// Flags contains yt-dlp flags for the download command.
	Flags *ytdlp.FlagConfig `json:"flags,omitempty"`
}

// Validate checks if the DownloadRequest is valid.
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

// Download downloads a video using ytdlp.
func (d *YtdlpDownloader) Download(ctx context.Context, body *DownloadRequest, onProgress func(ProgressUpdate)) error {
	cmd := ytdlp.New().
		SetExecutable("yt-dlp").
		NoUpdate().
		Newline() // Force progress on new lines for parsing

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

	if onProgress != nil {
		cmd.ProgressFunc(100*time.Millisecond, func(progress ytdlp.ProgressUpdate) {
			onProgress(ProgressUpdate{
				Status:     "downloading",
				Percent:    progress.PercentString(),
				ETA:        progress.ETA().String(),
				Downloaded: formatBytes(progress.DownloadedBytes),
				Total:      formatBytes(progress.TotalBytes),
			})
		})
	}

	_, err := cmd.Run(ctx, body.URL)

	return err
}

func formatBytes(b int) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
