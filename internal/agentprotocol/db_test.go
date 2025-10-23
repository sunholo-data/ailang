package agentprotocol

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDB(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := NewDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	// Verify database file was created
	dbPath := filepath.Join(tmpDir, "agents.db")
	_, err = os.Stat(dbPath)
	assert.NoError(t, err, "database file should exist")

	// Verify WAL mode is enabled
	var mode string
	err = db.conn.QueryRow("PRAGMA journal_mode").Scan(&mode)
	require.NoError(t, err)
	assert.Equal(t, "wal", mode, "should use WAL mode")
}

func TestRegisterAgent(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	// Register a new agent
	info := &AgentInfo{
		AgentID:       "test-agent",
		InboxPath:     "/tmp/inbox",
		Status:        "active",
		ProtocolCaps:  `["v1.0", "hmac"]`,
		LastHeartbeat: time.Now().UTC(),
		CreatedAt:     time.Now().UTC(),
	}

	err = db.RegisterAgent(info)
	assert.NoError(t, err)

	// Retrieve the agent
	retrieved, err := db.GetAgent("test-agent")
	require.NoError(t, err)
	assert.Equal(t, info.AgentID, retrieved.AgentID)
	assert.Equal(t, info.InboxPath, retrieved.InboxPath)
	assert.Equal(t, info.Status, retrieved.Status)
	assert.Equal(t, info.ProtocolCaps, retrieved.ProtocolCaps)
}

func TestRegisterAgentUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	// Initial registration
	info := &AgentInfo{
		AgentID:       "test-agent",
		InboxPath:     "/tmp/inbox1",
		Status:        "active",
		ProtocolCaps:  `["v1.0"]`,
		LastHeartbeat: time.Now().UTC(),
		CreatedAt:     time.Now().UTC(),
	}
	err = db.RegisterAgent(info)
	require.NoError(t, err)

	// Update registration
	info.InboxPath = "/tmp/inbox2"
	info.Status = "paused"
	info.ProtocolCaps = `["v1.0", "v1.1"]`
	info.LastHeartbeat = time.Now().UTC()

	err = db.RegisterAgent(info)
	require.NoError(t, err)

	// Verify update
	retrieved, err := db.GetAgent("test-agent")
	require.NoError(t, err)
	assert.Equal(t, "/tmp/inbox2", retrieved.InboxPath)
	assert.Equal(t, "paused", retrieved.Status)
	assert.Equal(t, `["v1.0", "v1.1"]`, retrieved.ProtocolCaps)
}

func TestGetAgentNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	_, err = db.GetAgent("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent not found")
}

func TestListActiveAgents(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	// Register multiple agents with different statuses
	agents := []*AgentInfo{
		{AgentID: "agent1", InboxPath: "/tmp/1", Status: "active", ProtocolCaps: `[]`, LastHeartbeat: time.Now().UTC(), CreatedAt: time.Now().UTC()},
		{AgentID: "agent2", InboxPath: "/tmp/2", Status: "paused", ProtocolCaps: `[]`, LastHeartbeat: time.Now().UTC(), CreatedAt: time.Now().UTC()},
		{AgentID: "agent3", InboxPath: "/tmp/3", Status: "active", ProtocolCaps: `[]`, LastHeartbeat: time.Now().UTC(), CreatedAt: time.Now().UTC()},
		{AgentID: "agent4", InboxPath: "/tmp/4", Status: "error", ProtocolCaps: `[]`, LastHeartbeat: time.Now().UTC(), CreatedAt: time.Now().UTC()},
	}

	for _, a := range agents {
		err := db.RegisterAgent(a)
		require.NoError(t, err)
	}

	// List active agents
	active, err := db.ListActiveAgents()
	require.NoError(t, err)
	assert.Len(t, active, 2)

	// Verify only active agents returned
	ids := make([]string, len(active))
	for i, a := range active {
		ids[i] = a.AgentID
	}
	assert.Contains(t, ids, "agent1")
	assert.Contains(t, ids, "agent3")
}

func TestUpdateAgentStatus(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	// Register agent
	info := &AgentInfo{
		AgentID:       "test-agent",
		InboxPath:     "/tmp/inbox",
		Status:        "active",
		ProtocolCaps:  `[]`,
		LastHeartbeat: time.Now().UTC(),
		CreatedAt:     time.Now().UTC(),
	}
	err = db.RegisterAgent(info)
	require.NoError(t, err)

	// Update status
	err = db.UpdateAgentStatus("test-agent", "paused")
	require.NoError(t, err)

	// Verify
	retrieved, err := db.GetAgent("test-agent")
	require.NoError(t, err)
	assert.Equal(t, "paused", retrieved.Status)
}

