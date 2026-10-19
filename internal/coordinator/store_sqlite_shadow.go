package coordinator

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// shadow_comparisons holds one JSON document per primary task
// (M-AILANG-LANE-EXECUTOR-SHADOW M4). A document rather than columns: the grade
// layers add fields milestone by milestone, and the Firestore twin stores the
// same JSON, so the two stores cannot drift field by field.
func (s *SQLiteStore) migrateShadowComparisons() error {
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS shadow_comparisons (
		parent_id TEXT PRIMARY KEY,
		data TEXT NOT NULL,
		updated_at DATETIME NOT NULL
	)`)
	return err
}

// RecordShadowRun merges one side into the parent's comparison inside a
// transaction.
func (s *SQLiteStore) RecordShadowRun(ctx context.Context, parentID, side string, run ShadowRunResult) (*ShadowComparison, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	cur := ShadowComparison{ParentID: parentID}
	var data string
	switch err := tx.QueryRowContext(ctx, `SELECT data FROM shadow_comparisons WHERE parent_id = ?`, parentID).Scan(&data); {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return nil, err
	default:
		if err := json.Unmarshal([]byte(data), &cur); err != nil {
			return nil, fmt.Errorf("shadow comparison %s: %w", parentID, err)
		}
	}
	next, err := MergeShadowRun(cur, side, run, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(next)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO shadow_comparisons (parent_id, data, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(parent_id) DO UPDATE SET data = excluded.data, updated_at = excluded.updated_at`,
		parentID, string(raw), next.UpdatedAt); err != nil {
		return nil, err
	}
	return &next, tx.Commit()
}

// GetShadowComparison returns ErrShadowComparisonNotFound for an unknown parent.
func (s *SQLiteStore) GetShadowComparison(ctx context.Context, parentID string) (*ShadowComparison, error) {
	var data string
	err := s.db.QueryRowContext(ctx, `SELECT data FROM shadow_comparisons WHERE parent_id = ?`, parentID).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: %s", ErrShadowComparisonNotFound, parentID)
	}
	if err != nil {
		return nil, err
	}
	var c ShadowComparison
	if err := json.Unmarshal([]byte(data), &c); err != nil {
		return nil, fmt.Errorf("shadow comparison %s: %w", parentID, err)
	}
	return &c, nil
}
