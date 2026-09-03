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
