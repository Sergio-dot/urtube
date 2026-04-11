package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"

	"github.com/Sergio-dot/urtube/internal/download"
	"github.com/Sergio-dot/urtube/internal/router"
	"github.com/Sergio-dot/urtube/internal/search"
	"github.com/Sergio-dot/urtube/internal/server"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			log.Println("No .env file found; using environment variables.")
		} else {
			log.Fatalf("Error loading .env file: %v\n", err)
		}
	}

	serverHost := os.Getenv("SERVER_HOST")
	if serverHost == "" {
		serverHost = "0.0.0.0"
	}

	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "8080"
	}

	downloadDir := os.Getenv("DOWNLOAD_DIR")
	if downloadDir == "" {
		downloadDir = "./downloads"
	}

	// ytdlp.MustInstallAll(context.Background())

	router := router.NewRouter(router.Dependencies{
		Searcher:   &search.YtdlpSearcher{},
		Downloader: &download.YtdlpDownloader{OutputDir: downloadDir},
	})
	srv, err := server.NewServer(fmt.Sprintf("%s:%s", serverHost, serverPort), router)
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	err = srv.Start()
	if err != nil {
		log.Fatalf("%v\n", err)
	}
}
