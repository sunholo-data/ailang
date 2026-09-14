package coordinator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

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

// errHandoffNotDispatched marks the case where the handoff row is durably stored
// but no dispatch notification went out. The work is not lost, but it will not
// start on its own — a distinction the caller must be able to report.
var errHandoffNotDispatched = errors.New("handoff stored but not dispatched")

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
	artifacts []string,
	issueNumber int,
) error {
	if msgStore == nil {
		return fmt.Errorf("no message store: a handoff cannot be delivered")
	}
	if targetAgent == nil || targetAgent.Inbox == "" {
		return fmt.Errorf("target agent %q has no inbox to deliver to", targetAgentID(targetAgent))
	}

	content := handoffContent(sourceAgent, task, issueNumber, artifacts)

	msg := &messaging.InboxMessage{
		FromAgent:     "coordinator",
		ToInbox:       targetAgent.Inbox,
		MessageType:   "handoff",
		Title:         handoffTitle(task.Title),
		Payload:       content,
		CorrelationID: task.ID,
		ParentTaskID:  task.ID,
		ChainID:       task.ChainID,
		Status:        messaging.InboxStatusUnread,
	}
	if err := msgStore.InsertInboxMessage(msg); err != nil {
		return err
	}

	// The row alone does not start work. The cloud coordinator dispatches from a
	// Pub/Sub notification — the daemon's own deliverHandoffToInbox says so:
	// "cross-machine chains hear handoffs only via Pub/Sub". Storing the message
	// and skipping this leaves a handoff that is visible in the inbox, correct in
	// every field, and never picked up (measured on task-f8ecc37c, 2026-09-07:
	// inbox row present within seconds, no task for minutes).
	//
	// A notify failure does NOT fail the handoff — the row is durable and a sweep
	// can still find it — but it must be VISIBLE, so it is returned as a typed
	// warning rather than swallowed.
	return notifyInboxMessage(msg)
}

// handoffContent is what the next agent actually reads.
//
// It used to say "Previous work has been approved. Please continue." and never
// name the ARTIFACT that work produced. design-doc-creator declares an
// output_marker of `DESIGN_DOC_PATH:`, the finalizer stores it on the task
// (task_chain.go SetTaskDesignDocPath), and the handoff then dropped it — so
// sprint-planner was asked to plan a design doc whose path it was never told,
// from a copy of the original REQUEST. The one fact the next stage needs was
// the one fact omitted.
//
// The GitHub issue line is likewise conditional. Cloud tasks have no issue, so
// it rendered "GitHub Issue: #0" — a reference to nothing, indistinguishable
// from a real one at a glance.
func handoffContent(sourceAgent *AgentConfig, task *TaskRecord, issueNumber int, artifacts []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**Handoff from %s**\n\n", sourceAgent.Label)
	fmt.Fprintf(&b, "Task: %s\n", task.ID)
	if issueNumber > 0 {
		fmt.Fprintf(&b, "GitHub Issue: #%d\n", issueNumber)
	}
	// The artifacts, most specific first. Named explicitly so the next stage
	// works from what was produced rather than re-deriving it from the request.
	if task.DesignDocPath != "" {
		fmt.Fprintf(&b, "Design doc: %s\n", task.DesignDocPath)
	}
	if task.SprintPlanPath != "" {
		fmt.Fprintf(&b, "Sprint plan: %s\n", task.SprintPlanPath)
	}
	// Anything else the task produced inside its declared artifact scope. This
	// is what carries a task whose marker was never recorded — see
	// resolveHandoffArtifacts.
	for _, a := range artifacts {
		if a == task.DesignDocPath || a == task.SprintPlanPath {
			continue
		}
		fmt.Fprintf(&b, "Artifact: %s\n", a)
	}
	// The branch the WORK is on, and the base to diff it against — both, named
	// as what they are.
	//
	// This said `Branch: <BaseBranch>`, which is the base the worktree was cut
	// FROM — "dev" — not the branch carrying the change. Measured 2026-09-14:
	// sprint-evaluator, whose whole job is to judge the previous stage's diff,
	// was told "Branch: dev", found nothing to evaluate, ran
	// `git diff origin/dev...HEAD` on its OWN empty branch and returned
	// FAIL 0/100 against a sprint it had never been handed. A wrong verdict on
	// unexamined work is worse than no verdict.
	if b2 := workBranchOf(task); b2 != "" {
		fmt.Fprintf(&b, "Work branch: %s\n", b2)
	}
	if task.BaseBranch != "" {
		fmt.Fprintf(&b, "Base branch: %s\n", task.BaseBranch)
	}
	fmt.Fprintf(&b, "\nOriginal Request: %s\n\n", rootRequestOf(task.Content))
	b.WriteString("Previous work has been approved. Please continue.")
	return b.String()
}

// resolveHandoffArtifacts names what the previous stage actually produced.
//
// DesignDocPath is populated from an OUTPUT MARKER the agent prints
// ("DESIGN_DOC_PATH:"), which makes it model-dependent: a run that does the work
// and forgets the marker records nothing, and the handoff then says "continue"
// without saying from what. Every design-doc task created before 2026-09-14 is
// in that state — nineteen of them sitting approved-pending — so this is not a
// hypothetical gap, it is the backlog.
//
// The approval record's changed_files is the mechanical answer to the same
// question: it is computed from the diff, not printed by a model. Filtering it
// through the agent's DECLARED artifact_patterns keeps it honest — the same
// bound auto-merge uses, and the reason patterns must be declared rather than
// defaulted to `**/*`.
//
// Order matters: the marker still WINS when present. It is the agent naming its
// own primary output, which a file list cannot distinguish among several.
func resolveHandoffArtifacts(ctx context.Context, store Store, task *TaskRecord, agent *AgentConfig) []string {
	if task == nil || agent == nil || len(agent.ArtifactPatterns) == 0 {
		return nil
	}
	if task.DesignDocPath != "" || task.SprintPlanPath != "" {
		return nil // already named
	}
	if store == nil {
		return nil
	}
	req, err := store.GetApprovalRequestByTaskAnyStatus(ctx, task.ID)
	if err != nil || req == nil || req.ContextJSON == "" {
		return nil
	}
	var ctxObj struct {
		ChangedFiles []string `json:"changed_files"`
	}
	if err := json.Unmarshal([]byte(req.ContextJSON), &ctxObj); err != nil {
		return nil
	}
	var out []string
	for _, f := range ctxObj.ChangedFiles {
		if MatchesArtifactPattern(agent.ArtifactPatterns, f) {
			out = append(out, f)
		}
	}
	sort.Strings(out) // deterministic: the same approval must render identically
	return out
}

