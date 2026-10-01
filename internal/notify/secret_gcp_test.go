package notify

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"testing"
)

func stubSecretAccessor(t *testing.T, fn func(ctx context.Context, project, name string) (string, error)) {
	t.Helper()
	prev := secretAccessor
	secretAccessor = fn
	t.Cleanup(func() { secretAccessor = prev })
}

func stubKeychain(t *testing.T, v string) {
	t.Helper()
	prev := keychainLookup
	keychainLookup = func() string { return v }
	t.Cleanup(func() { keychainLookup = prev })
}

func TestDiscordWebhookSecretName(t *testing.T) {
	for _, tc := range []struct{ prefix, want string }{
		{"ailang", "ailang-discord-webhook-url"},
		{"ailang-dev", "ailang-dev-discord-webhook-url"},
		{"ailang-test", "ailang-test-discord-webhook-url"},
		// A trailing dash must not double up.
		{"ailang-", "ailang-discord-webhook-url"},
		// No resolved environment: the unprefixed name, not a guessed "ailang".
		{"", "discord-webhook-url"},
		{"  ", "discord-webhook-url"},
	} {
		if got := DiscordWebhookSecretName(tc.prefix); got != tc.want {
			t.Errorf("DiscordWebhookSecretName(%q) = %q, want %q", tc.prefix, got, tc.want)
		}
	}
}

// Secret Manager must come BEFORE the Keychain. The Keychain locks whenever
// another user holds the console, so a machine that prefers it is one console
// switch away from losing the channel — which is exactly what happened on the
// rig while its Keychain item sat present and valid.
func TestDiscordWebhookURL_SecretManagerBeatsTheKeychain(t *testing.T) {
	t.Setenv(string(DiscordWebhookEnv), "")
	stubSecretAccessor(t, func(context.Context, string, string) (string, error) {
		return "https://discord.com/api/webhooks/from-secret-manager", nil
	})
	stubKeychain(t, "https://discord.com/api/webhooks/from-keychain")

	got, src := discordWebhookURL("ailang-multivac", "ailang", nil)
	if got != "https://discord.com/api/webhooks/from-secret-manager" {
		t.Errorf("webhook = %q, want the Secret Manager value", got)
	}
	if src != SourceSecretManager {
		t.Errorf("source = %q, want %q", src, SourceSecretManager)
	}
}

// The env var stays the explicit operator override and wins over both stores.
func TestDiscordWebhookURL_EnvWinsOverEverything(t *testing.T) {
	t.Setenv(string(DiscordWebhookEnv), "https://discord.com/api/webhooks/from-env")
	stubSecretAccessor(t, func(context.Context, string, string) (string, error) {
		return "https://discord.com/api/webhooks/from-secret-manager", nil
	})
	stubKeychain(t, "https://discord.com/api/webhooks/from-keychain")

	got, src := discordWebhookURL("ailang-multivac", "ailang", nil)
	if got != "https://discord.com/api/webhooks/from-env" {
		t.Errorf("webhook = %q, want the env value", got)
	}
	if src != SourceEnv {
		t.Errorf("source = %q, want %q", src, SourceEnv)
	}
}

// A FAILING Secret Manager read must fall through to the Keychain but say so.
// Silence here is what made the original outage read as "not configured".
func TestDiscordWebhookURL_SecretManagerFailureIsLoudThenFallsBack(t *testing.T) {
	t.Setenv(string(DiscordWebhookEnv), "")
	stubSecretAccessor(t, func(context.Context, string, string) (string, error) {
		return "", errors.New("PermissionDenied: caller lacks secretmanager.versions.access")
	})
	stubKeychain(t, "https://discord.com/api/webhooks/from-keychain")

	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)
	got, src := discordWebhookURL("ailang-multivac", "ailang", logger)

	if got != "https://discord.com/api/webhooks/from-keychain" {
		t.Errorf("webhook = %q, want the Keychain fallback", got)
	}
	if src != SourceKeychain {
		t.Errorf("source = %q, want %q", src, SourceKeychain)
	}
	out := buf.String()
	if !strings.Contains(out, "Secret Manager lookup") {
		t.Errorf("the failure was not logged at all:\n%s", out)
	}
	for _, want := range []string{"PermissionDenied", "ailang-discord-webhook-url", "ailang-multivac"} {
		if !strings.Contains(out, want) {
			t.Errorf("log does not name %q, so the next outage is undiagnosable:\n%s", want, out)
		}
	}
}

// No project configured is "not configured", not a failure: it must not log an
// error and must not be attempted.
func TestDiscordWebhookURL_NoProjectSkipsSecretManagerQuietly(t *testing.T) {
	t.Setenv(string(DiscordWebhookEnv), "")
	called := false
	stubSecretAccessor(t, func(context.Context, string, string) (string, error) {
		called = true
		return "", errors.New("should never be called")
	})
	stubKeychain(t, "https://discord.com/api/webhooks/from-keychain")

	var buf bytes.Buffer
	got, src := discordWebhookURL("", "", log.New(&buf, "", 0))
	if called {
		t.Error("Secret Manager was queried with no project configured")
	}
	if got != "https://discord.com/api/webhooks/from-keychain" || src != SourceKeychain {
		t.Errorf("got (%q, %q), want the Keychain", got, src)
	}
	if strings.Contains(buf.String(), "failed") {
		t.Errorf("an unconfigured project must not log a failure:\n%s", buf.String())
	}
}

// Registration names the source it used. A bare "registered" hides a machine
// still depending on the Keychain.
func TestRegisterChannelsFor_NamesTheSecretSource(t *testing.T) {
	t.Setenv(string(DiscordWebhookEnv), "")
	stubSecretAccessor(t, func(context.Context, string, string) (string, error) {
		return "https://discord.com/api/webhooks/x", nil
	})
	stubKeychain(t, "")

	var buf bytes.Buffer
	reg := NewRegistry()
	got := RegisterChannelsFor(reg, log.New(&buf, "", 0), "ailang-multivac", "ailang")

	if len(got) != 1 || got[0] != "discord" {
		t.Fatalf("registered = %v, want [discord]", got)
	}
	if !strings.Contains(buf.String(), string(SourceSecretManager)) {
		t.Errorf("the log does not name the source:\n%s", buf.String())
	}
}

// With nothing configured, the fail-closed message must name every place that
// was tried — the original said only "unset and not in keychain", which did not
// mention the store the fleet actually uses.
func TestRegisterChannelsFor_UnconfiguredNamesEverySourceTried(t *testing.T) {
	t.Setenv(string(DiscordWebhookEnv), "")
	stubSecretAccessor(t, func(context.Context, string, string) (string, error) { return "", nil })
	stubKeychain(t, "")

	var buf bytes.Buffer
	reg := NewRegistry()
	if got := RegisterChannelsFor(reg, log.New(&buf, "", 0), "ailang-multivac", "ailang"); len(got) != 0 {
		t.Fatalf("registered = %v, want none", got)
	}
	out := buf.String()
	for _, want := range []string{string(DiscordWebhookEnv), "ailang-discord-webhook-url", "ailang-multivac", "Keychain"} {
		if !strings.Contains(out, want) {
			t.Errorf("fail-closed message does not name %q:\n%s", want, out)
		}
	}
}
