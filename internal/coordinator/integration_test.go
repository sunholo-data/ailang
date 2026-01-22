package coordinator

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// IntegrationMockProvider is a test provider that simulates task execution without external dependencies
// Named differently from MockProvider in provider_test.go to avoid redeclaration
type IntegrationMockProvider struct {
	name       string
	handleFunc func(task *AnalyzedTask) bool
	execFunc   func(ctx context.Context, task *AnalyzedTask, opts *ExecuteOptions) (*ExecuteResult, error)
	execCount  int
}

func NewIntegrationMockProvider(name string) *IntegrationMockProvider {
	return &IntegrationMockProvider{
		name: name,
		handleFunc: func(task *AnalyzedTask) bool {
			return true // Handle all tasks by default
		},
		execFunc: func(ctx context.Context, task *AnalyzedTask, opts *ExecuteOptions) (*ExecuteResult, error) {
			return &ExecuteResult{
				Success:      true,
				Output:       "Mock execution completed successfully",
				Duration:     100 * time.Millisecond,
				Cost:         0.001,
				TokensUsed:   100,
				InputTokens:  50,
				OutputTokens: 50,
				Provider:     name,
			}, nil
		},
	}
}

func (m *IntegrationMockProvider) Name() string {
	return m.name
}

func (m *IntegrationMockProvider) CanHandle(task *AnalyzedTask) bool {
	return m.handleFunc(task)
}

func (m *IntegrationMockProvider) Execute(ctx context.Context, task *AnalyzedTask, opts *ExecuteOptions) (*ExecuteResult, error) {
	m.execCount++
	return m.execFunc(ctx, task, opts)
}

func (m *IntegrationMockProvider) SetExecuteFunc(fn func(ctx context.Context, task *AnalyzedTask, opts *ExecuteOptions) (*ExecuteResult, error)) {
	m.execFunc = fn
}

func (m *IntegrationMockProvider) SetHandleFunc(fn func(task *AnalyzedTask) bool) {
	m.handleFunc = fn
}

func (m *IntegrationMockProvider) ExecCount() int {
	return m.execCount
}

// TestIntegration_TaskLifecycle tests the full task lifecycle:
// create task -> mark running -> mark completed
func TestIntegration_TaskLifecycle(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	ctx := context.Background()

	// Create a task
	task := &TaskRecord{
		ID:        "int-task-1",
		MessageID: "msg-int-1",
		Title:     "Integration Test Task",
		Content:   "Fix the integration test bug",
		Type:      TaskTypeBugFix,
		Priority:  3,
		Status:    TaskStatusPending,
		CreatedAt: time.Now(),
	}

	// Create
	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Verify pending status
	retrieved, err := store.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}
	if retrieved.Status != TaskStatusPending {
		t.Errorf("expected status %s, got %s", TaskStatusPending, retrieved.Status)
	}

	// Mark running
	if err := store.MarkTaskRunning(ctx, task.ID, "mock-provider", "wt-int-1"); err != nil {
		t.Fatalf("failed to mark running: %v", err)
	}

	retrieved, _ = store.GetTask(ctx, task.ID)
	if retrieved.Status != TaskStatusRunning {
		t.Errorf("expected status %s, got %s", TaskStatusRunning, retrieved.Status)
	}
	if retrieved.Provider != "mock-provider" {
		t.Errorf("expected provider mock-provider, got %s", retrieved.Provider)
	}

	// Mark pending approval (simulating successful execution)
	result := &ExecuteResult{
		Success:      true,
		Output:       "Task completed successfully",
		Duration:     5 * time.Second,
		Cost:         0.05,
		TokensUsed:   1000,
		InputTokens:  600,
		OutputTokens: 400,
	}
	if err := store.MarkTaskPendingApproval(ctx, task.ID, "/tmp/worktree/int-task-1", "branch-int-1", "main", "", result); err != nil {
		t.Fatalf("failed to mark pending approval: %v", err)
	}

	retrieved, _ = store.GetTask(ctx, task.ID)
	if retrieved.Status != TaskStatusPendingApproval {
		t.Errorf("expected status %s, got %s", TaskStatusPendingApproval, retrieved.Status)
	}
	if retrieved.WorktreePath != "/tmp/worktree/int-task-1" {
		t.Errorf("expected worktree path /tmp/worktree/int-task-1, got %s", retrieved.WorktreePath)
	}

	// Mark completed (simulating approval)
	if err := store.MarkTaskCompleted(ctx, task.ID, result); err != nil {
		t.Fatalf("failed to mark completed: %v", err)
	}

	retrieved, _ = store.GetTask(ctx, task.ID)
	if retrieved.Status != TaskStatusCompleted {
		t.Errorf("expected status %s, got %s", TaskStatusCompleted, retrieved.Status)
	}
	if retrieved.Cost != 0.05 {
		t.Errorf("expected cost 0.05, got %f", retrieved.Cost)
	}
}

