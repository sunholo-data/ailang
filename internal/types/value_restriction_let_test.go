package types

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// isCoreValue follows ML's non-expansive rule for a let nested in a binding's
// value (#2388 keeps such lets nested instead of floating them): the let is
// a value when its bound value and its body both are.
func TestIsCoreValue_NestedLet(t *testing.T) {
	lit := &core.Lit{Kind: core.IntLit, Value: int64(1)}
	lam := &core.Lambda{Params: []string{"x"}, Body: &core.Var{Name: "x"}}
	call := &core.App{Func: &core.Var{Name: "mk"}, Args: []core.CoreExpr{lit}}
	cases := []struct {
		name string
		e    core.CoreExpr
		want bool
	}{
		{"let lit in lambda", &core.Let{Name: "k", Value: lit, Body: lam}, true},
		{"let lambda in let lit in lambda", &core.Let{Name: "a", Value: lam, Body: &core.Let{Name: "k", Value: lit, Body: lam}}, true},
		{"let call in lambda", &core.Let{Name: "c", Value: call, Body: lam}, false},
		{"let lit in call", &core.Let{Name: "k", Value: lit, Body: call}, false},
		{"letrec lambda in lambda", &core.LetRec{Bindings: []core.RecBinding{{Name: "g", Value: lam}}, Body: lam}, true},
		{"letrec call in lambda", &core.LetRec{Bindings: []core.RecBinding{{Name: "g", Value: call}}, Body: lam}, false},
		{"letrec lambda in call", &core.LetRec{Bindings: []core.RecBinding{{Name: "g", Value: lam}}, Body: call}, false},
	}
	for _, c := range cases {
		if got := isCoreValue(c.e); got != c.want {
			t.Errorf("%s: isCoreValue = %v, want %v", c.name, got, c.want)
		}
	}
}
