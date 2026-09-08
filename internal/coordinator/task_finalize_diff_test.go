package coordinator

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pubsub"
)

// M-COMPLETION-PATH-PARITY M3 — the approval card's evidence.
//
// #921: a card that rendered a confident "Files (0)" was approved blind, twice.
// So absence must read as absence, and presence must be replay-stable. With D2
// ruled per-edge, an auto-approved handoff can be released by that same card —
// which is why M3 lands before any new edge opens.

func TestCloudDiffSource_RequiresBothCommits(t *testing.T) {
	cases := []struct {
		name string
		base string
		head string
		want bool // true = a diff is available
	}{
		{"both present", "abc123", "def456", true},
		{"no head", "abc123", "", false},
		{"no base", "", "def456", false},
		{"neither", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &CloudStrategy{Completion: pubsub.TaskCompletion{
				BaseCommit:   tc.base,
				HeadCommit:   tc.head,
				DiffStat:     " file.go | 3 +++",
				ChangedFiles: []string{"file.go"},
				Diff:         "--- a/file.go",
			}}
			got, err := s.DiffSource(context.Background(), &TaskRecord{ID: "task-x"})
			if tc.want {
				if err != nil {
					t.Fatalf("expected a diff, got %v", err)
				}
				if got.Stat == "" || len(got.ChangedFiles) == 0 || got.Patch == "" {
					t.Errorf("diff is incomplete: %+v", got)
				}
				return
			}
			if !errors.Is(err, ErrNoDiffSource) {
				t.Errorf("a branch-bounded diff is not replay-stable, so a missing SHA must report ErrNoDiffSource; got %v", err)
			}
		})
	}
}

// TestFinalize_ApprovalCarriesTheExecutorDiff is the end-to-end shape: the
// executor computes the evidence, the payload carries it, and the approval card
// shows real files instead of a confident zero.
func TestFinalize_ApprovalCarriesTheExecutorDiff(t *testing.T) {
	h := newFinalizeHarness(t, handoffAgent(false))

	strategy := &CloudStrategy{Completion: pubsub.TaskCompletion{
		BaseCommit:   "abc123",
		HeadCommit:   "def456",
		DiffStat:     " design_docs/planned/m-pkg.md | 157 ++++++++",
		ChangedFiles: []string{"design_docs/planned/m-pkg.md"},
		Diff:         "--- /dev/null\n+++ b/design_docs/planned/m-pkg.md",
	}}
	if _, err := FinalizeTaskCompletion(context.Background(), h.deps, FinalizeInput{
		Task:    h.task,
		Result:  &ExecuteResult{Success: true, SessionID: "s"},
		Outcome: OutcomeCompleted,
	}, strategy); err != nil {
		t.Fatalf("finalize: %v", err)
	}

	got := h.approval(t)
	if got == nil {
		t.Fatal("no approval created")
	}
	if strings.Contains(got.ContextJSON, "diff_unavailable") {
		t.Errorf("the approval reported no diff although the executor supplied one: %s", got.ContextJSON)
	}
	for _, want := range []string{"design_docs/planned/m-pkg.md", "diff_stat", "changed_files"} {
		if !strings.Contains(got.ContextJSON, want) {
			t.Errorf("approval context is missing %q: %s", want, got.ContextJSON)
		}
	}
}

// TestFinalize_ApprovalIsIdenticalOnReplay: the same completion delivered twice
// must produce the same card. Two immutable SHAs are what guarantee it.
func TestFinalize_ApprovalIsIdenticalOnReplay(t *testing.T) {
	h := newFinalizeHarness(t, handoffAgent(false))
	strategy := &CloudStrategy{Completion: pubsub.TaskCompletion{
		BaseCommit: "abc123", HeadCommit: "def456",
		DiffStat: " a.go | 1 +", ChangedFiles: []string{"a.go"}, Diff: "patch",
	}}
	in := FinalizeInput{Task: h.task, Result: &ExecuteResult{Success: true}, Outcome: OutcomeCompleted}

	if _, err := FinalizeTaskCompletion(context.Background(), h.deps, in, strategy); err != nil {
		t.Fatalf("first: %v", err)
	}
	first := h.approval(t).ContextJSON

	if _, err := FinalizeTaskCompletion(context.Background(), h.deps, in, strategy); err != nil {
		t.Fatalf("replay: %v", err)
	}
	second := h.approval(t).ContextJSON

	if first != second {
		t.Errorf("the approval card changed between deliveries:\n first=%s\nsecond=%s", first, second)
	}
}