// TestIntegration_TaskExecutorWithMockProvider tests the TaskExecutor with a mock provider
func TestIntegration_TaskExecutorWithMockProvider(t *testing.T) {
	mockProvider := NewIntegrationMockProvider("test-mock")
	executor := NewTaskExecutor(mockProvider)

	ctx := context.Background()

	task := &AnalyzedTask{
		Task: &Task{
			ID:      "exec-task-1",
			Title:   "Test Executor Task",
			Content: "Fix a bug in the test",
		},
		Type: TaskTypeBugFix,
	}

	opts := &ExecuteOptions{
		Timeout:   10 * time.Second,
		Workspace: t.TempDir(),
	}

	result, err := executor.Execute(ctx, task, opts)
	if err != nil {
		t.Fatalf("executor returned error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got failure: %s", result.Error)
	}

	if mockProvider.ExecCount() != 1 {
		t.Errorf("expected 1 execution, got %d", mockProvider.ExecCount())
	}

	if result.Provider != "test-mock" {
		t.Errorf("expected provider test-mock, got %s", result.Provider)
	}
}

// TestIntegration_TaskExecutorWithRetry tests retry behavior
func TestIntegration_TaskExecutorWithRetry(t *testing.T) {
	attemptCount := 0
	mockProvider := NewIntegrationMockProvider("retry-mock")
	mockProvider.SetExecuteFunc(func(ctx context.Context, task *AnalyzedTask, opts *ExecuteOptions) (*ExecuteResult, error) {
		attemptCount++
		if attemptCount < 3 {
			return &ExecuteResult{
				Success: false,
				Error:   "rate limit exceeded - 429",
			}, nil
		}
		return &ExecuteResult{
			Success:  true,
			Output:   "Success after retries",
			Provider: "retry-mock",
		}, nil
	})

	executor := NewTaskExecutor(mockProvider)
	ctx := context.Background()

	task := &AnalyzedTask{
		Task: &Task{
			ID:      "retry-task-1",
			Title:   "Retry Test Task",
			Content: "Test retry logic",
		},
		Type: TaskTypeBugFix,
	}

	opts := &ExecuteOptions{
		Timeout:   30 * time.Second,
		Workspace: t.TempDir(),
	}

	result, err := executor.ExecuteWithRetry(ctx, task, opts, 3)
	if err != nil {
		t.Fatalf("executor returned error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success after retries, got failure: %s", result.Error)
	}

	if attemptCount != 3 {
		t.Errorf("expected 3 attempts, got %d", attemptCount)
	}
}

