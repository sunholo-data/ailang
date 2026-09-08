package messaging

import (
	"context"
	"path/filepath"
	"testing"
)

// M-COMPLETION-PATH-PARITY M0b — a replayed handoff must not duplicate.
//
// Finalisation dispatches handoffs and completion notices under a deterministic
// id, and Pub/Sub push is at-least-once, so the same insert arrives twice. With
// the bare INSERT that is a UNIQUE violation — a routine redelivery becomes a
// repeating failure. On the Firestore backend the same call silently overwrites,
// which would mark a message the recipient has already read as unread again.
// First write wins on both.

func newInboxStore(t *testing.T) *Store {
	t.Helper()
	store, err := OpenStore(filepath.Join(t.TempDir(), "inbox.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func handoffFixture() *InboxMessage {
	return &InboxMessage{
		ID:           "task-88a9fa95:handoff:sprint-planner",
		FromAgent:    "coordinator",
		ToInbox:      "sprint-planner",
		MessageType:  "handoff",
		Title:        "Handoff: deterministic lockfile (approved)",
		Payload:      "Please continue this work.",
		ParentTaskID: "task-88a9fa95",
		ChainID:      "87f3840e",
	}
}

func TestPutMessageIfAbsent_ReplayDoesNotDuplicate(t *testing.T) {
	store := newInboxStore(t)
	ctx := context.Background()

	created, err := store.PutMessageIfAbsent(ctx, handoffFixture())
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	if !created {
		t.Fatal("first application reported not-created")
	}

	for i := 0; i < 2; i++ {
		created, err = store.PutMessageIfAbsent(ctx, handoffFixture())
		if err != nil {
			t.Fatalf("replay %d errored; a replay must be a no-op, not a failure: %v", i+1, err)
		}
		if created {
			t.Errorf("replay %d reported created — a second handoff message would start the next agent twice", i+1)
		}
	}

	msgs, err := store.ListInboxMessages(InboxListOptions{Inbox: "sprint-planner"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(msgs) != 1 {
		t.Errorf("sprint-planner has %d messages after 3 deliveries, want 1", len(msgs))
	}
}

// TestPutMessageIfAbsent_DoesNotResetReadStatus: the recipient may have acted on
// the message before the replay arrives. Overwriting would make handled work look
// unhandled — which on a shared inbox means a second agent picks it up.
func TestPutMessageIfAbsent_DoesNotResetReadStatus(t *testing.T) {
	store := newInboxStore(t)
	ctx := context.Background()

	if _, err := store.PutMessageIfAbsent(ctx, handoffFixture()); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := store.MarkInboxMessageRead("task-88a9fa95:handoff:sprint-planner"); err != nil {
		t.Fatalf("mark read: %v", err)
	}

	if _, err := store.PutMessageIfAbsent(ctx, handoffFixture()); err != nil {
		t.Fatalf("replay: %v", err)
	}

	msg, err := store.GetInboxMessage("task-88a9fa95:handoff:sprint-planner")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if msg.Status != InboxStatusRead {
		t.Errorf("status = %q after a replayed delivery, want %q — the replay un-read a handled message", msg.Status, InboxStatusRead)
	}
}

// TestInsertInboxMessage_StillFailsOnDuplicateID is the control: if the plain
// insert stops erroring, the arm above no longer distinguishes anything.
func TestInsertInboxMessage_StillFailsOnDuplicateID(t *testing.T) {
	store := newInboxStore(t)
	ctx := context.Background()

	if err := store.InsertInboxMessageWithContext(ctx, handoffFixture()); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := store.InsertInboxMessageWithContext(ctx, handoffFixture()); err == nil {
		t.Fatal("control failed: the plain insert no longer errors on a duplicate id, so PutMessageIfAbsent proves nothing distinctive")
	}
}

// TestPutMessageIfAbsent_RequiresAnExplicitID: without a caller-supplied id the
// store generates one, so nothing would ever collide and the replay guarantee
// would be silently void — the worst kind of failure, since every test would
// still pass.
func TestPutMessageIfAbsent_RequiresAnExplicitID(t *testing.T) {
	store := newInboxStore(t)
	ctx := context.Background()

	msg := handoffFixture()
	msg.ID = ""
	if _, err := store.PutMessageIfAbsent(ctx, msg); err == nil {
		t.Error("accepted a message with no id; a generated id can never collide, so the guarantee would be void")
	}
}

// TestPutMessageIfAbsent_SharesTheNormalInsertPath guards against the two write
// paths drifting: the field preparation (defaults, simhash, envelope) is shared
// deliberately, because a replay normalized differently would produce a document
// that looks like a different message.
func TestPutMessageIfAbsent_SharesTheNormalInsertPath(t *testing.T) {
	store := newInboxStore(t)
	ctx := context.Background()

	msg := handoffFixture()
	if _, err := store.PutMessageIfAbsent(ctx, msg); err != nil {
		t.Fatalf("put: %v", err)
	}

	got, err := store.GetInboxMessage(msg.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Simhash == nil {
		t.Error("simhash was not computed — the if-absent path skipped shared field preparation")
	}
	if got.Status != InboxStatusUnread {
		t.Errorf("status = %q, want the shared default %q", got.Status, InboxStatusUnread)
	}
	if got.ParentTaskID != "task-88a9fa95" || got.ChainID != "87f3840e" {
		t.Errorf("chain linkage lost: parent=%q chain=%q", got.ParentTaskID, got.ChainID)
	}
}
