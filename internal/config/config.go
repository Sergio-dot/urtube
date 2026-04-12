package config

import (
	"errors"
	"fmt"
	"log"

	"github.com/fsnotify/fsnotify"
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

func init() {
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
		if errors.As(err, &viper.ConfigFileNotFoundError{}) {
			log.Println("No .env file found; using environment variables.")
		} else {
			log.Fatalf("Error reading .env file: %v\n", err)
		}
	}
	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Config file changed:", e.Name)
	})

	viper.WatchConfig()

	if viper.GetString("LOG_LEVEL") == "debug" {
		// print all config
		fmt.Println("Server Host:", viper.GetString("SERVER_HOST"))
		fmt.Println("Server Port:", viper.GetString("SERVER_PORT"))
		fmt.Println("Download Dir:", viper.GetString("DOWNLOAD_DIR"))
		fmt.Println("Log Level:", viper.GetString("LOG_LEVEL"))
		fmt.Println("JSON:", viper.GetBool("JSON"))
		fmt.Println("Concise:", viper.GetBool("CONCISE"))
		fmt.Println("Request Headers:", viper.GetBool("REQUEST_HEADERS"))
	}
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