// TestIntegration_TaskAnalyzerClassification tests task analysis and classification
func TestIntegration_TaskAnalyzerClassification(t *testing.T) {
	analyzer := NewTaskAnalyzer(0.8)

	tests := []struct {
		name         string
		content      string
		expectedType TaskType
	}{
		{
			name:         "bug fix detection",
			content:      "Fix the null pointer exception in parser.go",
			expectedType: TaskTypeBugFix,
		},
		{
			name:         "feature detection",
			content:      "Add a new command to list all tasks",
			expectedType: TaskTypeFeature,
		},
		{
			name:         "docs detection",
			content:      "Update the README with installation instructions",
			expectedType: TaskTypeDocs,
		},
		{
			name:         "test detection",
			content:      "Write unit tests for the coordinator package",
			expectedType: TaskTypeTest,
		},
		{
			name:         "refactor detection",
			content:      "Refactor the executor module for better maintainability",
			expectedType: TaskTypeRefactor,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			task := &Task{
				ID:      "analyze-" + tc.name,
				Title:   tc.name,
				Content: tc.content,
			}

			analyzed := analyzer.Analyze(task)

			if analyzed.Type != tc.expectedType {
				t.Errorf("expected type %s, got %s for content: %q", tc.expectedType, analyzed.Type, tc.content)
			}
		})
	}
}

// TestIntegration_ApprovalCheckpoint tests the approval checkpoint workflow
func TestIntegration_ApprovalCheckpoint(t *testing.T) {
	checkpoint := NewApprovalCheckpoint(1 * time.Hour)

	var callbackCalled atomic.Bool
	checkpoint.SetCallback(func(request *ApprovalRequest) {
		callbackCalled.Store(true)
	})

	// Create approval request
	request := &ApprovalRequest{
		ID:            "apr-test-1",
		TaskID:        "task-test-1",
		Type:          ApprovalTypeMerge,
		Title:         "Test Approval",
		Description:   "Please approve this test",
		SourceAgentID: "coordinator",
		Timeout:       100 * time.Millisecond, // Short timeout for test
	}

	// Submit request with background approval
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	go func() {
		time.Sleep(30 * time.Millisecond)
		_ = checkpoint.Approve(request.ID, "test-user")
	}()

	status, err := checkpoint.RequestApproval(ctx, request)
	if err != nil {
		t.Fatalf("approval failed: %v", err)
	}

	if status != ApprovalStatusApproved {
		t.Errorf("expected status approved, got %s", status)
	}

	if !callbackCalled.Load() {
		t.Error("expected callback to be called")
	}
}

