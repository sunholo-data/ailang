// Package coordinator provides unified approval processing for CLI, Dashboard, and Daemon.
package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/messaging"
	"github.com/sunholo/ailang/internal/observatory"
	"github.com/sunholo/ailang/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var approvalProcessorTracer = telemetry.Tracer("coordinator.approval")

// ApprovalParams contains all parameters for processing an approval or rejection.
// This is the single interface used by CLI, Dashboard, and Daemon.
type ApprovalParams struct {
	TaskID      string // Required: task ID or approval ID (apr-xxx converted to task-xxx)
	Action      string // Required: "approve" or "reject"
	ApprovedBy  string // Required: who is approving/rejecting (e.g., "cli-user", "dashboard-user")
	Channel     string // Required: source channel ("cli", "dashboard", "daemon")
	Feedback    string // Optional: feedback text for rejections
	MergeBranch string // Optional: target branch for merge (defaults from config, then channel)

	// Behavior options
	SkipMerge         bool // If true, don't merge worktree on approval
	KeepWorktree      bool // If true, don't clean up worktree after merge
	RetriggerOnReject bool // If true, send feedback to agent for re-attempt (feedback loop)

	// Dependencies (injected by caller)
	Store         Store               // Required: coordinator store for task/approval operations
	MsgStore      *messaging.Store    // Optional: messaging store for sending feedback to agents
	GitHubPoster  *GitHubPoster       // Optional: for posting feedback to GitHub issues
	AgentRegistry *AgentRegistry      // Optional: for looking up per-agent merge branch
	ObsBackend    observatory.Backend // Optional: for updating chain/stage status (M-CHAINS-SIMPLIFY)
}

// ApprovalResult contains the result of processing an approval or rejection.
type ApprovalResult struct {
	Success       bool
	Message       string   // Human-readable result message
	MergeCommit   string   // Commit hash if merged
	MergedFiles   []string // Files that were merged
	ConflictFiles []string // Files with conflicts (if merge failed)
	NewTaskID     string   // ID of new task if re-triggered
	Error         string   // Error message if failed
}

