package messaging

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/agentprotocol"
)

// Migration handles migration from file-based agent inbox to SQLite.
type Migration struct {
	db              *sql.DB
	messagesBaseDir string
}

// NewMigration creates a new migration instance.
// messagesBaseDir should be .ailang/state/messages/
func NewMigration(db *sql.DB, messagesBaseDir string) *Migration {
	return &Migration{
		db:              db,
		messagesBaseDir: messagesBaseDir,
	}
}

// Migrate scans the file-based inbox and migrates all messages to SQLite.
// It creates threads based on correlation_id or from_agent/to_agent pairs.
// Returns the number of messages migrated and any error encountered.
func (m *Migration) Migrate() (int, error) {
	// Scan all messages from file-based inbox
	envelopes, err := m.scanMessages()
	if err != nil {
		return 0, fmt.Errorf("failed to scan messages: %w", err)
	}

	if len(envelopes) == 0 {
		return 0, nil // No messages to migrate
	}

	// Group messages into threads
	threads := m.groupIntoThreads(envelopes)

	// Migrate in a transaction
	tx, err := m.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback() // Ignore error - commit will fail if needed
	}()

	// Create threads first
	for threadID, threadInfo := range threads {
		if err := m.createThread(tx, threadID, threadInfo); err != nil {
			return 0, fmt.Errorf("failed to create thread %s: %w", threadID, err)
		}
	}

	// Migrate messages
	messageCount := 0
	for _, envs := range threads {
		for _, env := range envs.messages {
			if err := m.createMessage(tx, env.threadID, env.seq, env.envelope); err != nil {
				return 0, fmt.Errorf("failed to migrate message %s: %w", env.envelope.MessageID, err)
			}
			messageCount++
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return messageCount, nil
}

// threadInfo groups messages that belong to the same thread
type threadInfo struct {
	id        string
	title     string
	createdAt int64
	createdBy string
	messages  []messageWithSeq
}

type messageWithSeq struct {
	threadID string
	seq      int
	envelope *agentprotocol.Envelope
}

// scanMessages recursively scans the messages directory and reads all JSON files
func (m *Migration) scanMessages() ([]*agentprotocol.Envelope, error) {
	var envelopes []*agentprotocol.Envelope

	err := filepath.Walk(m.messagesBaseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process .json files
		if filepath.Ext(path) != ".json" {
			return nil
		}

		// Read and unmarshal envelope
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		var env agentprotocol.Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			return fmt.Errorf("failed to unmarshal %s: %w", path, err)
		}

		envelopes = append(envelopes, &env)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return envelopes, nil
}

// groupIntoThreads groups messages into threads based on correlation_id or agent pairs
func (m *Migration) groupIntoThreads(envelopes []*agentprotocol.Envelope) map[string]*threadInfo {
	threads := make(map[string]*threadInfo)

	for _, env := range envelopes {
		// Determine thread ID
		// Priority: correlation_id > from_agent-to_agent pair
		threadID := env.CorrelationID
		if threadID == "" {
			// Fallback: use agent pair as thread ID
			threadID = fmt.Sprintf("%s_%s", env.FromAgent, env.ToAgent)
		}

		// Get or create thread info
		thread, exists := threads[threadID]
		if !exists {
			thread = &threadInfo{
				id:        threadID,
				title:     m.deriveThreadTitle(env),
				createdAt: m.parseTimestamp(env.Timestamp),
				createdBy: env.FromAgent,
				messages:  []messageWithSeq{},
			}
			threads[threadID] = thread
		}

		// Add message to thread
		thread.messages = append(thread.messages, messageWithSeq{
			threadID: threadID,
			envelope: env,
		})
	}

	// Sort messages within each thread by timestamp and assign sequence numbers
	for _, thread := range threads {
		sort.Slice(thread.messages, func(i, j int) bool {
			ti := m.parseTimestamp(thread.messages[i].envelope.Timestamp)
			tj := m.parseTimestamp(thread.messages[j].envelope.Timestamp)
			return ti < tj
		})

		// Assign sequence numbers (1-indexed)
		for i := range thread.messages {
			thread.messages[i].seq = i + 1
		}

		// Update thread createdAt to earliest message
		if len(thread.messages) > 0 {
			thread.createdAt = m.parseTimestamp(thread.messages[0].envelope.Timestamp)
		}
	}

	return threads
}

// deriveThreadTitle creates a human-readable title from the first message
func (m *Migration) deriveThreadTitle(env *agentprotocol.Envelope) string {
	// Try to extract title from payload
	if content, ok := env.Payload["content"].(string); ok && content != "" {
		// Truncate to first 100 characters
		if len(content) > 100 {
			return content[:97] + "..."
		}
		return content
	}

	// Fallback: use message type and agents
	return fmt.Sprintf("%s: %s → %s", env.MessageType, env.FromAgent, env.ToAgent)
}

// parseTimestamp converts RFC3339 timestamp to Unix milliseconds
func (m *Migration) parseTimestamp(timestamp string) int64 {
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		// Fallback to current time if parsing fails
		return time.Now().UnixMilli()
	}
	return t.UnixMilli()
}

