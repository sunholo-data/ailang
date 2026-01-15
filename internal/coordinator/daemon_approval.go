package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/websocket"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// approvalTracer provides OTEL tracing for approval decisions.
// Spans are linked to the task's trace context for hierarchy visibility.
var approvalTracer = otel.Tracer("coordinator.approval")

// handleAgentHandoffs checks if the completed task should trigger handoffs to other agents.
// This implements the agent-to-agent messaging with optional approval gates.
func (d *Daemon) handleAgentHandoffs(task *TaskRecord, result *ExecuteResult) error {
	if d.agentRegistry == nil {
		return nil // No agent registry configured
	}

	// Determine which agent handled this task
	// Priority: task.AgentID > thread.TargetAgent > "coordinator" (default)
	sourceAgentID := "coordinator" // default
	if task.AgentID != "" {
		// Prefer the agent_id stored on the task record (most reliable)
		sourceAgentID = task.AgentID
	} else if task.ThreadID != "" && d.msgStore != nil {
		// Fallback: look up from thread (for backwards compatibility with older tasks)
		if thread, err := d.msgStore.GetThread(task.ThreadID); err == nil && thread != nil && thread.TargetAgent != "" {
			sourceAgentID = thread.TargetAgent
		}
	}

	// Look up the source agent's configuration
	sourceAgent := d.agentRegistry.GetAgentByID(sourceAgentID)
	if sourceAgent == nil {
		d.logger.Printf("Agent %s not found in registry, skipping handoffs", sourceAgentID)
		return nil
	}

	// Check if this agent has any trigger_on_complete targets
	if len(sourceAgent.TriggerOnComplete) == 0 {
		return nil // No handoffs configured
	}

	d.logger.Printf("Task %s completed by agent %s, checking handoffs to: %v",
		task.ID, sourceAgentID, sourceAgent.TriggerOnComplete)

	// Process each handoff target
	for _, targetAgentID := range sourceAgent.TriggerOnComplete {
		targetAgent := d.agentRegistry.GetAgentByID(targetAgentID)
		if targetAgent == nil {
			d.logger.Printf("Warning: Handoff target agent %s not found in registry", targetAgentID)
			continue
		}

		// Get session ID from the result for continuity
		sessionID := ""
		if result != nil {
			sessionID = result.SessionID
		}

		// Build handoff message
		handoffMessage := fmt.Sprintf("**Handoff from %s**\n\n"+
			"Task: %s\n"+
			"Original Request: %s\n\n"+
			"Result: %s\n\n"+
			"Please continue this work.",
			sourceAgentID, task.ID, task.Content, result.Output)

		if sourceAgent.AutoApproveHandoffs {
			// Auto-approve: send message directly to target agent's inbox
			if err := d.sendHandoffMessage(targetAgent, task, handoffMessage, sessionID); err != nil {
				d.logger.Printf("Warning: Failed to send handoff to %s: %v", targetAgentID, err)
				continue
			}
			d.logger.Printf("Auto-approved handoff from %s to %s for task %s",
				sourceAgentID, targetAgentID, task.ID)
		} else {
			// Require approval: create approval request
			if err := d.requestHandoffApproval(sourceAgent, targetAgent, task, result, handoffMessage, sessionID); err != nil {
				d.logger.Printf("Warning: Failed to create handoff approval request: %v", err)
				continue
			}
			d.logger.Printf("Created handoff approval request from %s to %s for task %s",
				sourceAgentID, targetAgentID, task.ID)
		}
	}

	return nil
}

