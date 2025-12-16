// Package effects provides the SharedMem effect for shared memory caching.
// Part of M-DX15 (Semantic Caching MVP).
package effects

import (
	"sync"
)

// SharedCache is the interface for shared memory cache backends.
//
// All operations are thread-safe. Implementations must ensure:
//   - Get/Put/Delete are atomic
//   - CAS (compare-and-swap) is atomic
//   - Copy-on-read/write to prevent buffer sharing bugs
//
// The cache stores arbitrary bytes keyed by string IDs.
// For sem_frame storage, use JSON encoding via std/sem.encode_frame.
type SharedCache interface {
	// Get retrieves a value by key.
	// Returns (nil, false) if the key doesn't exist.
	// Returns (value, true) if the key exists.
	// The returned bytes are a copy - callers can safely modify them.
	Get(key string) ([]byte, bool)

	// Put stores a value at the given key.
	// Overwrites any existing value.
	// The value is copied - callers can safely modify the input after Put returns.
	Put(key string, value []byte)

	// Delete removes a value by key.
	// No-op if the key doesn't exist.
	Delete(key string)

	// CAS performs a compare-and-swap operation.
	// Returns true if the swap succeeded (oldValue matched current value).
	// Returns false if the current value != oldValue (another writer modified it).
	//
	// Special case: if oldValue is nil, CAS creates the key only if it doesn't exist.
	// This enables "create-if-absent" semantics for new entries.
	CAS(key string, oldValue, newValue []byte) bool

	// Keys returns all keys in the cache.
	// The returned slice is a snapshot - modifications don't affect the cache.
	Keys() []string

	// Len returns the number of entries in the cache.
	Len() int
}

// SharedMemContext holds the runtime state for the SharedMem effect.
//
// The context provides access to the shared cache and tracks statistics
// for debugging and monitoring.
type SharedMemContext struct {
	Cache SharedCache // The underlying cache implementation

	// Statistics (atomic updates, read-only access is safe)
	mu         sync.Mutex
	GetCount   int64 // Number of Get operations
	PutCount   int64 // Number of Put operations
	CASCount   int64 // Number of CAS operations
	CASSuccess int64 // Number of successful CAS operations
}

// NewSharedMemContext creates a new SharedMem context with the given cache.
//
// If cache is nil, a new InMemorySharedCache is created.
//
// Parameters:
//   - cache: The SharedCache implementation to use (nil for default in-memory)
//
// Returns:
//   - A new SharedMemContext ready for use
func NewSharedMemContext(cache SharedCache) *SharedMemContext {
	if cache == nil {
		cache = NewInMemorySharedCache()
	}
	return &SharedMemContext{
		Cache: cache,
	}
}

// IncrGetCount increments the Get counter (thread-safe)
func (ctx *SharedMemContext) IncrGetCount() {
	ctx.mu.Lock()
	ctx.GetCount++
	ctx.mu.Unlock()
}

// IncrPutCount increments the Put counter (thread-safe)
func (ctx *SharedMemContext) IncrPutCount() {
	ctx.mu.Lock()
	ctx.PutCount++
	ctx.mu.Unlock()
}

// IncrCASCount increments the CAS counter and optionally the success counter (thread-safe)
func (ctx *SharedMemContext) IncrCASCount(success bool) {
	ctx.mu.Lock()
	ctx.CASCount++
	if success {
		ctx.CASSuccess++
	}
	ctx.mu.Unlock()
}

// Stats returns a snapshot of the statistics
func (ctx *SharedMemContext) Stats() (gets, puts, cas, casSuccess int64) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	return ctx.GetCount, ctx.PutCount, ctx.CASCount, ctx.CASSuccess
}
