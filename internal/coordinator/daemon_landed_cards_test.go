package coordinator

import (
	"context"
	"io"
	"log"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
)

// landedFixture is the stage-crossing pipeline (design -> gated -> sprint-planner)
// with the designer pointed at a GitHub repo and GitHub stubbed out.
func landedFixture(t *testing.T, merged *LandedPR) (*stageCrossingFixture, *Daemon, *[]string) {
	t.Helper()
	f := newStageCrossingFixture(t)
	f.registry.GetAgentByID("design-doc-creator").Repo = "sunholo-data/ailang"

	var asked []string
	prevLookup, prevSince := mergedPRLookup, landedCardSweepSince
	mergedPRLookup = func(_ context.Context, _, repo, branch string) (*LandedPR, error) {
		asked = append(asked, repo+" "+branch)
		return merged, nil
	}
	landedCardSweepSince = time.Now().Add(-time.Hour)
	t.Cleanup(func() { mergedPRLookup, landedCardSweepSince = prevLookup, prevSince })

	d := &Daemon{
		taskStore: f.store, msgStore: f.msgStore, agentRegistry: f.registry,
		logger: log.New(io.Discard, "", 0), ctx: context.Background(),
	}
	return f, d, &asked
}

func sprintPlannerInbox(t *testing.T, f *stageCrossingFixture) []messaging.InboxMessage {
	t.Helper()
	msgs, err := f.msgStore.ListInboxMessages(messaging.InboxListOptions{Inbox: "sprint-planner"})
	if err != nil {
		t.Fatalf("list inbox: %v", err)
	}
	return msgs
}

// The ruling: a merge IS the approval, so the card resolves AND the chain moves.
func TestLandedCardSweep_MergeResolvesAndFiresHandoff(t *testing.T) {
	f, d, asked := landedFixture(t, nil)
	branch := BranchForTask(f.task.ID)
	mergedPRLookup = func(_ context.Context, _, repo, b string) (*LandedPR, error) {
		*asked = append(*asked, repo+" "+b)
		return &LandedPR{Number: 1301, State: "MERGED", HeadRefName: branch, MergedAt: "2026-09-23T16:00:00Z"}, nil
	}

	d.runLandedCardSweep(context.Background(), "tok")

	if len(*asked) != 1 || (*asked)[0] != "sunholo-data/ailang "+branch {
		t.Fatalf("looked up %v, want the task's own branch in the agent's repo", *asked)
	}
	task, _ := f.store.GetTask(context.Background(), f.task.ID)
	if task.Status != TaskStatusCompleted {
		t.Fatalf("task = %s, want completed", task.Status)
	}
	apr, _ := f.store.GetApprovalRequestByTaskAnyStatus(context.Background(), f.task.ID)
	if apr.Status != "approved" || apr.ResolvedBy != "pr-merge #1301 (unknown)" {
		t.Fatalf("approval = %s by %q", apr.Status, apr.ResolvedBy)
	}
	if n := len(sprintPlannerInbox(t, f)); n != 1 {
		t.Fatalf("sprint-planner got %d handoff(s), want 1 — the merge must continue the chain", n)
	}
}

func TestLandedCardSweep_NoMergedPRLeavesTheCard(t *testing.T) {
	f, d, asked := landedFixture(t, nil)
	d.runLandedCardSweep(context.Background(), "tok")
	if len(*asked) != 1 {
		t.Fatalf("expected one lookup, got %v", *asked)
	}
	task, _ := f.store.GetTask(context.Background(), f.task.ID)
	if task.Status != TaskStatusPendingApproval {
		t.Fatalf("task = %s, want still pending", task.Status)
	}
	if n := len(sprintPlannerInbox(t, f)); n != 0 {
		t.Fatalf("dispatched %d with no merge", n)
	}
}

// The backlog is excluded: its handoffs are superseded and must never fire.
func TestLandedCardSweep_BacklogBeforeCutoffIsNeverTouched(t *testing.T) {
	f, d, asked := landedFixture(t, &LandedPR{Number: 1, State: "MERGED"})
	landedCardSweepSince = time.Now().Add(time.Hour) // the fixture's card predates it
	d.runLandedCardSweep(context.Background(), "tok")
	if len(*asked) != 0 {
		t.Fatalf("a pre-cutoff card reached GitHub: %v", *asked)
	}
	if n := len(sprintPlannerInbox(t, f)); n != 0 {
		t.Fatalf("a pre-cutoff card fired %d handoff(s)", n)
	}
}

func TestMergedFromREST(t *testing.T) {
	at := "2026-09-23T16:00:00Z"
	branch := "coordinator/task-aaaa1111"
	mk := func(n int, ref string, merged *string) restPR {
		p := restPR{Number: n, MergedAt: merged}
		p.Head.Ref = ref
		return p
	}
	if got := mergedFromREST([]restPR{mk(1, branch, nil)}, branch); got != nil {
		t.Fatalf("closed-unmerged PR read as landed: %+v", got)
	}
	if got := mergedFromREST([]restPR{mk(2, branch+"2", &at)}, branch); got != nil {
		t.Fatalf("another branch's merge read as landed: %+v", got)
	}
	got := mergedFromREST([]restPR{mk(1, branch, nil), mk(3, branch, &at)}, branch)
	if got == nil || got.Number != 3 || got.State != "MERGED" || got.HeadRefName != branch {
		t.Fatalf("mergedFromREST = %+v, want #3 merged", got)
	}
}
