package parser

// Tests for M-SYNTAX-AI-FORGIVING: the parser accepts the two statement-separator
// forms small models naturally write, eliminating the PAR017/PAR020 parse-failure
// class without changing the model.
//
//   R1 — `;`-separated statement sequences in `=` function bodies.
//   R2 — a newline as a soft statement separator inside `{ }` blocks.
//
// Both are additive and backward-compatible: no currently-valid program changes
// meaning (proven structurally here and in corpus_astdiff_test.go).

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

// ignorePos ignores all positional data so two ASTs can be compared purely
// structurally (an `=`-body and a `{ }`-body differ only in columns/spans).
var ignorePos = cmpopts.IgnoreTypes(ast.Pos{}, ast.Span{})

// parseOK parses input, fails the test on any parse error, and returns the File.
func parseOK(t *testing.T, input string) *ast.File {
	t.Helper()
	p := New(lexer.New(input, "test://unit"))
	prog := p.Parse()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("unexpected parse errors for %q:\n%v", input, errs)
	}
	if prog == nil || prog.File == nil {
		t.Fatalf("nil program/file for %q", input)
	}
	return prog.File
}

// funcBody returns the body of the first FuncDecl in the file.
func funcBody(t *testing.T, f *ast.File) ast.Expr {
	t.Helper()
	if len(f.Funcs) == 0 {
		t.Fatalf("no function declarations in file")
	}
	return f.Funcs[0].Body
}

// --- R1: `;`-sequences in `=` function bodies ---------------------------------

// TestR1_EqBodyMatchesBracedBody is the core structural guarantee: an `=`-body
// `;`-sequence produces the SAME AST as the equivalent braced `{ }` body, so
// elaboration and typing are unchanged.
func TestR1_EqBodyMatchesBracedBody(t *testing.T) {
	cases := []struct {
		name  string
		eq    string
		brace string
	}{
		{
			"near_miss_let_seq",
			"func g() -> int = let x = 5; x + 1",
			"func g() -> int { let x = 5; x + 1 }",
		},
		{
			"three_stmt_seq",
			"func g() = let a = 1; let b = 2; a + b",
			"func g() { let a = 1; let b = 2; a + b }",
		},
		{
			"validate_version_shape",
			`pure func validateVersion(v: string) -> bool = let parts = split(v, "."); length(parts) == 3`,
			`pure func validateVersion(v: string) -> bool { let parts = split(v, "."); length(parts) == 3 }`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			eqBody := funcBody(t, parseOK(t, tc.eq))
			brBody := funcBody(t, parseOK(t, tc.brace))
			if diff := cmp.Diff(brBody, eqBody, ignorePos); diff != "" {
				t.Errorf("=-body AST differs from braced-body AST (-brace +eq):\n%s", diff)
			}
		})
	}
}

// TestR1_MultiStmtIsBlock confirms a multi-statement `=`-body parses as an
// N-expression Block (not a single expression).
func TestR1_MultiStmtIsBlock(t *testing.T) {
	body := funcBody(t, parseOK(t, "func g() = let a = 1; let b = 2; a + b"))
	block, ok := body.(*ast.Block)
	if !ok {
		t.Fatalf("expected *ast.Block, got %T", body)
	}
	if len(block.Exprs) != 3 {
		t.Fatalf("expected 3 exprs in block, got %d", len(block.Exprs))
	}
}

// TestR1_SingleExprUnchanged is the regression guard: a single-expression
// `=`-body still parses as a 1-expr Block (identical to pre-R1 behaviour).
func TestR1_SingleExprUnchanged(t *testing.T) {
	body := funcBody(t, parseOK(t, "func g() = x * 2"))
	block, ok := body.(*ast.Block)
	if !ok {
		t.Fatalf("expected *ast.Block (1-expr wrapper), got %T", body)
	}
	if len(block.Exprs) != 1 {
		t.Fatalf("expected 1 expr in block, got %d", len(block.Exprs))
	}
	if _, ok := block.Exprs[0].(*ast.BinaryOp); !ok {
		t.Fatalf("expected BinaryOp body, got %T", block.Exprs[0])
	}
}

// TestR1_DeclBoundaries verifies the `=`-body sequence ends at the right
// top-level declaration boundary, producing the correct number of decls.
func TestR1_DeclBoundaries(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		wantFuncs int
		wantDecls int // Funcs + type/other decls in file.Decls
	}{
		// back-to-back function decls (no `;`): body of f ends at `func g`.
		{"back_to_back_no_semi", "func f() = 1\nfunc g() = 2", 2, 2},
		// back-to-back with a trailing `;`: `func g` is still the boundary.
		{"back_to_back_trailing_semi", "func f() = 1;\nfunc g() = 2", 2, 2},
		// body followed by `export func`: body ends at `export`.
		{"export_boundary", "func f() = 1\nexport func g() = 2", 2, 2},
		// body followed by a type declaration.
		{"type_boundary", "func f() = 1\ntype T = int", 1, 2},
		// body at EOF.
		{"eof_boundary", "func f() = let x = 1; x", 1, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := parseOK(t, tc.input)
			if len(f.Funcs) != tc.wantFuncs {
				t.Errorf("funcs: got %d, want %d", len(f.Funcs), tc.wantFuncs)
			}
			if len(f.Decls) != tc.wantDecls {
				t.Errorf("decls: got %d, want %d", len(f.Decls), tc.wantDecls)
			}
		})
	}
}

// TestR1_FuncLitInBody is the M-TAINT guard: an anonymous-function expression
// `func ( ... )` inside a call in the `=`-body must NOT be cut at `func(`.
func TestR1_FuncLitInBody(t *testing.T) {
	f := parseOK(t, "func f() -> int = g(func(x) -> int { x + 1 }, xs)")
	if len(f.Funcs) != 1 {
		t.Fatalf("expected 1 func (funclit stays inside body), got %d", len(f.Funcs))
	}
	body := funcBody(t, f)
	// Body is a single-expr Block wrapping the whole call.
	block, ok := body.(*ast.Block)
	if !ok || len(block.Exprs) != 1 {
		t.Fatalf("expected single-expr Block wrapping the call, got %T", body)
	}
	if _, ok := block.Exprs[0].(*ast.FuncCall); !ok {
		t.Fatalf("expected the call FuncCall as the body, got %T", block.Exprs[0])
	}
}

// TestR1_BackslashLambdaInBody: a backslash-lambda argument parses in the body.
func TestR1_BackslashLambdaInBody(t *testing.T) {
	parseOK(t, `func f() = map(\x. x * 2, xs)`)
}

// TestR1_OperatorLineContinuation: `= 1\n+ 2` is ONE expression (operator is not
// a statement start), unchanged by R1.
func TestR1_OperatorLineContinuation(t *testing.T) {
	body := funcBody(t, parseOK(t, "func f() = 1\n+ 2"))
	block, ok := body.(*ast.Block)
	if !ok || len(block.Exprs) != 1 {
		t.Fatalf("expected single-expr Block (operator continuation), got %T", body)
	}
	if _, ok := block.Exprs[0].(*ast.BinaryOp); !ok {
		t.Fatalf("expected BinaryOp (1 + 2), got %T", block.Exprs[0])
	}
}
