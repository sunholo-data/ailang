package coordinator

import (
	"os"
)

// M-PIPELINE-RECONCILIATION M7: sweep approvals that were resolved on another
// machine and never finalized here.
//
// A dashboard (cloud) approval of a LOCAL-lane task flips the approval record
// in the shared store, then finds no worktree at the task's path — it is on
// another machine's disk — and returns success having merged nothing. The task
// sits in pending_approval forever, invisible from the machine that approved
// it and unannounced on the machine that owns it. Measured live 2026-08-26
// (task-6051f916, approved by=dashboard-user via dashboard, rig log silent).
//
// The sweep runs on the owning side: each poll tick, find MY agents' tasks
// still pending_approval whose approval record is already approved, and whose
// worktree exists on THIS machine — that existence is what makes this
// coordinator the owner — then run the local finalize.

// isStrandedApproval is the pure detection predicate, split out for testing.
// statFn abstracts os.Stat so tests can control "is the worktree here".
func isStrandedApproval(task *TaskRecord, approval *ApprovalRequestRecord, statFn func(string) error) bool {
	if task == nil || approval == nil {
		return false
	}
	if task.Status != TaskStatusPendingApproval {
		return false
	}
	if approval.Status != "approved" {
		return false
	}
	if task.WorktreePath == "" {
		return false
	}
	return statFn(task.WorktreePath) == nil
}

// sweepStrandedApprovals finds and finalizes stranded approvals for this
// coordinator's agents. Called from the poll loop; cheap when nothing is
// pending.
func (d *Daemon) sweepStrandedApprovals() {
	if d.taskStore == nil || d.agentRegistry == nil {
		return
	}
	ctx := d.ctx

	tasks, err := d.taskStore.ListTasks(ctx, &TaskFilter{Status: []TaskStatus{TaskStatusPendingApproval}})
	if err != nil {
		d.logger.Printf("Warning: stranded-approval sweep could not list tasks: %v", err)
		return
	}

	for _, task := range tasks {
		// Only MY agents: an unknown agent's task belongs to another
		// coordinator (or the cloud), and touching it would be the same
		// cross-lane overreach in the other direction.
		if task.AgentID == "" || d.agentRegistry.GetAgentByID(task.AgentID) == nil {
			continue
		}

		approval, err := d.taskStore.GetApprovalRequestByTaskAnyStatus(ctx, task.ID)
		if err != nil || approval == nil {
			continue
		}
		if !isStrandedApproval(task, approval, func(p string) error { _, err := os.Stat(p); return err }) {
			continue
		}

		// The human approved with the evaluation verdict displayed, so the
		// human decision dominates AllowsAutomation here: this sweep completes
		// a person's explicit action, it does not originate one.
		d.logger.Printf("Stranded approval: task %s approved by %q elsewhere (%s) but never finalized here — resuming",
			task.ID, approval.ResolvedBy, approval.ID)

		params := &ApprovalParams{
			TaskID:        task.ID,
			Action:        "approve",
			Channel:       "cross-lane-sweep",
			Store:         d.taskStore,
			MsgStore:      d.msgStore,
			AgentRegistry: d.agentRegistry,
			ObsBackend:    d.obsBackend,
		}
		result, err := ResumeStrandedApproval(ctx, params, task, approval.ResolvedBy)
		if err != nil {
			d.logger.Printf("ERROR: stranded approval for task %s could not be finalized: %v", task.ID, err)
			continue
		}
		d.logger.Printf("Stranded approval finalized: %s", result.Message)
	}
}
