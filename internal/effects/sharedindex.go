// Package effects provides the SharedIndex effect for similarity-based semantic retrieval.
// Part of M-DX16 (Deterministic Semantic Retrieval).
package effects

import (
	"sync"
	"time"
)

// timeNow is a package-level variable for testing (can be mocked).
var timeNow = time.Now

// DeterminismMode controls result ordering guarantees for similarity search.
type DeterminismMode int

const (
	// DeterminismStrict guarantees identical results for identical queries on fixed index state.
	// Uses deterministic tie-breaking: (score DESC, key ASC).
	// Slightly slower due to sorting overhead.
	DeterminismStrict DeterminismMode = iota

	// DeterminismBestEffort allows non-deterministic ordering when scores are equal.
	// Faster for large result sets but may produce different orderings between runs.
	DeterminismBestEffort
)

// String returns the string representation of the determinism mode.
func (m DeterminismMode) String() string {
	switch m {
	case DeterminismStrict:
		return "Strict"
	case DeterminismBestEffort:
		return "BestEffort"
	default:
		return "Unknown"
	}
}

// IndexEntry represents a single entry in the SharedIndex.
// Entries are keyed by (namespace, key) and contain similarity search metadata.
type IndexEntry struct {
	Namespace string    // Namespace for index partitioning
	Key       string    // Unique key within namespace
	SimHash   int64     // 64-bit SimHash for similarity matching
	Embedding []float64 // Optional: neural embedding vector (e.g., 768-dim from EmbeddingGemma)
	Version   int64     // Version for optimistic locking
	Timestamp int64     // Unix epoch timestamp in milliseconds
	Score     float64   // Similarity score (set by FindSimilar* methods)
}

// SearchResult represents a single result from similarity search.
// Returns key metadata only - caller loads full frame from SharedMem.
type SearchResult struct {
	Key       string  // The matching key
	Score     float64 // Similarity score: 1.0 - (hamming_distance / 64.0)
	Version   int64   // Version at time of index entry
	Timestamp int64   // Timestamp at time of index entry
}

// SharedIndex is the interface for similarity index backends.
//
// All operations are thread-safe. Implementations must ensure:
//   - Upsert/Delete are atomic within a namespace
//   - FindSimilarSimHash returns deterministic results in Strict mode
//   - Namespace isolation is enforced (operations don't cross namespace boundaries)
//
// The index stores minimal metadata for similarity search.
// Full frame data is stored in SharedMem; the index provides fast lookup.
type SharedIndex interface {
	// Upsert adds or updates an entry in the index.
	// If the key exists in the namespace, it is updated.
	// If the key doesn't exist, it is created.
	//
	// Parameters:
	//   - namespace: Index partition (e.g., "beliefs", "goals")
	//   - key: Unique identifier within namespace
	//   - simhash: 64-bit SimHash for similarity matching
	//   - version: Version number for optimistic locking
	//   - timestamp: Unix epoch timestamp in milliseconds
	Upsert(namespace, key string, simhash, version, timestamp int64)

	// Delete removes an entry from the index.
	// No-op if the key doesn't exist in the namespace.
	//
	// Parameters:
	//   - namespace: Index partition
	//   - key: Key to delete
	Delete(namespace, key string)

	// FindSimilarSimHash finds entries similar to the query simhash.
	//
	// Parameters:
	//   - namespace: Index partition to search (required)
	//   - querySimHash: The SimHash to find similar entries for
	//   - topK: Maximum number of results to return
	//   - maxScan: Maximum entries to scan (0 = unlimited, bounds O(N) search)
	//   - mode: DeterminismStrict or DeterminismBestEffort
	//
	// Returns:
	//   - Slice of SearchResult, ordered by score DESC (Strict: then key ASC)
	//   - Empty slice if no entries found
	//
	// Scoring formula: score = 1.0 - (hamming_distance / 64.0)
	// Score of 1.0 = identical SimHash, 0.0 = maximally different
	FindSimilarSimHash(namespace string, querySimHash int64, topK, maxScan int, mode DeterminismMode) []SearchResult

	// EntryCount returns the number of entries in a namespace.
	// Returns 0 if namespace doesn't exist.
	EntryCount(namespace string) int

	// Namespaces returns all namespace names in the index.
	Namespaces() []string

	// UpsertWithEmbedding adds or updates an entry with a neural embedding.
	// The embedding is stored alongside the SimHash for hybrid search.
	//
	// Parameters:
	//   - namespace: Index partition (e.g., "beliefs", "goals")
	//   - key: Unique identifier within namespace
	//   - simhash: 64-bit SimHash for fast similarity matching
	//   - embedding: Neural embedding vector (e.g., 768 floats from EmbeddingGemma)
	//   - version: Version number for optimistic locking
	//   - timestamp: Unix epoch timestamp in milliseconds
	UpsertWithEmbedding(namespace, key string, simhash int64, embedding []float64, version, timestamp int64)

	// FindSimilarByEmbedding finds entries similar to the query embedding using cosine similarity.
	//
	// Parameters:
	//   - namespace: Index partition to search (required)
	//   - queryEmbedding: The embedding vector to find similar entries for
	//   - topK: Maximum number of results to return
	//   - maxScan: Maximum entries to scan (0 = unlimited, bounds O(N) search)
	//   - mode: DeterminismStrict or DeterminismBestEffort
	//
	// Returns:
	//   - Slice of SearchResult, ordered by score DESC (Strict: then key ASC)
	//   - Empty slice if no entries found or no entries have embeddings
	//
	// Scoring: cosine similarity normalized to [0, 1]
	// Score of 1.0 = identical vectors, 0.0 = orthogonal vectors
	FindSimilarByEmbedding(namespace string, queryEmbedding []float64, topK, maxScan int, mode DeterminismMode) []SearchResult
}

