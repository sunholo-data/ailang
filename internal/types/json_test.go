package types

import (
	"encoding/json"
	"testing"
)

// M-INCREMENTAL-TYPECHECK: Round-trip tests for all 14 Type + 4 Kind JSON serialization.

// roundTripType marshals then unmarshals a Type, verifying structural equality.
func roundTripType(t *testing.T, typ Type) Type {
	t.Helper()
	data, err := json.Marshal(typ)
	if err != nil {
		t.Fatalf("Marshal(%T) failed: %v", typ, err)
	}
	got, err := UnmarshalType(data)
	if err != nil {
		t.Fatalf("UnmarshalType failed: %v\nJSON: %s", err, string(data))
	}
	if !typ.Equals(got) {
		t.Errorf("round-trip mismatch:\n  original: %s\n  got:      %s\n  JSON:     %s", typ, got, string(data))
	}
	return got
}

func TestRoundTrip_TCon(t *testing.T) {
	roundTripType(t, &TCon{Name: "int"})
	roundTripType(t, &TCon{Name: "string"})
	roundTripType(t, &TCon{Name: "()"})
}

func TestRoundTrip_TVar(t *testing.T) {
	roundTripType(t, &TVar{Name: "a"})
	roundTripType(t, &TVar{Name: "t42"})
}

func TestRoundTrip_TVar2(t *testing.T) {
	roundTripType(t, &TVar2{Name: "α1", Kind: KStar{}})
	roundTripType(t, &TVar2{Name: "α2", Kind: KRow{ElemKind: KEffect{}}})
}

func TestRoundTrip_RowVar(t *testing.T) {
	roundTripType(t, &RowVar{Name: "ρ1", Kind: KRow{ElemKind: KEffect{}}})
	roundTripType(t, &RowVar{Name: "ρ2", Kind: KRow{ElemKind: KRecord{}}})
}

func TestRoundTrip_TList(t *testing.T) {
	roundTripType(t, &TList{Element: &TCon{Name: "int"}})
	// Nested list
	roundTripType(t, &TList{Element: &TList{Element: &TCon{Name: "string"}}})
}

func TestRoundTrip_TArray(t *testing.T) {
	roundTripType(t, &TArray{Element: &TCon{Name: "float"}})
}

func TestRoundTrip_TMap(t *testing.T) {
	roundTripType(t, &TMap{Key: &TCon{Name: "string"}, Value: &TCon{Name: "int"}})
}

func TestRoundTrip_TTuple(t *testing.T) {
	roundTripType(t, &TTuple{Elements: []Type{&TCon{Name: "int"}, &TCon{Name: "string"}}})
	// Empty tuple
	roundTripType(t, &TTuple{Elements: []Type{}})
}

func TestRoundTrip_TFunc2(t *testing.T) {
	// Simple function
	roundTripType(t, &TFunc2{
		Params: []Type{&TCon{Name: "int"}},
		Return: &TCon{Name: "bool"},
	})
	// With effect row
	roundTripType(t, &TFunc2{
		Params:    []Type{&TCon{Name: "string"}},
		EffectRow: &Row{Kind: KRow{ElemKind: KEffect{}}, Labels: map[string]Type{"IO": &TCon{Name: "()"}}},
		Return:    &TCon{Name: "int"},
	})
	// Multi-param
	roundTripType(t, &TFunc2{
		Params: []Type{&TCon{Name: "int"}, &TCon{Name: "int"}},
		Return: &TCon{Name: "int"},
	})
}

func TestRoundTrip_TRecord(t *testing.T) {
	// Simple record
	roundTripType(t, &TRecord{
		Fields: map[string]Type{
			"name": &TCon{Name: "string"},
			"age":  &TCon{Name: "int"},
		},
	})
	// With TypeName
	roundTripType(t, &TRecord{
		Fields:   map[string]Type{"x": &TCon{Name: "float"}, "y": &TCon{Name: "float"}},
		TypeName: "Point",
	})
	// With row variable
	roundTripType(t, &TRecord{
		Fields: map[string]Type{"x": &TCon{Name: "int"}},
		Row:    &RowVar{Name: "ρ1", Kind: KRow{ElemKind: KRecord{}}},
	})
}

func TestRoundTrip_TRecordOpen(t *testing.T) {
	roundTripType(t, &TRecordOpen{
		Fields: map[string]Type{"x": &TCon{Name: "int"}},
		Row:    &RowVar{Name: "ρ1", Kind: KRow{ElemKind: KRecord{}}},
	})
}

