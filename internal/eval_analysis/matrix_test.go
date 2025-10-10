package eval_analysis

import (
	"testing"
	"time"
)

func TestGenerateMatrix(t *testing.T) {
	tests := []struct {
		name    string
		results []*BenchmarkResult
		wantErr bool
	}{
		{
			name:    "empty results",
			results: []*BenchmarkResult{},
			wantErr: true,
		},
		{
			name: "single result",
			results: []*BenchmarkResult{
				{
					ID:             "test1",
					Lang:           "ailang",
					Model:          "claude-sonnet-4-5",
					StdoutOk:       true,
					FirstAttemptOk: true,
					TotalTokens:    100,
					CostUSD:        0.001,
					DurationMs:     500,
					Timestamp:      time.Now(),
				},
			},
			wantErr: false,
		},
		{
			name: "division by zero safety - no repairs",
			results: []*BenchmarkResult{
				{
					ID:             "test1",
					Lang:           "ailang",
					Model:          "claude-sonnet-4-5",
					StdoutOk:       false,
					FirstAttemptOk: false,
					RepairUsed:     false, // No repairs attempted
					TotalTokens:    50,
					Timestamp:      time.Now(),
				},
			},
			wantErr: false,
		},
		{
			name: "multiple models and benchmarks",
			results: []*BenchmarkResult{
				{
					ID:             "fizzbuzz",
					Lang:           "ailang",
					Model:          "claude-sonnet-4-5",
					StdoutOk:       true,
					FirstAttemptOk: true,
					TotalTokens:    100,
					Timestamp:      time.Now(),
				},
				{
					ID:             "fizzbuzz",
					Lang:           "ailang",
					Model:          "gpt5",
					StdoutOk:       false,
					FirstAttemptOk: false,
					RepairUsed:     true,
					RepairOk:       true,
					TotalTokens:    150,
					Timestamp:      time.Now(),
				},
				{
					ID:             "factorial",
					Lang:           "ailang",
					Model:          "claude-sonnet-4-5",
					StdoutOk:       true,
					FirstAttemptOk: false,
					RepairUsed:     true,
					RepairOk:       true,
					TotalTokens:    80,
					Timestamp:      time.Now(),
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matrix, err := GenerateMatrix(tt.results, "v0.test")
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateMatrix() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Basic validation
			if matrix.Version != "v0.test" {
				t.Errorf("Version = %v, want v0.test", matrix.Version)
			}

			if matrix.TotalRuns != len(tt.results) {
				t.Errorf("TotalRuns = %v, want %v", matrix.TotalRuns, len(tt.results))
			}

			// Check aggregates are computed
			if matrix.Aggregates.ZeroShotSuccess < 0 || matrix.Aggregates.ZeroShotSuccess > 1 {
				t.Errorf("Invalid ZeroShotSuccess rate: %v", matrix.Aggregates.ZeroShotSuccess)
			}

			if matrix.Aggregates.RepairSuccessRate < 0 || matrix.Aggregates.RepairSuccessRate > 1 {
				t.Errorf("Invalid RepairSuccessRate: %v", matrix.Aggregates.RepairSuccessRate)
			}

			// Check models are grouped
			if len(matrix.Models) == 0 {
				t.Error("No models in matrix")
			}

			// Check benchmarks are grouped
			if len(matrix.Benchmarks) == 0 {
				t.Error("No benchmarks in matrix")
			}
		})
	}
}

func TestSafeDivZero(t *testing.T) {
	// Test that safeDiv handles division by zero
	result := safeDiv(10, 0)
	if result != 0 {
		t.Errorf("safeDiv(10, 0) = %v, want 0", result)
	}

	result = safeDiv(0, 0)
	if result != 0 {
		t.Errorf("safeDiv(0, 0) = %v, want 0", result)
	}

	result = safeDiv(10, 5)
	if result != 2 {
		t.Errorf("safeDiv(10, 5) = %v, want 2", result)
	}
}

func TestCalculateAggregates(t *testing.T) {
	results := []*BenchmarkResult{
		{
			FirstAttemptOk: true,
			StdoutOk:       true,
			RepairUsed:     false,
			TotalTokens:    100,
			CostUSD:        0.001,
			DurationMs:     100,
		},
		{
			FirstAttemptOk: false,
			StdoutOk:       true,
			RepairUsed:     true,
			RepairOk:       true,
			TotalTokens:    150,
			CostUSD:        0.002,
			DurationMs:     200,
		},
		{
			FirstAttemptOk: false,
			StdoutOk:       false,
			RepairUsed:     true,
			RepairOk:       false,
			TotalTokens:    80,
			CostUSD:        0.001,
			DurationMs:     150,
		},
	}

	agg := calculateAggregates(results)

	// Expected: 1/3 first attempt success
	expectedZeroShot := 1.0 / 3.0
	if agg.ZeroShotSuccess != expectedZeroShot {
		t.Errorf("ZeroShotSuccess = %v, want %v", agg.ZeroShotSuccess, expectedZeroShot)
	}

	// Expected: 2/3 final success
	expectedFinal := 2.0 / 3.0
	if agg.FinalSuccess != expectedFinal {
		t.Errorf("FinalSuccess = %v, want %v", agg.FinalSuccess, expectedFinal)
	}

	// Expected: 2 repairs used
	if agg.RepairUsed != 2 {
		t.Errorf("RepairUsed = %v, want 2", agg.RepairUsed)
	}

	// Expected: 1/2 repair success rate
	expectedRepair := 0.5
	if agg.RepairSuccessRate != expectedRepair {
		t.Errorf("RepairSuccessRate = %v, want %v", agg.RepairSuccessRate, expectedRepair)
	}

	// Expected: 330 total tokens
	if agg.TotalTokens != 330 {
		t.Errorf("TotalTokens = %v, want 330", agg.TotalTokens)
	}
}

func TestGroupByModel(t *testing.T) {
	results := []*BenchmarkResult{
		{
			ID:        "test1",
			Model:     "claude",
			StdoutOk:  true,
			Timestamp: time.Now(),
		},
		{
			ID:        "test2",
			Model:     "claude",
			StdoutOk:  false,
			Timestamp: time.Now(),
		},
		{
			ID:        "test1",
			Model:     "gpt5",
			StdoutOk:  true,
			Timestamp: time.Now(),
		},
	}

	models := groupByModel(results)

	if len(models) != 2 {
		t.Errorf("Expected 2 models, got %v", len(models))
	}

	if models["claude"].TotalRuns != 2 {
		t.Errorf("Expected claude to have 2 runs, got %v", models["claude"].TotalRuns)
	}

	if models["gpt5"].TotalRuns != 1 {
		t.Errorf("Expected gpt5 to have 1 run, got %v", models["gpt5"].TotalRuns)
	}
}

func TestGroupByErrorCode(t *testing.T) {
	results := []*BenchmarkResult{
		{
			StdoutOk: false,
			ErrCode:  "PAR_001",
			RepairOk: true,
		},
		{
			StdoutOk: false,
			ErrCode:  "PAR_001",
			RepairOk: false,
		},
		{
			StdoutOk: false,
			ErrCode:  "TC_REC_001",
			RepairOk: true,
		},
		{
			StdoutOk: true, // Success, should be ignored
			ErrCode:  "NONE",
		},
	}

	errorCodes := groupByErrorCode(results)

	// Should have 2 error codes (success is ignored)
	if len(errorCodes) != 2 {
		t.Errorf("Expected 2 error codes, got %v", len(errorCodes))
	}

	// Check PAR_001 stats
	var par001 *ErrorCodeStats
	for _, ec := range errorCodes {
		if ec.Code == "PAR_001" {
			par001 = ec
			break
		}
	}

	if par001 == nil {
		t.Fatal("PAR_001 not found")
	}

	if par001.Count != 2 {
		t.Errorf("PAR_001 count = %v, want 2", par001.Count)
	}

	expectedRepair := 0.5 // 1 success out of 2
	if par001.RepairSuccess != expectedRepair {
		t.Errorf("PAR_001 repair success = %v, want %v", par001.RepairSuccess, expectedRepair)
	}
}
