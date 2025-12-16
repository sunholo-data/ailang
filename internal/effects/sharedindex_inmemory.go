// Package effects provides the in-memory implementation of SharedIndex.
// Part of M-DX16 (Deterministic Semantic Retrieval).
package effects

import (
	"sort"
	"sync"
)

// InMemorySharedIndex is the default in-memory implementation of SharedIndex.
//
// Thread-safe for concurrent read/write access.
// Uses namespace-scoped maps with RWMutex for fine-grained locking.
//
// Implementation details:
//   - Entries stored in map[namespace]map[key]*IndexEntry
//   - FindSimilarSimHash scans all entries in namespace (O(N) with maxScan limit)
//   - Strict mode sorts by (score DESC, key ASC) for deterministic results
type InMemorySharedIndex struct {
	mu         sync.RWMutex
	namespaces map[string]map[string]*IndexEntry
}

// NewInMemorySharedIndex creates a new in-memory index.
func NewInMemorySharedIndex() *InMemorySharedIndex {
	return &InMemorySharedIndex{
		namespaces: make(map[string]map[string]*IndexEntry),
	}
}

// Upsert adds or updates an entry in the index.
func (idx *InMemorySharedIndex) Upsert(namespace, key string, simhash, version, timestamp int64) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if _, ok := idx.namespaces[namespace]; !ok {
		idx.namespaces[namespace] = make(map[string]*IndexEntry)
	}

	idx.namespaces[namespace][key] = &IndexEntry{
		Namespace: namespace,
		Key:       key,
		SimHash:   simhash,
		Version:   version,
		Timestamp: timestamp,
	}
}

// Delete removes an entry from the index.
func (idx *InMemorySharedIndex) Delete(namespace, key string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if ns, ok := idx.namespaces[namespace]; ok {
		delete(ns, key)
		// Clean up empty namespaces
		if len(ns) == 0 {
			delete(idx.namespaces, namespace)
		}
	}
}

// FindSimilarSimHash finds entries similar to the query simhash.
//
// Uses hamming distance for similarity scoring:
//
//	score = 1.0 - (hamming_distance / 64.0)
//
// In Strict mode, results are sorted deterministically by (score DESC, key ASC).
// In BestEffort mode, results may vary in ordering when scores are equal.
func (idx *InMemorySharedIndex) FindSimilarSimHash(
	namespace string,
	querySimHash int64,
	topK, maxScan int,
	mode DeterminismMode,
) []SearchResult {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	ns, ok := idx.namespaces[namespace]
	if !ok {
		return []SearchResult{}
	}

	// Collect and score entries
	results := make([]SearchResult, 0, len(ns))
	scanned := 0

	for key, entry := range ns {
		// Respect maxScan limit
		if maxScan > 0 && scanned >= maxScan {
			break
		}
		scanned++

		// Calculate hamming distance and score
		distance := hammingDistance(entry.SimHash, querySimHash)
		score := 1.0 - float64(distance)/64.0

		results = append(results, SearchResult{
			Key:       key,
			Score:     score,
			Version:   entry.Version,
			Timestamp: entry.Timestamp,
		})
	}

	// Sort by score DESC, then key ASC (for determinism)
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score // DESC by score
		}
		if mode == DeterminismStrict {
			return results[i].Key < results[j].Key // ASC by key for tie-breaking
		}
		return false // BestEffort: don't sort ties
	})

	// Limit to topK
	if topK > 0 && len(results) > topK {
		results = results[:topK]
	}

	return results
}

// hammingDistance counts the number of differing bits between two int64 values.
// Used for SimHash similarity scoring.
func hammingDistance(a, b int64) int {
	xor := uint64(a ^ b)
	count := 0
	for xor != 0 {
		count++
		xor &= xor - 1 // Clear lowest set bit
	}
	return count
}

// cosineSimilarity calculates the cosine similarity between two vectors.
// Returns a value in [-1, 1], normalized to [0, 1] for consistency with SimHash scores.
// Returns 0.0 if either vector is zero-length or has zero magnitude.
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	// Avoid division by zero
	if normA == 0 || normB == 0 {
		return 0.0
	}

	// Calculate cosine similarity (range: [-1, 1])
	cosine := dotProduct / (sqrt(normA) * sqrt(normB))

	// Normalize to [0, 1] for consistency with SimHash scoring
	// -1 -> 0, 0 -> 0.5, 1 -> 1
	return (cosine + 1.0) / 2.0
}

// sqrt is a simple square root implementation to avoid math import.
func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x / 2.0
	for i := 0; i < 20; i++ { // Newton's method, 20 iterations is plenty
		z = z - (z*z-x)/(2*z)
	}
	return z
}

// EntryCount returns the number of entries in a namespace.
func (idx *InMemorySharedIndex) EntryCount(namespace string) int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if ns, ok := idx.namespaces[namespace]; ok {
		return len(ns)
	}
	return 0
}

// Namespaces returns all namespace names in the index.
func (idx *InMemorySharedIndex) Namespaces() []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	names := make([]string, 0, len(idx.namespaces))
	for name := range idx.namespaces {
		names = append(names, name)
	}

	// Sort for deterministic iteration
	sort.Strings(names)
	return names
}

// UpsertWithEmbedding adds or updates an entry with a neural embedding.
func (idx *InMemorySharedIndex) UpsertWithEmbedding(namespace, key string, simhash int64, embedding []float64, version, timestamp int64) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if _, ok := idx.namespaces[namespace]; !ok {
		idx.namespaces[namespace] = make(map[string]*IndexEntry)
	}

	// Make a copy of the embedding to avoid external mutation
	embCopy := make([]float64, len(embedding))
	copy(embCopy, embedding)

	idx.namespaces[namespace][key] = &IndexEntry{
		Namespace: namespace,
		Key:       key,
		SimHash:   simhash,
		Embedding: embCopy,
		Version:   version,
		Timestamp: timestamp,
	}
}

// FindSimilarByEmbedding finds entries similar to the query embedding using cosine similarity.
//
// Only entries with non-empty embeddings are considered.
// Uses cosine similarity normalized to [0, 1] for scoring.
//
// In Strict mode, results are sorted deterministically by (score DESC, key ASC).
// In BestEffort mode, results may vary in ordering when scores are equal.
func (idx *InMemorySharedIndex) FindSimilarByEmbedding(
	namespace string,
	queryEmbedding []float64,
	topK, maxScan int,
	mode DeterminismMode,
) []SearchResult {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	ns, ok := idx.namespaces[namespace]
	if !ok || len(queryEmbedding) == 0 {
		return []SearchResult{}
	}

	// Collect and score entries with embeddings
	results := make([]SearchResult, 0, len(ns))
	scanned := 0

	for key, entry := range ns {
		// Respect maxScan limit
		if maxScan > 0 && scanned >= maxScan {
			break
		}
		scanned++

		// Skip entries without embeddings
		if len(entry.Embedding) == 0 {
			continue
		}

		// Calculate cosine similarity score
		score := cosineSimilarity(entry.Embedding, queryEmbedding)

		results = append(results, SearchResult{
			Key:       key,
			Score:     score,
			Version:   entry.Version,
			Timestamp: entry.Timestamp,
		})
	}

	// Sort by score DESC, then key ASC (for determinism)
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score // DESC by score
		}
		if mode == DeterminismStrict {
			return results[i].Key < results[j].Key // ASC by key for tie-breaking
		}
		return false // BestEffort: don't sort ties
	})

	// Limit to topK
	if topK > 0 && len(results) > topK {
		results = results[:topK]
	}

	return results
}