// TestIntegration_ApprovalCheckpointRejection tests rejection workflow
func TestIntegration_ApprovalCheckpointRejection(t *testing.T) {
	checkpoint := NewApprovalCheckpoint(1 * time.Hour)

	request := &ApprovalRequest{
		ID:            "apr-reject-1",
		TaskID:        "task-reject-1",
		Type:          ApprovalTypeMerge,
		Title:         "Test Rejection",
		Description:   "This will be rejected",
		SourceAgentID: "coordinator",
		Timeout:       100 * time.Millisecond, // Short timeout for test
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	go func() {
		time.Sleep(30 * time.Millisecond)
		_ = checkpoint.Reject(request.ID, "test-user")
	}()

	status, err := checkpoint.RequestApproval(ctx, request)
	if err != nil {
		t.Fatalf("rejection wait failed: %v", err)
	}

	if status != ApprovalStatusRejected {
		t.Errorf("expected status rejected, got %s", status)
	}
}

// TestIntegration_AgentRegistry tests agent registration and lookup
func TestIntegration_AgentRegistry(t *testing.T) {
	registry := NewAgentRegistry()

	agent1 := &AgentConfig{
		ID:                  "agent-1",
		Label:               "Test Agent 1",
		Inbox:               "inbox-1",
		Workspace:           "/tmp/workspace1",
		TriggerOnComplete:   []string{"agent-2"},
		AutoApproveHandoffs: true,
	}

	agent2 := &AgentConfig{
		ID:        "agent-2",
		Label:     "Test Agent 2",
		Inbox:     "inbox-2",
		Workspace: "/tmp/workspace2",
	}

	// Register agents
	if err := registry.Register(agent1); err != nil {
		t.Fatalf("failed to register agent1: %v", err)
	}
	if err := registry.Register(agent2); err != nil {
		t.Fatalf("failed to register agent2: %v", err)
	}

	// Count
	if registry.Count() != 2 {
		t.Errorf("expected count 2, got %d", registry.Count())
	}

	// Lookup by ID
	found := registry.GetAgentByID("agent-1")
	if found == nil {
		t.Fatal("expected to find agent-1")
	}
	if found.Label != "Test Agent 1" {
		t.Errorf("expected label 'Test Agent 1', got %q", found.Label)
	}

	// Lookup by inbox
	found = registry.GetAgentForInbox("inbox-2")
	if found == nil {
		t.Fatal("expected to find agent for inbox-2")
	}
	if found.ID != "agent-2" {
		t.Errorf("expected agent-2, got %s", found.ID)
	}

	// List inboxes
	inboxes := registry.ListInboxes()
	if len(inboxes) != 2 {
		t.Errorf("expected 2 inboxes, got %d", len(inboxes))
	}

	// Validation
	issues := registry.Validate()
	if len(issues) > 0 {
		t.Logf("validation issues (expected none): %v", issues)
	}
}

// TestIntegration_WorktreeManager tests worktree creation and cleanup
func TestIntegration_WorktreeManager(t *testing.T) {
	// Skip if coordinator is running (has existing worktrees)
	if _, err := os.Stat(filepath.Join(os.Getenv("HOME"), ".ailang", "state", "coordinator.pid")); err == nil {
		t.Skip("coordinator is running, skip worktree test to avoid conflicts")
	}

	// Skip if not in a git repo
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		// Find git root
		wd, _ := os.Getwd()
		for wd != "/" {
			if _, err := os.Stat(filepath.Join(wd, ".git")); err == nil {
				break
			}
			wd = filepath.Dir(wd)
		}
		if wd == "/" {
			t.Skip("not in a git repository")
		}
	}

	tmpDir := t.TempDir()
	worktreeBase := filepath.Join(tmpDir, "worktrees")

	// Get git root
	wd, _ := os.Getwd()
	gitRoot := wd
	for gitRoot != "/" {
		if _, err := os.Stat(filepath.Join(gitRoot, ".git")); err == nil {
			break
		}
		gitRoot = filepath.Dir(gitRoot)
	}

	// Get current branch name (CI may only have 'dev', not 'main')
	branchCmd := exec.Command("git", "-C", gitRoot, "rev-parse", "--abbrev-ref", "HEAD")
	branchOutput, err := branchCmd.Output()
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}
	currentBranch := strings.TrimSpace(string(branchOutput))

	mgr, err := NewWorktreeManager(gitRoot, worktreeBase, 3)
	if err != nil {
		t.Fatalf("failed to create worktree manager: %v", err)
	}

	// Record initial count (may have existing worktrees from other tests/runs)
	initialCount := mgr.Count()

	// Create a worktree from the current branch (not hardcoded 'main')
	wt, err := mgr.CreateWorktree("test-task-integ-1", currentBranch)
	if err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	if wt.TaskID != "test-task-integ-1" {
		t.Errorf("expected task ID test-task-integ-1, got %s", wt.TaskID)
	}

	// Verify directory exists
	if _, err := os.Stat(wt.Path); os.IsNotExist(err) {
		t.Error("worktree path does not exist")
	}

	// Count should increase by 1
	if mgr.Count() != initialCount+1 {
		t.Errorf("expected %d worktrees after create, got %d", initialCount+1, mgr.Count())
	}

	// Get the worktree we created
	retrieved, found := mgr.GetWorktree("test-task-integ-1")
	if !found {
		t.Error("expected to find worktree test-task-integ-1")
	}
	if retrieved.TaskID != "test-task-integ-1" {
		t.Errorf("expected task ID test-task-integ-1, got %s", retrieved.TaskID)
	}

	// Remove worktree
	if err := mgr.RemoveWorktree("test-task-integ-1"); err != nil {
		t.Fatalf("failed to remove worktree: %v", err)
	}

	// Count should be back to initial
	if mgr.Count() != initialCount {
		t.Errorf("expected %d worktrees after remove, got %d", initialCount, mgr.Count())
	}
}

