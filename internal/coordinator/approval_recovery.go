package coordinator

// Boot-time recovery of approval handoffs: replaying the handoffs an approved
// decision owes when the process that resolved it died before sending them,
// through the same first-write-wins sender the approve path uses
// (M-TASK-STATUS-TRUTH D3).

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/sunholo-data/ailang/internal/messaging"
)

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
		// Nothing owed is a decision too: record it, or this approval is
		// re-listed on every boot for the rest of the recovery window.
		if err := params.Store.MarkApprovalHandoffsTriggered(ctx, taskID, workIDFromContext(approvalReq.ContextJSON)); err != nil {
			span.AddEvent("warning: failed to record empty handoff decision", trace.WithAttributes(
				attribute.String("error", err.Error()),
			))
		}
		return false, nil
	}

	span.AddEvent("triggering embedded handoffs", trace.WithAttributes(
		attribute.StringSlice("handoff.targets", handoffContext.HandoffTargets),
		attribute.String("handoff.source", handoffContext.SourceAgent),
	))

	// Every target goes through the ONE approval-path sender: the same row
	// identity (task, target), the same body, the same suppression check. This
	// was a third producer with its own wording, a plain insert under a fresh id,
	// and no notify — so it could not see that the approve path had already sent
	// the handoff, and sent it again on every boot (M-TASK-STATUS-TRUTH S3).
	sourceAgent := params.AgentRegistry.GetAgentByID(handoffContext.SourceAgent)
	if sourceAgent == nil {
		sourceAgent = params.AgentRegistry.GetAgentByID(task.AgentID)
	}
	if sourceAgent == nil {
		sourceAgent = &AgentConfig{ID: handoffContext.SourceAgent, Label: handoffContext.SourceAgent}
	}
	artifacts := resolveHandoffArtifacts(ctx, params.Store, task, sourceAgent)

	triggered := false
	complete := true
	for _, targetAgentID := range handoffContext.HandoffTargets {
		targetAgent := params.AgentRegistry.GetAgentByID(targetAgentID)
		if targetAgent == nil {
			span.AddEvent("warning: handoff target not found", trace.WithAttributes(
				attribute.String("target.agent", targetAgentID),
			))
			complete = false
			continue
		}
		if legacyHandoffSent(params.MsgStore, targetAgent.Inbox, task.ID, approvalReq.ResolvedAt) {
			// Sent by the pre-M-TASK-STATUS-TRUTH approve path under a random id
			// and never recorded — the one case the (task, target, work) identity
			// cannot see. Re-sending it is the duplicate this work removes.
			triggered = true
			continue
		}
		err := sendAgentHandoffMessage(ctx, params.Store, params.MsgStore, sourceAgent, targetAgent, task, artifacts, task.GithubIssue)
		switch {
		case errors.Is(err, errHandoffSuppressed):
			return false, nil
		case errors.Is(err, errHandoffNotDispatched):
			// Written, not notified: the row is durable and the backstop sweep
			// delivers it. The handoff exists; only the fast path failed.
			span.AddEvent("warning: handoff written but not notified", trace.WithAttributes(
				attribute.String("target.agent", targetAgentID),
				attribute.String("error", err.Error()),
			))
			triggered = true
		case err != nil:
			span.AddEvent("warning: failed to send handoff", trace.WithAttributes(
				attribute.String("target.agent", targetAgentID),
				attribute.String("error", err.Error()),
			))
			complete = false
		default:
			triggered = true
		}
	}

	// Record the decision only when every target has its row. A partial pass
	// leaves the approval for the next boot, which re-writes exactly the missing
	// targets — the ones already written collide and do nothing.
	if complete {
		if err := params.Store.MarkApprovalHandoffsTriggered(ctx, taskID, workIDFromContext(approvalReq.ContextJSON)); err != nil {
			span.AddEvent("warning: failed to mark handoffs as triggered", trace.WithAttributes(
				attribute.String("error", err.Error()),
			))
		}
	}

	return triggered, nil
}

// legacyHandoffScan bounds legacyHandoffSent's read of one inbox.
const legacyHandoffScan = 500

// legacyHandoffSent reports whether the approve path that predates
// M-TASK-STATUS-TRUTH already delivered this handoff (quorum round 7).
//
// That path wrote handoff rows under a fresh random id (inbox_<ms>_<rand>) and
// never recorded that it had, so an approval it resolved in the window before
// this code deployed is still "without triggered handoffs" — and the
// deterministic id cannot collide with a random one. A row counts only if it
// has the legacy id form AND was written at or after this approval's
// resolution: an older handoff for the same task belongs to an earlier decision
// (ReopenApprovalForNewWork) and must not stand in for this one.
func legacyHandoffSent(msgStore messaging.MessageStore, inbox, taskID string, resolvedAt *time.Time) bool {
	if msgStore == nil || resolvedAt == nil {
		return false
	}
	rows, err := msgStore.ListInboxMessages(messaging.InboxListOptions{Inbox: inbox, Limit: legacyHandoffScan})
	if err != nil {
		return false // cannot tell; the first-write-wins path still bounds the damage to one legacy duplicate
	}
	for _, m := range rows {
		if m.ParentTaskID == taskID && m.MessageType == messaging.InboxTypeHandoff &&
			strings.HasPrefix(m.ID, "inbox_") && !m.CreatedAt.Before(resolvedAt.Add(-time.Minute)) {
			return true
		}
	}
	return false
}
