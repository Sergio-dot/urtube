package router

import (
	"bytes"
	"context"
	"embed"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/Sergio-dot/urtube/internal/config"
	"github.com/Sergio-dot/urtube/internal/download"
	"github.com/Sergio-dot/urtube/internal/handlers"
	"github.com/Sergio-dot/urtube/internal/search"
	"github.com/Sergio-dot/urtube/pkg/httputils"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v2"
)

// DownloadManager defines the interface for the download manager.
type DownloadManager interface {
	StartDownload(ctx context.Context, req *download.DownloadRequest) (string, error)
	CancelDownload(uuid string) bool
	Subscribe() chan download.ProgressUpdate
	Unsubscribe(ch chan download.ProgressUpdate)
}

// Dependencies holds the application dependencies.
type Dependencies struct {
	// Searcher is the searcher used to find videos.
	Searcher search.Searcher
	// Manager is the download manager used to start downloads and subscribe to updates.
	Manager DownloadManager
	// Config is the application configuration.
	Config config.Config
	// UI is the filesystem containing the UI assets.
	UI embed.FS
}

// NewRouter creates and returns a new HTTP handler with the defined routes.
func NewRouter(deps Dependencies) http.Handler {
	r := chi.NewRouter()

	var logLevel slog.Level
	if err := logLevel.UnmarshalText([]byte(deps.Config.LogLevel)); err != nil {
		logLevel = slog.LevelInfo
	}

	httpLogger := httplog.NewLogger("urtube", httplog.Options{
		JSON:           deps.Config.JSON,
		LogLevel:       logLevel,
		Concise:        deps.Config.Concise,
		RequestHeaders: deps.Config.RequestHeaders,
	})

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.StripSlashes)
	r.Use(middleware.CleanPath)
	r.Use(httplog.RequestLogger(httpLogger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/healthz"))

	r.Mount("/api/v1", routerV1(deps))

	r.Handle("/*", serveUI(deps.UI))

	return r
}

func serveUI(uiFS embed.FS) http.Handler {
	dist, err := fs.Sub(uiFS, "web/dist")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		})
	}

	fileServer := http.FileServer(http.FS(dist))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if len(path) > 0 && path[0] == '/' {
			path = path[1:]
		}
		if path == "" {
			path = "index.html"
		}

		f, err := dist.Open(path)
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		index, err := dist.Open("index.html")
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

// routerV1 creates and returns a new HTTP handler for the v1 API.
func routerV1(deps Dependencies) http.Handler {
	v1 := chi.NewRouter()

	v1.Get("/health", httputils.MakeHandler(handlers.HealthHandler))
	v1.Get("/events", httputils.MakeHandler((&handlers.EventsHandler{Manager: deps.Manager}).HandleEvents))

	v1.Group(func(r chi.Router) {
		r.Use(middleware.Timeout(time.Second * 30))

		r.Get("/search/{searchParam}", httputils.MakeHandler((&handlers.SearchHandler{Searcher: deps.Searcher}).SearchMedia))
		r.Post("/download", httputils.MakeHandler((&handlers.DownloadHandler{Manager: deps.Manager}).DownloadMedia))
		r.Delete("/download/{uuid}", httputils.MakeHandler((&handlers.DownloadHandler{Manager: deps.Manager}).CancelDownload))
	})

	return v1
}
