package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/Sergio-dot/urtube/internal/config"
	"github.com/Sergio-dot/urtube/internal/download"
	"github.com/Sergio-dot/urtube/internal/router"
	"github.com/Sergio-dot/urtube/internal/search"
	"github.com/Sergio-dot/urtube/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("server creation failed", "error", err)
		os.Exit(1)
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		slog.Error("invalid log level", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: false,
		Level:     level,
	}))

	slog.SetDefault(logger)

	router := router.NewRouter(router.Dependencies{
		Searcher:   &search.YtdlpSearcher{},
		Downloader: &download.YtdlpDownloader{DownloadDir: cfg.DownloadDir},
		Config:     *cfg,
	})

	srv, err := server.NewServer(fmt.Sprintf("%s:%s", cfg.ServerHost, cfg.ServerPort), router)
	if err != nil {
		slog.Error("server creation failed", "error", err)
		os.Exit(1)
	}

	err = srv.Start()
	if err != nil {
		slog.Error("server failed to start", "error", err)
		os.Exit(1)
	}
}
