package coordinator

import (
	"strings"
	"testing"
)

func designer() *AgentConfig {
	return &AgentConfig{ID: "design-doc-creator", ArtifactPatterns: []string{"design_docs/**/*.md"}}
}

// The 22 orphans: a PR whose task is rejected, failed or cancelled will never
// merge, and leaving it open is how 24 accumulated in one afternoon.
func TestDecidePR_DeadTasksClose(t *testing.T) {
	for _, st := range []TaskStatus{
		TaskStatusRejected, TaskStatusFailed, TaskStatusCancelled,
		TaskStatusDuplicate, TaskStatusNoChanges,
	} {
		d := DecidePR(&TaskRecord{ID: "task-x", Status: st}, designer(), []string{"design_docs/a.md"})
		if d.Verdict != PRClose {
			t.Errorf("status %s: verdict %s, want close (%s)", st, d.Verdict, d.Reason)
		}
		if !strings.Contains(d.Reason, string(st)) {
			t.Errorf("status %s: reason must name it, got %q", st, d.Reason)
		}
	}
}

// An undecided task's PR is not ours to touch — that is the operator's open
// question, not an orphan.
func TestDecidePR_UndecidedTasksAreLeftAlone(t *testing.T) {
	for _, st := range []TaskStatus{
		TaskStatusPendingApproval, TaskStatusPending, TaskStatusQueued, TaskStatusRunning,
	} {
		if d := DecidePR(&TaskRecord{ID: "t", Status: st}, designer(), []string{"design_docs/a.md"}); d.Verdict != PRLeave {
			t.Errorf("status %s: verdict %s, want leave", st, d.Verdict)
		}
	}
}

// The live gap: task-00ea08bc was approved at 14:58 and its PR was still open
// at 15:25, because approval sets SkipMerge for a cloud task and that was read
// as "do nothing".
func TestDecidePR_CompletedAndInScopeMerges(t *testing.T) {
	d := DecidePR(
		&TaskRecord{ID: "task-00ea08bc", Status: TaskStatusCompleted},
		&AgentConfig{ID: "sprint-planner", ArtifactPatterns: []string{"design_docs/**/*.md", ".ailang/state/sprints/*.json"}},
		[]string{"design_docs/planned/v0_38_0/m-openrouter-eu-routing-sprint-plan.md"},
	)
	if d.Verdict != PRMerge {
		t.Fatalf("verdict %s, want merge (%s)", d.Verdict, d.Reason)
	}
}

// Merging is the one irreversible action here, so scope is checked
// mechanically even though a human approved the card.
func TestDecidePR_OutOfScopeFileRefusesTheMerge(t *testing.T) {
	d := DecidePR(
		&TaskRecord{ID: "t", Status: TaskStatusCompleted}, designer(),
		[]string{"design_docs/planned/ok.md", "internal/coordinator/daemon.go"},
	)
	if d.Verdict != PRRefuse {
		t.Fatalf("verdict %s, want refuse", d.Verdict)
	}
	if !strings.Contains(d.Reason, "daemon.go") {
		t.Errorf("the refusal must NAME the offending file: %q", d.Reason)
	}
}

// An undeclared pattern list defaults to `**/*`, which bounds nothing. Same
// rule auto-merge already applies, for the same reason.
func TestDecidePR_UndeclaredPatternsRefuseTheMerge(t *testing.T) {
	d := DecidePR(&TaskRecord{ID: "t", Status: TaskStatusCompleted},
		&AgentConfig{ID: "loose"}, []string{"anything.md"})
	if d.Verdict != PRRefuse {
		t.Errorf("verdict %s, want refuse: nothing bounds this merge", d.Verdict)
	}
}

// A PR reporting no files gives nothing to check, so there is nothing to trust.
func TestDecidePR_EmptyFileListRefusesTheMerge(t *testing.T) {
	if d := DecidePR(&TaskRecord{ID: "t", Status: TaskStatusCompleted}, designer(), nil); d.Verdict != PRRefuse {
		t.Errorf("verdict %s, want refuse", d.Verdict)
	}
}

func TestDecidePR_NilTaskLeavesItAlone(t *testing.T) {
	if d := DecidePR(nil, designer(), []string{"design_docs/a.md"}); d.Verdict != PRLeave {
		t.Errorf("verdict %s, want leave", d.Verdict)
	}
}

// The ratchet: every task status must have a declared verdict. A status added
// without one falls through to "leave", which is silent — and silence here is
// precisely the bug.
func TestPRStatusVerdictIsExhaustive(t *testing.T) {
	var missing []string
	for st := range terminalByStatus {
		if _, ok := prStatusVerdict[st]; !ok {
			missing = append(missing, string(st))
		}
	}
	for st := range dedupSuppressesByStatus {
		if _, ok := prStatusVerdict[st]; !ok {
			missing = append(missing, string(st))
		}
	}
	if len(missing) > 0 {
		t.Errorf("task status(es) %v have no PR verdict — they would silently leave a PR open forever. "+
			"Add them to prStatusVerdict.", missing)
	}
}

func TestBranchForTask(t *testing.T) {
	if got := BranchForTask("task-abc"); got != "coordinator/task-abc" {
		t.Errorf("got %q", got)
	}
}
