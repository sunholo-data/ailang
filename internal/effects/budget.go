package effects

// BudgetContext tracks per-effect usage counts against configured limits
//
// Budgets are type-level constraints that limit how many times an effect
// can be invoked within a function scope. Each function invocation gets
// a fresh budget (per-invocation semantics).
//
// Example:
//
//	budget := NewBudgetContext(map[string]*int{"IO": intPtr(5)})
//	for i := 0; i < 6; i++ {
//	    if err := budget.CheckAndConsume("IO", "file.ail:10:5"); err != nil {
//	        // Returns BudgetExhaustedError on the 6th call
//	        return err
//	    }
//	}
type BudgetContext struct {
	limits map[string]int // Effect name → limit (absent = unlimited)
	used   map[string]int // Effect name → usage count
}

// NewBudgetContext creates a new budget context with the specified limits
//
// Parameters:
//   - limits: Map of effect name to budget limit. nil values mean unlimited.
//     A nil map means no budgets (all effects unlimited).
//
// Returns:
//   - A new BudgetContext ready to track usage
//
// Example:
//
//	// IO limited to 5 calls, FS unlimited
//	limits := map[string]*int{"IO": intPtr(5)}
//	budget := NewBudgetContext(limits)
func NewBudgetContext(limits map[string]*int) *BudgetContext {
	bc := &BudgetContext{
		limits: make(map[string]int),
		used:   make(map[string]int),
	}

	for name, limit := range limits {
		if limit != nil {
			bc.limits[name] = *limit
		}
	}

	return bc
}

// HasBudget checks if an effect has a budget limit configured
//
// Parameters:
//   - effect: The effect name to check
//
// Returns:
//   - true if a budget limit exists for this effect
//   - false if the effect is unlimited
func (bc *BudgetContext) HasBudget(effect string) bool {
	_, ok := bc.limits[effect]
	return ok
}

// Limit returns the budget limit for an effect
//
// Parameters:
//   - effect: The effect name
//
// Returns:
//   - The limit value and true if a budget exists
//   - 0 and false if the effect is unlimited
func (bc *BudgetContext) Limit(effect string) (int, bool) {
	limit, ok := bc.limits[effect]
	return limit, ok
}

// Used returns how many times an effect has been used
//
// Parameters:
//   - effect: The effect name
//
// Returns:
//   - The current usage count (0 if never used)
func (bc *BudgetContext) Used(effect string) int {
	return bc.used[effect]
}

// Remaining returns how many uses are left for an effect
//
// Parameters:
//   - effect: The effect name
//
// Returns:
//   - The remaining uses, or -1 if unlimited
func (bc *BudgetContext) Remaining(effect string) int {
	limit, hasLimit := bc.limits[effect]
	if !hasLimit {
		return -1 // Unlimited
	}
	remaining := limit - bc.used[effect]
	if remaining < 0 {
		return 0
	}
	return remaining
}

// CheckAndConsume attempts to use one unit of budget for an effect
//
// This is the main method called before each effect operation.
// It checks if budget is available and consumes one unit if so.
//
// Parameters:
//   - effect: The effect name
//   - position: Source position for error reporting (optional)
//
// Returns:
//   - nil if the operation is allowed (budget consumed or unlimited)
//   - BudgetExhaustedError if the budget is exhausted
//
// Example:
//
//	if err := ctx.Budget.CheckAndConsume("IO", "file.ail:10:5"); err != nil {
//	    return nil, err // Budget exhausted
//	}
//	// Proceed with IO operation
func (bc *BudgetContext) CheckAndConsume(effect, position string) error {
	limit, hasLimit := bc.limits[effect]
	if !hasLimit {
		// No budget limit, allow operation
		bc.used[effect]++ // Still track usage for debugging
		return nil
	}

	if bc.used[effect] >= limit {
		return NewBudgetExhaustedError(effect, limit, bc.used[effect], position)
	}

	bc.used[effect]++
	return nil
}

// Clone creates a copy of the budget context
//
// Used for nested scopes where a fresh budget is needed.
// The clone has the same limits but zero usage.
//
// Returns:
//   - A new BudgetContext with same limits, fresh usage counters
func (bc *BudgetContext) Clone() *BudgetContext {
	newLimits := make(map[string]*int)
	for name, limit := range bc.limits {
		val := limit
		newLimits[name] = &val
	}
	return NewBudgetContext(newLimits)
}

// Reset clears all usage counters
//
// Used when entering a new function scope with per-invocation budget semantics.
func (bc *BudgetContext) Reset() {
	bc.used = make(map[string]int)
}

// Merge combines two budget contexts
//
// For budget composition (e.g., nested scopes), limits are summed.
// Usage is taken from the current context.
//
// Parameters:
//   - other: Another budget context to merge
//
// Returns:
//   - A new BudgetContext with combined limits
func (bc *BudgetContext) Merge(other *BudgetContext) *BudgetContext {
	if other == nil {
		return bc.Clone()
	}

	merged := make(map[string]*int)

	// Add limits from bc
	for name, limit := range bc.limits {
		val := limit
		merged[name] = &val
	}

	// Add limits from other (sum if both have the same effect)
	for name, limit := range other.limits {
		if existing, ok := merged[name]; ok && existing != nil {
			sum := *existing + limit
			merged[name] = &sum
		} else {
			val := limit
			merged[name] = &val
		}
	}

	return NewBudgetContext(merged)
}
