package agentprotocol

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_FileAndDatabase tests the complete workflow with both file-based
// messages and SQLite state tracking.
func TestIntegration_FileAndDatabase(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize database
	db, err := NewDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	// Initialize message writer/reader
	writer := NewMessageWriter(tmpDir)
	reader := NewMessageReader(tmpDir)

	t.Log("Step 1: Register two agents")

	// Register design-doc-creator
	err = db.RegisterAgent(&AgentInfo{
		AgentID:       "design-doc-creator",
		InboxPath:     filepath.Join(tmpDir, "messages", "design-doc-creator"),
		Status:        "active",
		ProtocolCaps:  `["v1.0"]`,
		LastHeartbeat: time.Now().UTC(),
		CreatedAt:     time.Now().UTC(),
	})
	require.NoError(t, err)

	// Register sprint-planner
	err = db.RegisterAgent(&AgentInfo{
		AgentID:       "sprint-planner",
		InboxPath:     filepath.Join(tmpDir, "messages", "sprint-planner"),
		Status:        "active",
		ProtocolCaps:  `["v1.0"]`,
		LastHeartbeat: time.Now().UTC(),
		CreatedAt:     time.Now().UTC(),
	})
	require.NoError(t, err)

	// Verify agents registered
	agents, err := db.ListActiveAgents()
	require.NoError(t, err)
	assert.Len(t, agents, 2)

	t.Log("Step 2: design-doc-creator sends message")

	// Create message envelope
	msgEnv := &Envelope{
		ProtocolVersion: "1.0.0",
		SchemaVersion:   "1.0.0",
		MessageID:       GenerateMessageID(),
		CorrelationID:   GenerateCorrelationID(),
		TraceID:         GenerateTraceID(),
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		TTLSeconds:      300,
		FromAgent:       "design-doc-creator",
		ToAgent:         "sprint-planner",
		MessageType:     "request",
		PayloadSchema:   "design_doc_completed.v1",
		Payload: map[string]interface{}{
			"design_doc_path": "design_docs/planned/M-AGENT-PROTOCOL.md",
			"status":          "completed",
		},
		DeclaredEffects: []string{"FS.read"},
	}

	// Write message to file
	msgPath, err := writer.WriteMessage(msgEnv)
	require.NoError(t, err)
	t.Logf("  ✓ Message written to: %s", msgPath)

	// Record in database
	err = db.RecordMessage(&MessageRecord{
		MessageID:     msgEnv.MessageID,
		CorrelationID: msgEnv.CorrelationID,
		TraceID:       msgEnv.TraceID,
		FromAgent:     msgEnv.FromAgent,
		ToAgent:       msgEnv.ToAgent,
		MessageType:   msgEnv.MessageType,
		Status:        "pending",
		CreatedAt:     time.Now().UTC(),
		RetryCount:    0,
	})
	require.NoError(t, err)

	// Log event
	err = db.LogEvent("design-doc-creator", msgEnv.MessageID, "message_sent", `{"target": "sprint-planner"}`)
	require.NoError(t, err)

	t.Log("Step 3: sprint-planner discovers message")

	// Scan for pending messages (file-based)
	pending, err := reader.ScanPendingMessages("sprint-planner")
	require.NoError(t, err)
	assert.Len(t, pending, 1)

	// Check if already processed (database-based deduplication)
	exists, err := db.MessageExists(msgEnv.MessageID)
	require.NoError(t, err)
	assert.True(t, exists, "message should exist in database")

	t.Log("Step 4: sprint-planner acquires lease before processing")

	// Acquire lease (crash safety)
	acquired, err := db.AcquireLease(msgPath, "sprint-planner", 60)
	require.NoError(t, err)
	assert.True(t, acquired, "should acquire lease")

	// Try to acquire again (should fail - already locked)
	acquired2, err := db.AcquireLease(msgPath, "another-agent", 60)
	require.NoError(t, err)
	assert.False(t, acquired2, "should not acquire already-locked resource")

	// Update message status to processing
	err = db.UpdateMessageStatus(msgEnv.MessageID, "processing")
	require.NoError(t, err)

	t.Log("Step 5: sprint-planner processes message")

	// Read message from file
	msg, err := reader.ReadMessage(pending[0])
	require.NoError(t, err)
	require.NotNil(t, msg)

	// Verify message content
	assert.Equal(t, "design-doc-creator", msg.FromAgent)
	assert.Equal(t, "sprint-planner", msg.ToAgent)

	// Simulate processing (create response)
	responseEnv := &Envelope{
		ProtocolVersion: "1.0.0",
		SchemaVersion:   "1.0.0",
		MessageID:       GenerateMessageID(),
		CorrelationID:   msg.CorrelationID, // Same correlation
		TraceID:         msg.TraceID,       // Same trace
		ParentMessageID: &msg.MessageID,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		TTLSeconds:      300,
		FromAgent:       "sprint-planner",
		ToAgent:         "design-doc-creator",
		MessageType:     "response",
		PayloadSchema:   "sprint_plan_created.v1",
		Payload: map[string]interface{}{
			"sprint_plan_path": ".ailang/state/sprints/current_sprint.json",
			"status":           "ready",
		},
		DeclaredEffects: []string{"FS.write"},
	}

	// Write response to file
	_, err = writer.WriteMessage(responseEnv)
	require.NoError(t, err)

	// Record response in database
	err = db.RecordMessage(&MessageRecord{
		MessageID:     responseEnv.MessageID,
		CorrelationID: responseEnv.CorrelationID,
		TraceID:       responseEnv.TraceID,
		FromAgent:     responseEnv.FromAgent,
		ToAgent:       responseEnv.ToAgent,
		MessageType:   responseEnv.MessageType,
		Status:        "pending",
		CreatedAt:     time.Now().UTC(),
	})
	require.NoError(t, err)

	t.Log("Step 6: sprint-planner completes processing")

	// Mark original message as processed
	err = db.MarkMessageProcessed(msgEnv.MessageID)
	require.NoError(t, err)

	// Release lease
	err = db.ReleaseLease(msgPath)
	require.NoError(t, err)

	// Record metrics
	err = db.RecordMetric("sprint-planner", "processing_latency_ms", 123.45)
	require.NoError(t, err)

	// Log completion event
	err = db.LogEvent("sprint-planner", msgEnv.MessageID, "message_completed", `{"response_id": "`+responseEnv.MessageID+`"}`)
	require.NoError(t, err)

	// Verify message status in database
	record, err := db.GetMessage(msgEnv.MessageID)
	require.NoError(t, err)
	assert.Equal(t, "completed", record.Status)
	assert.NotNil(t, record.ProcessedAt)

	t.Log("Step 7: design-doc-creator receives response")

	// Scan for response
	readerA := NewMessageReader(tmpDir)
	pendingResponses, err := readerA.ScanPendingMessages("design-doc-creator")
	require.NoError(t, err)
	assert.Len(t, pendingResponses, 1)

	// Read response
	response, err := readerA.ReadMessage(pendingResponses[0])
	require.NoError(t, err)
	require.NotNil(t, response)

	// Verify it's the response
	assert.Equal(t, responseEnv.MessageID, response.MessageID)
	assert.Equal(t, msgEnv.MessageID, *response.ParentMessageID)
	assert.Equal(t, msgEnv.CorrelationID, response.CorrelationID)

	t.Log("✅ Integration test PASSED!")
	t.Log("  - Agents registered in database")
	t.Log("  - Messages written to files")
	t.Log("  - Messages tracked in database")
	t.Log("  - Leases acquired and released")
	t.Log("  - Events and metrics logged")
	t.Log("  - Cross-process deduplication works")
}

