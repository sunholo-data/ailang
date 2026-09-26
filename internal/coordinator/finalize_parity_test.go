package coordinator

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/observatory"
)

// M-COMPLETION-PATH-PARITY M5 — the anti-drift arm.
//
// This defect existed for four months because nothing could see the two
// completion paths side by side. One grew ten side effects; the other stopped at
// two; and no test, log line or dashboard could tell you they had diverged. The
// only reason it surfaced at all is that someone asked whether the pipeline
// worked and the answer had to be checked rather than recalled.
//
// So the last milestone is the one that stops the class returning.

// TestParity_CloudPathRoutesThroughTheOrchestrator: if a future change adds an
// effect straight to the completion handler instead of the shared orchestrator,
// the two paths start diverging again from that moment.
func TestParity_CloudPathRoutesThroughTheOrchestrator(t *testing.T) {
	calls := callsInFunc(t, "pubsub_completion_handler.go", "handleCompletion")

	if !calls["FinalizeTaskCompletion"] {
		t.Fatal("the cloud completion path no longer calls FinalizeTaskCompletion — the two paths have started to diverge again")
	}

	// Effects must reach the store through the orchestrator, not from here.
	for effect, why := range map[string]string{
		"CreateApprovalRequest":  "approval creation belongs to the orchestrator, and must use the if-absent variant",
		"CreateApprovalIfAbsent": "approval creation belongs to the orchestrator",
		"UpdateChainStatus":      "chain progression belongs to the orchestrator",
		"SetStageStatus":         "stage progression belongs to the orchestrator",
		"UpdateStageMetrics":     "metrics belong to the orchestrator, and this accumulating variant double-counts on replay",
		"UpdateChainMetrics":     "metrics belong to the orchestrator, and this accumulating variant double-counts on replay",
	} {
		if calls[effect] {
			t.Errorf("the cloud completion path calls %s directly — %s", effect, why)
		}
	}
}

// TestParity_OrchestratorCoversEveryEffectTheDaemonPerforms.
//
// The daemon path is the one that works. Every effect it performs must have a
// counterpart in the orchestrator, or consolidating onto the orchestrator would
// silently drop behaviour — the exact failure this milestone guards against, run
// in the opposite direction.
//
// Effect 10 (ProcessStageCompletion) is the one deliberate exception: its GitHub
// surface is twelve non-idempotent comment POSTs, and it is inert for every
// message-sourced task, so it stays daemon-only with its own follow-up.
func TestParity_OrchestratorCoversEveryEffectTheDaemonPerforms(t *testing.T) {
	daemon := callsInFunc(t, "daemon_tasks_exec_run.go", "executeTask")
	orchestrator := effectCounterparts(t)

	// The daemon's effect vocabulary, and the orchestrator call that subsumes it.
	subsumed := map[string]string{
		"MarkTaskCompleted":       "CompareAndSetTaskStatus",
		"MarkTaskPendingApproval": "CompareAndSetTaskStatus",
		"MarkTaskFailed":          "CompareAndSetTaskStatus",
		"updateChainStageStatus":  "SetStageStatus",
		"updateChainStatus":       "UpdateChainStatus",
		"updateStageSession":      "UpdateStageSession",
		"updateStageMetrics":      "SetStageMetrics",
		"updateChainMetrics":      "RecomputeChainAggregates",
		"updateStageError":        "SetStageError",
		"CreateApprovalRequest":   "CreateApprovalIfAbsent",
		"handleAgentHandoffs":     "PutMessageIfAbsent",
	}

	for daemonCall, orchestratorCall := range subsumed {
		if !daemon[daemonCall] {
			t.Errorf("the daemon path no longer calls %s; this map is stale and the parity claim is unverified", daemonCall)
			continue
		}
		if !orchestrator[orchestratorCall] {
			t.Errorf("the daemon performs %s but the orchestrator has no %s — consolidating would drop this effect", daemonCall, orchestratorCall)
		}
	}

	// The documented exception, asserted rather than assumed: if effect 10 ever
	// moves into the orchestrator, its non-idempotent GitHub writes come with it
	// and the out-of-scope decision needs revisiting.
	if orchestrator["ProcessStageCompletion"] {
		t.Error("effect 10 (ProcessStageCompletion) is now in the orchestrator — it makes twelve non-idempotent comment POSTs and was scoped out deliberately; revisit that decision before shipping this")
	}
}

