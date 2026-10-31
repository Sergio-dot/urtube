package ytdlp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/Sergio-dot/urtube/internal/media"
)

// Inspect looks the video at url up with yt-dlp without downloading it. The
// returned info's Raw field holds yt-dlp's JSON, which Fetch downloads from.
func (f *Fetcher) Inspect(ctx context.Context, url string) (media.VideoInfo, error) {
	var out bytes.Buffer
	cmd := newCommand(f.Executable).DumpJSON().SkipDownload().NoPlaylist().NoWarnings()
	if err := f.run(ctx, cmd, &out, url); err != nil {
		return media.VideoInfo{}, err
	}
	return parseInfo(out.Bytes())
}

// infoFields is the subset of yt-dlp's info JSON urtube reads.
type infoFields struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Channel     string   `json:"channel"`
	Uploader    string   `json:"uploader"`
	Track       string   `json:"track"`
	Artist      string   `json:"artist"`
	Artists     []string `json:"artists"`
	Album       string   `json:"album"`
	ReleaseYear int      `json:"release_year"`
	// ReleaseDate (YYYYMMDD) is the original release; ReleaseYear can be a
	// remaster's.
	ReleaseDate string `json:"release_date"`
}

func parseInfo(raw []byte) (media.VideoInfo, error) {
	raw = bytes.TrimSpace(raw)
	var in infoFields
	if err := json.Unmarshal(raw, &in); err != nil {
		return media.VideoInfo{}, fmt.Errorf("parse video info: %w", err)
	}

	info := media.VideoInfo{
		ID:          in.ID,
		Title:       in.Title,
		Channel:     in.Channel,
		Uploader:    in.Uploader,
		Track:       in.Track,
		Artists:     in.Artists,
		Album:       in.Album,
		ReleaseYear: in.ReleaseYear,
		Raw:         raw,
	}
	if len(info.Artists) == 0 && in.Artist != "" {
		info.Artists = []string{in.Artist}
	}
	if len(in.ReleaseDate) >= 4 {
		if y, err := strconv.Atoi(in.ReleaseDate[:4]); err == nil {
			info.ReleaseYear = y
		}
	}
	return info, nil
}

// patchInfo returns raw with meta_* fields, which yt-dlp writes as file
// tags in preference to anything it derived itself. Tags urtube doesn't set
// are cleared rather than filled with YouTube's own: the upload date isn't
// a release year, the YouTube category isn't a genre, and descriptions are
// long and noisy.
func patchInfo(raw []byte, tags media.Tags) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber() // keep large numbers exact
	var info map[string]any
	if err := dec.Decode(&info); err != nil {
		return nil, fmt.Errorf("parse video info: %w", err)
	}

	year := ""
	if tags.Year != 0 {
		year = strconv.Itoa(tags.Year)
	}
	for k, v := range map[string]string{
		"meta_title":        tags.Title,
		"meta_artist":       tags.Artist,
		"meta_album_artist": tags.Artist,
		"meta_album":        tags.Album,
		"meta_date":         year,
		"meta_genre":        "",
		"meta_description":  "",
		"meta_synopsis":     "",
	} {
		info[k] = v
	}
	return json.Marshal(info)
}
