// Package effects — persistent SharedCache seam.
//
// The language core defines the contract a persistent brain backend must
// satisfy (BrainCache), the frame types it stores, and the option surface;
// the SQLite implementation lives in internal/platform/sharedmem and is
// registered by the binary at startup (cmd/ailang/platform_init.go). With
// nothing registered, NewBrainStore returns ErrBackendNotRegistered — the
// core never links cgo sqlite. Part of M-V1-SIMPLIFICATION-PROGRAM Phase 1.3.
package effects

import (
	"encoding/binary"
	"fmt"
	"math"
	"sync"
	"time"
)

// Embedder is the interface for generating text embeddings.
// Matches messaging.Embedder but defined here to avoid circular imports.
type Embedder interface {
	Embed(text string) ([]float32, error)
	EmbedBatch(texts []string) ([][]float32, error)
	Dimension() int
	ModelName() string
}

// CacheOptions is the resolved option set a backend reads when opening a
// cache. Options are plain data so the option surface stays in the core while
// the implementation lives in the platform.
type CacheOptions struct {
	Embedder Embedder // optional, for auto-embedding on PutFrame
}

// CacheOption configures a cache at open time.
type CacheOption func(*CacheOptions)

// WithEmbedder sets an embedder for auto-embedding on PutFrame.
// When set, PutFrame will automatically compute and store embeddings
// for frames that have content but no embedding.
func WithEmbedder(e Embedder) CacheOption {
	return func(o *CacheOptions) {
		o.Embedder = e
	}
}

// ResolveCacheOptions applies opts to a zero CacheOptions. Backends call it.
func ResolveCacheOptions(opts ...CacheOption) CacheOptions {
	var o CacheOptions
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// BrainFrame represents a frame stored in the brain with its metadata.
type BrainFrame struct {
	Key          string
	Namespace    string
	Value        []byte
	SimHash      int64
	Content      string
	Version      int
	CreatedAt    int64
	UpdatedAt    int64
	ExpiresAt    *int64
	Source       string
	Embedding    []float32 // optional embedding vector
	EmbeddingDim int       // dimension of embedding (0 if none)
	EmbedModel   string    // model that produced the embedding
}

// BrainSearchResult represents a search hit with a relevance score.
type BrainSearchResult struct {
	Frame BrainFrame
	Score float64
	Tier  string // "user" or "project"
}

// BrainStats holds aggregate statistics about the brain.
type BrainStats struct {
	TotalFrames int
	Namespaces  map[string]int // namespace -> frame count
	OldestFrame int64          // unix millis
	NewestFrame int64          // unix millis
}

// BrainCache is what a registered persistent backend provides: the
// SharedCache byte contract plus the frame-level operations BrainStore
// composes across its two tiers.
type BrainCache interface {
	SharedCache
	Close() error

	PutFrame(f BrainFrame) error
	PutVector(key, namespace string, embedding []float32, model string, payload []byte) error
	GetFrame(key string) (*BrainFrame, bool)

	SearchBySimHash(namespace string, queryHash int64, limit int) []BrainSearchResult
	SearchByText(query string, namespace string, limit int) []BrainSearchResult
	SearchByEmbedding(queryEmbedding []float32, namespace string, limit int) []BrainSearchResult
	ListRecent(namespace string, limit int) []BrainFrame

	GarbageCollect() (int64, error)
	GarbageCollectOlderThan(namespace string, age time.Duration) (int64, error)
	DeleteNamespace(namespace string) (int64, error)

	Stats() BrainStats
	EmbeddingStats() (total, withEmbedding int, models map[string]int)

	SetEmbedder(e Embedder)
	GetEmbedder() Embedder
	BackfillEmbeddings(namespace string) (processed, errors int)
}

// SharedCacheOpener opens (creating if needed) the persistent cache at dbPath.
type SharedCacheOpener func(dbPath string, opts ...CacheOption) (BrainCache, error)

var (
	sharedCacheOpenerMu sync.RWMutex
	sharedCacheOpener   SharedCacheOpener
)

// RegisterSharedCacheOpener installs the persistent backend. Registering nil
// removes it. The platform registers at binary start; tests register per test.
func RegisterSharedCacheOpener(open SharedCacheOpener) {
	sharedCacheOpenerMu.Lock()
	defer sharedCacheOpenerMu.Unlock()
	sharedCacheOpener = open
}

// OpenSharedCache opens the persistent cache at dbPath through the registered
// backend. Without a registration it returns an error wrapping
// ErrBackendNotRegistered — never a nil cache.
func OpenSharedCache(dbPath string, opts ...CacheOption) (BrainCache, error) {
	sharedCacheOpenerMu.RLock()
	open := sharedCacheOpener
	sharedCacheOpenerMu.RUnlock()
	if open == nil {
		return nil, fmt.Errorf("shared cache backend for %s: %w", dbPath, ErrBackendNotRegistered)
	}
	return open(dbPath, opts...)
}

// EncodeEmbedding serializes a float32 slice to bytes using IEEE 754 little-endian encoding.
func EncodeEmbedding(v []float32) []byte {
	buf := make([]byte, len(v)*4)
	for i, f := range v {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

// DecodeEmbedding deserializes bytes back to a float32 slice (IEEE 754 little-endian).
func DecodeEmbedding(b []byte) []float32 {
	n := len(b) / 4
	v := make([]float32, n)
	for i := 0; i < n; i++ {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return v
}
