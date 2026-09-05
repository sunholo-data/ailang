package pipeline

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

func TestCacheArtifacts_Migration(t *testing.T) {
	if cacheKeyVersion != "v4" {
		t.Fatalf("cache key version = %q, want v4 migration boundary", cacheKeyVersion)
	}

	t.Run("v3_manifest_forces_cold_v4_publication", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		t.Setenv("AILANG_CACHE_DIR", "")
		writeCachePipelineSource(t, 99)
		cfg := Config{Mode: ModeCheck}
		if _, err := runModuleWithCacheDependencies(t.Context(), cfg, Source{Filename: "answer.ail"}, productionCacheDependencies()); err != nil {
			t.Fatalf("seed cache: %v", err)
		}

		manifestPath := filepath.Join(root, ".ailang", "cache", "compile", "manifest.json")
		manifest := readCacheManifest(t, manifestPath)
		manifest.Version = "v3"
		writeCacheManifest(t, manifestPath, manifest)

		var warnings bytes.Buffer
		result, err := runModuleWithCacheDependencies(t.Context(), cfg, Source{Filename: "answer.ail"}, cacheDependencies{newStore: NewCacheStore, stderr: &warnings})
		if err != nil || result.Interface == nil || result.Interface.Exports["main"] == nil {
			t.Fatalf("v3 migration did not compile current source: iface=%v err=%v", result.Interface, err)
		}
		migrated := readCacheManifest(t, manifestPath)
		if migrated.Version != "v4" {
			t.Fatalf("migrated manifest version = %q, want v4", migrated.Version)
		}
		stamp := readArtifactStamp(t, filepath.Join(root, ".ailang", "cache", "compile", "modules", "answer", artifactStampName))
		if stamp.Version != "v4" || stamp.ModuleID != "answer" {
			t.Fatalf("migrated stamp = %#v", stamp)
		}
	})

	t.Run("current_manifest_with_legacy_unstamped_blobs_misses", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		t.Setenv("AILANG_CACHE_DIR", "")
		writeCachePipelineSource(t, 99)
		cfg := Config{Mode: ModeCheck}
		if _, err := runModuleWithCacheDependencies(t.Context(), cfg, Source{Filename: "answer.ail"}, productionCacheDependencies()); err != nil {
			t.Fatalf("seed cache: %v", err)
		}
		stampPath := filepath.Join(root, ".ailang", "cache", "compile", "modules", "answer", artifactStampName)
		mustRemove(t, stampPath)

		var warnings bytes.Buffer
		result, err := runModuleWithCacheDependencies(t.Context(), cfg, Source{Filename: "answer.ail"}, cacheDependencies{newStore: NewCacheStore, stderr: &warnings})
		if err != nil || result.Interface == nil || result.Interface.Exports["main"] == nil {
			t.Fatalf("unstamped legacy cache did not compile current source: iface=%v err=%v", result.Interface, err)
		}
		if !strings.Contains(warnings.String(), "CACHE_INVALID module=answer") {
			t.Fatalf("legacy miss was silent: %q", warnings.String())
		}
		if stamp := readArtifactStamp(t, stampPath); stamp.Version != "v4" {
			t.Fatalf("repaired stamp version = %q, want v4", stamp.Version)
		}
	})
}

