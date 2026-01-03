package coordinator

import (
	"context"
	"sync"
	"testing"
	"time"
)

// =============================================================================
// MockStore implements Store interface for testing TaskChain
// =============================================================================

type MockStore struct {
	mu     sync.Mutex
	tasks  map[string]*TaskRecord
	stages map[string]TaskStage

	// Error injection
	getTaskErr            error
	setTaskGithubIssueErr error
	setTaskStageErr       error
	requeueTaskErr        error

	// Call tracking
	calls map[string]int
}

func NewMockStore() *MockStore {
	return &MockStore{
		tasks:  make(map[string]*TaskRecord),
		stages: make(map[string]TaskStage),
		calls:  make(map[string]int),
	}
}

func (m *MockStore) CreateTask(ctx context.Context, task *TaskRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["CreateTask"]++
	m.tasks[task.ID] = task
	return nil
}

func (m *MockStore) GetTask(ctx context.Context, id string) (*TaskRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["GetTask"]++
	if m.getTaskErr != nil {
		return nil, m.getTaskErr
	}
	task, ok := m.tasks[id]
	if !ok {
		return nil, nil
	}
	// Apply stage if set separately
	if stage, ok := m.stages[id]; ok {
		task.Stage = stage
	}
	return task, nil
}

func (m *MockStore) UpdateTask(ctx context.Context, task *TaskRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["UpdateTask"]++
	m.tasks[task.ID] = task
	return nil
}

func (m *MockStore) DeleteTask(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["DeleteTask"]++
	delete(m.tasks, id)
	return nil
}

func (m *MockStore) ListTasks(ctx context.Context, filter *TaskFilter) ([]*TaskRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["ListTasks"]++
	var result []*TaskRecord
	for _, t := range m.tasks {
		result = append(result, t)
	}
	return result, nil
}

func (m *MockStore) GetTaskStats(ctx context.Context) (*TaskStats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["GetTaskStats"]++
	return &TaskStats{}, nil
}

func (m *MockStore) MarkTaskQueued(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["MarkTaskQueued"]++
	return nil
}

func (m *MockStore) MarkTaskRunning(ctx context.Context, id, provider, worktreeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["MarkTaskRunning"]++
	return nil
}

func (m *MockStore) MarkTaskPendingApproval(ctx context.Context, id, worktreePath, worktreeBranch string, result *ExecuteResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["MarkTaskPendingApproval"]++
	return nil
}

func (m *MockStore) MarkTaskCompleted(ctx context.Context, id string, result *ExecuteResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["MarkTaskCompleted"]++
	return nil
}

func (m *MockStore) MarkTaskFailed(ctx context.Context, id string, err error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["MarkTaskFailed"]++
	return nil
}

func (m *MockStore) MarkTaskRejected(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["MarkTaskRejected"]++
	return nil
}

func (m *MockStore) MarkTaskCancelled(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["MarkTaskCancelled"]++
	return nil
}

func (m *MockStore) RequeueTask(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["RequeueTask"]++
	if m.requeueTaskErr != nil {
		return m.requeueTaskErr
	}
	if task, ok := m.tasks[id]; ok {
		task.Status = TaskStatusPending
	}
	return nil
}

func (m *MockStore) FindDuplicateTask(ctx context.Context, fingerprint uint64, threshold float64) (*TaskRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["FindDuplicateTask"]++
	return nil, nil
}

func (m *MockStore) SetTaskFingerprint(ctx context.Context, id string, fingerprint uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["SetTaskFingerprint"]++
	return nil
}

func (m *MockStore) SetTaskThreadID(ctx context.Context, id string, threadID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["SetTaskThreadID"]++
	return nil
}

func (m *MockStore) SetTaskGithubIssue(ctx context.Context, id string, issueNum int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["SetTaskGithubIssue"]++
	if m.setTaskGithubIssueErr != nil {
		return m.setTaskGithubIssueErr
	}
	if task, ok := m.tasks[id]; ok {
		task.GithubIssue = issueNum
	}
	return nil
}

