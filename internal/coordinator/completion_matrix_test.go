package coordinator

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
	"testing"
)

// M-COMPLETION-PATH-PARITY M0 — pin the daemon's completion behaviour BEFORE the
// extraction moves any of it.
//
// The extraction's whole safety argument is that it preserves the daemon path
// column-for-column. That claim is worthless unless the current columns are
// written down as something executable: the design doc's outcome × effect matrix
// was corrected THREE times in three quorum rounds (V43, V45, V48), the last of
// which found a thirteenth effect that had appeared in no matrix at all.
//
// So this test is deliberately inverted: it must pass on UNMODIFIED dev. It is
// not a RED-first arm. If it fails today, the matrix is wrong and the sprint
// stops until it is right.
//
// It asserts structure rather than behaviour on purpose. The effects live inside
// executeTask, which runs a real agent, so they cannot be exercised directly
// until M1 extracts them. What can be pinned now — and what actually broke in
// review — is WHICH BRANCH each effect sits in. V45 is the case in point:
// handleAgentHandoffs sits after the skipApproval if/else and inside the success
// arm, so auto-approved edges dispatch at completion for BOTH sub-branches. An
// earlier revision of the design read that as "on approval" and would have
// changed live behaviour while claiming to preserve it.
//
// Once M1 lands, the same matrix becomes assertable behaviourally against
// FinalizeTaskCompletion, and this file's job narrows to guarding the call sites.

// completionBranches holds the effect calls found in each arm of the daemon's
// completion block, keyed the same way as the design doc's matrix columns.
type completionBranches struct {
	skipApproval  map[string]bool // if result.Success → if skipApproval { … }
	normal        map[string]bool // if result.Success → else { … }
	successCommon map[string]bool // if result.Success → outside the nested if/else
	failure       map[string]bool // else { … } of if result.Success
}

// parseCompletionMatrix locates `if result.Success { … } else { … }` inside
// executeTask and reports the effect calls in each arm.
func parseCompletionMatrix(t *testing.T) *completionBranches {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "daemon_tasks_exec_run.go", nil, 0)
	if err != nil {
		t.Fatalf("parse daemon_tasks_exec_run.go: %v", err)
	}

	var execTask *ast.FuncDecl
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == "executeTask" && fn.Recv != nil {
			execTask = fn
			break
		}
	}
	if execTask == nil {
		t.Fatal("executeTask not found — the completion path moved; update this test deliberately, do not delete it")
	}

	// The outcome switch is `if result.Success { … } else { … }` — but it is NOT
	// the only one in the function. An earlier `if result.Success` drives
	// eventHandler.EmitStatus and also has an else, so selecting the first match
	// picks the wrong block. Select by structure instead: the completion block is
	// the one whose body directly contains `if skipApproval { … } else { … }`.
	var outcome *ast.IfStmt
	var approval *ast.IfStmt
	ast.Inspect(execTask, func(n ast.Node) bool {
		if outcome != nil {
			return false
		}
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok || ifStmt.Else == nil {
			return true
		}
		sel, ok := ifStmt.Cond.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		x, ok := sel.X.(*ast.Ident)
		if !ok || x.Name != "result" || sel.Sel.Name != "Success" {
			return true
		}
		for _, stmt := range ifStmt.Body.List {
			inner, ok := stmt.(*ast.IfStmt)
			if !ok || inner.Else == nil {
				continue
			}
			if ident, ok := inner.Cond.(*ast.Ident); ok && ident.Name == "skipApproval" {
				outcome, approval = ifStmt, inner
				return false
			}
		}
		return true
	})
	if outcome == nil {
		t.Fatal("the completion block (`if result.Success` whose body holds `if skipApproval { … } else { … }`) was not found in executeTask — the outcome branch structure changed")
	}

	b := &completionBranches{
		skipApproval:  callsIn(approval.Body),
		normal:        callsIn(approval.Else),
		successCommon: map[string]bool{},
		failure:       callsIn(outcome.Else),
	}

	// successCommon = calls in the success arm that are NOT inside the approval if/else.
	for _, stmt := range outcome.Body.List {
		if stmt == ast.Stmt(approval) {
			continue
		}
		for name := range callsIn(stmt) {
			b.successCommon[name] = true
		}
	}
	return b
}