func TestRoundTrip_TRecord2(t *testing.T) {
	roundTripType(t, &TRecord2{
		Row: &Row{
			Kind:   KRow{ElemKind: KRecord{}},
			Labels: map[string]Type{"a": &TCon{Name: "int"}, "b": &TCon{Name: "string"}},
		},
	})
	// Nil row
	roundTripType(t, &TRecord2{})
}

func TestRoundTrip_TApp(t *testing.T) {
	// Maybe[int]
	roundTripType(t, &TApp{
		Constructor: &TCon{Name: "Maybe"},
		Args:        []Type{&TCon{Name: "int"}},
	})
	// Result[string, Error]
	roundTripType(t, &TApp{
		Constructor: &TCon{Name: "Result"},
		Args:        []Type{&TCon{Name: "string"}, &TCon{Name: "Error"}},
	})
}

func TestRoundTrip_Row(t *testing.T) {
	// Effect row with labels
	roundTripType(t, &Row{
		Kind:   KRow{ElemKind: KEffect{}},
		Labels: map[string]Type{"IO": &TCon{Name: "()"}, "Net": &TCon{Name: "()"}},
	})
	// Row with tail
	roundTripType(t, &Row{
		Kind:   KRow{ElemKind: KEffect{}},
		Labels: map[string]Type{"IO": &TCon{Name: "()"}},
		Tail:   &RowVar{Name: "ε1", Kind: KRow{ElemKind: KEffect{}}},
	})
	// Row with budgets
	budget5 := 5
	minBudget1 := 1
	roundTripType(t, &Row{
		Kind:       KRow{ElemKind: KEffect{}},
		Labels:     map[string]Type{"IO": &TCon{Name: "()"}},
		Budgets:    map[string]*int{"IO": &budget5},
		MinBudgets: map[string]*int{"IO": &minBudget1},
	})
	// Row with nil budget values
	roundTripType(t, &Row{
		Kind:    KRow{ElemKind: KEffect{}},
		Labels:  map[string]Type{"IO": &TCon{Name: "()"}},
		Budgets: map[string]*int{"IO": nil},
	})
}

func TestRoundTrip_NestedComplex(t *testing.T) {
	// TFunc2 with TList[TRecord] params — the design doc's example
	roundTripType(t, &TFunc2{
		Params: []Type{
			&TList{Element: &TRecord{
				Fields: map[string]Type{
					"name": &TCon{Name: "string"},
					"id":   &TCon{Name: "int"},
				},
				TypeName: "Person",
			}},
		},
		EffectRow: &Row{
			Kind:   KRow{ElemKind: KEffect{}},
			Labels: map[string]Type{"IO": &TCon{Name: "()"}, "DB": &TCon{Name: "()"}},
			Tail:   &RowVar{Name: "ε", Kind: KRow{ElemKind: KEffect{}}},
		},
		Return: &TApp{
			Constructor: &TCon{Name: "Result"},
			Args: []Type{
				&TTuple{Elements: []Type{&TCon{Name: "int"}, &TCon{Name: "string"}}},
				&TCon{Name: "Error"},
			},
		},
	})
}

