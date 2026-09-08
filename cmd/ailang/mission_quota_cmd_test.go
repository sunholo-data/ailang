package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/mission"
)

func TestMissionQuotaMissingCodexBlocksWithoutLedger(t *testing.T) {
	t.Setenv("OLLAMA_API_KEY", "")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("OPENROUTER_API_KEY", "")
	paths := mission.Paths{Home: t.TempDir()}
	t.Setenv("CODEX_HOME", filepath.Join(paths.Home, "codex"))
	output := captureStdout(t, func() {
		if err := missionQuotaWithPaths([]string{"--over", "--bucket", "codex"}, paths, time.Now()); err != nil {
			t.Fatal(err)
		}
	})
	if output != "codex\n" {
		t.Fatalf("missing accounting must block exactly once: %q", output)
	}
}
func TestMissionQuotaCorruptLedgerCannotBypassCodex(t *testing.T) {
	t.Setenv("OLLAMA_API_KEY", "")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("OPENROUTER_API_KEY", "")
	paths := mission.Paths{Home: t.TempDir()}
	t.Setenv("CODEX_HOME", filepath.Join(paths.Home, "codex"))
	dir := filepath.Join(paths.Home, ".ailang", "state")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "quota-ledger.json"), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	output := captureStdout(t, func() {
		if err := missionQuotaWithPaths([]string{"--over"}, paths, time.Now()); err != nil {
			t.Fatal(err)
		}
	})
	// Anthropic joined codex and ollama here (2026-09-08), and OpenRouter joined all three
	// later the same day: each is a provider whose quota is unreadable without a credential,
	// and unknown quota fails closed. A corrupt ledger must not open ANY of the four.
	//
	// Every one of those credentials is neutralized above, and OPENROUTER_API_KEY was the one
	// that got missed: a dev box with the key set reads a real quota, drops openrouter from
	// this list and passes, while CI has no key and fails. The assertion is only meaningful
	// if the environment cannot supply an answer — keep the Setenv list in step with the
	// providers this output can name.
	if output != "codex\nollama\nanthropic\nopenrouter\n" {
		t.Fatalf("corrupt ledger bypass: %q", output)
	}
}
func TestMissionQuotaOtherBucketDoesNotInspectCodex(t *testing.T) {
	t.Setenv("OLLAMA_API_KEY", "")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("OPENROUTER_API_KEY", "")
	paths := mission.Paths{Home: t.TempDir()}
	t.Setenv("CODEX_HOME", filepath.Join(paths.Home, "codex"))
	if err := mission.AppendSpend(paths, "anthropic", 10, 1, time.Now()); err != nil {
		t.Fatal(err)
	}
	output := captureStdout(t, func() {
		if err := missionQuotaWithPaths([]string{"--over", "--bucket", "anthropic"}, paths, time.Now()); err != nil {
			t.Fatal(err)
		}
	})
	if strings.Contains(output, "codex") {
		t.Fatalf("unrequested bucket: %q", output)
	}
}
