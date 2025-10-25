package agentprotocol

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDeadLetterQueue_MoveToDeadLetter(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "dlq_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dlq := NewDeadLetterQueue(tmpDir)

	// Create test envelope
	env := &Envelope{
		MessageID:       "test-msg-001",
		ProtocolVersion: "1.0.0",
		FromAgent:       "agent-a",
		ToAgent:         "agent-b",
		MessageType:     "request",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		Retries:         3,
		Payload:         map[string]interface{}{"test": "data"},
	}

	// Move to DLQ
	dlqPath, err := dlq.MoveToDeadLetter(env, "max retries exceeded", "stack trace here")
	if err != nil {
		t.Fatalf("MoveToDeadLetter failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(dlqPath); os.IsNotExist(err) {
		t.Fatalf("DLQ file not created: %s", dlqPath)
	}

	// Verify file is in correct directory
	expectedDir := filepath.Join(tmpDir, "messages", "dead_letter")
	if filepath.Dir(dlqPath) != expectedDir {
		t.Errorf("DLQ file in wrong directory: got %s, want %s", filepath.Dir(dlqPath), expectedDir)
	}
}

func TestDeadLetterQueue_GetDeadLetterMessages(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "dlq_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dlq := NewDeadLetterQueue(tmpDir)

	// Initially empty
	messages, err := dlq.GetDeadLetterMessages()
	if err != nil {
		t.Fatalf("GetDeadLetterMessages failed: %v", err)
	}
	if len(messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(messages))
	}

	// Add messages to DLQ
	env1 := &Envelope{
		MessageID:       "dlq-msg-001",
		ProtocolVersion: "1.0.0",
		FromAgent:       "agent-a",
		ToAgent:         "agent-b",
		MessageType:     "request",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		Retries:         3,
		Payload:         map[string]interface{}{"test": "data1"},
	}

	env2 := &Envelope{
		MessageID:       "dlq-msg-002",
		ProtocolVersion: "1.0.0",
		FromAgent:       "agent-c",
		ToAgent:         "agent-d",
		MessageType:     "notification",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		Retries:         5,
		Payload:         map[string]interface{}{"test": "data2"},
	}

	if _, err := dlq.MoveToDeadLetter(env1, "error 1", "trace 1"); err != nil {
		t.Fatalf("MoveToDeadLetter failed: %v", err)
	}

	if _, err := dlq.MoveToDeadLetter(env2, "error 2", "trace 2"); err != nil {
		t.Fatalf("MoveToDeadLetter failed: %v", err)
	}

	// Retrieve messages
	messages, err = dlq.GetDeadLetterMessages()
	if err != nil {
		t.Fatalf("GetDeadLetterMessages failed: %v", err)
	}

	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}

	// Verify metadata
	for _, msg := range messages {
		if msg.FailureReason == "" {
			t.Error("expected failure reason to be set")
		}
		if msg.FailedAt.IsZero() {
			t.Error("expected failed_at timestamp to be set")
		}
		if msg.RetryCount == 0 {
			t.Error("expected retry count to be preserved")
		}
	}
}

func TestDeadLetterQueue_DeleteDeadLetterMessage(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "dlq_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dlq := NewDeadLetterQueue(tmpDir)

	// Add message to DLQ
	env := &Envelope{
		MessageID:       "dlq-msg-delete",
		ProtocolVersion: "1.0.0",
		FromAgent:       "agent-a",
		ToAgent:         "agent-b",
		MessageType:     "request",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		Retries:         3,
		Payload:         map[string]interface{}{"test": "data"},
	}

	dlqPath, err := dlq.MoveToDeadLetter(env, "test error", "test trace")
	if err != nil {
		t.Fatalf("MoveToDeadLetter failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(dlqPath); os.IsNotExist(err) {
		t.Fatalf("DLQ file not created")
	}

	// Delete message
	if err := dlq.DeleteDeadLetterMessage(env.MessageID); err != nil {
		t.Fatalf("DeleteDeadLetterMessage failed: %v", err)
	}

	// Verify file deleted
	if _, err := os.Stat(dlqPath); !os.IsNotExist(err) {
		t.Error("DLQ file should have been deleted")
	}

	// Verify DLQ is empty
	messages, err := dlq.GetDeadLetterMessages()
	if err != nil {
		t.Fatalf("GetDeadLetterMessages failed: %v", err)
	}
	if len(messages) != 0 {
		t.Errorf("expected 0 messages after delete, got %d", len(messages))
	}
}

func TestDeadLetterQueue_RetryFromDeadLetter(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "dlq_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dlq := NewDeadLetterQueue(tmpDir)

	// Add message to DLQ
	env := &Envelope{
		MessageID:       "dlq-msg-retry",
		ProtocolVersion: "1.0.0",
		FromAgent:       "agent-a",
		ToAgent:         "agent-b",
		MessageType:     "request",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		Retries:         3,
		Payload:         map[string]interface{}{"test": "retry data"},
	}

	dlqPath, err := dlq.MoveToDeadLetter(env, "temporary error", "trace")
	if err != nil {
		t.Fatalf("MoveToDeadLetter failed: %v", err)
	}

	// Retry message
	retried, err := dlq.RetryFromDeadLetter(env.MessageID)
	if err != nil {
		t.Fatalf("RetryFromDeadLetter failed: %v", err)
	}

	// Verify envelope is correct
	if retried.MessageID != env.MessageID {
		t.Errorf("expected message ID %s, got %s", env.MessageID, retried.MessageID)
	}

	// Verify retries counter is reset
	if retried.Retries != 0 {
		t.Errorf("expected retries to be reset to 0, got %d", retried.Retries)
	}

	// Verify file is removed from DLQ
	if _, err := os.Stat(dlqPath); !os.IsNotExist(err) {
		t.Error("DLQ file should have been removed after retry")
	}

	// Verify DLQ is empty
	messages, err := dlq.GetDeadLetterMessages()
	if err != nil {
		t.Fatalf("GetDeadLetterMessages failed: %v", err)
	}
	if len(messages) != 0 {
		t.Errorf("expected 0 messages after retry, got %d", len(messages))
	}
}

func TestDeadLetterQueue_EmptyQueue(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "dlq_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dlq := NewDeadLetterQueue(tmpDir)

	// Get messages from empty DLQ (directory doesn't exist yet)
	messages, err := dlq.GetDeadLetterMessages()
	if err != nil {
		t.Fatalf("GetDeadLetterMessages failed on empty queue: %v", err)
	}

	if messages != nil {
		t.Errorf("expected nil for empty queue, got %v", messages)
	}
}
