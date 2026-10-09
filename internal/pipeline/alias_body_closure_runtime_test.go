package pipeline

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func writeClosureModules(t *testing.T, imports, expr string) {
	t.Helper()
	for name, source := range map[string]string{
		"a.ail": closureA, "b.ail": closureB,
		"main.ail": "module main\n" + imports + "\nexport pure func main() -> int { " + expr + " }",
	} {
		if err := os.WriteFile(name, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAliasBodyClosure_RuntimeMatrix(t *testing.T) {
	for _, imports := range []string{
		"import a (Box, mk, count, mkItem, countItem)\nimport b (Item, one)",
		"import b (one)\nimport a (Box, mk, count, mkItem, countItem)",
		"import a (Box, mk, count, mkItem, countItem)\nimport b (one)",
	} {
		for _, expr := range []string{"count(mk())", "countItem(mkItem())", "let b = mk(); count({b | items: [{x: 7, w: 8}]})"} {
			t.Run(imports+expr, func(t *testing.T) {
				t.Chdir(t.TempDir())
				writeClosureModules(t, imports, expr)
				result, err := Run(Config{Mode: ModeCheck, NoCache: true}, Source{Filename: "main.ail"})
				if err != nil {
					t.Fatal(err)
				}
				if got := executePipelineMain(t, result); got != "1" {
					t.Fatalf("got %s, want 1", got)
				}
			})
		}
	}
}

func TestAliasBodyClosure_CacheInvalidatedOnce(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("AILANG_CACHE_DIR", "")
	writeClosureModules(t, "import a (Box, mk, count)\nimport b (Item, one)", "count(mk())")
	cfg := Config{Mode: ModeCheck}
	source := Source{Filename: "main.ail"}
	if _, err := Run(cfg, source); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".ailang", "cache", "compile", "manifest.json")
	manifest := readCacheManifest(t, path)
	manifest.Version = "v5"
	writeCacheManifest(t, path, manifest)
	// Instrument real core encodes to distinguish cold publication from warm hits.
	var reads, encodes int
	deps := newWarmHitInstrumentedDeps(&reads, &encodes)
	cold, err := runModuleWithCacheDependencies(t.Context(), cfg, source, deps)
	if err != nil {
		t.Fatal(err)
	}
	if encodes != len(manifest.Entries) {
		t.Fatalf("v5 cache should recompile all %d modules (including implicit imports), got %d", len(manifest.Entries), encodes)
	}
	if readCacheManifest(t, path).Version != "v6" {
		t.Fatal("cache did not publish v6 manifest")
	}
	if got := executePipelineMain(t, cold); got != "1" {
		t.Fatal(got)
	}
	encodes = 0
	warm, err := runModuleWithCacheDependencies(t.Context(), cfg, source, deps)
	if err != nil {
		t.Fatal(err)
	}
	if encodes != 0 {
		t.Fatalf("unchanged v6 modules recompiled: %d", encodes)
	}
	fresh, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck, NoCache: true}, source, cacheDependencies{newStore: NewCacheStore, stderr: io.Discard})
	if err != nil {
		t.Fatal(err)
	}
	if executePipelineMain(t, warm) != executePipelineMain(t, fresh) {
		t.Fatal("cached/uncached runtime mismatch")
	}
}
