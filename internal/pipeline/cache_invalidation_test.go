package pipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestCacheKey_InvalidatesOnSourceEdit is a regression test for the cache
// poisoning bug discovered while building M-LAT-BUDGET workloads (April 2026).
//
// The bug: pipeline_module.go fed os.ReadFile(mod.Path) into the cache key,
// but mod.Path holds the *canonical module ID* ("benchmarks/workloads/warm_eval"),
// not the on-disk file path. ReadFile silently failed and returned an empty
// string, so every module that shared the same imports collided on the same
// cache key. After M-INCREMENTAL-TYPECHECK wired cache hits to skip compilation
// (commit 4f91d27e, 2026-04-10), this meant edits to .ail files were silently
// ignored — the runtime kept executing the previously-cached version.
//
// The unit tests in cache_key_test.go and cache_store_test.go BOTH passed
// throughout: cache_key_test verifies that ModuleCacheKey produces different
// hashes for different source strings, and cache_store_test verifies that the
// store round-trips correctly. The bug lived at the seam — the integration
// point that decides what string to feed into ModuleCacheKey. This test
// closes that gap by exercising the full pipeline twice with a source edit
// in between and asserting that the on-disk manifest records two different
// cache keys.
func TestCacheKey_InvalidatesOnSourceEdit(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ailang-cache-invalidation-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	modulePath := filepath.Join(tempDir, "answer.ail")

	// Self-contained module — no imports — so any cache key change MUST come
	// from sourceContent. If sourceContent is empty (the bug), both versions
	// hash to the same key.
	sourceV1 := `module answer

export pure func main() -> int = 42
`
	sourceV2 := `module answer

export pure func main() -> int = 99
`

	if err := os.WriteFile(modulePath, []byte(sourceV1), 0644); err != nil {
		t.Fatalf("write v1 source: %v", err)
	}

	src := Source{Filename: "answer.ail"}
	cfg := Config{Mode: ModeCheck}

	if _, err := Run(cfg, src); err != nil {
		t.Fatalf("first compile failed: %v", err)
	}
	keyV1 := readCacheKey(t, tempDir, "answer")
	if keyV1 == "" {
		t.Fatal("first compile produced no cache entry — pipeline cache wiring is broken")
	}

	// Mutate the source. The literal change must alter the cache key.
	if err := os.WriteFile(modulePath, []byte(sourceV2), 0644); err != nil {
		t.Fatalf("write v2 source: %v", err)
	}

	if _, err := Run(cfg, src); err != nil {
		t.Fatalf("second compile failed: %v", err)
	}
	keyV2 := readCacheKey(t, tempDir, "answer")
	if keyV2 == "" {
		t.Fatal("second compile produced no cache entry")
	}

	if keyV1 == keyV2 {
		t.Fatalf("cache poisoning regression: source edit did not change cache key.\n"+
			"  Both versions hashed to %s.\n"+
			"  This means ModuleCacheKey received the same sourceContent for both edits, "+
			"which is the bug fixed in pipeline_module.go (mod.Path → mod.File.Path).\n"+
			"  See cache_invalidation_test.go header for full incident notes.", keyV1)
	}
}

// readCacheKey reads the on-disk cache manifest and returns the cache_key for
// the given moduleID, or "" if missing. The pipeline writes the manifest at
// <projectDir>/.ailang/cache/compile/manifest.json where projectDir =
// filepath.Dir(src.Filename) — same path the cache_store uses internally.
func readCacheKey(t *testing.T, projectDir, moduleID string) string {
	t.Helper()
	manifestPath := filepath.Join(projectDir, ".ailang", "cache", "compile", "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest %s: %v", manifestPath, err)
	}
	var m CacheManifest
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	entry, ok := m.Entries[moduleID]
	if !ok {
		return ""
	}
	return entry.CacheKey
}
