package main

import (
	"log"
	"os"

	"bookcabin-flight/internal/server"
)

func main() {
	addr := ":" + port()

	if err := server.Run(addr, server.NewRouter()); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func port() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return "8080"
}
