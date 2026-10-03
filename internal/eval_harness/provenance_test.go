package eval_harness

import (
	"errors"
	"reflect"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
)

// A failed or diagnostic row keeps its provenance (M-PI-HARNESS-UPGRADE M1,
// M-AGENT-AILANG-ONLY-EXECUTION M1): the first A/B row banked on 2026-09-16
// was an executor failure with executor_version and tool_policy both null,
// because the failure builders started from a blank row.
func TestWithProvenance_FailedRowKeepsHarnessAndLane(t *testing.T) {
	res := &executor.Result{Success: false, Error: "pi idle for 3m", ExecutorVersion: "pi@0.85.1", ToolPolicy: []string{"Read", "AilangRun"}, PolicyDigest: "d1"}
	row := withProvenance(&AgentBenchmarkResult{BenchmarkID: "x", Executor: "pi"}, res)
	if row.ExecutorVersion != "pi@0.85.1" || row.PolicyDigest != "d1" || !reflect.DeepEqual(row.ToolPolicy, []string{"Read", "AilangRun"}) {
		t.Fatalf("provenance dropped on a failed row: %+v", row)
	}
	if got := withProvenance(&AgentBenchmarkResult{BenchmarkID: "x"}, nil); got.ExecutorVersion != "" {
		t.Fatalf("nil result must leave provenance absent (unmeasured), got %q", got.ExecutorVersion)
	}
}

// A failed row keeps the executor's typed finish reason, and the categoriser
// reads it ahead of the error string. pi reports a token WORK-gate kill as
// FinishThrashAborted; before the reason rode along, the caller saw only
// "token budget exceeded ..." and banked api_error (7 rotation rows, 2026-10-02).
func TestWithProvenance_FailedRowKeepsFinishReasonForCategoriser(t *testing.T) {
	res := &executor.Result{Success: false, FinishReason: executor.FinishThrashAborted,
		Error: "token budget exceeded (3072394 > 3000000) — pi exited with error: signal: killed"}
	row := withProvenance(&AgentBenchmarkResult{BenchmarkID: "x", Executor: "pi"}, res)
	if row.FinishReason != executor.FinishThrashAborted {
		t.Fatalf("finish reason dropped on a failed row: %q", row.FinishReason)
	}
	err := errors.New(`executor "pi" failed for model "ollama-rig/qwen3.8:27b-mxfp8": ` + res.Error)
	if got := CategorizeAgentError(err, row.FinishReason); got != ErrorCategoryThrashAborted {
		t.Fatalf("CategorizeAgentError = %q, want %q", got, ErrorCategoryThrashAborted)
	}
}
