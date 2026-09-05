package codex

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
)

func authPath(home string) string { return filepath.Join(home, ".codex", "auth.json") }

func TestEnsureAPIKeyAuth_WritesWhenAbsent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME semantics differ on windows")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("OPENAI_API_KEY", "sk-test-value")

	wrote, err := EnsureAPIKeyAuth()
	if err != nil {
		t.Fatalf("EnsureAPIKeyAuth: %v", err)
	}
	if !wrote {
		t.Fatal("expected a file to be written when none exists")
	}

	data, err := os.ReadFile(authPath(home))
	if err != nil {
		t.Fatalf("read auth.json: %v", err)
	}
	var got apiKeyAuth
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("auth.json is not valid JSON: %v", err)
	}
	if got.AuthMode != "apikey" {
		t.Errorf("auth_mode = %q, want %q", got.AuthMode, "apikey")
	}
	if got.APIKey != "sk-test-value" {
		t.Error("OPENAI_API_KEY was not carried into auth.json")
	}

	// authLane must read this back as BILLED — the cost record depends on it,
	// and an api-key lane misreported as subscription would under-count spend.
	e := &CodexExecutor{}
	if lane := e.authLane(); lane != executor.AuthLaneBilled {
		t.Errorf("authLane = %v, want AuthLaneBilled for the file we just wrote", lane)
	}
}

// TestEnsureAPIKeyAuth_NeverOverwrites is the one that protects real money: the
// rig authenticates codex with a ChatGPT-subscription auth.json AND keeps
// OPENAI_API_KEY exported. Overwriting there would move flat-rate runs onto a
// metered key with nothing in the output to say so.
func TestEnsureAPIKeyAuth_NeverOverwrites(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME semantics differ on windows")
	}
	for _, existing := range []string{
		`{"auth_mode":"chatgpt","tokens":{"access_token":"real-subscription-token"}}`,
		`{"auth_mode":"apikey","OPENAI_API_KEY":"sk-someone-elses-key"}`,
		`not valid json at all`,
		``,
	} {
		t.Run("", func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("OPENAI_API_KEY", "sk-env-key-that-must-not-win")

			if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(authPath(home), []byte(existing), 0o600); err != nil {
				t.Fatal(err)
			}

			wrote, err := EnsureAPIKeyAuth()
			if err != nil {
				t.Fatalf("EnsureAPIKeyAuth: %v", err)
			}
			if wrote {
				t.Fatal("overwrote an existing auth.json")
			}
			after, err := os.ReadFile(authPath(home))
			if err != nil {
				t.Fatal(err)
			}
			if string(after) != existing {
				t.Errorf("auth.json was modified:\n before: %q\n  after: %q", existing, after)
			}
		})
	}
}

func TestEnsureAPIKeyAuth_NoKeyWritesNothing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME semantics differ on windows")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("OPENAI_API_KEY", "")

	wrote, err := EnsureAPIKeyAuth()
	if err != nil {
		t.Fatalf("EnsureAPIKeyAuth: %v", err)
	}
	if wrote {
		t.Fatal("wrote an auth.json with no key available")
	}
	if _, err := os.Stat(authPath(home)); !os.IsNotExist(err) {
		t.Error("auth.json exists but no key was available to write")
	}
}

func TestEnsureAPIKeyAuth_FileIsOwnerOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits differ on windows")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("OPENAI_API_KEY", "sk-test-value")

	if _, err := EnsureAPIKeyAuth(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(authPath(home))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != authFileMode {
		t.Errorf("auth.json mode = %o, want %o — it holds a live API key", perm, authFileMode)
	}
}
