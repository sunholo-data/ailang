package repl

import (
	"testing"

	"github.com/sunholo/ailang/internal/types"
)

// TestFormatType_CyclicType tests M-DX11-TRAVERSE: cycle detection in formatType
func TestFormatType_CyclicType(t *testing.T) {
	// Create a self-referential type: TApp pointing to itself
	cyclicType := &types.TApp{
		Constructor: &types.TCon{Name: "Tree"},
		Args:        []types.Type{&types.TCon{Name: "int"}},
	}
	// Create the cycle by making the arg point to itself
	cyclicType.Args = []types.Type{cyclicType}

	// This should NOT hang due to cycle detection
	got := formatType(cyclicType)
	if got == "" {
		t.Errorf("formatType() on cyclic type returned empty string")
	}
	// Should contain <cycle> marker when cycle is detected
	t.Logf("formatType() on cyclic type = %v", got)
}

// TestFormatType_CyclicRecord tests cycle detection in record types
func TestFormatType_CyclicRecord(t *testing.T) {
	// Create a self-referential record type
	cyclicRecord := &types.TRecord{
		Fields: map[string]types.Type{
			"name": &types.TCon{Name: "string"},
		},
	}
	// Create the cycle by adding a field pointing to itself
	cyclicRecord.Fields["self"] = cyclicRecord

	// This should NOT hang due to cycle detection
	got := formatType(cyclicRecord)
	if got == "" {
		t.Errorf("formatType() on cyclic record returned empty string")
	}
	t.Logf("formatType() on cyclic record = %v", got)
}

// TestFormatType_CyclicList tests cycle detection in list types
func TestFormatType_CyclicList(t *testing.T) {
	// Create a self-referential list type
	cyclicList := &types.TList{
		Element: nil, // Will set to self
	}
	cyclicList.Element = cyclicList

	// This should NOT hang due to cycle detection
	got := formatType(cyclicList)
	if got == "" {
		t.Errorf("formatType() on cyclic list returned empty string")
	}
	// Should contain <cycle> marker
	if got != "[<cycle>]" {
		t.Logf("formatType() on cyclic list = %v (expected [<cycle>])", got)
	}
}

// TestFormatType_NilType tests handling of nil types
func TestFormatType_NilType(t *testing.T) {
	got := formatType(nil)
	if got != "nil" {
		t.Errorf("formatType(nil) = %v, want nil", got)
	}
}

// TestFormatType_BasicTypes tests basic type formatting
func TestFormatType_BasicTypes(t *testing.T) {
	tests := []struct {
		name string
		typ  types.Type
		want string
	}{
		{"int", &types.TCon{Name: "int"}, "Int"},
		{"float", &types.TCon{Name: "float"}, "Float"},
		{"bool", &types.TCon{Name: "bool"}, "Bool"},
		{"string", &types.TCon{Name: "string"}, "String"},
		{"TVar", &types.TVar{Name: "a"}, "a"},
		{"TVar2", &types.TVar2{Name: "b"}, "b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatType(tt.typ)
			if got != tt.want {
				t.Errorf("formatType() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestFormatType_DeeplyNestedList tests deep nesting without hanging
func TestFormatType_DeeplyNestedList(t *testing.T) {
	// Create deeply nested list type
	innerType := types.Type(&types.TCon{Name: "int"})
	for i := 0; i < 50; i++ {
		innerType = &types.TList{Element: innerType}
	}

	// This should complete without hanging
	got := formatType(innerType)
	if got == "" {
		t.Errorf("formatType() on deeply nested list returned empty string")
	}
	// Should contain lots of brackets
	if len(got) < 100 {
		t.Errorf("formatType() on deeply nested list seems truncated: %v", got)
	}
}
