package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Sergio-dot/urtube/internal/download"
	"github.com/Sergio-dot/urtube/pkg/httputils"
)

// EventsDownloadManager defines the interface needed for subscribing to progress updates.
type EventsDownloadManager interface {
	Subscribe() chan download.ProgressUpdate
	Unsubscribe(ch chan download.ProgressUpdate)
}

// defaultKeepAlive is how often an idle event stream sends a comment so that
// reverse proxies (whose idle timeout is often 60s) keep the connection open.
const defaultKeepAlive = 15 * time.Second

// EventsHandler handles Server-Sent Events (SSE) for download progress updates.
type EventsHandler struct {
	// Manager is the download manager used to subscribe to progress updates.
	Manager EventsDownloadManager
	// KeepAlive overrides the keep-alive interval. Zero uses defaultKeepAlive.
	KeepAlive time.Duration
}

// HandleEvents establishes an SSE connection and streams download progress updates.
func (h *EventsHandler) HandleEvents(w http.ResponseWriter, r *http.Request) error {
	if h.Manager == nil {
		return httputils.APIError{StatusCode: http.StatusInternalServerError, Message: "download manager not available"}
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	ch := h.Manager.Subscribe()
	defer h.Manager.Unsubscribe(ch)

	flush := func() {
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}

	// Send the headers right away so the client sees the stream as open.
	w.WriteHeader(http.StatusOK)
	flush()

	keepAlive := h.KeepAlive
	if keepAlive == 0 {
		keepAlive = defaultKeepAlive
	}
	ticker := time.NewTicker(keepAlive)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return nil
		case <-ticker.C:
			if _, err := fmt.Fprint(w, ": keep-alive\n\n"); err != nil {
				return nil
			}
			flush()
		case p, ok := <-ch:
			if !ok {
				return nil
			}

			data, err := json.Marshal(p)
			if err != nil {
				continue
			}

			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return nil
			}
			flush()
		}
	}
}
