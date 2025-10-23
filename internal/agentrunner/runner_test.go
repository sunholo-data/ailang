package agentrunner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunholo/ailang/internal/agentprotocol"
)

// MockHandler is a test message handler.
type MockHandler struct {
	messages []*agentprotocol.Envelope
	response map[string]interface{}
	err      error
}

func (h *MockHandler) HandleMessage(msg *agentprotocol.Envelope) (map[string]interface{}, error) {
	h.messages = append(h.messages, msg)
	return h.response, h.err
}

func TestNewRunner(t *testing.T) {
	tmpDir := t.TempDir()

	handler := &MockHandler{}
	config := &AgentConfig{
		AgentID:       "test-agent",
		StateDir:      tmpDir,
		PollInterval:  1 * time.Second,
		LeaseDuration: 60,
		Handler:       handler,
	}

	runner, err := NewRunner(config)
	require.NoError(t, err)
	defer runner.Stop()

	assert.NotNil(t, runner.db)
	assert.NotNil(t, runner.writer)
	assert.NotNil(t, runner.reader)
}

func TestRunner_ProcessMessage(t *testing.T) {
	tmpDir := t.TempDir()

	handler := &MockHandler{
		response: map[string]interface{}{
			"status": "success",
			"result": "processed",
		},
	}

	config := &AgentConfig{
		AgentID:       "test-agent",
		StateDir:      tmpDir,
		PollInterval:  1 * time.Second,
		LeaseDuration: 60,
		Handler:       handler,
	}

	runner, err := NewRunner(config)
	require.NoError(t, err)
	defer runner.Stop()

	// Register sender agent
	runner.db.RegisterAgent(&agentprotocol.AgentInfo{
		AgentID:       "sender",
		InboxPath:     tmpDir + "/messages/sender",
		Status:        "active",
		ProtocolCaps:  `["v1.0"]`,
		LastHeartbeat: time.Now().UTC(),
		CreatedAt:     time.Now().UTC(),
	})

	// Create a message for test-agent
	writer := agentprotocol.NewMessageWriter(tmpDir)
	msgEnv := &agentprotocol.Envelope{
		ProtocolVersion: "1.0.0",
		MessageID:       agentprotocol.GenerateMessageID(),
		CorrelationID:   agentprotocol.GenerateCorrelationID(),
		TraceID:         agentprotocol.GenerateTraceID(),
		FromAgent:       "sender",
		ToAgent:         "test-agent",
		MessageType:     "request",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		Payload: map[string]interface{}{
			"action": "test",
		},
	}

	_, err = writer.WriteMessage(msgEnv)
	require.NoError(t, err)

	// Run once to process the message
	err = runner.RunOnce()
	require.NoError(t, err)

	// Verify handler was called
	assert.Len(t, handler.messages, 1)
	assert.Equal(t, msgEnv.MessageID, handler.messages[0].MessageID)
	assert.Equal(t, "sender", handler.messages[0].FromAgent)

	// Verify message was marked as completed in database
	record, err := runner.db.GetMessage(msgEnv.MessageID)
	require.NoError(t, err)
	assert.Equal(t, "completed", record.Status)
	assert.NotNil(t, record.ProcessedAt)

	// Verify response was sent
	reader := agentprotocol.NewMessageReader(tmpDir)
	pending, err := reader.ScanPendingMessages("sender")
	require.NoError(t, err)
	assert.Len(t, pending, 1, "response should be sent to sender")

	// Read response
	response, err := reader.ReadMessage(pending[0])
	require.NoError(t, err)
	assert.Equal(t, "test-agent", response.FromAgent)
	assert.Equal(t, "sender", response.ToAgent)
	assert.Equal(t, "response", response.MessageType)
	assert.Equal(t, msgEnv.MessageID, *response.ParentMessageID)
	assert.Equal(t, "success", response.Payload["status"])
}

