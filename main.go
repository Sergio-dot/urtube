package main

import (
	"fmt"
	"log"

	"github.com/Sergio-dot/urtube/internal/config"
	"github.com/Sergio-dot/urtube/internal/download"
	"github.com/Sergio-dot/urtube/internal/router"
	"github.com/Sergio-dot/urtube/internal/search"
	"github.com/Sergio-dot/urtube/internal/server"
)

func main() {
	config := config.NewConfig()

	// ytdlp.MustInstallAll(context.Background())

	router := router.NewRouter(router.Dependencies{
		Searcher:   &search.YtdlpSearcher{},
		Downloader: &download.YtdlpDownloader{DownloadDir: config.DownloadDir},
		Config:     *config,
	})
	srv, err := server.NewServer(fmt.Sprintf("%s:%s", config.ServerHost, config.ServerPort), router)
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	err = srv.Start()
	if err != nil {
		log.Fatalf("%v\n", err)
	}
}
