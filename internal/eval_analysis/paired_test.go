package eval_analysis

import (
	"math"
	"testing"
)

func pass(id string, trial int) *BenchmarkResult {
	return &BenchmarkResult{ID: id, Lang: "ailang", Trial: trial, CompileOk: true, RuntimeOk: true, StdoutOk: true}
}

func fail(id string, trial int) *BenchmarkResult {
	return &BenchmarkResult{ID: id, Lang: "ailang", Trial: trial}
}

// TestPairArms_JoinsOnBenchmarkLangTrial: trial MUST be part of the key. Both
// trials of a benchmark share (id, lang, seed), so joining without it would
// silently pair trial 1 of one arm against trial 2 of the other.
func TestPairArms_JoinsOnBenchmarkLangTrial(t *testing.T) {
	on := []*BenchmarkResult{pass("a", 1), fail("a", 2)}
	off := []*BenchmarkResult{fail("a", 1), pass("a", 2)}

	p := PairArms(on, off)

	if len(p.Pairs) != 2 {
		t.Fatalf("got %d pairs, want 2 (one per trial)", len(p.Pairs))
	}
	if p.Unpaired != 0 {
		t.Errorf("Unpaired = %d, want 0", p.Unpaired)
	}
	// a/trial1: on passed, off failed -> b. a/trial2: on failed, off passed -> c.
	if p.OnlyOnPassed != 1 || p.OnlyOffPassed != 1 {
		t.Errorf("discordant b=%d c=%d, want 1 and 1", p.OnlyOnPassed, p.OnlyOffPassed)
	}
}

// TestPairArms_ReportsUnpaired guards R2: a benchmark present in only one arm
// must be surfaced, never silently dropped — dropping biases the comparison.
func TestPairArms_ReportsUnpaired(t *testing.T) {
	on := []*BenchmarkResult{pass("a", 1), pass("orphan", 1)}
	off := []*BenchmarkResult{pass("a", 1)}

	p := PairArms(on, off)

	if len(p.Pairs) != 1 {
		t.Errorf("got %d pairs, want 1", len(p.Pairs))
	}
	if p.Unpaired != 1 {
		t.Errorf("Unpaired = %d, want 1 (the orphan must be surfaced)", p.Unpaired)
	}
}

func TestMcNemar(t *testing.T) {
	tests := []struct {
		name         string
		b, c         int // b = only-on passed, c = only-off passed
		wantReport   bool
		wantPGreater float64 // if wantReport, p must be >= this (loose sanity bound)
		wantPLess    float64 // ... and <= this
	}{
		{
			// The design-doc mitigation, enforced: with almost no discordant
			// pairs there is no evidence base, so reporting a p-value would be
			// false precision.
			name: "b+c=0 degenerate — no p-value", b: 0, c: 0, wantReport: false,
		},
		{
			name: "b+c below the floor — no p-value", b: 3, c: 2, wantReport: false,
		},
		{
			name: "balanced discordance is not significant", b: 10, c: 10,
			wantReport: true, wantPGreater: 0.9, wantPLess: 1.01,
		},
		{
			// Strongly one-sided: 18 vs 2 should be clearly significant.
			name: "strongly one-sided is significant", b: 18, c: 2,
			wantReport: true, wantPGreater: 0.0, wantPLess: 0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := McNemar(tt.b, tt.c)

			if r.Reportable != tt.wantReport {
				t.Fatalf("Reportable = %v, want %v (b=%d c=%d)", r.Reportable, tt.wantReport, tt.b, tt.c)
			}
			if !tt.wantReport {
				if r.PValue != 0 {
					t.Errorf("unreportable result must not carry a p-value, got %v", r.PValue)
				}
				if r.Note == "" {
					t.Error("unreportable result must explain WHY (too few discordant pairs)")
				}
				return
			}
			if r.PValue < tt.wantPGreater || r.PValue > tt.wantPLess {
				t.Errorf("PValue = %v, want in [%v, %v]", r.PValue, tt.wantPGreater, tt.wantPLess)
			}
		})
	}
}

// TestMcNemar_ExactVsChiSquare documents the b+c<25 switch. Both must agree on
// direction; the exact test is used for small samples where chi-square's
// normal approximation is unreliable.
func TestMcNemar_ExactVsChiSquare(t *testing.T) {
	small := McNemar(12, 2)  // b+c = 14 -> exact
	large := McNemar(40, 10) // b+c = 50 -> chi-square

	if small.Method != "exact_binomial" {
		t.Errorf("b+c=14 should use exact binomial, got %q", small.Method)
	}
	if large.Method != "chi_square_continuity" {
		t.Errorf("b+c=50 should use chi-square with continuity correction, got %q", large.Method)
	}
	if small.PValue <= 0 || small.PValue > 1 {
		t.Errorf("exact p-value out of range: %v", small.PValue)
	}
	if large.PValue <= 0 || large.PValue > 1 {
		t.Errorf("chi-square p-value out of range: %v", large.PValue)
	}
}

// TestPairedSummary_AggregateDeltaPreserved: M3 is ADDITIVE. The aggregate
// delta that has always been reported must survive unchanged, or the existing
// microrag_ab.jsonl trend becomes incomparable across the schema change.
func TestPairedSummary_AggregateDeltaPreserved(t *testing.T) {
	// 3 of 4 on, 2 of 4 off -> 75% vs 50% -> +25.0pp
	on := []*BenchmarkResult{pass("a", 1), pass("b", 1), pass("c", 1), fail("d", 1)}
	off := []*BenchmarkResult{pass("a", 1), pass("b", 1), fail("c", 1), fail("d", 1)}

	p := PairArms(on, off)

	if math.Abs(p.DeltaPP-25.0) > 1e-9 {
		t.Errorf("DeltaPP = %v, want +25.0", p.DeltaPP)
	}
	if p.OnPass != 3 || p.OffPass != 2 || p.OnTotal != 4 || p.OffTotal != 4 {
		t.Errorf("aggregate counts wrong: on=%d/%d off=%d/%d", p.OnPass, p.OnTotal, p.OffPass, p.OffTotal)
	}
}
