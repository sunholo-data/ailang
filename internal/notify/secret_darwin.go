//go:build darwin

package notify

import (
	"os/exec"
	"strings"
)

// keychainService is the generic-password service name under which the Discord
// webhook URL is stored in the macOS login Keychain. Store it with:
//
//	security add-generic-password -U -A -a "$USER" -s ailang-discord-webhook -w '<webhook-url>'
const keychainService = "ailang-discord-webhook"

// discordWebhookFromKeychain reads the webhook URL from the macOS login Keychain
// (generic password, service=keychainService). Returns "" if the item is absent
// or on any error — the caller treats that as "not configured". The daemon runs
// as a user LaunchAgent, so the login Keychain is unlocked and accessible.
func discordWebhookFromKeychain() string {
	out, err := exec.Command("security", "find-generic-password", "-s", keychainService, "-w").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
