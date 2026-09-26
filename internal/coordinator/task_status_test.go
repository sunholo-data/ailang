package coordinator

import "testing"

// M-COORDINATOR-EXECUTION-TRUST M2.
//
// A run that changed nothing reported `completed`, so "the executor exited 0"
// and "work landed" were the same number. Store-wide that produced 1,249
// completions, 16 of them `completed`, and 6 that ever changed a file.

// MU-8: the headline case.
func TestNoDiffTaskIsNotCompleted(t *testing.T) {
	got := ClassifyCompletionStatus(nil, false, true)
	if got != TaskStatusNoChanges {
		t.Fatalf("a task expected to change something and didn't must report %q, got %q",
			TaskStatusNoChanges, got)
	}
}

// MU-9 / Conflict Surface #4: the no-op path exists for a real caller. An
// acknowledge-only probe is SUPPOSED to change nothing and must stay clean —
// this is the arm that stops a naive "no diff = failure".
func TestAcknowledgeOnlyTaskStillCompletesCleanly(t *testing.T) {
	if got := ClassifyCompletionStatus(nil, false, false); got != TaskStatusCompleted {
		t.Fatalf("an acknowledge-only task must complete cleanly, got %q", got)
	}
}

func TestChangedFilesAlwaysComplete(t *testing.T) {
	for _, expect := range []bool{true, false} {
		if got := ClassifyCompletionStatus([]string{"a.go"}, true, expect); got != TaskStatusCompleted {
			t.Errorf("a task that changed files is completed (expectChanges=%v), got %q", expect, got)
		}
	}
	// A pushed branch with no discovered diff is still real work.
	if got := ClassifyCompletionStatus(nil, true, true); got != TaskStatusCompleted {
		t.Errorf("a pushed branch counts as work, got %q", got)
	}
}

// MU-8b / V20: pubsub_completion_handler.go decided terminality with a literal
// string compare against "completed"/"failed"/"cancelled". A new status would
// not have registered as terminal and the task could be re-processed — the
// exact silent failure the enum was chosen to avoid.
func TestNoChangesIsTerminal(t *testing.T) {
	if !IsTerminalStatus(TaskStatusNoChanges) {
		t.Fatal("no_changes must be terminal — the run is over and will not be retried")
	}
	for _, s := range []TaskStatus{TaskStatusCompleted, TaskStatusFailed, TaskStatusCancelled} {
		if !IsTerminalStatus(s) {
			t.Errorf("%q must remain terminal", s)
		}
	}
	for _, s := range []TaskStatus{TaskStatusPending, TaskStatusQueued, TaskStatusRunning} {
		if IsTerminalStatus(s) {
			t.Errorf("%q must not be terminal", s)
		}
	}
}

// MU-8c: the exhaustiveness arm. The design doc enumerated the consumers by
// hand (V19) — a list that goes stale the moment someone adds a status. This
// asserts coverage instead of trusting the list, so the NEXT status value
// cannot be added silently.
func TestEveryStatusConsumerHandlesEveryStatus(t *testing.T) {
	all := AllTaskStatuses()
	if len(all) < 9 {
		t.Fatalf("AllTaskStatuses looks incomplete (%d) — it must enumerate every declared status", len(all))
	}
	for _, s := range all {
		if _, ok := terminalByStatus[s]; !ok {
			t.Errorf("status %q has no terminality classification — add it to terminalByStatus", s)
		}
		if _, ok := observatoryByStatus[s]; !ok {
			t.Errorf("status %q has no observatory mapping — add it to observatoryByStatus", s)
		}
	}
}
