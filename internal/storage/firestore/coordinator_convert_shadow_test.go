package firestore

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// TestTaskMapShadowOfRoundTrip: the Firestore mapping is hand-written, so a
// field missing from either direction is dropped silently on read
// (M-AILANG-LANE-EXECUTOR-SHADOW M1). A shadow child read back without
// ShadowOf finalizes like a primary.
func TestTaskMapShadowOfRoundTrip(t *testing.T) {
	m := taskToMap(&coordinator.TaskRecord{ID: "task-p1-shadow", ShadowOf: "task-p1"})
	if m["shadow_of"] != "task-p1" {
		t.Fatalf("taskToMap shadow_of = %v, want task-p1", m["shadow_of"])
	}
	got, err := mapToTask(m)
	if err != nil {
		t.Fatalf("mapToTask: %v", err)
	}
	if got.ShadowOf != "task-p1" {
		t.Fatalf("mapToTask ShadowOf = %q, want task-p1", got.ShadowOf)
	}
	// An existing document has no shadow_of key: it must read as a primary.
	delete(m, "shadow_of")
	if got, err = mapToTask(m); err != nil || got.ShadowOf != "" {
		t.Fatalf("legacy document: ShadowOf = %q, err %v; want empty, nil", got.ShadowOf, err)
	}
}
