package coordinator

import (
	"testing"
	"time"
)

func TestAPIKeyCacheStoreAndRetrieve(t *testing.T) {
	c := NewAPIKeyCache(10 * time.Minute)
	defer c.Close()

	c.Store("msg-1", "sk-ant-api-key-1")

	key, ok := c.Retrieve("msg-1")
	if !ok {
		t.Fatal("expected key to be found")
	}
	if key != "sk-ant-api-key-1" {
		t.Errorf("key = %q, want %q", key, "sk-ant-api-key-1")
	}
}

func TestAPIKeyCacheOneTimeUse(t *testing.T) {
	c := NewAPIKeyCache(10 * time.Minute)
	defer c.Close()

	c.Store("msg-2", "sk-ant-api-key-2")

	// First retrieval succeeds
	_, ok := c.Retrieve("msg-2")
	if !ok {
		t.Fatal("first retrieve should succeed")
	}

	// Second retrieval fails (one-time use)
	_, ok = c.Retrieve("msg-2")
	if ok {
		t.Fatal("second retrieve should fail (one-time use)")
	}
}

func TestAPIKeyCacheExpiry(t *testing.T) {
	// Use a very short TTL for testing
	c := &APIKeyCache{
		entries: make(map[string]apiKeyCacheEntry),
		ttl:     1 * time.Millisecond,
	}

	c.Store("msg-3", "sk-ant-api-key-3")

	// Wait for expiry
	time.Sleep(5 * time.Millisecond)

	_, ok := c.Retrieve("msg-3")
	if ok {
		t.Fatal("expected key to be expired")
	}
}

func TestAPIKeyCacheMissing(t *testing.T) {
	c := NewAPIKeyCache(10 * time.Minute)
	defer c.Close()

	_, ok := c.Retrieve("nonexistent")
	if ok {
		t.Fatal("expected miss for nonexistent key")
	}
}
