package messaging

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// TestInitDB tests database initialization
func TestInitDB(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()

	// Verify database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}

	// Verify we can ping the database
	if err := db.Ping(); err != nil {
		t.Errorf("Database ping failed: %v", err)
	}
}

// TestSchemaVersion tests schema version tracking
func TestSchemaVersion(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	var version string
	err := db.QueryRow("SELECT version FROM schema_version").Scan(&version)
	if err != nil {
		t.Fatalf("Failed to query schema version: %v", err)
	}

	if version != schemaVersion {
		t.Errorf("Expected schema version %s, got %s", schemaVersion, version)
	}
}

// TestTablesExist tests that all required tables are created
// NOTE: attachments table removed in v1.6.0 (M-DB-CLEANUP)
func TestTablesExist(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tables := []string{
		"schema_version",
		"threads",
		"messages",
		"subscriptions",
		"approvals",
		"replay_snapshots",
	}

	for _, table := range tables {
		exists, err := tableExists(db, table)
		if err != nil {
			t.Errorf("Failed to check table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("Table %s does not exist", table)
		}
	}
}

// TestIndicesExist tests that all required indices are created
// NOTE: idx_attachments_message removed in v1.6.0 (M-DB-CLEANUP)
func TestIndicesExist(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	indices := []string{
		"idx_messages_thread_seq",
		"idx_messages_to",
		"idx_messages_created",
		"idx_threads_status",
		"idx_subscriptions_thread",
		"idx_approvals_status",
		"idx_replay_thread",
	}

	for _, index := range indices {
		exists, err := indexExists(db, index)
		if err != nil {
			t.Errorf("Failed to check index %s: %v", index, err)
		}
		if !exists {
			t.Errorf("Index %s does not exist", index)
		}
	}
}

// TestWALMode tests that WAL mode is enabled
func TestWALMode(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	var journalMode string
	err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Fatalf("Failed to query journal_mode: %v", err)
	}

	if journalMode != "wal" {
		t.Errorf("Expected WAL mode, got %s", journalMode)
	}
}

// TestForeignKeysEnabled tests that foreign keys are enforced
func TestForeignKeysEnabled(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	var fkEnabled int
	err := db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
	if err != nil {
		t.Fatalf("Failed to query foreign_keys: %v", err)
	}

	if fkEnabled != 1 {
		t.Error("Foreign keys are not enabled")
	}
}

