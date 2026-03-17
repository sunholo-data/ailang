package messaging

import (
	"database/sql"
	"fmt"

	"github.com/sunholo/ailang/internal/builtins"
)

// MigrateDB applies any necessary schema migrations to an existing database.
// This is called after InitDB to ensure existing databases are up-to-date.
func MigrateDB(db *sql.DB) error {
	// Check current schema version
	var currentVersion string
	err := db.QueryRow("SELECT version FROM schema_version ORDER BY created_at DESC LIMIT 1").Scan(&currentVersion)
	if err != nil {
		// No version table or no version - assume 1.0.0
		currentVersion = "1.0.0"
	}

	// Apply migrations based on current version (chain migrations)
	if currentVersion == "1.0.0" {
		if err := migrateV100ToV110(db); err != nil {
			return fmt.Errorf("migration to v1.1.0 failed: %w", err)
		}
		currentVersion = "1.1.0"
	}

	if currentVersion == "1.1.0" {
		if err := migrateV110ToV120(db); err != nil {
			return fmt.Errorf("migration to v1.2.0 failed: %w", err)
		}
		currentVersion = "1.2.0"
	}

	if currentVersion == "1.2.0" {
		if err := migrateV120ToV130(db); err != nil {
			return fmt.Errorf("migration to v1.3.0 failed: %w", err)
		}
		currentVersion = "1.3.0"
	}

	if currentVersion == "1.3.0" {
		if err := migrateV130ToV140(db); err != nil {
			return fmt.Errorf("migration to v1.4.0 failed: %w", err)
		}
		currentVersion = "1.4.0"
	}

	if currentVersion == "1.4.0" {
		if err := migrateV140ToV150(db); err != nil {
			return fmt.Errorf("migration to v1.5.0 failed: %w", err)
		}
		currentVersion = "1.5.0"
	}

	if currentVersion == "1.5.0" {
		if err := migrateV150ToV160(db); err != nil {
			return fmt.Errorf("migration to v1.6.0 failed: %w", err)
		}
		currentVersion = "1.6.0"
	}

	if currentVersion == "1.6.0" {
		if err := migrateV160ToV170(db); err != nil {
			return fmt.Errorf("migration to v1.7.0 failed: %w", err)
		}
		currentVersion = "1.7.0"
	}

	if currentVersion == "1.7.0" {
		if err := migrateV170ToV180(db); err != nil {
			return fmt.Errorf("migration to v1.8.0 failed: %w", err)
		}
	}

	return nil
}

// migrateV100ToV110 adds GitHub integration columns to inbox_messages
func migrateV100ToV110(db *sql.DB) error {
	// Add new columns (SQLite ADD COLUMN is safe if column already exists - it will error)
	// We use a transaction and check for column existence first
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Check if category column exists
	var colName string
	err = tx.QueryRow("SELECT name FROM pragma_table_info('inbox_messages') WHERE name='category'").Scan(&colName)
	if err == sql.ErrNoRows {
		// Column doesn't exist, add it
		if _, err := tx.Exec("ALTER TABLE inbox_messages ADD COLUMN category TEXT"); err != nil {
			return fmt.Errorf("failed to add category column: %w", err)
		}
	}

	// Check if github_issue_number column exists
	err = tx.QueryRow("SELECT name FROM pragma_table_info('inbox_messages') WHERE name='github_issue_number'").Scan(&colName)
	if err == sql.ErrNoRows {
		if _, err := tx.Exec("ALTER TABLE inbox_messages ADD COLUMN github_issue_number INTEGER"); err != nil {
			return fmt.Errorf("failed to add github_issue_number column: %w", err)
		}
	}

	// Check if github_repo column exists
	err = tx.QueryRow("SELECT name FROM pragma_table_info('inbox_messages') WHERE name='github_repo'").Scan(&colName)
	if err == sql.ErrNoRows {
		if _, err := tx.Exec("ALTER TABLE inbox_messages ADD COLUMN github_repo TEXT"); err != nil {
			return fmt.Errorf("failed to add github_repo column: %w", err)
		}
	}

	// Create index for GitHub fields
	if _, err := tx.Exec(inboxMessagesGitHubIndex); err != nil {
		return fmt.Errorf("failed to create github index: %w", err)
	}

	// Update schema version
	if _, err := tx.Exec("INSERT OR REPLACE INTO schema_version (version) VALUES (?)", "1.1.0"); err != nil {
		return fmt.Errorf("failed to update schema version: %w", err)
	}

	return tx.Commit()
}

