package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo/ailang/internal/messaging"
)

// TestAgent_NewAgent tests agent creation
func TestAgent_NewAgent(t *testing.T) {
	// Create temp database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	agent, err := NewAgent("test-agent", dbPath, 2)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}
	defer agent.Close()

	if agent.instanceID != "test-agent" {
		t.Errorf("Expected instanceID=test-agent, got %s", agent.instanceID)
	}

	if agent.pollInterval != 2*time.Second {
		t.Errorf("Expected pollInterval=2s, got %v", agent.pollInterval)
	}

	if agent.client == nil {
		t.Error("Expected client to be initialized")
	}
}

// TestAgent_Poll tests message polling
func TestAgent_Poll(t *testing.T) {
	// Create temp database and agent
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	agent, err := NewAgent("test-agent", dbPath, 2)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}
	defer agent.Close()

	// Create a message for the agent using a separate store instance
	store, err := messaging.OpenStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer store.Close()

	// First create a thread
	thread, err := store.CreateThread("Test Thread", "human", "user")
	if err != nil {
		t.Fatalf("Failed to create thread: %v", err)
	}

	// Create a directive message
	_, err = store.CreateMessage(
		thread.ID,
		"human", "user",
		"ailang_instance", "test-agent",
		"directive",
		"Create hello.txt",
	)
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	// Poll should find the message
	ctx := context.Background()
	if err := agent.poll(ctx); err != nil {
		t.Fatalf("Poll failed: %v", err)
	}

	// Message should now be acknowledged
	messages, err := agent.client.PollMessages()
	if err != nil {
		t.Fatalf("Failed to poll after acknowledgment: %v", err)
	}

	// Should be no pending messages after acknowledgment
	if len(messages) != 0 {
		t.Errorf("Expected 0 pending messages after acknowledgment, got %d", len(messages))
		for _, m := range messages {
			t.Logf("  Pending message: %s (%s)", m.ID, m.Kind)
		}
	}
}

// TestAgent_ProcessMessage tests individual message processing
func TestAgent_ProcessMessage(t *testing.T) {
	// Create temp database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create store and thread
	store, err := messaging.OpenStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer store.Close()

	thread, err := store.CreateThread("Test Thread", "human", "user")
	if err != nil {
		t.Fatalf("Failed to create thread: %v", err)
	}

	// Create agent
	agent, err := NewAgent("test-agent", dbPath, 2)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}
	defer agent.Close()

	// Create a test message
	msg := &messaging.Message{
		ID:       "msg-123",
		ThreadID: thread.ID, // Use real thread ID
		FromType: "human",
		FromID:   "user",
		ToType:   "ailang_instance",
		ToID:     "test-agent",
		Kind:     "directive",
		Content:  "Test directive",
	}

	// Process message
	ctx := context.Background()
	if err := agent.processMessage(ctx, msg); err != nil {
		t.Fatalf("Failed to process message: %v", err)
	}

	// Verify result was published
	messages, err := store.GetMessages("human", "user", "")
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}

	// Should have at least one result message
	var foundResult bool
	for _, m := range messages {
		if m.Kind == "result" && m.ThreadID == thread.ID {
			foundResult = true
			break
		}
	}

	if !foundResult {
		t.Error("Expected result message to be published, but none found")
	}
}

// TestAgent_RunShutdown tests graceful shutdown
func TestAgent_RunShutdown(t *testing.T) {
	// Create temp database and agent
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	agent, err := NewAgent("test-agent", dbPath, 1) // 1 second poll interval for faster test
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}
	defer agent.Close()

	// Start agent in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- agent.Run(ctx)
	}()

	// Let it run for a bit
	time.Sleep(500 * time.Millisecond)

	// Cancel context to trigger shutdown
	cancel()

	// Wait for shutdown with timeout
	select {
	case err := <-errChan:
		// context.Canceled is expected
		if err != nil && err != context.Canceled {
			t.Fatalf("Unexpected error during shutdown: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Agent did not shutdown within 2 seconds")
	}
}

// TestAgent_Close tests resource cleanup
func TestAgent_Close(t *testing.T) {
	// Create temp database and agent
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	agent, err := NewAgent("test-agent", dbPath, 2)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Close should not error
	if err := agent.Close(); err != nil {
		t.Errorf("Close returned error: %v", err)
	}

	// Note: Double-closing currently panics in messaging.Client.Close()
	// This is a known issue in the messaging client (closes channel twice)
	// For now, we just test single close works correctly
}

