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
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

type mockDownloader struct {
	DownloadFunc func(ctx context.Context, body *download.DownloadRequest, onProgress func(download.ProgressUpdate)) error
}

func (m *mockDownloader) Download(ctx context.Context, body *download.DownloadRequest, onProgress func(download.ProgressUpdate)) error {
	return m.DownloadFunc(ctx, body, onProgress)
}

// mockManager directly implements the handlers.DownloadManager interface,
// allowing fine-grained control over CancelDownload's return value.
type mockManager struct {
	startFunc  func(ctx context.Context, req *download.DownloadRequest) (string, error)
	cancelFunc func(uuid string) bool
}

func (m *mockManager) StartDownload(ctx context.Context, req *download.DownloadRequest) (string, error) {
	if m.startFunc != nil {
		return m.startFunc(ctx, req)
	}
	return "test-uuid", nil
}

func (m *mockManager) CancelDownload(uuid string) bool {
	if m.cancelFunc != nil {
		return m.cancelFunc(uuid)
	}
	return false
}

// withChiURLParam injects a chi URL parameter into a request context,
// needed because chi.URLParam reads from the route context.
func withChiURLParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
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
		jsonBody, err := json.Marshal(body)
		assert.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/download", bytes.NewBuffer(jsonBody))
		w := httptest.NewRecorder()

		err = h.DownloadMedia(w, req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusAccepted, w.Code)
		
		var resp map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
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
		jsonBody, err := json.Marshal(body)
		assert.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/download", bytes.NewBuffer(jsonBody))
		w := httptest.NewRecorder()

		err = h.DownloadMedia(w, req)

		assert.Error(t, err)
		apiErr, ok := err.(httputils.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusInternalServerError, apiErr.StatusCode)
		assert.Equal(t, "download manager not available", apiErr.Message)
	})
}
func TestCancelDownload(t *testing.T) {
	t.Run("successful cancel returns 200", func(t *testing.T) {
		h := &DownloadHandler{Manager: &mockManager{
			cancelFunc: func(uuid string) bool { return true },
		}}

		req := httptest.NewRequest(http.MethodDelete, "/download/some-uuid", nil)
		req = withChiURLParam(req, "uuid", "some-uuid")
		w := httptest.NewRecorder()

		err := h.CancelDownload(w, req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]string
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "some-uuid", resp["uuid"])
		assert.Equal(t, "download cancelled", resp["message"])
	})

	t.Run("unknown uuid returns 404", func(t *testing.T) {
		h := &DownloadHandler{Manager: &mockManager{
			cancelFunc: func(uuid string) bool { return false },
		}}

		req := httptest.NewRequest(http.MethodDelete, "/download/ghost-uuid", nil)
		req = withChiURLParam(req, "uuid", "ghost-uuid")
		w := httptest.NewRecorder()

		err := h.CancelDownload(w, req)

		assert.Error(t, err)
		apiErr, ok := err.(httputils.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
		assert.Contains(t, apiErr.Message, "ghost-uuid")
	})

	t.Run("missing uuid param returns 400", func(t *testing.T) {
		h := &DownloadHandler{Manager: &mockManager{}}

		req := httptest.NewRequest(http.MethodDelete, "/download/", nil)
		// No chi URL param injected — uuid will be empty string
		w := httptest.NewRecorder()

		err := h.CancelDownload(w, req)

		assert.Error(t, err)
		apiErr, ok := err.(httputils.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	})

	t.Run("nil manager returns 500", func(t *testing.T) {
		h := &DownloadHandler{Manager: nil}

		req := httptest.NewRequest(http.MethodDelete, "/download/some-uuid", nil)
		req = withChiURLParam(req, "uuid", "some-uuid")
		w := httptest.NewRecorder()

		err := h.CancelDownload(w, req)

		assert.Error(t, err)
		apiErr, ok := err.(httputils.APIError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusInternalServerError, apiErr.StatusCode)
		assert.Equal(t, "download manager not available", apiErr.Message)
	})
}
