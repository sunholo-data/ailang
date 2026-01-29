package messaging

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sunholo/ailang/internal/builtins"

	_ "github.com/mattn/go-sqlite3"
)

// Schema defines the SQLite database schema for the collaboration hub.
// This schema extends the existing file-based agent inbox (.ailang/state/messages/)
// to provide ordering guarantees, real-time updates, and effect-gated approvals.

const schemaVersion = "1.6.0" // v1.6.0: Removed unused approvals/attachments tables (M-DB-CLEANUP)

// InitDB creates and initializes a new SQLite database with the collaboration hub schema.
// Returns the database connection and any error encountered.
//
// The database is configured with:
// - WAL mode for write concurrency
// - NORMAL synchronous mode for performance
// - 5 second busy timeout for lock contention
func InitDB(dbPath string) (*sql.DB, error) {
	// Ensure the database directory exists
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory %s: %w", dbDir, err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure database for concurrent access
	if err := configureDB(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to configure database: %w", err)
	}

	// Create schema
	if err := createSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return db, nil
}

// configureDB sets SQLite pragmas for performance and concurrency
func configureDB(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA journal_mode=WAL",   // Write-Ahead Logging for concurrency
		"PRAGMA synchronous=NORMAL", // Balance safety and performance
		"PRAGMA busy_timeout=5000",  // 5 second timeout for locks
		"PRAGMA foreign_keys=ON",    // Enforce foreign key constraints
		"PRAGMA temp_store=MEMORY",  // Store temp tables in memory
		"PRAGMA cache_size=-64000",  // 64MB cache (negative = KB)
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return fmt.Errorf("failed to execute %s: %w", pragma, err)
		}
	}

	return nil
}

// createSchema creates all tables and indices
func createSchema(db *sql.DB) error {
	// Execute schema in a transaction for atomicity
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback() // Ignore error - commit will fail if needed
	}()

	// Create schema_version table first
	if _, err := tx.Exec(schemaVersionTable); err != nil {
		return fmt.Errorf("failed to create schema_version table: %w", err)
	}

	// Only insert schema version if no version exists yet
	// This prevents overwriting older versions that need migration
	var count int
	if err := tx.QueryRow("SELECT COUNT(*) FROM schema_version").Scan(&count); err != nil {
		return fmt.Errorf("failed to check schema version: %w", err)
	}
	if count == 0 {
		if _, err := tx.Exec("INSERT INTO schema_version (version) VALUES (?)", schemaVersion); err != nil {
			return fmt.Errorf("failed to insert schema version: %w", err)
		}
	}

	// Create core tables
	// NOTE: attachments table removed in v1.6.0 (M-DB-CLEANUP) - never used, no consumers
	// NOTE: approvals table kept - used by server handlers (but duplicates coordinator.approval_requests - see design doc)
	tables := []struct {
		name   string
		schema string
	}{
		{"threads", threadsTable},
		{"messages", messagesTable},
		{"subscriptions", subscriptionsTable},
		{"approvals", approvalsTable},
		{"replay_snapshots", replaySnapshotsTable},
		{"agents", agentsTable},
		{"metrics_aggregates", metricsAggregatesTable},
		{"approval_history", approvalHistoryTable},
		{"instance_history", instanceHistoryTable},
		{"inbox_messages", inboxMessagesTable},
	}

	for _, table := range tables {
		if _, err := tx.Exec(table.schema); err != nil {
			return fmt.Errorf("failed to create %s table: %w", table.name, err)
		}
	}

	// Create indices
	// NOTE: attachmentsMessageIndex removed in v1.6.0 (attachments table dropped)
	indices := []string{
		messagesThreadSeqIndex,
		messagesToIndex,
		messagesCreatedIndex,
		threadsStatusIndex,
		subscriptionsThreadIndex,
		approvalsStatusIndex,
		replayThreadIndex,
		metricsAggregatesPeriodIndex,
		approvalHistoryThreadIndex,
		instanceHistoryAgentIndex,
		inboxMessagesInboxIndex,
		inboxMessagesCorrelationIndex,
	}

	for _, index := range indices {
		if _, err := tx.Exec(index); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return tx.Commit()
}

