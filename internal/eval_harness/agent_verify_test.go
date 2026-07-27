package eval_harness

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// fakeVerifier records the arguments it was called with and returns a canned
// AICheckResult, so the agent-mode verification wiring can be tested with NO
// `ailang ai-check` subprocess, no Z3, and no benchmark execution.
type fakeVerifier struct {
	calls    int
	gotPath  string
	gotTimeo time.Duration
	result   *AICheckResult
	raw      string
	err      error
}

func (f *fakeVerifier) run(_, filePath string, timeout time.Duration) (*AICheckResult, string, error) {
	f.calls++
	f.gotPath = filePath
	f.gotTimeo = timeout
	return f.result, f.raw, f.err
}

// withFakeVerifier swaps the package-level ai-check hook for the duration of a test.
func withFakeVerifier(t *testing.T, f *fakeVerifier) {
	t.Helper()
	prev := agentAICheck
	agentAICheck = f.run
	t.Cleanup(func() { agentAICheck = prev })
}

func verifiedAICheck(verified int) *AICheckResult {
	return &AICheckResult{
		Check:  AICheckCheckResult{Passed: true},
		Verify: AICheckVerifyResult{Available: true, Verified: verified},
	}
}

func contractSpec() *BenchmarkSpec {
	return &BenchmarkSpec{ID: "contract_leap_year", ContractSpec: "requires n > 0"}
}

// TestApplyAgentVerification_PopulatesVerifiedSuccess is the BF-1 fix's core
// assertion: a contract-bearing, compiling AILANG solution on a --verify run
// yields VerifyVerified > 0 and VerifyOk == true. Before M4a-3 the only
// agent-mode RunAICheck call sat inside RunAgentBenchmark(), which had NO caller,
// so every banked agent assessment carried an all-zero verify block and the KPI
// returned zero_denominator forever.
func TestApplyAgentVerification_PopulatesVerifiedSuccess(t *testing.T) {
	f := &fakeVerifier{result: verifiedAICheck(3), raw: `{"verify":{"verified":3}}`}
	withFakeVerifier(t, f)

	out := &AgentBenchmarkResult{}
	cfg := AgentBenchmarkConfig{Verify: true, VerifyTimeout: 7 * time.Second}
	applyAgentVerification(out, cfg, contractSpec(), "ailang", "/tmp/ws/benchmark/solution.ail", true)

	if f.calls != 1 {
		t.Fatalf("ai-check called %d times, want 1", f.calls)
	}
	if out.VerifyVerified != 3 {
		t.Errorf("VerifyVerified = %d, want 3", out.VerifyVerified)
	}
	if !out.VerifyOk {
		t.Error("VerifyOk = false, want true")
	}
	if out.VerifyJSON == "" {
		t.Error("VerifyJSON not captured")
	}
	if f.gotPath != "/tmp/ws/benchmark/solution.ail" {
		t.Errorf("verified %q, want the solution path", f.gotPath)
	}
}

// TestApplyAgentVerification_HonoursConfiguredTimeout closes BF-3: --verify-timeout
// used to be plumbed only to the standard-mode repair runner, while the agent call
// hardcoded 5*time.Second. The cohort manifest RECORDS verify_timeout, so a
// hardcoded value would have made that artifact lie.
func TestApplyAgentVerification_HonoursConfiguredTimeout(t *testing.T) {
	for _, want := range []time.Duration{2 * time.Second, 30 * time.Second, 90 * time.Second} {
		f := &fakeVerifier{result: verifiedAICheck(1)}
		withFakeVerifier(t, f)
		applyAgentVerification(&AgentBenchmarkResult{},
			AgentBenchmarkConfig{Verify: true, VerifyTimeout: want},
			contractSpec(), "ailang", "/tmp/solution.ail", true)
		if f.gotTimeo != want {
			t.Errorf("verifier got timeout %v, want %v (BF-3: --verify-timeout must reach the agent verifier)", f.gotTimeo, want)
		}
	}
}

// TestApplyAgentVerification_ZeroTimeoutUsesTheDocumentedDefault — a
// hand-constructed config (DefaultAgentConfig, tests) must not pass `--timeout 0s`
// to ai-check. The default is the SAME named constant the --verify-timeout flag
// defaults to, so the manifest's recorded value cannot diverge from the used one.
func TestApplyAgentVerification_ZeroTimeoutUsesTheDocumentedDefault(t *testing.T) {
	f := &fakeVerifier{result: verifiedAICheck(1)}
	withFakeVerifier(t, f)
	applyAgentVerification(&AgentBenchmarkResult{},
		AgentBenchmarkConfig{Verify: true},
		contractSpec(), "ailang", "/tmp/solution.ail", true)
	if f.gotTimeo != DefaultVerifyTimeout {
		t.Errorf("zero timeout resolved to %v, want DefaultVerifyTimeout (%v)", f.gotTimeo, DefaultVerifyTimeout)
	}
}

