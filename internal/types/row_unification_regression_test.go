package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRowUnification_OpenClosedMatrix is a comprehensive regression test for the v0.3.11 bug fix.
// The bug was that unifying (closed, open) and (open, closed) rows swapped the label assignments,
// causing "closed row missing labels: [IO]" errors in stdlib.
//
// This test covers all permutations of open/closed rows with various label sets to ensure
// the fix is symmetric and handles all cases correctly.
func TestRowUnification_OpenClosedMatrix(t *testing.T) {
	tests := []struct {
		name           string
		r1             *Row          // First row
		r2             *Row          // Second row
		expectSuccess  bool          // Should unification succeed?
		expectedSub    map[string]string // Expected tail substitutions (tail_name -> labels_string)
		expectedError  string        // Expected error substring (if expectSuccess=false)
	}{
		// ═══════════════════════════════════════════════════════════════
		// Both closed - must have exact same labels
		// ═══════════════════════════════════════════════════════════════
		{
			name: "closed{IO} ∪ closed{IO} → success",
			r1: &Row{
				Kind:   EffectRow,
				Labels: map[string]Type{"IO": TUnit},
				Tail:   nil,
			},
			r2: &Row{
				Kind:   EffectRow,
				Labels: map[string]Type{"IO": TUnit},
				Tail:   nil,
			},
			expectSuccess: true,
			expectedSub:   map[string]string{}, // No tail variables
		},
		{
			name: "closed{IO} ∪ closed{Net} → failure (different labels)",
			r1: &Row{
				Kind:   EffectRow,
				Labels: map[string]Type{"IO": TUnit},
				Tail:   nil,
			},
			r2: &Row{
				Kind:   EffectRow,
				Labels: map[string]Type{"Net": TUnit},
				Tail:   nil,
			},
			expectSuccess: false,
			expectedError: "incompatible closed rows",
		},
		{
			name: "closed{IO,Net} ∪ closed{IO} → failure (r1 has extra labels)",
			r1: &Row{
				Kind:   EffectRow,
				Labels: map[string]Type{"IO": TUnit, "Net": TUnit},
				Tail:   nil,
			},
			r2: &Row{
				Kind:   EffectRow,
				Labels: map[string]Type{"IO": TUnit},
				Tail:   nil,
			},
			expectSuccess: false,
			expectedError: "incompatible closed rows",
		},

		// ═══════════════════════════════════════════════════════════════
		// Open ∪ Open - create fresh tail for unique labels
		// ═══════════════════════════════════════════════════════════════
		{
			name: "open{} | ε1 ∪ open{} | ε2 → success (both empty, different tails)",
			r1: &Row{
				Kind:   EffectRow,
				Labels: map[string]Type{},
				Tail:   &RowVar{Name: "ε1", Kind: EffectRow},
			},
			r2: &Row{
				Kind:   EffectRow,
				Labels: map[string]Type{},
				Tail:   &RowVar{Name: "ε2", Kind: EffectRow},
			},
			expectSuccess: true,
			expectedSub: map[string]string{
				"ε1": "{}",  // ε1 := {} | ρ (where ρ is fresh)
				"ε2": "{}",  // ε2 := {} | ρ
			},
		},
		{
			name: "open{IO} | ε1 ∪ open{Net} | ε2 → success (ε1 gets Net, ε2 gets IO)",
			r1: &Row{
				Kind:   EffectRow,
				Labels: map[string]Type{"IO": TUnit},
				Tail:   &RowVar{Name: "ε1", Kind: EffectRow},
			},
			r2: &Row{
				Kind:   EffectRow,
				Labels: map[string]Type{"Net": TUnit},
				Tail:   &RowVar{Name: "ε2", Kind: EffectRow},
			},
			expectSuccess: true,
			expectedSub: map[string]string{
				"ε1": "{Net}",  // ε1 := {Net} | ρ (r2's unique labels)
				"ε2": "{IO}",   // ε2 := {IO} | ρ (r1's unique labels)
			},
		},

		// ═══════════════════════════════════════════════════════════════
		// CRITICAL: Open ∪ Closed (the v0.3.11 regression bug)
		// ═══════════════════════════════════════════════════════════════
		{
			name: "open{} | ε1 ∪ closed{IO} → success (ε1 := {IO})",
			r1: &Row{
				Kind:   EffectRow,
				Labels: map[string]Type{},
				Tail:   &RowVar{Name: "ε1", Kind: EffectRow},
			},
			r2: &Row{
				Kind:   EffectRow,
				Labels: map[string]Type{"IO": TUnit},
				Tail:   nil,
			},
			expectSuccess: true,
			expectedSub: map[string]string{
				"ε1": "{IO}",  // ε1 gets r2's unique labels (IO)
			},
		},
		{
			name: "open{Net} | ε1 ∪ closed{IO,Net} → success (ε1 := {IO})",
			r1: &Row{
				Kind:   EffectRow,
				Labels: map[string]Type{"Net": TUnit},
				Tail:   &RowVar{Name: "ε1", Kind: EffectRow},
			},
			r2: &Row{
				Kind:   EffectRow,
				Labels: map[string]Type{"IO": TUnit, "Net": TUnit},
				Tail:   nil,
			},
			expectSuccess: true,
			expectedSub: map[string]string{
				"ε1": "{IO}",  // ε1 gets r2's unique labels (only IO, Net is common)
			},
		},

		// ═══════════════════════════════════════════════════════════════
		// CRITICAL: Closed ∪ Open (symmetric case of the bug)
		// ═══════════════════════════════════════════════════════════════
		{
			name: "closed{IO} ∪ open{} | ε2 → success (ε2 := {IO})",
			r1: &Row{
				Kind:   EffectRow,
				Labels: map[string]Type{"IO": TUnit},
				Tail:   nil,
			},
			r2: &Row{
				Kind:   EffectRow,
				Labels: map[string]Type{},
				Tail:   &RowVar{Name: "ε2", Kind: EffectRow},
			},
			expectSuccess: true,
			expectedSub: map[string]string{
				"ε2": "{IO}",  // ε2 gets r1's unique labels (IO)
			},
		},
		{
			name: "closed{IO,Net} ∪ open{Net} | ε2 → success (ε2 := {IO})",
			r1: &Row{
				Kind:   EffectRow,
				Labels: map[string]Type{"IO": TUnit, "Net": TUnit},
				Tail:   nil,
			},
			r2: &Row{
				Kind:   EffectRow,
				Labels: map[string]Type{"Net": TUnit},
				Tail:   &RowVar{Name: "ε2", Kind: EffectRow},
			},
			expectSuccess: true,
			expectedSub: map[string]string{
				"ε2": "{IO}",  // ε2 gets r1's unique labels (only IO, Net is common)
			},
		},
		{
			name: "closed{IO,Net,FS} ∪ open{} | ε2 → success (ε2 := {IO,Net,FS})",
			r1: &Row{
				Kind:   EffectRow,
				Labels: map[string]Type{"IO": TUnit, "Net": TUnit, "FS": TUnit},
				Tail:   nil,
			},
			r2: &Row{
				Kind:   EffectRow,
				Labels: map[string]Type{},
				Tail:   &RowVar{Name: "ε2", Kind: EffectRow},
			},
			expectSuccess: true,
			expectedSub: map[string]string{
				"ε2": "{FS,IO,Net}",  // ε2 gets all of r1's labels (sorted)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ru := NewRowUnifier()
			sub, err := ru.UnifyRows(tt.r1, tt.r2, make(Substitution))

			if tt.expectSuccess {
				require.NoError(t, err, "Expected successful unification")

				// Verify tail substitutions match expectations
				for tailName, expectedLabels := range tt.expectedSub {
					subRow, ok := sub[tailName]
					require.True(t, ok, "Expected substitution for %s", tailName)

					row, ok := subRow.(*Row)
					require.True(t, ok, "Expected Row type for %s substitution", tailName)

					// Convert labels to sorted string for comparison
					actualLabels := formatLabels(row.Labels)
					assert.Equal(t, expectedLabels, actualLabels,
						"Tail %s should have labels %s, got %s", tailName, expectedLabels, actualLabels)
				}
			} else {
				require.Error(t, err, "Expected unification to fail")
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			}
		})
	}
}

