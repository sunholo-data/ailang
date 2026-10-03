package eval_harness

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
)

// A run that used up its step budget is graded and costed, not discarded as a
// crash: it did real work for its whole budget (2026-10-01 lane A/B banked three
// such rows at $0 with no grading, one of them holding a 10 KB solution).
func TestStepBudgetExhausted(t *testing.T) {
	for _, tc := range []struct {
		name string
		res  *executor.Result
		want bool
	}{
		{"nil", nil, false},
		{"success", &executor.Result{Success: true}, false},
		{"finish_reason step_exhausted", &executor.Result{FinishReason: executor.FinishStepExhausted, Error: "x"}, true},
		{"motoko main error event", &executor.Result{Error: "motoko emitted error event without run_summary: step budget exhausted [motoko process: exit status 1]"}, true},
		{"crash", &executor.Result{Error: "motoko terminated without emitting run_summary (likely crash)"}, false},
		{"wall-clock timeout", &executor.Result{Error: "motoko exceeded its wall-clock bound (1h0m0s): timeout"}, false},
	} {
		if got := stepBudgetExhausted(tc.res); got != tc.want {
			t.Errorf("%s: stepBudgetExhausted = %v, want %v", tc.name, got, tc.want)
		}
	}
}