func TestRecordMessage(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	// Register agents first (foreign key constraint)
	db.RegisterAgent(&AgentInfo{AgentID: "sender", InboxPath: "/tmp/1", Status: "active", ProtocolCaps: `[]`, LastHeartbeat: time.Now().UTC(), CreatedAt: time.Now().UTC()})
	db.RegisterAgent(&AgentInfo{AgentID: "receiver", InboxPath: "/tmp/2", Status: "active", ProtocolCaps: `[]`, LastHeartbeat: time.Now().UTC(), CreatedAt: time.Now().UTC()})

	// Record message
	record := &MessageRecord{
		MessageID:     "msg_001",
		CorrelationID: "corr_123",
		TraceID:       "trace_456",
		FromAgent:     "sender",
		ToAgent:       "receiver",
		MessageType:   "request",
		Status:        "pending",
		CreatedAt:     time.Now().UTC(),
		RetryCount:    0,
	}

	err = db.RecordMessage(record)
	assert.NoError(t, err)

	// Retrieve message
	retrieved, err := db.GetMessage("msg_001")
	require.NoError(t, err)
	assert.Equal(t, record.MessageID, retrieved.MessageID)
	assert.Equal(t, record.CorrelationID, retrieved.CorrelationID)
	assert.Equal(t, record.FromAgent, retrieved.FromAgent)
	assert.Equal(t, record.ToAgent, retrieved.ToAgent)
	assert.Equal(t, "pending", retrieved.Status)
}

