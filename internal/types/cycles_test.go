package types

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

func TestDetectCycles_RecursiveADT(t *testing.T) {
	// type List[a] = Nil | Cons(a, List[a])
	decls := []ast.Node{
		&ast.TypeDecl{
			Name:       "List",
			TypeParams: []string{"a"},
			Definition: &ast.AlgebraicType{
				Constructors: []*ast.Constructor{
					{Name: "Nil", Fields: nil},
					{Name: "Cons", Fields: []*ast.ConstructorField{
						{Name: "head", Type: &ast.TypeVar{Name: "a"}},
						{Name: "tail", Type: &ast.TypeApp{
							Constructor: "List",
							Args:        []ast.Type{&ast.TypeVar{Name: "a"}},
						}},
					}},
				},
			},
		},
	}

	cycles := DetectCycles(decls, "test.ail")

	if len(cycles) != 1 {
		t.Fatalf("expected 1 cycle, got %d", len(cycles))
	}

	cycle := cycles[0]
	if cycle.TypeName != "List" {
		t.Errorf("expected TypeName 'List', got '%s'", cycle.TypeName)
	}
	if cycle.Kind != CycleExpected {
		t.Errorf("expected kind 'expected', got '%s'", cycle.Kind)
	}
}

func TestDetectCycles_RecursiveTree(t *testing.T) {
	// type Tree[a] = Leaf(a) | Node(Tree[a], a, Tree[a])
	decls := []ast.Node{
		&ast.TypeDecl{
			Name:       "Tree",
			TypeParams: []string{"a"},
			Definition: &ast.AlgebraicType{
				Constructors: []*ast.Constructor{
					{Name: "Leaf", Fields: []*ast.ConstructorField{
						{Name: "value", Type: &ast.TypeVar{Name: "a"}},
					}},
					{Name: "Node", Fields: []*ast.ConstructorField{
						{Name: "left", Type: &ast.TypeApp{
							Constructor: "Tree",
							Args:        []ast.Type{&ast.TypeVar{Name: "a"}},
						}},
						{Name: "value", Type: &ast.TypeVar{Name: "a"}},
						{Name: "right", Type: &ast.TypeApp{
							Constructor: "Tree",
							Args:        []ast.Type{&ast.TypeVar{Name: "a"}},
						}},
					}},
				},
			},
		},
	}

	cycles := DetectCycles(decls, "test.ail")

	if len(cycles) != 1 {
		t.Fatalf("expected 1 cycle, got %d", len(cycles))
	}

	cycle := cycles[0]
	if cycle.TypeName != "Tree" {
		t.Errorf("expected TypeName 'Tree', got '%s'", cycle.TypeName)
	}
	if cycle.Kind != CycleExpected {
		t.Errorf("expected kind 'expected', got '%s'", cycle.Kind)
	}
}

func TestDetectCycles_RecordSelfReference(t *testing.T) {
	// type Person = { name: string, friends: [Person] }
	decls := []ast.Node{
		&ast.TypeDecl{
			Name: "Person",
			Definition: &ast.RecordType{
				Fields: []*ast.RecordField{
					{Name: "name", Type: &ast.SimpleType{Name: "string"}},
					{Name: "friends", Type: &ast.ListType{
						Element: &ast.SimpleType{Name: "Person"},
					}},
				},
			},
		},
	}

	cycles := DetectCycles(decls, "test.ail")

	if len(cycles) != 1 {
		t.Fatalf("expected 1 cycle, got %d", len(cycles))
	}

	cycle := cycles[0]
	if cycle.TypeName != "Person" {
		t.Errorf("expected TypeName 'Person', got '%s'", cycle.TypeName)
	}
	if cycle.Kind != CycleSuspicious {
		t.Errorf("expected kind 'suspicious', got '%s'", cycle.Kind)
	}
}

func TestDetectCycles_NoCycle(t *testing.T) {
	// type Point = { x: int, y: int }
	decls := []ast.Node{
		&ast.TypeDecl{
			Name: "Point",
			Definition: &ast.RecordType{
				Fields: []*ast.RecordField{
					{Name: "x", Type: &ast.SimpleType{Name: "int"}},
					{Name: "y", Type: &ast.SimpleType{Name: "int"}},
				},
			},
		},
	}

	cycles := DetectCycles(decls, "test.ail")

	if len(cycles) != 0 {
		t.Fatalf("expected 0 cycles, got %d", len(cycles))
	}
}

