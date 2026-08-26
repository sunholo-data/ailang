package coordinator

import (
	"context"
	"path/filepath"
	"strings"
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

// A verdict for a task with NO approval at all is an error, not a no-op: it
// must say so (fail loud). (A RESOLVED approval is different — the verdict
// late-attaches for audit; see TestApprovalEvaluation_LateAttach.)
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

// The race measured live 2026-08-26: the human approved while the evaluator
// was scoring, and PASS score=95 was dropped. A late verdict now attaches to
// the resolved approval, marked as late.
func TestApprovalEvaluation_LateAttachAfterResolution(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()

	req := &ApprovalRequestRecord{ID: "appr-2", TaskID: "task-raced", Type: "merge", Description: "x", Status: "pending"}
	if err := store.CreateApprovalRequest(ctx, req); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := store.ResolveApprovalRequest(ctx, "appr-2", "approved", "dashboard-user"); err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if err := store.UpdateApprovalEvaluationByTask(ctx, "task-raced", "PASS score=95"); err != nil {
		t.Fatalf("late attach must succeed on a resolved approval, got: %v", err)
	}
	got, err := store.GetApprovalRequest(ctx, "appr-2")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Evaluation == "" || !strings.Contains(got.Evaluation, "late") {
		t.Errorf("late verdict must be recorded and marked late, got %q", got.Evaluation)
	}
}
