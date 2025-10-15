package types

import (
	"fmt"
)

// Substitution maps variable names to types
type Substitution map[string]Type

// Unifier handles type unification with occurs check
type Unifier struct {
	rowUnifier *RowUnifier
}

// NewUnifier creates a new unifier
func NewUnifier() *Unifier {
	return &Unifier{
		rowUnifier: NewRowUnifier(),
	}
}

// Unify attempts to unify two types, returning an updated substitution
func (u *Unifier) Unify(t1, t2 Type, sub Substitution) (Substitution, error) {
	// Apply current substitution
	t1 = ApplySubstitution(sub, t1)
	t2 = ApplySubstitution(sub, t2)

	// Check if already equal
	if t1.Equals(t2) {
		return sub, nil
	}

	switch t1 := t1.(type) {
	case *TVar2:
		// Type variable unification
		if u.occurs(t1.Name, t2, t1.Kind) {
			return nil, fmt.Errorf("occurs check failed: %s occurs in %s", t1.Name, t2.String())
		}
		if !u.kindsCompatible(t1.Kind, GetKind(t2)) {
			return nil, fmt.Errorf("kind mismatch: variable %s has kind %s, but %s has kind %s",
				t1.Name, t1.Kind, t2.String(), GetKind(t2))
		}
		sub[t1.Name] = t2
		return sub, nil

	case *RowVar:
		// Row variable unification
		if u.occurs(t1.Name, t2, t1.Kind) {
			return nil, fmt.Errorf("occurs check failed: row variable %s occurs in %s", t1.Name, t2.String())
		}
		if !u.kindsCompatible(t1.Kind, GetKind(t2)) {
			return nil, fmt.Errorf("kind mismatch: row variable %s has kind %s, but %s has kind %s",
				t1.Name, t1.Kind, t2.String(), GetKind(t2))
		}
		sub[t1.Name] = t2
		return sub, nil

	case *Row:
		// Row unification
		if t2Row, ok := t2.(*Row); ok {
			return u.rowUnifier.UnifyRows(t1, t2Row, sub)
		}
		if t2Var, ok := t2.(*RowVar); ok {
			// Swap and retry
			return u.Unify(t2Var, t1, sub)
		}
		return nil, fmt.Errorf("cannot unify row with %T", t2)

	case *TCon:
		// Type constructor unification
		if t2Con, ok := t2.(*TCon); ok {
			if t1.Name == t2Con.Name {
				return sub, nil
			}
			return nil, fmt.Errorf("cannot unify type constructors: %s vs %s", t1.Name, t2Con.Name)
		}
		if t2Var, ok := t2.(*TVar2); ok {
			// Swap and retry
			return u.Unify(t2Var, t1, sub)
		}
		return nil, fmt.Errorf("cannot unify type constructor %s with %T", t1.Name, t2)

	case *TFunc2:
		// Function type unification
		if t2Func, ok := t2.(*TFunc2); ok {
			if len(t1.Params) != len(t2Func.Params) {
				return nil, fmt.Errorf("function arity mismatch: %d vs %d", len(t1.Params), len(t2Func.Params))
			}

			// Unify parameters
			for i := range t1.Params {
				var err error
				sub, err = u.Unify(t1.Params[i], t2Func.Params[i], sub)
				if err != nil {
					return nil, fmt.Errorf("failed to unify parameter %d: %w", i, err)
				}
			}

			// Unify effect rows
			if t1.EffectRow != nil || t2Func.EffectRow != nil {
				eff1 := t1.EffectRow
				if eff1 == nil {
					eff1 = EmptyEffectRow()
				}
				eff2 := t2Func.EffectRow
				if eff2 == nil {
					eff2 = EmptyEffectRow()
				}
				var err error
				sub, err = u.Unify(eff1, eff2, sub)
				if err != nil {
					return nil, fmt.Errorf("failed to unify effect rows: %w", err)
				}
			}

			// Unify return type
			return u.Unify(t1.Return, t2Func.Return, sub)
		}
		if t2Var, ok := t2.(*TVar2); ok {
			// Swap and retry
			return u.Unify(t2Var, t1, sub)
		}
		return nil, fmt.Errorf("cannot unify function type with %T", t2)

	case *TList:
		// List type unification
		if t2List, ok := t2.(*TList); ok {
			return u.Unify(t1.Element, t2List.Element, sub)
		}
		if t2Var, ok := t2.(*TVar2); ok {
			// Swap and retry
			return u.Unify(t2Var, t1, sub)
		}
		// Handle old type system compatibility: TCon might represent String
		if t2Con, ok := t2.(*TCon); ok {
			// If trying to unify list with string, fail with better error
			if t2Con.Name == "String" {
				return nil, fmt.Errorf("type mismatch: cannot use list where string expected")
			}
			// Other TCon cases fail as before
			return nil, fmt.Errorf("cannot unify list type with %T", t2)
		}
		return nil, fmt.Errorf("cannot unify list type with %T", t2)

	case *TTuple:
		// Tuple type unification
		if t2Tuple, ok := t2.(*TTuple); ok {
			if len(t1.Elements) != len(t2Tuple.Elements) {
				return nil, fmt.Errorf("tuple size mismatch: %d vs %d", len(t1.Elements), len(t2Tuple.Elements))
			}
			for i := range t1.Elements {
				var err error
				sub, err = u.Unify(t1.Elements[i], t2Tuple.Elements[i], sub)
				if err != nil {
					return nil, fmt.Errorf("failed to unify tuple element %d: %w", i, err)
				}
			}
			return sub, nil
		}
		if t2Var, ok := t2.(*TVar2); ok {
			// Swap and retry
			return u.Unify(t2Var, t1, sub)
		}
		return nil, fmt.Errorf("cannot unify tuple type with %T", t2)

	case *TRecord2:
		// New row-polymorphic record type (v2)
		switch t2 := t2.(type) {
		case *TRecord2:
			// TRecord2 ~ TRecord2 - unify rows
			if t1.Row == nil && t2.Row == nil {
				return sub, nil // Both empty records
			}
			if t1.Row == nil || t2.Row == nil {
				return nil, fmt.Errorf("cannot unify closed record with open record")
			}
			// Delegate to row unification
			return u.unifyRows(t1.Row, t2.Row, sub)

		case *TRecord:
			// TRecord2 ~ TRecord (old) - convert and unify
			// Check all fields in old record exist in new record
			if t1.Row == nil {
				return nil, fmt.Errorf("empty TRecord2 cannot unify with non-empty TRecord")
			}
			for fieldName, oldFieldType := range t2.Fields {
				newFieldType, exists := t1.Row.Labels[fieldName]
				if !exists {
					return nil, fmt.Errorf("field '%s' not found in TRecord2", fieldName)
				}
				var err error
				sub, err = u.Unify(newFieldType, oldFieldType, sub)
				if err != nil {
					return nil, fmt.Errorf("failed to unify field '%s': %w", fieldName, err)
				}
			}
			// If TRecord2 has more fields and old TRecord has no row var, fail
			if len(t1.Row.Labels) > len(t2.Fields) && t2.Row == nil {
				return nil, fmt.Errorf("TRecord2 has extra fields not in closed TRecord")
			}
			return sub, nil

		case *TRecordOpen:
			// TRecord2 ~ TRecordOpen - convert and unify
			// Swap and let TRecordOpen case handle it
			return u.Unify(t2, t1, sub)

		case *TVar:
			// Swap and retry
			return u.Unify(t2, t1, sub)

		case *TVar2:
			// Swap and retry
			return u.Unify(t2, t1, sub)

		default:
			return nil, fmt.Errorf("cannot unify TRecord2 with %T", t2)
		}

	case *TRecord:
		// Old record type - unify field by field
		if t2Rec, ok := t2.(*TRecord); ok {
			// Check that both have the same fields
			if len(t1.Fields) != len(t2Rec.Fields) {
				return nil, fmt.Errorf("record field count mismatch: %d vs %d", len(t1.Fields), len(t2Rec.Fields))
			}
			// Unify each field
			for name, typ1 := range t1.Fields {
				typ2, exists := t2Rec.Fields[name]
				if !exists {
					return nil, fmt.Errorf("record field '%s' not found in second record", name)
				}
				var err error
				sub, err = u.Unify(typ1, typ2, sub)
				if err != nil {
					return nil, fmt.Errorf("failed to unify record field '%s': %w", name, err)
				}
			}
			// Unify row variables if present
			if t1.Row != nil || t2Rec.Row != nil {
				row1 := t1.Row
				if row1 == nil {
					row1 = &TVar2{Name: "ρ_empty", Kind: &KRow{ElemKind: &KRecord{}}}
				}
				row2 := t2Rec.Row
				if row2 == nil {
					row2 = &TVar2{Name: "ρ_empty", Kind: &KRow{ElemKind: &KRecord{}}}
				}
				return u.Unify(row1, row2, sub)
			}
			return sub, nil
		}
		if t2Open, ok := t2.(*TRecordOpen); ok {
			// TRecord ~ TRecordOpen (reverse subsumption)
			// Swap and let TRecordOpen case handle it
			return u.Unify(t2Open, t1, sub)
		}
		if t2Var, ok := t2.(*TVar2); ok {
			// Swap and retry
			return u.Unify(t2Var, t1, sub)
		}
		return nil, fmt.Errorf("cannot unify old record type with %T", t2)

	case *TRecordOpen:
		// Open record for subsumption: {x:α | ρ} ~ {x:α, y:β, ...}
		// This is one-way unification: open record can unify with larger closed record
		switch t2 := t2.(type) {
		case *TRecord:
			// TRecordOpen ~ TRecord (subsumption)
			// Check that all fields in open record exist in closed record
			for fieldName, openFieldType := range t1.Fields {
				closedFieldType, exists := t2.Fields[fieldName]
				if !exists {
					return nil, fmt.Errorf("record field '%s' not found in concrete record", fieldName)
				}
				// Unify the field types
				var err error
				sub, err = u.Unify(openFieldType, closedFieldType, sub)
				if err != nil {
					return nil, fmt.Errorf("failed to unify field '%s': %w", fieldName, err)
				}
			}

			// Unify row variable with remaining fields
			// Row variable captures all the extra fields not mentioned in TRecordOpen
			if t1.Row != nil {
				// Collect remaining fields (those not in TRecordOpen.Fields)
				remainingFields := make(map[string]Type)
				for name, typ := range t2.Fields {
					if _, inOpen := t1.Fields[name]; !inOpen {
						remainingFields[name] = typ
					}
				}

				// Create a closed record for remaining fields
				// The row variable gets unified with this closed record's row
				if len(remainingFields) > 0 {
					// Unify row variable with remaining record's structure
					// Note: This is a simplification; proper implementation would
					// need row unification support
					if rowVar, ok := t1.Row.(*RowVar); ok {
						// For now, just record the substitution
						// Full row unification will be added in Day 2
						_ = rowVar // Placeholder
						// remainingFields captured but not used until Day 2
						_ = remainingFields
					}
				}
			}
			return sub, nil

		case *TRecordOpen:
			// TRecordOpen ~ TRecordOpen
			// Both are open records - unify common fields
			for fieldName, field1Type := range t1.Fields {
				if field2Type, exists := t2.Fields[fieldName]; exists {
					var err error
					sub, err = u.Unify(field1Type, field2Type, sub)
					if err != nil {
						return nil, fmt.Errorf("failed to unify field '%s': %w", fieldName, err)
					}
				}
			}

			// Unify row variables
			if t1.Row != nil && t2.Row != nil {
				var err error
				sub, err = u.Unify(t1.Row, t2.Row, sub)
				if err != nil {
					return nil, fmt.Errorf("failed to unify row variables: %w", err)
				}
			}
			return sub, nil

		case *TRecord2:
			// TRecordOpen ~ TRecord2 (subsumption)
			// Check that all fields in open record exist in TRecord2
			if t2.Row == nil {
				return nil, fmt.Errorf("empty TRecord2 cannot unify with open record")
			}
			for fieldName, openFieldType := range t1.Fields {
				newFieldType, exists := t2.Row.Labels[fieldName]
				if !exists {
					return nil, fmt.Errorf("field '%s' not found in TRecord2", fieldName)
				}
				// Unify the field types
				var err error
				sub, err = u.Unify(openFieldType, newFieldType, sub)
				if err != nil {
					return nil, fmt.Errorf("failed to unify field '%s': %w", fieldName, err)
				}
			}
			// Row variable captures extra fields (handled by TRecord2's tail)
			return sub, nil

		case *TVar:
			// Swap and retry
			return u.Unify(t2, t1, sub)

		default:
			return nil, fmt.Errorf("cannot unify open record with %T", t2)
		}

	default:
		// Unhandled type - no more compatibility for old type system
		return nil, fmt.Errorf("unhandled type in unification: %T", t1)
	}
}

