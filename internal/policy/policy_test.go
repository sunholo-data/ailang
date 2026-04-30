package policy

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTemp(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "agent-policy.toml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoad_FullExample(t *testing.T) {
	p := writeTemp(t, `
allowed_caps = ["Net", "Clock"]
fs_sandbox = "/tmp/work"
net_allow = ["api.example.com"]
budgets = { Net = 100, FS = 0 }
timeout_ms = 5000
max_source_bytes = 65536
ai_provider = "stub"
entry = "main"
`)
	pol, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got, want := len(pol.AllowedCaps), 2; got != want {
		t.Errorf("AllowedCaps len = %d, want %d", got, want)
	}
	if pol.Budgets["Net"] != 100 {
		t.Errorf("Net budget = %d, want 100", pol.Budgets["Net"])
	}
	if pol.Entry != "main" {
		t.Errorf("Entry = %q", pol.Entry)
	}
}

func TestLoad_EmptyFileIsDenyAll(t *testing.T) {
	p := writeTemp(t, "")
	pol, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(pol.AllowedCaps) != 0 {
		t.Errorf("empty file should default to deny-all, got %v", pol.AllowedCaps)
	}
	if pol.Entry != "main" {
		t.Errorf("default entry should be 'main', got %q", pol.Entry)
	}
}

func TestLoad_UnknownFieldRejected(t *testing.T) {
	p := writeTemp(t, `
allowed_caps = ["Net"]
typo_field = "oops"
`)
	_, err := Load(p)
	if err == nil {
		t.Fatal("expected error for unknown field, got nil")
	}
}

func TestLoad_BadSyntaxRejected(t *testing.T) {
	p := writeTemp(t, `not valid = = toml`)
	_, err := Load(p)
	if err == nil {
		t.Fatal("expected error for bad TOML, got nil")
	}
}
