package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/sunholo-data/ailang/internal/executor/codex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// These cover the branches that decide BILLING without touching Secret Manager:
// a wrong turn here is exactly how every cloud codex run came to bill the API.

func TestInstallCodexCredential_NonCodexProviderIsANoOp(t *testing.T) {
	testutil.SetHomeDir(t, t.TempDir())
	cred, err := installCodexCredential(t.Context(), "claude", "proj")
	if err != nil || cred != nil {
		t.Fatalf("non-codex provider: got (%v, %v), want (nil, nil)", cred, err)
	}
}

func TestInstallCodexCredential_NoSecretAndNoAPIKeyModeFailsLoudly(t *testing.T) {
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	t.Setenv("AILANG_AUTH_MODE", "")
	t.Setenv("AILANG_CODEX_AUTH_SECRET", "")
	t.Setenv("OPENAI_API_KEY", "sk-must-not-be-used")

	_, err := installCodexCredential(t.Context(), "codex", "proj")
	if err == nil || !strings.Contains(err.Error(), "AILANG_CODEX_AUTH_SECRET") {
		t.Fatalf("want a loud error naming AILANG_CODEX_AUTH_SECRET, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(home, ".codex", "auth.json")); !os.IsNotExist(statErr) {
		t.Fatal("an api-key auth.json was written from OPENAI_API_KEY — the silent metered fallback is back")
	}
}

func TestInstallCodexCredential_APIKeyModeUsesTheKey(t *testing.T) {
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	t.Setenv("AILANG_AUTH_MODE", "apikey")
	t.Setenv("OPENAI_API_KEY", "sk-explicit-metered")

	cred, err := installCodexCredential(t.Context(), "codex", "proj")
	if err != nil || cred != nil {
		t.Fatalf("apikey mode: got (%v, %v), want (nil, nil) — nothing to write back", cred, err)
	}
	data, err := os.ReadFile(filepath.Join(home, ".codex", "auth.json"))
	if err != nil || !strings.Contains(string(data), `"apikey"`) {
		t.Fatalf("apikey mode must write the api-key auth.json, got %q, %v", data, err)
	}
}

func TestCodexCredentialPersist_NilAndUnrefreshedAreNoOps(t *testing.T) {
	var nilCred *codexCredential
	_ = nilCred.finish() // must not panic

	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	auth := []byte(`{"auth_mode":"chatgpt","last_refresh":"2026-09-30T10:00:00Z","tokens":{"refresh_token":"r"}}`)
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".codex", "auth.json"), auth, 0o600); err != nil {
		t.Fatal(err)
	}
	// Unchanged file: persist must return before any Secret Manager call (none is
	// reachable here, so reaching one would fail the run's log, not this test —
	// the assertion is that it returns promptly with nothing to do).
	f := &fakeCodexLease{owner: "test"}
	c := &codexCredential{baseline: auth, lease: f, owner: "test", key: "projects/p/secrets/s", read: func() ([]byte, error) { return auth, nil }, write: func(context.Context, []byte) error { t.Fatal("unchanged credential written"); return nil }}
	if err := c.finish(); err != nil {
		t.Fatal(err)
	}
}

// A fake lease/secret store checks the complete lifecycle without credentials.
type fakeCodexLease struct {
	owner       string
	quarantined bool
	lost        bool
	actions     []string
}

