package eval_harness

import (
	"errors"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
)

// The 2026-10-27/28 rig kills: a sibling eval agent's `pkill -f "ailang run"`
// SIGTERMed every agent CLI whose argv carried the teaching prompt. claude rows
// were banked api_error, codex rows were GRADED as compile_error. Both must now
// bank as external_kill, which is invalid (retried), never a model outcome.
func TestCategorizeAgentError_ExternalKill(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		finishReason string
		want         string
	}{
		{"structured finish reason (claude, codex)", errors.New("anything"), executor.FinishExternalKill, ErrorCategoryExternalKill},
		{"structured finish reason, graded path", nil, executor.FinishExternalKill, ErrorCategoryExternalKill},
		// The exact banked string from the 2026-10-28 baseline.
		{"claude exit 143 string", errors.New(`execution failed: executor "claude" failed for model "claude-haiku-4-5-20251001": exit status 143`), "error", ErrorCategoryExternalKill},
		// The rotation's opencode on the same seconds.
		{"opencode signal: terminated", errors.New(`executor "opencode" failed for model "ollama-rig/qwen3.8:27b-mxfp8": opencode exited with error: signal: terminated`), "", ErrorCategoryExternalKill},
		// Outranks a passing "timeout" mention in a stderr tail.
		{"signal outranks a timeout word in stderr", errors.New("opencode exited with error: signal: terminated\nstderr: request timeout retrying"), "", ErrorCategoryExternalKill},
		// Controls: the harness's own kills are SIGKILL and stay what they were.
		{"harness SIGKILL is not external", errors.New(`executor "motoko" failed: motoko exceeded its wall-clock bound (1h0m0s): timeout — signal: killed`), "", ErrorCategoryTimeout},
		{"timeout finish reason wins", errors.New("exit status 143"), executor.FinishTimeout, ErrorCategoryTimeout},
		{"cost kill finish reason wins", errors.New("exit status 143"), "cost_exhausted", ErrorCategoryCostKilled},
		{"plain non-zero exit stays api_error", errors.New(`executor "claude" failed: exit status 1`), "error", ErrorCategoryAPI},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CategorizeAgentError(tt.err, tt.finishReason); got != tt.want {
				t.Errorf("CategorizeAgentError(%v, %q) = %q, want %q", tt.err, tt.finishReason, got, tt.want)
			}
		})
	}
}

func TestValidityBackstop_ExternalKillIsNotAMeasurement(t *testing.T) {
	m := RunMetrics{ErrorCategory: ErrorCategoryExternalKill, Stderr: "API Error: execution failed: exit status 143"}
	m.applyValidityBackstop()
	if m.IsValid() {
		t.Fatal("an external_kill row counted as a valid measurement — it would be charged to the model and never retried")
	}
	if m.InvalidReason() != ReasonHarnessError {
		t.Errorf("InvalidReason() = %q, want %q", m.InvalidReason(), ReasonHarnessError)
	}
}

// The codex half of the bug lived HERE: an interrupted codex reported
// Success=false with an EMPTY Error, which passes the executor-crash gate, so
// the runner graded the half-written workspace and banked compile_error. With
// the executor's external_kill Error set, the run must come back as a failure
// to bank — never a grade.
func TestRunAgentBenchmarkWithExecutor_ExternalKillIsNotGraded(t *testing.T) {
	fake := &costFakeExecutor{name: "external-kill-fake", result: executor.Result{
		Success:       false,
		Error:         "external kill: codex exited 0 mid-turn (turn.started, no turn.completed or turn.failed) — the trace a SIGTERM leaves",
		FinishReason:  executor.FinishExternalKill,
		NumTurns:      1,
		ToolCallCount: 10,
	}}
	executor.GlobalFactory().Register(fake.name, func(*executor.Config) (executor.Executor, error) {
		return fake, nil
	})
	cfg := MultiExecutorConfig{ExecutorName: fake.name, ModelName: "gpt6-1-sol"}
	cfg.WorkspaceDir = t.TempDir()

	out, err := RunAgentBenchmarkWithExecutor(&BenchmarkSpec{ID: "cp_60153", Description: "fixture"}, cfg, "ailang")
	if err == nil {
		t.Fatalf("an externally killed run was graded (compile_ok=%v) instead of returned as a failure", out.CompileOk)
	}
	if out == nil || out.FinishReason != executor.FinishExternalKill {
		t.Fatalf("diagnostics row lost the finish reason: %+v", out)
	}
	if got := CategorizeAgentError(err, out.FinishReason); got != ErrorCategoryExternalKill {
		t.Errorf("banked category = %q, want %q", got, ErrorCategoryExternalKill)
	}
}
