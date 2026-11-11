package pipeline

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
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
				ws := deprecatedWarnings(res.Warnings)
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

// Every top-level node kind the parser produces, and every FuncDecl field
// that can hold an expression, is classified: reaches Core, walked from the
// parsed source by warn_deprecated_ast.go, or holds no expressions. A new
// construct fails here until it is classified (and covered above).
func TestDeprecatedASTConstructsClassified(t *testing.T) {
	coreTopLevel := map[string]bool{
		"*ast.FuncDecl": true, // body + contracts reach Core; tests/properties via the AST pass
		"*ast.Let":      true, // module-level let
		"*ast.LetRec":   true,
	}
	noExpressions := map[string]bool{
		"*ast.TypeDecl":   true,
		"*ast.ExternDecl": true,
		"*ast.ClassDecl":  true, // method signatures only
	}
	astOnly := map[string]bool{}
	for typ := range astOnlyTopLevel {
		astOnly[typ.String()] = true
	}

	// 1. FuncDecl fields that can hold an ast.Expr.
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

	// 2. Top-level node kinds in every .ail file of this repository.
	_, self, _, _ := runtime.Caller(0)
	repo := filepath.Join(filepath.Dir(self), "..", "..")
	seen := map[string]string{}
	for _, dir := range []string{"std", "examples", "tests", "benchmarks", "cmd/ailang/testdata", "internal/pipeline/testdata"} {
		_ = filepath.Walk(filepath.Join(repo, dir), func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(p, ".ail") {
				return nil
			}
			src, err := os.ReadFile(p)
			if err != nil {
				return nil
			}
			prs := parser.New(lexer.New(string(src), p))
			file := prs.ParseFile()
			if file == nil || len(prs.Errors()) > 0 {
				return nil
			}
			for _, st := range file.Statements {
				seen[reflect.TypeOf(st).String()] = p
			}
			return nil
		})
	}
	var kinds []string
	for k := range seen {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	for _, k := range kinds {
		if !coreTopLevel[k] && !noExpressions[k] && !astOnly[k] {
			// A top-level expression statement reaches Core as a module
			// binding; anything else must be classified explicitly.
			if isExprKind(k) {
				continue
			}
			t.Errorf("top-level node kind %s (e.g. %s) is not classified for TC_DEPRECATED_001", k, seen[k])
		}
	}
	for _, want := range []string{"*ast.TestDecl", "*ast.PropertyDecl"} {
		if _, ok := seen[want]; !ok {
			t.Errorf("control: no %s found in the repository, so this scan proves nothing about it", want)
		}
	}
}

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

// isExprKind reports whether a top-level statement kind is an expression: a
// top-level expression statement, elaborated into Core like a module let
// (the "top-level expression" coverage row).
func isExprKind(k string) bool {
	exprT := reflect.TypeOf((*ast.Expr)(nil)).Elem()
	for _, e := range []interface{}{&ast.FuncCall{}, &ast.Identifier{}, &ast.Literal{}, &ast.Block{}, &ast.If{},
		&ast.Match{}, &ast.BinaryOp{}, &ast.UnaryOp{}, &ast.Record{}, &ast.RecordAccess{}, &ast.RecordUpdate{},
		&ast.Tuple{}, &ast.List{}, &ast.Array{}, &ast.Lambda{}, &ast.FuncLit{}} {
		if reflect.TypeOf(e).String() == k && reflect.TypeOf(e).Implements(exprT) {
			return true
		}
	}
	return false
}
