package pipeline

import (
	"fmt"
	goast "go/ast"
	"go/importer"
	goparser "go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
)

// TC_DEPRECATED_001 covers every place a module can use a name. One fixture per
// construct that holds an expression; each uses the deprecated lib.f exactly
// once (or, for the shadowing rows, never really uses it). Both paths are
// covered: constructs elaborated into Core and the ones only the parsed source
// has (named test blocks, property blocks, inline tests, forall properties),
// which `check` never elaborates.
var deprecatedCoverageCases = []struct {
	construct string
	src       string // body after "module <name>\nimport lib (f)\nimport lib as L\n"
	want      int
}{
	{"function body", "export pure func k() -> int = f(1)\n", 1},
	{"lambda body", "export pure func k() -> int = (\\x. f(x))(1)\n", 1},
	{"function passed as a value", "import std/list (map)\nexport pure func k(xs: [int]) -> [int] = map(f, xs)\n", 1},
	{"match arm body", "export pure func k(n: int) -> int = match n { 0 => f(0), _ => 1 }\n", 1},
	{"match guard", "export pure func k(n: int) -> int = match n { m if f(m) > 0 => 1, _ => 0 }\n", 1},
	{"record field / list / tuple element", "export pure func k() -> [(int, int)] = [(f(1), 2)]\n", 1},
	{"if condition", "export pure func k(n: int) -> int = if f(n) > 0 then 1 else 0\n", 1},
	{"let value in a block", "export pure func k() -> int {\n  let y = f(1);\n  y\n}\n", 1},
	{"qualified L.f", "export pure func k() -> int = L.f(1)\n", 1},
	{"requires clause", "export pure func k(x: int) -> int\nrequires { f(x) > 0 }\n{ x }\n", 1},
	{"ensures clause", "export pure func k(x: int) -> int\nensures { result == f(x) - 1 }\n{ x }\n", 1},
	{"inline tests: input", "export pure func k(x: int) -> int\n  tests [(f(1), 2)]\n{ x }\n", 1},
	{"inline tests: expected", "export pure func k(x: int) -> int\n  tests [(1, f(0))]\n{ x }\n", 1},
	{"named test block", "export pure func k() -> int = 1\n\ntest \"t\" {\n  assert f(1) == 2\n}\n", 1},
	{"named test block, qualified", "export pure func k() -> int = 1\n\ntest \"t\" {\n  assert L.f(1) == 2\n}\n", 1},
	{"named test block, let then use", "export pure func k() -> int = 1\n\ntest \"t\" {\n  let y = f(1);\n  assert y == 2\n}\n", 1},
	{"named test block, in a lambda", "export pure func k() -> int = 1\n\ntest \"t\" {\n  assert (\\x. f(x))(1) == 2\n}\n", 1},
	{"function properties [forall]", "export pure func k(x: int) -> int\n  properties [forall(y: int) => f(y) > y]\n{ x }\n", 1},
	{"property block", "export pure func k() -> int = 1\n\nproperty \"p\" {\n  forall(x: int) => f(x) == x + 1\n}\n", 1},
	{"module-level let", "let c = f(1)\nexport pure func k() -> int = c\n", 1},
	{"top-level expression", "export pure func k() -> int = 1\nf(1)\n", 1},
	// Shadowing: a local binder named f is not the import.
	{"test block: let shadows the import", "export pure func k() -> int = 1\n\ntest \"t\" {\n  let f = 5;\n  assert f == 5\n}\n", 0},
	{"test block: lambda parameter shadows", "export pure func k() -> int = 1\n\ntest \"t\" {\n  assert (\\f. f + 1)(1) == 2\n}\n", 0},
	{"property: forall binder shadows", "export pure func k() -> int = 1\n\nproperty \"p\" {\n  forall(f: int) => f == f\n}\n", 0},
	{"test block: match variable shadows", "export pure func k() -> int = 1\n\ntest \"t\" {\n  assert (match 3 { f => f + 1 }) == 4\n}\n", 0},
}

