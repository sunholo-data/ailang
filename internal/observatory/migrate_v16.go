package observatory

import (
	"database/sql"
	"fmt"
)

// ratingTableDDL is the schema for the ELO rating tables (M-EVAL-RATING-EFFICIENCY
// part 2), shared by the base Migrate() (fresh DBs) and migrateV16 (backfill for
// DBs already past v1) so the two can never drift.
//
// Ratings are MODE-SEPARATED: standard and agent are different difficulty regimes
// (agent mode saturates — every strong harness passes nearly everything), so an
// ELO fitted across both is meaningless. `mode` ('standard'|'agent') is therefore
// part of the primary key, and a model/benchmark carries one rating per mode.
var ratingTableDDL = []string{
	`CREATE TABLE IF NOT EXISTS benchmark_ratings (
		benchmark_id TEXT NOT NULL,
		mode         TEXT NOT NULL DEFAULT 'standard',
		rating       REAL NOT NULL DEFAULT 1500.0,
		n_trials     INTEGER NOT NULL DEFAULT 0,
		last_updated TIMESTAMP NOT NULL,
		PRIMARY KEY (benchmark_id, mode)
	)`,
	`CREATE TABLE IF NOT EXISTS model_ratings (
		model_id     TEXT NOT NULL,
		mode         TEXT NOT NULL DEFAULT 'standard',
		rating       REAL NOT NULL DEFAULT 1500.0,
		n_trials     INTEGER NOT NULL DEFAULT 0,
		k_factor     INTEGER NOT NULL DEFAULT 32,
		last_updated TIMESTAMP NOT NULL,
		PRIMARY KEY (model_id, mode)
	)`,
	`CREATE TABLE IF NOT EXISTS trial_history (
		trial_id     TEXT PRIMARY KEY,
		benchmark_id TEXT NOT NULL,
		model_id     TEXT NOT NULL,
		mode         TEXT NOT NULL DEFAULT 'standard',
		outcome      INTEGER NOT NULL,
		prompt_version  TEXT,
		compiler_version TEXT,
		benchmark_rating_before REAL,
		model_rating_before     REAL,
		benchmark_rating_after  REAL,
		model_rating_after      REAL,
		recorded_at  TIMESTAMP NOT NULL
	)`,
}

// migrateV16 creates the mode-separated ELO rating tables on DBs already past v1.
// Every CREATE is IF NOT EXISTS, so a fresh DB that already has them from the base
// schema is a harmless no-op.
func migrateV16(db *sql.DB, currentVersion int) (int, error) {
	for _, stmt := range ratingTableDDL {
		if _, err := db.Exec(stmt); err != nil {
			return currentVersion, fmt.Errorf("v16 create ratings table: %w", err)
		}
	}
	if _, err := db.Exec("INSERT INTO schema_version (version) VALUES (16)"); err != nil {
		return currentVersion, fmt.Errorf("failed to record version 16: %w", err)
	}
	return 16, nil
}