// TestIntegration_StoreApprovalRequests tests approval request storage
func TestIntegration_StoreApprovalRequests(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	ctx := context.Background()

	// First create a task (approval requests need a task)
	task := &TaskRecord{
		ID:        "apr-store-task",
		Title:     "Task for approval",
		Content:   "Content",
		Type:      TaskTypeBugFix,
		Status:    TaskStatusPendingApproval,
		CreatedAt: time.Now(),
	}
	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Create approval request
	request := &ApprovalRequestRecord{
		ID:          "apr-store-1",
		TaskID:      "apr-store-task",
		Type:        string(ApprovalTypeMerge),
		Description: "Please approve",
		Status:      "pending",
		CreatedAt:   time.Now(),
	}

	if err := store.CreateApprovalRequest(ctx, request); err != nil {
		t.Fatalf("failed to create approval request: %v", err)
	}

	// List pending approvals
	pending, err := store.ListPendingApprovals(ctx)
	if err != nil {
		t.Fatalf("failed to list pending approvals: %v", err)
	}

	if len(pending) != 1 {
		t.Errorf("expected 1 pending approval, got %d", len(pending))
	}

	// Resolve approval
	if err := store.ResolveApprovalRequest(ctx, "apr-store-1", "approved", "test-user"); err != nil {
		t.Fatalf("failed to resolve approval: %v", err)
	}

	// Should be no pending approvals now
	pending, _ = store.ListPendingApprovals(ctx)
	if len(pending) != 0 {
		t.Errorf("expected 0 pending approvals after resolution, got %d", len(pending))
	}
}

