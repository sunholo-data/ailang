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
