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
}

type apiKeyCacheEntry struct {
	key      string
	storedAt time.Time
}

// NewAPIKeyCache creates a cache with the given TTL.
func NewAPIKeyCache(ttl time.Duration) *APIKeyCache {
	c := &APIKeyCache{
		entries: make(map[string]apiKeyCacheEntry),
		ttl:     ttl,
	}
	go c.cleanupLoop()
	return c
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

// cleanupLoop removes expired entries every minute.
func (c *APIKeyCache) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
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
