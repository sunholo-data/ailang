package coordinator

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/observatory"
)

// M-COMPLETION-PATH-PARITY M1 — the outcome × effect matrix, asserted behaviourally.
//
// completion_matrix_test.go pinned this matrix STRUCTURALLY against the daemon
// path before anything moved, because the effects lived inside executeTask and
// could not be exercised. Now they can. These are the same columns, run for real.
//
// The matrix is the parity contract: preserving the daemon's behaviour
// column-for-column is what makes the extraction safe, and three quorum rounds
// corrected this matrix (V43, V45, V48) — the last finding a thirteenth effect
// that had appeared in no version of it.

type finalizeHarness struct {
	deps   *FinalizeDeps
	store  *SQLiteStore
	msgs   *messaging.Store
	obs    *observatory.SQLiteBackend
	task   *TaskRecord
	stage  string
	chain  string
	cancel func()
}

// nilStrategy is a cloud-shaped strategy with no diff source yet: M3 supplies the
// real one. It returns the explicit error rather than an empty diff, so the
// approval records "unavailable" instead of a confident "Files (0)".
type nilStrategy struct{ kind StrategyKind }

func (s nilStrategy) Kind() StrategyKind { return s.kind }
func (s nilStrategy) DiffSource(ctx context.Context, task *TaskRecord) (DiffResult, error) {
	return DiffResult{}, ErrNoDiffSource
}

