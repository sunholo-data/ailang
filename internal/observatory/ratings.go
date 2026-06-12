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

// BenchmarkRating is one row of benchmark_ratings (rating = derived difficulty).
type BenchmarkRating struct {
	BenchmarkID string
	Rating      float64
	NTrials     int
	LastUpdated time.Time
}

// ModelRating is one row of model_ratings (rating = capability).
type ModelRating struct {
	ModelID     string
	Rating      float64
	NTrials     int
	KFactor     int
	LastUpdated time.Time
}

// SaveRatings upserts a full set of fitted model and benchmark ratings in one
// transaction. trialCounts (id -> N) is optional context; pass nil to record 0.
func SaveRatings(ctx context.Context, db *sql.DB, modelRatings, benchRatings map[string]float64, modelTrials, benchTrials map[string]int) error {
	now := time.Now()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin ratings save: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for id, r := range benchRatings {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO benchmark_ratings (benchmark_id, rating, n_trials, last_updated)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(benchmark_id) DO UPDATE SET
				rating = excluded.rating, n_trials = excluded.n_trials, last_updated = excluded.last_updated
		`, id, r, benchTrials[id], now); err != nil {
			return fmt.Errorf("upsert benchmark rating %s: %w", id, err)
		}
	}
	for id, r := range modelRatings {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO model_ratings (model_id, rating, n_trials, last_updated)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(model_id) DO UPDATE SET
				rating = excluded.rating, n_trials = excluded.n_trials, last_updated = excluded.last_updated
		`, id, r, modelTrials[id], now); err != nil {
			return fmt.Errorf("upsert model rating %s: %w", id, err)
		}
	}
	return tx.Commit()
}

// LoadBenchmarkRatings returns all benchmark ratings, hardest first.
func LoadBenchmarkRatings(ctx context.Context, db *sql.DB) ([]BenchmarkRating, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT benchmark_id, rating, n_trials, last_updated
		FROM benchmark_ratings ORDER BY rating DESC, benchmark_id`)
	if err != nil {
		return nil, fmt.Errorf("query benchmark_ratings: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []BenchmarkRating
	for rows.Next() {
		var b BenchmarkRating
		if err := rows.Scan(&b.BenchmarkID, &b.Rating, &b.NTrials, &b.LastUpdated); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// LoadModelRatings returns all model ratings, strongest first.
func LoadModelRatings(ctx context.Context, db *sql.DB) ([]ModelRating, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT model_id, rating, n_trials, k_factor, last_updated
		FROM model_ratings ORDER BY rating DESC, model_id`)
	if err != nil {
		return nil, fmt.Errorf("query model_ratings: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []ModelRating
	for rows.Next() {
		var m ModelRating
		if err := rows.Scan(&m.ModelID, &m.Rating, &m.NTrials, &m.KFactor, &m.LastUpdated); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
