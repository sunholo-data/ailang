package config

import "testing"

func TestCodexAuthSecret(t *testing.T) {
	t.Setenv(EnvCodexAuthSecret, "ailang-codex-auth-json")
	if got := CodexAuthSecret(); got != "ailang-codex-auth-json" {
		t.Fatalf("CodexAuthSecret() = %q", got)
	}
}
