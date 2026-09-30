package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	nilCred.persist() // must not panic

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
	(&codexCredential{project: "proj", secret: "s", baseline: auth}).persist()
}
