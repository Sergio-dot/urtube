package router

import (
	"net/http"
	"time"

	"github.com/Sergio-dot/urtube/internal/download"
	"github.com/Sergio-dot/urtube/internal/handlers"
	"github.com/Sergio-dot/urtube/internal/search"
	"github.com/Sergio-dot/urtube/pkg/httputils"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Dependencies struct {
	Searcher   search.Searcher
	Downloader download.Downloader
}

// NewRouter creates and returns a new HTTP handler with the defined routes.
func NewRouter(deps Dependencies) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.StripSlashes)
	r.Use(middleware.CleanPath)
	r.Use(middleware.Logger)
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
