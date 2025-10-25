// Package agent_protocol implements the agent-to-agent communication protocol
// for AILANG's autonomous development system.
//
// This protocol enables multi-agent cooperation as described in M-AGENT-PROTOCOL
// design doc. It provides:
//   - File-based message passing with at-least-once delivery
//   - Crash-safe handoff (temp → fsync → atomic rename)
//   - Idempotency via message_id deduplication
//   - SQLite control plane for leases, history, and metrics
//
// Architecture:
//   - Files (.ailang/state/messages/) = Observable transport
//   - SQLite (.ailang/state/agents.db) = Control plane (dedupe, leases, history)
//
// Design doc: design_docs/planned/M-AGENT-PROTOCOL.md
package agentprotocol

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Envelope represents a message in the agent protocol.
// All fields match the specification in M-AGENT-PROTOCOL.md.
type Envelope struct {
	// Protocol versioning
	ProtocolVersion string `json:"protocol_version"` // e.g., "1.0.0"
	SchemaVersion   string `json:"schema_version"`   // e.g., "1.0.0"

	// Message identification
	MessageID       string  `json:"message_id"`         // Globally unique (UUID)
	CorrelationID   string  `json:"correlation_id"`     // Groups related messages (e.g., "cycle_20251023_001")
	TraceID         string  `json:"trace_id"`           // Distributed tracing
	ParentMessageID *string `json:"parent_message_id"`  // For request-response chains
	Timestamp       string  `json:"timestamp"`          // RFC3339 with Z
	TTLSeconds      int     `json:"ttl_seconds"`        // Time-to-live in seconds
	Deadline        string  `json:"deadline,omitempty"` // RFC3339 with Z

	// Routing
	FromAgent   string `json:"from_agent"`   // Sender agent ID
	ToAgent     string `json:"to_agent"`     // Receiver agent ID
	MessageType string `json:"message_type"` // "request", "response", "notification"

	// Retry tracking
	Retries int `json:"retries"` // Sender-maintained retry counter

	// Payload
	PayloadSchema string                 `json:"payload_schema"` // URI for schema (e.g., "https://ailang.dev/schemas/run_eval_baseline/v1.json")
	Payload       map[string]interface{} `json:"payload"`        // Actual message content

	// Effects
	DeclaredEffects []string `json:"declared_effects"` // e.g., ["IO", "FS", "Net"]

	// Security
	SignatureAlg string `json:"signature_alg,omitempty"` // e.g., "hmac-sha256"
	KID          string `json:"kid,omitempty"`           // Key ID for rotation
	Signature    string `json:"signature,omitempty"`     // HMAC signature
}

// MessageWriter handles atomic message file writing.
type MessageWriter struct {
	stateDir string // Root directory (.ailang/state)
	mu       sync.Mutex
}

// NewMessageWriter creates a new message writer.
func NewMessageWriter(stateDir string) *MessageWriter {
	return &MessageWriter{
		stateDir: stateDir,
	}
}

// WriteMessage writes a message atomically to the recipient's inbox.
// It uses temp → fsync → atomic rename to ensure crash safety.
//
// Returns the path to the written message file.
func (w *MessageWriter) WriteMessage(env *Envelope) (string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Validate envelope
	if err := validateEnvelope(env); err != nil {
		return "", fmt.Errorf("invalid envelope: %w", err)
	}

	// Ensure messages directory exists
	messagesDir := filepath.Join(w.stateDir, "messages", env.ToAgent)
	if err := os.MkdirAll(messagesDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create messages directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal envelope: %w", err)
	}

	// Write to temp file first
	tempFile := filepath.Join(messagesDir, fmt.Sprintf("%s.tmp", env.MessageID))
	f, err := os.Create(tempFile)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	// Write data
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tempFile)
		return "", fmt.Errorf("failed to write data: %w", err)
	}

	// Fsync to ensure data is on disk
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tempFile)
		return "", fmt.Errorf("failed to fsync: %w", err)
	}

	f.Close()

	// Atomic rename to final location
	finalPath := filepath.Join(messagesDir, fmt.Sprintf("%s.pending.json", env.MessageID))
	if err := os.Rename(tempFile, finalPath); err != nil {
		os.Remove(tempFile)
		return "", fmt.Errorf("failed to rename to final path: %w", err)
	}

	return finalPath, nil
}

