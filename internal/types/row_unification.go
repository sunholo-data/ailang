package types

import (
	"fmt"
	"sort"
)

// RowUnifier handles row unification with principal types
type RowUnifier struct {
	freshCounter int
}

// NewRowUnifier creates a new row unifier
func NewRowUnifier() *RowUnifier {
	return &RowUnifier{freshCounter: 0}
}

// UnifyRows unifies two rows, returning an updated substitution or error
func (ru *RowUnifier) UnifyRows(r1, r2 *Row, sub Substitution) (Substitution, error) {
	// 1. Check kind compatibility
	if !r1.Kind.Equals(r2.Kind) {
		return nil, fmt.Errorf("kind mismatch in row unification: %s vs %s", r1.Kind, r2.Kind)
	}

	// 2. Apply current substitution and canonicalize
	r1 = ru.applySubToRow(sub, r1)
	r2 = ru.applySubToRow(sub, r2)
	r1 = ru.canonicalizeRow(r1)
	r2 = ru.canonicalizeRow(r2)

	// 3. Find common and unique labels
	common := make(map[string]bool)
	only1 := make(map[string]Type)
	only2 := make(map[string]Type)

	for label, typ := range r1.Labels {
		if _, ok := r2.Labels[label]; ok {
			common[label] = true
		} else {
			only1[label] = typ
		}
	}

	for label, typ := range r2.Labels {
		if !common[label] {
			only2[label] = typ
		}
	}

	// 4. Unify common labels
	unifier := NewUnifier()
	for label := range common {
		var err error
		sub, err = unifier.Unify(r1.Labels[label], r2.Labels[label], sub)
		if err != nil {
			return nil, fmt.Errorf("failed to unify label %s: %w", label, err)
		}
	}

	// 5. Handle tails and unique labels
	switch {
	case r1.Tail == nil && r2.Tail == nil:
		// Both closed rows - must have same labels
		if len(only1) > 0 || len(only2) > 0 {
			missing := ru.labelNames(only1)
			extra := ru.labelNames(only2)
			return nil, fmt.Errorf("incompatible closed rows: r1 has extra labels %v, r2 has extra labels %v", missing, extra)
		}

	case r1.Tail != nil && r2.Tail == nil:
		// r1 open, r2 closed - r1's tail gets r2's unique labels
		// CRITICAL FIX: Assign only2 (r2's unique labels) to r1.Tail, not only1!
		// This allows open row r1 to unify with closed row r2 by accepting r2's labels.
		sub[r1.Tail.Name] = &Row{
			Kind:   r1.Kind,
			Labels: only2, // r2's unique labels, not only1!
			Tail:   nil,   // Close the row after adding r2's labels
		}

	case r1.Tail == nil && r2.Tail != nil:
		// r1 closed, r2 open - r2's tail gets r1's unique labels
		// CRITICAL FIX: Assign only1 (r1's unique labels) to r2.Tail, not only2!
		// This allows open row r2 to unify with closed row r1 by accepting r1's labels.
		// For example: {IO} (closed) unifies with {} | ε (open) by setting ε := {IO}
		sub[r2.Tail.Name] = &Row{
			Kind:   r2.Kind,
			Labels: only1, // r1's unique labels, not only2!
			Tail:   nil,   // Close the row after adding r1's labels
		}

	case r1.Tail != nil && r2.Tail != nil:
		// Both open - need principal unifier
		if r1.Tail.Name == r2.Tail.Name {
			// Same row variable - must have same unique labels
			if len(only1) > 0 || len(only2) > 0 {
				return nil, fmt.Errorf("same row variable with different extensions")
			}
		} else {
			// Different row variables - create fresh var for remainder
			fresh := ru.freshRowVar(r1.Kind)

			// r1.Tail := only2 ∪ fresh
			sub[r1.Tail.Name] = &Row{
				Kind:   r1.Kind,
				Labels: only2,
				Tail:   fresh,
			}

			// r2.Tail := only1 ∪ fresh
			sub[r2.Tail.Name] = &Row{
				Kind:   r2.Kind,
				Labels: only1,
				Tail:   fresh,
			}
		}
	}

	return sub, nil
}

