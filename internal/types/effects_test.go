package types

import (
	"sort"
	"strings"
	"testing"
)

func TestElaborateEffectRow(t *testing.T) {
	tests := []struct {
		name           string
		effectNames    []string
		expectNil      bool
		expectedLabels []string
		expectError    bool
	}{
		{
			name:           "empty effects returns nil (purity sentinel)",
			effectNames:    []string{},
			expectNil:      true,
			expectedLabels: nil,
			expectError:    false,
		},
		{
			name:           "single effect",
			effectNames:    []string{"IO"},
			expectNil:      false,
			expectedLabels: []string{"IO"},
			expectError:    false,
		},
		{
			name:           "multiple effects sorted",
			effectNames:    []string{"Net", "IO", "FS"},
			expectNil:      false,
			expectedLabels: []string{"FS", "IO", "Net"}, // Alphabetically sorted
			expectError:    false,
		},
		{
			name:           "duplicates deduplicated",
			effectNames:    []string{"IO", "FS", "IO"},
			expectNil:      false,
			expectedLabels: []string{"FS", "IO"},
			expectError:    false,
		},
		{
			name:        "unknown effect returns error",
			effectNames: []string{"UnknownEffect"},
			expectError: true,
		},
		{
			name:           "all standard effects",
			effectNames:    []string{"IO", "FS", "Net", "Clock", "Rand", "DB", "Trace", "Async"},
			expectNil:      false,
			expectedLabels: []string{"Async", "Clock", "DB", "FS", "IO", "Net", "Rand", "Trace"},
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row, err := ElaborateEffectRow(tt.effectNames)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error, got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectNil {
				if row != nil {
					t.Errorf("expected nil row (purity), got %v", row)
				}
				return
			}

			if row == nil {
				t.Fatal("expected non-nil row, got nil")
			}

			// Check kind
			if !row.Kind.Equals(EffectRow) {
				t.Errorf("expected EffectRow kind, got %v", row.Kind)
			}

			// Check labels are sorted
			actualLabels := make([]string, 0, len(row.Labels))
			for k := range row.Labels {
				actualLabels = append(actualLabels, k)
			}
			sort.Strings(actualLabels)

			if len(actualLabels) != len(tt.expectedLabels) {
				t.Errorf("expected %d labels, got %d", len(tt.expectedLabels), len(actualLabels))
			}

			for i, expected := range tt.expectedLabels {
				if i >= len(actualLabels) {
					t.Errorf("missing label %s", expected)
					continue
				}
				if actualLabels[i] != expected {
					t.Errorf("label %d: expected %s, got %s", i, expected, actualLabels[i])
				}
			}

			// Check tail is nil (closed row)
			if row.Tail != nil {
				t.Errorf("expected closed row (Tail=nil), got Tail=%v", row.Tail)
			}
		})
	}
}

