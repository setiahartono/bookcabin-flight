package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"bookcabin-flight/internal/provider"
)

// fakeProvider answers every search with the flights it recorded and keeps the
// requests it received.
type fakeProvider struct {
	flights  []provider.FlightData
	err      error
	requests []provider.SearchRequest
}

func (f *fakeProvider) Search(_ context.Context, req provider.SearchRequest) ([]provider.FlightData, error) {
	f.requests = append(f.requests, req)

	return f.flights, f.err
}

// newFlight builds a flight through its JSON form, since the nested types of
// provider.FlightData are unexported. The flights carry a departure date and
// enough seats to survive the filter of a one traveller search.
func newFlight(t *testing.T, id, from, to string, amount, minutes, stops int) provider.FlightData {
	t.Helper()

	body := fmt.Sprintf(
		`{"id":%q,"departure":{"airport":%q,"datetime":"2025-12-15T08:00:00+07:00"},`+
			`"arrival":{"airport":%q},"price":{"amount":%d,"currency":"IDR"},`+
			`"duration":{"total_minutes":%d},"stops":%d,"available_seats":9,"cabin_class":"economy"}`,
		id, from, to, amount, minutes, stops,
	)

	var flight provider.FlightData
	if err := json.Unmarshal([]byte(body), &flight); err != nil {
		t.Fatalf("decode flight %q: %v", id, err)
	}

	return flight
}

func criteria() SearchCriteria {
	return SearchCriteria{
		Origin:        "CGK",
		Destination:   "DPS",
		DepartureDate: "2025-12-15",
		Passengers:    1,
		CabinClass:    "economy",
	}
}

func flightIds(flights []provider.FlightData) []string {
	ids := make([]string, 0, len(flights))

	for _, flight := range flights {
		ids = append(ids, flight.Id)
	}

	return ids
}

// recordedFlights is the data a provider holds for the searched route.
func recordedFlights(t *testing.T) []provider.FlightData {
	t.Helper()

	return []provider.FlightData{
		newFlight(t, "JT740_Lion Air", "CGK", "DPS", 950000, 105, 0),
		newFlight(t, "QZ520_AirAsia", "CGK", "DPS", 650000, 100, 0),
	}
}

func TestSearchReturnsTheFlightsBestValueFirst(t *testing.T) {
	searcher := &fakeProvider{flights: recordedFlights(t)}

	result, err := newService(searcher).Search(context.Background(), criteria())
	if err != nil {
		t.Fatalf("Search() error = %v, want nil", err)
	}

	want := []string{"QZ520_AirAsia", "JT740_Lion Air"}
	if got := flightIds(result.Flights); !slices.Equal(got, want) {
		t.Errorf("Flights = %v, want %v", got, want)
	}
	if got, want := result.Metadata.TotalResults, 2; got != want {
		t.Errorf("TotalResults = %d, want %d", got, want)
	}
	if got, want := result.Metadata.ProvidersQueried, 1; got != want {
		t.Errorf("ProvidersQueried = %d, want %d", got, want)
	}
	if got := result.Metadata.ProvidersFailed; got != 0 {
		t.Errorf("ProvidersFailed = %d, want 0", got)
	}

	if len(searcher.requests) != 1 {
		t.Fatalf("provider queried %d times, want 1", len(searcher.requests))
	}

	request := searcher.requests[0]
	if request.Origin != "CGK" || request.Destination != "DPS" {
		t.Errorf("provider searched %s-%s, want CGK-DPS", request.Origin, request.Destination)
	}
	if got, want := request.DepartureDate.Format(time.DateOnly), "2025-12-15"; got != want {
		t.Errorf("provider departure date = %q, want %q", got, want)
	}
	if request.Passengers != 1 || request.CabinClass != "economy" {
		t.Errorf("provider request = %+v, want the travellers and cabin of the search", request)
	}
}

func TestSearchScoresTheFlightsItReturns(t *testing.T) {
	searcher := &fakeProvider{flights: recordedFlights(t)}

	result, err := newService(searcher).Search(context.Background(), criteria())
	if err != nil {
		t.Fatalf("Search() error = %v, want nil", err)
	}

	if len(result.Flights) != 2 {
		t.Fatalf("len(Flights) = %d, want 2", len(result.Flights))
	}

	// The cheapest flight is the best value of the search, so it scores 1.
	tests := []struct {
		index int
		value float64
		price float64
	}{
		{index: 0, value: 1, price: 1},
		{index: 1, value: 0.77, price: 0.684},
	}

	for _, tt := range tests {
		t.Run(result.Flights[tt.index].Id, func(t *testing.T) {
			flight := result.Flights[tt.index]

			if math.Abs(flight.Score.Value-tt.value) > 1e-9 {
				t.Errorf("Score.Value = %v, want %v", flight.Score.Value, tt.value)
			}
			if math.Abs(flight.Score.Price-tt.price) > 1e-9 {
				t.Errorf("Score.Price = %v, want %v", flight.Score.Price, tt.price)
			}
		})
	}
}

