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
	"github.com/Sergio-dot/urtube/internal/ytdlp"
	"github.com/Sergio-dot/urtube/web"
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

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: false,
		Level:     cfg.LogLevel,
	}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	deps, err := health.CheckDependencies(ctx)
	if err != nil {
		return err
	}
	for _, dir := range []string{cfg.DownloadDir, cfg.TempDir} {
		if err := health.CheckWritableDir(dir); err != nil {
			return err
		}
	}

	ui, err := web.Dist()
	if err != nil {
		return fmt.Errorf("load embedded UI: %w", err)
	}

	fetcher := &ytdlp.Fetcher{DownloadDir: cfg.DownloadDir, TempDir: cfg.TempDir}
	if err := fetcher.RemoveStaleTempDirs(); err != nil {
		slog.Warn("could not remove leftover temp files", "error", err)
	}
	downloads := media.NewDownloadService(fetcher, cfg.MaxConcurrentDownloads)

	var shuttingDown atomic.Bool
	handler := httpapi.NewRouter(httpapi.Deps{
		Search:       media.NewSearchService(&ytdlp.Searcher{}),
		Downloads:    downloads,
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
	slog.Info("ready", "ytdlp", deps.YtdlpVersion, "downloadDir", cfg.DownloadDir,
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
