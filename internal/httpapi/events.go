package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Sergio-dot/urtube/internal/media"
)

// defaultKeepAlive is how often an idle event stream sends a comment so that
// reverse proxies (whose idle timeout is often 60s) keep the connection open.
const defaultKeepAlive = 15 * time.Second

// events handles GET /api/v1/events, a Server-Sent Events stream of download
// state. It first sends every active download, so a client that connects or
// reconnects mid-download catches up, then each change as it happens.
func (a *api) events(w http.ResponseWriter, r *http.Request) error {
	active, updates, unsubscribe := a.Downloads.Subscribe()
	defer unsubscribe()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)

	flush := func() {
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}

	for _, d := range active {
		if err := writeEvent(w, d); err != nil {
			return nil
		}
	}
	// Also sends the headers right away so the client sees the stream as open.
	flush()

	ticker := time.NewTicker(a.keepAlive)
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
		case d, ok := <-updates:
			if !ok {
				// The download service is shutting down.
				return nil
			}
			if err := writeEvent(w, d); err != nil {
				return nil
			}
			flush()
		}
	}
}

func writeEvent(w http.ResponseWriter, d media.Download) error {
	data, err := json.Marshal(newDownloadEvent(d))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", data)
	return err
}