func TestSearchLeavesOutTheFlightsThatDoNotMatch(t *testing.T) {
	searcher := &fakeProvider{flights: []provider.FlightData{
		newFlight(t, "QZ520_AirAsia", "CGK", "DPS", 650000, 100, 0),
		// The other way round, so the search never asked for it.
		newFlight(t, "GA404_Garuda", "DPS", "CGK", 1400000, 110, 0),
	}}

	result, err := newService(searcher).Search(context.Background(), criteria())
	if err != nil {
		t.Fatalf("Search() error = %v, want nil", err)
	}

	want := []string{"QZ520_AirAsia"}
	if got := flightIds(result.Flights); !slices.Equal(got, want) {
		t.Errorf("Flights = %v, want %v", got, want)
	}
}

func TestSearchReportsAFailedProviderWithoutFailing(t *testing.T) {
	searcher := &fakeProvider{err: provider.ErrUnavailable}

	result, err := newService(searcher).Search(context.Background(), criteria())
	if err != nil {
		t.Fatalf("Search() error = %v, want nil so a failed provider only shows in the metadata", err)
	}

	if got, want := result.Metadata.ProvidersFailed, 1; got != want {
		t.Errorf("ProvidersFailed = %d, want %d", got, want)
	}
	if result.Flights == nil {
		t.Fatal("Flights = nil, want an empty slice so the response marshals to []")
	}
	if len(result.Flights) != 0 {
		t.Errorf("len(Flights) = %d, want 0", len(result.Flights))
	}
}

func TestSearchWithoutFlights(t *testing.T) {
	searcher := &fakeProvider{}

	result, err := newService(searcher).Search(context.Background(), criteria())
	if err != nil {
		t.Fatalf("Search() error = %v, want nil", err)
	}

	if result.Flights == nil {
		t.Fatal("Flights = nil, want an empty slice so the response marshals to []")
	}
	if got := result.Metadata.TotalResults; got != 0 {
		t.Errorf("TotalResults = %d, want 0", got)
	}
}

func TestSearchRejectsAnInvalidDepartureDate(t *testing.T) {
	searcher := &fakeProvider{flights: recordedFlights(t)}

	invalid := criteria()
	invalid.DepartureDate = ""

	if _, err := newService(searcher).Search(context.Background(), invalid); !errors.Is(err, ErrInvalidCriteria) {
		t.Errorf("Search() error = %v, want %v", err, ErrInvalidCriteria)
	}
	if got := len(searcher.requests); got != 0 {
		t.Errorf("providers queried %d times, want 0", got)
	}
}

func TestSearchNeedsAReturnDateForARoundTrip(t *testing.T) {
	tests := []struct {
		name       string
		roundTrip  *bool
		returnDate string
		wantError  bool
	}{
		{name: "round trip without a return date", roundTrip: boolPtr(true), wantError: true},
		{name: "round trip with a return date", roundTrip: boolPtr(true), returnDate: "2025-12-20"},
		{name: "one way without a return date", roundTrip: boolPtr(false)},
		{name: "one way with a return date", returnDate: "2025-12-20"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			searcher := &fakeProvider{flights: recordedFlights(t)}

			criteria := criteria()
			criteria.RoundTrip = tt.roundTrip
			criteria.ReturnDate = tt.returnDate

			result, err := newService(searcher).Search(context.Background(), criteria)

			if tt.wantError {
				if !errors.Is(err, ErrInvalidCriteria) {
					t.Fatalf("Search() error = %v, want %v", err, ErrInvalidCriteria)
				}
				if !strings.Contains(err.Error(), "returnDate") {
					t.Errorf("Search() error = %v, want it to name returnDate", err)
				}
				if got := len(searcher.requests); got != 0 {
					t.Errorf("providers queried %d times, want 0", got)
				}

				return
			}

			if err != nil {
				t.Fatalf("Search() error = %v, want nil", err)
			}

			// The way back is not searched yet, so a round trip still costs one
			// search and answers with the flights of the outbound trip.
			if got, want := len(searcher.requests), 1; got != want {
				t.Errorf("providers queried %d times, want %d", got, want)
			}
			if got, want := result.Metadata.TotalResults, 2; got != want {
				t.Errorf("TotalResults = %d, want %d", got, want)
			}
		})
	}
}

func boolPtr(value bool) *bool { return &value }
