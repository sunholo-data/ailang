package coordinator

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/messaging"
)

// TaskChain manages the pipeline: design-doc → sprint-planner → sprint-executor.
// It handles stage transitions and GitHub notifications.
type TaskChain struct {
	poster   *GitHubPoster
	store    Store
	watcher  *ApprovalWatcher
	registry *AgentRegistry         // For config-driven workflows (M-GENERIC-PIPELINE)
	msgStore messaging.MessageStore // For handoff messages (M-GENERIC-PIPELINE)
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

// SetAgentRegistry sets the agent registry for config-driven workflows.
func (tc *TaskChain) SetAgentRegistry(registry *AgentRegistry) {
	tc.registry = registry
}

// SetMessageStore sets the message store for handoff messages.
func (tc *TaskChain) SetMessageStore(msgStore messaging.MessageStore) {
	tc.msgStore = msgStore
}

// StartTask initializes a new GitHub-linked task at the design stage.
// It first claims the issue to prevent race conditions with other coordinator instances.
func (tc *TaskChain) StartTask(ctx context.Context, taskID string, issueNum int) error {
	// Get the task to access the GithubRepo field
	task, err := tc.store.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	githubRepo := task.GithubRepo

	// Claim the issue first to prevent race conditions
	if tc.poster != nil {
		if err := tc.poster.ClaimIssueInRepo(githubRepo, issueNum); err != nil {
			log.Printf("[TaskChain] Issue #%d already claimed or claim failed: %v", issueNum, err)
			return fmt.Errorf("failed to claim issue #%d: %w", issueNum, err)
		}
		log.Printf("[TaskChain] Claimed issue #%d for task %s in repo %s", issueNum, taskID, githubRepo)
	}

	// Link task to GitHub issue
	if err := tc.store.SetTaskGithubIssue(ctx, taskID, issueNum); err != nil {
		if tc.poster != nil {
			_ = tc.poster.ReleaseIssueInRepo(githubRepo, issueNum)
		}
		return fmt.Errorf("failed to link task to issue: %w", err)
	}

	// Set initial stage
	if err := tc.store.SetTaskStage(ctx, taskID, TaskStageDesign); err != nil {
		if tc.poster != nil {
			_ = tc.poster.ReleaseIssueInRepo(githubRepo, issueNum)
		}
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
		if err := tc.poster.PostCommentInRepo(githubRepo, issueNum, comment); err != nil {
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

	// Store the design doc path for later use in merge comment
	if result.Path != "" {
		if err := tc.store.SetTaskDesignDocPath(ctx, taskID, result.Path); err != nil {
			log.Printf("[TaskChain] Warning: Failed to store design doc path: %v", err)
		}
	}

	if task.GithubIssue == 0 {
		log.Printf("[TaskChain] Task %s has no linked GitHub issue, skipping notification", taskID)
		return nil
	}

	// M-COORD-ARTIFACT-DISCOVERY: Handle case where no .md file found
	if result.Path == "" && len(result.AllArtifacts) > 0 {
		log.Printf("[TaskChain] No .md file found for task %s, but %d artifacts discovered", taskID, len(result.AllArtifacts))
		if tc.poster != nil {
			fileList := ""
			for i, f := range result.AllArtifacts {
				if i >= 20 {
					fileList += fmt.Sprintf("- ... and %d more files\n", len(result.AllArtifacts)-20)
					break
				}
				fileList += fmt.Sprintf("- `%s`\n", f)
			}
			infoComment := fmt.Sprintf(
				"## 📁 Files Changed\n\n"+
					"**Task:** %s\n"+
					"**Stage:** Design Doc Creation\n"+
					"**Duration:** %v\n\n"+
					"**%d files discovered** (no `.md` design doc found for inline display):\n\n%s\n"+
					"Review changes in the dashboard diff viewer.",
				taskID, result.Duration, len(result.AllArtifacts), fileList,
			)
			if err := tc.poster.PostCommentInRepo(task.GithubRepo, task.GithubIssue, infoComment); err != nil {
				log.Printf("[TaskChain] Failed to post files comment: %v", err)
			}
		}
	} else if result.Path == "" && len(result.AllArtifacts) == 0 {
		log.Printf("[TaskChain] No artifacts found for task %s, posting failure message", taskID)
		if tc.poster != nil {
			failureComment := fmt.Sprintf(
				"## ❌ No Changes Detected\n\n"+
					"**Task:** %s\n"+
					"**Stage:** Design Doc Creation\n"+
					"**Duration:** %v\n\n"+
					"No files were created or modified. The agent may have:\n"+
					"- Encountered an error\n"+
					"- Not made any changes\n\n"+
					"Check the agent output for details.",
				taskID, result.Duration,
			)
			if err := tc.poster.PostCommentInRepo(task.GithubRepo, task.GithubIssue, failureComment); err != nil {
				log.Printf("[TaskChain] Failed to post failure comment: %v", err)
			}
		}
		return fmt.Errorf("no artifacts found for task %s", taskID)
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
		if err := tc.poster.PostCommentInRepo(task.GithubRepo, task.GithubIssue, comment); err != nil {
			return fmt.Errorf("failed to post comment: %w", err)
		}

		approval := DefaultApprovalConfig("design-doc-creator")
		if approval != nil && approval.NeedsLabel != "" {
			if err := tc.poster.AddLabelInRepo(task.GithubRepo, task.GithubIssue, approval.NeedsLabel); err != nil {
				log.Printf("[TaskChain] Failed to add label: %v", err)
			}
		}
	}

	return nil
}

// OnDesignApproved is called when a design document is approved.
func (tc *TaskChain) OnDesignApproved(ctx context.Context, event *ApprovalEvent) error {
	log.Printf("[TaskChain] Design approved for task %s (issue #%d)", event.TaskID, event.IssueNumber)

	task, err := tc.store.GetTask(ctx, event.TaskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	if err := tc.store.SetTaskStage(ctx, event.TaskID, TaskStageSprint); err != nil {
		return fmt.Errorf("failed to update stage: %w", err)
	}

	comment, err := RenderWorkingComment(event.TaskID, "AILANG Coordinator", "Sprint Planning")
	if err != nil {
		log.Printf("[TaskChain] Failed to render working comment: %v", err)
	} else if tc.poster != nil {
		if err := tc.poster.PostCommentInRepo(task.GithubRepo, event.IssueNumber, comment); err != nil {
			log.Printf("[TaskChain] Failed to post working comment: %v", err)
		}
	}

	if err := tc.store.RequeueTask(ctx, event.TaskID); err != nil {
		return fmt.Errorf("failed to requeue task: %w", err)
	}
	log.Printf("[TaskChain] Task %s requeued for sprint planning stage", event.TaskID)

	return nil
}

// OnSprintPlanComplete is called when a sprint plan is created.
func (tc *TaskChain) OnSprintPlanComplete(ctx context.Context, taskID string, result *SprintPlanResult) error {
	task, err := tc.store.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	if result.Path != "" {
		if err := tc.store.SetTaskSprintPlanPath(ctx, taskID, result.Path); err != nil {
			log.Printf("[TaskChain] Warning: Failed to store sprint plan path: %v", err)
		}
	}

	if task.GithubIssue == 0 {
		log.Printf("[TaskChain] Task %s has no linked GitHub issue, skipping notification", taskID)
		return nil
	}

	if result.Path == "" && len(result.AllArtifacts) > 0 {
		log.Printf("[TaskChain] No .md file found for task %s sprint plan, but %d artifacts discovered", taskID, len(result.AllArtifacts))
		if tc.poster != nil {
			fileList := ""
			for i, f := range result.AllArtifacts {
				if i >= 20 {
					fileList += fmt.Sprintf("- ... and %d more files\n", len(result.AllArtifacts)-20)
					break
				}
				fileList += fmt.Sprintf("- `%s`\n", f)
			}
			infoComment := fmt.Sprintf(
				"## 📁 Files Changed\n\n"+
					"**Task:** %s\n"+
					"**Stage:** Sprint Planning\n"+
					"**Duration:** %v\n\n"+
					"**%d files discovered** (no `.md` sprint plan found for inline display):\n\n%s\n"+
					"Review changes in the dashboard diff viewer.",
				taskID, result.Duration, len(result.AllArtifacts), fileList,
			)
			if err := tc.poster.PostCommentInRepo(task.GithubRepo, task.GithubIssue, infoComment); err != nil {
				log.Printf("[TaskChain] Failed to post files comment: %v", err)
			}
		}
	} else if result.Path == "" && len(result.AllArtifacts) == 0 {
		log.Printf("[TaskChain] No artifacts found for task %s sprint plan, posting failure message", taskID)
		if tc.poster != nil {
			failureComment := fmt.Sprintf(
				"## ❌ No Changes Detected\n\n"+
					"**Task:** %s\n"+
					"**Stage:** Sprint Planning\n"+
					"**Duration:** %v\n\n"+
					"No files were created or modified. The agent may have:\n"+
					"- Encountered an error\n"+
					"- Not made any changes\n\n"+
					"Check the agent output for details.",
				taskID, result.Duration,
			)
			if err := tc.poster.PostCommentInRepo(task.GithubRepo, task.GithubIssue, failureComment); err != nil {
				log.Printf("[TaskChain] Failed to post failure comment: %v", err)
			}
		}
		return fmt.Errorf("no artifacts found for task %s", taskID)
	}

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
		if err := tc.poster.PostCommentInRepo(task.GithubRepo, task.GithubIssue, comment); err != nil {
			return fmt.Errorf("failed to post comment: %w", err)
		}

		approval := DefaultApprovalConfig("sprint-planner")
		if approval != nil && approval.NeedsLabel != "" {
			if err := tc.poster.AddLabelInRepo(task.GithubRepo, task.GithubIssue, approval.NeedsLabel); err != nil {
				log.Printf("[TaskChain] Failed to add label: %v", err)
			}
		}
	}

	return nil
}

// OnSprintApproved is called when a sprint plan is approved.
func (tc *TaskChain) OnSprintApproved(ctx context.Context, event *ApprovalEvent) error {
	log.Printf("[TaskChain] Sprint approved for task %s (issue #%d)", event.TaskID, event.IssueNumber)

	task, err := tc.store.GetTask(ctx, event.TaskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	if err := tc.store.SetTaskStage(ctx, event.TaskID, TaskStageImplementation); err != nil {
		return fmt.Errorf("failed to update stage: %w", err)
	}

	comment, err := RenderWorkingComment(event.TaskID, "AILANG Coordinator", "Implementation")
	if err != nil {
		log.Printf("[TaskChain] Failed to render working comment: %v", err)
	} else if tc.poster != nil {
		if err := tc.poster.PostCommentInRepo(task.GithubRepo, event.IssueNumber, comment); err != nil {
			log.Printf("[TaskChain] Failed to post working comment: %v", err)
		}
	}

	if err := tc.store.RequeueTask(ctx, event.TaskID); err != nil {
		return fmt.Errorf("failed to requeue task: %w", err)
	}
	log.Printf("[TaskChain] Task %s requeued for implementation stage", event.TaskID)

	return nil
}

// OnAgentApproved is called when an agent's work is approved via config-driven workflow.
func (tc *TaskChain) OnAgentApproved(ctx context.Context, event *ApprovalEvent, agentID string) error {
	log.Printf("[TaskChain] Agent %s work approved for task %s (issue #%d)",
		agentID, event.TaskID, event.IssueNumber)

	task, err := tc.store.GetTask(ctx, event.TaskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	if tc.registry == nil {
		return fmt.Errorf("agent registry not available")
	}

	agent := tc.registry.GetAgentByID(agentID)
	if agent == nil {
		return fmt.Errorf("agent %s not found in registry", agentID)
	}

	if len(agent.TriggerOnComplete) == 0 {
		log.Printf("[TaskChain] Agent %s has no trigger_on_complete, approval complete", agentID)
		return nil
	}

	nextAgents := strings.Join(agent.TriggerOnComplete, ", ")
	comment := fmt.Sprintf("**Approval Complete** - Work by %s approved.\n\n"+
		"Triggering next stage: %s", agent.Label, nextAgents)
	if tc.poster != nil {
		if err := tc.poster.PostCommentInRepo(task.GithubRepo, event.IssueNumber, comment); err != nil {
			log.Printf("[TaskChain] Failed to post approval comment: %v", err)
		}
	}

	for _, targetAgentID := range agent.TriggerOnComplete {
		targetAgent := tc.registry.GetAgentByID(targetAgentID)
		if targetAgent == nil {
			log.Printf("[TaskChain] Warning: target agent %s not found", targetAgentID)
			continue
		}

		log.Printf("[TaskChain] Triggering handoff: %s → %s for task %s",
			agentID, targetAgentID, event.TaskID)

		task, err := tc.store.GetTask(ctx, event.TaskID)
		if err != nil {
			log.Printf("[TaskChain] Warning: could not get task for handoff: %v", err)
			continue
		}

		if tc.msgStore != nil {
			handoffContent := fmt.Sprintf("**Handoff from %s**\n\n"+
				"Task: %s\n"+
				"GitHub Issue: #%d\n"+
				"Original Request: %s\n\n"+
				"Previous work has been approved. Please continue.",
				agent.Label, event.TaskID, event.IssueNumber, task.Content)

			metadata := fmt.Sprintf(`{"parent_task_id":"%s","source_agent":"%s","target_agent":"%s","github_issue":%d}`,
				event.TaskID, agentID, targetAgentID, event.IssueNumber)

			_, err := tc.msgStore.CreateMessage(
				"",                               // New thread
				"ailang_instance", "coordinator", // from
				targetAgent.Inbox, targetAgent.ID, // to
				"handoff",
				handoffContent,
				metadata,
			)
			if err != nil {
				log.Printf("[TaskChain] Warning: failed to send handoff message: %v", err)
			}
		}
	}

	return nil
}

// OnImplementationComplete is called when implementation is finished.
func (tc *TaskChain) OnImplementationComplete(ctx context.Context, taskID string, result *ImplementResult) error {
	task, err := tc.store.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	if task.GithubIssue == 0 {
		log.Printf("[TaskChain] Task %s has no linked GitHub issue, skipping notification", taskID)
		return nil
	}

	if err := tc.store.SetTaskStage(ctx, taskID, TaskStageMerge); err != nil {
		log.Printf("[TaskChain] Failed to update stage: %v", err)
	}

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
		if err := tc.poster.PostCommentInRepo(task.GithubRepo, task.GithubIssue, comment); err != nil {
			return fmt.Errorf("failed to post comment: %w", err)
		}

		approval := DefaultApprovalConfig("sprint-executor")
		if approval != nil && approval.NeedsLabel != "" {
			if err := tc.poster.AddLabelInRepo(task.GithubRepo, task.GithubIssue, approval.NeedsLabel); err != nil {
				log.Printf("[TaskChain] Failed to add label: %v", err)
			}
		}
	}

	return nil
}

// OnMergeApproved is called when merge is approved.
func (tc *TaskChain) OnMergeApproved(ctx context.Context, event *ApprovalEvent) error {
	log.Printf("[TaskChain] Merge approved for task %s (issue #%d)", event.TaskID, event.IssueNumber)

	task, err := tc.store.GetTask(ctx, event.TaskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	if tc.watcher != nil {
		tc.watcher.UnwatchIssue(event.IssueNumber)
	}

	if err := tc.store.SetTaskStage(ctx, event.TaskID, TaskStageNone); err != nil {
		log.Printf("[TaskChain] Failed to clear stage: %v", err)
	}

	data := &CommentData{
		TaskID:         event.TaskID,
		BranchName:     task.WorktreeID,
		Duration:       task.Duration,
		Cost:           task.Cost,
		TokensUsed:     task.TokensUsed,
		DesignDocPath:  task.DesignDocPath,
		SprintPlanPath: task.SprintPlanPath,
	}

	comment, err := RenderMergeCompleteComment(data)
	if err != nil {
		log.Printf("[TaskChain] Failed to render merge complete comment: %v", err)
	}

	if tc.poster != nil {
		if err := tc.poster.CloseIssueInRepo(task.GithubRepo, event.IssueNumber, comment); err != nil {
			return fmt.Errorf("failed to close issue: %w", err)
		}

		if err := tc.poster.ReleaseIssueInRepo(task.GithubRepo, event.IssueNumber); err != nil {
			log.Printf("[TaskChain] Warning: Failed to release issue claim: %v", err)
		}
	}

	return nil
}

// OnNeedsRevision is called when revision is requested.
func (tc *TaskChain) OnNeedsRevision(ctx context.Context, event *ApprovalEvent) error {
	log.Printf("[TaskChain] Revision requested for task %s (issue #%d)", event.TaskID, event.IssueNumber)

	task, err := tc.store.GetTask(ctx, event.TaskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	comment, err := RenderRevisionComment(event.TaskID, string(task.Stage))
	if err != nil {
		log.Printf("[TaskChain] Failed to render revision comment: %v", err)
	} else if tc.poster != nil {
		if err := tc.poster.PostCommentInRepo(task.GithubRepo, event.IssueNumber, comment); err != nil {
			log.Printf("[TaskChain] Failed to post revision comment: %v", err)
		}
	}

	return nil
}

// OnError is called when an error occurs during any stage.
func (tc *TaskChain) OnError(ctx context.Context, taskID string, errMsg string) error {
	task, err := tc.store.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	if task.GithubIssue == 0 {
		return nil
	}

	comment, err := RenderErrorComment(taskID, string(task.Stage), errMsg)
	if err != nil {
		log.Printf("[TaskChain] Failed to render error comment: %v", err)
	} else if tc.poster != nil {
		if err := tc.poster.PostCommentInRepo(task.GithubRepo, task.GithubIssue, comment); err != nil {
			log.Printf("[TaskChain] Failed to post error comment: %v", err)
		}
	}

	if tc.poster != nil {
		if err := tc.poster.ReleaseIssueInRepo(task.GithubRepo, task.GithubIssue); err != nil {
			log.Printf("[TaskChain] Warning: Failed to release issue claim: %v", err)
		}
	}

	return nil
}

// DesignDocResult contains the result of design document creation.
type DesignDocResult struct {
	Path         string   // Path to .md file (may be empty if no .md found)
	AllArtifacts []string // All discovered artifacts (any file type)
	Duration     time.Duration
	Cost         float64
	TokensUsed   int
	InputTokens  int
	OutputTokens int
}

// SprintPlanResult contains the result of sprint plan creation.
type SprintPlanResult struct {
	Path         string   // Path to .md file (may be empty if no .md found)
	AllArtifacts []string // All discovered artifacts (any file type)
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
