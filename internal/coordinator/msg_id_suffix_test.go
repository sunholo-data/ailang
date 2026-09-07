package coordinator

import "testing"

// TestMsgIDSuffix_DeterministicHandoffIDsDoNotCollide is the regression: every
// handoff to the same target once derived the SAME task id, because the tail of
// "task-<parent>:handoff:<target>" is the target's name, not a random suffix.
func TestMsgIDSuffix_DeterministicHandoffIDsDoNotCollide(t *testing.T) {
	a := msgIDSuffix("task-c871949f:handoff:sprint-evaluator", 8)
	b := msgIDSuffix("task-deadbeef:handoff:sprint-evaluator", 8)

	if a == "valuator" || b == "valuator" {
		t.Errorf("derived the literal tail %q — every evaluator handoff would share a task id", a)
	}
	if a == b {
		t.Fatalf("two different handoffs derived the same task id %q; on Firestore the second "+
			"silently overwrites the first", a)
	}
}

// TestMsgIDSuffix_InboxFormUnchanged: the random-tail form must keep its existing
// suffix, or an in-flight message changes identity between redeliveries and
// dispatches twice.
func TestMsgIDSuffix_InboxFormUnchanged(t *testing.T) {
	cases := map[string]string{
		"inbox_1788806422072_c871949f": "c871949f",
		"inbox_1788805635509_f8ecc37c": "f8ecc37c",
	}
	for id, want := range cases {
		if got := msgIDSuffix(id, 8); got != want {
			t.Errorf("msgIDSuffix(%q) = %q, want %q — existing task ids must not move", id, got, want)
		}
	}
}

// TestMsgIDSuffix_Idempotent: the same message must always yield the same task,
// which is what makes redelivery safe.
func TestMsgIDSuffix_Idempotent(t *testing.T) {
	for _, id := range []string{
		"task-c871949f:handoff:sprint-evaluator",
		"inbox_1788806422072_c871949f",
		"short",
	} {
		first := msgIDSuffix(id, 8)
		second := msgIDSuffix(id, 8)
		if first != second {
			t.Errorf("msgIDSuffix(%q) is not deterministic", id)
		}
	}
}