// Schema version tracking
const schemaVersionTable = `
CREATE TABLE IF NOT EXISTS schema_version (
    version TEXT PRIMARY KEY,
    created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now') * 1000)
)`

// Threads table - conversation threads between humans and instances
const threadsTable = `
CREATE TABLE IF NOT EXISTS threads (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    created_by_type TEXT NOT NULL,
    created_by_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    context_json TEXT,
    last_seq INTEGER NOT NULL DEFAULT 0,
    updated_at INTEGER NOT NULL,

    CHECK (created_by_type IN ('human', 'ailang_instance')),
    CHECK (status IN ('active', 'paused', 'resolved', 'archived'))
)`

// Messages table - individual messages within threads
const messagesTable = `
CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    thread_id TEXT NOT NULL,
    message_seq INTEGER NOT NULL,
    created_at INTEGER NOT NULL,

    -- Routing
    from_type TEXT NOT NULL,
    from_id TEXT NOT NULL,
    to_type TEXT,
    to_id TEXT,

    -- Content
    kind TEXT NOT NULL,
    subject TEXT,
    content TEXT,
    metadata_json TEXT,

    -- State
    delivery_state TEXT NOT NULL DEFAULT 'pending',
    business_state TEXT DEFAULT 'open',

    -- Threading
    reply_to TEXT,

    -- Soft delete
    deleted_at INTEGER,

    FOREIGN KEY (thread_id) REFERENCES threads(id) ON DELETE CASCADE,
    FOREIGN KEY (reply_to) REFERENCES messages(id) ON DELETE SET NULL,

    UNIQUE (thread_id, message_seq),
    CHECK (from_type IN ('human', 'ailang_instance')),
    CHECK (to_type IN ('human', 'ailang_instance', 'broadcast') OR to_type IS NULL),
    CHECK (kind IN ('directive', 'question', 'proposal', 'status', 'result')),
    CHECK (delivery_state IN ('pending', 'visible', 'claimed', 'acked')),
    CHECK (business_state IN ('open', 'resolved', 'archived') OR business_state IS NULL)
)`

// Subscriptions table - which instances/humans watch which threads
const subscriptionsTable = `
CREATE TABLE IF NOT EXISTS subscriptions (
    instance_id TEXT NOT NULL,
    thread_id TEXT NOT NULL,
    from_seq INTEGER NOT NULL DEFAULT 0,
    subscribed_at INTEGER NOT NULL,
    last_ack_seq INTEGER DEFAULT 0,

    PRIMARY KEY (instance_id, thread_id),
    FOREIGN KEY (thread_id) REFERENCES threads(id) ON DELETE CASCADE
)`

// Approvals table - effect-gated approval workflow
const approvalsTable = `
CREATE TABLE IF NOT EXISTS approvals (
    id TEXT PRIMARY KEY,
    thread_id TEXT NOT NULL,
    instance_id TEXT NOT NULL,
    created_at INTEGER NOT NULL,

    -- What the agent wants to do
    effect_delta_json TEXT NOT NULL,
    proposal TEXT NOT NULL,
    impact TEXT NOT NULL,
    estimated_cost REAL,

    -- Approval state
    status TEXT NOT NULL DEFAULT 'pending',
    reviewed_by TEXT,
    reviewed_at INTEGER,
    review_notes TEXT,

    -- Capability token
    capability_token TEXT,
    token_expires_at INTEGER,

    FOREIGN KEY (thread_id) REFERENCES threads(id) ON DELETE CASCADE,
    CHECK (impact IN ('low', 'medium', 'high')),
    CHECK (status IN ('pending', 'approved', 'rejected', 'modified'))
)`

