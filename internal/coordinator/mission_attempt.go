package coordinator

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// MissionAttemptKey is stable across receipt paths, retries and weekly threads.
type MissionAttemptKey struct {
	MissionID  string `json:"mission_id"`
	WorkItemID string `json:"work_item_id"`
	StageID    string `json:"stage_id"`
}
type MissionAttemptSpec struct {
	MissionID     string `json:"mission_id"`
	WorkItemID    string `json:"work_item_id"`
	StageID       string `json:"stage_id"`
	AttemptID     string `json:"attempt_id"`
	RequestDigest string `json:"request_digest"`
	RequestJSON   string `json:"request_json"`
}

func (s MissionAttemptSpec) Key() MissionAttemptKey {
	return MissionAttemptKey{s.MissionID, s.WorkItemID, s.StageID}
}

type MissionAttempt struct {
	MissionAttemptSpec
	State         string `json:"state"`
	OwnerToken    string `json:"-"` // fencing credential is never printed by status
	LeaseUntil    int64  `json:"lease_until"`
	Version       int64  `json:"version"`
	Outcome       string `json:"outcome,omitempty"`
	OutcomeDigest string `json:"outcome_digest,omitempty"`
}

var ErrMissionAttemptConflict = errors.New("mission attempt conflict: inspect status; do not retry execution blindly")

const missionNow = "CAST(strftime('%s','now') AS INTEGER)"

