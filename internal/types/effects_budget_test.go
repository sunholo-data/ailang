package types

import (
	"sort"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

func TestElaborateEffectRowWithBudgets(t *testing.T) {
	tests := []struct {
		name            string
		effects         []ast.EffectAnnotation
		expectNil       bool
		expectedLabels  []string
		expectedBudgets map[string]*int
		expectError     bool
	}{
		{
			name:      "empty effects returns nil",
			effects:   []ast.EffectAnnotation{},
			expectNil: true,
		},
		{
			name: "single effect without budget",
			effects: []ast.EffectAnnotation{
				{Name: "IO", Budget: nil},
			},
			expectedLabels:  []string{"IO"},
			expectedBudgets: nil,
		},
		{
			name: "single effect with budget",
			effects: []ast.EffectAnnotation{
				{Name: "IO", Budget: intPtr(5)},
			},
			expectedLabels:  []string{"IO"},
			expectedBudgets: map[string]*int{"IO": intPtr(5)},
		},
		{
			name: "mixed budgets",
			effects: []ast.EffectAnnotation{
				{Name: "IO", Budget: intPtr(5)},
				{Name: "FS", Budget: nil},
				{Name: "Net", Budget: intPtr(10)},
			},
			expectedLabels:  []string{"FS", "IO", "Net"},
			expectedBudgets: map[string]*int{"IO": intPtr(5), "Net": intPtr(10)},
		},
		{
			name: "zero budget is valid",
			effects: []ast.EffectAnnotation{
				{Name: "IO", Budget: intPtr(0)},
			},
			expectedLabels:  []string{"IO"},
			expectedBudgets: map[string]*int{"IO": intPtr(0)},
		},
		{
			name: "unknown effect returns error",
			effects: []ast.EffectAnnotation{
				{Name: "Unknown", Budget: intPtr(5)},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row, err := ElaborateEffectRowWithBudgets(tt.effects)

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
					t.Errorf("expected nil row, got %v", row)
				}
				return
			}

			if row == nil {
				t.Fatal("expected non-nil row, got nil")
			}

			// Check labels
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

			// Check budgets
			if tt.expectedBudgets == nil {
				if len(row.Budgets) > 0 {
					t.Errorf("expected no budgets, got %v", row.Budgets)
				}
			} else {
				if row.Budgets == nil {
					t.Errorf("expected budgets %v, got nil", tt.expectedBudgets)
				} else {
					for name, expectedBudget := range tt.expectedBudgets {
						actualBudget := row.Budgets[name]
						if expectedBudget == nil && actualBudget != nil {
							t.Errorf("budget for %s: expected nil, got %d", name, *actualBudget)
						} else if expectedBudget != nil && actualBudget == nil {
							t.Errorf("budget for %s: expected %d, got nil", name, *expectedBudget)
						} else if expectedBudget != nil && actualBudget != nil && *expectedBudget != *actualBudget {
							t.Errorf("budget for %s: expected %d, got %d", name, *expectedBudget, *actualBudget)
						}
					}
				}
			}
		})
	}
}