func newFinalizeHarness(t *testing.T, agent *AgentConfig) *finalizeHarness {
	t.Helper()
	dir := t.TempDir()

	store, err := NewSQLiteStore(filepath.Join(dir, "coordinator.db"))
	if err != nil {
		t.Fatalf("coordinator store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	msgs, err := messaging.OpenStore(filepath.Join(dir, "collaboration.db"))
	if err != nil {
		t.Fatalf("messaging store: %v", err)
	}
	t.Cleanup(func() { _ = msgs.Close() })

	obs, err := observatory.NewSQLiteBackendFromPath(filepath.Join(dir, "observatory.db"))
	if err != nil {
		t.Fatalf("observatory: %v", err)
	}

	registry := NewAgentRegistry()
	if agent != nil {
		if err := registry.Register(agent); err != nil {
			t.Fatalf("register agent: %v", err)
		}
		if err := registry.Register(&AgentConfig{ID: "sprint-planner", Inbox: "sprint-planner"}); err != nil {
			t.Fatalf("register target: %v", err)
		}
	}

	ctx := context.Background()
	chain, err := obs.CreateChain(ctx, &observatory.ChainCreateRequest{
		SourceType: observatory.ChainSourceMessage,
		SourceRef:  "inbox_matrix_test",
	})
	if err != nil {
		t.Fatalf("chain: %v", err)
	}
	stage, err := obs.CreateStage(ctx, &observatory.StageCreateRequest{
		ChainID: chain.ID, AgentID: "design-doc-creator", MessageID: "inbox_matrix_test", TaskID: "task-matrix",
	})
	if err != nil {
		t.Fatalf("stage: %v", err)
	}

	task := &TaskRecord{
		ID:      "task-matrix",
		Title:   "deterministic lockfile",
		Content: "ailang lock is not deterministic",
		Status:  TaskStatusRunning,
		AgentID: "design-doc-creator",
		ChainID: chain.ID,
		StageID: stage.ID,
	}
	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	// CreateTask does not necessarily persist the running status; the matrix is
	// about transitions FROM a live task, so make that explicit.
	if _, err := store.CompareAndSetTaskStatus(ctx, task.ID, AllTaskStatuses(), TaskStatusRunning); err != nil {
		t.Fatalf("seed status: %v", err)
	}

	return &finalizeHarness{
		deps: &FinalizeDeps{
			TaskStore: store, MsgStore: msgs, ObsBackend: obs,
			AgentRegistry: registry, Owner: "coord-test",
		},
		store: store, msgs: msgs, obs: obs, task: task,
		stage: stage.ID, chain: chain.ID,
	}
}

func (h *finalizeHarness) finalize(t *testing.T, outcome CompletionOutcome, skipApproval bool) *FinalizeReport {
	t.Helper()
	report, err := FinalizeTaskCompletion(context.Background(), h.deps, FinalizeInput{
		Task:   h.task,
		Result: &ExecuteResult{Success: outcome == OutcomeCompleted, SessionID: "sess-1", Cost: 0.42, InputTokens: 200, OutputTokens: 100, NumTurns: 5, ToolCallCount: 9, Duration: 3 * time.Second, Error: "pi idle for 3m0s"},
		Outcome:      outcome,
		SkipApproval: skipApproval,
		BranchName:   "coordinator/task-matrix",
	}, nilStrategy{kind: StrategyKindCloud})
	if err != nil {
		t.Fatalf("finalize: %v", err)
	}
	return report
}

func (h *finalizeHarness) taskStatus(t *testing.T) TaskStatus {
	t.Helper()
	task, err := h.store.GetTask(context.Background(), h.task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	return task.Status
}

func (h *finalizeHarness) approval(t *testing.T) *ApprovalRequestRecord {
	t.Helper()
	got, err := h.store.GetApprovalRequest(context.Background(), ApprovalIDForTask(h.task.ID))
	if err != nil {
		return nil
	}
	return got
}

func (h *finalizeHarness) handoffs(t *testing.T) []messaging.InboxMessage {
	t.Helper()
	msgs, err := h.msgs.ListInboxMessages(messaging.InboxListOptions{Inbox: "sprint-planner"})
	if err != nil {
		t.Fatalf("list handoffs: %v", err)
	}
	return msgs
}

func handoffAgent(autoApprove bool) *AgentConfig {
	a := &AgentConfig{
		ID:                "design-doc-creator",
		Inbox:             "design-doc-creator",
		TriggerOnComplete: []string{"sprint-planner"},
	}
	if autoApprove {
		a.AutoApproveHandoffTo = []string{"sprint-planner"}
	}
	return a
}

// --- Matrix row 1: exactly one terminal status per outcome ---

func TestFinalize_TaskStatusPerOutcome(t *testing.T) {
	cases := []struct {
		name         string
		outcome      CompletionOutcome
		skipApproval bool
		want         TaskStatus
	}{
		{"completed/skip_approval", OutcomeCompleted, true, TaskStatusCompleted},
		{"completed/normal", OutcomeCompleted, false, TaskStatusPendingApproval},
		{"no_changes", OutcomeNoChanges, false, TaskStatusNoChanges},
		{"failed", OutcomeFailed, false, TaskStatusFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newFinalizeHarness(t, handoffAgent(false))
			h.finalize(t, tc.outcome, tc.skipApproval)
			if got := h.taskStatus(t); got != tc.want {
				t.Errorf("status = %q, want %q", got, tc.want)
			}
		})
	}
}

// --- Matrix row 3: an approval exists only for a normal successful completion ---

func TestFinalize_ApprovalOnlyOnNormalCompletion(t *testing.T) {
	cases := []struct {
		name         string
		outcome      CompletionOutcome
		skipApproval bool
		wantApproval bool
	}{
		{"completed/normal", OutcomeCompleted, false, true},
		{"completed/skip_approval", OutcomeCompleted, true, false},
		{"no_changes", OutcomeNoChanges, false, false},
		{"failed", OutcomeFailed, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newFinalizeHarness(t, handoffAgent(false))
			h.finalize(t, tc.outcome, tc.skipApproval)
			got := h.approval(t)
			if tc.wantApproval && got == nil {
				t.Error("no approval was created for a normal successful completion")
			}
			if !tc.wantApproval && got != nil {
				t.Errorf("an approval was created for %s — there is nothing to approve", tc.name)
			}
		})
	}
}

// TestFinalize_ApprovalEmbedsOnlyNonAutoTargets: auto-approved edges are
// dispatched immediately by the handoff effect. Embedding them in the approval as
// well would fire them a SECOND time when the merge is approved.
func TestFinalize_ApprovalEmbedsOnlyNonAutoTargets(t *testing.T) {
	h := newFinalizeHarness(t, handoffAgent(false)) // requires approval
	h.finalize(t, OutcomeCompleted, false)

	got := h.approval(t)
	if got == nil {
		t.Fatal("no approval created")
	}
	if got.Type != "merge_handoff" {
		t.Errorf("approval type = %q, want merge_handoff when a non-auto edge is configured", got.Type)
	}
	if !strings.Contains(got.ContextJSON, "sprint-planner") {
		t.Errorf("non-auto target not embedded in the approval: %s", got.ContextJSON)
	}

	auto := newFinalizeHarness(t, handoffAgent(true)) // auto-approved edge
	auto.finalize(t, OutcomeCompleted, false)
	autoApproval := auto.approval(t)
	if autoApproval == nil {
		t.Fatal("no approval created for the auto-edge case")
	}
	if strings.Contains(autoApproval.ContextJSON, "handoff_targets") {
		t.Errorf("an auto-approved target was ALSO embedded in the approval; approving the merge would dispatch it twice: %s", autoApproval.ContextJSON)
	}
}

// TestFinalize_ApprovalRecordsMissingDiffExplicitly guards #921: a card that
// renders a confident "Files (0)" gets approved blind. Measured twice.
func TestFinalize_ApprovalRecordsMissingDiffExplicitly(t *testing.T) {
	h := newFinalizeHarness(t, handoffAgent(false))
	h.finalize(t, OutcomeCompleted, false)

	got := h.approval(t)
	if got == nil {
		t.Fatal("no approval created")
	}
	if !strings.Contains(got.ContextJSON, "diff_unavailable") {
		t.Errorf("a missing diff must be recorded explicitly, not left as an empty file list: %s", got.ContextJSON)
	}
}

// --- Matrix row 5: handoffs ---

// TestFinalize_AutoEdgeDispatchesAtCompletionForBothSuccessBranches is V45 made
// executable: handleAgentHandoffs sits outside the approval if/else and inside
// the success arm, so auto edges fire at COMPLETION for both sub-branches.
func TestFinalize_AutoEdgeDispatchesAtCompletionForBothSuccessBranches(t *testing.T) {
	for _, skipApproval := range []bool{true, false} {
		name := "normal"
		if skipApproval {
			name = "skip_approval"
		}
		t.Run(name, func(t *testing.T) {
			h := newFinalizeHarness(t, handoffAgent(true))
			h.finalize(t, OutcomeCompleted, skipApproval)

			got := h.handoffs(t)
			if len(got) != 1 {
				t.Fatalf("sprint-planner received %d handoffs, want 1 — auto edges dispatch at completion for BOTH success sub-branches (V45)", len(got))
			}
			if got[0].FromAgent != "coordinator" {
				t.Errorf("handoff from = %q, want coordinator: the sender must never choose the target", got[0].FromAgent)
			}
			if got[0].ParentTaskID != "task-matrix" || got[0].ChainID != h.chain {
				t.Errorf("handoff lost its chain linkage: parent=%q chain=%q", got[0].ParentTaskID, got[0].ChainID)
			}
		})
	}
}

// TestFinalize_NeverHandsOffOnFailureOrNoChanges is the hazard that makes
// unsupervised chaining dangerous: the next agent building on an error, or on a
// run that produced nothing.
func TestFinalize_NeverHandsOffOnFailureOrNoChanges(t *testing.T) {
	for _, outcome := range []CompletionOutcome{OutcomeFailed, OutcomeNoChanges} {
		t.Run(string(outcome), func(t *testing.T) {
			h := newFinalizeHarness(t, handoffAgent(true))
			h.finalize(t, outcome, false)

			if got := h.handoffs(t); len(got) != 0 {
				t.Errorf("%d handoffs dispatched for a %s task; the next agent would build on it", len(got), outcome)
			}
		})
	}
}

// TestFinalize_NonAutoEdgeDoesNotDispatchAtCompletion: those wait for the merge
// approval. Dispatching at completion would bypass the gate entirely.
func TestFinalize_NonAutoEdgeDoesNotDispatchAtCompletion(t *testing.T) {
	h := newFinalizeHarness(t, handoffAgent(false))
	h.finalize(t, OutcomeCompleted, false)

	if got := h.handoffs(t); len(got) != 0 {
		t.Errorf("%d handoffs dispatched at completion for an approval-gated edge; that bypasses the gate", len(got))
	}
}

// --- Matrix rows 6-9, 13: chain, stage, metrics, session, error ---

func TestFinalize_StageAndChainAdvanceOnEveryOutcome(t *testing.T) {
	cases := []struct {
		name         string
		outcome      CompletionOutcome
		skipApproval bool
		wantStage    observatory.ChainStageStatus
		wantChain    observatory.ChainStatus
	}{
		{"completed/skip_approval", OutcomeCompleted, true, observatory.StageStatusCompleted, observatory.ChainStatusCompleted},
		{"completed/normal", OutcomeCompleted, false, observatory.StageStatusAwaitingApproval, observatory.ChainStatusPendingApproval},
		{"no_changes", OutcomeNoChanges, false, observatory.StageStatusCompleted, observatory.ChainStatusCompleted},
		{"failed", OutcomeFailed, false, observatory.StageStatusFailed, observatory.ChainStatusFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newFinalizeHarness(t, handoffAgent(false))
			h.finalize(t, tc.outcome, tc.skipApproval)

			ctx := context.Background()
			stage, err := h.obs.GetStage(ctx, h.stage)
			if err != nil {
				t.Fatalf("get stage: %v", err)
			}
			if stage.Status != tc.wantStage {
				t.Errorf("stage status = %q, want %q — a stage frozen at pending is what made every prod chain look empty", stage.Status, tc.wantStage)
			}
			chain, err := h.obs.GetChain(ctx, h.chain, observatory.ChainReadOptions{})
			if err != nil {
				t.Fatalf("get chain: %v", err)
			}
			if chain.Status != tc.wantChain {
				t.Errorf("chain status = %q, want %q — a chain left active is the leak this closes", chain.Status, tc.wantChain)
			}
		})
	}
}

