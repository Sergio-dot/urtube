package main

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Sergio-dot/urtube/internal/config"
	"github.com/Sergio-dot/urtube/internal/download"
	"github.com/Sergio-dot/urtube/internal/health"
	"github.com/Sergio-dot/urtube/internal/router"
	"github.com/Sergio-dot/urtube/internal/search"
	"github.com/Sergio-dot/urtube/internal/server"
)

//go:embed all:web/dist
var uiFS embed.FS

// httpShutdownTimeout bounds closing the HTTP server once downloads have
// drained and event streams have ended; only short requests remain by then.
const httpShutdownTimeout = 2 * time.Second

func main() {
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

	dl := &download.YtdlpDownloader{DownloadDir: cfg.DownloadDir, TempDir: cfg.TempDir}
	manager := download.NewDownloadManager(dl)

	var shuttingDown atomic.Bool
	handler := router.NewRouter(router.Dependencies{
		Searcher:     &search.YtdlpSearcher{},
		Manager:      manager,
		Config:       *cfg,
		UI:           uiFS,
		YtdlpVersion: deps.YtdlpVersion,
		Ready:        func() bool { return !shuttingDown.Load() },
	})

	srv, err := server.NewServer(net.JoinHostPort(cfg.HTTPHost, cfg.HTTPPort), handler)
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Start() }()
	slog.Info("ready", "ytdlp", deps.YtdlpVersion, "downloadDir", cfg.DownloadDir)

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
	manager.Drain(drainCtx)

	stopCtx, cancelStop := context.WithTimeout(context.Background(), httpShutdownTimeout)
	defer cancelStop()
	if err := srv.Stop(stopCtx); err != nil {
		return fmt.Errorf("stop server: %w", err)
	}

	slog.Info("shutdown complete")
	return <-serveErr
}
