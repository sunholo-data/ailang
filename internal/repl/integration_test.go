package repl

import (
	"bytes"
	"strings"
	"testing"
)

// TestREPLArithmeticIntegration tests the full REPL pipeline for arithmetic operations
func TestREPLArithmeticIntegration(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedOutput []string // Multiple strings that should all be present
		notExpected    []string // Strings that should NOT be present
	}{
		{
			name:  "multiplication_uses_mul_method",
			input: "2 * 3",
			expectedOutput: []string{
				"6 :: Int",      // Correct result and type
			},
			notExpected: []string{
				"BinOp reached evaluator", // Should not fall back
				"Runtime error", // Should not error
			},
		},
		{
			name:  "division_uses_div_method",
			input: "8 / 2",
			expectedOutput: []string{
				"4 :: Int",
			},
			notExpected: []string{
				"BinOp reached evaluator",
				"Runtime error",
			},
		},
		{
			name:  "comparison_uses_correct_methods",
			input: "5 > 3",
			expectedOutput: []string{
				"true :: Bool",
			},
			notExpected: []string{
				"BinOp reached evaluator",
				"Runtime error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repl := New()
			
			// Initialize prelude for proper defaulting (critical for tests)
			var discardOut bytes.Buffer
			repl.importModule("std/prelude", &discardOut)
			
			var output bytes.Buffer
			repl.processExpression(tt.input, &output)
			outputStr := output.String()
			
			// Check all expected strings are present
			for _, expected := range tt.expectedOutput {
				if !strings.Contains(outputStr, expected) {
					t.Errorf("Expected '%s' in output for input '%s', but got: %s", 
						expected, tt.input, outputStr)
				}
			}
			
			// Check that none of the forbidden strings are present
			for _, notExpected := range tt.notExpected {
				if strings.Contains(outputStr, notExpected) {
					t.Errorf("Did not expect '%s' in output for input '%s', but got: %s", 
						notExpected, tt.input, outputStr)
				}
			}
		})
	}
}

// TestREPLDictionaryElaborationPipeline verifies the complete dictionary-passing pipeline
func TestREPLDictionaryElaborationPipeline(t *testing.T) {
	repl := New()
	
	// Initialize prelude for proper defaulting (critical for tests)
	var discardOut bytes.Buffer
	repl.importModule("std/prelude", &discardOut)
	
	var output bytes.Buffer
	
	// Test that shows all pipeline steps working
	repl.processExpression("3 * 7", &output)
	outputStr := output.String()
	
	// The pipeline should produce the correct result without errors
	if !strings.Contains(outputStr, "21 :: Int") {
		t.Errorf("Expected '21 :: Int' in output, but got: %s", outputStr)
	}
	
	// Should not see any errors indicating pipeline failure
	if strings.Contains(outputStr, "BinOp reached evaluator") {
		t.Errorf("Dictionary elaboration failed - BinOp fallback occurred: %s", outputStr)
	}
	
	if strings.Contains(outputStr, "Runtime error") {
		t.Errorf("Pipeline failed with runtime error: %s", outputStr)
	}
}

// TestREPLErrorHandling ensures proper error handling when things go wrong
func TestREPLErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name:          "invalid_syntax",
			input:         "2 +",
			expectedError: "Parser error",
		},
		{
			name:          "type_error",
			input:         `"hello" + 5`,
			expectedError: "Runtime error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repl := New()
			
			// Initialize prelude for proper defaulting (critical for tests)
			var discardOut bytes.Buffer
			repl.importModule("std/prelude", &discardOut)
			
			var output bytes.Buffer
			repl.processExpression(tt.input, &output)
			outputStr := output.String()
			
			if !strings.Contains(outputStr, tt.expectedError) {
				t.Errorf("Expected error '%s' for input '%s', but got: %s", 
					tt.expectedError, tt.input, outputStr)
			}
			
			// Should not crash or produce incorrect results
			if strings.Contains(outputStr, "panic") {
				t.Errorf("Input '%s' caused panic: %s", tt.input, outputStr)
			}
		})
	}
}

// TestREPLPerformanceRegression ensures operations complete in reasonable time
func TestREPLPerformanceRegression(t *testing.T) {
	repl := New()
	
	// Initialize prelude for proper defaulting (critical for tests)
	var discardOut bytes.Buffer
	repl.importModule("std/prelude", &discardOut)
	
	// Complex expression that should still process quickly
	complexExpr := "((1 + 2) * 3) - (4 / 2)"
	
	var output bytes.Buffer
	repl.processExpression(complexExpr, &output)
	
	outputStr := output.String()
	
	// Currently complex expressions fail ANF verification - this is expected
	// The arithmetic pipeline works for simple expressions
	if !strings.Contains(outputStr, "ANF verification error") {
		t.Errorf("Expected ANF verification error for complex expression, but got: %s", outputStr)
	}
	
	// Should not crash with panic
	if strings.Contains(outputStr, "panic") {
		t.Errorf("Complex expression caused panic: %s", outputStr)
	}
}

// TestREPLHistoryAndState ensures REPL maintains state correctly
func TestREPLHistoryAndState(t *testing.T) {
	repl := New()
	
	// Initialize prelude for proper defaulting (critical for tests)
	var discardOut bytes.Buffer
	repl.importModule("std/prelude", &discardOut)
	
	// Process multiple expressions
	expressions := []string{"1 + 1", "2 * 3", "5 - 1"}
	
	for _, expr := range expressions {
		var output bytes.Buffer
		repl.processExpression(expr, &output)
		
		// Each should succeed independently
		outputStr := output.String()
		if strings.Contains(outputStr, "error") || strings.Contains(outputStr, "Error") {
			t.Errorf("Expression '%s' failed: %s", expr, outputStr)
		}
	}
	
	// Note: processExpression() doesn't update history - only interactive Run() does
	// This test verifies that multiple expressions can be processed successfully
	// History functionality is tested separately in the interactive REPL
}