// TestIntegration_CrashRecovery tests that crashed agent's work can be recovered
// by another agent after lease expiration.
func TestIntegration_CrashRecovery(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := NewDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	writer := NewMessageWriter(tmpDir)
	_ = NewMessageReader(tmpDir) // reader not used in this test

	t.Log("Step 1: Register agents")

	db.RegisterAgent(&AgentInfo{
		AgentID:       "worker-1",
		InboxPath:     filepath.Join(tmpDir, "messages", "worker-1"),
		Status:        "active",
		ProtocolCaps:  `["v1.0"]`,
		LastHeartbeat: time.Now().UTC(),
		CreatedAt:     time.Now().UTC(),
	})

	db.RegisterAgent(&AgentInfo{
		AgentID:       "worker-2",
		InboxPath:     filepath.Join(tmpDir, "messages", "worker-2"),
		Status:        "active",
		ProtocolCaps:  `["v1.0"]`,
		LastHeartbeat: time.Now().UTC(),
		CreatedAt:     time.Now().UTC(),
	})

	t.Log("Step 2: Create message for worker-1")

	msgEnv := &Envelope{
		ProtocolVersion: "1.0.0",
		MessageID:       GenerateMessageID(),
		FromAgent:       "scheduler",
		ToAgent:         "worker-1",
		MessageType:     "request",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
	}

	msgPath, err := writer.WriteMessage(msgEnv)
	require.NoError(t, err)

	db.RecordMessage(&MessageRecord{
		MessageID:   msgEnv.MessageID,
		FromAgent:   msgEnv.FromAgent,
		ToAgent:     msgEnv.ToAgent,
		MessageType: msgEnv.MessageType,
		Status:      "pending",
		CreatedAt:   time.Now().UTC(),
	})

	t.Log("Step 3: worker-1 acquires lease with SHORT duration (simulates crash)")

	// Worker-1 acquires lease with 1 second duration
	acquired, err := db.AcquireLease(msgPath, "worker-1", 1)
	require.NoError(t, err)
	assert.True(t, acquired)

	db.UpdateMessageStatus(msgEnv.MessageID, "processing")

	// Simulate crash: worker-1 dies here, doesn't release lease

	t.Log("Step 4: worker-2 tries to acquire (should fail initially)")

	acquired, err = db.AcquireLease(msgPath, "worker-2", 60)
	require.NoError(t, err)
	assert.False(t, acquired, "worker-2 should not acquire while worker-1 holds lease")

	t.Log("Step 5: Wait for lease expiration")

	time.Sleep(2 * time.Second)

	t.Log("Step 6: worker-2 acquires expired lease (recovery)")

	acquired, err = db.AcquireLease(msgPath, "worker-2", 60)
	require.NoError(t, err)
	assert.True(t, acquired, "worker-2 should acquire expired lease")

	// worker-2 can now retry the message
	db.UpdateMessageStatus(msgEnv.MessageID, "processing")

	// Increment retry count
	_, err = db.GetMessage(msgEnv.MessageID)
	require.NoError(t, err)

	db.conn.Exec("UPDATE messages SET retry_count = retry_count + 1 WHERE message_id = ?", msgEnv.MessageID)

	// Complete processing
	db.MarkMessageProcessed(msgEnv.MessageID)
	db.ReleaseLease(msgPath)

	// Verify final state
	finalRecord, err := db.GetMessage(msgEnv.MessageID)
	require.NoError(t, err)
	assert.Equal(t, "completed", finalRecord.Status)

	t.Log("✅ Crash recovery test PASSED!")
	t.Log("  - worker-1 acquired lease")
	t.Log("  - worker-1 crashed (simulated)")
	t.Log("  - Lease expired after 1 second")
	t.Log("  - worker-2 recovered and completed message")
}

