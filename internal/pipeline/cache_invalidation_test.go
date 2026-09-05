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
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/loader"
	ailruntime "github.com/sunholo-data/ailang/internal/runtime"
	"github.com/sunholo-data/ailang/internal/testutil"
	"github.com/sunholo-data/ailang/internal/version"
)

func TestCacheSource_ExactSnapshot(t *testing.T) {
	t.Run("disk changes after load do not change key", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		t.Setenv("AILANG_CACHE_DIR", filepath.Join(root, "cache"))
		source := "module answer\nexport pure func value() -> int = 7\n"
		if err := os.WriteFile("answer.ail", []byte(source), 0o644); err != nil {
			t.Fatalf("write source: %v", err)
		}
		deps := productionCacheDependencies()
		afterLoad := func(modules map[string]*loader.LoadedModule) {
			if err := os.Remove("answer.ail"); err != nil {
				t.Fatalf("remove source after load: %v", err)
			}
		}
		if _, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: "answer.ail"}, deps, afterLoad); err != nil {
			t.Fatalf("compile retained snapshot: %v", err)
		}
		manifest := readCacheManifest(t, filepath.Join(root, "cache", "compile", "manifest.json"))
		entry := manifest.Entries["answer"]
		if entry == nil {
			t.Fatal("retained disk snapshot produced no cache entry")
		}
		if want := ModuleCacheKey(version.Commit, source, nil); entry.CacheKey != want {
			t.Fatalf("cache key = %q, want retained-source key %q", entry.CacheKey, want)
		}
	})

	t.Run("nil bypasses while known empty remains cacheable", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		cacheRoot := filepath.Join(root, "cache")
		t.Setenv("AILANG_CACHE_DIR", cacheRoot)
		writeCachePipelineSource(t, 42)
		if _, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: "answer.ail"}, productionCacheDependencies()); err != nil {
			t.Fatalf("seed cache: %v", err)
		}
		manifestPath := filepath.Join(cacheRoot, "compile", "manifest.json")
		seed := readCacheManifest(t, manifestPath).Entries["answer"]
		if seed == nil {
			t.Fatal("seed cache has no answer entry")
		}

		var warnings bytes.Buffer
		reads, writes := 0, 0
		deps := cacheDependencies{
			stderr: &warnings,
			newStore: func(projectDir string) (*CacheStore, error) {
				store, err := NewCacheStore(projectDir)
				if err != nil {
					return nil, err
				}
				open := store.artifactIO.open
				store.artifactIO.open = func(path string) (artifactReadFile, error) {
					if filepath.Base(filepath.Dir(path)) == "answer" {
						reads++
					}
					return open(path)
				}
				write := store.artifactIO.writeFile
				store.artifactIO.writeFile = func(path string, data []byte, mode os.FileMode) error {
					if filepath.Base(filepath.Dir(path)) == "answer" {
						writes++
					}
					return write(path, data, mode)
				}
				return store, nil
			},
		}
		nilSource := func(modules map[string]*loader.LoadedModule) {
			modules["answer"].SourceContent = nil
		}
		if _, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: "answer.ail"}, deps, nilSource); err != nil {
			t.Fatalf("compile unavailable snapshot: %v", err)
		}
		if reads != 0 || writes != 0 {
			t.Fatalf("nil snapshot touched cache artifacts: reads=%d writes=%d", reads, writes)
		}
		if warning := warnings.String(); !strings.Contains(warning, "CACHE_SOURCE_UNAVAILABLE module=answer") {
			t.Fatalf("nil snapshot diagnostic = %q", warning)
		}
		// The design says a module without a snapshot bypasses BOTH lookup and
		// publication -- it must never ATTEMPT to publish, not merely fail to.
		// Publication with the zero cache key is separately rejected one layer
		// down by StoreArtifacts, which would emit a write diagnostic here; the
		// bypass diagnostic must therefore be the only thing on stderr.
		if warning := warnings.String(); strings.Contains(warning, "CACHE_WRITE_FAILED") {
			t.Fatalf("nil snapshot attempted publication: stderr = %q", warning)
		}
		afterNil := readCacheManifest(t, manifestPath).Entries["answer"]
		if afterNil == nil || afterNil.CacheKey != seed.CacheKey || !afterNil.Timestamp.Equal(seed.Timestamp) {
			t.Fatalf("nil snapshot changed manifest entry: before=%#v after=%#v", seed, afterNil)
		}

		emptyRoot := t.TempDir()
		t.Chdir(emptyRoot)
		t.Setenv("AILANG_CACHE_DIR", filepath.Join(emptyRoot, "cache"))
		if err := os.WriteFile("answer.ail", []byte("module answer\nexport pure func value() -> int = 42\n"), 0o644); err != nil {
			t.Fatalf("write known-empty fixture: %v", err)
		}
		warnings.Reset()
		empty := ""
		emptyDeps := cacheDependencies{
			newStore: NewCacheStore,
			stderr:   &warnings,
		}
		emptySource := func(modules map[string]*loader.LoadedModule) {
			modules["answer"].SourceContent = &empty
		}
		if _, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: "answer.ail"}, emptyDeps, emptySource); err != nil {
			t.Fatalf("compile known-empty snapshot: %v", err)
		}
		emptyManifest := readCacheManifest(t, filepath.Join(emptyRoot, "cache", "compile", "manifest.json"))
		emptyEntry := emptyManifest.Entries["answer"]
		if emptyEntry == nil || emptyEntry.CacheKey != ModuleCacheKey(version.Commit, "", nil) {
			t.Fatalf("known-empty snapshot entry = %#v", emptyEntry)
		}
		if strings.Contains(warnings.String(), "CACHE_SOURCE_UNAVAILABLE") {
			t.Fatalf("known-empty snapshot treated as unavailable: %q", warnings.String())
		}
	})
}

