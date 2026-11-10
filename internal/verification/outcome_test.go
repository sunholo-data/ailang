package verification

import "testing"

// TestExitCode_Table pins the shared exit contract (M-V1-VERIFICATION-HONESTY
// M1, D1) over every outcome and both policy flags. Each row is one lane of
// the table in outcome.go.
//
// Red mutations, each measured: drop the `unavailable` arm of Classify (the
// "solver unavailable" rows fail: the run classifies as no_obligations, exit 0
// — the original bug); make OutcomeInconclusive return 0 (the inconclusive
// rows fail); make Strict ignore partial (the strict partial row fails); drop
// the `unproved` arm (skipped-only classifies as partial, exit 0).
func TestExitCode_Table(t *testing.T) {
	ok := func(f func(*Tally)) Tally {
		t := Tally{CheckPassed: true, SolverAvailable: true}
		f(&t)
		return t
	}
	strict := ExitPolicy{Strict: true}
	allow := ExitPolicy{AllowInconclusive: true}
	tests := []struct {
		name   string
		tally  Tally
		policy ExitPolicy
		want   Outcome
		code   int
	}{
		{"check failed", Tally{CheckPassed: false, SolverAvailable: true}, ExitPolicy{}, OutcomeNotRun, 1},
		{"check failed outranks a missing solver", Tally{CheckPassed: false, Obligations: 3}, ExitPolicy{}, OutcomeNotRun, 1},
		{"no obligations", ok(func(*Tally) {}), ExitPolicy{}, OutcomeNoObligations, 0},
		{"no obligations, no solver", Tally{CheckPassed: true}, ExitPolicy{}, OutcomeNoObligations, 0},
		{"no obligations, no solver, strict", Tally{CheckPassed: true}, strict, OutcomeNoObligations, 0},
		{"solver unavailable", Tally{CheckPassed: true, Obligations: 1}, ExitPolicy{}, OutcomeUnavailable, 1},
		{"solver unavailable, allow-inconclusive does not help", Tally{CheckPassed: true, Obligations: 2}, allow, OutcomeUnavailable, 1},
		{"verified", ok(func(t *Tally) { t.Obligations, t.Verified = 2, 2 }), ExitPolicy{}, OutcomeVerified, 0},
		{"verified, strict", ok(func(t *Tally) { t.Obligations, t.Verified = 2, 2 }), strict, OutcomeVerified, 0},
		{"partial", ok(func(t *Tally) { t.Obligations, t.Verified, t.Skipped = 2, 1, 1 }), ExitPolicy{}, OutcomePartial, 0},
		{"partial, strict", ok(func(t *Tally) { t.Obligations, t.Verified, t.Skipped = 2, 1, 1 }), strict, OutcomePartial, 1},
		// Review fix 1: obligations exist and none was verified — nothing was
		// proved, so it fails whatever the policy.
		{"skipped only is unproved", ok(func(t *Tally) { t.Obligations, t.Skipped = 1, 1 }), ExitPolicy{}, OutcomeUnproved, 1},
		{"skipped only is unproved, allow-inconclusive", ok(func(t *Tally) { t.Obligations, t.Skipped = 2, 2 }), allow, OutcomeUnproved, 1},
		{"inconclusive + skipped, nothing verified, allowed: still unproved",
			ok(func(t *Tally) { t.Obligations, t.Inconclusive, t.Skipped = 2, 1, 1 }), allow, OutcomeInconclusive, 1},
		{"inconclusive + skipped + verified, allowed: passes like partial",
			ok(func(t *Tally) { t.Obligations, t.Inconclusive, t.Skipped, t.Verified = 3, 1, 1, 1 }), allow, OutcomeInconclusive, 0},
		{"inconclusive", ok(func(t *Tally) { t.Obligations, t.Inconclusive = 1, 1 }), ExitPolicy{}, OutcomeInconclusive, 1},
		{"inconclusive, allowed", ok(func(t *Tally) { t.Obligations, t.Inconclusive = 1, 1 }), allow, OutcomeInconclusive, 0},
		{"inconclusive, allowed but strict", ok(func(t *Tally) { t.Obligations, t.Inconclusive = 1, 1 }),
			ExitPolicy{Strict: true, AllowInconclusive: true}, OutcomeInconclusive, 1},
		{"inconclusive outranks skipped", ok(func(t *Tally) { t.Obligations, t.Inconclusive, t.Skipped = 2, 1, 1 }), ExitPolicy{}, OutcomeInconclusive, 1},
		{"counterexample", ok(func(t *Tally) { t.Obligations, t.Counterexample = 1, 1 }), ExitPolicy{}, OutcomeCounterexample, 1},
		{"counterexample is not excused by allow-inconclusive",
			ok(func(t *Tally) { t.Obligations, t.Counterexample, t.Inconclusive = 2, 1, 1 }), allow, OutcomeCounterexample, 1},
		{"error", ok(func(t *Tally) { t.Obligations, t.Errors = 1, 1 }), ExitPolicy{}, OutcomeError, 1},
		{"error outranks inconclusive", ok(func(t *Tally) { t.Obligations, t.Errors, t.Inconclusive = 2, 1, 1 }), allow, OutcomeError, 1},
		{"counterexample outranks error", ok(func(t *Tally) { t.Obligations, t.Errors, t.Counterexample = 2, 1, 1 }), ExitPolicy{}, OutcomeCounterexample, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Classify(tt.tally); got != tt.want {
				t.Fatalf("Classify(%+v) = %q, want %q", tt.tally, got, tt.want)
			}
			if got := ExitCode(tt.tally, tt.policy); got != tt.code {
				t.Fatalf("ExitCode(%+v, %+v) = %d, want %d", tt.tally, tt.policy, got, tt.code)
			}
		})
	}
}
