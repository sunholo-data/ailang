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

// Queue-side lane guard (2026-08-27): a worker on shared storage must not
// execute tasks for agents it does not serve. The rig claimed a cloud-lane
// parse task, defaulted its worktree and provider, and ran a user's feedback
// under codex in the wrong repo — a completed-looking disaster averted only by
// codex failing.
func TestQueueSkipsForeignAgentTasks(t *testing.T) {
	r := NewAgentRegistry()
	if err := r.Register(&AgentConfig{ID: "eval-rig", Inbox: "eval-rig", Workspace: "/tmp/x"}); err != nil {
		t.Fatal(err)
	}
	if r.GetAgentByID("pkg-sunholo-ailang-parse") != nil {
		t.Fatal("foreign agent unexpectedly present")
	}
	// The predicate the queue applies: non-empty agent + not in registry = skip.
	task := &TaskRecord{ID: "task-x", AgentID: "pkg-sunholo-ailang-parse"}
	if !(task.AgentID != "" && r.GetAgentByID(task.AgentID) == nil) {
		t.Error("foreign-agent task must be skipped")
	}
	own := &TaskRecord{ID: "task-y", AgentID: "eval-rig"}
	if own.AgentID != "" && r.GetAgentByID(own.AgentID) == nil {
		t.Error("own-agent task must NOT be skipped")
	}
}