// TestRowUnification_StdlibRegressionCase is the exact scenario that broke in v0.3.10.
// When typechecking stdlib/std/io.ail:
//   export func print(s: string) -> () ! {IO} = _io_print(s)
//
// The application _io_print(s) created a fresh effect row {} | ε1 which needed to unify
// with the builtin's effect row {IO}. The bug assigned ε1 := {} instead of ε1 := {IO}.
func TestRowUnification_StdlibRegressionCase(t *testing.T) {
	// Simulate: _io_print : String -> () ! {IO}
	builtinEffects := &Row{
		Kind:   EffectRow,
		Labels: map[string]Type{"IO": TUnit},
		Tail:   nil, // Closed
	}

	// Simulate: fresh effect row from function application
	applicationEffects := &Row{
		Kind:   EffectRow,
		Labels: map[string]Type{},
		Tail:   &RowVar{Name: "ε1", Kind: EffectRow}, // Open
	}

	// Unify: {IO} with {} | ε1
	ru := NewRowUnifier()
	sub, err := ru.UnifyRows(builtinEffects, applicationEffects, make(Substitution))
	require.NoError(t, err, "stdlib case must unify successfully")

	// CRITICAL: ε1 must be assigned {IO}, not {}
	epsilon1, ok := sub["ε1"]
	require.True(t, ok, "ε1 must be in substitution")

	row, ok := epsilon1.(*Row)
	require.True(t, ok, "ε1 must substitute to a Row")

	// Verify ε1 := {IO}
	assert.Equal(t, 1, len(row.Labels), "ε1 should have exactly 1 label (IO)")
	_, hasIO := row.Labels["IO"]
	assert.True(t, hasIO, "ε1 must contain IO label")
	assert.Nil(t, row.Tail, "ε1 should be closed (tail=nil)")
}

// formatLabels converts a label map to a sorted string representation like "{IO,Net}"
func formatLabels(labels map[string]Type) string {
	if len(labels) == 0 {
		return "{}"
	}

	// Sort labels for deterministic comparison
	var keys []string
	for k := range labels {
		keys = append(keys, k)
	}
	// Simple sort (good enough for test assertions)
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	result := "{"
	for i, k := range keys {
		if i > 0 {
			result += ","
		}
		result += k
	}
	result += "}"
	return result
}
