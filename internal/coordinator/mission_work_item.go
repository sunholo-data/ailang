package coordinator

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

type MissionWorkItemKey struct {
	MissionID  string `json:"mission_id"`
	WorkItemID string `json:"work_item_id"`
}
type MissionWorkItemSpec struct {
	MissionWorkItemKey
	SpecJSON       string   `json:"spec_json"`
	SpecDigest     string   `json:"spec_digest"`
	StageIDs       []string `json:"stage_ids"`
	TimeoutSeconds int      `json:"timeout_seconds"`
}
type MissionWorkItem struct {
	MissionWorkItemSpec
	State      string `json:"state"`
	ReasonCode string `json:"reason_code"`
	NextAction string `json:"next_action"`
	OwnerToken string `json:"-"`
	LeaseUntil int64  `json:"lease_until"`
	Version    int64  `json:"version"`
	Deadline   int64  `json:"deadline"`
	NextStage  int    `json:"next_stage"`
}
type MissionStageAcceptance struct {
	MissionAttemptKey
	RequestDigest    string `json:"request_digest"`
	OutcomeDigest    string `json:"outcome_digest"`
	AcceptanceJSON   string `json:"acceptance_json"`
	AcceptanceDigest string `json:"acceptance_digest"`
}

func (k MissionWorkItemKey) args() []any { return []any{k.MissionID, k.WorkItemID} }
func (k MissionWorkItemKey) validate() error {
	if !missionIDValid(k.MissionID) || !missionIDValid(k.WorkItemID) {
		return fmt.Errorf("invalid work item key")
	}
	return nil
}
func workToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	return hex.EncodeToString(b), err
}
func workJSONValid(data, digest string, max int) bool {
	return len(data) <= max && json.Valid([]byte(data)) && digest == fmt.Sprintf("%x", sha256.Sum256([]byte(data)))
}
func workLease(seconds int) error {
	if seconds < 5 || seconds > 1800 {
		return fmt.Errorf("lease seconds must be 5..1800")
	}
	return nil
}

const workWhere = "mission_id=? AND work_item_id=?"
const workFields = "mission_id,work_item_id,spec_json,spec_digest,stage_ids,timeout_seconds,state,reason_code,next_action,owner_token,lease_until,version,deadline,next_stage"

