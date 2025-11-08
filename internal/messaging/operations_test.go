package messaging

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestOpenStore tests opening a new store
func TestOpenStore(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("OpenStore failed: %v", err)
	}
	defer store.Close()

	// Verify database file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}
}

// TestGetMessages tests retrieving messages
func TestGetMessages(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db)

	// Create test data
	createTestThread(t, db, "thread1", "human", "user1")
	createTestMessage(t, db, "msg1", "thread1", 1, "ailang_instance", "agent1", "human", "user1", "directive", "Test message 1", "pending")
	createTestMessage(t, db, "msg2", "thread1", 2, "ailang_instance", "agent1", "human", "user1", "result", "Test message 2", "acked")

	// Get all messages for user1
	messages, err := store.GetMessages("human", "user1", "")
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}

	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}

	// Get only pending messages
	messages, err = store.GetMessages("human", "user1", "pending")
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}

	if len(messages) != 1 {
		t.Errorf("Expected 1 pending message, got %d", len(messages))
	}

	if messages[0].ID != "msg1" {
		t.Errorf("Expected msg1, got %s", messages[0].ID)
	}
}

// TestMarkAsAcked tests marking a message as acknowledged
func TestMarkAsAcked(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db)

	// Create test data
	createTestThread(t, db, "thread1", "human", "user1")
	createTestMessage(t, db, "msg1", "thread1", 1, "ailang_instance", "agent1", "human", "user1", "directive", "Test", "pending")

	// Mark as acked
	err := store.MarkAsAcked("msg1")
	if err != nil {
		t.Fatalf("MarkAsAcked failed: %v", err)
	}

	// Verify state changed
	var state string
	err = db.QueryRow("SELECT delivery_state FROM messages WHERE id = 'msg1'").Scan(&state)
	if err != nil {
		t.Fatalf("Failed to query message: %v", err)
	}

	if state != "acked" {
		t.Errorf("Expected delivery_state 'acked', got %s", state)
	}
}

// TestMarkAsAckedNonExistent tests marking a non-existent message
func TestMarkAsAckedNonExistent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db)

	// Try to mark non-existent message
	err := store.MarkAsAcked("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent message, got nil")
	}
}

// TestMarkAsUnacked tests marking a message as unacknowledged
func TestMarkAsUnacked(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db)

	// Create test data
	createTestThread(t, db, "thread1", "human", "user1")
	createTestMessage(t, db, "msg1", "thread1", 1, "ailang_instance", "agent1", "human", "user1", "directive", "Test", "acked")

	// Mark as unacked
	err := store.MarkAsUnacked("msg1")
	if err != nil {
		t.Fatalf("MarkAsUnacked failed: %v", err)
	}

	// Verify state changed
	var state string
	err = db.QueryRow("SELECT delivery_state FROM messages WHERE id = 'msg1'").Scan(&state)
	if err != nil {
		t.Fatalf("Failed to query message: %v", err)
	}

	if state != "pending" {
		t.Errorf("Expected delivery_state 'pending', got %s", state)
	}
}

// TestMarkAllAsAcked tests marking all messages as acknowledged
func TestMarkAllAsAcked(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db)

	// Create test data
	createTestThread(t, db, "thread1", "human", "user1")
	createTestMessage(t, db, "msg1", "thread1", 1, "ailang_instance", "agent1", "human", "user1", "directive", "Test 1", "pending")
	createTestMessage(t, db, "msg2", "thread1", 2, "ailang_instance", "agent1", "human", "user1", "directive", "Test 2", "visible")
	createTestMessage(t, db, "msg3", "thread1", 3, "ailang_instance", "agent1", "human", "user1", "directive", "Test 3", "acked")

	// Mark all as acked
	count, err := store.MarkAllAsAcked("human", "user1")
	if err != nil {
		t.Fatalf("MarkAllAsAcked failed: %v", err)
	}

	// Should only update msg1 and msg2 (pending and visible)
	if count != 2 {
		t.Errorf("Expected 2 messages updated, got %d", count)
	}

	// Verify all are now acked
	messages, err := store.GetMessages("human", "user1", "")
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}

	for _, msg := range messages {
		if msg.DeliveryState != "acked" {
			t.Errorf("Message %s has delivery_state %s, expected 'acked'", msg.ID, msg.DeliveryState)
		}
	}
}

