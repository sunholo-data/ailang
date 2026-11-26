package parser

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

func mustParseModule(t *testing.T, src string) *ast.Program {
	t.Helper()
	p := New(lexer.New(src, "test.ail"))
	prog := p.Parse()
	if errs := p.Errors(); len(errs) > 0 {
		for _, err := range errs {
			t.Errorf("  parse error: %v", err)
		}
		t.Fatalf("expected 0 parse errors, got %d", len(errs))
	}
	return prog
}

// TST-TL-01: `let x = expr?` in `in`-chains, `{ ... }` `;`/newline blocks,
// unbraced match/if chains, typed lets, and tuple/record destructuring.
func TestTryLet_ParsesAllLetForms(t *testing.T) {
	src := `module try_parse

export pure func inChain(s: string) -> Result[int, string] =
  let a: int = parseA(s)? in
  let b = parseB(a)? in
  Ok(a + b)

export pure func blockForms(s: string) -> Result[int, string] {
  let a = parseA(s)?;
  let b = parseB(a)?
  let (x, y) = splitPair(b)?
  let {lo, hi} = bounds(x)?
  let z = (if y > 0 then Ok(lo) else Ok(hi))?
  let w = s |> parseA?
  Ok(z + w)
}

export pure func unbracedBranches(c: bool, opt: Option[string]) -> Option[int] =
  match opt {
    Some(s) =>
      if c then
        let a = toOpt(s)?
        Some(a + 1)
      else
        let b = toOpt(s)?;
        Some(b * 2)
    None => None
  }
`
	prog := mustParseModule(t, src)
	if got := len(prog.File.Funcs); got != 3 {
		t.Fatalf("expected 3 functions, got %d", got)
	}

	// Verify inChain has nested Let nodes with IsTry=true and non-zero TryPos.
	inFn := prog.File.Funcs[0]
	outerBlk, ok := inFn.Body.(*ast.Block)
	if !ok || len(outerBlk.Exprs) != 1 {
		t.Fatalf("expected 1-expr equation block, got %T", inFn.Body)
	}
	letA, ok := outerBlk.Exprs[0].(*ast.Let)
	if !ok || !letA.IsTry || letA.TryPos.Line == 0 || letA.Type == nil {
		t.Fatalf("letA = %#v, want IsTry=true with Type and TryPos", letA)
	}
	letB, ok := letA.Body.(*ast.Let)
	if !ok || !letB.IsTry || letB.TryPos.Line == 0 {
		t.Fatalf("letB = %#v, want IsTry=true with TryPos", letB)
	}

	// Verify blockForms statements carry IsTry=true on Let and LetPattern.
	blk, ok := prog.File.Funcs[1].Body.(*ast.Block)
	if !ok || len(blk.Exprs) != 7 {
		t.Fatalf("expected 7 statements in blockForms, got %T (%v)", prog.File.Funcs[1].Body, prog.File.Funcs[1].Body)
	}
	for _, idx := range []int{0, 1, 4, 5} {
		l, ok := blk.Exprs[idx].(*ast.Let)
		if !ok || !l.IsTry {
			t.Errorf("stmt[%d] = %#v, want *ast.Let{IsTry: true}", idx, blk.Exprs[idx])
		}
	}
	for _, idx := range []int{2, 3} {
		lp, ok := blk.Exprs[idx].(*ast.LetPattern)
		if !ok || !lp.IsTry {
			t.Errorf("stmt[%d] = %#v, want *ast.LetPattern{IsTry: true}", idx, blk.Exprs[idx])
		}
	}
}

// TST-TL-01: `_?` and `?name` typed holes stay 100% intact (`IsTry == false`),
// including when a named hole `?todo` appears on the line after a bare `let`.
func TestTryLet_PreservesTypedHoles(t *testing.T) {
	src := `module holes_intact

export pure func f(x: int) -> int {
  let a = _?
  let b = ?namedHole
  let c = x + 1;
  ?returnHole
}
`
	prog := mustParseModule(t, src)
	blk, ok := prog.File.Funcs[0].Body.(*ast.Block)
	if !ok || len(blk.Exprs) != 4 {
		t.Fatalf("expected 4 statements, got %T", prog.File.Funcs[0].Body)
	}

	letA, ok := blk.Exprs[0].(*ast.Let)
	if !ok || letA.IsTry {
		t.Fatalf("let a = _? got IsTry=%v, want false", letA != nil && letA.IsTry)
	}
	if h, ok := letA.Value.(*ast.Hole); !ok || h.Name != "" {
		t.Fatalf("let a value = %#v, want anonymous *ast.Hole", letA.Value)
	}

	letB, ok := blk.Exprs[1].(*ast.Let)
	if !ok || letB.IsTry {
		t.Fatalf("let b = ?namedHole got IsTry=%v, want false", letB != nil && letB.IsTry)
	}
	if h, ok := letB.Value.(*ast.Hole); !ok || h.Name != "namedHole" {
		t.Fatalf("let b value = %#v, want *ast.Hole{Name: \"namedHole\"}", letB.Value)
	}

	letC, ok := blk.Exprs[2].(*ast.Let)
	if !ok || letC.IsTry {
		t.Fatalf("let c = x + 1 before newline ?returnHole got IsTry=%v, want false", letC != nil && letC.IsTry)
	}

	retHole, ok := blk.Exprs[3].(*ast.Hole)
	if !ok || retHole.Name != "returnHole" {
		t.Fatalf("final expr = %#v, want *ast.Hole{Name: \"returnHole\"}", blk.Exprs[3])
	}
}

// TST-TL-01: Sub-expression and non-let postfix `?` emit PAR_TRY_LET_ONLY with
// an actionable hoist suggestion and no cascading delimiter errors.
func TestTryLet_RejectsSubexpressionTryWithPAR_TRY_LET_ONLY(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		wantHint string
	}{
		{"call_arg", `f(g()?)`, "let x = g()?"},
		{"nested_in_let_call_arg", `{ let y = f(g()?); Ok(y) }`, "let x = g()?"},
		{"binary_lhs", `{ let y = g()? + 1; Ok(y) }`, "let x = g()?"},
		{"binary_rhs", `{ let y = 1 + g()?; Ok(y) }`, "let x = g()?"},
		{"field_access_base", `{ let y = g()?.val; Ok(y) }`, "let x = g()?"},
		{"list_element", `[g()?]`, "let x = g()?"},
		{"record_field", `{ a: g()? }`, "let x = g()?"},
		{"return_expr_equation", `g()?`, "let x = g()?"},
		{"return_expr_block", `{ let a = f()?; g(a)? }`, "let x = g(...)?"},
		{"chained_double_try", `{ let a = f()??; Ok(a) }`, "let tmp = expr?"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := "module bad_try\nexport pure func run() -> Result[int, string] = " + tc.body + "\n"
			p := New(lexer.New(src, "bad_try.ail"))
			_ = p.Parse()
			errs := p.Errors()
			if len(errs) == 0 {
				t.Fatalf("expected %s error for %q, got none", CodeTryLetOnly, tc.body)
			}
			first := errs[0].Error()
			if !strings.Contains(first, CodeTryLetOnly) {
				t.Fatalf("expected first error to have code %s, got:\n%s", CodeTryLetOnly, first)
			}
			if !strings.Contains(first, tc.wantHint) {
				t.Errorf("expected error to contain hoist hint %q, got:\n%s", tc.wantHint, first)
			}
		})
	}
}
