package provider

import (
	"context"
	"errors"
	"fmt"
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

	flights, cached, err := svc.search(context.Background(), request("CGK", "DPS"), call)
	if err != nil {
		t.Fatalf("search() error = %v, want nil", err)
	}
	if cached {
		t.Error("search() cached = true, want false on the first search")
	}
	if len(flights) != 1 || calls != 1 {
		t.Fatalf("search() = %d flights after %d calls, want 1 flight after 1 call", len(flights), calls)
	}

	flights, cached, err = svc.search(context.Background(), request("CGK", "DPS"), call)
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

	if _, _, err := svc.search(context.Background(), request("CGK", "DPS"), call); err != nil {
		t.Fatalf("search() error = %v, want nil", err)
	}

	_, cached, err := svc.search(context.Background(), request("CGK", "SUB"), call)
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

func TestSearchDoesNotKeepACallThatFailedEveryAttempt(t *testing.T) {
	svc, err := newProviderService("Test Provider", garudaFixture)
	if err != nil {
		t.Fatalf("newProviderService() error = %v", err)
	}

	unavailable := fmt.Errorf("%s: %w", svc.name, ErrUnavailable)
	calls := 0
	failing := func() ([]FlightData, error) {
		calls++

		return nil, unavailable
	}

	if _, _, err := svc.search(context.Background(), request("CGK", "DPS"), failing); !errors.Is(err, unavailable) {
		t.Fatalf("search() error = %v, want %v", err, unavailable)
	}
	if calls != maxAttempts {
		t.Errorf("search() called the provider %d times, want its %d attempts", calls, maxAttempts)
	}

	flights, cached, err := svc.search(context.Background(), request("CGK", "DPS"), func() ([]FlightData, error) {
		calls++

		return []FlightData{{Id: "GA403_Garuda Indonesia"}}, nil
	})
	if err != nil {
		t.Fatalf("search() error = %v, want nil", err)
	}
	if cached {
		t.Error("search() cached = true, want the failed call not to be kept")
	}
	if len(flights) != 1 || calls != maxAttempts+1 {
		t.Errorf("search() = %d flights after %d calls, want the provider asked again", len(flights), calls)
	}
}

func TestSearchRetriesAProviderThatIsUnavailable(t *testing.T) {
	svc, err := newProviderService("Test Provider", garudaFixture)
	if err != nil {
		t.Fatalf("newProviderService() error = %v", err)
	}

	calls := 0
	flaky := func() ([]FlightData, error) {
		calls++

		if calls < maxAttempts {
			return nil, fmt.Errorf("%s: %w", svc.name, ErrUnavailable)
		}

		return []FlightData{{Id: "GA403_Garuda Indonesia"}}, nil
	}

	flights, cached, err := svc.search(context.Background(), request("CGK", "DPS"), flaky)
	if err != nil {
		t.Fatalf("search() error = %v, want the last attempt to answer", err)
	}
	if cached {
		t.Error("search() cached = true, want false for a call that had to be retried")
	}
	if len(flights) != 1 {
		t.Errorf("len(search()) = %d, want the flights of the last attempt", len(flights))
	}
	if calls != maxAttempts {
		t.Errorf("search() called the provider %d times, want %d", calls, maxAttempts)
	}
}

func TestSearchGivesUpAfterItsAttempts(t *testing.T) {
	svc, err := newProviderService("Test Provider", garudaFixture)
	if err != nil {
		t.Fatalf("newProviderService() error = %v", err)
	}

	calls := 0
	down := func() ([]FlightData, error) {
		calls++

		return nil, fmt.Errorf("%s: %w", svc.name, ErrUnavailable)
	}

	_, _, err = svc.search(context.Background(), request("CGK", "DPS"), down)
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("search() error = %v, want %v", err, ErrUnavailable)
	}
	if calls != maxAttempts {
		t.Errorf("search() called the provider %d times, want %d attempts", calls, maxAttempts)
	}
}

func TestSearchDoesNotRetryAnotherError(t *testing.T) {
	svc, err := newProviderService("Test Provider", garudaFixture)
	if err != nil {
		t.Fatalf("newProviderService() error = %v", err)
	}

	broken := errors.New("decode response: unexpected end of JSON input")
	calls := 0

	_, _, err = svc.search(context.Background(), request("CGK", "DPS"), func() ([]FlightData, error) {
		calls++

		return nil, broken
	})
	if !errors.Is(err, broken) {
		t.Fatalf("search() error = %v, want %v", err, broken)
	}
	if calls != 1 {
		t.Errorf("search() called the provider %d times, want 1: only an outage is retried", calls)
	}
}

func TestSearchStopsRetryingWhenTheCallerIsGone(t *testing.T) {
	svc, err := newProviderService("Test Provider", garudaFixture)
	if err != nil {
		t.Fatalf("newProviderService() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	calls := 0

	_, _, err = svc.search(ctx, request("CGK", "DPS"), func() ([]FlightData, error) {
		calls++

		return nil, fmt.Errorf("%s: %w", svc.name, ErrUnavailable)
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("search() error = %v, want context.Canceled", err)
	}
	if calls != 1 {
		t.Errorf("search() called the provider %d times, want 1: a cancelled search is not retried", calls)
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