// TestFinalize_MetricsRecordedOnEveryOutcomeIncludingFailure: a failed run still
// burned tokens. Dropping its metrics would make failures free in the rollup.
func TestFinalize_MetricsRecordedOnEveryOutcomeIncludingFailure(t *testing.T) {
	for _, outcome := range []CompletionOutcome{OutcomeCompleted, OutcomeNoChanges, OutcomeFailed} {
		t.Run(string(outcome), func(t *testing.T) {
			h := newFinalizeHarness(t, handoffAgent(false))
			h.finalize(t, outcome, false)

			ctx := context.Background()
			stage, err := h.obs.GetStage(ctx, h.stage)
			if err != nil {
				t.Fatalf("get stage: %v", err)
			}
			if stage.Cost != 0.42 {
				t.Errorf("stage cost = %v, want 0.42", stage.Cost)
			}
			if stage.SessionID != "sess-1" {
				t.Errorf("stage session = %q, want sess-1", stage.SessionID)
			}
			chain, err := h.obs.GetChain(ctx, h.chain, observatory.ChainReadOptions{})
			if err != nil {
				t.Fatalf("get chain: %v", err)
			}
			if chain.TotalCost != 0.42 {
				t.Errorf("chain total_cost = %v, want 0.42 (derived from the stage)", chain.TotalCost)
			}
			if chain.TotalTokens != 300 {
				t.Errorf("chain total_tokens = %d, want 300", chain.TotalTokens)
			}
		})
	}
}

// TestFinalize_StageErrorOnlyOnFailure is matrix row 13 — the effect thirteen
// revisions of the design doc omitted entirely.
func TestFinalize_StageErrorOnlyOnFailure(t *testing.T) {
	failed := newFinalizeHarness(t, handoffAgent(false))
	failed.finalize(t, OutcomeFailed, false)
	stage, err := failed.obs.GetStage(context.Background(), failed.stage)
	if err != nil {
		t.Fatalf("get stage: %v", err)
	}
	if stage.ErrorMessage == "" {
		t.Error("a failed task recorded no stage error")
	}

	ok := newFinalizeHarness(t, handoffAgent(false))
	ok.finalize(t, OutcomeCompleted, false)
	okStage, err := ok.obs.GetStage(context.Background(), ok.stage)
	if err != nil {
		t.Fatalf("get stage: %v", err)
	}
	if okStage.ErrorMessage != "" {
		t.Errorf("a successful task recorded a stage error: %q", okStage.ErrorMessage)
	}
}


