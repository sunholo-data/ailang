package pipeline

import (
	"os"
	"testing"
	"time"
)

func TestCacheStore_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	cs, err := NewCacheStore(dir)
	if err != nil {
		t.Fatalf("NewCacheStore: %v", err)
	}

	// Store an entry
	cs.Store("std/list", &CacheEntry{
		CacheKey:      "abc123",
		IfaceDigest:   "digest456",
		IfaceJSON:     []byte(`{"module":"std/list"}`),
		CompileTimeMs: 5,
		Timestamp:     time.Now(),
	})

	// Save to disk
	if err := cs.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Reload from disk
	cs2, err := NewCacheStore(dir)
	if err != nil {
		t.Fatalf("NewCacheStore reload: %v", err)
	}

	// Lookup with correct key
	entry, ok := cs2.Lookup("std/list", "abc123")
	if !ok {
		t.Fatal("expected cache hit for std/list")
	}
	if entry.IfaceDigest != "digest456" {
		t.Errorf("digest mismatch: got %s", entry.IfaceDigest)
	}
	if string(entry.IfaceJSON) != `{"module":"std/list"}` {
		t.Errorf("iface JSON mismatch: got %s", string(entry.IfaceJSON))
	}

	// Lookup with wrong key = miss
	_, ok = cs2.Lookup("std/list", "wrong_key")
	if ok {
		t.Error("expected cache miss for wrong key")
	}

	// Lookup missing module = miss
	_, ok = cs2.Lookup("std/nonexistent", "abc123")
	if ok {
		t.Error("expected cache miss for missing module")
	}
}

func TestCacheStore_Clear(t *testing.T) {
	dir := t.TempDir()
	cs, err := NewCacheStore(dir)
	if err != nil {
		t.Fatalf("NewCacheStore: %v", err)
	}

	cs.Store("mod1", &CacheEntry{CacheKey: "k1", Timestamp: time.Now()})
	cs.Store("mod2", &CacheEntry{CacheKey: "k2", Timestamp: time.Now()})
	if err := cs.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	entries, _ := cs.Stats()
	if entries != 2 {
		t.Errorf("expected 2 entries, got %d", entries)
	}

	if err := cs.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	entries, _ = cs.Stats()
	if entries != 0 {
		t.Errorf("expected 0 entries after clear, got %d", entries)
	}
}

func TestCacheStore_CorruptedManifest(t *testing.T) {
	dir := t.TempDir()

	// Create corrupt manifest
	cacheDir := dir + "/.ailang/cache/compile"
	os.MkdirAll(cacheDir, 0755)
	os.WriteFile(cacheDir+"/manifest.json", []byte("not json"), 0644)

	// Should handle gracefully
	cs, err := NewCacheStore(dir)
	if err != nil {
		t.Fatalf("NewCacheStore should not fail on corrupt: %v", err)
	}

	entries, _ := cs.Stats()
	if entries != 0 {
		t.Errorf("expected 0 entries for fresh cache, got %d", entries)
	}
}

func TestCacheStore_Stats(t *testing.T) {
	dir := t.TempDir()
	cs, err := NewCacheStore(dir)
	if err != nil {
		t.Fatalf("NewCacheStore: %v", err)
	}

	cs.Store("mod1", &CacheEntry{CacheKey: "k1", CompileTimeMs: 10, Timestamp: time.Now()})
	cs.Store("mod2", &CacheEntry{CacheKey: "k2", CompileTimeMs: 20, Timestamp: time.Now()})

	entries, totalMs := cs.Stats()
	if entries != 2 {
		t.Errorf("expected 2 entries, got %d", entries)
	}
	if totalMs != 30 {
		t.Errorf("expected 30ms total, got %d", totalMs)
	}
}
