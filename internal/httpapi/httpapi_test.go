package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/Sergio-dot/urtube/internal/media"
	"github.com/stretchr/testify/assert"
)

type fakeSearcher struct {
	videos []media.Video
	err    error

	gotQuery  string
	gotLimit  int
	gotLive   bool
	gotSource media.SearchSource
}

func (f *fakeSearcher) Search(ctx context.Context, q media.SearchQuery) ([]media.Video, error) {
	f.gotQuery, f.gotLimit, f.gotLive, f.gotSource = q.Text, q.Limit, q.IncludeLive, q.Source
	return f.videos, f.err
}

// blockingFetcher blocks every download until its context is cancelled.
type blockingFetcher struct{}

func (blockingFetcher) Inspect(ctx context.Context, url string) (media.VideoInfo, error) {
	return media.VideoInfo{Title: "Artist - Song"}, nil
}

func (blockingFetcher) Fetch(ctx context.Context, job media.FetchJob, onProgress func(media.Progress)) error {
	<-ctx.Done()
	return ctx.Err()
}

// fakeSuggester returns sug (or err) and records the URL it was asked about.
type fakeSuggester struct {
	sug    media.Suggestion
	err    error
	gotURL string
}

func (f *fakeSuggester) Suggest(ctx context.Context, url string) (media.Suggestion, error) {
	f.gotURL = url
	return f.sug, f.err
}

// fakeSettings stores the download folder in memory. dirErr, if set, is
// reported for the current folder; setErr is returned by SetDownloadDir.
type fakeSettings struct {
	root        string
	jellyfinURL string
	jellyfinKey string
	dir         string
	dirErr      error
	setErr      error
}

func (f *fakeSettings) DownloadDir() (string, error) {
	if f.dir == "" {
		return "/downloads", f.dirErr
	}
	return f.dir, f.dirErr
}

func (f *fakeSettings) DefaultDownloadDir() string { return "/downloads" }

func (f *fakeSettings) Root() string { return f.root }

func (f *fakeSettings) Jellyfin() (string, string) { return f.jellyfinURL, f.jellyfinKey }

func (f *fakeSettings) SetJellyfin(url string, apiKey *string) error {
	if strings.HasPrefix(url, "bad") {
		return media.InvalidInput("the Jellyfin URL must look like http://192.168.1.10:8096")
	}
	f.jellyfinURL = url
	if apiKey != nil {
		f.jellyfinKey = *apiKey
	}
	return nil
}

func (f *fakeSettings) SetDownloadDir(dir string) error {
	if f.setErr != nil {
		return f.setErr
	}
	f.dir, f.dirErr = dir, nil
	return nil
}

// drainNow drains svc, cancelling running downloads immediately.
func drainNow(svc *media.DownloadService) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	svc.Drain(ctx)
}

func newTestRouter(t *testing.T, deps Deps) http.Handler {
	t.Helper()
	if deps.Search == nil {
		deps.Search = &fakeSearcher{}
	}
	if deps.Settings == nil {
		deps.Settings = &fakeSettings{}
	}
	if deps.Suggest == nil {
		deps.Suggest = &fakeSuggester{}
	}
	if deps.Downloads == nil {
		svc := media.NewDownloadService(blockingFetcher{}, &fakeSettings{}, 2)
		t.Cleanup(func() { drainNow(svc) })
		deps.Downloads = svc
	}
	return NewRouter(deps)
}

func do(h http.Handler, method, target, body string) *httptest.ResponseRecorder {
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(method, target, r))
	return w
}

func errorMessage(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var resp errorResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp.Error
}

func TestErrorStatus(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		msg    string
	}{
		{name: "transport error", err: statusError{status: http.StatusBadRequest, msg: "invalid request body"}, status: 400, msg: "invalid request body"},
		{name: "validation error", err: media.ErrInvalidURL, status: 400, msg: media.ErrInvalidURL.Error()},
		{name: "wrapped validation error", err: fmt.Errorf("%w %q", media.ErrInvalidFormat, "exe"), status: 400, msg: `unsupported format "exe"`},
		{name: "not found", err: media.ErrNotFound, status: 404, msg: "download not found"},
		{name: "no results", err: media.ErrNoResults, status: 404, msg: "no results found"},
		{name: "draining", err: media.ErrDraining, status: 503, msg: "server is shutting down"},
		{name: "unknown errors don't leak", err: errors.New("exec: /usr/bin/yt-dlp: permission denied"), status: 500, msg: "internal server error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, msg := errorStatus(tt.err)
			assert.Equal(t, tt.status, status)
			assert.Equal(t, tt.msg, msg)
		})
	}
}