// Attachments table - large payloads separate from messages
//
//nolint:unused // Schema prepared for M-COLLAB-2 attachments feature
const attachmentsTable = `
CREATE TABLE IF NOT EXISTS attachments (
    id TEXT PRIMARY KEY,
    message_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    content_type TEXT,
    path TEXT,
    blob BLOB,
    size_bytes INTEGER NOT NULL,
    created_at INTEGER NOT NULL,

    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
    CHECK (kind IN ('code', 'diff', 'test_output', 'artifact'))
)`

// Replay snapshots table - deterministic replay metadata
const replaySnapshotsTable = `
CREATE TABLE IF NOT EXISTS replay_snapshots (
    id TEXT PRIMARY KEY,
    thread_id TEXT NOT NULL,
    message_id TEXT NOT NULL,
    created_at INTEGER NOT NULL,

    -- Full context to reproduce
    model_id TEXT NOT NULL,
    model_version TEXT,
    temperature REAL,
    seed INTEGER,
    top_p REAL,
    tool_list_json TEXT,
    prompt_slate_json TEXT,
    prompt_checksum TEXT,

    FOREIGN KEY (thread_id) REFERENCES threads(id) ON DELETE CASCADE,
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE
)`

// Agents table - tracks registered AI agents
const agentsTable = `
CREATE TABLE IF NOT EXISTS agents (
    id TEXT PRIMARY KEY,
    label TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'idle',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    last_active_at INTEGER,
    config_json TEXT,

    CHECK (status IN ('idle', 'running', 'error'))
)`

// Metrics aggregates table - pre-computed metrics at different scopes and periods
const metricsAggregatesTable = `
CREATE TABLE IF NOT EXISTS metrics_aggregates (
    id TEXT PRIMARY KEY,
    scope_type TEXT NOT NULL,
    scope_id TEXT NOT NULL,
    period TEXT NOT NULL,
    period_start INTEGER NOT NULL,

    total_runs INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    total_cost_cents INTEGER NOT NULL DEFAULT 0,
    total_duration_ms INTEGER NOT NULL DEFAULT 0,
    total_files_modified INTEGER NOT NULL DEFAULT 0,

    avg_tokens_per_run REAL DEFAULT 0,
    avg_cost_per_run REAL DEFAULT 0,
    avg_duration_per_run REAL DEFAULT 0,

    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,

    UNIQUE(scope_type, scope_id, period, period_start),
    CHECK (scope_type IN ('global', 'agent', 'thread')),
    CHECK (period IN ('minute', 'hour', 'day'))
)`

// Approval history table - audit trail for approval actions
const approvalHistoryTable = `
CREATE TABLE IF NOT EXISTS approval_history (
    id TEXT PRIMARY KEY,
    approval_id TEXT NOT NULL,
    thread_id TEXT NOT NULL,
    agent_id TEXT NOT NULL,
    action TEXT NOT NULL,
    actor TEXT NOT NULL,
    proposal TEXT,
    impact TEXT,
    estimated_cost REAL,
    capability_token TEXT,
    created_at INTEGER NOT NULL,

    CHECK (action IN ('created', 'approved', 'rejected', 'expired'))
)`

// Instance history table - tracks agent instance lifecycles
const instanceHistoryTable = `
CREATE TABLE IF NOT EXISTS instance_history (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    instance_id TEXT NOT NULL,
    started_at INTEGER NOT NULL,
    ended_at INTEGER,
    exit_code INTEGER,
    total_tokens INTEGER DEFAULT 0,
    total_cost_cents INTEGER DEFAULT 0,
    thread_count INTEGER DEFAULT 0
)`

