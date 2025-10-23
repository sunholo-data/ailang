package agentprotocol

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestEnvelopeMarshaling tests JSON marshaling/unmarshaling of Envelope.
func TestEnvelopeMarshaling(t *testing.T) {
	env := &Envelope{
		ProtocolVersion: "1.0.0",
		SchemaVersion:   "1.0.0",
		MessageID:       "msg_20251023_143022_abc123",
		CorrelationID:   "cycle_20251023_001",
		TraceID:         "trace_xyz789",
		Timestamp:       "2025-10-23T14:30:22Z",
		TTLSeconds:      3600,
		FromAgent:       "test-sender",
		ToAgent:         "test-receiver",
		MessageType:     "request",
		Retries:         0,
		PayloadSchema:   "https://ailang.dev/schemas/test/v1.json",
		Payload: map[string]interface{}{
			"action": "test_action",
			"params": map[string]interface{}{
				"key": "value",
			},
		},
		DeclaredEffects: []string{"IO", "FS"},
	}

	// Marshal
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("failed to marshal envelope: %v", err)
	}

	// Unmarshal
	var decoded Envelope
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal envelope: %v", err)
	}

	// Verify fields
	if decoded.MessageID != env.MessageID {
		t.Errorf("MessageID mismatch: got %s, want %s", decoded.MessageID, env.MessageID)
	}
	if decoded.FromAgent != env.FromAgent {
		t.Errorf("FromAgent mismatch: got %s, want %s", decoded.FromAgent, env.FromAgent)
	}
	if decoded.ToAgent != env.ToAgent {
		t.Errorf("ToAgent mismatch: got %s, want %s", decoded.ToAgent, env.ToAgent)
	}
}

// TestValidateEnvelope tests envelope validation.
func TestValidateEnvelope(t *testing.T) {
	tests := []struct {
		name    string
		env     *Envelope
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid envelope",
			env: &Envelope{
				ProtocolVersion: "1.0.0",
				MessageID:       "msg_001",
				FromAgent:       "sender",
				ToAgent:         "receiver",
				MessageType:     "request",
				Timestamp:       "2025-10-23T14:30:22Z",
			},
			wantErr: false,
		},
		{
			name: "missing protocol_version",
			env: &Envelope{
				MessageID:   "msg_001",
				FromAgent:   "sender",
				ToAgent:     "receiver",
				MessageType: "request",
				Timestamp:   "2025-10-23T14:30:22Z",
			},
			wantErr: true,
			errMsg:  "protocol_version is required",
		},
		{
			name: "missing message_id",
			env: &Envelope{
				ProtocolVersion: "1.0.0",
				FromAgent:       "sender",
				ToAgent:         "receiver",
				MessageType:     "request",
				Timestamp:       "2025-10-23T14:30:22Z",
			},
			wantErr: true,
			errMsg:  "message_id is required",
		},
		{
			name: "missing from_agent",
			env: &Envelope{
				ProtocolVersion: "1.0.0",
				MessageID:       "msg_001",
				ToAgent:         "receiver",
				MessageType:     "request",
				Timestamp:       "2025-10-23T14:30:22Z",
			},
			wantErr: true,
			errMsg:  "from_agent is required",
		},
		{
			name: "missing to_agent",
			env: &Envelope{
				ProtocolVersion: "1.0.0",
				MessageID:       "msg_001",
				FromAgent:       "sender",
				MessageType:     "request",
				Timestamp:       "2025-10-23T14:30:22Z",
			},
			wantErr: true,
			errMsg:  "to_agent is required",
		},
		{
			name: "invalid message_type",
			env: &Envelope{
				ProtocolVersion: "1.0.0",
				MessageID:       "msg_001",
				FromAgent:       "sender",
				ToAgent:         "receiver",
				MessageType:     "invalid",
				Timestamp:       "2025-10-23T14:30:22Z",
			},
			wantErr: true,
			errMsg:  "invalid message_type: invalid (must be request, response, or notification)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEnvelope(tt.env)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("error message mismatch: got %q, want %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestMessageWriterAtomicWrite tests atomic message writing.
func TestMessageWriterAtomicWrite(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	writer := NewMessageWriter(tmpDir)

	env := &Envelope{
		ProtocolVersion: "1.0.0",
		SchemaVersion:   "1.0.0",
		MessageID:       GenerateMessageID(),
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		FromAgent:       "test-sender",
		ToAgent:         "test-receiver",
		MessageType:     "request",
		Payload: map[string]interface{}{
			"test": "data",
		},
	}

	// Write message
	path, err := writer.WriteMessage(env)
	if err != nil {
		t.Fatalf("failed to write message: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("message file does not exist: %s", path)
	}

	// Verify file is .pending.json
	if filepath.Ext(path) != ".json" {
		t.Errorf("wrong file extension: got %s, want .json", filepath.Ext(path))
	}

	// Verify no .tmp file left behind
	tmpPath := filepath.Join(tmpDir, "messages", env.ToAgent, env.MessageID+".tmp")
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Errorf("temp file not cleaned up: %s", tmpPath)
	}

	// Read and verify content
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read message file: %v", err)
	}

	var decoded Envelope
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal message: %v", err)
	}

	if decoded.MessageID != env.MessageID {
		t.Errorf("MessageID mismatch: got %s, want %s", decoded.MessageID, env.MessageID)
	}
}

