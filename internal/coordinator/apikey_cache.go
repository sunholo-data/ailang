package coordinator

import (
	"sync"
	"time"
)

// APIKeyCache is a thread-safe in-memory cache for user-provided API keys.
// Keys are stored with a TTL and deleted after retrieval (one-time use).
// M-CLOUD-DUAL-AUTH: API keys NEVER touch Firestore or Secret Manager.
type APIKeyCache struct {
	mu      sync.Mutex
	entries map[string]apiKeyCacheEntry
	ttl     time.Duration
	stop    chan struct{}
}

type apiKeyCacheEntry struct {
	key      string
	storedAt time.Time
}

// NewAPIKeyCache creates a cache with the given TTL. Call Close to stop the
// background cleanup goroutine; tests must defer Close to avoid goroutine
// leaks across the package's test suite.
func NewAPIKeyCache(ttl time.Duration) *APIKeyCache {
	c := &APIKeyCache{
		entries: make(map[string]apiKeyCacheEntry),
		ttl:     ttl,
		stop:    make(chan struct{}),
	}
	go c.cleanupLoop()
	return c
}

// Close stops the background cleanup goroutine. Safe to call multiple times.
func (c *APIKeyCache) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	select {
	case <-c.stop:
		// already closed
	default:
		close(c.stop)
	}
}

// Store saves an API key keyed by message ID.
func (c *APIKeyCache) Store(messageID, apiKey string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[messageID] = apiKeyCacheEntry{
		key:      apiKey,
		storedAt: time.Now(),
	}
}

// Retrieve returns the API key for a message ID and deletes it (one-time use).
// Returns ("", false) if not found or expired.
func (c *APIKeyCache) Retrieve(messageID string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[messageID]
	if !ok {
		return "", false
	}
	delete(c.entries, messageID)
	if time.Since(entry.storedAt) > c.ttl {
		return "", false
	}
	return entry.key, true
}

// cleanupLoop removes expired entries every minute until Close is called.
func (c *APIKeyCache) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-c.stop:
			return
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for id, entry := range c.entries {
				if now.Sub(entry.storedAt) > c.ttl {
					delete(c.entries, id)
				}
			}
			c.mu.Unlock()
		}
	}
}