// sendHandoffMessage sends a message to the target agent's inbox
func (d *Daemon) sendHandoffMessage(targetAgent *AgentConfig, task *TaskRecord, message, sessionID string) error {
	if d.msgStore == nil {
		return fmt.Errorf("message store not available")
	}

	// Include hierarchy data in metadata for dashboard tracing
	// parent_task_id enables the dashboard to show handoff chains
	metadataMap := map[string]interface{}{
		"parent_task_id": task.ID,        // For hierarchy tracking
		"handoff_source": task.ID,        // Legacy field for backwards compatibility
		"source_agent":   task.AgentID,   // Which agent completed the work
		"target_agent":   targetAgent.ID, // Which agent is receiving the handoff
	}
	if sessionID != "" {
		metadataMap["session_id"] = sessionID
	}
	metadata := ""
	if data, err := json.Marshal(metadataMap); err == nil {
		metadata = string(data)
	}

	// Create message in the target agent's inbox
	// We create a new thread for the handoff
	_, err := d.msgStore.CreateMessage(
		"",                               // New thread (empty ThreadID)
		"ailang_instance", "coordinator", // from
		targetAgent.Inbox, targetAgent.ID, // to (inbox and agent)
		"handoff", // kind
		message,
		metadata,
	)

	return err
}

// requestHandoffApproval creates an approval request for a handoff
func (d *Daemon) requestHandoffApproval(
	sourceAgent, targetAgent *AgentConfig,
	task *TaskRecord,
	result *ExecuteResult,
	message, sessionID string,
) error {
	if d.approvalCheckpoint == nil {
		return fmt.Errorf("approval checkpoint not available")
	}

	// Create the approval request (non-blocking - we don't wait for it)
	request := &ApprovalRequest{
		ID:            fmt.Sprintf("handoff-%s-%s-%d", sourceAgent.ID, targetAgent.ID, time.Now().UnixNano()),
		TaskID:        task.ID,
		ThreadID:      task.ThreadID,
		Type:          ApprovalTypeHandoff,
		Title:         fmt.Sprintf("Handoff: %s → %s", sourceAgent.Label, targetAgent.Label),
		Description:   message,
		SourceAgentID: sourceAgent.ID,
		TargetAgentID: targetAgent.ID,
		SessionID:     sessionID,
		Timeout:       24 * time.Hour, // Allow 24 hours for human approval
		AutoReject:    true,           // Reject on timeout (don't auto-approve handoffs)
	}

	// Store the request for dashboard visibility
	// Note: We don't block waiting for approval here - the approval is handled
	// asynchronously when the human approves via CLI/dashboard
	d.approvalCheckpoint.mu.Lock()
	d.approvalCheckpoint.requests[request.ID] = request
	d.approvalCheckpoint.mu.Unlock()

	// Persist to database so CLI can see pending handoff approvals
	// Store handoff-specific data in ContextJSON
	contextData := map[string]interface{}{
		"source_agent_id": sourceAgent.ID,
		"target_agent_id": targetAgent.ID,
		"session_id":      sessionID,
		"handoff_message": message,
	}
	contextJSON, _ := json.Marshal(contextData)
	timeoutAt := time.Now().Add(request.Timeout)
	approvalReq := &ApprovalRequestRecord{
		ID:          request.ID,
		TaskID:      task.ID,
		Type:        string(ApprovalTypeHandoff),
		Description: request.Title,
		ContextJSON: string(contextJSON),
		Status:      "pending",
		CreatedAt:   time.Now(),
		TimeoutAt:   &timeoutAt,
		AutoReject:  request.AutoReject,
	}
	if err := d.taskStore.CreateApprovalRequest(d.ctx, approvalReq); err != nil {
		d.logger.Printf("Warning: Failed to persist handoff approval request: %v", err)
		// Continue anyway - in-memory request still works
	}

	// Broadcast the approval request event for dashboard
	if d.eventBroadcaster != nil {
		d.eventBroadcaster(&websocket.TaskStreamEvent{
			TaskID:     task.ID,
			ThreadID:   task.ThreadID,
			StreamType: websocket.TaskStreamStatus,
			Status:     "approval_requested",
			Text:       fmt.Sprintf("Handoff approval needed: %s → %s", sourceAgent.ID, targetAgent.ID),
		})
	}

	// Also post to the task's thread so it shows up in the message UI
	if task.ThreadID != "" && d.msgStore != nil {
		content := fmt.Sprintf("**Approval Required: Handoff to %s**\n\n%s\n\n"+
			"Use `ailang coordinator approve %s` to approve or `reject` to reject.",
			targetAgent.Label, message, request.ID)

		_, _ = d.msgStore.CreateMessage(
			task.ThreadID,
			"ailang_instance", "coordinator",
			"human", "user",
			"approval_request",
			content,
			"",
		)
	}

	return nil
}

