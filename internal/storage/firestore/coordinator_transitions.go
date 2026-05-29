package firestore

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// --- Task State Transitions ---

func (s *CoordinatorStore) MarkTaskQueued(ctx context.Context, id string) error {
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "status", Value: string(coordinator.TaskStatusQueued)},
	})
	if err == nil {
		s.invalidateStatsCache()
	}
	return err
}

func (s *CoordinatorStore) MarkTaskRunning(ctx context.Context, id, provider, worktreeID string) error {
	now := time.Now()
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "status", Value: string(coordinator.TaskStatusRunning)},
		{Path: "provider", Value: provider},
		{Path: "worktree_id", Value: worktreeID},
		{Path: "started_at", Value: now},
	})
	if err == nil {
		s.invalidateStatsCache()
	}
	return err
}

func (s *CoordinatorStore) MarkTaskPendingApproval(ctx context.Context, id, worktreePath, worktreeBranch, baseBranch, baseCommit string, result *coordinator.ExecuteResult) error {
	now := time.Now()
	updates := []firestore.Update{
		{Path: "status", Value: string(coordinator.TaskStatusPendingApproval)},
		{Path: "worktree_path", Value: worktreePath},
		{Path: "base_branch", Value: baseBranch},
		{Path: "base_commit", Value: baseCommit},
		{Path: "completed_at", Value: now},
	}
	if result != nil {
		updates = append(updates,
			firestore.Update{Path: "output", Value: result.Output},
			firestore.Update{Path: "cost", Value: result.Cost},
			firestore.Update{Path: "tokens_used", Value: result.TokensUsed},
			firestore.Update{Path: "input_tokens", Value: result.InputTokens},
			firestore.Update{Path: "output_tokens", Value: result.OutputTokens},
			firestore.Update{Path: "session_id", Value: result.SessionID},
			firestore.Update{Path: "duration", Value: int64(result.Duration)},
		)
		if result.ArtifactGCSPath != "" {
			updates = append(updates, firestore.Update{Path: "artifact_gcs_path", Value: result.ArtifactGCSPath})
		}
	}
	_, err := s.client.Doc(collTasks, id).Update(ctx, updates)
	if err != nil {
		return err
	}
	s.invalidateStatsCache()

	// Update in-memory cost counter.
	if result != nil && result.Cost > 0 {
		task, taskErr := s.GetTask(ctx, id)
		if taskErr == nil && task != nil {
			s.addCost(task.Provider, result.Cost)
		}
	}
	return nil
}

func (s *CoordinatorStore) MarkTaskCompleted(ctx context.Context, id string, result *coordinator.ExecuteResult) error {
	now := time.Now()
	updates := []firestore.Update{
		{Path: "status", Value: string(coordinator.TaskStatusCompleted)},
		{Path: "completed_at", Value: now},
	}
	if result != nil {
		updates = append(updates,
			firestore.Update{Path: "output", Value: result.Output},
			firestore.Update{Path: "cost", Value: result.Cost},
			firestore.Update{Path: "tokens_used", Value: result.TokensUsed},
			firestore.Update{Path: "input_tokens", Value: result.InputTokens},
			firestore.Update{Path: "output_tokens", Value: result.OutputTokens},
			firestore.Update{Path: "session_id", Value: result.SessionID},
			firestore.Update{Path: "duration", Value: int64(result.Duration)},
		)
		if !result.Success {
			updates = append(updates, firestore.Update{Path: "error", Value: result.Error})
		}
		if result.ArtifactGCSPath != "" {
			updates = append(updates, firestore.Update{Path: "artifact_gcs_path", Value: result.ArtifactGCSPath})
		}
	}
	_, err := s.client.Doc(collTasks, id).Update(ctx, updates)
	if err != nil {
		return err
	}
	s.invalidateStatsCache()

	// Update in-memory cost counter.
	if result != nil && result.Cost > 0 {
		task, taskErr := s.GetTask(ctx, id)
		if taskErr == nil && task != nil {
			s.addCost(task.Provider, result.Cost)
		}
	}
	return nil
}

func (s *CoordinatorStore) MarkTaskFailed(ctx context.Context, id string, taskErr error) error {
	now := time.Now()
	errMsg := ""
	if taskErr != nil {
		errMsg = taskErr.Error()
	}
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "status", Value: string(coordinator.TaskStatusFailed)},
		{Path: "error", Value: errMsg},
		{Path: "completed_at", Value: now},
	})
	if err == nil {
		s.invalidateStatsCache()
	}
	return err
}

