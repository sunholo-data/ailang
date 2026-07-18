package format

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// roundtrip_test.go holds the two formatter release-gate properties:
//
//   idempotence:  bytes(Source(Parse(Source(Parse(x))))) == bytes(Source(Parse(x)))
//   round-trip:   AST(Parse(Source(Parse(x)))) == AST(Parse(x))
//
// AST equality ignores ONLY positions/spans/file paths (go-cmp with explicit
// ignored types), never textual equality of debug String() methods.

// ignorePosSpan ignores positional metadata for structural AST comparison.
var ignorePosSpan = cmpopts.IgnoreTypes(ast.Pos{}, ast.Span{})

// parseProg parses source, failing the test on any parse error.
func parseProg(t *testing.T, src, path string) *ast.Program {
	t.Helper()
	p := parser.New(lexer.New(src, path))
	prog := p.Parse()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse errors for %s:\n%v", path, errs)
	}
	if prog == nil || prog.File == nil {
		t.Fatalf("nil program/file for %s", path)
	}
	return prog
}

// assertIdempotentAndRoundTrips formats src, then asserts both properties. It
// returns the canonical output for callers that want to make further assertions.
func assertIdempotentAndRoundTrips(t *testing.T, src, path string) string {
	t.Helper()
	prog := parseProg(t, src, path)

	out1, err := Source(prog, Options{})
	if err != nil {
		t.Fatalf("Source(%s): %v", path, err)
	}

	prog2 := parseProg(t, string(out1), path)

	// Idempotence: formatting the formatted output yields byte-identical text.
	out2, err := Source(prog2, Options{})
	if err != nil {
		t.Fatalf("Source(fmt(%s)): %v", path, err)
	}
	if string(out1) != string(out2) {
		t.Errorf("idempotence failed for %s:\n--- pass 1 ---\n%s\n--- pass 2 ---\n%s", path, out1, out2)
	}

	// Structural round-trip: re-parsed AST equals the original, ignoring pos/span.
	if diff := cmp.Diff(prog.File, prog2.File, ignorePosSpan); diff != "" {
		t.Errorf("round-trip AST changed for %s (-orig +formatted):\n%s", path, diff)
	}
	return string(out1)
}

// TestIdempotenceAndRoundTrip_GeneratedCases covers a hand-picked set of AST
// shapes exercising precedence, sequences, literals, patterns, and declarations.
func TestIdempotenceAndRoundTrip_GeneratedCases(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"module_import_func", "module m\nimport std/io (println)\nexport func main() -> () ! {IO} = println(\"hi\")"},
		{"eq_body_sequence", "module m\nfunc f() = let a = 1; let b = 2; a + b"},
		{"braced_sequence", "module m\nfunc f() { let a = 1; let b = 2; a + b }"},
		{"explicit_let_in", "module m\nfunc f() = let a = 1 in let b = 2 in a + b"},
		{"nested_if_blocks", "module m\nfunc n(x) = if x then { let a = 1; a } else { 0 }"},
		{"match_patterns", "module m\nfunc h(xs) = match xs { [] => 0, [x, ...r] => x }"},
		{"precedence_mix", "module m\nfunc p(a, b, c) = a && b || c == a"},
		{"paren_needed", "module m\nfunc p(a, b) = a * (b + 1)"},
		{"cons", "module m\nfunc c(a, b) = a :: b :: []"},
		{"adt", "module m\ntype Shape = Circle(int) | Rectangle(int, int) deriving (Eq)"},
		{"record_type", "module m\ntype R = { name: string, age: int }"},
		{"alias", "module m\ntype Id = int"},
		{"tuple_alias", "module m\ntype P = (int, string)"},
		{"top_level_let", "module m\nlet x: bool = let a = 1 in a == 1"},
		{"records_and_updates", "module m\nfunc r(base) = { base | x: 1, y: 2 }"},
		{"lambda_arg", "module m\nfunc f(g, ys) = g(\\x. x + 1, ys)"},
		{"string_escapes", "module m\nlet s = \"tab\\t newline\\n quote\\\" done\""},
		{"unary", "module m\nfunc u(x) = not x"},
		{"unit_statement", "module m\nfunc f() { let a = 1; () }"},
		{"array", "module m\nfunc a() = #[1, 2, 3]"},
		{"nested_records", "module m\nlet r = { a: { b: 1 }, c: 2 }"},
		{"multi_decl", "module m\nfunc a() = 1\nfunc b() = 2\ntype T = int"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertIdempotentAndRoundTrips(t, tc.src, "test://"+tc.name)
		})
	}
}
