package observatory

import (
	"database/sql"
	"fmt"
)

// migrateV16 creates the ELO rating tables (M-EVAL-RATING-EFFICIENCY part 2):
// benchmark_ratings (derived difficulty), model_ratings (capability), and
// trial_history (audit trail of every rating-changing trial). Mirrors the v15
// eval_baselines pattern — every CREATE is IF NOT EXISTS, so a fresh DB that
// already has them from the base schema is a harmless no-op, and existing DBs
// past v15 get backfilled here.
func migrateV16(db *sql.DB, currentVersion int) (int, error) {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS benchmark_ratings (
			benchmark_id TEXT PRIMARY KEY,
			rating       REAL NOT NULL DEFAULT 1500.0,
			n_trials     INTEGER NOT NULL DEFAULT 0,
			last_updated TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS model_ratings (
			model_id     TEXT PRIMARY KEY,
			rating       REAL NOT NULL DEFAULT 1500.0,
			n_trials     INTEGER NOT NULL DEFAULT 0,
			k_factor     INTEGER NOT NULL DEFAULT 32,
			last_updated TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS trial_history (
			trial_id     TEXT PRIMARY KEY,
			benchmark_id TEXT NOT NULL,
			model_id     TEXT NOT NULL,
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
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return currentVersion, fmt.Errorf("v16 create ratings table: %w", err)
		}
	}
	if _, err := db.Exec("INSERT INTO schema_version (version) VALUES (16)"); err != nil {
		return currentVersion, fmt.Errorf("failed to record version 16: %w", err)
	}
	return 16, nil
}