// ProcessApprovalRequest handles approval/rejection from any channel (CLI, dashboard, daemon).
// This is the SINGLE source of truth for approval logic.
func ProcessApprovalRequest(ctx context.Context, params *ApprovalParams) (*ApprovalResult, error) {
	if params.Store == nil {
		return nil, fmt.Errorf("store is required")
	}
	if params.TaskID == "" {
		return nil, fmt.Errorf("task ID is required")
	}
	if params.Action != "approve" && params.Action != "reject" {
		return nil, fmt.Errorf("action must be 'approve' or 'reject'")
	}

	// Normalize task ID (accept both task-xxx and apr-xxx)
	taskID := params.TaskID
	if strings.HasPrefix(taskID, "apr-") {
		taskID = "task-" + strings.TrimPrefix(taskID, "apr-")
	}

	// Get the task first (needed for agent lookup)
	task, err := params.Store.GetTask(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	// Resolve merge branch with priority: explicit param > per-agent > global > default
	mergeBranch := params.MergeBranch
	if mergeBranch == "" && params.AgentRegistry != nil && task.AgentID != "" {
		// Look up agent's merge branch
		if agent := params.AgentRegistry.GetAgentByID(task.AgentID); agent != nil && agent.MergeBranch != "" {
			mergeBranch = agent.MergeBranch
		}
	}
	if mergeBranch == "" {
		// Fall back to "dev" (global config default is applied during loading)
		mergeBranch = "dev"
	}

	// Verify task is pending approval
	if task.Status != TaskStatusPendingApproval {
		return nil, fmt.Errorf("task %s is not pending approval (status: %s)", taskID, task.Status)
	}

	// Create OTEL span for approval decision
	ctx, span := telemetry.StartSpan(ctx, approvalProcessorTracer, "approval.decision",
		trace.WithAttributes(
			attribute.String("task.id", taskID),
			attribute.String("approval.action", params.Action),
			attribute.String("approval.channel", params.Channel),
			attribute.String("approval.by", params.ApprovedBy),
			attribute.Int("task.iteration", task.Iteration),
			attribute.String("task.stage", string(task.Stage)),
		),
	)
	defer span.End()

	if params.Action == "approve" {
		return processApproval(ctx, span, params, task, taskID, mergeBranch)
	}
	return processRejection(ctx, span, params, task, taskID)
}

// processApproval handles the approval action.
func processApproval(ctx context.Context, span trace.Span, params *ApprovalParams, task *TaskRecord, taskID, mergeBranch string) (*ApprovalResult, error) {
	result := &ApprovalResult{Success: true}

	// 1. Store approval event for audit trail
	if err := StoreApprovalEvent(ctx, params.Store, taskID, params.ApprovedBy); err != nil {
		// Log but don't fail - approval can still proceed
		span.AddEvent("warning: failed to store approval event", trace.WithAttributes(
			attribute.String("error", err.Error()),
		))
	}

	// 1.5. Config-driven GitHub label handling (M-GENERIC-PIPELINE)
	// Look up agent from task.AgentID and use GetEffectiveApprovalConfig() for labels
	if task.GithubIssue > 0 && params.GitHubPoster != nil && params.AgentRegistry != nil && task.AgentID != "" {
		agent := params.AgentRegistry.GetAgentByID(task.AgentID)
		if agent != nil {
			approval := agent.GetEffectiveApprovalConfig()
			if approval != nil && approval.ApprovedLabel != "" {
				span.AddEvent("updating GitHub labels", trace.WithAttributes(
					attribute.String("agent.id", task.AgentID),
					attribute.String("label.approved", approval.ApprovedLabel),
					attribute.String("label.needs", approval.NeedsLabel),
				))

				// Add approved label
				if err := params.GitHubPoster.AddLabel(task.GithubIssue, approval.ApprovedLabel); err != nil {
					span.AddEvent("warning: failed to add approved label", trace.WithAttributes(
						attribute.String("error", err.Error()),
					))
				}

				// Remove needs-approval label
				if approval.NeedsLabel != "" {
					if err := params.GitHubPoster.RemoveLabel(task.GithubIssue, approval.NeedsLabel); err != nil {
						span.AddEvent("warning: failed to remove needs label", trace.WithAttributes(
							attribute.String("error", err.Error()),
						))
					}
				}
			}

			// 1.6. Post approval completion comment to GitHub
			var comment string
			if len(agent.TriggerOnComplete) > 0 {
				nextAgents := strings.Join(agent.TriggerOnComplete, ", ")
				comment = fmt.Sprintf("**Approval Complete** - Work by %s approved.\n\n"+
					"Triggering next stage: %s", agent.Label, nextAgents)
			} else {
				comment = fmt.Sprintf("**Approval Complete** - Work by %s approved.", agent.Label)
			}

			if err := params.GitHubPoster.PostComment(task.GithubIssue, comment); err != nil {
				span.AddEvent("warning: failed to post approval comment", trace.WithAttributes(
					attribute.String("error", err.Error()),
				))
			}
		}
	}

	// 2. Resolve the approval request in database
	if err := params.Store.ResolveApprovalRequestByTask(ctx, taskID, "approved", params.ApprovedBy); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to resolve approval")
		return nil, fmt.Errorf("failed to approve task: %w", err)
	}

	result.Message = fmt.Sprintf("Task approved: %s", taskID)

	// 3. Skip merge if requested or no worktree
	if params.SkipMerge {
		result.Message += " (merge skipped)"
		span.SetStatus(codes.Ok, "approved, merge skipped")
		return result, nil
	}

	if task.WorktreePath == "" {
		result.Message += " (no worktree to merge)"
		span.SetStatus(codes.Ok, "approved, no worktree")
		return result, nil
	}

	// Check worktree exists
	if _, err := os.Stat(task.WorktreePath); os.IsNotExist(err) {
		result.Message += " (worktree no longer exists)"
		span.SetStatus(codes.Ok, "approved, worktree missing")
		return result, nil
	}

	// 4. Auto-commit any uncommitted changes in the worktree
	if err := autoCommitWorktreeChanges(task.WorktreePath, task.Title); err != nil {
		span.AddEvent("warning: failed to auto-commit", trace.WithAttributes(
			attribute.String("error", err.Error()),
		))
	}

	// 5. Perform the merge
	mergeResult, err := MergeWorktree(ctx, task.WorktreePath, mergeBranch)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "merge failed")
		return nil, fmt.Errorf("merge failed: %w", err)
	}

	if !mergeResult.Success {
		if len(mergeResult.ConflictFiles) > 0 {
			result.Success = false
			result.ConflictFiles = mergeResult.ConflictFiles
			result.Error = fmt.Sprintf("merge conflicts: %v", mergeResult.ConflictFiles)
			span.SetStatus(codes.Error, "merge conflicts")
			return result, nil
		}
		result.Success = false
		result.Error = mergeResult.Error
		span.SetStatus(codes.Error, mergeResult.Error)
		return result, nil
	}

	result.MergeCommit = mergeResult.CommitHash
	result.MergedFiles = mergeResult.MergedFiles

	// 6. Update task status to completed
	if err := params.Store.MarkTaskCompleted(ctx, taskID, &ExecuteResult{
		Success: true,
		Output:  fmt.Sprintf("Merged to %s (commit: %s)", mergeBranch, mergeResult.CommitHash),
	}); err != nil {
		span.AddEvent("warning: failed to update task status", trace.WithAttributes(
			attribute.String("error", err.Error()),
		))
	}

	// 6.5 Update chain/stage status (M-CHAINS-SIMPLIFY)
	if params.ObsBackend != nil && task.StageID != "" {
		if err := params.ObsBackend.UpdateStageStatus(ctx, task.StageID, observatory.StageStatusCompleted); err != nil {
			span.AddEvent("warning: failed to update chain stage status", trace.WithAttributes(
				attribute.String("error", err.Error()),
			))
		}
	}
	if params.ObsBackend != nil && task.ChainID != "" {
		if err := params.ObsBackend.UpdateChainStatus(ctx, task.ChainID, observatory.ChainStatusCompleted); err != nil {
			span.AddEvent("warning: failed to update chain status", trace.WithAttributes(
				attribute.String("error", err.Error()),
			))
		}
	}

	result.Message = fmt.Sprintf("Task approved and merged to %s (commit: %s)", mergeBranch, mergeResult.CommitHash)

	// 7. Trigger embedded handoffs if this was a merge_handoff approval
	if params.MsgStore != nil && params.AgentRegistry != nil {
		if handoffTriggered, err := triggerEmbeddedHandoffsFromProcessor(ctx, span, params, task, taskID); err != nil {
			span.AddEvent("warning: failed to trigger handoffs", trace.WithAttributes(
				attribute.String("error", err.Error()),
			))
		} else if handoffTriggered {
			span.AddEvent("handoffs triggered successfully")
		}
	}

	// 8. Close GitHub issue with merge summary (M-COORD-GITHUB-CLOSE-ON-MERGE)
	if task.GithubIssue > 0 && params.GitHubPoster != nil {
		comment := fmt.Sprintf("**Merged to %s**\n\nCommit: `%s`\nApproved by: %s",
			mergeBranch, result.MergeCommit, params.ApprovedBy)
		if err := params.GitHubPoster.CloseIssueInRepo(task.GithubRepo, task.GithubIssue, comment); err != nil {
			span.AddEvent("warning: failed to close GitHub issue", trace.WithAttributes(
				attribute.Int("github.issue", task.GithubIssue),
				attribute.String("github.repo", task.GithubRepo),
				attribute.String("error", err.Error()),
			))
		}
		// Release the claim label
		if err := params.GitHubPoster.ReleaseIssueInRepo(task.GithubRepo, task.GithubIssue); err != nil {
			span.AddEvent("warning: failed to release issue claim")
		}
	}

	// 9. Clean up worktree (unless --keep-worktree)
	if !params.KeepWorktree {
		cleanupWorktree(task.WorktreePath)
	}

	span.SetAttributes(attribute.String("merge.commit", mergeResult.CommitHash))
	span.SetStatus(codes.Ok, "approved and merged")

	return result, nil
}