// callsIn returns the set of called function/method names under a node. For a
// selector chain (d.taskStore.MarkTaskCompleted) it records the final selector,
// which is the effect being named in the matrix.
func callsIn(n ast.Node) map[string]bool {
	found := map[string]bool{}
	if n == nil {
		return found
	}
	ast.Inspect(n, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fun := call.Fun.(type) {
		case *ast.SelectorExpr:
			found[fun.Sel.Name] = true
		case *ast.Ident:
			found[fun.Name] = true
		}
		return true
	})
	return found
}

func mustHave(t *testing.T, arm string, calls map[string]bool, name string) {
	t.Helper()
	if !calls[name] {
		t.Errorf("matrix drift: %s is NOT called in the %s arm, but the design doc's matrix says it is\n  present: %s",
			name, arm, sortedKeys(calls))
	}
}

func mustNotHave(t *testing.T, arm string, calls map[string]bool, name, why string) {
	t.Helper()
	if calls[name] {
		t.Errorf("matrix drift: %s IS called in the %s arm, but must not be — %s", name, arm, why)
	}
}

func sortedKeys(m map[string]bool) string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

// TestCompletionMatrix_TaskStatusIsOnePerOutcome pins matrix row 1: exactly one
// terminal-status write per outcome, and never the wrong one.
func TestCompletionMatrix_TaskStatusIsOnePerOutcome(t *testing.T) {
	b := parseCompletionMatrix(t)

	mustHave(t, "skipApproval", b.skipApproval, "MarkTaskCompleted")
	mustNotHave(t, "skipApproval", b.skipApproval, "MarkTaskPendingApproval", "a skip_approval task is completed, not queued for review")

	mustHave(t, "normal", b.normal, "MarkTaskPendingApproval")
	mustNotHave(t, "normal", b.normal, "MarkTaskCompleted", "a normal task must wait for approval")

	mustHave(t, "failure", b.failure, "MarkTaskFailed")
	mustNotHave(t, "failure", b.failure, "MarkTaskCompleted", "a failed task is not completed")
	mustNotHave(t, "failure", b.failure, "MarkTaskPendingApproval", "a failed task has nothing to approve")
}

// TestCompletionMatrix_ApprovalOnlyOnNormalCompletion pins matrix row 3: the
// approval record — and the handoff targets embedded in it — exist only for a
// normal successful completion.
func TestCompletionMatrix_ApprovalOnlyOnNormalCompletion(t *testing.T) {
	b := parseCompletionMatrix(t)

	mustHave(t, "normal", b.normal, "CreateApprovalRequest")
	mustNotHave(t, "skipApproval", b.skipApproval, "CreateApprovalRequest", "skip_approval means no approval is created")
	mustNotHave(t, "failure", b.failure, "CreateApprovalRequest", "a failed task must never create an approval")
	mustNotHave(t, "successCommon", b.successCommon, "CreateApprovalRequest", "approval creation belongs to the normal sub-branch only")
}

// TestCompletionMatrix_HandoffsFireAtCompletionForAllSuccesses pins V45, the
// single most consequential correction in review.
//
// handleAgentHandoffs sits AFTER the skipApproval if/else and INSIDE the success
// arm. So auto-approved edges dispatch at completion for both sub-branches — not
// "on approval" for normal completions, as an earlier revision of the design
// claimed. Only NON-auto targets are embedded in the merge approval and wait for
// it. Getting this wrong would silently change live daemon behaviour under a
// commit message promising to preserve it.
//
// It must also never appear in the failure arm: a handoff from a failed task
// would start the next agent on work that errored — the exact hazard that makes
// unsupervised chaining dangerous.
func TestCompletionMatrix_HandoffsFireAtCompletionForAllSuccesses(t *testing.T) {
	b := parseCompletionMatrix(t)

	mustHave(t, "successCommon", b.successCommon, "handleAgentHandoffs")
	mustNotHave(t, "skipApproval", b.skipApproval, "handleAgentHandoffs",
		"it must sit outside the approval if/else so it covers BOTH sub-branches (V45)")
	mustNotHave(t, "normal", b.normal, "handleAgentHandoffs",
		"it must sit outside the approval if/else so it covers BOTH sub-branches (V45)")
	mustNotHave(t, "failure", b.failure, "handleAgentHandoffs",
		"a failed task must never hand off; the next agent would build on an error")
}

