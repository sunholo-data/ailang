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

// Every bucket the -bucket flag advertises must be selectable.
//
// codex, ollama and openrouter are paced entirely by a provider gauge and never write
// token-ledger rows, so filtering the ledger by one of them legitimately yields nothing.
// The empty-result guard — which exists to catch a MISTYPED bucket name — listed only
// codex and ollama, so `--bucket openrouter` failed with
//
//	no bucket "openrouter" in the ledger (have: [anthropic ollama opencode])
//
// on a correctly spelled, documented name. Measured 2026-09-14, against a real ledger.
// The typo guard itself is still worth having, so the last case keeps it honest.
func TestMissionQuotaAdvertisedBucketsAreAllSelectable(t *testing.T) {
	for _, bucket := range []string{"codex", "ollama", "openrouter", "anthropic"} {
		t.Run(bucket, func(t *testing.T) {
			t.Setenv("OLLAMA_API_KEY", "")
			t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
			t.Setenv("OPENROUTER_API_KEY", "")
			paths := mission.Paths{Home: t.TempDir()}
			t.Setenv("CODEX_HOME", filepath.Join(paths.Home, "codex"))
			// Only anthropic gets a ledger row: the other three must be selectable with
			// an entirely empty ledger, which is their normal state.
			if bucket == "anthropic" {
				if err := mission.AppendSpend(paths, "anthropic", 10, 1, time.Now()); err != nil {
					t.Fatal(err)
				}
			}
			_ = captureStdout(t, func() {
				if err := missionQuotaWithPaths([]string{"--over", "--bucket", bucket}, paths, time.Now()); err != nil {
					t.Fatalf("advertised bucket %q is not selectable: %v", bucket, err)
				}
			})
		})
	}

	// The guard must still catch a genuine typo rather than accept anything.
	t.Run("typo still refused", func(t *testing.T) {
		t.Setenv("OLLAMA_API_KEY", "")
		t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
		t.Setenv("OPENROUTER_API_KEY", "")
		paths := mission.Paths{Home: t.TempDir()}
		t.Setenv("CODEX_HOME", filepath.Join(paths.Home, "codex"))
		if err := mission.AppendSpend(paths, "anthropic", 10, 1, time.Now()); err != nil {
			t.Fatal(err)
		}
		_ = captureStdout(t, func() {
			err := missionQuotaWithPaths([]string{"--over", "--bucket", "openrouterr"}, paths, time.Now())
			if err == nil {
				t.Fatal("a misspelled bucket must be refused, not silently reported as empty")
			}
			if !strings.Contains(err.Error(), "openrouterr") {
				t.Fatalf("error should name the bad bucket: %v", err)
			}
		})
	})
}