// unifyRows unifies two row types (for TRecord2)
func (u *Unifier) unifyRows(row1, row2 *Row, sub Substitution) (Substitution, error) {
	// Check kinds match
	if !row1.Kind.Equals(row2.Kind) {
		return nil, fmt.Errorf("row kind mismatch: %v vs %v", row1.Kind, row2.Kind)
	}

	// Collect all field names from both rows
	allFields := make(map[string]bool)
	for name := range row1.Labels {
		allFields[name] = true
	}
	for name := range row2.Labels {
		allFields[name] = true
	}

	// Unify common fields
	for fieldName := range allFields {
		type1, in1 := row1.Labels[fieldName]
		type2, in2 := row2.Labels[fieldName]

		if in1 && in2 {
			// Both have field - unify types
			var err error
			sub, err = u.Unify(type1, type2, sub)
			if err != nil {
				return nil, fmt.Errorf("failed to unify field '%s': %w", fieldName, err)
			}
		} else if in1 && !in2 {
			// Only row1 has field - must unify with row2's tail
			if row2.Tail == nil {
				return nil, fmt.Errorf("field '%s' in row1 but not in closed row2", fieldName)
			}
			// Field will be captured by tail (handled below)
		} else if !in1 && in2 {
			// Only row2 has field - must unify with row1's tail
			if row1.Tail == nil {
				return nil, fmt.Errorf("field '%s' in row2 but not in closed row1", fieldName)
			}
			// Field will be captured by tail (handled below)
		}
	}

	// Unify tails with occurs check
	if row1.Tail != nil && row2.Tail != nil {
		// Check for occurs before unifying (row.Tail is already *RowVar)
		if u.occurs(row1.Tail.Name, row2.Tail, row1.Tail.Kind) {
			return nil, fmt.Errorf("occurs check failed: %s occurs in %s", row1.Tail.Name, row2.Tail.String())
		}
		if u.occurs(row2.Tail.Name, row1.Tail, row2.Tail.Kind) {
			return nil, fmt.Errorf("occurs check failed: %s occurs in %s", row2.Tail.Name, row1.Tail.String())
		}

		// Both have tails - unify them
		var err error
		sub, err = u.Unify(row1.Tail, row2.Tail, sub)
		if err != nil {
			return nil, fmt.Errorf("failed to unify row tails: %w", err)
		}
	} else if row1.Tail != nil && row2.Tail == nil {
		// row1 open, row2 closed - row1 tail must unify with empty row
		// This is only valid if row1 has no extra fields
		for name := range row1.Labels {
			if _, in2 := row2.Labels[name]; !in2 {
				return nil, fmt.Errorf("field '%s' in open row1 but closed row2", name)
			}
		}
	} else if row1.Tail == nil && row2.Tail != nil {
		// row1 closed, row2 open - row2 tail must unify with empty row
		// This is only valid if row2 has no extra fields
		for name := range row2.Labels {
			if _, in1 := row1.Labels[name]; !in1 {
				return nil, fmt.Errorf("field '%s' in open row2 but closed row1", name)
			}
		}
	}
	// Both closed (nil tails) - already checked fields above

	return sub, nil
}