// TestIntegration_TaskPriorityOrdering tests that tasks are returned in priority order
func TestIntegration_TaskPriorityOrdering(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	ctx := context.Background()

	// Create tasks with different priorities
	tasks := []*TaskRecord{
		{ID: "low-pri", Title: "Low Priority", Content: "c", Type: TaskTypeDocs, Status: TaskStatusPending, Priority: 1, CreatedAt: time.Now()},
		{ID: "high-pri", Title: "High Priority", Content: "c", Type: TaskTypeBugFix, Status: TaskStatusPending, Priority: 5, CreatedAt: time.Now()},
		{ID: "med-pri", Title: "Medium Priority", Content: "c", Type: TaskTypeFeature, Status: TaskStatusPending, Priority: 3, CreatedAt: time.Now()},
	}

	for _, task := range tasks {
		if err := store.CreateTask(ctx, task); err != nil {
			t.Fatalf("failed to create task: %v", err)
		}
	}

	// List with priority ordering
	filter := &TaskFilter{
		Status:    []TaskStatus{TaskStatusPending},
		OrderBy:   "priority",
		OrderDesc: true,
	}

	result, err := store.ListTasks(ctx, filter)
	if err != nil {
		t.Fatalf("failed to list tasks: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(result))
	}

	// Verify order: high, med, low
	if result[0].ID != "high-pri" {
		t.Errorf("expected high-pri first, got %s", result[0].ID)
	}
	if result[1].ID != "med-pri" {
		t.Errorf("expected med-pri second, got %s", result[1].ID)
	}
	if result[2].ID != "low-pri" {
		t.Errorf("expected low-pri third, got %s", result[2].ID)
	}
}

// TestIntegration_DuplicateTaskDetection tests fingerprint-based deduplication
func TestIntegration_DuplicateTaskDetection(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	ctx := context.Background()

	// Create first task with fingerprint
	task1 := &TaskRecord{
		ID:        "orig-task",
		Title:     "Original Task",
		Content:   "Fix the parser bug",
		Type:      TaskTypeBugFix,
		Status:    TaskStatusPending,
		CreatedAt: time.Now(),
	}

	if err := store.CreateTask(ctx, task1); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	fingerprint := uint64(0xABCDEF123456)
	if err := store.SetTaskFingerprint(ctx, task1.ID, fingerprint); err != nil {
		t.Fatalf("failed to set fingerprint: %v", err)
	}

	// Try to find duplicate with same fingerprint
	dup, err := store.FindDuplicateTask(ctx, fingerprint, 0.9)
	if err != nil {
		t.Fatalf("failed to find duplicate: %v", err)
	}

	if dup == nil {
		t.Fatal("expected to find duplicate task")
	}

	if dup.ID != task1.ID {
		t.Errorf("expected duplicate ID %s, got %s", task1.ID, dup.ID)
	}

	// Different fingerprint should not match
	dup2, _ := store.FindDuplicateTask(ctx, 0x111111111111, 0.9)
	if dup2 != nil {
		t.Error("expected no duplicate for different fingerprint")
	}
}

// TestIntegration_TaskEventStorage tests streaming event storage and retrieval
func TestIntegration_TaskEventStorage(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	ctx := context.Background()

	// Create a task first
	task := &TaskRecord{
		ID:        "event-task",
		Title:     "Event Test Task",
		Content:   "Test event storage",
		Type:      TaskTypeBugFix,
		Status:    TaskStatusRunning,
		ThreadID:  "thread-event-1",
		CreatedAt: time.Now(),
	}
	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Store some events
	events := []*TaskEventRecord{
		{TaskID: task.ID, ThreadID: task.ThreadID, StreamType: "status", Status: "running"},
		{TaskID: task.ID, ThreadID: task.ThreadID, StreamType: "text", Text: "Processing..."},
		{TaskID: task.ID, ThreadID: task.ThreadID, StreamType: "text", Text: "Done!"},
		{TaskID: task.ID, ThreadID: task.ThreadID, StreamType: "status", Status: "completed"},
	}

	for _, event := range events {
		if err := store.StoreTaskEvent(ctx, event); err != nil {
			t.Fatalf("failed to store event: %v", err)
		}
	}

	// Retrieve events (limit 100)
	retrieved, err := store.GetTaskEvents(ctx, task.ID, 100)
	if err != nil {
		t.Fatalf("failed to get events: %v", err)
	}

	if len(retrieved) != 4 {
		t.Errorf("expected 4 events, got %d", len(retrieved))
	}

	// Verify order (should be chronological)
	if retrieved[0].StreamType != "status" {
		t.Errorf("expected first event type 'status', got %s", retrieved[0].StreamType)
	}
}

// TestIntegration_StaleTasks tests recovery of stale tasks
func TestIntegration_StaleTasks(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	ctx := context.Background()

	// Create a "stale" running task (started long ago)
	oldTime := time.Now().Add(-10 * time.Minute)
	task := &TaskRecord{
		ID:        "stale-task",
		Title:     "Stale Task",
		Content:   "This task got stuck",
		Type:      TaskTypeBugFix,
		Status:    TaskStatusRunning,
		CreatedAt: oldTime,
		StartedAt: &oldTime,
	}

	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Recover stale tasks (threshold: 5 minutes)
	recovered, err := store.RecoverStaleTasks(ctx, 5*time.Minute)
	if err != nil {
		t.Fatalf("failed to recover stale tasks: %v", err)
	}

	if recovered != 1 {
		t.Errorf("expected 1 recovered task, got %d", recovered)
	}

	// Task should now be cancelled
	retrieved, _ := store.GetTask(ctx, task.ID)
	if retrieved.Status != TaskStatusCancelled {
		t.Errorf("expected status %s, got %s", TaskStatusCancelled, retrieved.Status)
	}
}

// TestIntegration_EndToEnd_SimplePath tests a complete simple path through the system
func TestIntegration_EndToEnd_SimplePath(t *testing.T) {
	// This test simulates the minimum viable path:
	// 1. Create task in store
	// 2. Execute with mock provider
	// 3. Mark completed

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "e2e.db")

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Step 1: Create task
	task := &TaskRecord{
		ID:        "e2e-task-1",
		MessageID: "e2e-msg-1",
		Title:     "End-to-End Test Task",
		Content:   "Fix the critical bug in authentication",
		Type:      TaskTypeBugFix,
		Priority:  5,
		Status:    TaskStatusPending,
		Workspace: tmpDir,
		CreatedAt: time.Now(),
	}

	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("Step 1 failed - create task: %v", err)
	}

	// Verify task exists
	retrieved, err := store.GetTask(ctx, task.ID)
	if err != nil || retrieved == nil {
		t.Fatalf("Step 1 verification failed: %v", err)
	}

	// Step 2: Execute with mock provider
	mockProvider := NewIntegrationMockProvider("e2e-mock")
	mockProvider.SetExecuteFunc(func(ctx context.Context, task *AnalyzedTask, opts *ExecuteOptions) (*ExecuteResult, error) {
		return &ExecuteResult{
			Success:       true,
			Output:        "Fixed the authentication bug by checking token expiry",
			Duration:      2 * time.Second,
			Cost:          0.02,
			TokensUsed:    500,
			InputTokens:   300,
			OutputTokens:  200,
			Provider:      "e2e-mock",
			FilesModified: []string{"auth.go", "auth_test.go"},
		}, nil
	})

	executor := NewTaskExecutor(mockProvider)

	// Mark running
	if err := store.MarkTaskRunning(ctx, task.ID, "e2e-mock", ""); err != nil {
		t.Fatalf("Step 2 failed - mark running: %v", err)
	}

	// Execute
	analyzedTask := &AnalyzedTask{
		Task: &Task{
			ID:      task.ID,
			Title:   task.Title,
			Content: task.Content,
		},
		Type: task.Type,
	}

	opts := &ExecuteOptions{
		Timeout:   30 * time.Second,
		Workspace: tmpDir,
	}

	result, err := executor.Execute(ctx, analyzedTask, opts)
	if err != nil {
		t.Fatalf("Step 2 failed - execute: %v", err)
	}

	if !result.Success {
		t.Fatalf("Step 2 failed - execution not successful: %s", result.Error)
	}

	// Step 3: Mark completed
	if err := store.MarkTaskCompleted(ctx, task.ID, result); err != nil {
		t.Fatalf("Step 3 failed - mark completed: %v", err)
	}

	// Final verification
	final, _ := store.GetTask(ctx, task.ID)
	if final.Status != TaskStatusCompleted {
		t.Errorf("final status expected %s, got %s", TaskStatusCompleted, final.Status)
	}
	if final.Cost != 0.02 {
		t.Errorf("final cost expected 0.02, got %f", final.Cost)
	}
	if final.TokensUsed != 500 {
		t.Errorf("final tokens expected 500, got %d", final.TokensUsed)
	}

	t.Logf("End-to-end test passed: task %s completed with cost $%.4f", task.ID, final.Cost)
}

