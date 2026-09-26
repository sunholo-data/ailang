package eval_harness

import (
	"context"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/ai"
)

// fixedRunner is a LanguageRunner that returns one canned RunResult, so the
// standard-mode grading path can be driven without a toolchain.
type fixedRunner struct{ result RunResult }

func (r *fixedRunner) Run(string, time.Duration) (*RunResult, error) {
	out := r.result
	return &out, nil
}
func (r *fixedRunner) Language() string { return "python" }

// TestD9StandardModeStdoutOkGatedOnRuntime pins ruling D9 (M-V1-SIMPLIFY-S4
// M4, bank-forward from 2026-09-15): a standard-mode row whose run FAILED at
// runtime but whose stdout matches the expectation banks stdout_ok=false —
// the same gate every agent-mode lane already applied. Before D9 the flag was
// the bare stdout comparison, so the stored flags meant different things in
// the two modes (the read-side Passed predicate hid it: D2).
func TestD9StandardModeStdoutOkGatedOnRuntime(t *testing.T) {
	spec := &BenchmarkSpec{ID: "d9", ExpectedOut: "42\n", Languages: []string{"python"}}
	agent, _ := newTestAgent(&recordingProvider{}, ai.ProviderOpenAI)

	cases := []struct {
		name      string
		run       RunResult
		wantStdok bool
	}{
		{"runtime failure with matching stdout", RunResult{CompileOk: true, RuntimeOk: false, ExitCode: 1, Stdout: "42\n"}, false},
		{"runtime ok with matching stdout", RunResult{CompileOk: true, RuntimeOk: true, Stdout: "42\n"}, true},
		{"runtime ok with wrong stdout", RunResult{CompileOk: true, RuntimeOk: true, Stdout: "41\n"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := NewRepairRunner(agent, &fixedRunner{result: tc.run}, spec, time.Second, false)
			attempt, err := rr.runSingleAttempt(context.Background(), "print(42)")
			if err != nil {
				t.Fatalf("runSingleAttempt: %v", err)
			}
			if attempt.StdoutOk != tc.wantStdok {
				t.Fatalf("attempt.StdoutOk = %v, want %v (runtime_ok=%v, stdout=%q)", attempt.StdoutOk, tc.wantStdok, tc.run.RuntimeOk, tc.run.Stdout)
			}
			// What gets BANKED: the RunMetrics row populated from the attempt.
			var metrics RunMetrics
			rr.populateMetrics(&metrics, attempt)
			if metrics.StdoutOk != tc.wantStdok || metrics.RuntimeOk != tc.run.RuntimeOk {
				t.Fatalf("banked stdout_ok=%v runtime_ok=%v, want stdout_ok=%v runtime_ok=%v", metrics.StdoutOk, metrics.RuntimeOk, tc.wantStdok, tc.run.RuntimeOk)
			}
			if metrics.Passed() != (tc.run.RuntimeOk && tc.wantStdok) {
				t.Fatalf("Passed() = %v disagrees with the flags", metrics.Passed())
			}
		})
	}
}

// TestD9ExpectedEmptyStdoutDoesNotPassACrash is the case D2 measured (30 of
// its 31 discordant rows): an empty expected stdout used to grade a crash as
// stdout_ok because "" == "". The runtime gate closes it on the write side.
func TestD9ExpectedEmptyStdoutDoesNotPassACrash(t *testing.T) {
	spec := &BenchmarkSpec{ID: "d9-empty", ExpectedOut: "", Languages: []string{"python"}}
	agent, _ := newTestAgent(&recordingProvider{}, ai.ProviderOpenAI)
	rr := NewRepairRunner(agent, &fixedRunner{result: RunResult{CompileOk: false, RuntimeOk: false, ExitCode: 1, Stderr: "parse error"}}, spec, time.Second, false)
	attempt, err := rr.runSingleAttempt(context.Background(), "x")
	if err != nil {
		t.Fatal(err)
	}
	if attempt.StdoutOk {
		t.Fatal("a compile/runtime failure with empty expected stdout banked stdout_ok=true")
	}
}
