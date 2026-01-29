package observatory

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) (*sql.DB, *Store) {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}

	err = Migrate(db)
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db, NewStore(db)
}

func TestStore_CreateChain(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create a chain
	chain, err := store.CreateChain(ctx, &ChainCreateRequest{
		SourceType:        ChainSourceGitHubIssue,
		SourceRef:         "#123",
		GitHubRepo:        "sunholo-data/ailang",
		GitHubIssueNumber: 123,
		WorkspacePath:     "/path/to/workspace",
	})
	if err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}

	if chain.ID == "" {
		t.Error("chain ID should be generated")
	}
	if chain.Status != ChainStatusActive {
		t.Errorf("expected status %s, got %s", ChainStatusActive, chain.Status)
	}
	if chain.SourceType != ChainSourceGitHubIssue {
		t.Errorf("expected source type %s, got %s", ChainSourceGitHubIssue, chain.SourceType)
	}
	if chain.GitHubIssueNumber != 123 {
		t.Errorf("expected issue number 123, got %d", chain.GitHubIssueNumber)
	}
}

func TestStore_CreateChain_RequiresSourceType(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	_, err := store.CreateChain(ctx, &ChainCreateRequest{
		SourceRef: "#123",
	})
	if err == nil {
		t.Error("expected error for missing source_type")
	}
}

func TestStore_GetChain(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create a chain
	created, _ := store.CreateChain(ctx, &ChainCreateRequest{
		SourceType: ChainSourceMessage,
		SourceRef:  "msg-123",
	})

	// Get the chain without stages
	chain, err := store.GetChain(ctx, created.ID, ChainReadOptions{})
	if err != nil {
		t.Fatalf("failed to get chain: %v", err)
	}

	if chain == nil {
		t.Fatal("chain should not be nil")
	}
	if chain.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, chain.ID)
	}
	if chain.SourceRef != "msg-123" {
		t.Errorf("expected source_ref msg-123, got %s", chain.SourceRef)
	}
}

