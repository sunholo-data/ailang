package eval_analyzer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCalculateBenchmarkOverlap(t *testing.T) {
	tests := []struct {
		name        string
		benchmarks1 []string
		benchmarks2 []string
		want        float64
	}{
		{
			name:        "identical benchmarks",
			benchmarks1: []string{"fizzbuzz", "adt_option"},
			benchmarks2: []string{"fizzbuzz", "adt_option"},
			want:        1.0,
		},
		{
			name:        "partial overlap",
			benchmarks1: []string{"fizzbuzz", "adt_option"},
			benchmarks2: []string{"fizzbuzz", "pipeline"},
			want:        0.333, // 1 overlap / 3 union
		},
		{
			name:        "no overlap",
			benchmarks1: []string{"fizzbuzz"},
			benchmarks2: []string{"pipeline"},
			want:        0.0,
		},
		{
			name:        "empty lists",
			benchmarks1: []string{},
			benchmarks2: []string{},
			want:        0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateBenchmarkOverlap(tt.benchmarks1, tt.benchmarks2)
			diff := got - tt.want
			if diff < 0 {
				diff = -diff
			}
			if diff > 0.01 {
				t.Errorf("calculateBenchmarkOverlap() = %.3f, want %.3f", got, tt.want)
			}
		})
	}
}

