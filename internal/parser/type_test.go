package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
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
			// DX-17 Phase 2: List[T] now parses to TypeApp for uniform representation
			name:         "List[string] parses to TypeApp",
			input:        "type T = Foo(List[string])",
			expectedType: &ast.TypeApp{},
			checkElement: "string",
		},
		{
			// DX-17 Phase 2: List[T] now parses to TypeApp for uniform representation
			name:         "List[UserType] parses to TypeApp",
			input:        "type T = Foo(List[UserType])",
			expectedType: &ast.TypeApp{},
			checkElement: "UserType",
		},
		{
			// M-TAPP-FIX: Option[int] now parses to TypeApp (correct behavior for type annotations)
			name:         "Option[int] parses to TypeApp (generic type application)",
			input:        "type T = Foo(Option[int])",
			expectedType: &ast.TypeApp{},
			checkElement: "int", // TypeApp has args
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
			// DX-17 Phase 2: ListType case removed - List[T] now uses TypeApp
			case *ast.SimpleType:
				simpleType, ok := fieldType.(*ast.SimpleType)
				assert.True(t, ok, "expected SimpleType, got %T", fieldType)
				_ = simpleType
				_ = expected
			// M-TAPP-FIX: Handle TypeApp for generic type applications
			case *ast.TypeApp:
				typeApp, ok := fieldType.(*ast.TypeApp)
				assert.True(t, ok, "expected TypeApp, got %T", fieldType)
				if ok && tt.checkElement != "" {
					require.Len(t, typeApp.Args, 1, "expected 1 type argument")
					elemSimple, ok := typeApp.Args[0].(*ast.SimpleType)
					assert.True(t, ok, "expected SimpleType element in TypeApp, got %T", typeApp.Args[0])
					if ok {
						assert.Equal(t, tt.checkElement, elemSimple.Name)
					}
				}
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

// TestDerivingEq tests the deriving (Eq) syntax for ADT types (M-DX19)
func TestDerivingEq(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectDeriving []ast.DeriveKind
		expectTypeName string
	}{
		{
			name:           "simple enum with deriving Eq",
			input:          "type Color = Red | Green | Blue deriving (Eq)",
			expectDeriving: []ast.DeriveKind{ast.DeriveEq},
			expectTypeName: "Color",
		},
		{
			name:           "ADT with fields deriving Eq",
			input:          "type Tree = Leaf(int) | Node(Tree, int, Tree) deriving (Eq)",
			expectDeriving: []ast.DeriveKind{ast.DeriveEq},
			expectTypeName: "Tree",
		},
		{
			name:           "record type deriving Eq",
			input:          "type Point = { x: int, y: int } deriving (Eq)",
			expectDeriving: []ast.DeriveKind{ast.DeriveEq},
			expectTypeName: "Point",
		},
		{
			name:           "type without deriving",
			input:          "type Color = Red | Green | Blue",
			expectDeriving: nil,
			expectTypeName: "Color",
		},
		{
			name:           "single constructor with deriving",
			input:          "type Wrapper = Wrap(int) deriving (Eq)",
			expectDeriving: []ast.DeriveKind{ast.DeriveEq},
			expectTypeName: "Wrapper",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := mustParse(t, tt.input)
			require.NotNil(t, prog.File, "expected File")
			require.Len(t, prog.File.Statements, 1, "expected 1 statement")

			typeDecl, ok := prog.File.Statements[0].(*ast.TypeDecl)
			require.True(t, ok, "expected TypeDecl, got %T", prog.File.Statements[0])

			assert.Equal(t, tt.expectTypeName, typeDecl.Name)

			if tt.expectDeriving == nil {
				assert.Empty(t, typeDecl.Deriving, "expected no deriving clause")
			} else {
				assert.Equal(t, tt.expectDeriving, typeDecl.Deriving, "expected deriving clause to match")
			}
		})
	}
}

// TestOpenRecordTypes tests M-GAP4: open record type syntax
func TestOpenRecordTypes(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantRow    bool   // expect row variable in parsed record type
		wantRowVar string // expected row variable name (empty if generated)
	}{
		{
			name:       "explicit row variable",
			input:      "pure func f(x: {name: string | r}) -> string = x.name",
			wantRow:    true,
			wantRowVar: "r",
		},
		{
			name:       "explicit row variable with multiple fields",
			input:      "pure func f(x: {name: string, age: int | rest}) -> string = x.name",
			wantRow:    true,
			wantRowVar: "rest",
		},
		{
			name:       "ellipsis sugar syntax",
			input:      "pure func f(x: {name: string, ...}) -> string = x.name",
			wantRow:    true,
			wantRowVar: "_r", // generated names start with _r
		},
		{
			name:       "ellipsis sugar with multiple fields",
			input:      "pure func f(x: {name: string, age: int, ...}) -> string = x.name",
			wantRow:    true,
			wantRowVar: "_r",
		},
		{
			name:       "exact record (no row)",
			input:      "pure func f(x: {name: string}) -> string = x.name",
			wantRow:    false,
			wantRowVar: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := mustParse(t, tt.input)
			require.NotNil(t, prog.File, "expected File")
			require.Len(t, prog.File.Funcs, 1, "expected 1 function")

			// Get the function declaration
			funcDecl := prog.File.Funcs[0]
			require.NotNil(t, funcDecl, "expected FuncDecl")

			// Get the first parameter type
			require.Len(t, funcDecl.Params, 1, "expected 1 parameter")
			paramType := funcDecl.Params[0].Type

			// Should be a RecordType
			recordType, ok := paramType.(*ast.RecordType)
			require.True(t, ok, "expected RecordType, got %T", paramType)

			if tt.wantRow {
				assert.NotNil(t, recordType.Row, "expected row variable")
				if recordType.Row != nil && tt.wantRowVar != "" {
					if tt.wantRowVar == "_r" {
						// Generated name should start with _r
						assert.True(t, len(recordType.Row.Name) >= 2 && recordType.Row.Name[:2] == "_r",
							"expected generated row var name starting with _r, got %s", recordType.Row.Name)
					} else {
						assert.Equal(t, tt.wantRowVar, recordType.Row.Name)
					}
				}
			} else {
				assert.Nil(t, recordType.Row, "expected no row variable")
			}
		})
	}
}

// TestEllipsisSugarMarked tests that ellipsis sugar sets the SugarUsed flag
func TestEllipsisSugarMarked(t *testing.T) {
	input := "pure func f(x: {name: string, ...}) -> string = x.name"
	l := lexer.New(input, "test.ail")
	p := New(l)
	_ = p.Parse()
	assert.True(t, p.SugarUsed(), "expected SugarUsed to be true after parsing ellipsis syntax")
}

// TestDerivingEqErrors tests error handling for invalid deriving syntax
func TestDerivingEqErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "unsupported derive",
			input: "type Color = Red deriving (Ord)",
		},
		{
			name:  "missing lparen",
			input: "type Color = Red deriving Eq)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := mustParseError(t, tt.input)
			// We're mainly testing that the parser doesn't panic
			_ = errs
		})
	}
}
