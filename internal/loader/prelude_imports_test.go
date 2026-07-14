package loader

import (
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

func TestInjectEntryPreludeImports_EntryGetsBothModules(t *testing.T) {
	file := entryMainFile(nil, nil)
	out := injectEntryPreludeImports(file, nil)

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

func TestInjectEntryPreludeImports_LibraryModuleUntouched(t *testing.T) {
	// A file with no exported main is NOT an entry module.
	file := &ast.File{Decls: []ast.Node{
		&ast.FuncDecl{Name: "helper", IsExport: true},
	}}
	injectEntryPreludeImports(file, nil)
	if hasImport(file, "std/option") || hasImport(file, "std/result") {
		t.Error("library module must not receive implicit prelude imports")
	}
}

func TestInjectEntryPreludeImports_DedupExplicitImport(t *testing.T) {
	// User already imports std/option — must not be duplicated.
	file := entryMainFile([]*ast.ImportDecl{
		{Path: "std/option", Symbols: []string{"Some", "None"}},
	}, nil)
	injectEntryPreludeImports(file, nil)

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

func TestInjectEntryPreludeImports_LocalTypeShadows(t *testing.T) {
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
	injectEntryPreludeImports(file, nil)

	if hasImport(file, "std/option") {
		t.Error("local type Option must shadow the prelude: std/option must NOT be injected")
	}
	// std/result is unaffected and should still be injected.
	if !hasImport(file, "std/result") {
		t.Error("std/result should still be injected when only Option is locally shadowed")
	}
}

func TestInjectEntryPreludeImports_LocalResultDifferentCtorsShadows(t *testing.T) {
	// `type Result = Pending | Done` — same type name, different constructors.
	// Must shadow the prelude std/result cleanly.
	localResult := &ast.TypeDecl{
		Name: "Result",
		Definition: &ast.AlgebraicType{Constructors: []*ast.Constructor{
			{Name: "Pending"}, {Name: "Done"},
		}},
	}
	file := entryMainFile(nil, []*ast.TypeDecl{localResult})
	injectEntryPreludeImports(file, nil)

	if hasImport(file, "std/result") {
		t.Error("local type Result must shadow the prelude: std/result must NOT be injected")
	}
	if !hasImport(file, "std/option") {
		t.Error("std/option should still be injected when only Result is locally shadowed")
	}
}

func TestInjectEntryPreludeImports_LocalCtorNameCollisionShadows(t *testing.T) {
	// A local type that (re)defines a constructor named Some (even under a
	// different type name) must shadow std/option to avoid a ctor clash.
	file := entryMainFile(nil, []*ast.TypeDecl{{
		Name: "MyOpt",
		Definition: &ast.AlgebraicType{Constructors: []*ast.Constructor{
			{Name: "Some"}, {Name: "Nothing"},
		}},
	}})
	injectEntryPreludeImports(file, nil)
	if hasImport(file, "std/option") {
		t.Error("local constructor Some must shadow std/option (ctor-name collision)")
	}
}
