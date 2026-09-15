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
		d := DecidePR(&TaskRecord{ID: "task-x", Status: st}, designer(), []string{"design_docs/a.md"}, nil)
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
		if d := DecidePR(&TaskRecord{ID: "t", Status: st}, designer(), []string{"design_docs/a.md"}, nil); d.Verdict != PRLeave {
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
		[]string{"design_docs/planned/ok.md", "internal/coordinator/daemon.go"}, nil,
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
		&AgentConfig{ID: "loose"}, []string{"anything.md"}, nil)
	if d.Verdict != PRRefuse {
		t.Errorf("verdict %s, want refuse: nothing bounds this merge", d.Verdict)
	}
}

// A PR reporting no files gives nothing to check, so there is nothing to trust.
func TestDecidePR_EmptyFileListRefusesTheMerge(t *testing.T) {
	if d := DecidePR(&TaskRecord{ID: "t", Status: TaskStatusCompleted}, designer(), nil, nil); d.Verdict != PRRefuse {
		t.Errorf("verdict %s, want refuse", d.Verdict)
	}
}

func TestDecidePR_NilTaskLeavesItAlone(t *testing.T) {
	if d := DecidePR(nil, designer(), []string{"design_docs/a.md"}, nil); d.Verdict != PRLeave {
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

// The approve-A-merge-B hole: the card and the branch must describe the same
// change, or the recorded decision is about work that no longer exists.
//
// Measured 2026-09-15 across all eleven reopened ailang-core-triage approvals:
// every card named `design_docs/planned/ailang-core-backlog.md` while every
// branch carried a per-report file instead. One was approved, and the merge
// landed a file its card never mentioned.
func TestDecidePR_CardAndBranchMustAgree(t *testing.T) {
	completed := &TaskRecord{ID: "task-c0ca6301", Status: TaskStatusCompleted}
	triage := &AgentConfig{ID: "ailang-core-triage",
		ArtifactPatterns: []string{"design_docs/planned/ailang-core-triage/*.md"}}
	branch := []string{"design_docs/planned/ailang-core-triage/coordinator-completion-wrong-ref.md"}

	// The measured case: the card names a file the branch does not carry.
	d := DecidePR(completed, triage, branch, []string{"design_docs/planned/ailang-core-backlog.md"})
	if d.Verdict != PRRefuse {
		t.Fatalf("verdict %s, want refuse: the operator approved a card describing a different file", d.Verdict)
	}
	if !strings.Contains(d.Reason, "ailang-core-backlog.md") || !strings.Contains(d.Reason, "coordinator-completion-wrong-ref.md") {
		t.Errorf("the refusal must name BOTH sides so the operator can see the disagreement: %q", d.Reason)
	}

	// Agreement merges, in either order — GitHub and the approval record do not
	// list files in the same order, and treating that as disagreement would
	// refuse every merge, which is how a check gets switched off.
	two := []string{"design_docs/planned/ailang-core-triage/a.md", "design_docs/planned/ailang-core-triage/b.md"}
	rev := []string{two[1], two[0]}
	if d := DecidePR(completed, triage, two, rev); d.Verdict != PRMerge {
		t.Errorf("verdict %s, want merge: the same set in a different order is the same change (%s)", d.Verdict, d.Reason)
	}

	// An extra file on the branch is the more dangerous direction: it lands
	// something nobody approved.
	extra := append(append([]string{}, branch...), "design_docs/planned/ailang-core-triage/sneaked-in.md")
	if d := DecidePR(completed, triage, extra, branch); d.Verdict != PRRefuse {
		t.Errorf("verdict %s, want refuse: the branch carries a file the card never showed", d.Verdict)
	}

	// No recorded card: nothing to compare, so the merge proceeds — but the
	// reason must SAY the check did not run.
	d = DecidePR(completed, triage, branch, nil)
	if d.Verdict != PRMerge {
		t.Fatalf("verdict %s, want merge when there is no card to check", d.Verdict)
	}
	if !strings.Contains(d.Reason, "could not be checked") {
		t.Errorf("an unchecked merge must say so: %q", d.Reason)
	}
}

func TestCardFilesFromContext(t *testing.T) {
	if got := CardFilesFromContext(`{"changed_files":["a.md","b.md"]}`); len(got) != 2 {
		t.Errorf("got %v, want two files", got)
	}
	// No diff recorded is "no evidence", never "no files" — the difference
	// decides whether a merge is refused or allowed.
	for _, in := range []string{"", "{}", "not json", `{"diff_unavailable":"no diff source"}`} {
		if got := CardFilesFromContext(in); got != nil {
			t.Errorf("CardFilesFromContext(%q) = %v, want nil", in, got)
		}
	}
}
