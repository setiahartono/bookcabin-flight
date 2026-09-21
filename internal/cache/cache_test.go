package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestCacheKeepsAValue(t *testing.T) {
	c := New[[]string](time.Minute)
	c.Set("CGK-DPS", []string{"QZ520_AirAsia"})

	got, ok := c.Get("CGK-DPS")
	if !ok {
		t.Fatal("Get() found nothing, want the value that was set")
	}
	if len(got) != 1 || got[0] != "QZ520_AirAsia" {
		t.Errorf("Get() = %v, want the value that was set", got)
	}
}

func TestCacheWithoutAValue(t *testing.T) {
	c := New[string](time.Minute)

	if got, ok := c.Get("CGK-DPS"); ok {
		t.Errorf("Get() = %q, true, want a miss", got)
	}
}

func TestCacheForgetsAnExpiredValue(t *testing.T) {
	c := New[string](time.Millisecond)
	c.Set("CGK-DPS", "QZ520_AirAsia")

	time.Sleep(5 * time.Millisecond)

	if got, ok := c.Get("CGK-DPS"); ok {
		t.Errorf("Get() = %q, true, want the value to have expired", got)
	}
}

func TestCacheKeepsValuesAPart(t *testing.T) {
	c := New[string](time.Minute)
	c.Set("CGK-DPS", "QZ520_AirAsia")
	c.Set("CGK-SUB", "GA315_Garuda Indonesia")

	if got, _ := c.Get("CGK-DPS"); got != "QZ520_AirAsia" {
		t.Errorf("Get(CGK-DPS) = %q, want %q", got, "QZ520_AirAsia")
	}
	if got, _ := c.Get("CGK-SUB"); got != "GA315_Garuda Indonesia" {
		t.Errorf("Get(CGK-SUB) = %q, want %q", got, "GA315_Garuda Indonesia")
	}
}

func TestCacheWithoutATTLKeepsNothing(t *testing.T) {
	c := New[string](0)
	c.Set("CGK-DPS", "QZ520_AirAsia")

	if got, ok := c.Get("CGK-DPS"); ok {
		t.Errorf("Get() = %q, true, want a cache that keeps nothing", got)
	}
}

func TestZeroCacheKeepsNothing(t *testing.T) {
	var c Cache[string]

	c.Set("CGK-DPS", "QZ520_AirAsia")

	if got, ok := c.Get("CGK-DPS"); ok {
		t.Errorf("Get() = %q, true, want a zero cache to keep nothing", got)
	}

	var nilCache *Cache[string]
	if got, ok := nilCache.Get("CGK-DPS"); ok {
		t.Errorf("nil cache Get() = %q, true, want a miss", got)
	}
}

func TestCacheIsSafeForConcurrentUse(t *testing.T) {
	c := New[string](time.Minute)

	var wg sync.WaitGroup

	for i := range 50 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			key := fmt.Sprintf("key-%d", i%5)
			c.Set(key, key)

			if got, ok := c.Get(key); ok && got != key {
				t.Errorf("Get(%q) = %q, want %q", key, got, key)
			}
		}()
	}

	wg.Wait()
}
