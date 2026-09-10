package coordinator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
)

// The two effects that create new work: the approval record, and the handoff
// messages that start the next agent (M-COMPLETION-PATH-PARITY M1).
//
// Both write under a DETERMINISTIC id so a replay collides with itself instead of
// duplicating. That is the whole mechanism: the id is derived from the task, so
// the second delivery addresses the same row and the store's if-absent primitive
// turns it into a no-op.

// ApprovalIDForTask derives the approval id from the task id.
//
// The daemon has always used apr-<task hash>, via task.ID[5:] — which panics
// outright on any id shorter than six characters. Determinism is the property
// that matters, so this keeps the same shape without the assumption.
func ApprovalIDForTask(taskID string) string {
	return "apr-" + strings.TrimPrefix(taskID, "task-")
}

// HandoffMessageID derives a handoff message's id from its task and target, so a
// redelivered completion addresses the same message rather than dispatching the
// next agent a second time.
func HandoffMessageID(taskID, target string) string {
	return fmt.Sprintf("%s:handoff:%s", taskID, target)
}

// applyApproval creates the approval record for a normal successful completion.
//
// It also embeds the handoff targets that require approval. Auto-approved edges
// are dispatched immediately by the handoff effect and must NOT also be embedded
// here, or approving the merge would fire them a second time.
func (f *finalizer) applyApproval(ctx context.Context) (FinalizationState, error) {
	approvalContext := map[string]interface{}{}

	var embedded []string
	if f.deps.AgentRegistry != nil && f.in.Task.AgentID != "" {
		if agent := f.deps.AgentRegistry.GetAgentByID(f.in.Task.AgentID); agent != nil {
			for _, tgt := range agent.TriggerOnComplete {
				if !agent.AutoApproveHandoffs && !agent.AutoApprovesHandoffTo(tgt) {
					embedded = append(embedded, tgt)
				}
			}
		}
	}
	if len(embedded) > 0 {
		approvalContext["handoff_targets"] = embedded
		approvalContext["source_agent"] = f.in.Task.AgentID
		if f.in.Result != nil {
			approvalContext["session_id"] = f.in.Result.SessionID
		}
	}

	// The diff is the approval card's evidence. #921: a card that renders a
	// confident "Files (0)" gets approved blind — measured, twice. So a missing
	// diff is recorded as an explicit reason rather than an empty file list.
	if f.strategy != nil {
		diff, err := f.strategy.DiffSource(ctx, f.in.Task)
		switch {
		case err == nil:
			approvalContext["diff_stat"] = diff.Stat
			approvalContext["changed_files"] = diff.ChangedFiles
			approvalContext["diff"] = diff.Patch
		case errors.Is(err, ErrNoDiffSource):
			approvalContext["diff_unavailable"] = "this executor produced no diff source for the task"
			f.deps.logf("finalize %s: approval card has no diff (%s executor reported none)", f.in.Task.ID, f.strategy.Kind())
		default:
			approvalContext["diff_unavailable"] = err.Error()
			f.deps.logf("finalize %s: approval diff unavailable: %v", f.in.Task.ID, err)
		}
	}

	contextJSON := ""
	if len(approvalContext) > 0 {
		b, err := json.Marshal(approvalContext)
		if err != nil {
			return FinalizationPending, fmt.Errorf("encoding approval context: %w", err)
		}
		contextJSON = string(b)
	}

	approvalType := string(ApprovalTypeMerge)
	description := fmt.Sprintf("Agent completed work on: %s", f.in.Task.Title)
	if len(embedded) > 0 {
		approvalType = "merge_handoff"
		description = fmt.Sprintf("Agent completed work on: %s (will handoff to: %v)", f.in.Task.Title, embedded)
	}

	created, err := f.deps.TaskStore.CreateApprovalIfAbsent(ctx, &ApprovalRequestRecord{
		ID:          ApprovalIDForTask(f.in.Task.ID),
		TaskID:      f.in.Task.ID,
		Type:        approvalType,
		Description: description,
		ContextJSON: contextJSON,
		Status:      "pending",
		CreatedAt:   time.Now(),
	})
	if err != nil {
		return FinalizationPending, err
	}
	if !created {
		// The approval already exists — a replay, or a human has since resolved
		// it. Either way this completion has nothing left to add, and overwriting
		// would undo a decision.
		return FinalizationSuperseded, nil
	}

	f.notifyApproval(ctx, description)
	return FinalizationDone, nil
}

