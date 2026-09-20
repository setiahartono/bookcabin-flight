package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"bookcabin-flight/internal/service"
)

// Searcher is the search service the handler calls.
type Searcher interface {
	Search(ctx context.Context, criteria service.SearchCriteria) (service.SearchResult, error)
}

type SearchHandler struct {
	search Searcher
}

func NewSearchHandler(search Searcher) *SearchHandler {
	return &SearchHandler{search: search}
}

// searchRequest is the camelCase body the search endpoint accepts. The criteria
// the handler works with keep the snake_case shape a response reports.
type searchRequest struct {
	Origin        string `json:"origin"`
	Destination   string `json:"destination"`
	DepartureDate string `json:"departureDate"`
	Passengers    int    `json:"passengers"`
	CabinClass    string `json:"cabinClass"`
	RoundTrip     *bool  `json:"roundTrip"`
}

func (r searchRequest) criteria() service.SearchCriteria {
	return service.SearchCriteria{
		Origin:        r.Origin,
		Destination:   r.Destination,
		DepartureDate: r.DepartureDate,
		Passengers:    r.Passengers,
		CabinClass:    r.CabinClass,
		RoundTrip:     r.RoundTrip,
	}
}

type errorBody struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(errorBody{Error: message})
}

func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")

		return
	}

	var request searchRequest
	if err := json.Unmarshal(body, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")

		return
	}

	result, err := h.search.Search(r.Context(), request.criteria())

	switch {
	case errors.Is(err, service.ErrInvalidCriteria):
		writeError(w, http.StatusBadRequest, err.Error())

		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, err.Error())

		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}
