package testing

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

// Regression suite for #590: `test "name" { assert <cond> }` failed 100% of the
// time with PAR_NO_PREFIX_PARSE, because EvaluateNamedTestBodyExprs prints the
// body back to AILANG source and re-runs the general pipeline, which has no
// `assert` prefix parselet.
//
// The fix lowers AssertStmt in FoldTestBody (internal/testing/test_body_lowering.go)
// to a short-circuiting `if` chain over an int sentinel — see the sprint plan
// design_docs/planned/v0_33_1/m-named-test-assert-lowering-sprint-plan.md §4.

// assertFixtureSource builds a minimal module with a single named test body.
func assertFixtureSource(body string) string {
	return `module assert_fixture

export pure func add_one(x: int) -> int { x + 1 }

test "assert fixture" {
` + body + `
}
`
}

// TestNamedTest_Assert_Passes is AC-2: a body whose only check is a passing
// assert must run and pass. RED before the fix (PAR_NO_PREFIX_PARSE).
func TestNamedTest_Assert_Passes(t *testing.T) {
	result := runInlineTestsOnSource(t, assertFixtureSource(`  assert add_one(1) == 2`))

	if result.FailedTests != 0 {
		t.Errorf("expected 0 failures, got %d; first error: %s",
			result.FailedTests, firstFailureError(result))
	}
	if result.PassedTests != 1 {
		t.Errorf("expected 1 passing test, got %d", result.PassedTests)
	}
}

// TestNamedTest_Assert_FalseFails is AC-3: a body whose only check is a FALSE
// assert must fail — and must fail for the right reason, i.e. not because the
// printed source failed to parse.
func TestNamedTest_Assert_FalseFails(t *testing.T) {
	result := runInlineTestsOnSource(t, assertFixtureSource(`  assert add_one(1) == 99`))

	if result.FailedTests != 1 {
		t.Errorf("expected 1 failure, got %d (passed=%d)", result.FailedTests, result.PassedTests)
	}
	errText := firstFailureError(result)
	if strings.Contains(errText, "PAR_NO_PREFIX_PARSE") {
		t.Errorf("failure is a parse error, not an assertion failure: %s", errText)
	}
	if strings.Contains(errText, "assert") && strings.Contains(errText, "unexpected token") {
		t.Errorf("failure mentions an unexpected `assert` token: %s", errText)
	}
}

// TestNamedTest_Assert_ShortCircuitsOnFirstFalse is AC-4, the anti-F2 guard.
// A non-final FALSE assert must fail the test. Under the legacy fold every
// non-final expression was bound to a dead `_seq` and discarded, so this body
// would report a vacuous green if the lowering were done per-node instead of
// across the sequence.
func TestNamedTest_Assert_ShortCircuitsOnFirstFalse(t *testing.T) {
	result := runInlineTestsOnSource(t, assertFixtureSource(
		`  assert add_one(1) == 99;
  assert add_one(1) == 2`))

	if result.FailedTests != 1 {
		t.Errorf("expected 1 failure (non-final false assert must not be discarded), got %d (passed=%d); error: %s",
			result.FailedTests, result.PassedTests, firstFailureError(result))
	}
	errText := firstFailureError(result)
	if strings.Contains(errText, "PAR_NO_PREFIX_PARSE") {
		t.Errorf("failure is a parse error, not an assertion failure: %s", errText)
	}
}

// TestNamedTest_Assert_AllTrueMultiple checks that a multi-assert body where
// every check holds passes (sentinel 0).
func TestNamedTest_Assert_AllTrueMultiple(t *testing.T) {
	result := runInlineTestsOnSource(t, assertFixtureSource(
		`  assert add_one(1) == 2;
  assert add_one(3) == 4`))

	if result.FailedTests != 0 {
		t.Errorf("expected 0 failures, got %d; first error: %s",
			result.FailedTests, firstFailureError(result))
	}
	if result.PassedTests != 1 {
		t.Errorf("expected 1 passing test, got %d", result.PassedTests)
	}
}

// TestNamedTest_Assert_WithLetBinding checks that `let` bindings still thread
// correctly around the sentinel chain.
func TestNamedTest_Assert_WithLetBinding(t *testing.T) {
	result := runInlineTestsOnSource(t, assertFixtureSource(
		`  let x = add_one(1);
  assert x == 2`))

	if result.FailedTests != 0 {
		t.Errorf("expected 0 failures, got %d; first error: %s",
			result.FailedTests, firstFailureError(result))
	}
	if result.PassedTests != 1 {
		t.Errorf("expected 1 passing test, got %d", result.PassedTests)
	}
}

