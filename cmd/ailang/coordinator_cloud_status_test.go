package main

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// The executor must never publish a status the coordinator drops.
// MU: return (status, errMsg) unconditionally and the unknown case fails.
func TestContractCompletionStatus(t *testing.T) {
	for _, st := range coordinator.ExecutorCompletionStatuses() {
		if got, _ := contractCompletionStatus(string(st), ""); got != string(st) {
			t.Errorf("contract status %q rewritten to %q", st, got)
		}
	}
	got, msg := contractCompletionStatus("sort-of-done", "x")
	if got != string(coordinator.TaskStatusFailed) || !strings.Contains(msg, "sort-of-done") {
		t.Errorf("unknown status published as %q (%q), want failed naming the defect", got, msg)
	}
}
