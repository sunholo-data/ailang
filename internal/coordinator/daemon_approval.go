package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/websocket"
)

// handleAgentHandoffs checks if the completed task should trigger handoffs to other agents.
// This implements the agent-to-agent messaging with optional approval gates.
func (d *Daemon) handleAgentHandoffs(task *TaskRecord, result *ExecuteResult) error {
	if d.agentRegistry == nil {
		return nil // No agent registry configured
	}

	// Determine which agent handled this task
	sourceAgentID := "coordinator" // default
	if task.ThreadID != "" && d.msgStore != nil {
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

	// Include session ID in metadata for continuity
	metadata := ""
	if sessionID != "" {
		metadataMap := map[string]interface{}{
			"session_id":     sessionID,
			"handoff_source": task.ID,
		}
		if data, err := json.Marshal(metadataMap); err == nil {
			metadata = string(data)
		}
	}

	// Create message in the target agent's inbox
	// We create a new thread for the handoff
	_, err := d.msgStore.CreateMessage(
		"",                                       // New thread (empty ThreadID)
		"ailang_instance", "coordinator",         // from
		targetAgent.Inbox, targetAgent.ID,        // to (inbox and agent)
		"handoff",                                // kind
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

// HandleApproval processes an approved task - merges worktree changes to main branch.
func (d *Daemon) HandleApproval(ctx context.Context, taskID, approvedBy string) error {
	// Get the task
	task, err := d.taskStore.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// Verify task is pending approval
	if task.Status != TaskStatusPendingApproval {
		return fmt.Errorf("task %s is not pending approval (status: %s)", taskID, task.Status)
	}

	// Get worktree path
	if task.WorktreePath == "" {
		return fmt.Errorf("task %s has no worktree path", taskID)
	}

	d.logger.Printf("Processing approval for task %s (worktree: %s)", taskID, task.WorktreePath)

	// Attempt to merge worktree changes
	mergeResult, err := MergeWorktree(ctx, task.WorktreePath, "main")
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
		Output:  fmt.Sprintf("Merged to main (commit: %s)", mergeResult.CommitHash),
	}); err != nil {
		d.logger.Printf("Warning: Failed to update task status: %v", err)
	}

	// Clean up worktree (changes are now in main)
	if d.worktreeMgr != nil {
		if rmErr := d.worktreeMgr.RemoveWorktree(taskID); rmErr != nil {
			d.logger.Printf("Warning: Failed to remove worktree: %v", rmErr)
		}
	}

	// Notify success
	if task.ThreadID != "" && d.msgStore != nil {
		content := fmt.Sprintf("**Changes Merged Successfully**\n\n"+
			"Commit: `%s`\n"+
			"Files: %s\n"+
			"Approved by: %s",
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
			Text:       fmt.Sprintf("Changes merged to main (commit: %s)", mergeResult.CommitHash[:8]),
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

	return nil
}

// HandleRejection processes a rejected task - preserves worktree and marks task rejected.
func (d *Daemon) HandleRejection(ctx context.Context, taskID, rejectedBy, reason string) error {
	// Get the task
	task, err := d.taskStore.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// Verify task is pending approval
	if task.Status != TaskStatusPendingApproval {
		return fmt.Errorf("task %s is not pending approval (status: %s)", taskID, task.Status)
	}

	d.logger.Printf("Processing rejection for task %s (worktree preserved: %s)", taskID, task.WorktreePath)

	// Mark task as rejected - worktree is preserved for reference
	if err := d.taskStore.MarkTaskRejected(ctx, taskID); err != nil {
		return fmt.Errorf("failed to mark task rejected: %w", err)
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
	return nil
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

	metadata := ""
	if req.SessionID != "" {
		metadataMap := map[string]interface{}{
			"session_id":     req.SessionID,
			"handoff_source": req.TaskID,
		}
		if data, err := json.Marshal(metadataMap); err == nil {
			metadata = string(data)
		}
	}

	// Send to target agent's inbox
	_, err = d.msgStore.CreateMessage(
		"",                                // New thread
		"ailang_instance", "coordinator",  // from
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
