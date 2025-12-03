package golang

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/ast"
)

func TestGenerateSumType_Simple(t *testing.T) {
	// type Option = | Some(int) | None
	decl := &ast.TypeDecl{
		Name: "Option",
		Definition: &ast.AlgebraicType{
			Constructors: []*ast.Constructor{
				{Name: "Some", Fields: []*ast.ConstructorField{{Type: &ast.SimpleType{Name: "int"}}}},
				{Name: "None", Fields: []*ast.ConstructorField{}},
			},
		},
	}

	gen := NewADTGenerator("game")
	code, err := gen.GenerateTypeDecl(decl)
	if err != nil {
		t.Fatalf("GenerateTypeDecl failed: %v", err)
	}

	codeStr := string(code)

	// Check for kind type
	if !strings.Contains(codeStr, "type OptionKind int") {
		t.Errorf("Missing kind type declaration")
	}

	// Check for kind constants
	if !strings.Contains(codeStr, "OptionKindSome OptionKind = iota") {
		t.Errorf("Missing OptionKindSome constant")
	}
	if !strings.Contains(codeStr, "OptionKindNone") {
		t.Errorf("Missing OptionKindNone constant")
	}

	// Check for variant structs
	if !strings.Contains(codeStr, "type OptionSome struct") {
		t.Errorf("Missing OptionSome struct")
	}
	if !strings.Contains(codeStr, "type OptionNone struct") {
		t.Errorf("Missing OptionNone struct")
	}

	// Check for main struct with Kind and variant pointers
	if !strings.Contains(codeStr, "type Option struct") {
		t.Errorf("Missing Option struct")
	}
	if !strings.Contains(codeStr, "Kind OptionKind") {
		t.Errorf("Missing Kind field in Option struct")
	}
	if !strings.Contains(codeStr, "Some *OptionSome") {
		t.Errorf("Missing Some pointer in Option struct")
	}
	if !strings.Contains(codeStr, "None *OptionNone") {
		t.Errorf("Missing None pointer in Option struct")
	}

	// Check for constructors
	if !strings.Contains(codeStr, "func NewOptionSome(") {
		t.Errorf("Missing NewOptionSome constructor")
	}
	if !strings.Contains(codeStr, "func NewOptionNone(") {
		t.Errorf("Missing NewOptionNone constructor")
	}

	// Check for Is methods
	if !strings.Contains(codeStr, "func (v *Option) IsSome() bool") {
		t.Errorf("Missing IsSome method")
	}
	if !strings.Contains(codeStr, "func (v *Option) IsNone() bool") {
		t.Errorf("Missing IsNone method")
	}
}

func TestGenerateSumType_Tree(t *testing.T) {
	// type Tree = | Leaf(int) | Node(Tree, int, Tree)
	decl := &ast.TypeDecl{
		Name: "Tree",
		Definition: &ast.AlgebraicType{
			Constructors: []*ast.Constructor{
				{Name: "Leaf", Fields: []*ast.ConstructorField{{Type: &ast.SimpleType{Name: "int"}}}},
				{Name: "Node", Fields: []*ast.ConstructorField{
					{Type: &ast.SimpleType{Name: "Tree"}},
					{Type: &ast.SimpleType{Name: "int"}},
					{Type: &ast.SimpleType{Name: "Tree"}},
				}},
			},
		},
	}

	gen := NewADTGenerator("game")
	code, err := gen.GenerateTypeDecl(decl)
	if err != nil {
		t.Fatalf("GenerateTypeDecl failed: %v", err)
	}

	codeStr := string(code)

	// Check for recursive type reference
	if !strings.Contains(codeStr, "type TreeNode struct") {
		t.Errorf("Missing TreeNode struct")
	}

	// Node should have Tree fields (Value0, Value1, Value2)
	if !strings.Contains(codeStr, "Value0 Tree") {
		t.Errorf("Missing Value0 (Tree) in TreeNode")
	}
	if !strings.Contains(codeStr, "Value1 int64") {
		t.Errorf("Missing Value1 (int) in TreeNode")
	}
	if !strings.Contains(codeStr, "Value2 Tree") {
		t.Errorf("Missing Value2 (Tree) in TreeNode")
	}
}

func TestGenerateRecordType(t *testing.T) {
	// type Point = { x: int, y: int }
	decl := &ast.TypeDecl{
		Name: "Point",
		Definition: &ast.RecordType{
			Fields: []*ast.RecordField{
				{Name: "x", Type: &ast.SimpleType{Name: "int"}},
				{Name: "y", Type: &ast.SimpleType{Name: "int"}},
			},
		},
	}

	gen := NewADTGenerator("game")
	code, err := gen.GenerateTypeDecl(decl)
	if err != nil {
		t.Fatalf("GenerateTypeDecl failed: %v", err)
	}

	codeStr := string(code)

	// Check for struct
	if !strings.Contains(codeStr, "type Point struct") {
		t.Errorf("Missing Point struct")
	}

	// Check for fields with PascalCase
	if !strings.Contains(codeStr, "X int64") {
		t.Errorf("Missing X field")
	}
	if !strings.Contains(codeStr, "Y int64") {
		t.Errorf("Missing Y field")
	}
}

