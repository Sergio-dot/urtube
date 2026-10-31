package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
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
	// DownloadDir is the default directory where finished downloads are
	// saved; users can pick another one in the app's settings.
	DownloadDir string
	// DownloadRoot, if set, is the folder every download folder must be
	// inside, including one chosen in the app. Empty allows any folder.
	DownloadRoot string
	// JellyfinURL and JellyfinAPIKey, when both set, make urtube ask
	// Jellyfin to scan its libraries after downloads; they can also be set
	// in the app.
	JellyfinURL    string
	JellyfinAPIKey string
	// DataDir holds urtube's own state, such as the settings saved from the
	// app.
	DataDir string
	// TempDir is where in-progress downloads are written before being moved to
	// DownloadDir, so partial files never appear in a watched media library.
	TempDir string
	// MaxConcurrentDownloads is how many downloads run at once; the rest wait
	// in a queue.
	MaxConcurrentDownloads int
	// ShutdownTimeout is how long in-flight downloads may run after a shutdown
	// signal before they are cancelled.
	ShutdownTimeout time.Duration
	// LogLevel is the logging level.
	LogLevel slog.Level
	// JSON selects JSON logs instead of text, for both application and
	// request logs.
	JSON bool
	// Concise indicates if request logging should be concise.
	Concise bool
	// RequestHeaders indicates if request headers should be logged.
	RequestHeaders bool

	// Warnings are problems found while loading that don't stop urtube,
	// such as renamed keys. Load doesn't log them because logging isn't set
	// up yet; the caller should.
	Warnings []string
}

const (
	keyHTTPHost        = "URTUBE_HTTP_HOST"
	keyHTTPPort        = "URTUBE_HTTP_PORT"
	keyDownloadDir     = "URTUBE_DOWNLOAD_DIR"
	keyTempDir         = "URTUBE_TEMP_DIR"
	keyDataDir         = "URTUBE_DATA_DIR"
	keyDownloadRoot    = "URTUBE_DOWNLOAD_ROOT"
	keyJellyfinURL     = "URTUBE_JELLYFIN_URL"
	keyJellyfinAPIKey  = "URTUBE_JELLYFIN_API_KEY" //nolint:gosec // G101: a variable name, not a credential
	keyMaxConcurrent   = "URTUBE_MAX_CONCURRENT_DOWNLOADS"
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
	v.SetDefault(keyDataDir, "./data")
	v.SetDefault(keyMaxConcurrent, 2)
	v.SetDefault(keyShutdownTimeout, "5s")
	v.SetDefault(keyLogLevel, "info")
	v.SetDefault(keyLogJSON, false)
	v.SetDefault(keyLogConcise, true)
	v.SetDefault(keyLogHeaders, true)

	v.SetConfigFile(".env")
	v.AutomaticEnv()
	// The .env file is optional: containers use environment variables.
	err := v.ReadInConfig()
	if err != nil && !errors.As(err, &viper.ConfigFileNotFoundError{}) && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	cfg, err := NewConfigWithViper(v)
	if err != nil {
		return nil, err
	}

	for old, replacement := range renamedKeys {
		if v.IsSet(old) {
			cfg.Warnings = append(cfg.Warnings, fmt.Sprintf("ignoring %s: it was renamed to %s", old, replacement))
		}
	}
	slices.Sort(cfg.Warnings) // map order is random
	return cfg, nil
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

	maxConcurrent, err := strconv.Atoi(v.GetString(keyMaxConcurrent))
	if err != nil || maxConcurrent < 1 {
		return nil, fmt.Errorf("%s: must be a positive integer, got %q", keyMaxConcurrent, v.GetString(keyMaxConcurrent))
	}

	port := v.GetString(keyHTTPPort)
	if p, err := strconv.Atoi(port); err != nil || p < 1 || p > 65535 {
		return nil, fmt.Errorf("%s: must be a port number between 1 and 65535, got %q", keyHTTPPort, port)
	}

	cfg := &Config{
		HTTPHost:               v.GetString(keyHTTPHost),
		HTTPPort:               port,
		DownloadDir:            v.GetString(keyDownloadDir),
		TempDir:                v.GetString(keyTempDir),
		DataDir:                v.GetString(keyDataDir),
		DownloadRoot:           v.GetString(keyDownloadRoot),
		JellyfinURL:            v.GetString(keyJellyfinURL),
		JellyfinAPIKey:         v.GetString(keyJellyfinAPIKey),
		MaxConcurrentDownloads: maxConcurrent,
		ShutdownTimeout:        timeout,
		LogLevel:               level,
		JSON:                   v.GetBool(keyLogJSON),
		Concise:                v.GetBool(keyLogConcise),
		RequestHeaders:         v.GetBool(keyLogHeaders),
	}

	if cfg.DownloadDir == "" {
		return nil, fmt.Errorf("%s: must not be empty", keyDownloadDir)
	}
	if cfg.TempDir == "" {
		return nil, fmt.Errorf("%s: must not be empty", keyTempDir)
	}
	if cfg.DataDir == "" {
		return nil, fmt.Errorf("%s: must not be empty", keyDataDir)
	}
	if cfg.DownloadRoot != "" {
		if !filepath.IsAbs(cfg.DownloadRoot) {
			return nil, fmt.Errorf("%s: must be an absolute path, got %q", keyDownloadRoot, cfg.DownloadRoot)
		}
		cfg.DownloadRoot = filepath.Clean(cfg.DownloadRoot)
	}

	return cfg, nil
}
