package notify

import (
	"context"
	"errors"
	"testing"
)

// cfgChannel is a stub with configurable name, locality, and send error.
type cfgChannel struct {
	name  string
	local bool
	err   error
	calls int
}

func (c *cfgChannel) Name() string  { return c.name }
func (c *cfgChannel) IsLocal() bool { return c.local }
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