func TestSearch(t *testing.T) {
	t.Run("passes parameters and maps results", func(t *testing.T) {
		search := &fakeSearcher{videos: []media.Video{
			{ID: "a", Title: "Song", Uploader: "Artist", Duration: 212 * time.Second, URL: "https://youtube.com/watch?v=a",
				Thumbnails: []media.Thumbnail{{URL: "https://i.ytimg.com/a.jpg", Width: 320, Height: 180}}},
			{ID: "b", Title: "Live", URL: "https://youtube.com/watch?v=b", IsLive: true},
		}}
		h := newTestRouter(t, Deps{Search: search})

		w := do(h, http.MethodGet, "/api/v1/search/never%20gonna?limit=20&wantLiveStreams=true", "")

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "never gonna", search.gotQuery)
		assert.Equal(t, 20, search.gotLimit)
		assert.True(t, search.gotLive)
		assert.JSONEq(t, `[
			{"id":"a","title":"Song","uploader":"Artist","duration":212,"url":"https://youtube.com/watch?v=a","isLive":false,
			 "thumbnails":[{"url":"https://i.ytimg.com/a.jpg","width":320,"height":180}]},
			{"id":"b","title":"Live","uploader":"","duration":0,"url":"https://youtube.com/watch?v=b","isLive":true,"thumbnails":[]}
		]`, w.Body.String())
	})

	t.Run("invalid limit is left to the service", func(t *testing.T) {
		search := &fakeSearcher{videos: []media.Video{{ID: "a"}}}
		h := newTestRouter(t, Deps{Search: search})

		do(h, http.MethodGet, "/api/v1/search/q?limit=abc", "")
		assert.Equal(t, 0, search.gotLimit)
		assert.False(t, search.gotLive)
	})

	t.Run("errors are mapped", func(t *testing.T) {
		tests := []struct {
			err    error
			status int
		}{
			{err: media.ErrNoResults, status: http.StatusNotFound},
			{err: media.ErrQueryRequired, status: http.StatusBadRequest},
			{err: errors.New("yt-dlp crashed"), status: http.StatusInternalServerError},
		}
		for _, tt := range tests {
			h := newTestRouter(t, Deps{Search: &fakeSearcher{err: tt.err}})
			w := do(h, http.MethodGet, "/api/v1/search/q", "")
			assert.Equal(t, tt.status, w.Code, tt.err.Error())
		}
	})
}

func TestStartDownload(t *testing.T) {
	valid := `{"url":"https://www.youtube.com/watch?v=dQw4w9WgXcQ","videoId":"dQw4w9WgXcQ","title":"Song","options":{"type":"audio","format":"mp3"}}`

	t.Run("accepts a valid request", func(t *testing.T) {
		w := do(newTestRouter(t, Deps{}), http.MethodPost, "/api/v1/download", valid)

		assert.Equal(t, http.StatusAccepted, w.Code)
		var resp downloadResponse
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.NotEmpty(t, resp.UUID)
		assert.Equal(t, "download started", resp.Message)
	})

	t.Run("rejects bad requests", func(t *testing.T) {
		tests := []struct {
			name string
			body string
			msg  string
		}{
			{name: "malformed JSON", body: `{"url":`, msg: "invalid request body"},
			{name: "raw yt-dlp flags", body: `{"url":"https://youtu.be/x","options":{"type":"audio","format":"mp3"},"flags":{"post_processing":{"exec":["id"]}}}`, msg: "invalid request body"},
			{name: "env injection", body: `{"url":"https://youtu.be/x","options":{"type":"audio","format":"mp3"},"env":{"LD_PRELOAD":"x"}}`, msg: "invalid request body"},
			{name: "URL as a flag", body: `{"url":"--batch-file=/etc/passwd","options":{"type":"audio","format":"mp3"}}`, msg: media.ErrInvalidURL.Error()},
			{name: "format not allowed", body: `{"url":"https://youtu.be/x","options":{"type":"audio","format":"exe"}}`, msg: `unsupported format "exe" for type audio`},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				w := do(newTestRouter(t, Deps{}), http.MethodPost, "/api/v1/download", tt.body)
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.Equal(t, tt.msg, errorMessage(t, w))
			})
		}
	})

	t.Run("503 while shutting down", func(t *testing.T) {
		svc := media.NewDownloadService(blockingFetcher{}, &fakeSettings{}, 1)
		drainNow(svc)

		w := do(newTestRouter(t, Deps{Downloads: svc}), http.MethodPost, "/api/v1/download", valid)
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})
}

