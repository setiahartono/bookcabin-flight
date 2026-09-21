package provider

import (
	"context"
	"errors"
	"testing"
	"time"
)

// request is a search of the route a provider is asked about.
func request(origin, destination string) SearchRequest {
	return SearchRequest{
		Origin:        origin,
		Destination:   destination,
		DepartureDate: time.Date(2025, 12, 15, 0, 0, 0, 0, time.UTC),
		Passengers:    1,
		CabinClass:    "economy",
	}
}

func TestSearchKeepsWhatItAnswered(t *testing.T) {
	svc, err := newProviderService("Test Provider", garudaFixture)
	if err != nil {
		t.Fatalf("newProviderService() error = %v", err)
	}

	calls := 0
	call := func() ([]FlightData, error) {
		calls++

		return []FlightData{{Id: "GA403_Garuda Indonesia"}}, nil
	}

	flights, cached, err := svc.search(request("CGK", "DPS"), call)
	if err != nil {
		t.Fatalf("search() error = %v, want nil", err)
	}
	if cached {
		t.Error("search() cached = true, want false on the first search")
	}
	if len(flights) != 1 || calls != 1 {
		t.Fatalf("search() = %d flights after %d calls, want 1 flight after 1 call", len(flights), calls)
	}

	flights, cached, err = svc.search(request("CGK", "DPS"), call)
	if err != nil {
		t.Fatalf("search() error = %v, want nil", err)
	}
	if !cached {
		t.Error("search() cached = false, want true on the repeated search")
	}
	if len(flights) != 1 || calls != 1 {
		t.Errorf("search() = %d flights after %d calls, want the kept answer after 1 call", len(flights), calls)
	}
}

func TestSearchKeepsRequestsApart(t *testing.T) {
	svc, err := newProviderService("Test Provider", garudaFixture)
	if err != nil {
		t.Fatalf("newProviderService() error = %v", err)
	}

	calls := 0
	call := func() ([]FlightData, error) {
		calls++

		return []FlightData{{Id: "GA403_Garuda Indonesia"}}, nil
	}

	if _, _, err := svc.search(request("CGK", "DPS"), call); err != nil {
		t.Fatalf("search() error = %v, want nil", err)
	}

	_, cached, err := svc.search(request("CGK", "SUB"), call)
	if err != nil {
		t.Fatalf("search() error = %v, want nil", err)
	}
	if cached {
		t.Error("search() cached = true, want false for another route")
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2, one for each route", calls)
	}
}

func TestSearchDoesNotKeepAFailedCall(t *testing.T) {
	svc, err := newProviderService("Test Provider", garudaFixture)
	if err != nil {
		t.Fatalf("newProviderService() error = %v", err)
	}

	unavailable := errors.New("provider unavailable")
	calls := 0
	failing := func() ([]FlightData, error) {
		calls++

		return nil, unavailable
	}

	if _, _, err := svc.search(request("CGK", "DPS"), failing); !errors.Is(err, unavailable) {
		t.Fatalf("search() error = %v, want %v", err, unavailable)
	}

	flights, cached, err := svc.search(request("CGK", "DPS"), func() ([]FlightData, error) {
		calls++

		return []FlightData{{Id: "GA403_Garuda Indonesia"}}, nil
	})
	if err != nil {
		t.Fatalf("search() error = %v, want nil", err)
	}
	if cached {
		t.Error("search() cached = true, want the failed call not to be kept")
	}
	if len(flights) != 1 || calls != 2 {
		t.Errorf("search() = %d flights after %d calls, want the provider asked again", len(flights), calls)
	}
}

func TestGarudaAnswersARepeatedSearchFromItsCache(t *testing.T) {
	garuda, err := NewGaruda()
	if err != nil {
		t.Fatalf("NewGaruda() error = %v", err)
	}

	flights, cached, err := garuda.Search(context.Background(), request("CGK", "DPS"))
	if err != nil {
		t.Fatalf("Search() error = %v, want nil", err)
	}
	if cached {
		t.Error("Search() cached = true, want false on the first search")
	}
	if len(flights) == 0 {
		t.Fatal("len(Search()) = 0, want the recorded flights")
	}

	kept, cached, err := garuda.Search(context.Background(), request("CGK", "DPS"))
	if err != nil {
		t.Fatalf("Search() error = %v, want nil", err)
	}
	if !cached {
		t.Error("Search() cached = false, want true on the repeated search")
	}
	if len(kept) != len(flights) {
		t.Errorf("len(Search()) = %d, want the %d kept flights", len(kept), len(flights))
	}
}
