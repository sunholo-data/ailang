package types

import (
	"strings"
	"testing"
)

func aliasApp(head string, args ...Type) Type {
	return &TApp{Constructor: &TCon{Name: head}, Args: args}
}

// Equivalence is decided only for the forms it compares; everything else is
// "not known equivalent", so unification proceeds as without the shortcut and
// decides the same outcome (#2363 review, round 2). Each case would be
// accepted, or decided differently, if the shortcut took the pair as
// equivalent.
func TestUnify_EquivalenceOnlyForComparedForms(t *testing.T) {
	a := &TVar2{Name: "a", Kind: Star}
	r := &RowVar{Name: "r", Kind: RecordRow}
	aliases := map[string]Type{
		// e23_fnret: function types are not compared, and these differ.
		"A": &TFunc2{Params: []Type{a}, Return: TInt},
		"B": &TFunc2{Params: []Type{a}, Return: TString},
		// Same name, different modules: nominally different types.
		"C1": &TTuple{Elements: []Type{a, &TCon{Name: "Box", Module: "m1"}}},
		"C2": &TTuple{Elements: []Type{a, &TCon{Name: "Box", Module: "m2"}}},
		// A record with a row is not compared.
		"R1": &TRecord{Fields: map[string]Type{"x": TInt}, Row: r},
		"R2": &TRecord{Fields: map[string]Type{"x": TInt}},
		// Cyclic aliases (rejected at their declaration, but never assumed equal).
		"Y1": &TTuple{Elements: []Type{&TCon{Name: "Y1"}, TInt}},
		"Y2": &TTuple{Elements: []Type{&TCon{Name: "Y2"}, TInt}},
		// Different arities, same body (the second never uses its b).
		"N1": &TTuple{Elements: []Type{a, a}},
		"N2": &TTuple{Elements: []Type{a, a}},
		// Record bodies carry their alias names; unification ignores them.
		"T1": &TRecord{Fields: map[string]Type{"l": a, "r": a}, TypeName: "T1"},
		"T2": &TRecord{Fields: map[string]Type{"l": a, "r": a}, TypeName: "T2"},
	}
	params := map[string][]string{
		"A": {"a"}, "B": {"a"}, "C1": {"a"}, "C2": {"a"},
		"N1": {"a"}, "N2": {"a", "b"}, "T1": {"a"}, "T2": {"a"},
	}
	u := NewUnifierWithAliasesAndParams(aliases, params)

	if _, err := u.Unify(aliasApp("A", TInt), aliasApp("B", TInt), Substitution{}); err == nil {
		t.Error("(int) -> int unified with (int) -> string through two aliases")
	}
	_, err := u.Unify(aliasApp("C1", TInt), aliasApp("C2", TInt), Substitution{})
	if err == nil || !strings.Contains(err.Error(), TC_NOMINAL_001) {
		t.Errorf("m1.Box against m2.Box through two aliases: want %s, got %v", TC_NOMINAL_001, err)
	}
	for _, pair := range [][2]string{{"A", "B"}, {"C1", "C2"}, {"R1", "R2"}, {"Y1", "Y2"}, {"N1", "N2"}} {
		if u.aliasesEquivalent(pair[0], pair[1]) {
			t.Errorf("%s and %s taken as equivalent", pair[0], pair[1])
		}
	}
	if !u.aliasesEquivalent("T1", "T2") {
		t.Error("twin record aliases (differing only in the alias name they carry) not equivalent")
	}
	if _, err := u.Unify(aliasApp("T1", TInt), aliasApp("T2", TInt), Substitution{}); err != nil {
		t.Errorf("twin record aliases did not unify: %v", err)
	}
}
