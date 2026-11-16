package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRootModuleKey_IsTheEntryModulesKey: Result.RootModuleKey names the
// entry module's entry in Result.Modules, including under RelaxModules where
// a file declaring `module main` is keyed by something other than "main".
// Callers that must tell the entry module from its imports (the verifier's
// "imported contracts were not verified" note) compare against this key.
// Red mutation: leave RootModuleKey unset.
func TestRootModuleKey_IsTheEntryModulesKey(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "r")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "main.ail")
	src := "module main\n\nexport pure func f(x: int) -> int = x + 1\n"
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Run(Config{DryLink: true, RelaxModules: true}, Source{Code: src, Filename: file})
	if err != nil {
		t.Fatalf("pipeline: %v", err)
	}
	if res.RootModuleKey == "" {
		t.Fatal("RootModuleKey is empty")
	}
	root, ok := res.Modules[res.RootModuleKey]
	if !ok || root == nil || root.File == nil || root.File.Module == nil || root.File.Module.Path != "main" {
		t.Fatalf("RootModuleKey %q does not name the entry module in Modules (keys: %v)", res.RootModuleKey, moduleKeys(res.Modules))
	}
	if res.RootModuleKey == "main" {
		t.Fatalf("instrument: under RelaxModules the key was expected to differ from the declared name; got %q", res.RootModuleKey)
	}
}

func moduleKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