func TestCachePipeline_WriteFailure(t *testing.T) {
	t.Run("artifact_failure_keeps_fresh_result_and_manifest_unpublished", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		t.Setenv("AILANG_CACHE_DIR", "")
		writeCachePipelineSource(t, 42)
		var warnings bytes.Buffer
		var failedStore *CacheStore
		deps := cacheDependencies{stderr: &warnings, newStore: func(projectDir string) (*CacheStore, error) {
			store, err := NewCacheStore(projectDir)
			if err != nil {
				return nil, err
			}
			failedStore = store
			write := store.artifactIO.writeFile
			store.artifactIO.writeFile = func(path string, data []byte, mode os.FileMode) error {
				if filepath.Base(filepath.Dir(path)) == "answer" && filepath.Base(path) == artifactCoreName {
					return os.ErrPermission
				}
				return write(path, data, mode)
			}
			return store, nil
		}}

		result, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: "answer.ail"}, deps)
		if err != nil || result.Interface == nil || result.Interface.Exports["main"] == nil {
			t.Fatalf("optional artifact failure became fatal: iface=%v err=%v", result.Interface, err)
		}
		if failedStore == nil {
			t.Fatal("cache factory was not invoked")
		}
		if _, ok := failedStore.Lookup("answer", readCacheKeyIfPresent(t, failedStore, "answer")); ok {
			t.Fatal("failed artifact publication authorized answer in the manifest")
		}
		warning := warnings.String()
		if !strings.Contains(warning, "CACHE_WRITE_FAILED module=answer stage=publication") || !strings.Contains(warning, "using fresh compilation") {
			t.Fatalf("artifact failure diagnostic = %q", warning)
		}

		if _, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: "answer.ail"}, productionCacheDependencies()); err != nil {
			t.Fatalf("restored publication: %v", err)
		}
		decoded := 0
		hitDeps := cacheDependencies{stderr: io.Discard, newStore: func(projectDir string) (*CacheStore, error) {
			store, openErr := NewCacheStore(projectDir)
			if openErr == nil {
				decode := store.artifactCodec.decodeCore
				store.artifactCodec.decodeCore = func(data []byte) (*core.Program, error) {
					decoded++
					return decode(data)
				}
			}
			return store, openErr
		}}
		if _, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: "answer.ail"}, hitDeps); err != nil {
			t.Fatalf("restored warm hit: %v", err)
		}
		if decoded == 0 {
			t.Fatal("restored cache was not used as a verified warm hit")
		}
	})

	t.Run("initialization_failure_is_visible_and_nonfatal", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		writeCachePipelineSource(t, 42)
		var warnings bytes.Buffer
		deps := cacheDependencies{
			stderr: &warnings,
			newStore: func(string) (*CacheStore, error) {
				return nil, errors.New("injected initialization failure")
			},
		}
		result, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: "answer.ail"}, deps)
		if err != nil || result.Interface == nil || result.Interface.Exports["main"] == nil {
			t.Fatalf("initialization failure became fatal: iface=%v err=%v", result.Interface, err)
		}
		if warning := warnings.String(); !strings.Contains(warning, "CACHE_WRITE_FAILED stage=initialization") {
			t.Fatalf("initialization failure diagnostic = %q", warning)
		}
	})

	t.Run("manifest_save_failure_is_visible_and_nonfatal", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		t.Setenv("AILANG_CACHE_DIR", "")
		writeCachePipelineSource(t, 42)
		var warnings bytes.Buffer
		deps := cacheDependencies{stderr: &warnings, newStore: func(projectDir string) (*CacheStore, error) {
			store, err := NewCacheStore(projectDir)
			if err == nil {
				store.writeManifest = func(string, []byte, os.FileMode) error { return os.ErrPermission }
			}
			return store, err
		}}
		result, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: "answer.ail"}, deps)
		if err != nil || result.Interface == nil || result.Interface.Exports["main"] == nil {
			t.Fatalf("manifest save failure became fatal: iface=%v err=%v", result.Interface, err)
		}
		if warning := warnings.String(); !strings.Contains(warning, "CACHE_WRITE_FAILED stage=manifest_save") {
			t.Fatalf("manifest save failure diagnostic = %q", warning)
		}
	})
}

func writeCachePipelineSource(t *testing.T, value int) {
	t.Helper()
	data := []byte("module answer\nexport pure func main() -> int = " + strconv.Itoa(value) + "\n")
	if err := os.WriteFile("answer.ail", data, 0o644); err != nil {
		t.Fatalf("write pipeline source: %v", err)
	}
}

func readCacheManifest(t *testing.T, path string) CacheManifest {
	t.Helper()
	var manifest CacheManifest
	if err := json.Unmarshal(mustRead(t, path), &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	return manifest
}

func writeCacheManifest(t *testing.T, path string, manifest CacheManifest) {
	t.Helper()
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	mustWriteArtifactTest(t, path, data)
}

func readArtifactStamp(t *testing.T, path string) artifactStamp {
	t.Helper()
	var stamp artifactStamp
	if err := json.Unmarshal(mustRead(t, path), &stamp); err != nil {
		t.Fatalf("unmarshal artifact stamp: %v", err)
	}
	return stamp
}

func readCacheKeyIfPresent(t *testing.T, store *CacheStore, moduleID string) string {
	t.Helper()
	entry, ok := store.manifest.Entries[moduleID]
	if !ok {
		return ""
	}
	return entry.CacheKey
}

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
