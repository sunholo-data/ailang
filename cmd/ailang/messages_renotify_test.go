package main

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/messaging"
)

// A read message was dispatched or acked: re-notifying it runs the work twice.
func TestRenotifyGuard(t *testing.T) {
	if err := renotifyGuard(messaging.InboxStatusUnread, false); err != nil {
		t.Fatalf("unread must be allowed: %v", err)
	}
	for _, st := range []string{"read", "acked", "archived"} {
		if err := renotifyGuard(st, false); err == nil {
			t.Errorf("%s without --force must be refused", st)
		}
		if err := renotifyGuard(st, true); err != nil {
			t.Errorf("%s with --force must be allowed: %v", st, err)
		}
	}
}