// migrateV110ToV120 adds semantic search columns to inbox_messages
func migrateV110ToV120(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Check if simhash column exists
	var colName string
	err = tx.QueryRow("SELECT name FROM pragma_table_info('inbox_messages') WHERE name='simhash'").Scan(&colName)
	if err == sql.ErrNoRows {
		if _, err := tx.Exec("ALTER TABLE inbox_messages ADD COLUMN simhash INTEGER"); err != nil {
			return fmt.Errorf("failed to add simhash column: %w", err)
		}
	}

	// Check if dup_of column exists
	err = tx.QueryRow("SELECT name FROM pragma_table_info('inbox_messages') WHERE name='dup_of'").Scan(&colName)
	if err == sql.ErrNoRows {
		if _, err := tx.Exec("ALTER TABLE inbox_messages ADD COLUMN dup_of TEXT"); err != nil {
			return fmt.Errorf("failed to add dup_of column: %w", err)
		}
	}

	// Create indices for semantic search
	if _, err := tx.Exec(inboxMessagesSimhashIndex); err != nil {
		return fmt.Errorf("failed to create simhash index: %w", err)
	}
	if _, err := tx.Exec(inboxMessagesDupOfIndex); err != nil {
		return fmt.Errorf("failed to create dup_of index: %w", err)
	}

	// Backfill simhash for existing messages
	if err := backfillSimhash(tx); err != nil {
		return fmt.Errorf("failed to backfill simhash: %w", err)
	}

	// Update schema version
	if _, err := tx.Exec("INSERT OR REPLACE INTO schema_version (version) VALUES (?)", "1.2.0"); err != nil {
		return fmt.Errorf("failed to update schema version: %w", err)
	}

	return tx.Commit()
}

// migrateV120ToV130 adds neural embedding columns to inbox_messages
func migrateV120ToV130(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Check if embedding column exists
	var colName string
	err = tx.QueryRow("SELECT name FROM pragma_table_info('inbox_messages') WHERE name='embedding'").Scan(&colName)
	if err == sql.ErrNoRows {
		if _, err := tx.Exec("ALTER TABLE inbox_messages ADD COLUMN embedding TEXT"); err != nil {
			return fmt.Errorf("failed to add embedding column: %w", err)
		}
	}

	// Check if embedding_model column exists
	err = tx.QueryRow("SELECT name FROM pragma_table_info('inbox_messages') WHERE name='embedding_model'").Scan(&colName)
	if err == sql.ErrNoRows {
		if _, err := tx.Exec("ALTER TABLE inbox_messages ADD COLUMN embedding_model TEXT"); err != nil {
			return fmt.Errorf("failed to add embedding_model column: %w", err)
		}
	}

	// Check if embedding_updated_at column exists
	err = tx.QueryRow("SELECT name FROM pragma_table_info('inbox_messages') WHERE name='embedding_updated_at'").Scan(&colName)
	if err == sql.ErrNoRows {
		if _, err := tx.Exec("ALTER TABLE inbox_messages ADD COLUMN embedding_updated_at INTEGER"); err != nil {
			return fmt.Errorf("failed to add embedding_updated_at column: %w", err)
		}
	}

	// Update schema version
	if _, err := tx.Exec("INSERT OR REPLACE INTO schema_version (version) VALUES (?)", "1.3.0"); err != nil {
		return fmt.Errorf("failed to update schema version: %w", err)
	}

	return tx.Commit()
}

