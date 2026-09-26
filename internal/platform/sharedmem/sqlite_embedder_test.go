package sharedmem

import (
	"fmt"
	"github.com/sunholo-data/ailang/internal/effects"
	"path/filepath"
	"testing"
)

// --- effects.Embedder wiring tests (M-BRAIN-VECTORS M2) ---

func TestSQLiteSharedCache_WithEmbedder_AutoEmbed(t *testing.T) {
	mock := &mockEmbedder{dim: 4, model: "test-model"}
	dir := t.TempDir()
	cache, err := NewSQLiteSharedCache(filepath.Join(dir, "brain.db"), effects.WithEmbedder(mock))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()

	// PutFrame with content should auto-embed
	err = cache.PutFrame(effects.BrainFrame{
		Key: "auto_embed", Namespace: "test", Value: []byte("v"),
		Content: "This will be auto-embedded", SimHash: 42,
	})
	if err != nil {
		t.Fatal(err)
	}

	if mock.callCount != 1 {
		t.Errorf("expected 1 embed call, got %d", mock.callCount)
	}

	// Verify embedding was stored
	results := cache.SearchByEmbedding([]float32{0.1, 0.2, 0.3, 0.4}, "", 10)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Frame.EmbedModel != "test-model" {
		t.Errorf("expected model test-model, got %s", results[0].Frame.EmbedModel)
	}
	if results[0].Frame.EmbeddingDim != 4 {
		t.Errorf("expected dim 4, got %d", results[0].Frame.EmbeddingDim)
	}
}

func TestSQLiteSharedCache_WithEmbedder_SkipsExistingEmbedding(t *testing.T) {
	mock := &mockEmbedder{dim: 4, model: "test-model"}
	dir := t.TempDir()
	cache, err := NewSQLiteSharedCache(filepath.Join(dir, "brain.db"), effects.WithEmbedder(mock))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()

	// PutFrame with EXISTING embedding should NOT call embedder
	err = cache.PutFrame(effects.BrainFrame{
		Key: "pre_embedded", Namespace: "test", Value: []byte("v"),
		Content:   "Already has embedding",
		Embedding: []float32{1, 2, 3, 4}, EmbedModel: "original-model",
	})
	if err != nil {
		t.Fatal(err)
	}

	if mock.callCount != 0 {
		t.Errorf("embedder should NOT be called when frame already has embedding, got %d calls", mock.callCount)
	}

	// Verify original embedding preserved
	results := cache.SearchByEmbedding([]float32{1, 2, 3, 4}, "", 10)
	if results[0].Frame.EmbedModel != "original-model" {
		t.Errorf("original embedding should be preserved, got model %s", results[0].Frame.EmbedModel)
	}
}

func TestSQLiteSharedCache_WithEmbedder_ErrorFallback(t *testing.T) {
	mock := &mockEmbedder{dim: 4, model: "test-model", failNext: true}
	dir := t.TempDir()
	cache, err := NewSQLiteSharedCache(filepath.Join(dir, "brain.db"), effects.WithEmbedder(mock))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()

	// PutFrame should succeed even if embedder fails
	err = cache.PutFrame(effects.BrainFrame{
		Key: "embed_fail", Namespace: "test", Value: []byte("v"),
		Content: "effects.Embedder will fail", SimHash: 123,
	})
	if err != nil {
		t.Fatalf("PutFrame should succeed even with embedder error: %v", err)
	}

	// Frame stored without embedding
	val, ok := cache.Get("embed_fail")
	if !ok {
		t.Fatal("frame should be stored")
	}
	if string(val) != "v" {
		t.Errorf("unexpected value: %s", val)
	}

	// No embedding results
	results := cache.SearchByEmbedding([]float32{0, 0, 0, 0}, "", 10)
	if len(results) != 0 {
		t.Errorf("expected 0 embedding results, got %d", len(results))
	}
}

func TestSQLiteSharedCache_NilEmbedder(t *testing.T) {
	cache := newTestSQLiteCache(t) // no embedder

	// PutFrame should work identically without embedder
	err := cache.PutFrame(effects.BrainFrame{
		Key: "no_embedder", Namespace: "test", Value: []byte("v"),
		Content: "No embedder configured", SimHash: 42,
	})
	if err != nil {
		t.Fatalf("PutFrame should work without embedder: %v", err)
	}

	val, ok := cache.Get("no_embedder")
	if !ok || string(val) != "v" {
		t.Error("frame should be stored normally")
	}
}

