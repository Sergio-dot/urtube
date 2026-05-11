package router

import (
	"embed"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
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

// Dependencies holds the application dependencies.
type Dependencies struct {
	Searcher   search.Searcher
	Downloader download.Downloader
	Config     config.Config
	UI         embed.FS
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
	r.Use(middleware.Timeout(time.Second * 30))
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
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		_, err := fs.Stat(dist, path)
		if err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		index, err := dist.Open("index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer index.Close()
		http.ServeContent(w, r, "index.html", time.Now(), index.(io.ReadSeeker))
	})
}

// routerV1 creates and returns a new HTTP handler for the v1 API.
func routerV1(deps Dependencies) http.Handler {
	v1 := chi.NewRouter()

	v1.Get("/health", httputils.MakeHandler(handlers.HealthHandler))
	v1.Get("/search/{searchParam}", httputils.MakeHandler((&handlers.SearchHandler{Searcher: deps.Searcher}).SearchMedia))
	v1.Post("/download", httputils.MakeHandler((&handlers.DownloadHandler{Downloader: deps.Downloader}).DownloadMedia))

	return v1
}
