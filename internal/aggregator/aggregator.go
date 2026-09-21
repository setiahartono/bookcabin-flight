package aggregator

import (
	"context"
	"errors"
	"sync"

	"bookcabin-flight/internal/provider"
)

// Searcher is a flight provider the aggregator can query. It reports whether it
// answered from what it had kept for the request, which is how a search that asks
// no provider at all is recognised.
// Providers that normalize their response, such as *provider.AirAsia, satisfy it as they are.
type Searcher interface {
	Search(ctx context.Context, req provider.SearchRequest) ([]provider.FlightData, bool, error)
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

// Aggregate returns the flights every searcher answered for the request, and whether all of them
// could answer from what they had kept instead of asking their provider again.
// Provider failing doesn't compromise the aggregation process
// Provider that successfully answered will be aggregated
func (a *Aggregator) Aggregate(ctx context.Context, req provider.SearchRequest) ([]provider.FlightData, bool, error) {
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		found     = make([][]provider.FlightData, len(a.searchers))
		fromCache = make([]bool, len(a.searchers))
		errs      []error
	)

	for i, searcher := range a.searchers {
		wg.Add(1)

		go func(index int, s Searcher) {
			defer wg.Done()

			flights, cached, err := s.Search(ctx, req)
			if err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()

				return
			}

			found[index] = flights
			fromCache[index] = cached
		}(i, searcher)
	}

	wg.Wait()

	flights := make([]provider.FlightData, 0)
	for _, providerFlights := range found {
		flights = append(flights, providerFlights...)
	}

	// Only a search that needed no provider call at all is a cache hit.
	cacheHit := len(a.searchers) > 0

	for _, hit := range fromCache {
		if !hit {
			cacheHit = false

			break
		}
	}

	return flights, cacheHit, errors.Join(errs...)
}
