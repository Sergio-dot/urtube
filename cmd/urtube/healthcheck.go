package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/Sergio-dot/urtube/internal/config"
)

// healthcheckTimeout bounds the readiness request made by "urtube healthcheck".
const healthcheckTimeout = 3 * time.Second

// healthcheck implements "urtube healthcheck": it requests the readiness probe
// of the server configured for this host and fails unless it answers 200. The
// container image uses it as its HEALTHCHECK, since it ships no HTTP client.
func healthcheck() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), healthcheckTimeout)
	defer cancel()
	return probe(ctx, readyzURL(cfg.HTTPHost, cfg.HTTPPort))
}

// readyzURL returns the readiness probe URL for a server bound to host and
// port, using loopback when the server listens on all interfaces.
func readyzURL(host, port string) string {
	switch host {
	case "", "0.0.0.0", "::":
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port) + "/readyz"
}

func probe(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", url, resp.Status)
	}
	return nil
}