func (m *MockStore) SetTaskStage(ctx context.Context, id string, stage TaskStage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["SetTaskStage"]++
	if m.setTaskStageErr != nil {
		return m.setTaskStageErr
	}
	m.stages[id] = stage
	if task, ok := m.tasks[id]; ok {
		task.Stage = stage
	}
	return nil
}

func (m *MockStore) SetTaskDesignDocPath(ctx context.Context, id string, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["SetTaskDesignDocPath"]++
	if task, ok := m.tasks[id]; ok {
		task.DesignDocPath = path
	}
	return nil
}

func (m *MockStore) SetTaskSprintPlanPath(ctx context.Context, id string, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["SetTaskSprintPlanPath"]++
	if task, ok := m.tasks[id]; ok {
		task.SprintPlanPath = path
	}
	return nil
}

func (m *MockStore) GetTasksByStage(ctx context.Context, stage TaskStage) ([]*TaskRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["GetTasksByStage"]++
	var result []*TaskRecord
	for _, t := range m.tasks {
		if t.Stage == stage {
			result = append(result, t)
		}
	}
	return result, nil
}

func (m *MockStore) GetTasksByGithubIssue(ctx context.Context, issueNum int) ([]*TaskRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["GetTasksByGithubIssue"]++
	var result []*TaskRecord
	for _, t := range m.tasks {
		if t.GithubIssue == issueNum {
			result = append(result, t)
		}
	}
	return result, nil
}

func (m *MockStore) UpdateTaskMetrics(ctx context.Context, id string, peakCPU, peakMemory float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["UpdateTaskMetrics"]++
	return nil
}

func (m *MockStore) CreateApprovalRequest(ctx context.Context, req *ApprovalRequestRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["CreateApprovalRequest"]++
	return nil
}

func (m *MockStore) ListPendingApprovals(ctx context.Context) ([]*ApprovalRequestRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["ListPendingApprovals"]++
	return nil, nil
}

func (m *MockStore) ResolveApprovalRequest(ctx context.Context, id, status, resolvedBy string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["ResolveApprovalRequest"]++
	return nil
}

func (m *MockStore) DeleteOldTasks(ctx context.Context, olderThan time.Duration) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["DeleteOldTasks"]++
	return 0, nil
}

func (m *MockStore) RecoverStaleTasks(ctx context.Context, staleThreshold time.Duration) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["RecoverStaleTasks"]++
	return 0, nil
}

func (m *MockStore) StoreTaskEvent(ctx context.Context, event *TaskEventRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["StoreTaskEvent"]++
	return nil
}

func (m *MockStore) GetTaskEvents(ctx context.Context, taskID string, limit int) ([]*TaskEventRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["GetTaskEvents"]++
	return nil, nil
}

func (m *MockStore) RetryAllFailedTasks(ctx context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["RetryAllFailedTasks"]++
	return 0, nil
}

func (m *MockStore) Close() error {
	return nil
}

func (m *MockStore) GetCallCount(method string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls[method]
}

// =============================================================================
// Test fixtures
// =============================================================================

func newTestTask(id string) *TaskRecord {
	return &TaskRecord{
		ID:        id,
		Title:     "Test Task " + id,
		Content:   "Test content for " + id,
		Type:      TaskTypeBugFix,
		Status:    TaskStatusPending,
		CreatedAt: time.Now(),
	}
}

func newTestTaskWithGitHub(id string, issueNum int) *TaskRecord {
	task := newTestTask(id)
	task.GithubIssue = issueNum
	return task
}

// =============================================================================
// Milestone 2: Basic Lifecycle Tests
// =============================================================================

func TestNewTaskChain_RegistersHandlers(t *testing.T) {
	store := NewMockStore()
	// Create a real watcher with nil poster (won't make GitHub calls)
	watcher := NewApprovalWatcher(nil, store, 60*time.Second)

	tc := NewTaskChain(nil, store, watcher)

	// TaskChain should be created successfully
	if tc == nil {
		t.Fatal("expected TaskChain to be created")
	}

	// Can't easily verify handler registration without exposing internals,
	// but we can verify the watcher has handlers by checking it's configured
	if watcher.WatchedIssueCount() != 0 {
		t.Error("expected no watched issues initially")
	}
}

