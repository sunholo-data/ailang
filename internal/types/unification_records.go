package types

import (
	"fmt"
	"sort"
	"strings"
)

// unifyRecord2 unifies TRecord2 (new row-polymorphic record type)
func (u *Unifier) unifyRecord2(t1 *TRecord2, t2 Type, sub Substitution) (Substitution, error) {
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
}

// unifyRecord unifies TRecord (old record type)
func (u *Unifier) unifyRecord(t1 *TRecord, t2 Type, sub Substitution) (Substitution, error) {
	if t2Rec, ok := t2.(*TRecord); ok {
		// Check that both have the same fields
		if len(t1.Fields) != len(t2Rec.Fields) {
			// M-GAP4: Improved error message with field lists and suggestions
			return nil, recordFieldMismatchError(t1.Fields, t2Rec.Fields)
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
		// M-CODEGEN-RECORD-TYPENAME-PRESERVATION: Propagate TypeName during unification
		// When a record literal (no TypeName) is unified with a type alias expansion
		// (has TypeName), we need to propagate the TypeName so codegen knows the nominal type.
		// This mutation is safe because unification happens during type checking, before
		// CoreTypeInfo is finalized.
		if t1.TypeName == "" && t2Rec.TypeName != "" {
			t1.TypeName = t2Rec.TypeName
		} else if t2Rec.TypeName == "" && t1.TypeName != "" {
			t2Rec.TypeName = t1.TypeName
		}
		// Unify row variables if present
		if t1.Row != nil || t2Rec.Row != nil {
			row1 := t1.Row
			if row1 == nil {
				// M-FIX-NESTED-RECORD-LIST: Use fresh name to avoid conflicts with nested records
				row1 = &TVar2{Name: u.freshRowVarName(), Kind: &KRow{ElemKind: &KRecord{}}}
			}
			row2 := t2Rec.Row
			if row2 == nil {
				// M-FIX-NESTED-RECORD-LIST: Use fresh name to avoid conflicts with nested records
				row2 = &TVar2{Name: u.freshRowVarName(), Kind: &KRow{ElemKind: &KRecord{}}}
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
	// M-CROSS-MODULE-RECORD-UNIFICATION: Handle TCon by expanding alias
	// This occurs when a nested record field type is imported from another module
	// and hasn't been expanded yet (e.g., position: SystemPos where SystemPos is TCon)
	if t2Con, ok := t2.(*TCon); ok {
		expanded := u.expandAlias(t2Con)
		if expanded != t2Con {
			// Successfully expanded - retry unification with expanded type
			return u.Unify(t1, expanded, sub)
		}
		// Can't expand - might be an ADT or unknown type
		return nil, fmt.Errorf("cannot unify record with unexpandable type constructor %s", t2Con.Name)
	}
	return nil, fmt.Errorf("cannot unify old record type with %T", t2)
}

// unifyRecordOpen unifies TRecordOpen (open record for subsumption)
func (u *Unifier) unifyRecordOpen(t1 *TRecordOpen, t2 Type, sub Substitution) (Substitution, error) {
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

	case *TVar2:
		// Swap and retry - TVar2 should unify with open record
		return u.Unify(t2, t1, sub)

	default:
		return nil, fmt.Errorf("cannot unify open record with %T", t2)
	}
}

// unifyRows unifies two row types (for TRecord2)
func (u *Unifier) unifyRows(row1, row2 *Row, sub Substitution) (Substitution, error) {
	// Check kinds match
	if !row1.Kind.Equals(row2.Kind) {
		return nil, fmt.Errorf("row kind mismatch: %v vs %v", row1.Kind, row2.Kind)
	}

	// Phase 1 (M-EFFECT-REFINEMENT): for effect rows, parameterised effects
	// must have invariant param maps. !{Rand[mode=os]} unifies with !{Rand}
	// (after default-desugar both have Params["Rand"] = {"mode": "os"}) but
	// not with !{Rand[mode=seeded]}. Records leave Params nil so this is a
	// no-op for them.
	//
	// Comparison is done via effectParamsCompatible which normalises each
	// side to its effective params (applying DefaultModeFor for nil entries).
	// This makes rows built outside the elaborator (e.g. raw stringSliceToEffectRow
	// in the pipeline) compatible with rows built via ElaborateEffectRow.
	if row1.Kind.Equals(EffectRow) {
		// Only check effects present in BOTH rows. Effects unique to one
		// row are captured by the tail (handled below) and their params
		// flow through with them.
		for effectName := range row1.Labels {
			if _, in2 := row2.Labels[effectName]; !in2 {
				continue
			}
			if !effectParamsCompatible(row1, row2, effectName) {
				return nil, fmt.Errorf("effect %s param mismatch: %v vs %v",
					effectName, paramsOf(row1, effectName), paramsOf(row2, effectName))
			}
		}
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

// recordFieldMismatchError creates a helpful error message for record field mismatches
// M-GAP4: Lists missing/extra fields and suggests open record syntax
func recordFieldMismatchError(expectedFields, actualFields map[string]Type) error {
	// Collect field names
	expected := make([]string, 0, len(expectedFields))
	for name := range expectedFields {
		expected = append(expected, name)
	}
	sort.Strings(expected)

	actual := make([]string, 0, len(actualFields))
	for name := range actualFields {
		actual = append(actual, name)
	}
	sort.Strings(actual)

	// Find extra fields (in actual but not expected)
	extraFields := []string{}
	for _, name := range actual {
		if _, exists := expectedFields[name]; !exists {
			extraFields = append(extraFields, name)
		}
	}

	// Find missing fields (in expected but not actual)
	missingFields := []string{}
	for _, name := range expected {
		if _, exists := actualFields[name]; !exists {
			missingFields = append(missingFields, name)
		}
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("record field mismatch: expected %d fields, got %d\n", len(expected), len(actual)))
	msg.WriteString(fmt.Sprintf("  expected fields: {%s}\n", strings.Join(expected, ", ")))
	msg.WriteString(fmt.Sprintf("  actual fields:   {%s}\n", strings.Join(actual, ", ")))

	if len(extraFields) > 0 {
		msg.WriteString(fmt.Sprintf("  extra fields:    %s\n", strings.Join(extraFields, ", ")))
	}
	if len(missingFields) > 0 {
		msg.WriteString(fmt.Sprintf("  missing fields:  %s\n", strings.Join(missingFields, ", ")))
	}

	// Suggest open record syntax if there are extra fields
	if len(extraFields) > 0 && len(missingFields) == 0 {
		msg.WriteString("\n  Hint: Use open record syntax to accept extra fields:\n")
		msg.WriteString(fmt.Sprintf("        {%s | r} or {%s, ...}", strings.Join(expected, ", "), strings.Join(expected, ", ")))
	}

	return fmt.Errorf("%s", msg.String())
}
