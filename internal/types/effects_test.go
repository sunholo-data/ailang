package types

import (
	"sort"
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
			name:     "all effects",
			effects:  []string{"IO", "FS", "Net", "Clock", "Rand", "DB", "Trace", "Async"},
			expected: "! {Async, Clock, DB, FS, IO, Net, Rand, Trace}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var row *Row
			var err error

			if tt.effects != nil && len(tt.effects) > 0 {
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