func TestUnionEffectRows(t *testing.T) {
	tests := []struct {
		name           string
		a              []string
		b              []string
		expectedLabels []string
	}{
		{
			name:           "nil ∪ nil = nil",
			a:              nil,
			b:              nil,
			expectedLabels: nil,
		},
		{
			name:           "nil ∪ {IO} = {IO}",
			a:              nil,
			b:              []string{"IO"},
			expectedLabels: []string{"IO"},
		},
		{
			name:           "{FS} ∪ nil = {FS}",
			a:              []string{"FS"},
			b:              nil,
			expectedLabels: []string{"FS"},
		},
		{
			name:           "{IO} ∪ {FS} = {FS, IO} (sorted)",
			a:              []string{"IO"},
			b:              []string{"FS"},
			expectedLabels: []string{"FS", "IO"},
		},
		{
			name:           "{IO, FS} ∪ {Net, Clock} = {Clock, FS, IO, Net}",
			a:              []string{"IO", "FS"},
			b:              []string{"Net", "Clock"},
			expectedLabels: []string{"Clock", "FS", "IO", "Net"},
		},
		{
			name:           "{IO, FS} ∪ {IO, Net} = {FS, IO, Net}",
			a:              []string{"IO", "FS"},
			b:              []string{"IO", "Net"},
			expectedLabels: []string{"FS", "IO", "Net"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rowA, rowB *Row
			var err error

			if tt.a != nil {
				rowA, err = ElaborateEffectRow(tt.a)
				if err != nil {
					t.Fatalf("failed to create row A: %v", err)
				}
			}

			if tt.b != nil {
				rowB, err = ElaborateEffectRow(tt.b)
				if err != nil {
					t.Fatalf("failed to create row B: %v", err)
				}
			}

			result := UnionEffectRows(rowA, rowB)

			if tt.expectedLabels == nil {
				if result != nil {
					t.Errorf("expected nil result, got %v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result, got nil")
			}

			// Check labels
			actualLabels := make([]string, 0, len(result.Labels))
			for k := range result.Labels {
				actualLabels = append(actualLabels, k)
			}
			sort.Strings(actualLabels)

			if len(actualLabels) != len(tt.expectedLabels) {
				t.Errorf("expected %d labels, got %d", len(tt.expectedLabels), len(actualLabels))
			}

			for i, expected := range tt.expectedLabels {
				if i >= len(actualLabels) {
					t.Errorf("missing label %s", expected)
					continue
				}
				if actualLabels[i] != expected {
					t.Errorf("label %d: expected %s, got %s", i, expected, actualLabels[i])
				}
			}
		})
	}
}

func TestSubsumeEffectRows(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected bool
	}{
		{
			name:     "nil ⊆ nil",
			a:        nil,
			b:        nil,
			expected: true,
		},
		{
			name:     "nil ⊆ {IO}",
			a:        nil,
			b:        []string{"IO"},
			expected: true,
		},
		{
			name:     "{IO} ⊈ nil",
			a:        []string{"IO"},
			b:        nil,
			expected: false,
		},
		{
			name:     "{IO} ⊆ {IO, FS}",
			a:        []string{"IO"},
			b:        []string{"IO", "FS"},
			expected: true,
		},
		{
			name:     "{IO, FS} ⊈ {IO}",
			a:        []string{"IO", "FS"},
			b:        []string{"IO"},
			expected: false,
		},
		{
			name:     "{IO, FS} ⊆ {IO, FS, Net}",
			a:        []string{"IO", "FS"},
			b:        []string{"IO", "FS", "Net"},
			expected: true,
		},
		{
			name:     "{IO, FS} ⊆ {IO, FS} (reflexive)",
			a:        []string{"IO", "FS"},
			b:        []string{"IO", "FS"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rowA, rowB *Row
			var err error

			if tt.a != nil {
				rowA, err = ElaborateEffectRow(tt.a)
				if err != nil {
					t.Fatalf("failed to create row A: %v", err)
				}
			}

			if tt.b != nil {
				rowB, err = ElaborateEffectRow(tt.b)
				if err != nil {
					t.Fatalf("failed to create row B: %v", err)
				}
			}

			result := SubsumeEffectRows(rowA, rowB)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEffectRowDifference(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected []string
	}{
		{
			name:     "nil \\ nil = ∅",
			a:        nil,
			b:        nil,
			expected: nil,
		},
		{
			name:     "{IO, FS} \\ nil = {FS, IO}",
			a:        []string{"IO", "FS"},
			b:        nil,
			expected: []string{"FS", "IO"}, // Sorted
		},
		{
			name:     "{IO, FS} \\ {IO} = {FS}",
			a:        []string{"IO", "FS"},
			b:        []string{"IO"},
			expected: []string{"FS"},
		},
		{
			name:     "{IO, FS, Net} \\ {FS} = {IO, Net}",
			a:        []string{"IO", "FS", "Net"},
			b:        []string{"FS"},
			expected: []string{"IO", "Net"},
		},
		{
			name:     "{IO} \\ {IO, FS} = ∅",
			a:        []string{"IO"},
			b:        []string{"IO", "FS"},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rowA, rowB *Row
			var err error

			if tt.a != nil {
				rowA, err = ElaborateEffectRow(tt.a)
				if err != nil {
					t.Fatalf("failed to create row A: %v", err)
				}
			}

			if tt.b != nil {
				rowB, err = ElaborateEffectRow(tt.b)
				if err != nil {
					t.Fatalf("failed to create row B: %v", err)
				}
			}

			result := EffectRowDifference(rowA, rowB)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d effects, got %d: %v", len(tt.expected), len(result), result)
			}

			for i, expected := range tt.expected {
				if i >= len(result) {
					t.Errorf("missing effect %s", expected)
					continue
				}
				if result[i] != expected {
					t.Errorf("effect %d: expected %s, got %s", i, expected, result[i])
				}
			}
		})
	}
}

func TestFormatEffectRow(t *testing.T) {
	tests := []struct {
		name     string
		effects  []string
		expected string
	}{
		{
			name:     "nil formats as empty string",
			effects:  nil,
			expected: "",
		},
		{
			name:     "empty formats as empty string",
			effects:  []string{},
			expected: "",
		},
		{
			name:     "single effect",
			effects:  []string{"IO"},
			expected: "! {IO}",
		},
		{
			name:     "multiple effects sorted",
			effects:  []string{"Net", "IO", "FS"},
			expected: "! {FS, IO, Net}",
		},
		{
			// M-EFFECT-REFINEMENT Phase 1: Rand has a registered default mode
			// (mode=os) so bare !{Rand} desugars to !{Rand[mode=os]}. Other
			// effects (IO, FS, Net, Clock, DB, Trace, Async) have no default
			// registered and stay bare (back-compat).
			name:     "all effects",
			effects:  []string{"IO", "FS", "Net", "Clock", "Rand", "DB", "Trace", "Async"},
			expected: "! {Async, Clock, DB, FS, IO, Net, Rand[mode=os], Trace}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var row *Row
			var err error

			if len(tt.effects) > 0 {
				row, err = ElaborateEffectRow(tt.effects)
				if err != nil {
					t.Fatalf("failed to create row: %v", err)
				}
			}

			result := FormatEffectRow(row)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestSubsumeEffectRows_NoHierarchy is a regression test ensuring that no effect
// implicitly absorbs or subsumes another. For example, declaring FS must NOT
// satisfy an Env requirement. Each effect is an independent label — subsumption
// is strict set inclusion only.
//
// Background: A local gcp_auth.ail declared ! {FS} while using getEnvOr (which
// requires Env). The effect checker should reject this. This test guards against
// any future introduction of an effect hierarchy.
func TestSubsumeEffectRows_NoHierarchy(t *testing.T) {
	// Every canonical effect that should be independent
	independentEffects := []string{
		"IO", "FS", "Net", "Clock", "Rand", "DB", "Trace", "Async", "Env",
		"AI", "SharedMem", "SharedIndex", "Stream", "Process",
		// M-COG-RUNTIME (v0.21.x): Cognitive OS effect labels
		"DOM", "Msg",
	}

	for _, declared := range independentEffects {
		for _, required := range independentEffects {
			if declared == required {
				continue // Same effect trivially subsumes itself
			}

			t.Run(required+"_not_subsumed_by_"+declared, func(t *testing.T) {
				reqRow, err := ElaborateEffectRow([]string{required})
				if err != nil {
					t.Fatalf("failed to create required row {%s}: %v", required, err)
				}

				declRow, err := ElaborateEffectRow([]string{declared})
				if err != nil {
					t.Fatalf("failed to create declared row {%s}: %v", declared, err)
				}

				if SubsumeEffectRows(reqRow, declRow) {
					t.Errorf("{%s} should NOT be subsumed by {%s} — effects must be independent", required, declared)
				}
			})
		}
	}
}

// TestIsKnownEffect_CognitiveOS pins the M-COG-RUNTIME (v0.21.x) effect labels
// that the Cognitive OS substrate depends on. These labels are locked across
// three sibling design docs (M-COG-RUNTIME / M-COG-MEMORY / M-COG-MESH) — do
// not rename. Trace already shipped with M-WASM-TRACE (v0.11.1); DOM and Msg
// are net-new.
func TestIsKnownEffect_CognitiveOS(t *testing.T) {
	cognitiveEffects := []string{"DOM", "Msg", "Trace"}
	for _, name := range cognitiveEffects {
		t.Run(name, func(t *testing.T) {
			if !IsKnownEffect(name) {
				t.Errorf("Cognitive OS effect %q must be registered in IsKnownEffect", name)
			}
		})
	}
}

// TestElaborateEffectRow_CognitiveEffects pins that DOM, Msg, and Trace
// flow through effect-row elaboration both individually and combined.
// This is the row-inference acceptance test for M1 Day 1 of M-COG-RUNTIME.
func TestElaborateEffectRow_CognitiveEffects(t *testing.T) {
	cases := [][]string{
		{"DOM"},
		{"Msg"},
		{"Trace"},
		{"DOM", "Msg"},
		{"DOM", "Msg", "Trace"},
		{"DOM", "IO"},  // composes with existing effects
		{"Msg", "Net"}, // composes with existing effects
	}
	for _, eff := range cases {
		name := strings.Join(eff, ",")
		t.Run("{"+name+"}", func(t *testing.T) {
			row, err := ElaborateEffectRow(eff)
			if err != nil {
				t.Fatalf("ElaborateEffectRow(%v) failed: %v", eff, err)
			}
			if row == nil {
				t.Fatal("ElaborateEffectRow returned nil row")
			}
			for _, label := range eff {
				if _, ok := row.Labels[label]; !ok {
					t.Errorf("expected label %q in row, got labels=%v", label, row.Labels)
				}
			}
		})
	}
}

// TestSubsumeEffectRows_FS_Does_Not_Cover_Env is a focused regression test for
// the specific bug: declaring ! {FS} while calling Env-requiring functions.
func TestSubsumeEffectRows_FS_Does_Not_Cover_Env(t *testing.T) {
	envRow, err := ElaborateEffectRow([]string{"Env"})
	if err != nil {
		t.Fatalf("failed to create Env row: %v", err)
	}

	fsRow, err := ElaborateEffectRow([]string{"FS"})
	if err != nil {
		t.Fatalf("failed to create FS row: %v", err)
	}

	if SubsumeEffectRows(envRow, fsRow) {
		t.Fatal("{Env} is subsumed by {FS} — this is a soundness hole! FS must not cover Env")
	}

	// But {Env} ⊆ {FS, Env} should be true
	fsEnvRow, err := ElaborateEffectRow([]string{"FS", "Env"})
	if err != nil {
		t.Fatalf("failed to create FS+Env row: %v", err)
	}

	if !SubsumeEffectRows(envRow, fsEnvRow) {
		t.Fatal("{Env} should be subsumed by {FS, Env}")
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}