// Inbox messages table - simple async agent-to-agent/user messaging
// This is the unified inbox system for CLI (ailang messages) and hooks
const inboxMessagesTable = `
CREATE TABLE IF NOT EXISTS inbox_messages (
    id TEXT PRIMARY KEY,
    message_id TEXT UNIQUE NOT NULL,
    correlation_id TEXT,

    -- Routing
    from_agent TEXT NOT NULL,
    to_inbox TEXT NOT NULL,
    message_type TEXT NOT NULL DEFAULT 'notification',

    -- Content
    title TEXT NOT NULL,
    payload TEXT,

    -- Content category (bug, feature, general) - for GitHub sync
    category TEXT,

    -- GitHub integration (v1.1.0)
    github_issue_number INTEGER,
    github_repo TEXT,

    -- Semantic search (v1.2.0)
    simhash INTEGER,
    dup_of TEXT,

    -- Neural embeddings (v1.3.0)
    embedding TEXT,              -- JSON array of float32
    embedding_model TEXT,        -- e.g., "ollama:nomic-embed-text"
    embedding_updated_at INTEGER, -- unix millis

    -- Task hierarchy (v1.5.0, M-UNIFIED-AI-CONTROL-PLANE)
    parent_task_id TEXT,         -- Link to parent task for hierarchical execution

    -- Execution chain (v1.7.0, M-CHAINS-SIMPLIFY)
    chain_id TEXT,               -- Link to execution chain for unified hierarchy

    -- State
    status TEXT NOT NULL DEFAULT 'unread',
    created_at TEXT NOT NULL,
    read_at TEXT,
    expires_at TEXT,

    CHECK (message_type IN ('notification', 'request', 'response')),
    CHECK (status IN ('unread', 'read', 'archived', 'deleted'))
    -- category: any string allowed (bug/feature have special behavior)
)`

// Indices for performance
const messagesThreadSeqIndex = `CREATE INDEX IF NOT EXISTS idx_messages_thread_seq ON messages(thread_id, message_seq)`
const messagesToIndex = `CREATE INDEX IF NOT EXISTS idx_messages_to ON messages(to_type, to_id, delivery_state)`
const messagesCreatedIndex = `CREATE INDEX IF NOT EXISTS idx_messages_created ON messages(created_at, id)`
const threadsStatusIndex = `CREATE INDEX IF NOT EXISTS idx_threads_status ON threads(status, updated_at)`
const subscriptionsThreadIndex = `CREATE INDEX IF NOT EXISTS idx_subscriptions_thread ON subscriptions(thread_id)`
const approvalsStatusIndex = `CREATE INDEX IF NOT EXISTS idx_approvals_status ON approvals(status, created_at)`

//nolint:unused // Schema prepared for M-COLLAB-2 attachments feature
const attachmentsMessageIndex = `CREATE INDEX IF NOT EXISTS idx_attachments_message ON attachments(message_id)`
const replayThreadIndex = `CREATE INDEX IF NOT EXISTS idx_replay_thread ON replay_snapshots(thread_id, created_at)`
const metricsAggregatesPeriodIndex = `CREATE INDEX IF NOT EXISTS idx_metrics_period ON metrics_aggregates(scope_type, period, period_start)`
const approvalHistoryThreadIndex = `CREATE INDEX IF NOT EXISTS idx_approval_history_thread ON approval_history(thread_id, created_at)`
const instanceHistoryAgentIndex = `CREATE INDEX IF NOT EXISTS idx_instance_history_agent ON instance_history(agent_id, started_at)`
const inboxMessagesInboxIndex = `CREATE INDEX IF NOT EXISTS idx_inbox_messages_inbox ON inbox_messages(to_inbox, status, created_at)`
const inboxMessagesCorrelationIndex = `CREATE INDEX IF NOT EXISTS idx_inbox_messages_correlation ON inbox_messages(correlation_id)`
const inboxMessagesGitHubIndex = `CREATE INDEX IF NOT EXISTS idx_inbox_messages_github ON inbox_messages(github_repo, github_issue_number)`
const inboxMessagesSimhashIndex = `CREATE INDEX IF NOT EXISTS idx_inbox_messages_simhash ON inbox_messages(simhash)`
const inboxMessagesDupOfIndex = `CREATE INDEX IF NOT EXISTS idx_inbox_messages_dup_of ON inbox_messages(dup_of)`

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
