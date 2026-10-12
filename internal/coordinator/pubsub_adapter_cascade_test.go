package coordinator

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/pubsub"
)

// #1730: `ailang publish` stores the upgrade-available row in the PUBLISHER's
// local store and publishes a cascade notification naming that local id. The
// cloud store never has the row, so every cascade since 2026-09-17 was dropped
// ("names a message that does not exist in the store"). The notification
// carries the full envelope, so the coordinator must rebuild the row from it,
// store it under the notification's id, and dispatch.

// cascadeNotification is the data and attributes `ailang publish` sends, built
// the way PublishCascadeWithEnvelope builds them.
func cascadeNotification(t *testing.T, id string) ([]byte, map[string]string) {
	t.Helper()
	data, err := json.Marshal(pubsub.CascadeMessageData{
		MessageID: id,
		Envelope: &pubsub.CascadeEnvelopeFields{
			RootPackage:       "sunholo/json_tools@0.4.0",
			ChangeClass:       "B",
			FromVersion:       "0.3.2",
			ToVersion:         "0.4.0",
			FromInterfaceHash: "if-old",
			ToInterfaceHash:   "if-new",
			FromContentHash:   "c-old",
			ToContentHash:     "c-new",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	attrs := pubsub.MessageAttributes{
		Inbox:       messaging.FormatPackageInbox("sunholo/report_gen"),
		FromAgent:   "coordinator",
		Category:    "interface-change",
		MessageType: "upgrade-available",
		Source:      pubsub.SourceCascade,
	}.ToMap()
	return data, attrs
}

func cascadeStore(t *testing.T) *messaging.Store {
	t.Helper()
	store, err := messaging.OpenStore(filepath.Join(t.TempDir(), "collaboration.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestPubSubAdapter_CascadeWithoutStoredRow_RebuildsStoresAndDispatches(t *testing.T) {
	store := cascadeStore(t)
	a := NewPubSubInboxAdapter(nil, "sub", "", store, newSilentLogger())
	const id = "msg_20261007_171616_782f344c" // a publisher-local id, as in production

	data, attrs := cascadeNotification(t, id)
	if err := a.HandleNotification(data, attrs); err != nil {
		t.Fatalf("HandleNotification: %v", err)
	}

	if n := len(a.buffered); n != 1 {
		t.Fatalf("buffered %d messages, want 1: the cascade must dispatch even though the publisher's row is not in this store", n)
	}
	got := a.buffered[0]
	if got.Inbox != messaging.FormatPackageInbox("sunholo/report_gen") {
		t.Errorf("dispatched to inbox %q, want the dependent's package inbox", got.Inbox)
	}
	env, err := messaging.ExtractPackageEnvelope(&messaging.InboxMessage{Payload: got.Content})
	if err != nil || env == nil {
		t.Fatalf("dispatched content is not a package envelope (%v): %q", err, got.Content)
	}
	if env.Kind != messaging.PkgMsgUpgradeAvailable || env.Package.Name != "sunholo/json_tools" ||
		env.Package.FromVersion != "0.3.2" || env.Package.ToVersion != "0.4.0" ||
		env.Package.ChangeClass != "interface-change" {
		t.Errorf("rebuilt envelope does not carry the publish: %+v", env.Package)
	}

	// The row is now in the store under the notification's id, so the readers
	// that look a task's message up by id (the autonomy router) find it.
	stored, err := store.GetInboxMessage(id)
	if err != nil || stored == nil {
		t.Fatalf("rebuilt message not stored under %s: msg=%v err=%v", id, stored, err)
	}
	if !strings.Contains(stored.Payload, "sunholo/json_tools") {
		t.Errorf("stored payload does not describe the upgrade: %q", stored.Payload)
	}

	// A redelivery finds the stored row and does not write a second one.
	if err := a.HandleNotification(data, attrs); err != nil {
		t.Fatalf("redelivery: %v", err)
	}
	all, err := store.ListInboxMessages(messaging.InboxListOptions{Inbox: messaging.FormatPackageInbox("sunholo/report_gen")})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 {
		t.Errorf("store holds %d rows for the dependent after a redelivery, want 1", len(all))
	}
}

// Only a cascade notification is rebuilt. An ordinary notification naming a
// missing message is still dropped, exactly as before.
func TestPubSubAdapter_NonCascadeMissingMessage_StillDropped(t *testing.T) {
	store := cascadeStore(t)
	a := NewPubSubInboxAdapter(nil, "sub", "", store, newSilentLogger())

	data, attrs := cascadeNotification(t, "msg-gone")
	delete(attrs, "source")
	if err := a.HandleNotification(data, attrs); err != nil {
		t.Fatalf("HandleNotification: %v", err)
	}
	if n := len(a.buffered); n != 0 {
		t.Errorf("buffered %d messages for a non-cascade notification with no stored row", n)
	}
	if stored, _ := store.GetInboxMessage("msg-gone"); stored != nil {
		t.Errorf("a non-cascade notification wrote a row: %+v", stored)
	}
}
