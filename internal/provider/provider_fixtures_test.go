package provider

import (
	"context"
	"errors"
	"testing"
)

// recordedSearcher is a provider that replays the fixture it was built from.
type recordedSearcher interface {
	Search(ctx context.Context, req SearchRequest) ([]FlightData, bool, error)
}

func recorded[T recordedSearcher](t *testing.T, newProvider func() (T, error)) T {
	t.Helper()

	provider, err := newProvider()
	if err != nil {
		t.Fatalf("newProvider() error = %v", err)
	}

	return provider
}

// TestRecordedDataCoversTheAddedRoutes keeps the fixtures honest: a round trip
// needs the way back to Jakarta on both return dates, and the added cities need
// flights to and from the airports outside the western zone.
func TestRecordedDataCoversTheAddedRoutes(t *testing.T) {
	providers := []struct {
		name     string
		searcher recordedSearcher
	}{
		{name: "AirAsia", searcher: recorded(t, NewAirAsia)},
		{name: "Batik Air", searcher: recorded(t, NewBatikAir)},
		{name: "Garuda Indonesia", searcher: recorded(t, NewGaruda)},
		{name: "Lion Air", searcher: recorded(t, NewLionAir)},
	}

	want := []string{
		"CGK-DPS@2025-12-15",
		"CGK-UPG@2025-12-15",
		"CGK-DJJ@2025-12-15",
		"DPS-CGK@2025-12-20",
		"DPS-CGK@2025-12-22",
		"UPG-CGK@2025-12-20",
		"UPG-CGK@2025-12-22",
	}

	for _, tt := range providers {
		t.Run(tt.name, func(t *testing.T) {
			routes := make(map[string]bool)

			for range 10 {
				flights, _, err := tt.searcher.Search(context.Background(), SearchRequest{})
				if err == nil {
					for _, flight := range flights {
						routes[flight.Departure.Airport+"-"+flight.Arrival.Airport+"@"+departureDate(flight)] = true
					}

					break
				}

				if !errors.Is(err, ErrUnavailable) {
					t.Fatalf("Search() error = %v, want nil or ErrUnavailable", err)
				}
			}

			for _, route := range want {
				if !routes[route] {
					t.Errorf("the fixture holds no %s flight", route)
				}
			}
		})
	}
}

// departureDate reads the date a flight leaves on.
func departureDate(flight FlightData) string {
	if len(flight.Departure.Datetime) < len("2006-01-02") {
		return flight.Departure.Datetime
	}

	return flight.Departure.Datetime[:len("2006-01-02")]
}
