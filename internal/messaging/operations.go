package messaging

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sunholo/ailang/internal/agentprotocol"
)

// Store provides CRUD operations for the collaboration hub database.
type Store struct {
	db *sql.DB
}

// NewStore creates a new store instance from an existing database connection.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// OpenStore opens or creates a SQLite database at the given path.
// If dbPath doesn't exist, creates a new database with schema.
func OpenStore(dbPath string) (*Store, error) {
	db, err := InitDB(dbPath)
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// Message represents a message in the collaboration hub (simplified view for CLI)
type Message struct {
	ID            string    `json:"id"`
	ThreadID      string    `json:"thread_id"`
	MessageSeq    int       `json:"message_seq"`
	CreatedAt     time.Time `json:"created_at"`
	FromType      string    `json:"from_type"`
	FromID        string    `json:"from_id"`
	ToType        string    `json:"to_type"`
	ToID          string    `json:"to_id"`
	Kind          string    `json:"kind"`
	Content       string    `json:"content"`
	MetadataJSON  string    `json:"metadata_json"`
	DeliveryState string    `json:"delivery_state"`
	BusinessState string    `json:"business_state"`
}

// GetMessages retrieves messages for a specific recipient.
// If deliveryState is empty, returns all messages regardless of state.
func (s *Store) GetMessages(toType, toID string, deliveryState string) ([]Message, error) {
	query := `
		SELECT id, thread_id, message_seq, created_at, from_type, from_id, to_type, to_id,
		       kind, content, COALESCE(metadata_json, '') as metadata_json, delivery_state, business_state
		FROM messages
		WHERE to_type = ? AND to_id = ? AND deleted_at IS NULL
	`

	args := []interface{}{toType, toID}

	if deliveryState != "" {
		query += " AND delivery_state = ?"
		args = append(args, deliveryState)
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var createdAtMs int64
		var toTypeNullable, toIDNullable sql.NullString

		err := rows.Scan(
			&msg.ID, &msg.ThreadID, &msg.MessageSeq, &createdAtMs,
			&msg.FromType, &msg.FromID, &toTypeNullable, &toIDNullable,
			&msg.Kind, &msg.Content, &msg.MetadataJSON, &msg.DeliveryState, &msg.BusinessState,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		msg.CreatedAt = time.UnixMilli(createdAtMs)
		if toTypeNullable.Valid {
			msg.ToType = toTypeNullable.String
		}
		if toIDNullable.Valid {
			msg.ToID = toIDNullable.String
		}

		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

// MarkAsAcked updates a message's delivery_state to 'acked'.
func (s *Store) MarkAsAcked(messageID string) error {
	result, err := s.db.Exec(`
		UPDATE messages
		SET delivery_state = 'acked'
		WHERE id = ? AND deleted_at IS NULL
	`, messageID)

	if err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("message not found: %s", messageID)
	}

	return nil
}

// MarkAsUnacked updates a message's delivery_state back to 'pending'.
func (s *Store) MarkAsUnacked(messageID string) error {
	result, err := s.db.Exec(`
		UPDATE messages
		SET delivery_state = 'pending'
		WHERE id = ? AND deleted_at IS NULL
	`, messageID)

	if err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("message not found: %s", messageID)
	}

	return nil
}

// ClaimMessage atomically claims a pending message for processing.
// Returns nil if the message was successfully claimed by this caller.
// Returns error if message doesn't exist or was already claimed by another agent.
func (s *Store) ClaimMessage(messageID, claimedBy string) error {
	result, err := s.db.Exec(`
		UPDATE messages
		SET delivery_state = 'claimed'
		WHERE id = ?
		  AND delivery_state = 'pending'
		  AND deleted_at IS NULL
	`, messageID)

	if err != nil {
		return fmt.Errorf("failed to claim message: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("message %s already claimed or not found", messageID)
	}

	return nil
}

// MarkAllAsAcked updates all pending/visible messages for a recipient to 'acked'.
func (s *Store) MarkAllAsAcked(toType, toID string) (int64, error) {
	result, err := s.db.Exec(`
		UPDATE messages
		SET delivery_state = 'acked'
		WHERE to_type = ? AND to_id = ?
		  AND delivery_state IN ('pending', 'visible')
		  AND deleted_at IS NULL
	`, toType, toID)

	if err != nil {
		return 0, fmt.Errorf("failed to update messages: %w", err)
	}

	return result.RowsAffected()
}

// DatabaseExists checks if the collaboration database exists at the given path.
func DatabaseExists(dbPath string) bool {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return false
	}

	// Verify it's a valid SQLite database with our schema
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return false
	}
	defer db.Close()

	// Check if schema_version table exists
	var version string
	err = db.QueryRow("SELECT version FROM schema_version LIMIT 1").Scan(&version)
	return err == nil
}

// GetDefaultDatabasePath returns the default path for the collaboration database.
func GetDefaultDatabasePath() string {
	stateDir := os.Getenv("AILANG_STATE_DIR")
	if stateDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			stateDir = ".ailang/state"
		} else {
			stateDir = filepath.Join(homeDir, ".ailang", "state")
		}
	}
	return filepath.Join(stateDir, "collaboration.db")
}

