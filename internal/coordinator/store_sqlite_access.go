package coordinator

import (
	"context"
	"fmt"
)

// DurableAccessApprovalType is deliberately separate from merge/secret approvals.
// Durable rows also use access_* SQL statuses so legacy pending watchers cannot
// treat an access request as a merge or synchronous secret decision.
const DurableAccessApprovalType = "durable_access_v1"

// NewAccessApprovalSQLiteStore opts in to FULL commit durability for the
// isolated operator authority. Legacy coordinator openers retain NORMAL sync.
// A started marker must survive power loss before a protected effect begins.
func NewAccessApprovalSQLiteStore(path string) (*SQLiteStore, error) {
	if path == "" {
		return nil, fmt.Errorf("access approval store requires an explicit path")
	}
	options := coordinatorDBOptions
	options.FullSync = true
	return newSQLiteStoreWithOptions(path, options)
}

// CreateAccessApprovalIfAbsent atomically enforces the per-owner/worker pending
// cap and first-write-wins identity. False means replay or capacity exhausted;
// the caller distinguishes these by looking up the deterministic id.
func (s *SQLiteStore) CreateAccessApprovalIfAbsent(ctx context.Context, r *ApprovalRequestRecord, owner, worker string, now int64) (bool, error) {
	if r.Type != DurableAccessApprovalType {
		return false, fmt.Errorf("invalid access approval type")
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO approval_requests
 (id,task_id,type,description,context_json,status,created_at)
 SELECT ?,?,?,?,?,?,? WHERE
 (SELECT COUNT(*) FROM approval_requests WHERE type=? AND status='access_pending'
 AND json_valid(context_json) AND json_extract(context_json,'$.owner')=?
 AND json_extract(context_json,'$.worker')=? AND json_extract(context_json,'$.expires_at')>?) < 100
 ON CONFLICT(id) DO NOTHING`, r.ID, r.TaskID, r.Type, r.Description, r.ContextJSON, r.Status, r.CreatedAt, DurableAccessApprovalType, owner, worker, now)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

// CompareAndSwapAccessApproval makes every lifecycle mutation conditional on
// the exact prior durable record, including its revision. No legacy mutation
// method is used for access decisions or claims.
func (s *SQLiteStore) CompareAndSwapAccessApproval(ctx context.Context, id, before, after, status string) (bool, error) {
	result, err := s.db.ExecContext(ctx, `UPDATE approval_requests SET context_json=?,status=? WHERE id=? AND type=? AND context_json=?`, after, "access_"+status, id, DurableAccessApprovalType, before)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

// ListAccessApprovalContexts scopes and bounds in SQL. The effective decision
// incorporates expiry for accurate pending/history counts before page records
// are durably reconciled. No unbounded history is loaded into the CLI process.
func (s *SQLiteStore) ListAccessApprovalContexts(ctx context.Context, owner, worker, status string, now int64, limit, offset int) ([]string, int, error) {
	effective := `(CASE WHEN json_extract(context_json,'$.decision')='pending' AND json_extract(context_json,'$.expires_at')<=? THEN 'expired'
 WHEN json_extract(context_json,'$.decision')='approved' AND json_extract(context_json,'$.execution.state')='unclaimed' AND json_extract(context_json,'$.claim_until')<=? THEN 'expired'
 ELSE json_extract(context_json,'$.decision') END)`
	predicate := `type=? AND json_valid(context_json) AND json_extract(context_json,'$.owner')=? AND (?='' OR json_extract(context_json,'$.worker')=?)`
	args := []any{DurableAccessApprovalType, owner, worker, worker}
	if status == "pending" {
		predicate += " AND " + effective + "='pending'"
		args = append(args, now, now)
	}
	if status == "history" {
		predicate += " AND " + effective + "<>'pending'"
		args = append(args, now, now)
	}
	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM approval_requests WHERE "+predicate, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	order := "created_at DESC,id"
	if status == "pending" {
		order = "created_at ASC,id"
	}
	rows, err := s.db.QueryContext(ctx, "SELECT context_json FROM approval_requests WHERE "+predicate+" ORDER BY "+order+" LIMIT ? OFFSET ?", args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, 0, err
		}
		out = append(out, raw)
	}
	return out, total, rows.Err()
}
