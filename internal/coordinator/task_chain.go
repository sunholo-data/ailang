package coordinator

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// TaskChain manages the pipeline: design-doc → sprint-planner → sprint-executor.
// It handles stage transitions and GitHub notifications.
type TaskChain struct {
	poster  *GitHubPoster
	store   Store
	watcher *ApprovalWatcher
}

// NewTaskChain creates a new task chain manager.
func NewTaskChain(poster *GitHubPoster, store Store, watcher *ApprovalWatcher) *TaskChain {
	tc := &TaskChain{
		poster:  poster,
		store:   store,
		watcher: watcher,
	}

	// Register approval handlers
	if watcher != nil {
		watcher.RegisterHandler(ApprovalEventDesign, tc.OnDesignApproved)
		watcher.RegisterHandler(ApprovalEventSprint, tc.OnSprintApproved)
		watcher.RegisterHandler(ApprovalEventMerge, tc.OnMergeApproved)
		watcher.RegisterHandler(ApprovalEventRevision, tc.OnNeedsRevision)
	}

	return tc
}

// StartTask initializes a new GitHub-linked task at the design stage.
func (tc *TaskChain) StartTask(ctx context.Context, taskID string, issueNum int) error {
	// Link task to GitHub issue
	if err := tc.store.SetTaskGithubIssue(ctx, taskID, issueNum); err != nil {
		return fmt.Errorf("failed to link task to issue: %w", err)
	}

	// Set initial stage
	if err := tc.store.SetTaskStage(ctx, taskID, TaskStageDesign); err != nil {
		return fmt.Errorf("failed to set task stage: %w", err)
	}

	// Start watching the issue for approvals
	if tc.watcher != nil {
		tc.watcher.WatchIssue(issueNum, taskID)
	}

	// Post "working" comment to GitHub
	comment, err := RenderWorkingComment(taskID, "AILANG Coordinator", "Design Document")
	if err != nil {
		log.Printf("[TaskChain] Failed to render working comment: %v", err)
	} else if tc.poster != nil {
		if err := tc.poster.PostComment(issueNum, comment); err != nil {
			log.Printf("[TaskChain] Failed to post working comment: %v", err)
		}
	}

	return nil
}

// OnDesignDocComplete is called when a design document is created.
// Posts a summary to GitHub and adds the needs-design-approval label.
func (tc *TaskChain) OnDesignDocComplete(ctx context.Context, taskID string, result *DesignDocResult) error {
	task, err := tc.store.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	if task.GithubIssue == 0 {
		log.Printf("[TaskChain] Task %s has no linked GitHub issue, skipping notification", taskID)
		return nil
	}

	// Read the design doc content from the worktree
	var designDocContent string
	if result.Path != "" && task.WorktreePath != "" {
		fullPath := filepath.Join(task.WorktreePath, result.Path)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			log.Printf("[TaskChain] Warning: Could not read design doc at %s: %v", fullPath, err)
		} else {
			designDocContent = string(content)
		}
	}

	// Render the design doc complete comment
	data := &CommentData{
		TaskID:           taskID,
		DesignDocPath:    result.Path,
		DesignDocContent: designDocContent,
		Duration:         result.Duration,
		Cost:             result.Cost,
		TokensUsed:       result.TokensUsed,
		InputTokens:      result.InputTokens,
		OutputTokens:     result.OutputTokens,
	}

	comment, err := RenderDesignDocComment(data)
	if err != nil {
		return fmt.Errorf("failed to render design doc comment: %w", err)
	}

	if tc.poster != nil {
		// Post the comment
		if err := tc.poster.PostComment(task.GithubIssue, comment); err != nil {
			return fmt.Errorf("failed to post comment: %w", err)
		}

		// Add the needs-design-approval label
		if err := tc.poster.AddLabel(task.GithubIssue, LabelNeedsDesignApproval); err != nil {
			log.Printf("[TaskChain] Failed to add label: %v", err)
		}
	}

	return nil
}

// OnDesignApproved is called when a design document is approved.
// Transitions the task to the sprint stage and requeues for execution.
func (tc *TaskChain) OnDesignApproved(ctx context.Context, event *ApprovalEvent) error {
	log.Printf("[TaskChain] Design approved for task %s (issue #%d)", event.TaskID, event.IssueNumber)

	// Update stage to sprint planning
	if err := tc.store.SetTaskStage(ctx, event.TaskID, TaskStageSprint); err != nil {
		return fmt.Errorf("failed to update stage: %w", err)
	}

	// Post "working on sprint plan" comment
	comment, err := RenderWorkingComment(event.TaskID, "AILANG Coordinator", "Sprint Planning")
	if err != nil {
		log.Printf("[TaskChain] Failed to render working comment: %v", err)
	} else if tc.poster != nil {
		if err := tc.poster.PostComment(event.IssueNumber, comment); err != nil {
			log.Printf("[TaskChain] Failed to post working comment: %v", err)
		}
	}

	// Requeue the task for execution with the new stage
	// This makes the daemon pick it up again with stage-aware directive
	if err := tc.store.RequeueTask(ctx, event.TaskID); err != nil {
		return fmt.Errorf("failed to requeue task: %w", err)
	}
	log.Printf("[TaskChain] Task %s requeued for sprint planning stage", event.TaskID)

	return nil
}

