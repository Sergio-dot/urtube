package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Sergio-dot/urtube/internal/router"
	"github.com/Sergio-dot/urtube/internal/server"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	serverHost := os.Getenv("SERVER_HOST")
	serverPort := os.Getenv("SERVER_PORT")

	router := router.NewRouter()
	srv, err := server.NewServer(fmt.Sprintf("%s:%s", serverHost, serverPort), router)
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	err = srv.Start()
	if err != nil {
		log.Fatalf("%v\n", err)
	}
}
