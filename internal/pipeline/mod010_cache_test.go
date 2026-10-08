package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/loader"
)

// TestMOD010_CacheHitDoesNotSkipValidation (#1572): a module compiled once
// under RelaxModules is cached; a later strict run on the unchanged file must
// still fail MOD010 rather than be served from the cache.
func TestMOD010_CacheHitDoesNotSkipValidation(t *testing.T) {
	fixture, err := filepath.Abs(filepath.Join("testdata", "mod010_cache"))
	if err != nil {
		t.Fatal(err)
	}
	if loader.IsTempPath(filepath.Join(fixture, "sub", "m")) {
		t.Skip("checkout lives under a temp directory, where MOD010 auto-relaxes")
	}
	t.Chdir(fixture)
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	src := Source{Filename: filepath.Join("sub", "m.ail")}
	deps := productionCacheDependencies()

	if _, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, src, deps); err == nil || !strings.Contains(err.Error(), "MOD010") {
		t.Fatalf("cold strict run: err = %v, want MOD010", err)
	}
	if _, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck, RelaxModules: true}, src, deps); err != nil {
		t.Fatalf("relaxed run: %v", err)
	}
	if entries, _ := os.ReadDir(os.Getenv("AILANG_CACHE_DIR")); len(entries) == 0 {
		t.Fatal("relaxed run populated no cache — the strict rerun below would not exercise a cache hit")
	}
	if _, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, src, deps); err == nil || !strings.Contains(err.Error(), "MOD010") {
		t.Fatalf("warm strict run: err = %v, want MOD010 (cache hit must not skip validation)", err)
	}
}
