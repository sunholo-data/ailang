package main

import "testing"

func TestOneShotDeadline(t *testing.T) {
	for _, sub := range []string{"list", "ls", "read", "ack", "unack", "send", "reply", "search", "inboxes"} {
		if oneShotDeadline(sub) != messagesOneShotDeadline {
			t.Errorf("%q must be bounded: it is what hung inside the mission sandbox", sub)
		}
	}
	// Long-running by design, or bulk work that scales with the store.
	for _, sub := range []string{"watch", "cleanup", "dedupe", "import-github", "github-sync", "health", ""} {
		if d := oneShotDeadline(sub); d != 0 {
			t.Errorf("%q must not be bounded, got %s", sub, d)
		}
	}
	if messagesOneShotDeadline >= 600e9 {
		t.Errorf("deadline %s must stay under the pi runner's 600s stall bound", messagesOneShotDeadline)
	}
}
