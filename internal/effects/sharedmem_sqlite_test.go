package effects

import (
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
