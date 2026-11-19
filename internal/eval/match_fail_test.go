package eval

import (
	"errors"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
)

// colorMatch is `match <Blue> { Red => 1, Green => 2 }` positioned at
// p1.ail:3:38 — no arm matches.
func colorMatch() *core.Match {
	m := &core.Match{
		Scrutinee: &core.Var{Name: "c"},
		Arms: []core.MatchArm{
			{Pattern: &core.ConstructorPattern{Name: "Red"}, Body: &core.Lit{Kind: core.IntLit, Value: int64(1)}},
			{Pattern: &core.ConstructorPattern{Name: "Green"}, Body: &core.Lit{Kind: core.IntLit, Value: int64(2)}},
		},
	}
	m.OrigSpan = ast.Pos{File: "p1.ail", Line: 3, Column: 38}
	return m
}

// RT002 on both evaluator paths — the linear arm loop and the AILANG_DTREE
// decision tree — with the match's position and the scrutinee's tag only
// (M-V1-MATCH-EXHAUSTIVENESS M0).
func TestMatchFailure_EvaluatorPaths(t *testing.T) {
	for _, dtree := range []string{"0", "1"} {
		t.Run("AILANG_DTREE="+dtree, func(t *testing.T) {
			t.Setenv("AILANG_DTREE", dtree)
			ev := NewCoreEvaluator()
			ev.env.Set("c", &TaggedValue{TypeName: "Color", CtorName: "Blue"})
			_, err := ev.evalCoreMatch(colorMatch(), false)
			var mf *ailerrors.MatchFailureError
			if !errors.As(err, &mf) {
				t.Fatalf("want RT002, got %v", err)
			}
			want := "RT002: no arm matched in match at p1.ail:3:38 (scrutinee constructor: Blue)"
			if mf.Error() != want {
				t.Errorf("got %q, want %q", mf.Error(), want)
			}
		})
	}
}

func TestMatchFailureShapes(t *testing.T) {
	cases := []struct {
		v    Value
		want string
	}{
		{&TaggedValue{CtorName: "None"}, "RT002: no arm matched in match at f.ail:1:2 (scrutinee constructor: None)"},
		{&ListValue{}, "RT002: no arm matched in match at f.ail:1:2 (scrutinee: empty list)"},
		{&StringValue{Value: "secret"}, "RT002: no arm matched in match at f.ail:1:2 (scrutinee: string)"},
		{nil, "RT002: no arm matched in match at f.ail:1:2"},
	}
	for _, tc := range cases {
		if got := matchFailure(ast.Pos{File: "f.ail", Line: 1, Column: 2}, tc.v).Error(); got != tc.want {
			t.Errorf("got %q, want %q", got, tc.want)
		}
	}
	// No position: left empty for the caller; a set position is never
	// overwritten by an outer match.
	err := matchFailure(ast.Pos{}, &ListValue{})
	attachMatchPos(err, ast.Pos{File: "outer.ail", Line: 9, Column: 1})
	attachMatchPos(err, ast.Pos{File: "outermost.ail", Line: 1, Column: 1})
	if got := err.Error(); got != "RT002: no arm matched in match at outer.ail:9:1 (scrutinee: empty list)" {
		t.Errorf("got %q", got)
	}
	if p := matchPos(ast.Pos{}, ast.Pos{File: "c.ail", Line: 4}); p.Line != 4 {
		t.Errorf("matchPos ignored the Core span: %+v", p)
	}
}
