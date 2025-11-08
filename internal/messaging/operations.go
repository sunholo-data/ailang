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
	DeliveryState string    `json:"delivery_state"`
	BusinessState string    `json:"business_state"`
}

// GetMessages retrieves messages for a specific recipient.
// If deliveryState is empty, returns all messages regardless of state.
func (s *Store) GetMessages(toType, toID string, deliveryState string) ([]Message, error) {
	query := `
		SELECT id, thread_id, message_seq, created_at, from_type, from_id, to_type, to_id,
		       kind, content, delivery_state, business_state
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
			&msg.Kind, &msg.Content, &msg.DeliveryState, &msg.BusinessState,
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
	LastSeq       int       `json:"last_seq"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CreateThread creates a new thread in the database.
func (s *Store) CreateThread(title, createdByType, createdByID string) (*Thread, error) {
	now := time.Now()
	threadID := fmt.Sprintf("thread_%d_%s", now.UnixMilli(), generateRandomID(8))

	_, err := s.db.Exec(`
		INSERT INTO threads (id, title, created_at, created_by_type, created_by_id, status, last_seq, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, threadID, title, now.UnixMilli(), createdByType, createdByID, "active", 0, now.UnixMilli())

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
	}

	return &thread, nil
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

	var threads []Thread
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
func (s *Store) CreateMessage(threadID, fromType, fromID, toType, toID, kind, content string) (*Message, error) {
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
			kind, content, delivery_state, business_state
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, messageID, threadID, newSeq, now.UnixMilli(),
		fromType, fromID, toType, toID,
		kind, content, "pending", "open")

	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &Message{
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
		DeliveryState: "pending",
		BusinessState: "open",
	}, nil
}

// GetMessagesFromSeq retrieves messages in a thread starting from a given sequence number.
// This enables cursor-based resumption for WebSocket subscriptions.
func (s *Store) GetMessagesFromSeq(threadID string, fromSeq int, limit int) ([]Message, error) {
	query := `
		SELECT id, thread_id, message_seq, created_at, from_type, from_id, to_type, to_id,
		       kind, content, delivery_state, business_state
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
			&msg.Kind, &msg.Content, &msg.DeliveryState, &msg.BusinessState,
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
