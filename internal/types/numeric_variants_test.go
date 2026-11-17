package types

import "testing"

// The variant name is a contract between the exporting module, which compiles
// inc$numF, and the importing one, which calls it: both derive it from the
// scheme alone (#2378).
func TestNumVariantNames(t *testing.T) {
	cases := []struct {
		floats []bool
		want   string
	}{
		{nil, "f"},
		{[]bool{false}, "f"},
		{[]bool{true}, "f$numF"},
		{[]bool{false, true}, "f$numIF"},
		{[]bool{true, false}, "f$numFI"},
	}
	for _, c := range cases {
		got := NumVariantName("f", c.floats)
		if got != c.want {
			t.Errorf("NumVariantName(f, %v) = %q, want %q", c.floats, got, c.want)
		}
		base, ok := NumVariantBase(got)
		if want := got != "f"; ok != want || (ok && base != "f") {
			t.Errorf("NumVariantBase(%q) = %q, %v", got, base, ok)
		}
	}
	for _, name := range []string{"f$num", "f$numX", "$numF", "f$lambda1", "num"} {
		if b, ok := NumVariantBase(name); ok {
			t.Errorf("NumVariantBase(%q) = %q, want no variant", name, b)
		}
	}
	if !IsNumVariantOf("a$numF$numF", "a$numF") || IsNumVariantOf("ab$numF", "a") {
		t.Error("IsNumVariantOf must match the whole base name")
	}
}

// NumVariantVars: the Num-only quantified variables, in first-appearance order
// of the type; a Fractional variable (one instance, float) has no variants,
// and neither has an unconstrained one.
func TestNumVariantVars(t *testing.T) {
	a, b, c, d := &TVar2{Name: "a", Kind: Star}, &TVar2{Name: "b", Kind: Star}, &TVar2{Name: "c", Kind: Star}, &TVar2{Name: "d", Kind: Star}
	s := &Scheme{
		TypeVars: []string{"a", "b", "c", "d"},
		Constraints: []Constraint{
			{Class: "Num", Type: a}, {Class: "Ord", Type: a},
			{Class: "Num", Type: b},
			{Class: "Num", Type: c}, {Class: "Fractional", Type: c},
		},
		// (b, {x: a | ...}, c, d) -> b
		Type: &TFunc2{Params: []Type{b, &TRecordOpen{Fields: map[string]Type{"x": a}}, c, d}, Return: b, EffectRow: EmptyEffectRow()},
	}
	got := NumVariantVars(s)
	if len(got) != 2 || got[0] != "b" || got[1] != "a" {
		t.Fatalf("NumVariantVars = %v, want [b a]", got)
	}
	if v := NumVariantVars(&Scheme{TypeVars: []string{"a"}, Type: a}); v != nil {
		t.Errorf("an unconstrained scheme has no numeric variants, got %v", v)
	}
}

// A use's instance is read by matching the scheme against the use's type: an
// open record in the scheme matches a closed one by field name.
func TestMatchSchemeVarsReadsRecordFields(t *testing.T) {
	a := &TVar2{Name: "a", Kind: Star}
	scheme := &TFunc2{Params: []Type{&TRecordOpen{Fields: map[string]Type{"v": a}, Row: &RowVar{Name: "r"}}}, Return: &TList{Element: a}, EffectRow: EmptyEffectRow()}
	inst := &TFunc2{Params: []Type{&TRecord{Fields: map[string]Type{"v": TFloat, "w": TInt}}}, Return: &TList{Element: TFloat}, EffectRow: EmptyEffectRow()}
	bound := map[string]Type{}
	matchSchemeVars(scheme, inst, map[string]bool{"a": true}, bound, 0)
	if got := bound["a"]; got == nil || NormalizeTypeName(got) != "Float" {
		t.Fatalf("a bound to %v, want float", got)
	}
}

// Every structure a Num variable can sit under is read positionally (records
// by field name, in any representation), through a label on either side.
func TestMatchSchemeVarsReadsEveryStructure(t *testing.T) {
	v := func(n string) Type { return &TVar2{Name: n, Kind: Star} }
	opt := &TCon{Name: "Option"}
	scheme := &TFunc2{
		Params: []Type{
			&TArray{Element: v("a")},
			&TMap{Key: TString, Value: v("b")},
			&TTuple{Elements: []Type{TInt, v("c")}},
			&TApp{Constructor: opt, Args: []Type{v("d")}},
			&TRecord2{Row: &Row{Kind: RecordRow, Labels: map[string]Type{"e": v("e")}}},
			&TLabelled{Inner: v("f")},
		},
		Return:    &TRecord{Fields: map[string]Type{}, Row: &Row{Kind: RecordRow, Labels: map[string]Type{"g": v("g")}}},
		EffectRow: EmptyEffectRow(),
	}
	inst := &TFunc2{
		Params: []Type{
			&TArray{Element: TFloat},
			&TMap{Key: TString, Value: TFloat},
			&TTuple{Elements: []Type{TInt, TFloat}},
			&TApp{Constructor: opt, Args: []Type{TFloat}},
			&TRecordOpen{Fields: map[string]Type{"e": TFloat}},
			&TLabelled{Inner: TFloat},
		},
		Return:    &TRecord2{Row: &Row{Kind: RecordRow, Labels: map[string]Type{"g": TFloat}}},
		EffectRow: EmptyEffectRow(),
	}
	want := map[string]bool{}
	for _, n := range []string{"a", "b", "c", "d", "e", "f", "g"} {
		want[n] = true
	}
	bound := map[string]Type{}
	matchSchemeVars(scheme, inst, want, bound, 0)
	for n := range want {
		if got := bound[n]; got == nil || NormalizeTypeName(got) != "Float" {
			t.Errorf("%s bound to %v, want float", n, got)
		}
	}
}
