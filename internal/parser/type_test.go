package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunholo/ailang/internal/ast"
)

// TestTypeAliases tests basic type alias declarations
func TestTypeAliases(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		golden string
	}{
		{
			"simple_alias",
			"type UserId = int",
			"type/simple_alias",
		},
		// TODO: Type aliases to complex types not yet fully supported
		// These parse as sum types currently - deferred to future milestone
		// {
		// 	"alias_to_list",
		// 	"type Names = [string]",
		// 	"type/alias_to_list",
		// },
		// {
		// 	"alias_to_tuple",
		// 	"type Point = (int, int)",
		// 	"type/alias_to_tuple",
		// },
		// {
		// 	"alias_to_function",
		// 	"type Predicate = (int) -> bool",
		// 	"type/alias_to_function",
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestRecordTypes tests record type declarations
func TestRecordTypes(t *testing.T) {

	tests := []struct {
		name   string
		input  string
		golden string
	}{
		{
			"simple_record",
			"type Point = { x: int, y: int }",
			"type/simple_record",
		},
		{
			"nested_record",
			"type User = { name: string, address: { street: string, city: string } }",
			"type/nested_record",
		},
		{
			"record_with_optional",
			"type Config = { host: string, port: Option[int] }",
			"type/record_with_optional",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestSumTypes tests sum/variant type declarations
func TestSumTypes(t *testing.T) {

	tests := []struct {
		name   string
		input  string
		golden string
	}{
		{
			"simple_enum",
			"type Color = Red | Green | Blue",
			"type/simple_enum",
		},
		// TODO: Type variables in constructor fields need more work
		// {
		// 	"enum_with_data",
		// 	"type Option[a] = Some(a) | None",
		// 	"type/enum_with_data",
		// },
		// {
		// 	"complex_variant",
		// 	"type Result[a, e] = Ok(a) | Err(e)",
		// 	"type/complex_variant",
		// },
		{
			"multiple_fields",
			"type Shape = Circle(int) | Rectangle(int, int) | Point",
			"type/multiple_fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestGenericTypes tests type declarations with type parameters
func TestGenericTypes(t *testing.T) {

	tests := []struct {
		name   string
		input  string
		golden string
	}{
		{
			"single_param",
			"type Box[a] = { value: a }",
			"type/single_param",
		},
		{
			"multiple_params",
			"type Pair[a, b] = { first: a, second: b }",
			"type/multiple_params",
		},
		{
			"nested_generic",
			"type Tree[a] = Leaf(a) | Node(Tree[a], Tree[a])",
			"type/nested_generic",
		},
		// TODO: Type constraints with 'where' not yet supported
		// {
		// 	"constrained_generic",
		// 	"type Comparable[a] where Eq[a] = { value: a }",
		// 	"type/constrained_generic",
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestExportedTypes tests type export declarations
func TestExportedTypes(t *testing.T) {

	tests := []struct {
		name   string
		input  string
		golden string
	}{
		{
			"export_alias",
			"export type UserId = int",
			"type/export_alias",
		},
		{
			"export_record",
			"export type Point = { x: int, y: int }",
			"type/export_record",
		},
		{
			"export_sum",
			"export type Option[a] = Some(a) | None",
			"type/export_sum",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestComplexTypes tests complex type declarations
func TestComplexTypes(t *testing.T) {

	tests := []struct {
		name   string
		input  string
		golden string
	}{
		// TODO: Function types not yet supported - would need separate TypeAlias variant
		// {
		// 	"function_type",
		// 	"type Handler = (Request) -> Response",
		// 	"type/function_type",
		// },
		// {
		// 	"function_with_effects",
		// 	"type ReadFile = (string) -> string ! {IO}",
		// 	"type/function_with_effects",
		// },
		// TODO: List type aliases not yet supported
		// {
		// 	"nested_containers",
		// 	"type Matrix = [[int]]",
		// 	"type/nested_containers",
		// },
		{
			"map_type",
			"type Config = Map[string, int]",
			"type/map_type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestInvalidTypeSyntax tests error handling for invalid type syntax
func TestInvalidTypeSyntax(t *testing.T) {

	tests := []struct {
		name        string
		input       string
		expectError bool // true if we expect parse errors
	}{
		{"type_no_name", "type = int", true},
		{"type_no_body", "type Foo", true},
		{"type_trailing_pipe", "type Color = Red | Green |", true},
		{"type_empty_record", "type Empty = { }", false}, // Empty records are allowed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectError {
				errs := mustParseError(t, tt.input)
				// We're mainly testing that the parser doesn't panic
				_ = errs
			} else {
				// Should parse successfully
				_ = mustParse(t, tt.input)
			}
		})
	}
}

// TestMultipleTypes tests parsing multiple type declarations
func TestMultipleTypes(t *testing.T) {

	tests := []struct {
		name   string
		input  string
		golden string
	}{
		{
			"two_types",
			`type Point = { x: int, y: int }
			 type Color = Red | Green | Blue`,
			"type/two_types",
		},
		{
			"dependent_types",
			`type UserId = int
			 type User = { id: UserId, name: string }`,
			"type/dependent_types",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestArrayListTypeApplication tests M-TYPE1: Array[T] and List[T] parse to correct AST nodes
// This ensures array/list type applications preserve element types for proper type checking.
func TestArrayListTypeApplication(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedType interface{} // Expected AST node type
		checkElement string      // Expected element type name
	}{
		{
			name:         "Array[int] parses to ArrayType",
			input:        "type T = Foo(Array[int])",
			expectedType: &ast.ArrayType{},
			checkElement: "int",
		},
		{
			name:         "Array[Direction] parses to ArrayType",
			input:        "type T = Foo(Array[Direction])",
			expectedType: &ast.ArrayType{},
			checkElement: "Direction",
		},
		{
			name:         "List[string] parses to ListType",
			input:        "type T = Foo(List[string])",
			expectedType: &ast.ListType{},
			checkElement: "string",
		},
		{
			name:         "List[UserType] parses to ListType",
			input:        "type T = Foo(List[UserType])",
			expectedType: &ast.ListType{},
			checkElement: "UserType",
		},
		{
			name:         "Option[int] remains SimpleType (not special-cased)",
			input:        "type T = Foo(Option[int])",
			expectedType: &ast.SimpleType{},
			checkElement: "", // SimpleType doesn't have element
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := mustParse(t, tt.input)
			require.NotNil(t, prog.File, "expected File")
			require.Len(t, prog.File.Statements, 1)

			// Get the type declaration
			typeDecl, ok := prog.File.Statements[0].(*ast.TypeDecl)
			require.True(t, ok, "expected TypeDecl")

			// Get the algebraic type (sum type)
			algType, ok := typeDecl.Definition.(*ast.AlgebraicType)
			require.True(t, ok, "expected AlgebraicType")
			require.Len(t, algType.Constructors, 1, "expected 1 constructor")

			// Get the constructor's field type
			ctor := algType.Constructors[0]
			require.Len(t, ctor.Fields, 1, "expected 1 field")
			fieldType := ctor.Fields[0].Type

			// Check the field type matches expected
			switch expected := tt.expectedType.(type) {
			case *ast.ArrayType:
				arrayType, ok := fieldType.(*ast.ArrayType)
				assert.True(t, ok, "expected ArrayType, got %T", fieldType)
				if ok && tt.checkElement != "" {
					elemSimple, ok := arrayType.Element.(*ast.SimpleType)
					assert.True(t, ok, "expected SimpleType element")
					if ok {
						assert.Equal(t, tt.checkElement, elemSimple.Name)
					}
				}
			case *ast.ListType:
				listType, ok := fieldType.(*ast.ListType)
				assert.True(t, ok, "expected ListType, got %T", fieldType)
				if ok && tt.checkElement != "" {
					elemSimple, ok := listType.Element.(*ast.SimpleType)
					assert.True(t, ok, "expected SimpleType element")
					if ok {
						assert.Equal(t, tt.checkElement, elemSimple.Name)
					}
				}
			case *ast.SimpleType:
				simpleType, ok := fieldType.(*ast.SimpleType)
				assert.True(t, ok, "expected SimpleType, got %T", fieldType)
				_ = simpleType
				_ = expected
			}
		})
	}
}

// TestADTWithArrayField tests the full ADT-with-array scenario from M-TYPE1
func TestADTWithArrayField(t *testing.T) {
	input := `type Direction = North | South | East | West
type AIBehavior = PatternPatrol(Array[Direction]) | RandomWander`

	prog := mustParse(t, input)
	require.NotNil(t, prog.File, "expected File")
	require.Len(t, prog.File.Statements, 2)

	// Second type is AIBehavior
	aiTypeDecl, ok := prog.File.Statements[1].(*ast.TypeDecl)
	require.True(t, ok)
	assert.Equal(t, "AIBehavior", aiTypeDecl.Name)

	algType, ok := aiTypeDecl.Definition.(*ast.AlgebraicType)
	require.True(t, ok)
	require.Len(t, algType.Constructors, 2)

	// First constructor is PatternPatrol(Array[Direction])
	patrolCtor := algType.Constructors[0]
	assert.Equal(t, "PatternPatrol", patrolCtor.Name)
	require.Len(t, patrolCtor.Fields, 1)

	// Field type should be ArrayType with Direction element
	fieldType := patrolCtor.Fields[0].Type
	arrayType, ok := fieldType.(*ast.ArrayType)
	require.True(t, ok, "expected ArrayType, got %T", fieldType)

	elemType, ok := arrayType.Element.(*ast.SimpleType)
	require.True(t, ok, "expected SimpleType element")
	assert.Equal(t, "Direction", elemType.Name)
}
