package loader

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

// entryMainFile builds an AST file with an exported zero-arg main and the given
// existing imports + type decls. It is the minimal shape isEntryModuleFile needs.
func entryMainFile(imports []*ast.ImportDecl, typeDecls []*ast.TypeDecl) *ast.File {
	decls := []ast.Node{
		&ast.FuncDecl{Name: "main", IsExport: true, Params: nil},
	}
	for _, td := range typeDecls {
		decls = append(decls, td)
	}
	return &ast.File{Imports: imports, Decls: decls}
}

func hasImport(file *ast.File, path string) bool {
	for _, imp := range file.Imports {
		if imp.Path == path {
			return true
		}
	}
	return false
}

func TestInjectPreludeImports_EntryGetsBothModules(t *testing.T) {
	file := entryMainFile(nil, nil)
	out := injectPreludeImports(file, "main", "", nil)

	for _, want := range []string{"std/option", "std/result"} {
		if !hasImport(file, want) {
			t.Errorf("entry module missing implicit import %q", want)
		}
		found := false
		for _, p := range out {
			if p == want {
				found = true
			}
		}
		if !found {
			t.Errorf("returned import-path slice missing %q", want)
		}
	}

	// Implicit imports must be selective (carry Some/None etc.) so the
	// constructor auto-import path fires — a bare whole-module import would
	// not bring constructors into scope.
	for _, imp := range file.Imports {
		if imp.Path == "std/option" && len(imp.Symbols) == 0 {
			t.Error("implicit std/option import must name its symbols (Option, Some, None)")
		}
	}
}

// #2371: a library module (no exported main) gets the prelude too.
func TestInjectPreludeImports_LibraryModuleGetsPrelude(t *testing.T) {
	helper := &ast.FuncDecl{Name: "helper", IsExport: true}
	file := &ast.File{Module: &ast.ModuleDecl{Path: "lib/x"}, Decls: []ast.Node{helper}, Funcs: []*ast.FuncDecl{helper}}
	out := injectPreludeImports(file, "lib/x", "f(o: Option[int]) = Ok(1)", nil)
	if !hasImport(file, "std/option") || !hasImport(file, "std/result") || len(out) != 2 {
		t.Errorf("library module must receive the implicit prelude imports, got %v", out)
	}
}

// A library module gets a prelude module only when it writes one of that
// module's names; one that writes none keeps its imports (and its cache key
// and program image) as before.
func TestInjectPreludeImports_LibraryPaysForUse(t *testing.T) {
	for _, c := range []struct {
		source string
		want   []string
	}{
		{"export pure func f() -> int = 1", nil},
		{"pure func isSomeThing(SomeType: int) -> int = 1 -- isSome, Okay, Errs", nil}, // not whole words
		{"pure func f(x: int) = Some(x)", []string{"std/option"}},
		{"pure func f(r) = match r { Err(e) => e, _ => 0 }", []string{"std/result"}},
		{"-- returns None or Ok\n// Result too\nexport pure func f() -> int = 1", nil},           // comments do not count
		{"export pure func f() -> string = \"Some Result: ok\"", nil},                            // nor a string's text
		{"export pure func f(i: int) -> string = \"x${show(Some(i))}\"", []string{"std/option"}}, // interpolated code does
		{"export pure func f(o: Option[int], r: Result[int, string]) -> int = 0", []string{"std/option", "std/result"}},
	} {
		helper := &ast.FuncDecl{Name: "f", IsExport: true}
		file := &ast.File{Module: &ast.ModuleDecl{Path: "lib/x"}, Decls: []ast.Node{helper}, Funcs: []*ast.FuncDecl{helper}}
		out := injectPreludeImports(file, "lib/x", c.source, nil)
		if strings.Join(out, ",") != strings.Join(c.want, ",") {
			t.Errorf("%q: injected %v, want %v", c.source, out, c.want)
		}
	}
	// An entry module always gets both, as before.
	file := entryMainFile(nil, nil)
	if out := injectPreludeImports(file, "main", "", nil); len(out) != 2 {
		t.Errorf("entry module: injected %v, want both", out)
	}
}

