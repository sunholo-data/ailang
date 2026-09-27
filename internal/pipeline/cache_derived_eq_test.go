package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A dependency served from the compile cache is never elaborated, so the
// `deriving (Eq)` instances it declares were lost: an importer compiled in the
// same run failed with "No instance for Eq[Color]" (every release up to
// v0.44.1). The interface now carries DerivedEq, re-registered on a cache hit
// and folded into the digest importers' cache keys depend on.

func writeDerivedEqSources(t *testing.T, deriving bool, entries map[string]string) {
	t.Helper()
	decl := "module dep\nexport type Color = Red | Green"
	if deriving {
		decl += " deriving (Eq)"
	}
	if err := os.WriteFile("dep.ail", []byte(decl+"\n"), 0o644); err != nil {
		t.Fatalf("write dependency: %v", err)
	}
	for name, body := range entries {
		src := "module " + name + "\nimport dep (Color, Red, Green)\nexport pure func main() -> bool = " + body + "\n"
		if err := os.WriteFile(name+".ail", []byte(src), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}

func checkDerivedEqEntry(t *testing.T, entry string) (Result, error) {
	t.Helper()
	return runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: entry + ".ail"}, productionCacheDependencies())
}

func TestCacheDerivedEq_SurvivesCachedDependency(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("AILANG_CACHE_DIR", filepath.Join(root, "cache"))
	writeDerivedEqSources(t, true, map[string]string{"warm": "Red == Green", "main": "Red == Red"})

	if _, err := checkDerivedEqEntry(t, "warm"); err != nil {
		t.Fatalf("cold compile: %v", err)
	}

	// dep now comes from the cache; main is compiled fresh against it.
	result, err := checkDerivedEqEntry(t, "main")
	if err != nil {
		t.Fatalf("importer compiled against a cached dependency: %v", err)
	}
	if got := executePipelineMain(t, result); got != "true" {
		t.Fatalf("main = %s, want true", got)
	}
}

func TestCacheDerivedEq_DroppingDerivingInvalidatesImporter(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("AILANG_CACHE_DIR", filepath.Join(root, "cache"))
	writeDerivedEqSources(t, true, map[string]string{"main": "Red == Red"})
	if _, err := checkDerivedEqEntry(t, "main"); err != nil {
		t.Fatalf("cold compile: %v", err)
	}

	// Same importer source, dependency no longer derives Eq: the cached
	// importer must not be served.
	writeDerivedEqSources(t, false, map[string]string{"main": "Red == Red"})
	_, err := checkDerivedEqEntry(t, "main")
	if err == nil {
		t.Fatal("importer using == on a type that no longer derives Eq was served from the cache")
	}
	if !strings.Contains(err.Error(), "Eq[Color]") {
		t.Fatalf("want a missing Eq[Color] instance, got: %v", err)
	}
}
