package firestore

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// TestResolveApprovalIsCompareAndSet_Live races N resolvers — half suppressing,
// half not — at ONE pending approval in a real Firestore project and asserts
// exactly one wins (M-TASK-STATUS-TRUTH V19).
//
// The resolve used to be query-then-update: every resolver that read "pending"
// before the first write landed also succeeded, so a normal approve and an
// operator's suppressing approve could both "win" and disagree about whether
// handoffs fire. There is no Firestore emulator in this repo, so this runs only
// against a real project, and only when asked:
//
//	AILANG_FIRESTORE_LIVE_TEST_PROJECT=ailang-multivac-dev go test ./internal/storage/firestore/ -run CompareAndSet_Live -count=1
//
// It writes one approval document with a unique id and deletes it afterwards.
func TestResolveApprovalIsCompareAndSet_Live(t *testing.T) {
	project := os.Getenv("AILANG_FIRESTORE_LIVE_TEST_PROJECT") //nolint:forbidigo // opt-in live test knob, never read by the product
	if project == "" {
		t.Skip("set AILANG_FIRESTORE_LIVE_TEST_PROJECT to run against a real Firestore project")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := NewClientForProject(ctx, project)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	defer client.Close()
	store := NewCoordinatorStore(client)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	taskID := "task-cas-live-" + suffix
	aprID := coordinator.ApprovalIDForTask(taskID)
	if err := store.CreateApprovalRequest(ctx, &coordinator.ApprovalRequestRecord{
		ID: aprID, TaskID: taskID, Type: "merge_handoff", Status: "pending", CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	defer func() { _, _ = client.Collection(collApprovals).Doc(aprID).Delete(context.Background()) }()

	const n = 8
	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			if i%2 == 0 {
				errs[i] = store.ResolveApprovalSuppressingHandoffs(ctx, taskID, fmt.Sprintf("racer-%d", i))
			} else {
				errs[i] = store.ResolveApprovalRequestByTask(ctx, taskID, "approved", fmt.Sprintf("racer-%d", i))
			}
		}(i)
	}
	close(start)
	wg.Wait()

	winners := 0
	winner := -1
	for i, e := range errs {
		if e == nil {
			winners++
			winner = i
		}
	}
	if winners != 1 {
		t.Fatalf("%d of %d concurrent resolvers succeeded, want exactly 1 (errors: %v)", winners, n, errs)
	}

	snap, err := client.Collection(collApprovals).Doc(aprID).Get(ctx)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	data := snap.Data()
	suppressed, _ := data["handoffs_suppressed"].(bool)
	if wantSuppressed := winner%2 == 0; suppressed != wantSuppressed {
		t.Errorf("winner racer-%d (suppressing=%v) but document reads handoffs_suppressed=%v — the loser's write leaked",
			winner, wantSuppressed, suppressed)
	}
	if got, _ := data["resolved_by"].(string); got != fmt.Sprintf("racer-%d", winner) {
		t.Errorf("resolved_by = %q, want racer-%d — a losing resolver overwrote the winner", got, winner)
	}
	t.Logf("exactly one of %d concurrent resolvers won: racer-%d (suppressing=%v)", n, winner, winner%2 == 0)
}
