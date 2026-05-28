package notify

import (
	"log"
	"os"
)

// DiscordWebhookEnv holds the Discord incoming-webhook URL. Fail-closed: when it
// is neither set nor in the Keychain, the Discord channel is simply not
// registered (the daemon still boots — it just has no Discord output). Treat the
// value as a secret.
const DiscordWebhookEnv = "AILANG_DISCORD_WEBHOOK_URL"

// keychainLookup resolves the Discord webhook from the OS keychain. It is a var
// so tests can stub it and stay hermetic (independent of the dev's real Keychain).
var keychainLookup = discordWebhookFromKeychain

// discordWebhookURL resolves the webhook URL from the environment first
// (AILANG_DISCORD_WEBHOOK_URL), then the macOS login Keychain (darwin only).
// Returns "" when unconfigured.
func discordWebhookURL() string {
	if v := os.Getenv(DiscordWebhookEnv); v != "" {
		return v
	}
	return keychainLookup()
}

// RegisterChannels registers every env-gated outbound channel into reg and
// returns the names registered. It is the fail-closed entry point (Aitana's
// non-negotiable rule): a channel whose secret is absent is not registered, so
// a fresh host with no secrets configured boots with zero channels rather than
// crashing. The macOS desktop channel is host-conditional and is registered by
// the daemon wiring, not here.
func RegisterChannels(reg *Registry, logger *log.Logger) []string {
	var registered []string

	if url := discordWebhookURL(); url != "" {
		if err := reg.Register(NewDiscordChannel(url)); err != nil {
			logf(logger, "notify: discord registration failed: %v", err)
		} else {
			registered = append(registered, "discord")
		}
	} else {
		logf(logger, "notify: discord channel not registered (%s unset and not in keychain)", DiscordWebhookEnv)
	}

	return registered
}

func logf(logger *log.Logger, format string, args ...interface{}) {
	if logger != nil {
		logger.Printf(format, args...)
	}
}
