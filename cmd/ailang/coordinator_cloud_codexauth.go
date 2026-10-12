package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sync"

	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/credentiallease"
	storefs "github.com/sunholo-data/ailang/internal/storage/firestore"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/executor/codex"
)

// codexCredential is what a cloud codex job installed, kept so the refreshed file
// can be written back when the task ends. nil means nothing to persist.
type codexCredential struct {
	project       string
	secret        string
	baseline      []byte
	key, owner    string
	lease         credentiallease.Store
	ctx           context.Context
	cancel        context.CancelFunc
	stopHeartbeat func()
	closeStore    func() error
	read          func() ([]byte, error)
	write         func(context.Context, []byte) error
	once          sync.Once
	finishErr     error
}

// canonicalCodexSecret requires project IDs (numeric aliases are refused) and rejects versions: ownership always covers the secret,
// regardless of which Cloud Run job variant requested it.
func canonicalCodexSecret(project, secret string) (string, error) {
	if strings.HasPrefix(secret, "projects/") {
		parts := strings.Split(secret, "/")
		if len(parts) != 4 || parts[1] == "" || (parts[1][0] < 'a' || parts[1][0] > 'z') || parts[2] != "secrets" || parts[3] == "" {
			return "", fmt.Errorf("Codex credential must name an unversioned secret resource")
		}
		return secret, nil
	}
	if project == "" || (project[0] < 'a' || project[0] > 'z') || secret == "" || strings.Contains(secret, "/") {
		return "", fmt.Errorf("Codex credential requires project and secret name")
	}
	return fmt.Sprintf("projects/%s/secrets/%s", project, secret), nil
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
	key, err := canonicalCodexSecret(project, secret)
	if err != nil {
		return nil, err
	}
	if config.CodexHome() == "" {
		return nil, fmt.Errorf("cloud subscription execution requires an explicit isolated CODEX_HOME")
	}
	// Cloud Run supplies this execution identity; it is runtime metadata, not
	// operator-configured AILANG configuration.
	execution := config.CloudRunExecution()
	if execution == "" {
		return nil, fmt.Errorf("cloud credential ownership requires CLOUD_RUN_EXECUTION")
	}
	// Store the lock beside the secret, so alternate job/storage project settings
	// cannot create an independent lock for the same rotating credential.
	secretProject := strings.Split(key, "/")[1]
	client, err := storefs.NewClientForProject(ctx, secretProject)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, 16)
	if _, err = rand.Read(nonce); err != nil {
		_ = client.Close()
		return nil, err
	}
	c := &codexCredential{project: secretProject, secret: key, key: key, owner: execution + ":" + hex.EncodeToString(nonce), lease: storefs.NewCredentialLeaseStore(client), closeStore: client.Close}
	if err = c.restore(ctx, func() ([]byte, error) {
		data, err := fetchSecret(ctx, secretProject, key+"/versions/latest")
		return []byte(data), err
	}, codex.InstallSubscriptionAuth); err != nil {
		_ = client.Close()
		return nil, err
	}
	fmt.Printf("execute-job: codex auth = ChatGPT subscription (secret %s)\n", secret)
	c.read = func() ([]byte, error) {
		path, err := codex.InstalledAuthPath()
		if err != nil {
			return nil, err
		}
		return os.ReadFile(path)
	}
	c.write = func(ctx context.Context, data []byte) error { return addSecretVersion(ctx, c.project, c.secret, data) }
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.stopHeartbeat = c.heartbeat(10 * time.Second)
	return c, nil
}

// restore holds exclusive ownership before the first credential read. A failure
// before any subprocess ran can safely release it; ambiguous release stays held.
func (c *codexCredential) restore(ctx context.Context, fetch func() ([]byte, error), install func([]byte) error) (restoreErr error) {
	if err := c.lease.Change(ctx, c.key, c.owner, "acquire"); err != nil {
		return fmt.Errorf("acquire exclusive Codex credential: %w", err)
	}
	defer func() {
		if restoreErr != nil {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := c.lease.Change(cleanupCtx, c.key, c.owner, "release"); err != nil {
				restoreErr = fmt.Errorf("%v; credential ownership remains blocked: %w", restoreErr, err)
			}
		}
	}()
	data, err := fetch()
	if err != nil {
		return fmt.Errorf("read Codex subscription credential: %w", err)
	}
	if err := install(data); err != nil {
		return fmt.Errorf("install Codex subscription credential: %w", err)
	}
	c.baseline = data
	return nil
}