// createThread inserts a thread into the database
func (m *Migration) createThread(tx *sql.Tx, threadID string, thread *threadInfo) error {
	_, err := tx.Exec(`
		INSERT INTO threads (id, title, created_at, created_by_type, created_by_id, status, last_seq, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		threadID,
		thread.title,
		thread.createdAt,
		m.inferAgentType(thread.createdBy),
		thread.createdBy,
		"active",
		len(thread.messages), // last_seq is the highest message_seq
		thread.createdAt,
	)

	return err
}

// createMessage inserts a message into the database
func (m *Migration) createMessage(tx *sql.Tx, threadID string, seq int, env *agentprotocol.Envelope) error {
	// Marshal metadata (original envelope fields not in new schema)
	metadata := map[string]interface{}{
		"protocol_version":  env.ProtocolVersion,
		"schema_version":    env.SchemaVersion,
		"correlation_id":    env.CorrelationID,
		"trace_id":          env.TraceID,
		"parent_message_id": env.ParentMessageID,
		"ttl_seconds":       env.TTLSeconds,
		"payload_schema":    env.PayloadSchema,
		"declared_effects":  env.DeclaredEffects,
		"retries":           env.Retries,
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Marshal payload
	payloadJSON, err := json.Marshal(env.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Determine message kind from message_type
	kind := m.inferMessageKind(env.MessageType, env.Payload)

	_, err = tx.Exec(`
		INSERT INTO messages (
			id, thread_id, message_seq, created_at,
			from_type, from_id, to_type, to_id,
			kind, subject, content, metadata_json,
			delivery_state, business_state
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		env.MessageID,
		threadID,
		seq,
		m.parseTimestamp(env.Timestamp),
		m.inferAgentType(env.FromAgent),
		env.FromAgent,
		m.inferAgentType(env.ToAgent),
		env.ToAgent,
		kind,
		nil, // subject (not in original schema)
		string(payloadJSON),
		string(metadataJSON),
		"acked", // Existing messages are already delivered
		"open",  // Default business state
	)

	return err
}

// inferAgentType determines if an agent ID represents a human or ailang instance
func (m *Migration) inferAgentType(agentID string) string {
	// User inbox messages go to "user"
	if agentID == "user" {
		return "human"
	}

	// Agent IDs typically contain "-" (e.g., "claude-code", "eval-analyzer")
	if strings.Contains(agentID, "-") || strings.Contains(agentID, "_") {
		return "ailang_instance"
	}

	// Default to ailang_instance
	return "ailang_instance"
}

// inferMessageKind maps message_type to the new schema's kind field
func (m *Migration) inferMessageKind(messageType string, payload map[string]interface{}) string {
	switch messageType {
	case "request":
		return "directive"
	case "response":
		return "result"
	case "notification":
		return "status"
	default:
		// Try to infer from payload
		if _, ok := payload["question"]; ok {
			return "question"
		}
		if _, ok := payload["proposal"]; ok {
			return "proposal"
		}
		return "status" // Default fallback
	}
}