// TestCompletionMatrix_MetricsAndSessionOnEveryOutcome pins matrix row 8/9.
//
// A failed run still burned tokens, so the failure arm records metrics too. This
// cell was challenged in review as invented; the check (V48) showed it was right —
// and turned up updateStageError, a thirteenth effect that had appeared in no
// version of the matrix.
func TestCompletionMatrix_MetricsAndSessionOnEveryOutcome(t *testing.T) {
	b := parseCompletionMatrix(t)

	for _, arm := range []struct {
		name  string
		calls map[string]bool
	}{
		{"skipApproval", b.skipApproval},
		{"normal", b.normal},
		{"failure", b.failure},
	} {
		mustHave(t, arm.name, arm.calls, "updateStageMetrics")
		mustHave(t, arm.name, arm.calls, "updateChainMetrics")
		mustHave(t, arm.name, arm.calls, "updateStageSession")
		mustHave(t, arm.name, arm.calls, "updateChainStageStatus")
		mustHave(t, arm.name, arm.calls, "updateChainStatus")
	}
}

// TestCompletionMatrix_StageErrorOnlyOnFailure pins matrix row 13 — the effect
// thirteen revisions of the design doc omitted entirely (V48).
func TestCompletionMatrix_StageErrorOnlyOnFailure(t *testing.T) {
	b := parseCompletionMatrix(t)

	mustHave(t, "failure", b.failure, "updateStageError")
	mustNotHave(t, "skipApproval", b.skipApproval, "updateStageError", "a successful run has no stage error")
	mustNotHave(t, "normal", b.normal, "updateStageError", "a successful run has no stage error")
	mustNotHave(t, "successCommon", b.successCommon, "updateStageError", "a successful run has no stage error")
}

// TestCompletionMatrix_GitHubStageRoutingIsNormalCompletionOnly pins effect 10,
// which this sprint deliberately leaves daemon-only and unchanged. If it ever
// moves branches, the out-of-scope decision needs revisiting rather than
// silently inheriting.
func TestCompletionMatrix_GitHubStageRoutingIsNormalCompletionOnly(t *testing.T) {
	b := parseCompletionMatrix(t)

	mustHave(t, "normal", b.normal, "ProcessStageCompletion")
	mustNotHave(t, "skipApproval", b.skipApproval, "ProcessStageCompletion", "effect 10 is normal-completion only")
	mustNotHave(t, "failure", b.failure, "ProcessStageCompletion", "effect 10 is normal-completion only")
}

// TestCompletionMatrix_DaemonHasNoNoChangesArm pins the matrix's fourth column
// as EMPTY on this path (V48).
//
// no_changes exists only on the cloud path, added by M-COORDINATOR-EXECUTION-TRUST
// M2. There is therefore no daemon behaviour to preserve for it: the cloud
// column DEFINES behaviour rather than mirroring it, which is why it needed a
// ruling of its own (D5, Mark 2026-09-03: terminal, no approval, no handoff, no
// retry). If a no_changes arm ever appears here, that ruling must be revisited
// rather than quietly diverging.
func TestCompletionMatrix_DaemonHasNoNoChangesArm(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "daemon_tasks_exec_run.go", nil, 0)
	if err != nil {
		t.Fatalf("parse daemon_tasks_exec_run.go: %v", err)
	}

	var found []string
	ast.Inspect(file, func(n ast.Node) bool {
		ident, ok := n.(*ast.Ident)
		if ok && strings.Contains(ident.Name, "NoChanges") {
			found = append(found, ident.Name)
		}
		return true
	})
	if len(found) > 0 {
		t.Errorf("the daemon path now references no_changes (%s) — D5 ruled this status terminal with no approval, handoff or retry; confirm the daemon agrees before relying on the cloud column", strings.Join(found, ", "))
	}

	// Positive control: the same walk must find the statuses that ARE here, or
	// the instrument is blind and the negative above means nothing.
	var control bool
	ast.Inspect(file, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok && ident.Name == "MarkTaskFailed" {
			control = true
		}
		return true
	})
	if !control {
		t.Fatal("positive control failed: the AST walk cannot see MarkTaskFailed, so its no_changes negative proves nothing")
	}
}
