package eval

import (
	"fmt"
)

// BuiltinError represents a runtime error from a builtin function
type BuiltinError struct {
	Builtin  string // Name of the builtin that failed
	Expected string // Expected type(s)
	Got      string // Actual type received
	Hint     string // Helpful suggestion for fixing
}

func (e *BuiltinError) Error() string {
	msg := fmt.Sprintf("Runtime error in %s: expected %s, got %s", e.Builtin, e.Expected, e.Got)
	if e.Hint != "" {
		msg += fmt.Sprintf("\nHint: %s", e.Hint)
	}
	return msg
}

// ArgTypeMismatch creates a helpful error when an argument has the wrong type
//
// Examples:
//   ArgTypeMismatch("_str_len", "string", "int")
//   → "Runtime error in _str_len: expected string, got int"
//
//   ArgTypeMismatch("_list_concat", "list", "string")
//   → "Runtime error in _list_concat: expected list, got string
//      Hint: Use ++ for list concatenation, not string concatenation"
func ArgTypeMismatch(builtin string, expected string, got string) error {
	hint := getSmartHint(builtin, expected, got)
	return &BuiltinError{
		Builtin:  builtin,
		Expected: expected,
		Got:      got,
		Hint:     hint,
	}
}

// getSmartHint provides context-aware hints based on the type mismatch
func getSmartHint(builtin string, expected string, got string) string {
	// Concatenation operator confusion
	if builtin == "_list_concat" && got == "string" {
		return "Use ++ for list concatenation. For strings, ensure both operands are lists."
	}
	if builtin == "_str_concat" && got == "list" {
		return "Use ++ for string concatenation. For lists, ensure both operands are strings."
	}

	// String operations on non-strings
	if builtin == "_str_len" && got != "string" {
		return fmt.Sprintf("Cannot get length of %s. Use _str_len only on strings.", got)
	}
	if builtin == "_str_slice" && got != "string" {
		return fmt.Sprintf("Cannot slice %s. Use _str_slice only on strings.", got)
	}

	// List operations on non-lists
	if builtin == "_list_length" && got != "list" {
		return fmt.Sprintf("Cannot get length of %s. Use _list_length only on lists.", got)
	}
	if builtin == "_list_head" && got != "list" {
		return fmt.Sprintf("Cannot get head of %s. Use _list_head only on lists.", got)
	}
	if builtin == "_list_tail" && got != "list" {
		return fmt.Sprintf("Cannot get tail of %s. Use _list_tail only on lists.", got)
	}

	// Math operations on non-numbers
	if (builtin == "_add" || builtin == "_sub" || builtin == "_mul" || builtin == "_div") && got == "string" {
		return "Cannot perform arithmetic on strings. Use ++ for string concatenation."
	}
	if (builtin == "_add" || builtin == "_sub" || builtin == "_mul" || builtin == "_div") && got == "list" {
		return "Cannot perform arithmetic on lists."
	}

	// Comparison operations
	if (builtin == "_lt" || builtin == "_le" || builtin == "_gt" || builtin == "_ge") && got == "string" {
		return "Use _str_compare for string comparison, not numeric operators."
	}

	// JSON decoding
	if builtin == "std/json.decode" && got != "string" {
		return "JSON decoding requires a string input. Ensure the input is valid JSON text."
	}

	// IO operations
	if builtin == "_io_readFile" && got != "string" {
		return "File path must be a string."
	}
	if builtin == "_io_writeFile" && expected == "string" && got != "string" {
		return "File path and contents must be strings."
	}

	// Net operations
	if builtin == "_net_httpRequest" && expected == "string" && got != "string" {
		return "HTTP request URL must be a string."
	}

	// Generic fallback
	if expected == "string" && got == "int" {
		return "Did you forget to convert the integer to a string?"
	}
	if expected == "int" && got == "string" {
		return "Did you forget to parse the string as an integer?"
	}
	if expected == "list" && got == "string" {
		return "Did you mean to split the string into a list?"
	}
	if expected == "string" && got == "list" {
		return "Did you mean to join the list into a string?"
	}

	// No specific hint
	return ""
}

// IndexOutOfBounds creates an error for invalid list/string indexing
func IndexOutOfBounds(builtin string, index int, length int) error {
	hint := ""
	if index < 0 {
		hint = "Negative indices are not supported. Use positive indices starting from 0."
	} else if index >= length {
		hint = fmt.Sprintf("Valid indices are 0 to %d.", length-1)
	}

	return &BuiltinError{
		Builtin:  builtin,
		Expected: fmt.Sprintf("index in range 0..%d", length-1),
		Got:      fmt.Sprintf("index %d", index),
		Hint:     hint,
	}
}

// InvalidOperation creates an error for operations that can't be performed
func InvalidOperation(builtin string, reason string) error {
	return &BuiltinError{
		Builtin:  builtin,
		Expected: "valid operation",
		Got:      "invalid operation",
		Hint:     reason,
	}
}

// EmptyListError creates an error for operations on empty lists
func EmptyListError(builtin string) error {
	hint := ""
	if builtin == "_list_head" {
		hint = "Cannot get the head of an empty list. Check if the list is non-empty first."
	} else if builtin == "_list_tail" {
		hint = "Cannot get the tail of an empty list. Check if the list is non-empty first."
	} else {
		hint = "This operation requires a non-empty list."
	}

	return &BuiltinError{
		Builtin:  builtin,
		Expected: "non-empty list",
		Got:      "empty list",
		Hint:     hint,
	}
}
