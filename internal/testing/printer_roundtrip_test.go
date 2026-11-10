package testing

import (
	"fmt"
	goast "go/ast"
	goparser "go/parser"
	gotoken "go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// printer_roundtrip_test.go — M-V1-VERIFICATION-HONESTY M3b.
//
// Named-test bodies are not compiled from their source text: each folded body
// is re-printed with PrintAILANGSource and re-parsed (named_batch.go). Four
// bugs came through that seam one at a time — `assert` (#590), float literals
// (#1448), #1821, and `x :: xs` printed as the never-parsing `::(x, xs)`
// (#1914). The property that would have caught all four:
//
//	parse(print(e)) is AST-equal to e
//
// checked here over every function body in examples/ and std/, and over a
// synthetic corpus that must contain every ast.Expr node type, so a node type
// added later without a printer case fails this test instead of being skipped.

const rtWrapPrefix = "module rt\n\nfunc rtBody() -> int = "

// parseBody parses src as the body of a one-function module and returns it.
// An `= expr` body is wrapped by the parser in a one-element Block; that
// wrapper belongs to the harness, not to src, so it is removed.
func parseBody(src string) (ast.Expr, []error) {
	p := parser.New(lexer.New(rtWrapPrefix+src+"\n", "rt.ail"))
	prog := p.Parse()
	if errs := p.Errors(); len(errs) > 0 {
		return nil, errs
	}
	if prog == nil || prog.File == nil || len(prog.File.Funcs) != 1 {
		return nil, []error{fmt.Errorf("wrapper did not parse to one function")}
	}
	body := prog.File.Funcs[0].Body
	if b, ok := body.(*ast.Block); ok && len(b.Exprs) == 1 {
		return b.Exprs[0], nil
	}
	return body, nil
}

// roundTrip prints e, re-parses it and reports a non-empty reason when the
// result is not AST-equal (String() is the position-free structural rendering).
func roundTrip(e ast.Expr) string {
	printed := PrintAILANGSource(e)
	if strings.Contains(printed, "<printer-error") {
		return "printer refused: " + printed
	}
	back, errs := parseBody(printed)
	if len(errs) > 0 {
		return fmt.Sprintf("printed source does not parse (%v):\n  %s", errs[0], printed)
	}
	if got, want := back.String(), e.String(); got != want {
		return fmt.Sprintf("AST changed:\n  printed: %s\n  want:    %s\n  got:     %s", printed, want, got)
	}
	return ""
}

// rtCorpus is one expression per ast.Expr node type (and every operator form),
// written as the body of `func rtBody() -> int = <expr>`. The bodies only
// need to PARSE: the round trip is syntactic.
var rtCorpus = map[string][]string{
	// The hex literals parse to NEGATIVE int literals (0x8000000000000000 is
	// the minimum int, whose decimal spelling does not parse); "${" in a plain
	// string must not come back as an interpolation.
	"Literal":    {`1`, `-7`, `1.5`, `"a\n\t\"q\""`, `true`, `()`, `0x8000000000000000`, `0x9e3779b97f4a7c15`, `"cost: \${x}"`},
	"Identifier": {`x`},
	"BinaryOp": {`a + b * c`, `a - b / c % d`, `a == b`, `a != b`, `a < b`, `a <= b`, `a > b`, `a >= b`,
		`a && b || c`, `a ++ b`, `xs ++ [1]`, `"a" ++ "b"`, `a ^ b`, `a << 2`, `a >> 1`, `(if c then [1] else [2]) ++ ys`},
	"UnaryOp":      {`-x`, `!b`, `-(if c then 1 else 2)`},
	"FuncCall":     {`f()`, `f(1, 2)`, `f(g(x))`, `x :: xs`, `1 :: 2 :: []`, `f(x :: xs)`, `(x :: xs) ++ ys`, `length(identity(x :: xs)) == 2`, `(\f. f)(g)(8)`, `f(1)(2)`},
	"Lambda":       {`\x. x + 1`, `\x y. x`},
	"FuncLit":      {`func(x: int) -> int { x }`, `func(x: int) -> int ! {IO} { x }`},
	"Let":          {`{ let x = 1; x }`, `let y = 2 in y`},
	"Block":        {`{ f(1); g(2) }`},
	"If":           {`if a then 1 else 2`, `if a then if b then 1 else 2 else 3`},
	"Match":        {`match xs { [] => 0, x :: rest => x }`, `match o { Some(v) => v, None => 0 }`, `match n { 0 => 1, _ if n > 2 => 2, _ => 3 }`},
	"List":         {`[1, 2, 3]`, `[]`},
	"Array":        {`#[1, 2]`},
	"Tuple":        {`(1, "a")`},
	"Record":       {`{ a: 1, b: "x" }`},
	"RecordAccess": {`r.a`},
	"RecordUpdate": {`{ r | a: 2 }`},
}

// rtNotInCorpus lists the ast.Expr types that cannot appear as a parsed
// function body, each with the reason. Anything else missing from rtCorpus
// fails TestPrinterRoundTrip_CorpusCoversEveryExprType.
var rtNotInCorpus = map[string]string{
	"Error":      "parser error-recovery node; the printer emits a self-describing <printer-error>",
	"AssertStmt": "FoldTestBody lowers it before printing; reaching the printer is a tripwire (#590)",
	"LetRec":     "produced by desugaring, not by parsing a body",
	"QuasiQuote": "not accepted by the parser (LIMITATIONS: typed quasiquotes not implemented)",
	"Send":       "no channel surface in the parser (LIMITATIONS: CSP concurrency deferred)",
	"Recv":       "no channel surface in the parser (LIMITATIONS: CSP concurrency deferred)",
	"ForallExpr": "only valid inside requires/ensures, never as a function body",
}

// exprTypes reads internal/ast and returns every type with an exprNode method.
func exprTypes(t *testing.T) []string {
	t.Helper()
	fset := gotoken.NewFileSet()
	pkgs, err := goparser.ParseDir(fset, "../ast", func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parse internal/ast: %v", err)
	}
	var names []string
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			for _, d := range f.Decls {
				fd, ok := d.(*goast.FuncDecl)
				if !ok || fd.Name.Name != "exprNode" || fd.Recv == nil || len(fd.Recv.List) != 1 {
					continue
				}
				if star, ok := fd.Recv.List[0].Type.(*goast.StarExpr); ok {
					if id, ok := star.X.(*goast.Ident); ok {
						names = append(names, id.Name)
					}
				}
			}
		}
	}
	sort.Strings(names)
	if len(names) < 20 {
		t.Fatalf("found only %d ast.Expr types (%v): the instrument is broken", len(names), names)
	}
	return names
}