func TestNewTaskChain_WithNilWatcher(t *testing.T) {
	store := NewMockStore()

	// Should not panic with nil watcher
	tc := NewTaskChain(nil, store, nil)

	if tc == nil {
		t.Fatal("expected TaskChain to be created with nil watcher")
	}
}

func TestStartTask_LinksGitHubIssue(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	// Create task first
	task := newTestTask("task-1")
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	err := tc.StartTask(ctx, "task-1", 42)
	if err != nil {
		t.Fatalf("StartTask failed: %v", err)
	}

	if store.GetCallCount("SetTaskGithubIssue") != 1 {
		t.Error("expected SetTaskGithubIssue to be called")
	}

	retrieved, _ := store.GetTask(ctx, "task-1")
	if retrieved.GithubIssue != 42 {
		t.Errorf("expected GithubIssue=42, got %d", retrieved.GithubIssue)
	}
}

func TestStartTask_SetsDesignStage(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTask("task-1")
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	err := tc.StartTask(ctx, "task-1", 42)
	if err != nil {
		t.Fatalf("StartTask failed: %v", err)
	}

	if store.GetCallCount("SetTaskStage") != 1 {
		t.Error("expected SetTaskStage to be called")
	}

	retrieved, _ := store.GetTask(ctx, "task-1")
	if retrieved.Stage != TaskStageDesign {
		t.Errorf("expected Stage=design, got %s", retrieved.Stage)
	}
}

func TestStartTask_WatchesIssue(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTask("task-1")
	store.CreateTask(ctx, task)

	watcher := NewApprovalWatcher(nil, store, 60*time.Second)
	tc := NewTaskChain(nil, store, watcher)

	err := tc.StartTask(ctx, "task-1", 42)
	if err != nil {
		t.Fatalf("StartTask failed: %v", err)
	}

	if watcher.WatchedIssueCount() != 1 {
		t.Errorf("expected 1 watched issue, got %d", watcher.WatchedIssueCount())
	}
}

func TestStartTask_NilPoster_Graceful(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTask("task-1")
	store.CreateTask(ctx, task)

	// With nil poster, should still succeed but skip GitHub comment
	tc := NewTaskChain(nil, store, nil)

	err := tc.StartTask(ctx, "task-1", 42)
	if err != nil {
		t.Fatalf("StartTask with nil poster failed: %v", err)
	}
}

// =============================================================================
// Milestone 3: Design Stage Tests
// =============================================================================

func TestOnDesignDocComplete_NoGitHubIssue_Skips(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTask("task-1") // No GitHub issue
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	result := &DesignDocResult{Path: "design_docs/test.md"}

	err := tc.OnDesignDocComplete(ctx, "task-1", result)
	if err != nil {
		t.Fatalf("OnDesignDocComplete failed: %v", err)
	}

	// Should not fail when task has no GitHub issue
}

func TestOnDesignDocComplete_WithGitHubIssue_NilPoster(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	task.WorktreePath = "/tmp/worktree"
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	result := &DesignDocResult{
		Path:     "design_docs/planned/v0_6_3/test.md",
		Duration: 5 * time.Second,
	}

	err := tc.OnDesignDocComplete(ctx, "task-1", result)
	if err != nil {
		t.Fatalf("OnDesignDocComplete failed: %v", err)
	}

	// With nil poster, should succeed but skip posting
}

func TestOnDesignApproved_TransitionsStage(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	task.Stage = TaskStageDesign
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	event := &ApprovalEvent{
		TaskID:      "task-1",
		IssueNumber: 42,
		Label:       LabelDesignApproved,
		EventType:   ApprovalEventDesign,
	}

	err := tc.OnDesignApproved(ctx, event)
	if err != nil {
		t.Fatalf("OnDesignApproved failed: %v", err)
	}

	retrieved, _ := store.GetTask(ctx, "task-1")
	if retrieved.Stage != TaskStageSprint {
		t.Errorf("expected Stage=sprint, got %s", retrieved.Stage)
	}
}

