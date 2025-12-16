package types

import "testing"

func TestHead_BasicTypes(t *testing.T) {
	tests := []struct {
		name     string
		typ      Type
		expected TypeHead
	}{
		{"int", TInt, HeadInt},
		{"float", TFloat, HeadFloat},
		{"string", TString, HeadString},
		{"bool", TBool, HeadBool},
		{"unit", TUnit, HeadUnit},
		{"bytes", TBytes, HeadBytes},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Head(tt.typ)
			if result != tt.expected {
				t.Errorf("Head(%v) = %v, want %v", tt.typ, result, tt.expected)
			}
		})
	}
}

func TestHead_List(t *testing.T) {
	// list[int] - DX-17: canonical form is lowercase "list"
	listType := &TApp{
		Constructor: &TCon{Name: "list"},
		Args:        []Type{TInt},
	}

	result := Head(listType)
	if result != HeadList {
		t.Errorf("Head(list[int]) = %v, want %v", result, HeadList)
	}
}

func TestHead_Record(t *testing.T) {
	// TRecord
	rec1 := &TRecord{
		Fields: map[string]Type{"x": TInt, "y": TString},
		Row:    nil,
	}
	if Head(rec1) != HeadRecord {
		t.Errorf("Head(TRecord) = %v, want %v", Head(rec1), HeadRecord)
	}

	// TRecord2
	rec2 := &TRecord2{
		Row: &Row{
			Kind:   KRow{ElemKind: KRecord{}},
			Labels: map[string]Type{"x": TInt, "y": TString},
			Tail:   nil,
		},
	}
	if Head(rec2) != HeadRecord {
		t.Errorf("Head(TRecord2) = %v, want %v", Head(rec2), HeadRecord)
	}
}

func TestHead_Func(t *testing.T) {
	// TFunc: int -> string
	fn1 := &TFunc{
		Params: []Type{TInt},
		Return: TString,
	}
	if Head(fn1) != HeadFunc {
		t.Errorf("Head(TFunc) = %v, want %v", Head(fn1), HeadFunc)
	}

	// TFunc2: int -> string ! {IO}
	fn2 := &TFunc2{
		Params:    []Type{TInt},
		Return:    TString,
		EffectRow: &Row{Kind: KEffect{}, Labels: map[string]Type{}},
	}
	if Head(fn2) != HeadFunc {
		t.Errorf("Head(TFunc2) = %v, want %v", Head(fn2), HeadFunc)
	}
}

func TestHead_TypeVariable(t *testing.T) {
	// Type variable should return HeadUnknown
	tvar := &TVar{Name: "a"}
	if Head(tvar) != HeadUnknown {
		t.Errorf("Head(TVar) = %v, want %v", Head(tvar), HeadUnknown)
	}
}

func TestHead_Nil(t *testing.T) {
	if Head(nil) != HeadUnknown {
		t.Errorf("Head(nil) = %v, want %v", Head(nil), HeadUnknown)
	}
}

func TestHead_UnknownCon(t *testing.T) {
	// Unknown type constructor
	unknownType := &TCon{Name: "SomeCustomType"}
	if Head(unknownType) != HeadUnknown {
		t.Errorf("Head(SomeCustomType) = %v, want %v", Head(unknownType), HeadUnknown)
	}
}

func TestTypeHead_String(t *testing.T) {
	tests := []struct {
		head     TypeHead
		expected string
	}{
		{HeadInt, "Int"},
		{HeadFloat, "Float"},
		{HeadString, "String"},
		{HeadBool, "Bool"},
		{HeadList, "List"},
		{HeadRecord, "Record"},
		{HeadFunc, "Func"},
		{HeadUnit, "Unit"},
		{HeadBytes, "Bytes"},
		{HeadUnknown, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.head.String()
			if result != tt.expected {
				t.Errorf("%v.String() = %q, want %q", tt.head, result, tt.expected)
			}
		})
	}
}