// OnSprintPlanComplete is called when a sprint plan is created.
// Posts a summary to GitHub and adds the needs-sprint-approval label.
func (tc *TaskChain) OnSprintPlanComplete(ctx context.Context, taskID string, result *SprintPlanResult) error {
	task, err := tc.store.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	if task.GithubIssue == 0 {
		log.Printf("[TaskChain] Task %s has no linked GitHub issue, skipping notification", taskID)
		return nil
	}

	// Read the sprint plan content from the worktree
	var sprintPlanContent string
	if result.Path != "" && task.WorktreePath != "" {
		fullPath := filepath.Join(task.WorktreePath, result.Path)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			log.Printf("[TaskChain] Warning: Could not read sprint plan at %s: %v", fullPath, err)
		} else {
			sprintPlanContent = string(content)
		}
	}

	// Render the sprint plan ready comment
	data := &CommentData{
		TaskID:            taskID,
		SprintPlanPath:    result.Path,
		SprintPlanContent: sprintPlanContent,
		Duration:          result.Duration,
		Cost:              result.Cost,
		TokensUsed:        result.TokensUsed,
		InputTokens:       result.InputTokens,
		OutputTokens:      result.OutputTokens,
	}

	comment, err := RenderSprintPlanComment(data)
	if err != nil {
		return fmt.Errorf("failed to render sprint plan comment: %w", err)
	}

	if tc.poster != nil {
		// Post the comment
		if err := tc.poster.PostComment(task.GithubIssue, comment); err != nil {
			return fmt.Errorf("failed to post comment: %w", err)
		}

		// Add the needs-sprint-approval label
		if err := tc.poster.AddLabel(task.GithubIssue, LabelNeedsSprintApproval); err != nil {
			log.Printf("[TaskChain] Failed to add label: %v", err)
		}
	}

	return nil
}

// OnSprintApproved is called when a sprint plan is approved.
// Transitions the task to the implementation stage and requeues for execution.
func (tc *TaskChain) OnSprintApproved(ctx context.Context, event *ApprovalEvent) error {
	log.Printf("[TaskChain] Sprint approved for task %s (issue #%d)", event.TaskID, event.IssueNumber)

	// Update stage to implementation
	if err := tc.store.SetTaskStage(ctx, event.TaskID, TaskStageImplementation); err != nil {
		return fmt.Errorf("failed to update stage: %w", err)
	}

	// Post "working on implementation" comment
	comment, err := RenderWorkingComment(event.TaskID, "AILANG Coordinator", "Implementation")
	if err != nil {
		log.Printf("[TaskChain] Failed to render working comment: %v", err)
	} else if tc.poster != nil {
		if err := tc.poster.PostComment(event.IssueNumber, comment); err != nil {
			log.Printf("[TaskChain] Failed to post working comment: %v", err)
		}
	}

	// Requeue the task for execution with the new stage
	// This makes the daemon pick it up again with stage-aware directive
	if err := tc.store.RequeueTask(ctx, event.TaskID); err != nil {
		return fmt.Errorf("failed to requeue task: %w", err)
	}
	log.Printf("[TaskChain] Task %s requeued for implementation stage", event.TaskID)

	return nil
}

// OnImplementationComplete is called when implementation is finished.
// Posts a summary to GitHub and adds the needs-merge-approval label.
func (tc *TaskChain) OnImplementationComplete(ctx context.Context, taskID string, result *ImplementResult) error {
	task, err := tc.store.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	if task.GithubIssue == 0 {
		log.Printf("[TaskChain] Task %s has no linked GitHub issue, skipping notification", taskID)
		return nil
	}

	// Update stage to merge
	if err := tc.store.SetTaskStage(ctx, taskID, TaskStageMerge); err != nil {
		log.Printf("[TaskChain] Failed to update stage: %v", err)
	}

	// Render the implementation complete comment
	data := &CommentData{
		TaskID:        taskID,
		BranchName:    result.BranchName,
		WorktreePath:  result.WorktreePath,
		Duration:      result.Duration,
		Cost:          result.Cost,
		TokensUsed:    result.TokensUsed,
		InputTokens:   result.InputTokens,
		OutputTokens:  result.OutputTokens,
		FilesCreated:  result.FilesCreated,
		FilesModified: result.FilesModified,
	}

	comment, err := RenderImplementCompleteComment(data)
	if err != nil {
		return fmt.Errorf("failed to render implementation complete comment: %w", err)
	}

	if tc.poster != nil {
		// Post the comment
		if err := tc.poster.PostComment(task.GithubIssue, comment); err != nil {
			return fmt.Errorf("failed to post comment: %w", err)
		}

		// Add the needs-merge-approval label
		if err := tc.poster.AddLabel(task.GithubIssue, LabelNeedsMergeApproval); err != nil {
			log.Printf("[TaskChain] Failed to add label: %v", err)
		}
	}

	return nil
}

