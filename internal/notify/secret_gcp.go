package notify

import (
	"context"
	"fmt"
	"strings"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

// DiscordWebhookSecretSuffix is appended to the environment's resource prefix to
// form the Secret Manager secret name — "ailang-discord-webhook-url" in prod,
// "ailang-dev-discord-webhook-url" in dev. It follows the naming the other
// webhook secret in this estate already uses (ailang-form-webhook-url).
const DiscordWebhookSecretSuffix = "discord-webhook-url"

// secretFetchTimeout bounds the Secret Manager read. Channel registration runs
// on the daemon's startup path, and a notifier that cannot resolve its secret
// must fail closed quickly rather than hold the process open: an unreachable
// network would otherwise delay every notification behind it.
const secretFetchTimeout = 5 * time.Second

// DiscordWebhookSecretName returns the secret name for an environment prefix.
// An empty prefix yields the unprefixed name, which is what a caller with no
// resolved environment should look for rather than guessing "ailang".
func DiscordWebhookSecretName(prefix string) string {
	prefix = strings.TrimSuffix(strings.TrimSpace(prefix), "-")
	if prefix == "" {
		return DiscordWebhookSecretSuffix
	}
	return prefix + "-" + DiscordWebhookSecretSuffix
}

// secretAccessor reads the latest version of a secret. It is a var so tests can
// substitute one and stay hermetic — the real one needs ADC and a network.
var secretAccessor = accessLatestSecretVersion

func accessLatestSecretVersion(ctx context.Context, project, name string) (string, error) {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("secret manager client: %w", err)
	}
	defer func() { _ = client.Close() }()

	resource := name
	if !strings.HasPrefix(name, "projects/") {
		resource = fmt.Sprintf("projects/%s/secrets/%s/versions/latest", project, name)
	}
	resp, err := client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{Name: resource})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(resp.GetPayload().GetData())), nil
}

// discordWebhookFromSecretManager resolves the webhook from Secret Manager.
//
// This exists because the macOS login Keychain is the wrong store for a fleet
// daemon. Measured on the rig 2026-09-30: the webhook item had been present
// since 2026-05-28 and the job was running in the Aqua domain (gui/501), yet
// every read returned errSecInteractionNotAllowed — because /dev/console was
// owned by a DIFFERENT user (daneel) while the daemon ran as voightkampff, which
// locks that user's login Keychain to background processes. Discord posts stopped
// and the only trace was one fail-closed log line. A Secret Manager secret is
// the same value on every machine and does not depend on who holds the console.
//
// Returns ("", nil) when no project is configured — that is "not configured",
// not a failure. A project WITH a failing read returns the error so the caller
// can say so out loud instead of silently falling through.
func discordWebhookFromSecretManager(project, prefix string) (string, error) {
	if strings.TrimSpace(project) == "" {
		return "", nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), secretFetchTimeout)
	defer cancel()

	name := DiscordWebhookSecretName(prefix)
	v, err := secretAccessor(ctx, project, name)
	if err != nil {
		return "", fmt.Errorf("read %s from project %s: %w", name, project, err)
	}
	return v, nil
}