func TestSQLiteSharedCache_BackfillEmbeddings(t *testing.T) {
	cache := newTestSQLiteCache(t)

	// Store frames without embeddings
	for i := 0; i < 5; i++ {
		cache.PutFrame(effects.BrainFrame{
			Key: fmt.Sprintf("frame_%d", i), Namespace: "test",
			Value: []byte("v"), Content: fmt.Sprintf("Content %d", i),
		})
	}
	// Store one WITH embedding
	cache.PutVector("already_embedded", "test", []float32{1, 0}, "existing", []byte("p"))

	// No embedder = no backfill
	processed, errors := cache.BackfillEmbeddings("")
	if processed != 0 || errors != 0 {
		t.Errorf("backfill without embedder should be no-op, got %d/%d", processed, errors)
	}

	// Set embedder and backfill
	mock := &mockEmbedder{dim: 4, model: "backfill-model"}
	cache.SetEmbedder(mock)

	processed, errors = cache.BackfillEmbeddings("")
	if processed != 5 {
		t.Errorf("expected 5 processed, got %d", processed)
	}
	if errors != 0 {
		t.Errorf("expected 0 errors, got %d", errors)
	}
	if mock.callCount != 5 {
		t.Errorf("expected 5 embed calls, got %d", mock.callCount)
	}

	// All frames now have embeddings
	total, withEmb, _ := cache.EmbeddingStats()
	if total != 6 || withEmb != 6 {
		t.Errorf("expected 6/6 with embeddings, got %d/%d", withEmb, total)
	}

	// Backfill again should be no-op (all embedded)
	mock.callCount = 0
	processed, _ = cache.BackfillEmbeddings("")
	if processed != 0 {
		t.Errorf("second backfill should be no-op, got %d", processed)
	}
}

func TestSQLiteSharedCache_BackfillEmbeddings_NamespaceFilter(t *testing.T) {
	cache := newTestSQLiteCache(t)

	cache.PutFrame(effects.BrainFrame{Key: "ns1_a", Namespace: "learnings", Value: []byte("v"), Content: "A"})
	cache.PutFrame(effects.BrainFrame{Key: "ns2_b", Namespace: "patterns", Value: []byte("v"), Content: "B"})

	mock := &mockEmbedder{dim: 4, model: "test"}
	cache.SetEmbedder(mock)

	// Backfill only "learnings"
	processed, _ := cache.BackfillEmbeddings("learnings")
	if processed != 1 {
		t.Errorf("expected 1 processed for learnings, got %d", processed)
	}

	// "patterns" still has no embedding
	_, withEmb, _ := cache.EmbeddingStats()
	if withEmb != 1 {
		t.Errorf("expected 1 with embedding, got %d", withEmb)
	}
}

func TestSQLiteSharedCache_MixedFrames(t *testing.T) {
	cache := newTestSQLiteCache(t)

	// Store mix of frames: some with embedding, some without, some vector-only
	cache.PutFrame(effects.BrainFrame{
		Key: "text_only", Namespace: "test", Value: []byte("v"),
		Content: "just text", SimHash: 100,
	})
	cache.PutFrame(effects.BrainFrame{
		Key: "hybrid", Namespace: "test", Value: []byte("v"),
		Content: "with both", SimHash: 200,
		Embedding: []float32{1, 0, 0}, EmbedModel: "test",
	})
	cache.PutVector("vector_only", "test", []float32{0, 1, 0}, "test", []byte("p"))

	// ListRecent should return all
	recent := cache.ListRecent("test", 10)
	if len(recent) != 3 {
		t.Fatalf("expected 3 frames, got %d", len(recent))
	}

	// Text search should find text-bearing frames
	textResults := cache.SearchByText("text", "test", 10)
	if len(textResults) != 1 {
		t.Errorf("expected 1 text match, got %d", len(textResults))
	}

	// SimHash search should find all frames (vector_only has simhash=0 default)
	simResults := cache.SearchBySimHash("test", 100, 10)
	if len(simResults) != 3 {
		t.Errorf("expected 3 simhash results, got %d", len(simResults))
	}

	// Embedding search should find embedding-bearing frames
	embResults := cache.SearchByEmbedding([]float32{1, 0, 0}, "test", 10)
	if len(embResults) != 2 {
		t.Errorf("expected 2 embedding results, got %d", len(embResults))
	}
}
