package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pkg"
)

func TestReleaseCore(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"v0.25.0", "v0.25.0"},
		{"v0.25.0-197-g46d5405d2-dirty", "v0.25.0"},
		{"0.24.0", "v0.24.0"},      // missing leading v is normalized
		{"  v0.28.0  ", "v0.28.0"}, // trimmed
		{"v0.28.0-12-gabcdef", "v0.28.0"},
		{"dev", ""}, // no release core -> skip
		{"unknown", ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := releaseCore(tc.in); got != tc.want {
			t.Errorf("releaseCore(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// writeLock writes a valid ailang.lock into dir with the given ailang_version.
func writeLock(t *testing.T, dir, ailangVersion string) {
	t.Helper()
	lf := pkg.NewLockFile(nil, "test")
	lf.AILANGVersion = ailangVersion
	if err := lf.Save(dir); err != nil {
		t.Fatalf("save lock: %v", err)
	}
}

func TestFindLockFile(t *testing.T) {
	root := t.TempDir()
	writeLock(t, root, "v0.24.0")
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}

	// Found when starting at the lock's own dir and from a nested subdir (walk up).
	if got := findLockFile(root); got == "" {
		t.Errorf("findLockFile(root) found nothing")
	}
	if got := findLockFile(nested); got == "" {
		t.Errorf("findLockFile(nested) did not walk up to the lock")
	}

	// Not found when no lock exists anywhere up the tree.
	bare := t.TempDir()
	if got := findLockFile(bare); got != "" {
		t.Errorf("findLockFile(bare) = %q, want empty", got)
	}
}

func TestSkewWarning(t *testing.T) {
	// Mismatched release cores -> warning naming both versions + the fix.
	t.Run("mismatch warns", func(t *testing.T) {
		dir := t.TempDir()
		writeLock(t, dir, "v0.24.0")
		got := skewWarning(dir, "v0.25.0-3-gabc-dirty")
		if !strings.Contains(got, "VER001") {
			t.Fatalf("expected VER001 warning, got %q", got)
		}
		for _, want := range []string{"v0.24.0", "v0.25.0", "ailang lock"} {
			if !strings.Contains(got, want) {
				t.Errorf("warning missing %q: %q", want, got)
			}
		}
	})

	// Same release core (only git suffix differs) -> silent.
	t.Run("same core is silent", func(t *testing.T) {
		dir := t.TempDir()
		writeLock(t, dir, "v0.28.0")
		if got := skewWarning(dir, "v0.28.0-40-gdeadbee"); got != "" {
			t.Errorf("expected no warning for matching core, got %q", got)
		}
	})

	// A dev binary (no release core) -> never warns, even against a released lock.
	t.Run("dev binary is silent", func(t *testing.T) {
		dir := t.TempDir()
		writeLock(t, dir, "v0.24.0")
		if got := skewWarning(dir, "dev"); got != "" {
			t.Errorf("expected no warning for dev binary, got %q", got)
		}
	})

	// A lock with no recorded ailang_version -> silent (nothing to compare).
	t.Run("empty lock version is silent", func(t *testing.T) {
		dir := t.TempDir()
		writeLock(t, dir, "")
		if got := skewWarning(dir, "v0.28.0"); got != "" {
			t.Errorf("expected no warning for empty lock version, got %q", got)
		}
	})

	// No lock at all -> silent.
	t.Run("no lock is silent", func(t *testing.T) {
		if got := skewWarning(t.TempDir(), "v0.28.0"); got != "" {
			t.Errorf("expected no warning without a lock, got %q", got)
		}
	})
}
