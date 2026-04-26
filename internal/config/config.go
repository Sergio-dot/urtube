package config

import (
	"errors"
	"log/slog"
	"os"

	"github.com/spf13/viper"
)

// Config holds the application configuration.
type Config struct {
	ServerHost     string
	ServerPort     string
	DownloadDir    string
	LogLevel       string
	JSON           bool
	Concise        bool
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
