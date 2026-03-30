package parser

import (
	"fmt"
	"testing"
)

// TestOperatorPrecedence tests operator precedence and associativity
// using table-driven tests with expected parenthesized forms
func TestOperatorPrecedence(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string // Fully parenthesized form
	}{
		// Arithmetic precedence: * / % > + -
		{"add_vs_multiply", "1 + 2 * 3", "(1 + (2 * 3))"},
		{"multiply_vs_add", "2 * 3 + 1", "((2 * 3) + 1)"},
		{"subtract_vs_multiply", "10 - 2 * 3", "(10 - (2 * 3))"},
		{"divide_vs_add", "10 / 2 + 3", "((10 / 2) + 3)"},
		{"modulo_vs_add", "10 % 3 + 1", "((10 % 3) + 1)"},

		// Arithmetic left-associativity
		{"add_left_assoc", "1 + 2 + 3", "((1 + 2) + 3)"},
		{"subtract_left_assoc", "10 - 5 - 2", "((10 - 5) - 2)"},
		{"multiply_left_assoc", "2 * 3 * 4", "((2 * 3) * 4)"},
		{"divide_left_assoc", "12 / 3 / 2", "((12 / 3) / 2)"},

		// Complex arithmetic
		{"complex_arith_1", "1 + 2 * 3 + 4", "((1 + (2 * 3)) + 4)"},
		{"complex_arith_2", "2 * 3 + 4 * 5", "((2 * 3) + (4 * 5))"},
		{"complex_arith_3", "10 - 2 * 3 + 1", "((10 - (2 * 3)) + 1)"},

		// Comparison operators (lower precedence than arithmetic)
		{"compare_vs_add", "1 + 2 < 3 + 4", "((1 + 2) < (3 + 4))"},
		{"compare_vs_multiply", "2 * 3 == 3 * 2", "((2 * 3) == (3 * 2))"},
		{"compare_chain", "x < y && y < z", "((x < y) && (y < z))"},

		// Logical operators: && > ||
		{"and_vs_or", "x || y && z", "(x || (y && z))"},
		{"or_vs_and", "x && y || z", "((x && y) || z)"},

		// Logical left-associativity
		{"and_left_assoc", "a && b && c", "((a && b) && c)"},
		{"or_left_assoc", "a || b || c", "((a || b) || c)"},

		// Complex logical
		{"complex_logical_1", "a && b || c && d", "((a && b) || (c && d))"}, // && binds tighter
		{"complex_logical_2", "a || b && c || d", "((a || (b && c)) || d)"},

		// String concatenation (between shift and additive)
		{"concat_vs_compare", `"a" ++ "b" == "c"`, `((a ++ b) == c)`}, // Strings show without quotes in paren form

		// Mixed precedence chains
		{"mixed_1", "1 + 2 * 3 < 4 + 5", "((1 + (2 * 3)) < (4 + 5))"},
		{"mixed_2", "x < y && a + b > c", "((x < y) && ((a + b) > c))"},
		{"mixed_3", "a * b + c * d == e", "(((a * b) + (c * d)) == e)"},

		// Bitwise operator precedence (C-standard bands)
		// & and ^ are BELOW comparison (C convention)
		{"bitand_vs_eq", "8 & 3 == 0", "(8 & (3 == 0))"},             // C: & below ==
		{"bitxor_vs_eq", "a ^ b == c", "(a ^ (b == c))"},             // C: ^ below ==
		{"bitand_vs_xor", "a & b ^ c", "((a & b) ^ c)"},              // & binds tighter than ^
		{"shift_vs_add", "1 << 2 + 1", "(1 << (2 + 1))"},             // << below +
		{"shift_vs_multiply", "1 << 2 * 3", "(1 << (2 * 3))"},        // << below *
		{"not_vs_add", "~1 + 2", "((~1) + 2)"},                       // ~ is prefix, tightest
		{"bitand_vs_logical", "x & y && z", "((x & y) && z)"},        // & above &&
		{"bitxor_vs_logical", "x ^ y && z", "((x ^ y) && z)"},        // ^ above &&
		{"shift_vs_compare", "a << 2 < b", "((a << 2) < b)"},         // << above <
		{"compound_bitwise", "a & b ^ c & d", "((a & b) ^ (c & d))"}, // & tighter than ^
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertPrecedence(t, tt.input, tt.expected)
		})
	}
}

// TestUnaryPrecedence tests unary operator precedence
func TestUnaryPrecedence(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Unary operators bind tighter than binary
		{"negate_vs_add", "-x + y", "((-x) + y)"},
		{"not_vs_and", "!x && y", "((!x) && y)"},

		// Multiple unary operators
		// TODO: Double negation syntax not supported yet
		// {"double_negate", "--x", "(-(-x))"},
		{"negate_not", "-!x", "(-(!x))"},

		// Unary in complex expressions
		{"unary_in_arith", "1 + -2 * 3", "(1 + ((-2) * 3))"},
		{"not_in_logical", "!x || y && !z", "((!x) || (y && (!z)))"},

		// Bitwise NOT as prefix
		{"tilde_vs_add", "~x + 1", "((~x) + 1)"},
		{"tilde_vs_multiply", "~x * 2", "((~x) * 2)"},
		{"tilde_vs_bitand", "~x & y", "((~x) & y)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertPrecedence(t, tt.input, tt.expected)
		})
	}
}

// TestPrecedenceWithGrouping tests that parentheses override precedence
func TestPrecedenceWithGrouping(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"grouped_add_first", "(1 + 2) * 3", "((1 + 2) * 3)"},
		{"grouped_or_first", "(x || y) && z", "((x || y) && z)"},
		{"nested_grouping", "((1 + 2) * 3) + 4", "(((1 + 2) * 3) + 4)"},
		{"multiple_groups", "(a + b) * (c + d)", "((a + b) * (c + d))"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertPrecedence(t, tt.input, tt.expected)
		})
	}
}

