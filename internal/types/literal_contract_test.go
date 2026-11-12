package types

import "testing"

// hasConcreteEffectContract decides whether a function literal's parameter
// annotation is enforced as an equality instead of seeded (#2238): it must
// find a concrete-contract callback anywhere in the annotation.
func TestHasConcreteEffectContract(t *testing.T) {
	concrete := &TFunc2{Params: []Type{TInt}, Return: TUnit, ConcreteEffectContract: true,
		EffectRow: &Row{Kind: EffectRow, Labels: map[string]Type{"IO": TUnit}}}
	poly := &TFunc2{Params: []Type{TInt}, Return: TUnit,
		EffectRow: &Row{Kind: EffectRow, Labels: map[string]Type{}, Tail: &RowVar{Name: "e", Kind: EffectRow}}}
	for _, c := range []struct {
		name string
		t    Type
		want bool
	}{
		{"int", TInt, false},
		{"callback with a concrete contract", concrete, true},
		{"row-polymorphic callback", poly, false},
		{"concrete callback as a parameter of a function", &TFunc2{Params: []Type{concrete}, Return: TInt}, true},
		{"concrete callback as a return type", &TFunc2{Params: []Type{TInt}, Return: concrete}, true},
		{"list of concrete callbacks", &TList{Element: concrete}, true},
		{"array of concrete callbacks", &TArray{Element: concrete}, true},
		{"tuple holding one", &TTuple{Elements: []Type{TInt, concrete}}, true},
		{"tuple without one", &TTuple{Elements: []Type{TInt, poly}}, false},
		{"type application holding one", &TApp{Constructor: &TCon{Name: "Option", Module: "std/option"}, Args: []Type{concrete}}, true},
		{"closed record field", &TRecord{Fields: map[string]Type{"f": concrete}}, true},
		{"open record field", &TRecordOpen{Fields: map[string]Type{"f": concrete}}, true},
		{"record without one", &TRecord{Fields: map[string]Type{"f": poly}}, false},
	} {
		if got := hasConcreteEffectContract(c.t, 0); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
	// Depth is bounded (annotation types are acyclic; the bound is a guard).
	var deep Type = concrete
	for i := 0; i < 40; i++ {
		deep = &TList{Element: deep}
	}
	if hasConcreteEffectContract(deep, 0) {
		t.Error("a contract nested deeper than the bound is not looked for")
	}
}

func TestDeclaredEffectRow_EmptyIsClosedPure(t *testing.T) {
	row, err := declaredEffectRow(nil)
	if err != nil || row == nil || row.Tail != nil || len(row.Labels) != 0 {
		t.Fatalf("an explicit `! {}` must be the closed empty row, got %v %v", row, err)
	}
}