// TestNamedTest_Assert_LetBetweenAsserts checks assert / let / assert ordering.
func TestNamedTest_Assert_LetBetweenAsserts(t *testing.T) {
	result := runInlineTestsOnSource(t, assertFixtureSource(
		`  assert add_one(1) == 2;
  let x = add_one(3);
  assert x == 4`))

	if result.FailedTests != 0 {
		t.Errorf("expected 0 failures, got %d; first error: %s",
			result.FailedTests, firstFailureError(result))
	}
	if result.PassedTests != 1 {
		t.Errorf("expected 1 passing test, got %d", result.PassedTests)
	}
}

// TestNamedTest_Assert_FinalBareExprInAssertBody covers the mixed shape: a
// trailing bare bool expression in a body that also contains an assert. It is a
// check too, so a false one must fail.
func TestNamedTest_Assert_FinalBareExprInAssertBody(t *testing.T) {
	pass := runInlineTestsOnSource(t, assertFixtureSource(
		`  assert add_one(1) == 2;
  add_one(3) == 4`))
	if pass.PassedTests != 1 || pass.FailedTests != 0 {
		t.Errorf("all-true mixed body: expected 1 passed/0 failed, got %d/%d; error: %s",
			pass.PassedTests, pass.FailedTests, firstFailureError(pass))
	}

	fail := runInlineTestsOnSource(t, assertFixtureSource(
		`  assert add_one(1) == 2;
  add_one(3) == 99`))
	if fail.FailedTests != 1 {
		t.Errorf("false final bare expr: expected 1 failure, got %d (passed=%d)",
			fail.FailedTests, fail.PassedTests)
	}
}

// TestNamedTest_AssertFree_LegacyPathUnchanged is the stays-green guard (B3):
// an assert-free body must keep the byte-identical legacy bool path.
func TestNamedTest_AssertFree_LegacyPathUnchanged(t *testing.T) {
	result := runInlineTestsOnSource(t, assertFixtureSource(`  add_one(1) == 2`))
	if result.PassedTests != 1 || result.FailedTests != 0 {
		t.Errorf("assert-free control: expected 1 passed/0 failed, got %d/%d; error: %s",
			result.PassedTests, result.FailedTests, firstFailureError(result))
	}

	// And the fold must be byte-identical to the legacy FoldBodyExprs output.
	exprs := []ast.Expr{
		&ast.Let{Name: "x", Value: &ast.Literal{Kind: ast.IntLit, Value: int64(1)}},
		&ast.BinaryOp{
			Op:    "==",
			Left:  &ast.Identifier{Name: "x"},
			Right: &ast.Literal{Kind: ast.IntLit, Value: int64(1)},
		},
	}
	folded, checks := FoldTestBody(exprs)
	if len(checks) != 0 {
		t.Errorf("assert-free body should produce no CheckInfo, got %d", len(checks))
	}
	legacy := FoldBodyExprs(exprs)
	if got, want := PrintAILANGSource(folded), PrintAILANGSource(legacy); got != want {
		t.Errorf("assert-free fold diverged from legacy path:\n got: %s\nwant: %s", got, want)
	}
}

// TestFoldTestBody_SentinelShape pins the exact lowered source for a two-assert
// body. It is the unit-level counterpart to the end-to-end tests above: if the
// short-circuit structure regresses to a per-node lowering (each assert lowered
// in isolation, non-final ones swallowed by `let _seq = ... in ...`), this goes
// red immediately and names the shape that changed.
func TestFoldTestBody_SentinelShape(t *testing.T) {
	mkAssert := func(name string) ast.Expr {
		return &ast.AssertStmt{Condition: &ast.Identifier{Name: name}}
	}
	exprs := []ast.Expr{mkAssert("a"), mkAssert("b")}

	folded, checks := FoldTestBody(exprs)
	got := PrintAILANGSource(folded)
	want := "if a then if b then 0 else 2 else 1"
	if got != want {
		t.Errorf("lowered shape mismatch:\n got: %s\nwant: %s", got, want)
	}
	if len(checks) != 2 {
		t.Fatalf("expected 2 checks, got %d", len(checks))
	}
	if checks[0].Ordinal != 1 || checks[1].Ordinal != 2 {
		t.Errorf("expected ordinals 1,2; got %d,%d", checks[0].Ordinal, checks[1].Ordinal)
	}
	if checks[0].Source != "a" || checks[1].Source != "b" {
		t.Errorf("expected check sources a,b; got %q,%q", checks[0].Source, checks[1].Source)
	}
	// The sentinel must never be negative — negative literals would need a unary
	// minus in `else` position, which the printer does not bracket.
	if strings.Contains(got, "-") {
		t.Errorf("lowered sentinel contains a negative literal: %s", got)
	}
}
