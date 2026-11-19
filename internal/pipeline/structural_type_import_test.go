package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Map and Array are builtin structural types: in scope everywhere, declared by no
// module. std/set and std/option export their type names, and the prompt teaches
// `import std/set (Set)` beside `import std/map as M`, so models write
// `import std/map (Map)`. That was IMP010 until 2026-11-12 (4 of Gemini 4
// Hydrogen's 5 placement failures). Importing a structural type from the module
// that owns it is now a no-op (loader.getExport).
func writeStructuralModule(t *testing.T, name, src string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name+".ail")
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestStructuralTypeImport_FromOwningModule(t *testing.T) {
	p := writeStructuralModule(t, "imp", `module imp
import std/map as M
import std/map (Map)
import std/array as A
import std/array (Array)

export pure func count(m: Map[string, int], k: string) -> int =
  match M.lookup(m, k) { Some(n) => n, None => 0 }

export pure func firstOr(a: Array[int], d: int) -> int =
  if A.length(a) > 0 then A.get(a, 0) else d
`)
	result, err := Run(Config{Mode: ModeCheck, NoCache: true}, Source{Filename: p})
	if err != nil || len(result.Errors) > 0 {
		t.Fatalf("importing Map/Array from their own modules must check clean: %v %v", err, result.Errors)
	}
}

func TestStructuralTypeImport_FromWrongModuleStillIMP010(t *testing.T) {
	for _, c := range []struct{ name, imp, sym string }{
		{"wrongmap", "std/list", "Map"},
		{"wrongarr", "std/map", "Array"},
	} {
		p := writeStructuralModule(t, c.name, "module "+c.name+"\nimport "+c.imp+" ("+c.sym+")\nexport pure func one() -> int = 1\n")
		_, err := Run(Config{Mode: ModeCheck, NoCache: true}, Source{Filename: p})
		want := "symbol '" + c.sym + "' not exported by '" + c.imp + "'"
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("import %s (%s): want IMP010 %q, got %v", c.imp, c.sym, want, err)
		}
	}
}
