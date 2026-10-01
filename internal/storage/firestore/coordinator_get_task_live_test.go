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

// TestGetTaskUnknownIsErrTaskNotFound_Live proves, against real Firestore, that
// GetTask on a missing document wraps coordinator.ErrTaskNotFound, which the
// completion handler acks on instead of letting Pub/Sub redeliver.
//
//	AILANG_FIRESTORE_LIVE_TEST_PROJECT=ailang-multivac-dev go test ./internal/storage/firestore/ -run GetTaskUnknown -count=1
func TestGetTaskUnknownIsErrTaskNotFound_Live(t *testing.T) {
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

	id := fmt.Sprintf("task-livemissing-%d", time.Now().UnixNano())
	task, err := s.GetTask(ctx, id)
	if task != nil {
		t.Fatalf("task = %+v, want nil", task)
	}
	if !errors.Is(err, coordinator.ErrTaskNotFound) {
		t.Fatalf("err = %v, want coordinator.ErrTaskNotFound", err)
	}
}