// finalizeCodexCredential refuses to read or release rotating credentials when
// the executor cannot establish that its refresh-capable process has stopped.
func finalizeCodexCredential(c *codexCredential, execErr error) error {
	if errors.Is(execErr, codex.ErrProcessTerminationUnconfirmed) {
		c.quarantine()
		return execErr
	}
	if err := c.finish(); err != nil {
		if execErr == nil {
			return err
		}
		return fmt.Errorf("%v; %w", execErr, err)
	}
	return execErr
}

// finish only releases ownership after the current credential is durable.
// Changed files are persisted even if last_refresh has equal clock precision.
// Failures retain/quarantine ownership; they must never silently reuse a seed.
func (c *codexCredential) finish() error {
	if c == nil {
		return nil
	}
	c.once.Do(func() {
		if c.stopHeartbeat != nil {
			c.stopHeartbeat()
		}
		if c.cancel != nil {
			defer c.cancel()
		}
		if c.closeStore != nil {
			defer func() { _ = c.closeStore() }()
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		c.finishErr = c.persistOwned(ctx)
		if c.finishErr != nil {
			// A failed quarantine write still leaves the non-expiring owner blocking use.
			_ = c.lease.Change(ctx, c.key, c.owner, "quarantine")
			return
		}
		c.finishErr = c.lease.Change(ctx, c.key, c.owner, "release")
	})
	return c.finishErr
}
func (c *codexCredential) persistOwned(ctx context.Context) error {
	if err := c.lease.Change(ctx, c.key, c.owner, "heartbeat"); err != nil {
		return fmt.Errorf("Codex credential ownership check: %w", err)
	}
	cur, err := c.read()
	if err != nil {
		return fmt.Errorf("read current Codex credential; recovery required: %w", err)
	}
	if _, err := codex.ParseSubscriptionAuth(cur); err != nil {
		return fmt.Errorf("invalid current Codex credential; recovery required: %w", err)
	}
	if bytes.Equal(c.baseline, cur) {
		return nil
	}
	if err := c.write(ctx, cur); err != nil {
		return fmt.Errorf("Codex credential write-back failed; secret quarantined; attended recovery required: %w", err)
	}
	fmt.Println("execute-job: Codex credential durably persisted")
	return nil
}

// quarantine retains ownership when a panic cannot prove process termination.
func (c *codexCredential) quarantine() {
	if c == nil {
		return
	}
	c.once.Do(func() {
		if c.cancel != nil {
			c.cancel()
		}
		if c.stopHeartbeat != nil {
			c.stopHeartbeat()
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = c.lease.Change(ctx, c.key, c.owner, "quarantine")
		if c.closeStore != nil {
			_ = c.closeStore()
		}
	})
}

// heartbeat cancels the executor context on every unverified ownership result.
// The Codex executor configures process-tree cancellation for this context.
// No owner is ever automatically replaced, including after heartbeat expiry.
func (c *codexCredential) heartbeat(interval time.Duration) func() {
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		tick := time.NewTicker(interval)
		defer tick.Stop()
		for {
			select {
			case <-stop:
				return
			case <-c.ctx.Done():
				return
			case <-tick.C:
				ctx, cancel := context.WithTimeout(c.ctx, interval)
				err := c.lease.Change(ctx, c.key, c.owner, "heartbeat")
				cancel()
				if err != nil {
					fmt.Fprintln(os.Stderr, "execute-job: Codex credential ownership unverified; cancelling process tree:", err)
					c.cancel()
					return
				}
			}
		}
	}()
	return func() { close(stop); <-done }
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
