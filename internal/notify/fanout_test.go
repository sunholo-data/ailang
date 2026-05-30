package notify

import (
	"context"
	"errors"
	"testing"
)

// cfgChannel is a stub with configurable name, locality, send error, and
// optional event-type allow-list (empty = accept everything).
type cfgChannel struct {
	name    string
	local   bool
	err     error
	accepts []string // empty = accept all event types
	calls   int
}

func (c *cfgChannel) Name() string  { return c.name }
func (c *cfgChannel) IsLocal() bool { return c.local }
func (c *cfgChannel) Accepts(eventType string) bool {
	if len(c.accepts) == 0 {
		return true
	}
	for _, t := range c.accepts {
		if t == eventType {
			return true
		}
	}
	return false
}
func (c *cfgChannel) Send(_ context.Context, _ Notification) error {
	c.calls++
	return c.err
}

func sendAll(t *testing.T, chans ...Channel) error {
	t.Helper()
	reg := NewRegistry()
	for _, ch := range chans {
		if err := reg.Register(ch); err != nil {
			t.Fatalf("register %s: %v", ch.Name(), err)
		}
	}
	return reg.SendAll(context.Background(), Notification{Title: "x"}, nil)
}

func TestSendAll_RemoteOK_Acks(t *testing.T) {
	macos := &cfgChannel{name: "macos", local: true}
	discord := &cfgChannel{name: "discord"}
	if err := sendAll(t, macos, discord); err != nil {
		t.Fatalf("expected ack (nil) when remote succeeds, got %v", err)
	}
	if discord.calls != 1 || macos.calls != 1 {
		t.Fatalf("both channels should fire once; macos=%d discord=%d", macos.calls, discord.calls)
	}
}

func TestSendAll_RemoteFails_Nacks_EvenIfLocalOK(t *testing.T) {
	macos := &cfgChannel{name: "macos", local: true}                // succeeds
	discord := &cfgChannel{name: "discord", err: errors.New("429")} // fails
	err := sendAll(t, macos, discord)
	if err == nil {
		t.Fatal("expected error (nack) when an authoritative remote channel fails, even though local succeeded")
	}
}

func TestSendAll_LocalFails_DoesNotNack_WhenRemoteOK(t *testing.T) {
	macos := &cfgChannel{name: "macos", local: true, err: errors.New("no GUI")} // local fails
	discord := &cfgChannel{name: "discord"}                                     // remote ok
	if err := sendAll(t, macos, discord); err != nil {
		t.Fatalf("local failure must not gate the ack when remote succeeded, got %v", err)
	}
}

func TestSendAll_OnlyLocal_FallsBackToBestEffort(t *testing.T) {
	// No remote channels: ack if the local one delivered.
	okMac := &cfgChannel{name: "macos", local: true}
	if err := sendAll(t, okMac); err != nil {
		t.Fatalf("only-local success should ack, got %v", err)
	}
	// Only local, and it failed: nack.
	badMac := &cfgChannel{name: "macos", local: true, err: errors.New("no GUI")}
	if err := sendAll(t, badMac); err == nil {
		t.Fatal("only-local failure should nack")
	}
}

func TestSendAll_NoChannels_Acks(t *testing.T) {
	reg := NewRegistry()
	if err := reg.SendAll(context.Background(), Notification{}, nil); err != nil {
		t.Fatalf("empty registry should ack (nil), got %v", err)
	}
}

func TestSendAll_AllRemotesMustSucceed(t *testing.T) {
	d1 := &cfgChannel{name: "discord"}
	d2 := &cfgChannel{name: "slack", err: errors.New("boom")}
	if err := sendAll(t, d1, d2); err == nil {
		t.Fatal("ack requires ALL remote channels to succeed; one failed")
	}
}

// sendOne registers chans and fans out a single notification with the given
// EventType, returning the SendAll error (or nil).
func sendOne(t *testing.T, eventType string, chans ...Channel) error {
	t.Helper()
	reg := NewRegistry()
	for _, ch := range chans {
		if err := reg.Register(ch); err != nil {
			t.Fatalf("register %s: %v", ch.Name(), err)
		}
	}
	return reg.SendAll(context.Background(), Notification{EventType: eventType}, nil)
}

func TestSendAll_FilteredRemoteSkippedFallsBackToLocal(t *testing.T) {
	// Discord only accepts pending_approval. With a `completed` event, Discord
	// is skipped (not a failure), there are no qualifying remote channels, so
	// the ack falls back to local best-effort (macOS succeeded).
	macos := &cfgChannel{name: "macos", local: true}
	discord := &cfgChannel{name: "discord", accepts: []string{"pending_approval"}}
	if err := sendOne(t, "completed", macos, discord); err != nil {
		t.Fatalf("filtered-out remote should not gate the ack, got %v", err)
	}
	if discord.calls != 0 {
		t.Errorf("discord must NOT be called for filtered-out event, got %d sends", discord.calls)
	}
	if macos.calls != 1 {
		t.Errorf("macOS should still receive the event, got %d sends", macos.calls)
	}
}

func TestSendAll_FilterAcceptingRemoteFires(t *testing.T) {
	// Same channels, but event type matches Discord's filter — Discord fires.
	macos := &cfgChannel{name: "macos", local: true}
	discord := &cfgChannel{name: "discord", accepts: []string{"pending_approval"}}
	if err := sendOne(t, "pending_approval", macos, discord); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if discord.calls != 1 {
		t.Errorf("discord should be called for accepted event, got %d", discord.calls)
	}
	if macos.calls != 1 {
		t.Errorf("macOS should be called too, got %d", macos.calls)
	}
}

func TestSendAll_AllRemotesFilteredOutNotAFailure(t *testing.T) {
	// Two remote channels both filter out this event type. There are NO
	// qualifying remote channels left, so SendAll falls back to local
	// best-effort and returns nil (ack), not error.
	macos := &cfgChannel{name: "macos", local: true}
	discord := &cfgChannel{name: "discord", accepts: []string{"pending_approval"}}
	slack := &cfgChannel{name: "slack", accepts: []string{"pending_approval"}}
	if err := sendOne(t, "completed", macos, discord, slack); err != nil {
		t.Fatalf("event with no qualifying remote channels should ack via local, got %v", err)
	}
}

func TestSendAll_FilterDoesNotMaskRemoteFailure(t *testing.T) {
	// Discord accepts the event but its send fails — that IS an authoritative
	// failure (nack). Filter is orthogonal to error path.
	discord := &cfgChannel{name: "discord", accepts: []string{"failed"}, err: errors.New("429")}
	if err := sendOne(t, "failed", discord); err == nil {
		t.Fatal("accepting-but-failing remote should still nack")
	}
}
