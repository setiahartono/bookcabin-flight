package server

import (
	"log"
	"net/http"
)

func Run(addr string, handler http.Handler) error {
	log.Printf("Server is starting on port %s", addr)

	if err := http.ListenAndServe(addr, handler); err != nil {
		return err
	}

	return nil
}
