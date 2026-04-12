package router

import (
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

type Dependencies struct {
	Searcher   search.Searcher
	Downloader download.Downloader
	Config     config.Config
}

// NewRouter creates and returns a new HTTP handler with the defined routes.
func NewRouter(deps Dependencies) http.Handler {
	r := chi.NewRouter()

	logLevel := slog.LevelInfo
	switch strings.ToLower(deps.Config.LogLevel) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}

	logger := httplog.NewLogger("urtube", httplog.Options{
		JSON:           deps.Config.JSON,
		LogLevel:       logLevel,
		Concise:        deps.Config.Concise,
		RequestHeaders: deps.Config.RequestHeaders,
	})

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.StripSlashes)
	r.Use(middleware.CleanPath)
	r.Use(httplog.RequestLogger(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(time.Second * 30))
	r.Use(middleware.Heartbeat("/healthz"))

	r.Mount("/api/v1", routerV1(deps))

	return r
}

// routerV1 creates and returns a new HTTP handler for the v1 API.
func routerV1(deps Dependencies) http.Handler {
	v1 := chi.NewRouter()

	v1.Get("/search/{searchParam}", httputils.MakeHandler((&handlers.SearchHandler{Searcher: deps.Searcher}).SearchVideo))
	v1.Post("/download", httputils.MakeHandler((&handlers.DownloadHandler{Downloader: deps.Downloader}).DownloadVideo))

	return v1
}
