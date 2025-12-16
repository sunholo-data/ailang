package effects

import (
	"fmt"
	"sync"
	"testing"
)

func TestInMemorySharedIndex_BasicOperations(t *testing.T) {
	idx := NewInMemorySharedIndex()

	// Test Upsert
	idx.Upsert("test", "key1", 0x1234567890ABCDEF, 1, 1000)
	idx.Upsert("test", "key2", -0x0123456789ABCDEF, 1, 2000) // Use negative to avoid overflow

	// Test EntryCount
	if count := idx.EntryCount("test"); count != 2 {
		t.Errorf("expected 2 entries, got %d", count)
	}

	// Test Namespaces
	ns := idx.Namespaces()
	if len(ns) != 1 || ns[0] != "test" {
		t.Errorf("expected [test], got %v", ns)
	}

	// Test Delete
	idx.Delete("test", "key1")
	if count := idx.EntryCount("test"); count != 1 {
		t.Errorf("expected 1 entry after delete, got %d", count)
	}

	// Test delete non-existent key (no-op)
	idx.Delete("test", "nonexistent")
	if count := idx.EntryCount("test"); count != 1 {
		t.Errorf("expected 1 entry after no-op delete, got %d", count)
	}
}

func TestInMemorySharedIndex_FindSimilarSimHash(t *testing.T) {
	idx := NewInMemorySharedIndex()

	// Add entries with known SimHashes
	// Using simple values for predictable hamming distances
	idx.Upsert("beliefs", "belief1", 0, 1, 1000)      // Distance 0 to query 0
	idx.Upsert("beliefs", "belief2", 1, 1, 2000)      // Distance 1 to query 0
	idx.Upsert("beliefs", "belief3", 3, 1, 3000)      // Distance 2 to query 0
	idx.Upsert("beliefs", "belief4", 0xFF, 1, 4000)   // Distance 8 to query 0
	idx.Upsert("beliefs", "belief5", 0xFFFF, 1, 5000) // Distance 16 to query 0
	idx.Upsert("beliefs", "belief6", -1, 1, 6000)     // Distance 64 to query 0 (all bits flipped)

	// Find similar to 0 (top 3)
	results := idx.FindSimilarSimHash("beliefs", 0, 3, 0, DeterminismStrict)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Check ordering (score DESC)
	if results[0].Key != "belief1" || results[0].Score != 1.0 {
		t.Errorf("expected belief1 with score 1.0, got %s with %f", results[0].Key, results[0].Score)
	}
	if results[1].Key != "belief2" {
		t.Errorf("expected belief2, got %s", results[1].Key)
	}
	if results[2].Key != "belief3" {
		t.Errorf("expected belief3, got %s", results[2].Key)
	}

	// Test namespace isolation
	idx.Upsert("goals", "goal1", 0, 1, 7000)
	resultsGoals := idx.FindSimilarSimHash("goals", 0, 10, 0, DeterminismStrict)
	if len(resultsGoals) != 1 {
		t.Errorf("expected 1 result in goals, got %d", len(resultsGoals))
	}
}

func TestInMemorySharedIndex_MaxScan(t *testing.T) {
	idx := NewInMemorySharedIndex()

	// Add many entries
	for i := 0; i < 100; i++ {
		idx.Upsert("test", fmt.Sprintf("key%03d", i), int64(i), 1, int64(i))
	}

	// Search with maxScan limit
	results := idx.FindSimilarSimHash("test", 0, 100, 10, DeterminismStrict)

	// Should only scan 10 entries (maxScan limit)
	// Note: map iteration order is non-deterministic, so we just check count <= 10
	if len(results) > 10 {
		t.Errorf("maxScan should limit results, got %d", len(results))
	}
}

func TestInMemorySharedIndex_DeterminismStrict(t *testing.T) {
	idx := NewInMemorySharedIndex()

	// Add entries with same SimHash (will have same score)
	for i := 0; i < 10; i++ {
		idx.Upsert("test", fmt.Sprintf("key%02d", i), 0, 1, int64(i))
	}

	// Run multiple times and verify identical results
	var firstResults []SearchResult
	for run := 0; run < 5; run++ {
		results := idx.FindSimilarSimHash("test", 0, 10, 0, DeterminismStrict)

		if run == 0 {
			firstResults = results
		} else {
			// Compare with first run
			if len(results) != len(firstResults) {
				t.Fatalf("run %d: result count mismatch", run)
			}
			for i := range results {
				if results[i].Key != firstResults[i].Key {
					t.Errorf("run %d: result %d key mismatch: %s vs %s",
						run, i, results[i].Key, firstResults[i].Key)
				}
			}
		}
	}

	// In Strict mode with same scores, should be sorted by key ASC
	if len(firstResults) > 0 && firstResults[0].Key != "key00" {
		t.Errorf("Strict mode should sort by key ASC when scores equal, got %s", firstResults[0].Key)
	}
}