// occurs performs the occurs check - ensures variable doesn't occur in type
func (u *Unifier) occurs(varName string, t Type, varKind Kind) bool {
	switch t := t.(type) {
	case *TVar2:
		// Type vars only occur in type vars of same kind
		return t.Name == varName && t.Kind.Equals(varKind)

	case *RowVar:
		// Row vars only occur in row vars of same kind
		return t.Name == varName && t.Kind.Equals(varKind)

	case *Row:
		// Check if var occurs in tail
		if t.Tail != nil && u.occurs(varName, t.Tail, varKind) {
			return true
		}
		// Check if var occurs in label types (for record rows)
		if t.Kind.Equals(RecordRow) {
			for _, typ := range t.Labels {
				if u.occurs(varName, typ, varKind) {
					return true
				}
			}
		}
		// Effect labels don't contain types, just names
		return false

	case *TCon:
		return false

	case *TFunc2:
		// Check params, return, and effect row
		for _, p := range t.Params {
			if u.occurs(varName, p, varKind) {
				return true
			}
		}
		if u.occurs(varName, t.Return, varKind) {
			return true
		}
		if t.EffectRow != nil && u.occurs(varName, t.EffectRow, varKind) {
			return true
		}
		return false

	case *TList:
		return u.occurs(varName, t.Element, varKind)

	case *TTuple:
		for _, elem := range t.Elements {
			if u.occurs(varName, elem, varKind) {
				return true
			}
		}
		return false

	case *TRecord2:
		if t.Row != nil {
			return u.occurs(varName, t.Row, varKind)
		}
		return false

	default:
		// For old types, no occurs
		return false
	}
}

