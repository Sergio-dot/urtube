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
	viper.SetDefault("SERVER_HOST", "localhost")
	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("DOWNLOAD_DIR", "./downloads")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("JSON", false)
	viper.SetDefault("CONCISE", true)
	viper.SetDefault("REQUEST_HEADERS", true)

	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		if errors.As(err, &viper.ConfigFileNotFoundError{}) || errors.Is(err, os.ErrNotExist) {
			slog.Info("no .env file found, using environment variables")
		} else {
			return nil, err
		}
	}

	return NewConfig(), nil
}

// NewConfig creates and returns a new Config.
func NewConfig() *Config {
	return &Config{
		ServerHost:     viper.GetString("SERVER_HOST"),
		ServerPort:     viper.GetString("SERVER_PORT"),
		DownloadDir:    viper.GetString("DOWNLOAD_DIR"),
		LogLevel:       viper.GetString("LOG_LEVEL"),
		JSON:           viper.GetBool("JSON"),
		Concise:        viper.GetBool("CONCISE"),
		RequestHeaders: viper.GetBool("REQUEST_HEADERS"),
	}
}
