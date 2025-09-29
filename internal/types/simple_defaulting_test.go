package types

import (
	"testing"
)

// TestDefaulting_SimpleConstraints tests the core defaulting algorithm
func TestDefaulting_SimpleConstraints(t *testing.T) {
	tc := NewCoreTypeChecker()
	tc.instanceEnv = LoadBuiltinInstances()

	// Create a simple monotype and constraint scenario
	// Scenario: we have a type variable α that has a Num constraint
	// but doesn't appear in the monotype (making it ambiguous)

	tvar := &TVar{Name: "α1"}
	monotype := TInt // The monotype doesn't contain α1

	constraints := []ClassConstraint{
		{
			Class:  "Num",
			Type:   tvar,
			Path:   []string{"test"},
			NodeID: 1,
		},
	}

	// Apply defaulting
	sub, _, defaultedConstraints, err := tc.defaultAmbiguities(monotype, constraints)
	if err != nil {
		t.Fatalf("Defaulting failed: %v", err)
	}

	// Check that α1 was defaulted to Int
	if len(sub) != 1 {
		t.Errorf("Expected 1 substitution, got %d", len(sub))
	}

	if defaultedTy, ok := sub["α1"]; !ok {
		t.Error("Expected α1 to be defaulted")
	} else if defaultedTy.String() != "int" {
		t.Errorf("Expected α1 to default to int, got %s", defaultedTy.String())
	}

	// Check that the constraint was resolved
	if len(defaultedConstraints) != 1 {
		t.Errorf("Expected 1 constraint after defaulting, got %d", len(defaultedConstraints))
	}

	// The constraint should now have a ground type
	if !isGround(defaultedConstraints[0].Type) {
		t.Errorf("Constraint type should be ground after defaulting, got %s", defaultedConstraints[0].Type)
	}

	// Check that we can look up the instance
	_, err = tc.instanceEnv.Lookup("Num", defaultedConstraints[0].Type)
	if err != nil {
		t.Errorf("Should be able to look up Num[%s] instance: %v", defaultedConstraints[0].Type, err)
	}
}

// TestDefaulting_NoAmbiguousVars tests when there are no ambiguous variables
func TestDefaulting_NoAmbiguousVars(t *testing.T) {
	tc := NewCoreTypeChecker()
	tc.instanceEnv = LoadBuiltinInstances()

	// Scenario: type variable appears in both constraint and monotype (not ambiguous)
	tvar := &TVar{Name: "α1"}
	monotype := tvar // The type variable appears in the monotype

	constraints := []ClassConstraint{
		{
			Class:  "Num",
			Type:   tvar,
			Path:   []string{"test"},
			NodeID: 1,
		},
	}

	// Apply defaulting
	sub, defaultedType, defaultedConstraints, err := tc.defaultAmbiguities(monotype, constraints)
	if err != nil {
		t.Fatalf("Defaulting failed: %v", err)
	}

	// Should not default anything since the variable is not ambiguous
	if len(sub) != 0 {
		t.Errorf("Expected no substitutions for non-ambiguous variable, got %d", len(sub))
	}

	// Type and constraints should be unchanged
	if defaultedType != monotype {
		t.Error("Monotype should be unchanged when no defaulting occurs")
	}

	if len(defaultedConstraints) != 1 || defaultedConstraints[0].Type != tvar {
		t.Error("Constraints should be unchanged when no defaulting occurs")
	}
}

// TestDefaulting_MixedConstraintsSimple tests defaulting with neutral constraints
func TestDefaulting_MixedConstraintsSimple(t *testing.T) {
	tc := NewCoreTypeChecker()
	tc.instanceEnv = LoadBuiltinInstances()

	// Scenario: ambiguous type variable with both Num and Ord constraints
	// α1 appears in constraints but NOT in the monotype (making it ambiguous)
	// Since Ord is neutral, this should default based on Num alone
	tvar := &TVar{Name: "α1"}
	monotype := TInt // Monotype doesn't contain α1, so α1 is ambiguous

	constraints := []ClassConstraint{
		{
			Class:  "Num",
			Type:   tvar,
			Path:   []string{"test"},
			NodeID: 1,
		},
		{
			Class:  "Ord",
			Type:   tvar,
			Path:   []string{"test"},
			NodeID: 2,
		},
	}

	// Apply defaulting - should succeed since Ord is neutral
	subst, resultType, remainingConstraints, err := tc.defaultAmbiguities(monotype, constraints)
	if err != nil {
		t.Errorf("Expected defaulting to succeed with Num+Ord (Ord is neutral): %v", err)
		return
	}

	// Check that α1 was defaulted to int
	if defaultType, ok := subst["α1"]; !ok || !defaultType.Equals(TInt) {
		t.Errorf("Expected α1 to default to int, got %v", defaultType)
	}

	// Result type should still be int (monotype unchanged)
	if !resultType.Equals(TInt) {
		t.Errorf("Expected result type to be int, got %v", resultType)
	}

	// Constraints should now be grounded after substitution
	for _, c := range remainingConstraints {
		// Apply substitution to check if constraint is grounded
		substType := ApplySubstitution(subst, c.Type)
		if !isGround(substType) {
			t.Errorf("Expected constraint to be grounded after defaulting, got %v", substType)
		}
	}
}
