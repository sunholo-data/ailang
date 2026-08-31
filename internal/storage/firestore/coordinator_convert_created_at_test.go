package firestore

import (
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// TestTaskToMapNeverWritesNullCreatedAt pins the writer half of the prod
// incident of 2026-08-31.
//
// timeToFirestore maps a zero time.Time to nil, so a TaskRecord with an unset
// CreatedAt was persisted as created_at = null. Two consequences, both silent:
// the stale-task detector aged the task from the zero time and killed it seconds
// after dispatch, and every query that orders by created_at skipped the row
// entirely — which is why `tasks` appeared to hold nothing newer than 08-27
// while new tasks were being created every ten minutes.
//
// observatory_tasks.go has always stamped this. The coordinator writer did not,
// and the asymmetry was the defect.
//
// MU: drop the IsZero stamp in taskToMap and this fails.
func TestTaskToMapNeverWritesNullCreatedAt(t *testing.T) {
	m := taskToMap(&coordinator.TaskRecord{ID: "task-133e933b", AgentID: "docparse"})

	got, ok := m["created_at"]
	if !ok {
		t.Fatal("created_at absent from the task document")
	}
	if got == nil {
		t.Fatal("created_at written as null — the row is unorderable and reads back as " +
			"the zero time, which the stale detector treats as ~292 years old")
	}
	ts, ok := got.(time.Time)
	if !ok {
		t.Fatalf("created_at is %T, want time.Time", got)
	}
	if ts.IsZero() {
		t.Fatal("created_at stamped with the zero time")
	}
}

// TestTaskToMapPreservesRealCreatedAt: the stamp is a floor, not an overwrite.
func TestTaskToMapPreservesRealCreatedAt(t *testing.T) {
	want := time.Date(2026, 8, 31, 10, 20, 58, 0, time.UTC)
	m := taskToMap(&coordinator.TaskRecord{ID: "task-a855b349", CreatedAt: want})

	got, _ := m["created_at"].(time.Time)
	if !got.Equal(want) {
		t.Errorf("created_at = %v, want %v — a real timestamp must survive untouched", got, want)
	}
}