// collectTypes records the dynamic type name of every ast node reachable from e.
func collectTypes(e ast.Expr, seen map[string]bool) {
	var walk func(v reflect.Value)
	walk = func(v reflect.Value) {
		if !v.IsValid() {
			return
		}
		switch v.Kind() {
		case reflect.Interface, reflect.Ptr:
			if v.IsNil() {
				return
			}
			if v.Kind() == reflect.Ptr && v.Elem().Kind() == reflect.Struct {
				if _, ok := v.Interface().(ast.Expr); ok {
					seen[v.Elem().Type().Name()] = true
				}
			}
			walk(v.Elem())
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				if v.Type().Field(i).IsExported() {
					walk(v.Field(i))
				}
			}
		case reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				walk(v.Index(i))
			}
		}
	}
	walk(reflect.ValueOf(e))
}

// TestPrinterRoundTrip_Corpus round-trips every synthetic expression.
func TestPrinterRoundTrip_Corpus(t *testing.T) {
	for kind, srcs := range rtCorpus {
		for _, src := range srcs {
			e, errs := parseBody(src)
			if len(errs) > 0 {
				t.Errorf("%s: corpus entry %q does not parse: %v", kind, src, errs[0])
				continue
			}
			if reason := roundTrip(e); reason != "" {
				t.Errorf("%s %q: %s", kind, src, reason)
			}
		}
	}
}

// TestPrinterRoundTrip_CorpusCoversEveryExprType: every ast.Expr type is
// either exercised by rtCorpus or listed in rtNotInCorpus with a reason.
func TestPrinterRoundTrip_CorpusCoversEveryExprType(t *testing.T) {
	seen := map[string]bool{}
	for _, srcs := range rtCorpus {
		for _, src := range srcs {
			if e, errs := parseBody(src); len(errs) == 0 {
				collectTypes(e, seen)
			}
		}
	}
	for _, name := range exprTypes(t) {
		if seen[name] {
			continue
		}
		if _, ok := rtNotInCorpus[name]; ok {
			continue
		}
		t.Errorf("ast.%s is neither exercised by rtCorpus nor listed in rtNotInCorpus: add a corpus entry (and a PrintAILANGSource case)", name)
	}
	for name := range rtNotInCorpus {
		if seen[name] {
			t.Errorf("ast.%s is listed as not-in-corpus but rtCorpus exercises it: drop it from rtNotInCorpus", name)
		}
	}
}

// rtRoots are the trees that ship AILANG source. A missing root is a failure,
// not a skip: it would enumerate zero files and read like a clean sweep.
var rtRoots = []string{"../../examples", "../../std"}

// TestPrinterRoundTrip_ShippedSource round-trips every function body in
// examples/ and std/. Red mutation: delete the `::` case in PrintAILANGSource —
// every body containing a cons fails with "printed source does not parse".
func TestPrinterRoundTrip_ShippedSource(t *testing.T) {
	var bodies, files int
	var failures []string
	for _, root := range rtRoots {
		if _, err := os.Stat(root); err != nil {
			t.Fatalf("corpus root %s is missing: %v", root, err)
		}
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || !strings.HasSuffix(path, ".ail") {
				return nil
			}
			data, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			p := parser.New(lexer.New(string(data), path))
			prog := p.Parse()
			if len(p.Errors()) > 0 || prog == nil || prog.File == nil {
				return nil // negative fixtures and mid-edit files are not round-trip inputs
			}
			files++
			for _, fn := range prog.File.Funcs {
				if fn.Body == nil {
					continue
				}
				bodies++
				if reason := roundTrip(fn.Body); reason != "" {
					failures = append(failures, fmt.Sprintf("%s: func %s: %s", path, fn.Name, reason))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
	if files < 100 || bodies < 1000 {
		t.Fatalf("swept only %d files / %d bodies: the instrument is broken", files, bodies)
	}
	if len(failures) > 0 {
		sort.Strings(failures)
		max := len(failures)
		if max > 25 {
			max = 25
		}
		t.Fatalf("%d of %d function bodies do not round-trip through PrintAILANGSource (first %d):\n%s",
			len(failures), bodies, max, strings.Join(failures[:max], "\n"))
	}
	t.Logf("round-tripped %d function bodies from %d files", bodies, files)
}
