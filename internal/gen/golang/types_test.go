package golang

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

// TestTypeMapper_MapType_List tests M-DX25.1: TApp("List", T) maps to []T
func TestTypeMapper_MapType_List(t *testing.T) {
	tm := NewTypeMapper()

	tests := []struct {
		name     string
		typ      types.Type
		expected GoType
	}{
		{
			name: "TApp List int maps to []int64",
			typ: &types.TApp{
				Constructor: &types.TCon{Name: "List"},
				Args:        []types.Type{&types.TCon{Name: "int"}},
			},
			expected: "[]int64",
		},
		{
			name: "TApp List string maps to []string",
			typ: &types.TApp{
				Constructor: &types.TCon{Name: "List"},
				Args:        []types.Type{&types.TCon{Name: "string"}},
			},
			expected: "[]string",
		},
		{
			name: "TApp List bool maps to []bool",
			typ: &types.TApp{
				Constructor: &types.TCon{Name: "List"},
				Args:        []types.Type{&types.TCon{Name: "bool"}},
			},
			expected: "[]bool",
		},
		{
			name: "TApp List float maps to []float64",
			typ: &types.TApp{
				Constructor: &types.TCon{Name: "List"},
				Args:        []types.Type{&types.TCon{Name: "float"}},
			},
			expected: "[]float64",
		},
		{
			name: "Nested TApp List List int maps to [][]int64",
			typ: &types.TApp{
				Constructor: &types.TCon{Name: "List"},
				Args: []types.Type{
					&types.TApp{
						Constructor: &types.TCon{Name: "List"},
						Args:        []types.Type{&types.TCon{Name: "int"}},
					},
				},
			},
			expected: "[][]int64",
		},
		{
			name: "Empty TApp List maps to []interface{}",
			typ: &types.TApp{
				Constructor: &types.TCon{Name: "List"},
				Args:        []types.Type{},
			},
			expected: "[]interface{}",
		},
		{
			name: "TList int maps to []int64 (existing behavior)",
			typ: &types.TList{
				Element: &types.TCon{Name: "int"},
			},
			expected: "[]int64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tm.MapType(tt.typ)
			if err != nil {
				t.Fatalf("MapType() error = %v", err)
			}
			if got != tt.expected {
				t.Errorf("MapType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestTypeMapper_MapType_Primitives tests basic type mappings
func TestTypeMapper_MapType_Primitives(t *testing.T) {
	tm := NewTypeMapper()

	tests := []struct {
		name     string
		typ      types.Type
		expected GoType
	}{
		{"int", &types.TCon{Name: "int"}, GoInt64},
		{"float", &types.TCon{Name: "float"}, GoFloat64},
		{"bool", &types.TCon{Name: "bool"}, GoBool},
		{"string", &types.TCon{Name: "string"}, GoString},
		{"unit", &types.TCon{Name: "()"}, GoUnit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tm.MapType(tt.typ)
			if err != nil {
				t.Fatalf("MapType() error = %v", err)
			}
			if got != tt.expected {
				t.Errorf("MapType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestTypeMapper_MapType_CyclicType tests M-DX11-TRAVERSE: cycle detection in MapType
func TestTypeMapper_MapType_CyclicType(t *testing.T) {
	tm := NewTypeMapper()

	// Create a self-referential type: Tree = TApp(Tree, [int])
	// This simulates a recursive ADT like Tree[int]
	cyclicType := &types.TApp{
		Constructor: &types.TCon{Name: "Tree"},
		Args:        []types.Type{&types.TCon{Name: "int"}},
	}
	// Create the cycle by making the arg point to itself
	cyclicType.Args = []types.Type{cyclicType}

	// This should NOT hang due to cycle detection
	got, err := tm.MapType(cyclicType)
	if err != nil {
		t.Fatalf("MapType() on cyclic type should not error, got: %v", err)
	}
	// Cyclic types fall back to interface{}
	if got != GoType("interface{}") {
		t.Logf("MapType() on cyclic type = %v (interface{} expected for cycle)", got)
	}
}

// TestTypeMapper_MapType_CyclicFunc tests cycle detection in function types
func TestTypeMapper_MapType_CyclicFunc(t *testing.T) {
	tm := NewTypeMapper()

	// Create a self-referential function type
	cyclicFunc := &types.TFunc2{
		Params: []types.Type{&types.TCon{Name: "int"}},
		Return: nil, // Will set to self
	}
	cyclicFunc.Return = cyclicFunc // Function returns itself

	// This should NOT hang due to cycle detection
	got, err := tm.MapType(cyclicFunc)
	if err != nil {
		t.Fatalf("MapType() on cyclic function should not error, got: %v", err)
	}
	// Should produce some valid output (cycle detected returns interface{})
	if got == "" {
		t.Errorf("MapType() on cyclic function returned empty string")
	}
	t.Logf("MapType() on cyclic function = %v", got)
}

// TestTypeMapper_MapType_DeeplyNestedType tests that deep nesting works correctly
func TestTypeMapper_MapType_DeeplyNestedType(t *testing.T) {
	tm := NewTypeMapper()

	// Create deeply nested list type: [[[[int]]]]
	innerType := types.Type(&types.TCon{Name: "int"})
	for i := 0; i < 10; i++ {
		innerType = &types.TList{Element: innerType}
	}

	got, err := tm.MapType(innerType)
	if err != nil {
		t.Fatalf("MapType() on deeply nested type error: %v", err)
	}
	expected := GoType("[][][][][][][][][][]int64")
	if got != expected {
		t.Errorf("MapType() = %v, want %v", got, expected)
	}
}
