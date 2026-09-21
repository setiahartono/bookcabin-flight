package server

import (
	"net/http"
	"time"

	"bookcabin-flight/internal/handler"
	"bookcabin-flight/internal/logging"
	"bookcabin-flight/internal/ratelimit"
	"bookcabin-flight/internal/service"
)

const (
	// searchLimit and searchWindow are how much a client may search, and in what
	// time it may do it.
	searchLimit  = 3
	searchWindow = 5 * time.Second
)

// NewRouter routes the API and writes what every request did to the log it is
// given, failed requests included.
func NewRouter(logger *logging.Logger) http.Handler {
	mux := http.NewServeMux()

	searchHandler := handler.NewSearchHandler(service.NewSearchService(logger))
	searches := ratelimit.New(searchLimit, searchWindow)

	mux.HandleFunc("GET /api/v1/ping", handler.Ping)
	mux.Handle("POST /api/v1/search", searches.Middleware(http.HandlerFunc(searchHandler.Search)))

	return logger.Middleware(mux)
}
