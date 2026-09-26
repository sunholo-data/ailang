package pi

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
)

// Both flags or neither. --no-extensions alone was measured to disarm the repo's
// session-protocol gate AND lose the sprint-evaluator skill, because it also disables the
// globally installed workspace-trust extension that grants project trust headlessly.
// Emitting only the first would turn a visible deadlock into a silently unqualified judge.
func TestIsolationArgs_BothFlagsOrNeither(t *testing.T) {
	if got := isolationArgs(&executor.Task{IsolateFromProjectExtensions: true}); strings.Join(got, " ") != "--no-extensions --approve" {
		t.Errorf("isolated task got %v, want both flags in order", got)
	}
	if got := isolationArgs(&executor.Task{}); got != nil {
		t.Errorf("non-isolated task got %v, want nil", got)
	}
	if got := isolationArgs(nil); got != nil {
		t.Errorf("nil task got %v, want nil", got)
	}
	// --no-skills must never appear on this path: the evaluator's rubric is its terminator.
	for _, a := range isolationArgs(&executor.Task{IsolateFromProjectExtensions: true}) {
		if a == "--no-skills" {
			t.Fatal("--no-skills on the isolation path would remove the judge's rubric")
		}
	}
}

// pi SUMS per-turn deltas, so its cap sees cumulative work. Cache writes are usually 0 on
// OpenRouter, which is why pi's number barely moved when the canonical quantity landed —
// the change was for consistency, not to alter pi's behaviour.
func TestCapExceeded_SumsDeltasAndCountsCacheWrites(t *testing.T) {
	// Real measurement, 2026-09-14: the same five-file read on pi-or-minimax-m3.
	if tp, over := capExceeded(30000, 35992, 0, 311); !over || tp != 36303 {
		t.Errorf("got (%d, %v), want (36303, true)", tp, over)
	}
	// Unchanged from Input+Output when the provider reports no cache writes.
	if tp, _ := capExceeded(100000, 35992, 0, 311); tp != 35992+311 {
		t.Errorf("with zero cache writes the total must equal Input+Output, got %d", tp)
	}
	// But a harness that DOES report writes must have them counted.
	if tp, _ := capExceeded(100000, 100, 5000, 10); tp != 5110 {
		t.Errorf("cache writes not counted: got %d, want 5110", tp)
	}
	for _, noCap := range []int{0, -1} {
		if _, over := capExceeded(noCap, 1<<30, 1<<30, 1<<30); over {
			t.Errorf("maxTokens=%d must mean no cap", noCap)
		}
	}
}
