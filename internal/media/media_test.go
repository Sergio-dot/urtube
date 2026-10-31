package media

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRequestValidate(t *testing.T) {
	validURL := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	videoMP4 := Options{Type: MediaVideo, Format: "mp4"}

	tests := []struct {
		name          string
		url           string
		options       Options
		expectedError error
	}{
		{name: "URL is required", url: "", options: videoMP4, expectedError: ErrURLRequired},
		{name: "URL must not be a flag", url: "--batch-file=/etc/passwd", options: videoMP4, expectedError: ErrInvalidURL},
		{name: "URL must be http or https", url: "file:///etc/passwd", options: videoMP4, expectedError: ErrInvalidURL},
		{name: "URL must have a host", url: "https://", options: videoMP4, expectedError: ErrInvalidURL},
		{name: "options are required", url: validURL, expectedError: ErrInvalidMediaType},
		{name: "unknown media type", url: validURL, options: Options{Type: "exec", Format: "mp4"}, expectedError: ErrInvalidMediaType},
		{name: "format not allowed for type", url: validURL, options: Options{Type: MediaAudio, Format: "mp4"}, expectedError: ErrInvalidFormat},
		{name: "valid video request", url: validURL, options: videoMP4},
		{name: "valid audio request", url: "http://example.com/v", options: Options{Type: MediaAudio, Format: "mp3"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Request{URL: tt.url, Options: tt.options}.Validate()
			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
				assert.ErrorIs(t, err, ErrInvalidInput, "validation errors must match ErrInvalidInput")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProgressPercent(t *testing.T) {
	assert.Equal(t, 0.0, Progress{DownloadedBytes: 10}.Percent(), "unknown total")
	assert.Equal(t, 25.0, Progress{DownloadedBytes: 25, TotalBytes: 100}.Percent())
	assert.Equal(t, 100.0, Progress{DownloadedBytes: 120, TotalBytes: 100}.Percent(), "estimates can overshoot")
}

type fakeSource struct {
	videos   []Video
	err      error
	gotLimit int
}

func (f *fakeSource) Search(ctx context.Context, query string, limit int, includeLive bool) ([]Video, error) {
	f.gotLimit = limit
	return f.videos, f.err
}

func TestSearchService(t *testing.T) {
	t.Run("limits", func(t *testing.T) {
		tests := []struct {
			name     string
			limit    int
			expected int
		}{
			{name: "zero uses default", limit: 0, expected: DefaultSearchLimit},
			{name: "negative uses default", limit: -3, expected: DefaultSearchLimit},
			{name: "within range is kept", limit: 20, expected: 20},
			{name: "above max is capped", limit: 100000, expected: MaxSearchLimit},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				src := &fakeSource{videos: []Video{{ID: "a"}}}
				_, err := NewSearchService(src).Search(context.Background(), "q", tt.limit, false)
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, src.gotLimit)
			})
		}
	})

	t.Run("requires a query", func(t *testing.T) {
		_, err := NewSearchService(&fakeSource{}).Search(context.Background(), "  ", 5, false)
		assert.ErrorIs(t, err, ErrQueryRequired)
	})

	t.Run("no videos is ErrNoResults", func(t *testing.T) {
		_, err := NewSearchService(&fakeSource{}).Search(context.Background(), "q", 5, false)
		assert.ErrorIs(t, err, ErrNoResults)
	})

	t.Run("source errors are returned", func(t *testing.T) {
		boom := errors.New("boom")
		_, err := NewSearchService(&fakeSource{err: boom}).Search(context.Background(), "q", 5, false)
		assert.ErrorIs(t, err, boom)
	})
}

// fakeFetcher runs fetch for every Fetch call.
type fakeFetcher struct {
	fetch func(ctx context.Context, req Request, onProgress func(Progress)) error
}

func (f *fakeFetcher) Fetch(ctx context.Context, req Request, onProgress func(Progress)) error {
	if f.fetch == nil {
		return nil
	}
	return f.fetch(ctx, req, onProgress)
}

// blockingFetcher returns a fetcher that blocks until release is closed or
// its context is cancelled.
func blockingFetcher(release <-chan struct{}) *fakeFetcher {
	return &fakeFetcher{fetch: func(ctx context.Context, req Request, onProgress func(Progress)) error {
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}}
}

var validRequest = Request{
	URL:     "https://example.com/v",
	VideoID: "v1",
	Title:   "Song",
	Options: Options{Type: MediaAudio, Format: "mp3"},
}

// next returns the next update for id, skipping other downloads' updates.
func next(t *testing.T, updates <-chan Download, id string) Download {
	t.Helper()
	timeout := time.After(2 * time.Second)
	for {
		select {
		case d, ok := <-updates:
			if !ok {
				t.Fatal("updates channel closed")
			}
			if d.ID == id {
				return d
			}
		case <-timeout:
			t.Fatalf("timed out waiting for an update for %s", id)
		}
	}
}

// statusesUntilTerminal collects id's statuses up to and including a terminal one.
func statusesUntilTerminal(t *testing.T, updates <-chan Download, id string) []Status {
	t.Helper()
	var statuses []Status
	for {
		d := next(t, updates, id)
		if len(statuses) == 0 || statuses[len(statuses)-1] != d.Status {
			statuses = append(statuses, d.Status)
		}
		if d.Status.Terminal() {
			return statuses
		}
	}
}

func TestDownloadService_Lifecycle(t *testing.T) {
	t.Run("rejects invalid requests", func(t *testing.T) {
		svc := NewDownloadService(&fakeFetcher{}, 1)
		_, err := svc.Start(Request{URL: "--exec=id"})
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("successful download reports progress and finishes", func(t *testing.T) {
		svc := NewDownloadService(&fakeFetcher{fetch: func(ctx context.Context, req Request, onProgress func(Progress)) error {
			onProgress(Progress{DownloadedBytes: 50, TotalBytes: 100})
			return nil
		}}, 1)
		_, updates, unsubscribe := svc.Subscribe()
		defer unsubscribe()

		d, err := svc.Start(validRequest)
		assert.NoError(t, err)
		assert.Equal(t, StatusQueued, d.Status)
		assert.Equal(t, "v1", d.VideoID)

		var sawProgress bool
		var statuses []Status
		for {
			u := next(t, updates, d.ID)
			sawProgress = sawProgress || u.Progress.Percent() == 50
			statuses = append(statuses, u.Status)
			if u.Status.Terminal() {
				break
			}
		}
		assert.True(t, sawProgress)
		assert.Equal(t, StatusQueued, statuses[0])
		assert.Contains(t, statuses, StatusDownloading)
		assert.Equal(t, StatusFinished, statuses[len(statuses)-1])
	})

	t.Run("failed download reports the error", func(t *testing.T) {
		svc := NewDownloadService(&fakeFetcher{fetch: func(ctx context.Context, req Request, onProgress func(Progress)) error {
			return errors.New("HTTP Error 403: Forbidden")
		}}, 1)
		_, updates, unsubscribe := svc.Subscribe()
		defer unsubscribe()

		d, err := svc.Start(validRequest)
		assert.NoError(t, err)

		var last Download
		for !last.Status.Terminal() {
			last = next(t, updates, d.ID)
		}
		assert.Equal(t, StatusError, last.Status)
		assert.Equal(t, "HTTP Error 403: Forbidden", last.Error)
	})

	t.Run("cancel stops a running download", func(t *testing.T) {
		svc := NewDownloadService(blockingFetcher(make(chan struct{})), 1)
		_, updates, unsubscribe := svc.Subscribe()
		defer unsubscribe()

		d, err := svc.Start(validRequest)
		assert.NoError(t, err)
		assert.Equal(t, StatusQueued, next(t, updates, d.ID).Status)
		assert.Equal(t, StatusDownloading, next(t, updates, d.ID).Status)

		assert.NoError(t, svc.Cancel(d.ID))
		assert.Equal(t, StatusCancelled, next(t, updates, d.ID).Status)
	})

	t.Run("cancel of unknown or ended download is ErrNotFound", func(t *testing.T) {
		svc := NewDownloadService(&fakeFetcher{}, 1)
		assert.ErrorIs(t, svc.Cancel("nope"), ErrNotFound)

		_, updates, unsubscribe := svc.Subscribe()
		defer unsubscribe()
		d, _ := svc.Start(validRequest)
		statusesUntilTerminal(t, updates, d.ID)
		assert.ErrorIs(t, svc.Cancel(d.ID), ErrNotFound)
	})
}

func TestDownloadService_Concurrency(t *testing.T) {
	release := make(chan struct{})
	svc := NewDownloadService(blockingFetcher(release), 1)
	_, updates, unsubscribe := svc.Subscribe()
	defer unsubscribe()

	first, _ := svc.Start(validRequest)
	assert.Equal(t, StatusQueued, next(t, updates, first.ID).Status)
	assert.Equal(t, StatusDownloading, next(t, updates, first.ID).Status)

	second, _ := svc.Start(validRequest)
	assert.Equal(t, StatusQueued, next(t, updates, second.ID).Status)
	third, _ := svc.Start(validRequest)
	assert.Equal(t, StatusQueued, next(t, updates, third.ID).Status)

	t.Run("a queued download can be cancelled without running", func(t *testing.T) {
		assert.NoError(t, svc.Cancel(third.ID))
		assert.Equal(t, StatusCancelled, next(t, updates, third.ID).Status)
	})

	t.Run("the next queued download starts when a slot frees up", func(t *testing.T) {
		select {
		case d := <-updates:
			t.Fatalf("unexpected update while the slot is busy: %+v", d)
		case <-time.After(50 * time.Millisecond):
		}

		close(release)
		assert.Equal(t, StatusFinished, next(t, updates, first.ID).Status)
		assert.Equal(t, []Status{StatusDownloading, StatusFinished}, statusesUntilTerminal(t, updates, second.ID))
	})
}

func TestDownloadService_Subscribe(t *testing.T) {
	t.Run("returns active downloads oldest first", func(t *testing.T) {
		svc := NewDownloadService(blockingFetcher(make(chan struct{})), 1)
		a, _ := svc.Start(validRequest)
		b, _ := svc.Start(validRequest)

		active, _, unsubscribe := svc.Subscribe()
		defer unsubscribe()

		assert.Len(t, active, 2)
		assert.Equal(t, a.ID, active[0].ID)
		assert.Equal(t, b.ID, active[1].ID)

		// An already cancelled context makes Drain cancel the blocked download.
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		svc.Drain(ctx)
	})

	t.Run("unsubscribe closes the channel and is idempotent", func(t *testing.T) {
		svc := NewDownloadService(&fakeFetcher{}, 1)
		_, updates, unsubscribe := svc.Subscribe()
		unsubscribe()
		unsubscribe()
		_, ok := <-updates
		assert.False(t, ok)
	})

	t.Run("progress is dropped for a full subscriber, terminal updates are not", func(t *testing.T) {
		svc := NewDownloadService(&fakeFetcher{}, 1)
		_, updates, unsubscribe := svc.Subscribe()
		defer unsubscribe()

		svc.mu.Lock()
		for range subscriberBuffer + 5 {
			svc.publishLocked(Download{ID: "a", Status: StatusDownloading})
		}
		svc.publishLocked(Download{ID: "a", Status: StatusFinished})
		svc.mu.Unlock()

		assert.Len(t, updates, subscriberBuffer)
		var last Download
		for len(updates) > 0 {
			last = <-updates
		}
		assert.Equal(t, StatusFinished, last.Status)
	})
}

func TestDownloadService_Drain(t *testing.T) {
	// collect reads updates until the channel is closed.
	collect := func(t *testing.T, updates <-chan Download) map[string][]Status {
		t.Helper()
		got := map[string][]Status{}
		timeout := time.After(2 * time.Second)
		for {
			select {
			case d, ok := <-updates:
				if !ok {
					return got
				}
				got[d.ID] = append(got[d.ID], d.Status)
			case <-timeout:
				t.Fatal("timed out waiting for the updates channel to close")
			}
		}
	}

	t.Run("waits for running downloads, cancels queued ones, closes subscribers", func(t *testing.T) {
		release := make(chan struct{})
		svc := NewDownloadService(blockingFetcher(release), 1)
		_, updates, _ := svc.Subscribe()

		running, _ := svc.Start(validRequest)
		queued, _ := svc.Start(validRequest)

		// Wait until the first download holds the only slot.
		got := map[string][]Status{}
		for len(got[running.ID]) == 0 || got[running.ID][len(got[running.ID])-1] != StatusDownloading {
			d := <-updates
			got[d.ID] = append(got[d.ID], d.Status)
		}

		drained := make(chan struct{})
		go func() {
			svc.Drain(context.Background())
			close(drained)
		}()

		select {
		case <-drained:
			t.Fatal("Drain returned while a download was still running")
		case <-time.After(50 * time.Millisecond):
		}
		close(release)

		for id, statuses := range collect(t, updates) {
			got[id] = append(got[id], statuses...)
		}
		<-drained
		assert.Equal(t, StatusFinished, got[running.ID][len(got[running.ID])-1])
		assert.Equal(t, []Status{StatusQueued, StatusCancelled}, got[queued.ID])
	})

	t.Run("cancels downloads still running at the deadline", func(t *testing.T) {
		svc := NewDownloadService(blockingFetcher(make(chan struct{})), 1)
		_, updates, _ := svc.Subscribe()
		d, _ := svc.Start(validRequest)

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()
		svc.Drain(ctx)

		got := collect(t, updates)
		assert.Equal(t, StatusCancelled, got[d.ID][len(got[d.ID])-1])
	})

	t.Run("rejects downloads and subscriptions afterwards", func(t *testing.T) {
		svc := NewDownloadService(&fakeFetcher{}, 1)
		svc.Drain(context.Background())

		_, err := svc.Start(validRequest)
		assert.ErrorIs(t, err, ErrDraining)

		_, updates, _ := svc.Subscribe()
		_, ok := <-updates
		assert.False(t, ok, "Subscribe after Drain must return a closed channel")
	})
}

func TestDownloadService_StartsQueuedDownloadsInOrder(t *testing.T) {
	release := make(chan struct{})
	svc := NewDownloadService(blockingFetcher(release), 1)
	_, updates, unsubscribe := svc.Subscribe()
	defer unsubscribe()

	var ids []string
	for range 4 {
		d, err := svc.Start(validRequest)
		assert.NoError(t, err)
		ids = append(ids, d.ID)
	}
	close(release)

	var started []string
	timeout := time.After(2 * time.Second)
	for len(started) < len(ids) {
		select {
		case d := <-updates:
			if d.Status == StatusDownloading {
				started = append(started, d.ID)
			}
		case <-timeout:
			t.Fatalf("only %d of %d downloads started", len(started), len(ids))
		}
	}
	assert.Equal(t, ids, started)
}
