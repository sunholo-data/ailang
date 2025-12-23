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

// BudgetExhaustedError represents a budget exhaustion error
//
// This error is returned when an effect operation exceeds its allocated budget.
// Budgets limit the number of times an effect can be invoked within a scope.
//
// Example error output:
//
//	effect 'IO' budget exhausted: limit=5, used=5
//	Hint: Increase the budget with @limit=N or refactor to use fewer IO operations
type BudgetExhaustedError struct {
	Effect   string // The effect name (e.g., "IO", "FS")
	Limit    int    // The budget limit
	Used     int    // How many times the effect was used
	Position string // Source position where budget was exhausted (optional)
}

// Error implements the error interface
//
// Returns a formatted error message with budget details
func (e *BudgetExhaustedError) Error() string {
	msg := fmt.Sprintf("effect '%s' budget exhausted: limit=%d, used=%d",
		e.Effect, e.Limit, e.Used)
	if e.Position != "" {
		msg += fmt.Sprintf(" at %s", e.Position)
	}
	msg += fmt.Sprintf("\nHint: Increase the budget with @limit=N or refactor to use fewer %s operations", e.Effect)
	return msg
}

// NewBudgetExhaustedError creates a new budget exhausted error
//
// Parameters:
//   - effect: The name of the effect
//   - limit: The budget limit
//   - used: How many times the effect was used
//   - position: Source position (optional, can be empty)
//
// Returns:
//   - A new BudgetExhaustedError
//
// Example:
//
//	if budget.Remaining("IO") <= 0 {
//	    return NewBudgetExhaustedError("IO", 5, 5, "file.ail:10:5")
//	}
func NewBudgetExhaustedError(effect string, limit, used int, position string) *BudgetExhaustedError {
	return &BudgetExhaustedError{
		Effect:   effect,
		Limit:    limit,
		Used:     used,
		Position: position,
	}
}
