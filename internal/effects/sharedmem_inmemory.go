// Package effects provides InMemorySharedCache, a thread-safe in-memory cache.
// Part of M-DX15 (Semantic Caching MVP).
package effects

import (
	"bytes"
	"sync"
)

// InMemorySharedCache is a thread-safe in-memory implementation of SharedCache.
//
// Features:
//   - Thread-safe: all operations protected by RWMutex
//   - Copy-on-read: Get returns a copy to prevent buffer sharing bugs
//   - Copy-on-write: Put copies input to prevent caller mutations
//   - Atomic CAS: compare-and-swap is a single locked operation
//
// Memory management:
//   - No automatic eviction (suitable for bounded workloads)
//   - Use Delete to manually remove entries
//   - For production, consider Redis-backed implementation
type InMemorySharedCache struct {
	mu    sync.RWMutex
	store map[string][]byte
}

// NewInMemorySharedCache creates a new empty in-memory cache.
//
// Returns:
//   - A new InMemorySharedCache ready for use
func NewInMemorySharedCache() *InMemorySharedCache {
	return &InMemorySharedCache{
		store: make(map[string][]byte),
	}
}

// Get retrieves a value by key.
//
// Returns (nil, false) if the key doesn't exist.
// Returns (copy of value, true) if the key exists.
//
// The returned bytes are a defensive copy - callers can safely modify them
// without affecting the cached value.
func (c *InMemorySharedCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	val, ok := c.store[key]
	if !ok {
		return nil, false
	}

	// Defensive copy - prevent caller from mutating cached data
	cpy := make([]byte, len(val))
	copy(cpy, val)
	return cpy, true
}

// Put stores a value at the given key.
//
// Overwrites any existing value. The value is copied before storage -
// callers can safely modify the input after Put returns.
func (c *InMemorySharedCache) Put(key string, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Defensive copy - prevent caller from mutating cached data
	cpy := make([]byte, len(value))
	copy(cpy, value)
	c.store[key] = cpy
}

// Delete removes a value by key.
//
// No-op if the key doesn't exist.
func (c *InMemorySharedCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.store, key)
}

// CAS performs an atomic compare-and-swap operation.
//
// The operation atomically:
//  1. Reads the current value
//  2. Compares it to oldValue (byte-for-byte)
//  3. If equal, writes newValue and returns true
//  4. If not equal, returns false (no write)
//
// Special case: if oldValue is nil, CAS creates the key only if it doesn't exist.
// This enables "create-if-absent" semantics for new entries.
//
// Parameters:
//   - key: The key to update
//   - oldValue: Expected current value (nil = key must not exist)
//   - newValue: New value to write if oldValue matches
//
// Returns:
//   - true if the swap succeeded
//   - false if the current value didn't match oldValue
func (c *InMemorySharedCache) CAS(key string, oldValue, newValue []byte) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	current, exists := c.store[key]

	// Special case: oldValue=nil means "create if absent"
	if oldValue == nil {
		if exists {
			return false // Key exists, CAS fails
		}
		// Key doesn't exist, create it
		cpy := make([]byte, len(newValue))
		copy(cpy, newValue)
		c.store[key] = cpy
		return true
	}

	// Normal case: compare current value to oldValue
	if !exists {
		return false // Key doesn't exist but oldValue is non-nil
	}

	if !bytes.Equal(current, oldValue) {
		return false // Current value doesn't match expected
	}

	// Values match - perform the swap
	cpy := make([]byte, len(newValue))
	copy(cpy, newValue)
	c.store[key] = cpy
	return true
}

// Keys returns all keys in the cache.
//
// The returned slice is a snapshot - adding/removing keys after this call
// won't affect the returned slice, and modifying the slice won't affect the cache.
func (c *InMemorySharedCache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.store))
	for k := range c.store {
		keys = append(keys, k)
	}
	return keys
}

// Len returns the number of entries in the cache.
func (c *InMemorySharedCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.store)
}
