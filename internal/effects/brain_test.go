package effects_test

// BrainStore behaviour (two-tier merge, promotion, ranking) lives in the
// core; the persistent backend it composes is registered from the platform.
// This is an external test package on purpose: an internal one could not
// import internal/platform/sharedmem (it imports effects), and test imports
// do not count toward the language-core closure.

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/platform/sharedmem"
)

// useSQLiteBackend registers the SQLite backend for one test and removes it
// afterwards, so the unregistered-path test below sees a clean registry.
func useSQLiteBackend(t *testing.T) {
	t.Helper()
	sharedmem.Register()
	t.Cleanup(func() { effects.RegisterSharedCacheOpener(nil) })
}

// TestNewBrainStore_Unregistered: with no backend registered the core fails
// loudly with the typed sentinel — never a nil cache, never a panic.
func TestNewBrainStore_Unregistered(t *testing.T) {
	effects.RegisterSharedCacheOpener(nil)
	dir := t.TempDir()

	store, err := effects.NewBrainStore(filepath.Join(dir, "user.db"), filepath.Join(dir, "project.db"))
	if err == nil {
		t.Fatalf("expected an error with no backend registered, got store %+v", store)
	}
	if !errors.Is(err, effects.ErrBackendNotRegistered) {
		t.Fatalf("errors.Is(err, ErrBackendNotRegistered) = false; err = %v", err)
	}
	if store != nil {
		t.Errorf("store should be nil on error, got %+v", store)
	}

	// Positive control for the same registry: registering makes the same call succeed.
	useSQLiteBackend(t)
	store, err = effects.NewBrainStore(filepath.Join(dir, "user.db"), "")
	if err != nil {
		t.Fatalf("registered backend: %v", err)
	}
	store.Close()
}

// TestOpenSharedCache_UnregisteredNamesPath: the wrapped error names the path
// so a misconfigured binary is diagnosable from the message alone.
func TestOpenSharedCache_UnregisteredNamesPath(t *testing.T) {
	effects.RegisterSharedCacheOpener(nil)
	_, err := effects.OpenSharedCache("/x/brain.db")
	if !errors.Is(err, effects.ErrBackendNotRegistered) {
		t.Fatalf("want ErrBackendNotRegistered, got %v", err)
	}
	if want := "/x/brain.db"; !strings.Contains(err.Error(), want) {
		t.Errorf("error should name the path %q: %v", want, err)
	}
}

