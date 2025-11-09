package main

import (
	"context"
	"path/filepath"
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
	// Create temp database and agent
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	agent, err := NewAgent("test-agent", dbPath, 2)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}
	defer agent.Close()

	// Create a test message
	msg := &messaging.Message{
		ID:       "msg-123",
		ThreadID: "thread-123",
		FromType: "human",
		FromID:   "user",
		ToType:   "ailang_instance",
		ToID:     "test-agent",
		Kind:     "directive",
		Content:  "Test directive",
	}

	// Process message (should log but not fail)
	ctx := context.Background()
	if err := agent.processMessage(ctx, msg); err != nil {
		t.Fatalf("Failed to process message: %v", err)
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
