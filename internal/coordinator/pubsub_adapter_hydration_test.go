package coordinator

import (
	"errors"
	"testing"

	"github.com/sunholo-data/ailang/internal/messaging"
)

// A notification the adapter cannot read is NOT a task.
//
// The Pub/Sub notification carries only a message id by design; title and body
// live in Firestore. Hydration used to be best-effort, so a failed fetch fell
// through to a placeholder — and the coordinator dispatched a task whose entire
// content was the string "msg_20260914_153340_b787f96a", titled "Pub/Sub
// notification from coordinator".
//
// Measured in production 2026-09-14, 13:33–13:39: sixteen such tasks for
// sprint-planner in four minutes. Four died in the executor; the rest reported
// "completed" and each fired a handoff into sprint-executor — an agent that
// writes code — on the strength of a task containing no request at all.

type failingGetStore struct {
	messaging.MessageStore
	err error
	msg *messaging.InboxMessage
}

func (f *failingGetStore) GetInboxMessage(string) (*messaging.InboxMessage, error) {
	return f.msg, f.err
}

func notifyAttrs() map[string]string {
	return map[string]string{"inbox": "sprint-planner", "from_agent": "coordinator"}
}

// A fetch ERROR is transient — an eventually-consistent read, a blip. Return it
// so Pub/Sub redelivers; the message is durable and nothing is lost. What must
// NOT happen is a dispatch.
func TestPubSubAdapter_FetchError_NacksAndBuffersNothing(t *testing.T) {
	a := NewPubSubInboxAdapter(nil, "sub", "sprint-planner",
		&failingGetStore{err: errors.New("deadline exceeded")}, newSilentLogger())

	err := a.HandleNotification(validNotification("msg_20260914_153340_b787f96a"), notifyAttrs())
	if err == nil {
		t.Error("a transient fetch failure must NACK so Pub/Sub redelivers, not ack silently")
	}
	if n := len(a.buffered); n != 0 {
		t.Errorf("buffered %d message(s) — a notification that could not be read is not work", n)
	}
}

// A fetch returning NIL means the notification names a message that does not
// exist. Retrying cannot fix that, so ack — but still never dispatch.
func TestPubSubAdapter_MissingMessage_AcksAndBuffersNothing(t *testing.T) {
	a := NewPubSubInboxAdapter(nil, "sub", "sprint-planner",
		&failingGetStore{msg: nil}, newSilentLogger())

	if err := a.HandleNotification(validNotification("msg-gone"), notifyAttrs()); err != nil {
		t.Errorf("an absent message must be acked, not retried forever: %v", err)
	}
	if n := len(a.buffered); n != 0 {
		t.Errorf("buffered %d message(s) for a message that does not exist", n)
	}
}

// Belt and braces: hydration succeeded but produced nothing to act on. An agent
// handed this would have to invent the request.
func TestPubSubAdapter_HydratedButEmpty_BuffersNothing(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload string
	}{
		{"empty", ""},
		{"whitespace", "   \n\t"},
		{"just its own id", "msg-echo"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := NewPubSubInboxAdapter(nil, "sub", "sprint-planner",
				&failingGetStore{msg: &messaging.InboxMessage{
					MessageID: "msg-echo", ToInbox: "sprint-planner", Payload: tc.payload,
				}}, newSilentLogger())

			if err := a.HandleNotification(validNotification("msg-echo"), notifyAttrs()); err != nil {
				t.Errorf("unusable content is not an error, just not work: %v", err)
			}
			if n := len(a.buffered); n != 0 {
				t.Errorf("buffered %d task(s) with no request in them", n)
			}
		})
	}
}

// The control: a real message still dispatches, carrying the REAL title and
// body rather than the placeholder. Without this the fix would read as
// "dispatch nothing", which is not the intent.
func TestPubSubAdapter_RealMessageStillDispatches(t *testing.T) {
	a := NewPubSubInboxAdapter(nil, "sub", "sprint-planner",
		&failingGetStore{msg: &messaging.InboxMessage{
			MessageID: "msg-real", ToInbox: "sprint-planner", FromAgent: "coordinator",
			Title: "Handoff: Design: secondary-model fallback", Payload: "Plan the sprint for the attached design doc.",
		}}, newSilentLogger())

	if err := a.HandleNotification(validNotification("msg-real"), notifyAttrs()); err != nil {
		t.Fatalf("a readable message must dispatch: %v", err)
	}
	if len(a.buffered) != 1 {
		t.Fatalf("buffered %d, want 1", len(a.buffered))
	}
	got := a.buffered[0]
	if got.Content != "Plan the sprint for the attached design doc." {
		t.Errorf("content is not the hydrated body: %q", got.Content)
	}
	if got.Title != "Handoff: Design: secondary-model fallback" {
		t.Errorf("title is still the placeholder: %q", got.Title)
	}
}

// No store at all is a misconfiguration, not a reason to invent work.
func TestPubSubAdapter_NoStore_Refuses(t *testing.T) {
	a := NewPubSubInboxAdapter(nil, "sub", "sprint-planner", nil, newSilentLogger())
	if err := a.HandleNotification(validNotification("m1"), notifyAttrs()); err == nil {
		t.Error("an adapter with no store must refuse loudly, not dispatch a content-free task")
	}
	if n := len(a.buffered); n != 0 {
		t.Errorf("buffered %d message(s) with no store to hydrate from", n)
	}
}
