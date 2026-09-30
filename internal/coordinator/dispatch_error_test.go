package coordinator

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"testing"
)

// A permanent dispatch error fails the task; a transient one requeues it.
// Before this, both requeued: task-68771ff3 (2026-09-30) was refused by Cloud
// Run 18 times at five-minute intervals while its status said "pending".
func TestHandleDispatchError(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want TaskStatus
	}{
		{"permanent", fmt.Errorf("%w: env AILANG_DIRECTIVE is 43086 bytes", ErrDispatchPermanent), TaskStatusFailed},
		{"transient", errors.New("firestore unavailable"), TaskStatusPending},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, err := NewSQLiteStore(t.TempDir() + "/test.db")
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			ctx := context.Background()
			task := &TaskRecord{ID: "task-d1", AgentID: "a", Title: "t", Content: "c", Status: TaskStatusPending}
			if err := store.CreateTask(ctx, task); err != nil {
				t.Fatal(err)
			}
			if err := store.MarkTaskQueued(ctx, task.ID); err != nil {
				t.Fatal(err)
			}
			d := &Daemon{taskStore: store, logger: log.New(io.Discard, "", 0), ctx: ctx}

			d.handleDispatchError(task, tc.err)

			got, err := store.GetTask(ctx, task.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != tc.want {
				t.Errorf("status = %q, want %q", got.Status, tc.want)
			}
		})
	}
}
