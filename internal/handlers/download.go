package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Sergio-dot/urtube/internal/download"
	"github.com/Sergio-dot/urtube/pkg/httputils"
)

// DownloadHandler is the handler for the download video request
type DownloadHandler struct {
	Downloader download.Downloader
}

// DownloadVideo handles the download video request
func (h *DownloadHandler) DownloadVideo(w http.ResponseWriter, r *http.Request) error {
	if h.Downloader == nil {
		return httputils.APIError{StatusCode: http.StatusInternalServerError, Message: "downloader not available"}
	}

	body := &download.DownloadRequest{}
	if err := json.NewDecoder(r.Body).Decode(body); err != nil {
		return httputils.APIError{StatusCode: http.StatusBadRequest, Message: "invalid request body"}
	}

	if err := body.Validate(); err != nil {
		return httputils.APIError{StatusCode: http.StatusBadRequest, Message: err.Error()}
	}

	if err := h.Downloader.Download(r.Context(), body); err != nil {
		return httputils.APIError{StatusCode: http.StatusInternalServerError, Message: "failed to download video: " + err.Error()}
	}

	httputils.WriteJSON(w, http.StatusOK, map[string]string{"message": "download successful"})
	return nil
}
