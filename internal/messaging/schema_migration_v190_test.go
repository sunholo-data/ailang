package messaging

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// M-COMPLETION-PATH-PARITY M0b — the message_type vocabulary must match reality.
//
// The CHECK constraint allowed notification/request/response while the
// coordinator writes completion, handoff, info, audit and approval_request. Every
// one was rejected on SQLite and silently accepted on Firestore, so the two
// backends disagreed about what a valid message is. Handoffs never exposed it
// because handoffs have never fired in production — and a handoff insert is the
// first thing M1 attempts on both paths.
//
// The migration rebuilds the table, which is the highest-risk operation in this
// milestone, so it is tested for data preservation rather than just for the
// widened constraint.

// oldSchemaDB builds a database at the pre-1.9.0 shape: the narrow CHECK, and the
// columns four earlier migrations had accumulated.
func oldSchemaDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "old.db")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	stmts := []string{
		`CREATE TABLE schema_version (version TEXT PRIMARY KEY, created_at TEXT DEFAULT CURRENT_TIMESTAMP)`,
		`INSERT INTO schema_version (version, created_at) VALUES ('1.8.0', '2026-01-01T00:00:00Z')`,
		`CREATE TABLE inbox_messages (
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
			parent_task_id TEXT,
			chain_id TEXT,
			envelope TEXT DEFAULT '{}',
			status TEXT NOT NULL DEFAULT 'unread',
			created_at TEXT NOT NULL,
			read_at TEXT,
			expires_at TEXT,
			CHECK (message_type IN ('notification', 'request', 'response')),
			CHECK (status IN ('unread', 'read', 'archived', 'deleted'))
		)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}
	return db, path
}

func TestMigrateV190_RejectsHandoffBeforeAndAcceptsItAfter(t *testing.T) {
	db, _ := oldSchemaDB(t)

	// Control: the old constraint really does reject the type the coordinator
	// writes. Without this the widening below proves nothing.
	_, err := db.Exec(`INSERT INTO inbox_messages (id, message_id, from_agent, to_inbox, message_type, title, created_at)
		VALUES ('m1', 'm1', 'coordinator', 'sprint-planner', 'handoff', 'Handoff', '2026-09-03T00:00:00Z')`)
	if err == nil {
		t.Fatal("control failed: the pre-1.9.0 constraint accepted 'handoff', so this migration is unnecessary")
	}

	if err := migrateV180ToV190(db); err != nil {
		t.Fatalf("migration: %v", err)
	}

	for _, mt := range InboxMessageTypes {
		_, err := db.Exec(`INSERT INTO inbox_messages (id, message_id, from_agent, to_inbox, message_type, title, created_at)
			VALUES (?, ?, 'coordinator', 'sprint-planner', ?, 'T', '2026-09-03T00:00:00Z')`, "id-"+mt, "mid-"+mt, mt)
		if err != nil {
			t.Errorf("declared type %q still rejected after migration: %v", mt, err)
		}
	}
}

// TestMigrateV190_PreservesEveryRowAndColumn is the arm that matters: the rebuild
// copies by pragma-derived column intersection, so a column added by an earlier
// migration must survive rather than being dropped by a stale list.
func TestMigrateV190_PreservesEveryRowAndColumn(t *testing.T) {
	db, _ := oldSchemaDB(t)

	seed := [][]any{
		{"a1", "mid-a1", "mission-v1", "design-doc-creator", "request", "Fix the lockfile", "payload-a", "task-parent-1", "chain-1", `{"aspect":"x"}`},
		{"a2", "mid-a2", "coordinator", "sprint-planner", "notification", "Something happened", "payload-b", "task-parent-2", "chain-2", `{"aspect":"y"}`},
	}
	for _, row := range seed {
		_, err := db.Exec(`INSERT INTO inbox_messages
			(id, message_id, from_agent, to_inbox, message_type, title, payload, parent_task_id, chain_id, envelope, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '2026-09-03T00:00:00Z')`, row...)
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	if err := migrateV180ToV190(db); err != nil {
		t.Fatalf("migration: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM inbox_messages`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != len(seed) {
		t.Fatalf("row count = %d after migration, want %d — the rebuild lost messages", count, len(seed))
	}

	// Columns added by earlier migrations must survive the rebuild; these are
	// exactly the ones a hand-written copy list tends to forget.
	var parentTaskID, chainID, envelope, payload string
	err := db.QueryRow(`SELECT parent_task_id, chain_id, envelope, payload FROM inbox_messages WHERE id = 'a1'`).
		Scan(&parentTaskID, &chainID, &envelope, &payload)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if parentTaskID != "task-parent-1" {
		t.Errorf("parent_task_id = %q, want task-parent-1 (v1.5.0 column lost in the rebuild)", parentTaskID)
	}
	if chainID != "chain-1" {
		t.Errorf("chain_id = %q, want chain-1 (v1.7.0 column lost in the rebuild)", chainID)
	}
	if envelope != `{"aspect":"x"}` {
		t.Errorf("envelope = %q, want the seeded value (v1.8.0 column lost in the rebuild)", envelope)
	}
	if payload != "payload-a" {
		t.Errorf("payload = %q, want payload-a", payload)
	}
}

func TestMigrateV190_BumpsTheSchemaVersion(t *testing.T) {
	db, _ := oldSchemaDB(t)

	if err := migrateV180ToV190(db); err != nil {
		t.Fatalf("migration: %v", err)
	}

	var version string
	// Read the way the migrator does: the table keeps a row per version.
	if err := db.QueryRow(`SELECT version FROM schema_version ORDER BY created_at DESC LIMIT 1`).Scan(&version); err != nil {
		t.Fatalf("version: %v", err)
	}
	if version != "1.9.0" {
		t.Errorf("schema version = %q, want 1.9.0 — an unbumped version re-runs the rebuild on every open", version)
	}
}

// TestMigrateV190_IndicesSurvive: DROP TABLE takes its indices with it, so a
// rebuild that forgets to recreate them turns every inbox query into a scan.
func TestMigrateV190_IndicesSurvive(t *testing.T) {
	db, _ := oldSchemaDB(t)

	if err := migrateV180ToV190(db); err != nil {
		t.Fatalf("migration: %v", err)
	}

	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='inbox_messages'`)
	if err != nil {
		t.Fatalf("query indices: %v", err)
	}
	defer func() { _ = rows.Close() }()

	found := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan: %v", err)
		}
		found[name] = true
	}

	for _, want := range []string{
		"idx_inbox_messages_inbox",
		"idx_inbox_messages_correlation",
		"idx_inbox_messages_github",
		"idx_inbox_messages_simhash",
		"idx_inbox_messages_dup_of",
	} {
		if !found[want] {
			t.Errorf("index %s was not recreated after the table rebuild", want)
		}
	}
}

// TestInboxMessageTypes_MatchesTheConstraint keeps the Go vocabulary and the SQL
// constraint from drifting apart again — the drift is what produced the bug.
func TestInboxMessageTypes_MatchesTheConstraint(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "fresh.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = store.Close() }()

	for _, mt := range InboxMessageTypes {
		msg := &InboxMessage{
			ID:          "id-" + mt,
			FromAgent:   "coordinator",
			ToInbox:     "sprint-planner",
			MessageType: mt,
			Title:       "vocabulary check",
		}
		if err := store.InsertInboxMessage(msg); err != nil {
			t.Errorf("type %q is declared in InboxMessageTypes but rejected by a FRESH database: %v", mt, err)
		}
	}
}