// TestPrecedenceTable validates the complete precedence table
// Uses C-standard dedicated precedence bands (loosest to tightest):
//
//	||  &&  ^  &  ==/<  <</>>  ++  +/-  */%
func TestPrecedenceTable(t *testing.T) {
	// Generate all pairwise operator precedence tests
	operators := []struct {
		op         string
		precedence int // Lower number = lower precedence (looser binding)
	}{
		{"||", 1},
		{"&&", 2},
		{"^", 3}, // bitwise XOR
		{"&", 4}, // bitwise AND
		{"==", 5},
		{"!=", 5},
		{"<", 6},
		{"<=", 6},
		{">", 6},
		{">=", 6},
		{"<<", 7}, // shift
		{">>", 7},
		{"+", 8},
		{"-", 8},
		{"*", 9},
		{"/", 9},
		{"%", 9},
	}

	// Test that operators with different precedence parse correctly
	for i, op1 := range operators {
		for j, op2 := range operators {
			if i >= j {
				continue // Only test each pair once
			}

			if op1.precedence < op2.precedence {
				// op2 should bind tighter
				input := fmt.Sprintf("a %s b %s c", op1.op, op2.op)
				expected := fmt.Sprintf("(a %s (b %s c))", op1.op, op2.op)

				t.Run(fmt.Sprintf("%s_vs_%s", op1.op, op2.op), func(t *testing.T) {
					assertPrecedence(t, input, expected)
				})
			}
		}
	}
}

// TestAssociativity validates operator associativity
func TestAssociativity(t *testing.T) {
	tests := []struct {
		name      string
		op        string
		expected  string // Pattern for "a OP b OP c"
		leftAssoc bool   // true for left, false for right
	}{
		{"add_assoc", "+", "((a + b) + c)", true},
		{"subtract_assoc", "-", "((a - b) - c)", true},
		{"multiply_assoc", "*", "((a * b) * c)", true},
		{"divide_assoc", "/", "((a / b) / c)", true},
		{"modulo_assoc", "%", "((a % b) % c)", true},
		{"and_assoc", "&&", "((a && b) && c)", true},
		{"or_assoc", "||", "((a || b) || c)", true},
		{"equal_assoc", "==", "((a == b) == c)", true},
		{"less_assoc", "<", "((a < b) < c)", true},
		// Bitwise operators are left-associative
		{"bitand_assoc", "&", "((a & b) & c)", true},
		{"bitxor_assoc", "^", "((a ^ b) ^ c)", true},
		{"shl_assoc", "<<", "((a << b) << c)", true},
		{"shr_assoc", ">>", "((a >> b) >> c)", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := fmt.Sprintf("a %s b %s c", tt.op, tt.op)
			assertPrecedence(t, input, tt.expected)
		})
	}
}

// TestPrecedenceWithFunctionCalls tests precedence with function application
func TestPrecedenceWithFunctionCalls(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Function calls have highest precedence
		{"call_vs_add", "f(x) + 1", "(f(x) + 1)"},
		{"call_vs_multiply", "f(x) * 2", "(f(x) * 2)"},
		{"add_in_call", "f(x + 1)", "f((x + 1))"},
		{"multiply_in_call", "f(x * 2)", "f((x * 2))"},

		// Multiple calls
		{"chained_calls", "f(g(x))", "f(g(x))"},
		{"call_with_op", "f(x) + g(y)", "(f(x) + g(y))"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: These will need special handling in assertPrecedence
			// since function calls aren't simple binary operators
			t.Skip("Function call precedence tests need special parser handling")
		})
	}
}

// TestPrecedenceWithFieldAccess tests precedence with record field access
func TestPrecedenceWithFieldAccess(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Field access has highest precedence
		{"access_vs_add", "obj.field + 1", "(obj.field + 1)"},
		{"access_vs_multiply", "obj.field * 2", "(obj.field * 2)"},
		{"chained_access", "obj.a.b.c", "obj.a.b.c"},
		{"access_with_op", "obj1.x + obj2.y", "(obj1.x + obj2.y)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Skip("Field access precedence tests need special parser handling")
		})
	}
}

// TestPrecedenceEdgeCases tests edge cases and corner cases
func TestPrecedenceEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Long chains
		{"long_add_chain", "1 + 2 + 3 + 4 + 5", "((((1 + 2) + 3) + 4) + 5)"},
		{"long_multiply_chain", "1 * 2 * 3 * 4", "(((1 * 2) * 3) * 4)"},

		// Mixed long chains
		{"mixed_long_chain", "1 + 2 * 3 + 4 * 5 + 6", "(((1 + (2 * 3)) + (4 * 5)) + 6)"},

		// Deeply nested - && binds tighter than ||
		{"deep_nested", "a || b && c || d && e", "((a || (b && c)) || (d && e))"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertPrecedence(t, tt.input, tt.expected)
		})
	}
}

// TestInvalidPrecedence tests that parser correctly handles invalid expressions
func TestInvalidPrecedence(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"two_operators", "1 + * 2"},
		{"trailing_operator", "1 + 2 +"},
		{"leading_operator", "+ 1 + 2"},
		// TODO: () is valid unit literal, not an error
		// {"empty_parens_op", "1 + () + 2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := mustParseError(t, tt.input)
			if len(errs) == 0 {
				t.Errorf("Expected parse error for %q, but got none", tt.input)
			}
		})
	}
}
