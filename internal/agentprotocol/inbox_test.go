package agentprotocol

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUserInbox_SendToUser(t *testing.T) {
	tempDir := t.TempDir()
	inbox := NewUserInbox(tempDir)

	// Create test message
	msg := &Envelope{
		ProtocolVersion: "1.0.0",
		SchemaVersion:   "1.0.0",
		MessageID:       "msg-test-001",
		CorrelationID:   "corr-001",
		TraceID:         "trace-001",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		TTLSeconds:      300,
		FromAgent:       "test-agent",
		ToAgent:         "user",
		MessageType:     "notification",
		PayloadSchema:   "test.v1",
		Payload: map[string]interface{}{
			"message": "Test notification",
		},
	}

	// Send to user
	msgPath, err := inbox.SendToUser(msg)
	if err != nil {
		t.Fatalf("SendToUser failed: %v", err)
	}

	// Verify message was written to _unread folder
	expectedDir := filepath.Join(tempDir, "messages", "inbox", "user", "_unread")
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Error("_unread directory not created")
	}

	// Verify message file exists
	if _, err := os.Stat(msgPath); os.IsNotExist(err) {
		t.Errorf("Message file not created at %s", msgPath)
	}
}

func TestUserInbox_GetUnreadMessages(t *testing.T) {
	tempDir := t.TempDir()
	inbox := NewUserInbox(tempDir)

	// Send multiple messages
	for i := 0; i < 3; i++ {
		msg := &Envelope{
			ProtocolVersion: "1.0.0",
			SchemaVersion:   "1.0.0",
			MessageID:       GenerateMessageID(),
			CorrelationID:   "corr-001",
			TraceID:         "trace-001",
			Timestamp:       time.Now().UTC().Format(time.RFC3339),
			TTLSeconds:      300,
			FromAgent:       "test-agent",
			ToAgent:         "user",
			MessageType:     "notification",
			PayloadSchema:   "test.v1",
			Payload:         map[string]interface{}{},
		}

		if _, err := inbox.SendToUser(msg); err != nil {
			t.Fatalf("SendToUser failed: %v", err)
		}
	}

	// Get unread messages
	messages, err := inbox.GetUnreadMessages()
	if err != nil {
		t.Fatalf("GetUnreadMessages failed: %v", err)
	}

	if len(messages) != 3 {
		t.Errorf("Expected 3 unread messages, got %d", len(messages))
	}
}

func TestUserInbox_MarkAsRead(t *testing.T) {
	tempDir := t.TempDir()
	inbox := NewUserInbox(tempDir)

	// Send message
	msg := &Envelope{
		ProtocolVersion: "1.0.0",
		SchemaVersion:   "1.0.0",
		MessageID:       "msg-test-read",
		CorrelationID:   "corr-001",
		TraceID:         "trace-001",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		TTLSeconds:      300,
		FromAgent:       "test-agent",
		ToAgent:         "user",
		MessageType:     "notification",
		PayloadSchema:   "test.v1",
		Payload:         map[string]interface{}{},
	}

	if _, err := inbox.SendToUser(msg); err != nil {
		t.Fatalf("SendToUser failed: %v", err)
	}

	// Verify message is unread
	unread, err := inbox.GetUnreadMessages()
	if err != nil {
		t.Fatalf("GetUnreadMessages failed: %v", err)
	}
	if len(unread) != 1 {
		t.Errorf("Expected 1 unread message, got %d", len(unread))
	}

	// Mark as read
	if err := inbox.MarkAsRead(msg.MessageID); err != nil {
		t.Fatalf("MarkAsRead failed: %v", err)
	}

	// Verify message is no longer unread
	unread, err = inbox.GetUnreadMessages()
	if err != nil {
		t.Fatalf("GetUnreadMessages failed: %v", err)
	}
	if len(unread) != 0 {
		t.Errorf("Expected 0 unread messages after marking as read, got %d", len(unread))
	}

	// Verify message is in read folder
	read, err := inbox.GetReadMessages()
	if err != nil {
		t.Fatalf("GetReadMessages failed: %v", err)
	}
	if len(read) != 1 {
		t.Errorf("Expected 1 read message, got %d", len(read))
	}
}

