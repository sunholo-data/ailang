package config

// Notification channels (internal/notify and the coordinator's ntfy bridge).
const (
	EnvDiscordWebhookURL = "AILANG_DISCORD_WEBHOOK_URL"
	EnvNtfyServerURL     = "AILANG_NTFY_SERVER_URL"
	EnvNtfyTopic         = "AILANG_NTFY_TOPIC"
	EnvNtfyAuthToken     = "AILANG_NTFY_AUTH_TOKEN"
)

var notifyVars = []Var{
	{EnvDiscordWebhookURL, "", AreaNotify, "Discord webhook the notify daemon posts to; unset falls back to the macOS login keychain, then to no Discord channel."},
	{EnvNtfyServerURL, "", AreaNotify, "ntfy server the coordinator pushes secret-approval requests to; unset (or no topic) skips the push."},
	{EnvNtfyTopic, "", AreaNotify, "ntfy topic for secret-approval pushes."},
	{EnvNtfyAuthToken, "", AreaNotify, "Bearer token for the ntfy server, when it requires one."},
}

// DiscordWebhookURL returns AILANG_DISCORD_WEBHOOK_URL, "" when unset.
func DiscordWebhookURL() string { return get(EnvDiscordWebhookURL) }

// Ntfy is the ntfy push configuration.
type Ntfy struct {
	ServerURL, Topic, AuthToken string
}

// NtfyConfig returns the three AILANG_NTFY_* values verbatim.
func NtfyConfig() Ntfy {
	return Ntfy{ServerURL: get(EnvNtfyServerURL), Topic: get(EnvNtfyTopic), AuthToken: get(EnvNtfyAuthToken)}
}
