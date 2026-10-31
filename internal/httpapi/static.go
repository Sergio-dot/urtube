package httpapi

import (
	"bytes"
	"io"
	"io/fs"
	"net/http"
	"time"
)

// readiness answers GET/HEAD requests to path with 200 while ready reports
// true, and 503 otherwise, so orchestrators stop routing traffic during
// shutdown.
func readiness(path string, ready func() bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if (r.Method != http.MethodGet && r.Method != http.MethodHead) || r.URL.Path != path {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Content-Type", "text/plain")
			if ready != nil && !ready() {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte("shutting down"))
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("."))
		})
	}
}

// serveUI serves the single-page app from ui, falling back to index.html for
// paths that aren't files so client-side routes work on reload.
func serveUI(ui fs.FS) http.Handler {
	if ui == nil {
		return http.NotFoundHandler()
	}

	fileServer := http.FileServer(http.FS(ui))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if len(path) > 0 && path[0] == '/' {
			path = path[1:]
		}
		if path == "" {
			path = "index.html"
		}

		f, err := ui.Open(path)
		if err == nil {
			_ = f.Close() // only opened to check it exists
			fileServer.ServeHTTP(w, r)
			return
		}

		index, err := ui.Open("index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer index.Close()

		seeker, ok := index.(io.ReadSeeker)
		if !ok {
			data, err := io.ReadAll(index)
			if err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			seeker = bytes.NewReader(data)
		}

		http.ServeContent(w, r, "index.html", time.Now(), seeker)
	})
}
