package messaging

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo/ailang/internal/agentprotocol"
)

// TestScanMessages tests scanning messages from filesystem
func TestScanMessages(t *testing.T) {
	// Create temp directory with test messages
	tmpDir := t.TempDir()
	messagesDir := filepath.Join(tmpDir, "messages")

	// Create test envelopes
	testEnv1 := createTestEnvelope("msg1", "agent1", "agent2", "test message 1")
	testEnv2 := createTestEnvelope("msg2", "agent2", "agent1", "test message 2")

	// Write to files
	writeEnvelopeToFile(t, messagesDir, "agent2", "msg1.json", testEnv1)
	writeEnvelopeToFile(t, messagesDir, "agent1", "msg2.json", testEnv2)

	// Scan
	db := setupTestDB(t)
	defer db.Close()

	migration := NewMigration(db, messagesDir)
	envelopes, err := migration.scanMessages()
	if err != nil {
		t.Fatalf("scanMessages failed: %v", err)
	}

	// Verify
	if len(envelopes) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(envelopes))
	}
}

// TestScanMessagesEmptyDirectory tests scanning empty directory
func TestScanMessagesEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	db := setupTestDB(t)
	defer db.Close()

	migration := NewMigration(db, tmpDir)
	envelopes, err := migration.scanMessages()
	if err != nil {
		t.Fatalf("scanMessages failed: %v", err)
	}

	if len(envelopes) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(envelopes))
	}
}

// TestGroupIntoThreads tests grouping messages by correlation_id
func TestGroupIntoThreads(t *testing.T) {
	env1 := createTestEnvelopeWithCorrelation("msg1", "agent1", "agent2", "corr1", "test 1")
	env2 := createTestEnvelopeWithCorrelation("msg2", "agent1", "agent2", "corr1", "test 2")
	env3 := createTestEnvelopeWithCorrelation("msg3", "agent3", "agent4", "corr2", "test 3")

	db := setupTestDB(t)
	defer db.Close()

	migration := NewMigration(db, "")
	threads := migration.groupIntoThreads([]*agentprotocol.Envelope{env1, env2, env3})

	// Should have 2 threads (corr1 and corr2)
	if len(threads) != 2 {
		t.Errorf("Expected 2 threads, got %d", len(threads))
	}

	// Thread corr1 should have 2 messages
	thread1 := threads["corr1"]
	if thread1 == nil {
		t.Fatal("Thread corr1 not found")
	}
	if len(thread1.messages) != 2 {
		t.Errorf("Expected 2 messages in thread corr1, got %d", len(thread1.messages))
	}

	// Thread corr2 should have 1 message
	thread2 := threads["corr2"]
	if thread2 == nil {
		t.Fatal("Thread corr2 not found")
	}
	if len(thread2.messages) != 1 {
		t.Errorf("Expected 1 message in thread corr2, got %d", len(thread2.messages))
	}
}

// TestGroupIntoThreadsNoCorrelation tests fallback to agent pair
func TestGroupIntoThreadsNoCorrelation(t *testing.T) {
	env1 := createTestEnvelope("msg1", "agent1", "agent2", "test 1")
	env2 := createTestEnvelope("msg2", "agent1", "agent2", "test 2")
	env3 := createTestEnvelope("msg3", "agent3", "agent4", "test 3")

	db := setupTestDB(t)
	defer db.Close()

	migration := NewMigration(db, "")
	threads := migration.groupIntoThreads([]*agentprotocol.Envelope{env1, env2, env3})

	// Should have 2 threads (agent1_agent2 and agent3_agent4)
	if len(threads) != 2 {
		t.Errorf("Expected 2 threads, got %d", len(threads))
	}

	// Thread agent1_agent2 should have 2 messages
	thread1 := threads["agent1_agent2"]
	if thread1 == nil {
		t.Fatal("Thread agent1_agent2 not found")
	}
	if len(thread1.messages) != 2 {
		t.Errorf("Expected 2 messages in thread agent1_agent2, got %d", len(thread1.messages))
	}
}

// TestMessageSequenceOrdering tests that messages are ordered by timestamp
func TestMessageSequenceOrdering(t *testing.T) {
	// Create messages with different timestamps (out of order)
	env1 := createTestEnvelopeWithTimestamp("msg1", "agent1", "agent2", "corr1", "2025-01-03T12:00:00Z")
	env2 := createTestEnvelopeWithTimestamp("msg2", "agent1", "agent2", "corr1", "2025-01-01T12:00:00Z")
	env3 := createTestEnvelopeWithTimestamp("msg3", "agent1", "agent2", "corr1", "2025-01-02T12:00:00Z")

	db := setupTestDB(t)
	defer db.Close()

	migration := NewMigration(db, "")
	threads := migration.groupIntoThreads([]*agentprotocol.Envelope{env1, env2, env3})

	thread := threads["corr1"]
	if thread == nil {
		t.Fatal("Thread corr1 not found")
	}

	// Messages should be ordered by timestamp: msg2 (seq 1), msg3 (seq 2), msg1 (seq 3)
	if len(thread.messages) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(thread.messages))
	}

	if thread.messages[0].envelope.MessageID != "msg2" || thread.messages[0].seq != 1 {
		t.Errorf("Expected msg2 with seq 1, got %s with seq %d", thread.messages[0].envelope.MessageID, thread.messages[0].seq)
	}
	if thread.messages[1].envelope.MessageID != "msg3" || thread.messages[1].seq != 2 {
		t.Errorf("Expected msg3 with seq 2, got %s with seq %d", thread.messages[1].envelope.MessageID, thread.messages[1].seq)
	}
	if thread.messages[2].envelope.MessageID != "msg1" || thread.messages[2].seq != 3 {
		t.Errorf("Expected msg1 with seq 3, got %s with seq %d", thread.messages[2].envelope.MessageID, thread.messages[2].seq)
	}
}

