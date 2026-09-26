package observatory

import (
	"database/sql"
	"fmt"
)

// migrateV21 adds cache-token columns to chat_messages and chain_stages
// (M-V1-SIMPLIFY-S4 M3B).
//
// spans has carried cache_read_tokens/cache_creation_tokens since the
// receiver first decoded them, but the two tables an imported chain is made
// of never did. So ImportMotokoSession (S3 M5) decoded a run's cache tokens
// through the executor's SessionEvent and could only REPORT them — the
// numbers went to the terminal and nowhere else, and `ailang chains chat`
// showed a per-turn tokens_in that had already been paid for at cache-read
// price. The columns match spans' names and default so every reader can
// treat the three tables alike.
func migrateV21(db *sql.DB, currentVersion int) (int, error) {
	for _, stmt := range []string{
		"ALTER TABLE chat_messages ADD COLUMN cache_read_tokens INTEGER DEFAULT 0",
		"ALTER TABLE chat_messages ADD COLUMN cache_creation_tokens INTEGER DEFAULT 0",
		"ALTER TABLE chain_stages ADD COLUMN cache_read_tokens INTEGER DEFAULT 0",
		"ALTER TABLE chain_stages ADD COLUMN cache_creation_tokens INTEGER DEFAULT 0",
	} {
		if _, err := db.Exec(stmt); err != nil && !isColumnAlreadyExists(err) {
			return currentVersion, fmt.Errorf("v21 %s: %w", stmt, err)
		}
	}
	if _, err := db.Exec("INSERT INTO schema_version (version) VALUES (21)"); err != nil {
		return currentVersion, fmt.Errorf("failed to record version 21: %w", err)
	}
	return 21, nil
}
