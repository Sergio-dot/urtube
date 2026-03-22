package main

import (
	"log"

	"github.com/Sergio-dot/urtube/internal/router"
	"github.com/Sergio-dot/urtube/internal/server"
)

func main() {
	router := router.NewRouter()
	srv, err := server.NewServer("localhost:8080", router)
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	err = srv.Start()
	if err != nil {
		log.Fatalf("%v\n", err)
	}
}