// HandleApproval processes an approved task.
// For GitHub-linked tasks at design/sprint stages, this triggers the next stage.
// For merge-ready tasks, this merges worktree changes to main branch.
func (d *Daemon) HandleApproval(ctx context.Context, taskID, approvedBy string) error {
	// Get the task first to extract context for span
	task, err := d.taskStore.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// Determine channel source (M-DASHBOARD-APPROVAL-INTEGRATION)
	channel := "dashboard" // Default for daemon-handled approvals
	if ctx.Value("approval_channel") != nil {
		channel = ctx.Value("approval_channel").(string)
	}

	// Create OTEL span for approval decision (M-DASHBOARD-APPROVAL-INTEGRATION)
	ctx, span := approvalTracer.Start(ctx, "approval.decision",
		trace.WithAttributes(
			attribute.String("task.id", taskID),
			attribute.String("approval.action", "approve"),
			attribute.String("approval.channel", channel),
			attribute.String("approval.by", approvedBy),
			attribute.Int("task.iteration", task.Iteration),
			attribute.String("task.stage", string(task.Stage)),
		),
	)
	defer span.End()

	// Verify task is pending approval
	if task.Status != TaskStatusPendingApproval {
		span.SetStatus(codes.Error, "task not pending approval")
		return fmt.Errorf("task %s is not pending approval (status: %s)", taskID, task.Status)
	}

	// For GitHub-linked tasks at design/sprint stages, trigger next stage instead of merge
	if task.GithubIssue > 0 && d.taskChain != nil {
		switch task.Stage {
		case TaskStageDesign:
			d.logger.Printf("Approving design for task %s (GitHub #%d)", taskID, task.GithubIssue)
			// Add design-approved label on GitHub
			if d.taskChain.poster != nil {
				if err := d.taskChain.poster.AddLabel(task.GithubIssue, LabelDesignApproved); err != nil {
					d.logger.Printf("Warning: Failed to add design-approved label: %v", err)
				}
				// Remove needs-design-approval label
				if err := d.taskChain.poster.RemoveLabel(task.GithubIssue, LabelNeedsDesignApproval); err != nil {
					d.logger.Printf("Warning: Failed to remove needs-design-approval label: %v", err)
				}
			}
			// Trigger next stage via TaskChain
			return d.taskChain.OnDesignApproved(ctx, &ApprovalEvent{
				TaskID:      taskID,
				IssueNumber: task.GithubIssue,
			})

		case TaskStageSprint:
			d.logger.Printf("Approving sprint for task %s (GitHub #%d)", taskID, task.GithubIssue)
			// Add sprint-approved label on GitHub
			if d.taskChain.poster != nil {
				if err := d.taskChain.poster.AddLabel(task.GithubIssue, LabelSprintApproved); err != nil {
					d.logger.Printf("Warning: Failed to add sprint-approved label: %v", err)
				}
				// Remove needs-sprint-approval label
				if err := d.taskChain.poster.RemoveLabel(task.GithubIssue, LabelNeedsSprintApproval); err != nil {
					d.logger.Printf("Warning: Failed to remove needs-sprint-approval label: %v", err)
				}
			}
			// Trigger next stage via TaskChain
			return d.taskChain.OnSprintApproved(ctx, &ApprovalEvent{
				TaskID:      taskID,
				IssueNumber: task.GithubIssue,
			})
		}
		// For implementation/merge stage, fall through to merge logic
	}

	// Get worktree path for merge
	if task.WorktreePath == "" {
		return fmt.Errorf("task %s has no worktree path", taskID)
	}

	d.logger.Printf("Processing merge approval for task %s (worktree: %s)", taskID, task.WorktreePath)

	// Store approval event for audit trail (consistent with CLI/Dashboard via ProcessApprovalRequest)
	if storeErr := StoreApprovalEvent(ctx, d.taskStore, taskID, approvedBy); storeErr != nil {
		d.logger.Printf("Warning: Failed to store approval event: %v", storeErr)
	}

	// Resolve merge branch: per-agent > global config > default
	mergeBranch := "dev"
	if d.agentRegistry != nil && task.AgentID != "" {
		if agent := d.agentRegistry.GetAgentByID(task.AgentID); agent != nil && agent.MergeBranch != "" {
			mergeBranch = agent.MergeBranch
		}
	}
	if mergeBranch == "dev" {
		// Check global config as fallback
		if coordConfig, _ := LoadCoordinatorConfig(); coordConfig != nil && coordConfig.MergeBranch != "" {
			mergeBranch = coordConfig.MergeBranch
		}
	}

	// Attempt to merge worktree changes
	mergeResult, err := MergeWorktree(ctx, task.WorktreePath, mergeBranch)
	if err != nil {
		return fmt.Errorf("merge failed: %w", err)
	}

	// Handle merge conflicts
	if !mergeResult.Success {
		if len(mergeResult.ConflictFiles) > 0 {
			// Mark task as still pending but with conflict info
			d.logger.Printf("Merge conflicts in task %s: %v", taskID, mergeResult.ConflictFiles)

			// Notify via message
			if task.ThreadID != "" && d.msgStore != nil {
				content := fmt.Sprintf("**Merge Conflicts Detected**\n\n"+
					"The following files have conflicts:\n%s\n\n"+
					"Please resolve conflicts manually in the worktree at:\n`%s`\n\n"+
					"Then retry the approval.",
					strings.Join(mergeResult.ConflictFiles, "\n"), task.WorktreePath)

				_, _ = d.msgStore.CreateMessage(
					task.ThreadID,
					"ailang_instance", "coordinator",
					"human", "user",
					"merge_conflict",
					content,
					"",
				)
			}

			return fmt.Errorf("merge conflicts: %v", mergeResult.ConflictFiles)
		}
		return fmt.Errorf("merge failed: %s", mergeResult.Error)
	}

	// Merge succeeded - update task status
	if err := d.taskStore.MarkTaskCompleted(ctx, taskID, &ExecuteResult{
		Success: true,
		Output:  fmt.Sprintf("Merged to %s (commit: %s)", mergeBranch, mergeResult.CommitHash),
	}); err != nil {
		d.logger.Printf("Warning: Failed to update task status: %v", err)
	}

	// Clean up worktree (changes are now merged)
	if d.worktreeMgr != nil {
		if rmErr := d.worktreeMgr.RemoveWorktree(taskID); rmErr != nil {
			d.logger.Printf("Warning: Failed to remove worktree: %v", rmErr)
		}
	}

	// Notify success
	if task.ThreadID != "" && d.msgStore != nil {
		content := fmt.Sprintf("**Changes Merged Successfully**\n\n"+
			"Branch: %s\n"+
			"Commit: `%s`\n"+
			"Files: %s\n"+
			"Approved by: %s",
			mergeBranch,
			mergeResult.CommitHash,
			strings.Join(mergeResult.MergedFiles, ", "),
			approvedBy)

		_, _ = d.msgStore.CreateMessage(
			task.ThreadID,
			"ailang_instance", "coordinator",
			"human", "user",
			"merge_complete",
			content,
			"",
		)
	}

	// Broadcast event
	if d.eventBroadcaster != nil {
		d.eventBroadcaster(&websocket.TaskStreamEvent{
			TaskID:     taskID,
			ThreadID:   task.ThreadID,
			StreamType: websocket.TaskStreamStatus,
			Status:     "merged",
			Text:       fmt.Sprintf("Changes merged to %s (commit: %s)", mergeBranch, mergeResult.CommitHash[:8]),
		})
	}

	d.logger.Printf("Task %s approved and merged by %s (commit: %s)",
		taskID, approvedBy, mergeResult.CommitHash)

	// Check for pending handoff approvals for this task
	if d.approvalCheckpoint != nil {
		if req := d.approvalCheckpoint.GetRequestByTask(taskID); req != nil {
			if req.Type == ApprovalTypeHandoff {
				// Auto-approve the handoff now that the work is merged
				if err := d.processHandoffApproval(ctx, req); err != nil {
					d.logger.Printf("Warning: Failed to process handoff: %v", err)
				}
			}
		}
	}

	// Record success in span (M-DASHBOARD-APPROVAL-INTEGRATION)
	span.SetAttributes(attribute.String("merge.commit", mergeResult.CommitHash))
	span.SetStatus(codes.Ok, "approved and merged")

	return nil
}