func (s *SQLiteStore) migrateMissionAttempts() error {
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS mission_attempts (
 mission_id TEXT NOT NULL, work_item_id TEXT NOT NULL, stage_id TEXT NOT NULL,
 attempt_id TEXT NOT NULL, request_digest TEXT NOT NULL, request_json TEXT NOT NULL,
 state TEXT NOT NULL, owner_token TEXT NOT NULL, lease_until INTEGER NOT NULL,
 version INTEGER NOT NULL DEFAULT 1, outcome TEXT NOT NULL DEFAULT '', outcome_digest TEXT NOT NULL DEFAULT '',
 PRIMARY KEY(mission_id,work_item_id,stage_id)
 );`)
	return err
}
func (k MissionAttemptKey) args() []any { return []any{k.MissionID, k.WorkItemID, k.StageID} }
func (k MissionAttemptKey) validate() error {
	for _, id := range []string{k.MissionID, k.WorkItemID, k.StageID} {
		if !missionIDValid(id) {
			return fmt.Errorf("invalid mission attempt key")
		}
	}
	return nil
}
func missionIDValid(id string) bool {
	if len(id) < 1 || len(id) > 128 || id == "." || id == ".." {
		return false
	}
	for _, c := range id {
		if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_.-", c) {
			return false
		}
	}
	return true
}
func (s MissionAttemptSpec) validate() error {
	if err := s.Key().validate(); err != nil {
		return err
	}
	digest, err := hex.DecodeString(s.RequestDigest)
	if !missionIDValid(s.AttemptID) || err != nil || len(digest) != 32 || len(s.RequestJSON) > 1024*1024 || !json.Valid([]byte(s.RequestJSON)) {
		return fmt.Errorf("invalid mission attempt specification")
	}
	expected := fmt.Sprintf("%x", sha256.Sum256([]byte(s.RequestJSON)))
	if s.RequestDigest != expected {
		return fmt.Errorf("request digest does not match persisted request")
	}
	return nil
}

// ClaimMissionAttempt creates or reclaims only unstarted, expired work. The SQL
// conditional insert/update is atomic across processes. Running work never retries.
func (s *SQLiteStore) ClaimMissionAttempt(ctx context.Context, spec MissionAttemptSpec, leaseSeconds int) (*MissionAttempt, error) {
	if err := spec.validate(); err != nil {
		return nil, err
	}
	if leaseSeconds < 5 || leaseSeconds > 1800 {
		return nil, fmt.Errorf("lease seconds must be 5..1800")
	}
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return nil, err
	}
	owner := hex.EncodeToString(token)
	result, err := s.db.ExecContext(ctx, `INSERT INTO mission_attempts
 (mission_id,work_item_id,stage_id,attempt_id,request_digest,request_json,state,owner_token,lease_until)
 VALUES (?,?,?,?,?,?,'prepared',?,`+missionNow+`+?)
 ON CONFLICT(mission_id,work_item_id,stage_id) DO UPDATE SET
 owner_token=excluded.owner_token,lease_until=excluded.lease_until,version=mission_attempts.version+1
 WHERE mission_attempts.state='prepared' AND mission_attempts.lease_until<=`+missionNow+`
 AND mission_attempts.attempt_id=excluded.attempt_id AND mission_attempts.request_digest=excluded.request_digest
 AND mission_attempts.request_json=excluded.request_json`, spec.MissionID, spec.WorkItemID, spec.StageID, spec.AttemptID, spec.RequestDigest, spec.RequestJSON, owner, leaseSeconds)
	if err := missionChanged(result, err); err != nil {
		return nil, err
	}
	a, err := s.GetMissionAttempt(ctx, spec.Key())
	if err != nil {
		return nil, err
	}
	if a.OwnerToken != owner {
		return nil, ErrMissionAttemptConflict
	}
	return a, nil
}

const missionWhere = "mission_id=? AND work_item_id=? AND stage_id=?"

func (s *SQLiteStore) GetMissionAttempt(ctx context.Context, key MissionAttemptKey) (*MissionAttempt, error) {
	if err := key.validate(); err != nil {
		return nil, err
	}
	a := &MissionAttempt{}
	err := s.db.QueryRowContext(ctx, `SELECT mission_id,work_item_id,stage_id,attempt_id,request_digest,request_json,state,owner_token,lease_until,version,outcome,outcome_digest FROM mission_attempts WHERE `+missionWhere, key.args()...).Scan(&a.MissionID, &a.WorkItemID, &a.StageID, &a.AttemptID, &a.RequestDigest, &a.RequestJSON, &a.State, &a.OwnerToken, &a.LeaseUntil, &a.Version, &a.Outcome, &a.OutcomeDigest)
	if err != nil {
		return nil, err
	}
	return a, nil
}
func missionChanged(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrMissionAttemptConflict
	}
	return nil
}
func (s *SQLiteStore) StartMissionAttempt(ctx context.Context, key MissionAttemptKey, owner string) error {
	if err := key.validate(); err != nil {
		return err
	}
	args := append(key.args(), owner)
	return missionChanged(s.db.ExecContext(ctx, `UPDATE mission_attempts SET state='running',version=version+1 WHERE `+missionWhere+` AND owner_token=? AND state='prepared' AND lease_until>`+missionNow, args...))
}
func (s *SQLiteStore) RenewMissionAttempt(ctx context.Context, key MissionAttemptKey, owner string, seconds int) error {
	if err := key.validate(); err != nil {
		return err
	}
	if seconds < 5 || seconds > 1800 {
		return fmt.Errorf("lease seconds must be 5..1800")
	}
	args := append([]any{seconds}, key.args()...)
	args = append(args, owner)
	// Lease renewal does not change the semantic revision used by attended cancel.
	return missionChanged(s.db.ExecContext(ctx, `UPDATE mission_attempts SET lease_until=`+missionNow+`+? WHERE `+missionWhere+` AND owner_token=? AND state IN ('prepared','running') AND lease_until>`+missionNow, args...))
}
func (s *SQLiteStore) CompleteMissionAttempt(ctx context.Context, key MissionAttemptKey, owner, state, outcome string) error {
	if err := key.validate(); err != nil {
		return err
	}
	if (state != "execution_completed" && state != "execution_failed") || len(outcome) > 16*1024*1024 || !json.Valid([]byte(outcome)) {
		return fmt.Errorf("invalid mission completion")
	}
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte(outcome)))
	args := append([]any{state, outcome, digest}, key.args()...)
	args = append(args, owner, state, state, digest)
	// Exact terminal redelivery is harmless, but a changed outcome or fenced owner is not.
	return missionChanged(s.db.ExecContext(ctx, `UPDATE mission_attempts SET state=?,outcome=?,outcome_digest=?,
 version=CASE WHEN state IN ('prepared','running') THEN version+1 ELSE version END
 WHERE `+missionWhere+` AND owner_token=? AND
 (((state='running' OR (state='prepared' AND ?='execution_failed')) AND lease_until>`+missionNow+`) OR (state=? AND outcome_digest=?))`, args...))
}
func (s *SQLiteStore) CancelMissionAttempt(ctx context.Context, key MissionAttemptKey, version int64) error {
	if err := key.validate(); err != nil {
		return err
	}
	if version < 1 {
		return fmt.Errorf("expected version is required")
	}
	args := append(key.args(), version)
	return missionChanged(s.db.ExecContext(ctx, `UPDATE mission_attempts SET state='cancelled',version=version+1,lease_until=0 WHERE `+missionWhere+` AND version=? AND state IN ('prepared','running','needs_reconciliation')`, args...))
}
func (s *SQLiteStore) ReconcileMissionAttempts(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx, `UPDATE mission_attempts SET state='needs_reconciliation',version=version+1 WHERE state='running' AND lease_until<=`+missionNow)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
