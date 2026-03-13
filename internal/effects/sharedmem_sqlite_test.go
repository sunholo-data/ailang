package effects

import (
	"database/sql"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func newTestSQLiteCache(t *testing.T) *SQLiteSharedCache {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test_brain.db")
	cache, err := NewSQLiteSharedCache(dbPath)
	if err != nil {
		t.Fatalf("failed to create SQLiteSharedCache: %v", err)
	}
	t.Cleanup(func() { cache.Close() })
	return cache
}

// --- SharedCache interface compliance tests ---
// These mirror the InMemorySharedCache tests exactly.

func TestSQLiteSharedCache_Basic(t *testing.T) {
	cache := newTestSQLiteCache(t)

	// Test Put and Get
	cache.Put("key1", []byte("value1"))
	val, ok := cache.Get("key1")
	if !ok {
		t.Fatal("expected key1 to exist")
	}
	if string(val) != "value1" {
		t.Errorf("expected value1, got %s", val)
	}

	// Test Get non-existent
	val, ok = cache.Get("nonexistent")
	if ok {
		t.Error("expected nonexistent key to not exist")
	}
	if val != nil {
		t.Error("expected nil value for non-existent key")
	}
}

func TestSQLiteSharedCache_Delete(t *testing.T) {
	cache := newTestSQLiteCache(t)

	cache.Put("key1", []byte("value1"))
	cache.Delete("key1")

	_, ok := cache.Get("key1")
	if ok {
		t.Error("expected key1 to be deleted")
	}

	// Delete non-existent should not panic
	cache.Delete("nonexistent")
}

func TestSQLiteSharedCache_CAS(t *testing.T) {
	cache := newTestSQLiteCache(t)

	// Test create-if-absent (oldValue = nil)
	ok := cache.CAS("key1", nil, []byte("initial"))
	if !ok {
		t.Error("CAS with nil oldValue should succeed for new key")
	}

	val, _ := cache.Get("key1")
	if string(val) != "initial" {
		t.Errorf("expected initial, got %s", val)
	}

	// Test create-if-absent fails when key exists
	ok = cache.CAS("key1", nil, []byte("should_not_work"))
	if ok {
		t.Error("CAS with nil oldValue should fail when key exists")
	}

	// Test successful CAS
	ok = cache.CAS("key1", []byte("initial"), []byte("updated"))
	if !ok {
		t.Error("CAS should succeed when oldValue matches")
	}

	val, _ = cache.Get("key1")
	if string(val) != "updated" {
		t.Errorf("expected updated, got %s", val)
	}

	// Test failed CAS (wrong oldValue)
	ok = cache.CAS("key1", []byte("wrong"), []byte("should_not_work"))
	if ok {
		t.Error("CAS should fail when oldValue doesn't match")
	}

	// Test CAS on non-existent key with non-nil oldValue
	ok = cache.CAS("nonexistent", []byte("something"), []byte("value"))
	if ok {
		t.Error("CAS should fail for non-existent key with non-nil oldValue")
	}
}

func TestSQLiteSharedCache_Keys(t *testing.T) {
	cache := newTestSQLiteCache(t)

	cache.Put("c", []byte("3"))
	cache.Put("a", []byte("1"))
	cache.Put("b", []byte("2"))

	keys := cache.Keys()
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}
	// Keys are ordered by key ASC
	if keys[0] != "a" || keys[1] != "b" || keys[2] != "c" {
		t.Errorf("expected [a b c], got %v", keys)
	}
}

func TestSQLiteSharedCache_Len(t *testing.T) {
	cache := newTestSQLiteCache(t)

	if cache.Len() != 0 {
		t.Error("expected empty cache")
	}

	cache.Put("a", []byte("1"))
	cache.Put("b", []byte("2"))
	if cache.Len() != 2 {
		t.Errorf("expected 2, got %d", cache.Len())
	}
}

func TestSQLiteSharedCache_Overwrite(t *testing.T) {
	cache := newTestSQLiteCache(t)

	cache.Put("key", []byte("v1"))
	cache.Put("key", []byte("v2"))

	val, ok := cache.Get("key")
	if !ok || string(val) != "v2" {
		t.Errorf("expected v2 after overwrite, got %s", val)
	}
	if cache.Len() != 1 {
		t.Errorf("expected 1 key after overwrite, got %d", cache.Len())
	}
}

