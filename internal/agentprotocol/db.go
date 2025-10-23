package agentprotocol

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the SQLite database for agent protocol state.
type DB struct {
	conn     *sql.DB
	stateDir string
}

// AgentInfo represents an agent's registration information.
type AgentInfo struct {
	AgentID       string
	InboxPath     string
	Status        string // idle, active, paused, error
	ProtocolCaps  string // JSON array of capabilities
	LastHeartbeat time.Time
	CreatedAt     time.Time
}

// AgentState represents persistent agent memory.
type AgentState struct {
	AgentID       string
	StateVersion  int
	SchemaVersion string
	LastActive    time.Time
	CurrentTask   string
	StateJSON     string // JSON-encoded state
	Checksum      string // SHA256 of state_json
}

// MessageRecord represents a message in the database.
type MessageRecord struct {
	MessageID     string
	CorrelationID string
	TraceID       string
	FromAgent     string
	ToAgent       string
	MessageType   string
	Status        string // pending, processing, completed, failed
	CreatedAt     time.Time
	ProcessedAt   *time.Time
	RetryCount    int
}

// AgentLock represents a resource lease.
type AgentLock struct {
	ResourceID string
	LockedBy   string
	LockedAt   time.Time
	ExpiresAt  time.Time
}

// NewDB creates or opens the agent protocol database.
func NewDB(stateDir string) (*DB, error) {
	dbPath := filepath.Join(stateDir, "agents.db")

	// Ensure state directory exists
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	// Open database
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	db := &DB{
		conn:     conn,
		stateDir: stateDir,
	}

	// Initialize schema
	if err := db.initSchema(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// initSchema creates all tables if they don't exist.
func (db *DB) initSchema() error {
	schema := `
-- Agents registry (discovery + capabilities)
CREATE TABLE IF NOT EXISTS agents (
    agent_id TEXT PRIMARY KEY,
    inbox_path TEXT NOT NULL,
    status TEXT CHECK(status IN ('idle', 'active', 'paused', 'error')) NOT NULL,
    protocol_caps TEXT NOT NULL,  -- JSON array: ["v1.0", "hmac", "streaming"]
    last_heartbeat TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Agent state (persistent memory)
CREATE TABLE IF NOT EXISTS agent_state (
    agent_id TEXT PRIMARY KEY,
    state_version INTEGER NOT NULL,
    schema_version TEXT NOT NULL,  -- For state evolution
    last_active TIMESTAMP NOT NULL,
    current_task TEXT,
    state_json TEXT NOT NULL,  -- JSON-encoded state
    checksum TEXT NOT NULL,    -- SHA256 of state_json
    FOREIGN KEY(agent_id) REFERENCES agents(agent_id)
);

-- Message registry (deduplication + tracking)
CREATE TABLE IF NOT EXISTS messages (
    message_id TEXT PRIMARY KEY,
    correlation_id TEXT,
    trace_id TEXT,
    from_agent TEXT NOT NULL,
    to_agent TEXT NOT NULL,
    message_type TEXT CHECK(message_type IN ('request', 'response', 'notification')) NOT NULL,
    status TEXT CHECK(status IN ('pending', 'processing', 'completed', 'failed')) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP,
    retry_count INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY(from_agent) REFERENCES agents(agent_id),
    FOREIGN KEY(to_agent) REFERENCES agents(agent_id)
);

CREATE INDEX IF NOT EXISTS idx_messages_correlation ON messages(correlation_id);
CREATE INDEX IF NOT EXISTS idx_messages_trace ON messages(trace_id);
CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status);
CREATE INDEX IF NOT EXISTS idx_messages_to_agent ON messages(to_agent, status);

-- Agent history (audit log)
CREATE TABLE IF NOT EXISTS agent_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    agent_id TEXT NOT NULL,
    message_id TEXT,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    event_type TEXT NOT NULL,
    event_data TEXT,  -- JSON
    FOREIGN KEY(agent_id) REFERENCES agents(agent_id)
);

CREATE INDEX IF NOT EXISTS idx_history_agent_time ON agent_history(agent_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_history_message ON agent_history(message_id);

-- Resource locks (leases for crash recovery)
CREATE TABLE IF NOT EXISTS agent_locks (
    resource_id TEXT PRIMARY KEY,
    locked_by TEXT NOT NULL,
    locked_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    FOREIGN KEY(locked_by) REFERENCES agents(agent_id)
);

CREATE INDEX IF NOT EXISTS idx_locks_expires ON agent_locks(expires_at);

-- Verification results
CREATE TABLE IF NOT EXISTS verification_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    verifier_agent TEXT NOT NULL,
    target_agent TEXT NOT NULL,
    artifact_hash TEXT NOT NULL,  -- SHA256 of code/doc being verified
    artifact_path TEXT NOT NULL,  -- Content-addressed: /artifacts/sha256/<hash>
    status TEXT CHECK(status IN ('pending', 'verified', 'failed')) NOT NULL,
    verification_output TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    FOREIGN KEY(verifier_agent) REFERENCES agents(agent_id),
    FOREIGN KEY(target_agent) REFERENCES agents(agent_id)
);

CREATE INDEX IF NOT EXISTS idx_verification_status ON verification_results(status);
CREATE INDEX IF NOT EXISTS idx_verification_artifact ON verification_results(artifact_hash);

-- Metrics (observability)
CREATE TABLE IF NOT EXISTS agent_metrics (
    agent_id TEXT NOT NULL,
    metric_name TEXT NOT NULL,
    metric_value REAL NOT NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(agent_id) REFERENCES agents(agent_id)
);

CREATE INDEX IF NOT EXISTS idx_metrics_agent_time ON agent_metrics(agent_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_name ON agent_metrics(metric_name);

-- DX friction reports (agents report struggles)
CREATE TABLE IF NOT EXISTS dx_friction_reports (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    agent_id TEXT NOT NULL,
    correlation_id TEXT,  -- Links to cycle that encountered friction
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    friction_type TEXT CHECK(friction_type IN (
        'syntax_error',
        'missing_builtin',
        'type_error',
        'effect_error',
        'import_error',
        'unclear_error_message',
        'missing_documentation',
        'boilerplate',
        'other'
    )) NOT NULL,
    severity TEXT CHECK(severity IN ('low', 'medium', 'high', 'blocker')) NOT NULL,
    description TEXT NOT NULL,
    suggestion TEXT,
    code_snippet TEXT,
    error_message TEXT,
    impact TEXT,  -- "Blocked development for 2 hours"
    ailang_version TEXT NOT NULL,
    FOREIGN KEY(agent_id) REFERENCES agents(agent_id)
);

CREATE INDEX IF NOT EXISTS idx_friction_type ON dx_friction_reports(friction_type);
CREATE INDEX IF NOT EXISTS idx_friction_impact ON dx_friction_reports(severity);
CREATE INDEX IF NOT EXISTS idx_friction_version ON dx_friction_reports(ailang_version);

-- DX improvements (tracking proposed vs implemented)
CREATE TABLE IF NOT EXISTS dx_improvements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    friction_report_id INTEGER,  -- Links to original report
    design_doc_path TEXT,  -- e.g., "design_docs/planned/M-DX9-auto-imports.md"
    status TEXT CHECK(status IN ('proposed', 'planned', 'implemented', 'rejected')) NOT NULL,
    proposed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    implemented_version TEXT,  -- Version where fix shipped
    FOREIGN KEY(friction_report_id) REFERENCES dx_friction_reports(id)
);

-- DX metrics (measuring impact of improvements)
CREATE TABLE IF NOT EXISTS dx_metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    improvement_id INTEGER NOT NULL,
    metric_name TEXT NOT NULL,  -- 'compile_errors', 'avg_dev_time', 'retry_count', etc.
    before_value REAL NOT NULL,
    after_value REAL NOT NULL,
    measured_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(improvement_id) REFERENCES dx_improvements(id)
);
`

	_, err := db.conn.Exec(schema)
	return err
}

// RegisterAgent registers a new agent or updates an existing one.
func (db *DB) RegisterAgent(info *AgentInfo) error {
	query := `
INSERT INTO agents (agent_id, inbox_path, status, protocol_caps, last_heartbeat, created_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(agent_id) DO UPDATE SET
    inbox_path = excluded.inbox_path,
    status = excluded.status,
    protocol_caps = excluded.protocol_caps,
    last_heartbeat = excluded.last_heartbeat
`
	_, err := db.conn.Exec(query,
		info.AgentID,
		info.InboxPath,
		info.Status,
		info.ProtocolCaps,
		info.LastHeartbeat,
		info.CreatedAt,
	)
	return err
}

// GetAgent retrieves agent information.
func (db *DB) GetAgent(agentID string) (*AgentInfo, error) {
	query := `
SELECT agent_id, inbox_path, status, protocol_caps, last_heartbeat, created_at
FROM agents
WHERE agent_id = ?
`
	var info AgentInfo
	err := db.conn.QueryRow(query, agentID).Scan(
		&info.AgentID,
		&info.InboxPath,
		&info.Status,
		&info.ProtocolCaps,
		&info.LastHeartbeat,
		&info.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// ListActiveAgents returns all agents with status 'active'.
func (db *DB) ListActiveAgents() ([]*AgentInfo, error) {
	query := `
SELECT agent_id, inbox_path, status, protocol_caps, last_heartbeat, created_at
FROM agents
WHERE status = 'active'
ORDER BY agent_id
`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []*AgentInfo
	for rows.Next() {
		var info AgentInfo
		if err := rows.Scan(
			&info.AgentID,
			&info.InboxPath,
			&info.Status,
			&info.ProtocolCaps,
			&info.LastHeartbeat,
			&info.CreatedAt,
		); err != nil {
			return nil, err
		}
		agents = append(agents, &info)
	}

	return agents, rows.Err()
}

// UpdateAgentStatus updates an agent's status.
func (db *DB) UpdateAgentStatus(agentID, status string) error {
	query := `UPDATE agents SET status = ?, last_heartbeat = ? WHERE agent_id = ?`
	_, err := db.conn.Exec(query, status, time.Now().UTC(), agentID)
	return err
}

// RecordMessage records a message in the database for deduplication and tracking.
func (db *DB) RecordMessage(record *MessageRecord) error {
	query := `
INSERT INTO messages (message_id, correlation_id, trace_id, from_agent, to_agent, message_type, status, created_at, retry_count)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(message_id) DO NOTHING
`
	_, err := db.conn.Exec(query,
		record.MessageID,
		record.CorrelationID,
		record.TraceID,
		record.FromAgent,
		record.ToAgent,
		record.MessageType,
		record.Status,
		record.CreatedAt,
		record.RetryCount,
	)
	return err
}

// GetMessage retrieves a message record by ID.
func (db *DB) GetMessage(messageID string) (*MessageRecord, error) {
	query := `
SELECT message_id, correlation_id, trace_id, from_agent, to_agent, message_type, status, created_at, processed_at, retry_count
FROM messages
WHERE message_id = ?
`
	var record MessageRecord
	var processedAt sql.NullTime

	err := db.conn.QueryRow(query, messageID).Scan(
		&record.MessageID,
		&record.CorrelationID,
		&record.TraceID,
		&record.FromAgent,
		&record.ToAgent,
		&record.MessageType,
		&record.Status,
		&record.CreatedAt,
		&processedAt,
		&record.RetryCount,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("message not found: %s", messageID)
	}
	if err != nil {
		return nil, err
	}

	if processedAt.Valid {
		record.ProcessedAt = &processedAt.Time
	}

	return &record, nil
}

// UpdateMessageStatus updates a message's status.
func (db *DB) UpdateMessageStatus(messageID, status string) error {
	query := `UPDATE messages SET status = ? WHERE message_id = ?`
	_, err := db.conn.Exec(query, status, messageID)
	return err
}

// MarkMessageProcessed updates a message to 'completed' status with timestamp.
func (db *DB) MarkMessageProcessed(messageID string) error {
	query := `UPDATE messages SET status = 'completed', processed_at = ? WHERE message_id = ?`
	_, err := db.conn.Exec(query, time.Now().UTC(), messageID)
	return err
}

// MessageExists checks if a message has already been recorded (deduplication).
func (db *DB) MessageExists(messageID string) (bool, error) {
	query := `SELECT 1 FROM messages WHERE message_id = ? LIMIT 1`
	var exists int
	err := db.conn.QueryRow(query, messageID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// AcquireLease attempts to acquire a lease on a resource.
// Returns true if acquired, false if already locked.
func (db *DB) AcquireLease(resourceID, agentID string, durationSeconds int) (bool, error) {
	expiresAt := time.Now().UTC().Add(time.Duration(durationSeconds) * time.Second)

	query := `
INSERT INTO agent_locks (resource_id, locked_by, expires_at)
VALUES (?, ?, ?)
ON CONFLICT(resource_id) DO UPDATE SET
    locked_by = excluded.locked_by,
    locked_at = CURRENT_TIMESTAMP,
    expires_at = excluded.expires_at
WHERE expires_at < CURRENT_TIMESTAMP  -- Only update if lock expired
`
	result, err := db.conn.Exec(query, resourceID, agentID, expiresAt)
	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	// If rowsAffected == 0, lock was not acquired (already held by another agent)
	return rowsAffected > 0, nil
}

// ReleaseLease releases a lease on a resource.
func (db *DB) ReleaseLease(resourceID string) error {
	query := `DELETE FROM agent_locks WHERE resource_id = ?`
	_, err := db.conn.Exec(query, resourceID)
	return err
}

// ReapExpiredLeases removes all expired leases and returns the count.
func (db *DB) ReapExpiredLeases() (int, error) {
	query := `DELETE FROM agent_locks WHERE expires_at < ?`
	result, err := db.conn.Exec(query, time.Now().UTC())
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(rowsAffected), nil
}

// GetExpiredLeases returns all expired leases (for monitoring).
func (db *DB) GetExpiredLeases() ([]*AgentLock, error) {
	query := `
SELECT resource_id, locked_by, locked_at, expires_at
FROM agent_locks
WHERE expires_at < ?
`
	rows, err := db.conn.Query(query, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var locks []*AgentLock
	for rows.Next() {
		var lock AgentLock
		if err := rows.Scan(&lock.ResourceID, &lock.LockedBy, &lock.LockedAt, &lock.ExpiresAt); err != nil {
			return nil, err
		}
		locks = append(locks, &lock)
	}

	return locks, rows.Err()
}

// LogEvent records an event in the agent history.
func (db *DB) LogEvent(agentID, messageID, eventType, eventData string) error {
	query := `
INSERT INTO agent_history (agent_id, message_id, event_type, event_data)
VALUES (?, ?, ?, ?)
`
	_, err := db.conn.Exec(query, agentID, messageID, eventType, eventData)
	return err
}

// RecordMetric records a metric data point.
func (db *DB) RecordMetric(agentID, metricName string, metricValue float64) error {
	query := `
INSERT INTO agent_metrics (agent_id, metric_name, metric_value)
VALUES (?, ?, ?)
`
	_, err := db.conn.Exec(query, agentID, metricName, metricValue)
	return err
}
