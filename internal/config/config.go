package config

import (
	"errors"
	"log/slog"
	"os"

	"github.com/spf13/viper"
)

// Config holds the application configuration.
type Config struct {
	// ServerHost is the host the server will bind to.
	ServerHost string
	// ServerPort is the port the server will bind to.
	ServerPort string
	// DownloadDir is the directory where downloads will be saved.
	DownloadDir string
	// LogLevel is the logging level (e.g., info, debug, error).
	LogLevel string
	// JSON indicates if logging should be in JSON format.
	JSON bool
	// Concise indicates if logging should be concise.
	Concise bool
	// RequestHeaders indicates if request headers should be logged.
	RequestHeaders bool
}

// Load reads the configuration from environment variables.
func Load() (*Config, error) {
	v := viper.New()
	v.SetDefault("SERVER_HOST", "localhost")
	v.SetDefault("SERVER_PORT", "8080")
	v.SetDefault("DOWNLOAD_DIR", "./downloads")
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("JSON", false)
	v.SetDefault("CONCISE", true)
	v.SetDefault("REQUEST_HEADERS", true)

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

	return NewConfigWithViper(v), nil
}

// NewConfig creates and returns a new Config using the global viper instance.
func NewConfig() *Config {
	return NewConfigWithViper(viper.GetViper())
}

// NewConfigWithViper creates and returns a new Config using the provided viper instance.
func NewConfigWithViper(v *viper.Viper) *Config {
	return &Config{
		ServerHost:     v.GetString("SERVER_HOST"),
		ServerPort:     v.GetString("SERVER_PORT"),
		DownloadDir:    v.GetString("DOWNLOAD_DIR"),
		LogLevel:       v.GetString("LOG_LEVEL"),
		JSON:           v.GetBool("JSON"),
		Concise:        v.GetBool("CONCISE"),
		RequestHeaders: v.GetBool("REQUEST_HEADERS"),
	}
}
