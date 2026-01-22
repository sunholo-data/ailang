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

	// Migration v4: Remove unused span_events table (M-DB-CLEANUP)
	// - span_events: designed for OTEL events but never implemented (0 rows)
	// NOTE: messages table kept - has API endpoints in observatory/api.go
	if currentVersion < 4 {
		// Drop unused span_events table (IF EXISTS for idempotency)
		_, err = db.Exec("DROP TABLE IF EXISTS span_events")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to drop span_events table: %w", err)
		}

		// Drop associated indices
		_, _ = db.Exec("DROP INDEX IF EXISTS idx_span_events_span")
		_, _ = db.Exec("DROP INDEX IF EXISTS idx_span_events_type")
		_, _ = db.Exec("DROP INDEX IF EXISTS idx_span_events_time")

		_, err = db.Exec("INSERT INTO schema_version (version) VALUES (4)")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to record version 4: %w", err)
		}
		currentVersion = 4
	}

	// Migration v5: Add metrics table and cache token columns to spans
	// Captures Claude Code telemetry: LOC, commits, PRs, active time, cache tokens
	if currentVersion < 5 {
		// Create metrics table for OTLP counter/gauge metrics
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS metrics (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL,
				metric_type TEXT NOT NULL,
				session_id TEXT,
				workspace TEXT,
				provider TEXT,
				label_type TEXT,
				label_tool TEXT,
				label_decision TEXT,
				label_language TEXT,
				label_model TEXT,
				value_int INTEGER,
				value_float REAL,
				labels TEXT NOT NULL DEFAULT '{}',
				resource_attributes TEXT DEFAULT '{}',
				timestamp TIMESTAMP NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to create metrics table: %w", err)
		}

		// Create indices for metrics table
		_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_metrics_name ON metrics(name)`)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to create metrics name index: %w", err)
		}

		_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_metrics_session ON metrics(session_id)`)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to create metrics session index: %w", err)
		}

		_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics(timestamp DESC)`)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to create metrics timestamp index: %w", err)
		}

		// Add cache token columns to spans table (idempotent via checking if column exists)
		var cacheReadExists int
		err = db.QueryRow(`
			SELECT COUNT(*) FROM pragma_table_info('spans') WHERE name = 'cache_read_tokens'
		`).Scan(&cacheReadExists)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to check cache_read_tokens column: %w", err)
		}

		if cacheReadExists == 0 {
			_, err = db.Exec("ALTER TABLE spans ADD COLUMN cache_read_tokens INTEGER DEFAULT 0")
			if err != nil {
				return currentVersion, fmt.Errorf("failed to add cache_read_tokens column: %w", err)
			}
		}

		var cacheCreationExists int
		err = db.QueryRow(`
			SELECT COUNT(*) FROM pragma_table_info('spans') WHERE name = 'cache_creation_tokens'
		`).Scan(&cacheCreationExists)
		if err != nil {
			return currentVersion, fmt.Errorf("failed to check cache_creation_tokens column: %w", err)
		}

		if cacheCreationExists == 0 {
			_, err = db.Exec("ALTER TABLE spans ADD COLUMN cache_creation_tokens INTEGER DEFAULT 0")
			if err != nil {
				return currentVersion, fmt.Errorf("failed to add cache_creation_tokens column: %w", err)
			}
		}

		_, err = db.Exec("INSERT INTO schema_version (version) VALUES (5)")
		if err != nil {
			return currentVersion, fmt.Errorf("failed to record version 5: %w", err)
		}
		currentVersion = 5
	}

	return currentVersion, nil
}

// ValidateSchema checks that all expected tables exist.
// Returns nil if schema is valid, error describing what's missing otherwise.
// NOTE: span_events table removed in v4 migration (M-DB-CLEANUP)
func ValidateSchema(db *sql.DB) error {
	expectedTables := []string{
		"workspaces",
		"tasks",
		"agent_assignments",
		"spans",
		"messages",
		"sessions",
		"session_tools",
		"metrics",
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