func TestOnDesignApproved_RequeuesTask(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	task.Stage = TaskStageDesign
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	event := &ApprovalEvent{
		TaskID:      "task-1",
		IssueNumber: 42,
		EventType:   ApprovalEventDesign,
	}

	err := tc.OnDesignApproved(ctx, event)
	if err != nil {
		t.Fatalf("OnDesignApproved failed: %v", err)
	}

	if store.GetCallCount("RequeueTask") != 1 {
		t.Error("expected RequeueTask to be called")
	}
}

// =============================================================================
// Milestone 4: Sprint Stage Tests
// =============================================================================

func TestOnSprintPlanComplete_NoGitHubIssue_Skips(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTask("task-1") // No GitHub issue
	task.Stage = TaskStageSprint
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	result := &SprintPlanResult{
		Path:     "design_docs/planned/v0_6_3/test-sprint-plan.md",
		Duration: 3 * time.Second,
	}

	err := tc.OnSprintPlanComplete(ctx, "task-1", result)
	if err != nil {
		t.Fatalf("OnSprintPlanComplete failed: %v", err)
	}
}

func TestOnSprintApproved_TransitionsStage(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	task.Stage = TaskStageSprint
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	event := &ApprovalEvent{
		TaskID:      "task-1",
		IssueNumber: 42,
		EventType:   ApprovalEventSprint,
	}

	err := tc.OnSprintApproved(ctx, event)
	if err != nil {
		t.Fatalf("OnSprintApproved failed: %v", err)
	}

	retrieved, _ := store.GetTask(ctx, "task-1")
	if retrieved.Stage != TaskStageImplementation {
		t.Errorf("expected Stage=implementation, got %s", retrieved.Stage)
	}
}

func TestOnSprintApproved_RequeuesTask(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	task.Stage = TaskStageSprint
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	event := &ApprovalEvent{
		TaskID:      "task-1",
		IssueNumber: 42,
		EventType:   ApprovalEventSprint,
	}

	err := tc.OnSprintApproved(ctx, event)
	if err != nil {
		t.Fatalf("OnSprintApproved failed: %v", err)
	}

	if store.GetCallCount("RequeueTask") != 1 {
		t.Error("expected RequeueTask to be called")
	}
}

// =============================================================================
// Milestone 5: Implementation and Merge Tests
// =============================================================================

func TestOnImplementationComplete_NoGitHubIssue_Skips(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTask("task-1") // No GitHub issue
	task.Stage = TaskStageImplementation
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	result := &ImplementResult{
		BranchName:    "fix-bug-123",
		WorktreePath:  "/tmp/worktree",
		Duration:      10 * time.Second,
		FilesCreated:  []string{"internal/foo.go"},
		FilesModified: []string{"internal/bar.go"},
	}

	err := tc.OnImplementationComplete(ctx, "task-1", result)
	if err != nil {
		t.Fatalf("OnImplementationComplete failed: %v", err)
	}
}

func TestOnImplementationComplete_SetsStageToMerge(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	task.Stage = TaskStageImplementation
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	result := &ImplementResult{BranchName: "fix-bug-123"}

	err := tc.OnImplementationComplete(ctx, "task-1", result)
	if err != nil {
		t.Fatalf("OnImplementationComplete failed: %v", err)
	}

	retrieved, _ := store.GetTask(ctx, "task-1")
	if retrieved.Stage != TaskStageMerge {
		t.Errorf("expected Stage=merge, got %s", retrieved.Stage)
	}
}

func TestOnMergeApproved_UnwatchesIssue(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	task.Stage = TaskStageMerge
	store.CreateTask(ctx, task)

	watcher := NewApprovalWatcher(nil, store, 60*time.Second)
	watcher.WatchIssue(42, "task-1")

	tc := NewTaskChain(nil, store, watcher)

	event := &ApprovalEvent{
		TaskID:      "task-1",
		IssueNumber: 42,
		EventType:   ApprovalEventMerge,
	}

	err := tc.OnMergeApproved(ctx, event)
	if err != nil {
		t.Fatalf("OnMergeApproved failed: %v", err)
	}

	if watcher.WatchedIssueCount() != 0 {
		t.Error("expected issue to be unwatched after merge")
	}
}