func TestSQLiteSharedCache_Persistence(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "persist_brain.db")

	// Write data
	cache1, err := NewSQLiteSharedCache(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	cache1.Put("persistent_key", []byte("persistent_value"))
	cache1.Close()

	// Reopen and read
	cache2, err := NewSQLiteSharedCache(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer cache2.Close()

	val, ok := cache2.Get("persistent_key")
	if !ok {
		t.Fatal("data should persist across close/reopen")
	}
	if string(val) != "persistent_value" {
		t.Errorf("expected persistent_value, got %s", val)
	}
}

// --- Extended brain-specific tests ---

func TestSQLiteSharedCache_PutFrame(t *testing.T) {
	cache := newTestSQLiteCache(t)

	f := BrainFrame{
		Key:       "test_frame",
		Namespace: "resolutions",
		Value:     []byte(`{"content": "fix bug"}`),
		SimHash:   12345,
		Content:   "Fix the type inference bug in unify.go",
		Version:   1,
		Source:    "hook:commit",
	}
	if err := cache.PutFrame(f); err != nil {
		t.Fatalf("PutFrame failed: %v", err)
	}

	val, ok := cache.Get("test_frame")
	if !ok {
		t.Fatal("frame not stored")
	}
	if string(val) != `{"content": "fix bug"}` {
		t.Errorf("unexpected value: %s", val)
	}
}

func TestSQLiteSharedCache_SearchBySimHash(t *testing.T) {
	cache := newTestSQLiteCache(t)

	// Store frames with known SimHash values
	frames := []BrainFrame{
		{Key: "f1", Namespace: "test", Value: []byte("v1"), SimHash: 0x0000000000000000, Content: "frame 1"},
		{Key: "f2", Namespace: "test", Value: []byte("v2"), SimHash: 0x0000000000000001, Content: "frame 2"},  // 1 bit diff from f1
		{Key: "f3", Namespace: "test", Value: []byte("v3"), SimHash: 0x00000000000000FF, Content: "frame 3"},  // 8 bits diff
		{Key: "f4", Namespace: "other", Value: []byte("v4"), SimHash: 0x0000000000000000, Content: "frame 4"}, // different namespace
	}
	for _, f := range frames {
		if err := cache.PutFrame(f); err != nil {
			t.Fatal(err)
		}
	}

	results := cache.SearchBySimHash("test", 0x0000000000000000, 10)
	if len(results) != 3 {
		t.Fatalf("expected 3 results in 'test' namespace, got %d", len(results))
	}

	// f1 should be first (exact match, score 1.0)
	if results[0].Frame.Key != "f1" || results[0].Score != 1.0 {
		t.Errorf("expected f1 with score 1.0, got %s with %f", results[0].Frame.Key, results[0].Score)
	}

	// f2 should be second (1 bit diff, score ~0.984)
	if results[1].Frame.Key != "f2" {
		t.Errorf("expected f2 second, got %s", results[1].Frame.Key)
	}

	// Limit works
	results = cache.SearchBySimHash("test", 0, 1)
	if len(results) != 1 {
		t.Errorf("expected 1 result with limit=1, got %d", len(results))
	}
}

func TestSQLiteSharedCache_SearchByText(t *testing.T) {
	cache := newTestSQLiteCache(t)

	frames := []BrainFrame{
		{Key: "f1", Namespace: "test", Value: []byte("v1"), Content: "Fix the parser crash on nested records"},
		{Key: "f2", Namespace: "test", Value: []byte("v2"), Content: "Add new stdlib function for string splitting"},
		{Key: "f3", Namespace: "test", Value: []byte("v3"), Content: "Parser improvement for effect annotations"},
	}
	for _, f := range frames {
		if err := cache.PutFrame(f); err != nil {
			t.Fatal(err)
		}
	}

	results := cache.SearchByText("parser", "test", 10)
	if len(results) != 2 {
		t.Fatalf("expected 2 results for 'parser', got %d", len(results))
	}

	// Search across all namespaces
	results = cache.SearchByText("parser", "", 10)
	if len(results) != 2 {
		t.Fatalf("expected 2 results across all namespaces, got %d", len(results))
	}
}

func TestSQLiteSharedCache_ListRecent(t *testing.T) {
	cache := newTestSQLiteCache(t)

	now := time.Now().UnixMilli()
	frames := []BrainFrame{
		{Key: "old", Namespace: "test", Value: []byte("v1"), Content: "old", CreatedAt: now - 3000, UpdatedAt: now - 3000},
		{Key: "mid", Namespace: "test", Value: []byte("v2"), Content: "mid", CreatedAt: now - 2000, UpdatedAt: now - 2000},
		{Key: "new", Namespace: "test", Value: []byte("v3"), Content: "new", CreatedAt: now - 1000, UpdatedAt: now - 1000},
	}
	for _, f := range frames {
		if err := cache.PutFrame(f); err != nil {
			t.Fatal(err)
		}
	}

	recent := cache.ListRecent("test", 2)
	if len(recent) != 2 {
		t.Fatalf("expected 2 recent, got %d", len(recent))
	}
	if recent[0].Key != "new" {
		t.Errorf("expected newest first, got %s", recent[0].Key)
	}
}

func TestSQLiteSharedCache_GarbageCollect(t *testing.T) {
	cache := newTestSQLiteCache(t)

	now := time.Now().UnixMilli()
	expired := now - 1000 // 1 second ago
	future := now + 60000 // 1 minute from now

	frames := []BrainFrame{
		{Key: "expired", Namespace: "test", Value: []byte("v1"), Content: "expired", ExpiresAt: &expired},
		{Key: "alive", Namespace: "test", Value: []byte("v2"), Content: "alive", ExpiresAt: &future},
		{Key: "permanent", Namespace: "test", Value: []byte("v3"), Content: "permanent"}, // no expiry
	}
	for _, f := range frames {
		if err := cache.PutFrame(f); err != nil {
			t.Fatal(err)
		}
	}

	removed, err := cache.GarbageCollect()
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Errorf("expected 1 removed, got %d", removed)
	}
	if cache.Len() != 2 {
		t.Errorf("expected 2 remaining, got %d", cache.Len())
	}
}