// HandleRejection processes a rejected task - preserves worktree and marks task rejected.
// For tasks with linked GitHub issues, feedback is posted to GitHub (M-DASHBOARD-APPROVAL-INTEGRATION).
func (d *Daemon) HandleRejection(ctx context.Context, taskID, rejectedBy, reason string) error {
	// Get the task first to extract context for span
	task, err := d.taskStore.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// Determine channel source (M-DASHBOARD-APPROVAL-INTEGRATION)
	channel := "dashboard" // Default for daemon-handled rejections
	if ctx.Value("approval_channel") != nil {
		channel = ctx.Value("approval_channel").(string)
	}

	// Create OTEL span for rejection decision (M-DASHBOARD-APPROVAL-INTEGRATION)
	ctx, span := approvalTracer.Start(ctx, "approval.decision",
		trace.WithAttributes(
			attribute.String("task.id", taskID),
			attribute.String("approval.action", "reject"),
			attribute.String("approval.channel", channel),
			attribute.String("approval.by", rejectedBy),
			attribute.String("approval.reason", truncateForAttribute(reason, 500)),
			attribute.Int("task.iteration", task.Iteration),
			attribute.String("task.stage", string(task.Stage)),
		),
	)
	defer span.End()

	// Verify task is pending approval
	if task.Status != TaskStatusPendingApproval {
		span.SetStatus(codes.Error, "task not pending approval")
		return fmt.Errorf("task %s is not pending approval (status: %s)", taskID, task.Status)
	}

	d.logger.Printf("Processing rejection for task %s (worktree preserved: %s)", taskID, task.WorktreePath)

	// Store feedback event for audit trail (consistent with CLI/Dashboard via ProcessApprovalRequest)
	feedback := &HumanFeedback{
		TaskID:    taskID,
		Iteration: task.Iteration,
		Feedback:  reason,
		Action:    "reject",
		Timestamp: time.Now(),
		UserID:    rejectedBy,
	}
	if storeErr := StoreFeedbackEvent(ctx, d.taskStore, feedback); storeErr != nil {
		d.logger.Printf("Warning: Failed to store feedback event: %v", storeErr)
	}

	// Mark task as rejected - worktree is preserved for reference
	if err := d.taskStore.MarkTaskRejected(ctx, taskID); err != nil {
		return fmt.Errorf("failed to mark task rejected: %w", err)
	}

	// Post rejection feedback to GitHub if task has a linked issue (M-DASHBOARD-APPROVAL-INTEGRATION)
	if task.GithubIssue > 0 && d.githubPoster != nil {
		// Determine iteration (default to 1 if not set)
		iteration := task.Iteration
		if iteration < 1 {
			iteration = 1
		}
		// Determine channel - default to "dashboard" for dashboard-initiated rejections
		channel := "dashboard"
		if err := d.githubPoster.PostFeedback(task.GithubIssue, reason, iteration, channel); err != nil {
			d.logger.Printf("Warning: Failed to post feedback to GitHub issue #%d: %v", task.GithubIssue, err)
		} else {
			d.logger.Printf("Posted rejection feedback to GitHub issue #%d (iteration %d)", task.GithubIssue, iteration)
		}
	}

	// Notify via message
	if task.ThreadID != "" && d.msgStore != nil {
		content := fmt.Sprintf("**Task Rejected**\n\n"+
			"Rejected by: %s\n"+
			"Reason: %s\n\n"+
			"The worktree is preserved at:\n`%s`\n\n"+
			"You can review the changes or delete the worktree manually.",
			rejectedBy, reason, task.WorktreePath)

		_, _ = d.msgStore.CreateMessage(
			task.ThreadID,
			"ailang_instance", "coordinator",
			"human", "user",
			"task_rejected",
			content,
			"",
		)
	}

	// Broadcast event
	if d.eventBroadcaster != nil {
		d.eventBroadcaster(&websocket.TaskStreamEvent{
			TaskID:     taskID,
			ThreadID:   task.ThreadID,
			StreamType: websocket.TaskStreamStatus,
			Status:     "rejected",
			Text:       fmt.Sprintf("Task rejected by %s", rejectedBy),
		})
	}

	// Reject any pending handoff approvals for this task
	if d.approvalCheckpoint != nil {
		if req := d.approvalCheckpoint.GetRequestByTask(taskID); req != nil {
			_ = d.approvalCheckpoint.Reject(req.ID, rejectedBy)
		}
	}

	d.logger.Printf("Task %s rejected by %s (worktree preserved)", taskID, rejectedBy)

	// Record success in span (M-DASHBOARD-APPROVAL-INTEGRATION)
	span.SetStatus(codes.Ok, "rejected with feedback")

	return nil
}

