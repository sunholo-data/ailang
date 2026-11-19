package types

import (
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/types/exhaust"
)

// Unit tests of VerifyMatchExhaustiveness on hand-built Core: the fail-closed
// rules the pipeline cannot reach from source (an ADT with no universe, a
// stamp that disagrees with the scrutinee's type), the Exhaustive flag, error
// ordering and the structured report. Source-level coverage is in
// internal/pipeline (modules) and internal/vm (property test against both
// engines).

func matchOn(id uint64, line int, scrutType Type, tc *CoreTypeChecker, arms ...core.MatchArm) *core.Match {
	scrut := &core.Var{CoreNode: core.CoreNode{NodeID: id + 1000}, Name: "x"}
	tc.CoreTI.Set(scrut.ID(), scrutType)
	return &core.Match{
		CoreNode:  core.CoreNode{NodeID: id, OrigSpan: ast.Pos{File: "m.ail", Line: line, Column: 3}},
		Scrutinee: scrut,
		Arms:      arms,
	}
}

func ctorArm(module, typ, name string, guard bool) core.MatchArm {
	arm := core.MatchArm{Pattern: &core.ConstructorPattern{Name: name, TypeName: typ, TypeModule: module}, Body: &core.Lit{}}
	if guard {
		arm.Guard = &core.Lit{}
	}
	return arm
}

func wildArm() core.MatchArm {
	return core.MatchArm{Pattern: &core.WildcardPattern{}, Body: &core.Lit{}}
}

func newMatchTC() *CoreTypeChecker {
	tc := NewCoreTypeChecker()
	tc.CoreTI = CoreTypeInfo{}
	tc.SetMatchUniverse(MatchUniverse{
		NominalKey("m", "Color"): {{Name: "Blue"}, {Name: "Green"}, {Name: "Red"}},
		NominalKey("n", "Color"): {{Name: "Red"}, {Name: "Violet"}},
	})
	return tc
}

func TestVerifyMatchExhaustivenessVerdicts(t *testing.T) {
	tc := newMatchTC()
	color := &TCon{Name: "Color", Module: "m"}
	full := matchOn(1, 10, color, tc, ctorArm("m", "Color", "Red", false), ctorArm("m", "Color", "Green", false), ctorArm("m", "Color", "Blue", false))
	partial := matchOn(2, 20, color, tc, ctorArm("m", "Color", "Red", false), ctorArm("m", "Color", "Green", true))
	// n's Color has Red and Violet only: m's Blue/Green must not leak in.
	other := matchOn(3, 5, &TCon{Name: "Color", Module: "n"}, tc, ctorArm("n", "Color", "Red", false))

	warns, err := tc.VerifyMatchExhaustiveness(full, partial, other)
	if len(warns) != 0 {
		t.Errorf("unexpected warnings %v", warns)
	}
	var errs MatchErrors
	if !errors.As(err, &errs) || len(errs) != 2 {
		t.Fatalf("want 2 match errors, got %v", err)
	}
	// Sorted by position, whatever the walk order.
	if errs[0].Pos.Line != 5 || !strings.HasSuffix(errs[0].Message, "no arm matches: Violet") {
		t.Errorf("first error %q at line %d, want Violet at 5", errs[0].Message, errs[0].Pos.Line)
	}
	if errs[1].Pos.Line != 20 || !strings.HasSuffix(errs[1].Message, "no arm matches: Blue, Green") {
		t.Errorf("second error %q at line %d, want Blue, Green at 20", errs[1].Message, errs[1].Pos.Line)
	}
	if !strings.Contains(errs[1].Error(), "arms with `if` guards do not count") {
		t.Errorf("a match with a guarded arm must say guards do not count:\n%s", errs[1].Error())
	}
	if !full.Exhaustive || partial.Exhaustive || other.Exhaustive {
		t.Errorf("Exhaustive flags = %v %v %v, want true false false", full.Exhaustive, partial.Exhaustive, other.Exhaustive)
	}
	rep, ok := ailerrors.AsReport(err)
	if !ok || rep.Code != TC_MATCH_002 || rep.Span == nil || rep.Span.Start.Line != 5 || rep.Fix == nil ||
		rep.Fix.Suggestion != "add the missing arm: Violet => ..." {
		t.Errorf("report = %+v", rep)
	}

	restore := SetMatchExhaustivenessErrorsForTest(false)
	defer restore()
	if _, err := tc.VerifyMatchExhaustiveness(partial); err != nil {
		t.Errorf("with errors switched off: %v", err)
	}
}

