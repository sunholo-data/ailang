package coordinator

import (
	"context"
	"sync"
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

func (m *MockStore) MarkTaskPendingApproval(ctx context.Context, id, worktreePath, worktreeBranch, baseBranch, baseCommit string, result *ExecuteResult) error {
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

func (m *MockStore) ResetTaskToPending(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["ResetTaskToPending"]++
	if task, ok := m.tasks[id]; ok {
		task.Status = TaskStatusPending
		task.StartedAt = nil
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

func (m *MockStore) GetApprovalRequest(ctx context.Context, id string) (*ApprovalRequestRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["GetApprovalRequest"]++
	return nil, nil
}

func (m *MockStore) GetApprovalRequestByTask(ctx context.Context, taskID string) (*ApprovalRequestRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["GetApprovalRequestByTask"]++
	return nil, nil
}

func (m *MockStore) GetApprovalRequestByTaskAnyStatus(ctx context.Context, taskID string) (*ApprovalRequestRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["GetApprovalRequestByTaskAnyStatus"]++
	return nil, nil
}

func (m *MockStore) ListPendingApprovals(ctx context.Context) ([]*ApprovalRequestRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["ListPendingApprovals"]++
	return nil, nil
}

func (m *MockStore) ListResolvedApprovals(ctx context.Context, limit int) ([]*ApprovalRequestRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["ListResolvedApprovals"]++
	return nil, nil
}

func (m *MockStore) ResolveApprovalRequest(ctx context.Context, id, status, resolvedBy string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["ResolveApprovalRequest"]++
	return nil
}

func (m *MockStore) ResolveApprovalRequestByTask(ctx context.Context, taskID, status, resolvedBy string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["ResolveApprovalRequestByTask"]++
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

func (m *MockStore) GetTaskAgentInfo(ctx context.Context, taskID string) (agentID, inbox, title string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["GetTaskAgentInfo"]++
	task, ok := m.tasks[taskID]
	if !ok {
		return "", "", "", nil
	}
	// By convention, agent id == inbox in agent config
	return task.AgentID, task.AgentID, task.Title, nil
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

func (m *MockStore) GetCostByProvider() (map[string]float64, error) {
	return make(map[string]float64), nil
}

func (m *MockStore) ListApprovedMergeHandoffsWithoutTrigger(ctx context.Context) ([]*ApprovalRequestRecord, error) {
	return nil, nil
}

func (m *MockStore) MarkApprovalHandoffsTriggered(ctx context.Context, taskID string) error {
	return nil
}

func (m *MockStore) UpdateTaskChainInfo(ctx context.Context, id, chainID, stageID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls["UpdateTaskChainInfo"]++
	if task, ok := m.tasks[id]; ok {
		task.ChainID = chainID
		task.StageID = stageID
	}
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