// truncateForAttribute truncates a string for use in span attributes.
// OTEL has limits on attribute value sizes.
func truncateForAttribute(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// processHandoffApproval sends a handoff message after approval.
func (d *Daemon) processHandoffApproval(ctx context.Context, req *ApprovalRequest) error {
	if d.agentRegistry == nil || d.msgStore == nil {
		return fmt.Errorf("agent registry or message store not available")
	}

	targetAgent := d.agentRegistry.GetAgentByID(req.TargetAgentID)
	if targetAgent == nil {
		return fmt.Errorf("target agent %s not found", req.TargetAgentID)
	}

	// Get the task for context
	task, err := d.taskStore.GetTask(ctx, req.TaskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Create the handoff message
	message := fmt.Sprintf("**Approved Handoff from %s**\n\n%s",
		req.SourceAgentID, req.Description)

	// Include hierarchy data in metadata for dashboard tracing
	metadataMap := map[string]interface{}{
		"parent_task_id": req.TaskID,        // For hierarchy tracking
		"handoff_source": req.TaskID,        // Legacy field for backwards compatibility
		"source_agent":   req.SourceAgentID, // Which agent completed the work
		"target_agent":   req.TargetAgentID, // Which agent is receiving the handoff
	}
	if req.SessionID != "" {
		metadataMap["session_id"] = req.SessionID
	}
	metadata := ""
	if data, err := json.Marshal(metadataMap); err == nil {
		metadata = string(data)
	}

	// Send to target agent's inbox
	_, err = d.msgStore.CreateMessage(
		"",                               // New thread
		"ailang_instance", "coordinator", // from
		targetAgent.Inbox, targetAgent.ID, // to
		"handoff",
		message,
		metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to send handoff message: %w", err)
	}

	d.logger.Printf("Handoff approved: %s → %s (task: %s)",
		req.SourceAgentID, req.TargetAgentID, task.ID)

	return nil
}
