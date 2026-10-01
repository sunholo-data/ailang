package notify

import (
	"log"

	"github.com/sunholo-data/ailang/internal/config"
)

// DiscordWebhookEnv holds the Discord incoming-webhook URL. Fail-closed: when it
// is neither set nor in the Keychain, the Discord channel is simply not
// registered (the daemon still boots — it just has no Discord output). Treat the
// value as a secret.
const DiscordWebhookEnv = config.EnvDiscordWebhookURL

// keychainLookup resolves the Discord webhook from the OS keychain. It is a var
// so tests can stub it and stay hermetic (independent of the dev's real Keychain).
var keychainLookup = discordWebhookFromKeychain

// secretManagerLookup resolves the webhook from Secret Manager. A var for the
// same reason as keychainLookup.
var secretManagerLookup = discordWebhookFromSecretManager

// SecretSource names where a resolved webhook came from, so the daemon can say
// so in its startup log. "Discord is on" and "Discord is on, from the store you
// think it is" are different facts, and the second is the one that was missing
// when the rig silently fell back to no channel at all.
type SecretSource string

const (
	SourceEnv           SecretSource = "env AILANG_DISCORD_WEBHOOK_URL"
	SourceSecretManager SecretSource = "Secret Manager"
	SourceKeychain      SecretSource = "macOS login Keychain"
	SourceNone          SecretSource = ""
)

// discordWebhookURL resolves the webhook URL and reports where it came from.
//
// Order: the env var wins as an explicit operator override; then Secret Manager,
// which is the fleet's source of truth and the same value on every machine; then
// the macOS login Keychain, kept as a local fallback for a host with no cloud
// access.
//
// Secret Manager is ahead of the Keychain deliberately. The Keychain cannot be
// relied on for a daemon: it is the LOGIN keychain, so it locks whenever another
// user holds the console, which is exactly how the rig lost Discord while its
// item sat valid and present (see discordWebhookFromSecretManager).
//
// A Secret Manager read that FAILS is reported through logger and then falls
// through to the Keychain — a fallback is fine here because the channel is
// fail-closed either way, but it must never be silent, or the next outage looks
// like "not configured" again.
func discordWebhookURL(project, prefix string, logger *log.Logger) (string, SecretSource) {
	if v := config.DiscordWebhookURL(); v != "" {
		return v, SourceEnv
	}
	if v, err := secretManagerLookup(project, prefix); err != nil {
		logf(logger, "notify: Secret Manager lookup for the Discord webhook failed, trying the Keychain next: %v", err)
	} else if v != "" {
		return v, SourceSecretManager
	}
	if v := keychainLookup(); v != "" {
		return v, SourceKeychain
	}
	return "", SourceNone
}

// RegisterChannels registers every env-gated outbound channel into reg and
// returns the names registered. It is the fail-closed entry point (Aitana's
// non-negotiable rule): a channel whose secret is absent is not registered, so
// a fresh host with no secrets configured boots with zero channels rather than
// crashing. The macOS desktop channel is host-conditional and is registered by
// the daemon wiring, not here.
func RegisterChannels(reg *Registry, logger *log.Logger) []string {
	return RegisterChannelsFor(reg, logger, "", "")
}

// RegisterChannelsFor is RegisterChannels with the GCP project and resource
// prefix the Secret Manager lookup needs. RegisterChannels keeps the old
// signature for callers that have neither; they simply get no Secret Manager
// source rather than a guessed project.
func RegisterChannelsFor(reg *Registry, logger *log.Logger, project, prefix string) []string {
	var registered []string

	url, src := discordWebhookURL(project, prefix, logger)
	switch {
	case url == "":
		logf(logger, "notify: discord channel not registered (tried %s, Secret Manager secret %q in project %q, and the macOS login Keychain)",
			DiscordWebhookEnv, DiscordWebhookSecretName(prefix), project)
	default:
		if err := reg.Register(NewDiscordChannel(url)); err != nil {
			logf(logger, "notify: discord registration failed: %v", err)
		} else {
			// Name the source: a silent success hides a machine still depending
			// on the Keychain, which is the state that broke the rig.
			logf(logger, "notify: discord channel registered (webhook from %s)", src)
			registered = append(registered, "discord")
		}
	}

	return registered
}

func logf(logger *log.Logger, format string, args ...interface{}) {
	if logger != nil {
		logger.Printf(format, args...)
	}
}
