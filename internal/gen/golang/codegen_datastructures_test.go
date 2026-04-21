package golang

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

func TestGenerateList(t *testing.T) {
	// [1, 2, 3]
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "nums",
				Value: &core.List{
					Elements: []core.CoreExpr{
						&core.Lit{Kind: core.IntLit, Value: int64(1)},
						&core.Lit{Kind: core.IntLit, Value: int64(2)},
						&core.Lit{Kind: core.IntLit, Value: int64(3)},
					},
				},
				Body: &core.Var{Name: "nums"},
			},
		},
	}

	gen := New("test")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Check for slice literal
	if !strings.Contains(codeStr, "[]interface{}") {
		t.Errorf("Missing slice type")
	}
	if !strings.Contains(codeStr, "1") && !strings.Contains(codeStr, "2") && !strings.Contains(codeStr, "3") {
		t.Errorf("Missing list elements")
	}
}

func TestGenerateRecord(t *testing.T) {
	// { x: 10, y: 20 }
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "point",
				Value: &core.Record{
					Fields: map[string]core.CoreExpr{
						"x": &core.Lit{Kind: core.IntLit, Value: int64(10)},
						"y": &core.Lit{Kind: core.IntLit, Value: int64(20)},
					},
				},
				Body: &core.Var{Name: "point"},
			},
		},
	}

	gen := New("test")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Check for map
	if !strings.Contains(codeStr, "map[string]interface{}") {
		t.Errorf("Missing map type")
	}
}

func TestGenerateNestedLet(t *testing.T) {
	// Test nested let inside a function:
	// let f = \_ . let x = 1 in let y = 2 in x + y
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "f",
				Value: &core.Lambda{
					Params: []string{"_"},
					Body: &core.Let{
						Name:  "x",
						Value: &core.Lit{Kind: core.IntLit, Value: int64(1)},
						Body: &core.Let{
							Name:  "y",
							Value: &core.Lit{Kind: core.IntLit, Value: int64(2)},
							Body: &core.BinOp{
								Op:    "+",
								Left:  &core.Var{Name: "x"},
								Right: &core.Var{Name: "y"},
							},
						},
					},
				},
				Body: &core.Var{Name: "f"},
			},
		},
	}

	gen := New("test")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Should have nested let bindings inside the generated IIFE
	// M-DX13.3: Uses "var x interface{}" to allow type assertions on concrete values
	if !strings.Contains(codeStr, "var x interface{} =") {
		t.Errorf("Missing x binding, got:\n%s", codeStr)
	}
	if !strings.Contains(codeStr, "var y interface{} =") {
		t.Errorf("Missing y binding, got:\n%s", codeStr)
	}
}

// TestGetSliceConversion tests the unified slice conversion lookup.
// M-CODEGEN-UNIFIED-SLICE: Verifies all type registries are checked.
func TestGetSliceConversion(t *testing.T) {
	gen := New("test")

	// Test primitive types
	tests := []struct {
		name     string
		goType   string
		expected string
	}{
		{"int64 slice", "[]int64", "ConvertToInt64Slice"},
		{"float64 slice", "[]float64", "ConvertToFloat64Slice"},
		{"string slice", "[]string", "ConvertToStringSlice"},
		{"bool slice", "[]bool", "ConvertToBoolSlice"},
		{"map slice", "[]map[string]interface{}", "ConvertToRecordSlice"},
		{"unknown slice", "[]unknown", ""},
		{"non-slice", "int64", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.getSliceConversion(tt.goType)
			if result != tt.expected {
				t.Errorf("getSliceConversion(%q) = %q, want %q", tt.goType, result, tt.expected)
			}
		})
	}
}

// TestGetSliceConversionRecordType tests record type slice conversion lookup.
func TestGetSliceConversionRecordType(t *testing.T) {
	gen := New("test")

	// Register a record type
	gen.RegisterRecordType("Planet", []string{"name", "mass"}, map[string]string{"name": "string", "mass": "float64"})

	// Should find the converter
	result := gen.getSliceConversion("[]*Planet")
	if result != "ConvertToPlanetSlice" {
		t.Errorf("getSliceConversion(\"[]*Planet\") = %q, want \"ConvertToPlanetSlice\"", result)
	}

	// Unregistered type should return empty
	result = gen.getSliceConversion("[]*Unknown")
	if result != "" {
		t.Errorf("getSliceConversion(\"[]*Unknown\") = %q, want \"\"", result)
	}
}

// TestGetSliceConversionADTType tests ADT type slice conversion lookup via adtConstructors.
func TestGetSliceConversionADTType(t *testing.T) {
	gen := New("test")

	// Register an ADT constructor (not using adtSliceTypes)
	gen.RegisterADTConstructor("Star", "G", 0)

	// Should find the converter via adtConstructors
	result := gen.getSliceConversion("[]*Star")
	if result != "ConvertToStarSlice" {
		t.Errorf("getSliceConversion(\"[]*Star\") = %q, want \"ConvertToStarSlice\"", result)
	}
}

func TestGetOptHasReflectionPath(t *testing.T) {
	// Test that GetOpt runtime function includes reflection fallback for typed slices
	// Bug: GetOpt only handled []interface{}, but Go codegen creates typed slices
	// like []int64 for Array[int]. Without reflection, GetOpt returns None for all indices.
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name:  "x",
				Value: &core.Lit{Kind: core.IntLit, Value: int64(1)},
				Body:  &core.Var{Name: "x"},
			},
		},
	}

	gen := New("test")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Verify GetOpt has reflection path for typed slices (M-CODEGEN-GETOPT-TYPED-SLICES)
	if !strings.Contains(codeStr, "func GetOpt(arr interface{}, idx interface{})") {
		t.Errorf("Missing GetOpt function declaration")
	}

	// Check for fast path ([]interface{})
	if !strings.Contains(codeStr, "arr.([]interface{})") {
		t.Errorf("Missing fast path for []interface{} in GetOpt")
	}

	// Check for reflection path - critical fix for typed slices
	if !strings.Contains(codeStr, "reflect.ValueOf(arr)") {
		t.Errorf("Missing reflection path in GetOpt - typed slices like []int64 won't work")
	}
	if !strings.Contains(codeStr, "v.Kind() == reflect.Slice") {
		t.Errorf("Missing reflect.Slice check in GetOpt")
	}
	if !strings.Contains(codeStr, "v.Index(int(i)).Interface()") {
		t.Errorf("Missing v.Index access in GetOpt reflection path")
	}
}
