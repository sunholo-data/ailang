package messaging

import (
	"path/filepath"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	client, err := NewClient(dbPath, "test_instance")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	if client.instanceID != "test_instance" {
		t.Errorf("Expected instance ID 'test_instance', got %s", client.instanceID)
	}
	if client.store == nil {
		t.Error("Expected store to be initialized")
	}
}

func TestPollMessages(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	client, err := NewClient(dbPath, "agent1")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Create a thread and message
	thread, _ := client.store.CreateThread("Test Thread", "human", "user")
	_, _ = client.store.CreateMessage(thread.ID, "human", "user", "ailang_instance", "agent1", "directive", "Do something")

	// Poll for messages
	messages, err := client.PollMessages()
	if err != nil {
		t.Fatalf("PollMessages failed: %v", err)
	}

	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}
	if messages[0].Content != "Do something" {
		t.Errorf("Expected content 'Do something', got %s", messages[0].Content)
	}
}

func TestPublishMessage(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	client, err := NewClient(dbPath, "agent1")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Create a thread
	thread, _ := client.store.CreateThread("Test Thread", "ailang_instance", "agent1")

	// Publish a message
	msg, err := client.PublishMessage(thread.ID, "human", "user", "status", "Working on it")
	if err != nil {
		t.Fatalf("PublishMessage failed: %v", err)
	}

	if msg.Content != "Working on it" {
		t.Errorf("Expected content 'Working on it', got %s", msg.Content)
	}
	if msg.Kind != "status" {
		t.Errorf("Expected kind 'status', got %s", msg.Kind)
	}
}

func TestSendStatus(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	client, err := NewClient(dbPath, "agent1")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	thread, _ := client.store.CreateThread("Test Thread", "ailang_instance", "agent1")

	msg, err := client.SendStatus(thread.ID, "Processing request")
	if err != nil {
		t.Fatalf("SendStatus failed: %v", err)
	}

	if msg.Kind != "status" {
		t.Errorf("Expected kind 'status', got %s", msg.Kind)
	}
	if msg.ToType != "human" {
		t.Errorf("Expected to_type 'human', got %s", msg.ToType)
	}
}

func TestSendQuestion(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	client, err := NewClient(dbPath, "agent1")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	thread, _ := client.store.CreateThread("Test Thread", "ailang_instance", "agent1")

	msg, err := client.SendQuestion(thread.ID, "Should I proceed?")
	if err != nil {
		t.Fatalf("SendQuestion failed: %v", err)
	}

	if msg.Kind != "question" {
		t.Errorf("Expected kind 'question', got %s", msg.Kind)
	}
}

func TestSendResult(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	client, err := NewClient(dbPath, "agent1")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	thread, _ := client.store.CreateThread("Test Thread", "ailang_instance", "agent1")

	msg, err := client.SendResult(thread.ID, "Task completed successfully")
	if err != nil {
		t.Fatalf("SendResult failed: %v", err)
	}

	if msg.Kind != "result" {
		t.Errorf("Expected kind 'result', got %s", msg.Kind)
	}
}

func TestAcknowledgeMessage(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	client, err := NewClient(dbPath, "agent1")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Create message
	thread, _ := client.store.CreateThread("Test Thread", "human", "user")
	msg, _ := client.store.CreateMessage(thread.ID, "human", "user", "ailang_instance", "agent1", "directive", "Test")

	// Acknowledge it
	err = client.AcknowledgeMessage(msg.ID)
	if err != nil {
		t.Fatalf("AcknowledgeMessage failed: %v", err)
	}

	// Verify it's not in pending messages anymore
	messages, _ := client.PollMessages()
	if len(messages) != 0 {
		t.Errorf("Expected 0 pending messages after ack, got %d", len(messages))
	}
}

func TestRequestApproval(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	client, err := NewClient(dbPath, "agent1")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	thread, _ := client.store.CreateThread("Test Thread", "ailang_instance", "agent1")

	effectDelta := &EffectDelta{
		CapType:     "FS",
		Paths:       []string{"src/"},
		BudgetDelta: 0.5,
	}

	approvalID, err := client.RequestApproval(thread.ID, effectDelta, "Read source files", "low", 0.5)
	if err != nil {
		t.Fatalf("RequestApproval failed: %v", err)
	}

	if approvalID == "" {
		t.Error("Expected non-empty approval ID")
	}

	// Verify approval was created
	approval, err := client.store.GetApproval(approvalID)
	if err != nil {
		t.Fatalf("Failed to get approval: %v", err)
	}

	if approval.Status != "pending" {
		t.Errorf("Expected status 'pending', got %s", approval.Status)
	}
}

func TestCheckApprovalStatus(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	client, err := NewClient(dbPath, "agent1")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	thread, _ := client.store.CreateThread("Test Thread", "ailang_instance", "agent1")
	effectDelta := &EffectDelta{CapType: "IO", Paths: []string{}, BudgetDelta: 1.0}
	approvalID, _ := client.RequestApproval(thread.ID, effectDelta, "Test", "low", 1.0)

	// Check initial status
	status, err := client.CheckApprovalStatus(approvalID)
	if err != nil {
		t.Fatalf("CheckApprovalStatus failed: %v", err)
	}

	if status != "pending" {
		t.Errorf("Expected status 'pending', got %s", status)
	}

	// Approve it
	_ = client.store.ApproveApproval(approvalID, "user1", "OK", 24*time.Hour)

	// Check approved status
	status, err = client.CheckApprovalStatus(approvalID)
	if err != nil {
		t.Fatalf("CheckApprovalStatus failed: %v", err)
	}

	if status != "approved" {
		t.Errorf("Expected status 'approved', got %s", status)
	}
}

