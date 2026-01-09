package coordinator

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ApprovalRequestRecord is the database record for an approval request
type ApprovalRequestRecord struct {
	ID          string     `json:"id"`
	TaskID      string     `json:"task_id"`
	Type        string     `json:"type"`
	Description string     `json:"description"`
	ContextJSON string     `json:"context_json,omitempty"`
	Status      string     `json:"status"`
	ResolvedBy  string     `json:"resolved_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	TimeoutAt   *time.Time `json:"timeout_at,omitempty"`
	AutoReject  bool       `json:"auto_reject"`
}

// CreateApprovalRequest creates a new approval request in the database
func (s *SQLiteStore) CreateApprovalRequest(ctx context.Context, req *ApprovalRequestRecord) error {
	query := `
		INSERT INTO approval_requests (id, task_id, type, description, context_json, status, timeout_at, auto_reject, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		req.ID, req.TaskID, req.Type, req.Description, req.ContextJSON,
		req.Status, req.TimeoutAt, req.AutoReject, req.CreatedAt,
	)
	return err
}

// GetApprovalRequest retrieves an approval request by ID
func (s *SQLiteStore) GetApprovalRequest(ctx context.Context, id string) (*ApprovalRequestRecord, error) {
	query := `
		SELECT id, task_id, type, description, context_json, status, resolved_by, created_at, resolved_at, timeout_at, auto_reject
		FROM approval_requests WHERE id = ?
	`
	row := s.db.QueryRowContext(ctx, query, id)
	return s.scanApprovalRequest(row)
}