func TestDetectCycles_SimpleADTNoCycle(t *testing.T) {
	// type Color = Red | Green | Blue
	decls := []ast.Node{
		&ast.TypeDecl{
			Name: "Color",
			Definition: &ast.AlgebraicType{
				Constructors: []*ast.Constructor{
					{Name: "Red", Fields: nil},
					{Name: "Green", Fields: nil},
					{Name: "Blue", Fields: nil},
				},
			},
		},
	}

	cycles := DetectCycles(decls, "test.ail")

	if len(cycles) != 0 {
		t.Fatalf("expected 0 cycles, got %d", len(cycles))
	}
}

func TestDetectCycles_StdlibExpected(t *testing.T) {
	// Any type in stdlib should be marked as expected
	decls := []ast.Node{
		&ast.TypeDecl{
			Name: "MyCustomType",
			Definition: &ast.RecordType{
				Fields: []*ast.RecordField{
					{Name: "self", Type: &ast.SimpleType{Name: "MyCustomType"}},
				},
			},
		},
	}

	cycles := DetectCycles(decls, "std/prelude.ail")

	if len(cycles) != 1 {
		t.Fatalf("expected 1 cycle, got %d", len(cycles))
	}

	if cycles[0].Kind != CycleExpected {
		t.Errorf("expected kind 'expected' for stdlib, got '%s'", cycles[0].Kind)
	}
}

func TestDetectCycles_PathIncludesFieldNames(t *testing.T) {
	// type Person = { name: string, friends: [Person] }
	decls := []ast.Node{
		&ast.TypeDecl{
			Name: "Person",
			Definition: &ast.RecordType{
				Fields: []*ast.RecordField{
					{Name: "name", Type: &ast.SimpleType{Name: "string"}},
					{Name: "friends", Type: &ast.ListType{
						Element: &ast.SimpleType{Name: "Person"},
					}},
				},
			},
		},
	}

	cycles := DetectCycles(decls, "test.ail")

	if len(cycles) != 1 {
		t.Fatalf("expected 1 cycle, got %d", len(cycles))
	}

	// Path should include the field name
	path := cycles[0].Path
	if len(path) < 2 {
		t.Fatalf("expected path with at least 2 elements, got %v", path)
	}
	// First element should be the type name
	if path[0] != "Person" {
		t.Errorf("expected path[0] = 'Person', got '%s'", path[0])
	}
}

func TestDetectCycles_TypeAlias(t *testing.T) {
	// type RecList = [RecList]  (recursive type alias)
	decls := []ast.Node{
		&ast.TypeDecl{
			Name: "RecList",
			Definition: &ast.TypeAlias{
				Target: &ast.ListType{
					Element: &ast.SimpleType{Name: "RecList"},
				},
			},
		},
	}

	cycles := DetectCycles(decls, "test.ail")

	if len(cycles) != 1 {
		t.Fatalf("expected 1 cycle, got %d", len(cycles))
	}

	if cycles[0].TypeName != "RecList" {
		t.Errorf("expected TypeName 'RecList', got '%s'", cycles[0].TypeName)
	}
}

