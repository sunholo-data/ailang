package main

import (
	"strings"
	"testing"
)

// M-V1-SIMPLIFY-S5 M4, D7: `ailang storage` carried a one-time local-SQLite →
// GCP-Firestore migration (`migrate`, `verify`) backed by internal/storage/migrate.
// `git log -S storageMigrate` and `git log -S NewMigrator` each return exactly
// one commit — 56ac6827c, 2026-02-18 — and the package had no tests and no
// caller outside cmd/ailang/storage.go. The plane switch (AILANG_STORAGE and
// its per-store overrides, M-V1-SIMPLIFY-S3 M3) is how a store moves now.
//
// `status` is deliberately NOT removed: CLAUDE.md's session-start check tells
// every machine to confirm the messaging store with it.
func TestStorage_MigrationSubcommandsAreGone(t *testing.T) {
	for _, sub := range []string{"migrate", "verify"} {
		err := storageCommand([]string{sub})
		if err == nil {
			t.Fatalf("`ailang storage %s` was accepted — the D7 removal did not take", sub)
		}
		if !strings.Contains(err.Error(), "unknown storage subcommand") {
			t.Errorf("`ailang storage %s` error = %q, want it to say the subcommand is unknown", sub, err)
		}
		// The name has to appear, or the operator cannot tell which word was
		// rejected from a command line that had several.
		if !strings.Contains(err.Error(), sub) {
			t.Errorf("`ailang storage %s` error = %q, does not name %q", sub, err, sub)
		}
	}
}

// TestStorage_StatusSurvives is the other half, and the reason this is a pair:
// a removal that also broke the surviving subcommand would pass the test above.
func TestStorage_StatusSurvives(t *testing.T) {
	if err := storageCommand([]string{"--help"}); err != nil {
		t.Fatalf("`ailang storage --help` = %v, want nil", err)
	}
	// `status` is reached through the same switch; dispatching it must not fall
	// through to the unknown-subcommand arm. Its own output depends on the
	// environment's storage plane, so only the routing is asserted here.
	out := captureStdout(t, func() {
		if err := storageCommand([]string{"status"}); err != nil {
			// A plane that cannot resolve a project is a legitimate failure of
			// status on this box; what must never happen is "unknown".
			if strings.Contains(err.Error(), "unknown storage subcommand") {
				t.Fatalf("`ailang storage status` no longer routes: %v", err)
			}
		}
	})
	_ = out
}

// TestStorage_HelpDoesNotAdvertiseRemovedSubcommands is the doc-drift guard.
// Help that names a subcommand the binary rejects is the same defect class
// Phase 3 item 6 fixes in the docs, and it is the one a removal creates most
// easily — the switch and the help text are two places saying the same thing.
func TestStorage_HelpDoesNotAdvertiseRemovedSubcommands(t *testing.T) {
	help := captureStdout(t, printStorageHelp)
	for _, gone := range []string{"migrate", "verify"} {
		if strings.Contains(help, gone) {
			t.Errorf("`ailang storage --help` still advertises %q, which the binary now rejects:\n%s", gone, help)
		}
	}
	if !strings.Contains(help, "status") {
		t.Errorf("`ailang storage --help` no longer lists `status`:\n%s", help)
	}
}

// The messaging row must name the project the messaging store ACTUALLY reads.
//
// AILANG_MESSAGES_PROJECT pins the messaging store's project and wins over the
// generic resolver (resolveMessagesTarget, which is what `ailang messages`
// opens). status reported the generic project for every GCP store, so on the
// laptop 2026-09-30 it read `project ailang-multivac-dev (AILANG_CLOUD_PROJECT)`
// — the stale -dev graveyard — while `ailang messages list` was reading prod and
// said so in its own header. CLAUDE.md's session-start check tells every machine
// to confirm the messaging store with this command, so a wrong project here sends
// the reader hunting a misconfiguration that does not exist.
func TestStorageStatus_MessagingRowNamesTheMessagesProjectPin(t *testing.T) {
	t.Setenv("AILANG_STORAGE_MESSAGING", "gcp")
	t.Setenv("AILANG_CLOUD_PROJECT", "project-the-pin-overrides")
	t.Setenv("AILANG_MESSAGES_PROJECT", "project-messaging-really-reads")

	var buf strings.Builder
	if err := writeStorageStatus(&buf); err != nil {
		t.Fatalf("writeStorageStatus: %v", err)
	}
	out := buf.String()

	var msgLine string
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "messaging") {
			msgLine = line
			break
		}
	}
	if msgLine == "" {
		t.Fatalf("no messaging row in status output:\n%s", out)
	}
	if !strings.Contains(msgLine, "project-messaging-really-reads") {
		t.Errorf("messaging row does not name the AILANG_MESSAGES_PROJECT pin:\n  %s", msgLine)
	}
	if strings.Contains(msgLine, "project-the-pin-overrides") {
		t.Errorf("messaging row names the project the pin overrides:\n  %s", msgLine)
	}
	if !strings.Contains(msgLine, "AILANG_MESSAGES_PROJECT") {
		t.Errorf("messaging row does not attribute the project to AILANG_MESSAGES_PROJECT:\n  %s", msgLine)
	}
}

// With no pin, the messaging row falls back to the generic resolver like the
// other stores — the override must not invent a source that was not set.
func TestStorageStatus_MessagingRowFallsBackToTheCloudProject(t *testing.T) {
	t.Setenv("AILANG_STORAGE_MESSAGING", "gcp")
	t.Setenv("AILANG_CLOUD_PROJECT", "the-only-project-set")
	t.Setenv("AILANG_MESSAGES_PROJECT", "")

	var buf strings.Builder
	if err := writeStorageStatus(&buf); err != nil {
		t.Fatalf("writeStorageStatus: %v", err)
	}
	for _, line := range strings.Split(buf.String(), "\n") {
		if !strings.Contains(line, "messaging") {
			continue
		}
		if !strings.Contains(line, "the-only-project-set") {
			t.Errorf("messaging row does not name the resolved cloud project:\n  %s", line)
		}
		if strings.Contains(line, "AILANG_MESSAGES_PROJECT") {
			t.Errorf("messaging row credits an unset pin:\n  %s", line)
		}
		return
	}
	t.Fatal("no messaging row in status output")
}
