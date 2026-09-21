package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
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
	ReturnFlights  []provider.FlightData `json:"return_flights"`
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

func NewSearchService(failures aggregator.Failures) *SearchService {
	return newService(failures, loadProviders()...)
}

// newService wires the aggregator over the given searchers and failure log, which
// lets the service tests query providers of their own.
func newService(failures aggregator.Failures, searchers ...aggregator.Searcher) *SearchService {
	return &SearchService{
		aggregator: aggregator.New(failures, searchers...),
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

// processFlightData collects the flights of one leg and returns them ready to be
// reported, ordered by the score component the search asked for, together with
// whether the providers answered from the cache.
func (s *SearchService) processFlightData(ctx context.Context, req provider.SearchRequest, criteria SearchCriteria, sortBy scoring.SortKey) ([]provider.FlightData, bool, error) {
	flights, cacheHit, aggregateErr := s.aggregator.Aggregate(ctx, req)

	// Only the flights that match the criteria take part in the scoring
	filtered := filter.FilterFlights(flights, criteria)
	ranked := scoring.Rank(filtered, sortBy)

	return ranked, cacheHit, aggregateErr
}

func (s *SearchService) Search(ctx context.Context, criteria SearchCriteria) (SearchResult, error) {
	start := time.Now()

	departureDate, err := time.Parse(time.DateOnly, criteria.DepartureDate)
	if err != nil {
		return SearchResult{}, fmt.Errorf(
			"%w: departureDate %q",
			ErrInvalidCriteria,
			criteria.DepartureDate,
		)
	}

	roundTrip := isRoundTrip(criteria)

	if roundTrip && criteria.ReturnDate == "" {
		return SearchResult{}, fmt.Errorf(
			"%w: returnDate is mandatory when roundTrip is true",
			ErrInvalidCriteria,
		)
	}

	sortBy, ok := scoring.ParseSortKey(criteria.SortBy)
	if !ok {
		return SearchResult{}, fmt.Errorf(
			"%w: sortBy %q, want %s, %s or %s",
			ErrInvalidCriteria,
			criteria.SortBy,
			scoring.SortByValue,
			scoring.SortByPrice,
			scoring.SortByConvenience,
		)
	}

	// The response echoes the criteria it applied, so a search that asked for no
	// order reports the one it was given.
	criteria.SortBy = string(sortBy)

	departureReq := provider.SearchRequest{
		Origin:        criteria.Origin,
		Destination:   criteria.Destination,
		DepartureDate: departureDate,
		Passengers:    criteria.Passengers,
		CabinClass:    criteria.CabinClass,
	}

	var (
		wg sync.WaitGroup

		rankedDeparture []provider.FlightData
		rankedReturn    []provider.FlightData
		departureCached bool
		returnCached    bool
		departureAggErr error
		returnAggErr    error
	)

	wg.Add(1)
	go func() {
		defer wg.Done()

		rankedDeparture, departureCached, departureAggErr =
			s.processFlightData(ctx, departureReq, criteria, sortBy)
	}()

	if roundTrip {
		returnDate, err := time.Parse(time.DateOnly, criteria.ReturnDate)
		if err != nil {
			return SearchResult{}, fmt.Errorf(
				"%w: returnDate %q",
				ErrInvalidCriteria,
				criteria.ReturnDate,
			)
		}

		returnReq := provider.SearchRequest{
			Origin:        criteria.Destination,
			Destination:   criteria.Origin,
			DepartureDate: returnDate,
			Passengers:    criteria.Passengers,
			CabinClass:    criteria.CabinClass,
		}
		returnCriteria := criteria
		returnCriteria.Origin = criteria.Destination
		returnCriteria.Destination = criteria.Origin
		returnCriteria.DepartureDate = criteria.ReturnDate

		wg.Add(1)
		go func() {
			defer wg.Done()

			rankedReturn, returnCached, returnAggErr =
				s.processFlightData(ctx, returnReq, returnCriteria, sortBy)
		}()
	}

	wg.Wait()

	// The way back asks every provider again, but those are the same providers: only
	// the flights they answer with are counted twice.
	totalResults := len(rankedDeparture) + len(rankedReturn)

	// The search only counts as a cache hit when the providers of every leg it
	// covered could answer from what they had kept.
	cacheHit := departureCached
	if roundTrip {
		cacheHit = cacheHit && returnCached
	}

	result := SearchResult{
		SearchCriteria: criteria,
		Flights:        rankedDeparture,
		ReturnFlights:  rankedReturn,
		Metadata: metadata{
			TotalResults:     totalResults,
			ProvidersQueried: s.aggregator.Count(),
			ProvidersFailed: failureCount(departureAggErr) +
				failureCount(returnAggErr),
			SearchTimeMs: int(time.Since(start).Milliseconds()),
			CacheHit:     cacheHit,
		},
	}

	return result, nil
}

// isRoundTrip reports whether the client asked for the way back.
func isRoundTrip(criteria SearchCriteria) bool {
	return criteria.RoundTrip != nil && *criteria.RoundTrip
}
