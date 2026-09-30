package daemon

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/pubsub"
)

// errFetcher returns a fixed (message, error) pair for every id, so a test can
// drive each absence/failure shape through the real handler.
type errFetcher struct {
	msg *messaging.InboxMessage
	err error
}

func (f *errFetcher) Fetch(_ context.Context, _ string) (*messaging.InboxMessage, error) {
	return f.msg, f.err
}

// deliver pushes one notification through the daemon's real message handler and
// returns what the Pub/Sub layer would do with it: a non-nil error is a nack.
func deliver(t *testing.T, d *Daemon, fetcher MessageFetcher, id string) error {
	t.Helper()
	h := d.messageHandlerFor(MessageSource{Fetcher: fetcher, SubName: "messages-laptop", Label: "prod"})
	return h(context.Background(), msgNotificationJSON(t, pubsub.MessageNotification{MessageID: id}), nil)
}

// A notification naming a message that is not in the store is worth retrying
// only while it could still be replication lag. Past the grace window it is
// acked, because nacking it forever puts it straight back at the head of the
// backlog where it blocks every real message behind it.
//
// Regression for the laptop daemon, 2026-09-30: 360 such notifications produced
// 160,947 nacks against 10 acks in six hours and the Mac stopped alerting.
func TestDaemon_AbsentMessageIsDroppedAfterGraceInsteadOfRetriedForever(t *testing.T) {
	for _, tc := range []struct {
		name    string
		fetcher MessageFetcher
	}{
		// SQLite's GetInboxMessage says "absent" with (nil, nil)...
		{"sqlite shape: nil message, nil error", &errFetcher{msg: nil, err: nil}},
		// ...Firestore's by wrapping the shared sentinel.
		{"firestore shape: wrapped ErrMessageNotFound", &errFetcher{
			msg: nil,
			err: fmt.Errorf("%w: task-08032ebc:handoff:sprint-planner", messaging.ErrMessageNotFound),
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, _, _, rec := newTestDaemon(t)
			clock := time.Now()
			d.msgAbsent.now = func() time.Time { return clock }

			// First sighting: could still be replication lag, so retry.
			if err := deliver(t, d, tc.fetcher, "m-absent"); err == nil {
				t.Fatal("first delivery of an absent message must nack (retry), got ack")
			}
			// Still inside the window: still a retry.
			clock = clock.Add(absentGrace - time.Second)
			if err := deliver(t, d, tc.fetcher, "m-absent"); err == nil {
				t.Fatal("inside the grace window an absent message must still nack, got ack")
			}
			// Past the window it will never resolve: ack so it leaves the backlog.
			clock = clock.Add(2 * time.Second)
			if err := deliver(t, d, tc.fetcher, "m-absent"); err != nil {
				t.Fatalf("past the grace window an absent message must be acked, got nack: %v", err)
			}
			if got := rec.calls(); len(got) != 0 {
				t.Fatalf("a dropped message must not fire a notification, got %d", len(got))
			}
		})
	}
}

// A fetch that failed for a retryable reason is NOT an absence, so it keeps
// being retried however long it lasts. Collapsing the two is what would turn a
// Firestore outage into silently discarded notifications.
func TestDaemon_TransientFetchFailureIsRetriedPastTheGraceWindow(t *testing.T) {
	d, _, _, _ := newTestDaemon(t)
	clock := time.Now()
	d.msgAbsent.now = func() time.Time { return clock }
	fetcher := &errFetcher{err: errors.New("rpc error: code = Unavailable desc = transport closing")}

	for _, at := range []time.Duration{0, absentGrace, 10 * absentGrace} {
		clock = clock.Add(at)
		if err := deliver(t, d, fetcher, "m-flaky"); err == nil {
			t.Fatalf("a retryable fetch failure must nack at +%s, got ack", at)
		}
	}
}

// A message that shows up after a genuine replication lag must fire normally and
// leave no tracker entry behind.
func TestDaemon_AbsentThenVisibleMessageFiresAndClearsTheTracker(t *testing.T) {
	d, _, _, rec := newTestDaemon(t)
	absent := &errFetcher{}
	if err := deliver(t, d, absent, "m-late"); err == nil {
		t.Fatal("want nack while absent")
	}
	if d.msgAbsent.size() != 1 {
		t.Fatalf("want the absence tracked, size=%d", d.msgAbsent.size())
	}

	visible := &errFetcher{msg: &messaging.InboxMessage{
		ID: "m-late", FromAgent: "cli", ToInbox: "daneel", Title: "late but real",
	}}
	if err := deliver(t, d, visible, "m-late"); err != nil {
		t.Fatalf("want ack once visible, got %v", err)
	}
	if got := rec.calls(); len(got) != 1 {
		t.Fatalf("want exactly one notification, got %d", len(got))
	}
	if d.msgAbsent.size() != 0 {
		t.Fatalf("a resolved message must leave no tracker entry, size=%d", d.msgAbsent.size())
	}
}

func TestUnresolved_SweepDropsStaleEntries(t *testing.T) {
	u := newUnresolved()
	clock := time.Now()
	u.now = func() time.Time { return clock }
	u.age("old")
	clock = clock.Add(11 * absentGrace)
	u.age("new")
	u.sweep()
	if u.size() != 1 {
		t.Fatalf("want only the recent entry to survive, size=%d", u.size())
	}
}

func TestIsMessageNotFound(t *testing.T) {
	if messaging.IsMessageNotFound(nil) {
		t.Error("nil is not an absence")
	}
	if messaging.IsMessageNotFound(errors.New("transport closing")) {
		t.Error("an unrelated error is not an absence")
	}
	if !messaging.IsMessageNotFound(fmt.Errorf("%w: abc", messaging.ErrMessageNotFound)) {
		t.Error("a wrapped sentinel is an absence")
	}
}
