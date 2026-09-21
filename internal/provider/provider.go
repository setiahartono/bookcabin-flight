package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"bookcabin-flight/data"
	"bookcabin-flight/internal/cache"
)

var ErrUnavailable = errors.New("provider unavailable")

// CacheTTL is how long a provider keeps what it answered, so a repeated search
// does not have to wait for the simulated call again, or fail it again.
const CacheTTL = 10 * time.Second

type SearchRequest struct {
	Origin        string
	Destination   string
	DepartureDate time.Time
	Passengers    int
	CabinClass    string
}

type providerService struct {
	name  string
	body  []byte
	cache *cache.Cache[[]FlightData]
}

func newProviderService(name, fixture string) (*providerService, error) {
	body, err := data.FS.ReadFile(fixture)
	if err != nil {
		return nil, fmt.Errorf("provider %s: read fixture %s: %w", name, fixture, err)
	}

	return &providerService{
		name:  name,
		body:  body,
		cache: cache.New[[]FlightData](CacheTTL),
	}, nil
}

// search answers a request from what the provider kept for it, or runs the
// simulated call it is given and keeps what that answered, so a repeated search
// needs neither the latency nor the failure chance of the provider again.
// Only an answer is kept, a failed call is asked for again on the next search.
func (s *providerService) search(req SearchRequest, call func() ([]FlightData, error)) ([]FlightData, bool, error) {
	key := cacheKey(req)

	if flights, ok := s.cache.Get(key); ok {
		return flights, true, nil
	}

	flights, err := call()
	if err != nil {
		return nil, false, err
	}

	s.cache.Set(key, flights)

	return flights, false, nil
}

// cacheKey names what the provider answered for a request: the route, the date,
// the travellers and the cabin class it would carry them in.
func cacheKey(req SearchRequest) string {
	return fmt.Sprintf("%s|%s|%s|%d|%s",
		strings.ToUpper(strings.TrimSpace(req.Origin)),
		strings.ToUpper(strings.TrimSpace(req.Destination)),
		req.DepartureDate.Format(time.DateOnly),
		req.Passengers,
		strings.ToUpper(strings.TrimSpace(req.CabinClass)),
	)
}

func (s *providerService) Name() string { return s.name }

func (s *providerService) wait(ctx context.Context, minDelay, maxDelay time.Duration) error {
	d := minDelay
	if maxDelay > minDelay {
		d = minDelay + time.Duration(rand.Int64N(int64(maxDelay-minDelay)))
	}

	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return fmt.Errorf("%s: %w", s.name, ctx.Err())
	}
}

func (s *providerService) maybeFail(successRate float64) error {
	if rand.Float64() < successRate {
		return nil
	}

	return fmt.Errorf("%s: %w", s.name, ErrUnavailable)
}

func (s *providerService) decode(out any) error {
	if err := json.Unmarshal(s.body, out); err != nil {
		return fmt.Errorf("%s: decode response: %w", s.name, err)
	}

	return nil
}
