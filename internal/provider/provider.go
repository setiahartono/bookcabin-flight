package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"bookcabin-flight/data"
)

var ErrUnavailable = errors.New("provider unavailable")

type SearchRequest struct {
	Origin        string
	Destination   string
	DepartureDate time.Time
	Passengers    int
	CabinClass    string
}

type providerService struct {
	name string
	body []byte
}

func newProviderService(name, fixture string) (*providerService, error) {
	body, err := data.FS.ReadFile(fixture)
	if err != nil {
		return nil, fmt.Errorf("provider %s: read fixture %s: %w", name, fixture, err)
	}

	return &providerService{name: name, body: body}, nil
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
