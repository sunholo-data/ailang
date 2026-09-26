package coordinator

import (
	"context"
	"time"
)

// Task state transitions for the SQLite store.
//
// Split out of store_sqlite.go when the claim below grew a precondition and
// pushed that file past the 800-line ceiling. They belong together anyway:
// these nine functions are the only things that move a task between statuses,
// and terminalByStatus / dedupSuppressesByStatus in task_status.go are the
// tables that say what those statuses MEAN.

// MarkTaskQueued marks a task as queued AND clears its finalization ledger.
//
// The ledger makes finalisation idempotent across REDELIVERIES of one
// completion. It is not meant to span EXECUTIONS — but it lives on the task
// record, so a re-dispatched task carried the previous run's ledger, every
// effect read as already-done, and the new work was never finalised at all.
//
// Measured 2026-09-15: eleven ailang-core-triage tasks were rejected,
// re-dispatched, ran again and produced new rows. Each second finalisation
// skipped `approval (already done)`, so the task settled into
// `pending_approval` behind the previous attempt's `rejected` record —
// invisible to `coordinator approvals` and refused by approve, because an
// already-resolved approval cannot be resolved again. Eleven finished pieces of
// work with no way to accept them.
//
// Queueing is the right place: a task entering the queue is about to produce
// new work, and the previous execution's effects are history. Clearing on the
// FIRST queue is a no-op, so there is no separate re-dispatch path to keep in
// step — which is exactly the kind of second path this codebase keeps growing.
// The WHERE clause carries `status = pending` because this is a CLAIM: the
// update is the atomic compare-and-set that decides which dispatcher owns the
// task. Without it every concurrent caller succeeded and every one dispatched.
func (s *SQLiteStore) MarkTaskQueued(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx,
		"UPDATE tasks SET status = ?, finalization = NULL, queued_at = ? WHERE id = ? AND status = ?",
		TaskStatusQueued, time.Now(), id, TaskStatusPending,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		// Zero rows means the row was not pending — claimed by someone else, or
		// already moved on. NOT a missing row: a task can only reach here from a
		// pending listing. Either way this caller must not dispatch.
		return ErrTaskNotClaimable
	}
	return nil
}

// MarkTaskRunning marks a task as running
func (s *SQLiteStore) MarkTaskRunning(ctx context.Context, id, provider, worktreeID string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE tasks SET status = ?, provider = ?, worktree_id = ?, started_at = ? WHERE id = ?",
		TaskStatusRunning, provider, worktreeID, time.Now(), id,
	)
	return err
}

// MarkTaskCompleted marks a task as completed with results
func (s *SQLiteStore) MarkTaskCompleted(ctx context.Context, id string, result *ExecuteResult) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET
			status = ?, completed_at = ?, duration_ns = ?,
			output = ?, cost = ?, tokens_used = ?,
			session_id = ?
		WHERE id = ?`,
		TaskStatusCompleted, now, int64(result.Duration),
		result.Output, result.Cost, result.TokensUsed,
		result.SessionID, id,
	)
	return err
}

// MarkTaskFailed marks a task as failed
func (s *SQLiteStore) MarkTaskFailed(ctx context.Context, id string, taskErr error) error {
	now := time.Now()
	errMsg := ""
	if taskErr != nil {
		errMsg = taskErr.Error()
	}
	_, err := s.db.ExecContext(ctx,
		"UPDATE tasks SET status = ?, completed_at = ?, error = ? WHERE id = ?",
		TaskStatusFailed, now, errMsg, id,
	)
	return err
}

// MarkTaskCancelled marks a task as cancelled
func (s *SQLiteStore) MarkTaskCancelled(ctx context.Context, id string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		"UPDATE tasks SET status = ?, completed_at = ? WHERE id = ?",
		TaskStatusCancelled, now, id,
	)
	return err
}

// MarkTaskPendingApproval marks a task as awaiting human approval
func (s *SQLiteStore) MarkTaskPendingApproval(ctx context.Context, id, worktreePath, worktreeBranch, baseBranch, baseCommit string, result *ExecuteResult) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET
			status = ?, completed_at = ?, worktree_path = ?, worktree_id = ?, base_branch = ?, base_commit = ?,
			duration_ns = ?, output = ?, cost = ?, tokens_used = ?, session_id = ?
		WHERE id = ?`,
		TaskStatusPendingApproval, now, worktreePath, worktreeBranch, baseBranch, baseCommit,
		int64(result.Duration), result.Output, result.Cost, result.TokensUsed, result.SessionID, id,
	)
	return err
}

// MarkTaskRejected marks a task as rejected by human
func (s *SQLiteStore) MarkTaskRejected(ctx context.Context, id string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		"UPDATE tasks SET status = ?, completed_at = ? WHERE id = ?",
		TaskStatusRejected, now, id,
	)
	return err
}

// RequeueTask resets a task to pending status for re-execution.
func (s *SQLiteStore) RequeueTask(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE tasks SET status = ?, started_at = NULL, completed_at = NULL WHERE id = ?",
		TaskStatusPending, id,
	)
	return err
}

// ResetTaskToPending resets a running task back to pending state.
func (s *SQLiteStore) ResetTaskToPending(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE tasks SET status = ?, started_at = NULL WHERE id = ?",
		TaskStatusPending, id,
	)
	return err
}
