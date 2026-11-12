package types

import "testing"

// TestExpandAliasesForConstraint pins what alias expansion of a constraint
// type sees through and what it refuses (#2247 review).
func TestExpandAliasesForConstraint(t *testing.T) {
	a := &TVar2{Name: "a", Kind: Star}
	b := &TVar2{Name: "b", Kind: Star}
	con := func(name string) *TCon { return &TCon{Name: name, Module: "m"} }
	idCon, scoreCon, pointCon, refCon := con("Id"), con("Score"), con("Point"), con("Ref")
	countCon, any2Con, wCon, lCon := con("Count"), con("Any2"), con("W"), con("L")
	u := NewUnifierWithAliasesAndParams(map[string]Type{
		idCon.KeyOf():    TInt,                                                       // type Id[a] = int
		scoreCon.KeyOf(): TInt,                                                       // type Score = int
		refCon.KeyOf():   &TApp{Constructor: idCon, Args: []Type{b}},                 // type Ref[b] = Id[b]
		pointCon.KeyOf(): &TRecord{Fields: map[string]Type{"x": TInt}},               // type Point = {x: int}
		countCon.KeyOf(): &TVar2{Name: "integer", Kind: Star},                        // type Count = integer (a typo)
		any2Con.KeyOf():  &TVar2{Name: "c", Kind: Star},                              // type Any2[b] = c
		wCon.KeyOf():     &TRecord{Fields: map[string]Type{"x": TInt}},               // type W = W({x: int}), a newtype
		lCon.KeyOf():     &TList{Element: &TApp{Constructor: lCon, Args: []Type{b}}}, // type L[b] = [L[b]]
	}, map[string][]string{idCon.KeyOf(): {"a"}, refCon.KeyOf(): {"b"}, any2Con.KeyOf(): {"b"}, lCon.KeyOf(): {"b"}})
	u.SetNewtypes(map[string]bool{wCon.KeyOf(): true})

	idA := &TApp{Constructor: idCon, Args: []Type{a}}
	lInt := &TApp{Constructor: lCon, Args: []Type{TInt}}
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
		// A record alias expands to the record carrying its declared name, which
		// is how its `deriving (Eq)` resolves (instances_eq_synth.go).
		{"record alias", pointCon, &TRecord{Fields: map[string]Type{"x": TInt}, TypeName: "Point"}},
		{"non-alias ADT untouched", optionOf(a), optionOf(a)},
		// Refused: an alias body naming a variable it does not bind would turn a
		// ground constraint into a deferred (and dropped) one.
		{"unbound variable in a nullary alias body", countCon, countCon},
		{"unbound variable in an applied alias body", &TApp{Constructor: any2Con, Args: []Type{TInt}}, &TApp{Constructor: any2Con, Args: []Type{TInt}}},
		// Refused: a newtype is nominal.
		{"newtype stays nominal", wCon, wCon},
	}
	// A self-referential alias terminates (the diagnostic keeps the written
	// type: TestConstraintOnAppliedTypeIsResolved/self_referential_alias).
	if got := expandAliasesForConstraint(u, lInt, 0); got == nil {
		t.Fatal("self-referential alias: nil")
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
