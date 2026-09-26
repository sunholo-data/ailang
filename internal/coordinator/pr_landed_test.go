package coordinator

import (
	"context"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/messaging"
)

func TestDecideLandedCard(t *testing.T) {
	pending := &TaskRecord{ID: "task-aaaa1111", Status: TaskStatusPendingApproval}
	apr := &ApprovalRequestRecord{Status: "pending"}
	merged := &LandedPR{Number: 7, State: "MERGED", HeadRefName: "coordinator/task-aaaa1111", MergedBy: "someone"}

	cases := []struct {
		name string
		task *TaskRecord
		apr  *ApprovalRequestRecord
		pr   *LandedPR
		want bool
	}{
		{"merged PR on a pending card", pending, apr, merged, true},
		{"task no longer pending", &TaskRecord{ID: pending.ID, Status: TaskStatusCompleted}, apr, merged, false},
		{"approval already resolved", pending, &ApprovalRequestRecord{Status: "rejected"}, merged, false},
		{"no approval record", pending, nil, merged, false},
		{"PR closed, not merged", pending, apr, &LandedPR{Number: 7, State: "CLOSED", HeadRefName: merged.HeadRefName}, false},
		// The only link between task and PR is the head name; anything looser
		// lets one task's merge resolve another's card.
		{"another task's branch", pending, apr, &LandedPR{Number: 7, State: "MERGED", HeadRefName: "coordinator/task-aaaa11112"}, false},
		{"no PR", pending, apr, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, reason := DecideLandedCard(tc.task, tc.apr, tc.pr)
			if got != tc.want {
				t.Fatalf("DecideLandedCard = %v (%s), want %v", got, reason, tc.want)
			}
			if reason == "" {
				t.Fatal("a verdict must carry its reason")
			}
		})
	}
}

func TestLandedApprover_NamesTheMergeNotAnOperator(t *testing.T) {
	got := LandedApprover(&LandedPR{Number: 182, MergedBy: "MarkEdmondson1234"})
	if got != "pr-merge #182 (MarkEdmondson1234)" {
		t.Fatalf("LandedApprover = %q", got)
	}
	if got := LandedApprover(&LandedPR{Number: 1}); !strings.Contains(got, "unknown") {
		t.Fatalf("missing merger must read as unknown, got %q", got)
	}
}

// Resolving a landed card records the decision and completes the task, and —
// with SkipHandoffs — dispatches NOTHING. The stage-crossing fixture is the one
// that proves an ordinary approval DOES put a handoff in sprint-planner's inbox,
// so an empty inbox here is the flag working, not a fixture that never fires.
func TestSkipHandoffs_ResolvesWithoutDispatch(t *testing.T) {
	f := newStageCrossingFixture(t)
	ctx := context.Background()

	res, err := ProcessApprovalRequest(ctx, &ApprovalParams{
		TaskID: f.task.ID, Action: "approve", Channel: "pr-merged",
		ApprovedBy: LandedApprover(&LandedPR{Number: 9, MergedBy: "m"}),
		Store:      f.store, MsgStore: f.msgStore, AgentRegistry: f.registry,
		SkipMerge: true, SkipHandoffs: true,
	})
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if len(res.HandoffTargets) != 0 {
		t.Fatalf("SkipHandoffs still reported dispatch to %v", res.HandoffTargets)
	}
	if !strings.Contains(res.Message, "handoffs NOT fired") {
		t.Errorf("result must say the handoff was withheld, got %q", res.Message)
	}
	msgs, err := f.msgStore.ListInboxMessages(messaging.InboxListOptions{Inbox: "sprint-planner"})
	if err != nil {
		t.Fatalf("list inbox: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("SkipHandoffs dispatched %d message(s) to sprint-planner", len(msgs))
	}

	task, err := f.store.GetTask(ctx, f.task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if task.Status != TaskStatusCompleted {
		t.Errorf("task status = %s, want completed — a landed card must leave the queue", task.Status)
	}
	apr, err := f.store.GetApprovalRequestByTaskAnyStatus(ctx, f.task.ID)
	if err != nil || apr == nil {
		t.Fatalf("read approval: %v", err)
	}
	if apr.Status != "approved" || !strings.HasPrefix(apr.ResolvedBy, "pr-merge #9") {
		t.Errorf("approval = %s by %q, want approved by the merge", apr.Status, apr.ResolvedBy)
	}
}
