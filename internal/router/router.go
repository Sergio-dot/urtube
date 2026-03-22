package router

import (
	"net/http"
	"time"

	"github.com/Sergio-dot/urtube/internal/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.StripSlashes)
	r.Use(middleware.CleanPath)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(time.Second * 300))
	r.Use(middleware.Heartbeat("/"))

	r.Mount("/api/v1", routerV1())

	return r
}

func routerV1() http.Handler {
	v1 := chi.NewRouter()

	v1.Get("/search/{searchParam}", handlers.MakeHandler(handlers.SearchVideo))

	return v1
}