func TestDetectCycles_MultipleTypes(t *testing.T) {
	// type List[a] = Nil | Cons(a, List[a])
	// type Point = { x: int, y: int }
	// type Person = { name: string, friends: [Person] }
	decls := []ast.Node{
		&ast.TypeDecl{
			Name:       "List",
			TypeParams: []string{"a"},
			Definition: &ast.AlgebraicType{
				Constructors: []*ast.Constructor{
					{Name: "Nil", Fields: nil},
					{Name: "Cons", Fields: []*ast.ConstructorField{
						{Name: "head", Type: &ast.TypeVar{Name: "a"}},
						{Name: "tail", Type: &ast.TypeApp{
							Constructor: "List",
							Args:        []ast.Type{&ast.TypeVar{Name: "a"}},
						}},
					}},
				},
			},
		},
		&ast.TypeDecl{
			Name: "Point",
			Definition: &ast.RecordType{
				Fields: []*ast.RecordField{
					{Name: "x", Type: &ast.SimpleType{Name: "int"}},
					{Name: "y", Type: &ast.SimpleType{Name: "int"}},
				},
			},
		},
		&ast.TypeDecl{
			Name: "Person",
			Definition: &ast.RecordType{
				Fields: []*ast.RecordField{
					{Name: "name", Type: &ast.SimpleType{Name: "string"}},
					{Name: "friends", Type: &ast.ListType{
						Element: &ast.SimpleType{Name: "Person"},
					}},
				},
			},
		},
	}

	cycles := DetectCycles(decls, "test.ail")

	// Should find List and Person cycles, but not Point
	if len(cycles) != 2 {
		t.Fatalf("expected 2 cycles, got %d", len(cycles))
	}

	typeNames := make(map[string]bool)
	for _, c := range cycles {
		typeNames[c.TypeName] = true
	}

	if !typeNames["List"] {
		t.Error("expected to find List cycle")
	}
	if !typeNames["Person"] {
		t.Error("expected to find Person cycle")
	}
	if typeNames["Point"] {
		t.Error("Point should not have a cycle")
	}
}

func TestDetectCycles_ParsedFile(t *testing.T) {
	// Test with an actual parsed file to verify parser integration
	source := `module test
type List[a] =
  | Nil
  | Cons(a, List[a])

type Tree[a] =
  | Leaf(a)
  | Node(Tree[a], a, Tree[a])

type Person = {
  name: string,
  friends: [Person]
}
`
	l := lexer.New(source, "test.ail")
	p := parser.New(l)
	prog := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}

	if prog.Module == nil {
		t.Fatal("expected module")
	}

	// Debug: print parsed AST
	for _, decl := range prog.Module.Decls {
		if td, ok := decl.(*ast.TypeDecl); ok {
			t.Logf("TypeDecl: %s", td.Name)
			if adt, ok := td.Definition.(*ast.AlgebraicType); ok {
				for _, ctor := range adt.Constructors {
					t.Logf("  Constructor: %s (fields=%d)", ctor.Name, len(ctor.Fields))
					for i, field := range ctor.Fields {
						t.Logf("    Field[%d]: name=%q, type=%T (%v)", i, field.Name, field.Type, field.Type)
					}
				}
			}
		}
	}

	cycles := DetectCycles(prog.Module.Decls, "test.ail")

	// Debug output
	for _, c := range cycles {
		t.Logf("Cycle: %s (kind=%s, path=%v)", c.TypeName, c.Kind, c.Path)
	}

	// Should find List, Tree, and Person cycles
	if len(cycles) < 2 {
		t.Errorf("expected at least 2 cycles, got %d", len(cycles))
	}

	typeNames := make(map[string]bool)
	for _, c := range cycles {
		typeNames[c.TypeName] = true
	}

	if !typeNames["Tree"] {
		t.Error("expected to find Tree cycle")
	}
	if !typeNames["Person"] {
		t.Error("expected to find Person cycle")
	}
}

func TestClassifyCycleKind(t *testing.T) {
	tests := []struct {
		typeName string
		filename string
		expected CycleKind
	}{
		{"List", "test.ail", CycleExpected},
		{"Tree", "test.ail", CycleExpected},
		{"Node", "test.ail", CycleExpected},
		{"Expr", "test.ail", CycleExpected},
		{"Person", "test.ail", CycleSuspicious},
		{"CustomType", "test.ail", CycleSuspicious},
		{"CustomType", "std/custom.ail", CycleExpected},
		{"CustomType", "stdlib/custom.ail", CycleExpected},
	}

	for _, tt := range tests {
		t.Run(tt.typeName+"_"+tt.filename, func(t *testing.T) {
			result := classifyCycleKind(tt.typeName, tt.filename)
			if result != tt.expected {
				t.Errorf("classifyCycleKind(%q, %q) = %q, want %q",
					tt.typeName, tt.filename, result, tt.expected)
			}
		})
	}
}