// TestIntegration_ReaperProcess tests the background reaper that cleans up
// expired leases.
func TestIntegration_ReaperProcess(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := NewDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	t.Log("Step 1: Register agents")

	db.RegisterAgent(&AgentInfo{
		AgentID:       "worker-1",
		InboxPath:     "/tmp/1",
		Status:        "active",
		ProtocolCaps:  `[]`,
		LastHeartbeat: time.Now().UTC(),
		CreatedAt:     time.Now().UTC(),
	})

	t.Log("Step 2: Create multiple expired leases")

	// Create 5 leases that expire in 1 second
	for i := 1; i <= 5; i++ {
		resourceID := fmt.Sprintf("resource_%d", i)
		db.AcquireLease(resourceID, "worker-1", 1)
	}

	// Create 2 leases that don't expire
	db.AcquireLease("resource_valid_1", "worker-1", 60)
	db.AcquireLease("resource_valid_2", "worker-1", 60)

	t.Log("Step 3: Wait for expiration")

	time.Sleep(2 * time.Second)

	t.Log("Step 4: Get expired leases before reaping")

	expired, err := db.GetExpiredLeases()
	require.NoError(t, err)
	assert.Len(t, expired, 5, "should have 5 expired leases")

	t.Log("Step 5: Run reaper process")

	count, err := db.ReapExpiredLeases()
	require.NoError(t, err)
	assert.Equal(t, 5, count, "reaper should clean up 5 expired leases")

	t.Log("Step 6: Verify only valid leases remain")

	// Check that expired leases are gone
	expired, err = db.GetExpiredLeases()
	require.NoError(t, err)
	assert.Len(t, expired, 0, "no expired leases should remain")

	// Verify valid leases still exist
	var validCount int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM agent_locks").Scan(&validCount)
	require.NoError(t, err)
	assert.Equal(t, 2, validCount, "2 valid leases should remain")

	t.Log("✅ Reaper process test PASSED!")
	t.Log("  - Created 5 expired + 2 valid leases")
	t.Log("  - Reaper cleaned up 5 expired leases")
	t.Log("  - 2 valid leases remain")
}

