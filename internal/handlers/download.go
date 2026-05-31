package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Sergio-dot/urtube/internal/download"
	"github.com/Sergio-dot/urtube/pkg/httputils"
)

// DownloadManager defines the interface needed for starting and cancelling downloads.
type DownloadManager interface {
	StartDownload(ctx context.Context, req *download.DownloadRequest) (string, error)
	CancelDownload(uuid string) bool
}

// DownloadHandler is the handler for the download video request.
type DownloadHandler struct {
	// Manager is the download manager used to start downloads.
	Manager DownloadManager
}

// DownloadMedia handles the download media request.
func (h *DownloadHandler) DownloadMedia(w http.ResponseWriter, r *http.Request) error {
	if h.Manager == nil {
		return httputils.APIError{StatusCode: http.StatusInternalServerError, Message: "download manager not available"}
	}

	var body download.DownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return httputils.APIError{StatusCode: http.StatusBadRequest, Message: "invalid request body"}
	}

	if err := body.Validate(); err != nil {
		return httputils.APIError{StatusCode: http.StatusBadRequest, Message: err.Error()}
	}

	downloadUUID, err := h.Manager.StartDownload(r.Context(), &body)
	if err != nil {
		return httputils.APIError{StatusCode: http.StatusInternalServerError, Message: "failed to download video: " + err.Error()}
	}

	httputils.WriteJSON(w, http.StatusAccepted, map[string]string{
		"uuid":    downloadUUID,
		"message": "download started",
	})

	return nil
}
