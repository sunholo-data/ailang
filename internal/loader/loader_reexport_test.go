package loader

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

// Unit tests for the load-time re-export rewrite (M-PKG-PARALLEL-AGENTS C1).
// The end-to-end identity tests live in internal/runner; these pin the
// rewrite itself on hand-built module graphs.

func imp(path string, syms ...string) *ast.ImportDecl {
	d := &ast.ImportDecl{Path: path, Symbols: syms, SymbolAliases: map[string]string{}}
	for i := range syms {
		d.SymbolPos = append(d.SymbolPos, ast.Pos{Line: 1, Column: 10 + i})
	}
	return d
}

func reexportImp(path, movedTo string, syms ...string) *ast.ImportDecl {
	d := imp(path, syms...)
	d.Reexport, d.MovedTo = true, movedTo
	return d
}

// facadeModule is a loaded module whose Reexports come from its own imports,
// as Load builds them.
func facadeModule(path string, imports ...*ast.ImportDecl) *LoadedModule {
	f := &ast.File{Imports: imports}
	return &LoadedModule{Path: path, File: f, Reexports: buildReexports(f)}
}

func TestBuildReexports(t *testing.T) {
	if got := buildReexports(nil); got != nil {
		t.Fatalf("nil file: got %v", got)
	}
	if got := buildReexports(&ast.File{Imports: []*ast.ImportDecl{imp("lib/a", "x"), nil}}); got != nil {
		t.Fatalf("no export import: got %v", got)
	}
	renamed := reexportImp("lib/color", "new/color", "Color", "Red")
	renamed.SymbolAliases["Red"] = "R"
	got := buildReexports(&ast.File{Imports: []*ast.ImportDecl{imp("lib/other", "y"), renamed}})
	want := map[string]Reexport{
		"Color": {From: "lib/color", Orig: "Color", MovedTo: "new/color"},
		"R":     {From: "lib/color", Orig: "Red", MovedTo: "new/color"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestResolveReexports_SplitsMixedImport(t *testing.T) {
	user := imp("lib/facade", "own", "Color", "Red")
	user.SymbolAliases["Red"] = "R"
	modules := map[string]*LoadedModule{
		"lib/color":  {Path: "lib/color", File: &ast.File{}},
		"lib/facade": facadeModule("lib/facade", reexportImp("lib/color", "new/color", "Color", "Red")),
		"app/main":   {Path: "app/main", File: &ast.File{Imports: []*ast.ImportDecl{user, imp("lib/color", "Color")}}},
		"app/empty":  {Path: "app/empty"},
	}
	if err := resolveReexports(modules); err != nil {
		t.Fatal(err)
	}
	got := modules["app/main"].File.Imports
	if len(got) != 3 {
		t.Fatalf("want facade remainder + origin + untouched import, got %d: %v", len(got), got)
	}
	kept, origin, untouched := got[0], got[1], got[2]
	if kept.Path != "lib/facade" || !reflect.DeepEqual(kept.Symbols, []string{"own"}) || len(kept.SymbolPos) != 1 {
		t.Errorf("kept import: %+v", kept)
	}
	if origin.Path != "lib/color" || !reflect.DeepEqual(origin.Symbols, []string{"Color", "Red"}) {
		t.Errorf("origin import: %+v", origin)
	}
	if origin.SymbolAliases["Red"] != "R" || len(origin.SymbolPos) != 2 {
		t.Errorf("alias or positions lost: %+v", origin)
	}
	if origin.ViaModule != "lib/facade" || origin.ViaMovedTo != "new/color" {
		t.Errorf("facade not recorded: via=%q movedTo=%q", origin.ViaModule, origin.ViaMovedTo)
	}
	if untouched.Path != "lib/color" || untouched.ViaModule != "" {
		t.Errorf("a direct import must stay as written: %+v", untouched)
	}
}

func TestResolveReexports_DropsEmptyRemainderButKeepsModuleAlias(t *testing.T) {
	only := imp("lib/facade", "x")
	aliased := imp("lib/facade", "x")
	aliased.ModuleAlias = "F"
	modules := map[string]*LoadedModule{
		"lib/src":    {Path: "lib/src", File: &ast.File{}},
		"lib/facade": facadeModule("lib/facade", reexportImp("lib/src", "", "x")),
		"app/a":      {Path: "app/a", File: &ast.File{Imports: []*ast.ImportDecl{only}}},
		"app/b":      {Path: "app/b", File: &ast.File{Imports: []*ast.ImportDecl{aliased}}},
	}
	if err := resolveReexports(modules); err != nil {
		t.Fatal(err)
	}
	if a := modules["app/a"].File.Imports; len(a) != 1 || a[0].Path != "lib/src" {
		t.Errorf("app/a: want only the origin import, got %+v", a)
	}
	b := modules["app/b"].File.Imports
	if len(b) != 2 || b[0].Path != "lib/facade" || b[0].ModuleAlias != "F" || len(b[0].Symbols) != 0 {
		t.Errorf("app/b: the aliased facade import must survive with no symbols, got %+v", b)
	}
}

func TestResolveReexports_FollowsChainAndKeepsFirstMovedTo(t *testing.T) {
	modules := map[string]*LoadedModule{
		"lib/def": {Path: "lib/def", File: &ast.File{}},
		"lib/f1":  facadeModule("lib/f1", reexportImp("lib/def", "inner/path", "v")),
		"lib/f2":  facadeModule("lib/f2", reexportImp("lib/f1", "outer/path", "v")),
		"app/m":   {Path: "app/m", File: &ast.File{Imports: []*ast.ImportDecl{imp("lib/f2", "v")}}},
	}
	if err := resolveReexports(modules); err != nil {
		t.Fatal(err)
	}
	got := modules["app/m"].File.Imports
	if len(got) != 1 || got[0].Path != "lib/def" || got[0].ViaModule != "lib/f2" || got[0].ViaMovedTo != "outer/path" {
		t.Fatalf("chain: %+v", got)
	}
	// The facades' own lines are rewritten too, and keep their export marker.
	f2 := modules["lib/f2"].File.Imports
	if len(f2) != 1 || f2[0].Path != "lib/def" || !f2[0].Reexport || f2[0].MovedTo != "outer/path" {
		t.Fatalf("facade line: %+v", f2)
	}
}

func TestResolveReexports_CycleIsMOD016(t *testing.T) {
	modules := map[string]*LoadedModule{
		"lib/a": facadeModule("lib/a", reexportImp("lib/b", "", "x")),
		"lib/b": facadeModule("lib/b", reexportImp("lib/a", "", "x")),
	}
	err := resolveReexports(modules)
	if err == nil || !strings.Contains(err.Error(), "MOD016") || !strings.Contains(err.Error(), "re-export cycle") {
		t.Fatalf("want MOD016 re-export cycle, got %v", err)
	}
}

func TestRewrittenImport_PackageOrigin(t *testing.T) {
	d := rewrittenImport(imp("lib/facade", "x"), "pkg/acme/core/types", "")
	if !d.IsPackage || d.PackageName != "acme/core" || d.ViaModule != "lib/facade" {
		t.Fatalf("package origin: %+v", d)
	}
	if d := rewrittenImport(imp("lib/facade", "x"), "lib/src", ""); d.IsPackage || d.PackageName != "" {
		t.Fatalf("local origin marked as package: %+v", d)
	}
}

func TestSplitReexportImport_Passthrough(t *testing.T) {
	modules := map[string]*LoadedModule{"lib/plain": {Path: "lib/plain", File: &ast.File{}}}
	for _, d := range []*ast.ImportDecl{nil, imp("lib/plain"), imp("lib/plain", "x"), imp("lib/missing", "x")} {
		got, err := splitReexportImport(modules, d)
		if err != nil || len(got) != 1 || got[0] != d {
			t.Errorf("%+v: want unchanged, got %+v, %v", d, got, err)
		}
	}
	// A facade import naming none of its re-exports is returned as is.
	modules["lib/facade"] = facadeModule("lib/facade", reexportImp("lib/plain", "", "x"))
	own := imp("lib/facade", "own")
	if got, _ := splitReexportImport(modules, own); len(got) != 1 || got[0] != own {
		t.Errorf("own names only: got %+v", got)
	}
}