func TestCachePipeline_EmbeddedKeys(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	testutil.SetHomeDir(t, filepath.Join(root, "home"))
	t.Setenv("AILANG_STDLIB_PATH", filepath.Join(root, "missing-stdlib"))
	t.Setenv("AILANG_CACHE_DIR", filepath.Join(root, "cache"))
	if err := os.WriteFile("main.ail", []byte("module main\nexport pure func main() -> int = 1\n"), 0o644); err != nil {
		t.Fatalf("write entry source: %v", err)
	}

	var loaded map[string]*loader.LoadedModule
	deps := productionCacheDependencies()
	afterLoad := func(modules map[string]*loader.LoadedModule) { loaded = modules }
	result, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: "main.ail"}, deps, afterLoad)
	if err != nil {
		t.Fatalf("compile embedded stdlib: %v", err)
	}
	manifest := readCacheManifest(t, filepath.Join(root, "cache", "compile", "manifest.json"))
	for _, moduleID := range []string{"std/option", "std/result"} {
		mod := loaded[moduleID]
		if mod == nil || mod.File == nil {
			t.Fatalf("loaded module %s = %#v", moduleID, mod)
		}
		wantPath := "<embedded>/" + moduleID + ".ail"
		if got := filepath.ToSlash(mod.File.Path); got != wantPath {
			t.Fatalf("%s source path = %q, want %q", moduleID, got, wantPath)
		}
		if mod.SourceContent == nil || *mod.SourceContent == "" {
			t.Fatalf("%s retained empty/unavailable embedded content", moduleID)
		}
		if len(mod.Imports) != 0 {
			t.Fatalf("%s unexpected dependencies: %v", moduleID, mod.Imports)
		}
		entry := manifest.Entries[moduleID]
		if entry == nil {
			t.Fatalf("manifest has no %s entry", moduleID)
		}
		want := ModuleCacheKey(version.Commit, *mod.SourceContent, map[string]string{})
		if entry.CacheKey != want {
			t.Fatalf("%s key = %q, want %q", moduleID, entry.CacheKey, want)
		}
		if empty := ModuleCacheKey(version.Commit, "", map[string]string{}); entry.CacheKey == empty {
			t.Fatalf("%s embedded key equals empty-source key %q", moduleID, empty)
		}
		if result.Modules[moduleID].SourceContent != nil {
			t.Fatalf("runtime result copied %s source snapshot", moduleID)
		}
	}
}

