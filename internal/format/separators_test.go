package format

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// separators_test.go proves two design guarantees about statement separators:
//
//  1. The three equivalent statement-sequence spellings the parser accepts
//     (equation `;`, braced `;`, braced newline) all format to the SAME canonical
//     newline-per-statement braced text.
//  2. Explicit `let ... in` is structurally distinct (a nested ast.Let with a
//     non-nil Body) and MUST remain explicit `let ... in` after formatting,
//     round-tripping to the same AST.

const sepDir = "testdata/separators"

func readFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(sepDir, name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(data)
}

// TestThreeSeparatorSpellingsFormatIdentically checks that the equation-semicolon,
// braced-semicolon, and braced-newline fixtures all produce byte-identical
// canonical output.
func TestThreeSeparatorSpellingsFormatIdentically(t *testing.T) {
	fixtures := []string{
		"equation_semicolon.ail",
		"braced_semicolon.ail",
		"braced_newline.ail",
	}
	var canonical string
	for i, name := range fixtures {
		src := readFixture(t, name)
		got := formatSrc(t, src)
		if i == 0 {
			canonical = got
			continue
		}
		if got != canonical {
			t.Errorf("%s formatted differently from %s:\n--- %s ---\n%s\n--- %s ---\n%s",
				name, fixtures[0], name, got, fixtures[0], canonical)
		}
	}

	// The canonical form is the braced newline-per-statement block with no
	// semicolons.
	want := "module test/sep\n\nfunc f() -> int {\n  let a = 1\n  let b = 2\n  a + b\n}\n"
	if canonical != want {
		t.Errorf("unexpected canonical form:\n got %q\nwant %q", canonical, want)
	}
}

// TestExplicitLetInStaysExplicit proves the let..in fixture keeps its explicit
// form and that its body lets are non-nil-Body ast.Let nodes (distinct from the
// nil-Body statement lets of the sequence spellings).
func TestExplicitLetInStaysExplicit(t *testing.T) {
	src := readFixture(t, "let_in.ail")

	// The sequence fixtures format to a braced block; the let..in fixture must NOT.
	got := formatSrc(t, src)
	want := "module test/sep\n\nfunc f() -> int = let a = 1 in let b = 2 in a + b\n"
	if got != want {
		t.Errorf("explicit let..in not preserved:\n got %q\nwant %q", got, want)
	}

	// Structural check: the outermost body expression is a non-nil-Body ast.Let.
	prog := parseProg(t, src, "let_in.ail")
	body := prog.File.Funcs[0].Body
	blk, ok := body.(*ast.Block)
	if !ok || len(blk.Exprs) != 1 {
		t.Fatalf("expected single-expr Block wrapping the let..in, got %T", body)
	}
	outerLet, ok := blk.Exprs[0].(*ast.Let)
	if !ok {
		t.Fatalf("expected outer ast.Let, got %T", blk.Exprs[0])
	}
	if outerLet.Body == nil {
		t.Fatal("explicit let..in must have a non-nil Body")
	}

	// And it must round-trip.
	assertIdempotentAndRoundTrips(t, src, "let_in.ail")
}

// TestSequenceLetsAreNilBody confirms the sequence spellings produce nil-Body
// statement lets — the structural distinction that lets the formatter choose the
// newline-statement form over `let..in`.
func TestSequenceLetsAreNilBody(t *testing.T) {
	prog := parseProg(t, readFixture(t, "equation_semicolon.ail"), "eq.ail")
	body := prog.File.Funcs[0].Body
	blk, ok := body.(*ast.Block)
	if !ok {
		t.Fatalf("expected Block body, got %T", body)
	}
	for i, e := range blk.Exprs {
		if let, ok := e.(*ast.Let); ok && let.Body != nil {
			t.Errorf("sequence let %d unexpectedly has a non-nil Body", i)
		}
	}
}

// TestSeparatorFixturesShareAST is a stronger cross-check: the three sequence
// spellings must parse to structurally identical ASTs (ignoring pos/span), which
// is WHY they format identically. This mirrors the parser's own R1/R2 tests.
func TestSeparatorFixturesShareAST(t *testing.T) {
	parseBody := func(name string) ast.Expr {
		src := readFixture(t, name)
		p := parser.New(lexer.New(src, name))
		prog := p.Parse()
		if len(p.Errors()) > 0 {
			t.Fatalf("%s parse errors: %v", name, p.Errors())
		}
		return prog.File.Funcs[0].Body
	}
	eq := parseBody("equation_semicolon.ail")
	bs := parseBody("braced_semicolon.ail")
	bn := parseBody("braced_newline.ail")
	if diff := cmp.Diff(eq, bs, ignorePosSpan); diff != "" {
		t.Errorf("equation vs braced-semicolon AST differ:\n%s", diff)
	}
	if diff := cmp.Diff(eq, bn, ignorePosSpan); diff != "" {
		t.Errorf("equation vs braced-newline AST differ:\n%s", diff)
	}
}
