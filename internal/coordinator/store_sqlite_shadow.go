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
// same JSON, so the two stores cannot drift field by field. `complete` and
// `grade_state` are copied out of the document only so the grade sweep can
// query them.
func (s *SQLiteStore) migrateShadowComparisons() error {
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS shadow_comparisons (
		parent_id TEXT PRIMARY KEY,
		data TEXT NOT NULL,
		complete INTEGER NOT NULL DEFAULT 0,
		grade_state TEXT NOT NULL DEFAULT '',
		updated_at DATETIME NOT NULL
	)`)
	return err
}

// shadowTx reads, transforms and writes one comparison inside a transaction.
// fn returns the next value and whether to write it.
func (s *SQLiteStore) shadowTx(ctx context.Context, parentID string, fn func(ShadowComparison, bool) (ShadowComparison, bool, error)) (*ShadowComparison, bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = tx.Rollback() }()

	cur := ShadowComparison{ParentID: parentID}
	found := true
	var data string
	switch err := tx.QueryRowContext(ctx, `SELECT data FROM shadow_comparisons WHERE parent_id = ?`, parentID).Scan(&data); {
	case errors.Is(err, sql.ErrNoRows):
		found = false
	case err != nil:
		return nil, false, err
	default:
		if err := json.Unmarshal([]byte(data), &cur); err != nil {
			return nil, false, fmt.Errorf("shadow comparison %s: %w", parentID, err)
		}
	}
	next, write, err := fn(cur, found)
	if err != nil || !write {
		return &cur, false, err
	}
	raw, err := json.Marshal(next)
	if err != nil {
		return nil, false, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO shadow_comparisons (parent_id, data, complete, grade_state, updated_at) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(parent_id) DO UPDATE SET data = excluded.data, complete = excluded.complete,
		grade_state = excluded.grade_state, updated_at = excluded.updated_at`,
		parentID, string(raw), next.Complete(), next.GradeState, next.UpdatedAt); err != nil {
		return nil, false, err
	}
	return &next, true, tx.Commit()
}

// RecordShadowRun merges one side into the parent's comparison.
func (s *SQLiteStore) RecordShadowRun(ctx context.Context, parentID, side string, run ShadowRunResult) (*ShadowComparison, error) {
	c, _, err := s.shadowTx(ctx, parentID, func(cur ShadowComparison, _ bool) (ShadowComparison, bool, error) {
		next, err := MergeShadowRun(cur, side, run, time.Now().UTC())
		return next, err == nil, err
	})
	return c, err
}

// UpdateShadowComparison applies fn to an existing comparison atomically.
func (s *SQLiteStore) UpdateShadowComparison(ctx context.Context, parentID string, fn func(*ShadowComparison) bool) (bool, error) {
	_, wrote, err := s.shadowTx(ctx, parentID, func(cur ShadowComparison, found bool) (ShadowComparison, bool, error) {
		if !found {
			return cur, false, fmt.Errorf("%w: %s", ErrShadowComparisonNotFound, parentID)
		}
		if !fn(&cur) {
			return cur, false, nil
		}
		cur.UpdatedAt = time.Now().UTC()
		return cur, true, nil
	})
	return wrote, err
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

// ListShadowComparisonsByGradeState lists complete comparisons in one grade state.
func (s *SQLiteStore) ListShadowComparisonsByGradeState(ctx context.Context, state string, limit int) ([]*ShadowComparison, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.queryShadow(ctx, `SELECT data FROM shadow_comparisons WHERE complete = 1 AND grade_state = ?
		ORDER BY updated_at ASC LIMIT ?`, state, limit)
}

// ListShadowComparisonsSince lists every comparison updated at or after since.
func (s *SQLiteStore) ListShadowComparisonsSince(ctx context.Context, since time.Time) ([]*ShadowComparison, error) {
	return s.queryShadow(ctx, `SELECT data FROM shadow_comparisons WHERE updated_at >= ? ORDER BY updated_at ASC`, since.UTC())
}

func (s *SQLiteStore) queryShadow(ctx context.Context, query string, args ...any) ([]*ShadowComparison, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*ShadowComparison
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var c ShadowComparison
		if err := json.Unmarshal([]byte(data), &c); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}
