package coordinator

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/pubsub"
)

// Completion binding (M-SEC2 SEC2.2).
//
// The completions topic is a shared channel: every Cloud Run executor lane can
// publish to it, and a completion names its own task. Before this check the
// handler fetched whatever task a message named and finalised it, so any
// executor — including an external-user lane — could complete or fail ANY task
// in the store: force-fail a sibling, or complete an evaluator task with a
// fabricated verdict and release a handoff.
//
// Pub/Sub push authenticates the SUBSCRIPTION, not the original publisher, so
// the binding available here is data-level: the claim must come from the agent
// the task was dispatched to, and the task must be in a state in which a
// dispatched execution can still be reporting. It does not stop an executor
// that knows another task's agent ID (the store is readable from the lanes
// today); it closes cross-agent hijack, and completing work nobody dispatched.
//
// How AgentID reaches both sides — there is exactly one producer:
//
//	dispatchTasksCloud: DispatchParams.AgentID = task.AgentID
//	cloudrun.Dispatcher: env override AILANG_AGENT_ID = params.AgentID
//	execute-job (every executor variant image runs it): completion.AgentID =
//	    config.AgentID(), i.e. that env var, unmodified
//
// so a legitimate completion carries the task record's own AgentID verbatim.

// completionArrivalByStatus answers "can a completion from the executor this
// task was dispatched to legitimately arrive while the task is in this state?".
//
// true  — apply it (terminal states are then skipped as duplicates by the
//
//	handler's existing idempotency check, not rejected).
//
// false — reject it loudly.
//
// A table, like terminalByStatus, so the exhaustiveness test catches the next
// status someone adds instead of it defaulting into an answer.
var completionArrivalByStatus = map[TaskStatus]bool{
	// Not dispatched, or put back for redispatch. A cloud task is claimed
	// (pending -> queued) BEFORE its job is started, so no execution of the
	// current dispatch can report while it is pending. The one way a real
	// execution's claim lands here — the Jobs API errored after starting it and
	// the task was reset — loses nothing by being dropped: a pending task is
	// redispatched on the next tick. Local-lane tasks sit pending in the cloud
	// store by design; this is what stops a cloud executor completing them.
	TaskStatusPending: false,
	// The cloud in-flight states. Every cloud completion arrives in queued
	// (MarkTaskRunning is local-only); a Cloud Run task retry (max_retries=1)
	// re-runs with the same env and reports into the same state.
	TaskStatusQueued:  true,
	TaskStatusRunning: true,
	// A redelivery of a completion already applied. The status write lands
	// first and the ledger is written last, so a delivery that died mid-way is
	// finished by its redelivery ONLY if that redelivery is let through; the
	// status CAS then records itself superseded and the ledger skips what ran.
	TaskStatusPendingApproval: true,
	// Terminal: the handler skips these as duplicates (idempotent redelivery,
	// a retried job reporting after the first attempt, or a late report after
	// the stale detector failed the task). Let through here so the skip stays
	// the quiet idempotent path rather than a loud rejection.
	TaskStatusCompleted: true,
	TaskStatusNoChanges: true,
	TaskStatusFailed:    true,
	TaskStatusRejected:  true,
	TaskStatusCancelled: true,
	TaskStatusDuplicate: true,
	TaskStatusBlocked:   true,
}

// completionMayArriveIn reports whether a completion may be applied to (or, if
// terminal, skipped as a duplicate by) a task in status s. An unknown status is
// refused: treating it as dispatched would apply a claim to a record whose
// meaning this code does not know.
func completionMayArriveIn(s TaskStatus) bool {
	return completionArrivalByStatus[s]
}

// completionBindingViolation returns why a completion must not be applied to
// task, or "" when it is bound to the task's dispatch.
func completionBindingViolation(task *TaskRecord, c pubsub.TaskCompletion) string {
	if task == nil {
		return "no task record to bind the completion to"
	}
	// An empty identity binds to nothing. The executor refuses to run without
	// AILANG_AGENT_ID and the dispatcher refuses an inbox with no agent, so an
	// empty claim has no legitimate producer; a task whose own AgentID is empty
	// is a data defect the stale detector fails on its own clock.
	if c.AgentID == "" {
		return "the completion names no agent, so it cannot be bound to the task's dispatch"
	}
	if task.AgentID == "" {
		return "the task record names no agent, so no completion can be bound to it"
	}
	if c.AgentID != task.AgentID {
		return fmt.Sprintf("the completion claims agent %q but the task was dispatched to %q", c.AgentID, task.AgentID)
	}
	if !completionMayArriveIn(task.Status) {
		return fmt.Sprintf("the task is %q, a state no dispatched execution can be reporting from", task.Status)
	}
	return ""
}
