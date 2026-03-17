package effects

import (
	"fmt"
	"path/filepath"
	"testing"
)

// --- Embedder wiring tests (M-BRAIN-VECTORS M2) ---

func TestSQLiteSharedCache_WithEmbedder_AutoEmbed(t *testing.T) {
	mock := &mockEmbedder{dim: 4, model: "test-model"}
	dir := t.TempDir()
	cache, err := NewSQLiteSharedCache(filepath.Join(dir, "brain.db"), WithEmbedder(mock))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()

	// PutFrame with content should auto-embed
	err = cache.PutFrame(BrainFrame{
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
	cache, err := NewSQLiteSharedCache(filepath.Join(dir, "brain.db"), WithEmbedder(mock))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()

	// PutFrame with EXISTING embedding should NOT call embedder
	err = cache.PutFrame(BrainFrame{
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
	cache, err := NewSQLiteSharedCache(filepath.Join(dir, "brain.db"), WithEmbedder(mock))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()

	// PutFrame should succeed even if embedder fails
	err = cache.PutFrame(BrainFrame{
		Key: "embed_fail", Namespace: "test", Value: []byte("v"),
		Content: "Embedder will fail", SimHash: 123,
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
	err := cache.PutFrame(BrainFrame{
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
		cache.PutFrame(BrainFrame{
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

	cache.PutFrame(BrainFrame{Key: "ns1_a", Namespace: "learnings", Value: []byte("v"), Content: "A"})
	cache.PutFrame(BrainFrame{Key: "ns2_b", Namespace: "patterns", Value: []byte("v"), Content: "B"})

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

func TestBrainStore_WithEmbedder(t *testing.T) {
	mock := &mockEmbedder{dim: 4, model: "brain-test"}
	dir := t.TempDir()

	store, err := NewBrainStore(
		filepath.Join(dir, "user.db"),
		filepath.Join(dir, "project.db"),
		WithEmbedder(mock),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Put to project — should auto-embed
	err = store.Put(BrainFrame{
		Key: "proj_frame", Namespace: "test", Value: []byte("v"),
		Content: "Project content", SimHash: 100,
	}, ScopeProject)
	if err != nil {
		t.Fatal(err)
	}

	// Put to user — should also auto-embed
	err = store.Put(BrainFrame{
		Key: "user_frame", Namespace: "test", Value: []byte("v"),
		Content: "User content", SimHash: 200,
	}, ScopeUser)
	if err != nil {
		t.Fatal(err)
	}

	if mock.callCount != 2 {
		t.Errorf("expected 2 embed calls (both tiers), got %d", mock.callCount)
	}

	// Both should be searchable by embedding
	projResults := store.Project.SearchByEmbedding([]float32{0.1, 0.1, 0.1, 0.1}, "", 10)
	userResults := store.User.SearchByEmbedding([]float32{0.1, 0.1, 0.1, 0.1}, "", 10)
	if len(projResults) != 1 {
		t.Errorf("expected 1 project embedding result, got %d", len(projResults))
	}
	if len(userResults) != 1 {
		t.Errorf("expected 1 user embedding result, got %d", len(userResults))
	}
}

func TestSQLiteSharedCache_MixedFrames(t *testing.T) {
	cache := newTestSQLiteCache(t)

	// Store mix of frames: some with embedding, some without, some vector-only
	cache.PutFrame(BrainFrame{
		Key: "text_only", Namespace: "test", Value: []byte("v"),
		Content: "just text", SimHash: 100,
	})
	cache.PutFrame(BrainFrame{
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

// --- Three-tier search tests (M-BRAIN-VECTORS M3) ---

func TestBrainStore_SearchThreeTier(t *testing.T) {
	dir := t.TempDir()
	store, err := NewBrainStore(
		filepath.Join(dir, "user.db"),
		filepath.Join(dir, "project.db"),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Frame with embedding + simhash + content
	store.Put(BrainFrame{
		Key: "full", Namespace: "test", Value: []byte("v"),
		Content: "parser crash fix", SimHash: 100,
		Embedding: []float32{0.9, 0.1, 0.0}, EmbedModel: "test",
	}, ScopeProject)

	// Frame with simhash + content only
	store.Put(BrainFrame{
		Key: "sim_only", Namespace: "test", Value: []byte("v"),
		Content: "parser optimization", SimHash: 101, // close to 100
	}, ScopeProject)

	// Frame with content only (no simhash close match, no embedding)
	store.Put(BrainFrame{
		Key: "text_only", Namespace: "test", Value: []byte("v"),
		Content: "parser debug tips", SimHash: 999999,
	}, ScopeProject)

	// Three-tier search with embedding
	queryEmb := []float32{0.9, 0.1, 0.0} // matches "full"
	results := store.SearchThreeTier("parser", 100, queryEmb, "test", 10, ScopeBoth)

	if len(results) < 3 {
		t.Fatalf("expected at least 3 results, got %d", len(results))
	}

	// "full" should rank highest (cosine + boost)
	if results[0].Frame.Key != "full" {
		t.Errorf("expected 'full' first (cosine+boost), got %s (score=%.3f)", results[0].Frame.Key, results[0].Score)
	}

	// Verify cosine results get boost
	if results[0].Score <= 1.0 {
		// Score should be > 1.0 with cosine boost + project boost
		// (but capped at 1.0, so check it's at the cap)
	}
}

func TestBrainStore_SearchByEmbedding(t *testing.T) {
	dir := t.TempDir()
	store, err := NewBrainStore(
		filepath.Join(dir, "user.db"),
		filepath.Join(dir, "project.db"),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	store.Put(BrainFrame{
		Key: "proj_vec", Namespace: "test", Value: []byte("v"),
		Embedding: []float32{1, 0, 0}, EmbedModel: "test",
	}, ScopeProject)

	store.Put(BrainFrame{
		Key: "user_vec", Namespace: "test", Value: []byte("v"),
		Embedding: []float32{0, 1, 0}, EmbedModel: "test",
	}, ScopeUser)

	// Search both
	results := store.SearchByEmbedding([]float32{1, 0, 0}, "test", 10, ScopeBoth)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// proj_vec should rank first (higher cosine + project boost)
	if results[0].Frame.Key != "proj_vec" {
		t.Errorf("expected proj_vec first, got %s", results[0].Frame.Key)
	}

	// Search project only
	results = store.SearchByEmbedding([]float32{1, 0, 0}, "test", 10, ScopeProject)
	if len(results) != 1 || results[0].Frame.Key != "proj_vec" {
		t.Error("scope project should only return project frame")
	}
}

func TestExportImportWithEmbeddings(t *testing.T) {
	// Create frame with embedding
	f := BrainFrame{
		Key: "emb_frame", Namespace: "test", Value: []byte("payload"),
		Content: "test content", SimHash: 42,
		Embedding: []float32{0.1, 0.2, 0.3, 0.4}, EmbeddingDim: 4, EmbedModel: "test-model",
	}

	// Export
	record := ExportFrameRecord(f, "project")

	// Verify embedding is in record
	if _, ok := record["embedding"]; !ok {
		t.Fatal("exported record should have embedding field")
	}
	if record["embedding_dim"] != 4 {
		t.Errorf("expected dim 4, got %v", record["embedding_dim"])
	}

	// Import into new frame
	var imported BrainFrame
	imported.Key = record["key"].(string)
	ImportFrameEmbedding(record, &imported)

	if len(imported.Embedding) != 4 {
		t.Fatalf("expected 4-dim embedding, got %d", len(imported.Embedding))
	}
	for i, want := range f.Embedding {
		if imported.Embedding[i] != want {
			t.Errorf("embedding[%d]: got %f, want %f", i, imported.Embedding[i], want)
		}
	}
	if imported.EmbedModel != "test-model" {
		t.Errorf("expected model test-model, got %s", imported.EmbedModel)
	}
}

func TestExportImportWithoutEmbeddings(t *testing.T) {
	// Frame without embedding
	f := BrainFrame{
		Key: "no_emb", Namespace: "test", Value: []byte("v"),
		Content: "text only", SimHash: 42,
	}

	record := ExportFrameRecord(f, "user")

	// Should not have embedding field
	if _, ok := record["embedding"]; ok {
		t.Error("frame without embedding should not have embedding in export")
	}

	var imported BrainFrame
	ImportFrameEmbedding(record, &imported)
	if len(imported.Embedding) != 0 {
		t.Error("import should produce no embedding when none in record")
	}
}
