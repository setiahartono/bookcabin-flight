package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bookcabin-flight/internal/aggregator"
	"bookcabin-flight/internal/filter"
	"bookcabin-flight/internal/provider"
	"bookcabin-flight/internal/scoring"
)

var ErrInvalidCriteria = errors.New("invalid search criteria")

type SearchCriteria = filter.SearchCriteria

type SearchResult struct {
	SearchCriteria SearchCriteria        `json:"search_criteria"`
	Metadata       metadata              `json:"metadata"`
	Flights        []provider.FlightData `json:"flights"`
}

type metadata struct {
	TotalResults     int  `json:"total_results"`
	ProvidersQueried int  `json:"providers_queried"`
	ProvidersFailed  int  `json:"providers_failed"`
	SearchTimeMs     int  `json:"search_time_ms"`
	CacheHit         bool `json:"cache_hit"`
}

type SearchService struct {
	aggregator *aggregator.Aggregator
}

func loadProviders() []aggregator.Searcher {
	searchers := make([]aggregator.Searcher, 0)

	if airAsia, err := provider.NewAirAsia(); err == nil {
		searchers = append(searchers, airAsia)
	}
	if batikAir, err := provider.NewBatikAir(); err == nil {
		searchers = append(searchers, batikAir)
	}
	if lionAir, err := provider.NewLionAir(); err == nil {
		searchers = append(searchers, lionAir)
	}
	if garuda, err := provider.NewGaruda(); err == nil {
		searchers = append(searchers, garuda)
	}

	return searchers
}

func NewSearchService() *SearchService {
	return newService(loadProviders()...)
}

// newService wires the aggregator over the given searchers, which lets the
// service tests query providers of their own.
func newService(searchers ...aggregator.Searcher) *SearchService {
	return &SearchService{
		aggregator: aggregator.New(searchers...),
	}
}

// Count how many providers are failing
func failureCount(err error) int {
	if err == nil {
		return 0
	}

	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		return len(joined.Unwrap())
	}

	return 1
}

func (s *SearchService) Search(ctx context.Context, criteria SearchCriteria) (SearchResult, error) {
	start := time.Now()

	departureDate, err := time.Parse(time.DateOnly, criteria.DepartureDate)
	if err != nil {
		return SearchResult{}, fmt.Errorf("%w: departureDate %q", ErrInvalidCriteria, criteria.DepartureDate)
	}

	flights, aggregateErr := s.aggregator.Aggregate(ctx, provider.SearchRequest{
		Origin:        criteria.Origin,
		Destination:   criteria.Destination,
		DepartureDate: departureDate,
		Passengers:    criteria.Passengers,
		CabinClass:    criteria.CabinClass,
	})

	// Only the flights that match the criteria take part in the scoring, and the
	// result lists them best value first.
	filtered := filter.FilterFlights(flights, criteria)
	ranked := scoring.Rank(filtered)

	result := SearchResult{
		SearchCriteria: criteria,
		Flights:        ranked,
		Metadata: metadata{
			TotalResults:     len(ranked),
			ProvidersQueried: s.aggregator.Count(),
			ProvidersFailed:  failureCount(aggregateErr),
			SearchTimeMs:     int(time.Since(start).Milliseconds()),
		},
	}

	return result, nil
}