// TestIntegration_GitHubPipelineStages tests the GitHub-driven pipeline stages
// This verifies that:
// 1. Tasks with GithubIssue get stage-aware directives
// 2. Output parsing extracts design doc/sprint plan paths
// 3. Stage transitions work correctly with RequeueTask
func TestIntegration_GitHubPipelineStages(t *testing.T) {
	store := createTestStore(t)
	defer store.Close()
	ctx := context.Background()

	// Create a GitHub-linked task
	task := &TaskRecord{
		ID:          "gh-pipeline-1",
		MessageID:   "msg-gh-1",
		Title:       "Add new feature",
		Content:     "Implement the frobnitz widget",
		Type:        TaskTypeFeature,
		Status:      TaskStatusPending,
		GithubIssue: 42, // Linked to GitHub issue #42
		Stage:       TaskStageDesign,
		CreatedAt:   time.Now(),
	}

	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Verify stage-aware directive for design stage
	directive := BuildStageDirective(task)
	if directive == task.Content {
		t.Error("expected stage-aware directive, got original content")
	}
	if !containsString(directive, "design-doc-creator") {
		t.Error("design stage directive should mention design-doc-creator skill")
	}
	if !containsString(directive, "DESIGN_DOC_PATH:") {
		t.Error("design stage directive should include output format")
	}

	// Test output parsing for design stage
	designOutput := `Created design document for frobnitz widget.
DESIGN_DOC_PATH: design_docs/planned/v0_6_3/frobnitz-widget.md
Ready for review.`

	designResult := ParseStageOutput(designOutput, TaskStageDesign)
	if designResult.DesignDocPath != "design_docs/planned/v0_6_3/frobnitz-widget.md" {
		t.Errorf("expected design doc path, got: %s", designResult.DesignDocPath)
	}

	// Transition to sprint stage
	if err := store.SetTaskStage(ctx, task.ID, TaskStageSprint); err != nil {
		t.Fatalf("failed to set stage: %v", err)
	}

	// Verify sprint directive
	task.Stage = TaskStageSprint
	sprintDirective := BuildStageDirective(task)
	if !containsString(sprintDirective, "sprint-planner") {
		t.Error("sprint stage directive should mention sprint-planner skill")
	}

	// Test output parsing for sprint stage
	sprintOutput := `Created sprint plan.
SPRINT_PLAN_PATH: design_docs/planned/v0_6_3/frobnitz-sprint-plan.md
Ready for execution.`

	sprintResult := ParseStageOutput(sprintOutput, TaskStageSprint)
	if sprintResult.SprintPlanPath != "design_docs/planned/v0_6_3/frobnitz-sprint-plan.md" {
		t.Errorf("expected sprint plan path, got: %s", sprintResult.SprintPlanPath)
	}

	// Test RequeueTask
	if err := store.MarkTaskRunning(ctx, task.ID, "test-provider", ""); err != nil {
		t.Fatalf("failed to mark running: %v", err)
	}

	retrieved, _ := store.GetTask(ctx, task.ID)
	if retrieved.Status != TaskStatusRunning {
		t.Errorf("expected running, got %s", retrieved.Status)
	}

	if err := store.RequeueTask(ctx, task.ID); err != nil {
		t.Fatalf("failed to requeue task: %v", err)
	}

	retrieved, _ = store.GetTask(ctx, task.ID)
	if retrieved.Status != TaskStatusPending {
		t.Errorf("expected pending after requeue, got %s", retrieved.Status)
	}

	// Verify implementation stage parsing
	implOutput := `Implementation complete.
IMPLEMENTATION_COMPLETE: true
BRANCH_NAME: feature/frobnitz-widget
FILES_CREATED: internal/frobnitz/widget.go, internal/frobnitz/widget_test.go
FILES_MODIFIED: internal/registry/registry.go`

	implResult := ParseStageOutput(implOutput, TaskStageImplementation)
	if implResult.BranchName != "feature/frobnitz-widget" {
		t.Errorf("expected branch name, got: %s", implResult.BranchName)
	}
	if len(implResult.FilesCreated) != 2 {
		t.Errorf("expected 2 files created, got: %v", implResult.FilesCreated)
	}
	if len(implResult.FilesModified) != 1 {
		t.Errorf("expected 1 file modified, got: %v", implResult.FilesModified)
	}

	t.Log("GitHub pipeline stages test passed")
}

// containsString checks if s contains substr
func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) > len(substr) &&
			(s[:len(substr)] == substr || containsString(s[1:], substr)))
}
