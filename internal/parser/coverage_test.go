package parser

import (
	"testing"
)

// TestCharLiterals tests parsing of character literals
func TestCharLiterals(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		golden string
	}{
		{"char_simple", "'a'", "expr/char_simple"},
		{"char_digit", "'5'", "expr/char_digit"},
		{"char_escape_n", "'\\n'", "expr/char_escape_n"},
		{"char_escape_t", "'\\t'", "expr/char_escape_t"},
		{"char_unicode", "'λ'", "expr/char_unicode"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestBackslashLambdas tests backslash lambda parsing
// Note: parseLambda and parsePureLambda are already covered by other tests
func TestBackslashLambdas(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		golden string
	}{
		// Backslash lambda syntax: \x. x + 1
		{"lambda_curried_two_params", "\\x y. x + y", "expr/lambda_curried_two_params"},
		{"lambda_curried_three_params", "\\x y z. x + y + z", "expr/lambda_curried_three_params"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// NOTE: The following functions are at 0% coverage but are not fully implemented:
// - parseSendExpression (CSP <- operator not wired up)
// - parseClassDeclaration/parseInstanceDeclaration (type class syntax incomplete)
// - parseRecordPattern (record pattern syntax not complete)
// These will be tested when the features are fully implemented in v0.2+

// NOTE: Test utility functions (assertHasErrorCode, assertHasCode, assertErrorCount, assertContains)
// are used throughout the test suite and get covered by other tests. They don't need dedicated
// tests as they're just test helpers, not parser functionality.

// TestEdgeCases tests various edge cases to improve coverage
func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		golden string
	}{
		// Prefix operators
		{"unary_minus_complex", "-(x + y)", "expr/unary_minus_complex"},
		{"unary_not_nested", "!(!x)", "expr/unary_not_nested"},

		// Call arguments edge cases
		{"call_complex_args", "f(x + 1, g(y), z * 2)", "expr/call_complex_args"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}