// Fail closed: a constructor whose type has no universe, or whose stamp
// disagrees with the scrutinee's type, is an error — unless an unguarded
// catch-all arm closes the match, which is exhaustive whatever the rest says.
func TestVerifyMatchExhaustivenessFailsClosed(t *testing.T) {
	tc := newMatchTC()
	unknown := &TCon{Name: "Shade", Module: "q"}
	cases := []struct {
		name    string
		match   *core.Match
		wantErr string
	}{
		{"no universe", matchOn(1, 1, unknown, tc, ctorArm("q", "Shade", "Dark", false)), "constructor Dark is not a known constructor of Shade"},
		{"stamp disagrees with the scrutinee", matchOn(2, 2, &TCon{Name: "Color", Module: "n"}, tc, ctorArm("m", "Color", "Red", false)), "compiler bug"},
		{"no universe, catch-all", matchOn(3, 3, unknown, tc, ctorArm("q", "Shade", "Dark", false), wildArm()), ""},
		{"no universe, guarded catch-all", matchOn(4, 4, unknown, tc, ctorArm("q", "Shade", "Dark", false),
			core.MatchArm{Pattern: &core.WildcardPattern{}, Guard: &core.Lit{}, Body: &core.Lit{}}), "cannot be checked"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := tc.VerifyMatchExhaustiveness(c.match)
			if c.wantErr == "" {
				if err != nil || !c.match.Exhaustive {
					t.Fatalf("want accepted and Exhaustive, got %v (Exhaustive=%v)", err, c.match.Exhaustive)
				}
				return
			}
			var me *MatchError
			if !errors.As(err, &me) || me.Code != TC_MATCH_002 || !strings.Contains(me.Message, c.wantErr) {
				t.Fatalf("want TC_MATCH_002 containing %q, got %v", c.wantErr, err)
			}
		})
	}
}

// TC_MATCH_006: a constructor pattern with the wrong number of arguments is
// located at the pattern and names the right shape, even under a catch-all.
func TestVerifyMatchExhaustivenessArity(t *testing.T) {
	tc := newMatchTC()
	tc.SetMatchUniverse(MatchUniverse{NominalKey("m", "U"): {{Name: "C", Arity: 2}, {Name: "E"}}})
	p := &core.ConstructorPattern{Name: "C", TypeName: "U", TypeModule: "m", Args: []core.CorePattern{&core.VarPattern{Name: "n"}},
		Pos: ast.Pos{Line: 7, Column: 9}}
	m := matchOn(1, 7, &TCon{Name: "U", Module: "m"}, tc, core.MatchArm{Pattern: p, Body: &core.Lit{}}, wildArm())
	_, err := tc.VerifyMatchExhaustiveness(m)
	var me *MatchError
	if !errors.As(err, &me) || me.Code != TC_MATCH_006 || me.Pos.Column != 9 || me.Pos.File != "m.ail" ||
		me.Suggestion != "write C(_, _)" {
		t.Fatalf("got %v (%+v)", err, me)
	}
}

// TC_MATCH_003 names the arm, and walks into every Core form that can hold a
// match (here a lambda inside a let inside a letrec).
func TestVerifyMatchExhaustivenessRedundantNested(t *testing.T) {
	tc := newMatchTC()
	color := &TCon{Name: "Color", Module: "m"}
	inner := matchOn(1, 3, color, tc, ctorArm("m", "Color", "Red", false), wildArm(), ctorArm("m", "Color", "Blue", false))
	expr := &core.LetRec{Bindings: []core.RecBinding{{Name: "f", Value: &core.Lambda{Body: &core.Let{Name: "y", Value: &core.Lit{}, Body: inner}}}},
		Body: &core.Lit{}}
	warns, err := tc.VerifyMatchExhaustiveness(expr)
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 1 || warns[0].Arm != 3 || warns[0].Pattern != "Blue" || warns[0].Code() != TC_MATCH_003 {
		t.Fatalf("warnings %+v", warns)
	}
	if got := exhaust.Render(exhaust.Ctor{Name: "Blue"}); got != "Blue" {
		t.Fatal(got)
	}
}
