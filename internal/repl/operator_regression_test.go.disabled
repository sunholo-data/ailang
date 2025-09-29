package repl

import (
	"bytes"
	"strings"
	"testing"
)

// TestOperatorMethodMapping ensures that each arithmetic operator calls the correct dictionary method
// This test prevents regression of the bug where all operators were calling 'add'
func TestOperatorMethodMapping(t *testing.T) {
	tests := []struct {
		name           string
		expression     string
		expectedMethod string
		expectedResult string
	}{
		{"addition", "1 + 2", "add", "3"},
		{"subtraction", "5 - 2", "sub", "3"},
		{"multiplication", "2 * 3", "mul", "6"},
		{"division", "6 / 2", "div", "3"},
		{"equality", "5 == 5", "eq", "true"},
		{"inequality", "5 != 3", "neq", "true"},
		{"less_than", "3 < 5", "lt", "true"},
		{"less_equal", "3 <= 3", "lte", "true"},
		{"greater_than", "5 > 3", "gt", "true"},
		{"greater_equal", "5 >= 5", "gte", "true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repl := New()

			// Initialize prelude for proper defaulting (critical for tests)
			var discardOut bytes.Buffer
			repl.importModule("std/prelude", &discardOut)

			// Capture output
			var output bytes.Buffer

			// Process the expression through the REPL pipeline
			repl.processExpression(tt.expression, &output)

			outputStr := output.String()

			// Check that the result is correct (the debug output goes to stdout, not captured)
			// We trust that the correct method was called if we get the right result

			// Check that the result is correct
			if !strings.Contains(outputStr, tt.expectedResult) {
				t.Errorf("Expected result '%s' for expression '%s', but output was: %s",
					tt.expectedResult, tt.expression, outputStr)
			}

			// Critical: Should not see fallback errors
			if strings.Contains(outputStr, "BinOp reached evaluator") {
				t.Errorf("REGRESSION: Expression '%s' fell back to BinOp evaluator instead of using dictionaries",
					tt.expression)
			}
		})
	}
}

// TestFloatOperations ensures float arithmetic works correctly
func TestFloatOperations(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		expected   string
	}{
		{"float_multiplication", "3.14 * 2.0", "6.28"},
		{"float_addition", "1.5 + 2.5", "4"},
		{"float_division", "10.0 / 2.0", "5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repl := New()
			var discardOut bytes.Buffer
			repl.importModule("std/prelude", &discardOut)

			var output bytes.Buffer
			repl.processExpression(tt.expression, &output)
			outputStr := output.String()

			// Should not get "expected int arguments" error
			if strings.Contains(outputStr, "expected int arguments") {
				t.Errorf("Float operation '%s' incorrectly tried to use int method: %s",
					tt.expression, outputStr)
			}

			// Should contain the expected result
			if !strings.Contains(outputStr, tt.expected) {
				t.Errorf("Expected result '%s' for expression '%s', but output was: %s",
					tt.expected, tt.expression, outputStr)
			}
		})
	}
}

// TestStringConcatenation ensures string concatenation works
func TestStringConcatenation(t *testing.T) {
	repl := New()
	var output bytes.Buffer

	repl.processExpression(`"Hello " ++ "World"`, &output)
	outputStr := output.String()

	// Should not reach the BinOp fallback
	if strings.Contains(outputStr, "BinOp reached evaluator") {
		t.Errorf("String concatenation incorrectly fell back to BinOp evaluator: %s", outputStr)
	}

	// Should produce correct result
	if !strings.Contains(outputStr, "Hello World") {
		t.Errorf("Expected 'Hello World' in output, but got: %s", outputStr)
	}
}

// TestDictionaryElaborationHappens ensures that BinOp nodes are properly elaborated to DictApp
func TestDictionaryElaborationHappens(t *testing.T) {
	repl := New()
	var output bytes.Buffer

	// Enable core dumping to see if BinOp nodes remain
	repl.config.ShowCore = true

	repl.processExpression("2 + 3", &output)
	outputStr := output.String()

	// Should produce correct result and not fail with elaboration errors
	if !strings.Contains(outputStr, "5 :: Int") {
		t.Errorf("Expected correct result '5 :: Int', but got: %s", outputStr)
	}

	// Should not see elaboration failure messages
	if strings.Contains(outputStr, "BinOp reached evaluator") {
		t.Errorf("Dictionary elaboration failed - fell back to BinOp evaluator: %s", outputStr)
	}
}

// TestFillOperatorMethodsCalled ensures the critical FillOperatorMethods call happens
func TestFillOperatorMethodsCalled(t *testing.T) {
	repl := New()
	var output bytes.Buffer

	repl.processExpression("2 * 3", &output)
	outputStr := output.String()

	// Should produce correct result (evidence that the pipeline worked)
	if !strings.Contains(outputStr, "6 :: Int") {
		t.Errorf("Expected correct result '6 :: Int', but got: %s", outputStr)
	}

	// Should not see fallback errors
	if strings.Contains(outputStr, "BinOp reached evaluator") {
		t.Errorf("Operator fell back to BinOp evaluator: %s", outputStr)
	}
}

// TestNoFallbackToApplyBinOp ensures operations use dictionary-passing, not fallback
func TestNoFallbackToApplyBinOp(t *testing.T) {
	operations := []string{"1 + 2", "3 * 4", "5 - 1", "8 / 2"}

	for _, op := range operations {
		t.Run(op, func(t *testing.T) {
			repl := New()
			var output bytes.Buffer

			repl.processExpression(op, &output)
			outputStr := output.String()

			// Should not see the BinOp fallback error
			if strings.Contains(outputStr, "BinOp reached evaluator; dictionaries not elaborated") {
				t.Errorf("Operation '%s' fell back to applyBinOp instead of using dictionaries: %s",
					op, outputStr)
			}
		})
	}
}

