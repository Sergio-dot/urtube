package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Sergio-dot/urtube/internal/download"
)

type EventsHandler struct {
	Manager *download.DownloadManager
}

func (h *EventsHandler) HandleEvents(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	ch := h.Manager.Subscribe()
	defer h.Manager.Unsubscribe(ch)

	for {
		select {
		case <-r.Context().Done():
			return nil
		case p, ok := <-ch:
			if !ok {
				return nil
			}

			data, err := json.Marshal(p)
			if err != nil {
				continue
			}

			fmt.Fprintf(w, "data: %s\n\n", data)

			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}