func TestDeprecatedUseCoverage(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("AILANG_CACHE_DIR", filepath.Join(root, "cache"))
	writeModule(t, "lib.ail", "module lib\n\nexport pure func g(x: int) -> int = x + 1\n\n@deprecated(\"g\")\nexport pure func f(x: int) -> int = g(x)\n")
	for i, c := range deprecatedCoverageCases {
		t.Run(c.construct, func(t *testing.T) {
			name := "cov" + string(rune('a'+i%26)) + strings.Repeat("x", i/26)
			src := "module " + name + "\nimport lib (f)\nimport lib as L\n" + c.src
			writeModule(t, name+".ail", src)
			for _, run := range []string{"cold", "warm"} {
				res, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: name + ".ail"}, productionCacheDependencies())
				if err != nil {
					t.Fatalf("%s: fixture does not compile: %v\n%s", run, err, src)
				}
				var ws, unusedImport []*DeprecatedUseWarning
				for _, w := range deprecatedWarnings(res.Warnings) {
					if w.UnusedImport {
						unusedImport = append(unusedImport, w)
					} else {
						ws = append(ws, w)
					}
				}
				// No use of f (the shadowing rows): the import warns, once.
				// Any use: the import does not warn.
				wantUnused := 0
				if c.want == 0 {
					wantUnused = 1
				}
				if len(unusedImport) != wantUnused {
					t.Fatalf("%s: %d unused-import warnings, want %d\n%s\n%v", run, len(unusedImport), wantUnused, src, unusedImport)
				}
				if len(ws) != c.want {
					t.Fatalf("%s: %d TC_DEPRECATED_001, want %d\n%s\n%v", run, len(ws), c.want, src, ws)
				}
				for _, w := range ws {
					line, col := w.Position()
					if filepath.Base(w.WarningFile()) != name+".ail" || line < 4 || col < 1 || w.Name != "f" {
						t.Errorf("%s: misplaced warning %s at %s:%d:%d", run, w.Name, w.WarningFile(), line, col)
					}
				}
			}
		})
	}
}

// The guard is driven by the AST package's own type definitions, not by what
// some .ail file happens to use: it parses internal/ast/*.go and lists every
// type with a stmtNode() method (top-level statements) or an exprNode() method
// (expressions). Each must be classified below — how its expressions are
// searched, and whether it binds names — so adding a node type fails here until
// someone decides. A second part runs the AST pass over a synthetic module
// holding every AST-only statement kind and checks each one is searched.
var deprecatedStmtKinds = map[string]string{
	"FuncDecl":     "body and contracts reach Core; tests and properties are walked by the AST pass",
	"TypeDecl":     "no expressions",
	"TestDecl":     "AST pass (a sequence: let scopes over the rest)",
	"PropertyDecl": "AST pass (forall binders in scope)",
	"TypeClass":    "AST pass, generic walk (method defaults)",
	"Instance":     "AST pass, generic walk (method bodies)",
	"AssertStmt":   "only inside test bodies (top-level assert does not parse); generic walk",
}

// deprecatedExprKinds: "binder" kinds are handled explicitly by
// astUseWalker.walk (they bring names into scope); "plain" kinds bind nothing
// and are searched field by field.
var deprecatedExprKinds = map[string]string{
	"Identifier": "use", "RecordAccess": "use (L.f)",
	"Lambda": "binder", "FuncLit": "binder", "Let": "binder", "LetRec": "binder",
	"Block": "binder (sequential let)", "Match": "binder (pattern variables)", "ForallExpr": "binder",
	"Literal": "plain", "BinaryOp": "plain", "UnaryOp": "plain", "AssertStmt": "plain",
	"FuncCall": "plain", "If": "plain", "List": "plain", "Array": "plain", "Tuple": "plain",
	"Record": "plain", "RecordUpdate": "plain", "Error": "plain", "QuasiQuote": "plain",
	"Send": "plain", "Recv": "plain",
}