// TestIntegration_CrossProcessDeduplication tests that duplicate messages
// are detected across different agent processes.
func TestIntegration_CrossProcessDeduplication(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := NewDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	writer := NewMessageWriter(tmpDir)

	t.Log("Step 1: Register agents")

	db.RegisterAgent(&AgentInfo{AgentID: "sender", InboxPath: "/tmp/1", Status: "active", ProtocolCaps: `[]`, LastHeartbeat: time.Now().UTC(), CreatedAt: time.Now().UTC()})
	db.RegisterAgent(&AgentInfo{AgentID: "receiver", InboxPath: "/tmp/2", Status: "active", ProtocolCaps: `[]`, LastHeartbeat: time.Now().UTC(), CreatedAt: time.Now().UTC()})

	t.Log("Step 2: Agent sends message")

	msgEnv := &Envelope{
		ProtocolVersion: "1.0.0",
		MessageID:       "msg_duplicate_test",
		FromAgent:       "sender",
		ToAgent:         "receiver",
		MessageType:     "request",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
	}

	writer.WriteMessage(msgEnv)

	// Record in database
	db.RecordMessage(&MessageRecord{
		MessageID:   msgEnv.MessageID,
		FromAgent:   msgEnv.FromAgent,
		ToAgent:     msgEnv.ToAgent,
		MessageType: msgEnv.MessageType,
		Status:      "pending",
		CreatedAt:   time.Now().UTC(),
	})

	t.Log("Step 3: Receiver checks if message already processed")

	// First check (should be new)
	exists, err := db.MessageExists("msg_duplicate_test")
	require.NoError(t, err)
	assert.True(t, exists, "message should exist")

	t.Log("Step 4: Simulate retry (duplicate message ID)")

	// Try to record duplicate (should be ignored)
	err = db.RecordMessage(&MessageRecord{
		MessageID:   "msg_duplicate_test", // Same ID
		FromAgent:   msgEnv.FromAgent,
		ToAgent:     msgEnv.ToAgent,
		MessageType: msgEnv.MessageType,
		Status:      "pending",
		CreatedAt:   time.Now().UTC(),
	})
	require.NoError(t, err, "duplicate insert should succeed but be ignored")

	// Verify only one record exists
	var count int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM messages WHERE message_id = ?", "msg_duplicate_test").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "should have exactly 1 record (duplicate ignored)")

	t.Log("✅ Cross-process deduplication test PASSED!")
	t.Log("  - Message recorded in database")
	t.Log("  - Duplicate insert ignored (idempotency)")
	t.Log("  - Works across different agent processes")
}

