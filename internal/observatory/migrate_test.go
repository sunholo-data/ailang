package observatory

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestMigrate(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	defer db.Close()

	// Run migration
	err = Migrate(db)
	if err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	// Validate schema
	err = ValidateSchema(db)
	if err != nil {
		t.Fatalf("ValidateSchema failed after migration: %v", err)
	}
}

func TestMigrate_Idempotent(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	defer db.Close()

	// Run migration twice
	err = Migrate(db)
	if err != nil {
		t.Fatalf("first Migrate failed: %v", err)
	}

	err = Migrate(db)
	if err != nil {
		t.Fatalf("second Migrate should be idempotent but failed: %v", err)
	}

	// Validate schema still works
	err = ValidateSchema(db)
	if err != nil {
		t.Fatalf("ValidateSchema failed: %v", err)
	}
}

func TestMigrateWithVersion(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	defer db.Close()

	// First migration - should run all migrations up to current version
	version, err := MigrateWithVersion(db)
	if err != nil {
		t.Fatalf("MigrateWithVersion failed: %v", err)
	}
	// Current schema version is 7:
	// v1=base, v2=parent_task_id, v3=sessions, v4=remove unused tables,
	// v5=metrics+cache tokens, v6=chat_messages (M-CHAT-HISTORY-DB),
	// v7=execution_chains (M-CHAINS-SIMPLIFY)
	expectedVersion := 7
	if version != expectedVersion {
		t.Errorf("expected version %d, got %d", expectedVersion, version)
	}

	// Second call should return same version without error (idempotent)
	version, err = MigrateWithVersion(db)
	if err != nil {
		t.Fatalf("second MigrateWithVersion failed: %v", err)
	}
	if version != 7 {
		t.Errorf("expected version 7 on second call, got %d", version)
	}
}

func TestValidateSchema_MissingTables(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	defer db.Close()

	// Don't run migration - schema should be invalid
	err = ValidateSchema(db)
	if err == nil {
		t.Error("ValidateSchema should fail on empty database")
	}
}

func TestSchema_TablesExist(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	defer db.Close()

	err = Migrate(db)
	if err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	// Check each table exists
	// NOTE: span_events removed in v4 migration (M-DB-CLEANUP)
	tables := []string{
		"workspaces",
		"tasks",
		"agent_assignments",
		"spans",
		"messages",
		"sessions",
		"session_tools",
	}

	for _, table := range tables {
		var count int
		err := db.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?",
			table,
		).Scan(&count)
		if err != nil {
			t.Errorf("error checking table %s: %v", table, err)
		}
		if count != 1 {
			t.Errorf("table %s not found", table)
		}
	}
}

func TestSchema_IndexesExist(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	defer db.Close()

	err = Migrate(db)
	if err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	// Check some key indexes exist
	// NOTE: idx_span_events_* removed in v4 migration (M-DB-CLEANUP)
	indexes := []string{
		"idx_tasks_workspace",
		"idx_tasks_status",
		"idx_spans_trace",
		"idx_spans_task",
		"idx_messages_inbox",
	}

	for _, idx := range indexes {
		var count int
		err := db.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?",
			idx,
		).Scan(&count)
		if err != nil {
			t.Errorf("error checking index %s: %v", idx, err)
		}
		if count != 1 {
			t.Errorf("index %s not found", idx)
		}
	}
}

func TestSchema_ViewsExist(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	defer db.Close()

	err = Migrate(db)
	if err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	// Check aggregation views exist
	views := []string{
		"workspace_stats",
		"agent_stats",
		"task_timeline",
		"provider_comparison",
	}

	for _, view := range views {
		var count int
		err := db.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='view' AND name=?",
			view,
		).Scan(&count)
		if err != nil {
			t.Errorf("error checking view %s: %v", view, err)
		}
		if count != 1 {
			t.Errorf("view %s not found", view)
		}
	}
}