// MigrateIfNeeded checks if migration is needed and performs it.
// Returns true if migration was performed, false if database already exists.
func MigrateIfNeeded(dbPath string, messagesDir string) (bool, error) {
	// If database already exists, no migration needed
	if DatabaseExists(dbPath) {
		return false, nil
	}

	// Check if there are any messages to migrate
	if _, err := os.Stat(messagesDir); os.IsNotExist(err) {
		// No messages directory, just create empty database
		db, err := InitDB(dbPath)
		if err != nil {
			return false, fmt.Errorf("failed to initialize database: %w", err)
		}
		db.Close()
		return false, nil
	}

	// Perform migration
	db, err := InitDB(dbPath)
	if err != nil {
		return false, fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	migration := NewMigration(db, messagesDir)
	count, err := migration.Migrate()
	if err != nil {
		return false, fmt.Errorf("migration failed: %w", err)
	}

	if count > 0 {
		return true, nil
	}

	return false, nil
}

// ConvertToAgentProtocolEnvelope converts a Message to agentprotocol.Envelope for backward compatibility.
func ConvertToAgentProtocolEnvelope(msg Message) (*agentprotocol.Envelope, error) {
	// Parse content as JSON payload
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(msg.Content), &payload); err != nil {
		// If content isn't JSON, wrap it
		payload = map[string]interface{}{
			"content": msg.Content,
		}
	}

	// Map kind to message_type
	messageType := "notification"
	switch msg.Kind {
	case "directive":
		messageType = "request"
	case "result":
		messageType = "response"
	case "status", "question", "proposal":
		messageType = "notification"
	}

	return &agentprotocol.Envelope{
		ProtocolVersion: "1.0.0",
		SchemaVersion:   "1.0.0",
		MessageID:       msg.ID,
		Timestamp:       msg.CreatedAt.Format(time.RFC3339),
		FromAgent:       msg.FromID,
		ToAgent:         msg.ToID,
		MessageType:     messageType,
		Payload:         payload,
	}, nil
}

// Thread represents a conversation thread
type Thread struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	CreatedAt     time.Time `json:"created_at"`
	CreatedByType string    `json:"created_by_type"`
	CreatedByID   string    `json:"created_by_id"`
	Status        string    `json:"status"`
	ContextJSON   string    `json:"context_json,omitempty"`
	TargetAgent   string    `json:"target_agent,omitempty"` // Which agent this conversation is with
	Workspace     string    `json:"workspace,omitempty"`    // Working directory for this thread
	LastSeq       int       `json:"last_seq"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ThreadContext represents the parsed context_json for a thread
type ThreadContext struct {
	TargetAgent string `json:"target_agent,omitempty"`
	Workspace   string `json:"workspace,omitempty"`
}

// CreateThread creates a new thread in the database.
// targetAgent specifies which agent this conversation is with (optional).
func (s *Store) CreateThread(title, createdByType, createdByID, targetAgent string) (*Thread, error) {
	now := time.Now()
	threadID := fmt.Sprintf("thread_%d_%s", now.UnixMilli(), generateRandomID(8))

	// Store target_agent in context_json
	var contextJSON *string
	if targetAgent != "" {
		ctx := fmt.Sprintf(`{"target_agent":"%s"}`, targetAgent)
		contextJSON = &ctx
	}

	_, err := s.db.Exec(`
		INSERT INTO threads (id, title, created_at, created_by_type, created_by_id, status, context_json, last_seq, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, threadID, title, now.UnixMilli(), createdByType, createdByID, "active", contextJSON, 0, now.UnixMilli())

	if err != nil {
		return nil, fmt.Errorf("failed to create thread: %w", err)
	}

	return &Thread{
		ID:            threadID,
		Title:         title,
		CreatedAt:     now,
		CreatedByType: createdByType,
		CreatedByID:   createdByID,
		Status:        "active",
		TargetAgent:   targetAgent,
		LastSeq:       0,
		UpdatedAt:     now,
	}, nil
}

