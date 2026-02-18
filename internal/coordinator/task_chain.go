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
	// If another coordinator already claimed it, we skip this task
	if tc.poster != nil {
		if err := tc.poster.ClaimIssueInRepo(githubRepo, issueNum); err != nil {
			log.Printf("[TaskChain] Issue #%d already claimed or claim failed: %v", issueNum, err)
			return fmt.Errorf("failed to claim issue #%d: %w", issueNum, err)
		}
		log.Printf("[TaskChain] Claimed issue #%d for task %s in repo %s", issueNum, taskID, githubRepo)
	}

	// Link task to GitHub issue
	if err := tc.store.SetTaskGithubIssue(ctx, taskID, issueNum); err != nil {
		// Release claim on failure
		if tc.poster != nil {
			_ = tc.poster.ReleaseIssueInRepo(githubRepo, issueNum)
		}
		return fmt.Errorf("failed to link task to issue: %w", err)
	}

	// Set initial stage
	if err := tc.store.SetTaskStage(ctx, taskID, TaskStageDesign); err != nil {
		// Release claim on failure
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
// If no artifact was found, posts a failure message instead.
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
	// Don't fail - show all artifacts instead for review
	if result.Path == "" && len(result.AllArtifacts) > 0 {
		log.Printf("[TaskChain] No .md file found for task %s, but %d artifacts discovered", taskID, len(result.AllArtifacts))
		if tc.poster != nil {
			// Build file list (limit to 20 for GitHub comment)
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
		// Don't return error - proceed with approval workflow
	} else if result.Path == "" && len(result.AllArtifacts) == 0 {
		// Only fail if NO files were changed at all
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
		// Post the comment
		if err := tc.poster.PostCommentInRepo(task.GithubRepo, task.GithubIssue, comment); err != nil {
			return fmt.Errorf("failed to post comment: %w", err)
		}

		// Add the needs-approval label (config-driven)
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
// Transitions the task to the sprint stage and requeues for execution.
func (tc *TaskChain) OnDesignApproved(ctx context.Context, event *ApprovalEvent) error {
	log.Printf("[TaskChain] Design approved for task %s (issue #%d)", event.TaskID, event.IssueNumber)

	// Get the task to access GithubRepo
	task, err := tc.store.GetTask(ctx, event.TaskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Update stage to sprint planning
	if err := tc.store.SetTaskStage(ctx, event.TaskID, TaskStageSprint); err != nil {
		return fmt.Errorf("failed to update stage: %w", err)
	}

	// Post "working on sprint plan" comment
	comment, err := RenderWorkingComment(event.TaskID, "AILANG Coordinator", "Sprint Planning")
	if err != nil {
		log.Printf("[TaskChain] Failed to render working comment: %v", err)
	} else if tc.poster != nil {
		if err := tc.poster.PostCommentInRepo(task.GithubRepo, event.IssueNumber, comment); err != nil {
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
// If no artifact was found, posts a failure message instead.
func (tc *TaskChain) OnSprintPlanComplete(ctx context.Context, taskID string, result *SprintPlanResult) error {
	task, err := tc.store.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Store the sprint plan path for later use in merge comment
	if result.Path != "" {
		if err := tc.store.SetTaskSprintPlanPath(ctx, taskID, result.Path); err != nil {
			log.Printf("[TaskChain] Warning: Failed to store sprint plan path: %v", err)
		}
	}

	if task.GithubIssue == 0 {
		log.Printf("[TaskChain] Task %s has no linked GitHub issue, skipping notification", taskID)
		return nil
	}

	// M-COORD-ARTIFACT-DISCOVERY: Handle case where no .md file found
	// Don't fail - show all artifacts instead for review
	if result.Path == "" && len(result.AllArtifacts) > 0 {
		log.Printf("[TaskChain] No .md file found for task %s sprint plan, but %d artifacts discovered", taskID, len(result.AllArtifacts))
		if tc.poster != nil {
			// Build file list (limit to 20 for GitHub comment)
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
		// Don't return error - proceed with approval workflow
	} else if result.Path == "" && len(result.AllArtifacts) == 0 {
		// Only fail if NO files were changed at all
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
		if err := tc.poster.PostCommentInRepo(task.GithubRepo, task.GithubIssue, comment); err != nil {
			return fmt.Errorf("failed to post comment: %w", err)
		}

		// Add the needs-approval label (config-driven)
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
// Transitions the task to the implementation stage and requeues for execution.
func (tc *TaskChain) OnSprintApproved(ctx context.Context, event *ApprovalEvent) error {
	log.Printf("[TaskChain] Sprint approved for task %s (issue #%d)", event.TaskID, event.IssueNumber)

	// Get the task to access GithubRepo
	task, err := tc.store.GetTask(ctx, event.TaskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Update stage to implementation
	if err := tc.store.SetTaskStage(ctx, event.TaskID, TaskStageImplementation); err != nil {
		return fmt.Errorf("failed to update stage: %w", err)
	}

	// Post "working on implementation" comment
	comment, err := RenderWorkingComment(event.TaskID, "AILANG Coordinator", "Implementation")
	if err != nil {
		log.Printf("[TaskChain] Failed to render working comment: %v", err)
	} else if tc.poster != nil {
		if err := tc.poster.PostCommentInRepo(task.GithubRepo, event.IssueNumber, comment); err != nil {
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

// OnAgentApproved is called when an agent's work is approved via config-driven workflow.
// Triggers handoffs to the next agent(s) based on trigger_on_complete configuration.
// This replaces the stage-specific approved handlers for the generic pipeline.
func (tc *TaskChain) OnAgentApproved(ctx context.Context, event *ApprovalEvent, agentID string) error {
	log.Printf("[TaskChain] Agent %s work approved for task %s (issue #%d)",
		agentID, event.TaskID, event.IssueNumber)

	// Get the task to access GithubRepo
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

	// Check for handoff targets
	if len(agent.TriggerOnComplete) == 0 {
		log.Printf("[TaskChain] Agent %s has no trigger_on_complete, approval complete", agentID)
		return nil
	}

	// Post "triggering next agent" comment
	nextAgents := strings.Join(agent.TriggerOnComplete, ", ")
	comment := fmt.Sprintf("**Approval Complete** - Work by %s approved.\n\n"+
		"Triggering next stage: %s", agent.Label, nextAgents)
	if tc.poster != nil {
		if err := tc.poster.PostCommentInRepo(task.GithubRepo, event.IssueNumber, comment); err != nil {
			log.Printf("[TaskChain] Failed to post approval comment: %v", err)
		}
	}

	// Trigger handoffs to next agents
	for _, targetAgentID := range agent.TriggerOnComplete {
		targetAgent := tc.registry.GetAgentByID(targetAgentID)
		if targetAgent == nil {
			log.Printf("[TaskChain] Warning: target agent %s not found", targetAgentID)
			continue
		}

		log.Printf("[TaskChain] Triggering handoff: %s → %s for task %s",
			agentID, targetAgentID, event.TaskID)

		// Update the task's agent_id to the next agent
		// This enables the task to be picked up by the correct agent's inbox
		task, err := tc.store.GetTask(ctx, event.TaskID)
		if err != nil {
			log.Printf("[TaskChain] Warning: could not get task for handoff: %v", err)
			continue
		}

		// Create a handoff message to the target agent's inbox
		// This will be processed by the daemon's message polling
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
		if err := tc.poster.PostCommentInRepo(task.GithubRepo, task.GithubIssue, comment); err != nil {
			return fmt.Errorf("failed to post comment: %w", err)
		}

		// Add the needs-approval label (config-driven)
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
		TaskID:         event.TaskID,
		BranchName:     task.WorktreeID, // Branch name is stored in WorktreeID
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
		// Close the issue with the comment
		if err := tc.poster.CloseIssueInRepo(task.GithubRepo, event.IssueNumber, comment); err != nil {
			return fmt.Errorf("failed to close issue: %w", err)
		}

		// Release the claim label since task is complete
		if err := tc.poster.ReleaseIssueInRepo(task.GithubRepo, event.IssueNumber); err != nil {
			log.Printf("[TaskChain] Warning: Failed to release issue claim: %v", err)
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
		if err := tc.poster.PostCommentInRepo(task.GithubRepo, event.IssueNumber, comment); err != nil {
			log.Printf("[TaskChain] Failed to post revision comment: %v", err)
		}
	}

	// Pipeline stays paused until human removes needs-revision and adds approval label
	return nil
}

// OnError is called when an error occurs during any stage.
// Posts an error comment to GitHub and releases the claim.
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
		if err := tc.poster.PostCommentInRepo(task.GithubRepo, task.GithubIssue, comment); err != nil {
			log.Printf("[TaskChain] Failed to post error comment: %v", err)
		}
	}

	// Release the claim label so the issue can be retried
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

// =============================================================================
// Generic Agent Completion Handler (M-GENERIC-PIPELINE)
// =============================================================================
// These handlers support config-driven agent pipelines, replacing the need for
// hardcoded stage-to-agent mappings. Any agent with an ApprovalConfig will
// automatically get the right GitHub labels and comments.

// AgentResult contains the unified result of any agent completion.
// This struct supports all agent types (design-doc-creator, sprint-planner,
// sprint-executor, or any custom agent).
type AgentResult struct {
	// Artifact paths (set whichever is relevant)
	ArtifactPath    string   // Primary artifact (design doc, sprint plan, etc.)
	ArtifactContent string   // Optional: content for GitHub comment display
	AllArtifacts    []string // All discovered artifacts from git diff

	// Execution metrics
	Duration     time.Duration
	Cost         float64
	TokensUsed   int
	InputTokens  int
	OutputTokens int

	// Implementation-specific (for sprint-executor type agents)
	BranchName    string
	WorktreePath  string
	FilesCreated  []string
	FilesModified []string
}

// OnAgentComplete is the unified handler for any agent completion.
// It uses the agent's ApprovalConfig to determine the appropriate GitHub labels
// and comment template, eliminating the need for hardcoded stage handlers.
//
// This handler:
// 1. Stores the artifact path for later use
// 2. Reads artifact content from worktree (if applicable)
// 3. Posts completion comment to GitHub
// 4. Adds needs-approval label from agent config
//
// For agents without ApprovalConfig, this is a no-op (agent handles its own workflow).
func (tc *TaskChain) OnAgentComplete(ctx context.Context, taskID, agentID string, result *AgentResult, registry *AgentRegistry) error {
	task, err := tc.store.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Get agent config for approval workflow
	var agent *AgentConfig
	if registry != nil {
		agent = registry.GetAgentByID(agentID)
	}
	if agent == nil {
		// Create temporary config to get effective defaults
		agent = &AgentConfig{ID: agentID}
	}

	approval := agent.GetEffectiveApprovalConfig()
	if approval == nil {
		// No approval workflow configured - agent handles its own completion
		log.Printf("[TaskChain] Agent %s has no approval config, skipping GitHub workflow", agentID)
		return nil
	}

	// Store artifact path for later use (in merge comment, etc.)
	if result.ArtifactPath != "" {
		// Use stage-specific storage for backwards compatibility
		switch agentID {
		case "design-doc-creator":
			if err := tc.store.SetTaskDesignDocPath(ctx, taskID, result.ArtifactPath); err != nil {
				log.Printf("[TaskChain] Warning: Failed to store design doc path: %v", err)
			}
		case "sprint-planner":
			if err := tc.store.SetTaskSprintPlanPath(ctx, taskID, result.ArtifactPath); err != nil {
				log.Printf("[TaskChain] Warning: Failed to store sprint plan path: %v", err)
			}
		}
	}

	if task.GithubIssue == 0 {
		log.Printf("[TaskChain] Task %s has no linked GitHub issue, skipping notification", taskID)
		return nil
	}

	// Handle case where no artifact found but files were changed
	if result.ArtifactPath == "" && len(result.AllArtifacts) > 0 {
		log.Printf("[TaskChain] No primary artifact found for task %s, but %d files discovered", taskID, len(result.AllArtifacts))
		if tc.poster != nil {
			fileList := formatFileList(result.AllArtifacts, 20)
			infoComment := fmt.Sprintf(
				"## 📁 Files Changed\n\n"+
					"**Task:** %s\n"+
					"**Agent:** %s\n"+
					"**Duration:** %v\n\n"+
					"**%d files discovered** (no primary artifact for inline display):\n\n%s\n"+
					"Review changes in the dashboard diff viewer.",
				taskID, agentID, result.Duration, len(result.AllArtifacts), fileList,
			)
			if err := tc.poster.PostCommentInRepo(task.GithubRepo, task.GithubIssue, infoComment); err != nil {
				log.Printf("[TaskChain] Failed to post files comment: %v", err)
			}
		}
		// Continue with approval workflow
	} else if result.ArtifactPath == "" && len(result.AllArtifacts) == 0 {
		// No files changed at all - this is a failure
		log.Printf("[TaskChain] No artifacts found for task %s, posting failure message", taskID)
		if tc.poster != nil {
			failureComment := fmt.Sprintf(
				"## ❌ No Changes Detected\n\n"+
					"**Task:** %s\n"+
					"**Agent:** %s\n"+
					"**Duration:** %v\n\n"+
					"No files were created or modified. The agent may have:\n"+
					"- Encountered an error\n"+
					"- Not made any changes\n\n"+
					"Check the agent output for details.",
				taskID, agentID, result.Duration,
			)
			if err := tc.poster.PostCommentInRepo(task.GithubRepo, task.GithubIssue, failureComment); err != nil {
				log.Printf("[TaskChain] Failed to post failure comment: %v", err)
			}
		}
		return fmt.Errorf("no artifacts found for task %s", taskID)
	}

	// Read artifact content from worktree for display
	artifactContent := result.ArtifactContent
	if artifactContent == "" && result.ArtifactPath != "" && task.WorktreePath != "" {
		fullPath := filepath.Join(task.WorktreePath, result.ArtifactPath)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			log.Printf("[TaskChain] Warning: Could not read artifact at %s: %v", fullPath, err)
		} else {
			artifactContent = string(content)
		}
	}

	// Render completion comment using template based on agent ID
	comment, err := tc.renderAgentCompleteComment(agentID, taskID, result, artifactContent, approval)
	if err != nil {
		return fmt.Errorf("failed to render completion comment: %w", err)
	}

	if tc.poster != nil {
		// Post the comment
		if err := tc.poster.PostCommentInRepo(task.GithubRepo, task.GithubIssue, comment); err != nil {
			return fmt.Errorf("failed to post comment: %w", err)
		}

		// Add the needs-approval label from config
		if approval.NeedsLabel != "" {
			if err := tc.poster.AddLabelInRepo(task.GithubRepo, task.GithubIssue, approval.NeedsLabel); err != nil {
				log.Printf("[TaskChain] Failed to add label %s: %v", approval.NeedsLabel, err)
			}
		}
	}

	return nil
}

// renderAgentCompleteComment selects and renders the appropriate comment template.
// Uses the agent's GithubCommentTemplate to select from predefined templates,
// or falls back to a generic template for unknown agents.
func (tc *TaskChain) renderAgentCompleteComment(agentID, taskID string, result *AgentResult, content string, approval *ApprovalConfig) (string, error) {
	// Build CommentData for template rendering
	data := &CommentData{
		TaskID:       taskID,
		Duration:     result.Duration,
		Cost:         result.Cost,
		TokensUsed:   result.TokensUsed,
		InputTokens:  result.InputTokens,
		OutputTokens: result.OutputTokens,
	}

	// Select template based on agent's comment template or agent ID
	templateName := approval.GithubCommentTemplate
	if templateName == "" {
		// Fall back to agent ID for template selection
		templateName = agentID
	}

	switch templateName {
	case "design_doc", "design-doc-creator":
		data.DesignDocPath = result.ArtifactPath
		data.DesignDocContent = content
		return RenderDesignDocComment(data)

	case "sprint_plan", "sprint-planner":
		data.SprintPlanPath = result.ArtifactPath
		data.SprintPlanContent = content
		return RenderSprintPlanComment(data)

	case "implementation", "sprint-executor":
		data.BranchName = result.BranchName
		data.WorktreePath = result.WorktreePath
		data.FilesCreated = result.FilesCreated
		data.FilesModified = result.FilesModified
		return RenderImplementCompleteComment(data)

	default:
		// Generic completion comment for unknown agents
		return tc.renderGenericCompleteComment(agentID, taskID, result, content, approval)
	}
}

// renderGenericCompleteComment renders a generic completion comment for agents
// without a predefined template. This supports custom agents added via config.
func (tc *TaskChain) renderGenericCompleteComment(agentID, taskID string, result *AgentResult, content string, approval *ApprovalConfig) (string, error) {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("**✅ Agent Complete: %s**\n\n", agentID))

	// Summary table
	sb.WriteString("### Summary\n\n")
	sb.WriteString("| Field | Value |\n")
	sb.WriteString("|-------|-------|\n")
	sb.WriteString(fmt.Sprintf("| **Task ID** | `%s` |\n", taskID))
	sb.WriteString(fmt.Sprintf("| **Agent** | %s |\n", agentID))
	sb.WriteString(fmt.Sprintf("| **Duration** | %v |\n", result.Duration))
	if result.Cost > 0 {
		sb.WriteString(fmt.Sprintf("| **Cost** | $%.4f |\n", result.Cost))
	}
	if result.TokensUsed > 0 {
		sb.WriteString(fmt.Sprintf("| **Tokens** | %d (%d in / %d out) |\n",
			result.TokensUsed, result.InputTokens, result.OutputTokens))
	}

	// Artifact content if available
	if result.ArtifactPath != "" && content != "" {
		sb.WriteString("\n---\n\n")
		sb.WriteString(fmt.Sprintf("<details>\n<summary><strong>📄 Artifact: %s</strong> (click to expand)</summary>\n\n", result.ArtifactPath))
		sb.WriteString(content)
		sb.WriteString("\n\n</details>\n")
	}

	// Files list if no content
	if content == "" && len(result.AllArtifacts) > 0 {
		sb.WriteString("\n### Files Changed\n\n")
		for i, f := range result.AllArtifacts {
			if i >= 20 {
				sb.WriteString(fmt.Sprintf("- ... and %d more files\n", len(result.AllArtifacts)-20))
				break
			}
			sb.WriteString(fmt.Sprintf("- `%s`\n", f))
		}
	}

	// Next steps
	sb.WriteString("\n---\n\n")
	sb.WriteString("### Next Steps\n\n")
	sb.WriteString("1. **Review the output** above\n")
	if approval.ApprovedLabel != "" {
		sb.WriteString(fmt.Sprintf("2. **Add the `%s` label** to approve and continue\n", approval.ApprovedLabel))
	}
	sb.WriteString("3. **Add the `needs-revision` label** if changes are needed\n")

	return sb.String(), nil
}

// formatFileList formats a list of files for display, limiting to maxFiles.
func formatFileList(files []string, maxFiles int) string {
	var sb strings.Builder
	for i, f := range files {
		if i >= maxFiles {
			sb.WriteString(fmt.Sprintf("- ... and %d more files\n", len(files)-maxFiles))
			break
		}
		sb.WriteString(fmt.Sprintf("- `%s`\n", f))
	}
	return sb.String()
}
