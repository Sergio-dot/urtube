package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

// Config holds the application configuration.
//
// Environment variables are prefixed with URTUBE_ to avoid collisions. Avoid
// URTUBE_PORT and URTUBE_SERVICE_*: Kubernetes injects those for a Service
// named "urtube".
type Config struct {
	// HTTPHost is the host the server will bind to. Empty binds all interfaces.
	HTTPHost string
	// HTTPPort is the port the server will bind to.
	HTTPPort string
	// DownloadDir is the directory where finished downloads are saved.
	DownloadDir string
	// TempDir is where in-progress downloads are written before being moved to
	// DownloadDir, so partial files never appear in a watched media library.
	TempDir string
	// ShutdownTimeout is how long in-flight downloads may run after a shutdown
	// signal before they are cancelled.
	ShutdownTimeout time.Duration
	// LogLevel is the logging level.
	LogLevel slog.Level
	// JSON indicates if request logging should be in JSON format.
	JSON bool
	// Concise indicates if request logging should be concise.
	Concise bool
	// RequestHeaders indicates if request headers should be logged.
	RequestHeaders bool
}

const (
	keyHTTPHost        = "URTUBE_HTTP_HOST"
	keyHTTPPort        = "URTUBE_HTTP_PORT"
	keyDownloadDir     = "URTUBE_DOWNLOAD_DIR"
	keyTempDir         = "URTUBE_TEMP_DIR"
	keyShutdownTimeout = "URTUBE_SHUTDOWN_TIMEOUT"
	keyLogLevel        = "URTUBE_LOG_LEVEL"
	keyLogJSON         = "URTUBE_LOG_JSON"
	keyLogConcise      = "URTUBE_LOG_CONCISE"
	keyLogHeaders      = "URTUBE_LOG_REQUEST_HEADERS"
)

// renamedKeys maps configuration keys used before the URTUBE_ prefix to their
// replacements. The old names are ignored; Load only warns about them.
var renamedKeys = map[string]string{
	"SERVER_HOST":     keyHTTPHost,
	"SERVER_PORT":     keyHTTPPort,
	"DOWNLOAD_DIR":    keyDownloadDir,
	"LOG_LEVEL":       keyLogLevel,
	"JSON":            keyLogJSON,
	"CONCISE":         keyLogConcise,
	"REQUEST_HEADERS": keyLogHeaders,
}

// Load reads the configuration from an optional .env file and environment
// variables, which take precedence, and validates it.
func Load() (*Config, error) {
	v := viper.New()
	v.SetDefault(keyHTTPHost, "")
	v.SetDefault(keyHTTPPort, "8080")
	v.SetDefault(keyDownloadDir, "./downloads")
	v.SetDefault(keyTempDir, filepath.Join(os.TempDir(), "urtube"))
	v.SetDefault(keyShutdownTimeout, "5s")
	v.SetDefault(keyLogLevel, "info")
	v.SetDefault(keyLogJSON, false)
	v.SetDefault(keyLogConcise, true)
	v.SetDefault(keyLogHeaders, true)

	v.SetConfigFile(".env")
	v.AutomaticEnv()
	err := v.ReadInConfig()
	if err != nil {
		if errors.As(err, &viper.ConfigFileNotFoundError{}) || errors.Is(err, os.ErrNotExist) {
			slog.Info("no .env file found, using environment variables")
		} else {
			return nil, err
		}
	}

	for old, replacement := range renamedKeys {
		if v.IsSet(old) {
			slog.Warn("ignoring renamed config key", "key", old, "use", replacement)
		}
	}

	return NewConfigWithViper(v)
}

// NewConfigWithViper creates, validates and returns a new Config using the
// provided viper instance.
func NewConfigWithViper(v *viper.Viper) (*Config, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(v.GetString(keyLogLevel))); err != nil {
		return nil, fmt.Errorf("%s: %w", keyLogLevel, err)
	}

	timeout, err := time.ParseDuration(v.GetString(keyShutdownTimeout))
	if err != nil || timeout <= 0 {
		return nil, fmt.Errorf("%s: must be a positive duration such as 30s, got %q", keyShutdownTimeout, v.GetString(keyShutdownTimeout))
	}

	port := v.GetString(keyHTTPPort)
	if p, err := strconv.Atoi(port); err != nil || p < 1 || p > 65535 {
		return nil, fmt.Errorf("%s: must be a port number between 1 and 65535, got %q", keyHTTPPort, port)
	}

	cfg := &Config{
		HTTPHost:        v.GetString(keyHTTPHost),
		HTTPPort:        port,
		DownloadDir:     v.GetString(keyDownloadDir),
		TempDir:         v.GetString(keyTempDir),
		ShutdownTimeout: timeout,
		LogLevel:        level,
		JSON:            v.GetBool(keyLogJSON),
		Concise:         v.GetBool(keyLogConcise),
		RequestHeaders:  v.GetBool(keyLogHeaders),
	}

	if cfg.DownloadDir == "" {
		return nil, fmt.Errorf("%s: must not be empty", keyDownloadDir)
	}
	if cfg.TempDir == "" {
		return nil, fmt.Errorf("%s: must not be empty", keyTempDir)
	}

	return cfg, nil
}