// kindsCompatible checks if two kinds are compatible for unification
func (u *Unifier) kindsCompatible(k1, k2 Kind) bool {
	return k1.Equals(k2)
}

// ApplySubstitution applies a substitution to a type
func ApplySubstitution(sub Substitution, t Type) Type {
	if len(sub) == 0 {
		return t
	}
	return t.Substitute(sub)
}

// ComposeSubstitutions composes two substitutions
func ComposeSubstitutions(s1, s2 Substitution) Substitution {
	result := make(Substitution)

	// Apply s2 to all values in s1
	for k, v := range s1 {
		result[k] = ApplySubstitution(s2, v)
	}

	// Add all mappings from s2 not in s1
	for k, v := range s2 {
		if _, ok := result[k]; !ok {
			result[k] = v
		}
	}

	return result
}

// Helper functions for record operations (M-R5 Day 1.4)

// RecordHasField checks if a record type has a specific field
func RecordHasField(rec Type, field string) bool {
	switch r := rec.(type) {
	case *TRecord:
		_, exists := r.Fields[field]
		return exists
	case *TRecordOpen:
		_, exists := r.Fields[field]
		return exists
	case *TRecord2:
		if r.Row != nil {
			_, exists := r.Row.Labels[field]
			return exists
		}
		return false
	default:
		return false
	}
}

