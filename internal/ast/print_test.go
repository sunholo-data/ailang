package ast

import (
	"testing"
)

// TestTypeDecl_Alias tests that type alias declarations serialize correctly
func TestTypeDecl_Alias(t *testing.T) {
	// Test type alias: type UserId = int
	// Note: Type aliases are represented as RecordType with a single field for now
	// Or we can use a different representation. For testing, let's use AlgebraicType with one constructor
	typeDecl := &TypeDecl{
		Name:       "UserId",
		TypeParams: nil,
		Definition: &RecordType{
			Fields: []*RecordField{},
		},
		Pos: Pos{Line: 1, Column: 1, File: "test.ail"},
	}

	output := Print(typeDecl)
	if output == "" {
		t.Fatal("Print returned empty string")
	}

	// Should contain type and name
	if !contains(output, "TypeDecl") {
		t.Errorf("Output missing TypeDecl type: %s", output)
	}
	if !contains(output, "UserId") {
		t.Errorf("Output missing name: %s", output)
	}
}

// TestTypeDecl_AlgebraicType tests that sum types serialize correctly
func TestTypeDecl_AlgebraicType(t *testing.T) {
	// Test type Option[a] = Some(a) | None
	typeDecl := &TypeDecl{
		Name:       "Option",
		TypeParams: []string{"a"},
		Definition: &AlgebraicType{
			Constructors: []*Constructor{
				{
					Name:   "Some",
					Fields: []Type{&TypeVar{Name: "a"}},
					Pos:    Pos{Line: 1, Column: 10},
				},
				{
					Name:   "None",
					Fields: nil,
					Pos:    Pos{Line: 1, Column: 20},
				},
			},
			Pos: Pos{Line: 1, Column: 1},
		},
		Pos: Pos{Line: 1, Column: 1},
	}

	output := Print(typeDecl)
	if output == "" {
		t.Fatal("Print returned empty string")
	}

	// Should contain TypeDecl, AlgebraicType, Some, None
	if !contains(output, "TypeDecl") {
		t.Errorf("Output missing TypeDecl type: %s", output)
	}
	if !contains(output, "AlgebraicType") {
		t.Errorf("Output missing AlgebraicType type: %s", output)
	}
	if !contains(output, "Some") {
		t.Errorf("Output missing Some constructor: %s", output)
	}
	if !contains(output, "None") {
		t.Errorf("Output missing None constructor: %s", output)
	}
}

// TestTypeDecl_RecordType tests that product types serialize correctly
func TestTypeDecl_RecordType(t *testing.T) {
	// Test type Point = {x: int, y: int}
	typeDecl := &TypeDecl{
		Name:       "Point",
		TypeParams: nil,
		Definition: &RecordType{
			Fields: []*RecordField{
				{
					Name: "x",
					Type: &SimpleType{Name: "int"},
					Pos:  Pos{Line: 1, Column: 10},
				},
				{
					Name: "y",
					Type: &SimpleType{Name: "int"},
					Pos:  Pos{Line: 1, Column: 20},
				},
			},
			Pos: Pos{Line: 1, Column: 1},
		},
		Pos: Pos{Line: 1, Column: 1},
	}

	output := Print(typeDecl)
	if output == "" {
		t.Fatal("Print returned empty string")
	}

	// Should contain TypeDecl, RecordType, x, y
	if !contains(output, "TypeDecl") {
		t.Errorf("Output missing TypeDecl type: %s", output)
	}
	if !contains(output, "RecordType") {
		t.Errorf("Output missing RecordType type: %s", output)
	}
	if !contains(output, "x") {
		t.Errorf("Output missing field x: %s", output)
	}
	if !contains(output, "y") {
		t.Errorf("Output missing field y: %s", output)
	}
}

// TestTuple_Print tests that tuple expressions serialize correctly
func TestTuple_Print(t *testing.T) {
	// Test (1, 2, 3)
	tuple := &Tuple{
		Elements: []Expr{
			&Literal{Kind: IntLit, Value: int64(1)},
			&Literal{Kind: IntLit, Value: int64(2)},
			&Literal{Kind: IntLit, Value: int64(3)},
		},
		Pos: Pos{Line: 1, Column: 1},
	}

	output := Print(tuple)
	if output == "" {
		t.Fatal("Print returned empty string")
	}

	// Should contain Tuple and elements
	if !contains(output, "Tuple") {
		t.Errorf("Output missing Tuple type: %s", output)
	}
	if !contains(output, "elements") {
		t.Errorf("Output missing elements: %s", output)
	}
}

// TestDeterministicMarshaling tests that serialization is deterministic
func TestDeterministicMarshaling(t *testing.T) {
	// Create a complex type declaration
	typeDecl := &TypeDecl{
		Name:       "Result",
		TypeParams: []string{"a", "e"},
		Definition: &AlgebraicType{
			Constructors: []*Constructor{
				{
					Name:   "Ok",
					Fields: []Type{&TypeVar{Name: "a"}},
				},
				{
					Name:   "Err",
					Fields: []Type{&TypeVar{Name: "e"}},
				},
			},
		},
	}

	// Serialize 100 times and ensure all outputs are identical
	var outputs []string
	for i := 0; i < 100; i++ {
		output := Print(typeDecl)
		outputs = append(outputs, output)
	}

	baseline := outputs[0]
	for i, output := range outputs[1:] {
		if output != baseline {
			t.Errorf("Iteration %d produced different output", i+1)
			t.Logf("Baseline: %s", baseline)
			t.Logf("Variant: %s", output)
			break
		}
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && hasSubstring(s, substr)
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