func TestRecordMessageIdempotency(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	// Register agents
	db.RegisterAgent(&AgentInfo{AgentID: "sender", InboxPath: "/tmp/1", Status: "active", ProtocolCaps: `[]`, LastHeartbeat: time.Now().UTC(), CreatedAt: time.Now().UTC()})
	db.RegisterAgent(&AgentInfo{AgentID: "receiver", InboxPath: "/tmp/2", Status: "active", ProtocolCaps: `[]`, LastHeartbeat: time.Now().UTC(), CreatedAt: time.Now().UTC()})

	// Record same message twice
	record := &MessageRecord{
		MessageID:   "msg_duplicate",
		FromAgent:   "sender",
		ToAgent:     "receiver",
		MessageType: "request",
		Status:      "pending",
		CreatedAt:   time.Now().UTC(),
	}

	err = db.RecordMessage(record)
	require.NoError(t, err)

	// Second insert should be ignored (ON CONFLICT DO NOTHING)
	err = db.RecordMessage(record)
	assert.NoError(t, err)

	// Verify only one record exists
	exists, err := db.MessageExists("msg_duplicate")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestMessageExists(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	// Register agents
	db.RegisterAgent(&AgentInfo{AgentID: "sender", InboxPath: "/tmp/1", Status: "active", ProtocolCaps: `[]`, LastHeartbeat: time.Now().UTC(), CreatedAt: time.Now().UTC()})
	db.RegisterAgent(&AgentInfo{AgentID: "receiver", InboxPath: "/tmp/2", Status: "active", ProtocolCaps: `[]`, LastHeartbeat: time.Now().UTC(), CreatedAt: time.Now().UTC()})

	// Check non-existent message
	exists, err := db.MessageExists("msg_nonexistent")
	require.NoError(t, err)
	assert.False(t, exists)

	// Record message
	record := &MessageRecord{
		MessageID:   "msg_exists",
		FromAgent:   "sender",
		ToAgent:     "receiver",
		MessageType: "request",
		Status:      "pending",
		CreatedAt:   time.Now().UTC(),
	}
	err = db.RecordMessage(record)
	require.NoError(t, err)

	// Check existing message
	exists, err = db.MessageExists("msg_exists")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestUpdateMessageStatus(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	// Register agents and message
	db.RegisterAgent(&AgentInfo{AgentID: "sender", InboxPath: "/tmp/1", Status: "active", ProtocolCaps: `[]`, LastHeartbeat: time.Now().UTC(), CreatedAt: time.Now().UTC()})
	db.RegisterAgent(&AgentInfo{AgentID: "receiver", InboxPath: "/tmp/2", Status: "active", ProtocolCaps: `[]`, LastHeartbeat: time.Now().UTC(), CreatedAt: time.Now().UTC()})

	record := &MessageRecord{
		MessageID:   "msg_status",
		FromAgent:   "sender",
		ToAgent:     "receiver",
		MessageType: "request",
		Status:      "pending",
		CreatedAt:   time.Now().UTC(),
	}
	db.RecordMessage(record)

	// Update status
	err = db.UpdateMessageStatus("msg_status", "processing")
	require.NoError(t, err)

	// Verify
	retrieved, err := db.GetMessage("msg_status")
	require.NoError(t, err)
	assert.Equal(t, "processing", retrieved.Status)
}

func TestMarkMessageProcessed(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	// Register agents and message
	db.RegisterAgent(&AgentInfo{AgentID: "sender", InboxPath: "/tmp/1", Status: "active", ProtocolCaps: `[]`, LastHeartbeat: time.Now().UTC(), CreatedAt: time.Now().UTC()})
	db.RegisterAgent(&AgentInfo{AgentID: "receiver", InboxPath: "/tmp/2", Status: "active", ProtocolCaps: `[]`, LastHeartbeat: time.Now().UTC(), CreatedAt: time.Now().UTC()})

	record := &MessageRecord{
		MessageID:   "msg_complete",
		FromAgent:   "sender",
		ToAgent:     "receiver",
		MessageType: "request",
		Status:      "pending",
		CreatedAt:   time.Now().UTC(),
	}
	db.RecordMessage(record)

	// Mark as processed
	err = db.MarkMessageProcessed("msg_complete")
	require.NoError(t, err)

	// Verify status and timestamp
	retrieved, err := db.GetMessage("msg_complete")
	require.NoError(t, err)
	assert.Equal(t, "completed", retrieved.Status)
	assert.NotNil(t, retrieved.ProcessedAt)
}

func TestAcquireLease(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	// Register agent
	db.RegisterAgent(&AgentInfo{AgentID: "agent1", InboxPath: "/tmp/1", Status: "active", ProtocolCaps: `[]`, LastHeartbeat: time.Now().UTC(), CreatedAt: time.Now().UTC()})

	// Acquire lease
	acquired, err := db.AcquireLease("resource_1", "agent1", 60)
	require.NoError(t, err)
	assert.True(t, acquired, "should acquire lease on first attempt")

	// Try to acquire same lease with different agent
	db.RegisterAgent(&AgentInfo{AgentID: "agent2", InboxPath: "/tmp/2", Status: "active", ProtocolCaps: `[]`, LastHeartbeat: time.Now().UTC(), CreatedAt: time.Now().UTC()})
	acquired, err = db.AcquireLease("resource_1", "agent2", 60)
	require.NoError(t, err)
	assert.False(t, acquired, "should not acquire already-held lease")
}

func TestAcquireLeaseExpired(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	// Register agents
	db.RegisterAgent(&AgentInfo{AgentID: "agent1", InboxPath: "/tmp/1", Status: "active", ProtocolCaps: `[]`, LastHeartbeat: time.Now().UTC(), CreatedAt: time.Now().UTC()})
	db.RegisterAgent(&AgentInfo{AgentID: "agent2", InboxPath: "/tmp/2", Status: "active", ProtocolCaps: `[]`, LastHeartbeat: time.Now().UTC(), CreatedAt: time.Now().UTC()})

	// Acquire lease with 1 second duration
	acquired, err := db.AcquireLease("resource_exp", "agent1", 1)
	require.NoError(t, err)
	assert.True(t, acquired)

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Another agent should be able to acquire now
	acquired, err = db.AcquireLease("resource_exp", "agent2", 60)
	require.NoError(t, err)
	assert.True(t, acquired, "should acquire expired lease")
}

func TestReleaseLease(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	// Register agents
	db.RegisterAgent(&AgentInfo{AgentID: "agent1", InboxPath: "/tmp/1", Status: "active", ProtocolCaps: `[]`, LastHeartbeat: time.Now().UTC(), CreatedAt: time.Now().UTC()})
	db.RegisterAgent(&AgentInfo{AgentID: "agent2", InboxPath: "/tmp/2", Status: "active", ProtocolCaps: `[]`, LastHeartbeat: time.Now().UTC(), CreatedAt: time.Now().UTC()})

	// Acquire and release lease
	acquired, err := db.AcquireLease("resource_rel", "agent1", 60)
	require.NoError(t, err)
	assert.True(t, acquired)

	err = db.ReleaseLease("resource_rel")
	require.NoError(t, err)

	// Another agent should be able to acquire now
	acquired, err = db.AcquireLease("resource_rel", "agent2", 60)
	require.NoError(t, err)
	assert.True(t, acquired, "should acquire released lease")
}

func TestReapExpiredLeases(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	// Register agent
	db.RegisterAgent(&AgentInfo{AgentID: "agent1", InboxPath: "/tmp/1", Status: "active", ProtocolCaps: `[]`, LastHeartbeat: time.Now().UTC(), CreatedAt: time.Now().UTC()})

	// Acquire multiple leases with 1 second duration
	db.AcquireLease("res1", "agent1", 1)
	db.AcquireLease("res2", "agent1", 1)
	db.AcquireLease("res3", "agent1", 60) // This one won't expire

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Reap expired leases
	count, err := db.ReapExpiredLeases()
	require.NoError(t, err)
	assert.Equal(t, 2, count, "should reap 2 expired leases")
}

func TestGetExpiredLeases(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	// Register agent
	db.RegisterAgent(&AgentInfo{AgentID: "agent1", InboxPath: "/tmp/1", Status: "active", ProtocolCaps: `[]`, LastHeartbeat: time.Now().UTC(), CreatedAt: time.Now().UTC()})

	// Acquire lease with 1 second duration
	db.AcquireLease("res_exp", "agent1", 1)

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Get expired leases
	expired, err := db.GetExpiredLeases()
	require.NoError(t, err)
	assert.Len(t, expired, 1)
	assert.Equal(t, "res_exp", expired[0].ResourceID)
	assert.Equal(t, "agent1", expired[0].LockedBy)
}

func TestLogEvent(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	// Register agent
	db.RegisterAgent(&AgentInfo{AgentID: "agent1", InboxPath: "/tmp/1", Status: "active", ProtocolCaps: `[]`, LastHeartbeat: time.Now().UTC(), CreatedAt: time.Now().UTC()})

	// Log event
	err = db.LogEvent("agent1", "msg_123", "message_sent", `{"target": "agent2"}`)
	assert.NoError(t, err)

	// Verify event was recorded (query directly)
	var count int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM agent_history WHERE agent_id = ?", "agent1").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestRecordMetric(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	// Register agent
	db.RegisterAgent(&AgentInfo{AgentID: "agent1", InboxPath: "/tmp/1", Status: "active", ProtocolCaps: `[]`, LastHeartbeat: time.Now().UTC(), CreatedAt: time.Now().UTC()})

	// Record metric
	err = db.RecordMetric("agent1", "latency_ms", 123.45)
	assert.NoError(t, err)

	// Verify metric was recorded
	var value float64
	err = db.conn.QueryRow("SELECT metric_value FROM agent_metrics WHERE agent_id = ? AND metric_name = ?",
		"agent1", "latency_ms").Scan(&value)
	require.NoError(t, err)
	assert.InDelta(t, 123.45, value, 0.01)
}

func TestConcurrentLeaseAcquisition(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := NewDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	// Register multiple agents
	for i := 1; i <= 5; i++ {
		db.RegisterAgent(&AgentInfo{
			AgentID:       fmt.Sprintf("agent%d", i),
			InboxPath:     fmt.Sprintf("/tmp/%d", i),
			Status:        "active",
			ProtocolCaps:  `[]`,
			LastHeartbeat: time.Now().UTC(),
			CreatedAt:     time.Now().UTC(),
		})
	}

	// Try to acquire same lease concurrently
	const numGoroutines = 5
	results := make(chan bool, numGoroutines)

	for i := 1; i <= numGoroutines; i++ {
		agentID := fmt.Sprintf("agent%d", i)
		go func(id string) {
			acquired, _ := db.AcquireLease("shared_resource", id, 60)
			results <- acquired
		}(agentID)
	}

	// Collect results
	acquired := 0
	for i := 0; i < numGoroutines; i++ {
		if <-results {
			acquired++
		}
	}

	// Only one should have acquired the lease
	assert.Equal(t, 1, acquired, "exactly one agent should acquire the lease")
}