func (s *CoordinatorStore) MarkTaskRejected(ctx context.Context, id string) error {
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "status", Value: string(coordinator.TaskStatusRejected)},
		{Path: "completed_at", Value: time.Now()},
	})
	if err == nil {
		s.invalidateStatsCache()
	}
	return err
}

func (s *CoordinatorStore) MarkTaskCancelled(ctx context.Context, id string) error {
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "status", Value: string(coordinator.TaskStatusCancelled)},
		{Path: "completed_at", Value: time.Now()},
	})
	if err == nil {
		s.invalidateStatsCache()
	}
	return err
}

func (s *CoordinatorStore) RequeueTask(ctx context.Context, id string) error {
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "status", Value: string(coordinator.TaskStatusPending)},
		{Path: "started_at", Value: nil},
		{Path: "completed_at", Value: nil},
		{Path: "error", Value: ""},
		{Path: "output", Value: ""},
	})
	if err == nil {
		s.invalidateStatsCache()
	}
	return err
}

func (s *CoordinatorStore) ResetTaskToPending(ctx context.Context, id string) error {
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "status", Value: string(coordinator.TaskStatusPending)},
		{Path: "worktree_id", Value: ""},
		{Path: "provider", Value: ""},
	})
	if err == nil {
		s.invalidateStatsCache()
	}
	return err
}

// --- Duplicate Detection ---

func (s *CoordinatorStore) FindDuplicateTask(ctx context.Context, fingerprint uint64, _ float64) (*coordinator.TaskRecord, error) {
	// Firestore doesn't support bitwise operations, so we do exact fingerprint match
	iter := s.client.Collection(collTasks).
		Where("fingerprint", "==", int64(fingerprint)).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return mapToTask(doc.Data()), nil
}

func (s *CoordinatorStore) SetTaskFingerprint(ctx context.Context, id string, fingerprint uint64) error {
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "fingerprint", Value: int64(fingerprint)},
	})
	return err
}

// --- Thread & Chain Linking ---

func (s *CoordinatorStore) SetTaskThreadID(ctx context.Context, id, threadID string) error {
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "thread_id", Value: threadID},
	})
	return err
}

func (s *CoordinatorStore) UpdateTaskChainInfo(ctx context.Context, id, chainID, stageID string) error {
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "chain_id", Value: chainID},
		{Path: "stage_id", Value: stageID},
	})
	return err
}

func (s *CoordinatorStore) GetTaskAgentInfo(ctx context.Context, taskID string) (agentID, inbox, title string, err error) {
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return "", "", "", err
	}
	// By convention, agent_id == inbox in agent config
	return task.AgentID, task.AgentID, task.Title, nil
}

// --- GitHub Integration ---

func (s *CoordinatorStore) SetTaskGithubIssue(ctx context.Context, id string, issueNum int) error {
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "github_issue", Value: issueNum},
	})
	return err
}

func (s *CoordinatorStore) SetTaskStage(ctx context.Context, id string, stage coordinator.TaskStage) error {
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "stage", Value: string(stage)},
	})
	return err
}

func (s *CoordinatorStore) SetTaskDesignDocPath(ctx context.Context, id, path string) error {
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "design_doc_path", Value: path},
	})
	return err
}

func (s *CoordinatorStore) SetTaskSprintPlanPath(ctx context.Context, id, path string) error {
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "sprint_plan_path", Value: path},
	})
	return err
}

func (s *CoordinatorStore) GetTasksByGithubIssue(ctx context.Context, issueNum int) ([]*coordinator.TaskRecord, error) {
	iter := s.client.Collection(collTasks).
		Where("github_issue", "==", issueNum).
		OrderBy("created_at", firestore.Desc).
		Documents(ctx)
	defer iter.Stop()

	var tasks []*coordinator.TaskRecord
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, mapToTask(doc.Data()))
	}
	return tasks, nil
}

func (s *CoordinatorStore) GetTasksByStage(ctx context.Context, stage coordinator.TaskStage) ([]*coordinator.TaskRecord, error) {
	iter := s.client.Collection(collTasks).
		Where("stage", "==", string(stage)).
		OrderBy("created_at", firestore.Desc).
		Documents(ctx)
	defer iter.Stop()

	var tasks []*coordinator.TaskRecord
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, mapToTask(doc.Data()))
	}
	return tasks, nil
}

// --- Resource Metrics ---

func (s *CoordinatorStore) UpdateTaskMetrics(ctx context.Context, id string, peakCPU, peakMemory float64) error {
	_, err := s.client.Doc(collTasks, id).Update(ctx, []firestore.Update{
		{Path: "peak_cpu", Value: peakCPU},
		{Path: "peak_memory_mb", Value: peakMemory},
	})
	return err
}
