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

	// The work id: what this completion is asking approval FOR. It is what lets
	// a later collision tell a replay from a re-run.
	var cf []string
	if v, ok := approvalContext["changed_files"].([]string); ok {
		cf = v
	}
	ds, _ := approvalContext["diff_stat"].(string)
	if wid := WorkIDForApproval(cf, ds); wid != "" {
		approvalContext["work_id"] = wid
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
		// The approval row already exists, and that is THREE different events
		// wearing one shape:
		//
		//   a replay of this completion        -> nothing to add
		//   a human already decided it         -> must not be disturbed
		//   a LATER EXECUTION with new work    -> needs a fresh decision
		//
		// Treating all three as superseded stranded eleven tasks on 2026-09-15:
		// rejected, re-dispatched, ran again, and each finished
		// `pending_approval` behind a `rejected` record — invisible to
		// `coordinator approvals` and refused by approve, because an
		// already-resolved approval cannot be resolved again.
		existing, getErr := f.deps.TaskStore.GetApprovalRequestByTaskAnyStatus(ctx, f.in.Task.ID)
		if getErr != nil {
			// Cannot tell them apart, so change nothing. The old behaviour is
			// the safe one when the evidence is missing.
			f.deps.logf("finalize %s: approval exists and could not be read (%v) — leaving the standing decision alone",
				f.in.Task.ID, getErr)
			return FinalizationSuperseded, nil
		}
		newWorkID, _ := approvalContext["work_id"].(string)
		switch ClassifyApprovalCollision(existing, newWorkID) {
		case CollisionStaleCard:
			// Nobody has decided anything yet, so nothing is reopened — the
			// PENDING card is simply describing work that no longer exists, and
			// it has to be replaced before someone approves the wrong evidence.
			refreshed, rErr := f.deps.TaskStore.RefreshPendingApproval(ctx, f.in.Task.ID, description, contextJSON)
			if rErr != nil {
				return FinalizationPending, rErr
			}
			if !refreshed {
				// The row moved out of pending between the read and the write —
				// a human decided it. That is the reopen case, and it belongs to
				// the NEXT delivery, which will read the resolved status.
				return FinalizationSuperseded, nil
			}
			f.deps.logf("finalize %s: a later execution produced DIFFERENT work (%s) than the pending approval described — card refreshed before anyone reads it",
				f.in.Task.ID, newWorkID)
			return FinalizationDone, nil
		case CollisionNewWork:
			// handled below
		default:
			return FinalizationSuperseded, nil
		}
		reopened, rErr := f.deps.TaskStore.ReopenApprovalForNewWork(ctx, f.in.Task.ID, description, contextJSON)
		if rErr != nil {
			return FinalizationPending, rErr
		}
		if !reopened {
			return FinalizationSuperseded, nil
		}
		// Loud: a decision that was made has been set aside because the work it
		// described no longer exists.
		f.deps.logf("finalize %s: a later execution produced DIFFERENT work (%s) than the %s approval described — reopened for a fresh decision",
			f.in.Task.ID, newWorkID, existing.Status)
		f.notifyApproval(ctx, description)
		return FinalizationDone, nil
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

		// handoffContent, not a body built here.
		//
		// This was the THIRD handoff producer, with its own wording, its own
		// title format, no artifact line and a 500-character truncation of the
		// request. Three producers meant three descriptions of the same event:
		// one of them accidentally dodged dedup by being worded differently
		// (2026-09-14, 13:33), and this one never named the artifact at all — an
		// omission masked for as long as handoffs nested, because the parent's
		// envelope happened to mention it.
		//
		// Sharing the builder is the point. A fourth description of the same
		// event is not a feature anyone asked for.
		source := f.deps.AgentRegistry.GetAgentByID(f.in.Task.AgentID)
		if source == nil {
			source = &AgentConfig{ID: f.in.Task.AgentID, Label: f.in.Task.AgentID}
		}
		artifacts := resolveHandoffArtifacts(ctx, f.deps.TaskStore, f.in.Task, source)
		body := handoffContent(source, f.in.Task, f.in.Task.GithubIssue, artifacts)

		created, err := f.deps.MsgStore.PutMessageIfAbsent(ctx, &messaging.InboxMessage{
			ID:           HandoffMessageID(f.in.Task.ID, targetID),
			FromAgent:    "coordinator",
			ToInbox:      target.Inbox,
			MessageType:  messaging.InboxTypeHandoff,
			Title:        handoffTitle(f.in.Task.Title),
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
