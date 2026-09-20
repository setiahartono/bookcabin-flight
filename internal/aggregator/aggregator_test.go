package aggregator

import (
	"context"
	"errors"
	"testing"
	"time"

	"bookcabin-flight/internal/provider"
)

var _ Searcher = (*provider.AirAsia)(nil)
var _ Searcher = (*provider.LionAir)(nil)
var _ Searcher = (*provider.Garuda)(nil)

type fakeSearcher struct {
	flights []provider.FlightData
	err     error
	delay   time.Duration
	calls   int
}

func (f *fakeSearcher) Search(ctx context.Context, _ provider.SearchRequest) ([]provider.FlightData, error) {
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	f.calls++

	return f.flights, f.err
}

func TestSearchCollectsFlightsFromEverySearcher(t *testing.T) {
	airAsia := &fakeSearcher{flights: []provider.FlightData{{Id: "QZ520_AirAsia"}, {Id: "QZ524_AirAsia"}}}
	lionAir := &fakeSearcher{flights: []provider.FlightData{{Id: "JT740_Lion Air"}}}

	got, err := New(airAsia, lionAir).Aggregate(context.Background(), provider.SearchRequest{})
	if err != nil {
		t.Fatalf("Search() error = %v, want nil", err)
	}

	if len(got) != 3 {
		t.Fatalf("len(Search()) = %d, want 3", len(got))
	}

	ids := map[string]int{}

	for _, flight := range got {
		ids[flight.Id]++
	}

	for _, want := range []string{"QZ520_AirAsia", "QZ524_AirAsia", "JT740_Lion Air"} {
		if ids[want] != 1 {
			t.Errorf("ids[%q] = %d, want 1", want, ids[want])
		}
	}

	if airAsia.calls != 1 {
		t.Errorf("AirAsia calls = %d, want 1", airAsia.calls)
	}
	if lionAir.calls != 1 {
		t.Errorf("Lion Air calls = %d, want 1", lionAir.calls)
	}
}

func TestSearchKeepsFlightsWhenASearcherFails(t *testing.T) {
	unavailable := errors.New("provider unavailable")

	airAsia := &fakeSearcher{flights: []provider.FlightData{{Id: "QZ520_AirAsia"}}}
	batikAir := &fakeSearcher{err: unavailable}

	got, err := New(airAsia, batikAir).Aggregate(context.Background(), provider.SearchRequest{})

	if !errors.Is(err, unavailable) {
		t.Errorf("Search() error = %v, want %v", err, unavailable)
	}
	if len(got) != 1 {
		t.Fatalf("len(Search()) = %d, want 1", len(got))
	}
	if got[0].Id != "QZ520_AirAsia" {
		t.Errorf("Id = %q, want %q", got[0].Id, "QZ520_AirAsia")
	}
}

func TestSearchWithoutSearchers(t *testing.T) {
	got, err := New().Aggregate(context.Background(), provider.SearchRequest{})

	if err != nil {
		t.Errorf("Search() error = %v, want nil", err)
	}
	if len(got) != 0 {
		t.Errorf("len(Search()) = %d, want 0", len(got))
	}
}

func TestSearchQueriesSearchersInParallel(t *testing.T) {
	first := &fakeSearcher{flights: []provider.FlightData{{Id: "QZ520_AirAsia"}}, delay: 200 * time.Millisecond}
	second := &fakeSearcher{flights: []provider.FlightData{{Id: "JT740_Lion Air"}}, delay: 200 * time.Millisecond}

	start := time.Now()
	got, err := New(first, second).Aggregate(context.Background(), provider.SearchRequest{})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Search() error = %v, want nil", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(Search()) = %d, want 2", len(got))
	}
	if elapsed > 350*time.Millisecond {
		t.Errorf("Search() took %v, want the searchers to overlap", elapsed)
	}
}

func TestSearchPropagatesContext(t *testing.T) {
	airAsia := &fakeSearcher{flights: []provider.FlightData{{Id: "QZ520_AirAsia"}}, delay: time.Second}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got, err := New(airAsia).Aggregate(ctx, provider.SearchRequest{})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Search() error = %v, want context.Canceled", err)
	}
	if len(got) != 0 {
		t.Errorf("len(Search()) = %d, want 0", len(got))
	}
}