// notifyApproval posts the approval to the agent's inbox.
//
// The ping IS the product: an approval nobody hears about gets approved blind
// from a context-free queue. A failure here does not fail the effect — the
// approval itself is durable — but it is never silent.
func (f *finalizer) notifyApproval(ctx context.Context, description string) {
	if f.deps.MsgStore == nil {
		return
	}
	inbox := f.in.Task.AgentID
	if f.deps.AgentRegistry != nil {
		if resolved, ok := f.deps.AgentRegistry.InboxForAgent(f.in.Task.AgentID); ok {
			inbox = resolved
		}
	}
	if inbox == "" {
		return
	}

	msg := &messaging.InboxMessage{
		ID:           f.in.Task.ID + ":approval",
		FromAgent:    "coordinator",
		ToInbox:      inbox,
		MessageType:  messaging.InboxTypeApprovalRequest,
		Title:        fmt.Sprintf("Approval needed: %s", f.in.Task.Title),
		Payload:      description,
		ParentTaskID: f.in.Task.ID,
		ChainID:      f.in.Task.ChainID,
		Status:       messaging.InboxStatusUnread,
	}
	if _, err := f.deps.MsgStore.PutMessageIfAbsent(ctx, msg); err != nil {
		f.deps.logf("finalize %s: approval created but its notification failed (the queue will show it, nobody will be told): %v", f.in.Task.ID, err)
	}
}

// applyHandoff dispatches the auto-approved edges.
//
// The message is authored by the coordinator and addressed to an inbox resolved
// through the agent registry — the sender never chooses the target. It carries
// ParentTaskID and ChainID so the chain stays linked, which is what makes a
// multi-agent run legible afterwards.
func (f *finalizer) applyHandoff(ctx context.Context) (FinalizationState, error) {
	if f.deps.MsgStore == nil {
		return FinalizationPending, fmt.Errorf("no message store")
	}

	targets := f.autoHandoffTargets()
	var dispatched, skipped int

	for _, targetID := range targets {
		target := f.deps.AgentRegistry.GetAgentByID(targetID)
		if target == nil {
			// A configured edge pointing at an agent that does not exist is a
			// configuration error, and silently dropping it is how a pipeline
			// comes to stop without anyone noticing.
			f.deps.logf("finalize %s: handoff target %q is not a registered agent — the edge is configured but cannot be dispatched", f.in.Task.ID, targetID)
			continue
		}

		body := fmt.Sprintf("**Handoff from %s**\n\nTask: %s\nTitle: %s\n\nOriginal request:\n%s\n\nPlease continue this work.",
			f.in.Task.AgentID, f.in.Task.ID, f.in.Task.Title, truncateString(f.in.Task.Content, 500))

		created, err := f.deps.MsgStore.PutMessageIfAbsent(ctx, &messaging.InboxMessage{
			ID:           HandoffMessageID(f.in.Task.ID, targetID),
			FromAgent:    "coordinator",
			ToInbox:      target.Inbox,
			MessageType:  messaging.InboxTypeHandoff,
			Title:        fmt.Sprintf("Handoff: %s", f.in.Task.Title),
			Payload:      body,
			ParentTaskID: f.in.Task.ID,
			ChainID:      f.in.Task.ChainID,
			Status:       messaging.InboxStatusUnread,
		})
		if err != nil {
			return FinalizationPending, fmt.Errorf("dispatching handoff to %s: %w", targetID, err)
		}
		if created {
			dispatched++
			f.deps.logf("finalize %s: handed off to %s (inbox=%s, chain=%s)", f.in.Task.ID, targetID, target.Inbox, f.in.Task.ChainID)
		} else {
			skipped++
		}
	}

	if dispatched == 0 && skipped > 0 {
		// Every target was already dispatched by an earlier delivery.
		return FinalizationSuperseded, nil
	}
	return FinalizationDone, nil
}
