package observatory

import (
	"database/sql"
	"fmt"
)

// migrateV20 adds chain_stages.quota_tokens (M-QUOTA-RATIONING-ROUTING M2).
//
// It is a SEPARATE column from tokens_in/tokens_out rather than a reuse of them,
// because `tokens > 0` is the fleet's structural marker for "this stage is metered
// and can be priced" — the M1 cost estimator has no schema flag and keys off exactly
// that. Recording a subscription run's real token count in tokens_in/tokens_out would
// make the estimator invoice a run nobody was billed for, corrupting the metered KPI
// in the act of fixing the quota one.
//
// Measured 2026-09-06: chains holds 4.55B tokens, all of them metered, and 4,979 quota
// stages recording zero BY DESIGN. So nothing in the fleet could answer "how much of
// the codex bucket have we spent this window?" — which is the one question a ration
// exists to answer, and the reason half a codex bucket went in a single day.
func migrateV20(db *sql.DB, currentVersion int) (int, error) {
	_, err := db.Exec("ALTER TABLE chain_stages ADD COLUMN quota_tokens INTEGER DEFAULT 0")
	if err != nil && !isColumnAlreadyExists(err) {
		return currentVersion, fmt.Errorf("v20 add chain_stages.quota_tokens: %w", err)
	}
	if _, err := db.Exec("INSERT INTO schema_version (version) VALUES (20)"); err != nil {
		return currentVersion, fmt.Errorf("failed to record version 20: %w", err)
	}
	return 20, nil
}
