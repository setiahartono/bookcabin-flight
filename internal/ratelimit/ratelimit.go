// Package ratelimit turns down the requests a visitor makes over its share.
package ratelimit

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// sweepThreshold is how many visitors the limiter tracks before it clears the
// ones whose window has ended.
const sweepThreshold = 128

// Limiter counts what every visitor asks for and turns down what is over the
// limit. It is safe to use from several goroutines.
type Limiter struct {
	limit  int
	window time.Duration

	mu    sync.Mutex
	now   func() time.Time
	usage map[string]usage
}

// usage is what one visitor asked for in its current window.
type usage struct {
	count int
	start time.Time
}

// New returns a Limiter that allows limit requests per visitor in every window.
func New(limit int, window time.Duration) *Limiter {
	return &Limiter{
		limit:  limit,
		window: window,
		now:    time.Now,
		usage:  make(map[string]usage),
	}
}

// Allow reports whether a visitor may make one more request, together with how
// long it has to wait before its window starts over when it may not.
func (l *Limiter) Allow(visitor string) (bool, time.Duration) {
	if l.limit <= 0 {
		return false, l.window
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()

	seen, ok := l.usage[visitor]
	if !ok || !now.Before(seen.start.Add(l.window)) {
		l.usage[visitor] = usage{count: 1, start: now}
		l.sweep(now)

		return true, 0
	}

	if seen.count >= l.limit {
		return false, seen.start.Add(l.window).Sub(now)
	}

	seen.count++
	l.usage[visitor] = seen

	return true, 0
}

// sweep drops the visitors whose window has ended, so the limiter does not keep
// growing with every visitor it has ever seen.
func (l *Limiter) sweep(now time.Time) {
	if len(l.usage) < sweepThreshold {
		return
	}

	for visitor, seen := range l.usage {
		if !now.Before(seen.start.Add(l.window)) {
			delete(l.usage, visitor)
		}
	}
}

// Middleware passes a request on while the visitor has requests left, and answers
// 429 Too Many Requests once it does not.
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed, retryAfter := l.Allow(visitor(r))
		if !allowed {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": fmt.Sprintf("rate limit exceeded: %d requests per %s", l.limit, l.window),
			})

			return
		}

		next.ServeHTTP(w, r)
	})
}

// visitor names who a request belongs to: the address a proxy forwarded, or the
// address of the connection when there is no proxy in front.
func visitor(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		address, _, _ := strings.Cut(forwarded, ",")

		return strings.TrimSpace(address)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}