func TestRoundTrip_NullType(t *testing.T) {
	data, err := json.Marshal((*TCon)(nil))
	if err != nil {
		t.Fatalf("Marshal nil failed: %v", err)
	}
	got, err := UnmarshalType(data)
	if err != nil {
		t.Fatalf("UnmarshalType nil failed: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

// --- Kind round-trip tests ---

func TestRoundTrip_Kind(t *testing.T) {
	tests := []struct {
		name string
		kind Kind
	}{
		{"KStar", KStar{}},
		{"KEffect", KEffect{}},
		{"KRecord", KRecord{}},
		{"KRow_Effect", KRow{ElemKind: KEffect{}}},
		{"KRow_Record", KRow{ElemKind: KRecord{}}},
		{"KRow_Nested", KRow{ElemKind: KRow{ElemKind: KStar{}}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := MarshalKind(tt.kind)
			if err != nil {
				t.Fatalf("MarshalKind failed: %v", err)
			}
			got, err := UnmarshalKind(data)
			if err != nil {
				t.Fatalf("UnmarshalKind failed: %v\nJSON: %s", err, string(data))
			}
			if !tt.kind.Equals(got) {
				t.Errorf("Kind round-trip mismatch: %v != %v", tt.kind, got)
			}
		})
	}
}

// --- Scheme round-trip tests ---

func TestRoundTrip_Scheme(t *testing.T) {
	// Simple monomorphic scheme
	s1 := &Scheme{
		TypeVars: nil,
		Type:     &TCon{Name: "int"},
	}
	data, err := MarshalScheme(s1)
	if err != nil {
		t.Fatalf("MarshalScheme failed: %v", err)
	}
	got, err := UnmarshalScheme(data)
	if err != nil {
		t.Fatalf("UnmarshalScheme failed: %v", err)
	}
	if !s1.Type.Equals(got.Type) {
		t.Errorf("Scheme type mismatch")
	}

	// Polymorphic scheme: ∀a. a -> a
	s2 := &Scheme{
		TypeVars: []string{"a"},
		Type: &TFunc2{
			Params: []Type{&TVar2{Name: "a", Kind: KStar{}}},
			Return: &TVar2{Name: "a", Kind: KStar{}},
		},
	}
	data, err = MarshalScheme(s2)
	if err != nil {
		t.Fatalf("MarshalScheme polymorphic failed: %v", err)
	}
	got, err = UnmarshalScheme(data)
	if err != nil {
		t.Fatalf("UnmarshalScheme polymorphic failed: %v", err)
	}
	if len(got.TypeVars) != 1 || got.TypeVars[0] != "a" {
		t.Errorf("TypeVars mismatch: %v", got.TypeVars)
	}
	if !s2.Type.Equals(got.Type) {
		t.Errorf("Scheme type mismatch: %s vs %s", s2.Type, got.Type)
	}

	// With constraints
	s3 := &Scheme{
		TypeVars:    []string{"a"},
		Constraints: []Constraint{{Class: "Num", Type: &TVar2{Name: "a", Kind: KStar{}}}},
		Type: &TFunc2{
			Params: []Type{&TVar2{Name: "a", Kind: KStar{}}, &TVar2{Name: "a", Kind: KStar{}}},
			Return: &TVar2{Name: "a", Kind: KStar{}},
		},
	}
	data, err = MarshalScheme(s3)
	if err != nil {
		t.Fatalf("MarshalScheme with constraints failed: %v", err)
	}
	got, err = UnmarshalScheme(data)
	if err != nil {
		t.Fatalf("UnmarshalScheme with constraints failed: %v", err)
	}
	if len(got.Constraints) != 1 || got.Constraints[0].Class != "Num" {
		t.Errorf("Constraints mismatch: %v", got.Constraints)
	}
}

// --- CoreTypeInfo round-trip test ---

func TestRoundTrip_CoreTypeInfo(t *testing.T) {
	cti := CoreTypeInfo{
		1:  &TCon{Name: "int"},
		2:  &TCon{Name: "string"},
		42: &TFunc2{Params: []Type{&TCon{Name: "int"}}, Return: &TCon{Name: "bool"}},
		99: &TRecord{Fields: map[string]Type{"x": &TCon{Name: "float"}}, TypeName: "Point"},
	}

	data, err := MarshalCoreTypeInfo(cti)
	if err != nil {
		t.Fatalf("MarshalCoreTypeInfo failed: %v", err)
	}

	got, err := UnmarshalCoreTypeInfo(data)
	if err != nil {
		t.Fatalf("UnmarshalCoreTypeInfo failed: %v\nJSON: %s", err, string(data))
	}

	if len(got) != len(cti) {
		t.Fatalf("CoreTypeInfo length mismatch: %d vs %d", len(got), len(cti))
	}

	for id, origType := range cti {
		gotType, ok := got[id]
		if !ok {
			t.Errorf("CoreTypeInfo missing key %d", id)
			continue
		}
		if !origType.Equals(gotType) {
			t.Errorf("CoreTypeInfo[%d] mismatch: %s vs %s", id, origType, gotType)
		}
	}
}

// --- Double round-trip: marshal → unmarshal → marshal → compare bytes ---

func TestDoubleRoundTrip_StableJSON(t *testing.T) {
	types := []Type{
		&TCon{Name: "int"},
		&TVar{Name: "a"},
		&TVar2{Name: "α", Kind: KStar{}},
		&TList{Element: &TCon{Name: "string"}},
		&TFunc2{
			Params: []Type{&TCon{Name: "int"}},
			Return: &TCon{Name: "bool"},
		},
	}

	for _, typ := range types {
		data1, err := json.Marshal(typ)
		if err != nil {
			t.Fatalf("first marshal of %T failed: %v", typ, err)
		}
		mid, err := UnmarshalType(data1)
		if err != nil {
			t.Fatalf("unmarshal of %T failed: %v", typ, err)
		}
		data2, err := json.Marshal(mid)
		if err != nil {
			t.Fatalf("second marshal of %T failed: %v", typ, err)
		}
		if string(data1) != string(data2) {
			t.Errorf("double round-trip unstable for %T:\n  pass1: %s\n  pass2: %s", typ, data1, data2)
		}
	}
}
