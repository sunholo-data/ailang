package messaging

import "testing"

// TestNotificationIDFor_PrefersTheIDBothStoresResolve.
//
// Firestore's GetInboxMessage is client.Doc(collInbox, id) — the doc key, which
// is msg.ID, and nothing else. SQLite matches `id OR message_id`. Publishing
// MessageID therefore works on one store and fails on the other, and looks
// correct on Firestore only because normalizeInboxDefaults usually sets
// MessageID = ID.
//
// The failure it produced on 2026-09-14: notifications carrying
// "msg_20260914_153340_b787f96a" — the SQLite form, suffixed with the first
// eight characters of a UUID — reaching a coordinator that reads Firestore,
// where no such document exists.
func TestNotificationIDFor_PrefersTheIDBothStoresResolve(t *testing.T) {
	// The divergent case: a SQLite-minted business id on a Firestore document.
	msg := &InboxMessage{
		ID:        "inbox_1789394315570_56f3af3e",
		MessageID: "msg_20260914_153340_b787f96a",
	}
	if got := NotificationIDFor(msg); got != msg.ID {
		t.Errorf("NotificationIDFor = %q, want the doc key %q — Firestore cannot resolve the other one",
			got, msg.ID)
	}
}

// Pre-insert, or a caller that only set the business id: MessageID still
// resolves on SQLite, and publishing nothing resolves nowhere.
func TestNotificationIDFor_FallsBackToMessageID(t *testing.T) {
	if got := NotificationIDFor(&InboxMessage{MessageID: "msg_x"}); got != "msg_x" {
		t.Errorf("with no ID the business id is better than an empty publish, got %q", got)
	}
	if got := NotificationIDFor(nil); got != "" {
		t.Errorf("nil message = no id, got %q", got)
	}
	if got := NotificationIDFor(&InboxMessage{}); got != "" {
		t.Errorf("an empty message has no id to publish, got %q", got)
	}
}
