package eval_analysis

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// counterbalancedPair builds one benchmark's two rows in the section 5.3
// schedule: ON leads on even indexes, OFF on odd.
func counterbalancedPair(t *testing.T, index int, onTokens, offTokens int) (on, off *BenchmarkResult) {
	t.Helper()
	id := "bench-" + string(rune('a'+index))
	base := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC).Add(time.Duration(index*4) * time.Second)
	on = &BenchmarkResult{ID: id, Lang: "ailang", Model: "fixture", Trial: 1,
		CompileOk: true, RuntimeOk: true, StdoutOk: true, TotalTokens: onTokens}
	off = &BenchmarkResult{ID: id, Lang: "ailang", Model: "fixture", Trial: 1,
		CompileOk: true, RuntimeOk: true, StdoutOk: true, TotalTokens: offTokens}
	if index%2 == 0 {
		on.Timestamp, off.Timestamp = base, base.Add(time.Second)
	} else {
		off.Timestamp, on.Timestamp = base, base.Add(time.Second)
	}
	return on, off
}

// TestD2InvalidOffRowNeverScores pins that a NON-MEASUREMENT row in the OFF arm
// is dropped from every statistic and counted, exactly as an ON one is.
//
// Killed mutation: passing the raw `off` slice to PairArms/offByKey instead of
// `validOff` (i.e. reverting partitionMeasurements for the OFF arm). That is
// how the milestone was first delivered, and it survived the whole package: the
// treatment gate only scans OFF rows for CONTAMINATION (FmtHookEvents), so an
// OFF row invalid for any other reason — harness_error, config_mismatch — sailed
// through and scored a full OFF win.
//
// The observable is deliberately the WIN TALLY and the quarantine count, not
// the verdict: every refusal branch also produces VOID, and this pair is
// INCONCLUSIVE under both arms, so the verdict cannot discriminate.
func TestD2InvalidOffRowNeverScores(t *testing.T) {
	build := func(markInvalid bool) ([]*BenchmarkResult, []*BenchmarkResult) {
		var on, off []*BenchmarkResult
		// b0: an exact tie. b1: OFF 10x cheaper, i.e. a decisive OFF win.
		o0, f0 := counterbalancedPair(t, 0, 1000, 1000)
		o1, f1 := counterbalancedPair(t, 1, 1000, 100)
		if markInvalid {
			f1.Validity = &eval_harness.Validity{
				Valid: false, Reason: "harness_error", Detail: "not a measurement",
			}
		}
		on = append(on, o0, o1)
		off = append(off, f0, f1)
		return on, off
	}

	// Control: with every row valid the decisive OFF win MUST be counted, or the
	// fixture cannot discriminate and the assertion below is vacuous.
	ctlOn, ctlOff := build(false)
	ctl := AnalyzeCensoredPairs(ctlOn, ctlOff)
	if ctl.OffWins != 1 || ctl.BothPassPairs != 2 {
		t.Fatalf("control failed: all-valid arms gave off_wins=%d both_pass=%d, want 1 and 2 — fixture does not discriminate",
			ctl.OffWins, ctl.BothPassPairs)
	}
	if ctl.OffQuarantined != 0 {
		t.Fatalf("control failed: all-valid arms reported off_quarantined=%d, want 0", ctl.OffQuarantined)
	}

	// Treatment: the same row, marked a non-measurement.
	gotOn, gotOff := build(true)
	got := AnalyzeCensoredPairs(gotOn, gotOff)
	if got.OffWins != 0 {
		t.Errorf("off_wins = %d, want 0 — an invalid OFF row scored a win", got.OffWins)
	}
	if got.BothPassPairs != 1 {
		t.Errorf("both_pass_pairs = %d, want 1 — an invalid OFF row polluted the token statistics", got.BothPassPairs)
	}
	if got.OffQuarantined != 1 {
		t.Errorf("off_quarantined = %d, want 1 — the dropped row must be COUNTED, not silently discarded", got.OffQuarantined)
	}
	if got.OffRows != 2 {
		t.Errorf("off_rows = %d, want 2 (the banked set, before filtering)", got.OffRows)
	}
}

