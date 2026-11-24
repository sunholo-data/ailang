package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/credentiallease"
	"github.com/sunholo-data/ailang/internal/executor"
	storefs "github.com/sunholo-data/ailang/internal/storage/firestore"
)

// Rotating subscription credentials on the cloud plane.
//
// codex (ChatGPT auth.json) and agy (Google AI token file) both keep their sign-in
// in a FILE the CLI rewrites when it refreshes. A container is discarded after one
// task, so the job:
//  1. takes an exclusive Firestore lease on the secret BEFORE reading it,
//  2. installs the latest version into the CLI's isolated home,
//  3. runs the task, heartbeating the lease (loss cancels the process tree),
//  4. writes the file back as a new secret version if it changed, then releases,
//  5. QUARANTINES instead (keeps the lease, needs attended recovery) when the write
//     fails or the executor cannot prove its process stopped.
//
// This was codex-only until M-AGY-CLOUD (design_docs/planned/m-agy-cloud-executor.md);
// the machinery is shared and only the rotatingCredentialSpec differs per provider.

// rotatingCredentialSpec is what differs between providers.
type rotatingCredentialSpec struct {
	label     string                 // for logs and errors: "Codex", "agy"
	secretEnv string                 // env var naming the Secret Manager secret
	secret    func() string          // its value
	precheck  func() error           // the isolated home must be explicit
	validate  func([]byte) error     // refuses anything that is not this provider's subscription credential
	install   func([]byte) error     // writes it where the CLI reads it
	path      func() (string, error) // where the CLI keeps it after the task
	billing   string                 // printed on install
}

// rotatingSpecs: providers whose cloud jobs run on a rotating credential.
var rotatingSpecs = map[string]*rotatingCredentialSpec{
	"codex": codexCredentialSpec,
	"agy":   agyCredentialSpec,
}

// rotatingCredential is what a cloud job installed, kept so the refreshed file can be
// written back when the task ends. nil means nothing to persist.
type rotatingCredential struct {
	spec          *rotatingCredentialSpec
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

func (c *rotatingCredential) label() string {
	if c == nil || c.spec == nil {
		return "rotating"
	}
	return c.spec.label
}

// canonicalSecretKey requires project IDs (numeric aliases are refused) and rejects
// versions: ownership always covers the secret, regardless of which Cloud Run job
// variant requested it.
func canonicalSecretKey(project, secret string) (string, error) {
	if strings.HasPrefix(secret, "projects/") {
		parts := strings.Split(secret, "/")
		if len(parts) != 4 || parts[1] == "" || (parts[1][0] < 'a' || parts[1][0] > 'z') || parts[2] != "secrets" || parts[3] == "" {
			return "", fmt.Errorf("rotating credential must name an unversioned secret resource")
		}
		return secret, nil
	}
	if project == "" || (project[0] < 'a' || project[0] > 'z') || secret == "" || strings.Contains(secret, "/") {
		return "", fmt.Errorf("rotating credential requires project and secret name")
	}
	return fmt.Sprintf("projects/%s/secrets/%s", project, secret), nil
}

// installRotatingCredential gives a cloud job its subscription credential, BEFORE
// the preflight health check. Providers with no rotating credential get (nil, nil).
// No secret is a hard failure: there is no metered fallback for these lanes.
func installRotatingCredential(ctx context.Context, provider, project string) (*rotatingCredential, error) {
	spec := rotatingSpecs[provider]
	if spec == nil {
		return nil, nil
	}
	secret := spec.secret()
	if secret == "" {
		return nil, fmt.Errorf("%s job has no subscription credential: set %s to the Secret Manager secret holding it",
			provider, spec.secretEnv)
	}
	key, err := canonicalSecretKey(project, secret)
	if err != nil {
		return nil, err
	}
	if err := spec.precheck(); err != nil {
		return nil, err
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
	c := &rotatingCredential{spec: spec, project: secretProject, secret: key, key: key, owner: execution + ":" + hex.EncodeToString(nonce), lease: storefs.NewCredentialLeaseStore(client), closeStore: client.Close}
	if err = c.restore(ctx, func() ([]byte, error) {
		data, err := fetchSecret(ctx, secretProject, key+"/versions/latest")
		return []byte(data), err
	}, spec.install); err != nil {
		_ = client.Close()
		return nil, err
	}
	fmt.Printf("execute-job: %s auth = %s (secret %s)\n", provider, spec.billing, secret)
	c.read = func() ([]byte, error) {
		path, err := spec.path()
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
func (c *rotatingCredential) restore(ctx context.Context, fetch func() ([]byte, error), install func([]byte) error) (restoreErr error) {
	if err := c.lease.Change(ctx, c.key, c.owner, "acquire"); err != nil {
		return fmt.Errorf("acquire exclusive %s credential: %w", c.label(), err)
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
		return fmt.Errorf("read %s subscription credential: %w", c.label(), err)
	}
	if err := install(data); err != nil {
		return fmt.Errorf("install %s subscription credential: %w", c.label(), err)
	}
	c.baseline = data
	return nil
}

// finalizeRotatingCredential refuses to read or release a rotating credential when
// the executor cannot establish that its refresh-capable process has stopped.
func finalizeRotatingCredential(c *rotatingCredential, execErr error) error {
	if errors.Is(execErr, executor.ErrProcessTerminationUnconfirmed) {
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
// Changed files are persisted even if a refresh timestamp has equal clock precision.
// Failures retain/quarantine ownership; they must never silently reuse a seed.
func (c *rotatingCredential) finish() error {
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

func (c *rotatingCredential) persistOwned(ctx context.Context) error {
	if err := c.lease.Change(ctx, c.key, c.owner, "heartbeat"); err != nil {
		return fmt.Errorf("%s credential ownership check: %w", c.label(), err)
	}
	cur, err := c.read()
	if err != nil {
		return fmt.Errorf("read current %s credential; recovery required: %w", c.label(), err)
	}
	if c.spec != nil {
		if err := c.spec.validate(cur); err != nil {
			return fmt.Errorf("invalid current %s credential; recovery required: %w", c.label(), err)
		}
	}
	if bytes.Equal(c.baseline, cur) {
		return nil
	}
	if err := c.write(ctx, cur); err != nil {
		return fmt.Errorf("%s credential write-back failed; secret quarantined; attended recovery required: %w", c.label(), err)
	}
	fmt.Printf("execute-job: %s credential durably persisted\n", c.label())
	return nil
}

// quarantine retains ownership when a panic cannot prove process termination.
func (c *rotatingCredential) quarantine() {
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
// The executors configure process-tree cancellation for this context.
// No owner is ever automatically replaced, including after heartbeat expiry.
func (c *rotatingCredential) heartbeat(interval time.Duration) func() {
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
					fmt.Fprintf(os.Stderr, "execute-job: %s credential ownership unverified; cancelling process tree: %v\n", c.label(), err)
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
