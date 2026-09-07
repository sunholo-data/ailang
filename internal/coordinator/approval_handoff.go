package coordinator

import (
	"context"
	"fmt"

	"github.com/sunholo-data/ailang/internal/messaging"
)

// Approval-gated handoffs, for every caller and not just the daemon
// (2026-09-07).
//
// The handoff topology has two dispatch moments, and only one of them worked:
//
//   - auto edges (auto_approve_handoffs, or a per-edge auto) dispatch at
//     COMPLETION, from the daemon's success arm via autoHandoffTargets. Fine.
//   - non-auto edges WAIT for approval, and the only code that released them was
//     Daemon.handleApproval -> tc.OnAgentApproved — a daemon method.
//
// ProcessApprovalRequest is the shared path used by the CLI and the dashboard,
// and it never called OnAgentApproved. It used TriggerOnComplete solely to
// compose a GitHub comment reading "Triggering next stage: X" — a claim nothing
// then acted on. So approving from anywhere but the daemon recorded the approval,
// dispatched nothing, and left the task pending_approval permanently, because an
// already-resolved approval cannot be approved again.
//
// Measured on two real prod chains (task-3807b3e1, task-58c17e89, 2026-09-07):
// both produced a design doc, both were approved, neither handed off. It is the
// same signature as tasks stranded since 2026-08-26. Every pipeline edge
// (design-doc-creator -> sprint-planner -> sprint-executor) is non-auto, so no
// pipeline edge could EVER chain from CLI or dashboard.
//
// The complement rule is what keeps this from double-dispatching: this fires
// exactly the targets autoHandoffTargets excluded. The two functions partition
// TriggerOnComplete, so a target dispatches once and at one moment.

// approvalHandoffTargets returns the edges that waited for this approval.
//
// Exact complement of finalizer.autoHandoffTargets: anything auto already
// dispatched at completion, and firing it again here would duplicate the work.
func approvalHandoffTargets(agent *AgentConfig) []string {
	if agent == nil {
		return nil
	}
	var out []string
	for _, target := range agent.TriggerOnComplete {
		if agent.AutoApproveHandoffs || agent.AutoApprovesHandoffTo(target) {
			continue // already dispatched at completion
		}
		out = append(out, target)
	}
	return out
}

// sendAgentHandoffMessage delivers one handoff into the target agent's INBOX.
//
// It must be an InboxMessage, not a thread message. `CreateMessage` writes to the
// thread-message collection, which dispatch never polls — so OnAgentApproved's
// handoffs were being written somewhere nothing reads, and the CLI reported
// "dispatched" for a message that could never become a task (measured
// 2026-09-07 on task-4082add5: approval said dispatched, no inbox message
// existed, no task appeared). The daemon's own deliverHandoffToInbox always used
// InsertInboxMessage; there were two mechanisms and only one worked.
//
// Shared by OnAgentApproved and the approval path so the two cannot diverge
// again — which is exactly how one of them ended up on the wrong collection.
func sendAgentHandoffMessage(
	msgStore messaging.MessageStore,
	sourceAgent, targetAgent *AgentConfig,
	task *TaskRecord,
	issueNumber int,
) error {
	if msgStore == nil {
		return fmt.Errorf("no message store: a handoff cannot be delivered")
	}
	if targetAgent == nil || targetAgent.Inbox == "" {
		return fmt.Errorf("target agent %q has no inbox to deliver to", targetAgentID(targetAgent))
	}

	content := fmt.Sprintf("**Handoff from %s**\n\n"+
		"Task: %s\n"+
		"GitHub Issue: #%d\n"+
		"Original Request: %s\n\n"+
		"Previous work has been approved. Please continue.",
		sourceAgent.Label, task.ID, issueNumber, task.Content)

	return msgStore.InsertInboxMessage(&messaging.InboxMessage{
		FromAgent:     "coordinator",
		ToInbox:       targetAgent.Inbox,
		MessageType:   "handoff",
		Title:         fmt.Sprintf("Handoff: %s", task.Title),
		Payload:       content,
		CorrelationID: task.ID,
		ParentTaskID:  task.ID,
		ChainID:       task.ChainID,
		Status:        messaging.InboxStatusUnread,
	})
}

// targetAgentID is nil-safe, for an error message that must not panic.
func targetAgentID(a *AgentConfig) string {
	if a == nil {
		return "<nil>"
	}
	return a.ID
}

// dispatchApprovalHandoffs fires the edges that were waiting on this approval.
//
// Returns the targets actually dispatched so the caller can report them: an
// approval that says "approved" while silently dispatching nothing is what this
// exists to end, so a caller that cannot see what happened is only half fixed.
func dispatchApprovalHandoffs(
	_ context.Context,
	registry *AgentRegistry,
	msgStore messaging.MessageStore,
	task *TaskRecord,
) (dispatched []string, err error) {
	if registry == nil || task == nil || task.AgentID == "" {
		return nil, nil
	}
	sourceAgent := registry.GetAgentByID(task.AgentID)
	if sourceAgent == nil {
		// Loud: the caller holds a registry that cannot see the task's own agent,
		// so it cannot know whether handoffs were owed.
		return nil, fmt.Errorf("agent %q not found in registry — cannot determine handoffs for task %s",
			task.AgentID, task.ID)
	}

	targets := approvalHandoffTargets(sourceAgent)
	if len(targets) == 0 {
		return nil, nil
	}
	if msgStore == nil {
		return nil, fmt.Errorf("task %s owes handoffs to %v but no message store is configured",
			task.ID, targets)
	}

	for _, targetID := range targets {
		targetAgent := registry.GetAgentByID(targetID)
		if targetAgent == nil {
			return dispatched, fmt.Errorf("handoff target %q not found in registry (task %s)", targetID, task.ID)
		}
		if sErr := sendAgentHandoffMessage(msgStore, sourceAgent, targetAgent, task, task.GithubIssue); sErr != nil {
			return dispatched, fmt.Errorf("handoff %s -> %s: %w", sourceAgent.ID, targetID, sErr)
		}
		dispatched = append(dispatched, targetID)
	}
	return dispatched, nil
}
