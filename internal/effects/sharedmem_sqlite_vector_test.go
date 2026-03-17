package effects

import (
	"database/sql"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// --- Embedding / Vector tests (M-BRAIN-VECTORS M1) ---

func TestEncodeDecodeEmbedding(t *testing.T) {
	tests := []struct {
		name string
		vec  []float32
	}{
		{"empty", nil},
		{"single", []float32{1.0}},
		{"typical", []float32{0.1, -0.5, 0.9, 0.0, -1.0}},
		{"zeros", []float32{0, 0, 0, 0}},
		{"large_dim", func() []float32 {
			v := make([]float32, 768)
			for i := range v {
				v[i] = float32(i) / 768.0
			}
			return v
		}()},
		{"extremes", []float32{math.MaxFloat32, -math.MaxFloat32, math.SmallestNonzeroFloat32}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.vec == nil {
				// nil encodes to empty, decodes to empty slice
				b := encodeEmbedding(nil)
				if len(b) != 0 {
					t.Errorf("expected empty encoding for nil, got %d bytes", len(b))
				}
				return
			}
			encoded := encodeEmbedding(tt.vec)
			if len(encoded) != len(tt.vec)*4 {
				t.Fatalf("expected %d bytes, got %d", len(tt.vec)*4, len(encoded))
			}
			decoded := decodeEmbedding(encoded)
			if len(decoded) != len(tt.vec) {
				t.Fatalf("expected %d elements, got %d", len(tt.vec), len(decoded))
			}
			for i, v := range tt.vec {
				if decoded[i] != v {
					t.Errorf("index %d: expected %v, got %v", i, v, decoded[i])
				}
			}
		})
	}
}

func TestCosineSimilarityF32(t *testing.T) {
	tests := []struct {
		name string
		a, b []float32
		want float64
		tol  float64
	}{
		{"identical", []float32{1, 0, 0}, []float32{1, 0, 0}, 1.0, 1e-9},
		{"opposite", []float32{1, 0, 0}, []float32{-1, 0, 0}, -1.0, 1e-9},
		{"orthogonal", []float32{1, 0, 0}, []float32{0, 1, 0}, 0.0, 1e-9},
		{"similar", []float32{1, 1, 0}, []float32{1, 0, 0}, 0.7071, 0.001},
		{"empty_a", nil, []float32{1, 0, 0}, 0.0, 1e-9},
		{"empty_b", []float32{1, 0, 0}, nil, 0.0, 1e-9},
		{"zero_vec", []float32{0, 0, 0}, []float32{1, 1, 1}, 0.0, 1e-9},
		{"dim_mismatch", []float32{1, 0}, []float32{1, 0, 0}, 1.0, 1e-9}, // uses shorter
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cosineSimilarityF32(tt.a, tt.b)
			diff := got - tt.want
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.tol {
				t.Errorf("cosineSimilarityF32(%v, %v) = %f, want %f (±%f)", tt.a, tt.b, got, tt.want, tt.tol)
			}
		})
	}
}

