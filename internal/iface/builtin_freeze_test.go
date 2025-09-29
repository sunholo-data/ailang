package iface_test

import (
	"testing"
	
	"github.com/sunholo/ailang/internal/iface"
)

// TestBuiltinInterfaceStability ensures the builtin interface doesn't change accidentally
func TestBuiltinInterfaceStability(t *testing.T) {
	// This is the golden digest - if builtins change, this test will fail
	// Update this ONLY when intentionally changing the builtin interface
	const goldenDigest = "c968028800ca5e5b1992c6ded1a0dc147d5d4789c6f73bea73914225a5b2226a"
	
	frozen := iface.FrozenBuiltinInterface()
	
	// For initial run, uncomment this to get the actual digest
	// t.Logf("Current builtin digest: %s", frozen.Digest)
	
	if frozen.Digest != goldenDigest {
		t.Errorf("Builtin interface has changed!\nExpected digest: %s\nActual digest: %s\n"+
			"If this is intentional, update the golden digest in this test",
			goldenDigest, frozen.Digest)
			
		// Dump the interface for inspection
		dump, _ := iface.DumpBuiltinInterface()
		t.Logf("Current interface:\n%s", dump)
	}
}

// TestBuiltinValidation verifies builtin validation works correctly
func TestBuiltinValidation(t *testing.T) {
	tests := []struct {
		name    string
		builtin string
		valid   bool
	}{
		{"valid add_Int", "add_Int", true},
		{"valid eq_String", "eq_String", true},
		{"valid show_Bool", "show_Bool", true},
		{"invalid foo", "foo", false},
		{"invalid add", "add", false}, // Must be type-specific
		{"invalid Add_Int", "Add_Int", false}, // Case sensitive
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := iface.ValidateBuiltin(tt.builtin)
			if tt.valid && err != nil {
				t.Errorf("expected %s to be valid, got error: %v", tt.builtin, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("expected %s to be invalid, got no error", tt.builtin)
			}
		})
	}
}

// TestBuiltinTypes verifies type signatures are correct
func TestBuiltinTypes(t *testing.T) {
	tests := []struct {
		builtin  string
		wantType string
		wantArity int
	}{
		{"add_Int", "Int -> Int -> Int", 2},
		{"neg_Float", "Float -> Float", 1},
		{"eq_String", "String -> String -> Bool", 2},
		{"show_Bool", "Bool -> String", 1},
		{"print", "String -> ()", 1},
	}
	
	for _, tt := range tests {
		t.Run(tt.builtin, func(t *testing.T) {
			gotType, err := iface.GetBuiltinType(tt.builtin)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotType != tt.wantType {
				t.Errorf("type mismatch for %s: got %s, want %s", tt.builtin, gotType, tt.wantType)
			}
			
			gotArity, err := iface.GetBuiltinArity(tt.builtin)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotArity != tt.wantArity {
				t.Errorf("arity mismatch for %s: got %d, want %d", tt.builtin, gotArity, tt.wantArity)
			}
		})
	}
}

// TestBuiltinCategories verifies builtins are properly categorized
func TestBuiltinCategories(t *testing.T) {
	frozen := iface.FrozenBuiltinInterface()
	
	categories := make(map[string][]string)
	for name, export := range frozen.Exports {
		categories[export.Category] = append(categories[export.Category], name)
	}
	
	// Verify we have expected categories
	expectedCategories := []string{"arithmetic", "comparison", "string", "logical", "show", "io"}
	for _, cat := range expectedCategories {
		if _, ok := categories[cat]; !ok {
			t.Errorf("missing expected category: %s", cat)
		}
	}
	
	// Log category contents for inspection
	for cat, builtins := range categories {
		t.Logf("Category %s: %v", cat, builtins)
	}
}