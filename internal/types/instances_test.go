package types

import (
	"testing"
)

func TestInstanceEnvCoherence(t *testing.T) {
	env := NewInstanceEnv()

	// Add first Num[Int] instance
	numInt := &ClassInstance{
		ClassName: "Num",
		TypeHead:  TInt,
		Dict: Dict{
			"add": "test_num_int_add",
		},
	}

	if err := env.Add(numInt); err != nil {
		t.Fatalf("Failed to add first instance: %v", err)
	}

	// Try to add duplicate Num[Int] instance - should fail
	numInt2 := &ClassInstance{
		ClassName: "Num",
		TypeHead:  TInt,
		Dict: Dict{
			"add": "test_num_int_add_2",
		},
	}

	err := env.Add(numInt2)
	if err == nil {
		t.Fatal("Expected error for overlapping instance, got nil")
	}

	if err.Error() != "overlapping instance: Num[int]" {
		t.Errorf("Wrong error message: %v", err)
	}
}

func TestInstanceLookup(t *testing.T) {
	env := LoadBuiltinInstances()

	tests := []struct {
		name       string
		className  string
		typeHead   Type
		shouldFind bool
	}{
		{"Num[Int]", "Num", TInt, true},
		{"Num[Float]", "Num", TFloat, true},
		{"Eq[Int]", "Eq", TInt, true},
		{"Eq[String]", "Eq", TString, true},
		{"Ord[Int]", "Ord", TInt, true},
		{"Show[Bool]", "Show", TBool, true},
		{"Num[String]", "Num", TString, false}, // No instance
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst, err := env.Lookup(tt.className, tt.typeHead)

			if tt.shouldFind {
				if err != nil {
					t.Errorf("Failed to find instance: %v", err)
				}
				if inst == nil {
					t.Error("Expected instance, got nil")
				}
				// Verify instance has the right class and type
				if inst != nil {
					if inst.ClassName != tt.className {
						t.Errorf("Wrong class name: got %s, want %s", inst.ClassName, tt.className)
					}
					// Note: TypeHead comparison would need proper equality
				}
			} else {
				if err == nil {
					t.Error("Expected error for missing instance, got nil")
				}
				// Verify error message contains helpful hint
				if err != nil {
					if missingErr, ok := err.(*MissingInstanceError); ok {
						if missingErr.Hint == "" {
							t.Error("Missing instance error should include hint")
						}
					}
				}
			}
		})
	}
}

func TestSuperclassProvision(t *testing.T) {
	// Create environment with just Ord[Bytes] to test superclass provision
	env := NewInstanceEnv()

	// Add only Ord[Bytes] (no Eq[Bytes])
	err := env.Add(&ClassInstance{
		ClassName: "Ord",
		TypeHead:  TBytes,
		Dict: Dict{
			"lt":  "builtin_ord_bytes_lt",
			"lte": "builtin_ord_bytes_lte",
			"gt":  "builtin_ord_bytes_gt",
			"gte": "builtin_ord_bytes_gte",
		},
	})
	if err != nil {
		t.Fatalf("Failed to add Ord[Bytes]: %v", err)
	}

	// Verify Ord[Bytes] exists
	ordInst, err := env.Lookup("Ord", TBytes)
	if err != nil {
		t.Fatalf("Failed to find Ord[Bytes]: %v", err)
	}
	if ordInst == nil {
		t.Fatal("Ord[Bytes] should exist")
	}

	// Now verify Eq[Bytes] is provided via Ord[Bytes] superclass provision
	eqInst, err := env.Lookup("Eq", TBytes)
	if err != nil {
		t.Fatalf("Failed to get Eq[Bytes] via Ord[Bytes]: %v", err)
	}

	if eqInst.ClassName != "Eq" {
		t.Errorf("Expected Eq instance, got %s", eqInst.ClassName)
	}

	// Check that the derived Eq has the expected methods
	if eqInst.Dict["eq"] == "" {
		t.Error("Derived Eq should have 'eq' method")
	}
	if eqInst.Dict["neq"] == "" {
		t.Error("Derived Eq should have 'neq' method")
	}

	// The actual method implementation names should indicate derivation
	// Note: NormalizeTypeName converts "bytes" to "Bytes"
	expectedEq := "derived_eq_from_ord_Bytes"
	if eqInst.Dict["eq"] != expectedEq {
		t.Errorf("Expected derived eq method name %s, got %s", expectedEq, eqInst.Dict["eq"])
	}
}

func TestBuiltinInstances(t *testing.T) {
	env := LoadBuiltinInstances()

	// Test that all expected instances are present
	expectedInstances := []struct {
		className string
		typeHead  Type
		methods   []string
	}{
		{"Num", TInt, []string{"add", "sub", "mul", "div"}},
		{"Num", TFloat, []string{"add", "sub", "mul", "div"}},
		{"Eq", TInt, []string{"eq", "neq"}},
		{"Eq", TFloat, []string{"eq", "neq"}},
		{"Eq", TString, []string{"eq", "neq"}},
		{"Eq", TBool, []string{"eq", "neq"}},
		{"Ord", TInt, []string{"lt", "lte", "gt", "gte"}},
		{"Ord", TFloat, []string{"lt", "lte", "gt", "gte"}},
		{"Ord", TString, []string{"lt", "lte", "gt", "gte"}},
		{"Show", TInt, []string{"show"}},
		{"Show", TFloat, []string{"show"}},
		{"Show", TString, []string{"show"}},
		{"Show", TBool, []string{"show"}},
	}

	for _, expected := range expectedInstances {
		name := expected.className + "[" + expected.typeHead.String() + "]"
		t.Run(name, func(t *testing.T) {
			inst, err := env.Lookup(expected.className, expected.typeHead)
			if err != nil {
				t.Fatalf("Failed to find %s: %v", name, err)
			}

			// Check all expected methods are present
			for _, method := range expected.methods {
				if inst.Dict[method] == "" {
					t.Errorf("%s missing method '%s'", name, method)
				}
			}
		})
	}
}

func TestDefaulting(t *testing.T) {
	env := LoadBuiltinInstances()

	// Test default for Num
	if def := env.DefaultFor("Num"); def != TInt {
		t.Errorf("Default for Num should be Int, got %v", def)
	}

	// Test default for Fractional
	if def := env.DefaultFor("Fractional"); def != TFloat {
		t.Errorf("Default for Fractional should be Float, got %v", def)
	}

	// Test no default
	if def := env.DefaultFor("Unknown"); def != nil {
		t.Errorf("Default for Unknown should be nil, got %v", def)
	}
}

func TestNoAmbientInstances(t *testing.T) {
	// Create an empty environment (no preloaded instances)
	env := NewInstanceEnv()

	// Should not find any instances without explicit loading
	_, err := env.Lookup("Num", TInt)
	if err == nil {
		t.Error("Empty environment should not have Num[Int]")
	}

	// Error should suggest importing
	if missingErr, ok := err.(*MissingInstanceError); ok {
		if missingErr.Hint != "Import std/prelude or define instance" {
			t.Errorf("Wrong hint: %s", missingErr.Hint)
		}
	}
}