// TestAgent_Integration tests end-to-end message flow
func TestAgent_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Skip unless explicitly enabled (executes real Claude Code, costs money)
	if os.Getenv("TEST_AGENT_INTEGRATION") == "" {
		t.Skip("Skipping agent integration test (set TEST_AGENT_INTEGRATION=1 to enable)")
	}

	// Create temp database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create store and thread
	store, err := messaging.OpenStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer store.Close()

	thread, err := store.CreateThread("Integration Test", "human", "user")
	if err != nil {
		t.Fatalf("Failed to create thread: %v", err)
	}

	// Create 3 directive messages
	for i := 0; i < 3; i++ {
		_, err := store.CreateMessage(
			thread.ID,
			"human", "user",
			"ailang_instance", "test-agent",
			"directive",
			"Directive "+string(rune('A'+i)),
		)
		if err != nil {
			t.Fatalf("Failed to create message %d: %v", i, err)
		}
	}

	// Create agent
	agent, err := NewAgent("test-agent", dbPath, 1) // 1 second poll for faster test
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}
	defer agent.Close()

	// Start agent in background
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		_ = agent.Run(ctx)
	}()

	// Wait for messages to be processed (give it 3 seconds)
	time.Sleep(3 * time.Second)

	// Check that all messages were acknowledged
	messages, err := agent.client.PollMessages()
	if err != nil {
		t.Fatalf("Failed to poll messages: %v", err)
	}

	if len(messages) != 0 {
		t.Errorf("Expected all 3 messages to be acknowledged, but %d are still pending", len(messages))
	}
}

// TestAgent_ResultPublishing tests that execution results are published back to the UI
func TestAgent_ResultPublishing(t *testing.T) {
	if os.Getenv("TEST_AGENT_INTEGRATION") == "" {
		t.Skip("Skipping result publishing test (set TEST_AGENT_INTEGRATION=1 to enable)")
	}

	// Create temp database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create store and thread
	store, err := messaging.OpenStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer store.Close()

	thread, err := store.CreateThread("Result Publishing Test", "human", "user")
	if err != nil {
		t.Fatalf("Failed to create thread: %v", err)
	}

	// Create a directive message
	directive := "Write a simple hello world AILANG program"
	_, err = store.CreateMessage(
		thread.ID,
		"human", "user",
		"ailang_instance", "test-agent",
		"directive",
		directive,
	)
	if err != nil {
		t.Fatalf("Failed to create directive message: %v", err)
	}

	// Create agent
	agent, err := NewAgent("test-agent", dbPath, 1)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}
	defer agent.Close()

	// Process messages once
	ctx := context.Background()
	if err := agent.poll(ctx); err != nil {
		t.Logf("Poll error (expected if directive fails): %v", err)
	}

	// Check that a result message was published
	allMessages, err := store.GetMessages("human", "user", "")
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}

	// Should have 2 messages: 1 directive (acked) + 1 result (pending)
	if len(allMessages) < 2 {
		t.Fatalf("Expected at least 2 messages (directive + result), got %d", len(allMessages))
	}

	// Find the result message
	var resultMsg *messaging.Message
	for i := range allMessages {
		if allMessages[i].Kind == "result" {
			resultMsg = &allMessages[i]
			break
		}
	}

	if resultMsg == nil {
		t.Fatal("No result message found")
	}

	// Verify result message properties
	if resultMsg.FromType != "ailang_instance" || resultMsg.FromID != "test-agent" {
		t.Errorf("Expected result from ailang_instance/test-agent, got %s/%s",
			resultMsg.FromType, resultMsg.FromID)
	}

	if resultMsg.ToType != "human" || resultMsg.ToID != "user" {
		t.Errorf("Expected result to human/user, got %s/%s",
			resultMsg.ToType, resultMsg.ToID)
	}

	if resultMsg.ThreadID != thread.ID {
		t.Errorf("Expected result in thread %s, got %s", thread.ID, resultMsg.ThreadID)
	}

	// Verify result content contains expected markdown sections
	expectedSections := []string{
		"Directive Completed", // Status header
		"Duration:",           // Summary section
		"Cost:",               // Summary section
		"Turns:",              // Summary section
	}

	for _, section := range expectedSections {
		if !strings.Contains(resultMsg.Content, section) {
			t.Errorf("Result content missing expected section: %s", section)
		}
	}

	t.Logf("Result message published successfully:")
	t.Logf("  ID: %s", resultMsg.ID)
	t.Logf("  Thread: %s", resultMsg.ThreadID)
	t.Logf("  Content preview: %s...", resultMsg.Content[:min(100, len(resultMsg.Content))])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
