package local

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSecureBrowserRootCreatesOwnerOnlyDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "ailang-browser")

	if err := secureBrowserRoot(root); err != nil {
		t.Fatalf("secureBrowserRoot: %v", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		t.Fatalf("stat root: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("root is not a directory")
	}
	if runtime.GOOS != "windows" {
		if perm := info.Mode().Perm(); perm != browserRootMode {
			t.Fatalf("root mode = %04o, want %04o", perm, browserRootMode)
		}
	}

	// Every provider construction runs through here, so a second call must be a
	// no-op rather than an error.
	if err := secureBrowserRoot(root); err != nil {
		t.Fatalf("second secureBrowserRoot: %v", err)
	}
}

// The case MkdirAll alone cannot handle: the directory already exists, so
// MkdirAll returns nil WITHOUT re-applying the mode. Left alone, an
// authenticated session's Chromium profile would sit in a world-readable
// directory.
func TestSecureBrowserRootTightensAPreExistingLooseDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not meaningful on Windows")
	}
	root := filepath.Join(t.TempDir(), "ailang-browser")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("fixture: %v", err)
	}

	// Control: MkdirAll accepts this happily, which is exactly the gap.
	if err := os.MkdirAll(root, browserRootMode); err != nil {
		t.Fatalf("expected MkdirAll to accept an existing 0755 dir, got %v", err)
	}
	if perm := mustPerm(t, root); perm != 0o755 {
		t.Fatalf("control failed: MkdirAll changed the mode to %04o", perm)
	}

	if err := secureBrowserRoot(root); err != nil {
		t.Fatalf("secureBrowserRoot: %v", err)
	}
	if perm := mustPerm(t, root); perm != browserRootMode {
		t.Fatalf("root mode = %04o after tightening, want %04o", perm, browserRootMode)
	}
}

// A symlink at the predictable path is the other half of the attack: MkdirAll
// follows it and reports success, so session state lands wherever it points.
func TestSecureBrowserRootRejectsASymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation needs elevation on Windows")
	}
	base := t.TempDir()
	elsewhere := filepath.Join(base, "attacker-owned")
	if err := os.MkdirAll(elsewhere, browserRootMode); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	root := filepath.Join(base, "ailang-browser")
	if err := os.Symlink(elsewhere, root); err != nil {
		t.Fatalf("symlink fixture: %v", err)
	}

	// Control: MkdirAll is happy to follow it.
	if err := os.MkdirAll(root, browserRootMode); err != nil {
		t.Fatalf("expected MkdirAll to accept the symlink, got %v", err)
	}

	err := secureBrowserRoot(root)
	if err == nil {
		t.Fatalf("a symlinked browser root was accepted")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("error %q does not name the symlink", err)
	}
}

func TestSecureBrowserRootRejectsAFile(t *testing.T) {
	root := filepath.Join(t.TempDir(), "ailang-browser")
	if err := os.WriteFile(root, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("fixture: %v", err)
	}

	if err := secureBrowserRoot(root); err == nil {
		t.Fatalf("a regular file was accepted as the browser root")
	}
}

func TestSecureBrowserRootRejectsAnEmptyPath(t *testing.T) {
	if err := secureBrowserRoot(""); err == nil {
		t.Fatalf("an empty browser root path was accepted")
	}
}

// New must surface an unsafe root as a structured provisioning failure rather
// than continuing with it.
func TestNewRefusesAnUnsafeBrowserRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation needs elevation on Windows")
	}
	base := t.TempDir()
	elsewhere := filepath.Join(base, "attacker-owned")
	if err := os.MkdirAll(elsewhere, browserRootMode); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	root := filepath.Join(base, "ailang-browser")
	if err := os.Symlink(elsewhere, root); err != nil {
		t.Fatalf("symlink fixture: %v", err)
	}

	provider, err := New(Config{BaseDir: root, NpxPath: "/usr/bin/true"})
	if err == nil {
		t.Fatalf("New accepted a symlinked browser root")
	}
	if provider != nil {
		t.Fatalf("New returned a provider alongside an error")
	}
}

func mustPerm(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("lstat %s: %v", path, err)
	}
	return info.Mode().Perm()
}