// TestD2OrderRefusalRepeatedBenchmarkIsUnreachable establishes that
// order_integrity_repeated_benchmark is a DEFENSIVE invariant with no reachable
// input, and pins the reasoning rather than asserting the claim.
//
// Why it cannot fire: blocks are deduplicated by (benchmark, arm), and any
// non-contiguous repeat of one of those keys returns
// order_integrity_noncontiguous_block first. With exactly two arms a benchmark
// owns at most two distinct block keys, so for seenBenchmark to be true at a
// later pair the benchmark must contribute a THIRD block, which is necessarily a
// duplicate key. And a pair holding only ONE of its blocks fails the
// first.benchmark != second.benchmark check as nonadjacent_arms.
//
// The evaluator flagged this branch as the one refusal with no coverage: its
// if false && ... mutant survived the entire package. That is correct and the
// cause is unreachability, not a missing fixture — so it is DECLARED here and in
// the code rather than left as a guard nobody is protecting.
func TestD2OrderRefusalRepeatedBenchmarkIsUnreachable(t *testing.T) {
	at := func(sec int) time.Time {
		return time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC).Add(time.Duration(sec) * time.Second)
	}
	row := func(id string, sec int) *BenchmarkResult {
		return &BenchmarkResult{ID: id, Lang: "ailang", Model: "fixture", Trial: 1,
			CompileOk: true, RuntimeOk: true, StdoutOk: true, TotalTokens: 1000, Timestamp: at(sec)}
	}

	cases := []struct {
		name       string
		on, off    []*BenchmarkResult
		wantCaught string
	}{
		{
			// a:on a:off | b:off b:on | a:on a:off  -- `a` runs a second time in full.
			name:       "benchmark repeated as a whole pair",
			on:         []*BenchmarkResult{row("a", 0), row("b", 3), row("a", 4)},
			off:        []*BenchmarkResult{row("a", 1), row("b", 2), row("a", 5)},
			wantCaught: "order_integrity_noncontiguous_block",
		},
		{
			// a:on b:on | a:off b:off  -- one benchmark spread across two pairs.
			name:       "benchmark split across pairs",
			on:         []*BenchmarkResult{row("a", 0), row("b", 1)},
			off:        []*BenchmarkResult{row("a", 2), row("b", 3)},
			wantCaught: "order_integrity_nonadjacent_arms",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CheckFmtOrderIntegrity(tc.on, tc.off)
			if got == "order_integrity_repeated_benchmark" {
				t.Fatalf("branch is reachable after all — this test and the code comment are wrong, fix both")
			}
			if got != tc.wantCaught {
				t.Fatalf("reason = %q, want %q — the earlier branch that makes repeated_benchmark unreachable", got, tc.wantCaught)
			}
		})
	}

	// Control: a well-formed counterbalanced schedule must pass the gate, so the
	// refusals above are attributable to the shapes and not to the fixture style.
	on0, off0 := counterbalancedPair(t, 0, 1000, 1000)
	on1, off1 := counterbalancedPair(t, 1, 1000, 1000)
	if got := CheckFmtOrderIntegrity([]*BenchmarkResult{on0, on1}, []*BenchmarkResult{off0, off1}); got != "" {
		t.Fatalf("control failed: a valid counterbalanced schedule was refused with %q", got)
	}
}

// TestD2QuarantineRateBoundary pins the doc's STRICT ">20%": exactly 20% does
// not void, just over it does. A boundary nobody tests is a boundary nobody
// notices moving.
func TestD2QuarantineRateBoundary(t *testing.T) {
	arm := func(total, invalid int) []*BenchmarkResult {
		rows := make([]*BenchmarkResult, 0, total)
		for i := 0; i < total; i++ {
			r := &BenchmarkResult{ID: "b", Lang: "ailang", Model: "fixture", Trial: i}
			if i < invalid {
				r.Validity = &eval_harness.Validity{Valid: false, Reason: "treatment_unproven"}
			}
			rows = append(rows, r)
		}
		return rows
	}
	if got := treatmentIntegrityReason(arm(5, 1), nil); got != "" {
		t.Errorf("1 of 5 = exactly 20%%: reason = %q, want no refusal (the doc says strictly >20%%)", got)
	}
	if got := treatmentIntegrityReason(arm(4, 1), nil); got != "treatment_unproven_rate" {
		t.Errorf("1 of 4 = 25%%: reason = %q, want treatment_unproven_rate", got)
	}
}

// TestD2VerdictMatrixFixtureIsNonEmpty makes an emptied or truncated
// testdata/censored_cases.json FAIL LOUDLY. Without it, `[]` yields a green
// TestCensoredVerdictMatrix that ran zero subtests and asserted nothing — the
// vacuous pass this milestone exists to prevent, aimed at its own fixture.
func TestD2VerdictMatrixFixtureIsNonEmpty(t *testing.T) {
	data, err := os.ReadFile("testdata/censored_cases.json")
	if err != nil {
		t.Fatalf("instrument failure: cannot read the verdict-matrix fixture: %v", err)
	}
	var cases []censoredFixture
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatalf("instrument failure: fixture does not parse: %v", err)
	}
	if len(cases) < 4 {
		t.Fatalf("verdict-matrix fixture has %d cases, want >= 4 — the matrix must cover VOID, KEEP, RETIRE and INCONCLUSIVE", len(cases))
	}
	seen := map[string]bool{}
	for _, c := range cases {
		seen[c.WantVerdict] = true
	}
	for _, want := range []string{"KEEP", "RETIRE", "INCONCLUSIVE"} {
		if !seen[want] {
			t.Errorf("verdict-matrix fixture covers no %s case", want)
		}
	}
}