// The standard library is unchanged: std/option cannot import itself, and std
// modules import what they use.
func TestInjectPreludeImports_StdModuleUntouched(t *testing.T) {
	for _, id := range []string{"std/list", "std/option", "std/result"} {
		helper := &ast.FuncDecl{Name: "helper", IsExport: true}
		file := &ast.File{Module: &ast.ModuleDecl{Path: id}, Decls: []ast.Node{helper}, Funcs: []*ast.FuncDecl{helper}}
		if out := injectPreludeImports(file, id, "Option Some None Result Ok Err", nil); len(out) != 0 || len(file.Imports) != 0 {
			t.Errorf("%s must not receive implicit prelude imports, got %v", id, out)
		}
	}
}

// A file with a top-level expression keeps no prelude: with any import its
// expression would not be type-checked (#2372).
func TestInjectPreludeImports_TopLevelExpressionUntouched(t *testing.T) {
	expr := &ast.Identifier{Name: "x"}
	file := &ast.File{Module: &ast.ModuleDecl{Path: "probe"}, Decls: []ast.Node{expr}, Statements: []ast.Node{expr}}
	if out := injectPreludeImports(file, "probe", "Some(1)", nil); len(out) != 0 {
		t.Errorf("a top-level-expression file must not receive implicit prelude imports, got %v", out)
	}
}

func TestInjectPreludeImports_DedupExplicitImport(t *testing.T) {
	// User already imports std/option — must not be duplicated.
	file := entryMainFile([]*ast.ImportDecl{
		{Path: "std/option", Symbols: []string{"Some", "None"}},
	}, nil)
	injectPreludeImports(file, "main", "", nil)

	optCount := 0
	for _, imp := range file.Imports {
		if imp.Path == "std/option" {
			optCount++
		}
	}
	if optCount != 1 {
		t.Errorf("expected exactly 1 std/option import after dedup, got %d", optCount)
	}
	// std/result (not user-imported) should still be injected.
	if !hasImport(file, "std/result") {
		t.Error("std/result should still be injected when only std/option is explicit")
	}
}

func TestInjectPreludeImports_LocalTypeShadows(t *testing.T) {
	// User locally defines `type Option[a] = Some(a) | None` — the prelude
	// std/option must be shadowed (not injected) to avoid ambiguity.
	localOption := &ast.TypeDecl{
		Name:       "Option",
		TypeParams: []string{"a"},
		Definition: &ast.AlgebraicType{Constructors: []*ast.Constructor{
			{Name: "Some"}, {Name: "None"},
		}},
	}
	file := entryMainFile(nil, []*ast.TypeDecl{localOption})
	injectPreludeImports(file, "main", "", nil)

	if hasImport(file, "std/option") {
		t.Error("local type Option must shadow the prelude: std/option must NOT be injected")
	}
	// std/result is unaffected and should still be injected.
	if !hasImport(file, "std/result") {
		t.Error("std/result should still be injected when only Option is locally shadowed")
	}
}

func TestInjectPreludeImports_LocalResultDifferentCtorsShadows(t *testing.T) {
	// `type Result = Pending | Done` — same type name, different constructors.
	// Must shadow the prelude std/result cleanly.
	localResult := &ast.TypeDecl{
		Name: "Result",
		Definition: &ast.AlgebraicType{Constructors: []*ast.Constructor{
			{Name: "Pending"}, {Name: "Done"},
		}},
	}
	file := entryMainFile(nil, []*ast.TypeDecl{localResult})
	injectPreludeImports(file, "main", "", nil)

	if hasImport(file, "std/result") {
		t.Error("local type Result must shadow the prelude: std/result must NOT be injected")
	}
	if !hasImport(file, "std/option") {
		t.Error("std/option should still be injected when only Result is locally shadowed")
	}
}

func TestInjectPreludeImports_LocalCtorNameCollisionShadows(t *testing.T) {
	// A local type that (re)defines a constructor named Some (even under a
	// different type name) must shadow std/option to avoid a ctor clash.
	file := entryMainFile(nil, []*ast.TypeDecl{{
		Name: "MyOpt",
		Definition: &ast.AlgebraicType{Constructors: []*ast.Constructor{
			{Name: "Some"}, {Name: "Nothing"},
		}},
	}})
	injectPreludeImports(file, "main", "", nil)
	if hasImport(file, "std/option") {
		t.Error("local constructor Some must shadow std/option (ctor-name collision)")
	}
}
