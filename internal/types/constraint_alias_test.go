package types

import "testing"

// TestExpandAliasesForConstraint pins what alias expansion of a constraint
// type sees through and what it leaves nominal (#2247 review).
func TestExpandAliasesForConstraint(t *testing.T) {
	a := &TVar2{Name: "a", Kind: Star}
	idCon := &TCon{Name: "Id", Module: "m"}
	scoreCon := &TCon{Name: "Score", Module: "m"}
	pointCon := &TCon{Name: "Point", Module: "m"}
	refCon := &TCon{Name: "Ref", Module: "m"}
	u := NewUnifierWithAliasesAndParams(map[string]Type{
		idCon.KeyOf():    TInt,                                                                   // type Id[a] = int
		scoreCon.KeyOf(): TInt,                                                                   // type Score = int
		refCon.KeyOf():   &TApp{Constructor: idCon, Args: []Type{&TVar2{Name: "b", Kind: Star}}}, // type Ref[b] = Id[b]
		pointCon.KeyOf(): &TRecord{Fields: map[string]Type{"x": TInt}},                           // type Point = {x: int}
	}, map[string][]string{idCon.KeyOf(): {"a"}, refCon.KeyOf(): {"b"}})

	idA := &TApp{Constructor: idCon, Args: []Type{a}}
	cases := []struct {
		name string
		in   Type
		want Type
	}{
		{"applied alias of a primitive", idA, TInt},
		{"nullary alias of a primitive", scoreCon, TInt},
		{"alias chain", &TApp{Constructor: refCon, Args: []Type{a}}, TInt},
		{"inside a list", &TList{Element: idA}, &TList{Element: TInt}},
		{"inside a type application", optionOf(idA), optionOf(TInt)},
		{"inside a tuple", &TTuple{Elements: []Type{idA, a}}, &TTuple{Elements: []Type{TInt, a}}},
		// A record alias stays nominal: its `deriving (Eq)` instance is keyed by
		// the alias, and expanding it would re-route Eq to the bare-name record
		// path (instances_eq_synth.go), losing the defining module.
		{"record alias stays nominal", pointCon, pointCon},
		{"non-alias ADT untouched", optionOf(a), optionOf(a)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := expandAliasesForConstraint(u, c.in, 0)
			if !got.Equals(c.want) {
				t.Fatalf("expand(%s) = %s, want %s", c.in, got, c.want)
			}
		})
	}
}
