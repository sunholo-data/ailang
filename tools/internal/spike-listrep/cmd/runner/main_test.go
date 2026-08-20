package main

import (
	"strings"
	"testing"
)

func valid(values ...float64) []trial {
	result := make([]trial, len(values))
	for i, value := range values {
		result[i] = trial{Ordinal: i + 1, Value: value, Valid: true}
	}
	return result
}

func TestAdjudicatorSyntheticVectors(t *testing.T) {
	ones5 := valid(1, 1, 1, 1, 1)
	ones10 := valid(1, 1, 1, 1, 1, 1, 1, 1, 1, 1)
	tests := []struct {
		name                                       string
		candidate, control                         []trial
		threshold                                  float64
		op                                         string
		allowRerun, wantValid, wantRerun, wantPass bool
		wantMedian                                 float64
	}{
		{"unsorted odd median", valid(5, 1, 4, 2, 3), ones5, 10, "<=", true, true, false, true, 3},
		{"median exactly threshold", valid(1, 2, 3, 4, 5), ones5, 3, "<=", true, true, true, false, 3},
		{"spans threshold", valid(1, 2, 2, 4, 5), ones5, 3, "<=", true, true, true, false, 2},
		{"equality touches both", valid(1, 1, 1, 1, 2), ones5, 2, "<=", true, true, true, false, 1},
		{"ten equality passes less", valid(1, 1, 1, 1, 2, 2, 3, 3, 3, 3), ones10, 2, "<=", false, true, false, true, 2},
		{"ten equality passes greater", valid(1, 1, 1, 1, 2, 2, 3, 3, 3, 3), ones10, 2, ">=", false, true, false, true, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := paired(tt.candidate, tt.control, tt.threshold, tt.op, tt.allowRerun)
			if got.Valid != tt.wantValid || got.Rerun != tt.wantRerun || got.Pass != tt.wantPass || got.Median != tt.wantMedian {
				t.Fatalf("got valid=%v rerun=%v pass=%v median=%v; want %v %v %v %v", got.Valid, got.Rerun, got.Pass, got.Median, tt.wantValid, tt.wantRerun, tt.wantPass, tt.wantMedian)
			}
		})
	}
}

func TestInvalidTrialDoesNotShrinkSample(t *testing.T) {
	candidate := valid(1, 2, 3, 4, 5)
	candidate[2] = trial{Ordinal: 3, Error: "deadline killed subprocess"}
	got := paired(candidate, valid(1, 1, 1, 1, 1), 3, "<=", true)
	if got.Valid || len(got.Candidate) != 5 || !strings.Contains(got.Error, "invalid trial") {
		t.Fatalf("invalid trial was not surfaced: %+v", got)
	}
}

func TestValidateCellRejectsNonSingletonRegex(t *testing.T) {
	for _, pattern := range []string{"BenchmarkListRep_B3", "^BenchmarkListRep_B3_Iteration/.*$", "^does-not-exist$"} {
		if _, err := validateCell(pattern); err == nil {
			t.Errorf("validateCell(%q) accepted a non-singleton", pattern)
		}
	}
	if _, err := validateCell("^BenchmarkListRep_B3_Iteration/arm=C0/n=4096$"); err != nil {
		t.Fatalf("exact cell rejected: %v", err)
	}
}
