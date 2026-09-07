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
	if output != "codex\n" {
		t.Fatalf("corrupt ledger bypass: %q", output)
	}
}
func TestMissionQuotaOtherBucketDoesNotInspectCodex(t *testing.T) {
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