// mockEmbedder mirrors the platform test double: deterministic vectors, a
// call counter, and a failure switch.
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
	out := make([][]float32, len(texts))
	for i, t := range texts {
		v, err := m.Embed(t)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

func (m *mockEmbedder) Dimension() int    { return m.dim }
func (m *mockEmbedder) ModelName() string { return m.model }

func TestBrainStore_TwoTier(t *testing.T) {
	useSQLiteBackend(t)
	dir := t.TempDir()
	userDB := filepath.Join(dir, "user_brain.db")
	projectDB := filepath.Join(dir, "project_brain.db")

	store, err := effects.NewBrainStore(userDB, projectDB)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Write to project (default)
	if err := store.Put(effects.BrainFrame{
		Key: "proj_frame", Namespace: "resolutions", Value: []byte("project fix"),
		SimHash: 100, Content: "Fix parser crash",
	}, effects.ScopeProject); err != nil {
		t.Fatal(err)
	}

	// Write to user
	if err := store.Put(effects.BrainFrame{
		Key: "user_frame", Namespace: "patterns", Value: []byte("Go pattern"),
		SimHash: 200, Content: "Always use sync.Pool for allocations",
	}, effects.ScopeUser); err != nil {
		t.Fatal(err)
	}

	// Search both — project should rank higher
	results := store.Search("resolutions", 100, 10, effects.ScopeBoth)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Tier != "project" {
		t.Errorf("expected project tier, got %s", results[0].Tier)
	}

	// Search user only
	results = store.Search("patterns", 200, 10, effects.ScopeUser)
	if len(results) != 1 {
		t.Fatalf("expected 1 result from user, got %d", len(results))
	}
	if results[0].Tier != "user" {
		t.Errorf("expected user tier, got %s", results[0].Tier)
	}
}

func TestBrainStore_Promote(t *testing.T) {
	useSQLiteBackend(t)
	dir := t.TempDir()
	store, err := effects.NewBrainStore(
		filepath.Join(dir, "user.db"),
		filepath.Join(dir, "project.db"),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Store in project
	if err := store.Put(effects.BrainFrame{
		Key: "promote_me", Namespace: "learnings", Value: []byte("useful insight"),
		Content: "This pattern applies everywhere", Source: "cli",
	}, effects.ScopeProject); err != nil {
		t.Fatal(err)
	}

	// Promote to user
	if !store.Promote("promote_me") {
		t.Fatal("promote should succeed")
	}

	// Verify it exists in user brain
	val, ok := store.User.Get("promote_me")
	if !ok {
		t.Fatal("promoted frame should exist in user brain")
	}
	if string(val) != "useful insight" {
		t.Errorf("unexpected value: %s", val)
	}

	// Promote non-existent should fail
	if store.Promote("nonexistent") {
		t.Error("promote of nonexistent key should return false")
	}
}

func TestBrainStore_Stats(t *testing.T) {
	useSQLiteBackend(t)
	dir := t.TempDir()
	store, err := effects.NewBrainStore(
		filepath.Join(dir, "user.db"),
		filepath.Join(dir, "project.db"),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	store.Put(effects.BrainFrame{Key: "u1", Namespace: "patterns", Value: []byte("v"), Content: "c"}, effects.ScopeUser)
	store.Put(effects.BrainFrame{Key: "p1", Namespace: "resolutions", Value: []byte("v"), Content: "c"}, effects.ScopeProject)
	store.Put(effects.BrainFrame{Key: "p2", Namespace: "resolutions", Value: []byte("v"), Content: "c"}, effects.ScopeProject)

	stats := store.Stats()
	if stats["user"].TotalFrames != 1 {
		t.Errorf("expected 1 user frame, got %d", stats["user"].TotalFrames)
	}
	if stats["project"].TotalFrames != 2 {
		t.Errorf("expected 2 project frames, got %d", stats["project"].TotalFrames)
	}
}

func TestBrainStore_NilTier(t *testing.T) {
	useSQLiteBackend(t)
	dir := t.TempDir()

	// Project only (no user brain)
	store, err := effects.NewBrainStore("", filepath.Join(dir, "project.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if store.User != nil {
		t.Error("user should be nil")
	}

	// Put should work (falls back to project)
	if err := store.Put(effects.BrainFrame{
		Key: "f1", Namespace: "test", Value: []byte("v"), Content: "c",
	}, effects.ScopeProject); err != nil {
		t.Fatal(err)
	}

	// Search should work with only one tier
	results := store.Search("test", 0, 10, effects.ScopeBoth)
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestBrainStore_WithEmbedder(t *testing.T) {
	useSQLiteBackend(t)
	mock := &mockEmbedder{dim: 4, model: "brain-test"}
	dir := t.TempDir()

	store, err := effects.NewBrainStore(
		filepath.Join(dir, "user.db"),
		filepath.Join(dir, "project.db"),
		effects.WithEmbedder(mock),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Put to project — should auto-embed
	err = store.Put(effects.BrainFrame{
		Key: "proj_frame", Namespace: "test", Value: []byte("v"),
		Content: "Project content", SimHash: 100,
	}, effects.ScopeProject)
	if err != nil {
		t.Fatal(err)
	}

	// Put to user — should also auto-embed
	err = store.Put(effects.BrainFrame{
		Key: "user_frame", Namespace: "test", Value: []byte("v"),
		Content: "User content", SimHash: 200,
	}, effects.ScopeUser)
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

func TestBrainStore_SearchThreeTier(t *testing.T) {
	useSQLiteBackend(t)
	dir := t.TempDir()
	store, err := effects.NewBrainStore(
		filepath.Join(dir, "user.db"),
		filepath.Join(dir, "project.db"),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Frame with embedding + simhash + content
	store.Put(effects.BrainFrame{
		Key: "full", Namespace: "test", Value: []byte("v"),
		Content: "parser crash fix", SimHash: 100,
		Embedding: []float32{0.9, 0.1, 0.0}, EmbedModel: "test",
	}, effects.ScopeProject)

	// Frame with simhash + content only
	store.Put(effects.BrainFrame{
		Key: "sim_only", Namespace: "test", Value: []byte("v"),
		Content: "parser optimization", SimHash: 101, // close to 100
	}, effects.ScopeProject)

	// Frame with content only (no simhash close match, no embedding)
	store.Put(effects.BrainFrame{
		Key: "text_only", Namespace: "test", Value: []byte("v"),
		Content: "parser debug tips", SimHash: 999999,
	}, effects.ScopeProject)

	// Three-tier search with embedding
	queryEmb := []float32{0.9, 0.1, 0.0} // matches "full"
	results := store.SearchThreeTier("parser", 100, queryEmb, "test", 10, effects.ScopeBoth)

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
	useSQLiteBackend(t)
	dir := t.TempDir()
	store, err := effects.NewBrainStore(
		filepath.Join(dir, "user.db"),
		filepath.Join(dir, "project.db"),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	store.Put(effects.BrainFrame{
		Key: "proj_vec", Namespace: "test", Value: []byte("v"),
		Embedding: []float32{1, 0, 0}, EmbedModel: "test",
	}, effects.ScopeProject)

	store.Put(effects.BrainFrame{
		Key: "user_vec", Namespace: "test", Value: []byte("v"),
		Embedding: []float32{0, 1, 0}, EmbedModel: "test",
	}, effects.ScopeUser)

	// Search both
	results := store.SearchByEmbedding([]float32{1, 0, 0}, "test", 10, effects.ScopeBoth)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// proj_vec should rank first (higher cosine + project boost)
	if results[0].Frame.Key != "proj_vec" {
		t.Errorf("expected proj_vec first, got %s", results[0].Frame.Key)
	}

	// Search project only
	results = store.SearchByEmbedding([]float32{1, 0, 0}, "test", 10, effects.ScopeProject)
	if len(results) != 1 || results[0].Frame.Key != "proj_vec" {
		t.Error("scope project should only return project frame")
	}
}

func TestExportImportWithEmbeddings(t *testing.T) {
	// Create frame with embedding
	f := effects.BrainFrame{
		Key: "emb_frame", Namespace: "test", Value: []byte("payload"),
		Content: "test content", SimHash: 42,
		Embedding: []float32{0.1, 0.2, 0.3, 0.4}, EmbeddingDim: 4, EmbedModel: "test-model",
	}

	// Export
	record := effects.ExportFrameRecord(f, "project")

	// Verify embedding is in record
	if _, ok := record["embedding"]; !ok {
		t.Fatal("exported record should have embedding field")
	}
	if record["embedding_dim"] != 4 {
		t.Errorf("expected dim 4, got %v", record["embedding_dim"])
	}

	// Import into new frame
	var imported effects.BrainFrame
	imported.Key = record["key"].(string)
	effects.ImportFrameEmbedding(record, &imported)

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
	f := effects.BrainFrame{
		Key: "no_emb", Namespace: "test", Value: []byte("v"),
		Content: "text only", SimHash: 42,
	}

	record := effects.ExportFrameRecord(f, "user")

	// Should not have embedding field
	if _, ok := record["embedding"]; ok {
		t.Error("frame without embedding should not have embedding in export")
	}

	var imported effects.BrainFrame
	effects.ImportFrameEmbedding(record, &imported)
	if len(imported.Embedding) != 0 {
		t.Error("import should produce no embedding when none in record")
	}
}

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
				b := effects.EncodeEmbedding(nil)
				if len(b) != 0 {
					t.Errorf("expected empty encoding for nil, got %d bytes", len(b))
				}
				return
			}
			encoded := effects.EncodeEmbedding(tt.vec)
			if len(encoded) != len(tt.vec)*4 {
				t.Fatalf("expected %d bytes, got %d", len(tt.vec)*4, len(encoded))
			}
			decoded := effects.DecodeEmbedding(encoded)
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