// TraceEntry captures details of a SharedIndex operation for debugging and replay.
// Trace entries are only recorded when TraceEnabled is true.
type TraceEntry struct {
	Operation   string          `json:"op"`        // "upsert", "delete", "find_simhash"
	Namespace   string          `json:"namespace"` // Index partition
	Key         string          `json:"key,omitempty"`
	QueryHash   int64           `json:"query_hash,omitempty"`
	TopK        int             `json:"top_k,omitempty"`
	MaxScan     int             `json:"max_scan,omitempty"`
	Mode        DeterminismMode `json:"mode,omitempty"`
	ResultCount int             `json:"result_count,omitempty"`
	ChosenKey   string          `json:"chosen_key,omitempty"` // Only for resolve_best_match
	Timestamp   int64           `json:"ts"`                   // Unix nano timestamp
}

// SharedIndexContext holds the runtime state for the SharedIndex effect.
//
// The context provides access to the shared index and tracks statistics
// for debugging and monitoring.
type SharedIndexContext struct {
	Index SharedIndex // The underlying index implementation

	// Statistics (atomic updates, read-only access is safe)
	mu           sync.Mutex
	UpsertCount  int64 // Number of Upsert operations
	DeleteCount  int64 // Number of Delete operations
	SearchCount  int64 // Number of FindSimilarSimHash operations
	ScannedTotal int64 // Total entries scanned across all searches

	// Trace logging (optional, for debugging/replay)
	TraceEnabled bool         // If true, operations are logged to Trace
	Trace        []TraceEntry // Recorded operations (append-only)
}

// NewSharedIndexContext creates a new SharedIndex context with the given index.
//
// If index is nil, a new InMemorySharedIndex is created.
//
// Parameters:
//   - index: The SharedIndex implementation to use (nil for default in-memory)
//
// Returns:
//   - A new SharedIndexContext ready for use
func NewSharedIndexContext(index SharedIndex) *SharedIndexContext {
	if index == nil {
		index = NewInMemorySharedIndex()
	}
	return &SharedIndexContext{
		Index: index,
	}
}

// IncrementUpsert increments the upsert counter (thread-safe).
func (c *SharedIndexContext) IncrementUpsert() {
	c.mu.Lock()
	c.UpsertCount++
	c.mu.Unlock()
}

// IncrementDelete increments the delete counter (thread-safe).
func (c *SharedIndexContext) IncrementDelete() {
	c.mu.Lock()
	c.DeleteCount++
	c.mu.Unlock()
}

// IncrementSearch increments the search counter (thread-safe).
func (c *SharedIndexContext) IncrementSearch(scanned int64) {
	c.mu.Lock()
	c.SearchCount++
	c.ScannedTotal += scanned
	c.mu.Unlock()
}