// processRejection handles the rejection action.
func processRejection(ctx context.Context, span trace.Span, params *ApprovalParams, task *TaskRecord, taskID string) (*ApprovalResult, error) {
	result := &ApprovalResult{Success: true}

	feedback := params.Feedback
	if feedback == "" {
		feedback = "Rejected without specific feedback"
	}

	// Add feedback to span attributes
	span.SetAttributes(attribute.String("approval.reason", truncateForAttribute(feedback, 500)))

	// 1. Store feedback event for audit trail
	feedbackEvent := &HumanFeedback{
		TaskID:    taskID,
		Iteration: task.Iteration,
		Feedback:  feedback,
		Action:    "reject",
		Timestamp: time.Now(),
		UserID:    params.ApprovedBy,
	}
	if err := StoreFeedbackEvent(ctx, params.Store, feedbackEvent); err != nil {
		span.AddEvent("warning: failed to store feedback event", trace.WithAttributes(
			attribute.String("error", err.Error()),
		))
	}

	// 2. Post feedback to GitHub if task has a linked issue
	if task.GithubIssue > 0 && params.GitHubPoster != nil {
		iteration := task.Iteration
		if iteration < 1 {
			iteration = 1
		}
		if err := params.GitHubPoster.PostFeedback(task.GithubIssue, feedback, iteration, params.Channel); err != nil {
			span.AddEvent("warning: failed to post to GitHub", trace.WithAttributes(
				attribute.String("error", err.Error()),
			))
		}
	}

	// 3. Check if we can re-trigger (human rejections have no limit; agent-to-agent limited)
	canRetrigger := params.RetriggerOnReject && CanRetrigger(task, params.Channel)

	if canRetrigger {
		// Calculate next iteration
		nextIteration := task.Iteration + 1
		if nextIteration < 2 {
			nextIteration = 2 // First rejection creates iteration 2
		}

		// Store iteration start event
		if err := StoreIterationStartEvent(ctx, params.Store, taskID, nextIteration); err != nil {
			span.AddEvent("warning: failed to store iteration start", trace.WithAttributes(
				attribute.String("error", err.Error()),
			))
		}

		// Mark OLD task as rejected
		if err := params.Store.MarkTaskRejected(ctx, taskID); err != nil {
			span.AddEvent("warning: failed to mark task rejected", trace.WithAttributes(
				attribute.String("error", err.Error()),
			))
		}

		// Resolve the approval request as rejected (so dashboard shows correct state)
		if err := params.Store.ResolveApprovalRequestByTask(ctx, taskID, "rejected", params.ApprovedBy); err != nil {
			span.AddEvent("warning: failed to resolve approval", trace.WithAttributes(
				attribute.String("error", err.Error()),
			))
		}

		// Send feedback message to agent inbox
		if params.MsgStore != nil {
			agentInbox := task.AgentID
			if agentInbox == "" {
				agentInbox = "coordinator"
			}

			payload := fmt.Sprintf("Task rejected with feedback:\n\n%s\n\n---\n**Original Task:** %s\n**Iteration:** %d/%d\n**Context:** Session will resume with feedback incorporated.",
				feedback, task.Title, nextIteration, MaxAgentIterations)

			msg := &messaging.InboxMessage{
				FromAgent:     params.ApprovedBy,
				ToInbox:       agentInbox,
				MessageType:   messaging.InboxTypeNotification,
				Title:         fmt.Sprintf("Feedback: %s (iteration %d)", truncateString(task.Title, 30), nextIteration),
				Payload:       payload,
				CorrelationID: taskID,
				ParentTaskID:  taskID,
				GitHubIssue:   &task.GithubIssue,
				Iteration:     nextIteration,
			}
			if err := params.MsgStore.InsertInboxMessageWithContext(ctx, msg); err != nil {
				span.AddEvent("warning: failed to send feedback message", trace.WithAttributes(
					attribute.String("error", err.Error()),
				))
			} else {
				result.NewTaskID = msg.ID
			}
		}

		result.Message = fmt.Sprintf("Task rejected and re-queued for iteration %d (max: %d)", nextIteration, MaxAgentIterations)
		span.SetStatus(codes.Ok, "rejected with re-trigger")

	} else {
		// Max iterations reached or re-trigger disabled - permanent rejection
		if err := params.Store.ResolveApprovalRequestByTask(ctx, taskID, "rejected", params.ApprovedBy); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to resolve rejection")
			return nil, fmt.Errorf("failed to reject task: %w", err)
		}

		// Update task status
		if err := params.Store.MarkTaskRejected(ctx, taskID); err != nil {
			span.AddEvent("warning: failed to update task status", trace.WithAttributes(
				attribute.String("error", err.Error()),
			))
		}

		// Update chain/stage status to failed (M-CHAINS-SIMPLIFY)
		if params.ObsBackend != nil && task.StageID != "" {
			if err := params.ObsBackend.UpdateStageStatus(ctx, task.StageID, observatory.StageStatusFailed); err != nil {
				span.AddEvent("warning: failed to update chain stage status", trace.WithAttributes(
					attribute.String("error", err.Error()),
				))
			}
		}
		if params.ObsBackend != nil && task.ChainID != "" {
			if err := params.ObsBackend.UpdateChainStatus(ctx, task.ChainID, observatory.ChainStatusFailed); err != nil {
				span.AddEvent("warning: failed to update chain status", trace.WithAttributes(
					attribute.String("error", err.Error()),
				))
			}
		}

		// Clean up worktree
		if task.WorktreePath != "" {
			cleanupWorktree(task.WorktreePath)
		}

		if !params.RetriggerOnReject {
			result.Message = fmt.Sprintf("Task rejected: %s (re-trigger disabled)", taskID)
		} else {
			result.Message = fmt.Sprintf("Task permanently rejected: %s (max iterations %d reached)", taskID, MaxAgentIterations)
		}
		span.SetStatus(codes.Ok, "rejected permanently")
	}

	return result, nil
}