// TestDatabaseExists tests checking if database exists
func TestDatabaseExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Non-existent database
	nonExistentPath := filepath.Join(tmpDir, "nonexistent.db")
	if DatabaseExists(nonExistentPath) {
		t.Error("DatabaseExists returned true for non-existent database")
	}

	// Create database
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	db.Close()

	// Should now exist
	if !DatabaseExists(dbPath) {
		t.Error("DatabaseExists returned false for existing database")
	}
}

// TestMigrateIfNeeded tests automatic migration
func TestMigrateIfNeeded(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	messagesDir := filepath.Join(tmpDir, "messages")

	// Create test messages
	env1 := createTestEnvelopeWithTimestamp("msg1", "agent1", "agent2", "corr1", "2025-01-01T12:00:00Z")
	writeEnvelopeToFile(t, messagesDir, "agent2", "msg1.json", env1)

	// First call should migrate
	migrated, err := MigrateIfNeeded(dbPath, messagesDir)
	if err != nil {
		t.Fatalf("MigrateIfNeeded failed: %v", err)
	}

	if !migrated {
		t.Error("Expected migration to occur, but it didn't")
	}

	// Second call should not migrate (database exists)
	migrated, err = MigrateIfNeeded(dbPath, messagesDir)
	if err != nil {
		t.Fatalf("MigrateIfNeeded failed on second call: %v", err)
	}

	if migrated {
		t.Error("Expected no migration on second call, but it did migrate")
	}
}

// TestMigrateIfNeededEmptyMessages tests migration with no messages
func TestMigrateIfNeededEmptyMessages(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	messagesDir := filepath.Join(tmpDir, "nonexistent")

	// Should create empty database
	migrated, err := MigrateIfNeeded(dbPath, messagesDir)
	if err != nil {
		t.Fatalf("MigrateIfNeeded failed: %v", err)
	}

	if migrated {
		t.Error("Expected no migration (no messages), but it did migrate")
	}

	// Database should exist
	if !DatabaseExists(dbPath) {
		t.Error("Database was not created")
	}
}

// TestConvertToAgentProtocolEnvelope tests conversion to legacy format
func TestConvertToAgentProtocolEnvelope(t *testing.T) {
	msg := Message{
		ID:        "msg1",
		ThreadID:  "thread1",
		CreatedAt: time.Now(),
		FromType:  "ailang_instance",
		FromID:    "agent1",
		ToType:    "human",
		ToID:      "user1",
		Kind:      "directive",
		Content:   `{"task": "do something"}`,
	}

	env, err := ConvertToAgentProtocolEnvelope(msg)
	if err != nil {
		t.Fatalf("ConvertToAgentProtocolEnvelope failed: %v", err)
	}

	if env.MessageID != "msg1" {
		t.Errorf("Expected MessageID 'msg1', got %s", env.MessageID)
	}

	if env.FromAgent != "agent1" {
		t.Errorf("Expected FromAgent 'agent1', got %s", env.FromAgent)
	}

	if env.ToAgent != "user1" {
		t.Errorf("Expected ToAgent 'user1', got %s", env.ToAgent)
	}

	if env.MessageType != "request" {
		t.Errorf("Expected MessageType 'request', got %s", env.MessageType)
	}

	if task, ok := env.Payload["task"].(string); !ok || task != "do something" {
		t.Errorf("Expected payload task 'do something', got %v", env.Payload["task"])
	}
}

// Helper functions

func createTestThread(t *testing.T, db *sql.DB, id, createdByType, createdByID string) {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO threads (id, title, created_at, created_by_type, created_by_id, status, last_seq, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, id, "Test thread", time.Now().UnixMilli(), createdByType, createdByID, "active", 0, time.Now().UnixMilli())

	if err != nil {
		t.Fatalf("Failed to create test thread: %v", err)
	}
}

func createTestMessage(t *testing.T, db *sql.DB, id, threadID string, seq int, fromType, fromID, toType, toID, kind, content, deliveryState string) {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO messages (
			id, thread_id, message_seq, created_at,
			from_type, from_id, to_type, to_id,
			kind, content, delivery_state, business_state
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, threadID, seq, time.Now().UnixMilli(),
		fromType, fromID, toType, toID,
		kind, content, deliveryState, "open")

	if err != nil {
		t.Fatalf("Failed to create test message: %v", err)
	}
}

