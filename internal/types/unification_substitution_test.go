package types

import (
	"testing"
)

// TestTRecordTypeNamePreservedThroughSubstitution ensures that TypeName is preserved
// when substitution creates a new TRecord. This test prevents regression of the
// M-CODEGEN-RECORD-TYPENAME-PRESERVATION bug where record literals compiled to
// map[string]interface{} instead of &Planet{} because TypeName was lost.
func TestTRecordTypeNamePreservedThroughSubstitution(t *testing.T) {
	// Create a TRecord with TypeName set (simulates what expandAlias does)
	original := &TRecord{
		Fields: map[string]Type{
			"x": &TVar{Name: "α1"}, // Type variable that will be substituted
			"y": TFloat,
		},
		TypeName: "Coord", // This MUST be preserved
	}

	// Create a substitution that changes one of the field types
	sub := Substitution{
		"α1": TInt, // α1 -> int
	}

	// Apply substitution
	result := ApplySubstitution(sub, original)

	// Verify result is a TRecord
	resultRec, ok := result.(*TRecord)
	if !ok {
		t.Fatalf("expected *TRecord, got %T", result)
	}

	// CRITICAL: TypeName must be preserved
	if resultRec.TypeName != "Coord" {
		t.Errorf("TypeName lost during substitution: expected 'Coord', got '%s'", resultRec.TypeName)
	}

	// Verify the substitution was actually applied
	if xType, ok := resultRec.Fields["x"].(*TCon); !ok || xType.Name != "int" {
		t.Errorf("expected x to be int, got %v", resultRec.Fields["x"])
	}
}

// TestTRecordTypeNamePreservedWhenUnchanged verifies that when no substitution
// is needed, the original TRecord (with TypeName) is returned.
func TestTRecordTypeNamePreservedWhenUnchanged(t *testing.T) {
	original := &TRecord{
		Fields: map[string]Type{
			"x": TInt,
			"y": TFloat,
		},
		TypeName: "Point",
	}

	// Empty substitution - nothing should change
	sub := Substitution{}
	result := ApplySubstitution(sub, original)

	// Should return the same object when nothing changes
	if result != original {
		t.Error("expected same TRecord when no substitution applied")
	}

	// TypeName should definitely still be there
	if resultRec, ok := result.(*TRecord); ok {
		if resultRec.TypeName != "Point" {
			t.Errorf("TypeName lost: expected 'Point', got '%s'", resultRec.TypeName)
		}
	}
}

// TestTRecordTypeNamePreservedInNestedRecord verifies TypeName preservation
// in nested records (e.g., StarSystem containing Position).
func TestTRecordTypeNamePreservedInNestedRecord(t *testing.T) {
	// Inner record with TypeName
	inner := &TRecord{
		Fields: map[string]Type{
			"x": &TVar{Name: "α1"},
			"y": &TVar{Name: "α2"},
		},
		TypeName: "Position",
	}

	// Outer record containing the inner record
	outer := &TRecord{
		Fields: map[string]Type{
			"name":     TString,
			"position": inner,
		},
		TypeName: "StarSystem",
	}

	// Substitution that affects nested fields
	sub := Substitution{
		"α1": TFloat,
		"α2": TFloat,
	}

	result := ApplySubstitution(sub, outer)

	// Verify outer TypeName preserved
	outerRec, ok := result.(*TRecord)
	if !ok {
		t.Fatalf("expected outer *TRecord, got %T", result)
	}
	if outerRec.TypeName != "StarSystem" {
		t.Errorf("outer TypeName lost: expected 'StarSystem', got '%s'", outerRec.TypeName)
	}

	// Verify inner TypeName preserved
	innerResult, ok := outerRec.Fields["position"].(*TRecord)
	if !ok {
		t.Fatalf("expected inner *TRecord, got %T", outerRec.Fields["position"])
	}
	if innerResult.TypeName != "Position" {
		t.Errorf("inner TypeName lost: expected 'Position', got '%s'", innerResult.TypeName)
	}
}