// TestParity_OrchestratorUsesOnlyReplaySafeWrites.
//
// The accumulating Update* family keeps its semantics for importers and
// evaluators, which is exactly why it is still reachable. A finalisation effect
// that calls one double-counts on every redelivery — invisibly, because the write
// succeeds.
func TestParity_OrchestratorUsesOnlyReplaySafeWrites(t *testing.T) {
	calls := effectCounterparts(t)

	for unsafe, why := range map[string]string{
		"UpdateStageMetrics":    "accumulates (cost = cost + ?); use SetStageMetrics",
		"UpdateChainMetrics":    "accumulates; use RecomputeChainAggregates",
		"UpdateStageStatus":     "also increments the chain's stages_completed counter; use SetStageStatus",
		"UpdateStageError":      "increments error_count; use SetStageError",
		"CreateApprovalRequest": "bare INSERT on SQLite, silent overwrite on Firestore; use CreateApprovalIfAbsent",
		"InsertInboxMessage":    "same split; use PutMessageIfAbsent",
	} {
		if calls[unsafe] {
			t.Errorf("finalisation calls %s, which is not replay-safe: %s", unsafe, why)
		}
	}
}

// TestParity_EveryLedgerEffectIsReachable: an effect constant nothing runs is a
// column of the matrix that silently does nothing.
func TestParity_EveryLedgerEffectIsReachable(t *testing.T) {
	src := readSource(t, "task_finalize.go") + readSource(t, "task_finalize_approval.go")

	// Effects the orchestrator owns. The remainder are daemon-only by design and
	// are named here so the exclusion is deliberate rather than an oversight.
	owned := []string{
		EffectTaskStatus, EffectStageStatus, EffectStageSession,
		EffectMetrics, EffectStageError, EffectChainStatus,
		EffectApproval, EffectHandoff,
	}
	for _, effect := range owned {
		constName := effectConstName(effect)
		if !strings.Contains(src, constName) {
			t.Errorf("ledger effect %s (%s) is declared but never used by the orchestrator", effect, constName)
		}
	}
}

// --- helpers ---

func readSource(t *testing.T, file string) string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	var sb strings.Builder
	ast.Inspect(f, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok {
			sb.WriteString(ident.Name)
			sb.WriteString(" ")
		}
		return true
	})
	return sb.String()
}

// callsInFunc returns the set of calls made inside one named function.
func callsInFunc(t *testing.T, file, fnName string) map[string]bool {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == fnName {
			return callsIn(fn)
		}
	}
	t.Fatalf("%s not found in %s — this guard is pointing at code that moved", fnName, file)
	return nil
}

// effectCounterparts returns every call the orchestrator makes, across both of
// its files.
func effectCounterparts(t *testing.T) map[string]bool {
	t.Helper()
	all := map[string]bool{}
	for _, file := range []string{"task_finalize.go", "task_finalize_approval.go"} {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for name := range callsIn(f) {
			all[name] = true
		}
	}
	return all
}

func effectConstName(effect string) string {
	switch effect {
	case EffectTaskStatus:
		return "EffectTaskStatus"
	case EffectStageStatus:
		return "EffectStageStatus"
	case EffectStageSession:
		return "EffectStageSession"
	case EffectMetrics:
		return "EffectMetrics"
	case EffectStageError:
		return "EffectStageError"
	case EffectChainStatus:
		return "EffectChainStatus"
	case EffectApproval:
		return "EffectApproval"
	case EffectHandoff:
		return "EffectHandoff"
	}
	return effect
}

var _ = observatory.StageStatusCompleted
