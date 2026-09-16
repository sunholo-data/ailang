package eval_harness

import (
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