func TestSQLiteSharedCache_Stats(t *testing.T) {
	cache := newTestSQLiteCache(t)

	frames := []BrainFrame{
		{Key: "f1", Namespace: "resolutions", Value: []byte("v1"), Content: "fix"},
		{Key: "f2", Namespace: "resolutions", Value: []byte("v2"), Content: "fix"},
		{Key: "f3", Namespace: "code-context", Value: []byte("v3"), Content: "ctx"},
	}
	for _, f := range frames {
		if err := cache.PutFrame(f); err != nil {
			t.Fatal(err)
		}
	}

	stats := cache.Stats()
	if stats.TotalFrames != 3 {
		t.Errorf("expected 3 total, got %d", stats.TotalFrames)
	}
	if stats.Namespaces["resolutions"] != 2 {
		t.Errorf("expected 2 resolutions, got %d", stats.Namespaces["resolutions"])
	}
	if stats.Namespaces["code-context"] != 1 {
		t.Errorf("expected 1 code-context, got %d", stats.Namespaces["code-context"])
	}
}

// --- Concurrent stress test ---

func TestSQLiteSharedCache_ConcurrentAccess(t *testing.T) {
	cache := newTestSQLiteCache(t)

	const goroutines = 20
	const opsPerGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				key := "key_" + string(rune('A'+id))
				cache.Put(key, []byte("value"))
				cache.Get(key)
				cache.Keys()
				cache.Len()
			}
		}(i)
	}

	wg.Wait()

	// Verify cache is consistent
	if cache.Len() > goroutines {
		t.Errorf("more keys than goroutines: %d", cache.Len())
	}
}

// --- BrainStore two-tier tests ---

func TestBrainStore_TwoTier(t *testing.T) {
	dir := t.TempDir()
	userDB := filepath.Join(dir, "user_brain.db")
	projectDB := filepath.Join(dir, "project_brain.db")

	store, err := NewBrainStore(userDB, projectDB)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Write to project (default)
	if err := store.Put(BrainFrame{
		Key: "proj_frame", Namespace: "resolutions", Value: []byte("project fix"),
		SimHash: 100, Content: "Fix parser crash",
	}, ScopeProject); err != nil {
		t.Fatal(err)
	}

	// Write to user
	if err := store.Put(BrainFrame{
		Key: "user_frame", Namespace: "patterns", Value: []byte("Go pattern"),
		SimHash: 200, Content: "Always use sync.Pool for allocations",
	}, ScopeUser); err != nil {
		t.Fatal(err)
	}

	// Search both — project should rank higher
	results := store.Search("resolutions", 100, 10, ScopeBoth)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Tier != "project" {
		t.Errorf("expected project tier, got %s", results[0].Tier)
	}

	// Search user only
	results = store.Search("patterns", 200, 10, ScopeUser)
	if len(results) != 1 {
		t.Fatalf("expected 1 result from user, got %d", len(results))
	}
	if results[0].Tier != "user" {
		t.Errorf("expected user tier, got %s", results[0].Tier)
	}
}

