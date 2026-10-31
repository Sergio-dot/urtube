package httpapi

import (
	"fmt"
	"time"

	"github.com/Sergio-dot/urtube/internal/media"
)

// The types in this file define the JSON contract with the frontend
// (web/src/api.ts). Keep the two in sync.

type errorResponse struct {
	Error string `json:"error"`
}

// tagsDTO describes a song. On requests, empty fields are guessed.
type tagsDTO struct {
	Artist string `json:"artist"`
	Title  string `json:"title"`
	Album  string `json:"album"`
	Year   int    `json:"year,omitempty"`
}

func (t tagsDTO) toMedia() media.Tags {
	return media.Tags{Artist: t.Artist, Title: t.Title, Album: t.Album, Year: t.Year}
}

func newTagsDTO(t media.Tags) tagsDTO {
	return tagsDTO{Artist: t.Artist, Title: t.Title, Album: t.Album, Year: t.Year}
}

type downloadRequest struct {
	URL     string `json:"url"`
	VideoID string `json:"videoId,omitempty"`
	Title   string `json:"title,omitempty"`
	Options struct {
		Type   string `json:"type"`
		Format string `json:"format"`
	} `json:"options"`
	Tags *tagsDTO `json:"tags,omitempty"`
}

func (d downloadRequest) toMedia() media.Request {
	r := media.Request{
		URL:     d.URL,
		VideoID: d.VideoID,
		Title:   d.Title,
		Options: media.Options{Type: media.MediaType(d.Options.Type), Format: d.Options.Format},
	}
	if d.Tags != nil {
		r.Tags = d.Tags.toMedia()
	}
	return r
}

type suggestionResponse struct {
	Tags tagsDTO `json:"tags"`
	// KnownArtists are the artist folders already in the library.
	KnownArtists []string `json:"knownArtists"`
}

type downloadResponse struct {
	UUID    string `json:"uuid"`
	Message string `json:"message"`
}

type thumbnailResponse struct {
	URL    string `json:"url"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

type videoResponse struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Uploader string `json:"uploader"`
	// Duration is in whole seconds; zero for live streams.
	Duration   int                 `json:"duration"`
	Thumbnails []thumbnailResponse `json:"thumbnails"`
	URL        string              `json:"url"`
	IsLive     bool                `json:"isLive"`
	Album      string              `json:"album,omitempty"`
	// Tags are known for music results; the download dialog starts from
	// them.
	Tags *tagsDTO `json:"tags,omitempty"`
}

func newVideoResponse(v media.Video) videoResponse {
	thumbnails := make([]thumbnailResponse, 0, len(v.Thumbnails))
	for _, t := range v.Thumbnails {
		thumbnails = append(thumbnails, thumbnailResponse(t))
	}
	r := videoResponse{
		ID:         v.ID,
		Title:      v.Title,
		Uploader:   v.Uploader,
		Duration:   int(v.Duration.Seconds()),
		Thumbnails: thumbnails,
		URL:        v.URL,
		IsLive:     v.IsLive,
		Album:      v.Album,
	}
	if v.Tags != nil {
		t := newTagsDTO(*v.Tags)
		r.Tags = &t
	}
	return r
}

type settingsRequest struct {
	// DownloadDir is an absolute path; empty restores the default.
	DownloadDir string `json:"downloadDir"`
}

type settingsResponse struct {
	DownloadDir        string `json:"downloadDir"`
	DefaultDownloadDir string `json:"defaultDownloadDir"`
	// DownloadRoot is the folder a download folder must be inside, if any.
	DownloadRoot string `json:"downloadRoot,omitempty"`
	// DownloadDirError explains why downloads can't be saved right now.
	DownloadDirError string `json:"downloadDirError,omitempty"`
}

// downloadEvent is one message on the event stream. Progress values are
// preformatted for display and empty when unknown.
type downloadEvent struct {
	UUID         string `json:"uuid"`
	VideoID      string `json:"videoId"`
	Title        string `json:"title"`
	Status       string `json:"status"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	Percent      string `json:"percent"`
	Speed        string `json:"speed"`
	ETA          string `json:"eta"`
	Downloaded   string `json:"downloaded"`
	Total        string `json:"total"`
}

func newDownloadEvent(d media.Download) downloadEvent {
	e := downloadEvent{
		UUID:         d.ID,
		VideoID:      d.VideoID,
		Title:        d.Title,
		Status:       string(d.Status),
		ErrorMessage: d.Error,
	}

	p := d.Progress
	switch {
	case d.Status == media.StatusFinished:
		e.Percent = "100%"
	case p.TotalBytes > 0:
		e.Percent = fmt.Sprintf("%.2f%%", p.Percent())
	}
	if p.BytesPerSecond > 0 {
		e.Speed = formatBytes(int64(p.BytesPerSecond)) + "/s"
	}
	// yt-dlp estimates fractional seconds; "1m5s" reads better than "1m5.4s".
	if eta := p.ETA.Round(time.Second); eta > 0 {
		e.ETA = eta.String()
	}
	if p.DownloadedBytes > 0 {
		e.Downloaded = formatBytes(p.DownloadedBytes)
	}
	if p.TotalBytes > 0 {
		e.Total = formatBytes(p.TotalBytes)
	}
	return e
}

// formatBytes formats b with binary units, e.g. "5.0 MiB".
func formatBytes(b int64) string {
	const unit = 1024
	if b < 0 {
		return "0 B"
	}
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