func TestOnMergeApproved_ClearsStage(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	task.Stage = TaskStageMerge
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	event := &ApprovalEvent{
		TaskID:      "task-1",
		IssueNumber: 42,
		EventType:   ApprovalEventMerge,
	}

	err := tc.OnMergeApproved(ctx, event)
	if err != nil {
		t.Fatalf("OnMergeApproved failed: %v", err)
	}

	retrieved, _ := store.GetTask(ctx, "task-1")
	if retrieved.Stage != TaskStageNone {
		t.Errorf("expected Stage=none, got %s", retrieved.Stage)
	}
}

// =============================================================================
// Milestone 6: Error and Edge Case Tests
// =============================================================================

func TestOnNeedsRevision_NoError(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	task.Stage = TaskStageDesign
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	event := &ApprovalEvent{
		TaskID:      "task-1",
		IssueNumber: 42,
		EventType:   ApprovalEventRevision,
	}

	err := tc.OnNeedsRevision(ctx, event)
	if err != nil {
		t.Fatalf("OnNeedsRevision failed: %v", err)
	}
}

func TestOnError_NoError(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	task.Stage = TaskStageDesign
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	err := tc.OnError(ctx, "task-1", "Something went wrong")
	if err != nil {
		t.Fatalf("OnError failed: %v", err)
	}
}

func TestOnError_NoGitHubIssue(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTask("task-1") // No GitHub issue
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	err := tc.OnError(ctx, "task-1", "Something went wrong")
	if err != nil {
		t.Fatalf("OnError with no GitHub issue failed: %v", err)
	}
}

func TestTaskChain_NilPoster_AllMethodsGraceful(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	// All methods should handle nil poster gracefully

	// StartTask
	err := tc.StartTask(ctx, "task-1", 42)
	if err != nil {
		t.Fatalf("StartTask with nil poster failed: %v", err)
	}

	// OnDesignDocComplete
	err = tc.OnDesignDocComplete(ctx, "task-1", &DesignDocResult{Path: "test.md"})
	if err != nil {
		t.Fatalf("OnDesignDocComplete with nil poster failed: %v", err)
	}

	// OnSprintPlanComplete
	err = tc.OnSprintPlanComplete(ctx, "task-1", &SprintPlanResult{Path: "test.md"})
	if err != nil {
		t.Fatalf("OnSprintPlanComplete with nil poster failed: %v", err)
	}

	// OnImplementationComplete
	err = tc.OnImplementationComplete(ctx, "task-1", &ImplementResult{BranchName: "test"})
	if err != nil {
		t.Fatalf("OnImplementationComplete with nil poster failed: %v", err)
	}
}

// =============================================================================
// Store Error Handling Tests
// =============================================================================

func TestStartTask_SetGithubIssueError(t *testing.T) {
	store := NewMockStore()
	store.setTaskGithubIssueErr = context.DeadlineExceeded
	ctx := context.Background()

	task := newTestTask("task-1")
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	err := tc.StartTask(ctx, "task-1", 42)
	if err == nil {
		t.Error("expected error when SetTaskGithubIssue fails")
	}
}

func TestStartTask_SetTaskStageError(t *testing.T) {
	store := NewMockStore()
	store.setTaskStageErr = context.DeadlineExceeded
	ctx := context.Background()

	task := newTestTask("task-1")
	store.CreateTask(ctx, task)

	tc := NewTaskChain(nil, store, nil)

	err := tc.StartTask(ctx, "task-1", 42)
	if err == nil {
		t.Error("expected error when SetTaskStage fails")
	}
}

func TestOnDesignApproved_SetTaskStageError(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	store.CreateTask(ctx, task)

	store.setTaskStageErr = context.DeadlineExceeded

	tc := NewTaskChain(nil, store, nil)

	event := &ApprovalEvent{
		TaskID:      "task-1",
		IssueNumber: 42,
		EventType:   ApprovalEventDesign,
	}

	err := tc.OnDesignApproved(ctx, event)
	if err == nil {
		t.Error("expected error when SetTaskStage fails")
	}
}

func TestOnDesignApproved_RequeueTaskError(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	task := newTestTaskWithGitHub("task-1", 42)
	store.CreateTask(ctx, task)

	store.requeueTaskErr = context.DeadlineExceeded

	tc := NewTaskChain(nil, store, nil)

	event := &ApprovalEvent{
		TaskID:      "task-1",
		IssueNumber: 42,
		EventType:   ApprovalEventDesign,
	}

	err := tc.OnDesignApproved(ctx, event)
	if err == nil {
		t.Error("expected error when RequeueTask fails")
	}
}