func TestStore_GetChain_NotFound(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	chain, err := store.GetChain(ctx, "nonexistent", ChainReadOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if chain != nil {
		t.Error("expected nil for nonexistent chain")
	}
}

func TestStore_ListChains(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create multiple chains
	store.CreateChain(ctx, &ChainCreateRequest{
		SourceType: ChainSourceGitHubIssue,
		SourceRef:  "#1",
	})
	store.CreateChain(ctx, &ChainCreateRequest{
		SourceType: ChainSourceMessage,
		SourceRef:  "msg-1",
	})
	store.CreateChain(ctx, &ChainCreateRequest{
		SourceType: ChainSourceGitHubIssue,
		SourceRef:  "#2",
	})

	// List all chains
	chains, err := store.ListChains(ctx, ChainListOptions{})
	if err != nil {
		t.Fatalf("failed to list chains: %v", err)
	}
	if len(chains) != 3 {
		t.Errorf("expected 3 chains, got %d", len(chains))
	}

	// Filter by source type
	ghChains, err := store.ListChains(ctx, ChainListOptions{
		SourceType: "github_issue",
	})
	if err != nil {
		t.Fatalf("failed to list filtered chains: %v", err)
	}
	if len(ghChains) != 2 {
		t.Errorf("expected 2 github_issue chains, got %d", len(ghChains))
	}
}

func TestStore_UpdateChainStatus(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	chain, _ := store.CreateChain(ctx, &ChainCreateRequest{
		SourceType: ChainSourceManual,
	})

	// Update status to completed
	err := store.UpdateChainStatus(ctx, chain.ID, ChainStatusCompleted)
	if err != nil {
		t.Fatalf("failed to update status: %v", err)
	}

	// Verify status changed
	updated, _ := store.GetChain(ctx, chain.ID, ChainReadOptions{})
	if updated.Status != ChainStatusCompleted {
		t.Errorf("expected status %s, got %s", ChainStatusCompleted, updated.Status)
	}
	if updated.CompletedAt == nil {
		t.Error("completed_at should be set when status is completed")
	}
}

func TestStore_DeleteChain(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	chain, _ := store.CreateChain(ctx, &ChainCreateRequest{
		SourceType: ChainSourceManual,
	})

	// Delete the chain
	err := store.DeleteChain(ctx, chain.ID)
	if err != nil {
		t.Fatalf("failed to delete chain: %v", err)
	}

	// Verify chain is gone
	deleted, _ := store.GetChain(ctx, chain.ID, ChainReadOptions{})
	if deleted != nil {
		t.Error("chain should be deleted")
	}
}

func TestStore_CreateStage(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	chain, _ := store.CreateChain(ctx, &ChainCreateRequest{
		SourceType: ChainSourceGitHubIssue,
		SourceRef:  "#123",
	})

	// Create first stage
	stage1, err := store.CreateStage(ctx, &StageCreateRequest{
		ChainID:   chain.ID,
		AgentID:   "design-doc-creator",
		Provider:  ProviderClaude,
		MessageID: "msg-123",
	})
	if err != nil {
		t.Fatalf("failed to create stage: %v", err)
	}

	if stage1.StageNumber != 1 {
		t.Errorf("expected stage_number 1, got %d", stage1.StageNumber)
	}
	if stage1.Status != StageStatusPending {
		t.Errorf("expected status %s, got %s", StageStatusPending, stage1.Status)
	}
	if stage1.Iteration != 1 {
		t.Errorf("expected iteration 1, got %d", stage1.Iteration)
	}

	// Create second stage
	stage2, err := store.CreateStage(ctx, &StageCreateRequest{
		ChainID: chain.ID,
		AgentID: "sprint-planner",
	})
	if err != nil {
		t.Fatalf("failed to create second stage: %v", err)
	}

	if stage2.StageNumber != 2 {
		t.Errorf("expected stage_number 2, got %d", stage2.StageNumber)
	}

	// Verify chain's current_stage updated
	updatedChain, _ := store.GetChain(ctx, chain.ID, ChainReadOptions{})
	if updatedChain.CurrentStage != 2 {
		t.Errorf("expected chain.current_stage 2, got %d", updatedChain.CurrentStage)
	}
}

func TestStore_GetChainStages(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	chain, _ := store.CreateChain(ctx, &ChainCreateRequest{
		SourceType: ChainSourceGitHubIssue,
	})

	store.CreateStage(ctx, &StageCreateRequest{ChainID: chain.ID, AgentID: "agent-1"})
	store.CreateStage(ctx, &StageCreateRequest{ChainID: chain.ID, AgentID: "agent-2"})
	store.CreateStage(ctx, &StageCreateRequest{ChainID: chain.ID, AgentID: "agent-3"})

	stages, err := store.GetChainStages(ctx, chain.ID, ChainReadOptions{})
	if err != nil {
		t.Fatalf("failed to get stages: %v", err)
	}

	if len(stages) != 3 {
		t.Errorf("expected 3 stages, got %d", len(stages))
	}

	// Verify ordering
	for i, stage := range stages {
		if stage.StageNumber != i+1 {
			t.Errorf("stage %d: expected stage_number %d, got %d", i, i+1, stage.StageNumber)
		}
	}
}

func TestStore_UpdateStageStatus(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	chain, _ := store.CreateChain(ctx, &ChainCreateRequest{
		SourceType: ChainSourceManual,
	})
	stage, _ := store.CreateStage(ctx, &StageCreateRequest{
		ChainID: chain.ID,
		AgentID: "test-agent",
	})

	// Update to running
	err := store.UpdateStageStatus(ctx, stage.ID, StageStatusRunning)
	if err != nil {
		t.Fatalf("failed to update status to running: %v", err)
	}

	updated, _ := store.GetStage(ctx, stage.ID)
	if updated.Status != StageStatusRunning {
		t.Errorf("expected status %s, got %s", StageStatusRunning, updated.Status)
	}
	if updated.StartedAt == nil {
		t.Error("started_at should be set when status is running")
	}

	// Update to completed
	err = store.UpdateStageStatus(ctx, stage.ID, StageStatusCompleted)
	if err != nil {
		t.Fatalf("failed to update status to completed: %v", err)
	}

	completed, _ := store.GetStage(ctx, stage.ID)
	if completed.Status != StageStatusCompleted {
		t.Errorf("expected status %s, got %s", StageStatusCompleted, completed.Status)
	}
	if completed.CompletedAt == nil {
		t.Error("completed_at should be set when status is completed")
	}
}

func TestStore_UpdateStageApproval(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	chain, _ := store.CreateChain(ctx, &ChainCreateRequest{
		SourceType: ChainSourceGitHubIssue,
	})
	stage, _ := store.CreateStage(ctx, &StageCreateRequest{
		ChainID: chain.ID,
		AgentID: "design-doc-creator",
	})

	// Set approval to pending
	err := store.UpdateStageApproval(ctx, stage.ID, ApprovalStatusPending, ApprovalTypeMergeHandoff, "")
	if err != nil {
		t.Fatalf("failed to set pending approval: %v", err)
	}

	pending, _ := store.GetStage(ctx, stage.ID)
	if pending.ApprovalStatus != ApprovalStatusPending {
		t.Errorf("expected approval_status %s, got %s", ApprovalStatusPending, pending.ApprovalStatus)
	}
	if pending.ApprovalType != ApprovalTypeMergeHandoff {
		t.Errorf("expected approval_type %s, got %s", ApprovalTypeMergeHandoff, pending.ApprovalType)
	}
	if pending.Status != StageStatusAwaitingApproval {
		t.Errorf("expected status %s, got %s", StageStatusAwaitingApproval, pending.Status)
	}

	// Approve
	err = store.UpdateStageApproval(ctx, stage.ID, ApprovalStatusApproved, ApprovalTypeMergeHandoff, "")
	if err != nil {
		t.Fatalf("failed to approve: %v", err)
	}

	approved, _ := store.GetStage(ctx, stage.ID)
	if approved.ApprovalStatus != ApprovalStatusApproved {
		t.Errorf("expected approval_status %s, got %s", ApprovalStatusApproved, approved.ApprovalStatus)
	}
	if approved.Status != StageStatusCompleted {
		t.Errorf("expected status %s, got %s", StageStatusCompleted, approved.Status)
	}
}

func TestStore_UpdateStageApproval_Rejected(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	chain, _ := store.CreateChain(ctx, &ChainCreateRequest{
		SourceType: ChainSourceGitHubIssue,
	})
	stage, _ := store.CreateStage(ctx, &StageCreateRequest{
		ChainID: chain.ID,
		AgentID: "sprint-executor",
	})

	// Reject with feedback
	err := store.UpdateStageApproval(ctx, stage.ID, ApprovalStatusRejected, ApprovalTypeMerge, "Need more tests")
	if err != nil {
		t.Fatalf("failed to reject: %v", err)
	}

	rejected, _ := store.GetStage(ctx, stage.ID)
	if rejected.ApprovalStatus != ApprovalStatusRejected {
		t.Errorf("expected approval_status %s, got %s", ApprovalStatusRejected, rejected.ApprovalStatus)
	}
	if rejected.HumanFeedback != "Need more tests" {
		t.Errorf("expected feedback 'Need more tests', got %s", rejected.HumanFeedback)
	}
	if rejected.Status != StageStatusFailed {
		t.Errorf("expected status %s, got %s", StageStatusFailed, rejected.Status)
	}
}

func TestStore_GetChainByTaskID(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	chain, _ := store.CreateChain(ctx, &ChainCreateRequest{
		SourceType: ChainSourceGitHubIssue,
	})
	store.CreateStage(ctx, &StageCreateRequest{
		ChainID: chain.ID,
		AgentID: "test-agent",
		TaskID:  "task-12345678",
	})

	// Find chain by task ID
	found, err := store.GetChainByTaskID(ctx, "task-12345678")
	if err != nil {
		t.Fatalf("failed to find chain: %v", err)
	}
	if found == nil {
		t.Fatal("chain should be found")
	}
	if found.ID != chain.ID {
		t.Errorf("expected chain ID %s, got %s", chain.ID, found.ID)
	}
}

func TestStore_GetChainByGitHubIssue(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	chain, _ := store.CreateChain(ctx, &ChainCreateRequest{
		SourceType:        ChainSourceGitHubIssue,
		GitHubRepo:        "sunholo-data/ailang",
		GitHubIssueNumber: 456,
	})

	// Find chain by GitHub issue
	found, err := store.GetChainByGitHubIssue(ctx, "sunholo-data/ailang", 456)
	if err != nil {
		t.Fatalf("failed to find chain: %v", err)
	}
	if found == nil {
		t.Fatal("chain should be found")
	}
	if found.ID != chain.ID {
		t.Errorf("expected chain ID %s, got %s", chain.ID, found.ID)
	}
}

func TestStore_ListPendingApprovals(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create chain with stages needing approval
	chain, _ := store.CreateChain(ctx, &ChainCreateRequest{
		SourceType: ChainSourceGitHubIssue,
		SourceRef:  "#789",
	})

	stage1, _ := store.CreateStage(ctx, &StageCreateRequest{
		ChainID: chain.ID,
		AgentID: "design-doc-creator",
	})
	store.UpdateStageStatus(ctx, stage1.ID, StageStatusRunning)
	store.UpdateStageApproval(ctx, stage1.ID, ApprovalStatusPending, ApprovalTypeMergeHandoff, "")

	stage2, _ := store.CreateStage(ctx, &StageCreateRequest{
		ChainID: chain.ID,
		AgentID: "sprint-planner",
	})
	store.UpdateStageStatus(ctx, stage2.ID, StageStatusRunning)
	store.UpdateStageApproval(ctx, stage2.ID, ApprovalStatusPending, ApprovalTypeMerge, "")

	// List pending approvals
	approvals, err := store.ListPendingApprovals(ctx, 10)
	if err != nil {
		t.Fatalf("failed to list pending approvals: %v", err)
	}
	if len(approvals) != 2 {
		t.Errorf("expected 2 pending approvals, got %d", len(approvals))
	}
}

func TestStore_GetChainStats(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create chains with different statuses
	chain1, _ := store.CreateChain(ctx, &ChainCreateRequest{SourceType: ChainSourceManual})
	store.UpdateChainStatus(ctx, chain1.ID, ChainStatusCompleted)

	store.CreateChain(ctx, &ChainCreateRequest{SourceType: ChainSourceManual})

	chain3, _ := store.CreateChain(ctx, &ChainCreateRequest{SourceType: ChainSourceManual})
	store.UpdateChainStatus(ctx, chain3.ID, ChainStatusFailed)

	stats, err := store.GetChainStats(ctx)
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}

	if stats.TotalChains != 3 {
		t.Errorf("expected 3 total chains, got %d", stats.TotalChains)
	}
	if stats.CompletedChains != 1 {
		t.Errorf("expected 1 completed chain, got %d", stats.CompletedChains)
	}
	if stats.ActiveChains != 1 {
		t.Errorf("expected 1 active chain, got %d", stats.ActiveChains)
	}
	if stats.FailedChains != 1 {
		t.Errorf("expected 1 failed chain, got %d", stats.FailedChains)
	}
}