func TestFormatEffectRowWithBudgets(t *testing.T) {
	tests := []struct {
		name     string
		effects  []ast.EffectAnnotation
		expected string
	}{
		{
			name:     "nil formats as empty string",
			effects:  nil,
			expected: "",
		},
		{
			name: "single effect with budget",
			effects: []ast.EffectAnnotation{
				{Name: "IO", Budget: intPtr(5)},
			},
			expected: "! {IO @limit=5}",
		},
		{
			name: "multiple effects with mixed budgets",
			effects: []ast.EffectAnnotation{
				{Name: "IO", Budget: intPtr(5)},
				{Name: "FS", Budget: nil},
			},
			expected: "! {FS, IO @limit=5}",
		},
		{
			name: "all effects with budgets sorted",
			effects: []ast.EffectAnnotation{
				{Name: "Net", Budget: intPtr(3)},
				{Name: "IO", Budget: intPtr(5)},
			},
			expected: "! {IO @limit=5, Net @limit=3}",
		},
		{
			name: "zero budget",
			effects: []ast.EffectAnnotation{
				{Name: "IO", Budget: intPtr(0)},
			},
			expected: "! {IO @limit=0}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var row *Row
			var err error

			if len(tt.effects) > 0 {
				row, err = ElaborateEffectRowWithBudgets(tt.effects)
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

func TestUnionEffectRowsWithBudgets(t *testing.T) {
	tests := []struct {
		name            string
		a               []ast.EffectAnnotation
		b               []ast.EffectAnnotation
		expectedLabels  []string
		expectedBudgets map[string]*int
	}{
		{
			name:            "nil ∪ nil = nil",
			a:               nil,
			b:               nil,
			expectedLabels:  nil,
			expectedBudgets: nil,
		},
		{
			name:            "nil ∪ {IO @limit=5} = {IO @limit=5}",
			a:               nil,
			b:               []ast.EffectAnnotation{{Name: "IO", Budget: intPtr(5)}},
			expectedLabels:  []string{"IO"},
			expectedBudgets: map[string]*int{"IO": intPtr(5)},
		},
		{
			name:            "{IO @limit=3} ∪ {IO @limit=5} = {IO @limit=8} (sum)",
			a:               []ast.EffectAnnotation{{Name: "IO", Budget: intPtr(3)}},
			b:               []ast.EffectAnnotation{{Name: "IO", Budget: intPtr(5)}},
			expectedLabels:  []string{"IO"},
			expectedBudgets: map[string]*int{"IO": intPtr(8)},
		},
		{
			name:            "{IO @limit=5} ∪ {IO} = {IO @limit=5} (one has budget)",
			a:               []ast.EffectAnnotation{{Name: "IO", Budget: intPtr(5)}},
			b:               []ast.EffectAnnotation{{Name: "IO", Budget: nil}},
			expectedLabels:  []string{"IO"},
			expectedBudgets: map[string]*int{"IO": intPtr(5)},
		},
		{
			name:            "{IO @limit=5} ∪ {FS @limit=3} = {FS @limit=3, IO @limit=5}",
			a:               []ast.EffectAnnotation{{Name: "IO", Budget: intPtr(5)}},
			b:               []ast.EffectAnnotation{{Name: "FS", Budget: intPtr(3)}},
			expectedLabels:  []string{"FS", "IO"},
			expectedBudgets: map[string]*int{"FS": intPtr(3), "IO": intPtr(5)},
		},
		{
			name:            "{IO, FS} ∪ {Net} = {FS, IO, Net} (no budgets)",
			a:               []ast.EffectAnnotation{{Name: "IO"}, {Name: "FS"}},
			b:               []ast.EffectAnnotation{{Name: "Net"}},
			expectedLabels:  []string{"FS", "IO", "Net"},
			expectedBudgets: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rowA, rowB *Row
			var err error

			if tt.a != nil {
				rowA, err = ElaborateEffectRowWithBudgets(tt.a)
				if err != nil {
					t.Fatalf("failed to create row A: %v", err)
				}
			}

			if tt.b != nil {
				rowB, err = ElaborateEffectRowWithBudgets(tt.b)
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

			// Check budgets
			if tt.expectedBudgets == nil {
				if len(result.Budgets) > 0 {
					t.Errorf("expected no budgets, got %v", result.Budgets)
				}
			} else {
				if result.Budgets == nil {
					t.Errorf("expected budgets %v, got nil", tt.expectedBudgets)
				} else {
					for name, expectedBudget := range tt.expectedBudgets {
						actualBudget := result.Budgets[name]
						if expectedBudget == nil && actualBudget != nil {
							t.Errorf("budget for %s: expected nil, got %d", name, *actualBudget)
						} else if expectedBudget != nil && actualBudget == nil {
							t.Errorf("budget for %s: expected %d, got nil", name, *expectedBudget)
						} else if expectedBudget != nil && actualBudget != nil && *expectedBudget != *actualBudget {
							t.Errorf("budget for %s: expected %d, got %d", name, *expectedBudget, *actualBudget)
						}
					}
				}
			}
		})
	}
}

func TestRowEqualsWithBudgets(t *testing.T) {
	tests := []struct {
		name     string
		a        []ast.EffectAnnotation
		b        []ast.EffectAnnotation
		expected bool
	}{
		{
			name:     "same effects, same budgets",
			a:        []ast.EffectAnnotation{{Name: "IO", Budget: intPtr(5)}},
			b:        []ast.EffectAnnotation{{Name: "IO", Budget: intPtr(5)}},
			expected: true,
		},
		{
			name:     "same effects, different budgets",
			a:        []ast.EffectAnnotation{{Name: "IO", Budget: intPtr(5)}},
			b:        []ast.EffectAnnotation{{Name: "IO", Budget: intPtr(10)}},
			expected: false,
		},
		{
			name:     "same effects, one with budget, one without",
			a:        []ast.EffectAnnotation{{Name: "IO", Budget: intPtr(5)}},
			b:        []ast.EffectAnnotation{{Name: "IO", Budget: nil}},
			expected: false,
		},
		{
			name:     "same effects, no budgets",
			a:        []ast.EffectAnnotation{{Name: "IO"}},
			b:        []ast.EffectAnnotation{{Name: "IO"}},
			expected: true,
		},
		{
			name:     "different effects",
			a:        []ast.EffectAnnotation{{Name: "IO", Budget: intPtr(5)}},
			b:        []ast.EffectAnnotation{{Name: "FS", Budget: intPtr(5)}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rowA, err := ElaborateEffectRowWithBudgets(tt.a)
			if err != nil {
				t.Fatalf("failed to create row A: %v", err)
			}

			rowB, err := ElaborateEffectRowWithBudgets(tt.b)
			if err != nil {
				t.Fatalf("failed to create row B: %v", err)
			}

			result := rowA.Equals(rowB)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
