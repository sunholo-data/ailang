package types

import (
	"fmt"
	"sort"

	"github.com/sunholo/ailang/internal/ast"
)

// isEffectRowVar returns true if name is a lowercase identifier (effect row variable)
func isEffectRowVar(name string) bool {
	return len(name) > 0 && name[0] >= 'a' && name[0] <= 'z'
}

// ElaborateEffectRow converts AST effect names to a normalized effect row
// Returns nil for empty effect sets (purity sentinel)
// Labels are sorted alphabetically for determinism
// Supports effect row variables (lowercase identifiers like 'e') for polymorphism
func ElaborateEffectRow(effectNames []string) (*Row, error) {
	if len(effectNames) == 0 {
		return nil, nil // Purity sentinel
	}

	// Validate and collect effects, separating row variables from concrete effects
	validatedEffects := make(map[string]bool)
	var rowVarName string
	for _, name := range effectNames {
		if isEffectRowVar(name) {
			if rowVarName != "" && rowVarName != name {
				return nil, fmt.Errorf("multiple effect row variables: %s, %s", rowVarName, name)
			}
			rowVarName = name
			continue
		}
		// Validate against known effects
		if !IsKnownEffect(name) {
			return nil, fmt.Errorf("unknown effect: %s", name)
		}
		validatedEffects[name] = true
	}

	// Convert to sorted slice for determinism
	sortedNames := make([]string, 0, len(validatedEffects))
	for name := range validatedEffects {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	// Build effect row with sorted labels
	labels := make(map[string]Type)
	for _, name := range sortedNames {
		// Each effect maps to Unit type (effects are just labels)
		labels[name] = Unit()
	}

	// Set row tail for effect polymorphism
	var tail *RowVar
	if rowVarName != "" {
		tail = &RowVar{Name: rowVarName, Kind: EffectRow}
	}

	return &Row{
		Kind:   EffectRow,
		Labels: labels,
		Tail:   tail,
	}, nil
}

// ElaborateEffectRowWithBudgets converts AST effect annotations (with optional budgets)
// to a normalized effect row. Returns nil for empty effect sets (purity sentinel).
// Labels are sorted alphabetically for determinism.
// Supports @limit=N (max) and @min=N (minimum) annotations (M-DX25 M4).
// Supports effect row variables (lowercase identifiers like 'e') for polymorphism.
func ElaborateEffectRowWithBudgets(effects []ast.EffectAnnotation) (*Row, error) {
	if len(effects) == 0 {
		return nil, nil // Purity sentinel
	}

	// Validate and collect effects with budgets, separating row variables
	validatedEffects := make(map[string]bool)
	budgets := make(map[string]*int)
	minBudgets := make(map[string]*int)
	var rowVarName string

	for _, eff := range effects {
		if eff.IsRowVar || isEffectRowVar(eff.Name) {
			if rowVarName != "" && rowVarName != eff.Name {
				return nil, fmt.Errorf("multiple effect row variables: %s, %s", rowVarName, eff.Name)
			}
			rowVarName = eff.Name
			continue
		}
		// Validate against known effects
		if !IsKnownEffect(eff.Name) {
			return nil, fmt.Errorf("unknown effect: %s", eff.Name)
		}
		validatedEffects[eff.Name] = true
		if eff.Budget != nil {
			// Copy budget value to avoid pointer aliasing
			val := *eff.Budget
			budgets[eff.Name] = &val
		}
		if eff.Min != nil {
			// Copy min value to avoid pointer aliasing
			val := *eff.Min
			minBudgets[eff.Name] = &val
		}
	}

	// Convert to sorted slice for determinism
	sortedNames := make([]string, 0, len(validatedEffects))
	for name := range validatedEffects {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	// Build effect row with sorted labels
	labels := make(map[string]Type)
	for _, name := range sortedNames {
		// Each effect maps to Unit type (effects are just labels)
		labels[name] = Unit()
	}

	// Only include budgets/minBudgets maps if there are any values
	var budgetsMap map[string]*int
	if len(budgets) > 0 {
		budgetsMap = budgets
	}
	var minBudgetsMap map[string]*int
	if len(minBudgets) > 0 {
		minBudgetsMap = minBudgets
	}

	// Set row tail for effect polymorphism
	var tail *RowVar
	if rowVarName != "" {
		tail = &RowVar{Name: rowVarName, Kind: EffectRow}
	}

	return &Row{
		Kind:       EffectRow,
		Labels:     labels,
		Tail:       tail,
		Budgets:    budgetsMap,
		MinBudgets: minBudgetsMap,
	}, nil
}

// IsKnownEffect checks if an effect name is one of the canonical effects
func IsKnownEffect(name string) bool {
	knownEffects := map[string]bool{
		"IO":          true,
		"FS":          true,
		"Net":         true,
		"Clock":       true,
		"Rand":        true,
		"DB":          true,
		"Trace":       true,
		"Async":       true,
		"Env":         true, // v0.4.0: Environment variable access
		"Debug":       true, // v0.4.10: Structured tracing/assertions (ghost effect)
		"AI":          true, // v0.5.1: General-purpose AI oracle
		"SharedMem":   true, // v0.5.11: Shared memory cache effect (M-DX15)
		"SharedIndex": true, // v0.5.11: Similarity index for semantic retrieval (M-DX16)
	}
	return knownEffects[name]
}

// Unit returns the Unit type
func Unit() Type {
	return &TCon{Name: "()"}
}

// UnionEffectRows creates the union of two effect rows
// nil is treated as the identity (pure)
// Result is always sorted for determinism
// For budgets: when both rows have a budget for the same effect, budgets are summed
// (nested scopes compose by addition for total allowed invocations)
func UnionEffectRows(a, b *Row) *Row {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}

	// Merge labels
	merged := make(map[string]struct{})
	for k := range a.Labels {
		merged[k] = struct{}{}
	}
	for k := range b.Labels {
		merged[k] = struct{}{}
	}

	// Convert to sorted slice
	labels := make([]string, 0, len(merged))
	for k := range merged {
		labels = append(labels, k)
	}
	sort.Strings(labels)

	// Build effect row
	effectLabels := make(map[string]Type)
	for _, name := range labels {
		effectLabels[name] = Unit()
	}

	// Merge budgets - sum budgets when both have them
	var budgets map[string]*int
	if a.Budgets != nil || b.Budgets != nil {
		budgets = make(map[string]*int)
		for _, name := range labels {
			var aBudget, bBudget *int
			if a.Budgets != nil {
				aBudget = a.Budgets[name]
			}
			if b.Budgets != nil {
				bBudget = b.Budgets[name]
			}

			// Compose budgets: sum if both present, take available otherwise
			if aBudget != nil && bBudget != nil {
				sum := *aBudget + *bBudget
				budgets[name] = &sum
			} else if aBudget != nil {
				val := *aBudget
				budgets[name] = &val
			} else if bBudget != nil {
				val := *bBudget
				budgets[name] = &val
			}
			// If neither has budget, the effect is unlimited (no entry in budgets map)
		}
		// Clean up empty budgets map
		if len(budgets) == 0 {
			budgets = nil
		}
	}

	// If no effects after merging, return nil (pure)
	if len(effectLabels) == 0 {
		return nil
	}

	return &Row{
		Kind:    EffectRow,
		Labels:  effectLabels,
		Tail:    nil,
		Budgets: budgets,
	}
}

// SubsumeEffectRows checks if effect row 'a' is subsumed by effect row 'b'
// Returns true if all effects in 'a' are present in 'b'
// Pure (nil) is subsumed by anything
func SubsumeEffectRows(a, b *Row) bool {
	if a == nil {
		return true // Pure is subsumed by anything
	}
	if b == nil {
		return a == nil // Only pure is subsumed by pure
	}

	// All labels in 'a' must be in 'b'
	for k := range a.Labels {
		if _, ok := b.Labels[k]; !ok {
			return false
		}
	}
	return true
}

// EffectRowDifference returns the effects in 'a' that are not in 'b'
// Result is sorted alphabetically
func EffectRowDifference(a, b *Row) []string {
	if a == nil {
		return nil
	}

	var diff []string
	for k := range a.Labels {
		if b == nil || b.Labels[k] == nil {
			diff = append(diff, k)
		}
	}

	sort.Strings(diff)
	return diff
}

// FormatEffectRow formats an effect row for display
// Returns "! {IO, FS}" for non-empty rows, "" for pure (nil)
// Includes budget annotations when present: "! {IO @limit=5, FS}"
func FormatEffectRow(row *Row) string {
	if row == nil || len(row.Labels) == 0 {
		return "" // Pure function, omit effect annotation
	}

	// Sort labels for deterministic output
	labels := make([]string, 0, len(row.Labels))
	for k := range row.Labels {
		labels = append(labels, k)
	}
	sort.Strings(labels)

	// Format as ! {Effect1, Effect2, ...} with optional budgets
	result := "! {"
	for i, label := range labels {
		if i > 0 {
			result += ", "
		}
		// Include budget annotation if present
		if row.Budgets != nil {
			if budget, ok := row.Budgets[label]; ok && budget != nil {
				result += fmt.Sprintf("%s @limit=%d", label, *budget)
			} else {
				result += label
			}
		} else {
			result += label
		}
	}
	result += "}"

	return result
}
