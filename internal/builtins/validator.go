package builtins

import (
	"fmt"
	"sort"
)

// ValidationError represents a builtin registration issue
type ValidationError struct {
	Builtin  string // Builtin name
	Message  string // Error description
	Fix      string // Suggested fix with example code
	Location string // File where fix should be applied
	Severity string // "error" or "warning"
}

// RegistryStats holds statistics about registered builtins
type RegistryStats struct {
	Total       int            // Total builtins registered
	Pure        int            // Pure functions (no side effects)
	Effect      int            // Functions with effects
	ByModule    map[string]int // Count by module
	ByEffect    map[string]int // Count by effect type
}

// GetRegistryStats returns statistics about registered builtins
func GetRegistryStats() RegistryStats {
	stats := RegistryStats{
		ByModule: make(map[string]int),
		ByEffect: make(map[string]int),
	}

	specs := AllSpecs() // Use AllSpecs() to access frozen registry
	for _, spec := range specs {
		stats.Total++

		if spec.IsPure {
			stats.Pure++
		} else {
			stats.Effect++
		}

		// Count by module
		if spec.Module != "" {
			stats.ByModule[spec.Module]++
		}

		// Count by effect
		if spec.Effect != "" {
			stats.ByEffect[spec.Effect]++
		} else {
			stats.ByEffect["Pure"]++
		}
	}

	return stats
}

// ValidateBuiltins performs comprehensive validation on all registered builtins
// Returns a list of validation errors (empty if all valid)
func ValidateBuiltins() []ValidationError {
	var errors []ValidationError

	// Get all registered specs
	specs := AllSpecs()

	// Validation Rule 1: All specs must have non-nil Type function
	for name, spec := range specs {
		if spec.Type == nil {
			errors = append(errors, ValidationError{
				Builtin:  name,
				Message:  "Missing type signature function",
				Fix:      fmt.Sprintf("Add 'Type: make%sType' to BuiltinSpec in register.go", toCamelCase(name)),
				Location: "internal/builtins/register.go",
				Severity: "error",
			})
			continue // Skip further checks if Type is nil
		}

		// Check that Type() returns non-nil
		typ := spec.Type()
		if typ == nil {
			errors = append(errors, ValidationError{
				Builtin:  name,
				Message:  "Type function returns nil",
				Fix:      fmt.Sprintf("Fix make%sType() to return valid type", toCamelCase(name)),
				Location: "internal/builtins/register.go",
				Severity: "error",
			})
		}
	}

	// Validation Rule 2: All specs must have non-nil Impl function
	for name, spec := range specs {
		if spec.Impl == nil {
			errors = append(errors, ValidationError{
				Builtin:  name,
				Message:  "Missing implementation function",
				Fix:      fmt.Sprintf("Add 'Impl: effects.%sImpl' to BuiltinSpec", toCamelCase(name)),
				Location: "internal/builtins/register.go",
				Severity: "error",
			})
		}
	}

	// Validation Rule 3: Effect functions must have Effect field set
	for name, spec := range specs {
		if !spec.IsPure && spec.Effect == "" {
			errors = append(errors, ValidationError{
				Builtin:  name,
				Message:  "Effect function missing Effect field",
				Fix:      fmt.Sprintf("Set 'Effect: \"IO\"' (or Net/FS) in BuiltinSpec for %s", name),
				Location: "internal/builtins/register.go",
				Severity: "warning",
			})
		}
	}

	// Validation Rule 4: Pure functions shouldn't have Effect field set
	for name, spec := range specs {
		if spec.IsPure && spec.Effect != "" {
			errors = append(errors, ValidationError{
				Builtin:  name,
				Message:  "Pure function has Effect field set",
				Fix:      fmt.Sprintf("Set 'Effect: \"\"' for pure function %s", name),
				Location: "internal/builtins/register.go",
				Severity: "warning",
			})
		}
	}

	// Validation Rule 5: NumArgs must be > 0
	for name, spec := range specs {
		if spec.NumArgs < 0 {
			errors = append(errors, ValidationError{
				Builtin:  name,
				Message:  fmt.Sprintf("Invalid NumArgs: %d (must be >= 0)", spec.NumArgs),
				Fix:      fmt.Sprintf("Set correct 'NumArgs' in BuiltinSpec for %s", name),
				Location: "internal/builtins/register.go",
				Severity: "error",
			})
		}
	}

	// Validation Rule 6: Module should be specified
	for name, spec := range specs {
		if spec.Module == "" {
			errors = append(errors, ValidationError{
				Builtin:  name,
				Message:  "Missing Module field",
				Fix:      fmt.Sprintf("Set 'Module: \"std/...\"' in BuiltinSpec for %s", name),
				Location: "internal/builtins/register.go",
				Severity: "warning",
			})
		}
	}

	return errors
}

// GroupByEffect groups builtin names by their effect type
func GroupByEffect() map[string][]string {
	grouped := make(map[string][]string)

	specs := AllSpecs() // Use AllSpecs() to access frozen registry
	for name, spec := range specs {
		effect := spec.Effect
		if effect == "" {
			effect = "Pure"
		}
		grouped[effect] = append(grouped[effect], name)
	}

	// Sort names within each group for deterministic output
	for effect := range grouped {
		sort.Strings(grouped[effect])
	}

	return grouped
}

// GroupByModule groups builtin names by their module
func GroupByModule() map[string][]string {
	grouped := make(map[string][]string)

	specs := AllSpecs() // Use AllSpecs() to access frozen registry
	for name, spec := range specs {
		module := spec.Module
		if module == "" {
			module = "unknown"
		}
		grouped[module] = append(grouped[module], name)
	}

	// Sort names within each group
	for module := range grouped {
		sort.Strings(grouped[module])
	}

	return grouped
}

// toCamelCase converts snake_case to CamelCase
// Example: _str_len -> StrLen, _net_httpRequest -> NetHttpRequest
func toCamelCase(s string) string {
	if s == "" {
		return ""
	}

	// Remove leading underscore
	if s[0] == '_' {
		s = s[1:]
	}

	// Simple conversion: just capitalize first letter
	// For more complex cases, implement proper snake_case -> CamelCase
	if len(s) > 0 {
		return string(s[0]-32) + s[1:] // ASCII uppercase trick
	}
	return s
}
