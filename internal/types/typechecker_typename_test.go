package types

import (
	"testing"
)

// TestNestedRecordTypeNamePropagation verifies that TypeName propagates during
// unification of nested record types.
//
// This is the M-TYPENAME-NESTED-PROPAGATION test case.
//
// Scenario:
//
//	type SystemPos = {x: float, y: float, z: float}
//	type StarSystem = {name: string, position: SystemPos}
//
// When a record literal {name: "Sol", position: {x: 0, y: 0, z: 0}} is unified
// with StarSystem, the nested {x, y, z} should get TypeName: "SystemPos".
func TestNestedRecordTypeNamePropagation(t *testing.T) {
	// Create type alias for SystemPos
	systemPosType := &TRecord{
		Fields: map[string]Type{
			"x": TFloat,
			"y": TFloat,
			"z": TFloat,
		},
		TypeName: "SystemPos",
	}

	// Create type alias for StarSystem with nested SystemPos
	starSystemType := &TRecord{
		Fields: map[string]Type{
			"name":     TString,
			"position": systemPosType, // Reference to SystemPos type WITH TypeName
		},
		TypeName: "StarSystem",
	}

	// Create alias environment for the unifier
	aliases := map[string]Type{
		"SystemPos":  systemPosType,
		"StarSystem": starSystemType,
	}

	// Create the record literal types (as they would be created by inferRecord)
	// Inner record literal {x: 0.0, y: 0.0, z: 0.0} - NO TypeName initially
	innerLiteral := &TRecord{
		Fields: map[string]Type{
			"x": TFloat,
			"y": TFloat,
			"z": TFloat,
		},
		TypeName: "", // Record literal has no TypeName initially
	}

	// Outer record literal {name: "Sol", position: innerLiteral} - NO TypeName initially
	outerLiteral := &TRecord{
		Fields: map[string]Type{
			"name":     TString,
			"position": innerLiteral, // Points to inner literal (same object)
		},
		TypeName: "", // Record literal has no TypeName initially
	}

	// Create unifier with aliases
	u := NewUnifierWithAliases(aliases)

	// Unify the outer record literal with the StarSystem type alias
	// This simulates what happens when a function return type is unified with the literal
	_, err := u.Unify(outerLiteral, starSystemType, make(Substitution))
	if err != nil {
		t.Fatalf("Unification failed: %v", err)
	}

	// Check that outer record got TypeName
	if outerLiteral.TypeName != "StarSystem" {
		t.Errorf("Outer record TypeName not propagated: expected 'StarSystem', got '%s'",
			outerLiteral.TypeName)
	} else {
		t.Logf("SUCCESS: Outer record has TypeName = '%s'", outerLiteral.TypeName)
	}

	// THE KEY TEST: Does the inner record have TypeName set?
	// This tests whether field-level unification propagates TypeName to nested records
	if innerLiteral.TypeName == "" {
		t.Errorf("FAIL: Nested record has empty TypeName - this is the bug we're fixing!")
		t.Logf("  Expected: TypeName = \"SystemPos\"")
		t.Logf("  Got:      TypeName = \"\"")
		t.Logf("  This means field unification doesn't propagate TypeName to nested records")
	} else if innerLiteral.TypeName != "SystemPos" {
		t.Errorf("Wrong TypeName for nested record: expected 'SystemPos', got '%s'",
			innerLiteral.TypeName)
	} else {
		t.Logf("SUCCESS: Nested record has TypeName = '%s'", innerLiteral.TypeName)
	}
}

// TestUnificationPropagatesTypeName verifies TypeName propagation during direct unification.
// This should pass (v0.5.10 fix).
func TestUnificationPropagatesTypeName(t *testing.T) {
	// Record literal (no TypeName)
	recordLiteral := &TRecord{
		Fields: map[string]Type{
			"x": TFloat,
			"y": TFloat,
		},
		TypeName: "",
	}

	// Type alias expansion (has TypeName)
	typeAlias := &TRecord{
		Fields: map[string]Type{
			"x": TFloat,
			"y": TFloat,
		},
		TypeName: "Point",
	}

	// Unify
	u := NewUnifier()
	_, err := u.Unify(recordLiteral, typeAlias, make(Substitution))
	if err != nil {
		t.Fatalf("Unification failed: %v", err)
	}

	// Check that TypeName was propagated to record literal
	if recordLiteral.TypeName != "Point" {
		t.Errorf("TypeName not propagated during unification: expected 'Point', got '%s'",
			recordLiteral.TypeName)
	} else {
		t.Logf("SUCCESS: Record literal has TypeName = '%s'", recordLiteral.TypeName)
	}
}

// TestTypeNamePreservedThroughSubstitution verifies the v0.5.10 fix.
func TestTypeNamePreservedThroughSubstitution(t *testing.T) {
	rec := &TRecord{
		Fields: map[string]Type{
			"x": &TVar{Name: "α1"},
			"y": TInt,
		},
		TypeName: "Coord",
	}

	sub := Substitution{
		"α1": TFloat,
	}

	result := ApplySubstitution(sub, rec)

	resultRec, ok := result.(*TRecord)
	if !ok {
		t.Fatalf("Expected TRecord, got %T", result)
	}

	if resultRec.TypeName != "Coord" {
		t.Errorf("TypeName not preserved: expected 'Coord', got '%s'", resultRec.TypeName)
	} else {
		t.Logf("SUCCESS: TypeName preserved through substitution")
	}
}

// TestNestedRecordFieldUnification tests the specific scenario where
// unification of record fields should propagate TypeName to nested records.
func TestNestedRecordFieldUnification(t *testing.T) {
	// Expected field type with TypeName (from type alias)
	expectedFieldType := &TRecord{
		Fields: map[string]Type{
			"a": TInt,
			"b": TInt,
		},
		TypeName: "Inner",
	}

	// Actual field type from record literal (no TypeName)
	actualFieldType := &TRecord{
		Fields: map[string]Type{
			"a": TInt,
			"b": TInt,
		},
		TypeName: "",
	}

	// Unify the field types directly
	u := NewUnifier()
	_, err := u.Unify(actualFieldType, expectedFieldType, make(Substitution))
	if err != nil {
		t.Fatalf("Field unification failed: %v", err)
	}

	// TypeName should propagate to the actual field type
	if actualFieldType.TypeName != "Inner" {
		t.Errorf("TypeName not propagated to field: expected 'Inner', got '%s'",
			actualFieldType.TypeName)
	} else {
		t.Logf("SUCCESS: Field TypeName = '%s'", actualFieldType.TypeName)
	}
}