// GetApprovalRequestByTask retrieves a pending approval request for a task
func (s *SQLiteStore) GetApprovalRequestByTask(ctx context.Context, taskID string) (*ApprovalRequestRecord, error) {
	query := `
		SELECT id, task_id, type, description, context_json, status, resolved_by, created_at, resolved_at, timeout_at, auto_reject
		FROM approval_requests WHERE task_id = ? AND status = 'pending'
		ORDER BY created_at DESC LIMIT 1
	`
	row := s.db.QueryRowContext(ctx, query, taskID)
	req, err := s.scanApprovalRequest(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return req, err
}

// ListPendingApprovals retrieves all pending approval requests
func (s *SQLiteStore) ListPendingApprovals(ctx context.Context) ([]*ApprovalRequestRecord, error) {
	query := `
		SELECT id, task_id, type, description, context_json, status, resolved_by, created_at, resolved_at, timeout_at, auto_reject
		FROM approval_requests WHERE status = 'pending'
		ORDER BY created_at ASC
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []*ApprovalRequestRecord
	for rows.Next() {
		req, err := s.scanApprovalRequestFromRows(rows)
		if err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}
	return requests, rows.Err()
}

// ResolveApprovalRequest marks an approval request as approved or rejected
func (s *SQLiteStore) ResolveApprovalRequest(ctx context.Context, id string, status string, resolvedBy string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		"UPDATE approval_requests SET status = ?, resolved_by = ?, resolved_at = ? WHERE id = ?",
		status, resolvedBy, now, id,
	)
	return err
}

// ResolveApprovalRequestByTask marks approval requests for a task as approved or rejected.
// Only resolves "merge" type approvals - handoff approvals must be resolved separately via ResolveApprovalRequest.
func (s *SQLiteStore) ResolveApprovalRequestByTask(ctx context.Context, taskID string, status string, resolvedBy string) error {
	now := time.Now()
	// Only resolve merge approvals - handoff approvals require explicit approval via separate CLI command
	result, err := s.db.ExecContext(ctx,
		"UPDATE approval_requests SET status = ?, resolved_by = ?, resolved_at = ? WHERE task_id = ? AND status = 'pending' AND type = 'merge'",
		status, resolvedBy, now, taskID,
	)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return fmt.Errorf("no pending approval request found for task: %s", taskID)
	}

	// Also update the task status to match the approval decision
	var taskStatus TaskStatus
	switch status {
	case "rejected":
		taskStatus = TaskStatusRejected
	case "approved":
		taskStatus = TaskStatusCompleted
	default:
		return nil // Unknown status, don't update task
	}

	_, err = s.db.ExecContext(ctx,
		"UPDATE tasks SET status = ?, completed_at = ? WHERE id = ?",
		taskStatus, now, taskID,
	)
	return err
}

// ReopenTask moves a rejected/cancelled task back to pending_approval status
func (s *SQLiteStore) ReopenTask(ctx context.Context, taskID string) error {
	// First check if the task exists and is in a reopenable state
	var currentStatus string
	err := s.db.QueryRowContext(ctx, "SELECT status FROM tasks WHERE id = ?", taskID).Scan(&currentStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("task not found: %s", taskID)
		}
		return err
	}

	// Only allow reopening rejected or cancelled tasks
	if currentStatus != string(TaskStatusRejected) && currentStatus != string(TaskStatusCancelled) {
		return fmt.Errorf("cannot reopen task with status %q (only rejected or cancelled tasks can be reopened)", currentStatus)
	}

	// Update task status back to pending_approval
	_, err = s.db.ExecContext(ctx,
		"UPDATE tasks SET status = ?, completed_at = NULL WHERE id = ?",
		TaskStatusPendingApproval, taskID,
	)
	if err != nil {
		return err
	}

	// Reset the approval request back to pending (or create one if missing)
	result, err := s.db.ExecContext(ctx,
		"UPDATE approval_requests SET status = 'pending', resolved_by = NULL, resolved_at = NULL WHERE task_id = ?",
		taskID,
	)
	if err != nil {
		return err
	}

	// If no approval request existed, create one
	count, _ := result.RowsAffected()
	if count == 0 {
		approvalID := fmt.Sprintf("apr-%s", taskID[5:]) // apr-<hash> from task-<hash>
		_, err = s.db.ExecContext(ctx,
			`INSERT INTO approval_requests (id, task_id, type, description, status, created_at)
			 VALUES (?, ?, 'merge', 'Task reopened for approval', 'pending', ?)`,
			approvalID, taskID, time.Now(),
		)
		if err != nil {
			return fmt.Errorf("failed to create approval request: %w", err)
		}
	}

	return nil
}

// DeleteOldApprovals removes approval requests older than the specified duration
func (s *SQLiteStore) DeleteOldApprovals(ctx context.Context, olderThan time.Duration) (int, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := s.db.ExecContext(ctx,
		"DELETE FROM approval_requests WHERE created_at < ? AND status != 'pending'",
		cutoff,
	)
	if err != nil {
		return 0, err
	}
	count, _ := result.RowsAffected()
	return int(count), nil
}

// scanApprovalRequest scans a single approval request from a row
func (s *SQLiteStore) scanApprovalRequest(row *sql.Row) (*ApprovalRequestRecord, error) {
	req := &ApprovalRequestRecord{}
	var resolvedBy, contextJSON sql.NullString
	var resolvedAt, timeoutAt sql.NullTime

	err := row.Scan(
		&req.ID, &req.TaskID, &req.Type, &req.Description, &contextJSON,
		&req.Status, &resolvedBy, &req.CreatedAt, &resolvedAt, &timeoutAt, &req.AutoReject,
	)
	if err != nil {
		return nil, err
	}

	if resolvedBy.Valid {
		req.ResolvedBy = resolvedBy.String
	}
	if contextJSON.Valid {
		req.ContextJSON = contextJSON.String
	}
	if resolvedAt.Valid {
		req.ResolvedAt = &resolvedAt.Time
	}
	if timeoutAt.Valid {
		req.TimeoutAt = &timeoutAt.Time
	}

	return req, nil
}

// scanApprovalRequestFromRows scans an approval request from rows
func (s *SQLiteStore) scanApprovalRequestFromRows(rows *sql.Rows) (*ApprovalRequestRecord, error) {
	req := &ApprovalRequestRecord{}
	var resolvedBy, contextJSON sql.NullString
	var resolvedAt, timeoutAt sql.NullTime

	err := rows.Scan(
		&req.ID, &req.TaskID, &req.Type, &req.Description, &contextJSON,
		&req.Status, &resolvedBy, &req.CreatedAt, &resolvedAt, &timeoutAt, &req.AutoReject,
	)
	if err != nil {
		return nil, err
	}

	if resolvedBy.Valid {
		req.ResolvedBy = resolvedBy.String
	}
	if contextJSON.Valid {
		req.ContextJSON = contextJSON.String
	}
	if resolvedAt.Valid {
		req.ResolvedAt = &resolvedAt.Time
	}
	if timeoutAt.Valid {
		req.TimeoutAt = &timeoutAt.Time
	}

	return req, nil
}
