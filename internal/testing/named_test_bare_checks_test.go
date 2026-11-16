package testing

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

// Regression suite for #604: in a named test body only the LAST expression was
// graded. Earlier bare expressions were bound to a dead `let _seq = ... in`
// and discarded, so `test "x" { add_one(1) == 99; add_one(1) == 2 }` reported
// "All tests passed!" with exit 0.
//
// The rule now: every TOP-LEVEL bare expression of a named test body is a
// check, numbered together with asserts and lowered through #590's sentinel
// path. Design: design_docs/planned/v0_33_1/m-named-test-body-check-semantics.md
// (or its implemented/ successor). Nested blocks inside a check are NOT covered
// — that residual is #614, pinned by testdata/assert/nested_block_residual.ail.

// TestNamedTest_BareNonFinal_FalseFails is the #604 regression in its filed
// shape: the false check is first, the true check is final.
func TestNamedTest_BareNonFinal_FalseFails(t *testing.T) {
	result := runInlineTestsOnSource(t, assertFixtureSource(
		`  add_one(1) == 99;
  add_one(1) == 2`))

	if result.FailedTests != 1 || result.PassedTests != 0 {
		t.Fatalf("non-final false bare check must fail the test: got %d passed/%d failed",
			result.PassedTests, result.FailedTests)
	}
	errText := firstFailureError(result)
	if !strings.Contains(errText, "check 1 failed: `(add_one(1) == 99)`") {
		t.Errorf("expected the failing bare check reported as check 1, got: %s", errText)
	}
	// It is a bare expression: quoting it back as `assert ...` would show source
	// the user never wrote.
	if strings.Contains(errText, "assert (") {
		t.Errorf("bare check must not be quoted as an assert, got: %s", errText)
	}
	// The position is the user's, captured before the round-trip.
	if !strings.Contains(errText, "test_input.ail:") || strings.Contains(errText, "_namedtest_body_") {
		t.Errorf("expected the user's source position, got: %s", errText)
	}
}

// TestNamedTest_BareNonFinal_ControlUnchanged is the discriminating control:
// the same body with the order swapped fails too, now naming check 2.
func TestNamedTest_BareNonFinal_ControlUnchanged(t *testing.T) {
	result := runInlineTestsOnSource(t, assertFixtureSource(
		`  add_one(1) == 2;
  add_one(1) == 99`))

	if result.FailedTests != 1 || result.PassedTests != 0 {
		t.Fatalf("final false bare check must fail the test: got %d passed/%d failed",
			result.PassedTests, result.FailedTests)
	}
	if errText := firstFailureError(result); !strings.Contains(errText, "check 2 failed") {
		t.Errorf("expected check 2 to be reported, got: %s", errText)
	}
}

// TestNamedTest_BareNonFinal_ShortCircuits: a failing check stops evaluation,
// so a later check that would crash at runtime is never reached.
func TestNamedTest_BareNonFinal_ShortCircuits(t *testing.T) {
	result := runInlineTestsOnSource(t, assertFixtureSource(
		`  add_one(1) == 99;
  1 / 0 == 0`))

	if result.FailedTests != 1 {
		t.Fatalf("expected 1 failure, got %d passed/%d failed", result.PassedTests, result.FailedTests)
	}
	if errText := firstFailureError(result); !strings.Contains(errText, "check 1 failed") {
		t.Errorf("expected check 1 to fail before check 2 is evaluated, got: %s", errText)
	}
}

// TestFoldTestBody_BareChecksSentinelShape pins the lowered source for a body
// of two bare checks: identical in shape to the two-assert body
// (TestFoldTestBody_SentinelShape), with IsAssert false on both checks.
func TestFoldTestBody_BareChecksSentinelShape(t *testing.T) {
	exprs := []ast.Expr{&ast.Identifier{Name: "a"}, &ast.Identifier{Name: "b"}}

	folded, checks := FoldTestBody(exprs)
	if got, want := PrintAILANGSource(folded), "if a then if b then 0 else 2 else 1"; got != want {
		t.Errorf("lowered shape mismatch:\n got: %s\nwant: %s", got, want)
	}
	if len(checks) != 2 {
		t.Fatalf("expected 2 checks, got %d", len(checks))
	}
	for i, c := range checks {
		if c.Ordinal != i+1 || c.IsAssert {
			t.Errorf("check %d: got ordinal %d IsAssert %v; want ordinal %d, IsAssert false",
				i, c.Ordinal, c.IsAssert, i+1)
		}
	}
	if strings.Contains(PrintAILANGSource(folded), "_seq") {
		t.Errorf("a bare non-final expression must not be bound to a dead _seq: %s", PrintAILANGSource(folded))
	}
}