func TestUserInbox_MarkAsArchived(t *testing.T) {
	tempDir := t.TempDir()
	inbox := NewUserInbox(tempDir)

	// Send and read a message
	msg := &Envelope{
		ProtocolVersion: "1.0.0",
		SchemaVersion:   "1.0.0",
		MessageID:       "msg-test-archive",
		CorrelationID:   "corr-001",
		TraceID:         "trace-001",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		TTLSeconds:      300,
		FromAgent:       "test-agent",
		ToAgent:         "user",
		MessageType:     "notification",
		PayloadSchema:   "test.v1",
		Payload:         map[string]interface{}{},
	}

	if _, err := inbox.SendToUser(msg); err != nil {
		t.Fatalf("SendToUser failed: %v", err)
	}

	if err := inbox.MarkAsRead(msg.MessageID); err != nil {
		t.Fatalf("MarkAsRead failed: %v", err)
	}

	// Archive the message
	if err := inbox.MarkAsArchived(msg.MessageID); err != nil {
		t.Fatalf("MarkAsArchived failed: %v", err)
	}

	// Verify message is no longer in read folder
	read, err := inbox.GetReadMessages()
	if err != nil {
		t.Fatalf("GetReadMessages failed: %v", err)
	}
	if len(read) != 0 {
		t.Errorf("Expected 0 read messages after archiving, got %d", len(read))
	}

	// Verify message is in archive folder
	archived, err := inbox.GetArchivedMessages()
	if err != nil {
		t.Fatalf("GetArchivedMessages failed: %v", err)
	}
	if len(archived) != 1 {
		t.Errorf("Expected 1 archived message, got %d", len(archived))
	}
}

func TestUserInbox_ArchiveFromUnread(t *testing.T) {
	tempDir := t.TempDir()
	inbox := NewUserInbox(tempDir)

	// Send message
	msg := &Envelope{
		ProtocolVersion: "1.0.0",
		SchemaVersion:   "1.0.0",
		MessageID:       "msg-test-archive-unread",
		CorrelationID:   "corr-001",
		TraceID:         "trace-001",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		TTLSeconds:      300,
		FromAgent:       "test-agent",
		ToAgent:         "user",
		MessageType:     "notification",
		PayloadSchema:   "test.v1",
		Payload:         map[string]interface{}{},
	}

	if _, err := inbox.SendToUser(msg); err != nil {
		t.Fatalf("SendToUser failed: %v", err)
	}

	// Archive directly from unread (skip read step)
	if err := inbox.MarkAsArchived(msg.MessageID); err != nil {
		t.Fatalf("MarkAsArchived failed: %v", err)
	}

	// Verify message is not unread
	unread, err := inbox.GetUnreadMessages()
	if err != nil {
		t.Fatalf("GetUnreadMessages failed: %v", err)
	}
	if len(unread) != 0 {
		t.Errorf("Expected 0 unread messages, got %d", len(unread))
	}

	// Verify message is in archive
	archived, err := inbox.GetArchivedMessages()
	if err != nil {
		t.Fatalf("GetArchivedMessages failed: %v", err)
	}
	if len(archived) != 1 {
		t.Errorf("Expected 1 archived message, got %d", len(archived))
	}
}

