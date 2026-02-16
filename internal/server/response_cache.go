package server

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

// responseCache provides simple TTL-based caching for expensive polling endpoints.
// This prevents repeated expensive database queries when the dashboard polls
// the same endpoints every few seconds.
type responseCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	ttl     time.Duration
}

type cacheEntry struct {
	data      []byte
	expiresAt time.Time
}

func newResponseCache(ttl time.Duration) *responseCache {
	return &responseCache{
		entries: make(map[string]*cacheEntry),
		ttl:     ttl,
	}
}

// cacheKey generates a cache key from the request URL (path + query string).
func cacheKey(r *http.Request) string {
	h := sha256.Sum256([]byte(r.URL.RequestURI()))
	return hex.EncodeToString(h[:8])
}

// get returns cached data if it exists and hasn't expired.
func (c *responseCache) get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.data, true
}

// set stores data in the cache with the configured TTL.
func (c *responseCache) set(key string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(c.ttl),
	}

	// Lazy cleanup: evict expired entries if cache grows large
	if len(c.entries) > 100 {
		now := time.Now()
		for k, e := range c.entries {
			if now.After(e.expiresAt) {
				delete(c.entries, k)
			}
		}
	}
}

// withCache wraps an HTTP handler with response caching.
// Cached responses are served directly; cache misses call the handler
// and cache the response for future requests.
func (c *responseCache) withCache(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := cacheKey(r)

		// Try cache first
		if data, ok := c.get(key); ok {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "HIT")
			w.Write(data)
			return
		}

		// Cache miss - capture the response
		rec := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}
		next.ServeHTTP(rec, r)

		// Only cache successful JSON responses
		if rec.statusCode == http.StatusOK && len(rec.body) > 0 {
			c.set(key, rec.body)
		}
	}
}

// responseRecorder captures the response body for caching.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       []byte
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body = append(r.body, b...)
	return r.ResponseWriter.Write(b)
}