// TraceUpsert records an upsert operation if tracing is enabled.
func (c *SharedIndexContext) TraceUpsert(namespace, key string) {
	if !c.TraceEnabled {
		return
	}
	c.mu.Lock()
	c.Trace = append(c.Trace, TraceEntry{
		Operation: "upsert",
		Namespace: namespace,
		Key:       key,
		Timestamp: timeNowNano(),
	})
	c.mu.Unlock()
}

// TraceDelete records a delete operation if tracing is enabled.
func (c *SharedIndexContext) TraceDelete(namespace, key string) {
	if !c.TraceEnabled {
		return
	}
	c.mu.Lock()
	c.Trace = append(c.Trace, TraceEntry{
		Operation: "delete",
		Namespace: namespace,
		Key:       key,
		Timestamp: timeNowNano(),
	})
	c.mu.Unlock()
}

// TraceFindSimHash records a similarity search operation if tracing is enabled.
func (c *SharedIndexContext) TraceFindSimHash(namespace string, queryHash int64, topK, maxScan int, mode DeterminismMode, resultCount int) {
	if !c.TraceEnabled {
		return
	}
	c.mu.Lock()
	c.Trace = append(c.Trace, TraceEntry{
		Operation:   "find_simhash",
		Namespace:   namespace,
		QueryHash:   queryHash,
		TopK:        topK,
		MaxScan:     maxScan,
		Mode:        mode,
		ResultCount: resultCount,
		Timestamp:   timeNowNano(),
	})
	c.mu.Unlock()
}

// TraceResolveBestMatch records a resolve operation with the chosen key if tracing is enabled.
func (c *SharedIndexContext) TraceResolveBestMatch(namespace string, queryHash int64, chosenKey string) {
	if !c.TraceEnabled {
		return
	}
	c.mu.Lock()
	c.Trace = append(c.Trace, TraceEntry{
		Operation: "resolve_best_match",
		Namespace: namespace,
		QueryHash: queryHash,
		ChosenKey: chosenKey,
		Timestamp: timeNowNano(),
	})
	c.mu.Unlock()
}

// TraceUpsertWithEmbedding records an upsert with embedding operation if tracing is enabled.
func (c *SharedIndexContext) TraceUpsertWithEmbedding(namespace, key string, embeddingDim int) {
	if !c.TraceEnabled {
		return
	}
	c.mu.Lock()
	c.Trace = append(c.Trace, TraceEntry{
		Operation: "upsert_embedding",
		Namespace: namespace,
		Key:       key,
		TopK:      embeddingDim, // Reuse TopK field for embedding dimension
		Timestamp: timeNowNano(),
	})
	c.mu.Unlock()
}

// TraceFindByEmbedding records an embedding-based similarity search operation if tracing is enabled.
func (c *SharedIndexContext) TraceFindByEmbedding(namespace string, embeddingDim, topK, maxScan int, mode DeterminismMode, resultCount int) {
	if !c.TraceEnabled {
		return
	}
	c.mu.Lock()
	c.Trace = append(c.Trace, TraceEntry{
		Operation:   "find_embedding",
		Namespace:   namespace,
		TopK:        topK,
		MaxScan:     maxScan,
		Mode:        mode,
		ResultCount: resultCount,
		Timestamp:   timeNowNano(),
	})
	c.mu.Unlock()
}

// EnableTracing enables trace logging for this context.
func (c *SharedIndexContext) EnableTracing() {
	c.mu.Lock()
	c.TraceEnabled = true
	c.mu.Unlock()
}

// DisableTracing disables trace logging for this context.
func (c *SharedIndexContext) DisableTracing() {
	c.mu.Lock()
	c.TraceEnabled = false
	c.mu.Unlock()
}

// GetTrace returns a copy of the trace entries.
func (c *SharedIndexContext) GetTrace() []TraceEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]TraceEntry, len(c.Trace))
	copy(result, c.Trace)
	return result
}

// ClearTrace removes all trace entries.
func (c *SharedIndexContext) ClearTrace() {
	c.mu.Lock()
	c.Trace = nil
	c.mu.Unlock()
}

// timeNowNano returns current Unix nanoseconds (package-level for testing).
var timeNowNano = func() int64 {
	return timeNow().UnixNano()
}
