package messaging

import (
	"os"
	"testing"
)

func TestRecordApprovalHistory(t *testing.T) {
	dbPath := t.TempDir() + "/test_history.db"
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer func() {
		store.Close()
		os.Remove(dbPath)
	}()

	// Record approval history
	cost := 0.05
	err = store.RecordApprovalHistory(
		"appr_123", "thread_456", "agent_789",
		"created", "system",
		"Create test file", "low",
		&cost, "",
	)
	if err != nil {
		t.Fatalf("Failed to record approval history: %v", err)
	}

	// Retrieve history
	entries, err := store.GetApprovalHistory("", 10)
	if err != nil {
		t.Fatalf("Failed to get approval history: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.ApprovalID != "appr_123" {
		t.Errorf("Expected approval_id 'appr_123', got '%s'", entry.ApprovalID)
	}
	if entry.ThreadID != "thread_456" {
		t.Errorf("Expected thread_id 'thread_456', got '%s'", entry.ThreadID)
	}
	if entry.AgentID != "agent_789" {
		t.Errorf("Expected agent_id 'agent_789', got '%s'", entry.AgentID)
	}
	if entry.Action != "created" {
		t.Errorf("Expected action 'created', got '%s'", entry.Action)
	}
	if entry.Actor != "system" {
		t.Errorf("Expected actor 'system', got '%s'", entry.Actor)
	}
	if entry.Proposal != "Create test file" {
		t.Errorf("Expected proposal 'Create test file', got '%s'", entry.Proposal)
	}
	if entry.Impact != "low" {
		t.Errorf("Expected impact 'low', got '%s'", entry.Impact)
	}
	if entry.EstimatedCost == nil || *entry.EstimatedCost != 0.05 {
		t.Errorf("Expected estimated_cost 0.05, got %v", entry.EstimatedCost)
	}
}

func TestGetApprovalHistoryFiltered(t *testing.T) {
	dbPath := t.TempDir() + "/test_history_filter.db"
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer func() {
		store.Close()
		os.Remove(dbPath)
	}()

	// Record multiple entries for different threads
	cost := 0.01
	_ = store.RecordApprovalHistory("appr_1", "thread_A", "agent_1", "created", "system", "", "", &cost, "")
	_ = store.RecordApprovalHistory("appr_2", "thread_B", "agent_1", "created", "system", "", "", &cost, "")
	_ = store.RecordApprovalHistory("appr_3", "thread_A", "agent_1", "approved", "user", "", "", nil, "token_123")

	// Get all entries
	allEntries, err := store.GetApprovalHistory("", 10)
	if err != nil {
		t.Fatalf("Failed to get all approval history: %v", err)
	}
	if len(allEntries) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(allEntries))
	}

	// Get filtered entries for thread_A
	filteredEntries, err := store.GetApprovalHistory("thread_A", 10)
	if err != nil {
		t.Fatalf("Failed to get filtered approval history: %v", err)
	}
	if len(filteredEntries) != 2 {
		t.Errorf("Expected 2 entries for thread_A, got %d", len(filteredEntries))
	}

	// Verify all filtered entries are for thread_A
	for _, entry := range filteredEntries {
		if entry.ThreadID != "thread_A" {
			t.Errorf("Expected thread_id 'thread_A', got '%s'", entry.ThreadID)
		}
	}
}

