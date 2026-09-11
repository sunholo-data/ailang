package coordinator

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// MissionStopAttestation records an operator assertion, not inferred process death.
// Children identify the exact attempts and versions observed in the same transaction.
type MissionStopAttestation struct {
	MissionWorkItemKey
	ExpectedVersion int64                 `json:"expected_version"`
	Attestation     string                `json:"attestation"`
	RecordedAt      int64                 `json:"recorded_at"`
	Children        []MissionStoppedChild `json:"children"`
}
type MissionStoppedChild struct {
	StageID   string `json:"stage_id"`
	AttemptID string `json:"attempt_id"`
	Version   int64  `json:"version"`
}

func (s *SQLiteStore) ConfirmMissionWorkItemStoppedAttested(ctx context.Context, k MissionWorkItemKey, version int64, attestation string) error {
	if err := k.validate(); err != nil {
		return err
	}
	if version < 1 || strings.TrimSpace(attestation) == "" || len(attestation) > 2048 {
		return fmt.Errorf("expected version and explicit operator attestation (1..2048 bytes) required")
	}
	return s.workTx(ctx, func(tx *sql.Tx) error {
		// Lazily add this optional evidence table; transaction rollback also rolls back DDL.
		if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS mission_stop_attestations (mission_id TEXT NOT NULL,work_item_id TEXT NOT NULL,expected_version INTEGER NOT NULL,evidence_json TEXT NOT NULL,PRIMARY KEY(mission_id,work_item_id,expected_version))`); err != nil {
			return err
		}
		evidence := MissionStopAttestation{MissionWorkItemKey: k, ExpectedVersion: version, Attestation: attestation, Children: []MissionStoppedChild{}}
		if err := tx.QueryRowContext(ctx, "SELECT "+missionNow).Scan(&evidence.RecordedAt); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, "SELECT stage_id,attempt_id,version FROM mission_attempts WHERE "+workWhere+" AND state='needs_reconciliation' ORDER BY stage_id", k.args()...)
		if err != nil {
			return err
		}
		for rows.Next() {
			var child MissionStoppedChild
			if err = rows.Scan(&child.StageID, &child.AttemptID, &child.Version); err != nil {
				rows.Close()
				return err
			}
			evidence.Children = append(evidence.Children, child)
		}
		err = rows.Err()
		closeErr := rows.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		if err = confirmMissionStoppedTx(ctx, tx, k, version); err != nil {
			return err
		}
		encoded, err := json.Marshal(evidence)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "INSERT INTO mission_stop_attestations(mission_id,work_item_id,expected_version,evidence_json) VALUES(?,?,?,?)", k.MissionID, k.WorkItemID, version, string(encoded))
		return err
	})
}

func (s *SQLiteStore) GetMissionStopAttestation(ctx context.Context, k MissionWorkItemKey, version int64) (*MissionStopAttestation, error) {
	if err := k.validate(); err != nil {
		return nil, err
	}
	var encoded string
	err := s.db.QueryRowContext(ctx, "SELECT evidence_json FROM mission_stop_attestations WHERE mission_id=? AND work_item_id=? AND expected_version=?", k.MissionID, k.WorkItemID, version).Scan(&encoded)
	if err != nil {
		return nil, err
	}
	var evidence MissionStopAttestation
	if err = json.Unmarshal([]byte(encoded), &evidence); err != nil {
		return nil, err
	}
	return &evidence, nil
}

// OpenMissionExistingWriteStore never creates a database or runs migrations.
func OpenMissionExistingWriteStore(path string) (*SQLiteStore, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("runtime database path must be absolute")
	}
	db, err := sql.Open("sqlite3", sqliteFileURI(path, "mode=rw&_busy_timeout=5000"))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err = db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &SQLiteStore{db: db}, nil
}
