package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Sergio-dot/urtube/internal/download"
	"github.com/Sergio-dot/urtube/pkg/httputils"
	"github.com/stretchr/testify/assert"
)

func TestHandleEvents_NilManager(t *testing.T) {
	h := &EventsHandler{Manager: nil}
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	w := httptest.NewRecorder()

	err := h.HandleEvents(w, req)
	assert.Error(t, err)

	apiErr, ok := err.(httputils.APIError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusInternalServerError, apiErr.StatusCode)
	assert.Equal(t, "download manager not available", apiErr.Message)
}

func TestHandleEvents_StreamsUpdatesAndCancels(t *testing.T) {
	md := &mockDownloader{
		DownloadFunc: func(ctx context.Context, body *download.DownloadRequest, onProgress func(download.ProgressUpdate)) error {
			return nil
		},
	}
	mgr := download.NewDownloadManager(md)
	h := &EventsHandler{Manager: mgr}

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	ctx, cancel := context.WithCancel(context.Background())
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = h.HandleEvents(w, req)
	}()

	// Wait for connection to start and subscribe
	time.Sleep(10 * time.Millisecond)

	// Broadcast an update
	ch := mgr.Subscribe()
	defer mgr.Unsubscribe(ch)

	mgr.StartDownload(context.Background(), &download.DownloadRequest{URL: "test"})

	// Give the broadcast a moment to be processed by HandleEvents
	time.Sleep(20 * time.Millisecond)

	cancel()
	wg.Wait()

	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), `data: {`)
	assert.Contains(t, w.Body.String(), `"status":"downloading"`)
}
