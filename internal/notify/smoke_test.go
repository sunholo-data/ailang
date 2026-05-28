package notify

import "testing"

// noKeychain stubs the Keychain lookup to empty so registration tests stay
// hermetic (independent of whatever the dev has stored in their real Keychain).
func noKeychain(t *testing.T) {
	t.Helper()
	prev := keychainLookup
	keychainLookup = func() string { return "" }
	t.Cleanup(func() { keychainLookup = prev })
}

// TestRegisterChannelsFailClosed: with no secret configured, no channel registers.
func TestRegisterChannelsFailClosed(t *testing.T) {
	noKeychain(t)
	t.Setenv(DiscordWebhookEnv, "") // present-but-empty == unset for our purposes
	reg := NewRegistry()
	if got := RegisterChannels(reg, nil); len(got) != 0 {
		t.Fatalf("expected no channels without a secret, got %v", got)
	}
	if _, err := reg.Get("discord"); err == nil {
		t.Fatal("discord must not be registered without its webhook secret")
	}
}

// TestRegisterChannelsWithDiscordSecret: secret present -> discord registers.
func TestRegisterChannelsWithDiscordSecret(t *testing.T) {
	noKeychain(t)
	t.Setenv(DiscordWebhookEnv, "https://discord.test/webhook/abc")
	reg := NewRegistry()
	got := RegisterChannels(reg, nil)
	if len(got) != 1 || got[0] != "discord" {
		t.Fatalf("expected [discord], got %v", got)
	}
	if _, err := reg.Get("discord"); err != nil {
		t.Fatalf("discord should be retrievable: %v", err)
	}
}

// TestRegisteredChannelsContract is the cross-channel smoke contract every
// channel enrolls in automatically: a registered channel reports a non-empty
// Name that matches its registry key. (Port of Aitana's test_smoke_all_channels.)
func TestRegisteredChannelsContract(t *testing.T) {
	noKeychain(t)
	t.Setenv(DiscordWebhookEnv, "https://discord.test/webhook/abc")
	reg := NewRegistry()
	RegisterChannels(reg, nil)

	names := reg.Names()
	if len(names) == 0 {
		t.Fatal("expected at least one registered channel for the contract test")
	}
	for _, name := range names {
		ch, err := reg.Get(name)
		if err != nil {
			t.Fatalf("Get(%q): %v", name, err)
		}
		if ch.Name() == "" {
			t.Errorf("channel %q has empty Name()", name)
		}
		if ch.Name() != name {
			t.Errorf("channel registered as %q reports Name()=%q", name, ch.Name())
		}
	}
}