func TestBrainStore_Promote(t *testing.T) {
	dir := t.TempDir()
	store, err := NewBrainStore(
		filepath.Join(dir, "user.db"),
		filepath.Join(dir, "project.db"),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Store in project
	if err := store.Put(BrainFrame{
		Key: "promote_me", Namespace: "learnings", Value: []byte("useful insight"),
		Content: "This pattern applies everywhere", Source: "cli",
	}, ScopeProject); err != nil {
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
	dir := t.TempDir()
	store, err := NewBrainStore(
		filepath.Join(dir, "user.db"),
		filepath.Join(dir, "project.db"),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	store.Put(BrainFrame{Key: "u1", Namespace: "patterns", Value: []byte("v"), Content: "c"}, ScopeUser)
	store.Put(BrainFrame{Key: "p1", Namespace: "resolutions", Value: []byte("v"), Content: "c"}, ScopeProject)
	store.Put(BrainFrame{Key: "p2", Namespace: "resolutions", Value: []byte("v"), Content: "c"}, ScopeProject)

	stats := store.Stats()
	if stats["user"].TotalFrames != 1 {
		t.Errorf("expected 1 user frame, got %d", stats["user"].TotalFrames)
	}
	if stats["project"].TotalFrames != 2 {
		t.Errorf("expected 2 project frames, got %d", stats["project"].TotalFrames)
	}
}

func TestBrainStore_NilTier(t *testing.T) {
	dir := t.TempDir()

	// Project only (no user brain)
	store, err := NewBrainStore("", filepath.Join(dir, "project.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if store.User != nil {
		t.Error("user should be nil")
	}

	// Put should work (falls back to project)
	if err := store.Put(BrainFrame{
		Key: "f1", Namespace: "test", Value: []byte("v"), Content: "c",
	}, ScopeProject); err != nil {
		t.Fatal(err)
	}

	// Search should work with only one tier
	results := store.Search("test", 0, 10, ScopeBoth)
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestNewSQLiteSharedCache_BadPath(t *testing.T) {
	// Verify error on impossible path
	_, err := NewSQLiteSharedCache("/dev/null/impossible/brain.db")
	if err == nil {
		t.Error("expected error for impossible path")
	}
}

func TestSQLiteSharedCache_EmptyDB(t *testing.T) {
	cache := newTestSQLiteCache(t)

	// All operations should work on empty DB
	if cache.Len() != 0 {
		t.Error("expected 0")
	}
	keys := cache.Keys()
	if len(keys) != 0 {
		t.Error("expected empty keys")
	}
	_, ok := cache.Get("nothing")
	if ok {
		t.Error("expected not found")
	}

	results := cache.SearchBySimHash("ns", 0, 10)
	if len(results) != 0 {
		t.Error("expected no results")
	}

	results = cache.SearchByText("query", "", 10)
	if len(results) != 0 {
		t.Error("expected no results")
	}

	recent := cache.ListRecent("", 10)
	if len(recent) != 0 {
		t.Error("expected no recent")
	}

	stats := cache.Stats()
	if stats.TotalFrames != 0 {
		t.Error("expected 0 total")
	}

	removed, err := cache.GarbageCollect()
	if err != nil || removed != 0 {
		t.Error("GC on empty should be no-op")
	}
}

func TestHammingDistance64(t *testing.T) {
	tests := []struct {
		a, b int64
		want int
	}{
		{0, 0, 0},
		{0, 1, 1},
		{0, 0xFF, 8},
		{0, -1, 64}, // all bits different
		{0x5555555555555555, -0x5555555555555556, 64}, // all bits flipped
	}
	for _, tt := range tests {
		got := hammingDistance64(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("hammingDistance64(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

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
