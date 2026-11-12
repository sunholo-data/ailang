package types

import (
	"errors"
	"testing"
)

// TestReduceConstraintsEveryClass pins #2247: outside Eq, a constraint that is
// not on a bare type variable must be solved by an instance or rejected. Before
// the fix every non-Eq constraint "passed through unchanged" and the top-level
// boundary then dropped it, so Num[Box[α]] from `g(7)` never failed.
func TestReduceConstraintsEveryClass(t *testing.T) {
	env := LoadBuiltinInstances()
	alpha := &TVar2{Name: "α1", Kind: Star}
	box := func(t Type) Type { return &TApp{Constructor: &TCon{Name: "Box", Module: "m"}, Args: []Type{t}} }
	cases := []struct {
		name     string
		class    string
		typ      Type
		residual int  // constraints left (deferred to a caller)
		ok       bool // false: must be a missing instance
	}{
		{"Num on a bare variable is deferred", "Num", alpha, 1, true},
		{"Ord on a bare variable is deferred", "Ord", alpha, 1, true},
		{"Fractional on a bare variable is deferred", "Fractional", alpha, 1, true},
		{"Show on a bare variable is deferred", "Show", alpha, 1, true},
		{"Num[int] is solved", "Num", TInt, 0, true},
		{"Ord[string] is solved", "Ord", TString, 0, true},
		{"Num[Box[α]]", "Num", box(alpha), 0, false},
		{"Num[[α]]", "Num", listOf(alpha), 0, false},
		{"Num[Option[α]]", "Num", optionOf(alpha), 0, false},
		{"Fractional[Result[α, int]]", "Fractional", resultOf(alpha, TInt), 0, false},
		{"Ord[[α]]", "Ord", listOf(alpha), 0, false},
		{"Ord[Box[α]]", "Ord", box(alpha), 0, false},
		{"Show[Box[α]]", "Show", box(alpha), 0, false},
		{"Num[(α, int)]", "Num", &TTuple{Elements: []Type{alpha, TInt}}, 0, false},
		// Eq keeps its structural decomposition.
		{"Eq[Option[α]] decomposes", "Eq", optionOf(alpha), 1, true},
		{"Eq[Box[α]] has no instance", "Eq", box(alpha), 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := env.ReduceConstraints([]ClassConstraint{{Class: c.class, Type: c.typ, Path: []string{"here"}}})
			if !c.ok {
				var missing *MissingInstanceError
				if !errors.As(err, &missing) {
					t.Fatalf("%s[%s]: want a missing instance, got out=%v err=%v", c.class, c.typ, out, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("%s[%s]: %v", c.class, c.typ, err)
			}
			if len(out) != c.residual {
				t.Fatalf("%s[%s]: want %d residual, got %v", c.class, c.typ, c.residual, out)
			}
		})
	}
}