func TestInMemorySharedIndex_ConcurrentAccess(t *testing.T) {
	idx := NewInMemorySharedIndex()

	// 100+ concurrent goroutines writing and reading
	var wg sync.WaitGroup
	numWriters := 50
	numReaders := 50
	iterations := 100

	// Writers
	for w := 0; w < numWriters; w++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			ns := fmt.Sprintf("ns%d", writerID%5) // 5 namespaces
			for i := 0; i < iterations; i++ {
				key := fmt.Sprintf("key%d_%d", writerID, i)
				idx.Upsert(ns, key, int64(writerID*1000+i), 1, int64(i))

				// Occasionally delete
				if i%10 == 0 && i > 0 {
					delKey := fmt.Sprintf("key%d_%d", writerID, i-5)
					idx.Delete(ns, delKey)
				}
			}
		}(w)
	}

	// Readers
	for r := 0; r < numReaders; r++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			ns := fmt.Sprintf("ns%d", readerID%5)
			for i := 0; i < iterations; i++ {
				_ = idx.FindSimilarSimHash(ns, int64(i), 10, 100, DeterminismStrict)
				_ = idx.EntryCount(ns)
				_ = idx.Namespaces()
			}
		}(r)
	}

	wg.Wait()

	// Verify index is still consistent
	total := 0
	for _, ns := range idx.Namespaces() {
		total += idx.EntryCount(ns)
	}

	if total == 0 {
		t.Error("index should have some entries after concurrent operations")
	}
}

func TestHammingDistance(t *testing.T) {
	tests := []struct {
		a, b     int64
		expected int
	}{
		{0, 0, 0},
		{0, 1, 1},
		{0, 3, 2},
		{0, 0xFF, 8},
		{0, 0xFFFF, 16},
		{0, -1, 64}, // All bits different
		{-1, -1, 0}, // Same
		{0x5555555555555555, -0x5555555555555556, 64}, // Alternating bits (0xAAAA... as negative)
	}

	for _, tt := range tests {
		got := hammingDistance(tt.a, tt.b)
		if got != tt.expected {
			t.Errorf("hammingDistance(%x, %x) = %d, want %d", tt.a, tt.b, got, tt.expected)
		}
	}
}

func TestSharedIndexContext(t *testing.T) {
	ctx := NewSharedIndexContext(nil)

	if ctx.Index == nil {
		t.Fatal("Index should be initialized")
	}

	// Test counter increments
	ctx.IncrementUpsert()
	ctx.IncrementUpsert()
	if ctx.UpsertCount != 2 {
		t.Errorf("expected UpsertCount=2, got %d", ctx.UpsertCount)
	}

	ctx.IncrementDelete()
	if ctx.DeleteCount != 1 {
		t.Errorf("expected DeleteCount=1, got %d", ctx.DeleteCount)
	}

	ctx.IncrementSearch(100)
	ctx.IncrementSearch(50)
	if ctx.SearchCount != 2 || ctx.ScannedTotal != 150 {
		t.Errorf("expected SearchCount=2, ScannedTotal=150, got %d, %d",
			ctx.SearchCount, ctx.ScannedTotal)
	}
}

func TestSharedIndexContext_Tracing(t *testing.T) {
	ctx := NewSharedIndexContext(nil)

	// Tracing disabled by default - operations should not be recorded
	ctx.TraceUpsert("beliefs", "key1")
	if len(ctx.GetTrace()) != 0 {
		t.Error("trace should be empty when disabled")
	}

	// Enable tracing
	ctx.EnableTracing()

	// Record operations
	ctx.TraceUpsert("beliefs", "key1")
	ctx.TraceDelete("beliefs", "key2")
	ctx.TraceFindSimHash("goals", 12345, 5, 100, DeterminismStrict, 3)
	ctx.TraceResolveBestMatch("actions", 67890, "best_key")

	trace := ctx.GetTrace()
	if len(trace) != 4 {
		t.Fatalf("expected 4 trace entries, got %d", len(trace))
	}

	// Check upsert entry
	if trace[0].Operation != "upsert" || trace[0].Namespace != "beliefs" || trace[0].Key != "key1" {
		t.Errorf("upsert trace mismatch: %+v", trace[0])
	}

	// Check delete entry
	if trace[1].Operation != "delete" || trace[1].Namespace != "beliefs" || trace[1].Key != "key2" {
		t.Errorf("delete trace mismatch: %+v", trace[1])
	}

	// Check find_simhash entry
	if trace[2].Operation != "find_simhash" || trace[2].Namespace != "goals" || trace[2].TopK != 5 || trace[2].Mode != DeterminismStrict {
		t.Errorf("find_simhash trace mismatch: %+v", trace[2])
	}

	// Check resolve_best_match entry
	if trace[3].Operation != "resolve_best_match" || trace[3].ChosenKey != "best_key" {
		t.Errorf("resolve_best_match trace mismatch: %+v", trace[3])
	}

	// All entries should have timestamps
	for i, entry := range trace {
		if entry.Timestamp == 0 {
			t.Errorf("trace entry %d missing timestamp", i)
		}
	}

	// Test clear
	ctx.ClearTrace()
	if len(ctx.GetTrace()) != 0 {
		t.Error("trace should be empty after clear")
	}

	// Test disable
	ctx.DisableTracing()
	ctx.TraceUpsert("test", "key")
	if len(ctx.GetTrace()) != 0 {
		t.Error("trace should not record after disable")
	}
}