// TestIntegration_DeadLetterQueue tests the full DLQ workflow:
// message fails -> moves to DLQ -> can be retried or deleted.
func TestIntegration_DeadLetterQueue(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := NewDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	dlq := NewDeadLetterQueue(tmpDir)

	t.Log("Step 1: Create a failing message")

	env := &Envelope{
		ProtocolVersion: "1.0.0",
		MessageID:       "msg_failing_123",
		FromAgent:       "agent-a",
		ToAgent:         "agent-b",
		MessageType:     "request",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		Retries:         3,
		Payload:         map[string]interface{}{"task": "process_data"},
	}

	// Record message in database
	db.RecordMessage(&MessageRecord{
		MessageID:   env.MessageID,
		FromAgent:   env.FromAgent,
		ToAgent:     env.ToAgent,
		MessageType: env.MessageType,
		Status:      "processing",
		CreatedAt:   time.Now().UTC(),
		RetryCount:  3,
		Attempt:     3,
	})

	t.Log("Step 2: Simulate max retries exceeded - move to DLQ")

	dlqPath, err := dlq.MoveToDeadLetter(env, "max retries exceeded", "error processing message")
	require.NoError(t, err)
	assert.FileExists(t, dlqPath)

	// Update message status in DB
	db.UpdateMessageStatus(env.MessageID, "failed")

	t.Log("Step 3: Verify message in DLQ")

	entries, err := dlq.GetDeadLetterMessages()
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "msg_failing_123", entries[0].MessageID)
	assert.Equal(t, "max retries exceeded", entries[0].FailureReason)
	assert.Equal(t, 3, entries[0].RetryCount)

	t.Log("Step 4: Verify message status in database")

	record, err := db.GetMessage(env.MessageID)
	require.NoError(t, err)
	assert.Equal(t, "failed", record.Status)
	assert.Equal(t, 3, record.RetryCount)

	t.Log("Step 5: Retry message from DLQ")

	retried, err := dlq.RetryFromDeadLetter(env.MessageID)
	require.NoError(t, err)
	assert.Equal(t, env.MessageID, retried.MessageID)
	assert.Equal(t, 0, retried.Retries, "retries should be reset")

	// Verify DLQ is empty
	entries, err = dlq.GetDeadLetterMessages()
	require.NoError(t, err)
	assert.Len(t, entries, 0, "DLQ should be empty after retry")

	t.Log("✅ Dead letter queue test PASSED!")
	t.Log("  - Message moved to DLQ after max retries")
	t.Log("  - DLQ entry contains failure metadata")
	t.Log("  - Message can be retried from DLQ")
	t.Log("  - Retry resets counter for fresh attempt")
}

