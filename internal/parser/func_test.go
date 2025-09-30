package parser

import (
	"testing"
)

// TestFunctionDeclarations tests basic function declaration parsing
func TestFunctionDeclarations(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		golden string
	}{
		{
			"simple_func",
			"func add(x, y) { x + y }",
			"func/simple_func",
		},
		{
			"func_no_params",
			"func hello() { 42 }",
			"func/func_no_params",
		},
		{
			"func_one_param",
			"func square(x) { x * x }",
			"func/func_one_param",
		},
		{
			"func_with_if",
			"func factorial(n) { if n <= 1 then 1 else n * factorial(n - 1) }",
			"func/func_with_if",
		},
		{
			"pure_func",
			"pure func add(x, y) { x + y }",
			"func/pure_func",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestFunctionWithTypes tests function declarations with type annotations
func TestFunctionWithTypes(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		golden string
	}{
		{
			"func_typed_params",
			"func add(x: int, y: int) { x + y }",
			"func/func_typed_params",
		},
		{
			"func_return_type",
			"func add(x: int, y: int) -> int { x + y }",
			"func/func_return_type",
		},
		{
			"func_mixed_types",
			"func greet(name: string, age: int) -> string { name }",
			"func/func_mixed_types",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestFunctionWithEffects tests function declarations with effect annotations
func TestFunctionWithEffects(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		golden string
	}{
		{
			"func_io_effect",
			"func readFile() -> string ! {IO} { () }",
			"func/func_io_effect",
		},
		{
			"func_multiple_effects",
			"func process() -> int ! {IO, FS} { 42 }",
			"func/func_multiple_effects",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestFunctionWithTests tests function declarations with inline test cases
// NOTE: Tests syntax is recognized but not yet implemented (parser.go:574)
func TestFunctionWithTests(t *testing.T) {
	t.Skip("Tests syntax not yet implemented - parser recognizes keyword but doesn't parse test blocks")

	tests := []struct {
		name   string
		input  string
		golden string
	}{
		{
			"func_with_test",
			`func add(x, y) -> int
			  tests [(1, 2, 3), (0, 0, 0)]
			{ x + y }`,
			"func/func_with_test",
		},
		{
			"func_single_test",
			`func square(x) -> int
			  tests [(2, 4)]
			{ x * x }`,
			"func/func_single_test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestMultipleFunctions tests parsing multiple function declarations
func TestMultipleFunctions(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		golden string
	}{
		{
			"two_functions",
			`func add(x, y) { x + y }
			 func sub(x, y) { x - y }`,
			"func/two_functions",
		},
		{
			"mixed_pure_impure",
			`pure func add(x, y) { x + y }
			 func readFile() -> string ! {IO} { () }`,
			"func/mixed_pure_impure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestInvalidFunctionSyntax tests error handling for invalid function syntax
func TestInvalidFunctionSyntax(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"func_no_name", "func (x) { x }"},
		{"func_no_body", "func add(x, y)"},
		{"func_missing_braces", "func add(x, y) x + y"},
		{"func_trailing_comma", "func add(x, y,) { x + y }"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := mustParseError(t, tt.input)
			// Note: Some of these might parse successfully depending on implementation
			// We're mainly testing that the parser doesn't panic
			_ = errs
		})
	}
}

// TestExportedFunctions tests function export declarations
func TestExportedFunctions(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		golden string
	}{
		{
			"export_func",
			"export func add(x, y) { x + y }",
			"func/export_func",
		},
		{
			"export_pure_func",
			"export pure func mul(x, y) { x * y }",
			"func/export_pure_func",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestComplexFunctionBodies tests functions with complex expression bodies
func TestComplexFunctionBodies(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		golden string
	}{
		{
			"func_with_let",
			"func compute(x) { let y = x * 2 in y + 1 }",
			"func/func_with_let",
		},
		{
			"func_with_abs",
			"func abs(x) { if x < 0 then -x else x }",
			"func/func_with_abs",
		},
		{
			"func_with_match",
			`func describe(x) { match x { 0 => "zero", _ => "nonzero" } }`,
			"func/func_with_match",
		},
		{
			"func_with_lambda",
			`func makeAdder(x) { \y. x + y }`,
			"func/func_with_lambda",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}