package link

import (
	"errors"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/loader"
)

func mod(path string, imports ...*ast.ImportDecl) *loader.LoadedModule {
	m := &loader.LoadedModule{Path: path, File: &ast.File{Imports: imports}}
	for _, imp := range imports {
		m.Imports = append(m.Imports, imp.Path)
	}
	return m
}

func TestTopoSort_DependencyOrder(t *testing.T) {
	ml := NewModuleLinker(nil)
	sorted, err := ml.TopoSortFromRoot("main", map[string]*loader.LoadedModule{
		"main": mod("main", &ast.ImportDecl{Path: "a"}),
		"a":    mod("a", &ast.ImportDecl{Path: "b"}),
		"b":    mod("b"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sorted) != 3 || sorted[0] != "b" || sorted[1] != "a" || sorted[2] != "main" {
		t.Fatalf("order = %v, want [b a main]", sorted)
	}
}

// LDR002 prints the cycle once, and names the spelling the author wrote for
// an edge the loader rewrote to the canonical identity (M1b review, #2224).
func TestTopoSort_CycleNamesSpellingOnce(t *testing.T) {
	ml := NewModuleLinker(nil)
	_, err := ml.TopoSortFromRoot("main", map[string]*loader.LoadedModule{
		"main":                  mod("main", &ast.ImportDecl{Path: "pkg/acme/shapes/cyc/a"}),
		"pkg/acme/shapes/cyc/a": mod("pkg/acme/shapes/cyc/a", &ast.ImportDecl{Path: "pkg/acme/shapes/cyc/b", WrittenPath: "shp/cyc/b"}),
		"pkg/acme/shapes/cyc/b": mod("pkg/acme/shapes/cyc/b", &ast.ImportDecl{Path: "pkg/acme/shapes/cyc/a"}),
	})
	var cyc *CycleError
	if !errors.As(err, &cyc) {
		t.Fatalf("want a CycleError, got %v", err)
	}
	want := "LDR002: dependency cycle detected: pkg/acme/shapes/cyc/a -> pkg/acme/shapes/cyc/b (imported as shp/cyc/b) -> pkg/acme/shapes/cyc/a"
	if err.Error() != want {
		t.Fatalf("cycle error\n got: %s\nwant: %s", err.Error(), want)
	}
	if got := cyc.GetSuggestion(); got == "" {
		t.Error("a two-module cycle should suggest extracting shared code")
	}
	if (&CycleError{Code: "LDR002", Cycle: []ModuleID{"x"}}).GetSuggestion() != "" {
		t.Error("a one-element cycle has no suggestion")
	}
}

// The first module of a cycle shows how the module that led into the cycle
// imported it.
func TestTopoSort_CycleFirstElementShowsItsSpelling(t *testing.T) {
	ml := NewModuleLinker(nil)
	_, err := ml.TopoSortFromRoot("main", map[string]*loader.LoadedModule{
		"main":                  mod("main", &ast.ImportDecl{Path: "pkg/acme/shapes/cyc/a", WrittenPath: "shp/cyc/a"}),
		"pkg/acme/shapes/cyc/a": mod("pkg/acme/shapes/cyc/a", &ast.ImportDecl{Path: "pkg/acme/shapes/cyc/b"}),
		"pkg/acme/shapes/cyc/b": mod("pkg/acme/shapes/cyc/b", &ast.ImportDecl{Path: "pkg/acme/shapes/cyc/a", WrittenPath: "shp/cyc/a"}),
	})
	want := "LDR002: dependency cycle detected: pkg/acme/shapes/cyc/a (imported as shp/cyc/a) -> pkg/acme/shapes/cyc/b -> pkg/acme/shapes/cyc/a (imported as shp/cyc/a)"
	if err == nil || err.Error() != want {
		t.Fatalf("cycle error\n got: %v\nwant: %s", err, want)
	}
}

func TestTopoSort_MissingModule(t *testing.T) {
	ml := NewModuleLinker(nil)
	if _, err := ml.TopoSortFromRoot("main", map[string]*loader.LoadedModule{
		"main": mod("main", &ast.ImportDecl{Path: "gone"}),
	}); err == nil {
		t.Fatal("an import of a module that was not loaded must fail")
	}
}