// TestFoldTestBody_MixedOrdinals: bare checks and asserts share one
// left-to-right numbering; lets consume no ordinal.
func TestFoldTestBody_MixedOrdinals(t *testing.T) {
	exprs := []ast.Expr{
		&ast.Identifier{Name: "a"},
		&ast.AssertStmt{Condition: &ast.Identifier{Name: "b"}},
		&ast.Let{Name: "x", Value: &ast.Identifier{Name: "a"}},
		&ast.Identifier{Name: "c"},
	}

	folded, checks := FoldTestBody(exprs)
	if got, want := PrintAILANGSource(folded),
		"if a then if b then let x = a in\nif c then 0 else 3 else 2 else 1"; got != want {
		t.Errorf("lowered shape mismatch:\n got: %s\nwant: %s", got, want)
	}
	type row struct {
		ord      int
		src      string
		isAssert bool
	}
	var got []row
	for _, c := range checks {
		got = append(got, row{c.Ordinal, c.Source, c.IsAssert})
	}
	want := []row{{1, "a", false}, {2, "b", true}, {3, "c", false}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("checks = %+v, want %+v", got, want)
	}
}

// TestFoldTestBody_Trigger: the sentinel path is taken iff the body has an
// assert or a bare non-final expression; lets never trigger it.
func TestFoldTestBody_Trigger(t *testing.T) {
	id := func(n string) ast.Expr { return &ast.Identifier{Name: n} }
	let := func(n string) ast.Expr { return &ast.Let{Name: n, Value: id("v")} }
	cases := []struct {
		name     string
		exprs    []ast.Expr
		sentinel bool
	}{
		{"single bare", []ast.Expr{id("a")}, false},
		{"lets + final", []ast.Expr{let("x"), let("y"), id("a")}, false},
		{"bare non-final", []ast.Expr{id("a"), id("b")}, true},
		{"bare non-final after a let", []ast.Expr{let("x"), id("a"), id("b")}, true},
		{"assert only", []ast.Expr{&ast.AssertStmt{Condition: id("a")}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, checks := FoldTestBody(tc.exprs)
			if got := len(checks) > 0; got != tc.sentinel {
				t.Errorf("sentinel path = %v, want %v (%d checks)", got, tc.sentinel, len(checks))
			}
		})
	}
}

// TestEngineParity_BareCheckFixtures: the batched evaluator, the per-body path
// and the bytecode VM all consume FoldTestBody, so each #604 fixture must
// report the same outcome on both engines — a vacuous pass on either one is
// the bug again.
func TestEngineParity_BareCheckFixtures(t *testing.T) {
	for _, fixture := range []string{
		"bare_first_fails.ail", "bare_second_fails.ail", "bare_all_pass.ail",
		"bare_mixed_assert.ail", "bare_nonbool_nonfinal.ail",
	} {
		t.Run(fixture, func(t *testing.T) {
			src, err := os.ReadFile(filepath.Join("testdata", "assert", fixture))
			if err != nil {
				t.Fatal(err)
			}
			path := writeEngineSource(t, fixture, string(src))
			ev := runEngine(t, path, false, false, 0)
			bc := runEngine(t, path, true, false, 0)
			if got, want := outcomes(bc), outcomes(ev); !reflect.DeepEqual(got, want) {
				t.Errorf("bytecode report differs from evaluator report\nbytecode:  %+v\nevaluator: %+v", got, want)
			}
			if strings.HasPrefix(fixture, "bare_first_fails") && bc.FailedTests != 1 {
				t.Errorf("bytecode engine passed the #604 shape: %+v", bc.Tests)
			}
		})
	}
}