// TestThreadsTableConstraints tests threads table check constraints
func TestThreadsTableConstraints(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tests := []struct {
		name      string
		sql       string
		shouldErr bool
	}{
		{
			name:      "valid thread",
			sql:       `INSERT INTO threads (id, title, created_at, created_by_type, created_by_id, status, updated_at) VALUES ('t1', 'Test', 1000, 'human', 'user1', 'active', 1000)`,
			shouldErr: false,
		},
		{
			name:      "invalid created_by_type",
			sql:       `INSERT INTO threads (id, title, created_at, created_by_type, created_by_id, status, updated_at) VALUES ('t2', 'Test', 1000, 'invalid', 'user1', 'active', 1000)`,
			shouldErr: true,
		},
		{
			name:      "invalid status",
			sql:       `INSERT INTO threads (id, title, created_at, created_by_type, created_by_id, status, updated_at) VALUES ('t3', 'Test', 1000, 'human', 'user1', 'invalid', 1000)`,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := db.Exec(tt.sql)
			if tt.shouldErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestMessagesTableConstraints tests messages table check constraints
func TestMessagesTableConstraints(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create a thread first (foreign key dependency)
	_, err := db.Exec(`INSERT INTO threads (id, title, created_at, created_by_type, created_by_id, status, updated_at) VALUES ('t1', 'Test', 1000, 'human', 'user1', 'active', 1000)`)
	if err != nil {
		t.Fatalf("Failed to create test thread: %v", err)
	}

	tests := []struct {
		name      string
		sql       string
		shouldErr bool
	}{
		{
			name:      "valid message",
			sql:       `INSERT INTO messages (id, thread_id, message_seq, created_at, from_type, from_id, kind) VALUES ('m1', 't1', 1, 1000, 'human', 'user1', 'directive')`,
			shouldErr: false,
		},
		{
			name:      "invalid from_type",
			sql:       `INSERT INTO messages (id, thread_id, message_seq, created_at, from_type, from_id, kind) VALUES ('m2', 't1', 2, 1000, 'invalid', 'user1', 'directive')`,
			shouldErr: true,
		},
		{
			name:      "invalid kind",
			sql:       `INSERT INTO messages (id, thread_id, message_seq, created_at, from_type, from_id, kind) VALUES ('m3', 't1', 3, 1000, 'human', 'user1', 'invalid')`,
			shouldErr: true,
		},
		{
			name:      "duplicate message_seq in thread",
			sql:       `INSERT INTO messages (id, thread_id, message_seq, created_at, from_type, from_id, kind) VALUES ('m4', 't1', 1, 1000, 'human', 'user1', 'directive')`,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := db.Exec(tt.sql)
			if tt.shouldErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestApprovalsTableConstraints tests approvals table check constraints
func TestApprovalsTableConstraints(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create a thread first
	_, err := db.Exec(`INSERT INTO threads (id, title, created_at, created_by_type, created_by_id, status, updated_at) VALUES ('t1', 'Test', 1000, 'human', 'user1', 'active', 1000)`)
	if err != nil {
		t.Fatalf("Failed to create test thread: %v", err)
	}

	tests := []struct {
		name      string
		sql       string
		shouldErr bool
	}{
		{
			name:      "valid approval",
			sql:       `INSERT INTO approvals (id, thread_id, instance_id, created_at, effect_delta_json, proposal, impact) VALUES ('a1', 't1', 'inst1', 1000, '{}', 'Test', 'low')`,
			shouldErr: false,
		},
		{
			name:      "invalid impact",
			sql:       `INSERT INTO approvals (id, thread_id, instance_id, created_at, effect_delta_json, proposal, impact) VALUES ('a2', 't1', 'inst1', 1000, '{}', 'Test', 'invalid')`,
			shouldErr: true,
		},
		{
			name:      "invalid status",
			sql:       `INSERT INTO approvals (id, thread_id, instance_id, created_at, effect_delta_json, proposal, impact, status) VALUES ('a3', 't1', 'inst1', 1000, '{}', 'Test', 'low', 'invalid')`,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := db.Exec(tt.sql)
			if tt.shouldErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestForeignKeyEnforcement tests that foreign key constraints are enforced
func TestForeignKeyEnforcement(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Try to insert message with non-existent thread
	_, err := db.Exec(`INSERT INTO messages (id, thread_id, message_seq, created_at, from_type, from_id, kind) VALUES ('m1', 'nonexistent', 1, 1000, 'human', 'user1', 'directive')`)
	if err == nil {
		t.Error("Expected foreign key constraint violation but got none")
	}

	// Create thread
	_, err = db.Exec(`INSERT INTO threads (id, title, created_at, created_by_type, created_by_id, status, updated_at) VALUES ('t1', 'Test', 1000, 'human', 'user1', 'active', 1000)`)
	if err != nil {
		t.Fatalf("Failed to create thread: %v", err)
	}

	// Now message should succeed
	_, err = db.Exec(`INSERT INTO messages (id, thread_id, message_seq, created_at, from_type, from_id, kind) VALUES ('m1', 't1', 1, 1000, 'human', 'user1', 'directive')`)
	if err != nil {
		t.Errorf("Failed to insert message: %v", err)
	}
}

// TestCascadeDelete tests that cascade deletes work
func TestCascadeDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create thread
	_, err := db.Exec(`INSERT INTO threads (id, title, created_at, created_by_type, created_by_id, status, updated_at) VALUES ('t1', 'Test', 1000, 'human', 'user1', 'active', 1000)`)
	if err != nil {
		t.Fatalf("Failed to create thread: %v", err)
	}

	// Create message
	_, err = db.Exec(`INSERT INTO messages (id, thread_id, message_seq, created_at, from_type, from_id, kind) VALUES ('m1', 't1', 1, 1000, 'human', 'user1', 'directive')`)
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	// Delete thread (should cascade delete message)
	_, err = db.Exec(`DELETE FROM threads WHERE id = 't1'`)
	if err != nil {
		t.Fatalf("Failed to delete thread: %v", err)
	}

	// Verify message was deleted
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM messages WHERE id = 'm1'`).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count messages: %v", err)
	}

	if count != 0 {
		t.Error("Message was not cascade deleted")
	}
}

// TestUniqueThreadMessageSeq tests that message_seq is unique per thread
func TestUniqueThreadMessageSeq(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create thread
	_, err := db.Exec(`INSERT INTO threads (id, title, created_at, created_by_type, created_by_id, status, updated_at) VALUES ('t1', 'Test', 1000, 'human', 'user1', 'active', 1000)`)
	if err != nil {
		t.Fatalf("Failed to create thread: %v", err)
	}

	// Create message with seq 1
	_, err = db.Exec(`INSERT INTO messages (id, thread_id, message_seq, created_at, from_type, from_id, kind) VALUES ('m1', 't1', 1, 1000, 'human', 'user1', 'directive')`)
	if err != nil {
		t.Fatalf("Failed to create first message: %v", err)
	}

	// Try to create another message with seq 1 (should fail)
	_, err = db.Exec(`INSERT INTO messages (id, thread_id, message_seq, created_at, from_type, from_id, kind) VALUES ('m2', 't1', 1, 1001, 'human', 'user1', 'directive')`)
	if err == nil {
		t.Error("Expected unique constraint violation but got none")
	}

	// Message with seq 2 should succeed
	_, err = db.Exec(`INSERT INTO messages (id, thread_id, message_seq, created_at, from_type, from_id, kind) VALUES ('m3', 't1', 2, 1002, 'human', 'user1', 'directive')`)
	if err != nil {
		t.Errorf("Failed to create message with different seq: %v", err)
	}
}

// Helper functions

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	return db
}

func tableExists(db *sql.DB, tableName string) (bool, error) {
	var name string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&name)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func indexExists(db *sql.DB, indexName string) (bool, error) {
	var name string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='index' AND name=?", indexName).Scan(&name)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
