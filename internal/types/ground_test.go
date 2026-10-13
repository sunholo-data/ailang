package types

import "testing"

func groundRecord(n int) *TRecord {
	fields := map[string]Type{}
	for i := 0; i < n; i++ {
		fields[string(rune('a'+i%26))+string(rune('a'+i/26))] = &TList{Element: &TCon{Name: "string"}}
	}
	return &TRecord{Fields: fields}
}

func TestIsGroundType(t *testing.T) {
	cases := []struct {
		name string
		typ  Type
		want bool
	}{
		{"con", &TCon{Name: "int"}, true},
		{"var", &TVar2{Name: "a"}, false},
		{"ground record", groundRecord(3), true},
		{"record with var field", &TRecord{Fields: map[string]Type{"x": &TVar2{Name: "a"}}}, false},
		{"record with row var", &TRecord{Fields: map[string]Type{"x": &TCon{Name: "int"}}, Row: &RowVar{Name: "r"}}, false},
		{"func with open effect row", &TFunc2{Return: &TCon{Name: "int"}, EffectRow: &Row{Tail: &RowVar{Name: "e"}}}, false},
		{"func closed", &TFunc2{Params: []Type{&TCon{Name: "int"}}, Return: &TCon{Name: "int"}, EffectRow: &Row{Labels: map[string]Type{"IO": &TCon{Name: "unit"}}}}, true},
	}
	for _, c := range cases {
		if got := isGroundType(c.typ); got != c.want {
			t.Errorf("%s: isGroundType = %v, want %v", c.name, got, c.want)
		}
		// Second call answers from the memo and must agree.
		if got := isGroundType(c.typ); got != c.want {
			t.Errorf("%s (memoised): isGroundType = %v, want %v", c.name, got, c.want)
		}
	}
}

// A cycle must not let an inner node memoise "ground" on the strength of the
// outer node's provisional answer when the outer node is not ground.
func TestIsGroundType_CycleDoesNotMemoiseProvisionalGround(t *testing.T) {
	outer := &TRecord{Fields: map[string]Type{}}
	inner := &TRecord{Fields: map[string]Type{"back": outer}}
	outer.Fields["inner"] = inner
	outer.Fields["z_var"] = &TVar2{Name: "a"}
	if isGroundType(outer) {
		t.Fatal("outer has a variable but reported ground")
	}
	if isGroundType(inner) {
		t.Fatal("inner reaches a variable through outer but reported ground")
	}

	a := &TRecord{Fields: map[string]Type{}}
	b := &TRecord{Fields: map[string]Type{"a": a}}
	a.Fields["b"] = b
	if !isGroundType(a) || !isGroundType(b) {
		t.Fatal("a variable-free cycle must be ground")
	}
}

// TestSubstitute_GroundRecordIsNotRewalked guards #1787: expanded alias records
// are ground, and substituting into one must return it untouched without
// walking or copying its fields (motoko session.ail: 34 s -> 77 s when it did).
func TestSubstitute_GroundRecordIsNotRewalked(t *testing.T) {
	rec := groundRecord(200)
	sub := Substitution{"a": &TCon{Name: "int"}}
	if got := ApplySubstitution(sub, rec); got != Type(rec) {
		t.Fatal("ground record was copied")
	}
	visited := map[Type]Type{}
	_ = safeSubstitute(rec, sub, visited)
	if len(visited) > 1 {
		t.Fatalf("substituting into a ground 200-field record visited %d nodes; it should stop at the record", len(visited))
	}

	open := &TRecord{Fields: map[string]Type{"x": &TVar2{Name: "a"}, "y": &TCon{Name: "string"}}}
	got, ok := ApplySubstitution(sub, open).(*TRecord)
	if !ok || got.Fields["x"].String() != "int" || got.Fields["y"].String() != "string" {
		t.Fatalf("non-ground record substituted wrongly: %v", got)
	}
	if open.Fields["x"].String() == "int" {
		t.Fatal("substitution mutated the original record")
	}
}