func TestCancelDownload(t *testing.T) {
	svc := media.NewDownloadService(blockingFetcher{}, &fakeSettings{}, 1)
	t.Cleanup(func() { drainNow(svc) })
	h := newTestRouter(t, Deps{Downloads: svc})

	t.Run("unknown id is 404", func(t *testing.T) {
		w := do(h, http.MethodDelete, "/api/v1/download/nope", "")
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("active download is cancelled", func(t *testing.T) {
		d, err := svc.Start(media.Request{URL: "https://example.com/v", Options: media.Options{Type: media.MediaAudio, Format: "mp3"}})
		assert.NoError(t, err)

		w := do(h, http.MethodDelete, "/api/v1/download/"+d.ID, "")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"uuid":"`+d.ID+`","message":"download cancelled"}`, w.Body.String())
	})
}

// streamEvents runs the events handler until ctx is done or the stream ends
// and returns what it wrote.
func streamEvents(t *testing.T, a *api, ctx context.Context) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil).WithContext(ctx)
	assert.NoError(t, a.events(w, req))
	return w
}

func TestEvents(t *testing.T) {
	audio := media.Options{Type: media.MediaAudio, Format: "mp3"}

	t.Run("sends active downloads on connect", func(t *testing.T) {
		svc := media.NewDownloadService(blockingFetcher{}, &fakeSettings{}, 1)
		t.Cleanup(func() { drainNow(svc) })
		running, _ := svc.Start(media.Request{URL: "https://example.com/1", Title: "One", Options: audio})
		queued, _ := svc.Start(media.Request{URL: "https://example.com/2", Title: "Two", Options: audio})

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
		defer cancel()
		w := streamEvents(t, &api{Deps: Deps{Downloads: svc}, keepAlive: time.Hour}, ctx)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
		body := w.Body.String()
		assert.Contains(t, body, `"uuid":"`+running.ID+`","videoId":"","title":"One","status":"downloading"`)
		assert.Contains(t, body, `"uuid":"`+queued.ID+`","videoId":"","title":"Two","status":"queued"`)
		assert.Less(t, strings.Index(body, running.ID), strings.Index(body, queued.ID), "oldest first")
	})

	t.Run("sends keep-alives", func(t *testing.T) {
		svc := media.NewDownloadService(blockingFetcher{}, &fakeSettings{}, 1)
		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Millisecond)
		defer cancel()

		w := streamEvents(t, &api{Deps: Deps{Downloads: svc}, keepAlive: 10 * time.Millisecond}, ctx)
		assert.Contains(t, w.Body.String(), ": keep-alive\n\n")
	})

	t.Run("ends when the service drains", func(t *testing.T) {
		svc := media.NewDownloadService(blockingFetcher{}, &fakeSettings{}, 1)
		d, _ := svc.Start(media.Request{URL: "https://example.com/1", Options: audio})

		done := make(chan *httptest.ResponseRecorder, 1)
		go func() {
			done <- streamEvents(t, &api{Deps: Deps{Downloads: svc}, keepAlive: time.Hour}, context.Background())
		}()
		time.Sleep(20 * time.Millisecond)
		drainNow(svc)

		select {
		case w := <-done:
			assert.Contains(t, w.Body.String(), `"uuid":"`+d.ID+`","videoId":"","title":"","status":"cancelled"`)
		case <-time.After(2 * time.Second):
			t.Fatal("event stream did not end after Drain")
		}
	})
}

func TestNewDownloadEvent(t *testing.T) {
	t.Run("formats progress", func(t *testing.T) {
		e := newDownloadEvent(media.Download{
			ID: "id", VideoID: "v", Title: "Song", Status: media.StatusDownloading,
			Progress: media.Progress{DownloadedBytes: 1 << 20, TotalBytes: 4 << 20, BytesPerSecond: 512 << 10, ETA: 6 * time.Second},
		})
		assert.Equal(t, downloadEvent{
			UUID: "id", VideoID: "v", Title: "Song", Status: "downloading",
			Percent: "25.00%", Speed: "512.0 KiB/s", ETA: "6s", Downloaded: "1.0 MiB", Total: "4.0 MiB",
		}, e)
	})

	t.Run("eta is rounded to whole seconds", func(t *testing.T) {
		eta := func(d time.Duration) string {
			return newDownloadEvent(media.Download{Progress: media.Progress{ETA: d}}).ETA
		}
		assert.Equal(t, "1m5s", eta(65400*time.Millisecond))
		assert.Equal(t, "", eta(400*time.Millisecond), "under half a second is unknown, not 0s")
	})

	t.Run("unknown progress is empty", func(t *testing.T) {
		e := newDownloadEvent(media.Download{ID: "id", Status: media.StatusQueued})
		assert.Equal(t, downloadEvent{UUID: "id", Status: "queued"}, e)
	})

	t.Run("finished is 100%", func(t *testing.T) {
		assert.Equal(t, "100%", newDownloadEvent(media.Download{Status: media.StatusFinished}).Percent)
	})

	t.Run("errors carry the message", func(t *testing.T) {
		e := newDownloadEvent(media.Download{Status: media.StatusError, Error: "Video unavailable"})
		assert.Equal(t, "Video unavailable", e.ErrorMessage)
	})
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{input: -10, expected: "0 B"},
		{input: 0, expected: "0 B"},
		{input: 500, expected: "500 B"},
		{input: 1024, expected: "1.0 KiB"},
		{input: 5 << 20, expected: "5.0 MiB"},
		{input: 2 << 30, expected: "2.0 GiB"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, formatBytes(tt.input))
	}
}

func TestHealthAndProbes(t *testing.T) {
	var ready atomic.Bool
	ready.Store(true)
	h := newTestRouter(t, Deps{YtdlpVersion: "2026.10.01", Ready: ready.Load})

	t.Run("health reports the startup version", func(t *testing.T) {
		w := do(h, http.MethodGet, "/api/v1/health", "")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"status":"ok","dependencies":{"ytdlp":true,"version":"2026.10.01"}}`, w.Body.String())
	})

	t.Run("health is degraded without yt-dlp", func(t *testing.T) {
		w := do(newTestRouter(t, Deps{}), http.MethodGet, "/api/v1/health", "")
		assert.JSONEq(t, `{"status":"degraded","dependencies":{"ytdlp":false}}`, w.Body.String())
	})

	t.Run("livez", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, do(h, http.MethodGet, "/livez", "").Code)
	})

	t.Run("readyz follows Ready", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, do(h, http.MethodGet, "/readyz", "").Code)
		ready.Store(false)
		assert.Equal(t, http.StatusServiceUnavailable, do(h, http.MethodGet, "/readyz", "").Code)
	})
}

