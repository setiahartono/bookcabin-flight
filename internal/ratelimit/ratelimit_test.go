package ratelimit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestLimiterAllowsUpToTheLimit(t *testing.T) {
	limiter := New(3, 5*time.Second)

	for request := 1; request <= 3; request++ {
		if allowed, _ := limiter.Allow("10.0.0.1"); !allowed {
			t.Fatalf("Allow() = false on request %d, want true", request)
		}
	}

	allowed, retryAfter := limiter.Allow("10.0.0.1")
	if allowed {
		t.Fatal("Allow() = true on the fourth request, want false")
	}
	if retryAfter <= 0 || retryAfter > 5*time.Second {
		t.Errorf("Allow() retry after = %v, want the rest of the window", retryAfter)
	}
}

func TestLimiterCountsVisitorsApart(t *testing.T) {
	limiter := New(3, 5*time.Second)

	for _, visitor := range []string{"10.0.0.1", "10.0.0.2"} {
		for request := 1; request <= 3; request++ {
			if allowed, _ := limiter.Allow(visitor); !allowed {
				t.Fatalf("Allow(%q) = false on request %d, want true", visitor, request)
			}
		}
	}
}

func TestLimiterStartsOverInTheNextWindow(t *testing.T) {
	limiter := New(3, 5*time.Second)
	now := time.Date(2025, 12, 15, 12, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return now }

	for request := 1; request <= 3; request++ {
		if allowed, _ := limiter.Allow("10.0.0.1"); !allowed {
			t.Fatalf("Allow() = false on request %d, want true", request)
		}
	}
	if allowed, _ := limiter.Allow("10.0.0.1"); allowed {
		t.Fatal("Allow() = true on the fourth request, want false")
	}

	now = now.Add(5 * time.Second)

	if allowed, _ := limiter.Allow("10.0.0.1"); !allowed {
		t.Error("Allow() = false in the next window, want true")
	}
}

func TestLimiterWithoutALimitAllowsNothing(t *testing.T) {
	if allowed, _ := New(0, time.Second).Allow("10.0.0.1"); allowed {
		t.Error("Allow() = true without a limit, want false")
	}
}

func TestLimiterIsSafeForConcurrentUse(t *testing.T) {
	limiter := New(3, time.Minute)
	now := time.Date(2025, 12, 15, 12, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return now }

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		allowed int
	)

	for range 100 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			if ok, _ := limiter.Allow("10.0.0.1"); ok {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if allowed != 3 {
		t.Errorf("allowed = %d, want 3", allowed)
	}
}

func TestMiddlewarePassesAllowedRequestsOn(t *testing.T) {
	limiter := New(3, time.Minute)
	calls := 0

	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++

		w.WriteHeader(http.StatusOK)
	}))

	for request := 1; request <= 3; request++ {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/search", nil))

		if got := recorder.Code; got != http.StatusOK {
			t.Fatalf("status = %d on request %d, want %d", got, request, http.StatusOK)
		}
	}

	if calls != 3 {
		t.Errorf("next handler called %d times, want 3", calls)
	}
}

func TestMiddlewareTurnsDownRequestsOverTheLimit(t *testing.T) {
	limiter := New(3, 5*time.Second)

	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for range 3 {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/v1/search", nil))
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/search", nil))

	if got, want := recorder.Code, http.StatusTooManyRequests; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}

	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body %s: %v", recorder.Body, err)
	}
	if !strings.Contains(body.Error, "rate limit exceeded") {
		t.Errorf("error = %q, want a rate limit message", body.Error)
	}
	if got := recorder.Header().Get("Retry-After"); got == "" {
		t.Error("Retry-After header is missing, want the seconds to wait")
	}
}

func TestMiddlewareCountsForwardedVisitorsApart(t *testing.T) {
	limiter := New(3, time.Minute)

	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	search := func(forwarded string) int {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/search", nil)
		request.Header.Set("X-Forwarded-For", forwarded)

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)

		return recorder.Code
	}

	for request := 1; request <= 3; request++ {
		if got := search("10.0.0.1"); got != http.StatusOK {
			t.Fatalf("status = %d on request %d for the first visitor, want %d", got, request, http.StatusOK)
		}
		if got := search("10.0.0.2"); got != http.StatusOK {
			t.Fatalf("status = %d on request %d for the second visitor, want %d", got, request, http.StatusOK)
		}
	}

	if got := search("10.0.0.1"); got != http.StatusTooManyRequests {
		t.Errorf("status = %d for the visitor over its limit, want %d", got, http.StatusTooManyRequests)
	}
}
