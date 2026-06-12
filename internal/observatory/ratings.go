// Package observatory: ELO rating persistence (M-EVAL-RATING-EFFICIENCY part 2).
//
// Stores the fitted per-benchmark difficulty and per-model capability ratings
// produced by internal/eval_harness.FitFromTrials. The rating math lives in
// eval_harness; this layer is pure storage (it takes plain maps, so it does not
// import eval_harness and cannot create an import cycle).
package observatory

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// BenchmarkRating is one row of benchmark_ratings (rating = derived difficulty,
// per mode).
type BenchmarkRating struct {
	BenchmarkID string
	Mode        string
	Rating      float64
	NTrials     int
	LastUpdated time.Time
}

// ModelRating is one row of model_ratings (rating = capability, per mode).
type ModelRating struct {
	ModelID     string
	Mode        string
	Rating      float64
	NTrials     int
	KFactor     int
	LastUpdated time.Time
}

// SaveRatings upserts a full set of fitted model and benchmark ratings for ONE
// mode ('standard'|'agent') in a single transaction. Ratings are mode-separated
// because standard and agent are different difficulty regimes; callers fit and
// persist each mode independently. trial maps (id -> N) are optional (nil → 0).
func SaveRatings(ctx context.Context, db *sql.DB, mode string, modelRatings, benchRatings map[string]float64, modelTrials, benchTrials map[string]int) error {
	now := time.Now()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin ratings save: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for id, r := range benchRatings {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO benchmark_ratings (benchmark_id, mode, rating, n_trials, last_updated)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(benchmark_id, mode) DO UPDATE SET
				rating = excluded.rating, n_trials = excluded.n_trials, last_updated = excluded.last_updated
		`, id, mode, r, benchTrials[id], now); err != nil {
			return fmt.Errorf("upsert benchmark rating %s/%s: %w", id, mode, err)
		}
	}
	for id, r := range modelRatings {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO model_ratings (model_id, mode, rating, n_trials, last_updated)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(model_id, mode) DO UPDATE SET
				rating = excluded.rating, n_trials = excluded.n_trials, last_updated = excluded.last_updated
		`, id, mode, r, modelTrials[id], now); err != nil {
			return fmt.Errorf("upsert model rating %s/%s: %w", id, mode, err)
		}
	}
	return tx.Commit()
}

// LoadBenchmarkRatings returns benchmark ratings for a mode, hardest first.
func LoadBenchmarkRatings(ctx context.Context, db *sql.DB, mode string) ([]BenchmarkRating, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT benchmark_id, mode, rating, n_trials, last_updated
		FROM benchmark_ratings WHERE mode = ? ORDER BY rating DESC, benchmark_id`, mode)
	if err != nil {
		return nil, fmt.Errorf("query benchmark_ratings: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []BenchmarkRating
	for rows.Next() {
		var b BenchmarkRating
		if err := rows.Scan(&b.BenchmarkID, &b.Mode, &b.Rating, &b.NTrials, &b.LastUpdated); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// LoadModelRatings returns model ratings for a mode, strongest first.
func LoadModelRatings(ctx context.Context, db *sql.DB, mode string) ([]ModelRating, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT model_id, mode, rating, n_trials, k_factor, last_updated
		FROM model_ratings WHERE mode = ? ORDER BY rating DESC, model_id`, mode)
	if err != nil {
		return nil, fmt.Errorf("query model_ratings: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []ModelRating
	for rows.Next() {
		var m ModelRating
		if err := rows.Scan(&m.ModelID, &m.Mode, &m.Rating, &m.NTrials, &m.KFactor, &m.LastUpdated); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