// handoffTitle prefixes ONCE, however many stages the work has crossed.
//
// Each stage prefixed the parent's title unconditionally, so by the fourth the
// subject read
//
//	Handoff: Handoff: Handoff: Daneel design 8adb4ff62af619b745106cbe...
//
// measured on the first chain to reach the evaluator (2026-09-14). The one line
// a reader sees was three-quarters bookkeeping and the remaining quarter a hex
// digest.
func handoffTitle(parentTitle string) string {
	const p = "Handoff: "
	t := strings.TrimSpace(parentTitle)
	for strings.HasPrefix(t, p) {
		t = strings.TrimSpace(strings.TrimPrefix(t, p))
	}
	return p + t
}

// rootRequestOf unwraps nested handoff envelopes down to the ORIGINAL request.
//
// A handoff embeds its predecessor's content verbatim, and that content is
// itself a handoff once the chain is two stages deep — so each stage carried
// every stage before it. Measured on the same run: the planner's task was 1466
// bytes and the executor's 1728, all of it the same request re-quoted, with the
// actual ask at the bottom of three envelopes.
//
// That is not only waste. It is what makes consecutive stages simhash alike,
// which is the collision DedupScope exists to survive — and an agent reading its
// own instructions should not have to unwrap them first.
//
// Takes the LAST "Original Request:" because envelopes nest outermost-first, so
// the deepest one is the original.
func rootRequestOf(content string) string {
	const marker = "Original Request:"
	if !strings.HasPrefix(strings.TrimSpace(content), "**Handoff from") {
		return content // not an envelope: this IS the request
	}
	if i := strings.LastIndex(content, marker); i >= 0 {
		if inner := strings.TrimSpace(content[i+len(marker):]); inner != "" {
			return inner
		}
	}
	return content
}

// workBranchOf is the branch carrying a task's change.
//
// WorktreeID holds it when a worktree was used; otherwise the cloud wrapper's
// convention applies. Returns "" for a task that produced no branch, so the
// handoff omits the line rather than naming one that does not exist.
func workBranchOf(task *TaskRecord) string {
	if task == nil {
		return ""
	}
	if task.WorktreeID != "" {
		return task.WorktreeID
	}
	if task.ID != "" {
		return BranchForTask(task.ID)
	}
	return ""
}

// notifyInboxMessage publishes the dispatch notification for a stored message.
//
// Returns errHandoffNotDispatched when the row is safely stored but nothing was
// told about it, so the caller can say "stored, not dispatched" instead of
// reporting a handoff that will not start.
func notifyInboxMessage(msg *messaging.InboxMessage) error {
	cfg, err := messaging.LoadConfig()
	if err != nil {
		return fmt.Errorf("%w: messaging config unreadable: %v", errHandoffNotDispatched, err)
	}
	if cfg == nil || cfg.PubSub == nil || !cfg.PubSub.Enabled {
		return fmt.Errorf("%w: Pub/Sub notification is not enabled in the messaging config", errHandoffNotDispatched)
	}
	notifier, err := messaging.NewPubSubNotifier(cfg.PubSub)
	if err != nil {
		return fmt.Errorf("%w: %v", errHandoffNotDispatched, err)
	}
	if notifier == nil {
		return fmt.Errorf("%w: no notifier configured", errHandoffNotDispatched)
	}
	defer notifier.Close()

	if err := notifier.Notify(context.Background(), msg); err != nil {
		return fmt.Errorf("%w: %v", errHandoffNotDispatched, err)
	}
	return nil
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
	ctx context.Context,
	registry *AgentRegistry,
	msgStore messaging.MessageStore,
	store Store,
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
	// Resolved once, not per target: every target of the same task is being
	// handed the same work.
	artifacts := resolveHandoffArtifacts(ctx, store, task, sourceAgent)
	if msgStore == nil {
		return nil, fmt.Errorf("task %s owes handoffs to %v but no message store is configured",
			task.ID, targets)
	}

	for _, targetID := range targets {
		targetAgent := registry.GetAgentByID(targetID)
		if targetAgent == nil {
			return dispatched, fmt.Errorf("handoff target %q not found in registry (task %s)", targetID, task.ID)
		}
		sErr := sendAgentHandoffMessage(msgStore, sourceAgent, targetAgent, task, artifacts, task.GithubIssue)
		switch {
		case sErr == nil:
			dispatched = append(dispatched, targetID)
		case errors.Is(sErr, errHandoffNotDispatched):
			// Stored but not notified: report it as a warning naming the target,
			// never as a success and never as a lost handoff.
			return dispatched, fmt.Errorf("handoff %s -> %s: %w", sourceAgent.ID, targetID, sErr)
		default:
			return dispatched, fmt.Errorf("handoff %s -> %s: %w", sourceAgent.ID, targetID, sErr)
		}
	}
	return dispatched, nil
}