// TestMessageWriterInvalidEnvelope tests error handling for invalid envelopes.
func TestMessageWriterInvalidEnvelope(t *testing.T) {
	tmpDir := t.TempDir()
	writer := NewMessageWriter(tmpDir)

	// Missing required field
	env := &Envelope{
		MessageID: "msg_001",
		// Missing ProtocolVersion, FromAgent, ToAgent, etc.
	}

	_, err := writer.WriteMessage(env)
	if err == nil {
		t.Errorf("expected error for invalid envelope, got nil")
	}
}

// TestMessageReaderScanPending tests scanning for pending messages.
func TestMessageReaderScanPending(t *testing.T) {
	tmpDir := t.TempDir()
	agentID := "test-agent"

	// Create messages directory
	messagesDir := filepath.Join(tmpDir, "messages", agentID)
	if err := os.MkdirAll(messagesDir, 0755); err != nil {
		t.Fatalf("failed to create messages directory: %v", err)
	}

	// Create some test message files
	files := []string{
		"msg_001.pending.json",
		"msg_002.pending.json",
		"msg_003.processing.json", // Should not be picked up
	}

	for _, file := range files {
		path := filepath.Join(messagesDir, file)
		if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	reader := NewMessageReader(tmpDir)
	pending, err := reader.ScanPendingMessages(agentID)
	if err != nil {
		t.Fatalf("failed to scan pending messages: %v", err)
	}

	// Should find 2 .pending.json files
	if len(pending) != 2 {
		t.Errorf("expected 2 pending messages, got %d", len(pending))
	}
}

// TestMessageReaderDeduplication tests message deduplication.
func TestMessageReaderDeduplication(t *testing.T) {
	tmpDir := t.TempDir()
	agentID := "test-agent"

	// Create a test message file
	env := &Envelope{
		ProtocolVersion: "1.0.0",
		SchemaVersion:   "1.0.0",
		MessageID:       "msg_dedup_test",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		FromAgent:       "sender",
		ToAgent:         agentID,
		MessageType:     "request",
		Payload:         map[string]interface{}{},
	}

	messagesDir := filepath.Join(tmpDir, "messages", agentID)
	if err := os.MkdirAll(messagesDir, 0755); err != nil {
		t.Fatalf("failed to create messages directory: %v", err)
	}

	path := filepath.Join(messagesDir, "msg_dedup_test.pending.json")
	data, _ := json.Marshal(env)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to create test message: %v", err)
	}

	reader := NewMessageReader(tmpDir)

	// First read - should succeed
	msg1, err := reader.ReadMessage(path)
	if err != nil {
		t.Fatalf("first read failed: %v", err)
	}
	if msg1 == nil {
		t.Fatalf("first read returned nil")
	}

	// Second read - should return nil (deduplicated)
	msg2, err := reader.ReadMessage(path)
	if err != nil {
		t.Fatalf("second read failed: %v", err)
	}
	if msg2 != nil {
		t.Errorf("second read should return nil (deduplicated), got %v", msg2)
	}
}

// TestMessageReaderInvalidJSON tests error handling for invalid JSON.
func TestMessageReaderInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	agentID := "test-agent"

	messagesDir := filepath.Join(tmpDir, "messages", agentID)
	if err := os.MkdirAll(messagesDir, 0755); err != nil {
		t.Fatalf("failed to create messages directory: %v", err)
	}

	// Create invalid JSON file
	path := filepath.Join(messagesDir, "msg_invalid.pending.json")
	if err := os.WriteFile(path, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	reader := NewMessageReader(tmpDir)
	_, err := reader.ReadMessage(path)
	if err == nil {
		t.Errorf("expected error for invalid JSON, got nil")
	}
}

