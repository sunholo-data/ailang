package coordinator

import (
	"context"
	"database/sql"
	"fmt"
)

// ExpireMissionWorkItem durably closes a deadline, fencing all uncompleted child
// generations before returning. An expired lease may finalize only its still-current
// owner token; reclaim by another owner always wins that fence. Ambiguous processes
// retain mission admission and can never become retryable through expiry.
func (s *SQLiteStore) ExpireMissionWorkItem(ctx context.Context, k MissionWorkItemKey, owner string) error {
	return s.expireMissionDeadline(ctx, k, owner, "")
}

// ExpireMissionStage closes the current stage's durable deadline without making
// any potentially dispatched child retryable.
func (s *SQLiteStore) ExpireMissionStage(ctx context.Context, k MissionWorkItemKey, owner, stageID string) error {
	if !missionIDValid(stageID) {
		return fmt.Errorf("invalid stage id")
	}
	return s.expireMissionDeadline(ctx, k, owner, stageID)
}

func (s *SQLiteStore) expireMissionDeadline(ctx context.Context, k MissionWorkItemKey, owner, stageID string) error {
	if err := k.validate(); err != nil {
		return err
	}
	if owner == "" {
		return fmt.Errorf("expiry requires owner token")
	}
	return s.workTx(ctx, func(tx *sql.Tx) error {
		condition := "deadline<=" + missionNow
		args := append(k.args(), owner)
		if stageID != "" {
			w, err := scanWork(tx.QueryRowContext(ctx, "SELECT "+workFields+" FROM mission_work_items WHERE "+workWhere, k.args()...))
			if err != nil {
				return err
			}
			if !workChildMatches(w, MissionAttemptKey{MissionID: k.MissionID, WorkItemID: k.WorkItemID, StageID: stageID}) {
				return ErrMissionAttemptConflict
			}
			condition = "EXISTS(SELECT 1 FROM mission_stage_deadlines d WHERE d.mission_id=mission_work_items.mission_id AND d.work_item_id=mission_work_items.work_item_id AND d.stage_id=? AND d.deadline<=" + missionNow + ")"
			args = append(args, stageID)
		}
		if err := missionChanged(tx.ExecContext(ctx, "UPDATE mission_work_items SET state='failed',reason_code='deadline_exceeded',next_action='',owner_token='',lease_until=0,version=version+1 WHERE "+workWhere+" AND owner_token=? AND "+condition+" AND state NOT IN ('completed','failed','cancelled')", args...)); err != nil {
			return err
		}
		var ambiguous int
		if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM mission_attempts WHERE "+workWhere+" AND state IN ('running','needs_reconciliation')", k.args()...).Scan(&ambiguous); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE mission_attempts SET state=CASE WHEN state='prepared' THEN 'cancelled' ELSE 'needs_reconciliation' END,owner_token='',lease_until=0,version=version+1 WHERE "+workWhere+" AND state IN ('prepared','running','needs_reconciliation')", k.args()...); err != nil {
			return err
		}
		if ambiguous > 0 {
			_, err := tx.ExecContext(ctx, "UPDATE mission_work_items SET state='needs_reconciliation',next_action='confirm_process_stopped' WHERE "+workWhere, k.args()...)
			return err
		}
		_, err := tx.ExecContext(ctx, "DELETE FROM mission_admissions WHERE "+workWhere, k.args()...)
		return err
	})
}
