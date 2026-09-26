package coordinator

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
)

// The WHOLE chain, edge by edge, in one test.
//
// pipeline_stage_crossing_test.go proves ONE boundary works. That is not the
// same claim as "the pipeline runs": on 2026-09-14 every edge worked
// individually and the chain still could not complete, because the faults lived
// between them — a handoff that simhashed to its own parent, an artifact the
// next stage was never told about, a status nobody could report. Each edge
// passing is necessary and was never sufficient.
//
// So this walks designer -> planner -> executor -> evaluator using the code
// production uses at each transition, and asserts at every boundary the three
// things that were separately false during the week:
//
//	1. the next stage has a message, in the collection dispatch polls
//	2. the task built from it is NOT suppressed as a duplicate
//	3. it names the artifact the previous stage produced
//
// No models, no network, no cloud plane — every fault it covers was plumbing,
// and plumbing reproduces in milliseconds.

type chainFixture struct {
	store    *SQLiteStore
	msgStore messaging.MessageStore
	reg      *AgentRegistry
	dir      string
}

func newChainFixture(t *testing.T) *chainFixture {
	t.Helper()
	dir := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(dir, "coordinator.db"))
	if err != nil {
		t.Fatalf("coordinator store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	msgStore, err := messaging.OpenStore(filepath.Join(dir, "collaboration.db"))
	if err != nil {
		t.Fatalf("message store: %v", err)
	}
	t.Cleanup(func() { _ = msgStore.Close() })

	// The real prod topology as of 2026-09-14: one human gate at the design
	// doc, gated planner->executor, AUTO executor->evaluator.
	reg := NewAgentRegistry()
	for _, a := range []*AgentConfig{
		{ID: "design-doc-creator", Inbox: "design-doc-creator", Label: "Design Doc Creator",
			Workspace: dir, TriggerOnComplete: []string{"sprint-planner"},
			ArtifactPatterns: []string{"design_docs/**/*.md"}},
		{ID: "sprint-planner", Inbox: "sprint-planner", Label: "Sprint Planner",
			Workspace: dir, TriggerOnComplete: []string{"sprint-executor"},
			ArtifactPatterns: []string{"design_docs/**/*.md", ".ailang/state/sprints/*.json"}},
		{ID: "sprint-executor", Inbox: "sprint-executor", Label: "Sprint Executor",
			Workspace: dir, TriggerOnComplete: []string{"sprint-evaluator"},
			AutoApproveHandoffTo: []string{"sprint-evaluator"},
			ArtifactPatterns:     []string{"**/*.go", "**/*.md", ".github/workflows/*.yml"}},
		{ID: "sprint-evaluator", Inbox: "sprint-evaluator", Label: "Sprint Evaluator",
			Workspace: dir, SkipApproval: true},
	} {
		if err := reg.Register(a); err != nil {
			t.Fatalf("register %s: %v", a.ID, err)
		}
	}
	return &chainFixture{store: store, msgStore: msgStore, reg: reg, dir: dir}
}

// stageDone puts a task in the state a finished agent leaves it: pending
// approval, fingerprinted, with an approval record carrying its gated targets.
func (f *chainFixture) stageDone(t *testing.T, id, agentID, content, artifact string, gated []string) *TaskRecord {
	t.Helper()
	ctx := context.Background()
	task := &TaskRecord{
		ID: id, AgentID: agentID, Title: "stage " + agentID, Content: content,
		Type: TaskTypeFeature, Status: TaskStatusPending, Workspace: f.dir,
		BaseBranch: "dev", CreatedAt: time.Now(),
	}
	if err := f.store.CreateTask(ctx, task); err != nil {
		t.Fatalf("create %s: %v", id, err)
	}
	if artifact != "" {
		if err := f.store.SetTaskDesignDocPath(ctx, id, artifact); err != nil {
			t.Fatalf("artifact %s: %v", id, err)
		}
		task.DesignDocPath = artifact
	}
	fp := NewTaskAnalyzer(0.8).Analyze(&Task{ID: id, Content: content}).Fingerprint
	if err := f.store.SetTaskFingerprint(ctx, id, fp); err != nil {
		t.Fatalf("fingerprint %s: %v", id, err)
	}
	if err := f.store.MarkTaskPendingApproval(ctx, id, f.dir, "coordinator/"+id, "dev", "", &ExecuteResult{Success: true}); err != nil {
		t.Fatalf("pending %s: %v", id, err)
	}
	ctxJSON := ""
	if len(gated) > 0 {
		b, _ := json.Marshal(map[string]any{"handoff_targets": gated, "source_agent": agentID})
		ctxJSON = string(b)
	}
	if err := f.store.CreateApprovalRequest(ctx, &ApprovalRequestRecord{
		ID: ApprovalIDForTask(id), TaskID: id, Type: string(ApprovalTypeMerge),
		Description: "done", Status: "pending", ContextJSON: ctxJSON, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("approval %s: %v", id, err)
	}
	return task
}

// crossBoundary asserts the three properties at one edge and returns the task
// the next stage would run.
func (f *chainFixture) crossBoundary(t *testing.T, from *TaskRecord, toInbox, toAgent, wantArtifact string) *TaskRecord {
	t.Helper()
	ctx := context.Background()

	msgs, err := f.msgStore.ListInboxMessages(messaging.InboxListOptions{Inbox: toInbox})
	if err != nil {
		t.Fatalf("list %s: %v", toInbox, err)
	}
	if len(msgs) == 0 {
		t.Fatalf("%s -> %s: NOTHING in the next stage's inbox; the chain stops here", from.AgentID, toAgent)
	}
	m := msgs[len(msgs)-1]

	// 1. the parent link dedup needs
	if m.ParentTaskID != from.ID {
		t.Errorf("%s -> %s: ParentTaskID %q, want %q — without it dedup suppresses this against its own parent",
			from.AgentID, toAgent, m.ParentTaskID, from.ID)
	}
	// 2. the artifact the next stage must work FROM
	if wantArtifact != "" && !strings.Contains(m.Payload, wantArtifact) {
		t.Errorf("%s -> %s: handoff does not name %q, so the next stage is told to continue without saying from what:\n%s",
			from.AgentID, toAgent, wantArtifact, m.Payload)
	}
	// 3. the task built from it must survive dedup
	next := &TaskRecord{
		ID: "task-" + toAgent, AgentID: toAgent, ParentTaskID: m.ParentTaskID,
		Content: m.Payload, Workspace: f.dir, BaseBranch: "dev",
	}
	// Both fingerprints, deliberately.
	//
	// The handoff's OWN hash is the ordinary case; the PARENT's is the one that
	// broke production. A handoff embeds its predecessor's request verbatim, so
	// in the real system the two collide — but whether a given fixture's wording
	// happens to collide is an accident, and a test that only checks its own
	// hash asserts nothing on the day it does not. Checking the parent's makes
	// the collision certain rather than hoped for.
	ownFP := NewTaskAnalyzer(0.8).Analyze(&Task{ID: next.ID, Content: next.Content}).Fingerprint
	parentFP := NewTaskAnalyzer(0.8).Analyze(&Task{ID: from.ID, Content: from.Content}).Fingerprint
	for label, fp := range map[string]uint64{"handoff": ownFP, "parent": parentFP} {
		dup, err := f.store.FindDuplicateTask(ctx, fp, DedupScopeFor(next, time.Now()))
		if err != nil {
			t.Fatalf("dedup lookup (%s fingerprint): %v", label, err)
		}
		if dup != nil {
			t.Fatalf("%s -> %s: the next task was SUPPRESSED by %s (%s) on the %s fingerprint — the pipeline cannot advance",
				from.AgentID, toAgent, dup.ID, dup.Status, label)
		}
	}
	return next
}

func (f *chainFixture) approve(t *testing.T, taskID string) *ApprovalResult {
	t.Helper()
	res, err := ProcessApprovalRequest(context.Background(), &ApprovalParams{
		TaskID: taskID, Action: "approve", ApprovedBy: "chain-test", Channel: "cli",
		Store: f.store, MsgStore: f.msgStore, AgentRegistry: f.reg, SkipMerge: true,
	})
	if err != nil {
		t.Fatalf("approve %s: %v", taskID, err)
	}
	return res
}

// TestFullChain_DesignDocToEvaluator walks all four stages.
func TestFullChain_DesignDocToEvaluator(t *testing.T) {
	f := newChainFixture(t)
	ctx := context.Background()

	// ---- stage 1: the design doc, approved by a human. The ONE gate.
	doc := f.stageDone(t, "task-designer", "design-doc-creator",
		"Design a secondary-model fallback for cloud executor agents",
		"design_docs/planned/m-fallback.md", []string{"sprint-planner"})
	if res := f.approve(t, doc.ID); !strings.Contains(res.Message, "sprint-planner") {
		t.Fatalf("approving the design doc must dispatch sprint-planner, got %q", res.Message)
	}
	plannerTask := f.crossBoundary(t, doc, "sprint-planner", "sprint-planner",
		"design_docs/planned/m-fallback.md")
	if err := f.store.MarkTaskCompleted(ctx, doc.ID, &ExecuteResult{Success: true}); err != nil {
		t.Fatalf("complete designer: %v", err)
	}

	// ---- stage 2: the plan. Gated, so approving it is what starts the executor.
	plan := f.stageDone(t, plannerTask.ID, "sprint-planner", plannerTask.Content,
		".ailang/state/sprints/sprint_M-FALLBACK.json", []string{"sprint-executor"})
	if res := f.approve(t, plan.ID); !strings.Contains(res.Message, "sprint-executor") {
		t.Fatalf("approving the plan must dispatch sprint-executor, got %q", res.Message)
	}
	execTask := f.crossBoundary(t, plan, "sprint-executor", "sprint-executor",
		".ailang/state/sprints/sprint_M-FALLBACK.json")
	if err := f.store.MarkTaskCompleted(ctx, plan.ID, &ExecuteResult{Success: true}); err != nil {
		t.Fatalf("complete planner: %v", err)
	}

	// ---- stage 3 -> 4: an AUTO edge. It fires at COMPLETION, through the
	// finalizer, not at approval — and the two paths must not BOTH fire it.
	impl := &TaskRecord{
		ID: execTask.ID, AgentID: "sprint-executor", Title: "impl",
		Content: execTask.Content, Type: TaskTypeFeature, Status: TaskStatusPending,
		Workspace: f.dir, BaseBranch: "dev", CreatedAt: time.Now(),
	}
	if err := f.store.CreateTask(ctx, impl); err != nil {
		t.Fatalf("create executor task: %v", err)
	}
	if err := f.store.SetTaskDesignDocPath(ctx, impl.ID, "design_docs/planned/m-fallback.md"); err != nil {
		t.Fatalf("artifact: %v", err)
	}
	impl.DesignDocPath = "design_docs/planned/m-fallback.md"

	if _, err := FinalizeTaskCompletion(ctx, &FinalizeDeps{
		TaskStore: f.store, MsgStore: f.msgStore, AgentRegistry: f.reg, Owner: "chain-test",
	}, FinalizeInput{
		Task: impl, Result: &ExecuteResult{Success: true},
		Outcome: OutcomeCompleted, BranchName: "coordinator/" + impl.ID,
	}, nil); err != nil {
		t.Fatalf("finalize executor: %v", err)
	}

	f.crossBoundary(t, impl, "sprint-evaluator", "sprint-evaluator",
		"design_docs/planned/m-fallback.md")

	// The auto edge must have fired ONCE. Approving afterwards must not send a
	// second: approvalHandoffTargets and autoHandoffTargets partition
	// TriggerOnComplete, and a drift there double-runs an agent.
	before, err := f.msgStore.ListInboxMessages(messaging.InboxListOptions{Inbox: "sprint-evaluator"})
	if err != nil {
		t.Fatalf("list evaluator: %v", err)
	}
	if len(before) != 1 {
		t.Fatalf("evaluator inbox has %d messages after ONE completion, want 1", len(before))
	}

	// ...and APPROVING it afterwards must not send a second. This is the half
	// the first version of this test missed: finalising alone cannot show a
	// double-dispatch, because only the approval path could produce the second
	// one. An auto edge released twice runs an executor twice on one plan.
	// No approval record is created here: FinalizeTaskCompletion already wrote
	// one (applyApproval), and trying again fails on the UNIQUE constraint —
	// which is the finalizer proving it did its half.
	f.approve(t, impl.ID)

	after, err := f.msgStore.ListInboxMessages(messaging.InboxListOptions{Inbox: "sprint-evaluator"})
	if err != nil {
		t.Fatalf("list evaluator: %v", err)
	}
	if len(after) != 1 {
		t.Fatalf("evaluator inbox has %d messages after completion AND approval, want 1 — "+
			"approvalHandoffTargets and autoHandoffTargets must partition TriggerOnComplete, "+
			"or an auto edge fires at both moments", len(after))
	}
}

// Every stage's outcome must be one the rest of the system can act on. A status
// no consumer maps is how a task sits forever looking actionable.
func TestFullChain_EveryStageOutcomeIsActionable(t *testing.T) {
	for _, st := range AllTaskStatuses() {
		if _, ok := prStatusVerdict[st]; !ok {
			t.Errorf("status %q has no PR verdict", st)
		}
		if _, ok := dedupSuppressesByStatus[st]; !ok {
			t.Errorf("status %q has no dedup rule", st)
		}
		if _, ok := terminalByStatus[st]; !ok {
			t.Errorf("status %q has no terminal rule", st)
		}
	}
}
