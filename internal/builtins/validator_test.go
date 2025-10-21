package builtins

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ensureBuiltinsRegistered ensures test builtins are in the registry
// This is needed because other tests may clear the registry
func ensureBuiltinsRegistered() {
	// Check if already registered
	if len(specRegistry) >= 2 {
		return
	}

	// Clear and reset
	specRegistry = make(map[string]*BuiltinSpec)
	frozen = false

	// Re-register our test builtins
	registerStringLen()
	registerNetHTTPRequest()
}

func TestValidateBuiltins(t *testing.T) {
	ensureBuiltinsRegistered()

	// Run validation on current registry
	errors := ValidateBuiltins()

	// Should have no errors (all migrated builtins are valid)
	if len(errors) > 0 {
		t.Logf("Validation errors found:")
		for _, err := range errors {
			t.Logf("  - %s: %s", err.Builtin, err.Message)
			t.Logf("    Fix: %s", err.Fix)
		}
	}

	// For now, allow warnings but no errors
	for _, err := range errors {
		if err.Severity == "error" {
			t.Errorf("Validation error in %s: %s", err.Builtin, err.Message)
		}
	}
}

func TestGetRegistryStats(t *testing.T) {
	ensureBuiltinsRegistered()

	// Use AllSpecs() which properly accesses the registry
	specs := AllSpecs()
	assert.GreaterOrEqual(t, len(specs), 2, "Should have at least 2 registered builtins")

	stats := GetRegistryStats()

	// Should have at least our 2 migrated builtins
	assert.GreaterOrEqual(t, stats.Total, 2)

	// Should have at least 1 pure and 1 effect
	assert.GreaterOrEqual(t, stats.Pure, 1)   // _str_len
	assert.GreaterOrEqual(t, stats.Effect, 1) // _net_httpRequest

	// Check module grouping
	assert.NotEmpty(t, stats.ByModule)
	assert.Contains(t, stats.ByModule, "std/string")
	assert.Contains(t, stats.ByModule, "std/net")

	// Check effect grouping
	assert.NotEmpty(t, stats.ByEffect)
	assert.Contains(t, stats.ByEffect, "Pure")
	assert.Contains(t, stats.ByEffect, "Net")
}

func TestGroupByEffect(t *testing.T) {
	ensureBuiltinsRegistered()

	grouped := GroupByEffect()

	assert.NotEmpty(t, grouped)
	assert.Contains(t, grouped, "Pure")
	assert.Contains(t, grouped, "Net")

	// Check that _str_len is in Pure
	assert.Contains(t, grouped["Pure"], "_str_len")

	// Check that _net_httpRequest is in Net
	assert.Contains(t, grouped["Net"], "_net_httpRequest")

	// Lists should be sorted
	for _, names := range grouped {
		assert.True(t, isSorted(names), "names should be sorted")
	}
}

func TestGroupByModule(t *testing.T) {
	ensureBuiltinsRegistered()

	grouped := GroupByModule()

	assert.NotEmpty(t, grouped)
	assert.Contains(t, grouped, "std/string")
	assert.Contains(t, grouped, "std/net")

	// Check specific builtins
	assert.Contains(t, grouped["std/string"], "_str_len")
	assert.Contains(t, grouped["std/net"], "_net_httpRequest")

	// Lists should be sorted
	for _, names := range grouped {
		assert.True(t, isSorted(names), "names should be sorted")
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"_str_len", "Str_len"},
		{"_net_httpRequest", "Net_httpRequest"},
		{"", ""},
		{"x", "X"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toCamelCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper: check if string slice is sorted
func isSorted(strs []string) bool {
	for i := 1; i < len(strs); i++ {
		if strs[i-1] > strs[i] {
			return false
		}
	}
	return true
}
