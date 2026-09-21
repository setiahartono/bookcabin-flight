package handler

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"bookcabin-flight/internal/service"
)

// fakeSearcher answers a search with the criteria it received, the way the
// search service echoes them, and keeps what the handler passed on.
type fakeSearcher struct {
	criteria service.SearchCriteria
	err      error
	calls    int
}

func (f *fakeSearcher) Search(_ context.Context, criteria service.SearchCriteria) (service.SearchResult, error) {
	f.calls++
	f.criteria = criteria

	return service.SearchResult{SearchCriteria: criteria}, f.err
}

func post(t *testing.T, searcher Searcher, body string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/search", strings.NewReader(body))

	NewSearchHandler(searcher).Search(recorder, request)

	return recorder
}

func boolPtr(value bool) *bool { return &value }

func assertCriteria(t *testing.T, got, want service.SearchCriteria) {
	t.Helper()

	if got.Origin != want.Origin || got.Destination != want.Destination {
		t.Errorf("criteria route = %s-%s, want %s-%s", got.Origin, got.Destination, want.Origin, want.Destination)
	}
	if got.DepartureDate != want.DepartureDate {
		t.Errorf("criteria DepartureDate = %q, want %q", got.DepartureDate, want.DepartureDate)
	}
	if got.Passengers != want.Passengers || got.CabinClass != want.CabinClass {
		t.Errorf("criteria = %+v, want %+v", got, want)
	}
	if got.SortBy != want.SortBy {
		t.Errorf("criteria SortBy = %q, want %q", got.SortBy, want.SortBy)
	}
}

func TestSearchMapsTheCamelCaseRequest(t *testing.T) {
	searcher := &fakeSearcher{}

	recorder := post(t, searcher, `{"origin":"CGK","destination":"DPS","departureDate":"2025-12-15","returnDate":"2025-12-20","passengers":2,"cabinClass":"economy","roundTrip":true,"sortBy":"price"}`)

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Fatalf("status = %d, want %d (body %s)", got, want, recorder.Body)
	}

	assertCriteria(t, searcher.criteria, service.SearchCriteria{
		Origin:        "CGK",
		Destination:   "DPS",
		DepartureDate: "2025-12-15",
		Passengers:    2,
		CabinClass:    "economy",
		SortBy:        "price",
	})

	if got, want := searcher.criteria.ReturnDate, "2025-12-20"; got != want {
		t.Errorf("criteria ReturnDate = %q, want %q", got, want)
	}

	if searcher.criteria.RoundTrip == nil {
		t.Fatal("criteria RoundTrip = nil, want true")
	}
	if !*searcher.criteria.RoundTrip {
		t.Errorf("criteria RoundTrip = %v, want true", *searcher.criteria.RoundTrip)
	}
}

func TestSearchWithoutRoundTripLeavesTheWayBackOut(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "key left out",
			body: `{"origin":"CGK","destination":"DPS","departureDate":"2025-12-15","passengers":1,"cabinClass":"economy"}`,
		},
		{
			name: "key null",
			body: `{"origin":"CGK","destination":"DPS","departureDate":"2025-12-15","passengers":1,"cabinClass":"economy","roundTrip":null}`,
		},
		{
			name: "key false",
			body: `{"origin":"CGK","destination":"DPS","departureDate":"2025-12-15","passengers":1,"cabinClass":"economy","roundTrip":false}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			searcher := &fakeSearcher{}

			if got, want := post(t, searcher, tt.body).Code, http.StatusOK; got != want {
				t.Fatalf("status = %d, want %d", got, want)
			}
			if searcher.criteria.RoundTrip != nil && *searcher.criteria.RoundTrip {
				t.Error("criteria RoundTrip = true, want the way back left out")
			}
		})
	}
}

func TestSearchIgnoresASnakeCaseRequest(t *testing.T) {
	searcher := &fakeSearcher{}

	recorder := post(t, searcher, `{"origin":"CGK","destination":"DPS","departure_date":"2025-12-15","return_date":"2025-12-20","passengers":1,"cabin_class":"economy"}`)

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}

	// The handler reads camelCase only, so the snake_case keys are left unread
	// and the search service answers with invalid criteria.
	if searcher.criteria.DepartureDate != "" {
		t.Errorf("criteria DepartureDate = %q, want the snake_case key ignored", searcher.criteria.DepartureDate)
	}
	if searcher.criteria.ReturnDate != "" {
		t.Errorf("criteria ReturnDate = %q, want the snake_case key ignored", searcher.criteria.ReturnDate)
	}
	if searcher.criteria.CabinClass != "" {
		t.Errorf("criteria CabinClass = %q, want the snake_case key ignored", searcher.criteria.CabinClass)
	}
	if searcher.criteria.Origin != "CGK" || searcher.criteria.Destination != "DPS" {
		t.Errorf("criteria route = %s-%s, want CGK-DPS", searcher.criteria.Origin, searcher.criteria.Destination)
	}
}

func TestSearchRejectsAnUnreadableBody(t *testing.T) {
	searcher := &fakeSearcher{}

	recorder := post(t, searcher, `{"origin":`)

	if got, want := recorder.Code, http.StatusBadRequest; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}

	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if got, want := body.Error, "invalid request body"; got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
	if searcher.calls != 0 {
		t.Errorf("search service called %d times, want 0", searcher.calls)
	}
}

func TestSearchReportsAFailedSearch(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "invalid criteria", err: service.ErrInvalidCriteria, wantStatus: http.StatusBadRequest},
		{name: "unexpected error", err: errors.New("boom"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := post(t, &fakeSearcher{err: tt.err}, `{"origin":"CGK","departureDate":"2025-12-15"}`)

			if got := recorder.Code; got != tt.wantStatus {
				t.Errorf("status = %d, want %d", got, tt.wantStatus)
			}
			if body := recorder.Body.String(); !strings.Contains(body, tt.err.Error()) {
				t.Errorf("body = %s, want the error %q", body, tt.err)
			}
		})
	}
}

func TestSearchRendersTheCriteriaInSnakeCase(t *testing.T) {
	recorder := post(t, &fakeSearcher{}, `{"origin":"CGK","destination":"DPS","departureDate":"2025-12-15","passengers":1,"cabinClass":"economy","roundTrip":true,"sortBy":"price"}`)

	var body struct {
		SearchCriteria map[string]any `json:"search_criteria"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	got := slices.Sorted(maps.Keys(body.SearchCriteria))

	want := []string{"cabin_class", "departure_date", "destination", "origin", "passengers", "return_date", "round_trip", "sort_by"}
	if !slices.Equal(got, want) {
		t.Errorf("search_criteria keys = %v, want %v", got, want)
	}

	if got, want := body.SearchCriteria["sort_by"], "price"; got != want {
		t.Errorf("search_criteria.sort_by = %v, want %v", got, want)
	}
}
