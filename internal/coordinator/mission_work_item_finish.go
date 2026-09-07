package coordinator

import (
	"context"
	"database/sql"
	"fmt"
)

func (s *SQLiteStore) FinishMissionWorkItem(ctx context.Context, k MissionWorkItemKey, owner, state, reason string) error {
	if state != "completed" && state != "failed" {
		return fmt.Errorf("invalid work item finish state")
	}
	return s.workTx(ctx, func(tx *sql.Tx) error {
		w, err := workFence(ctx, tx, k, owner)
		if err != nil {
			return err
		}
		if state == "completed" && w.NextStage != len(w.StageIDs) {
			return ErrMissionAttemptConflict
		}
		var live int
		if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM mission_attempts WHERE "+workWhere+" AND state IN ('prepared','running','needs_reconciliation')", k.args()...).Scan(&live); err != nil {
			return err
		}
		if live > 0 {
			return ErrMissionAttemptConflict
		}
		args := append([]any{state, reason}, k.args()...)
		if err = missionChanged(tx.ExecContext(ctx, "UPDATE mission_work_items SET state=?,reason_code=?,next_action='',lease_until=0,version=version+1 WHERE "+workWhere, args...)); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "DELETE FROM mission_admissions WHERE "+workWhere, k.args()...)
		return err
	})
}

// Cancel commits the fence before the caller terminates any process. Running
// children retain mission admission until explicit confirmation of process exit.
func (s *SQLiteStore) CancelMissionWorkItem(ctx context.Context, k MissionWorkItemKey, version int64) error {
	if err := k.validate(); err != nil {
		return err
	}
	if version < 1 {
		return fmt.Errorf("expected version is required")
	}
	return s.workTx(ctx, func(tx *sql.Tx) error {
		args := append(k.args(), version)
		if err := missionChanged(tx.ExecContext(ctx, "UPDATE mission_work_items SET state='cancelled',reason_code='operator_cancelled',next_action='',owner_token='',lease_until=0,version=version+1 WHERE "+workWhere+" AND version=? AND state NOT IN ('completed','failed','cancelled')", args...)); err != nil {
			return err
		}
		var running int
		if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM mission_attempts WHERE "+workWhere+" AND state IN ('running','needs_reconciliation')", k.args()...).Scan(&running); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE mission_attempts SET state=CASE WHEN state='prepared' THEN 'cancelled' ELSE 'needs_reconciliation' END,owner_token='',lease_until=0,version=version+1 WHERE "+workWhere+" AND state IN ('prepared','running','needs_reconciliation')", k.args()...); err != nil {
			return err
		}
		if running > 0 {
			_, err := tx.ExecContext(ctx, "UPDATE mission_work_items SET state='needs_reconciliation',next_action='confirm_process_stopped' WHERE "+workWhere, k.args()...)
			return err
		}
		_, err := tx.ExecContext(ctx, "DELETE FROM mission_admissions WHERE "+workWhere, k.args()...)
		return err
	})
}

// ConfirmMissionWorkItemStopped is explicit reconciliation, not a retry or lease
// expiry inference. Call only after the external process has been confirmed dead.
func (s *SQLiteStore) ConfirmMissionWorkItemStopped(ctx context.Context, k MissionWorkItemKey, version int64) error {
	if err := k.validate(); err != nil {
		return err
	}
	if version < 1 {
		return fmt.Errorf("expected version is required")
	}
	return s.workTx(ctx, func(tx *sql.Tx) error {
		args := append(k.args(), version)
		if err := missionChanged(tx.ExecContext(ctx, "UPDATE mission_work_items SET state=CASE WHEN reason_code='deadline_exceeded' THEN 'failed' ELSE 'cancelled' END,next_action='',lease_until=0,owner_token='',version=version+1 WHERE "+workWhere+" AND version=? AND state='needs_reconciliation' AND reason_code IN ('operator_cancelled','deadline_exceeded')", args...)); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE mission_attempts SET state='cancelled',lease_until=0,owner_token='',version=version+1 WHERE "+workWhere+" AND state='needs_reconciliation'", k.args()...); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "DELETE FROM mission_admissions WHERE "+workWhere, k.args()...)
		return err
	})
}

// ReconcileMissionWorkItem conservatively marks an expired running child as
// ambiguous. It preserves admission and never makes execution retryable.
func (s *SQLiteStore) ReconcileMissionWorkItem(ctx context.Context, k MissionWorkItemKey) error {
	if err := k.validate(); err != nil {
		return err
	}
	return s.workTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "UPDATE mission_attempts SET state='needs_reconciliation',version=version+1 WHERE "+workWhere+" AND state='running' AND lease_until<="+missionNow, k.args()...); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "UPDATE mission_work_items SET state='needs_reconciliation',reason_code='outcome_unknown',next_action='reconcile_child',version=version+1 WHERE "+workWhere+" AND state NOT IN ('completed','failed','cancelled','needs_reconciliation') AND EXISTS(SELECT 1 FROM mission_attempts a WHERE a.mission_id=mission_work_items.mission_id AND a.work_item_id=mission_work_items.work_item_id AND a.state='needs_reconciliation')", k.args()...)
		return err
	})
}
