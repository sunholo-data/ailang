package messaging

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sunholo/ailang/internal/agentprotocol"
)

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