// autoCommitWorktreeChanges commits any uncommitted changes in the worktree.
func autoCommitWorktreeChanges(worktreePath, taskTitle string) error {
	// Check for changes
	statusCmd := exec.Command("git", "-C", worktreePath, "status", "--porcelain")
	statusOutput, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	if len(strings.TrimSpace(string(statusOutput))) == 0 {
		return nil // No changes to commit
	}

	// Add all changes
	addCmd := exec.Command("git", "-C", worktreePath, "add", "-A")
	if output, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stage changes: %s", output)
	}

	// Commit
	commitMsg := fmt.Sprintf("Agent work: %s\n\nAuto-committed on approval", taskTitle)
	commitCmd := exec.Command("git", "-C", worktreePath, "commit", "-m", commitMsg)
	if output, err := commitCmd.CombinedOutput(); err != nil {
		// Check if it's just "nothing to commit"
		if strings.Contains(string(output), "nothing to commit") {
			return nil
		}
		return fmt.Errorf("failed to commit: %s", output)
	}

	return nil
}

// cleanupWorktree removes the worktree and its branch.
func cleanupWorktree(worktreePath string) {
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		return
	}

	// Get the branch name before removing worktree
	branchCmd := exec.Command("git", "-C", worktreePath, "rev-parse", "--abbrev-ref", "HEAD")
	branchOutput, _ := branchCmd.Output()
	branchName := strings.TrimSpace(string(branchOutput))

	// Remove the worktree
	removeCmd := exec.Command("git", "worktree", "remove", worktreePath, "--force")
	removeCmd.Run() // Ignore errors

	// Also delete the branch
	if branchName != "" && branchName != "HEAD" {
		deleteCmd := exec.Command("git", "branch", "-D", branchName)
		deleteCmd.Run() // Ignore errors
	}
}

