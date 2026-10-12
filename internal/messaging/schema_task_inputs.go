package messaging

import (
	"database/sql"
	"fmt"
)

// migrateV190ToV1100 records a terminal version so startup no longer rebuilds
// the inbox table repeatedly. The preceding vocabulary refresh still runs once.
func migrateV190ToV1100(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var found int
	if err := tx.QueryRow("SELECT COUNT(*) FROM pragma_table_info('inbox_messages') WHERE name='inputs'").Scan(&found); err != nil {
		return err
	}
	if found == 0 {
		if _, err := tx.Exec("ALTER TABLE inbox_messages ADD COLUMN inputs TEXT DEFAULT '[]'"); err != nil {
			return fmt.Errorf("add inputs: %w", err)
		}
	}
	if _, err := tx.Exec("INSERT OR REPLACE INTO schema_version(version) VALUES ('1.10.0')"); err != nil {
		return err
	}
	return tx.Commit()
}
