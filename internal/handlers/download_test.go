package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sergio-dot/urtube/internal/download"
	"github.com/Sergio-dot/urtube/pkg/httputils"
	"github.com/stretchr/testify/assert"
)

type mockDownloader struct {
	DownloadFunc func(ctx context.Context, body *download.DownloadRequest, onProgress func(download.ProgressUpdate)) error
}

func (m *mockDownloader) Download(ctx context.Context, body *download.DownloadRequest, onProgress func(download.ProgressUpdate)) error {
	return m.DownloadFunc(ctx, body, onProgress)
}

func TestDownloadMedia(t *testing.T) {
	t.Run("successful async download request", func(t *testing.T) {
		mock := &mockDownloader{
			DownloadFunc: func(ctx context.Context, body *download.DownloadRequest, onProgress func(download.ProgressUpdate)) error {
				return nil
			},
		}
		mgr := download.NewDownloadManager(mock)
		h := &DownloadHandler{Manager: mgr}

		body := download.DownloadRequest{
			URL: "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		}
		jsonBody, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/download", bytes.NewBuffer(jsonBody))
		w := httptest.NewRecorder()

		err := h.DownloadMedia(w, req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusAccepted, w.Code)
		
		var resp map[string]string
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NotEmpty(t, resp["uuid"])
		assert.Equal(t, "download started", resp["message"])
	})

	t.Run("invalid request body", func(t *testing.T) {
		h := &DownloadHandler{Manager: download.NewDownloadManager(nil)}

		req := httptest.NewRequest(http.MethodPost, "/download", bytes.NewBufferString("invalid json"))
		w := httptest.NewRecorder()

		err := h.DownloadMedia(w, req)

		assert.Error(t, err)
		apiErr, ok := err.(httputils.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	})

	t.Run("nil manager dependency", func(t *testing.T) {
		h := &DownloadHandler{Manager: nil}

		body := download.DownloadRequest{URL: "https://www.youtube.com/watch?v=dQw4w9WgXcQ"}
		jsonBody, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/download", bytes.NewBuffer(jsonBody))
		w := httptest.NewRecorder()

		err := h.DownloadMedia(w, req)

		assert.Error(t, err)
		apiErr, ok := err.(httputils.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusInternalServerError, apiErr.StatusCode)
		assert.Equal(t, "download manager not available", apiErr.Message)
	})
}