func TestCachePipeline_SourceEditBehavior(t *testing.T) {
	t.Run("cache edit and verified warm hit", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		t.Setenv("AILANG_CACHE_DIR", filepath.Join(root, "cache"))
		writeCacheBehaviorSources(t, 2)
		cfg := Config{Mode: ModeCheck}
		first, err := runModuleWithCacheDependencies(t.Context(), cfg, Source{Filename: "main.ail"}, productionCacheDependencies())
		if err != nil {
			t.Fatalf("compile dependency v1: %v", err)
		}
		if got := executePipelineMain(t, first); got != "3" {
			t.Fatalf("v1 output = %s, want 3", got)
		}

		writeCacheBehaviorSources(t, 40)
		second, err := runModuleWithCacheDependencies(t.Context(), cfg, Source{Filename: "main.ail"}, productionCacheDependencies())
		if err != nil {
			t.Fatalf("compile dependency v2: %v", err)
		}
		if second.Modules["dep"] == nil || second.Modules["dep"].Core == nil {
			t.Fatal("edited dependency has no updated Core")
		}
		if got := executePipelineMain(t, second); got != "41" {
			t.Fatalf("v2 output = %s, want 41", got)
		}

		depCoreReads, warmEncodes := 0, 0
		warmDeps := cacheDependencies{stderr: io.Discard, newStore: func(projectDir string) (*CacheStore, error) {
			store, openErr := NewCacheStore(projectDir)
			if openErr == nil {
				open := store.artifactIO.open
				store.artifactIO.open = func(path string) (artifactReadFile, error) {
					if filepath.Base(filepath.Dir(path)) == "dep" && filepath.Base(path) == artifactCoreName {
						depCoreReads++
					}
					return open(path)
				}
				encode := store.artifactCodec.encodeCore
				store.artifactCodec.encodeCore = func(program *core.Program) ([]byte, error) {
					warmEncodes++
					return encode(program)
				}
			}
			return store, openErr
		}}
		warm, err := runModuleWithCacheDependencies(t.Context(), cfg, Source{Filename: "main.ail"}, warmDeps)
		if err != nil {
			t.Fatalf("verified warm compile: %v", err)
		}
		if depCoreReads == 0 {
			t.Fatal("edited dependency was not loaded through a verified warm hit")
		}
		if warmEncodes != 0 {
			t.Fatalf("verified warm run recompiled %d modules", warmEncodes)
		}
		if got := executePipelineMain(t, warm); got != "41" {
			t.Fatalf("warm output = %s, want 41", got)
		}
	})

	t.Run("NoCache parity without persistence", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		cacheRoot := filepath.Join(root, "cache")
		t.Setenv("AILANG_CACHE_DIR", cacheRoot)
		cfg := Config{Mode: ModeCheck, NoCache: true}
		writeCacheBehaviorSources(t, 2)
		first, err := runModuleWithCacheDependencies(t.Context(), cfg, Source{Filename: "main.ail"}, productionCacheDependencies())
		if err != nil {
			t.Fatalf("NoCache compile v1: %v", err)
		}
		if got := executePipelineMain(t, first); got != "3" {
			t.Fatalf("NoCache v1 output = %s, want 3", got)
		}
		writeCacheBehaviorSources(t, 40)
		second, err := runModuleWithCacheDependencies(t.Context(), cfg, Source{Filename: "main.ail"}, productionCacheDependencies())
		if err != nil {
			t.Fatalf("NoCache compile v2: %v", err)
		}
		if got := executePipelineMain(t, second); got != "41" {
			t.Fatalf("NoCache v2 output = %s, want 41", got)
		}
		if _, err := os.Stat(filepath.Join(cacheRoot, "compile")); !os.IsNotExist(err) {
			t.Fatalf("NoCache persisted compile cache: err=%v", err)
		}
	})
}

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

func writeCacheBehaviorSources(t *testing.T, depValue int) {
	t.Helper()
	dep := []byte("module dep\nexport pure func value() -> int = " + strconv.Itoa(depValue) + "\n")
	main := []byte("module main\nimport dep (value)\nexport pure func main() -> int = value() + 1\n")
	if err := os.WriteFile("dep.ail", dep, 0o644); err != nil {
		t.Fatalf("write dependency: %v", err)
	}
	if err := os.WriteFile("main.ail", main, 0o644); err != nil {
		t.Fatalf("write entry: %v", err)
	}
}

func executePipelineMain(t *testing.T, result Result) string {
	t.Helper()
	rt := ailruntime.NewModuleRuntime(".")
	for moduleID, loaded := range result.Modules {
		if loaded.SourceContent != nil {
			t.Fatalf("runtime module %s retained source snapshot", moduleID)
		}
		rt.PreloadModule(moduleID, loaded)
	}
	inst, err := rt.LoadAndEvaluate("main")
	if err != nil {
		t.Fatalf("load runtime entry: %v", err)
	}
	value, err := ailruntime.CallEntrypoint(rt, inst, "main", []eval.Value{})
	if err != nil {
		t.Fatalf("call runtime entry: %v", err)
	}
	return value.String()
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
// The bug: pipeline_module.go reread a path opportunistically at key time and
// silently used an empty string when that second read failed. A canonical
// module ID was initially used as the path; even after switching to File.Path,
// the second read could describe bytes different from the AST. Modules sharing
// imports could therefore collide or execute stale artifacts. The loader-owned
// snapshot now binds the cache key to the exact text handed to the lexer.
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
			"  This means ModuleCacheKey received the same source snapshot for both edits, "+
			"which violates the loader-owned source identity contract.\n"+
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