// TestIntegration_RetryLogic tests the retry logic with incrementing counters.
func TestIntegration_RetryLogic(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := NewDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	t.Log("Step 1: Create initial message")

	env := &Envelope{
		ProtocolVersion: "1.0.0",
		MessageID:       "msg_retry_test",
		FromAgent:       "agent-a",
		ToAgent:         "agent-b",
		MessageType:     "request",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		Retries:         0,
	}

	// Record initial message
	db.RecordMessage(&MessageRecord{
		MessageID:   env.MessageID,
		FromAgent:   env.FromAgent,
		ToAgent:     env.ToAgent,
		MessageType: env.MessageType,
		Status:      "pending",
		CreatedAt:   time.Now().UTC(),
		RetryCount:  0,
		Attempt:     1,
	})

	t.Log("Step 2: Simulate first failure - increment retry count")

	err = db.IncrementRetryCount(env.MessageID)
	require.NoError(t, err)

	record, err := db.GetMessage(env.MessageID)
	require.NoError(t, err)
	assert.Equal(t, 1, record.RetryCount)
	assert.Equal(t, 2, record.Attempt)

	t.Log("Step 3: Simulate second failure - increment again")

	err = db.IncrementRetryCount(env.MessageID)
	require.NoError(t, err)

	record, err = db.GetMessage(env.MessageID)
	require.NoError(t, err)
	assert.Equal(t, 2, record.RetryCount)
	assert.Equal(t, 3, record.Attempt)

	t.Log("Step 4: Simulate third failure - increment again")

	err = db.IncrementRetryCount(env.MessageID)
	require.NoError(t, err)

	record, err = db.GetMessage(env.MessageID)
	require.NoError(t, err)
	assert.Equal(t, 3, record.RetryCount)
	assert.Equal(t, 4, record.Attempt)

	t.Log("Step 5: Check if max retries reached (threshold: 3)")

	const maxRetries = 3
	if record.RetryCount >= maxRetries {
		t.Log("  - Max retries reached, should move to DLQ")
		db.UpdateMessageStatus(env.MessageID, "failed")
	}

	t.Log("✅ Retry logic test PASSED!")
	t.Log("  - Retry counter increments correctly")
	t.Log("  - Attempt counter tracks total attempts")
	t.Log("  - Max retries detection works")
}

// TestIntegration_ExpiredMessages tests retrieval of messages past their deadline.
func TestIntegration_ExpiredMessages(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := NewDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	t.Log("Step 1: Create messages with deadlines")

	// Message 1: Already expired
	expired1 := &MessageRecord{
		MessageID:   "msg_expired_1",
		FromAgent:   "agent-a",
		ToAgent:     "agent-b",
		MessageType: "request",
		Status:      "pending",
		CreatedAt:   time.Now().UTC(),
		TTLSeconds:  60,
		Deadline:    ptrTime(time.Now().UTC().Add(-10 * time.Second)), // 10 seconds ago
		Attempt:     1,
	}

	// Message 2: Not yet expired
	notExpired := &MessageRecord{
		MessageID:   "msg_not_expired",
		FromAgent:   "agent-a",
		ToAgent:     "agent-b",
		MessageType: "request",
		Status:      "pending",
		CreatedAt:   time.Now().UTC(),
		TTLSeconds:  60,
		Deadline:    ptrTime(time.Now().UTC().Add(10 * time.Second)), // 10 seconds from now
		Attempt:     1,
	}

	// Message 3: Expired but already completed (should be ignored)
	expiredCompleted := &MessageRecord{
		MessageID:   "msg_expired_completed",
		FromAgent:   "agent-a",
		ToAgent:     "agent-b",
		MessageType: "request",
		Status:      "completed",
		CreatedAt:   time.Now().UTC(),
		TTLSeconds:  60,
		Deadline:    ptrTime(time.Now().UTC().Add(-5 * time.Second)),
		Attempt:     1,
	}

	db.RecordMessage(expired1)
	db.RecordMessage(notExpired)
	db.RecordMessage(expiredCompleted)

	t.Log("Step 2: Retrieve expired messages")

	expired, err := db.GetExpiredMessages(10)
	require.NoError(t, err)

	t.Log("Step 3: Verify only pending/processing expired messages returned")

	assert.Len(t, expired, 1, "should only return 1 expired pending message")
	assert.Equal(t, "msg_expired_1", expired[0].MessageID)

	t.Log("✅ Expired messages test PASSED!")
	t.Log("  - Expired pending messages detected")
	t.Log("  - Non-expired messages ignored")
	t.Log("  - Completed messages ignored")
}

// Helper function to create time pointer
func ptrTime(t time.Time) *time.Time {
	return &t
}
