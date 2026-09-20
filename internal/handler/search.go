package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"bookcabin-flight/internal/service"
)

type SearchHandler struct {
	search *service.SearchService
}

func NewSearchHandler(search *service.SearchService) *SearchHandler {
	return &SearchHandler{search: search}
}

func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)

		return
	}

	var criteria service.SearchCriteria
	if err := json.Unmarshal(body, &criteria); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)

		return
	}

	var camelCase struct {
		DepartureDate string `json:"departureDate"`
		CabinClass    string `json:"cabinClass"`
	}
	_ = json.Unmarshal(body, &camelCase)

	if criteria.DepartureDate == "" {
		criteria.DepartureDate = camelCase.DepartureDate
	}
	if criteria.CabinClass == "" {
		criteria.CabinClass = camelCase.CabinClass
	}

	result, err := h.search.Search(r.Context(), criteria)

	switch {
	case errors.Is(err, service.ErrInvalidCriteria):
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	case err != nil:
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}
