package daemon

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/pubsub"
)

// newSplitDaemon builds a daemon whose per-device subscription reaches the
// local notifier and whose shared subscription reaches the remote one, the
// way `ailang daemon run` wires it when Discord is registered.
func newSplitDaemon(t *testing.T) (*fakeSubscriber, *recordingNotifier, *recordingNotifier, *recordingNotifier) {
	t.Helper()
	sub := newFakeSubscriber()
	fetch := &fakeFetcher{msgs: map[string]*messaging.InboxMessage{
		"inbox_1": {MessageID: "inbox_1", ToInbox: "user", FromAgent: "stapledons_godot", Title: "a report"},
	}}
	full, local, remote := &recordingNotifier{}, &recordingNotifier{}, &recordingNotifier{}
	d := New(Config{
		EventsSub:   "events-shared",
		MessagesSub: "messages-rig",
		TaskWindow:  60 * time.Second,
		MsgWindow:   5 * time.Minute,
	}, sub, fetch, full.notify)
	d.SplitRemote(MessageSource{Sub: sub, Fetcher: fetch, SubName: RemoteMessagesSub, Label: "remote"},
		local.notify, remote.notify)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = d.Run(ctx) }()
	time.Sleep(30 * time.Millisecond) // let every source register its handler
	return sub, full, local, remote
}

// Each copy of a message reaches only its own channels: the per-device copy
// shows the banner, the shared copy posts to Discord, and neither is lost to
// the other's dedup. Before the split, every daemon posted its own copy to
// Discord, so two running daemons meant two Discord posts.
func TestSplitRemote_EachCopyReachesOnlyItsOwnChannels(t *testing.T) {
	sub, full, local, remote := newSplitDaemon(t)
	data := msgNotificationJSON(t, pubsub.MessageNotification{MessageID: "inbox_1"})

	if err := sub.deliver(t, "messages-rig", data, nil); err != nil {
		t.Fatalf("per-device deliver: %v", err)
	}
	if err := sub.deliver(t, RemoteMessagesSub, data, nil); err != nil {
		t.Fatalf("shared deliver: %v", err)
	}

	if got := len(local.calls()); got != 1 {
		t.Errorf("local notifications = %d, want 1 (from the per-device subscription)", got)
	}
	if got := len(remote.calls()); got != 1 {
		t.Errorf("remote notifications = %d, want 1 (from the shared subscription)", got)
	}
	if got := len(full.calls()); got != 0 {
		t.Errorf("inbox messages reached the full fan-out %d times; it would post to Discord from every device", got)
	}
}

// A Discord failure retries the shared copy only; the banner's copy acks.
func TestSplitRemote_RemoteFailureNacksOnlyTheSharedCopy(t *testing.T) {
	sub, _, local, remote := newSplitDaemon(t)
	remote.fail = errors.New("discord send: unexpected status 500")
	data := msgNotificationJSON(t, pubsub.MessageNotification{MessageID: "inbox_1"})

	if err := sub.deliver(t, "messages-rig", data, nil); err != nil {
		t.Errorf("per-device copy nacked (%v); a Discord failure must not re-fire the banner", err)
	}
	if err := sub.deliver(t, RemoteMessagesSub, data, nil); err == nil {
		t.Errorf("shared copy acked although Discord failed; the post would be lost")
	}
	remote.fail = nil
	if err := sub.deliver(t, RemoteMessagesSub, data, nil); err != nil {
		t.Fatalf("redelivery after recovery: %v", err)
	}
	if got := len(remote.calls()); got != 1 {
		t.Errorf("remote notifications after redelivery = %d, want 1", got)
	}
	if got := len(local.calls()); got != 1 {
		t.Errorf("local notifications = %d, want 1", got)
	}
}
