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
