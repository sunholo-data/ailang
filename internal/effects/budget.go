package effects

// BudgetContext tracks per-effect usage counts against configured limits
//
// Budgets are type-level constraints that limit how many times an effect
// can be invoked within a function scope. Each function invocation gets
// a fresh budget (per-invocation semantics).
//
// M-DX25 M4: Now supports minimum usage requirements (@min=N).
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
	limits       map[string]int // Effect name → max limit (absent = unlimited)
	minLimits    map[string]int // Effect name → minimum required (absent = no minimum) (M-DX25 M4)
	used         map[string]int // Effect name → semantic usage count (enforced against limits)
	physicalUsed map[string]int // Effect name → physical usage count (always tracked)
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
		limits:       make(map[string]int),
		minLimits:    make(map[string]int),
		used:         make(map[string]int),
		physicalUsed: make(map[string]int),
	}

	for name, limit := range limits {
		if limit != nil {
			bc.limits[name] = *limit
		}
	}

	return bc
}

// NewBudgetContextWithMin creates a new budget context with limits and minimums
//
// M-DX25 M4: Supports minimum usage requirements to verify effects were exercised.
//
// Parameters:
//   - limits: Map of effect name to max budget limit. nil values mean unlimited.
//   - minLimits: Map of effect name to minimum required usage. nil values mean no minimum.
//
// Example:
//
//	// IO: at least 1 call, at most 5 calls
//	limits := map[string]*int{"IO": intPtr(5)}
//	minLimits := map[string]*int{"IO": intPtr(1)}
//	budget := NewBudgetContextWithMin(limits, minLimits)
func NewBudgetContextWithMin(limits map[string]*int, minLimits map[string]*int) *BudgetContext {
	bc := NewBudgetContext(limits)
	for name, min := range minLimits {
		if min != nil {
			bc.minLimits[name] = *min
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

// Used returns how many times an effect has been used (semantic count)
//
// Parameters:
//   - effect: The effect name
//
// Returns:
//   - The current semantic usage count (0 if never used)
func (bc *BudgetContext) Used(effect string) int {
	return bc.used[effect]
}

// PhysicalUsed returns the physical usage count for an effect
//
// Physical counts track actual builtin invocations, independent of
// semantic budget charging. Always incremented, even with --no-budgets.
//
// Parameters:
//   - effect: The effect name
//
// Returns:
//   - The current physical usage count (0 if never used)
func (bc *BudgetContext) PhysicalUsed(effect string) int {
	return bc.physicalUsed[effect]
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

// HasMinimum checks if an effect has a minimum usage requirement
//
// M-DX25 M4: Minimum budgets ensure effects were actually exercised.
//
// Parameters:
//   - effect: The effect name to check
//
// Returns:
//   - true if a minimum requirement exists for this effect
func (bc *BudgetContext) HasMinimum(effect string) bool {
	_, ok := bc.minLimits[effect]
	return ok
}

// Minimum returns the minimum usage requirement for an effect
//
// Parameters:
//   - effect: The effect name
//
// Returns:
//   - The minimum value and true if a minimum exists
//   - 0 and false if no minimum is set
func (bc *BudgetContext) Minimum(effect string) (int, bool) {
	min, ok := bc.minLimits[effect]
	return min, ok
}

// CheckMinimum verifies that minimum usage requirements are met
//
// M-DX25 M4: Called on scope exit to ensure effects were exercised.
// Uses physical usage count (actual calls) for verification.
//
// Parameters:
//   - position: Source position for error reporting (optional)
//
// Returns:
//   - nil if all minimums are met
//   - BudgetUnderrunError if any minimum is not satisfied
func (bc *BudgetContext) CheckMinimum(position string) error {
	for effect, min := range bc.minLimits {
		physical := bc.physicalUsed[effect]
		if physical < min {
			return NewBudgetUnderrunError(effect, min, physical, position)
		}
	}
	return nil
}

// MinLimitsMap returns a copy of the minimum limits per effect
//
// Returns:
//   - Map of effect name to minimum requirement
func (bc *BudgetContext) MinLimitsMap() map[string]int {
	result := make(map[string]int, len(bc.minLimits))
	for k, v := range bc.minLimits {
		result[k] = v
	}
	return result
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
	// M-DX25: Always increment physical usage (for reporting)
	bc.physicalUsed[effect]++

	limit, hasLimit := bc.limits[effect]
	if !hasLimit {
		// No budget limit, allow operation
		bc.used[effect]++ // Still track semantic usage for debugging
		return nil
	}

	if bc.used[effect] >= limit {
		return NewBudgetExhaustedError(effect, limit, bc.used[effect], position, bc.physicalUsed[effect])
	}

	bc.used[effect]++
	return nil
}

// ChargeSemanticOnly increments only the semantic usage count
//
// M-DX25: Used by PopScopeAndChargeCaller to charge the caller's
// semantic budget with the callee's declared amount. This does NOT
// increment physical usage (which is tracked separately at builtin invocation).
//
// Parameters:
//   - effect: The effect name
//   - count: The amount to charge (typically callee's declared @limit)
func (bc *BudgetContext) ChargeSemanticOnly(effect string, count int) {
	bc.used[effect] += count
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
	newMinLimits := make(map[string]*int)
	for name, min := range bc.minLimits {
		val := min
		newMinLimits[name] = &val
	}
	return NewBudgetContextWithMin(newLimits, newMinLimits)
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

// UsageMap returns a copy of the semantic usage counts per effect
//
// Returns:
//   - Map of effect name to semantic usage count
func (bc *BudgetContext) UsageMap() map[string]int {
	result := make(map[string]int, len(bc.used))
	for k, v := range bc.used {
		result[k] = v
	}
	return result
}

// PhysicalUsageMap returns a copy of the physical usage counts per effect
//
// Returns:
//   - Map of effect name to physical usage count
func (bc *BudgetContext) PhysicalUsageMap() map[string]int {
	result := make(map[string]int, len(bc.physicalUsed))
	for k, v := range bc.physicalUsed {
		result[k] = v
	}
	return result
}

// LimitsMap returns a copy of the limits per effect
//
// Returns:
//   - Map of effect name to limit (-1 means check if it exists in the map)
func (bc *BudgetContext) LimitsMap() map[string]int {
	result := make(map[string]int, len(bc.limits))
	for k, v := range bc.limits {
		result[k] = v
	}
	return result
}

// BudgetReport tracks per-function, per-effect budget usage for reporting
//
// M-DX25: Tracks both semantic (declared budget) and physical (actual calls) counts.
//
// Example output:
//
//	Budget Report:
//	  main:               IO semantic 3/5 physical 19
//	  getDefaultProject:  FS semantic 5/5 physical 12
type BudgetReport struct {
	// FunctionUsage maps function name -> effect name -> semantic usage count
	FunctionUsage map[string]map[string]int
	// FunctionPhysicalUsage maps function name -> effect name -> physical usage count
	FunctionPhysicalUsage map[string]map[string]int
	// FunctionLimits maps function name -> effect name -> limit (nil = unlimited)
	FunctionLimits map[string]map[string]*int
	// TotalUsage aggregates all semantic effect usage
	TotalUsage map[string]int
	// TotalPhysicalUsage aggregates all physical effect usage
	TotalPhysicalUsage map[string]int
	// CurrentFunction tracks which function we're recording for
	CurrentFunction string
}

// NewBudgetReport creates a new budget report
func NewBudgetReport() *BudgetReport {
	return &BudgetReport{
		FunctionUsage:         make(map[string]map[string]int),
		FunctionPhysicalUsage: make(map[string]map[string]int),
		FunctionLimits:        make(map[string]map[string]*int),
		TotalUsage:            make(map[string]int),
		TotalPhysicalUsage:    make(map[string]int),
	}
}

// EnterFunction marks entry into a function for attribution
func (br *BudgetReport) EnterFunction(name string, limits map[string]int) {
	br.CurrentFunction = name
	if _, ok := br.FunctionUsage[name]; !ok {
		br.FunctionUsage[name] = make(map[string]int)
	}
	if _, ok := br.FunctionPhysicalUsage[name]; !ok {
		br.FunctionPhysicalUsage[name] = make(map[string]int)
	}
	if _, ok := br.FunctionLimits[name]; !ok {
		br.FunctionLimits[name] = make(map[string]*int)
		for effect, limit := range limits {
			val := limit
			br.FunctionLimits[name][effect] = &val
		}
	}
}

// RecordUsage records physical effect usage for the current function
// This is called for each actual builtin invocation (physical count)
func (br *BudgetReport) RecordUsage(effect string, count int) {
	if br.CurrentFunction == "" {
		br.CurrentFunction = "<global>"
	}
	// Track physical usage (actual calls)
	if _, ok := br.FunctionPhysicalUsage[br.CurrentFunction]; !ok {
		br.FunctionPhysicalUsage[br.CurrentFunction] = make(map[string]int)
	}
	br.FunctionPhysicalUsage[br.CurrentFunction][effect] += count
	br.TotalPhysicalUsage[effect] += count

	// Also track semantic usage (for now same as physical until M3 scoped charging)
	if _, ok := br.FunctionUsage[br.CurrentFunction]; !ok {
		br.FunctionUsage[br.CurrentFunction] = make(map[string]int)
	}
	br.FunctionUsage[br.CurrentFunction][effect] += count
	br.TotalUsage[effect] += count
}

// ExitFunction marks exit from a function, recording final usage from budget context
func (br *BudgetReport) ExitFunction(name string, budget *BudgetContext) {
	if budget == nil {
		return
	}
	// Record semantic usage from budget context
	usage := budget.UsageMap()
	for effect, count := range usage {
		if _, ok := br.FunctionUsage[name]; !ok {
			br.FunctionUsage[name] = make(map[string]int)
		}
		br.FunctionUsage[name][effect] = count
		br.TotalUsage[effect] += count
	}
	// Record physical usage from budget context
	physicalUsage := budget.PhysicalUsageMap()
	for effect, count := range physicalUsage {
		if _, ok := br.FunctionPhysicalUsage[name]; !ok {
			br.FunctionPhysicalUsage[name] = make(map[string]int)
		}
		br.FunctionPhysicalUsage[name][effect] = count
		br.TotalPhysicalUsage[effect] += count
	}
}

// HasUsage returns true if any usage was recorded
func (br *BudgetReport) HasUsage() bool {
	return len(br.TotalUsage) > 0
}
