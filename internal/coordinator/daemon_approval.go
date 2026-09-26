package coordinator

import (
	"github.com/sunholo-data/ailang/internal/messaging"
)

// handleAgentHandoffs checks if the completed task should trigger handoffs to other agents.
// This implements the agent-to-agent messaging with optional approval gates.
//
// The ExecuteResult is no longer read: the handoff body is handoffContent,
// which names the artifacts the stage produced rather than quoting the
// executor's raw output, and the session id it used to carry went into a
// thread-trail message nothing read (see handoff in approval_handoff.go).
func (d *Daemon) handleAgentHandoffs(task *TaskRecord, _ *ExecuteResult) error {
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

	// Resolved once, not per target: every target is handed the same work.
	artifacts := resolveHandoffArtifacts(d.ctx, d.taskStore, task, sourceAgent)

	// Process each handoff target
	for _, targetAgentID := range sourceAgent.TriggerOnComplete {
		targetAgent := d.agentRegistry.GetAgentByID(targetAgentID)
		if targetAgent == nil {
			d.logger.Printf("Warning: Handoff target agent %s not found in registry", targetAgentID)
			continue
		}

		if sourceAgent.AutoApprovesHandoffTo(targetAgentID) {
			// Auto-approve: send message directly to target agent's inbox
			if err := d.sendHandoffMessage(sourceAgent, targetAgent, task, artifacts); err != nil {
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

// sendHandoffMessage is the daemon's auto-approve entry into the ONE handoff
// sender (approval_handoff.go). Notification goes through the daemon's own
// publisher, which treats "no publisher" as a local plane where the poller
// reads the store directly — see publishInboxNotification.
func (d *Daemon) sendHandoffMessage(sourceAgent, targetAgent *AgentConfig, task *TaskRecord, artifacts []string) error {
	return handoffSender{
		msgStore: d.msgStore,
		notify: func(msg *messaging.InboxMessage) error {
			// Cross-machine chains hear handoffs only via Pub/Sub; the notify
			// daemon drops non-human inboxes, so this cannot spam Discord.
			d.publishInboxNotification(msg)
			return nil
		},
	}.send(handoff{
		Source:      sourceAgent,
		Target:      targetAgent,
		Task:        task,
		Artifacts:   artifacts,
		IssueNumber: task.GithubIssue,
	})
}

// truncateForAttribute truncates a string for use in span attributes.
// OTEL has limits on attribute value sizes.
func truncateForAttribute(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