// TestCreateThread tests thread creation
func TestCreateThread(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db)

	thread, err := store.CreateThread("Test Thread", "human", "user1")
	if err != nil {
		t.Fatalf("CreateThread failed: %v", err)
	}

	if thread.ID == "" {
		t.Error("Thread ID is empty")
	}

	if thread.Title != "Test Thread" {
		t.Errorf("Expected title 'Test Thread', got %s", thread.Title)
	}

	if thread.CreatedByType != "human" {
		t.Errorf("Expected created_by_type 'human', got %s", thread.CreatedByType)
	}

	if thread.LastSeq != 0 {
		t.Errorf("Expected initial last_seq 0, got %d", thread.LastSeq)
	}
}

// TestGetThread tests thread retrieval
func TestGetThread(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db)

	// Create thread
	created, err := store.CreateThread("Test Thread", "human", "user1")
	if err != nil {
		t.Fatalf("CreateThread failed: %v", err)
	}

	// Get thread
	retrieved, err := store.GetThread(created.ID)
	if err != nil {
		t.Fatalf("GetThread failed: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("Expected ID %s, got %s", created.ID, retrieved.ID)
	}

	if retrieved.Title != created.Title {
		t.Errorf("Expected title %s, got %s", created.Title, retrieved.Title)
	}
}

// TestGetThreadNonExistent tests retrieving a non-existent thread
func TestGetThreadNonExistent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db)

	_, err := store.GetThread("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent thread, got nil")
	}
}

// TestCreateMessage tests message creation with atomic seq allocation
func TestCreateMessage(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db)

	// Create thread first
	thread, err := store.CreateThread("Test Thread", "human", "user1")
	if err != nil {
		t.Fatalf("CreateThread failed: %v", err)
	}

	// Create message
	msg, err := store.CreateMessage(
		thread.ID,
		"ailang_instance", "agent1",
		"human", "user1",
		"directive", `{"task": "test"}`,
	)

	if err != nil {
		t.Fatalf("CreateMessage failed: %v", err)
	}

	if msg.ID == "" {
		t.Error("Message ID is empty")
	}

	if msg.MessageSeq != 1 {
		t.Errorf("Expected message_seq 1, got %d", msg.MessageSeq)
	}

	if msg.Kind != "directive" {
		t.Errorf("Expected kind 'directive', got %s", msg.Kind)
	}

	// Verify thread's last_seq was updated
	updatedThread, err := store.GetThread(thread.ID)
	if err != nil {
		t.Fatalf("GetThread failed: %v", err)
	}

	if updatedThread.LastSeq != 1 {
		t.Errorf("Expected thread last_seq 1, got %d", updatedThread.LastSeq)
	}
}

// TestCreateMessageSequencing tests that message_seq is monotonic
func TestCreateMessageSequencing(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db)

	// Create thread
	thread, err := store.CreateThread("Test Thread", "human", "user1")
	if err != nil {
		t.Fatalf("CreateThread failed: %v", err)
	}

	// Create 5 messages
	for i := 1; i <= 5; i++ {
		msg, err := store.CreateMessage(
			thread.ID,
			"ailang_instance", "agent1",
			"human", "user1",
			"directive", fmt.Sprintf("message %d", i),
		)

		if err != nil {
			t.Fatalf("CreateMessage %d failed: %v", i, err)
		}

		if msg.MessageSeq != i {
			t.Errorf("Expected message_seq %d, got %d", i, msg.MessageSeq)
		}
	}

	// Verify thread's last_seq
	updatedThread, err := store.GetThread(thread.ID)
	if err != nil {
		t.Fatalf("GetThread failed: %v", err)
	}

	if updatedThread.LastSeq != 5 {
		t.Errorf("Expected thread last_seq 5, got %d", updatedThread.LastSeq)
	}
}

// TestGetMessagesFromSeq tests cursor-based message retrieval
func TestGetMessagesFromSeq(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db)

	// Create thread
	thread, err := store.CreateThread("Test Thread", "human", "user1")
	if err != nil {
		t.Fatalf("CreateThread failed: %v", err)
	}

	// Create 10 messages
	for i := 1; i <= 10; i++ {
		_, err := store.CreateMessage(
			thread.ID,
			"ailang_instance", "agent1",
			"human", "user1",
			"directive", fmt.Sprintf("message %d", i),
		)
		if err != nil {
			t.Fatalf("CreateMessage %d failed: %v", i, err)
		}
	}

	// Get messages from seq 5 onwards
	messages, err := store.GetMessagesFromSeq(thread.ID, 5, 0)
	if err != nil {
		t.Fatalf("GetMessagesFromSeq failed: %v", err)
	}

	// Should get messages 6-10 (5 messages)
	if len(messages) != 5 {
		t.Errorf("Expected 5 messages, got %d", len(messages))
	}

	if messages[0].MessageSeq != 6 {
		t.Errorf("Expected first message seq 6, got %d", messages[0].MessageSeq)
	}

	if messages[4].MessageSeq != 10 {
		t.Errorf("Expected last message seq 10, got %d", messages[4].MessageSeq)
	}
}

