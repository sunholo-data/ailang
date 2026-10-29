package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
)

func isolatePublishCLI(t *testing.T) {
	t.Helper()
	clearMessagesEnv(t)
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		upper := strings.ToUpper(key)
		if strings.HasPrefix(upper, "OTEL_") || strings.HasPrefix(upper, "OTLP_") || strings.HasPrefix(upper, "GOOGLE_") || strings.HasPrefix(upper, "GCLOUD_") || strings.HasPrefix(upper, "AILANG_TRACE") {
			t.Setenv(key, "")
		}
	}
	t.Setenv(config.EnvRegistryAPIKey, "")
	t.Setenv(config.EnvAgentPolicy, "")
}

func TestPublishAuthCLI_HelpAndSelection(t *testing.T) {
	bin := buildAilang(t)
	isolatePublishCLI(t)
	home, work := t.TempDir(), t.TempDir()
	for _, prefix := range [][]string{{"publish"}, {"pkg", "publish"}} {
		missing := filepath.Join(home, "absent-key")
		help := runCLIIn(t, bin, home, work, append(append([]string{}, prefix...), "--key-file", missing, "--help")...)
		if help.exitCode != 0 || !strings.Contains(help.stdout, "--key-file") {
			t.Fatalf("publish file help: %+v", help)
		}
		dry := runCLIIn(t, bin, home, work, append(append([]string{}, prefix...), "--key-file", missing, "--dry-run")...)
		if dry.exitCode == 0 || !strings.Contains(dry.stderr, "no ailang.toml") {
			t.Fatalf("unused key should reach ordinary package error: %+v", dry)
		}
		absent := runCLIIn(t, bin, home, work, prefix...)
		if absent.exitCode == 0 || !strings.Contains(absent.stderr, "publish requires") {
			t.Fatalf("missing credential should precede package: %+v", absent)
		}
		empty := runCLIIn(t, bin, home, work, append(append([]string{}, prefix...), "--key-file=")...)
		if empty.exitCode == 0 || !strings.Contains(empty.stderr, "path is empty") {
			t.Fatalf("explicit empty file: %+v", empty)
		}
	}
}

func TestPublishAuthCLI_EnvWarningAndAuthority(t *testing.T) {
	bin := buildAilang(t)
	isolatePublishCLI(t)
	home, work := t.TempDir(), t.TempDir()
	t.Setenv(config.EnvRegistryAPIKey, "dummy-legacy-key")
	r := runCLIIn(t, bin, home, work, "publish")
	if strings.Count(r.stderr, "is inherited") != 1 || strings.Contains(r.stderr, "dummy-legacy-key") {
		t.Fatalf("legacy warning: %+v", r)
	}
	r = runCLIIn(t, bin, home, work, "publish", "--key-file", "absent")
	if r.exitCode == 0 || !strings.Contains(r.stderr, "both --key-file") {
		t.Fatalf("ambiguous credential: %+v", r)
	}
	t.Setenv(config.EnvAgentPolicy, "private-policy")
	for _, args := range [][]string{{"publish", "--key-file", "absent"}, {"pkg", "publish", "--key-file", "absent", "--dry-run"}} {
		r = runCLIIn(t, bin, home, work, args...)
		if r.exitCode == 0 || !strings.Contains(r.stderr, "confined by AILANG_AGENT_POLICY") || strings.Contains(r.stderr, "both --key-file") {
			t.Fatalf("authority ordering: %+v", r)
		}
	}
}
