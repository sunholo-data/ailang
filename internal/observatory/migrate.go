// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"database/sql"
	"embed"
	"fmt"
)

//go:embed schema.sql
var schemaFS embed.FS

// Migrate runs database migrations to create or update the observatory schema.
// It is idempotent - safe to call multiple times.
func Migrate(db *sql.DB) error {
	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read embedded schema: %w", err)
	}

	_, err = db.Exec(string(schema))
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

// MigrateWithVersion runs migrations and tracks schema version.
// Returns the current schema version after migration.
func MigrateWithVersion(db *sql.DB) (int, error) {
	// First ensure the schema_version table exists
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_version (
			version INTEGER PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to create schema_version table: %w", err)
	}

	// Get current version
	var currentVersion int
	err = db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&currentVersion)
	if err != nil {
		return 0, fmt.Errorf("failed to get current version: %w", err)
	}

	// Run base migration if not applied
	if currentVersion < 1 {
		if err := Migrate(db); err != nil {
			return currentVersion, err
		}

		_, err = db.Exec("INSERT INTO schema_version (version) VALUES (1)")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to record version: %w", err)
		}
		currentVersion = 1
	}

	// Migration v2: Add parent_task_id for task hierarchy tracking
	if currentVersion < 2 {
		// Check if column already exists (idempotent)
		var colExists int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM pragma_table_info('tasks') WHERE name = 'parent_task_id'
		`).Scan(&colExists)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to check parent_task_id column: %w", err)
		}

		if colExists == 0 {
			_, err = db.Exec("ALTER TABLE tasks ADD COLUMN parent_task_id TEXT")
			if err != nil {
				return currentVersion, fmt.Errorf("failed to add parent_task_id column: %w", err)
			}
		}

		// Create index (idempotent via IF NOT EXISTS)
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_tasks_parent ON tasks(parent_task_id)")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to create parent_task_id index: %w", err)
		}

		_, err = db.Exec("INSERT INTO schema_version (version) VALUES (2)")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to record version 2: %w", err)
		}
		currentVersion = 2
	}

	// Migration v3: Add sessions and session_tools tables for Claude Code hook integration
	if currentVersion < 3 {
		// Check if sessions table already exists (idempotent)
		var sessionsExists int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='sessions'
		`).Scan(&sessionsExists)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to check sessions table: %w", err)
		}

		if sessionsExists == 0 {
			_, err = db.Exec(`
				CREATE TABLE IF NOT EXISTS sessions (
					session_id TEXT PRIMARY KEY,
					workspace TEXT NOT NULL,
					claude_version TEXT,
					source TEXT DEFAULT 'hook',
					started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					ended_at TIMESTAMP,
					turn_count INTEGER DEFAULT 0
				)
			`)
			if err != nil {
				return currentVersion, fmt.Errorf("failed to create sessions table: %w", err)
			}

			_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_sessions_workspace ON sessions(workspace)`)
			if err != nil {
				return currentVersion, fmt.Errorf("failed to create sessions workspace index: %w", err)
			}

			_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_sessions_started ON sessions(started_at DESC)`)
			if err != nil {
				return currentVersion, fmt.Errorf("failed to create sessions started index: %w", err)
			}
		}

		// Check if session_tools table already exists
		var toolsExists int
		err = db.QueryRow(`
			SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='session_tools'
		`).Scan(&toolsExists)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to check session_tools table: %w", err)
		}

		if toolsExists == 0 {
			_, err = db.Exec(`
				CREATE TABLE IF NOT EXISTS session_tools (
					tool_use_id TEXT PRIMARY KEY,
					session_id TEXT NOT NULL,
					tool_name TEXT NOT NULL,
					tool_input TEXT,
					tool_response TEXT,
					start_time TIMESTAMP NOT NULL,
					end_time TIMESTAMP,
					success BOOLEAN,
					FOREIGN KEY (session_id) REFERENCES sessions(session_id) ON DELETE CASCADE
				)
			`)
			if err != nil {
				return currentVersion, fmt.Errorf("failed to create session_tools table: %w", err)
			}

			_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_session_tools_session ON session_tools(session_id)`)
			if err != nil {
				return currentVersion, fmt.Errorf("failed to create session_tools session index: %w", err)
			}

			_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_session_tools_name ON session_tools(tool_name)`)
			if err != nil {
				return currentVersion, fmt.Errorf("failed to create session_tools name index: %w", err)
			}

			_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_session_tools_time ON session_tools(start_time DESC)`)
			if err != nil {
				return currentVersion, fmt.Errorf("failed to create session_tools time index: %w", err)
			}
		}

		_, err = db.Exec("INSERT INTO schema_version (version) VALUES (3)")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to record version 3: %w", err)
		}
		currentVersion = 3
	}

	return currentVersion, nil
}

// ValidateSchema checks that all expected tables exist.
// Returns nil if schema is valid, error describing what's missing otherwise.
func ValidateSchema(db *sql.DB) error {
	expectedTables := []string{
		"workspaces",
		"tasks",
		"agent_assignments",
		"spans",
		"span_events",
		"messages",
		"sessions",
		"session_tools",
	}

	for _, table := range expectedTables {
		var name string
		err := db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?",
			table,
		).Scan(&name)
		if err == sql.ErrNoRows {
			return fmt.Errorf("missing table: %s", table)
		}
		if err != nil {
			return fmt.Errorf("error checking table %s: %w", table, err)
		}
	}

	return nil
}