func TestGenerateTypeAlias(t *testing.T) {
	// type Names = [string]
	decl := &ast.TypeDecl{
		Name: "Names",
		Definition: &ast.TypeAlias{
			Target: &ast.ListType{Element: &ast.SimpleType{Name: "string"}},
		},
	}

	gen := NewADTGenerator("game")
	code, err := gen.GenerateTypeDecl(decl)
	if err != nil {
		t.Fatalf("GenerateTypeDecl failed: %v", err)
	}

	codeStr := string(code)

	// Check for type alias
	if !strings.Contains(codeStr, "type Names = []string") {
		t.Errorf("Missing Names type alias, got: %s", codeStr)
	}
}

func TestGenerateMultipleTypes(t *testing.T) {
	decls := []*ast.TypeDecl{
		{
			Name: "Direction",
			Definition: &ast.AlgebraicType{
				Constructors: []*ast.Constructor{
					{Name: "North", Fields: []*ast.ConstructorField{}},
					{Name: "South", Fields: []*ast.ConstructorField{}},
					{Name: "East", Fields: []*ast.ConstructorField{}},
					{Name: "West", Fields: []*ast.ConstructorField{}},
				},
			},
		},
		{
			Name: "Position",
			Definition: &ast.RecordType{
				Fields: []*ast.RecordField{
					{Name: "x", Type: &ast.SimpleType{Name: "int"}},
					{Name: "y", Type: &ast.SimpleType{Name: "int"}},
				},
			},
		},
	}

	gen := NewADTGenerator("game")
	code, err := gen.GenerateTypeDecls(decls)
	if err != nil {
		t.Fatalf("GenerateTypeDecls failed: %v", err)
	}

	codeStr := string(code)

	// Check both types are present
	if !strings.Contains(codeStr, "type Direction struct") {
		t.Errorf("Missing Direction struct")
	}
	if !strings.Contains(codeStr, "type Position struct") {
		t.Errorf("Missing Position struct")
	}

	// Check Direction has all variants
	if !strings.Contains(codeStr, "DirectionKindNorth") {
		t.Errorf("Missing DirectionKindNorth")
	}
	if !strings.Contains(codeStr, "DirectionKindWest") {
		t.Errorf("Missing DirectionKindWest")
	}
}

func TestGeneratedCodeCompiles(t *testing.T) {
	// Generate a complex type and verify it's valid Go syntax
	decl := &ast.TypeDecl{
		Name: "DrawCmd",
		Definition: &ast.AlgebraicType{
			Constructors: []*ast.Constructor{
				{Name: "Sprite", Fields: []*ast.ConstructorField{
					{Type: &ast.SimpleType{Name: "int"}},    // x
					{Type: &ast.SimpleType{Name: "int"}},    // y
					{Type: &ast.SimpleType{Name: "string"}}, // texture
				}},
				{Name: "Rect", Fields: []*ast.ConstructorField{
					{Type: &ast.SimpleType{Name: "int"}}, // x
					{Type: &ast.SimpleType{Name: "int"}}, // y
					{Type: &ast.SimpleType{Name: "int"}}, // w
					{Type: &ast.SimpleType{Name: "int"}}, // h
				}},
				{Name: "Text", Fields: []*ast.ConstructorField{
					{Type: &ast.SimpleType{Name: "int"}},    // x
					{Type: &ast.SimpleType{Name: "int"}},    // y
					{Type: &ast.SimpleType{Name: "string"}}, // text
				}},
			},
		},
	}

	gen := NewADTGenerator("game")
	code, err := gen.GenerateTypeDecl(decl)
	if err != nil {
		t.Fatalf("GenerateTypeDecl failed: %v", err)
	}

	// If we got here, the code was formatted successfully by go/format
	// which validates it's syntactically correct Go

	codeStr := string(code)
	if !strings.Contains(codeStr, "package game") {
		t.Errorf("Missing package declaration")
	}
}

func TestMapASTType_Primitives(t *testing.T) {
	gen := NewADTGenerator("test")

	tests := []struct {
		input    ast.Type
		expected string
	}{
		{&ast.SimpleType{Name: "int"}, "int64"},
		{&ast.SimpleType{Name: "float"}, "float64"},
		{&ast.SimpleType{Name: "bool"}, "bool"},
		{&ast.SimpleType{Name: "string"}, "string"},
		{&ast.SimpleType{Name: "unit"}, "struct{}"},
		{&ast.SimpleType{Name: "Tree"}, "Tree"},
		{&ast.ListType{Element: &ast.SimpleType{Name: "int"}}, "[]int64"},
		{&ast.ListType{Element: &ast.SimpleType{Name: "string"}}, "[]string"},
	}

	for _, tt := range tests {
		result := gen.mapASTType(tt.input)
		if result != tt.expected {
			t.Errorf("mapASTType(%v) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
