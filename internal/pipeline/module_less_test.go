package pipeline

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/loader"
)

// makeModuleLessModule builds a LoadedModule whose File has NO module
// declaration. The caller decides whether it carries top-level Funcs /
// Statements / Decls to exercise the MOD014 gate.
func makeModuleLessModule(canonicalID, declPath string, funcs []*ast.FuncDecl, stmts, decls []ast.Node) *loader.LoadedModule {
	file := &ast.File{
		Path:       declPath,
		Funcs:      funcs,
		Statements: stmts,
		Decls:      decls,
	}
	return &loader.LoadedModule{
		Path: canonicalID,
		File: file,
	}
}

// TestValidateModulePath_ModuleLessWithFuncs is the core MOD014 test:
// a file that has top-level func declarations but no `module` header must
// FAIL LOUDLY (silent success footgun) with a fix-carrying diagnostic.
func TestValidateModulePath_ModuleLessWithFuncs(t *testing.T) {
	cfg := &Config{}
	mod := makeModuleLessModule(
		"nomod",
		"/tmp/nomod.ail",
		[]*ast.FuncDecl{{Name: "main"}},
		nil,
		nil,
	)
	err := validateModulePath(mod, "nomod", cfg)
	if err == nil {
		t.Fatal("expected MOD014 error for module-less file with top-level funcs, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "MOD014") {
		t.Errorf("error should mention MOD014, got: %s", msg)
	}
	if !strings.Contains(msg, "module") {
		t.Errorf("error should mention the missing 'module' declaration, got: %s", msg)
	}
	// Must carry an actionable fix naming the canonical module path.
	if !strings.Contains(msg, "Fix:") {
		t.Errorf("error should carry an actionable Fix: hint, got: %s", msg)
	}
	if !strings.Contains(msg, "module nomod") {
		t.Errorf("Fix should suggest the canonical 'module nomod' line, got: %s", msg)
	}
}

// TestValidateModulePath_ModuleLessDeclsOnlyNoFuncs guards the bare-expression
// eval path at the Decls level. The parser mirrors every top-level statement
// into BOTH Statements and Decls (back-compat, see parser_file.go), so a bare
// `1 + 1` file has a non-empty Decls slice but an empty Funcs slice. MOD014
// gates on Funcs ONLY, so this file must NOT fire — otherwise `ailang run` on a
// lone expression would break.
func TestValidateModulePath_ModuleLessDeclsOnlyNoFuncs(t *testing.T) {
	cfg := &Config{}
	expr := &ast.Literal{Kind: ast.IntLit, Value: int64(2)}
	mod := makeModuleLessModule(
		"bareexpr",
		"/tmp/bareexpr.ail",
		nil,
		[]ast.Node{expr}, // Statements
		[]ast.Node{expr}, // Decls (back-compat mirror)
	)
	if err := validateModulePath(mod, "bareexpr", cfg); err != nil {
		t.Fatalf("Decls-only (no Funcs) bare-expression file must not trip MOD014: %v", err)
	}
}

// TestValidateModulePath_EmptyFileNoModule confirms a genuinely empty
// module-less file (no funcs, no statements, no decls) is NOT flagged.
func TestValidateModulePath_EmptyFileNoModule(t *testing.T) {
	cfg := &Config{}
	mod := makeModuleLessModule("empty", "/tmp/empty.ail", nil, nil, nil)
	if err := validateModulePath(mod, "empty", cfg); err != nil {
		t.Fatalf("genuinely empty module-less file must not trip MOD014: %v", err)
	}
}

// TestValidateModulePath_BareExpressionNoModule guards the bare-expression
// escape hatch: a file that is a single top-level expression (a Statement)
// to be evaluated (e.g. `1 + 1`) must NOT trip MOD014 — it has no Funcs.
func TestValidateModulePath_BareExpressionNoModule(t *testing.T) {
	cfg := &Config{}
	mod := makeModuleLessModule(
		"bareexpr",
		"/tmp/bareexpr.ail",
		nil,
		[]ast.Node{&ast.Literal{Kind: ast.IntLit, Value: int64(2)}},
		nil,
	)
	if err := validateModulePath(mod, "bareexpr", cfg); err != nil {
		t.Fatalf("bare-expression file must not trip MOD014: %v", err)
	}
}

// TestValidateModulePath_ProperModuleUntouched confirms a well-formed module
// file with a matching declaration still validates clean.
func TestValidateModulePath_ProperModuleUntouched(t *testing.T) {
	cfg := &Config{}
	file := &ast.File{
		Path:   "/tmp/good.ail",
		Module: &ast.ModuleDecl{Path: "examples/good"},
		Funcs:  []*ast.FuncDecl{{Name: "main"}},
	}
	mod := &loader.LoadedModule{Path: "examples/good", File: file}
	if err := validateModulePath(mod, "examples/good", cfg); err != nil {
		t.Fatalf("proper module file must validate clean: %v", err)
	}
}