func (s *SQLiteStore) migrateMissionWorkItems() error {
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS mission_work_items (
 mission_id TEXT NOT NULL,work_item_id TEXT NOT NULL,spec_json TEXT NOT NULL,spec_digest TEXT NOT NULL,
 stage_ids TEXT NOT NULL,timeout_seconds INTEGER NOT NULL,state TEXT NOT NULL DEFAULT 'ready',
 reason_code TEXT NOT NULL DEFAULT '',next_action TEXT NOT NULL DEFAULT '',owner_token TEXT NOT NULL,
 lease_until INTEGER NOT NULL,version INTEGER NOT NULL DEFAULT 1,deadline INTEGER NOT NULL,next_stage INTEGER NOT NULL DEFAULT 0,
 PRIMARY KEY(mission_id,work_item_id));
 CREATE TABLE IF NOT EXISTS mission_admissions(mission_id TEXT PRIMARY KEY,work_item_id TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS mission_stage_deadlines(mission_id TEXT NOT NULL,work_item_id TEXT NOT NULL,stage_id TEXT NOT NULL,timeout_seconds INTEGER NOT NULL,deadline INTEGER NOT NULL,PRIMARY KEY(mission_id,work_item_id,stage_id));
 CREATE TABLE IF NOT EXISTS mission_work_children(mission_id TEXT NOT NULL,work_item_id TEXT NOT NULL,stage_id TEXT NOT NULL,parent_owner TEXT NOT NULL,child_owner TEXT NOT NULL,PRIMARY KEY(mission_id,work_item_id,stage_id));
 CREATE TABLE IF NOT EXISTS mission_stage_acceptances(mission_id TEXT NOT NULL,work_item_id TEXT NOT NULL,stage_id TEXT NOT NULL,
 request_digest TEXT NOT NULL,outcome_digest TEXT NOT NULL,acceptance_json TEXT NOT NULL,acceptance_digest TEXT NOT NULL,
 PRIMARY KEY(mission_id,work_item_id,stage_id));`)
	return err
}

// workTx takes the SQLite write lock before any read, avoiding read-to-write upgrade races.
func (s *SQLiteStore) workTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if _, err = tx.ExecContext(ctx, "UPDATE mission_admissions SET work_item_id=work_item_id WHERE 0"); err != nil {
		return err
	}
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
func scanWork(row *sql.Row) (*MissionWorkItem, error) {
	w := &MissionWorkItem{}
	var stages string
	err := row.Scan(&w.MissionID, &w.WorkItemID, &w.SpecJSON, &w.SpecDigest, &stages, &w.TimeoutSeconds, &w.State, &w.ReasonCode, &w.NextAction, &w.OwnerToken, &w.LeaseUntil, &w.Version, &w.Deadline, &w.NextStage)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(stages), &w.StageIDs)
	return w, err
}
func (s *SQLiteStore) GetMissionWorkItem(ctx context.Context, k MissionWorkItemKey) (*MissionWorkItem, error) {
	if err := k.validate(); err != nil {
		return nil, err
	}
	return scanWork(s.db.QueryRowContext(ctx, "SELECT "+workFields+" FROM mission_work_items WHERE "+workWhere, k.args()...))
}
func workFence(ctx context.Context, tx *sql.Tx, k MissionWorkItemKey, owner string) (*MissionWorkItem, error) {
	args := append(k.args(), owner)
	w, err := scanWork(tx.QueryRowContext(ctx, "SELECT "+workFields+" FROM mission_work_items WHERE "+workWhere+" AND owner_token=? AND lease_until>"+missionNow+" AND state NOT IN ('completed','failed','cancelled','needs_reconciliation')", args...))
	if err == sql.ErrNoRows {
		return nil, ErrMissionAttemptConflict
	}
	return w, err
}
func (s *SQLiteStore) ClaimMissionWorkItem(ctx context.Context, spec MissionWorkItemSpec, seconds int) (*MissionWorkItem, error) {
	if err := spec.MissionWorkItemKey.validate(); err != nil {
		return nil, err
	}
	if err := workLease(seconds); err != nil {
		return nil, err
	}
	if !workJSONValid(spec.SpecJSON, spec.SpecDigest, 4*1024*1024) || spec.TimeoutSeconds < 1 || spec.TimeoutSeconds > 7200 || len(spec.StageIDs) < 1 || len(spec.StageIDs) > 4 {
		return nil, fmt.Errorf("invalid work item snapshot")
	}
	seen := map[string]bool{}
	for _, id := range spec.StageIDs {
		if !missionIDValid(id) || seen[id] {
			return nil, fmt.Errorf("invalid stage ids")
		}
		seen[id] = true
	}
	stages, _ := json.Marshal(spec.StageIDs)
	owner, err := workToken()
	if err != nil {
		return nil, err
	}
	err = s.workTx(ctx, func(tx *sql.Tx) error {
		var orphan int
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM mission_attempts WHERE mission_id=? AND state IN ('prepared','running','needs_reconciliation') AND NOT EXISTS(SELECT 1 FROM mission_work_items WHERE mission_id=? AND work_item_id=?)`, spec.MissionID, spec.MissionID, spec.WorkItemID).Scan(&orphan); err != nil {
			return err
		}
		if orphan > 0 {
			return ErrMissionAttemptConflict
		}
		if err := missionChanged(tx.ExecContext(ctx, `INSERT INTO mission_admissions(mission_id,work_item_id) VALUES(?,?) ON CONFLICT(mission_id) DO UPDATE SET work_item_id=excluded.work_item_id WHERE mission_admissions.work_item_id=excluded.work_item_id`, spec.MissionID, spec.WorkItemID)); err != nil {
			return err
		}
		return missionChanged(tx.ExecContext(ctx, `INSERT INTO mission_work_items(mission_id,work_item_id,spec_json,spec_digest,stage_ids,timeout_seconds,owner_token,lease_until,deadline)
 VALUES(?,?,?,?,?,?,?,`+missionNow+`+?,`+missionNow+`+?) ON CONFLICT(mission_id,work_item_id) DO UPDATE SET owner_token=excluded.owner_token,lease_until=excluded.lease_until,version=mission_work_items.version+1
 WHERE mission_work_items.lease_until<=`+missionNow+` AND mission_work_items.state NOT IN ('completed','failed','cancelled')
 AND mission_work_items.spec_json=excluded.spec_json AND mission_work_items.spec_digest=excluded.spec_digest AND mission_work_items.stage_ids=excluded.stage_ids AND mission_work_items.timeout_seconds=excluded.timeout_seconds`, spec.MissionID, spec.WorkItemID, spec.SpecJSON, spec.SpecDigest, string(stages), spec.TimeoutSeconds, owner, seconds, spec.TimeoutSeconds))
	})
	if err != nil {
		return nil, err
	}
	return s.GetMissionWorkItem(ctx, spec.MissionWorkItemKey)
}
func (s *SQLiteStore) RenewMissionWorkItem(ctx context.Context, k MissionWorkItemKey, owner string, seconds int) error {
	if err := workLease(seconds); err != nil {
		return err
	}
	args := append([]any{seconds}, k.args()...)
	args = append(args, owner)
	return missionChanged(s.db.ExecContext(ctx, "UPDATE mission_work_items SET lease_until="+missionNow+"+? WHERE "+workWhere+" AND owner_token=? AND lease_until>"+missionNow+" AND state NOT IN ('completed','failed','cancelled')", args...))
}
func (s *SQLiteStore) WaitMissionWorkItem(ctx context.Context, k MissionWorkItemKey, owner, reason, nextAction string) error {
	return s.workTx(ctx, func(tx *sql.Tx) error {
		if _, err := workFence(ctx, tx, k, owner); err != nil {
			return err
		}
		return workWait(ctx, tx, k, reason, nextAction)
	})
}
func workWait(ctx context.Context, tx *sql.Tx, k MissionWorkItemKey, reason, nextAction string) error {
	if reason == "" || nextAction == "" {
		return fmt.Errorf("wait reason and next action required")
	}
	args := append([]any{reason, nextAction}, k.args()...)
	return missionChanged(tx.ExecContext(ctx, "UPDATE mission_work_items SET state='waiting',reason_code=?,next_action=?,lease_until=0,version=version+1 WHERE "+workWhere, args...))
}
func (s *SQLiteStore) MarkMissionWorkItemValidating(ctx context.Context, k MissionWorkItemKey, owner string) error {
	return s.workTx(ctx, func(tx *sql.Tx) error {
		if _, err := workFence(ctx, tx, k, owner); err != nil {
			return err
		}
		return missionChanged(tx.ExecContext(ctx, "UPDATE mission_work_items SET state='validating',version=version+1 WHERE "+workWhere, k.args()...))
	})
}