// MessageReader handles message file reading and deduplication.
type MessageReader struct {
	stateDir string // Root directory (.ailang/state)
	seen     map[string]bool
	mu       sync.RWMutex
}

// NewMessageReader creates a new message reader.
func NewMessageReader(stateDir string) *MessageReader {
	return &MessageReader{
		stateDir: stateDir,
		seen:     make(map[string]bool),
	}
}

// ScanPendingMessages scans for pending messages in the given agent's inbox.
// Returns a list of message file paths.
func (r *MessageReader) ScanPendingMessages(agentID string) ([]string, error) {
	messagesDir := filepath.Join(r.stateDir, "messages", agentID)

	// Check if directory exists
	if _, err := os.Stat(messagesDir); os.IsNotExist(err) {
		return nil, nil // No messages yet
	}

	// Read directory
	entries, err := os.ReadDir(messagesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read messages directory: %w", err)
	}

	var pendingFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process .pending.json files
		if filepath.Ext(entry.Name()) == ".json" &&
			len(entry.Name()) > 13 &&
			entry.Name()[len(entry.Name())-13:] == ".pending.json" {
			pendingFiles = append(pendingFiles, filepath.Join(messagesDir, entry.Name()))
		}
	}

	return pendingFiles, nil
}

// ReadMessage reads and parses a message from a file path.
// Returns nil if the message has already been seen (deduplication).
func (r *MessageReader) ReadMessage(filePath string) (*Envelope, error) {
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read message file: %w", err)
	}

	// Parse JSON
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("failed to unmarshal envelope: %w", err)
	}

	// Deduplication check
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.seen[env.MessageID] {
		return nil, nil // Already processed
	}

	// Mark as seen
	r.seen[env.MessageID] = true

	return &env, nil
}

// MarkSeen explicitly marks a message ID as seen (for testing).
func (r *MessageReader) MarkSeen(messageID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seen[messageID] = true
}

// validateEnvelope checks that required fields are present and valid.
func validateEnvelope(env *Envelope) error {
	if env == nil {
		return fmt.Errorf("envelope cannot be nil")
	}
	if env.ProtocolVersion == "" {
		return fmt.Errorf("protocol_version is required")
	}
	if env.MessageID == "" {
		return fmt.Errorf("message_id is required")
	}
	if env.FromAgent == "" {
		return fmt.Errorf("from_agent is required")
	}
	if env.ToAgent == "" {
		return fmt.Errorf("to_agent is required")
	}
	if env.MessageType == "" {
		return fmt.Errorf("message_type is required")
	}
	if env.Timestamp == "" {
		return fmt.Errorf("timestamp is required")
	}

	// Validate message type
	validTypes := map[string]bool{
		"request":      true,
		"response":     true,
		"notification": true,
	}
	if !validTypes[env.MessageType] {
		return fmt.Errorf("invalid message_type: %s (must be request, response, or notification)", env.MessageType)
	}

	return nil
}

// GenerateMessageID generates a unique message ID.
// Format: msg_YYYYMMDD_HHMMSS_random
func GenerateMessageID() string {
	now := time.Now().UTC()
	timestamp := now.Format("20060102_150405")

	// Add random suffix for uniqueness
	randomBytes := make([]byte, 6)
	if _, err := rand.Read(randomBytes); err != nil {
		panic(fmt.Sprintf("failed to generate random bytes: %v", err))
	}

	return fmt.Sprintf("msg_%s_%x", timestamp, randomBytes)
}

// GenerateCorrelationID generates a correlation ID for grouping related messages.
// Format: cycle_YYYYMMDD_NNN
func GenerateCorrelationID() string {
	now := time.Now().UTC()
	date := now.Format("20060102")

	// Use hour + minute as sequence number (simple approach)
	seq := now.Hour()*60 + now.Minute()

	return fmt.Sprintf("cycle_%s_%03d", date, seq)
}

// GenerateTraceID generates a trace ID for distributed tracing.
// Format: trace_random16chars
func GenerateTraceID() string {
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		panic(fmt.Sprintf("failed to generate random bytes: %v", err))
	}

	return fmt.Sprintf("trace_%x", randomBytes)
}

