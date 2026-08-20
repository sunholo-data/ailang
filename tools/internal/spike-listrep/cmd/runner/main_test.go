package main

import (
	"context"
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

// --- iteration 238: rerun-ORCHESTRATION arms ------------------------------
//
// The evaluator showed that mutating main()'s rerun batch size (5 -> 4) or
// re-enabling allowRerun on the second paired() call both SURVIVED the whole
// suite, because every existing arm calls paired() directly with a hand-built
// vector and nothing exercised the orchestration. These arms pin it via an
// injected collector, so no benchmark runs.

type recordedCall struct {
	selector string
	start    int
	count    int
}

// fakeCollector returns trials whose values are supplied per invocation, and
// records the (start, count) of every batch it was asked for.
func fakeCollector(values map[string][]float64, calls *[]recordedCall) collector {
	return func(_ context.Context, in invocation, start, count int) []trial {
		*calls = append(*calls, recordedCall{in.Selector, start, count})
		out := make([]trial, 0, count)
		for i := range count {
			ordinal := start + i
			out = append(out, trial{Ordinal: ordinal, Value: values[in.Selector][ordinal-1], Valid: true})
		}
		return out
	}
}

func TestAdjudicateNoRerunCollectsFiveTrialsOnly(t *testing.T) {
	// Ratios all strictly below the threshold: no tie, no spread, no rerun.
	values := map[string][]float64{
		"cand": {1.0, 1.0, 1.0, 1.0, 1.0},
		"ctrl": {1.0, 1.0, 1.0, 1.0, 1.0},
	}
	var calls []recordedCall
	got := adjudicate(context.Background(), fakeCollector(values, &calls),
		invocation{Selector: "cand"}, invocation{Selector: "ctrl"}, 2.0, "<=")

	if got.Rerun {
		t.Fatalf("no rerun expected, got Rerun=true")
	}
	if len(calls) != 2 {
		t.Fatalf("want exactly 2 collector batches (one per arm), got %d: %+v", len(calls), calls)
	}
	if len(got.Candidate) != initialTrials || len(got.Control) != initialTrials {
		t.Fatalf("want %d trials per arm, got cand=%d ctrl=%d",
			initialTrials, len(got.Candidate), len(got.Control))
	}
}

func TestAdjudicateRerunCollectsExactlyFiveMoreAndDoesNotCascade(t *testing.T) {
	// Ratios that TOUCH the threshold on both sides -> rerun is required.
	// Ten values per arm so the rerun batch has data to draw on.
	values := map[string][]float64{
		"cand": {0.5, 1.0, 2.0, 1.0, 2.0, 1.0, 1.0, 1.0, 1.0, 1.0},
		"ctrl": {1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0},
	}
	var calls []recordedCall
	got := adjudicate(context.Background(), fakeCollector(values, &calls),
		invocation{Selector: "cand"}, invocation{Selector: "ctrl"}, 1.0, "<=")

	// 1. The rerun happened: four batches, not two.
	if len(calls) != 4 {
		t.Fatalf("want 4 collector batches (2 initial + 2 rerun), got %d: %+v", len(calls), calls)
	}
	// 2. The rerun batch is exactly rerunTrials, numbered contiguously.
	//    Kills the "rerun batch 5 -> 4" mutant.
	for _, c := range calls[2:] {
		if c.count != rerunTrials {
			t.Errorf("rerun batch for %q: want count=%d, got %d", c.selector, rerunTrials, c.count)
		}
		if c.start != initialTrials+1 {
			t.Errorf("rerun batch for %q: want start=%d (contiguous), got %d",
				c.selector, initialTrials+1, c.start)
		}
	}
	// 3. The median of ALL TEN is final, and the analysis is over ten operands.
	if len(got.Candidate) != initialTrials+rerunTrials || len(got.Control) != initialTrials+rerunTrials {
		t.Fatalf("want %d trials per arm after rerun, got cand=%d ctrl=%d",
			initialTrials+rerunTrials, len(got.Candidate), len(got.Control))
	}
	if len(got.Ratios) != initialTrials+rerunTrials {
		t.Fatalf("want %d paired ratios after rerun, got %d", initialTrials+rerunTrials, len(got.Ratios))
	}
	// 4. A rerun may never cascade: the final analysis is computed with
	//    allowRerun=false, so Rerun must be cleared. Kills the
	//    "second paired() call allowRerun false -> true" mutant.
	if got.Rerun {
		t.Errorf("rerun cascaded: final analysis still reports Rerun=true; " +
			"the median of all ten must be final")
	}
}
