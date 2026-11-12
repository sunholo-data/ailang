package types

import (
	"fmt"
	"testing"
)

// TestExpandConstraintAliases pins when a constraint is judged through its
// aliases and when it keeps its written type (#2247 reviews 1-3).
func TestExpandConstraintAliases(t *testing.T) {
	a := &TVar2{Name: "a", Kind: Star}
	b := &TVar2{Name: "b", Kind: Star}
	con := func(name string) *TCon { return &TCon{Name: name, Module: "m"} }
	idCon, scoreCon, pointCon, refCon := con("Id"), con("Score"), con("Point"), con("Ref")
	countCon, sameCon, wCon, lCon := con("Count"), con("Same"), con("W"), con("L")
	tc := NewCoreTypeChecker()
	tc.aliasEnv = map[string]Type{
		idCon.KeyOf():    TInt,                                                       // type Id[a] = int
		scoreCon.KeyOf(): TInt,                                                       // type Score = int
		refCon.KeyOf():   &TApp{Constructor: idCon, Args: []Type{b}},                 // type Ref[b] = Id[b]
		pointCon.KeyOf(): &TRecord{Fields: map[string]Type{"x": TInt}},               // type Point = {x: int}
		countCon.KeyOf(): &TVar2{Name: "integer", Kind: Star},                        // type Count = integer (a typo)
		sameCon.KeyOf():  b,                                                          // type Same[b] = b
		wCon.KeyOf():     &TRecord{Fields: map[string]Type{"x": TInt}},               // type W = W({x: int}), a newtype
		lCon.KeyOf():     &TList{Element: &TApp{Constructor: lCon, Args: []Type{b}}}, // type L[b] = [L[b]]
	}
	tc.aliasParams = map[string][]string{idCon.KeyOf(): {"a"}, refCon.KeyOf(): {"b"}, sameCon.KeyOf(): {"b"}, lCon.KeyOf(): {"b"}}
	tc.newtypes = map[string]bool{wCon.KeyOf(): true}

	app := func(c *TCon, args ...Type) Type { return &TApp{Constructor: c, Args: args} }
	cases := []struct {
		name  string
		class string
		in    Type
		want  Type
	}{
		// Expanded: the expansion solves the constraint outright.
		{"applied alias of a primitive", "Num", app(idCon, a), TInt},
		{"nullary alias of a primitive", "Num", scoreCon, TInt},
		{"alias chain", "Ord", app(refCon, a), TInt},
		{"identity alias at a ground type", "Num", app(sameCon, TFloat), TFloat},
		{"inside a list (Eq)", "Eq", &TList{Element: app(idCon, a)}, &TList{Element: TInt}},
		{"inside a type application (Eq)", "Eq", optionOf(app(idCon, a)), optionOf(TInt)},
		{"inside a tuple (Eq)", "Eq", &TTuple{Elements: []Type{app(idCon, a), TString}}, &TTuple{Elements: []Type{TInt, TString}}},
		// Kept as written: the expansion would only defer the constraint. With
		// Same[b] = b, Num[Same[b]] would become Num[b]; the `+` node keeps the
		// type Same[b] and lowering would pick the Int implementation.
		{"identity alias at a variable", "Num", app(sameCon, a), app(sameCon, a)},
		{"alias body with an unbound variable", "Num", countCon, countCon},
		{"Eq leaving a residual", "Eq", optionOf(app(sameCon, a)), optionOf(app(sameCon, a))},
		// Kept as written: nominal types.
		{"record alias stays nominal", "Eq", pointCon, pointCon},
		{"newtype stays nominal", "Eq", wCon, wCon},
		// Kept as written: a self-referential alias is unsolvable expanded.
		{"self-referential alias", "Eq", app(lCon, TInt), app(lCon, TInt)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := tc.expandConstraintAliases([]ClassConstraint{{Class: c.class, Type: c.in}})
			if got := out[0].Type; !got.Equals(c.want) {
				t.Fatalf("%s[%s] -> %s, want %s", c.class, c.in, got, c.want)
			}
		})
	}
}

// TestAliasExpansionIsBounded pins the work done expanding one constraint type
// (#2285 fourth review). A branching self-referential alias
// (`type T2[a] = (T2[a], T2[a])`) was unfolded to a fixed depth at every
// occurrence, k^32 nodes: `check` reached 65 GB. The expansion must stop at
// the alias's first recurrence, and a legitimately huge expansion (a nest of
// pair aliases) must stop at the node budget, both reporting !ok.
func TestAliasExpansionIsBounded(t *testing.T) {
	con := func(name string) *TCon { return &TCon{Name: name, Module: "m"} }
	b := &TVar2{Name: "b", Kind: Star}
	t2, t3 := con("T2"), con("T3")
	aliases := map[string]Type{
		t2.KeyOf(): &TTuple{Elements: []Type{&TApp{Constructor: t2, Args: []Type{b}}, &TApp{Constructor: t2, Args: []Type{b}}}},
		t3.KeyOf(): optionOf(&TTuple{Elements: []Type{&TApp{Constructor: t3, Args: []Type{b}}, &TApp{Constructor: t3, Args: []Type{b}}, &TApp{Constructor: t3, Args: []Type{b}}}}),
	}
	params := map[string][]string{t2.KeyOf(): {"b"}, t3.KeyOf(): {"b"}}
	// type P1 = (int, int); type Pn = (Pn-1, Pn-1): 2^20 leaves at P20.
	var prev Type = TInt
	var p20 *TCon
	for i := 1; i <= 20; i++ {
		p := con(fmt.Sprintf("P%d", i))
		aliases[p.KeyOf()] = &TTuple{Elements: []Type{prev, prev}}
		prev, p20 = p, p
	}
	u := NewUnifierWithAliasesAndParams(aliases, params)

	cases := []struct {
		name     string
		in       Type
		maxNodes int
	}{
		{"branching self-reference", &TApp{Constructor: t2, Args: []Type{TInt}}, 16},
		{"branching self-reference under Option", &TApp{Constructor: t3, Args: []Type{TInt}}, 16},
		{"2^20-leaf pair nest", p20, aliasExpandNodeBudget + 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			x := &aliasExpander{u: u, onPath: map[string]bool{}, budget: aliasExpandNodeBudget}
			if _, ok := x.expand(c.in); ok {
				t.Fatalf("expand(%s) must stop (judge as written), got ok", c.in)
			}
			if used := aliasExpandNodeBudget - x.budget; used > c.maxNodes {
				t.Fatalf("expand(%s) visited %d nodes, want <= %d", c.in, used, c.maxNodes)
			}
		})
	}
}