// TestMessageReaderNoMessages tests scanning when no messages exist.
func TestMessageReaderNoMessages(t *testing.T) {
	tmpDir := t.TempDir()
	reader := NewMessageReader(tmpDir)

	// Scan non-existent agent directory
	pending, err := reader.ScanPendingMessages("nonexistent-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pending != nil {
		t.Errorf("expected nil for non-existent directory, got %v", pending)
	}
}

// TestGenerateMessageID tests message ID generation.
func TestGenerateMessageID(t *testing.T) {
	id1 := GenerateMessageID()
	id2 := GenerateMessageID()

	// Should be unique
	if id1 == id2 {
		t.Errorf("message IDs should be unique, got duplicate: %s", id1)
	}

	// Should start with "msg_"
	if len(id1) < 4 || id1[:4] != "msg_" {
		t.Errorf("message ID should start with 'msg_', got: %s", id1)
	}
}

// TestGenerateCorrelationID tests correlation ID generation.
func TestGenerateCorrelationID(t *testing.T) {
	id := GenerateCorrelationID()

	// Should start with "cycle_"
	if len(id) < 6 || id[:6] != "cycle_" {
		t.Errorf("correlation ID should start with 'cycle_', got: %s", id)
	}
}

// TestGenerateTraceID tests trace ID generation.
func TestGenerateTraceID(t *testing.T) {
	id1 := GenerateTraceID()
	id2 := GenerateTraceID()

	// Should be unique
	if id1 == id2 {
		t.Errorf("trace IDs should be unique, got duplicate: %s", id1)
	}

	// Should start with "trace_"
	if len(id1) < 6 || id1[:6] != "trace_" {
		t.Errorf("trace ID should start with 'trace_', got: %s", id1)
	}
}

// TestConcurrentWrites tests concurrent message writes (stress test).
func TestConcurrentWrites(t *testing.T) {
	tmpDir := t.TempDir()
	writer := NewMessageWriter(tmpDir)

	// Write 10 messages concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			env := &Envelope{
				ProtocolVersion: "1.0.0",
				SchemaVersion:   "1.0.0",
				MessageID:       GenerateMessageID(),
				Timestamp:       time.Now().UTC().Format(time.RFC3339),
				FromAgent:       "sender",
				ToAgent:         "receiver",
				MessageType:     "request",
				Payload: map[string]interface{}{
					"index": idx,
				},
			}

			_, err := writer.WriteMessage(env)
			if err != nil {
				t.Errorf("concurrent write %d failed: %v", idx, err)
			}
			done <- true
		}(i)
	}

	// Wait for all writes to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all files were created
	reader := NewMessageReader(tmpDir)
	pending, err := reader.ScanPendingMessages("receiver")
	if err != nil {
		t.Fatalf("failed to scan messages: %v", err)
	}

	if len(pending) != 10 {
		t.Errorf("expected 10 messages, got %d", len(pending))
	}
}

// TestMarkSeen tests the MarkSeen helper.
func TestMarkSeen(t *testing.T) {
	tmpDir := t.TempDir()
	reader := NewMessageReader(tmpDir)

	// Mark a message as seen
	reader.MarkSeen("msg_test_123")

	// Verify it's in the seen map
	reader.mu.RLock()
	seen := reader.seen["msg_test_123"]
	reader.mu.RUnlock()

	if !seen {
		t.Errorf("expected message to be marked as seen")
	}
}

// TestWriteMessageErrorHandling tests error paths in WriteMessage.
func TestWriteMessageErrorHandling(t *testing.T) {
	tmpDir := t.TempDir()
	writer := NewMessageWriter(tmpDir)

	// Test with nil envelope
	_, err := writer.WriteMessage(nil)
	if err == nil {
		t.Errorf("expected error for nil envelope")
	}

	// Test with invalid directory (read-only)
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0444); err != nil {
		t.Fatalf("failed to create readonly dir: %v", err)
	}

	writerRO := NewMessageWriter(readOnlyDir)
	env := &Envelope{
		ProtocolVersion: "1.0.0",
		MessageID:       "msg_001",
		FromAgent:       "sender",
		ToAgent:         "receiver",
		MessageType:     "request",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
	}

	_, err = writerRO.WriteMessage(env)
	if err == nil {
		t.Errorf("expected error for read-only directory")
	}
}
