package coordinator

import (
	"context"
	"database/sql"
	"fmt"
)

// BeginMissionStage durably starts a stage's time budget before preflight. Resume
// returns the first deadline, even after quota waiting or before a child exists.
func (s *SQLiteStore) BeginMissionStage(ctx context.Context, k MissionWorkItemKey, owner, stageID string, timeoutSeconds int) (int64, error) {
	if timeoutSeconds < 1 || timeoutSeconds > 1800 {
		return 0, fmt.Errorf("stage timeout seconds must be 1..1800")
	}
	var deadline int64
	err := s.workTx(ctx, func(tx *sql.Tx) error {
		w, err := workFence(ctx, tx, k, owner)
		if err != nil {
			return err
		}
		key := MissionAttemptKey{MissionID: k.MissionID, WorkItemID: k.WorkItemID, StageID: stageID}
		if !workChildMatches(w, key) {
			return ErrMissionAttemptConflict
		}
		if err = missionChanged(tx.ExecContext(ctx, `INSERT INTO mission_stage_deadlines(mission_id,work_item_id,stage_id,timeout_seconds,deadline) VALUES(?,?,?,?,MIN(`+missionNow+`+?,?)) ON CONFLICT(mission_id,work_item_id,stage_id) DO UPDATE SET deadline=mission_stage_deadlines.deadline WHERE mission_stage_deadlines.timeout_seconds=excluded.timeout_seconds`, k.MissionID, k.WorkItemID, stageID, timeoutSeconds, timeoutSeconds, w.Deadline)); err != nil {
			return err
		}
		return tx.QueryRowContext(ctx, "SELECT deadline FROM mission_stage_deadlines WHERE "+missionWhere, key.args()...).Scan(&deadline)
	})
	return deadline, err
}
