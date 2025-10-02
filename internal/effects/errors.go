package effects

import "fmt"

// CapabilityError represents a missing capability error
//
// This error is returned when an effect operation requires a capability
// that has not been granted. The error message includes the effect name
// and helpful hints for the user.
//
// Example error output:
//
//	effect 'IO' requires capability, but none provided
//	Hint: Run with --caps IO
type CapabilityError struct {
	Effect string // The required effect name (e.g., "IO", "FS")
}

// Error implements the error interface
//
// Returns a formatted error message with the missing capability name
func (e *CapabilityError) Error() string {
	return fmt.Sprintf("effect '%s' requires capability, but none provided\nHint: Run with --caps %s",
		e.Effect, e.Effect)
}

// NewCapabilityError creates a new capability error
//
// Parameters:
//   - effect: The name of the required effect
//
// Returns:
//   - A new CapabilityError
//
// Example:
//
//	if !ctx.HasCap("FS") {
//	    return NewCapabilityError("FS")
//	}
func NewCapabilityError(effect string) *CapabilityError {
	return &CapabilityError{Effect: effect}
}
