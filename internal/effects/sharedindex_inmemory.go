// Package effects provides the in-memory implementation of SharedIndex.
// Part of M-DX16 (Deterministic Semantic Retrieval).
package effects

import (
	"sort"
	"sync"

	"github.com/sunholo-data/ailang/internal/simhash"
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

		score := simhash.Similarity(entry.SimHash, querySimHash)

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

		// Cosine on the [0, 1] scale so it ranks against SimHash scores.
		score := simhash.CosineUnit(entry.Embedding, queryEmbedding)

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
