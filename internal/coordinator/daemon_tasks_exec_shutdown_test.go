package coordinator

import (
	"context"
	"strings"
	"testing"
)

// TestExecuteTaskQueueSurvivesClosedStore pins the shutdown race.
//
// Close() releases the task store and sets the field to nil
// (daemon_lifecycle.go). A poll tick already in flight then dereferenced it and
// took the whole process down with a nil-pointer panic — observed as a test
// binary crash that masked the real assertion failure in
// TestRun_FailsLoudlyWhenTaskProcessingCannotInit for the entire run.
//
// It returns an ERROR rather than a quiet nil: "the store is gone" and "there is
// nothing to do" are different facts, and only one of them is normal.
func TestExecuteTaskQueueSurvivesClosedStore(t *testing.T) {
	d := &Daemon{
		ctx:      context.Background(),
		executor: &TaskExecutor{}, // non-nil so the local path is taken
	}
	// taskStore deliberately nil, exactly as Close() leaves it.

	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("executeTaskQueue panicked on a closed store: %v", r)
			}
		}()
		err = d.executeTaskQueue()
	}()

	if err == nil {
		t.Fatal("a missing task store must be reported, not silently treated as an empty queue")
	}
	if !strings.Contains(err.Error(), "task store") {
		t.Errorf("error should name the missing store, got: %v", err)
	}
}
