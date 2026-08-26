package coordinator

import (
	"context"
	"path/filepath"
	"testing"
)

// M1: the Evaluation field must persist and be updatable by TASK id after the
// approval is created — the evaluator only knows its parent task, not the
// approval id.
func TestApprovalEvaluation_UpdateByTask(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()

	req := &ApprovalRequestRecord{
		ID:          "appr-1",
		TaskID:      "task-parent",
		Type:        "merge",
		Description: "test approval",
		Status:      "pending",
	}
	if err := store.CreateApprovalRequest(ctx, req); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := store.UpdateApprovalEvaluationByTask(ctx, "task-parent", "PASS score=84"); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, err := store.GetApprovalRequest(ctx, "appr-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Evaluation != "PASS score=84" {
		t.Errorf("evaluation not persisted: got %q", got.Evaluation)
	}
}

// Updating a task with no pending approval is an error, not a no-op: a verdict
// that lands nowhere must say so (fail loud), or evaluator results silently
// vanish exactly the way this program's failures used to.
func TestApprovalEvaluation_UpdateWithNoApprovalIsLoud(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer func() { _ = store.Close() }()

	if err := store.UpdateApprovalEvaluationByTask(context.Background(), "task-ghost", "PASS score=99"); err == nil {
		t.Fatal("expected an error when no pending approval exists for the task")
	}
}
