package notify

import (
	"log"
	"os"
)

// DiscordWebhookEnv holds the Discord incoming-webhook URL. Fail-closed: when it
// is unset, the Discord channel is simply not registered (the daemon still boots
// — it just has no Discord output). Treat the value as a secret.
const DiscordWebhookEnv = "AILANG_DISCORD_WEBHOOK_URL"

// RegisterChannels registers every env-gated outbound channel into reg and
// returns the names registered. It is the fail-closed entry point (Aitana's
// non-negotiable rule): a channel whose secret is absent is not registered, so
// a fresh host with no secrets configured boots with zero channels rather than
// crashing. The macOS desktop channel is host-conditional and is registered by
// the daemon wiring, not here.
func RegisterChannels(reg *Registry, logger *log.Logger) []string {
	var registered []string

	if url := os.Getenv(DiscordWebhookEnv); url != "" {
		if err := reg.Register(NewDiscordChannel(url)); err != nil {
			logf(logger, "notify: discord registration failed: %v", err)
		} else {
			registered = append(registered, "discord")
		}
	} else {
		logf(logger, "notify: discord channel not registered (%s unset)", DiscordWebhookEnv)
	}

	return registered
}

func logf(logger *log.Logger, format string, args ...interface{}) {
	if logger != nil {
		logger.Printf(format, args...)
	}
}