// GetThread retrieves a thread by ID.
func (s *Store) GetThread(threadID string) (*Thread, error) {
	var thread Thread
	var createdAtMs, updatedAtMs int64
	var contextJSON sql.NullString

	err := s.db.QueryRow(`
		SELECT id, title, created_at, created_by_type, created_by_id, status, context_json, last_seq, updated_at
		FROM threads
		WHERE id = ?
	`, threadID).Scan(
		&thread.ID, &thread.Title, &createdAtMs, &thread.CreatedByType, &thread.CreatedByID,
		&thread.Status, &contextJSON, &thread.LastSeq, &updatedAtMs,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("thread not found: %s", threadID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}

	thread.CreatedAt = time.UnixMilli(createdAtMs)
	thread.UpdatedAt = time.UnixMilli(updatedAtMs)
	if contextJSON.Valid {
		thread.ContextJSON = contextJSON.String
		ctx := parseThreadContext(contextJSON.String)
		thread.TargetAgent = ctx.TargetAgent
		thread.Workspace = ctx.Workspace
	}

	return &thread, nil
}

// parseThreadContext extracts thread context from context_json
func parseThreadContext(contextJSON string) ThreadContext {
	if contextJSON == "" {
		return ThreadContext{}
	}
	var ctx ThreadContext
	if err := json.Unmarshal([]byte(contextJSON), &ctx); err != nil {
		return ThreadContext{}
	}
	return ctx
}

// SetThreadWorkspace updates the workspace path for a thread.
// This persists the workspace so all messages in the thread use the same working directory.
func (s *Store) SetThreadWorkspace(threadID, workspace string) error {
	// Get existing context
	thread, err := s.GetThread(threadID)
	if err != nil {
		return err
	}

	// Parse existing context or create new one
	ctx := parseThreadContext(thread.ContextJSON)
	ctx.Workspace = workspace

	// Serialize back to JSON
	contextBytes, err := json.Marshal(ctx)
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	// Update thread
	_, err = s.db.Exec(`
		UPDATE threads SET context_json = ?, updated_at = ? WHERE id = ?
	`, string(contextBytes), time.Now().UnixMilli(), threadID)

	if err != nil {
		return fmt.Errorf("failed to update thread workspace: %w", err)
	}

	return nil
}

// GetThreadWorkspace returns the workspace path for a thread.
func (s *Store) GetThreadWorkspace(threadID string) (string, error) {
	thread, err := s.GetThread(threadID)
	if err != nil {
		return "", err
	}
	return thread.Workspace, nil
}

// UpdateThreadTitle updates the title of a thread.
func (s *Store) UpdateThreadTitle(threadID, title string) error {
	result, err := s.db.Exec(`
		UPDATE threads SET title = ?, updated_at = ? WHERE id = ?
	`, title, time.Now().UnixMilli(), threadID)

	if err != nil {
		return fmt.Errorf("failed to update thread title: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("thread not found: %s", threadID)
	}

	return nil
}

// DeleteThread deletes a thread and all its messages.
func (s *Store) DeleteThread(threadID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Delete all messages in the thread
	_, err = tx.Exec(`DELETE FROM messages WHERE thread_id = ?`, threadID)
	if err != nil {
		return fmt.Errorf("failed to delete messages: %w", err)
	}

	// Delete all subscriptions to the thread
	_, err = tx.Exec(`DELETE FROM subscriptions WHERE thread_id = ?`, threadID)
	if err != nil {
		return fmt.Errorf("failed to delete subscriptions: %w", err)
	}

	// Delete all approvals for the thread
	_, err = tx.Exec(`DELETE FROM approvals WHERE thread_id = ?`, threadID)
	if err != nil {
		return fmt.Errorf("failed to delete approvals: %w", err)
	}

	// Delete the thread itself
	result, err := tx.Exec(`DELETE FROM threads WHERE id = ?`, threadID)
	if err != nil {
		return fmt.Errorf("failed to delete thread: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("thread not found: %s", threadID)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetThreadsByStatus returns all threads matching a status filter
// If status is empty, returns all threads
func (s *Store) GetThreadsByStatus(status string, limit int) ([]Thread, error) {
	var query string
	var args []interface{}

	if status == "" {
		query = `
			SELECT id, title, created_at, created_by_type, created_by_id, status, context_json, last_seq, updated_at
			FROM threads
			ORDER BY updated_at DESC
			LIMIT ?
		`
		args = []interface{}{limit}
	} else {
		query = `
			SELECT id, title, created_at, created_by_type, created_by_id, status, context_json, last_seq, updated_at
			FROM threads
			WHERE status = ?
			ORDER BY updated_at DESC
			LIMIT ?
		`
		args = []interface{}{status, limit}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query threads: %w", err)
	}
	defer rows.Close()

	// Initialize as empty slice (not nil) so JSON marshals to [] instead of null
	threads := []Thread{}
	for rows.Next() {
		var thread Thread
		var createdAtMs, updatedAtMs int64
		var contextJSON sql.NullString

		err := rows.Scan(
			&thread.ID, &thread.Title, &createdAtMs, &thread.CreatedByType, &thread.CreatedByID,
			&thread.Status, &contextJSON, &thread.LastSeq, &updatedAtMs,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan thread: %w", err)
		}

		thread.CreatedAt = time.UnixMilli(createdAtMs)
		thread.UpdatedAt = time.UnixMilli(updatedAtMs)
		if contextJSON.Valid {
			thread.ContextJSON = contextJSON.String
			ctx := parseThreadContext(contextJSON.String)
			thread.TargetAgent = ctx.TargetAgent
			thread.Workspace = ctx.Workspace
		}

		threads = append(threads, thread)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating threads: %w", err)
	}

	return threads, nil
}

// CreateMessage creates a new message in a thread with automatic message_seq allocation.
// This function atomically increments threads.last_seq and assigns it to the new message.
func (s *Store) CreateMessage(threadID, fromType, fromID, toType, toID, kind, content, metadataJSON string) (*Message, error) {
	// Begin transaction for atomic seq allocation
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback() // Ignore error - commit will fail if needed
	}()

	// Get current last_seq (transaction ensures atomicity)
	var lastSeq int
	err = tx.QueryRow(`
		SELECT last_seq FROM threads WHERE id = ?
	`, threadID).Scan(&lastSeq)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("thread not found: %s", threadID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to lock thread: %w", err)
	}

	// Allocate new sequence number
	newSeq := lastSeq + 1

	// Update thread's last_seq
	_, err = tx.Exec(`
		UPDATE threads SET last_seq = ?, updated_at = ? WHERE id = ?
	`, newSeq, time.Now().UnixMilli(), threadID)

	if err != nil {
		return nil, fmt.Errorf("failed to update thread seq: %w", err)
	}

	// Create message
	now := time.Now()
	messageID := fmt.Sprintf("msg_%d_%s", now.UnixMilli(), generateRandomID(12))

	_, err = tx.Exec(`
		INSERT INTO messages (
			id, thread_id, message_seq, created_at,
			from_type, from_id, to_type, to_id,
			kind, content, metadata_json, delivery_state, business_state
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, messageID, threadID, newSeq, now.UnixMilli(),
		fromType, fromID, toType, toID,
		kind, content, metadataJSON, "pending", "open")

	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	msg := &Message{
		ID:            messageID,
		ThreadID:      threadID,
		MessageSeq:    newSeq,
		CreatedAt:     now,
		FromType:      fromType,
		FromID:        fromID,
		ToType:        toType,
		ToID:          toID,
		Kind:          kind,
		Content:       content,
		MetadataJSON:  metadataJSON,
		DeliveryState: "pending",
		BusinessState: "open",
	}

	// Record metrics for result messages with execution_stats
	if kind == "result" && metadataJSON != "" {
		if stats, err := ParseMessageExecutionStats(metadataJSON); err == nil && stats != nil {
			// Use fromID as the agent ID (the agent sending the result)
			_ = s.RecordMetrics(threadID, fromID, stats)
		}
	}

	return msg, nil
}

// GetMessagesFromSeq retrieves messages in a thread starting from a given sequence number.
// This enables cursor-based resumption for WebSocket subscriptions.
func (s *Store) GetMessagesFromSeq(threadID string, fromSeq int, limit int) ([]Message, error) {
	query := `
		SELECT id, thread_id, message_seq, created_at, from_type, from_id, to_type, to_id,
		       kind, content, COALESCE(metadata_json, '') as metadata_json, delivery_state, business_state
		FROM messages
		WHERE thread_id = ? AND message_seq > ? AND deleted_at IS NULL
		ORDER BY message_seq ASC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.Query(query, threadID, fromSeq)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var createdAtMs int64
		var toTypeNullable, toIDNullable sql.NullString

		err := rows.Scan(
			&msg.ID, &msg.ThreadID, &msg.MessageSeq, &createdAtMs,
			&msg.FromType, &msg.FromID, &toTypeNullable, &toIDNullable,
			&msg.Kind, &msg.Content, &msg.MetadataJSON, &msg.DeliveryState, &msg.BusinessState,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		msg.CreatedAt = time.UnixMilli(createdAtMs)
		if toTypeNullable.Valid {
			msg.ToType = toTypeNullable.String
		}
		if toIDNullable.Valid {
			msg.ToID = toIDNullable.String
		}

		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

// Subscribe creates or updates a subscription for an instance to a thread.
func (s *Store) Subscribe(instanceID, threadID string) error {
	now := time.Now().UnixMilli()

	_, err := s.db.Exec(`
		INSERT INTO subscriptions (instance_id, thread_id, from_seq, subscribed_at, last_ack_seq)
		VALUES (?, ?, 0, ?, 0)
		ON CONFLICT (instance_id, thread_id) DO NOTHING
	`, instanceID, threadID, now)

	return err
}

// UpdateAckSeq updates the last acknowledged sequence number for a subscription.
func (s *Store) UpdateAckSeq(instanceID, threadID string, ackSeq int) error {
	result, err := s.db.Exec(`
		UPDATE subscriptions
		SET last_ack_seq = ?
		WHERE instance_id = ? AND thread_id = ?
	`, ackSeq, instanceID, threadID)

	if err != nil {
		return fmt.Errorf("failed to update ack seq: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("subscription not found: instance=%s thread=%s", instanceID, threadID)
	}

	return nil
}

// generateRandomID generates a random hex string of the given length using crypto/rand
func generateRandomID(length int) string {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based pseudo-random
		now := time.Now().UnixNano()
		for i := range bytes {
			bytes[i] = byte(now >> (i * 8))
		}
	}

	return fmt.Sprintf("%x", bytes)
}

// AgentInfo represents information about a known agent
type AgentInfo struct {
	ID         string `json:"id"`
	LastActive int64  `json:"last_active,omitempty"`
}

// Badge represents a status badge on a hierarchy node
type Badge struct {
	Type  string `json:"type"`  // "unread", "pending", "running"
	Count int    `json:"count"` // Number of items
}

// HierarchyNode represents a node in the agent/thread hierarchy tree
type HierarchyNode struct {
	Type     string          `json:"type"`               // "root", "agent", "thread"
	ID       string          `json:"id"`                 // Unique identifier
	Label    string          `json:"label"`              // Display label
	Status   string          `json:"status,omitempty"`   // "active", "idle", "pending"
	Badges   []Badge         `json:"badges,omitempty"`   // Status badges
	Children []HierarchyNode `json:"children,omitempty"` // Child nodes
}

// AgentStats represents detailed statistics for a single agent
type AgentStats struct {
	AgentID          string        `json:"agent_id"`
	Status           string        `json:"status"` // "active", "idle", "pending"
	ThreadCount      int           `json:"thread_count"`
	UnreadMessages   int           `json:"unread_messages"`
	PendingApprovals int           `json:"pending_approvals"`
	RunningProcesses int           `json:"running_processes"`
	LastActivity     string        `json:"last_activity,omitempty"`
	Threads          []ThreadStats `json:"threads,omitempty"`
}

// ThreadStats represents statistics for a thread within an agent
type ThreadStats struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	UnreadCount      int    `json:"unread_count"`
	PendingApprovals int    `json:"pending_approvals"`
	RunningProcesses int    `json:"running_processes"`
	LastMessageAt    string `json:"last_message_at,omitempty"`
}

// ExecutionStats represents aggregated execution statistics
type ExecutionStats struct {
	TotalExecutions        int     `json:"total_executions"`
	SuccessfulExecutions   int     `json:"successful_executions"`
	FailedExecutions       int     `json:"failed_executions"`
	TotalDurationMS        int64   `json:"total_duration_ms"`
	TotalCost              float64 `json:"total_cost"`
	TotalInputTokens       int     `json:"total_input_tokens"`
	TotalOutputTokens      int     `json:"total_output_tokens"`
	TotalCacheReadTokens   int     `json:"total_cache_read_tokens"`
	TotalCacheCreateTokens int     `json:"total_cache_create_tokens"`
	TotalFilesCreated      int     `json:"total_files_created"`
}

// AggregateStats represents overall statistics across all agents
type AggregateStats struct {
	TotalAgents      int            `json:"total_agents"`
	ActiveAgents     int            `json:"active_agents"`
	IdleAgents       int            `json:"idle_agents"`
	PendingApprovals int            `json:"pending_approvals"`
	RunningProcesses int            `json:"running_processes"`
	TotalThreads     int            `json:"total_threads"`
	Execution        ExecutionStats `json:"execution"`
}

// HierarchyResponse is the response for the /api/hierarchy endpoint
type HierarchyResponse struct {
	Root      HierarchyNode  `json:"root"`
	Aggregate AggregateStats `json:"aggregate"`
}

// ExecutionMetadata contains per-message execution details including file list
type ExecutionMetadata struct {
	Success             bool     `json:"success"`
	DurationMS          int      `json:"duration_ms"`
	NumTurns            int      `json:"num_turns"`
	Cost                float64  `json:"cost"`
	SessionID           string   `json:"session_id"`
	InputTokens         int      `json:"input_tokens"`
	OutputTokens        int      `json:"output_tokens"`
	CacheReadTokens     int      `json:"cache_read_tokens"`
	CacheCreationTokens int      `json:"cache_creation_tokens"`
	FilesCreatedCount   int      `json:"files_created_count"`
	FilesCreated        []string `json:"files_created"`
	Workspace           string   `json:"workspace"`
}

// parseExecutionMetadataFromMessage extracts full execution metadata from a message's metadata_json
func parseExecutionMetadataFromMessage(metadataJSON string) *ExecutionMetadata {
	if metadataJSON == "" {
		return nil
	}

	var metadata struct {
		ExecutionStats ExecutionMetadata `json:"execution_stats"`
	}

	if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
		return nil
	}

	stats := metadata.ExecutionStats
	// Check if we actually have execution stats (not empty struct)
	if stats.DurationMS == 0 && stats.Cost == 0 && stats.InputTokens == 0 && stats.OutputTokens == 0 {
		return nil
	}

	return &stats
}

// parseExecutionStatsFromMetadata extracts execution stats from a message's metadata_json
func parseExecutionStatsFromMetadata(metadataJSON string) *ExecutionStats {
	meta := parseExecutionMetadataFromMessage(metadataJSON)
	if meta == nil {
		return nil
	}

	execStats := &ExecutionStats{
		TotalExecutions:        1,
		TotalDurationMS:        int64(meta.DurationMS),
		TotalCost:              meta.Cost,
		TotalInputTokens:       meta.InputTokens,
		TotalOutputTokens:      meta.OutputTokens,
		TotalCacheReadTokens:   meta.CacheReadTokens,
		TotalCacheCreateTokens: meta.CacheCreationTokens,
		TotalFilesCreated:      meta.FilesCreatedCount,
	}
	if meta.Success {
		execStats.SuccessfulExecutions = 1
	} else {
		execStats.FailedExecutions = 1
	}

	return execStats
}

// GetAggregatedExecutionStats aggregates execution stats from all result messages
func (s *Store) GetAggregatedExecutionStats() (*ExecutionStats, error) {
	// Query all result messages with metadata
	rows, err := s.db.Query(`
		SELECT metadata_json
		FROM messages
		WHERE kind = 'result'
		  AND metadata_json IS NOT NULL
		  AND metadata_json != ''
		  AND deleted_at IS NULL
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query result messages: %w", err)
	}
	defer rows.Close()

	aggregate := &ExecutionStats{}

	for rows.Next() {
		var metadataJSON string
		if err := rows.Scan(&metadataJSON); err != nil {
			continue
		}

		stats := parseExecutionStatsFromMetadata(metadataJSON)
		if stats != nil {
			aggregate.TotalExecutions += stats.TotalExecutions
			aggregate.SuccessfulExecutions += stats.SuccessfulExecutions
			aggregate.FailedExecutions += stats.FailedExecutions
			aggregate.TotalDurationMS += stats.TotalDurationMS
			aggregate.TotalCost += stats.TotalCost
			aggregate.TotalInputTokens += stats.TotalInputTokens
			aggregate.TotalOutputTokens += stats.TotalOutputTokens
			aggregate.TotalCacheReadTokens += stats.TotalCacheReadTokens
			aggregate.TotalCacheCreateTokens += stats.TotalCacheCreateTokens
			aggregate.TotalFilesCreated += stats.TotalFilesCreated
		}
	}

	return aggregate, rows.Err()
}

// GetExecutionStatsByThread aggregates execution stats for a specific thread
func (s *Store) GetExecutionStatsByThread(threadID string) (*ExecutionStats, error) {
	rows, err := s.db.Query(`
		SELECT metadata_json
		FROM messages
		WHERE thread_id = ?
		  AND kind = 'result'
		  AND metadata_json IS NOT NULL
		  AND metadata_json != ''
		  AND deleted_at IS NULL
	`, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to query result messages: %w", err)
	}
	defer rows.Close()

	aggregate := &ExecutionStats{}

	for rows.Next() {
		var metadataJSON string
		if err := rows.Scan(&metadataJSON); err != nil {
			continue
		}

		stats := parseExecutionStatsFromMetadata(metadataJSON)
		if stats != nil {
			aggregate.TotalExecutions += stats.TotalExecutions
			aggregate.SuccessfulExecutions += stats.SuccessfulExecutions
			aggregate.FailedExecutions += stats.FailedExecutions
			aggregate.TotalDurationMS += stats.TotalDurationMS
			aggregate.TotalCost += stats.TotalCost
			aggregate.TotalInputTokens += stats.TotalInputTokens
			aggregate.TotalOutputTokens += stats.TotalOutputTokens
			aggregate.TotalCacheReadTokens += stats.TotalCacheReadTokens
			aggregate.TotalCacheCreateTokens += stats.TotalCacheCreateTokens
			aggregate.TotalFilesCreated += stats.TotalFilesCreated
		}
	}

	return aggregate, rows.Err()
}

// GetHierarchy returns the complete agent/thread hierarchy tree
func (s *Store) GetHierarchy() (*HierarchyResponse, error) {
	// Get all agents
	agents, err := s.GetKnownAgents()
	if err != nil {
		return nil, fmt.Errorf("failed to get agents: %w", err)
	}

	// Get all threads
	threads, err := s.GetThreadsByStatus("", 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to get threads: %w", err)
	}

	// Get all pending approvals
	pendingApprovals, err := s.GetApprovalsByStatus("pending", 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending approvals: %w", err)
	}

	// Build approval counts by thread
	approvalsByThread := make(map[string]int)
	for _, approval := range pendingApprovals {
		approvalsByThread[approval.ThreadID]++
	}

	// Build thread nodes grouped by agent
	threadsByAgent := make(map[string][]HierarchyNode)
	for _, thread := range threads {
		agentID := thread.TargetAgent
		if agentID == "" {
			agentID = "unassigned"
		}

		var badges []Badge
		pendingCount := approvalsByThread[thread.ID]
		if pendingCount > 0 {
			badges = append(badges, Badge{Type: "pending", Count: pendingCount})
		}

		threadNode := HierarchyNode{
			Type:   "thread",
			ID:     thread.ID,
			Label:  thread.Title,
			Badges: badges,
		}
		threadsByAgent[agentID] = append(threadsByAgent[agentID], threadNode)
	}

	// Build set of known agent IDs for lookup
	knownAgents := make(map[string]bool)
	for _, agent := range agents {
		knownAgents[agent.ID] = true
	}

	// Build agent nodes
	var agentNodes []HierarchyNode
	activeCount := 0
	idleCount := 0

	for _, agent := range agents {
		childThreads := threadsByAgent[agent.ID]

		// Calculate agent status based on pending approvals
		status := "idle"
		var agentBadges []Badge
		pendingCount := 0
		for _, threadNode := range childThreads {
			for _, badge := range threadNode.Badges {
				if badge.Type == "pending" {
					pendingCount += badge.Count
				}
			}
		}
		if pendingCount > 0 {
			status = "pending"
			agentBadges = append(agentBadges, Badge{Type: "pending", Count: pendingCount})
		}

		if status == "idle" {
			idleCount++
		} else {
			activeCount++
		}

		agentNode := HierarchyNode{
			Type:     "agent",
			ID:       agent.ID,
			Label:    agent.ID,
			Status:   status,
			Badges:   agentBadges,
			Children: childThreads,
		}
		agentNodes = append(agentNodes, agentNode)
	}

	// Add threads for unknown agents (agents referenced by threads but not in database)
	for agentID, childThreads := range threadsByAgent {
		if agentID == "unassigned" {
			continue // Handle separately below
		}
		if !knownAgents[agentID] && len(childThreads) > 0 {
			// This is an unknown agent with threads - create a node for it
			agentNodes = append(agentNodes, HierarchyNode{
				Type:     "agent",
				ID:       agentID,
				Label:    agentID,
				Status:   "idle",
				Children: childThreads,
			})
			idleCount++
		}
	}

	// Add unassigned threads if any
	if unassignedThreads, ok := threadsByAgent["unassigned"]; ok && len(unassignedThreads) > 0 {
		agentNodes = append(agentNodes, HierarchyNode{
			Type:     "agent",
			ID:       "unassigned",
			Label:    "Unassigned",
			Status:   "idle",
			Children: unassignedThreads,
		})
	}

	// Build root node
	root := HierarchyNode{
		Type:     "root",
		ID:       "all",
		Label:    "All Agents",
		Children: agentNodes,
	}

	// Get aggregated execution stats
	execStats, err := s.GetAggregatedExecutionStats()
	if err != nil {
		// Non-fatal - use empty stats
		execStats = &ExecutionStats{}
	}

	// Build aggregate stats
	aggregate := AggregateStats{
		TotalAgents:      len(agents),
		ActiveAgents:     activeCount,
		IdleAgents:       idleCount,
		PendingApprovals: len(pendingApprovals),
		RunningProcesses: 0, // TODO: Add process tracking
		TotalThreads:     len(threads),
		Execution:        *execStats,
	}

	return &HierarchyResponse{
		Root:      root,
		Aggregate: aggregate,
	}, nil
}

// GetAgentStats returns detailed statistics for a single agent
func (s *Store) GetAgentStats(agentID string) (*AgentStats, error) {
	// Get threads for this agent
	threads, err := s.GetThreadsByStatus("", 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to get threads: %w", err)
	}

	// Filter threads for this agent
	var agentThreads []Thread
	for _, thread := range threads {
		if thread.TargetAgent == agentID || (thread.TargetAgent == "" && agentID == "unassigned") {
			agentThreads = append(agentThreads, thread)
		}
	}

	// Get pending approvals
	pendingApprovals, err := s.GetApprovalsByStatus("pending", 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending approvals: %w", err)
	}

	// Build approval counts by thread
	approvalsByThread := make(map[string]int)
	for _, approval := range pendingApprovals {
		approvalsByThread[approval.ThreadID]++
	}

	// Build thread stats
	var threadStats []ThreadStats
	totalPending := 0
	for _, thread := range agentThreads {
		pendingCount := approvalsByThread[thread.ID]
		totalPending += pendingCount

		threadStats = append(threadStats, ThreadStats{
			ID:               thread.ID,
			Title:            thread.Title,
			UnreadCount:      0, // TODO: Calculate unread
			PendingApprovals: pendingCount,
			RunningProcesses: 0, // TODO: Add process tracking
			LastMessageAt:    thread.UpdatedAt.Format(time.RFC3339),
		})
	}

	// Determine status
	status := "idle"
	if totalPending > 0 {
		status = "pending"
	}

	return &AgentStats{
		AgentID:          agentID,
		Status:           status,
		ThreadCount:      len(agentThreads),
		UnreadMessages:   0, // TODO: Calculate unread
		PendingApprovals: totalPending,
		RunningProcesses: 0, // TODO: Add process tracking
		Threads:          threadStats,
	}, nil
}

// GetKnownAgents returns a list of known agent IDs from the database
func (s *Store) GetKnownAgents() ([]AgentInfo, error) {
	// Query for distinct agent IDs from multiple sources:
	// 1. Messages sent to ailang_instance (to_id)
	// 2. Subscriptions (instance_id)
	// 3. Thread target_agent from context_json
	query := `
		SELECT DISTINCT agent_id, MAX(last_active) as last_active FROM (
			-- Agents that received messages
			SELECT DISTINCT to_id as agent_id, created_at as last_active
			FROM messages
			WHERE to_type = 'ailang_instance' AND to_id IS NOT NULL AND to_id != ''

			UNION

			-- Agents with subscriptions
			SELECT DISTINCT instance_id as agent_id, subscribed_at as last_active
			FROM subscriptions
			WHERE instance_id IS NOT NULL AND instance_id != ''

			UNION

			-- Agents that sent messages
			SELECT DISTINCT from_id as agent_id, created_at as last_active
			FROM messages
			WHERE from_type = 'ailang_instance' AND from_id IS NOT NULL AND from_id != ''
		)
		GROUP BY agent_id
		ORDER BY last_active DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query agents: %w", err)
	}
	defer rows.Close()

	var agents []AgentInfo
	for rows.Next() {
		var agent AgentInfo
		var lastActive sql.NullInt64
		if err := rows.Scan(&agent.ID, &lastActive); err != nil {
			return nil, fmt.Errorf("failed to scan agent: %w", err)
		}
		if lastActive.Valid {
			agent.LastActive = lastActive.Int64
		}
		agents = append(agents, agent)
	}

	// Always include a default agent if none found
	if len(agents) == 0 {
		agents = append(agents, AgentInfo{ID: "my-agent"})
	}

	return agents, rows.Err()
}
