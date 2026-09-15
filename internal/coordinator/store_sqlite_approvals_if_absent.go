package coordinator

import (
	"context"
	"fmt"
)

// CreateApprovalIfAbsent creates an approval unless one with the same id already
// exists, and reports whether it created it (M-COMPLETION-PATH-PARITY M0b).
//
// The approval id is already deterministic — apr-<task hash> — so a redelivered
// completion targets the same row. CreateApprovalRequest issues a bare INSERT,
// which means that second delivery raises a UNIQUE violation and turns a routine
// Pub/Sub replay into a repeating failure. The Firestore backend fails the
// opposite way, overwriting an approval a human has already resolved.
//
// First write wins, replays are a no-op, and the caller is told which happened so
// it can record the difference rather than guess.
func (s *SQLiteStore) CreateApprovalIfAbsent(ctx context.Context, req *ApprovalRequestRecord) (bool, error) {
	if req == nil || req.ID == "" {
		return false, fmt.Errorf("CreateApprovalIfAbsent requires an approval with an explicit id")
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO approval_requests (id, task_id, type, description, context_json, status, timeout_at, auto_reject, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING
	`, req.ID, req.TaskID, req.Type, req.Description, req.ContextJSON,
		req.Status, req.TimeoutAt, req.AutoReject, req.CreatedAt)
	if err != nil {
		return false, fmt.Errorf("failed to create approval %s: %w", req.ID, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to read rows affected for approval %s: %w", req.ID, err)
	}
	return rows > 0, nil
}

// ReopenApprovalForNewWork puts a RESOLVED approval back to pending because a
// later execution of the same task produced a different change.
//
// Guarded on status so it can never reopen a pending row (nothing to reopen)
// and reports whether it moved one, so the caller can say what happened instead
// of assuming. The work id and the fresh diff go in with it — a reopened
// approval that still describes the previous change is worse than no approval,
// because the card looks authoritative.
func (s *SQLiteStore) ReopenApprovalForNewWork(ctx context.Context, taskID, description, contextJSON string) (bool, error) {
	if taskID == "" {
		return false, fmt.Errorf("ReopenApprovalForNewWork requires a task id")
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE approval_requests
		   SET status = 'pending', resolved_by = NULL, resolved_at = NULL,
		       description = ?, context_json = ?
		 WHERE task_id = ? AND status IN ('approved','rejected')
	`, description, contextJSON, taskID)
	if err != nil {
		return false, fmt.Errorf("reopening approval for %s: %w", taskID, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected reopening %s: %w", taskID, err)
	}
	return n > 0, nil
}

// RefreshPendingApproval rewrites a PENDING approval's evidence because a later
// execution of the same task produced a different change.
//
// The card the operator reads and the branch the merge lands were allowed to
// come apart. Measured 2026-09-15 on task-c0ca6301: run 1 wrote
// `design_docs/planned/ailang-core-backlog.md` (+7) and created the approval;
// the task was re-dispatched and run 2 wrote
// `design_docs/planned/ailang-core-triage/coordinator-completion-wrong-ref.md`
// (+11) onto the same branch. CreateApprovalIfAbsent said "already exists",
// ClassifyApprovalCollision called a pending row a replay, and the approval kept
// run 1's diff. The operator approved a card describing a file that no longer
// existed and merged a file the card never named.
//
// Guarded on 'pending' so it can never touch a decision that has been made —
// that case is ReopenApprovalForNewWork's, and the two must not overlap.
func (s *SQLiteStore) RefreshPendingApproval(ctx context.Context, taskID, description, contextJSON string) (bool, error) {
	if taskID == "" {
		return false, fmt.Errorf("RefreshPendingApproval requires a task id")
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE approval_requests
		   SET description = ?, context_json = ?
		 WHERE task_id = ? AND status = 'pending'
	`, description, contextJSON, taskID)
	if err != nil {
		return false, fmt.Errorf("refreshing pending approval for %s: %w", taskID, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected refreshing %s: %w", taskID, err)
	}
	return n > 0, nil
}
