package server

import (
	"net/http"

	"bookcabin-flight/internal/handler"
	"bookcabin-flight/internal/service"
)

// Handle Routes
func NewRouter() http.Handler {
	mux := http.NewServeMux()

	searchHandler := handler.NewSearchHandler(service.NewSearchService())

	mux.HandleFunc("GET /api/v1/ping", handler.Ping)
	mux.HandleFunc("POST /api/v1/search", searchHandler.Search)

	return mux
}
