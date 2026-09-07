package coordinator

import (
	"context"
	"database/sql"
	"fmt"
)

func workChildMatches(w *MissionWorkItem, key MissionAttemptKey) bool {
	return key.MissionID == w.MissionID && key.WorkItemID == w.WorkItemID && w.NextStage < len(w.StageIDs) && key.StageID == w.StageIDs[w.NextStage]
}
func (s *SQLiteStore) ClaimMissionChild(ctx context.Context, k MissionWorkItemKey, owner string, spec MissionAttemptSpec, seconds int) (*MissionAttempt, error) {
	if err := spec.validate(); err != nil {
		return nil, err
	}
	if err := workLease(seconds); err != nil {
		return nil, err
	}
	token, err := workToken()
	if err != nil {
		return nil, err
	}
	err = s.workTx(ctx, func(tx *sql.Tx) error {
		w, err := workFence(ctx, tx, k, owner)
		if err != nil {
			return err
		}
		if !workChildMatches(w, spec.Key()) {
			return ErrMissionAttemptConflict
		}
		var remaining int64
		if err = tx.QueryRowContext(ctx, "SELECT deadline-"+missionNow+" FROM mission_work_items WHERE "+workWhere, k.args()...).Scan(&remaining); err != nil {
			return err
		}
		if remaining <= 0 {
			return fmt.Errorf("iteration deadline exceeded")
		}
		if err = missionChanged(tx.ExecContext(ctx, `INSERT INTO mission_attempts(mission_id,work_item_id,stage_id,attempt_id,request_digest,request_json,state,owner_token,lease_until)
 VALUES(?,?,?,?,?,?,'prepared',?,`+missionNow+`+?) ON CONFLICT(mission_id,work_item_id,stage_id) DO UPDATE SET owner_token=excluded.owner_token,lease_until=excluded.lease_until,version=mission_attempts.version+1
 WHERE mission_attempts.state='prepared' AND mission_attempts.lease_until<=`+missionNow+` AND mission_attempts.attempt_id=excluded.attempt_id AND mission_attempts.request_digest=excluded.request_digest AND mission_attempts.request_json=excluded.request_json`, spec.MissionID, spec.WorkItemID, spec.StageID, spec.AttemptID, spec.RequestDigest, spec.RequestJSON, token, seconds)); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO mission_work_children(mission_id,work_item_id,stage_id,parent_owner,child_owner) VALUES(?,?,?,?,?) ON CONFLICT(mission_id,work_item_id,stage_id) DO UPDATE SET parent_owner=excluded.parent_owner,child_owner=excluded.child_owner`, spec.MissionID, spec.WorkItemID, spec.StageID, owner, token)
		return err
	})
	if err != nil {
		return nil, err
	}
	return s.GetMissionAttempt(ctx, spec.Key())
}
func (s *SQLiteStore) StartMissionChild(ctx context.Context, k MissionWorkItemKey, parentOwner string, key MissionAttemptKey, childOwner string) error {
	return s.workTx(ctx, func(tx *sql.Tx) error {
		w, err := workFence(ctx, tx, k, parentOwner)
		if err != nil {
			return err
		}
		if !workChildMatches(w, key) {
			return ErrMissionAttemptConflict
		}
		var bound int
		bindingArgs := append(key.args(), parentOwner, childOwner)
		if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM mission_work_children WHERE "+missionWhere+" AND parent_owner=? AND child_owner=?", bindingArgs...).Scan(&bound); err != nil {
			return err
		}
		if bound != 1 {
			return ErrMissionAttemptConflict
		}
		args := append(key.args(), childOwner)
		if err = missionChanged(tx.ExecContext(ctx, "UPDATE mission_attempts SET state='running',version=version+1 WHERE "+missionWhere+" AND owner_token=? AND state='prepared' AND lease_until>"+missionNow, args...)); err != nil {
			return err
		}
		startArgs := append(k.args(), parentOwner, key.StageID)
		return missionChanged(tx.ExecContext(ctx, "UPDATE mission_work_items SET state='running',reason_code='',next_action='',version=version+1 WHERE "+workWhere+" AND owner_token=? AND lease_until>"+missionNow+" AND deadline>"+missionNow+" AND NOT EXISTS(SELECT 1 FROM mission_stage_deadlines d WHERE d.mission_id=mission_work_items.mission_id AND d.work_item_id=mission_work_items.work_item_id AND d.stage_id=? AND d.deadline<="+missionNow+")", startArgs...))
	})
}
func (s *SQLiteStore) ReleaseMissionPreparedChild(ctx context.Context, k MissionWorkItemKey, parentOwner string, key MissionAttemptKey, childOwner, reason, nextAction string) error {
	return s.workTx(ctx, func(tx *sql.Tx) error {
		w, err := workFence(ctx, tx, k, parentOwner)
		if err != nil {
			return err
		}
		if !workChildMatches(w, key) {
			return ErrMissionAttemptConflict
		}
		var bound int
		bindingArgs := append(key.args(), parentOwner, childOwner)
		if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM mission_work_children WHERE "+missionWhere+" AND parent_owner=? AND child_owner=?", bindingArgs...).Scan(&bound); err != nil {
			return err
		}
		if bound != 1 {
			return ErrMissionAttemptConflict
		}
		args := append(key.args(), childOwner)
		if err = missionChanged(tx.ExecContext(ctx, "UPDATE mission_attempts SET lease_until=0,version=version+1 WHERE "+missionWhere+" AND owner_token=? AND state='prepared' AND lease_until>"+missionNow, args...)); err != nil {
			return err
		}
		return workWait(ctx, tx, k, reason, nextAction)
	})
}
func (s *SQLiteStore) AcceptMissionStage(ctx context.Context, k MissionWorkItemKey, owner string, a MissionStageAcceptance) error {
	if !workJSONValid(a.AcceptanceJSON, a.AcceptanceDigest, 16*1024*1024) {
		return fmt.Errorf("invalid acceptance snapshot")
	}
	return s.workTx(ctx, func(tx *sql.Tx) error {
		w, err := workFence(ctx, tx, k, owner)
		if err != nil {
			return err
		}
		if a.MissionID != k.MissionID || a.WorkItemID != k.WorkItemID {
			return ErrMissionAttemptConflict
		}
		existing, err := scanAcceptance(tx.QueryRowContext(ctx, "SELECT "+acceptanceFields+" FROM mission_stage_acceptances WHERE "+missionWhere, a.MissionAttemptKey.args()...))
		if err == nil {
			if *existing == a {
				return nil
			}
			return ErrMissionAttemptConflict
		}
		if err != sql.ErrNoRows {
			return err
		}
		if !workChildMatches(w, a.MissionAttemptKey) {
			return ErrMissionAttemptConflict
		}
		var request, outcome, state string
		if err = tx.QueryRowContext(ctx, "SELECT request_digest,outcome_digest,state FROM mission_attempts WHERE "+missionWhere, a.MissionAttemptKey.args()...).Scan(&request, &outcome, &state); err != nil {
			return err
		}
		if state != "execution_completed" || request != a.RequestDigest || outcome != a.OutcomeDigest {
			return ErrMissionAttemptConflict
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO mission_stage_acceptances("+acceptanceFields+") VALUES(?,?,?,?,?,?,?)", a.MissionID, a.WorkItemID, a.StageID, a.RequestDigest, a.OutcomeDigest, a.AcceptanceJSON, a.AcceptanceDigest); err != nil {
			return err
		}
		acceptArgs := append(k.args(), owner, a.StageID)
		return missionChanged(tx.ExecContext(ctx, "UPDATE mission_work_items SET next_stage=next_stage+1,state='ready',reason_code='',next_action='',version=version+1 WHERE "+workWhere+" AND owner_token=? AND lease_until>"+missionNow+" AND deadline>"+missionNow+" AND NOT EXISTS(SELECT 1 FROM mission_stage_deadlines d WHERE d.mission_id=mission_work_items.mission_id AND d.work_item_id=mission_work_items.work_item_id AND d.stage_id=? AND d.deadline<="+missionNow+")", acceptArgs...))
	})
}

const acceptanceFields = "mission_id,work_item_id,stage_id,request_digest,outcome_digest,acceptance_json,acceptance_digest"

func scanAcceptance(row *sql.Row) (*MissionStageAcceptance, error) {
	a := &MissionStageAcceptance{}
	err := row.Scan(&a.MissionID, &a.WorkItemID, &a.StageID, &a.RequestDigest, &a.OutcomeDigest, &a.AcceptanceJSON, &a.AcceptanceDigest)
	return a, err
}
func (s *SQLiteStore) GetMissionStageAcceptance(ctx context.Context, key MissionAttemptKey) (*MissionStageAcceptance, error) {
	if err := key.validate(); err != nil {
		return nil, err
	}
	return scanAcceptance(s.db.QueryRowContext(ctx, "SELECT "+acceptanceFields+" FROM mission_stage_acceptances WHERE "+missionWhere, key.args()...))
}
