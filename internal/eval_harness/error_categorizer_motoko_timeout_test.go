package eval_harness

import (
	"errors"
	"testing"
)

// A motoko run SIGKILLed at its wall-clock bound must bank as a timeout (the
// model needed more time), not api_error (cause unknown). The executor prefixes
// exactly this wording (2026-09-28: quine killed at 1h after ~22k reasoning tokens).
func TestCategorizeAgentError_MotokoWallClockKillIsTimeout(t *testing.T) {
	err := errors.New(`executor "motoko" failed for model "x": motoko exceeded its wall-clock bound (1h0m0s): timeout — motoko terminated with finish_reason=tool_calls and no run_summary [motoko process: signal: killed]`)
	if got := CategorizeAgentError(err, ""); got != ErrorCategoryTimeout {
		t.Fatalf("got %q, want %q", got, ErrorCategoryTimeout)
	}
	// Without the prefix the same kill is unexplained: keep it api_error.
	bare := errors.New(`executor "motoko" failed for model "x": motoko terminated with finish_reason=tool_calls and no run_summary [motoko process: signal: killed]`)
	if got := CategorizeAgentError(bare, ""); got == ErrorCategoryTimeout {
		t.Fatalf("an unexplained kill must not read as a timeout")
	}
}
