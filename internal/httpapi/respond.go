package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Sergio-dot/urtube/internal/media"
)

// handlerFunc is an HTTP handler that returns an error instead of writing it.
type handlerFunc func(w http.ResponseWriter, r *http.Request) error

// handle adapts h to http.HandlerFunc, turning a returned error into a JSON
// error response.
func handle(h handlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			status, msg := errorStatus(err)
			if status == http.StatusInternalServerError {
				slog.Error("request failed", "path", r.URL.Path, "error", err)
			}
			writeJSON(w, status, errorResponse{Error: msg})
		}
	}
}

// statusError is a transport-level error, such as a malformed body, with the
// status and client-safe message to respond with.
type statusError struct {
	status int
	msg    string
}

func (e statusError) Error() string { return e.msg }

// errorStatus maps an error to the HTTP status and message sent to the
// client. Unknown errors become a generic 500 so internals never leak.
func errorStatus(err error) (int, string) {
	var se statusError
	switch {
	case errors.As(err, &se):
		return se.status, se.msg
	case errors.Is(err, media.ErrInvalidInput):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, media.ErrNotFound), errors.Is(err, media.ErrNoResults):
		return http.StatusNotFound, err.Error()
	case errors.Is(err, media.ErrDraining), errors.Is(err, media.ErrDownloadDirUnavailable):
		return http.StatusServiceUnavailable, err.Error()
	case errors.Is(err, media.ErrVideoUnavailable), errors.Is(err, media.ErrSearchFailed):
		// The message is yt-dlp's explanation, e.g. "Private video".
		return http.StatusBadGateway, err.Error()
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}

// writeJSON writes data as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data any) {
	buf, err := json.Marshal(data)
	if err != nil {
		slog.Error("failed to marshal json", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(buf); err != nil {
		slog.Error("failed to write json", "error", err)
	}
}