func (f *fakeCodexLease) Change(ctx context.Context, key, owner, action string) error {
	f.actions = append(f.actions, action)
	if f.lost {
		return fmt.Errorf("ownership lost")
	}
	switch action {
	case "acquire":
		if f.owner != "" || f.quarantined {
			return fmt.Errorf("busy")
		}
		f.owner = owner
	case "heartbeat":
		if f.owner != owner {
			return fmt.Errorf("stale")
		}
	case "release":
		if f.owner != owner {
			return fmt.Errorf("stale")
		}
		f.owner = ""
	case "quarantine":
		f.quarantined = true
	}
	return nil
}
func TestCodexCredentialFinishPersistsBeforeRelease(t *testing.T) {
	old := []byte(`{"auth_mode":"chatgpt","last_refresh":"2026-09-30T10:00:00Z","tokens":{"refresh_token":"old"}}`)
	next := []byte(`{"auth_mode":"chatgpt","last_refresh":"2026-09-30T10:00:00Z","tokens":{"refresh_token":"new"}}`)
	f := &fakeCodexLease{owner: "test"}
	written := false
	c := &codexCredential{baseline: old, lease: f, owner: "test", key: "projects/p/secrets/s", read: func() ([]byte, error) { return next, nil }, write: func(context.Context, []byte) error {
		written = true
		if f.owner == "" {
			t.Fatal("release before write")
		}
		return nil
	}}
	if err := c.finish(); err != nil {
		t.Fatal(err)
	}
	if !written || f.owner != "" {
		t.Fatal("credential not durably persisted and released")
	}
}
func TestCodexCredentialFailedWriteQuarantines(t *testing.T) {
	old := []byte(`{"auth_mode":"chatgpt","last_refresh":"2026-09-30T10:00:00Z","tokens":{"refresh_token":"old"}}`)
	next := []byte(`{"auth_mode":"chatgpt","last_refresh":"2026-10-01T10:00:00Z","tokens":{"refresh_token":"new"}}`)
	f := &fakeCodexLease{owner: "test"}
	c := &codexCredential{baseline: old, lease: f, owner: "test", key: "projects/p/secrets/s", read: func() ([]byte, error) { return next, nil }, write: func(context.Context, []byte) error { return fmt.Errorf("offline") }}
	if err := c.finish(); err == nil {
		t.Fatal("failed write hidden")
	}
	if !f.quarantined || f.owner == "" {
		t.Fatal("failed write released obsolete secret")
	}
}
func TestCodexCredentialLostOwnershipCannotWrite(t *testing.T) {
	f := &fakeCodexLease{owner: "other", lost: true}
	c := &codexCredential{lease: f, owner: "test", key: "projects/p/secrets/s", read: func() ([]byte, error) { t.Fatal("stale owner read auth"); return nil, nil }}
	if err := c.finish(); err == nil {
		t.Fatal("ownership loss hidden")
	}
}
func TestCodexLeaseKeyCanonical(t *testing.T) {
	a, err := canonicalCodexSecret("p", "s")
	if err != nil {
		t.Fatal(err)
	}
	b, err := canonicalCodexSecret("ignored", "projects/p/secrets/s")
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatal("same secret has distinct lease keys")
	}
	if _, err := canonicalCodexSecret("p", "projects/p/secrets/s/versions/latest"); err == nil {
		t.Fatal("versioned secret accepted")
	}
}

func TestCodexCredentialHeartbeatLossCancelsExecution(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	c := &codexCredential{ctx: ctx, cancel: cancel, lease: &fakeCodexLease{lost: true}, owner: "owner", key: "projects/p/secrets/s"}
	stop := c.heartbeat(time.Millisecond)
	defer stop()
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("lease loss did not cancel executor")
	}
}
func TestCodexCredentialQuarantineOnPanicKeepsOwnership(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	f := &fakeCodexLease{owner: "owner"}
	c := &codexCredential{ctx: ctx, cancel: cancel, lease: f, owner: "owner", key: "projects/p/secrets/s"}
	c.quarantine()
	if ctx.Err() == nil || !f.quarantined || f.owner == "" {
		t.Fatal("panic released potentially live credential owner")
	}
}

func TestCodexCredentialRestoreOnlyAfterExclusiveAcquire(t *testing.T) {
	f := &fakeCodexLease{}
	c := &codexCredential{lease: f, owner: "codex-execution", key: "projects/p/secrets/s"}
	restored := false
	err := c.restore(t.Context(), func() ([]byte, error) {
		if f.owner != c.owner {
			t.Fatal("restored credential before ownership")
		}
		restored = true
		return []byte("fake"), nil
	}, func([]byte) error { return nil })
	if err != nil || !restored {
		t.Fatalf("restore: %v", err)
	}
	other := &codexCredential{lease: f, owner: "codex-go-execution", key: c.key}
	if err = other.restore(t.Context(), func() ([]byte, error) { t.Fatal("overlapping job fetched credential"); return nil, nil }, func([]byte) error { return nil }); err == nil {
		t.Fatal("codex-go overlapped codex")
	}
}
func TestCodexCredentialRestoreFailureReleasesWithoutRunning(t *testing.T) {
	f := &fakeCodexLease{}
	c := &codexCredential{lease: f, owner: "execution", key: "projects/p/secrets/s"}
	err := c.restore(t.Context(), func() ([]byte, error) { return nil, fmt.Errorf("unavailable") }, func([]byte) error { t.Fatal("installed after failed read"); return nil })
	if err == nil || f.owner != "" {
		t.Fatal("failed restore leaked ownership")
	}
}

func TestCodexCredentialUnconfirmedTerminationQuarantinesWithoutRelease(t *testing.T) {
	f := &fakeCodexLease{owner: "owner"}
	c := &codexCredential{lease: f, owner: "owner", key: "projects/p/secrets/s", read: func() ([]byte, error) { t.Fatal("credential read while termination unconfirmed"); return nil, nil }}
	execErr := fmt.Errorf("executor failed: %w", codex.ErrProcessTerminationUnconfirmed)
	if err := finalizeCodexCredential(c, execErr); !errors.Is(err, codex.ErrProcessTerminationUnconfirmed) {
		t.Fatalf("termination error hidden: %v", err)
	}
	if !f.quarantined || f.owner != "owner" {
		t.Fatal("potentially live owner released")
	}
	_ = c.finish() // deferred cleanup must also leave the owner blocked
	if f.owner != "owner" {
		t.Fatal("deferred cleanup released quarantined ownership")
	}
}
