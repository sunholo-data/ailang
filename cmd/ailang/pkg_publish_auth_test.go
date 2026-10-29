package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/pkg"
	"github.com/sunholo-data/ailang/internal/testutil"
)

func TestPublishCredential_Selection(t *testing.T) {
	testutil.SetHomeDir(t, t.TempDir())
	t.Setenv(config.EnvAgentPolicy, "")
	for _, tt := range []struct {
		name, path, env, want string
		selected              bool
	}{
		{"ambiguous", "missing", "dummy-env-key", "both", true},
		{"empty-file", "", "", "empty", true},
		{"empty-file-and-env", "", "dummy-env-key", "empty", true},
		{"missing-credential", "", "", "requires", false},
		{"missing-file", "missing", "", "", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(config.EnvRegistryAPIKey, tt.env)
			got, err := resolvePublishCredential(tt.path, tt.selected, false)
			if err == nil || got.key != "" || strings.Contains(err.Error(), "dummy-env-key") {
				t.Fatalf("missing/ambiguous source accepted or exposed: %v", err)
			}
			if tt.want != "" && !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("wrong source failure: %v", err)
			}
		})
	}
}

func TestPublishCredential_NoDefaultOrUnusedFile(t *testing.T) {
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	t.Setenv(config.EnvAgentPolicy, "")
	t.Setenv(config.EnvRegistryAPIKey, "")
	dir := filepath.Join(home, ".config", "ailang")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "registry.key"), []byte("dummy-default"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := resolvePublishCredential("", false, false); err == nil || !strings.Contains(err.Error(), "requires") {
		t.Fatalf("implicit default probe: %v", err)
	}
	for _, path := range []string{"", filepath.Join(home, "absent")} {
		key, err := resolvePublishCredential(path, true, true)
		if err != nil || key.key != "" {
			t.Fatalf("unused file accessed: %v", err)
		}
	}
}

func TestPublishCredential_FrozenLegacyAndAuthority(t *testing.T) {
	t.Setenv(config.EnvAgentPolicy, "")
	t.Setenv(config.EnvRegistryAPIKey, "dummy-original-env")
	got, err := resolvePublishCredential("", false, false)
	if err != nil || got.key != "dummy-original-env" || !got.fromEnv {
		t.Fatal("legacy source not selected")
	}
	t.Setenv(config.EnvRegistryAPIKey, "dummy-changed-env")
	if got.key != "dummy-original-env" {
		t.Fatal("credential was not frozen")
	}
	t.Setenv(config.EnvAgentPolicy, "private-policy")
	_, err = resolvePublishCredential("absent", true, false)
	var refusal *pkg.ConfinedError
	if !errors.As(err, &refusal) {
		t.Fatalf("authority did not precede selection: %v", err)
	}
}

func TestPublishCredential_FileAndPlatform(t *testing.T) {
	t.Setenv(config.EnvAgentPolicy, "")
	t.Setenv(config.EnvRegistryAPIKey, "")
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile("key", []byte(" dummy-file-key\n"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := resolvePublishCredential("key", true, false)
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		if !errors.Is(err, pkg.ErrPublishKeyFileUnsupported) {
			t.Fatalf("unsafe platform fallback: %v", err)
		}
		return
	}
	if err != nil || got.key != "dummy-file-key" || got.fromEnv {
		t.Fatalf("relative file source: %v", err)
	}
	if err := os.WriteFile("key", []byte("replacement-dummy"), 0600); err != nil {
		t.Fatal(err)
	}
	if got.key != "dummy-file-key" {
		t.Fatal("selected file reread")
	}
	if os.Getenv(config.EnvRegistryAPIKey) != "" {
		t.Fatal("file key exported")
	}
}
