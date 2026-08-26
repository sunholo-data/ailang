package coordinator

import (
	"context"
	"fmt"
	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/pubsub"
)

// M-PIPELINE-RECONCILIATION M2 (D1(b)): attach an evaluator's verdict to the
// PARENT task's pending approval.
//
// The evaluator runs as its own task (per-edge auto-approved handoff from the
// executor) while the executor's merge approval is still pending. Its verdict
// must land on that approval — on SUCCESS as whatever it emitted, and on
// FAILURE as UNAVAILABLE, because "the evaluator died" is itself a verdict the
// approver needs to see. Nothing here ever blocks the human gate; the verdict
// only gates AUTOMATIC progression (ApprovalRequestRecord.AllowsAutomation).

// attachEvaluationToParent extracts the verdict for a completed-or-failed
// EvaluatesParent task and writes it to the parent's pending approval.
//
// evalErr non-empty means the evaluator task itself failed; the verdict is then
// UNAVAILABLE with that reason regardless of any output.
func (d *Daemon) attachEvaluationToParent(ctx context.Context, task *TaskRecord, output, evalErr string) {
	if task == nil || d.taskStore == nil {
		return
	}
	if task.ParentTaskID == "" {
		// An evaluator with no parent has nowhere to report. Loud: this is a
		// wiring mistake (the handoff metadata failed to carry parent_task_id),
		// not a normal condition.
		d.logger.Printf("ERROR: evaluator task %s has no parent_task_id — its verdict cannot reach any approval", task.ID)
		return
	}

	var verdict EvaluationVerdict
	if evalErr != "" {
		verdict = UnavailableVerdict(fmt.Sprintf("evaluator task %s failed: %s", task.ID, truncateVerdict(evalErr, 300)))
	} else {
		verdict = ExtractEvaluationVerdict(output)
	}

	if err := d.taskStore.UpdateApprovalEvaluationByTask(ctx, task.ParentTaskID, verdict.String()); err != nil {
		// The store already made "nowhere to land" an error; restate with both
		// ids so the operator can reconstruct the chain from one log line.
		d.logger.Printf("ERROR: could not attach evaluation %q from task %s to parent %s's approval: %v",
			verdict.String(), task.ID, task.ParentTaskID, err)
		return
	}
	d.logger.Printf("Evaluation attached to parent %s's approval: %s (from evaluator task %s)",
		task.ParentTaskID, verdict.String(), task.ID)
}

// agentEvaluatesParent reports whether the task's agent is an evaluator.
func (d *Daemon) agentEvaluatesParent(task *TaskRecord) bool {
	if d.agentRegistry == nil || task == nil || task.AgentID == "" {
		return false
	}
	agent := d.agentRegistry.GetAgentByID(task.AgentID)
	return agent != nil && agent.EvaluatesParent
}

// publishInboxNotification tells the message plane an inbox row exists.
//
// InsertInboxMessage writes the STORE; subscribers (the notify daemon, hence
// Discord) hear only the Pub/Sub notification, which the CLI send publishes
// but the coordinator historically did not — so coordinator-created approval
// requests never pinged anyone. Nil publisher (pure-local setups) is a no-op;
// on the shared plane init guarantees one.
func (d *Daemon) publishInboxNotification(msg *messaging.InboxMessage) {
	if d.pubsubPublisher == nil || msg == nil {
		return
	}
	if err := d.pubsubPublisher.PublishMessage(d.ctx, msg.ID, pubsub.MessageAttributes{
		Inbox:       msg.ToInbox,
		FromAgent:   msg.FromAgent,
		MessageType: msg.MessageType,
	}); err != nil {
		d.logger.Printf("Warning: inbox notification not published for %s (row stored; Discord will not ping): %v", msg.ID, err)
	}
}
