package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()
	assert.NotNil(t, cfg)
}

func TestLoad_Defaults(t *testing.T) {
	// Clear relevant env vars to test defaults
	t.Setenv("SERVER_HOST", "")
	t.Setenv("SERVER_PORT", "")
	t.Setenv("DOWNLOAD_DIR", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("JSON", "")
	t.Setenv("CONCISE", "")
	t.Setenv("REQUEST_HEADERS", "")

	cfg, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, "localhost", cfg.ServerHost)
	assert.Equal(t, "8080", cfg.ServerPort)
	assert.Equal(t, "./downloads", cfg.DownloadDir)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.False(t, cfg.JSON)
	assert.True(t, cfg.Concise)
	assert.True(t, cfg.RequestHeaders)
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("SERVER_HOST", "127.0.0.1")
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("DOWNLOAD_DIR", "/tmp/downloads")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("JSON", "true")
	t.Setenv("CONCISE", "false")
	t.Setenv("REQUEST_HEADERS", "false")

	cfg, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, "127.0.0.1", cfg.ServerHost)
	assert.Equal(t, "9090", cfg.ServerPort)
	assert.Equal(t, "/tmp/downloads", cfg.DownloadDir)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.True(t, cfg.JSON)
	assert.False(t, cfg.Concise)
	assert.False(t, cfg.RequestHeaders)
}

func TestLoad_WithDotEnvFile(t *testing.T) {
	// Clear env variables so they don't override the .env file
	t.Setenv("SERVER_HOST", "")
	t.Setenv("SERVER_PORT", "")

	envData := []byte("SERVER_HOST=testenvfile\nSERVER_PORT=7070\n")
	err := os.WriteFile(".env", envData, 0644)
	if err != nil {
		t.Fatalf("failed to write .env: %v", err)
	}
	defer os.Remove(".env")

	cfg, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, "testenvfile", cfg.ServerHost)
	assert.Equal(t, "7070", cfg.ServerPort)
}