// applySubToRow applies a substitution to a row
func (ru *RowUnifier) applySubToRow(sub Substitution, r *Row) *Row {
	if r == nil {
		return nil
	}

	// Apply substitution to labels
	labels := make(map[string]Type)
	for k, v := range r.Labels {
		labels[k] = ApplySubstitution(sub, v)
	}

	// Apply substitution to tail
	var tail *RowVar
	if r.Tail != nil {
		if subType, ok := sub[r.Tail.Name]; ok {
			// Tail is substituted
			if subRow, ok := subType.(*Row); ok {
				// Merge labels
				for k, v := range subRow.Labels {
					if _, exists := labels[k]; exists {
						// This shouldn't happen with correct unification
						panic(fmt.Sprintf("label collision during substitution: %s", k))
					}
					labels[k] = v
				}
				tail = subRow.Tail
			} else if subVar, ok := subType.(*RowVar); ok {
				tail = subVar
			} else {
				panic(fmt.Sprintf("row variable substituted with non-row type: %T", subType))
			}
		} else {
			tail = r.Tail
		}
	}

	return &Row{
		Kind:   r.Kind,
		Labels: labels,
		Tail:   tail,
	}
}

// canonicalizeRow returns the canonical representation of a row
func (ru *RowUnifier) canonicalizeRow(r *Row) *Row {
	if r == nil {
		return nil
	}
	// Labels are already stored in a map, just ensure consistent ordering in String()
	// The actual canonical form is maintained by sorted output
	return r
}

// freshRowVar creates a fresh row variable
func (ru *RowUnifier) freshRowVar(kind Kind) *RowVar {
	ru.freshCounter++
	return &RowVar{
		Name: fmt.Sprintf("ρ%d", ru.freshCounter),
		Kind: kind,
	}
}

// labelNames extracts sorted label names from a map
func (ru *RowUnifier) labelNames(labels map[string]Type) []string {
	var names []string
	for k := range labels {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// UnionEffects computes the union of multiple effect rows
func UnionEffects(rows ...*Row) *Row {
	if len(rows) == 0 {
		return EmptyEffectRow()
	}

	// Check all are effect rows
	for _, r := range rows {
		if r != nil && !r.Kind.Equals(EffectRow) {
			panic(fmt.Sprintf("UnionEffects called with non-effect row: %s", r.Kind))
		}
	}

	// Collect all labels
	allLabels := make(map[string]Type)
	var tails []*RowVar

	for _, r := range rows {
		if r == nil {
			continue
		}
		for k, v := range r.Labels {
			allLabels[k] = v // For effects, value is usually unit
		}
		if r.Tail != nil {
			tails = append(tails, r.Tail)
		}
	}

	// For now, if any row has a tail, we need a fresh tail
	// (proper handling would require constraint solving)
	var tail *RowVar
	if len(tails) > 0 {
		// In a full implementation, we'd generate constraints
		// For now, just take the first tail
		tail = tails[0]
	}

	return &Row{
		Kind:   EffectRow,
		Labels: allLabels,
		Tail:   tail,
	}
}

// RecordSelection checks if a record type has a field and returns its type
func RecordSelection(record *Row, field string) (Type, error) {
	if !record.Kind.Equals(RecordRow) {
		return nil, fmt.Errorf("selection from non-record row")
	}

	if typ, ok := record.Labels[field]; ok {
		return typ, nil
	}

	if record.Tail != nil {
		// Field might be in the tail - return a constraint
		return nil, fmt.Errorf("field %s not found; may be in row extension", field)
	}

	return nil, fmt.Errorf("field %s not found in record", field)
}
