package coordinator

import (
	"encoding/json"
	"fmt"

	"github.com/sunholo-data/ailang/internal/messaging"
)

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

		if sourceAgent.AutoApprovesHandoffTo(targetAgentID) {
			// Auto-approve: send message directly to target agent's inbox
			if err := d.sendHandoffMessage(targetAgent, task, handoffMessage, sessionID); err != nil {
				d.logger.Printf("Warning: Failed to send handoff to %s: %v", targetAgentID, err)
				continue
			}
			d.logger.Printf("Auto-approved handoff from %s to %s for task %s",
				sourceAgentID, targetAgentID, task.ID)
		} else {
			// Handoff is embedded in merge approval (combined approval created in daemon_tasks.go)
			// Skip creating separate handoff approval - will be triggered when merge is approved
			d.logger.Printf("Handoff to %s embedded in merge approval for task %s (will trigger on approval)",
				targetAgentID, task.ID)
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
	// chain_id enables the new task to join the existing chain (M-CHAINS-SIMPLIFY)
	metadataMap := map[string]interface{}{
		"parent_task_id": task.ID,        // For hierarchy tracking
		"handoff_source": task.ID,        // Legacy field for backwards compatibility
		"source_agent":   task.AgentID,   // Which agent completed the work
		"target_agent":   targetAgent.ID, // Which agent is receiving the handoff
	}
	if sessionID != "" {
		metadataMap["session_id"] = sessionID
	}
	// Include chain_id so the new task joins the existing chain
	if task.ChainID != "" {
		metadataMap["chain_id"] = task.ChainID
	}
	metadata := ""
	if data, err := json.Marshal(metadataMap); err == nil {
		metadata = string(data)
	}

	// THE DELIVERY: an inbox row the poller reads. CreateMessage below is only
	// the human-visible thread trail — on Firestore it reaches no inbox at all
	// (see deliverHandoffToInbox).
	if err := d.deliverHandoffToInbox(targetAgent, task, fmt.Sprintf("Handoff: %s", task.Title), message); err != nil {
		return fmt.Errorf("handoff inbox delivery failed: %w", err)
	}

	// Thread trail (best-effort; the inbox row above is what triggers work).
	if _, err := d.msgStore.CreateMessage(
		"",                               // New thread (empty ThreadID)
		"ailang_instance", "coordinator", // from
		targetAgent.Inbox, targetAgent.ID, // to (inbox and agent)
		"handoff", // kind
		message,
		metadata,
	); err != nil {
		d.logger.Printf("Warning: handoff thread trail not recorded: %v", err)
	}
	return nil
}

// deliverHandoffToInbox is THE delivery step of a handoff: an inbox row the
// poller actually reads.
//
// Found live 2026-08-26 by the M-PIPELINE-RECONCILIATION e2e test: handoffs
// were sent with msgStore.CreateMessage, which on the Firestore backend writes
// ONLY the thread-messages collection — no inbox row — while the task poller
// consumes ListInboxMessages. So on a gcp-storage coordinator the handoff was
// "sent", logged as auto-approved, and landed in a collection nothing polls:
// the executor completed, and the evaluator it handed off to never existed.
// SQLite happened to work because its tables coincide; the parity gap was
// invisible until the first local-mode coordinator ran on shared storage
// (the rig joined the plane the same day).
func (d *Daemon) deliverHandoffToInbox(targetAgent *AgentConfig, task *TaskRecord, title, message string) error {
	msg := &messaging.InboxMessage{
		FromAgent:     "coordinator",
		ToInbox:       targetAgent.Inbox,
		MessageType:   "handoff",
		Title:         title,
		Payload:       message,
		CorrelationID: task.ID,
		ParentTaskID:  task.ID,
		ChainID:       task.ChainID,
		Status:        messaging.InboxStatusUnread,
	}
	if err := d.msgStore.InsertInboxMessage(msg); err != nil {
		return err
	}
	// Cross-machine chains hear handoffs only via Pub/Sub; the notify daemon
	// drops non-human inboxes, so this cannot spam Discord.
	d.publishInboxNotification(msg)
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
