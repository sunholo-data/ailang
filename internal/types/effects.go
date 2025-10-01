package types

import (
	"fmt"
	"sort"
)

// ElaborateEffectRow converts AST effect names to a normalized effect row
// Returns nil for empty effect sets (purity sentinel)
// Labels are sorted alphabetically for determinism
func ElaborateEffectRow(effectNames []string) (*Row, error) {
	if len(effectNames) == 0 {
		return nil, nil // Purity sentinel
	}

	// Validate and collect effects
	validatedEffects := make(map[string]bool)
	for _, name := range effectNames {
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

	return &Row{
		Kind:   EffectRow,
		Labels: labels,
		Tail:   nil, // Closed row (no polymorphism in v0.1.0)
	}, nil
}

// IsKnownEffect checks if an effect name is one of the canonical effects
func IsKnownEffect(name string) bool {
	knownEffects := map[string]bool{
		"IO":    true,
		"FS":    true,
		"Net":   true,
		"Clock": true,
		"Rand":  true,
		"DB":    true,
		"Trace": true,
		"Async": true,
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

	return &Row{
		Kind:   EffectRow,
		Labels: effectLabels,
		Tail:   nil,
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

	// Format as ! {Effect1, Effect2, ...}
	result := "! {"
	for i, label := range labels {
		if i > 0 {
			result += ", "
		}
		result += label
	}
	result += "}"

	return result
}
