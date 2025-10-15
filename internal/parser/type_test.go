package parser

import (
	"testing"
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
