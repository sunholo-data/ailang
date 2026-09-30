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

// TestCancelAndResetAreCompareAndSet_Live proves, against real Firestore, that
// a cancel sticks: MarkTaskCancelled refuses a non-pending task, and
// ResetTaskToPending (a failed dispatch's cleanup) leaves a cancelled task
// cancelled — the unconditional write it replaced put one straight back.
//
//	AILANG_FIRESTORE_LIVE_TEST_PROJECT=ailang-multivac-dev go test ./internal/storage/firestore/ -run CancelAndReset -count=1
func TestCancelAndResetAreCompareAndSet_Live(t *testing.T) {
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

	id := fmt.Sprintf("task-livecancel-%d", time.Now().UnixNano())
	defer func() { _, _ = client.Doc(collTasks, id).Delete(ctx) }()
	if err := s.CreateTask(ctx, &coordinator.TaskRecord{ID: id, AgentID: "live-test", Title: "live cancel test", Content: id, Status: coordinator.TaskStatusPending}); err != nil {
		t.Fatal(err)
	}
	status := func() coordinator.TaskStatus {
		task, err := s.GetTask(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		return task.Status
	}

	if err := s.MarkTaskCancelled(ctx, id); err != nil {
		t.Fatalf("cancel pending: %v", err)
	}
	if st := status(); st != coordinator.TaskStatusCancelled {
		t.Fatalf("after cancel: %s", st)
	}
	if err := s.ResetTaskToPending(ctx, id); err != nil {
		t.Fatal(err)
	}
	if st := status(); st != coordinator.TaskStatusCancelled {
		t.Fatalf("a failed dispatch's reset resurrected the task: %s", st)
	}
	if err := s.MarkTaskCancelled(ctx, id); !errors.Is(err, coordinator.ErrTaskNotCancellable) {
		t.Fatalf("second cancel: err = %v, want ErrTaskNotCancellable", err)
	}
}