// OnMergeApproved is called when merge is approved.
// Performs the merge and closes the GitHub issue.
func (tc *TaskChain) OnMergeApproved(ctx context.Context, event *ApprovalEvent) error {
	log.Printf("[TaskChain] Merge approved for task %s (issue #%d)", event.TaskID, event.IssueNumber)

	task, err := tc.store.GetTask(ctx, event.TaskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Stop watching the issue
	if tc.watcher != nil {
		tc.watcher.UnwatchIssue(event.IssueNumber)
	}

	// Clear the stage (task is complete)
	if err := tc.store.SetTaskStage(ctx, event.TaskID, TaskStageNone); err != nil {
		log.Printf("[TaskChain] Failed to clear stage: %v", err)
	}

	// Render the merge complete comment
	data := &CommentData{
		TaskID:        event.TaskID,
		BranchName:    task.WorktreeID, // Branch name is stored in WorktreeID
		Duration:      task.Duration,
		Cost:          task.Cost,
		TokensUsed:    task.TokensUsed,
		DesignDocPath: "", // TODO: Store and retrieve from task extras
	}

	comment, err := RenderMergeCompleteComment(data)
	if err != nil {
		log.Printf("[TaskChain] Failed to render merge complete comment: %v", err)
	}

	if tc.poster != nil {
		// Close the issue with the comment
		if err := tc.poster.CloseIssue(event.IssueNumber, comment); err != nil {
			return fmt.Errorf("failed to close issue: %w", err)
		}
	}

	// The actual merge is handled by the daemon's approval workflow
	return nil
}

// OnNeedsRevision is called when revision is requested.
// Pauses the pipeline and notifies via GitHub.
func (tc *TaskChain) OnNeedsRevision(ctx context.Context, event *ApprovalEvent) error {
	log.Printf("[TaskChain] Revision requested for task %s (issue #%d)", event.TaskID, event.IssueNumber)

	task, err := tc.store.GetTask(ctx, event.TaskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Post revision comment
	comment, err := RenderRevisionComment(event.TaskID, string(task.Stage))
	if err != nil {
		log.Printf("[TaskChain] Failed to render revision comment: %v", err)
	} else if tc.poster != nil {
		if err := tc.poster.PostComment(event.IssueNumber, comment); err != nil {
			log.Printf("[TaskChain] Failed to post revision comment: %v", err)
		}
	}

	// Pipeline stays paused until human removes needs-revision and adds approval label
	return nil
}

// OnError is called when an error occurs during any stage.
// Posts an error comment to GitHub.
func (tc *TaskChain) OnError(ctx context.Context, taskID string, errMsg string) error {
	task, err := tc.store.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	if task.GithubIssue == 0 {
		return nil
	}

	// Post error comment
	comment, err := RenderErrorComment(taskID, string(task.Stage), errMsg)
	if err != nil {
		log.Printf("[TaskChain] Failed to render error comment: %v", err)
	} else if tc.poster != nil {
		if err := tc.poster.PostComment(task.GithubIssue, comment); err != nil {
			log.Printf("[TaskChain] Failed to post error comment: %v", err)
		}
	}

	return nil
}

// DesignDocResult contains the result of design document creation.
type DesignDocResult struct {
	Path         string
	Duration     time.Duration
	Cost         float64
	TokensUsed   int
	InputTokens  int
	OutputTokens int
}

// SprintPlanResult contains the result of sprint plan creation.
type SprintPlanResult struct {
	Path         string
	Duration     time.Duration
	Cost         float64
	TokensUsed   int
	InputTokens  int
	OutputTokens int
}

// ImplementResult contains the result of implementation.
type ImplementResult struct {
	BranchName    string
	WorktreePath  string
	Duration      time.Duration
	Cost          float64
	TokensUsed    int
	InputTokens   int
	OutputTokens  int
	FilesCreated  []string
	FilesModified []string
}
