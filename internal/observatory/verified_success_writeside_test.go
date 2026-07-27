package observatory

import "testing"

// This file closes the loop on BF-1 from the READ side.
//
// M4a-3 moved agent-mode contract verification onto the live multi-executor path
// so an agent run can finally bank verify_verified > 0. That is only useful if
// the WRITE side's field values actually satisfy the M1 verified-success
// predicate. These tests assert the contract at the exact shape
// cmd/ailang/eval_benchmark.go builds an EvalAssessment in, so writer and reader
// cannot drift apart silently — the failure mode that made the KPI return
// zero_denominator forever.

// agentAssessment mirrors the EvalAssessment that
// cmd/ailang/eval_benchmark.go assembles from an AgentBenchmarkResult on the live
// multi-executor path (executor + the 3 exec-grade flags + the 5 verify fields).
func agentAssessment(compile, runtime, stdout, verifyOk bool, verified, counterex, skipped, errs int) *EvalAssessment {
	return &EvalAssessment{
		BenchmarkID:     "contract_leap_year",
		Model:           "claude-haiku-4-5",
		Language:        "ailang",
		EvalMode:        "agent",
		Executor:        "claude",
		CompileOk:       compile,
		RuntimeOk:       runtime,
		StdoutOk:        stdout,
		VerifyOk:        verifyOk,
		VerifyVerified:  verified,
		VerifyCounterex: counterex,
		VerifySkipped:   skipped,
		VerifyErrors:    errs,
	}
}

// TestAgentVerifiedResultSatisfiesVerifiedSuccess: a passing, contract-verified
// agent run IS a verified success. Before M4a-3 this state was unreachable in
// practice because nothing on the live agent path ever set the verify fields.
func TestAgentVerifiedResultSatisfiesVerifiedSuccess(t *testing.T) {
	a := agentAssessment(true, true, true, true, 3, 0, 0, 0)
	if !isVerifiedSuccess(a) {
		t.Errorf("a passing, verified agent assessment is not a verified success: %+v", a)
	}
	if !isPass(a) {
		t.Error("expected isPass to also hold")
	}
	if isVerificationFailure(a) {
		t.Error("a fully verified run must not be a verification failure")
	}
}

// TestAgentUnverifiedResultIsAnUnverifiedPass is the pre-M4a-3 reality, retained
// as the negative control: an all-zero verify block on a passing run is an
// UNVERIFIED pass, never a verified success. This is exactly why the cohort
// returned zero_denominator.
func TestAgentUnverifiedResultIsAnUnverifiedPass(t *testing.T) {
	a := agentAssessment(true, true, true, false, 0, 0, 0, 0)
	if isVerifiedSuccess(a) {
		t.Error("an all-zero verify block must never count as a verified success")
	}
	if !isPass(a) {
		t.Error("it is still an execution-grade pass")
	}
	if isVerificationFailure(a) {
		t.Error("no verification attempted is verification_missing, not a failure")
	}
}

// TestAgentVerifyEdgeCasesAreNotVerifiedSuccesses pins every partial-evidence
// shape the moved verifier can produce.
func TestAgentVerifyEdgeCasesAreNotVerifiedSuccesses(t *testing.T) {
	cases := []struct {
		name string
		a    *EvalAssessment
	}{
		{"verified but counterexample", agentAssessment(true, true, true, false, 2, 1, 0, 0)},
		{"verified but skipped obligation", agentAssessment(true, true, true, true, 2, 0, 1, 0)},
		{"verified but z3 error", agentAssessment(true, true, true, false, 2, 0, 0, 1)},
		{"verify_ok without any verified function", agentAssessment(true, true, true, true, 0, 0, 0, 0)},
		{"verified but wrong stdout", agentAssessment(true, true, false, true, 3, 0, 0, 0)},
		{"verified but did not compile", agentAssessment(false, false, false, true, 3, 0, 0, 0)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if isVerifiedSuccess(c.a) {
				t.Errorf("%s must not be a verified success: %+v", c.name, c.a)
			}
		})
	}
}
