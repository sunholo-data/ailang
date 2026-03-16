package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// M-PERF6 M3: Disk-based compilation cache for module interfaces.
//
// Stores a manifest mapping module IDs to their cache keys. When a module's
// source + dependency digests haven't changed, the pipeline can skip
// recompilation for that module.
//
// Cache location: .ailang/cache/compile/manifest.json

// CacheStore manages the on-disk compilation cache.
type CacheStore struct {
	dir      string
	manifest *CacheManifest
}

// CacheManifest tracks cached module compilation state.
type CacheManifest struct {
	Version string                 `json:"version"`
	Entries map[string]*CacheEntry `json:"entries"`
}

// CacheEntry represents a single cached module.
type CacheEntry struct {
	CacheKey      string    `json:"cache_key"`
	IfaceDigest   string    `json:"iface_digest"`
	IfaceJSON     []byte    `json:"iface_json,omitempty"`
	CompileTimeMs int64     `json:"compile_time_ms"`
	Timestamp     time.Time `json:"timestamp"`
}

// NewCacheStore creates or loads a cache store from the given directory.
func NewCacheStore(projectDir string) (*CacheStore, error) {
	dir := filepath.Join(projectDir, ".ailang", "cache", "compile")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}

	cs := &CacheStore{dir: dir}
	if err := cs.load(); err != nil {
		// Corrupted cache — start fresh
		cs.manifest = &CacheManifest{
			Version: cacheKeyVersion,
			Entries: make(map[string]*CacheEntry),
		}
	}
	return cs, nil
}

// Lookup checks if a module has a valid cache entry for the given key.
func (cs *CacheStore) Lookup(moduleID, cacheKey string) (*CacheEntry, bool) {
	entry, ok := cs.manifest.Entries[moduleID]
	if !ok || entry.CacheKey != cacheKey {
		return nil, false
	}
	return entry, true
}

// Store writes a cache entry for a module.
func (cs *CacheStore) Store(moduleID string, entry *CacheEntry) {
	cs.manifest.Entries[moduleID] = entry
}

// Save persists the manifest to disk.
func (cs *CacheStore) Save() error {
	data, err := json.MarshalIndent(cs.manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	path := filepath.Join(cs.dir, "manifest.json")
	return os.WriteFile(path, data, 0644)
}

// Clear removes all cache entries.
func (cs *CacheStore) Clear() error {
	cs.manifest = &CacheManifest{
		Version: cacheKeyVersion,
		Entries: make(map[string]*CacheEntry),
	}
	return cs.Save()
}

// Stats returns cache statistics.
func (cs *CacheStore) Stats() (entries int, totalCompileMs int64) {
	for _, e := range cs.manifest.Entries {
		entries++
		totalCompileMs += e.CompileTimeMs
	}
	return
}

func (cs *CacheStore) load() error {
	path := filepath.Join(cs.dir, "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var manifest CacheManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return err
	}
	if manifest.Version != cacheKeyVersion {
		return fmt.Errorf("cache version mismatch: %s != %s", manifest.Version, cacheKeyVersion)
	}
	if manifest.Entries == nil {
		manifest.Entries = make(map[string]*CacheEntry)
	}
	cs.manifest = &manifest
	return nil
}