func TestFuzzyErrorMatch(t *testing.T) {
	tests := []struct {
		name string
		err1 string
		err2 string
		want bool
	}{
		{
			name: "exact match",
			err1: "Error: builtin eq_Int expects Int arguments",
			err2: "Error: builtin eq_Int expects Int arguments",
			want: true,
		},
		{
			name: "case insensitive match",
			err1: "Error: Builtin eq_Int Expects Int Arguments",
			err2: "error: builtin eq_int expects int arguments",
			want: true,
		},
		{
			name: "substring match",
			err1: "Error: builtin eq_Int expects Int arguments",
			err2: "builtin eq_Int expects Int",
			want: true,
		},
		{
			name: "similar pattern different types",
			err1: "Error: builtin eq_Int expects Int arguments and more text",
			err2: "Error: builtin eq_Float expects Float arguments and similar text",
			want: false, // Different error types - should NOT match
		},
		{
			name: "different errors",
			err1: "Error: parse error expected 'then'",
			err2: "Error: module not found",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fuzzyErrorMatch(tt.err1, tt.err2)
			if got != tt.want {
				t.Errorf("fuzzyErrorMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetermineMergeStrategy(t *testing.T) {
	config := DefaultDedupConfig()

	tests := []struct {
		name     string
		issue    IssueReport
		similar  []SimilarDoc
		wantStrategy MergeStrategy
	}{
		{
			name: "no similar docs - create",
			issue: IssueReport{
				Category: "compile_error",
				Frequency: 5,
			},
			similar: []SimilarDoc{},
			wantStrategy: StrategyCreate,
		},
		{
			name: "very high similarity - merge",
			issue: IssueReport{
				Category: "runtime_error",
				Frequency: 5,
			},
			similar: []SimilarDoc{
				{
					SimilarityScore: 0.95,
					Frequency: 3,
				},
			},
			wantStrategy: StrategyMerge,
		},
		{
			name: "high similarity - merge",
			issue: IssueReport{
				Category: "runtime_error",
				Frequency: 5,
			},
			similar: []SimilarDoc{
				{
					SimilarityScore: 0.80,
					Frequency: 3,
				},
			},
			wantStrategy: StrategyMerge,
		},
		{
			name: "moderate similarity - link",
			issue: IssueReport{
				Category: "runtime_error",
				Frequency: 5,
			},
			similar: []SimilarDoc{
				{
					SimilarityScore: 0.60,
					Frequency: 3,
				},
			},
			wantStrategy: StrategyLink,
		},
		{
			name: "low similarity - create",
			issue: IssueReport{
				Category: "runtime_error",
				Frequency: 5,
			},
			similar: []SimilarDoc{
				{
					SimilarityScore: 0.40,
					Frequency: 3,
				},
			},
			wantStrategy: StrategyCreate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStrategy, _ := DetermineMergeStrategy(tt.issue, tt.similar, config)
			if gotStrategy != tt.wantStrategy {
				t.Errorf("DetermineMergeStrategy() strategy = %v, want %v", gotStrategy, tt.wantStrategy)
			}
		})
	}
}

func TestFindSimilarDesignDocs(t *testing.T) {
	// Create temporary directory for test docs
	tmpDir := t.TempDir()

	// Create mock design doc
	mockDoc := `# AILANG: Runtime Errors

**Discovered**: AI Eval Analysis - 2025-10-06
**Frequency**: 7 failures across 2 benchmark(s)
**Category**: runtime_error
**Impact**: high

## Problem Statement

AILANG benchmarks adt_option and fizzbuzz exhibit runtime_error failures.

## Evidence from AI Eval

**Affected Benchmarks**: adt_option, fizzbuzz

**Error 1:**
` + "```" + `
Error: builtin eq_Int expects Int arguments
` + "```" + `
`

	docPath := filepath.Join(tmpDir, "20251006_runtime_error_ailang.md")
	if err := os.WriteFile(docPath, []byte(mockDoc), 0644); err != nil {
		t.Fatalf("failed to write mock doc: %v", err)
	}

	// Test finding similar docs
	issue := IssueReport{
		Category:   "runtime_error",
		Lang:       "ailang",
		Benchmarks: []string{"adt_option", "fizzbuzz"},
		ErrorMessages: []string{"Error: builtin eq_Int expects Int arguments"},
	}

	config := DefaultDedupConfig()
	similar, err := FindSimilarDesignDocs(issue, tmpDir, config)

	if err != nil {
		t.Fatalf("FindSimilarDesignDocs() error = %v", err)
	}

	if len(similar) != 1 {
		t.Errorf("FindSimilarDesignDocs() found %d docs, want 1", len(similar))
	}

	if len(similar) > 0 {
		if similar[0].Category != "runtime_error" {
			t.Errorf("similar[0].Category = %s, want runtime_error", similar[0].Category)
		}
		if similar[0].Language != "ailang" {
			t.Errorf("similar[0].Language = %s, want ailang", similar[0].Language)
		}
		if similar[0].SimilarityScore < 0.75 {
			t.Errorf("similar[0].SimilarityScore = %.2f, want >= 0.75", similar[0].SimilarityScore)
		}
	}
}

func TestMergeDesignDoc(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create original doc
	originalDoc := `# AILANG: Runtime Errors

**Discovered**: AI Eval Analysis - 2025-10-06
**Frequency**: 5 failures across 2 benchmark(s)
**Category**: runtime_error
**Impact**: high

## Problem Statement

Test problem statement.

## Evidence from AI Eval

**Affected Benchmarks**: adt_option, fizzbuzz

### Example Failures

**Error 1:**
` + "```" + `
Error: original error
` + "```" + `

---


## Root Cause Analysis

Test root cause.
`

	docPath := filepath.Join(tmpDir, "test_runtime_error.md")
	if err := os.WriteFile(docPath, []byte(originalDoc), 0644); err != nil {
		t.Fatalf("failed to write original doc: %v", err)
	}

	// Create issue to merge
	issue := IssueReport{
		Category:   "runtime_error",
		Lang:       "ailang",
		Frequency:  3,
		Benchmarks: []string{"pipeline"},
		ErrorMessages: []string{"Error: new error"},
		Examples:   []string{"new code example"},
	}

	// Merge
	err := MergeDesignDoc(docPath, issue, 10)
	if err != nil {
		t.Fatalf("MergeDesignDoc() error = %v", err)
	}

	// Read merged doc
	merged, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("failed to read merged doc: %v", err)
	}

	mergedStr := string(merged)

	// Debug: print first 500 chars
	t.Logf("Merged doc preview:\n%s\n...", mergedStr[:min(500, len(mergedStr))])

	// Verify frequency was updated (5 + 3 = 8)
	if !strings.Contains(mergedStr, "Frequency**: 8") {
		t.Errorf("merged doc should have updated frequency to 8")
	}

	// Verify new benchmark was added
	if !strings.Contains(mergedStr, "pipeline") {
		t.Error("merged doc should include new benchmark 'pipeline'")
	}

	// Verify update timestamp was added
	if !strings.Contains(mergedStr, "Last Updated") {
		t.Error("merged doc should have 'Last Updated' timestamp")
	}

	// Verify new examples were added
	if !strings.Contains(mergedStr, "Additional Examples") {
		t.Error("merged doc should have 'Additional Examples' section")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