func TestUI(t *testing.T) {
	ui := fstest.MapFS{
		"index.html":    {Data: []byte("<html>app</html>")},
		"assets/app.js": {Data: []byte("console.log(1)")},
	}
	h := newTestRouter(t, Deps{UI: ui})

	t.Run("serves files", func(t *testing.T) {
		w := do(h, http.MethodGet, "/assets/app.js", "")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "console.log(1)", w.Body.String())
	})

	t.Run("falls back to index.html for client routes", func(t *testing.T) {
		w := do(h, http.MethodGet, "/some/client/route", "")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "<html>app</html>", w.Body.String())
	})

	t.Run("unknown API paths are 404, not the app", func(t *testing.T) {
		w := do(h, http.MethodGet, "/api/v1/nope", "")
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.False(t, bytes.Contains(w.Body.Bytes(), []byte("<html>")))
	})

	t.Run("no UI is 404", func(t *testing.T) {
		w := do(newTestRouter(t, Deps{}), http.MethodGet, "/", "")
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestSettings(t *testing.T) {
	t.Run("get reports the folder and its problem", func(t *testing.T) {
		h := newTestRouter(t, Deps{Settings: &fakeSettings{root: "/", dirErr: errors.New("/downloads is not writable: permission denied")}})

		w := do(h, http.MethodGet, "/api/v1/settings", "")

		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{
			"downloadDir": "/downloads",
			"defaultDownloadDir": "/downloads",
			"downloadRoot": "/",
			"downloadDirError": "/downloads is not writable: permission denied",
			"jellyfin": {"url": "", "apiKeySet": false, "enabled": false}
		}`, w.Body.String())
	})

	t.Run("put changes the folder", func(t *testing.T) {
		s := &fakeSettings{}
		h := newTestRouter(t, Deps{Settings: s})

		w := do(h, http.MethodPut, "/api/v1/settings", `{"downloadDir": " /media/music "}`)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"downloadDir": "/media/music", "defaultDownloadDir": "/downloads",
			"jellyfin": {"url": "", "apiKeySet": false, "enabled": false}}`, w.Body.String())
		assert.Equal(t, "/media/music", s.dir, "surrounding whitespace is trimmed")
	})

	t.Run("put rejects invalid folders with the reason", func(t *testing.T) {
		h := newTestRouter(t, Deps{Settings: &fakeSettings{setErr: media.InvalidInput("download folder must be an absolute path")}})

		w := do(h, http.MethodPut, "/api/v1/settings", `{"downloadDir": "music"}`)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "download folder must be an absolute path", errorMessage(t, w))
	})

	t.Run("put rejects unknown fields", func(t *testing.T) {
		h := newTestRouter(t, Deps{})
		w := do(h, http.MethodPut, "/api/v1/settings", `{"tempDir": "/x"}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("downloads are refused while the folder is unavailable", func(t *testing.T) {
		settings := &fakeSettings{dirErr: errors.New("/downloads is not writable: permission denied")}
		svc := media.NewDownloadService(blockingFetcher{}, settings, 1)
		t.Cleanup(func() { drainNow(svc) })
		h := newTestRouter(t, Deps{Downloads: svc, Settings: settings})

		w := do(h, http.MethodPost, "/api/v1/download", `{"url":"https://example.com/v","options":{"type":"audio","format":"mp3"}}`)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Equal(t, "download folder unavailable: /downloads is not writable: permission denied", errorMessage(t, w))
	})
}

func TestSuggestTags(t *testing.T) {
	t.Run("returns guessed tags and known artists", func(t *testing.T) {
		sug := &fakeSuggester{sug: media.Suggestion{
			Tags:         media.Tags{Artist: "Queen", Title: "Bohemian Rhapsody", Album: "A Night At The Opera", Year: 1975},
			KnownArtists: []string{"ABBA", "Queen"},
		}}
		h := newTestRouter(t, Deps{Suggest: sug})

		w := do(h, http.MethodGet, "/api/v1/tags?url="+url.QueryEscape("https://www.youtube.com/watch?v=x&t=1"), "")

		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{
			"tags": {"artist": "Queen", "title": "Bohemian Rhapsody", "album": "A Night At The Opera", "year": 1975},
			"knownArtists": ["ABBA", "Queen"]
		}`, w.Body.String())
		assert.Equal(t, "https://www.youtube.com/watch?v=x&t=1", sug.gotURL)
	})

	t.Run("no known artists is an empty list", func(t *testing.T) {
		h := newTestRouter(t, Deps{Suggest: &fakeSuggester{}})
		w := do(h, http.MethodGet, "/api/v1/tags?url=https://example.com/v", "")
		assert.Contains(t, w.Body.String(), `"knownArtists":[]`)
	})

	t.Run("unavailable videos are 502 with the reason", func(t *testing.T) {
		h := newTestRouter(t, Deps{Suggest: &fakeSuggester{err: fmt.Errorf("%w: Private video", media.ErrVideoUnavailable)}})
		w := do(h, http.MethodGet, "/api/v1/tags?url=https://example.com/v", "")
		assert.Equal(t, http.StatusBadGateway, w.Code)
		assert.Equal(t, "could not read the video: Private video", errorMessage(t, w))
	})

	t.Run("invalid URLs are 400", func(t *testing.T) {
		h := newTestRouter(t, Deps{Suggest: &fakeSuggester{err: media.ErrInvalidURL}})
		w := do(h, http.MethodGet, "/api/v1/tags?url=nope", "")
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestStartDownload_Tags(t *testing.T) {
	t.Run("invalid tags are 400", func(t *testing.T) {
		h := newTestRouter(t, Deps{})
		w := do(h, http.MethodPost, "/api/v1/download",
			`{"url":"https://example.com/v","options":{"type":"audio","format":"mp3"},"tags":{"artist":"a\nb"}}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "tags.artist must not contain control characters", errorMessage(t, w))
	})

	t.Run("tags are accepted", func(t *testing.T) {
		h := newTestRouter(t, Deps{})
		w := do(h, http.MethodPost, "/api/v1/download",
			`{"url":"https://example.com/v","options":{"type":"audio","format":"mp3"},"tags":{"artist":"Queen","title":"Song","album":"","year":1975}}`)
		assert.Equal(t, http.StatusAccepted, w.Code)
	})
}

func TestSearch_Music(t *testing.T) {
	search := &fakeSearcher{videos: []media.Video{{
		ID: "a1", Title: "Bohemian Rhapsody", Uploader: "Queen", URL: "https://music.youtube.com/watch?v=a1",
		Album: "A Night At The Opera",
		Tags:  &media.Tags{Artist: "Queen", Title: "Bohemian Rhapsody", Album: "A Night At The Opera", Year: 1975},
	}}}
	h := newTestRouter(t, Deps{Search: search})

	w := do(h, http.MethodGet, "/api/v1/search/bohemian?source=music&limit=10", "")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, media.SourceMusic, search.gotSource)
	assert.JSONEq(t, `[{
		"id": "a1", "title": "Bohemian Rhapsody", "uploader": "Queen", "duration": 0, "thumbnails": [],
		"url": "https://music.youtube.com/watch?v=a1", "isLive": false, "album": "A Night At The Opera",
		"tags": {"artist": "Queen", "title": "Bohemian Rhapsody", "album": "A Night At The Opera", "year": 1975}
	}]`, w.Body.String())
}

func TestSearch_SourceFailure(t *testing.T) {
	h := newTestRouter(t, Deps{Search: &fakeSearcher{err: fmt.Errorf("%w: Sign in to confirm you’re not a bot.", media.ErrSearchFailed)}})
	w := do(h, http.MethodGet, "/api/v1/search/q?source=music", "")
	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Equal(t, "search failed: Sign in to confirm you’re not a bot.", errorMessage(t, w))
}

func TestSettings_Jellyfin(t *testing.T) {
	t.Run("saving reports the key as set, never its value", func(t *testing.T) {
		s := &fakeSettings{dir: "/music"}
		h := newTestRouter(t, Deps{Settings: s})

		w := do(h, http.MethodPut, "/api/v1/settings", `{"jellyfin": {"url": "http://jellyfin:8096", "apiKey": "s3cret"}}`)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.NotContains(t, w.Body.String(), "s3cret")
		assert.Contains(t, w.Body.String(), `"jellyfin":{"url":"http://jellyfin:8096","apiKeySet":true,"enabled":true}`)
		assert.Equal(t, "/music", s.dir, "the download folder is left alone")
	})

	t.Run("omitting the key keeps it", func(t *testing.T) {
		s := &fakeSettings{jellyfinKey: "old"}
		h := newTestRouter(t, Deps{Settings: s})
		do(h, http.MethodPut, "/api/v1/settings", `{"jellyfin": {"url": "http://new:8096"}}`)
		assert.Equal(t, "old", s.jellyfinKey)
		assert.Equal(t, "http://new:8096", s.jellyfinURL)
	})

	t.Run("invalid URL is 400", func(t *testing.T) {
		h := newTestRouter(t, Deps{})
		w := do(h, http.MethodPut, "/api/v1/settings", `{"jellyfin": {"url": "bad"}}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestJellyfinTest(t *testing.T) {
	check := func(gotURL, gotKey *string, err error) func(context.Context, string, string) (string, error) {
		return func(ctx context.Context, url, key string) (string, error) {
			*gotURL, *gotKey = url, key
			return "Home (Jellyfin 10.11.2)", err
		}
	}

	t.Run("uses the saved key unless one is given", func(t *testing.T) {
		var url, key string
		h := newTestRouter(t, Deps{Settings: &fakeSettings{jellyfinKey: "saved"}, CheckJellyfin: check(&url, &key, nil)})

		w := do(h, http.MethodPost, "/api/v1/settings/jellyfin/test", `{"url": " http://jellyfin:8096 "}`)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"message": "Connected to Home (Jellyfin 10.11.2)"}`, w.Body.String())
		assert.Equal(t, "http://jellyfin:8096", url)
		assert.Equal(t, "saved", key)

		do(h, http.MethodPost, "/api/v1/settings/jellyfin/test", `{"url": "http://j", "apiKey": "typed"}`)
		assert.Equal(t, "typed", key)
	})

	t.Run("failures are 502 with the reason", func(t *testing.T) {
		var url, key string
		h := newTestRouter(t, Deps{Settings: &fakeSettings{}, CheckJellyfin: check(&url, &key, errors.New("Jellyfin rejected the API key"))})
		w := do(h, http.MethodPost, "/api/v1/settings/jellyfin/test", `{"url": "http://j", "apiKey": "k"}`)
		assert.Equal(t, http.StatusBadGateway, w.Code)
		assert.Equal(t, "Jellyfin rejected the API key", errorMessage(t, w))
	})

	t.Run("needs a URL and key", func(t *testing.T) {
		h := newTestRouter(t, Deps{Settings: &fakeSettings{}})
		w := do(h, http.MethodPost, "/api/v1/settings/jellyfin/test", `{"url": "http://j"}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
