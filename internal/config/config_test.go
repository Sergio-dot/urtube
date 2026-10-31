package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// clearEnv unsets every config key so tests see defaults or the .env file.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		keyHTTPHost, keyHTTPPort, keyDownloadDir, keyTempDir, keyMaxConcurrent, keyShutdownTimeout,
		keyLogLevel, keyLogJSON, keyLogConcise, keyLogHeaders,
	} {
		t.Setenv(k, "")
	}
}

func TestLoad_Defaults(t *testing.T) {
	clearEnv(t)

	cfg, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, "", cfg.HTTPHost, "must bind all interfaces so the server is reachable from outside a container")
	assert.Equal(t, "8080", cfg.HTTPPort)
	assert.Equal(t, "./downloads", cfg.DownloadDir)
	assert.Equal(t, filepath.Join(os.TempDir(), "urtube"), cfg.TempDir)
	assert.Equal(t, 2, cfg.MaxConcurrentDownloads)
	assert.Equal(t, 5*time.Second, cfg.ShutdownTimeout)
	assert.Equal(t, slog.LevelInfo, cfg.LogLevel)
	assert.False(t, cfg.JSON)
	assert.True(t, cfg.Concise)
	assert.True(t, cfg.RequestHeaders)
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv(keyHTTPHost, "127.0.0.1")
	t.Setenv(keyHTTPPort, "9090")
	t.Setenv(keyDownloadDir, "/srv/media/music")
	t.Setenv(keyTempDir, "/scratch")
	t.Setenv(keyMaxConcurrent, "4")
	t.Setenv(keyShutdownTimeout, "1m")
	t.Setenv(keyLogLevel, "debug")
	t.Setenv(keyLogJSON, "true")
	t.Setenv(keyLogConcise, "false")
	t.Setenv(keyLogHeaders, "false")

	cfg, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, "127.0.0.1", cfg.HTTPHost)
	assert.Equal(t, "9090", cfg.HTTPPort)
	assert.Equal(t, "/srv/media/music", cfg.DownloadDir)
	assert.Equal(t, "/scratch", cfg.TempDir)
	assert.Equal(t, 4, cfg.MaxConcurrentDownloads)
	assert.Equal(t, time.Minute, cfg.ShutdownTimeout)
	assert.Equal(t, slog.LevelDebug, cfg.LogLevel)
	assert.True(t, cfg.JSON)
	assert.False(t, cfg.Concise)
	assert.False(t, cfg.RequestHeaders)
}

func TestLoad_IgnoresLegacyKeys(t *testing.T) {
	clearEnv(t)
	t.Setenv("SERVER_PORT", "7070")
	t.Setenv("DOWNLOAD_DIR", "/legacy")

	cfg, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, "8080", cfg.HTTPPort)
	assert.Equal(t, "./downloads", cfg.DownloadDir)
}

func TestLoad_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "unknown log level", key: keyLogLevel, value: "loud"},
		{name: "non-numeric port", key: keyHTTPPort, value: "http"},
		{name: "port out of range", key: keyHTTPPort, value: "70000"},
		{name: "unparsable shutdown timeout", key: keyShutdownTimeout, value: "soon"},
		{name: "non-positive shutdown timeout", key: keyShutdownTimeout, value: "0s"},
		{name: "zero concurrent downloads", key: keyMaxConcurrent, value: "0"},
		{name: "non-numeric concurrent downloads", key: keyMaxConcurrent, value: "many"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			t.Setenv(tt.key, tt.value)

			_, err := Load()
			assert.ErrorContains(t, err, tt.key)
		})
	}
}

func TestLoad_WithDotEnvFile(t *testing.T) {
	// Clear env variables so they don't override the .env file
	clearEnv(t)

	envData := []byte("URTUBE_HTTP_HOST=testenvfile\nURTUBE_HTTP_PORT=7070\n")
	err := os.WriteFile(".env", envData, 0644)
	if err != nil {
		t.Fatalf("failed to write .env: %v", err)
	}
	defer os.Remove(".env")

	cfg, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, "testenvfile", cfg.HTTPHost)
	assert.Equal(t, "7070", cfg.HTTPPort)
}
