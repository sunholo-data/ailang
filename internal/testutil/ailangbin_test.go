package testutil

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeFileAt creates path (and parents) with the given mtime.
func writeFileAt(t *testing.T, path string, mtime time.Time) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}
}

// TestNewestSourceUnder_PicksNewestGoFile asserts the scan finds the newest
// *.go across nested cmd/ and internal/ dirs and reports which file it was.
func TestNewestSourceUnder_PicksNewestGoFile(t *testing.T) {
	root := t.TempDir()
	base := time.Now().Add(-24 * time.Hour).Truncate(time.Second)

	writeFileAt(t, filepath.Join(root, "go.mod"), base)
	writeFileAt(t, filepath.Join(root, "cmd", "ailang", "main.go"), base.Add(1*time.Hour))
	newestPath := filepath.Join(root, "internal", "types", "deep", "infer.go")
	writeFileAt(t, newestPath, base.Add(3*time.Hour))
	// Non-Go files must not count, even when newest overall.
	writeFileAt(t, filepath.Join(root, "internal", "types", "notes.md"), base.Add(9*time.Hour))

	got, err := newestSourceUnder(root)
	if err != nil {
		t.Fatalf("newestSourceUnder: %v", err)
	}
	if got.path != newestPath {
		t.Errorf("newest path = %s, want %s", got.path, newestPath)
	}
	if !got.mtime.Equal(base.Add(3 * time.Hour)) {
		t.Errorf("newest mtime = %v, want %v", got.mtime, base.Add(3*time.Hour))
	}
}

// TestNewestSourceUnder_GoModCounts asserts go.mod/go.sum participate — a
// dependency bump without .go edits still makes an old binary stale.
func TestNewestSourceUnder_GoModCounts(t *testing.T) {
	root := t.TempDir()
	base := time.Now().Add(-24 * time.Hour).Truncate(time.Second)

	writeFileAt(t, filepath.Join(root, "internal", "a", "a.go"), base)
	gomod := filepath.Join(root, "go.mod")
	writeFileAt(t, gomod, base.Add(2*time.Hour))

	got, err := newestSourceUnder(root)
	if err != nil {
		t.Fatalf("newestSourceUnder: %v", err)
	}
	if got.path != gomod {
		t.Errorf("newest path = %s, want %s", got.path, gomod)
	}
}

// TestNewestSourceUnder_FailsLoudlyOnWrongRoot asserts the guard refuses to
// vouch for freshness when the tree has no Go sources at all — a silent zero
// time would make every binary look fresh.
func TestNewestSourceUnder_FailsLoudlyOnWrongRoot(t *testing.T) {
	root := t.TempDir()
	writeFileAt(t, filepath.Join(root, "go.mod"), time.Now())

	if _, err := newestSourceUnder(root); err == nil {
		t.Error("expected error for a root with no .go files, got nil")
	}
}

// TestFindAilangBinary_SelfCheck runs the real helper against this repo: if
// it returns a path, that binary must exist and be no older than the newest
// Go source (the exact invariant the 2026-07-10 incident violated).
func TestFindAilangBinary_SelfCheck(t *testing.T) {
	bin := FindAilangBinary(t) // skips when absent/stale — that path is fine
	st, err := os.Stat(bin)
	if err != nil {
		t.Fatalf("returned binary does not stat: %v", err)
	}
	newest := newestSource(t, repoRoot(t))
	if st.ModTime().Before(newest.mtime) {
		t.Errorf("FindAilangBinary returned a stale binary: %s (built %v) older than %s (%v)",
			bin, st.ModTime(), newest.path, newest.mtime)
	}
}