func TestUserInbox_DeleteMessage(t *testing.T) {
	tempDir := t.TempDir()
	inbox := NewUserInbox(tempDir)

	// Send message
	msg := &Envelope{
		ProtocolVersion: "1.0.0",
		SchemaVersion:   "1.0.0",
		MessageID:       "msg-test-delete",
		CorrelationID:   "corr-001",
		TraceID:         "trace-001",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		TTLSeconds:      300,
		FromAgent:       "test-agent",
		ToAgent:         "user",
		MessageType:     "notification",
		PayloadSchema:   "test.v1",
		Payload:         map[string]interface{}{},
	}

	if _, err := inbox.SendToUser(msg); err != nil {
		t.Fatalf("SendToUser failed: %v", err)
	}

	// Delete message
	if err := inbox.DeleteMessage(msg.MessageID); err != nil {
		t.Fatalf("DeleteMessage failed: %v", err)
	}

	// Verify message is gone
	unread, err := inbox.GetUnreadMessages()
	if err != nil {
		t.Fatalf("GetUnreadMessages failed: %v", err)
	}
	if len(unread) != 0 {
		t.Errorf("Expected 0 messages after deletion, got %d", len(unread))
	}

	// Try to delete again (should fail)
	err = inbox.DeleteMessage(msg.MessageID)
	if err == nil {
		t.Error("Expected error when deleting non-existent message")
	}
}

func TestUserInbox_EmptyFolders(t *testing.T) {
	tempDir := t.TempDir()
	inbox := NewUserInbox(tempDir)

	// Get messages from empty inbox
	unread, err := inbox.GetUnreadMessages()
	if err != nil {
		t.Fatalf("GetUnreadMessages failed: %v", err)
	}
	if len(unread) != 0 {
		t.Errorf("Expected 0 unread messages, got %d", len(unread))
	}

	read, err := inbox.GetReadMessages()
	if err != nil {
		t.Fatalf("GetReadMessages failed: %v", err)
	}
	if len(read) != 0 {
		t.Errorf("Expected 0 read messages, got %d", len(read))
	}

	archived, err := inbox.GetArchivedMessages()
	if err != nil {
		t.Fatalf("GetArchivedMessages failed: %v", err)
	}
	if len(archived) != 0 {
		t.Errorf("Expected 0 archived messages, got %d", len(archived))
	}
}

func TestUserInbox_MultipleMessages(t *testing.T) {
	tempDir := t.TempDir()
	inbox := NewUserInbox(tempDir)

	// Send 10 messages
	messageIDs := make([]string, 10)
	for i := 0; i < 10; i++ {
		msg := &Envelope{
			ProtocolVersion: "1.0.0",
			SchemaVersion:   "1.0.0",
			MessageID:       GenerateMessageID(),
			CorrelationID:   "corr-001",
			TraceID:         "trace-001",
			Timestamp:       time.Now().UTC().Format(time.RFC3339),
			TTLSeconds:      300,
			FromAgent:       "test-agent",
			ToAgent:         "user",
			MessageType:     "notification",
			PayloadSchema:   "test.v1",
			Payload:         map[string]interface{}{},
		}

		if _, err := inbox.SendToUser(msg); err != nil {
			t.Fatalf("SendToUser failed: %v", err)
		}

		messageIDs[i] = msg.MessageID
		time.Sleep(1 * time.Millisecond) // Ensure unique timestamps
	}

	// Mark first 5 as read
	for i := 0; i < 5; i++ {
		if err := inbox.MarkAsRead(messageIDs[i]); err != nil {
			t.Fatalf("MarkAsRead failed: %v", err)
		}
	}

	// Archive first 3 of the read messages
	for i := 0; i < 3; i++ {
		if err := inbox.MarkAsArchived(messageIDs[i]); err != nil {
			t.Fatalf("MarkAsArchived failed: %v", err)
		}
	}

	// Verify counts
	unread, err := inbox.GetUnreadMessages()
	if err != nil {
		t.Fatalf("GetUnreadMessages failed: %v", err)
	}
	if len(unread) != 5 {
		t.Errorf("Expected 5 unread messages, got %d", len(unread))
	}

	read, err := inbox.GetReadMessages()
	if err != nil {
		t.Fatalf("GetReadMessages failed: %v", err)
	}
	if len(read) != 2 {
		t.Errorf("Expected 2 read messages, got %d", len(read))
	}

	archived, err := inbox.GetArchivedMessages()
	if err != nil {
		t.Fatalf("GetArchivedMessages failed: %v", err)
	}
	if len(archived) != 3 {
		t.Errorf("Expected 3 archived messages, got %d", len(archived))
	}
}
