package types

import (
	"os"
	"testing"
)

// TestAutoImportInstancesLoaded tests that instances are auto-loaded by default
func TestAutoImportInstancesLoaded(t *testing.T) {
	tc := NewCoreTypeChecker()

	// Check that Ord[Int] instance exists
	_, err := tc.instanceEnv.Lookup("Ord", TInt)
	if err != nil {
		t.Fatalf("Expected Ord[Int] to be auto-loaded, got error: %v", err)
	}

	// Check that Eq[Int] instance exists
	_, err = tc.instanceEnv.Lookup("Eq", TInt)
	if err != nil {
		t.Fatalf("Expected Eq[Int] to be auto-loaded, got error: %v", err)
	}

	// Check that Num[Int] instance exists
	_, err = tc.instanceEnv.Lookup("Num", TInt)
	if err != nil {
		t.Fatalf("Expected Num[Int] to be auto-loaded, got error: %v", err)
	}

	// Check that Show[Int] instance exists
	_, err = tc.instanceEnv.Lookup("Show", TInt)
	if err != nil {
		t.Fatalf("Expected Show[Int] to be auto-loaded, got error: %v", err)
	}
}

// TestNoPreludeFlag tests that AILANG_NO_PRELUDE=1 disables auto-import
func TestNoPreludeFlag(t *testing.T) {
	os.Setenv("AILANG_NO_PRELUDE", "1")
	defer os.Unsetenv("AILANG_NO_PRELUDE")

	tc := NewCoreTypeChecker()

	// Check that Ord[Int] instance does NOT exist
	_, err := tc.instanceEnv.Lookup("Ord", TInt)
	if err == nil {
		t.Fatal("Expected Ord[Int] to NOT be loaded when AILANG_NO_PRELUDE=1")
	}

	// Should mention missing instance
	if _, ok := err.(*MissingInstanceError); !ok {
		t.Fatalf("Expected MissingInstanceError, got: %T", err)
	}
}

// TestAutoImportWithVariables tests that auto-import works when using variables
// (regression test for TVar2 not being recognized as non-ground)
func TestAutoImportWithVariables(t *testing.T) {
	// This test verifies that the fix for isGround() to recognize TVar2
	// correctly allows type variables to go through defaulting before
	// instance lookup occurs
	tc := NewCoreTypeChecker()

	// Create a type variable (simulating inferred type from `let x = 10`)
	typeVar := &TVar2{Name: "α4", Kind: Star}

	// Before defaulting, looking up Ord[α4] should fail gracefully
	// (it should be recognized as non-ground and skipped)
	// After our fix, isGround(TVar2) returns false, so this constraint
	// won't be resolved until after defaulting

	// Verify that TVar2 is recognized as non-ground
	if isGround(typeVar) {
		t.Fatal("TVar2 should NOT be ground - this would cause premature instance lookup")
	}

	// After defaulting (simulated by substitution to int), lookup should succeed
	_, err := tc.instanceEnv.Lookup("Ord", TInt)
	if err != nil {
		t.Fatalf("Expected Ord[Int] to exist after defaulting, got error: %v", err)
	}
}
