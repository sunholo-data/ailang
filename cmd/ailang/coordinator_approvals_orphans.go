package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// A task can be stuck in pending_approval with NO approval record, and the
// approvals view reported "No pending approvals" while 16 such tasks sat in
// prod (2026-09-07 audit, oldest 2026-04-28).
//
// Two separate defects put them there, both fixed for NEW work by
// M-COMPLETION-PATH-PARITY (v0.35.0) but neither self-healing:
//
//   - the cloud completion path set pending_approval and created no approval
//     record, so there was nothing to approve — 14 of the 16;
//   - an approval was resolved but the task status never followed it, leaving
//     the task pending against an already-terminal record — the other 2.
//
// Either way the task is invisible AND unactionable: `approvals` does not list
// it because it filters on approval records, and `approve`/`reject` refuse it
// with "no pending approval for task". A queue that cannot show you its stuck
// rows is worse than one that is merely empty, because the emptiness reads as
// health.

// orphanedApproval is a task awaiting approval that no approval record covers.
type orphanedApproval struct {
	Task   *coordinator.TaskRecord
	Reason string // why it is unactionable, in the operator's terms
	// HasCode is whether the worktree is STILL THERE, not merely recorded. A
	// path to a directory that no longer exists protects nothing, and treating
	// it as precious leaves a row nobody can clear without --force.
	HasCode bool
}

// worktreeStillExists reports whether a recorded worktree path is real.
//
// Worktrees are swept, machines change, and the task record keeps the path
// either way. Both prod rows that survived the first --clear-orphans pass
// pointed at directories deleted long ago (2026-09-07).
func worktreeStillExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// findOrphanedApprovals returns pending_approval tasks with no PENDING approval
// record, newest first.
//
// It asks the task store rather than the approval store because the defect is
// precisely that the two disagree; consulting only the side that lost the row
// is how this stayed invisible.
func findOrphanedApprovals(ctx context.Context, store coordinator.Store) ([]orphanedApproval, error) {
	tasks, err := store.ListTasks(ctx, &coordinator.TaskFilter{
		Status: []coordinator.TaskStatus{coordinator.TaskStatusPendingApproval},
		Limit:  500,
	})
	if err != nil {
		return nil, fmt.Errorf("list pending_approval tasks: %w", err)
	}

	var out []orphanedApproval
	for _, t := range tasks {
		if t == nil {
			continue
		}
		if _, err := store.GetApprovalRequestByTask(ctx, t.ID); err == nil {
			continue // a real pending approval — the normal view already shows it
		}
		reason := "no approval record was ever created"
		if prior, err := store.GetApprovalRequestByTaskAnyStatus(ctx, t.ID); err == nil && prior != nil {
			reason = fmt.Sprintf("approval already %s, but the task status never followed", prior.Status)
		}
		out = append(out, orphanedApproval{
			Task:    t,
			Reason:  reason,
			HasCode: worktreeStillExists(t.WorktreePath),
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Task.CreatedAt.After(out[j].Task.CreatedAt)
	})
	return out, nil
}

// reportOrphanedApprovals prints the stuck rows. Silent when there are none.
func reportOrphanedApprovals(orphans []orphanedApproval, planeWord string) {
	if len(orphans) == 0 {
		return
	}
	fmt.Printf("\n⚠ %d task(s) await approval that nothing can approve:\n\n", len(orphans))
	for _, o := range orphans {
		age := time.Since(o.Task.CreatedAt).Round(time.Hour)
		fmt.Printf("  %-18s %-26s %s old\n", o.Task.ID, o.Task.AgentID, age)
		fmt.Printf("      %s\n", o.Reason)
		if title := o.Task.Title; title != "" {
			fmt.Printf("      %q\n", truncateOrphanTitle(title, 66))
		}
		// Say what approving would release, so "clear" is an informed choice.
		switch {
		case o.HasCode:
			fmt.Printf("      worktree PRESENT: %s\n", o.Task.WorktreePath)
		case o.Task.WorktreePath != "":
			fmt.Printf("      worktree recorded but GONE (%s) — nothing to merge\n", o.Task.WorktreePath)
		default:
			fmt.Printf("      no worktree — this task left no code to merge\n")
		}
		fmt.Println()
	}
	fmt.Printf("These predate the v0.35.0 fix and will not clear themselves. To cancel them:\n")
	fmt.Printf("  ailang coordinator approvals --remote %s --clear-orphans\n", planeWord)
}

// clearOrphanedApprovals cancels orphaned tasks, and refuses to touch one that
// still has code nobody has looked at.
//
// A task whose worktree STILL EXISTS may hold real work; cancelling it silently
// discards the only pointer to it. Those are skipped unless --force. A path that
// no longer resolves is not protected — the guard would otherwise strand rows
// forever over code that was swept months ago.
func clearOrphanedApprovals(ctx context.Context, store coordinator.Store, orphans []orphanedApproval, force bool) (cleared, skipped int, err error) {
	for _, o := range orphans {
		if o.HasCode && !force {
			fmt.Printf("  SKIP  %s — worktree still present at %s (pass --force to cancel anyway)\n",
				o.Task.ID, o.Task.WorktreePath)
			skipped++
			continue
		}
		t := o.Task
		t.Status = coordinator.TaskStatusCancelled
		t.Error = fmt.Sprintf("cancelled by approvals --clear-orphans: %s", o.Reason)
		if uErr := store.UpdateTask(ctx, t); uErr != nil {
			return cleared, skipped, fmt.Errorf("cancel %s: %w", t.ID, uErr)
		}
		fmt.Printf("  cancelled %s (%s)\n", t.ID, o.Reason)
		cleared++
	}
	return cleared, skipped, nil
}

func truncateOrphanTitle(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