// backfillSimhash computes and stores simhash for existing messages without one
func backfillSimhash(tx *sql.Tx) error {
	rows, err := tx.Query(`SELECT id, title, payload FROM inbox_messages WHERE simhash IS NULL`)
	if err != nil {
		return err
	}
	defer rows.Close()

	stmt, err := tx.Prepare(`UPDATE inbox_messages SET simhash = ? WHERE id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for rows.Next() {
		var id, title string
		var payload sql.NullString
		if err := rows.Scan(&id, &title, &payload); err != nil {
			return err
		}

		// Compute simhash from title + payload
		searchText := title
		if payload.Valid && payload.String != "" {
			searchText += " " + payload.String
		}
		hash := computeSimhash(searchText)

		if _, err := stmt.Exec(hash, id); err != nil {
			return err
		}
	}

	return rows.Err()
}

// computeSimhash computes a SimHash for the given text using the builtins implementation
func computeSimhash(text string) int64 {
	return builtins.SimHash(text)
}

// migrateV130ToV140 removes the category CHECK constraint to allow any string
// SQLite doesn't support DROP CONSTRAINT, so we recreate the table
func migrateV130ToV140(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Create new table without the category CHECK constraint
	newTable := `
	CREATE TABLE inbox_messages_new (
		id TEXT PRIMARY KEY,
		message_id TEXT UNIQUE NOT NULL,
		correlation_id TEXT,
		from_agent TEXT NOT NULL,
		to_inbox TEXT NOT NULL,
		message_type TEXT NOT NULL DEFAULT 'notification',
		title TEXT NOT NULL,
		payload TEXT,
		category TEXT,
		github_issue_number INTEGER,
		github_repo TEXT,
		simhash INTEGER,
		dup_of TEXT,
		embedding TEXT,
		embedding_model TEXT,
		embedding_updated_at INTEGER,
		status TEXT NOT NULL DEFAULT 'unread',
		created_at TEXT NOT NULL,
		read_at TEXT,
		expires_at TEXT,
		CHECK (message_type IN ('notification', 'request', 'response')),
		CHECK (status IN ('unread', 'read', 'archived', 'deleted'))
	)`

	if _, err := tx.Exec(newTable); err != nil {
		return fmt.Errorf("failed to create new table: %w", err)
	}

	// Copy all data (coalesce NULL status to 'unread')
	copySQL := `INSERT INTO inbox_messages_new
		SELECT id, message_id, correlation_id, from_agent, to_inbox,
		       COALESCE(message_type, 'notification'), title, payload, category,
		       github_issue_number, github_repo, simhash, dup_of,
		       embedding, embedding_model, embedding_updated_at,
		       COALESCE(status, 'unread'), created_at, read_at, expires_at
		FROM inbox_messages`
	if _, err := tx.Exec(copySQL); err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	// Drop old table
	if _, err := tx.Exec(`DROP TABLE inbox_messages`); err != nil {
		return fmt.Errorf("failed to drop old table: %w", err)
	}

	// Rename new table
	if _, err := tx.Exec(`ALTER TABLE inbox_messages_new RENAME TO inbox_messages`); err != nil {
		return fmt.Errorf("failed to rename table: %w", err)
	}

	// Recreate indices
	indices := []string{
		inboxMessagesInboxIndex,
		inboxMessagesCorrelationIndex,
		inboxMessagesGitHubIndex,
		inboxMessagesSimhashIndex,
		inboxMessagesDupOfIndex,
	}
	for _, idx := range indices {
		if _, err := tx.Exec(idx); err != nil {
			return fmt.Errorf("failed to recreate index: %w", err)
		}
	}

	// Update schema version
	if _, err := tx.Exec("INSERT OR REPLACE INTO schema_version (version) VALUES (?)", "1.4.0"); err != nil {
		return fmt.Errorf("failed to update schema version: %w", err)
	}

	return tx.Commit()
}

// migrateV140ToV150 adds parent_task_id column for task hierarchy (M-UNIFIED-AI-CONTROL-PLANE)
func migrateV140ToV150(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Check if column already exists
	var columnExists bool
	rows, err := tx.Query("PRAGMA table_info(inbox_messages)")
	if err != nil {
		return fmt.Errorf("failed to check table schema: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dfltValue, &pk); err != nil {
			return fmt.Errorf("failed to scan column info: %w", err)
		}
		if name == "parent_task_id" {
			columnExists = true
			break
		}
	}

	if !columnExists {
		// Add the parent_task_id column
		if _, err := tx.Exec("ALTER TABLE inbox_messages ADD COLUMN parent_task_id TEXT"); err != nil {
			return fmt.Errorf("failed to add parent_task_id column: %w", err)
		}
	}

	// Update schema version
	if _, err := tx.Exec("INSERT OR REPLACE INTO schema_version (version) VALUES (?)", "1.5.0"); err != nil {
		return fmt.Errorf("failed to update schema version: %w", err)
	}

	return tx.Commit()
}

// migrateV150ToV160 removes unused attachments table (M-DB-CLEANUP)
// - attachments: designed for large payloads but never implemented (0 rows)
// NOTE: approvals table kept - used by server handlers even though it duplicates coordinator.approval_requests
func migrateV150ToV160(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Drop unused attachments table (IF EXISTS for idempotency)
	if _, err := tx.Exec("DROP TABLE IF EXISTS attachments"); err != nil {
		return fmt.Errorf("failed to drop attachments table: %w", err)
	}

	// Update schema version
	if _, err := tx.Exec("INSERT OR REPLACE INTO schema_version (version) VALUES (?)", "1.6.0"); err != nil {
		return fmt.Errorf("failed to update schema version: %w", err)
	}

	return tx.Commit()
}

// migrateV160ToV170 adds chain_id column to inbox_messages (M-CHAINS-SIMPLIFY)
func migrateV160ToV170(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Check if chain_id column exists
	var colName string
	err = tx.QueryRow("SELECT name FROM pragma_table_info('inbox_messages') WHERE name='chain_id'").Scan(&colName)
	if err == sql.ErrNoRows {
		if _, err := tx.Exec("ALTER TABLE inbox_messages ADD COLUMN chain_id TEXT"); err != nil {
			return fmt.Errorf("failed to add chain_id column: %w", err)
		}
	}

	// Update schema version
	if _, err := tx.Exec("INSERT OR REPLACE INTO schema_version (version) VALUES (?)", "1.7.0"); err != nil {
		return fmt.Errorf("failed to update schema version: %w", err)
	}

	return tx.Commit()
}

// migrateV170ToV180 adds envelope column for multi-aspect semantic embeddings (M-SEMANTIC-ENVELOPE)
func migrateV170ToV180(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Check if envelope column exists
	var colName string
	err = tx.QueryRow("SELECT name FROM pragma_table_info('inbox_messages') WHERE name='envelope'").Scan(&colName)
	if err == sql.ErrNoRows {
		if _, err := tx.Exec("ALTER TABLE inbox_messages ADD COLUMN envelope TEXT DEFAULT '{}'"); err != nil {
			return fmt.Errorf("failed to add envelope column: %w", err)
		}
	}

	// Update schema version
	if _, err := tx.Exec("INSERT OR REPLACE INTO schema_version (version) VALUES (?)", "1.8.0"); err != nil {
		return fmt.Errorf("failed to update schema version: %w", err)
	}

	return tx.Commit()
}
