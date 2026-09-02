package coordinator

import (
	"fmt"
	"log"
	"strings"
	"testing"
)

type fakeReadMarker struct {
	marked []string
	err    error
}

func (f *fakeReadMarker) MarkInboxMessageRead(id string) error {
	if f.err != nil {
		return f.err
	}
	f.marked = append(f.marked, id)
	return nil
}

// The headline arm: a message that became a task is marked read, so health's
// "routable but never dispatched" count can actually reach zero.
func TestDispatchedMessageIsMarkedRead(t *testing.T) {
	store := &fakeReadMarker{}
	if !markDispatchedMessageRead(store, "inbox_123_abc", nil) {
		t.Fatal("expected the message to be marked read")
	}
	if len(store.marked) != 1 || store.marked[0] != "inbox_123_abc" {
		t.Fatalf("marked = %v, want [inbox_123_abc]", store.marked)
	}
}

// Best-effort: the task is already dispatched, so a bookkeeping failure must not
// look like success — but it must be LOUD, because the only other symptom is
// health silently never reaching zero.
func TestReadMarkingFailureIsLoudAndNotFatal(t *testing.T) {
	var buf strings.Builder
	store := &fakeReadMarker{err: fmt.Errorf("firestore unavailable")}

	if markDispatchedMessageRead(store, "inbox_123_abc", log.New(&buf, "", 0)) {
		t.Fatal("a failed write must not report success")
	}
	out := buf.String()
	if !strings.Contains(out, "inbox_123_abc") {
		t.Errorf("the warning must name the message, got %q", out)
	}
	if !strings.Contains(out, "messages health") {
		t.Errorf("the warning must name the symptom it causes, got %q", out)
	}
}

func TestNilStoreAndEmptyIDAreBenign(t *testing.T) {
	if markDispatchedMessageRead(nil, "x", nil) {
		t.Error("nil store must be benign")
	}
	if markDispatchedMessageRead(&fakeReadMarker{}, "", nil) {
		t.Error("empty id must be benign")
	}
}
