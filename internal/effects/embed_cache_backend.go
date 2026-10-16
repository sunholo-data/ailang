package effects

import (
	"fmt"
	"sync"
)

// EmbeddingCache is an optional durable content-hash store. Misses are distinct
// from errors; a broken database must never masquerade as a cache miss.
type EmbeddingCache interface {
	Get(model, hash string, dimension int) ([]float32, bool, error)
	Put(model, hash string, vector []float32) error
	Close() error
}
type EmbeddingCacheOpener func(string) (EmbeddingCache, error)

var embedCacheBackend struct {
	sync.RWMutex
	open EmbeddingCacheOpener
}

func RegisterEmbeddingCacheOpener(open EmbeddingCacheOpener) {
	embedCacheBackend.Lock()
	defer embedCacheBackend.Unlock()
	embedCacheBackend.open = open
}
func OpenEmbeddingCache(path string) (EmbeddingCache, error) {
	embedCacheBackend.RLock()
	open := embedCacheBackend.open
	embedCacheBackend.RUnlock()
	if open == nil {
		return nil, fmt.Errorf("embedding cache backend not registered")
	}
	return open(path)
}