func TestRecordInstanceHistory(t *testing.T) {
	dbPath := t.TempDir() + "/test_instance.db"
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer func() {
		store.Close()
		os.Remove(dbPath)
	}()

	// Record instance start
	err = store.RecordInstanceStart("agent_test", "instance_123")
	if err != nil {
		t.Fatalf("Failed to record instance start: %v", err)
	}

	// Verify instance is recorded
	entries, err := store.GetInstanceHistory("agent_test", 10)
	if err != nil {
		t.Fatalf("Failed to get instance history: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.AgentID != "agent_test" {
		t.Errorf("Expected agent_id 'agent_test', got '%s'", entry.AgentID)
	}
	if entry.InstanceID != "instance_123" {
		t.Errorf("Expected instance_id 'instance_123', got '%s'", entry.InstanceID)
	}
	if entry.EndedAt != nil {
		t.Errorf("Expected ended_at to be nil for running instance")
	}
	if entry.ExitCode != nil {
		t.Errorf("Expected exit_code to be nil for running instance")
	}

	// Record instance end
	err = store.RecordInstanceEnd("instance_123", 0, 5000, 150, 3)
	if err != nil {
		t.Fatalf("Failed to record instance end: %v", err)
	}

	// Verify instance is updated
	entries, err = store.GetInstanceHistory("agent_test", 10)
	if err != nil {
		t.Fatalf("Failed to get updated instance history: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	entry = entries[0]
	if entry.EndedAt == nil {
		t.Errorf("Expected ended_at to be set")
	}
	if entry.ExitCode == nil || *entry.ExitCode != 0 {
		t.Errorf("Expected exit_code 0, got %v", entry.ExitCode)
	}
	if entry.TotalTokens != 5000 {
		t.Errorf("Expected total_tokens 5000, got %d", entry.TotalTokens)
	}
	if entry.TotalCostCent != 150 {
		t.Errorf("Expected total_cost_cents 150, got %d", entry.TotalCostCent)
	}
	if entry.ThreadCount != 3 {
		t.Errorf("Expected thread_count 3, got %d", entry.ThreadCount)
	}
}

func TestGetInstanceHistoryFiltered(t *testing.T) {
	dbPath := t.TempDir() + "/test_instance_filter.db"
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer func() {
		store.Close()
		os.Remove(dbPath)
	}()

	// Record multiple instances for different agents
	_ = store.RecordInstanceStart("agent_A", "instance_1")
	_ = store.RecordInstanceStart("agent_B", "instance_2")
	_ = store.RecordInstanceStart("agent_A", "instance_3")

	// Get all entries
	allEntries, err := store.GetInstanceHistory("", 10)
	if err != nil {
		t.Fatalf("Failed to get all instance history: %v", err)
	}
	if len(allEntries) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(allEntries))
	}

	// Get filtered entries for agent_A
	filteredEntries, err := store.GetInstanceHistory("agent_A", 10)
	if err != nil {
		t.Fatalf("Failed to get filtered instance history: %v", err)
	}
	if len(filteredEntries) != 2 {
		t.Errorf("Expected 2 entries for agent_A, got %d", len(filteredEntries))
	}

	// Verify all filtered entries are for agent_A
	for _, entry := range filteredEntries {
		if entry.AgentID != "agent_A" {
			t.Errorf("Expected agent_id 'agent_A', got '%s'", entry.AgentID)
		}
	}
}

func TestCleanupOldHistory(t *testing.T) {
	dbPath := t.TempDir() + "/test_cleanup.db"
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer func() {
		store.Close()
		os.Remove(dbPath)
	}()

	// Record some history
	cost := 0.01
	_ = store.RecordApprovalHistory("appr_1", "thread_1", "agent_1", "created", "system", "", "", &cost, "")
	_ = store.RecordInstanceStart("agent_1", "instance_1")

	// Cleanup with 30 retention days should keep recent entries (created today)
	approvalDeleted, instanceDeleted, err := store.CleanupOldHistory(30)
	if err != nil {
		t.Fatalf("Failed to cleanup history: %v", err)
	}

	// With 30 retention days, entries created today should NOT be deleted
	if approvalDeleted != 0 || instanceDeleted != 0 {
		t.Errorf("Expected 0 deletions with 30 retention days, got %d approvals, %d instances", approvalDeleted, instanceDeleted)
	}

	// Verify entries still exist
	approvals, _ := store.GetApprovalHistory("", 10)
	instances, _ := store.GetInstanceHistory("", 10)
	if len(approvals) != 1 || len(instances) != 1 {
		t.Errorf("Expected 1 approval and 1 instance after cleanup, got %d and %d", len(approvals), len(instances))
	}
}
