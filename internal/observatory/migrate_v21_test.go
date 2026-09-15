package observatory

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

func columnExists(t *testing.T, db *sql.DB, table, column string) bool {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, table, column).Scan(&n); err != nil {
		t.Fatalf("pragma_table_info(%s): %v", table, err)
	}
	return n > 0
}

// A database an older binary left at v20 — chat_messages and chain_stages
// without cache columns — gains them on the next open, and the version
// advances. Fresh databases get the columns from schema.sql, so the ALTERs
// must also tolerate "already exists" (the second MigrateWithVersion).
//
// The v20 shape is built by hand rather than by DROP COLUMN on a migrated
// database: a legacy view (pending_approvals_view) references a column that
// no longer exists, and SQLite refuses any DROP COLUMN while it does.
func TestMigrateV21_AddsCacheColumnsToAV20Database(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, stmt := range []string{
		`CREATE TABLE schema_version (version INTEGER PRIMARY KEY, applied_at DATETIME DEFAULT CURRENT_TIMESTAMP)`,
		`INSERT INTO schema_version (version) VALUES (20)`,
		`CREATE TABLE chat_messages (id TEXT PRIMARY KEY, session_id TEXT NOT NULL, turn_number INTEGER NOT NULL,
		   role TEXT NOT NULL, content_text TEXT, content_thinking TEXT, content_json TEXT,
		   tokens_in INTEGER DEFAULT 0, tokens_out INTEGER DEFAULT 0, model TEXT, request_id TEXT,
		   timestamp TIMESTAMP NOT NULL, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, task_id TEXT, chain_id TEXT)`,
		`CREATE TABLE chain_stages (id TEXT PRIMARY KEY, chain_id TEXT NOT NULL, stage_number INTEGER NOT NULL,
		   agent_id TEXT NOT NULL, tokens_in INTEGER DEFAULT 0, tokens_out INTEGER DEFAULT 0, quota_tokens INTEGER DEFAULT 0)`,
		`INSERT INTO chat_messages (id, session_id, turn_number, role, tokens_in, timestamp) VALUES ('m1', 's1', 1, 'assistant', 10, '2026-09-15')`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	if v := maxSchemaVersion(t, db); v != 20 {
		t.Fatalf("precondition: version %d, want 20", v)
	}
	if columnExists(t, db, "chain_stages", "cache_read_tokens") {
		t.Fatal("precondition: cache_read_tokens should be absent")
	}

	version, err := MigrateWithVersion(db)
	if err != nil {
		t.Fatalf("v21 migration: %v", err)
	}
	if version != CurrentSchemaVersion {
		t.Errorf("version = %d, want %d", version, CurrentSchemaVersion)
	}
	for _, table := range []string{"chat_messages", "chain_stages"} {
		for _, col := range []string{"cache_read_tokens", "cache_creation_tokens"} {
			if !columnExists(t, db, table, col) {
				t.Errorf("%s.%s missing after v21", table, col)
			}
		}
	}
	// A pre-existing row reads 0 through the column default, not NULL.
	var cacheRead sql.NullInt64
	if err := db.QueryRow(`SELECT cache_read_tokens FROM chat_messages WHERE id = 'm1'`).Scan(&cacheRead); err != nil {
		t.Fatal(err)
	}
	if !cacheRead.Valid || cacheRead.Int64 != 0 {
		t.Errorf("pre-existing row cache_read_tokens = %+v, want 0", cacheRead)
	}
	if _, err := MigrateWithVersion(db); err != nil {
		t.Fatalf("re-running on a v21 database must be a no-op: %v", err)
	}
}

// The importer stores what it decodes: the fixture's 128 cache-read tokens
// land on the stage and on the assistant turn that carried them, and read
// back through the store's models — not only through the import summary.
func TestImportMotokoSession_StoresCacheTokens(t *testing.T) {
	store, err := OpenStore(filepath.Join(t.TempDir(), "observatory.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()

	fixture := filepath.Join("..", "executor", "motoko", "testdata", "session_success.jsonl")
	res, err := store.ImportMotokoSession(ctx, fixture)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if res.CacheReadTokens != 128 {
		t.Fatalf("instrument: fixture cache read = %d, want 128", res.CacheReadTokens)
	}

	stages, err := store.GetChainStages(ctx, res.ChainID, ChainReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(stages) != 1 {
		t.Fatalf("stages = %d, want 1", len(stages))
	}
	if stages[0].CacheReadTokens != 128 || stages[0].CacheCreationTokens != res.CacheCreationTokens {
		t.Errorf("stage cache tokens = read %d / creation %d, want %d / %d",
			stages[0].CacheReadTokens, stages[0].CacheCreationTokens, res.CacheReadTokens, res.CacheCreationTokens)
	}

	msgs, err := store.GetChatMessagesBySession(ctx, stages[0].SessionID, time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	var sumRead int
	for _, m := range msgs {
		sumRead += m.CacheReadTokens
	}
	if sumRead != 128 {
		t.Errorf("chat_messages cache_read_tokens sum = %d, want 128 (stored per assistant turn)", sumRead)
	}

	// The per-agent rollup sums the new columns.
	byAgent, err := store.GetChainStatsByAgent(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, a := range byAgent {
		if a.AgentID == "motoko-agent" {
			found = true
			if a.CacheReadTokens != 128 {
				t.Errorf("rollup cache read = %d, want 128", a.CacheReadTokens)
			}
		}
	}
	if !found {
		t.Error("motoko-agent absent from GetChainStatsByAgent")
	}
}
