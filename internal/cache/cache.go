// Package cache is a small in-memory store that keeps a value for a while.
package cache

import (
	"sync"
	"time"
)

// Cache keeps values under a key until they expire. It is safe to use from
// several goroutines. A time to live of zero or less turns it into a store that
// never keeps anything, and New has to be called to get a store that does.
type Cache[T any] struct {
	mu      sync.Mutex
	entries map[string]entry[T]
	ttl     time.Duration
}

type entry[T any] struct {
	value   T
	expires time.Time
}

// New returns a Cache that keeps the values it is given for ttl.
func New[T any](ttl time.Duration) *Cache[T] {
	return &Cache[T]{
		entries: make(map[string]entry[T]),
		ttl:     ttl,
	}
}

// Get returns the value of a key that is still fresh.
func (c *Cache[T]) Get(key string) (T, bool) {
	var zero T

	if c == nil || c.ttl <= 0 {
		return zero, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	found, ok := c.entries[key]
	if !ok {
		return zero, false
	}

	if !time.Now().Before(found.expires) {
		delete(c.entries, key)

		return zero, false
	}

	return found.value, true
}

// Set keeps a value under a key, replacing the value that was there.
func (c *Cache[T]) Set(key string, value T) {
	if c == nil || c.ttl <= 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.entries == nil {
		c.entries = make(map[string]entry[T])
	}

	c.entries[key] = entry[T]{value: value, expires: time.Now().Add(c.ttl)}
}