// TestGetMessagesFromSeqWithLimit tests cursor-based retrieval with limit
func TestGetMessagesFromSeqWithLimit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db)

	// Create thread
	thread, err := store.CreateThread("Test Thread", "human", "user1")
	if err != nil {
		t.Fatalf("CreateThread failed: %v", err)
	}

	// Create 10 messages
	for i := 1; i <= 10; i++ {
		_, err := store.CreateMessage(
			thread.ID,
			"ailang_instance", "agent1",
			"human", "user1",
			"directive", fmt.Sprintf("message %d", i),
		)
		if err != nil {
			t.Fatalf("CreateMessage %d failed: %v", i, err)
		}
	}

	// Get messages from seq 0 with limit 3
	messages, err := store.GetMessagesFromSeq(thread.ID, 0, 3)
	if err != nil {
		t.Fatalf("GetMessagesFromSeq failed: %v", err)
	}

	// Should get messages 1-3
	if len(messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}

	if messages[0].MessageSeq != 1 {
		t.Errorf("Expected first message seq 1, got %d", messages[0].MessageSeq)
	}
}

// TestSubscribe tests subscription creation
func TestSubscribe(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db)

	// Create thread
	thread, err := store.CreateThread("Test Thread", "human", "user1")
	if err != nil {
		t.Fatalf("CreateThread failed: %v", err)
	}

	// Subscribe
	err = store.Subscribe("instance1", thread.ID)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Verify subscription exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM subscriptions WHERE instance_id = ? AND thread_id = ?",
		"instance1", thread.ID).Scan(&count)

	if err != nil {
		t.Fatalf("Failed to query subscription: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 subscription, got %d", count)
	}
}

// TestSubscribeDuplicate tests that subscribing twice doesn't create duplicates
func TestSubscribeDuplicate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db)

	// Create thread
	thread, err := store.CreateThread("Test Thread", "human", "user1")
	if err != nil {
		t.Fatalf("CreateThread failed: %v", err)
	}

	// Subscribe twice
	err = store.Subscribe("instance1", thread.ID)
	if err != nil {
		t.Fatalf("First Subscribe failed: %v", err)
	}

	err = store.Subscribe("instance1", thread.ID)
	if err != nil {
		t.Fatalf("Second Subscribe failed: %v", err)
	}

	// Should still have only 1 subscription
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM subscriptions WHERE instance_id = ? AND thread_id = ?",
		"instance1", thread.ID).Scan(&count)

	if err != nil {
		t.Fatalf("Failed to query subscription: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 subscription, got %d", count)
	}
}

// TestUpdateAckSeq tests updating acknowledgement sequence
func TestUpdateAckSeq(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db)

	// Create thread and subscribe
	thread, err := store.CreateThread("Test Thread", "human", "user1")
	if err != nil {
		t.Fatalf("CreateThread failed: %v", err)
	}

	err = store.Subscribe("instance1", thread.ID)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Update ack seq
	err = store.UpdateAckSeq("instance1", thread.ID, 42)
	if err != nil {
		t.Fatalf("UpdateAckSeq failed: %v", err)
	}

	// Verify ack seq was updated
	var ackSeq int
	err = db.QueryRow("SELECT last_ack_seq FROM subscriptions WHERE instance_id = ? AND thread_id = ?",
		"instance1", thread.ID).Scan(&ackSeq)

	if err != nil {
		t.Fatalf("Failed to query ack seq: %v", err)
	}

	if ackSeq != 42 {
		t.Errorf("Expected ack_seq 42, got %d", ackSeq)
	}
}

// TestUpdateAckSeqNonExistent tests updating a non-existent subscription
func TestUpdateAckSeqNonExistent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db)

	err := store.UpdateAckSeq("nonexistent", "nonexistent", 42)
	if err == nil {
		t.Error("Expected error for non-existent subscription, got nil")
	}
}
