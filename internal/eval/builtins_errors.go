package eval

import "fmt"

// NewRuntimeError creates a runtime error with structured information
func NewRuntimeError(code, message string, pos interface{}) error {
	// TODO: integrate with error encoder
	return fmt.Errorf("[%s] %s", code, message)
}

// buildTypeMismatchError creates a detailed type mismatch error for builtin functions
// This provides helpful hints when the wrong type is passed to a builtin, especially
// for common issues like using eq_Int with Float values.
func buildTypeMismatchError(builtinName, expectedType string, actualValue Value) error {
	actualType := actualValue.Type()

	// Base error message with actual type information
	baseMsg := fmt.Sprintf("builtin %s expects %s arguments, but received %s",
		builtinName, expectedType, actualType)

	// Add helpful hint for Float equality issue (common AI generation mistake)
	if builtinName == "eq_Int" && actualType == "float" {
		return fmt.Errorf("%s\nHint: Use Float literals and eq_Float for floating-point comparisons. The type system may have incorrectly selected eq_Int for a Float comparison.", baseMsg)
	}

	// Add hint for other numeric type mismatches
	if (expectedType == "Int" && actualType == "float") ||
		(expectedType == "Float" && actualType == "int") {
		return fmt.Errorf("%s\nHint: Check that numeric literals have the correct type (use .0 for Floats)", baseMsg)
	}

	return fmt.Errorf("%s", baseMsg)
}