// Note: truncateString and truncateForAttribute are defined in other files in this package

// triggerEmbeddedHandoffsFromProcessor triggers handoffs embedded in a merge_handoff approval.
// This is called from ProcessApprovalRequest to handle dashboard/CLI approvals that bypass the daemon.
// Returns (triggered bool, error) - triggered is true if handoffs were sent.
func triggerEmbeddedHandoffsFromProcessor(ctx context.Context, span trace.Span, params *ApprovalParams, task *TaskRecord, taskID string) (bool, error) {
	if params.MsgStore == nil || params.AgentRegistry == nil {
		return false, nil
	}

	// Get the approval request to retrieve context_json with handoff targets
	// Note: Use GetApprovalRequestByTaskAnyStatus since the approval may already be marked approved
	approvalReq, err := params.Store.GetApprovalRequestByTaskAnyStatus(ctx, taskID)
	if err != nil || approvalReq == nil {
		return false, nil // No approval request found, not an error
	}

	return triggerHandoffsFromApprovalRecord(ctx, span, params, task, taskID, approvalReq)
}

// triggerHandoffsFromApprovalRecord triggers handoffs from an already-fetched approval record.
// This variant is used by the catch-up mechanism which already has the approval record.
func triggerHandoffsFromApprovalRecord(ctx context.Context, span trace.Span, params *ApprovalParams, task *TaskRecord, taskID string, approvalReq *ApprovalRequestRecord) (bool, error) {
	if params.MsgStore == nil || params.AgentRegistry == nil {
		return false, nil
	}

	// Only trigger handoffs for merge_handoff type approvals
	if approvalReq.Type != "merge_handoff" || approvalReq.ContextJSON == "" {
		return false, nil
	}

	// Parse the embedded handoff data
	var handoffContext struct {
		HandoffTargets []string `json:"handoff_targets"`
		SessionID      string   `json:"session_id"`
		SourceAgent    string   `json:"source_agent"`
	}
	if err := json.Unmarshal([]byte(approvalReq.ContextJSON), &handoffContext); err != nil {
		return false, fmt.Errorf("failed to parse handoff context: %w", err)
	}

	if len(handoffContext.HandoffTargets) == 0 {
		return false, nil
	}

	span.AddEvent("triggering embedded handoffs", trace.WithAttributes(
		attribute.StringSlice("handoff.targets", handoffContext.HandoffTargets),
		attribute.String("handoff.source", handoffContext.SourceAgent),
	))

	// Build handoff message
	handoffMessage := fmt.Sprintf("**Handoff from %s (approved)**\n\n"+
		"Task: %s\n"+
		"Title: %s\n"+
		"Original Request: %s\n\n"+
		"Please continue this work.",
		handoffContext.SourceAgent, task.ID, task.Title, truncateString(task.Content, 500))

	// Trigger each handoff
	triggered := false
	for _, targetAgentID := range handoffContext.HandoffTargets {
		targetAgent := params.AgentRegistry.GetAgentByID(targetAgentID)
		if targetAgent == nil {
			span.AddEvent("warning: handoff target not found", trace.WithAttributes(
				attribute.String("target.agent", targetAgentID),
			))
			continue
		}

		// Send to target agent's inbox
		msg := &messaging.InboxMessage{
			FromAgent:    "coordinator",
			ToInbox:      targetAgent.Inbox,
			MessageType:  "handoff",
			Title:        fmt.Sprintf("Handoff: %s (approved)", task.Title),
			Payload:      handoffMessage,
			ParentTaskID: task.ID,      // M-TASK-HIERARCHY: Link to parent task for handoff chains
			ChainID:      task.ChainID, // M-CHAINS-SIMPLIFY: Link to existing chain
		}

		if err := params.MsgStore.InsertInboxMessage(msg); err != nil {
			span.AddEvent("warning: failed to send handoff", trace.WithAttributes(
				attribute.String("target.agent", targetAgentID),
				attribute.String("error", err.Error()),
			))
			continue
		}

		span.AddEvent("handoff sent", trace.WithAttributes(
			attribute.String("target.agent", targetAgentID),
			attribute.String("target.inbox", targetAgent.Inbox),
		))
		triggered = true
	}

	// Mark handoffs as triggered to prevent re-triggering on daemon restart
	if triggered {
		if err := params.Store.MarkApprovalHandoffsTriggered(ctx, taskID); err != nil {
			span.AddEvent("warning: failed to mark handoffs as triggered", trace.WithAttributes(
				attribute.String("error", err.Error()),
			))
		}
	}

	return triggered, nil
}
