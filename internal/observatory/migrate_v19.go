package observatory

import (
	"database/sql"
	"fmt"
)

// migrateV19 adds execution_chains.reason (M-COMPLETION-PATH-PARITY M4).
//
// A chain can now end without a verdict. 315 chains in production sat "active"
// — the oldest since April — because the cloud completion path performed no
// chain progression at all, so nothing ever moved them off it.
//
// Marking those completed or failed would assert an outcome nobody observed, and
// would poison the first dataset anyone uses to check whether the fix worked. The
// reconciler marks them abandoned instead, and this column records WHY, so a
// reader six months from now can tell "we gave up on this in the 2026-09
// reconciliation" from "this genuinely failed".
func migrateV19(db *sql.DB, currentVersion int) (int, error) {
	_, err := db.Exec("ALTER TABLE execution_chains ADD COLUMN reason TEXT")
	if err != nil && !isColumnAlreadyExists(err) {
		return currentVersion, fmt.Errorf("v19 add execution_chains.reason: %w", err)
	}
	if _, err := db.Exec("INSERT INTO schema_version (version) VALUES (19)"); err != nil {
		return currentVersion, fmt.Errorf("failed to record version 19: %w", err)
	}
	return 19, nil
}
