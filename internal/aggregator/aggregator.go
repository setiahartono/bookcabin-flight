package aggregator

import (
	"context"
	"errors"
	"sync"

	"bookcabin-flight/internal/provider"
)

// Searcher is a flight provider the aggregator can query.
// Providers that normalize their response, such as *provider.AirAsia, satisfy it as they are.
type Searcher interface {
	Search(ctx context.Context, req provider.SearchRequest) ([]provider.FlightData, error)
}

// Aggregator collects the flight data answered by every searcher it holds.
type Aggregator struct {
	searchers []Searcher
}

// New returns an Aggregator that queries the given searchers.
func New(searchers ...Searcher) *Aggregator {
	return &Aggregator{searchers: searchers}
}

// Count returns how many searchers the aggregator queries.
func (a *Aggregator) Count() int {
	return len(a.searchers)
}

// Aggregate queries every searcher in parallel and returns the flights they
// answered with. A searcher that fails does not hide the others: its error is
// joined into the returned error while the flights of the searchers that
// answered are still returned.
//
// Each searcher writes to its own slot of the result, so the returned flights
// follow the order the searchers were given to New and stay stable between
// calls, no matter which provider answers first.
func (a *Aggregator) Aggregate(ctx context.Context, req provider.SearchRequest) ([]provider.FlightData, error) {
	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		found = make([][]provider.FlightData, len(a.searchers))
		errs  []error
	)

	for i, searcher := range a.searchers {
		wg.Add(1)

		go func(index int, s Searcher) {
			defer wg.Done()

			flights, err := s.Search(ctx, req)
			if err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()

				return
			}

			found[index] = flights
		}(i, searcher)
	}

	wg.Wait()

	flights := make([]provider.FlightData, 0)
	for _, providerFlights := range found {
		flights = append(flights, providerFlights...)
	}

	return flights, errors.Join(errs...)
}