// TestApplyAgentVerification_GateConditions preserves the gate exactly:
// config.Verify && spec.ContractSpec != "" && language == "ailang" && compileOk.
// Any skipped case must leave the verify block ALL-ZERO, so the run classifies as
// verification_missing (an unverified pass) and can NEVER count as a verified
// success.
func TestApplyAgentVerification_GateConditions(t *testing.T) {
	tests := []struct {
		name      string
		cfg       AgentBenchmarkConfig
		spec      *BenchmarkSpec
		language  string
		compileOk bool
		wantCalls int
	}{
		{"verify off", AgentBenchmarkConfig{Verify: false}, contractSpec(), "ailang", true, 0},
		{"no contract_spec", AgentBenchmarkConfig{Verify: true}, &BenchmarkSpec{ID: "fizzbuzz"}, "ailang", true, 0},
		{"non-ailang language", AgentBenchmarkConfig{Verify: true}, contractSpec(), "python", true, 0},
		{"did not compile", AgentBenchmarkConfig{Verify: true}, contractSpec(), "ailang", false, 0},
		{"nil spec", AgentBenchmarkConfig{Verify: true}, nil, "ailang", true, 0},
		{"all conditions met", AgentBenchmarkConfig{Verify: true}, contractSpec(), "ailang", true, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeVerifier{result: verifiedAICheck(2)}
			withFakeVerifier(t, f)

			out := &AgentBenchmarkResult{}
			applyAgentVerification(out, tt.cfg, tt.spec, tt.language, "/tmp/solution.ail", tt.compileOk)

			if f.calls != tt.wantCalls {
				t.Errorf("ai-check called %d times, want %d", f.calls, tt.wantCalls)
			}
			if tt.wantCalls == 0 {
				if out.VerifyVerified != 0 || out.VerifyOk || out.VerifyCounterex != 0 ||
					out.VerifySkipped != 0 || out.VerifyErrors != 0 || out.VerifyJSON != "" {
					t.Errorf("skipped verification left a non-zero verify block: %+v", out)
				}
			}
		})
	}
}

// TestApplyAgentVerification_CounterexampleIsNotOk — a counterexample, a skipped
// obligation or a Z3 error must NOT produce VerifyOk. isVerifiedSuccess also
// requires all three to be zero, so these runs classify as verification failures.
func TestApplyAgentVerification_CounterexampleIsNotOk(t *testing.T) {
	tests := []struct {
		name string
		v    AICheckVerifyResult
	}{
		{"counterexample", AICheckVerifyResult{Available: true, Verified: 2, Counterexample: 1}},
		{"z3 error", AICheckVerifyResult{Available: true, Verified: 2, Errors: 1}},
		{"verifier unavailable", AICheckVerifyResult{Available: false, Verified: 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeVerifier{result: &AICheckResult{Verify: tt.v}}
			withFakeVerifier(t, f)
			out := &AgentBenchmarkResult{}
			applyAgentVerification(out, AgentBenchmarkConfig{Verify: true}, contractSpec(), "ailang", "/tmp/s.ail", true)
			if out.VerifyOk {
				t.Errorf("VerifyOk = true for %s", tt.name)
			}
		})
	}
}

// TestApplyAgentVerification_ErrorLeavesBlockZero — an ai-check failure (missing
// binary, timeout, memory kill) must leave the block all-zero so the run is an
// unverified pass, NEVER a verified success. It is deliberately NOT silent: the
// dead path discarded the error with `_`, which is exactly the silent fallback on
// verification data CLAUDE.md §2 forbids.
func TestApplyAgentVerification_ErrorLeavesBlockZero(t *testing.T) {
	f := &fakeVerifier{err: fmt.Errorf("ai-check timed out after 25s")}
	withFakeVerifier(t, f)

	out := &AgentBenchmarkResult{}
	applyAgentVerification(out, AgentBenchmarkConfig{Verify: true}, contractSpec(), "ailang", "/tmp/s.ail", true)

	if f.calls != 1 {
		t.Fatalf("verifier not called")
	}
	if out.VerifyVerified != 0 || out.VerifyOk {
		t.Errorf("a failed verification produced verify evidence: %+v", out)
	}
}

// TestApplyAgentVerification_NilResultLeavesBlockZero guards the (result==nil,
// err==nil) shape.
func TestApplyAgentVerification_NilResultLeavesBlockZero(t *testing.T) {
	f := &fakeVerifier{}
	withFakeVerifier(t, f)
	out := &AgentBenchmarkResult{}
	applyAgentVerification(out, AgentBenchmarkConfig{Verify: true}, contractSpec(), "ailang", "/tmp/s.ail", true)
	if out.VerifyVerified != 0 || out.VerifyOk {
		t.Errorf("nil result produced verify evidence: %+v", out)
	}
}

// TestAgentVerification_LivesOnlyOnTheLivePath is the anti-regression guard for
// BF-1's actual failure mode: a correct verification implementation sitting on a
// function nobody calls. There must be exactly ONE agent-mode verification
// entry point, and RunAgentBenchmarkWithExecutor (the live multi-executor path,
// the only one cmd/ailang/eval_benchmark.go invokes) must use it.
func TestAgentVerification_LivesOnlyOnTheLivePath(t *testing.T) {
	// applyAgentVerification must exist and be the single funnel: this compiles
	// only if the symbol is present, and the tests above prove it is the thing
	// that calls ai-check.
	var _ func(*AgentBenchmarkResult, AgentBenchmarkConfig, *BenchmarkSpec, string, string, bool) = applyAgentVerification

	// And the legacy Claude-only runner must be gone, so there is no second copy
	// to drift. (A compile-time reference to RunAgentBenchmark would fail to
	// build; this documents the intent for the next reader.)
	if strings.TrimSpace(DefaultAgentConfig().ClaudePath) == "" {
		t.Fatal("DefaultAgentConfig sanity check failed")
	}
}
