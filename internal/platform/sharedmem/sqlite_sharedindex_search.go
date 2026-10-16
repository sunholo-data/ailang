package sharedmem

import (
	"fmt"
	"sort"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/simhash"
)

// scan selects the same key prefix as the strict in-memory implementation.
// maxScan counts every entry, including ones without an embedding.
func (i *SQLiteSharedIndex) scan(ns string, maxScan int) ([]effects.IndexEntry, error) {
	query := "SELECT key,simhash,embedding,embedding_dim,version,timestamp FROM sharedindex_entries WHERE namespace=? ORDER BY key"
	args := []any{ns}
	if maxScan > 0 {
		query += " LIMIT ?"
		args = append(args, maxScan)
	}
	rows, err := i.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("shared index scan: %w", err)
	}
	defer rows.Close()
	entries := []effects.IndexEntry{}
	for rows.Next() {
		var e effects.IndexEntry
		var blob []byte
		var dim int
		if err := rows.Scan(&e.Key, &e.SimHash, &blob, &dim, &e.Version, &e.Timestamp); err != nil {
			return nil, err
		}
		if dim < 0 || len(blob)%4 != 0 || len(blob)/4 != dim {
			return nil, fmt.Errorf("shared index corrupt embedding for %q", e.Key)
		}
		v := effects.DecodeEmbedding(blob)
		e.Embedding = make([]float64, len(v))
		for n, f := range v {
			e.Embedding[n] = float64(f)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("shared index scan: %w", err)
	}
	return entries, nil
}

func rankIndex(entries []effects.IndexEntry, topK int, mode effects.DeterminismMode, score func(effects.IndexEntry) (float64, bool)) []effects.SearchResult {
	results := []effects.SearchResult{}
	for _, e := range entries {
		value, ok := score(e)
		if !ok {
			continue
		}
		results = append(results, effects.SearchResult{Key: e.Key, Score: value, Version: e.Version, Timestamp: e.Timestamp})
	}
	sort.SliceStable(results, func(a, b int) bool {
		if results[a].Score != results[b].Score {
			return results[a].Score > results[b].Score
		}
		return mode == effects.DeterminismStrict && results[a].Key < results[b].Key
	})
	if topK > 0 && len(results) > topK {
		results = results[:topK]
	}
	return results
}

func (i *SQLiteSharedIndex) FindSimilarSimHashChecked(ns string, query int64, topK, maxScan int, mode effects.DeterminismMode) ([]effects.SearchResult, error) {
	entries, err := i.scan(ns, maxScan)
	if err != nil {
		return nil, err
	}
	return rankIndex(entries, topK, mode, func(e effects.IndexEntry) (float64, bool) { return simhash.Similarity(e.SimHash, query), true }), nil
}

func (i *SQLiteSharedIndex) FindSimilarByEmbeddingChecked(ns string, query []float64, topK, maxScan int, mode effects.DeterminismMode) ([]effects.SearchResult, error) {
	entries, err := i.scan(ns, maxScan)
	if err != nil {
		return nil, err
	}
	if len(query) == 0 {
		return []effects.SearchResult{}, nil
	}
	return rankIndex(entries, topK, mode, func(e effects.IndexEntry) (float64, bool) {
		if len(e.Embedding) == 0 {
			return 0, false
		}
		// Existing cosine helper defines dimension mismatch and zero vectors as 0.
		return simhash.CosineUnit(e.Embedding, query), true
	}), nil
}
