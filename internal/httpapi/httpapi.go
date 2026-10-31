// Package httpapi exposes urtube's use cases over HTTP: the JSON API under
// /api/v1, the download event stream, health probes and the embedded UI.
package httpapi

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/Sergio-dot/urtube/internal/config"
	"github.com/Sergio-dot/urtube/internal/media"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v2"
)

// Searcher finds videos. *media.SearchService implements it.
type Searcher interface {
	Search(ctx context.Context, q media.SearchQuery) ([]media.Video, error)
}

// Downloads starts, cancels and reports downloads. *media.DownloadService
// implements it.
type Downloads interface {
	Start(req media.Request) (media.Download, error)
	Cancel(id string) error
	Subscribe() (active []media.Download, updates <-chan media.Download, unsubscribe func())
}

// Suggester suggests tags for a video before it is downloaded.
// *media.SuggestService implements it.
type Suggester interface {
	Suggest(ctx context.Context, url string) (media.Suggestion, error)
}

// Settings reads and changes the settings users edit in the app.
// *settings.Store implements it.
type Settings interface {
	// DownloadDir returns the current download folder and why it can't be
	// used, if it can't.
	DownloadDir() (string, error)
	DefaultDownloadDir() string
	// Root is the folder download folders must be inside; "" allows any.
	Root() string
	// SetDownloadDir changes the download folder; "" restores the default.
	// Validation errors match media.ErrInvalidInput.
	SetDownloadDir(dir string) error
}

// Deps holds what the HTTP layer needs.
type Deps struct {
	Search    Searcher
	Downloads Downloads
	Suggest   Suggester
	Settings  Settings
	// UI is the built frontend, containing index.html. Nil serves 404s.
	UI fs.FS
	// Config configures request logging.
	Config config.Config
	// YtdlpVersion is the yt-dlp version detected at startup.
	YtdlpVersion string
	// Ready reports whether the server should receive traffic. It turns false
	// once shutdown begins. Nil means always ready.
	Ready func() bool
}

// requestTimeout bounds API requests other than the event stream.
const requestTimeout = 30 * time.Second

type api struct {
	Deps
	keepAlive time.Duration
}

// NewRouter returns the HTTP handler for the whole application.
func NewRouter(deps Deps) http.Handler {
	return newRouter(&api{Deps: deps, keepAlive: defaultKeepAlive})
}

func newRouter(a *api) http.Handler {
	r := chi.NewRouter()

	httpLogger := httplog.NewLogger("urtube", httplog.Options{
		JSON:           a.Config.JSON,
		LogLevel:       a.Config.LogLevel,
		Concise:        a.Config.Concise,
		RequestHeaders: a.Config.RequestHeaders,
	})
	// Write request logs through the application's logger so both share one
	// format and destination; httplog's own handler only sets the options.
	httpLogger.Logger = slog.Default().With("component", "http")

	// Probes are answered before the request logger so they don't flood the logs.
	r.Use(middleware.Heartbeat("/livez"))
	r.Use(readiness("/readyz", a.Ready))
	r.Use(middleware.RequestID)
	r.Use(middleware.StripSlashes)
	r.Use(middleware.CleanPath)
	r.Use(httplog.RequestLogger(httpLogger))
	r.Use(middleware.Recoverer)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", handle(a.health))
		// The event stream stays open indefinitely, so it has no timeout.
		r.Get("/events", handle(a.events))

		r.Group(func(r chi.Router) {
			r.Use(middleware.Timeout(requestTimeout))
			r.Get("/search/{query}", handle(a.search))
			r.Get("/tags", handle(a.suggestTags))
			r.Post("/download", handle(a.startDownload))
			r.Delete("/download/{id}", handle(a.cancelDownload))
			r.Get("/settings", handle(a.getSettings))
			r.Put("/settings", handle(a.updateSettings))
		})
	})

	r.Handle("/*", serveUI(a.UI))

	return r
}