// RecordFieldType gets the type of a field in a record
func RecordFieldType(rec Type, field string) (Type, bool) {
	switch r := rec.(type) {
	case *TRecord:
		if typ, exists := r.Fields[field]; exists {
			return typ, true
		}
		return nil, false
	case *TRecordOpen:
		if typ, exists := r.Fields[field]; exists {
			return typ, true
		}
		return nil, false
	case *TRecord2:
		if r.Row != nil {
			if typ, exists := r.Row.Labels[field]; exists {
				return typ, true
			}
		}
		return nil, false
	default:
		return nil, false
	}
}

// IsOpenRecord checks if a record type is open (has row variable)
func IsOpenRecord(rec Type) bool {
	switch r := rec.(type) {
	case *TRecord:
		// Old TRecord: Row is Type, not *Row
		return r.Row != nil
	case *TRecordOpen:
		return r.Row != nil
	case *TRecord2:
		// New TRecord2: Row is *Row with Tail
		return r.Row != nil && r.Row.Tail != nil
	default:
		return false
	}
}

// Conversion helpers for record types (M-R5 Day 2.3)

// TRecordToTRecord2 converts old TRecord to new TRecord2
func TRecordToTRecord2(old *TRecord) *TRecord2 {
	if old == nil {
		return nil
	}

	// Convert to Row type
	var tail *RowVar
	if old.Row != nil {
		// Old TRecord.Row is Type (could be RowVar)
		if rv, ok := old.Row.(*RowVar); ok {
			tail = rv
		}
	}

	return &TRecord2{
		Row: &Row{
			Kind:   RecordRow,
			Labels: old.Fields,
			Tail:   tail,
		},
	}
}

// TRecord2ToTRecord converts new TRecord2 to old TRecord (for compatibility)
func TRecord2ToTRecord(new *TRecord2) *TRecord {
	if new == nil || new.Row == nil {
		return &TRecord{Fields: make(map[string]Type)}
	}

	var rowType Type
	if new.Row.Tail != nil {
		rowType = new.Row.Tail
	}

	return &TRecord{
		Fields: new.Row.Labels,
		Row:    rowType,
	}
}
