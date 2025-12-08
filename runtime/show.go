// Package runtime provides helper functions for AILANG Go codegen.
// This is the canonical runtime for the Go backend - semantics defined here
// are the source of truth for compiled AILANG programs.
package runtime

import "fmt"

// Show converts any AILANG value to its string representation.
// Handles the core AILANG types: int64, float64, string, bool.
// For other types, falls back to fmt.Sprintf("%v", v).
func Show(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case int64:
		return fmt.Sprintf("%d", x)
	case int:
		return fmt.Sprintf("%d", x)
	case float64:
		return fmt.Sprintf("%g", x)
	case bool:
		if x {
			return "true"
		}
		return "false"
	case nil:
		return "()"
	default:
		return fmt.Sprintf("%v", x)
	}
}
