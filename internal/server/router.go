package server

import (
	"net/http"
	"time"

	"bookcabin-flight/internal/handler"
	"bookcabin-flight/internal/ratelimit"
	"bookcabin-flight/internal/service"
)

const (
	// searchLimit and searchWindow are how much a client may search, and in what
	// time it may do it.
	searchLimit  = 3
	searchWindow = 5 * time.Second
)

// Handle Routes
func NewRouter() http.Handler {
	mux := http.NewServeMux()

	searchHandler := handler.NewSearchHandler(service.NewSearchService())
	searches := ratelimit.New(searchLimit, searchWindow)

	mux.HandleFunc("GET /api/v1/ping", handler.Ping)
	mux.Handle("POST /api/v1/search", searches.Middleware(http.HandlerFunc(searchHandler.Search)))

	return mux
}
