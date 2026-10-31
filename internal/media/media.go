// Package media is urtube's core: the domain types, the search and download
// use cases, and the interfaces they need from the outside world. It imports
// nothing from the transport or yt-dlp packages.
package media

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"
)

// Thumbnail is a preview image of a video.
type Thumbnail struct {
	URL    string
	Width  int
	Height int
}

// Video is a search result.
type Video struct {
	ID       string
	Title    string
	Uploader string
	// Duration is zero for live streams and when unknown.
	Duration   time.Duration
	Thumbnails []Thumbnail
	URL        string
	IsLive     bool
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

// Options describes the desired output of a download.
type Options struct {
	// Type is the kind of media to produce.
	Type MediaType
	// Format is the output container or audio codec (e.g. mp4, mp3).
	Format string
}

// Validate checks that the media type and format are in the allowlist.
func (o Options) Validate() error {
	formats, ok := allowedFormats[o.Type]
	if !ok {
		return ErrInvalidMediaType
	}
	if !slices.Contains(formats, o.Format) {
		return fmt.Errorf("%w %q for type %s", ErrInvalidFormat, o.Format, o.Type)
	}
	return nil
}

// Request asks for one video to be downloaded.
type Request struct {
	// URL is the URL of the video to download.
	URL string
	// VideoID and Title identify the video to clients; they are not trusted.
	VideoID string
	Title   string
	// Options describes the desired output.
	Options Options
	// Tags are the client's corrections; empty fields are guessed from the
	// video when the download starts.
	Tags Tags
}

// Validate checks that the URL is an absolute http(s) URL and the options are
// allowed.
func (r Request) Validate() error {
	if err := validateURL(r.URL); err != nil {
		return err
	}
	if err := r.Options.Validate(); err != nil {
		return err
	}
	return r.Tags.Validate()
}

// validateURL accepts absolute http(s) URLs only. Sources pass the URL to a
// command line, so requiring the scheme also keeps it from being read as a
// flag.
func validateURL(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return ErrURLRequired
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ErrInvalidURL
	}
	return nil
}

// Status is the lifecycle state of a download.
type Status string

const (
	StatusQueued      Status = "queued"
	StatusDownloading Status = "downloading"
	StatusFinished    Status = "finished"
	StatusError       Status = "error"
	StatusCancelled   Status = "cancelled"
)

// Terminal reports whether the download has ended.
func (s Status) Terminal() bool {
	return s == StatusFinished || s == StatusError || s == StatusCancelled
}

// Progress describes how far a running download has got. Zero values mean
// unknown.
type Progress struct {
	DownloadedBytes int64
	TotalBytes      int64
	// BytesPerSecond is the current transfer speed.
	BytesPerSecond float64
	ETA            time.Duration
}

// Percent returns the completed percentage, or 0 when the total is unknown.
func (p Progress) Percent() float64 {
	if p.TotalBytes <= 0 {
		return 0
	}
	return min(float64(p.DownloadedBytes)/float64(p.TotalBytes)*100, 100)
}

// Download is a snapshot of a download's state. Every change is published to
// subscribers as a new snapshot.
type Download struct {
	ID       string
	VideoID  string
	Title    string
	Status   Status
	Progress Progress
	// Error describes why the download failed when Status is StatusError.
	Error string
}

// VideoSource finds videos.
type VideoSource interface {
	// Search returns up to limit videos matching query, excluding live
	// streams unless includeLive is set.
	Search(ctx context.Context, query string, limit int, includeLive bool) ([]Video, error)
}

// Inspector looks up a video without downloading it.
type Inspector interface {
	// Inspect returns what the source knows about the video at url.
	Inspect(ctx context.Context, url string) (VideoInfo, error)
}

// Fetcher downloads a single video to storage.
type Fetcher interface {
	Inspector
	// Fetch carries out job, reporting progress through onProgress, until
	// it completes or ctx is cancelled.
	Fetch(ctx context.Context, job FetchJob, onProgress func(Progress)) error
}

// FetchJob is a download with every decision made: what to fetch, how to
// tag it and where to save it.
type FetchJob struct {
	// Info comes from Inspector.Inspect for the requested URL.
	Info    VideoInfo
	Options Options
	Tags    Tags
	// Dir is the library folder; Path is the file's location inside it,
	// as path components without the extension (see PlanPath).
	Dir  string
	Path []string
	// TrimNonMusic asks to cut segments marked as non-music, if the source
	// knows any.
	TrimNonMusic bool
}

// Destination decides where finished downloads are saved.
type Destination interface {
	// DownloadDir returns the folder to save downloads to, or an error if it
	// can't be used right now.
	DownloadDir() (string, error)
}
