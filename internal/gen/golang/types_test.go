package golang

import (
	"testing"

	"github.com/sunholo/ailang/internal/types"
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