func TestRunner_Idempotency(t *testing.T) {
	tmpDir := t.TempDir()

	handler := &MockHandler{
		response: map[string]interface{}{"status": "ok"},
	}

	config := &AgentConfig{
		AgentID:       "test-agent",
		StateDir:      tmpDir,
		PollInterval:  1 * time.Second,
		LeaseDuration: 60,
		Handler:       handler,
	}

	runner, err := NewRunner(config)
	require.NoError(t, err)
	defer runner.Stop()

	// Register sender
	runner.db.RegisterAgent(&agentprotocol.AgentInfo{
		AgentID:       "sender",
		InboxPath:     tmpDir + "/messages/sender",
		Status:        "active",
		ProtocolCaps:  `["v1.0"]`,
		LastHeartbeat: time.Now().UTC(),
		CreatedAt:     time.Now().UTC(),
	})

	// Create message
	writer := agentprotocol.NewMessageWriter(tmpDir)
	msgEnv := &agentprotocol.Envelope{
		ProtocolVersion: "1.0.0",
		MessageID:       "msg_idempotency_test",
		FromAgent:       "sender",
		ToAgent:         "test-agent",
		MessageType:     "request",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
	}

	writer.WriteMessage(msgEnv)

	// Process first time
	err = runner.RunOnce()
	require.NoError(t, err)
	assert.Len(t, handler.messages, 1)

	// Process second time (should skip - database deduplication)
	err = runner.RunOnce()
	require.NoError(t, err)
	assert.Len(t, handler.messages, 1, "handler should not be called again")
}

func TestRunner_LeaseAcquisition(t *testing.T) {
	tmpDir := t.TempDir()

	handler1 := &MockHandler{response: map[string]interface{}{"agent": "1"}}
	handler2 := &MockHandler{response: map[string]interface{}{"agent": "2"}}

	// Create two runners for same message
	config1 := &AgentConfig{
		AgentID:       "agent-1",
		StateDir:      tmpDir,
		PollInterval:  1 * time.Second,
		LeaseDuration: 60,
		Handler:       handler1,
	}

	config2 := &AgentConfig{
		AgentID:       "agent-2",
		StateDir:      tmpDir,
		PollInterval:  1 * time.Second,
		LeaseDuration: 60,
		Handler:       handler2,
	}

	runner1, err := NewRunner(config1)
	require.NoError(t, err)
	defer runner1.Stop()

	runner2, err := NewRunner(config2)
	require.NoError(t, err)
	defer runner2.Stop()

	// Register sender
	runner1.db.RegisterAgent(&agentprotocol.AgentInfo{
		AgentID:       "sender",
		InboxPath:     tmpDir + "/messages/sender",
		Status:        "active",
		ProtocolCaps:  `["v1.0"]`,
		LastHeartbeat: time.Now().UTC(),
		CreatedAt:     time.Now().UTC(),
	})

	// Create message for both agents (simulate broadcast or race condition)
	writer := agentprotocol.NewMessageWriter(tmpDir)
	msgEnv := &agentprotocol.Envelope{
		ProtocolVersion: "1.0.0",
		MessageID:       agentprotocol.GenerateMessageID(),
		FromAgent:       "sender",
		ToAgent:         "agent-1",
		MessageType:     "notification",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
	}

	msgPath, _ := writer.WriteMessage(msgEnv)

	// Both runners try to acquire lease on same message
	// First one should succeed, second should fail
	acquired1, _ := runner1.db.AcquireLease(msgPath, "agent-1", 60)
	acquired2, _ := runner2.db.AcquireLease(msgPath, "agent-2", 60)

	// Exactly one should have acquired the lease
	assert.True(t, acquired1 != acquired2, "exactly one agent should acquire lease")

	if acquired1 {
		assert.True(t, acquired1)
		assert.False(t, acquired2)
	} else {
		assert.False(t, acquired1)
		assert.True(t, acquired2)
	}
}
