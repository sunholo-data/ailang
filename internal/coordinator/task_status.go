package coordinator

import "github.com/sunholo-data/ailang/internal/observatory"

// M-COORDINATOR-EXECUTION-TRUST M2 — a no-op is an outcome, not a silence.
//
// `completed` used to mean "the executor exited 0", which is not the same as
// "work landed" and could not be told apart from it. Measured store-wide: 1,249
// completion records, 16 marked completed, 6 that ever changed a file.
//
// The fix is a new terminal status rather than a boolean on `completed`, and
// the reason is which way an old consumer is wrong. A consumer that has never
// heard of `no_changes` reads it as NOT completed, i.e. as not-success — it
// surfaces the problem. A boolean would have been read as success by exactly
// the queries that matter. Ruled by Mark 2026-09-02 (design doc D3).
//
// The tables below exist because the enumeration of consumers was, in the
// design doc's first draft, a hand-written list that was wrong in kind (V19).
// A hand-written list goes stale the moment someone adds a status; a table the
// tests iterate cannot.

// TaskStatusNoChanges is terminal: the run finished, was expected to change
// something, and produced no diff and no branch.
const TaskStatusNoChanges TaskStatus = "no_changes"

// AllTaskStatuses enumerates every declared status. Anything added to the
// constant block belongs here, and the exhaustiveness test will say so.
func AllTaskStatuses() []TaskStatus {
	return []TaskStatus{
		TaskStatusPending,
		TaskStatusQueued,
		TaskStatusRunning,
		TaskStatusPendingApproval,
		TaskStatusCompleted,
		TaskStatusNoChanges,
		TaskStatusFailed,
		TaskStatusRejected,
		TaskStatusCancelled,
		TaskStatusDuplicate,
	}
}

// terminalByStatus is the single answer to "is this task over?".
//
// It replaces a literal string comparison in pubsub_completion_handler.go
// (`task.Status == "completed" || "failed" || "cancelled"`) which would not
// have recognised no_changes as terminal, letting the task be re-processed.
var terminalByStatus = map[TaskStatus]bool{
	TaskStatusPending:         false,
	TaskStatusQueued:          false,
	TaskStatusRunning:         false,
	TaskStatusPendingApproval: false, // work done, but the task is not over — a human still rules
	TaskStatusCompleted:       true,
	TaskStatusNoChanges:       true,
	TaskStatusFailed:          true,
	TaskStatusRejected:        true,
	TaskStatusCancelled:       true,
	TaskStatusDuplicate:       true,
}

// observatoryByStatus maps each status onto the observatory's smaller vocabulary.
//
// This was a switch with a `default: pending` arm, which meant an unmapped
// status silently reported as still-running — and TaskStatusDuplicate was
// ALREADY falling through it before this milestone added anything.
var observatoryByStatus = map[TaskStatus]observatory.TaskStatus{
	TaskStatusPending:         observatory.TaskStatusPending,
	TaskStatusQueued:          observatory.TaskStatusPending,
	TaskStatusRunning:         observatory.TaskStatusRunning,
	TaskStatusPendingApproval: observatory.TaskStatusCompleted,
	TaskStatusCompleted:       observatory.TaskStatusCompleted,
	// A no-op run is not a success. Reporting it as completed would restore
	// exactly the ambiguity this milestone removes.
	TaskStatusNoChanges: observatory.TaskStatusFailed,
	TaskStatusFailed:    observatory.TaskStatusFailed,
	TaskStatusRejected:  observatory.TaskStatusFailed,
	TaskStatusCancelled: observatory.TaskStatusFailed,
	TaskStatusDuplicate: observatory.TaskStatusFailed,
}

// TerminalStatuses returns every status that means the task is over, derived
// from the same table IsTerminalStatus reads. It replaces a hand-maintained
// list in the worktree sweep that had already drifted — it omitted
// TaskStatusDuplicate, so those worktrees were never cleaned up.
func TerminalStatuses() []TaskStatus {
	out := make([]TaskStatus, 0, len(terminalByStatus))
	for _, s := range AllTaskStatuses() { // AllTaskStatuses for a stable order
		if terminalByStatus[s] {
			out = append(out, s)
		}
	}
	return out
}

// IsTerminalStatus reports whether a task in this status is over.
// An unknown status is NOT terminal: treating an unrecognised value as finished
// would silently drop work, so the safe direction is to keep looking at it.
func IsTerminalStatus(s TaskStatus) bool {
	return terminalByStatus[s]
}

// ClassifyCompletionStatus decides the terminal status of a finished run.
//
// expectChanges comes from the coordinator's trusted dispatch metadata — the
// SAME authority boundary as WorkTier, and deliberately not from
// classifyTaskType(), which is a substring match over sender-controlled message
// content (design doc V18). M2's first draft used the task type here and
// inherited exactly the hole M1a was rewritten to remove.
//
// An acknowledge-only dispatch (expectChanges=false) is supposed to change
// nothing: it completes cleanly, creates no branch and is not an error. That
// path was added deliberately on 2026-08-26 for probe tasks, to stop orphan
// branches and `422 "No commits between..."` PR failures, and it stays.
func ClassifyCompletionStatus(changedFiles []string, branchPushed bool, expectChanges bool) TaskStatus {
	if len(changedFiles) > 0 || branchPushed {
		return TaskStatusCompleted
	}
	if !expectChanges {
		return TaskStatusCompleted
	}
	return TaskStatusNoChanges
}