// TestMigrateEndToEnd tests full migration from files to database
func TestMigrateEndToEnd(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	messagesDir := filepath.Join(tmpDir, "messages")

	// Create test messages with different timestamps to ensure ordering
	env1 := createTestEnvelopeWithTimestamp("msg1", "agent1", "agent2", "corr1", "2025-01-01T12:00:00Z")
	env2 := createTestEnvelopeWithTimestamp("msg2", "agent2", "agent1", "corr1", "2025-01-01T12:00:01Z")

	writeEnvelopeToFile(t, messagesDir, "agent2", "msg1.json", env1)
	writeEnvelopeToFile(t, messagesDir, "agent1", "msg2.json", env2)

	// Migrate
	db := setupTestDB(t)
	defer db.Close()

	migration := NewMigration(db, messagesDir)
	count, err := migration.Migrate()
	if err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 messages migrated, got %d", count)
	}

	// Verify threads table
	var threadCount int
	err = db.QueryRow("SELECT COUNT(*) FROM threads").Scan(&threadCount)
	if err != nil {
		t.Fatalf("Failed to count threads: %v", err)
	}
	if threadCount != 1 {
		t.Errorf("Expected 1 thread, got %d", threadCount)
	}

	// Verify messages table
	var messageCount int
	err = db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&messageCount)
	if err != nil {
		t.Fatalf("Failed to count messages: %v", err)
	}
	if messageCount != 2 {
		t.Errorf("Expected 2 messages, got %d", messageCount)
	}

	// Verify message_seq ordering
	rows, err := db.Query("SELECT id, message_seq FROM messages WHERE thread_id = ? ORDER BY message_seq", "corr1")
	if err != nil {
		t.Fatalf("Failed to query messages: %v", err)
	}
	defer rows.Close()

	var seq1ID string
	var seq1 int
	if !rows.Next() {
		t.Fatal("Expected first message")
	}
	rows.Scan(&seq1ID, &seq1)

	if seq1ID != "msg1" || seq1 != 1 {
		t.Errorf("Expected msg1 with seq 1, got %s with seq %d", seq1ID, seq1)
	}

	var seq2ID string
	var seq2 int
	if !rows.Next() {
		t.Fatal("Expected second message")
	}
	rows.Scan(&seq2ID, &seq2)

	if seq2ID != "msg2" || seq2 != 2 {
		t.Errorf("Expected msg2 with seq 2, got %s with seq %d", seq2ID, seq2)
	}
}

// TestMigrateEmptyDirectory tests migration with no messages
func TestMigrateEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	db := setupTestDB(t)
	defer db.Close()

	migration := NewMigration(db, tmpDir)
	count, err := migration.Migrate()
	if err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 messages migrated, got %d", count)
	}
}

// TestInferAgentType tests agent type inference
func TestInferAgentType(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	migration := NewMigration(db, "")

	tests := []struct {
		agentID  string
		expected string
	}{
		{"user", "human"},
		{"claude-code", "ailang_instance"},
		{"eval-analyzer", "ailang_instance"},
		{"agent_123", "ailang_instance"},
		{"someagent", "ailang_instance"},
	}

	for _, tt := range tests {
		t.Run(tt.agentID, func(t *testing.T) {
			result := migration.inferAgentType(tt.agentID)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestInferMessageKind tests message kind inference
func TestInferMessageKind(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	migration := NewMigration(db, "")

	tests := []struct {
		messageType string
		payload     map[string]interface{}
		expected    string
	}{
		{"request", map[string]interface{}{}, "directive"},
		{"response", map[string]interface{}{}, "result"},
		{"notification", map[string]interface{}{}, "status"},
		{"unknown", map[string]interface{}{"question": "What?"}, "question"},
		{"unknown", map[string]interface{}{"proposal": "Try this"}, "proposal"},
		{"unknown", map[string]interface{}{}, "status"},
	}

	for _, tt := range tests {
		t.Run(tt.messageType, func(t *testing.T) {
			result := migration.inferMessageKind(tt.messageType, tt.payload)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// Helper functions

func createTestEnvelope(msgID, fromAgent, toAgent, content string) *agentprotocol.Envelope {
	return createTestEnvelopeWithCorrelation(msgID, fromAgent, toAgent, "", content)
}

func createTestEnvelopeWithCorrelation(msgID, fromAgent, toAgent, correlationID, content string) *agentprotocol.Envelope {
	return createTestEnvelopeWithTimestamp(msgID, fromAgent, toAgent, correlationID, "2025-01-01T12:00:00Z")
}

func createTestEnvelopeWithTimestamp(msgID, fromAgent, toAgent, correlationID, timestamp string) *agentprotocol.Envelope {
	env := &agentprotocol.Envelope{
		ProtocolVersion: "1.0.0",
		SchemaVersion:   "1.0.0",
		MessageID:       msgID,
		CorrelationID:   correlationID,
		Timestamp:       timestamp,
		FromAgent:       fromAgent,
		ToAgent:         toAgent,
		MessageType:     "request",
		Payload: map[string]interface{}{
			"content": timestamp,
		},
	}

	// Set content if provided
	if timestamp != "" {
		env.Payload["content"] = timestamp
	}

	return env
}

func writeEnvelopeToFile(t *testing.T, baseDir, agentID, filename string, env *agentprotocol.Envelope) {
	t.Helper()

	dir := filepath.Join(baseDir, agentID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create directory %s: %v", dir, err)
	}

	data, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal envelope: %v", err)
	}

	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("Failed to write file %s: %v", path, err)
	}
}
