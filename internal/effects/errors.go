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
//	effect 'IO' budget exhausted: semantic limit=5, used=5 (physical: 19)
//	Hint: Increase the budget with @limit=N or refactor to use fewer IO operations
type BudgetExhaustedError struct {
	Effect   string // The effect name (e.g., "IO", "FS")
	Limit    int    // The budget limit (semantic)
	Used     int    // How many times the effect was used (semantic)
	Physical int    // Physical usage count (actual builtin calls)
	Position string // Source position where budget was exhausted (optional)
}

// Error implements the error interface
//
// Returns a formatted error message with budget details including physical count
func (e *BudgetExhaustedError) Error() string {
	msg := fmt.Sprintf("effect '%s' budget exhausted: semantic limit=%d, used=%d (physical: %d)",
		e.Effect, e.Limit, e.Used, e.Physical)
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
//   - limit: The budget limit (semantic)
//   - used: How many times the effect was used (semantic)
//   - position: Source position (optional, can be empty)
//   - physical: Physical usage count (actual builtin calls)
//
// Returns:
//   - A new BudgetExhaustedError
//
// Example:
//
//	if budget.Remaining("IO") <= 0 {
//	    return NewBudgetExhaustedError("IO", 5, 5, "file.ail:10:5", 19)
//	}
func NewBudgetExhaustedError(effect string, limit, used int, position string, physical int) *BudgetExhaustedError {
	return &BudgetExhaustedError{
		Effect:   effect,
		Limit:    limit,
		Used:     used,
		Physical: physical,
		Position: position,
	}
}

// BudgetUnderrunError represents a minimum usage requirement not met
//
// M-DX25 M4: This error is returned when a function with @min=N budget constraint
// returns without having exercised the effect at least N times.
//
// Example error output:
//
//	effect 'Net' budget underrun: min=1, actual=0
//	Hint: Ensure the function actually performs at least 1 Net operation
type BudgetUnderrunError struct {
	Effect   string // The effect name (e.g., "Net", "IO")
	Min      int    // The minimum required usage
	Actual   int    // The actual physical usage count
	Position string // Source position where check occurred (optional)
}

// Error implements the error interface
//
// Returns a formatted error message with underrun details
func (e *BudgetUnderrunError) Error() string {
	msg := fmt.Sprintf("effect '%s' budget underrun: min=%d, actual=%d",
		e.Effect, e.Min, e.Actual)
	if e.Position != "" {
		msg += fmt.Sprintf(" at %s", e.Position)
	}
	msg += fmt.Sprintf("\nHint: Ensure the function actually performs at least %d %s operation(s)", e.Min, e.Effect)
	return msg
}

// NewBudgetUnderrunError creates a new budget underrun error
//
// Parameters:
//   - effect: The name of the effect
//   - min: The minimum required usage
//   - actual: The actual physical usage count
//   - position: Source position (optional, can be empty)
//
// Returns:
//   - A new BudgetUnderrunError
func NewBudgetUnderrunError(effect string, min, actual int, position string) *BudgetUnderrunError {
	return &BudgetUnderrunError{
		Effect:   effect,
		Min:      min,
		Actual:   actual,
		Position: position,
	}
}
