package types

// M-EFFECT-ROW-SHOW-INTERP (#386) Section B: every free row variable in a scheme
// must be generalized so each imported row-polymorphic use gets FRESH row vars.
//
// M1 STATUS: TestGeneralizeWithConstraints_RowVars_386 is RED — generalizeWithConstraints
// emits `RowVars: []string{} // Simplified for now`, so a mapE-shaped scheme has NO
// quantified row var and separate uses share the literal name `e`. M3 flips it GREEN.

import (
	"strconv"
	"testing"
)

// mapEShapedType builds a type of the same shape as std/list.mapE after inference:
//
//	(f: (a) -> b ! {e}, xs: [a]) -> [b] ! {e}
//
// with a single shared effect-row tail var `e` appearing in BOTH the callback
// parameter's effect row and the outer function's effect row.
func mapEShapedType() *TFunc2 {
	rowE := func() *Row {
		return &Row{
			Kind:   EffectRow,
			Labels: map[string]Type{},
			Tail:   &RowVar{Name: "e", Kind: EffectRow},
		}
	}
	callback := &TFunc2{
		Params:    []Type{&TVar2{Name: "a", Kind: Star}},
		Return:    &TVar2{Name: "b", Kind: Star},
		EffectRow: rowE(),
	}
	return &TFunc2{
		Params:    []Type{callback, &TList{Element: &TVar2{Name: "a", Kind: Star}}},
		Return:    &TList{Element: &TVar2{Name: "b", Kind: Star}},
		EffectRow: rowE(),
	}
}

// TestGeneralizeWithConstraints_RowVars_386 asserts that generalization quantifies
// the free effect-row variable `e` of a mapE-shaped scheme.
//
// M1: RED — RowVars is currently empty (the `// Simplified for now` bug).
// M3: GREEN — RowVars must contain "e".
func TestGeneralizeWithConstraints_RowVars_386(t *testing.T) {
	tc := NewCoreTypeChecker()
	typ := mapEShapedType()

	// Top-level declaration: currentEnv == baseEnv, so nothing is withheld and the
	// free row var `e` (which is not owned by any enclosing binder) must generalize.
	baseEnv := NewTypeEnv()
	scheme := tc.generalizeWithConstraints(typ, EmptyEffectRow(), nil, baseEnv, baseEnv.FreeTypeVars())

	found := false
	for _, rv := range scheme.RowVars {
		if rv == "e" {
			found = true
		}
	}
	if !found {
		t.Fatalf("generalizeWithConstraints did not quantify the free effect-row var 'e' "+
			"(RowVars=%v); the mapE-shaped scheme leaks 'e' so separate uses share one row "+
			"identity — this is #386 Section B. (RED in M1, must be GREEN after M3.)", scheme.RowVars)
	}
}

// TestInstantiateWithConstraints_FreshRows_386 proves that once a scheme quantifies
// its effect-row var, two instantiations produce DIFFERENT fresh row names (no
// cross-use contamination). This mechanism already exists in
// Scheme.InstantiateWithConstraints; the test documents the M3 contract that a
// populated RowVars yields per-use freshening. It is GREEN from the start (it does
// not depend on generalizeWithConstraints) and guards against regressions.
func TestInstantiateWithConstraints_FreshRows_386(t *testing.T) {
	scheme := &Scheme{
		TypeVars: []string{"a", "b"},
		RowVars:  []string{"e"},
		Type:     mapEShapedType(),
	}

	counter := 0
	fresh := func(k Kind) Type {
		counter++
		switch k {
		case EffectRow:
			return &RowVar{Name: freshRowName(counter), Kind: EffectRow}
		default:
			return &TVar2{Name: freshTypeName(counter), Kind: Star}
		}
	}

	rowNameOf := func(fn *TFunc2) string {
		if fn.EffectRow != nil && fn.EffectRow.Tail != nil {
			return fn.EffectRow.Tail.Name
		}
		return ""
	}

	t1, _ := scheme.InstantiateWithConstraints(fresh)
	t2, _ := scheme.InstantiateWithConstraints(fresh)

	f1, ok1 := t1.(*TFunc2)
	f2, ok2 := t2.(*TFunc2)
	if !ok1 || !ok2 {
		t.Fatalf("instantiation did not yield *TFunc2: %T, %T", t1, t2)
	}

	n1 := rowNameOf(f1)
	n2 := rowNameOf(f2)
	if n1 == "" || n2 == "" {
		t.Fatalf("instantiated outer effect row lost its tail var (n1=%q n2=%q)", n1, n2)
	}
	if n1 == n2 {
		t.Fatalf("two instantiations reused the same row var %q; each use must get a FRESH row (#386)", n1)
	}
	// The callback param's effect row must be freshened to the SAME name as the outer
	// row within a single instantiation (source-level `e` is shared across both).
	cb1, ok := f1.Params[0].(*TFunc2)
	if !ok {
		t.Fatalf("callback param is not *TFunc2: %T", f1.Params[0])
	}
	if got := rowNameOf(cb1); got != n1 {
		t.Errorf("within one instantiation, callback row %q != outer row %q; the shared source 'e' must map to one fresh var", got, n1)
	}
}

func freshRowName(n int) string  { return "ε_test" + strconv.Itoa(n) }
func freshTypeName(n int) string { return "α_test" + strconv.Itoa(n) }