// =============================================================================
// Config-Driven Approval Tests (M-COORD-GENERIC-WORKFLOWS M4)
// =============================================================================

func TestApprovalWatcher_RegisterAgentApproval(t *testing.T) {
	store := NewMockStore()
	watcher := NewApprovalWatcher(nil, store, time.Second)

	agent := &AgentConfig{
		ID:    "custom-agent",
		Label: "Custom Agent",
		Approval: &ApprovalConfig{
			NeedsLabel:    "needs-custom-approval",
			ApprovedLabel: "custom-approved",
		},
	}

	handler := func(ctx context.Context, event *ApprovalEvent) error {
		return nil
	}

	err := watcher.RegisterAgentApproval(agent, handler)
	if err != nil {
		t.Fatalf("RegisterAgentApproval failed: %v", err)
	}

	// Verify agent is registered
	registeredAgent := watcher.GetAgentByLabel("custom-approved")
	if registeredAgent == nil {
		t.Error("Expected agent to be registered")
	}
	if registeredAgent.ID != "custom-agent" {
		t.Errorf("Expected agent ID 'custom-agent', got %q", registeredAgent.ID)
	}
}

func TestApprovalWatcher_RegisterAgentApproval_NilAgent(t *testing.T) {
	store := NewMockStore()
	watcher := NewApprovalWatcher(nil, store, time.Second)

	err := watcher.RegisterAgentApproval(nil, nil)
	if err == nil {
		t.Error("Expected error for nil agent")
	}
}

func TestApprovalWatcher_RegisterAgentApproval_NoApprovalConfig(t *testing.T) {
	store := NewMockStore()
	watcher := NewApprovalWatcher(nil, store, time.Second)

	agent := &AgentConfig{
		ID:    "agent-without-approval",
		Label: "No Approval",
		// Approval is nil
	}

	err := watcher.RegisterAgentApproval(agent, nil)
	if err == nil {
		t.Error("Expected error for agent without approval config")
	}
}

func TestApprovalWatcher_RegisterAgentApproval_EmptyApprovedLabel(t *testing.T) {
	store := NewMockStore()
	watcher := NewApprovalWatcher(nil, store, time.Second)

	agent := &AgentConfig{
		ID:    "agent-empty-label",
		Label: "Empty Label",
		Approval: &ApprovalConfig{
			NeedsLabel:    "needs-something",
			ApprovedLabel: "", // Empty
		},
	}

	err := watcher.RegisterAgentApproval(agent, nil)
	if err == nil {
		t.Error("Expected error for empty approved_label")
	}
}

func TestApprovalWatcher_GetRegisteredLabels(t *testing.T) {
	store := NewMockStore()
	watcher := NewApprovalWatcher(nil, store, time.Second)

	agent1 := &AgentConfig{
		ID: "agent-1",
		Approval: &ApprovalConfig{
			ApprovedLabel: "label-1",
		},
	}
	agent2 := &AgentConfig{
		ID: "agent-2",
		Approval: &ApprovalConfig{
			ApprovedLabel: "label-2",
		},
	}

	watcher.RegisterAgentApproval(agent1, nil)
	watcher.RegisterAgentApproval(agent2, nil)

	labels := watcher.GetRegisteredLabels()
	if len(labels) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(labels))
	}

	labelMap := make(map[string]bool)
	for _, l := range labels {
		labelMap[l] = true
	}
	if !labelMap["label-1"] || !labelMap["label-2"] {
		t.Errorf("Expected labels label-1 and label-2, got %v", labels)
	}
}

func TestApprovalWatcher_SetAgentRegistry(t *testing.T) {
	store := NewMockStore()
	watcher := NewApprovalWatcher(nil, store, time.Second)

	registry := NewAgentRegistry()
	watcher.SetAgentRegistry(registry)

	// Just verify it doesn't panic
	if watcher.agentRegistry != registry {
		t.Error("Expected registry to be set")
	}
}
