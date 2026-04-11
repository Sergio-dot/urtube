package router

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sergio-dot/urtube/internal/download"
	"github.com/Sergio-dot/urtube/internal/search"
	"github.com/lrstanley/go-ytdlp"
	"github.com/stretchr/testify/assert"
)

type mockSearcher struct {
	Called bool
}

func (m *mockSearcher) Search(ctx context.Context, param string) ([]*ytdlp.ExtractedInfo, error) {
	m.Called = true
	return nil, search.ErrNoResults
}

type mockDownloader struct {
	Called bool
}

func (m *mockDownloader) Download(ctx context.Context, body *download.DownloadRequest) error {
	m.Called = true
	return nil
}

func TestNewRouter_Routes(t *testing.T) {
	ms := &mockSearcher{}
	md := &mockDownloader{}
	r := NewRouter(Dependencies{
		Searcher:   ms,
		Downloader: md,
	})

	t.Run("GET /api/v1/search/{searchParam}", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/search/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.True(t, ms.Called, "Expected search handler to be invoked")
		assert.Equal(t, http.StatusNotFound, w.Code) // ErrNoResults configured in mock returns 404
	})

	t.Run("POST /api/v1/download", func(t *testing.T) {
		body := []byte(`{"url":"https://example.com","preset_alias":"audio"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/download", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.True(t, md.Called, "Expected download handler to be invoked")
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
