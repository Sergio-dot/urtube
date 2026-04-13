package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sergio-dot/urtube/internal/download"
	"github.com/Sergio-dot/urtube/pkg/httputils"
	"github.com/stretchr/testify/assert"
)

type mockDownloader struct {
	MockDownload func(ctx context.Context, body *download.DownloadRequest) error
}

func (m *mockDownloader) Download(ctx context.Context, body *download.DownloadRequest) error {
	return m.MockDownload(ctx, body)
}

func TestDownloadMedia(t *testing.T) {
	t.Run("successful download request", func(t *testing.T) {
		mock := &mockDownloader{
			MockDownload: func(ctx context.Context, body *download.DownloadRequest) error {
				return nil
			},
		}
		h := &DownloadHandler{Downloader: mock}

		body := download.DownloadRequest{
			URL: "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		}
		jsonBody, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/download", bytes.NewBuffer(jsonBody))
		w := httptest.NewRecorder()

		err := h.DownloadMedia(w, req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"message":"download successful"}`, w.Body.String())
	})

	t.Run("invalid request body", func(t *testing.T) {
		h := &DownloadHandler{Downloader: &mockDownloader{}}

		req := httptest.NewRequest(http.MethodPost, "/download", bytes.NewBufferString("invalid json"))
		w := httptest.NewRecorder()

		err := h.DownloadMedia(w, req)

		assert.Error(t, err)
		apiErr, ok := err.(httputils.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	})

	t.Run("missing url triggers validation error", func(t *testing.T) {
		mock := &mockDownloader{
			MockDownload: func(ctx context.Context, body *download.DownloadRequest) error {
				return nil
			},
		}
		h := &DownloadHandler{Downloader: mock}

		body := download.DownloadRequest{}
		jsonBody, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/download", bytes.NewBuffer(jsonBody))
		w := httptest.NewRecorder()

		err := h.DownloadMedia(w, req)

		assert.Error(t, err)
		apiErr, ok := err.(httputils.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
		assert.Equal(t, "url is required", apiErr.Message)
	})

	t.Run("downloader returns error", func(t *testing.T) {
		mock := &mockDownloader{
			MockDownload: func(ctx context.Context, body *download.DownloadRequest) error {
				return errors.New("mock failed")
			},
		}
		h := &DownloadHandler{Downloader: mock}

		body := download.DownloadRequest{
			URL: "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		}
		jsonBody, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/download", bytes.NewBuffer(jsonBody))
		w := httptest.NewRecorder()

		err := h.DownloadMedia(w, req)

		assert.Error(t, err)
		apiErr, ok := err.(httputils.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusInternalServerError, apiErr.StatusCode)
		assert.Equal(t, "failed to download video: mock failed", apiErr.Message)
	})

	t.Run("nil downloader dependency", func(t *testing.T) {
		h := &DownloadHandler{Downloader: nil}

		body := download.DownloadRequest{URL: "https://www.youtube.com/watch?v=dQw4w9WgXcQ"}
		jsonBody, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/download", bytes.NewBuffer(jsonBody))
		w := httptest.NewRecorder()

		err := h.DownloadMedia(w, req)

		assert.Error(t, err)
		apiErr, ok := err.(httputils.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusInternalServerError, apiErr.StatusCode)
		assert.Equal(t, "downloader not available", apiErr.Message)
	})
}