func TestWaitForApprovalApproved(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	client, err := NewClient(dbPath, "agent1")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	thread, _ := client.store.CreateThread("Test Thread", "ailang_instance", "agent1")
	effectDelta := &EffectDelta{CapType: "FS", Paths: []string{}, BudgetDelta: 0.5}
	approvalID, _ := client.RequestApproval(thread.ID, effectDelta, "Test", "low", 0.5)

	// Approve in background after short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		client.store.ApproveApproval(approvalID, "user1", "OK", 24*time.Hour)
	}()

	// Wait for approval
	approved, err := client.WaitForApproval(approvalID, 2*time.Second)
	if err != nil {
		t.Fatalf("WaitForApproval failed: %v", err)
	}

	if !approved {
		t.Error("Expected approval to be approved")
	}
}

func TestWaitForApprovalRejected(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	client, err := NewClient(dbPath, "agent1")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	thread, _ := client.store.CreateThread("Test Thread", "ailang_instance", "agent1")
	effectDelta := &EffectDelta{CapType: "Net", Paths: []string{}, BudgetDelta: 5.0}
	approvalID, _ := client.RequestApproval(thread.ID, effectDelta, "Expensive", "high", 5.0)

	// Reject in background after short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		client.store.RejectApproval(approvalID, "user1", "Too expensive")
	}()

	// Wait for approval
	approved, err := client.WaitForApproval(approvalID, 2*time.Second)
	if err != nil {
		t.Fatalf("WaitForApproval failed: %v", err)
	}

	if approved {
		t.Error("Expected approval to be rejected")
	}
}

func TestWaitForApprovalTimeout(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	client, err := NewClient(dbPath, "agent1")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	thread, _ := client.store.CreateThread("Test Thread", "ailang_instance", "agent1")
	effectDelta := &EffectDelta{CapType: "FS", Paths: []string{}, BudgetDelta: 0.5}
	approvalID, _ := client.RequestApproval(thread.ID, effectDelta, "Test", "low", 0.5)

	// Don't approve - let it timeout
	_, err = client.WaitForApproval(approvalID, 500*time.Millisecond)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}

func TestGetCapabilityToken(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	client, err := NewClient(dbPath, "agent1")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	thread, _ := client.store.CreateThread("Test Thread", "ailang_instance", "agent1")
	effectDelta := &EffectDelta{CapType: "FS", Paths: []string{"src/"}, BudgetDelta: 0.5}
	approvalID, _ := client.RequestApproval(thread.ID, effectDelta, "Test", "low", 0.5)

	// Approve it
	_ = client.store.ApproveApproval(approvalID, "user1", "OK", 24*time.Hour)

	// Get token
	token, err := client.GetCapabilityToken(approvalID)
	if err != nil {
		t.Fatalf("GetCapabilityToken failed: %v", err)
	}

	if token == "" {
		t.Error("Expected non-empty capability token")
	}
}

func TestGetCapabilityTokenNotApproved(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	client, err := NewClient(dbPath, "agent1")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	thread, _ := client.store.CreateThread("Test Thread", "ailang_instance", "agent1")
	effectDelta := &EffectDelta{CapType: "FS", Paths: []string{}, BudgetDelta: 0.5}
	approvalID, _ := client.RequestApproval(thread.ID, effectDelta, "Test", "low", 0.5)

	// Don't approve - try to get token
	_, err = client.GetCapabilityToken(approvalID)
	if err == nil {
		t.Error("Expected error when getting token for non-approved request")
	}
}

func TestSubscribeToThread(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	client, err := NewClient(dbPath, "agent1")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	thread, _ := client.store.CreateThread("Test Thread", "ailang_instance", "agent1")

	err = client.SubscribeToThread(thread.ID)
	if err != nil {
		t.Fatalf("SubscribeToThread failed: %v", err)
	}

	// Verify subscription exists (we can't easily check this without exposing internals,
	// but the test passes if no error occurs)
}

func TestClientUpdateAckSeq(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	client, err := NewClient(dbPath, "agent1")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	thread, _ := client.store.CreateThread("Test Thread", "ailang_instance", "agent1")
	_ = client.SubscribeToThread(thread.ID)

	err = client.UpdateAckSeq(thread.ID, 5)
	if err != nil {
		t.Fatalf("UpdateAckSeq failed: %v", err)
	}
}

func TestClientGetThread(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	client, err := NewClient(dbPath, "agent1")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	created, _ := client.store.CreateThread("Test Thread", "ailang_instance", "agent1")

	thread, err := client.GetThread(created.ID)
	if err != nil {
		t.Fatalf("GetThread failed: %v", err)
	}

	if thread.Title != "Test Thread" {
		t.Errorf("Expected title 'Test Thread', got %s", thread.Title)
	}
}

func TestStartStopPolling(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	client, err := NewClient(dbPath, "agent1")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Create a thread and message
	thread, _ := client.store.CreateThread("Test Thread", "human", "user")

	callCount := 0
	callback := func(messages []*Message) error {
		callCount++
		return nil
	}

	// Start polling every 100ms
	client.StartPolling(100*time.Millisecond, callback)

	// Create a message
	_, _ = client.store.CreateMessage(thread.ID, "human", "user", "ailang_instance", "agent1", "directive", "Test")

	// Wait for polling to trigger
	time.Sleep(250 * time.Millisecond)

	// Stop polling
	client.StopPolling()

	if callCount == 0 {
		t.Error("Expected callback to be called at least once")
	}
}
