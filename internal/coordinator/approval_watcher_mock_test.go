package coordinator

import (
	"context"
	"time"
)

// Mock implementations for approval_watcher_test.go

//nolint:unused // Scaffolded for future GitHub API mocking tests
type mockGitHubPoster struct {
	getLabels              func(issueNum int) ([]string, error)
	removeLabel            func(issueNum int, label string) error
	getRecentHumanComments func(issueNum int, since time.Time) ([]IssueComment, error)
}

//nolint:unused // Implements interface method for mock
func (m *mockGitHubPoster) GetLabels(issueNum int) ([]string, error) {
	if m.getLabels != nil {
		return m.getLabels(issueNum)
	}
	return []string{}, nil
}

//nolint:unused // Implements interface method for mock
func (m *mockGitHubPoster) RemoveLabel(issueNum int, label string) error {
	if m.removeLabel != nil {
		return m.removeLabel(issueNum, label)
	}
	return nil
}

//nolint:unused // Implements interface method for mock
func (m *mockGitHubPoster) GetRecentHumanComments(issueNum int, since time.Time) ([]IssueComment, error) {
	if m.getRecentHumanComments != nil {
		return m.getRecentHumanComments(issueNum, since)
	}
	return []IssueComment{}, nil
}

type mockStore struct {
	tasks []*TaskRecord
	err   error
}

func (m *mockStore) GetTasksByStage(ctx context.Context, stage TaskStage) ([]*TaskRecord, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.tasks, nil
}

func (m *mockStore) Close() error {
	return nil
}

// Implement other required Store interface methods as no-ops
func (m *mockStore) CreateTask(ctx context.Context, task *TaskRecord) error      { return nil }
func (m *mockStore) GetTask(ctx context.Context, id string) (*TaskRecord, error) { return nil, nil }
func (m *mockStore) UpdateTask(ctx context.Context, task *TaskRecord) error      { return nil }
func (m *mockStore) DeleteTask(ctx context.Context, id string) error             { return nil }
func (m *mockStore) ListTasks(ctx context.Context, filter *TaskFilter) ([]*TaskRecord, error) {
	return nil, nil
}
func (m *mockStore) GetTaskStats(ctx context.Context) (*TaskStats, error) { return nil, nil }
func (m *mockStore) MarkTaskQueued(ctx context.Context, id string) error  { return nil }
func (m *mockStore) MarkTaskRunning(ctx context.Context, id, provider, worktreeID string) error {
	return nil
}
func (m *mockStore) MarkTaskPendingApproval(ctx context.Context, id, worktreePath, worktreeBranch, baseBranch, baseCommit string, result *ExecuteResult) error {
	return nil
}
func (m *mockStore) MarkTaskCompleted(ctx context.Context, id string, result *ExecuteResult) error {
	return nil
}
func (m *mockStore) MarkTaskFailed(ctx context.Context, id string, err error) error { return nil }
func (m *mockStore) MarkTaskRejected(ctx context.Context, id string) error          { return nil }
func (m *mockStore) MarkTaskCancelled(ctx context.Context, id string) error         { return nil }
func (m *mockStore) RequeueTask(ctx context.Context, id string) error               { return nil }
func (m *mockStore) FindDuplicateTask(ctx context.Context, fingerprint uint64, threshold float64) (*TaskRecord, error) {
	return nil, nil
}
func (m *mockStore) SetTaskFingerprint(ctx context.Context, id string, fingerprint uint64) error {
	return nil
}
func (m *mockStore) SetTaskThreadID(ctx context.Context, id string, threadID string) error {
	return nil
}
func (m *mockStore) GetTaskAgentInfo(ctx context.Context, taskID string) (agentID, inbox, title string, err error) {
	return "", "", "", nil
}
func (m *mockStore) SetTaskGithubIssue(ctx context.Context, id string, issueNum int) error {
	return nil
}
func (m *mockStore) SetTaskStage(ctx context.Context, id string, stage TaskStage) error {
	return nil
}
func (m *mockStore) SetTaskDesignDocPath(ctx context.Context, id string, path string) error {
	return nil
}
func (m *mockStore) SetTaskSprintPlanPath(ctx context.Context, id string, path string) error {
	return nil
}
func (m *mockStore) GetTasksByGithubIssue(ctx context.Context, issueNum int) ([]*TaskRecord, error) {
	return nil, nil
}
func (m *mockStore) UpdateTaskMetrics(ctx context.Context, id string, peakCPU, peakMemory float64) error {
	return nil
}
func (m *mockStore) CreateApprovalRequest(ctx context.Context, req *ApprovalRequestRecord) error {
	return nil
}
func (m *mockStore) GetApprovalRequest(ctx context.Context, id string) (*ApprovalRequestRecord, error) {
	return nil, nil
}
func (m *mockStore) GetApprovalRequestByTask(ctx context.Context, taskID string) (*ApprovalRequestRecord, error) {
	return nil, nil
}
func (m *mockStore) GetApprovalRequestByTaskAnyStatus(ctx context.Context, taskID string) (*ApprovalRequestRecord, error) {
	return nil, nil
}
func (m *mockStore) ListPendingApprovals(ctx context.Context) ([]*ApprovalRequestRecord, error) {
	return nil, nil
}
func (m *mockStore) ListResolvedApprovals(ctx context.Context, limit int) ([]*ApprovalRequestRecord, error) {
	return nil, nil
}
func (m *mockStore) ResolveApprovalRequest(ctx context.Context, id, status, resolvedBy string) error {
	return nil
}
func (m *mockStore) ResolveApprovalRequestByTask(ctx context.Context, taskID, status, resolvedBy string) error {
	return nil
}
func (m *mockStore) DeleteOldTasks(ctx context.Context, olderThan time.Duration) (int, error) {
	return 0, nil
}
func (m *mockStore) RecoverStaleTasks(ctx context.Context, staleThreshold time.Duration) (int, error) {
	return 0, nil
}
func (m *mockStore) RetryAllFailedTasks(ctx context.Context) (int, error) {
	return 0, nil
}
func (m *mockStore) StoreTaskEvent(ctx context.Context, event *TaskEventRecord) error {
	return nil
}
func (m *mockStore) GetTaskEvents(ctx context.Context, taskID string, limit int) ([]*TaskEventRecord, error) {
	return nil, nil
}
func (m *mockStore) ResetTaskToPending(ctx context.Context, id string) error {
	return nil
}
func (m *mockStore) GetCostByProvider() (map[string]float64, error) {
	return make(map[string]float64), nil
}
func (m *mockStore) ListApprovedMergeHandoffsWithoutTrigger(ctx context.Context) ([]*ApprovalRequestRecord, error) {
	return nil, nil
}
func (m *mockStore) MarkApprovalHandoffsTriggered(ctx context.Context, taskID string) error {
	return nil
}
func (m *mockStore) UpdateTaskChainInfo(ctx context.Context, id, chainID, stageID string) error {
	return nil
}
