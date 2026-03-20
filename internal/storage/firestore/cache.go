package firestore

import (
	"sync"
	"time"
)

// ttlCache is a simple thread-safe cache with time-based expiry.
// Used to avoid repeated full collection scans for dashboard aggregation endpoints.
type ttlCache[T any] struct {
	mu     sync.RWMutex
	value  *T
	expiry time.Time
	ttl    time.Duration
}

// newTTLCache creates a cache with the given TTL.
func newTTLCache[T any](ttl time.Duration) *ttlCache[T] {
	return &ttlCache[T]{ttl: ttl}
}

// get returns the cached value if it exists and hasn't expired.
// Returns (value, true) on hit, (zero, false) on miss.
func (c *ttlCache[T]) get() (T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.value != nil && time.Now().Before(c.expiry) {
		return *c.value, true
	}
	var zero T
	return zero, false
}

// set stores a value in the cache with the configured TTL.
func (c *ttlCache[T]) set(v T) {
	c.mu.Lock()
	c.value = &v
	c.expiry = time.Now().Add(c.ttl)
	c.mu.Unlock()
}

// invalidate clears the cached value.
func (c *ttlCache[T]) invalidate() {
	c.mu.Lock()
	c.value = nil
	c.mu.Unlock()
}
