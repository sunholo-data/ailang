package coordinator

import (
	"errors"
	"testing"
)

// M7: the detection predicate. The worktree-exists check is the ownership
// test — a coordinator only touches strandings it can actually finalize.
func TestIsStrandedApproval(t *testing.T) {
	here := func(string) error { return nil }
	gone := func(string) error { return errors.New("not here") }

	base := func() (*TaskRecord, *ApprovalRequestRecord) {
		return &TaskRecord{ID: "task-x", Status: TaskStatusPendingApproval, WorktreePath: "/wt/task-x"},
			&ApprovalRequestRecord{ID: "apr-x", TaskID: "task-x", Status: "approved"}
	}

	task, apr := base()
	if !isStrandedApproval(task, apr, here) {
		t.Error("pending task + approved record + local worktree = stranded")
	}

	task, apr = base()
	apr.Status = "pending"
	if isStrandedApproval(task, apr, here) {
		t.Error("a still-pending approval is not stranded — the human has not decided")
	}

	task, apr = base()
	task.Status = TaskStatusCompleted
	if isStrandedApproval(task, apr, here) {
		t.Error("a completed task is not stranded")
	}

	task, apr = base()
	if isStrandedApproval(task, apr, gone) {
		t.Error("worktree on another machine = NOT ours to finalize (that is the cloud side of the same bug)")
	}

	task, apr = base()
	task.WorktreePath = ""
	if isStrandedApproval(task, apr, here) {
		t.Error("no worktree path = nothing to finalize")
	}

	if isStrandedApproval(nil, apr, here) || isStrandedApproval(task, nil, here) {
		t.Error("nil inputs are never stranded")
	}
}