func TestStore_CascadeDeleteStages(t *testing.T) {
	db, store := setupTestDB(t)
	defer db.Close()

	// Enable foreign keys
	db.Exec("PRAGMA foreign_keys = ON")

	ctx := context.Background()

	chain, _ := store.CreateChain(ctx, &ChainCreateRequest{
		SourceType: ChainSourceManual,
	})
	store.CreateStage(ctx, &StageCreateRequest{ChainID: chain.ID, AgentID: "agent-1"})
	store.CreateStage(ctx, &StageCreateRequest{ChainID: chain.ID, AgentID: "agent-2"})

	// Verify stages exist
	stages, _ := store.GetChainStages(ctx, chain.ID, ChainReadOptions{})
	if len(stages) != 2 {
		t.Fatalf("expected 2 stages before delete, got %d", len(stages))
	}

	// Delete chain - stages should be cascade deleted
	err := store.DeleteChain(ctx, chain.ID)
	if err != nil {
		t.Fatalf("failed to delete chain: %v", err)
	}

	// Verify stages are gone
	stagesAfter, _ := store.GetChainStages(ctx, chain.ID, ChainReadOptions{})
	if len(stagesAfter) != 0 {
		t.Errorf("expected 0 stages after cascade delete, got %d", len(stagesAfter))
	}
}

func TestStore_ChainEnvironmentVars(t *testing.T) {
	chain := &ExecutionChain{ID: "chain-123"}

	envVars := chain.ChainEnvironmentVars("stage-456", "task-789", "msg-abc")

	expected := map[string]string{
		"AILANG_CHAIN_ID":   "chain-123",
		"AILANG_STAGE_ID":   "stage-456",
		"AILANG_TASK_ID":    "task-789",
		"AILANG_MESSAGE_ID": "msg-abc",
	}

	for key, expectedVal := range expected {
		if envVars[key] != expectedVal {
			t.Errorf("expected %s=%s, got %s", key, expectedVal, envVars[key])
		}
	}
}