// UserInbox manages the user's message inbox with read/unread/archive support.
type UserInbox struct {
	stateDir string
}

// NewUserInbox creates a new user inbox manager.
func NewUserInbox(stateDir string) *UserInbox {
	return &UserInbox{
		stateDir: stateDir,
	}
}

// SendToUser sends a message to the user's inbox.
// Messages start in the _unread folder.
func (u *UserInbox) SendToUser(env *Envelope) (string, error) {
	// Ensure inbox directories exist
	unreadDir := filepath.Join(u.stateDir, "messages", "inbox", "user", "_unread")
	if err := os.MkdirAll(unreadDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create unread directory: %w", err)
	}

	// Marshal message
	data, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal message: %w", err)
	}

	// Write to unread folder
	msgPath := filepath.Join(unreadDir, fmt.Sprintf("%s.json", env.MessageID))
	if err := os.WriteFile(msgPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write message: %w", err)
	}

	return msgPath, nil
}

// GetUnreadMessages returns all unread messages.
func (u *UserInbox) GetUnreadMessages() ([]*Envelope, error) {
	unreadDir := filepath.Join(u.stateDir, "messages", "inbox", "user", "_unread")
	return u.readMessagesFromDir(unreadDir)
}

// GetReadMessages returns all read messages.
func (u *UserInbox) GetReadMessages() ([]*Envelope, error) {
	readDir := filepath.Join(u.stateDir, "messages", "inbox", "user", "_read")
	return u.readMessagesFromDir(readDir)
}

// GetArchivedMessages returns all archived messages.
func (u *UserInbox) GetArchivedMessages() ([]*Envelope, error) {
	archiveDir := filepath.Join(u.stateDir, "messages", "inbox", "user", "_archive")
	return u.readMessagesFromDir(archiveDir)
}

// MarkAsRead moves a message from _unread to _read.
func (u *UserInbox) MarkAsRead(messageID string) error {
	unreadDir := filepath.Join(u.stateDir, "messages", "inbox", "user", "_unread")
	readDir := filepath.Join(u.stateDir, "messages", "inbox", "user", "_read")

	// Ensure read directory exists
	if err := os.MkdirAll(readDir, 0755); err != nil {
		return fmt.Errorf("failed to create read directory: %w", err)
	}

	// Move file
	srcPath := filepath.Join(unreadDir, fmt.Sprintf("%s.json", messageID))
	dstPath := filepath.Join(readDir, fmt.Sprintf("%s.json", messageID))

	return os.Rename(srcPath, dstPath)
}

// MarkAsArchived moves a message to _archive (from either _unread or _read).
func (u *UserInbox) MarkAsArchived(messageID string) error {
	archiveDir := filepath.Join(u.stateDir, "messages", "inbox", "user", "_archive")

	// Ensure archive directory exists
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return fmt.Errorf("failed to create archive directory: %w", err)
	}

	// Try to move from unread
	unreadPath := filepath.Join(u.stateDir, "messages", "inbox", "user", "_unread", fmt.Sprintf("%s.json", messageID))
	archivePath := filepath.Join(archiveDir, fmt.Sprintf("%s.json", messageID))

	if err := os.Rename(unreadPath, archivePath); err == nil {
		return nil // Success
	}

	// Try to move from read
	readPath := filepath.Join(u.stateDir, "messages", "inbox", "user", "_read", fmt.Sprintf("%s.json", messageID))
	if err := os.Rename(readPath, archivePath); err != nil {
		return fmt.Errorf("failed to archive message (not found in unread or read): %w", err)
	}

	return nil
}

// DeleteMessage permanently deletes a message from any folder.
func (u *UserInbox) DeleteMessage(messageID string) error {
	// Try all three folders
	folders := []string{"_unread", "_read", "_archive"}
	for _, folder := range folders {
		msgPath := filepath.Join(u.stateDir, "messages", "inbox", "user", folder, fmt.Sprintf("%s.json", messageID))
		if err := os.Remove(msgPath); err == nil {
			return nil // Success
		}
	}

	return fmt.Errorf("message not found in any folder: %s", messageID)
}