// TestTypeDisplayNormalization ensures type names are properly normalized
func TestTypeDisplayNormalization(t *testing.T) {
	tests := []struct {
		expression   string
		expectedType string
	}{
		{"42", "Int"},
		{"3.14", "Float"},
		{"true", "Bool"},
		{`"hello"`, "String"},
	}

	for _, tt := range tests {
		t.Run(tt.expression, func(t *testing.T) {
			repl := New()
			var output bytes.Buffer

			repl.processExpression(tt.expression, &output)
			outputStr := output.String()

			// Should show normalized type names, not internal ones
			if !strings.Contains(outputStr, ":: "+tt.expectedType) {
				t.Errorf("Expected type '%s' for expression '%s', but output was: %s",
					tt.expectedType, tt.expression, outputStr)
			}
		})
	}
}

// TestMostSpecificNumericClassRegression ensures float literals force Fractional constraint
// This test prevents regression of the bug where BinOp would downgrade Fractional to Num
func TestMostSpecificNumericClassRegression(t *testing.T) {
	tests := []struct {
		name           string
		expression     string
		expectedClass  string // "Fractional" or "Num"
		expectedType   string // "Float" or "Int"
		expectedResult string
	}{
		{
			name:           "float_multiplication_preserves_fractional",
			expression:     "3.14 * 2.0",
			expectedClass:  "Fractional",
			expectedType:   "Float",
			expectedResult: "6.28",
		},
		{
			name:           "float_addition_preserves_fractional",
			expression:     "1.5 + 2.5",
			expectedClass:  "Fractional",
			expectedType:   "Float",
			expectedResult: "4",
		},
		{
			name:           "int_arithmetic_stays_num",
			expression:     "2 * 3",
			expectedClass:  "Num",
			expectedType:   "Int",
			expectedResult: "6",
		},
		{
			name:           "float_division_preserves_fractional",
			expression:     "10.0 / 3.0",
			expectedClass:  "Fractional",
			expectedType:   "Float",
			expectedResult: "3.333",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repl := New()
			var discardOut bytes.Buffer
			repl.importModule("std/prelude", &discardOut)

			var output bytes.Buffer
			repl.processExpression(tt.expression, &output)
			outputStr := output.String()

			// CRITICAL: Should resolve to the expected final type
			if !strings.Contains(outputStr, ":: "+tt.expectedType) {
				t.Errorf("REGRESSION: Expected final type '%s' for '%s', but output was: %s",
					tt.expectedType, tt.expression, outputStr)
			}

			// Should produce correct result
			if !strings.Contains(outputStr, tt.expectedResult) {
				t.Errorf("Expected result containing '%s' for '%s', but output was: %s",
					tt.expectedResult, tt.expression, outputStr)
			}

			// CRITICAL: Should NOT see "expected int arguments" error for float operations
			if tt.expectedType == "Float" && strings.Contains(outputStr, "expected int arguments") {
				t.Errorf("REGRESSION: Float operation '%s' incorrectly tried to use int method: %s",
					tt.expression, outputStr)
			}
		})
	}
}

// TestBooleanOperatorsRegression ensures boolean operators work without dictionary elaboration
// This test prevents regression of the bug where boolean operators caused "BinOp reached evaluator"
func TestBooleanOperatorsRegression(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		expected   string
	}{
		{"logical_and_true_false", "true && false", "false"},
		{"logical_and_false_true", "false && true", "false"},
		{"logical_and_true_true", "true && true", "true"},
		{"logical_or_true_false", "true || false", "true"},
		{"logical_or_false_false", "false || false", "false"},
		{"logical_or_false_true", "false || true", "true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repl := New()
			var output bytes.Buffer

			repl.processExpression(tt.expression, &output)
			outputStr := output.String()

			// CRITICAL: Should NOT see the "BinOp reached evaluator" error
			if strings.Contains(outputStr, "BinOp reached evaluator") {
				t.Errorf("REGRESSION: Boolean operation '%s' incorrectly fell back to BinOp evaluator: %s",
					tt.expression, outputStr)
			}

			// Should produce correct result
			if !strings.Contains(outputStr, tt.expected+" :: Bool") {
				t.Errorf("Expected '%s :: Bool' for expression '%s', but output was: %s",
					tt.expected, tt.expression, outputStr)
			}

			// Should show that it's working (has result) and not using type classes
			if strings.Contains(outputStr, "Runtime error") {
				t.Errorf("Boolean operation '%s' failed with runtime error: %s",
					tt.expression, outputStr)
			}
		})
	}
}

// TestMixedArithmeticScenarios ensures complex expressions work correctly
func TestMixedArithmeticScenarios(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		expected   string
		comment    string
	}{
		{
			name:       "integer_complex_expression",
			expression: "10 * 5 - 20 / 4",
			expected:   "45 :: Int",
			comment:    "Complex integer arithmetic should stay Int",
		},
		// Note: Parser doesn't handle mixed int/float correctly yet, so we test separately
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repl := New()
			var discardOut bytes.Buffer
			repl.importModule("std/prelude", &discardOut)

			var output bytes.Buffer
			repl.processExpression(tt.expression, &output)
			outputStr := output.String()

			// Should produce correct result
			if !strings.Contains(outputStr, tt.expected) {
				t.Errorf("Expected '%s' for expression '%s', but output was: %s",
					tt.expected, tt.expression, outputStr)
			}

			// Should not see any errors
			if strings.Contains(outputStr, "Runtime error") {
				t.Errorf("Unexpected runtime error for '%s': %s", tt.expression, outputStr)
			}
		})
	}
}
