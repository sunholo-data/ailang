//go:build !darwin

package notify

// discordWebhookFromKeychain is a no-op off macOS — only the daemon's macOS host
// uses the login Keychain. Other hosts configure the webhook via the env var.
func discordWebhookFromKeychain() string { return "" }
