package main

import (
	"os"
	"path/filepath"
	"testing"
)

// M-COORDINATOR-EXECUTION-TRUST M6 (design doc V30).
//
// `ailang pi install` materialises the suite into ~/.pi/agent/extensions/ —
// "global — every repo on this machine". The AILANG repo ALSO ships
// .pi/extensions/ with the same files. pi loads both, sees two registrations of
// `ailang_check`, and exits 1 BEFORE TURN 1. Measured on the dev plane
// 2026-09-02: task-6825b3fc and task-48a365bb (workspace sunholo-data/ailang)
// both died this way; task-f8213acd (a package repo) did not.
//
// That takes out design-doc-creator, sprint-planner, sprint-evaluator and
// coordinator — the whole AILANG-repo pipeline on the pi provider.

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestWorkspaceExtensionsWinOverGlobal(t *testing.T) {
	root := t.TempDir()
	global := filepath.Join(root, "home", ".pi", "agent", "extensions")
	work := filepath.Join(root, "workspace", "task-1")

	writeFile(t, filepath.Join(global, "ailang-lsp-lite.ts"), "global copy")
	writeFile(t, filepath.Join(global, "session-protocol-gate.ts"), "global copy")
	writeFile(t, filepath.Join(global, "only-global.ts"), "global copy")
	writeFile(t, filepath.Join(work, ".pi", "extensions", "ailang-lsp-lite.ts"), "repo copy")
	writeFile(t, filepath.Join(work, ".pi", "extensions", "session-protocol-gate.ts"), "repo copy")

	removed, err := resolveExtensionCollisions(global, work)
	if err != nil {
		t.Fatalf("resolveExtensionCollisions: %v", err)
	}
	if len(removed) != 2 {
		t.Fatalf("expected 2 collisions resolved, got %d: %v", len(removed), removed)
	}

	// The colliding global copies are gone...
	for _, name := range []string{"ailang-lsp-lite.ts", "session-protocol-gate.ts"} {
		if _, err := os.Stat(filepath.Join(global, name)); !os.IsNotExist(err) {
			t.Errorf("global %s should have been removed (workspace ships its own)", name)
		}
		if _, err := os.Stat(filepath.Join(work, ".pi", "extensions", name)); err != nil {
			t.Errorf("workspace %s must survive — it is the copy that wins: %v", name, err)
		}
	}
	// ...and one the workspace does NOT ship is untouched.
	if _, err := os.Stat(filepath.Join(global, "only-global.ts")); err != nil {
		t.Errorf("a global extension with no workspace counterpart must survive: %v", err)
	}
}

// The common case: a package repo with no .pi/ of its own keeps the full global
// suite. This is what makes the gate apply everywhere (D2) — M6 must not undo it.
func TestWorkspaceWithoutExtensionsKeepsGlobalSuite(t *testing.T) {
	root := t.TempDir()
	global := filepath.Join(root, "home", ".pi", "agent", "extensions")
	work := filepath.Join(root, "workspace", "task-2")

	writeFile(t, filepath.Join(global, "session-protocol-gate.ts"), "global copy")
	writeFile(t, filepath.Join(work, "README.md"), "a package repo")

	removed, err := resolveExtensionCollisions(global, work)
	if err != nil {
		t.Fatalf("resolveExtensionCollisions: %v", err)
	}
	if len(removed) != 0 {
		t.Fatalf("no workspace extensions means no collisions, got %v", removed)
	}
	if _, err := os.Stat(filepath.Join(global, "session-protocol-gate.ts")); err != nil {
		t.Fatalf("the global suite must survive in a repo that ships none: %v", err)
	}
}

// Missing directories are the normal case on most repos and must not error —
// this runs on every dispatch, so a hard failure here would take out the fleet.
func TestMissingDirectoriesAreNotAnError(t *testing.T) {
	root := t.TempDir()
	for _, tc := range []struct{ global, work string }{
		{filepath.Join(root, "nope", "extensions"), filepath.Join(root, "w1")},
		{filepath.Join(root, "g2"), filepath.Join(root, "nope2")},
	} {
		if _, err := resolveExtensionCollisions(tc.global, tc.work); err != nil {
			t.Errorf("missing dir must be benign, got %v", err)
		}
	}
}
