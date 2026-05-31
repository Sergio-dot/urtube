package router

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Sergio-dot/urtube/internal/config"
	"github.com/Sergio-dot/urtube/internal/download"
	"github.com/Sergio-dot/urtube/internal/search"
	"github.com/lrstanley/go-ytdlp"
	"github.com/stretchr/testify/assert"
)

type mockSearcher struct {
	Called bool
}

func (m *mockSearcher) Search(ctx context.Context, param string, limit int, wantLiveStreams bool) ([]*ytdlp.ExtractedInfo, error) {
	m.Called = true
	return nil, search.ErrNoResults
}

type mockDownloader struct {
	mu     sync.Mutex
	called bool
}

func (m *mockDownloader) Download(ctx context.Context, body *download.DownloadRequest, onProgress func(download.ProgressUpdate)) error {
	m.mu.Lock()
	m.called = true
	m.mu.Unlock()
	return nil
}

func (m *mockDownloader) isCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.called
}

func TestNewRouter_Routes(t *testing.T) {
	ms := &mockSearcher{}
	md := &mockDownloader{}
	mgr := download.NewDownloadManager(md)

	r := NewRouter(Dependencies{
		Searcher: ms,
		Manager:  mgr,
		Config: config.Config{
			ServerHost:     "localhost",
			ServerPort:     "8080",
			DownloadDir:    "./downloads",
			LogLevel:       "info",
			JSON:           false,
			Concise:        true,
			RequestHeaders: true,
		},
	})

	t.Run("GET /healthz", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "text/plain", w.Header().Get("Content-Type"))
	})

	t.Run("GET /api/v1/search/{searchParam}", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/search/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.True(t, ms.Called, "Expected search handler to be invoked")
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("POST /api/v1/download", func(t *testing.T) {
		body := []byte(`{"url":"https://example.com"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/download", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusAccepted, w.Code)

		// Wait a bit for the async goroutine to set the flag
		assert.Eventually(t, func() bool {
			return md.isCalled()
		}, 1*time.Second, 10*time.Millisecond, "Expected download handler to be invoked through manager")
	})

	t.Run("GET /api/v1/events", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		req = req.WithContext(ctx)

		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	})
}