// astMarkerTypes returns every named type of internal/ast (non-test sources)
// whose method set has a method called marker.
func astMarkerTypes(t *testing.T, marker string) []string {
	t.Helper()
	_, self, _, _ := runtime.Caller(0)
	out, err := markerTypesIn(filepath.Join(filepath.Dir(self), "..", "ast"), marker)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// markerTypesIn type-checks the Go package in dir and returns the names of its
// non-interface named types whose method set (of T or *T) has a method called
// marker. Method sets come from go/types, so a marker promoted from an embedded
// struct and one declared on a generic receiver both count — a scan of
// receiver syntax misses both.
func markerTypesIn(dir, marker string) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil || len(files) == 0 {
		return nil, fmt.Errorf("no Go sources in %s: %v", dir, err)
	}
	fset := token.NewFileSet()
	var parsed []*goast.File
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		file, err := goparser.ParseFile(fset, f, nil, 0)
		if err != nil {
			return nil, err
		}
		parsed = append(parsed, file)
	}
	conf := types.Config{Importer: importer.Default()}
	pkg, err := conf.Check(parsed[0].Name.Name, fset, parsed, nil)
	if err != nil {
		return nil, err
	}
	var out []string
	scope := pkg.Scope()
	for _, name := range scope.Names() {
		tn, ok := scope.Lookup(name).(*types.TypeName)
		if !ok || tn.IsAlias() || types.IsInterface(tn.Type()) {
			continue
		}
		if types.NewMethodSet(types.NewPointer(tn.Type())).Lookup(pkg, marker) != nil {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out, nil
}

// The guard itself: a node kind whose marker is promoted from an embedded
// struct, or declared on a generic receiver, is still found.
func TestMarkerTypesSeesEmbeddedAndGenericReceivers(t *testing.T) {
	dir := t.TempDir()
	src := `package fake

type Expr interface{ exprNode() }

type base struct{}

func (base) exprNode() {}

type ByEmbedding struct{ base }

type ByPointerEmbedding struct{ *base }

type Generic[T any] struct{ v T }

func (*Generic[T]) exprNode() {}

type Plain struct{}

func (Plain) exprNode() {}

type NotANode struct{}
`
	if err := os.WriteFile(filepath.Join(dir, "fake.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := markerTypesIn(dir, "exprNode")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"ByEmbedding", "ByPointerEmbedding", "Generic", "Plain", "base"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("marker types = %v, want %v", got, want)
	}
}

func TestDeprecatedASTConstructsClassified(t *testing.T) {
	stmts := astMarkerTypes(t, "stmtNode")
	exprs := astMarkerTypes(t, "exprNode")
	if len(stmts) < 5 || len(exprs) < 20 {
		t.Fatalf("control: found %d statement and %d expression kinds in internal/ast; the scan is broken", len(stmts), len(exprs))
	}
	for _, k := range stmts {
		if _, ok := deprecatedStmtKinds[k]; !ok {
			t.Errorf("ast.%s is a statement kind that TC_DEPRECATED_001 has not classified: decide how its expressions are searched (warn_deprecated_ast.go) and add it here", k)
		}
	}
	for _, k := range exprs {
		if _, ok := deprecatedExprKinds[k]; !ok {
			t.Errorf("ast.%s is an expression kind that TC_DEPRECATED_001 has not classified: if it binds names, handle it in astUseWalker.walk; then add it here", k)
		}
	}
	for k := range deprecatedStmtKinds {
		if !contains(stmts, k) {
			t.Errorf("deprecatedStmtKinds lists ast.%s, which is no longer a statement kind", k)
		}
	}
	for k := range deprecatedExprKinds {
		if !contains(exprs, k) {
			t.Errorf("deprecatedExprKinds lists ast.%s, which is no longer an expression kind", k)
		}
	}

	// FuncDecl fields that can hold an expression.
	funcFields := map[string]string{
		"Body":        "core",
		"Properties":  "core (requires/ensures/invariant) + ast (forall)",
		"Tests":       "ast",
		"Annotations": "string literals only",
	}
	ft := reflect.TypeOf(ast.FuncDecl{})
	for i := 0; i < ft.NumField(); i++ {
		f := ft.Field(i)
		if holdsExpr(f.Type, map[reflect.Type]bool{}) {
			if _, ok := funcFields[f.Name]; !ok {
				t.Errorf("ast.FuncDecl.%s can hold an expression and is not classified for TC_DEPRECATED_001", f.Name)
			}
		}
	}
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

// Every AST-only statement kind — and a statement kind the pass has never
// heard of — is searched: one synthetic module holds a use of the deprecated
// lib.f in each, and each yields its warning.
func TestDeprecatedASTPassSearchesEveryStatementKind(t *testing.T) {
	use := func(line int) ast.Expr {
		return &ast.Identifier{Name: "f", Pos: ast.Pos{File: "m.ail", Line: line, Column: 1}}
	}
	stmts := []ast.Node{
		&ast.TestDecl{Name: "t", Body: []ast.Expr{&ast.AssertStmt{Condition: use(10)}}},
		&ast.PropertyDecl{Name: "p", Property: &ast.Property{Kind: ast.PropertyKind, Expr: use(11)}},
		&ast.Instance{ClassName: "C", Methods: map[string]ast.Expr{"m": use(12)}},
		&ast.TypeClass{Name: "C", Methods: []*ast.Method{{Name: "m", Default: use(13)}}},
		&unknownStmt{Payload: use(14)},
	}
	u := &CompileUnit{ID: "m", Surface: &ast.File{
		Imports:    []*ast.ImportDecl{{Path: "lib", Symbols: []string{"f"}}},
		Statements: stmts,
	}}
	idx := map[core.GlobalRef]deprecation{{Module: "lib", Name: "f"}: {replacement: "g"}}
	var lines []int
	for _, w := range deprecatedUsesOutsideCore(u, idx) {
		lines = append(lines, w.Location.Line)
	}
	sort.Ints(lines)
	if !reflect.DeepEqual(lines, []int{10, 11, 12, 13, 14}) {
		t.Errorf("uses found at lines %v, want one in each statement kind [10 11 12 13 14]", lines)
	}
}

// unknownStmt stands in for a statement kind added to the parser later: the
// pass must search it without being told about it.
type unknownStmt struct {
	Payload ast.Expr
}

func (*unknownStmt) String() string    { return "unknown" }
func (*unknownStmt) Position() ast.Pos { return ast.Pos{} }

// holdsExpr reports whether a value of type t can contain an ast.Expr.
func holdsExpr(t reflect.Type, visiting map[reflect.Type]bool) bool {
	exprT := reflect.TypeOf((*ast.Expr)(nil)).Elem()
	if t == exprT || (t.Kind() != reflect.Interface && t.Implements(exprT)) {
		return true
	}
	if visiting[t] {
		return false
	}
	visiting[t] = true
	switch t.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Array, reflect.Map:
		return holdsExpr(t.Elem(), visiting)
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			if holdsExpr(t.Field(i).Type, visiting) {
				return true
			}
		}
	}
	return false
}
