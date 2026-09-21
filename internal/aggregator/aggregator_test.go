package aggregator

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"bookcabin-flight/internal/provider"
)

var _ Searcher = (*provider.AirAsia)(nil)
var _ Searcher = (*provider.LionAir)(nil)
var _ Searcher = (*provider.Garuda)(nil)

type fakeSearcher struct {
	name    string
	flights []provider.FlightData
	err     error
	delay   time.Duration
	cached  bool
	calls   int
}

func (f *fakeSearcher) Name() string { return f.name }

func (f *fakeSearcher) Search(ctx context.Context, _ provider.SearchRequest) ([]provider.FlightData, bool, error) {
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return nil, false, ctx.Err()
		}
	}

	f.calls++

	return f.flights, f.cached, f.err
}

func TestSearchCollectsFlightsFromEverySearcher(t *testing.T) {
	airAsia := &fakeSearcher{flights: []provider.FlightData{{Id: "QZ520_AirAsia"}, {Id: "QZ524_AirAsia"}}}
	lionAir := &fakeSearcher{flights: []provider.FlightData{{Id: "JT740_Lion Air"}}}

	got, _, err := New(nil, airAsia, lionAir).Aggregate(context.Background(), provider.SearchRequest{})
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

	got, _, err := New(nil, airAsia, batikAir).Aggregate(context.Background(), provider.SearchRequest{})

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
	got, _, err := New(nil).Aggregate(context.Background(), provider.SearchRequest{})

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
	got, _, err := New(nil, first, second).Aggregate(context.Background(), provider.SearchRequest{})
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

	got, _, err := New(nil, airAsia).Aggregate(ctx, provider.SearchRequest{})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Search() error = %v, want context.Canceled", err)
	}
	if len(got) != 0 {
		t.Errorf("len(Search()) = %d, want 0", len(got))
	}
}

// request is a search of the route the fakes record their flights for.
func request(origin, destination string) provider.SearchRequest {
	departure, _ := time.Parse(time.DateOnly, "2025-12-15")

	return provider.SearchRequest{
		Origin:        origin,
		Destination:   destination,
		DepartureDate: departure,
		Passengers:    1,
		CabinClass:    "economy",
	}
}

func TestAggregateReportsACacheHitWhenEverySearcherAnsweredFromItsCache(t *testing.T) {
	airAsia := &fakeSearcher{flights: []provider.FlightData{{Id: "QZ520_AirAsia"}}, cached: true}
	lionAir := &fakeSearcher{flights: []provider.FlightData{{Id: "JT740_Lion Air"}}, cached: true}

	flights, cacheHit, err := New(nil, airAsia, lionAir).Aggregate(context.Background(), request("CGK", "DPS"))
	if err != nil {
		t.Fatalf("Aggregate() error = %v, want nil", err)
	}
	if !cacheHit {
		t.Error("Aggregate() cacheHit = false, want true when every searcher answered from its cache")
	}
	if len(flights) != 2 {
		t.Errorf("len(Aggregate()) = %d, want 2", len(flights))
	}
}

func TestAggregateReportsNoCacheHitWhileASearcherStillAsks(t *testing.T) {
	cached := &fakeSearcher{flights: []provider.FlightData{{Id: "QZ520_AirAsia"}}, cached: true}
	asked := &fakeSearcher{flights: []provider.FlightData{{Id: "JT740_Lion Air"}}}

	_, cacheHit, err := New(nil, cached, asked).Aggregate(context.Background(), request("CGK", "DPS"))
	if err != nil {
		t.Fatalf("Aggregate() error = %v, want nil", err)
	}
	if cacheHit {
		t.Error("Aggregate() cacheHit = true, want false while a searcher still asks its provider")
	}
}

// recorded is a provider failure the aggregator reported.
type recorded struct {
	provider      string
	origin        string
	destination   string
	departureDate time.Time
	err           error
}

// failureRecorder keeps what the aggregator reported, the way the error log does.
type failureRecorder struct {
	mu       sync.Mutex
	failures []recorded
}

func (r *failureRecorder) ProviderFailed(provider, origin, destination string, departureDate time.Time, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.failures = append(r.failures, recorded{provider, origin, destination, departureDate, err})
}

func (r *failureRecorder) failed() []recorded {
	r.mu.Lock()
	defer r.mu.Unlock()

	return slices.Clone(r.failures)
}

func TestAggregateReportsAFailedProvider(t *testing.T) {
	unavailable := errors.New("provider unavailable")
	airAsia := &fakeSearcher{name: "AirAsia", flights: []provider.FlightData{{Id: "QZ520_AirAsia"}}}
	batikAir := &fakeSearcher{name: "Batik Air", err: unavailable}
	recorder := &failureRecorder{}

	flights, _, err := New(recorder, airAsia, batikAir).Aggregate(context.Background(), request("CGK", "DPS"))
	if !errors.Is(err, unavailable) {
		t.Fatalf("Aggregate() error = %v, want %v", err, unavailable)
	}
	if len(flights) != 1 {
		t.Fatalf("len(Aggregate()) = %d, want 1", len(flights))
	}

	failures := recorder.failed()
	if len(failures) != 1 {
		t.Fatalf("reported failures = %d, want 1 for the provider that failed", len(failures))
	}

	failure := failures[0]

	if got, want := failure.provider, "Batik Air"; got != want {
		t.Errorf("failure provider = %q, want %q", got, want)
	}
	if got, want := failure.origin+"-"+failure.destination, "CGK-DPS"; got != want {
		t.Errorf("failure route = %q, want %q", got, want)
	}
	if got, want := failure.departureDate.Format(time.DateOnly), "2025-12-15"; got != want {
		t.Errorf("failure departure date = %q, want %q", got, want)
	}
	if !errors.Is(failure.err, unavailable) {
		t.Errorf("failure error = %v, want %v", failure.err, unavailable)
	}
}

func TestAggregateReportsNoFailureWhenEverySearcherAnswers(t *testing.T) {
	recorder := &failureRecorder{}
	airAsia := &fakeSearcher{name: "AirAsia", flights: []provider.FlightData{{Id: "QZ520_AirAsia"}}}

	if _, _, err := New(recorder, airAsia).Aggregate(context.Background(), request("CGK", "DPS")); err != nil {
		t.Fatalf("Aggregate() error = %v, want nil", err)
	}

	if got := recorder.failed(); len(got) != 0 {
		t.Errorf("reported failures = %v, want none when every provider answers", got)
	}
}