func TestSchema_ForeignKeys(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	defer db.Close()

	// Enable foreign keys
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}

	err = Migrate(db)
	if err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	// Insert a workspace
	_, err = db.Exec(`
		INSERT INTO workspaces (id, name, path, created_at, updated_at)
		VALUES ('ws1', 'Test', '/test', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`)
	if err != nil {
		t.Fatalf("failed to insert workspace: %v", err)
	}

	// Try to insert task with invalid workspace - should fail
	_, err = db.Exec(`
		INSERT INTO tasks (id, workspace_id, title, source_type, status, priority, created_at)
		VALUES ('task1', 'invalid_ws', 'Test', 'manual', 'pending', 'P1', CURRENT_TIMESTAMP)
	`)
	if err == nil {
		t.Error("expected foreign key violation for invalid workspace_id")
	}

	// Insert task with valid workspace - should succeed
	_, err = db.Exec(`
		INSERT INTO tasks (id, workspace_id, title, source_type, status, priority, created_at)
		VALUES ('task1', 'ws1', 'Test', 'manual', 'pending', 'P1', CURRENT_TIMESTAMP)
	`)
	if err != nil {
		t.Fatalf("failed to insert task with valid workspace: %v", err)
	}
}

func TestSchema_CascadeDelete_TasksAndAgents(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	defer db.Close()

	// Enable foreign keys
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}

	err = Migrate(db)
	if err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	// Insert workspace -> task -> agent_assignment
	_, err = db.Exec(`
		INSERT INTO workspaces (id, name, path, created_at, updated_at)
		VALUES ('ws1', 'Test', '/test', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`)
	if err != nil {
		t.Fatalf("failed to insert workspace: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO tasks (id, workspace_id, title, source_type, status, priority, created_at)
		VALUES ('task1', 'ws1', 'Test', 'manual', 'pending', 'P1', CURRENT_TIMESTAMP)
	`)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO agent_assignments (id, task_id, agent_id, provider, status, assigned_at)
		VALUES ('aa1', 'task1', 'agent1', 'claude', 'running', CURRENT_TIMESTAMP)
	`)
	if err != nil {
		t.Fatalf("failed to insert agent_assignment: %v", err)
	}

	// Delete workspace - should cascade delete tasks and agent_assignments
	_, err = db.Exec("DELETE FROM workspaces WHERE id = 'ws1'")
	if err != nil {
		t.Fatalf("failed to delete workspace: %v", err)
	}

	// Verify task is deleted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM tasks WHERE id = 'task1'").Scan(&count)
	if count != 0 {
		t.Error("task should be deleted on workspace delete")
	}

	// Verify agent_assignment is deleted
	db.QueryRow("SELECT COUNT(*) FROM agent_assignments WHERE id = 'aa1'").Scan(&count)
	if count != 0 {
		t.Error("agent_assignment should be deleted on task delete")
	}
}

func TestSchema_SetNull_SpansOnTaskDelete(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	defer db.Close()

	// Enable foreign keys
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}

	err = Migrate(db)
	if err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	// Insert workspace -> task -> span (spans are preserved for historical analysis)
	_, err = db.Exec(`
		INSERT INTO workspaces (id, name, path, created_at, updated_at)
		VALUES ('ws1', 'Test', '/test', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`)
	if err != nil {
		t.Fatalf("failed to insert workspace: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO tasks (id, workspace_id, title, source_type, status, priority, created_at)
		VALUES ('task1', 'ws1', 'Test', 'manual', 'pending', 'P1', CURRENT_TIMESTAMP)
	`)
	if err != nil {
		t.Fatalf("failed to insert task: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO spans (id, trace_id, name, kind, status, start_time, created_at, task_id)
		VALUES ('span1', 'trace1', 'test', 'internal', 'ok', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'task1')
	`)
	if err != nil {
		t.Fatalf("failed to insert span: %v", err)
	}

	// Delete workspace - spans should be preserved with NULL task_id
	_, err = db.Exec("DELETE FROM workspaces WHERE id = 'ws1'")
	if err != nil {
		t.Fatalf("failed to delete workspace: %v", err)
	}

	// Verify span still exists (SET NULL behavior)
	var count int
	db.QueryRow("SELECT COUNT(*) FROM spans WHERE id = 'span1'").Scan(&count)
	if count != 1 {
		t.Error("span should be preserved on task delete (SET NULL)")
	}

	// Verify task_id is now NULL
	var taskID sql.NullString
	db.QueryRow("SELECT task_id FROM spans WHERE id = 'span1'").Scan(&taskID)
	if taskID.Valid {
		t.Error("span.task_id should be NULL after task delete")
	}
}

// NOTE: TestSchema_CascadeDelete_SpanEvents removed - span_events table dropped in v4 migration (M-DB-CLEANUP)
