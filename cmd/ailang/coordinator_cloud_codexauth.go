package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/executor/codex"
)

// codexCredential is what a cloud codex job installed, kept so the refreshed file
// can be written back when the task ends. nil means nothing to persist.
type codexCredential struct {
	project  string
	secret   string
	baseline []byte
}

// installCodexCredential gives a cloud codex job its credential, BEFORE the
// preflight health check.
//
// Default is the ChatGPT SUBSCRIPTION (Mark, 2026-09-30: cloud executors bill
// OAuth, never the metered API; OpenAI has confirmed this use). The metered key is
// used only by a job that explicitly declares AILANG_AUTH_MODE=apikey. Neither
// present is a hard failure — the old behaviour of quietly writing an api-key
// auth.json from OPENAI_API_KEY is how every cloud codex run billed the API.
func installCodexCredential(ctx context.Context, provider, project string) (*codexCredential, error) {
	if provider != "codex" {
		return nil, nil
	}
	if config.AuthMode() == "apikey" {
		if wrote, err := codex.EnsureAPIKeyAuth(); err != nil {
			return nil, fmt.Errorf("codex api-key auth (AILANG_AUTH_MODE=apikey): %w", err)
		} else if wrote {
			fmt.Println("execute-job: codex auth = METERED api key (AILANG_AUTH_MODE=apikey)")
		}
		return nil, nil
	}
	secret := config.CodexAuthSecret()
	if secret == "" {
		return nil, fmt.Errorf("codex job has no subscription credential: set %s to the Secret Manager secret holding a ChatGPT auth.json (or %s=apikey for the explicitly metered job)",
			config.EnvCodexAuthSecret, config.EnvAuthMode)
	}
	data, err := fetchSecret(ctx, project, secret)
	if err != nil {
		return nil, fmt.Errorf("read codex subscription credential %s: %w", secret, err)
	}
	if err := codex.InstallSubscriptionAuth([]byte(data)); err != nil {
		return nil, fmt.Errorf("install codex subscription credential from %s: %w", secret, err)
	}
	fmt.Printf("execute-job: codex auth = ChatGPT subscription (secret %s)\n", secret)
	return &codexCredential{project: project, secret: secret, baseline: []byte(data)}, nil
}

// persist writes a refreshed auth.json back to Secret Manager as a new version.
//
// Guarded twice: the installed file must be newer than what this job started
// with, AND newer than the secret's CURRENT latest — a parallel job may have
// refreshed and written first, and an older write must never shadow it.
// Failures are logged, not returned: the task's outcome is already decided.
func (c *codexCredential) persist() {
	if c == nil {
		return
	}
	path, err := codex.InstalledAuthPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "execute-job: WARNING: codex auth write-back: %v\n", err)
		return
	}
	cur, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "execute-job: WARNING: codex auth write-back: read %s: %v\n", path, err)
		return
	}
	if !codex.NewerRefresh(c.baseline, cur) {
		return // not refreshed during this task — nothing to persist
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	latest, err := fetchSecret(ctx, c.project, c.secret)
	if err != nil {
		fmt.Fprintf(os.Stderr, "execute-job: WARNING: codex auth write-back: re-read %s: %v\n", c.secret, err)
		return
	}
	if !codex.NewerRefresh([]byte(latest), cur) {
		fmt.Println("execute-job: codex auth refreshed, but the stored secret is already as new — not written back")
		return
	}
	if err := addSecretVersion(ctx, c.project, c.secret, cur); err != nil {
		fmt.Fprintf(os.Stderr, "execute-job: WARNING: codex auth refreshed but write-back to %s FAILED: %v — the stored credential is now stale\n", c.secret, err)
		return
	}
	fmt.Printf("execute-job: codex auth refreshed during the task — wrote a new version of %s\n", c.secret)
}

// addSecretVersion stores data as the new latest version of a secret.
func addSecretVersion(ctx context.Context, project, name string, data []byte) error {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("secret manager client: %w", err)
	}
	defer func() { _ = client.Close() }()
	parent := name
	if !strings.HasPrefix(name, "projects/") {
		parent = fmt.Sprintf("projects/%s/secrets/%s", project, name)
	}
	_, err = client.AddSecretVersion(ctx, &secretmanagerpb.AddSecretVersionRequest{
		Parent:  parent,
		Payload: &secretmanagerpb.SecretPayload{Data: data},
	})
	return err
}