func TestSQLiteSharedCache_SchemaMigrationV1toV2(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "migrate_brain.db")

	// Create a v1 database manually (without embedding columns)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		CREATE TABLE brain_frames (
			key TEXT PRIMARY KEY, namespace TEXT NOT NULL DEFAULT 'default',
			value BLOB NOT NULL, simhash INTEGER, content TEXT,
			version INTEGER DEFAULT 1, created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL, expires_at INTEGER, source TEXT
		);
		CREATE TABLE brain_meta (key TEXT PRIMARY KEY, value TEXT);
		INSERT INTO brain_meta(key, value) VALUES('schema_version', '1.0.0');
		INSERT INTO brain_frames(key, namespace, value, simhash, content, version, created_at, updated_at)
			VALUES('existing_frame', 'learnings', X'68656C6C6F', 42, 'Hello world', 1, 1000, 2000);
	`)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	// Open with v2 code — should migrate non-destructively
	cache, err := NewSQLiteSharedCache(dbPath)
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	defer cache.Close()

	// Verify existing data is preserved
	val, ok := cache.Get("existing_frame")
	if !ok {
		t.Fatal("existing frame should survive migration")
	}
	if string(val) != "hello" {
		t.Errorf("expected 'hello', got %q", val)
	}

	// Verify new columns exist and accept data
	err = cache.PutFrame(BrainFrame{
		Key: "new_frame", Namespace: "test", Value: []byte("v"),
		Content: "test", Embedding: []float32{0.1, 0.2, 0.3},
		EmbedModel: "ollama:gemma",
	})
	if err != nil {
		t.Fatalf("PutFrame with embedding after migration: %v", err)
	}

	// Verify embedding round-trips
	results := cache.SearchByEmbedding([]float32{0.1, 0.2, 0.3}, "", 10)
	if len(results) != 1 {
		t.Fatalf("expected 1 embedding result, got %d", len(results))
	}
	if results[0].Frame.Key != "new_frame" {
		t.Errorf("expected new_frame, got %s", results[0].Frame.Key)
	}
	if results[0].Score < 0.999 {
		t.Errorf("expected ~1.0 for identical embedding, got %f", results[0].Score)
	}

	// Verify schema version updated
	var version string
	cache.db.QueryRow(`SELECT value FROM brain_meta WHERE key='schema_version'`).Scan(&version)
	if version != "2.0.0" {
		t.Errorf("expected schema version 2.0.0, got %s", version)
	}
}

func TestSQLiteSharedCache_PutVector(t *testing.T) {
	cache := newTestSQLiteCache(t)

	embedding := []float32{0.5, -0.3, 0.8, 0.1}
	payload := []byte(`{"type": "task_embedding", "task_id": "T123"}`)

	err := cache.PutVector("vec_001", "vectors", embedding, "ollama:gemma", payload)
	if err != nil {
		t.Fatalf("PutVector failed: %v", err)
	}

	// Verify stored
	val, ok := cache.Get("vec_001")
	if !ok {
		t.Fatal("vector frame not stored")
	}
	if string(val) != string(payload) {
		t.Errorf("payload mismatch: got %s", val)
	}

	// Verify embedding stored (search should find it)
	results := cache.SearchByEmbedding(embedding, "vectors", 10)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Frame.EmbedModel != "ollama:gemma" {
		t.Errorf("expected model ollama:gemma, got %s", results[0].Frame.EmbedModel)
	}
	if results[0].Frame.EmbeddingDim != 4 {
		t.Errorf("expected dim 4, got %d", results[0].Frame.EmbeddingDim)
	}

	// Content should be empty (vector-only frame)
	if results[0].Frame.Content != "" {
		t.Errorf("vector frame should have no content, got %q", results[0].Frame.Content)
	}
}

func TestSQLiteSharedCache_SearchByEmbedding(t *testing.T) {
	cache := newTestSQLiteCache(t)

	// Store frames with known embeddings
	frames := []struct {
		key       string
		embedding []float32
	}{
		{"close", []float32{0.9, 0.1, 0.0}},      // very similar to query
		{"medium", []float32{0.5, 0.5, 0.5}},     // somewhat similar
		{"far", []float32{-0.9, -0.1, 0.0}},      // opposite
		{"orthogonal", []float32{0.0, 0.0, 1.0}}, // orthogonal
	}
	for _, f := range frames {
		err := cache.PutVector(f.key, "test", f.embedding, "test-model", []byte("payload"))
		if err != nil {
			t.Fatal(err)
		}
	}

	// Also store a frame WITHOUT embedding
	err := cache.PutFrame(BrainFrame{
		Key: "no_embed", Namespace: "test", Value: []byte("v"),
		Content: "no embedding here", SimHash: 42,
	})
	if err != nil {
		t.Fatal(err)
	}

	query := []float32{1.0, 0.0, 0.0}
	results := cache.SearchByEmbedding(query, "test", 10)

	// Should not include no_embed (no embedding)
	if len(results) != 4 {
		t.Fatalf("expected 4 results (excluding no_embed), got %d", len(results))
	}

	// "close" should rank first (most similar to [1,0,0])
	if results[0].Frame.Key != "close" {
		t.Errorf("expected 'close' first, got %s (score=%f)", results[0].Frame.Key, results[0].Score)
	}

	// "far" should rank last (opposite direction)
	if results[len(results)-1].Frame.Key != "far" {
		t.Errorf("expected 'far' last, got %s", results[len(results)-1].Frame.Key)
	}

	// Verify scores are in descending order
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("results not sorted: [%d].Score=%f > [%d].Score=%f",
				i, results[i].Score, i-1, results[i-1].Score)
		}
	}

	// Test namespace filtering
	err = cache.PutVector("other_ns", "other", []float32{1.0, 0.0, 0.0}, "test-model", []byte("p"))
	if err != nil {
		t.Fatal(err)
	}

	results = cache.SearchByEmbedding(query, "test", 10)
	for _, r := range results {
		if r.Frame.Namespace != "test" {
			t.Errorf("namespace filter not working: got %s", r.Frame.Namespace)
		}
	}

	// Empty namespace searches all
	results = cache.SearchByEmbedding(query, "", 10)
	if len(results) != 5 {
		t.Errorf("expected 5 results across all namespaces, got %d", len(results))
	}

	// Limit works
	results = cache.SearchByEmbedding(query, "", 2)
	if len(results) != 2 {
		t.Errorf("expected 2 results with limit=2, got %d", len(results))
	}
}

func TestSQLiteSharedCache_PutFrameWithEmbedding(t *testing.T) {
	cache := newTestSQLiteCache(t)

	// Store a frame with both content and embedding
	err := cache.PutFrame(BrainFrame{
		Key: "hybrid", Namespace: "learnings", Value: []byte("v"),
		Content: "Use sync.Pool for hot paths", SimHash: 12345,
		Embedding: []float32{0.1, 0.2, 0.3, 0.4}, EmbedModel: "gemini",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Should be findable by both SimHash and embedding
	simResults := cache.SearchBySimHash("learnings", 12345, 10)
	if len(simResults) != 1 || simResults[0].Frame.Key != "hybrid" {
		t.Error("should be found by SimHash")
	}

	embResults := cache.SearchByEmbedding([]float32{0.1, 0.2, 0.3, 0.4}, "learnings", 10)
	if len(embResults) != 1 || embResults[0].Frame.Key != "hybrid" {
		t.Error("should be found by embedding")
	}

	// Embedding fields should round-trip
	f := embResults[0].Frame
	if f.EmbeddingDim != 4 {
		t.Errorf("expected dim 4, got %d", f.EmbeddingDim)
	}
	if f.EmbedModel != "gemini" {
		t.Errorf("expected model gemini, got %s", f.EmbedModel)
	}
	if len(f.Embedding) != 4 {
		t.Fatalf("expected 4 floats, got %d", len(f.Embedding))
	}
	for i, want := range []float32{0.1, 0.2, 0.3, 0.4} {
		if f.Embedding[i] != want {
			t.Errorf("embedding[%d]: got %f, want %f", i, f.Embedding[i], want)
		}
	}
}

func TestSQLiteSharedCache_EmbeddingStats(t *testing.T) {
	cache := newTestSQLiteCache(t)

	// 2 frames with embedding, 1 without
	cache.PutVector("v1", "test", []float32{1, 0}, "ollama:gemma", []byte("p"))
	cache.PutVector("v2", "test", []float32{0, 1}, "gemini", []byte("p"))
	cache.PutFrame(BrainFrame{Key: "no_emb", Namespace: "test", Value: []byte("v"), Content: "c"})

	total, withEmb, models := cache.EmbeddingStats()
	if total != 3 {
		t.Errorf("expected 3 total, got %d", total)
	}
	if withEmb != 2 {
		t.Errorf("expected 2 with embeddings, got %d", withEmb)
	}
	if models["ollama:gemma"] != 1 {
		t.Errorf("expected 1 ollama:gemma, got %d", models["ollama:gemma"])
	}
	if models["gemini"] != 1 {
		t.Errorf("expected 1 gemini, got %d", models["gemini"])
	}
}

// --- Mock embedder for testing ---

type mockEmbedder struct {
	dim       int
	model     string
	callCount int
	failNext  bool
}

func (m *mockEmbedder) Embed(text string) ([]float32, error) {
	m.callCount++
	if m.failNext {
		return nil, fmt.Errorf("mock embedder error")
	}
	// Generate deterministic embedding from text length
	v := make([]float32, m.dim)
	for i := range v {
		v[i] = float32(len(text)+i) / float32(m.dim*100)
	}
	return v, nil
}

func (m *mockEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, t := range texts {
		emb, err := m.Embed(t)
		if err != nil {
			return nil, err
		}
		results[i] = emb
	}
	return results, nil
}

func (m *mockEmbedder) Dimension() int    { return m.dim }
func (m *mockEmbedder) ModelName() string { return m.model }

func TestSQLiteSharedCache_FileCreated(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "subdir", "brain.db")

	cache, err := NewSQLiteSharedCache(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()

	// Verify file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file should have been created")
	}
}
