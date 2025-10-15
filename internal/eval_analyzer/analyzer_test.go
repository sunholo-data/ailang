package eval_analyzer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sunholo/ailang/internal/eval_harness"
)

func TestAnalyzer(t *testing.T) {
	// Create temporary directory for test results
	tmpDir := t.TempDir()

	// Create sample metrics files
	metrics := []eval_harness.RunMetrics{
		{
			ID:            "fizzbuzz",
			Lang:          "ailang",
			Model:         "gpt5",
			Seed:          42,
			InputTokens:   100,
			OutputTokens:  200,
			TotalTokens:   300,
			CostUSD:       0.01,
			CompileOk:     false,
			RuntimeOk:     false,
			StdoutOk:      false,
			DurationMs:    1000,
			ErrorCategory: eval_harness.ErrorCategoryCompile,
			Stderr:        "parse error: expected 'then' got 'else'",
			Timestamp:     time.Now(),
			Code:          "if x > 0 else x",
		},
		{
			ID:            "fizzbuzz",
			Lang:          "ailang",
			Model:         "gpt5",
			Seed:          42,
			InputTokens:   100,
			OutputTokens:  200,
			TotalTokens:   300,
			CostUSD:       0.01,
			CompileOk:     false,
			RuntimeOk:     false,
			StdoutOk:      false,
			DurationMs:    1000,
			ErrorCategory: eval_harness.ErrorCategoryCompile,
			Stderr:        "parse error: expected 'then' got 'else'",
			Timestamp:     time.Now(),
			Code:          "if x > 0 else x",
		},
		{
			ID:            "json_parse",
			Lang:          "ailang",
			Model:         "gpt5",
			Seed:          42,
			InputTokens:   150,
			OutputTokens:  250,
			TotalTokens:   400,
			CostUSD:       0.012,
			CompileOk:     true,
			RuntimeOk:     true,
			StdoutOk:      true,
			DurationMs:    800,
			ErrorCategory: eval_harness.ErrorCategoryNone,
			Stderr:        "",
			Timestamp:     time.Now(),
			Code:          "module test\nexport func main() -> () ! {IO} { println(\"ok\") }",
		},
	}

	// Write metrics to files
	for i, m := range metrics {
		data, err := json.Marshal(m)
		if err != nil {
			t.Fatalf("failed to marshal metrics: %v", err)
		}

		filename := filepath.Join(tmpDir, m.ID+"_"+m.Lang+"_"+m.Model+"_"+string(rune('0'+i))+".json")
		if err := os.WriteFile(filename, data, 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
	}

	// Test analyzer
	t.Run("basic analysis", func(t *testing.T) {
		analyzer := NewAnalyzer(tmpDir, 1, nil)
		result, err := analyzer.Analyze()

		if err != nil {
			t.Fatalf("Analyze() failed: %v", err)
		}

		if result.TotalRuns != 3 {
			t.Errorf("expected 3 total runs, got %d", result.TotalRuns)
		}

		if result.FailureCount != 2 {
			t.Errorf("expected 2 failures, got %d", result.FailureCount)
		}

		expectedSuccessRate := (1.0 / 3.0) * 100.0
		if result.SuccessRate < expectedSuccessRate-0.1 || result.SuccessRate > expectedSuccessRate+0.1 {
			t.Errorf("expected success rate ~%.1f%%, got %.1f%%", expectedSuccessRate, result.SuccessRate)
		}

		if len(result.Issues) == 0 {
			t.Error("expected at least one issue")
		}
	})

	t.Run("frequency filtering", func(t *testing.T) {
		// Require at least 2 occurrences
		analyzer := NewAnalyzer(tmpDir, 2, nil)
		result, err := analyzer.Analyze()

		if err != nil {
			t.Fatalf("Analyze() failed: %v", err)
		}

		// Should have 1 issue (fizzbuzz with 2 failures)
		if len(result.Issues) != 1 {
			t.Errorf("expected 1 issue with frequency >= 2, got %d", len(result.Issues))
		}

		if len(result.Issues) > 0 {
			issue := result.Issues[0]
			if issue.Frequency != 2 {
				t.Errorf("expected frequency 2, got %d", issue.Frequency)
			}
			if issue.Category != eval_harness.ErrorCategoryCompile {
				t.Errorf("expected category %s, got %s", eval_harness.ErrorCategoryCompile, issue.Category)
			}
		}
	})

	t.Run("category filtering", func(t *testing.T) {
		// Only analyze compile errors
		analyzer := NewAnalyzer(tmpDir, 1, []string{eval_harness.ErrorCategoryCompile})
		result, err := analyzer.Analyze()

		if err != nil {
			t.Fatalf("Analyze() failed: %v", err)
		}

		for _, issue := range result.Issues {
			if issue.Category != eval_harness.ErrorCategoryCompile {
				t.Errorf("expected only compile_error category, got %s", issue.Category)
			}
		}
	})
}

func TestImpactCalculation(t *testing.T) {
	tests := []struct {
		name          string
		frequency     int
		totalFailures int
		category      string
		wantImpact    string
	}{
		{
			name:          "critical - high percentage",
			frequency:     60,
			totalFailures: 100,
			category:      eval_harness.ErrorCategoryLogic,
			wantImpact:    "critical",
		},
		{
			name:          "critical - compile error",
			frequency:     10,
			totalFailures: 100,
			category:      eval_harness.ErrorCategoryCompile,
			wantImpact:    "critical",
		},
		{
			name:          "high - runtime error",
			frequency:     30,
			totalFailures: 100,
			category:      eval_harness.ErrorCategoryRuntime,
			wantImpact:    "high",
		},
		{
			name:          "medium - moderate percentage",
			frequency:     15,
			totalFailures: 100,
			category:      eval_harness.ErrorCategoryLogic,
			wantImpact:    "medium",
		},
		{
			name:          "low - small percentage",
			frequency:     5,
			totalFailures: 100,
			category:      eval_harness.ErrorCategoryLogic,
			wantImpact:    "low",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := IssueReport{
				Frequency: tt.frequency,
				Category:  tt.category,
			}

			impact := calculateImpact(issue, tt.totalFailures)

			if impact != tt.wantImpact {
				t.Errorf("calculateImpact() = %s, want %s", impact, tt.wantImpact)
			}
		})
	}
}

func TestGenerateTitle(t *testing.T) {
	tests := []struct {
		category   string
		lang       string
		benchmarks []string
		want       string
	}{
		{
			category:   eval_harness.ErrorCategoryCompile,
			lang:       "ailang",
			benchmarks: []string{"fizzbuzz"},
			want:       "AILANG: Compilation Failures",
		},
		{
			category:   eval_harness.ErrorCategoryRuntime,
			lang:       "python",
			benchmarks: []string{"json_parse"},
			want:       "PYTHON: Runtime Errors",
		},
		{
			category:   eval_harness.ErrorCategoryLogic,
			lang:       "ailang",
			benchmarks: []string{"fizzbuzz", "pipeline"},
			want:       "AILANG: Logic Errors in fizzbuzz, pipeline",
		},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := generateTitle(tt.category, tt.lang, tt.benchmarks)
			if got != tt.want {
				t.Errorf("generateTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}
