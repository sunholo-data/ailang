package observatory

import (
	"database/sql"
	"fmt"
)

// migrateV17 adds chain_stages.cost_provenance — whether a stage's `cost`
// is money anyone was actually charged.
//
// Why (found 2026-07-30): ClassifyStageCost treated any `cost > 0` as
// authoritative reported spend, but the agent CLIs that authenticate by
// subscription (codex on `auth_mode: chatgpt`, claude on OAuth) still emit a
// non-zero cost figure that is never billed. A cohort mixing those with
// genuinely metered OpenRouter/Vertex stages blended notional and real dollars
// under one label — directly under the v1.0 `cost-per-verified-success` KPI,
// whose numerator is defined as attributable METERED dollars.
//
// Values mirror executor.CostProvenance: "metered", "list-price-equivalent",
// "free-local", "unknown". NULL/empty means the stage predates this column;
// it reads as unknown provenance, NEVER as metered — backfilling a guess is
// exactly the fabrication this column exists to stop.
func migrateV17(db *sql.DB, currentVersion int) (int, error) {
	_, err := db.Exec("ALTER TABLE chain_stages ADD COLUMN cost_provenance TEXT")
	if err != nil && !isColumnAlreadyExists(err) {
		return currentVersion, fmt.Errorf("v17 add chain_stages.cost_provenance: %w", err)
	}
	if _, err := db.Exec("INSERT INTO schema_version (version) VALUES (17)"); err != nil {
		return currentVersion, fmt.Errorf("failed to record version 17: %w", err)
	}
	return 17, nil
}
