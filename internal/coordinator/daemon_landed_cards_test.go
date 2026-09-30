package coordinator

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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
	prevLookup, prevSince := settledPRLookup, landedCardSweepSince
	settledPRLookup = func(_ context.Context, _, repo, branch string) (*LandedPR, error) {
		asked = append(asked, repo+" "+branch)
		return merged, nil
	}
	landedCardSweepSince = time.Now().Add(-time.Hour)
	t.Cleanup(func() { settledPRLookup, landedCardSweepSince = prevLookup, prevSince })

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
	settledPRLookup = func(_ context.Context, _, repo, b string) (*LandedPR, error) {
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

func TestSettledFromREST(t *testing.T) {
	at := "2026-09-23T16:00:00Z"
	branch := "coordinator/task-aaaa1111"
	mk := func(n int, ref, state string, merged *string) restPR {
		p := restPR{Number: n, State: state, MergedAt: merged}
		p.Head.Ref = ref
		return p
	}
	for _, tc := range []struct {
		name      string
		prs       []restPR
		wantState string
		wantNum   int
	}{
		{"none", nil, "", 0},
		{"open only", []restPR{mk(1, branch, "open", nil)}, "", 0},
		{"closed unmerged", []restPR{mk(1, branch, "closed", nil)}, "CLOSED", 1},
		{"merged", []restPR{mk(3, branch, "closed", &at)}, "MERGED", 3},
		{"merged beats closed", []restPR{mk(1, branch, "closed", nil), mk(3, branch, "closed", &at)}, "MERGED", 3},
		{"merged beats open, either order", []restPR{mk(4, branch, "open", nil), mk(3, branch, "closed", &at)}, "MERGED", 3},
		// Closed and then replaced from the same branch: still undecided.
		{"reopened as a new PR", []restPR{mk(1, branch, "closed", nil), mk(2, branch, "open", nil)}, "", 0},
		{"newest closed wins", []restPR{mk(1, branch, "closed", nil), mk(5, branch, "closed", nil)}, "CLOSED", 5},
		{"another branch's merge", []restPR{mk(2, branch+"2", "closed", &at)}, "", 0},
		{"another branch's close", []restPR{mk(2, branch+"2", "closed", nil)}, "", 0},
	} {
		got := settledFromREST(tc.prs, branch)
		switch {
		case tc.wantState == "" && got != nil:
			t.Errorf("%s: got %+v, want undecided", tc.name, got)
		case tc.wantState != "" && (got == nil || got.State != tc.wantState || got.Number != tc.wantNum || got.HeadRefName != branch):
			t.Errorf("%s: got %+v, want %s #%d", tc.name, got, tc.wantState, tc.wantNum)
		}
	}
}

// Closing the PR is the rejection: the card resolves as rejected, the agent is
// NOT asked to try again, and no handoff fires.
func TestLandedCardSweep_ClosedPRRejectsWithoutRetriggerOrHandoff(t *testing.T) {
	f, d, _ := landedFixture(t, nil)
	branch := BranchForTask(f.task.ID)
	settledPRLookup = func(context.Context, string, string, string) (*LandedPR, error) {
		return &LandedPR{Number: 1344, State: "CLOSED", HeadRefName: branch}, nil
	}
	before, _ := f.store.ListTasks(context.Background(), &TaskFilter{})

	d.runLandedCardSweep(context.Background(), "tok")

	task, _ := f.store.GetTask(context.Background(), f.task.ID)
	if task.Status != TaskStatusRejected {
		t.Fatalf("task = %s, want rejected", task.Status)
	}
	apr, _ := f.store.GetApprovalRequestByTaskAnyStatus(context.Background(), f.task.ID)
	if apr.Status != "rejected" || apr.ResolvedBy != "pr-closed #1344" {
		t.Fatalf("approval = %s by %q", apr.Status, apr.ResolvedBy)
	}
	after, _ := f.store.ListTasks(context.Background(), &TaskFilter{})
	if len(after) != len(before) {
		t.Fatalf("tasks %d -> %d: a closed PR must not re-trigger the agent", len(before), len(after))
	}
	if n := len(sprintPlannerInbox(t, f)); n != 0 {
		t.Fatalf("a closed PR fired %d handoff(s)", n)
	}
}

// A shutdown mid-sweep stops it with ONE line, not a warning per card. The
// context is canceled DURING the first lookup, as SIGTERM does, so the second
// card must not be looked up and the failed first lookup must not warn.
func TestLandedCardSweep_CanceledContextStopsQuietly(t *testing.T) {
	f, d, _ := landedFixture(t, nil)
	second := *f.task
	second.ID = "task-bbbb0002"
	if err := f.store.CreateTask(context.Background(), &second); err != nil {
		t.Fatal(err)
	}
	var logged strings.Builder
	d.logger = log.New(&logged, "", 0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lookups := 0
	settledPRLookup = func(ctx context.Context, _, _, _ string) (*LandedPR, error) {
		lookups++
		cancel() // SIGTERM arrives mid-request
		return nil, ctx.Err()
	}

	d.runLandedCardSweep(ctx, "tok")

	if lookups != 1 {
		t.Fatalf("%d lookups; the sweep must stop at the first sign of shutdown", lookups)
	}
	if strings.Contains(logged.String(), "cannot read PRs") {
		t.Fatalf("an interrupted lookup warned as a failure:\n%s", logged.String())
	}
	if !strings.Contains(logged.String(), "landed-card sweep interrupted") || strings.Count(logged.String(), "\n") != 1 {
		t.Fatalf("want exactly one interruption line, got:\n%s", logged.String())
	}
}

// The request itself: state=all, or an open PR is invisible and a closed-then-
// replaced branch would reject a live card.
func TestLookupSettledPRREST_AsksForEveryState(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		_, _ = w.Write([]byte(`[{"number":1,"state":"closed","merged_at":null,"head":{"ref":"coordinator/task-x"}},` +
			`{"number":2,"state":"open","merged_at":null,"head":{"ref":"coordinator/task-x"}}]`))
	}))
	defer srv.Close()
	prev := githubAPIBase
	githubAPIBase = srv.URL
	t.Cleanup(func() { githubAPIBase = prev })

	pr, err := lookupSettledPRREST(context.Background(), "tok", "sunholo-data/ailang", "coordinator/task-x")
	if err != nil {
		t.Fatal(err)
	}
	if gotQuery.Get("state") != "all" || gotQuery.Get("head") != "sunholo-data:coordinator/task-x" {
		t.Fatalf("query = %v, want state=all and the owner-qualified head", gotQuery)
	}
	if pr != nil {
		t.Fatalf("got %+v; a branch with an open PR is undecided", pr)
	}
}