// readMessagesFromDir reads all messages from a directory.
func (u *UserInbox) readMessagesFromDir(dir string) ([]*Envelope, error) {
	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil // No messages yet
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var messages []*Envelope
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		// Read message file
		msgPath := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(msgPath)
		if err != nil {
			continue // Skip files we can't read
		}

		var env Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			continue // Skip invalid JSON
		}

		messages = append(messages, &env)
	}

	return messages, nil
}

// DeadLetterQueue manages failed messages that couldn't be processed after retries.
type DeadLetterQueue struct {
	stateDir string
}

// NewDeadLetterQueue creates a new DLQ manager.
func NewDeadLetterQueue(stateDir string) *DeadLetterQueue {
	return &DeadLetterQueue{
		stateDir: stateDir,
	}
}

// DLQEntry represents a message in the dead letter queue with failure metadata.
type DLQEntry struct {
	*Envelope
	FailureReason string    `json:"failure_reason"`
	FailedAt      time.Time `json:"failed_at"`
	RetryCount    int       `json:"retry_count"`
	OriginalPath  string    `json:"original_path"`
	StackTrace    string    `json:"stack_trace,omitempty"`
}

// MoveToDeadLetter moves a message to the DLQ with failure metadata.
func (dlq *DeadLetterQueue) MoveToDeadLetter(env *Envelope, reason, stackTrace string) (string, error) {
	// Ensure DLQ directory exists
	dlqDir := filepath.Join(dlq.stateDir, "messages", "dead_letter")
	if err := os.MkdirAll(dlqDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create DLQ directory: %w", err)
	}

	// Create DLQ entry
	entry := &DLQEntry{
		Envelope:      env,
		FailureReason: reason,
		FailedAt:      time.Now().UTC(),
		RetryCount:    env.Retries,
		StackTrace:    stackTrace,
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal DLQ entry: %w", err)
	}

	// Write to DLQ
	dlqPath := filepath.Join(dlqDir, fmt.Sprintf("%s.dlq.json", env.MessageID))
	if err := os.WriteFile(dlqPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write DLQ entry: %w", err)
	}

	return dlqPath, nil
}

// GetDeadLetterMessages retrieves all messages in the DLQ.
func (dlq *DeadLetterQueue) GetDeadLetterMessages() ([]*DLQEntry, error) {
	dlqDir := filepath.Join(dlq.stateDir, "messages", "dead_letter")

	// Check if directory exists
	if _, err := os.Stat(dlqDir); os.IsNotExist(err) {
		return nil, nil // No DLQ entries yet
	}

	entries, err := os.ReadDir(dlqDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read DLQ directory: %w", err)
	}

	var messages []*DLQEntry
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		// Read DLQ entry
		entryPath := filepath.Join(dlqDir, entry.Name())
		data, err := os.ReadFile(entryPath)
		if err != nil {
			continue // Skip files we can't read
		}

		var dlqEntry DLQEntry
		if err := json.Unmarshal(data, &dlqEntry); err != nil {
			continue // Skip invalid JSON
		}

		messages = append(messages, &dlqEntry)
	}

	return messages, nil
}

// DeleteDeadLetterMessage removes a message from the DLQ.
func (dlq *DeadLetterQueue) DeleteDeadLetterMessage(messageID string) error {
	dlqPath := filepath.Join(dlq.stateDir, "messages", "dead_letter", fmt.Sprintf("%s.dlq.json", messageID))
	if err := os.Remove(dlqPath); err != nil {
		return fmt.Errorf("failed to delete DLQ message: %w", err)
	}
	return nil
}

// RetryFromDeadLetter attempts to retry a message from the DLQ.
// This removes it from the DLQ and returns the envelope for re-processing.
func (dlq *DeadLetterQueue) RetryFromDeadLetter(messageID string) (*Envelope, error) {
	dlqPath := filepath.Join(dlq.stateDir, "messages", "dead_letter", fmt.Sprintf("%s.dlq.json", messageID))

	// Read DLQ entry
	data, err := os.ReadFile(dlqPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read DLQ entry: %w", err)
	}

	var entry DLQEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal DLQ entry: %w", err)
	}

	// Delete from DLQ
	if err := os.Remove(dlqPath); err != nil {
		return nil, fmt.Errorf("failed to remove from DLQ: %w", err)
	}

	// Reset retry counter for fresh attempt
	entry.Envelope.Retries = 0

	return entry.Envelope, nil
}
