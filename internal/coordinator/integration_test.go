package coordinator

import (
	"context"
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

// containsString checks if s contains substr
func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) > len(substr) &&
			(s[:len(substr)] == substr || containsString(s[1:], substr)))
}