func TestDecideClosedCard(t *testing.T) {
	task := &TaskRecord{ID: "task-cccc2222", Status: TaskStatusPendingApproval}
	pending := &ApprovalRequestRecord{Status: "pending"}
	closed := &LandedPR{Number: 9, State: "CLOSED", HeadRefName: BranchForTask(task.ID)}
	if ok, _ := DecideClosedCard(task, pending, closed); !ok {
		t.Fatal("a pending card with a closed PR on its own branch must be rejected")
	}
	for name, tc := range map[string]struct {
		task *TaskRecord
		apr  *ApprovalRequestRecord
		pr   *LandedPR
	}{
		"task already decided":     {&TaskRecord{ID: task.ID, Status: TaskStatusCompleted}, pending, closed},
		"approval already decided": {task, &ApprovalRequestRecord{Status: "approved"}, closed},
		"no approval record":       {task, nil, closed},
		"PR merged, not closed":    {task, pending, &LandedPR{Number: 9, State: "MERGED", HeadRefName: closed.HeadRefName}},
		"another task's branch":    {task, pending, &LandedPR{Number: 9, State: "CLOSED", HeadRefName: BranchForTask("task-dddd3333")}},
	} {
		if ok, why := DecideClosedCard(tc.task, tc.apr, tc.pr); ok {
			t.Errorf("%s: rejected, want left alone (%s)", name, why)
		}
	}
}
