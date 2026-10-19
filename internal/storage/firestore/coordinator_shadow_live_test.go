package firestore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// RecordShadowRun merges both sides in a transaction and reads back as one
// comparison; an unknown parent is ErrShadowComparisonNotFound.
//
//	AILANG_FIRESTORE_LIVE_TEST_PROJECT=ailang-multivac-dev go test ./internal/storage/firestore/ -run ShadowComparison -count=1
func TestShadowComparisonRoundTrip_Live(t *testing.T) {
	project := os.Getenv("AILANG_FIRESTORE_LIVE_TEST_PROJECT") //nolint:forbidigo // opt-in live test knob, never read by the product
	if project == "" {
		t.Skip("set AILANG_FIRESTORE_LIVE_TEST_PROJECT to run against a real Firestore project")
	}
	ctx := context.Background()
	client, err := NewClientForProject(ctx, project)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	s := NewCoordinatorStore(client)

	parent := fmt.Sprintf("task-liveshadow-%d", time.Now().UnixNano())
	t.Cleanup(func() { _, _ = client.Doc(collShadowComparisons, parent).Delete(context.Background()) })

	if _, err := s.GetShadowComparison(ctx, parent); !errors.Is(err, coordinator.ErrShadowComparisonNotFound) {
		t.Fatalf("unknown parent: err = %v, want ErrShadowComparisonNotFound", err)
	}
	if _, err := s.RecordShadowRun(ctx, parent, coordinator.ShadowSideShadow, coordinator.ShadowRunResult{TaskID: parent + "-shadow", BaseCommit: "b2", CostUSD: 0.2}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RecordShadowRun(ctx, parent, coordinator.ShadowSidePrimary, coordinator.ShadowRunResult{TaskID: parent, BaseCommit: "b1"}); err != nil {
		t.Fatal(err)
	}
	c, err := s.GetShadowComparison(ctx, parent)
	if err != nil {
		t.Fatal(err)
	}
	if !c.Complete() || !c.BaseDrift || c.Shadow.CostUSD != 0.2 || c.Primary.TaskID != parent {
		t.Fatalf("round trip: %+v", c)
	}

	// The grade sweep's query and claim (M5).
	listed := func(state string) bool {
		list, err := s.ListShadowComparisonsByGradeState(ctx, state, 50)
		if err != nil {
			t.Fatal(err)
		}
		for _, x := range list {
			if x.ParentID == parent {
				return true
			}
		}
		return false
	}
	if !listed(coordinator.ShadowGradeNone) {
		t.Fatal("complete, ungraded comparison not listed for grading")
	}
	claim := func(cur *coordinator.ShadowComparison) bool {
		if cur.GradeState != coordinator.ShadowGradeNone {
			return false
		}
		cur.GradeState = coordinator.ShadowGradeDispatched
		return true
	}
	if won, err := s.UpdateShadowComparison(ctx, parent, claim); err != nil || !won {
		t.Fatalf("first claim: won=%v err=%v", won, err)
	}
	if won, err := s.UpdateShadowComparison(ctx, parent, claim); err != nil || won {
		t.Fatalf("second claim: won=%v err=%v, want lost", won, err)
	}
	if listed(coordinator.ShadowGradeNone) || !listed(coordinator.ShadowGradeDispatched) {
		t.Fatal("claimed comparison listed in the wrong grade state")
	}
}
