package notify

import (
	"context"
	"os"
	"testing"
)

// TestDiscordLive is a manual reachability probe: it performs a REAL send to the
// webhook in AILANG_DISCORD_WEBHOOK_URL through the production code path
// (render → chunk → POST). It is skipped unless the env var is set, so it never
// runs in CI. Run it manually after configuring a webhook:
//
//	AILANG_DISCORD_WEBHOOK_URL='https://discord.com/api/webhooks/…' \
//	  go test ./internal/notify/ -run TestDiscordLive -count=1 -v
func TestDiscordLive(t *testing.T) {
	url := os.Getenv(DiscordWebhookEnv)
	if url == "" {
		t.Skipf("set %s to run the live Discord smoke test", DiscordWebhookEnv)
	}
	ch := NewDiscordChannel(url)
	err := ch.Send(context.Background(), Notification{
		Title:    "AILANG notify — live test",
		Subtitle: "internal/notify Discord channel",
		Body:     "If this reached your phone, the Discord webhook channel works ✅",
		URL:      "https://github.com/sunholo-data/ailang",
	})
	if err != nil {
		t.Fatalf("live Discord send failed: %v", err)
	}
	t.Log("live Discord send OK — check your Discord mobile app")
}
