// Command urtube serves the urtube web app: YouTube search and downloads
// backed by yt-dlp.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Sergio-dot/urtube/internal/config"
	"github.com/Sergio-dot/urtube/internal/health"
	"github.com/Sergio-dot/urtube/internal/httpapi"
	"github.com/Sergio-dot/urtube/internal/media"
	"github.com/Sergio-dot/urtube/internal/server"
	"github.com/Sergio-dot/urtube/internal/settings"
	"github.com/Sergio-dot/urtube/internal/ui"
	"github.com/Sergio-dot/urtube/internal/ytdlp"
)

// httpShutdownTimeout bounds closing the HTTP server once downloads have
// drained and event streams have ended; only short requests remain by then.
const httpShutdownTimeout = 2 * time.Second

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		if err := healthcheck(); err != nil {
			fmt.Fprintln(os.Stderr, "unhealthy:", err)
			os.Exit(1)
		}
		return
	}

	if err := run(); err != nil {
		slog.Error("urtube stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	slog.SetDefault(newLogger(cfg))
	for _, w := range cfg.Warnings {
		slog.Warn(w)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	deps, err := health.CheckDependencies(ctx)
	if err != nil {
		return err
	}
	// In-progress files have nowhere else to go, so the temp dir must work.
	if err := health.CheckWritableDir(cfg.TempDir); err != nil {
		return err
	}

	// The download folder can be fixed from the app, so a bad one only warns:
	// the UI stays reachable and downloads are refused until it's fixed.
	store, err := settings.Open(cfg.DataDir, cfg.DownloadDir, cfg.DownloadRoot)
	if err != nil {
		slog.Warn("could not load saved settings, using defaults", "error", err)
	}
	downloadDir, err := store.DownloadDir()
	if err != nil {
		slog.Warn("download folder unavailable: downloads will fail until it is fixed or another folder is chosen in Settings",
			"error", err)
	}

	ui, err := ui.Dist()
	if err != nil {
		return fmt.Errorf("load embedded UI: %w", err)
	}

	fetcher := &ytdlp.Fetcher{TempDir: cfg.TempDir}
	if err := fetcher.RemoveStaleTempDirs(); err != nil {
		slog.Warn("could not remove leftover temp files", "error", err)
	}
	downloads := media.NewDownloadService(fetcher, store, cfg.MaxConcurrentDownloads)

	var shuttingDown atomic.Bool
	handler := httpapi.NewRouter(httpapi.Deps{
		Search:       media.NewSearchService(&ytdlp.Searcher{}),
		Downloads:    downloads,
		Settings:     store,
		UI:           ui,
		Config:       *cfg,
		YtdlpVersion: deps.YtdlpVersion,
		Ready:        func() bool { return !shuttingDown.Load() },
	})

	srv, err := server.NewServer(net.JoinHostPort(cfg.HTTPHost, cfg.HTTPPort), handler)
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Start() }()
	slog.Info("ready", "ytdlp", deps.YtdlpVersion, "downloadDir", downloadDir,
		"maxConcurrentDownloads", cfg.MaxConcurrentDownloads)

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}
	// Restore default signal handling: a second signal kills the process.
	stop()

	slog.Info("shutting down", "timeout", cfg.ShutdownTimeout.String())
	shuttingDown.Store(true)

	drainCtx, cancelDrain := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancelDrain()
	downloads.Drain(drainCtx)

	stopCtx, cancelStop := context.WithTimeout(context.Background(), httpShutdownTimeout)
	defer cancelStop()
	if err := srv.Stop(stopCtx); err != nil {
		return fmt.Errorf("stop server: %w", err)
	}

	slog.Info("shutdown complete")
	return <-serveErr
}

// newLogger returns the logger for application and request logs alike: JSON
// for log collectors (the container image's default) or text for terminals.
func newLogger(cfg *config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{Level: cfg.LogLevel}
	if cfg.JSON {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}
