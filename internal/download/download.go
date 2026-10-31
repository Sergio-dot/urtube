package download

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
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

// MediaType is the kind of media a download produces.
type MediaType string

const (
	// MediaVideo downloads the video and audio streams and merges them.
	MediaVideo MediaType = "video"
	// MediaAudio extracts the audio stream only.
	MediaAudio MediaType = "audio"
)

// allowedFormats lists the output formats accepted for each media type.
var allowedFormats = map[MediaType][]string{
	MediaVideo: {"mp4", "mkv", "webm"},
	MediaAudio: {"mp3", "m4a", "wav", "opus"},
}

var (
	ErrURLRequired      = errors.New("url is required")
	ErrInvalidURL       = errors.New("url must be an absolute http or https URL")
	ErrInvalidMediaType = errors.New("options.type must be one of: video, audio")
	ErrInvalidFormat    = errors.New("unsupported format")
)

// DownloadOptions describes the desired output of a download. It is translated
// into yt-dlp flags server-side so that clients can never pass raw flags.
type DownloadOptions struct {
	// Type is the kind of media to produce.
	Type MediaType `json:"type"`
	// Format is the output container or audio codec (e.g. mp4, mp3).
	Format string `json:"format"`
}

// Validate checks that the media type and format are in the allowlist.
func (o DownloadOptions) Validate() error {
	formats, ok := allowedFormats[o.Type]
	if !ok {
		return ErrInvalidMediaType
	}
	if !slices.Contains(formats, o.Format) {
		return fmt.Errorf("%w %q for type %s", ErrInvalidFormat, o.Format, o.Type)
	}
	return nil
}

// DownloadRequest is the request for downloading a video.
type DownloadRequest struct {
	// URL is the URL of the video to download.
	URL string `json:"url"`
	// VideoID is the identifier of the video.
	VideoID string `json:"videoId,omitempty"`
	// Title is the title of the video.
	Title string `json:"title,omitempty"`
	// Options describes the desired output.
	Options DownloadOptions `json:"options"`
}

// Validate checks if the DownloadRequest is valid.
func (r *DownloadRequest) Validate() error {
	if strutils.IsEmpty(r.URL) {
		return ErrURLRequired
	}
	// yt-dlp receives the URL as a positional argument after its flags, so
	// requiring an http(s) scheme also prevents it from being parsed as a flag.
	u, err := url.Parse(r.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ErrInvalidURL
	}
	return r.Options.Validate()
}

// passthroughEnv lists the parent environment variables forwarded to yt-dlp.
// Everything else (including any secrets in the container environment) is
// withheld.
var passthroughEnv = []string{
	"PATH", "HOME", "TMPDIR", "XDG_CACHE_HOME", "LANG", "LC_ALL", "TZ",
	"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "http_proxy", "https_proxy", "no_proxy",
}

// newCommand builds the yt-dlp command for the given options, without any
// progress handling.
func (d *YtdlpDownloader) newCommand(opts DownloadOptions) *ytdlp.Command {
	cmd := ytdlp.New().
		SetExecutable("yt-dlp").
		SetEnvVarInherit(false).
		NoUpdate().
		Newline(). // Force progress on new lines for parsing
		Output(filepath.Join(d.DownloadDir, "%(title)s.%(ext)s"))

	for _, k := range passthroughEnv {
		if v, ok := os.LookupEnv(k); ok {
			cmd.SetEnvVar(k, v)
		}
	}

	switch opts.Type {
	case MediaAudio:
		cmd.ExtractAudio().AudioFormat(opts.Format)
	case MediaVideo:
		cmd.Format("bestvideo+bestaudio/best").RecodeVideo(opts.Format)
	}

	return cmd
}

// Download downloads a video using ytdlp.
func (d *YtdlpDownloader) Download(ctx context.Context, body *DownloadRequest, onProgress func(ProgressUpdate)) error {
	cmd := d.newCommand(body.Options)

	if onProgress != nil {
		cmd.ProgressFunc(100*time.Millisecond, func(progress ytdlp.ProgressUpdate) {
			onProgress(ProgressUpdate{
				Status:     "downloading",
				VideoID:    body.VideoID,
				Title:      body.Title,
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
	val := int64(b)
	const unit = 1024
	if val < 0 {
		return "0 B"
	}
	if val < unit {
		return fmt.Sprintf("%d B", val)
	}
	div, exp := int64(unit), 0
	for n := val / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(val)/float64(div), "KMGTPE"[exp])
}