// ============================================================================
// DX-17: Embedding-based Similarity Search Tests
// ============================================================================

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a, b     []float64
		expected float64
	}{
		{
			name:     "identical vectors",
			a:        []float64{1.0, 0.0, 0.0},
			b:        []float64{1.0, 0.0, 0.0},
			expected: 1.0, // (1+1)/2 = 1
		},
		{
			name:     "orthogonal vectors",
			a:        []float64{1.0, 0.0, 0.0},
			b:        []float64{0.0, 1.0, 0.0},
			expected: 0.5, // (0+1)/2 = 0.5
		},
		{
			name:     "opposite vectors",
			a:        []float64{1.0, 0.0, 0.0},
			b:        []float64{-1.0, 0.0, 0.0},
			expected: 0.0, // (-1+1)/2 = 0
		},
		{
			name:     "similar vectors",
			a:        []float64{1.0, 1.0, 0.0},
			b:        []float64{1.0, 0.9, 0.0},
			expected: 0.999, // Very close to 1
		},
		{
			name:     "zero vector a",
			a:        []float64{0.0, 0.0, 0.0},
			b:        []float64{1.0, 1.0, 1.0},
			expected: 0.0,
		},
		{
			name:     "empty vectors",
			a:        []float64{},
			b:        []float64{},
			expected: 0.0,
		},
		{
			name:     "mismatched lengths",
			a:        []float64{1.0, 2.0},
			b:        []float64{1.0, 2.0, 3.0},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cosineSimilarity(tt.a, tt.b)
			// Use tolerance for floating point comparison
			if diff := got - tt.expected; diff > 0.01 || diff < -0.01 {
				t.Errorf("cosineSimilarity(%v, %v) = %f, want ~%f", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

func TestInMemorySharedIndex_UpsertWithEmbedding(t *testing.T) {
	idx := NewInMemorySharedIndex()

	// Create embeddings
	emb1 := []float64{1.0, 0.0, 0.0, 0.0}
	emb2 := []float64{0.9, 0.1, 0.0, 0.0}
	emb3 := []float64{0.0, 1.0, 0.0, 0.0}

	// Upsert with embeddings
	idx.UpsertWithEmbedding("beliefs", "b1", 100, emb1, 1, 1000)
	idx.UpsertWithEmbedding("beliefs", "b2", 200, emb2, 1, 2000)
	idx.UpsertWithEmbedding("beliefs", "b3", 300, emb3, 1, 3000)

	// Verify entries exist
	if count := idx.EntryCount("beliefs"); count != 3 {
		t.Errorf("expected 3 entries, got %d", count)
	}

	// Verify embedding is stored (internal access via map)
	idx.mu.RLock()
	entry := idx.namespaces["beliefs"]["b1"]
	idx.mu.RUnlock()

	if len(entry.Embedding) != 4 {
		t.Errorf("expected 4-dim embedding, got %d-dim", len(entry.Embedding))
	}
	if entry.Embedding[0] != 1.0 {
		t.Errorf("expected embedding[0]=1.0, got %f", entry.Embedding[0])
	}
}

func TestInMemorySharedIndex_FindSimilarByEmbedding(t *testing.T) {
	idx := NewInMemorySharedIndex()

	// Create test embeddings
	// b1 and b2 are similar (both point roughly in +x direction)
	// b3 is orthogonal (points in +y direction)
	// b4 is opposite (points in -x direction)
	idx.UpsertWithEmbedding("test", "b1", 0, []float64{1.0, 0.0, 0.0, 0.0}, 1, 1000)
	idx.UpsertWithEmbedding("test", "b2", 0, []float64{0.95, 0.05, 0.0, 0.0}, 1, 2000)
	idx.UpsertWithEmbedding("test", "b3", 0, []float64{0.0, 1.0, 0.0, 0.0}, 1, 3000)
	idx.UpsertWithEmbedding("test", "b4", 0, []float64{-1.0, 0.0, 0.0, 0.0}, 1, 4000)

	// Query with embedding similar to b1 and b2
	query := []float64{1.0, 0.0, 0.0, 0.0}
	results := idx.FindSimilarByEmbedding("test", query, 10, 0, DeterminismStrict)

	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}

	// b1 should be first (exact match)
	if results[0].Key != "b1" {
		t.Errorf("expected b1 first, got %s", results[0].Key)
	}
	if results[0].Score != 1.0 {
		t.Errorf("expected score 1.0 for exact match, got %f", results[0].Score)
	}

	// b2 should be second (very similar)
	if results[1].Key != "b2" {
		t.Errorf("expected b2 second, got %s", results[1].Key)
	}
	if results[1].Score < 0.99 {
		t.Errorf("expected score >0.99 for similar vector, got %f", results[1].Score)
	}

	// b3 should be third (orthogonal, score ~0.5)
	if results[2].Key != "b3" {
		t.Errorf("expected b3 third, got %s", results[2].Key)
	}
	if results[2].Score < 0.49 || results[2].Score > 0.51 {
		t.Errorf("expected score ~0.5 for orthogonal vector, got %f", results[2].Score)
	}

	// b4 should be last (opposite direction, score ~0)
	if results[3].Key != "b4" {
		t.Errorf("expected b4 last, got %s", results[3].Key)
	}
	if results[3].Score > 0.01 {
		t.Errorf("expected score ~0 for opposite vector, got %f", results[3].Score)
	}
}

func TestInMemorySharedIndex_FindSimilarByEmbedding_TopK(t *testing.T) {
	idx := NewInMemorySharedIndex()

	// Add 10 entries
	for i := 0; i < 10; i++ {
		emb := make([]float64, 4)
		emb[0] = float64(10-i) / 10.0 // Decreasing similarity to [1,0,0,0]
		emb[1] = float64(i) / 10.0
		idx.UpsertWithEmbedding("test", fmt.Sprintf("e%d", i), 0, emb, 1, int64(i*1000))
	}

	// Query top 3
	query := []float64{1.0, 0.0, 0.0, 0.0}
	results := idx.FindSimilarByEmbedding("test", query, 3, 0, DeterminismStrict)

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	// Results should be ordered by score DESC
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("results not sorted by score DESC: [%d]=%f > [%d]=%f",
				i, results[i].Score, i-1, results[i-1].Score)
		}
	}
}

func TestInMemorySharedIndex_FindSimilarByEmbedding_SkipsEntriesWithoutEmbeddings(t *testing.T) {
	idx := NewInMemorySharedIndex()

	// Add some with embeddings, some without
	idx.UpsertWithEmbedding("test", "with_emb", 0, []float64{1.0, 0.0, 0.0, 0.0}, 1, 1000)
	idx.Upsert("test", "without_emb", 0, 1, 2000) // No embedding

	query := []float64{1.0, 0.0, 0.0, 0.0}
	results := idx.FindSimilarByEmbedding("test", query, 10, 0, DeterminismStrict)

	if len(results) != 1 {
		t.Errorf("expected 1 result (entry without embedding should be skipped), got %d", len(results))
	}
	if results[0].Key != "with_emb" {
		t.Errorf("expected 'with_emb', got %s", results[0].Key)
	}
}

func TestInMemorySharedIndex_FindSimilarByEmbedding_EmptyNamespace(t *testing.T) {
	idx := NewInMemorySharedIndex()

	query := []float64{1.0, 0.0, 0.0, 0.0}
	results := idx.FindSimilarByEmbedding("nonexistent", query, 10, 0, DeterminismStrict)

	if len(results) != 0 {
		t.Errorf("expected 0 results for nonexistent namespace, got %d", len(results))
	}
}

func TestInMemorySharedIndex_FindSimilarByEmbedding_DeterministicOrdering(t *testing.T) {
	idx := NewInMemorySharedIndex()

	// Add entries with identical scores (same embedding)
	emb := []float64{1.0, 0.0, 0.0, 0.0}
	idx.UpsertWithEmbedding("test", "c", 0, emb, 1, 1000)
	idx.UpsertWithEmbedding("test", "a", 0, emb, 1, 2000)
	idx.UpsertWithEmbedding("test", "b", 0, emb, 1, 3000)

	query := []float64{1.0, 0.0, 0.0, 0.0}

	// Strict mode: should be alphabetically sorted for ties
	results := idx.FindSimilarByEmbedding("test", query, 10, 0, DeterminismStrict)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// All scores should be 1.0 (identical embeddings)
	for _, r := range results {
		if r.Score != 1.0 {
			t.Errorf("expected score 1.0, got %f", r.Score)
		}
	}

	// Keys should be sorted alphabetically in Strict mode
	if results[0].Key != "a" || results[1].Key != "b" || results[2].Key != "c" {
		t.Errorf("expected [a,b,c], got [%s,%s,%s]",
			results[0].Key, results[1].Key, results[2].Key)
	}
}
