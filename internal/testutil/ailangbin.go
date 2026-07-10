// Package testutil provides shared helpers for integration tests that shell
// out to a real ailang binary.
//
// Every test that runs ailang as a subprocess must locate the binary through
// this package, never via ad-hoc filepath.Join(root, "bin", "ailang") or bare
// exec.LookPath("ailang"). Both helpers guard against STALE binaries: a
// binary older than the newest Go source is skipped with an actionable
// message instead of being silently used.
//
// Why the guard exists (2026-07-10 incident): internal/pkg's local helper
// preferred bin/ailang unconditionally. A stale bin/ailang (a v0.26.0 build
// from June, while HEAD was v0.28.x) made TestRunSmokeInTempDir_Pass and
// _Isolation fail with phantom stdlib errors ("undefined variable: _io_flush
// at std/io.ail:29:36") because the on-disk stdlib referenced builtins the
// old binary didn't have. It looked like a merge regression and cost real
// debugging time.
//
// Staleness is judged by mtime (binary older than the newest *.go under cmd/
// and internal/, or than go.mod/go.sum) rather than by comparing --version to
// git describe: version stamps only change on commit, so they can't catch
// uncommitted edits, and exact-matching git describe false-positives in
// dirty/ahead worktrees. The mtime check is hermetic and catches the observed
// class (old build vs. new checkout). Embedded assets (prompts, editor
// assets) are copied under cmd/ by `make prepare-embed` immediately before
// each build, so they never postdate a fresh binary.
package testutil

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// FindAilangBinary returns the path to an ailang binary that is at least as
// new as the Go sources, preferring the repo-local bin/ailang over PATH.
// It skips the test when no binary exists or every candidate is stale.
//
// Use this when the test execs the returned path itself (or injects it into
// the code under test). When the code under test invokes a bare "ailang"
// from PATH and cannot be pointed elsewhere, use RequireAilangOnPath instead
// — otherwise this helper would vouch for a binary the test never runs.
func FindAilangBinary(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	newest := newestSource(t, root)

	var stale []string
	local := filepath.Join(root, "bin", "ailang")
	if runtime.GOOS == "windows" {
		local += ".exe"
	}
	if st, err := os.Stat(local); err == nil {
		if !st.ModTime().Before(newest.mtime) {
			return local
		}
		stale = append(stale, describeStale(local, st.ModTime()))
	}
	if path, err := exec.LookPath("ailang"); err == nil {
		if st, err := os.Stat(path); err == nil {
			if !st.ModTime().Before(newest.mtime) {
				t.Logf("using ailang from PATH (%s); bin/ailang is stale or absent", path)
				return path
			}
			stale = append(stale, describeStale(path, st.ModTime()))
		}
	}
	if len(stale) > 0 {
		t.Skipf("every ailang binary is STALE — newest Go source is %s (modified %s):\n  %s\n"+
			"Run 'make build' (bin/ailang) or 'make quick-install' (PATH). "+
			"Stale binaries cause phantom stdlib errors; see the 2026-07-10 note in internal/testutil/ailangbin.go.",
			newest.path, newest.mtime.Format(time.RFC3339), strings.Join(stale, "\n  "))
	}
	t.Skip("ailang binary not found (run 'make build' first)")
	return ""
}

// RequireAilangOnPath ensures the `ailang` resolved via PATH exists and is
// not stale, and returns its path. It is for tests whose code under test
// hardcodes exec.Command("ailang", ...) — the freshness check must apply to
// the binary that will actually run, which bin/ailang is not.
func RequireAilangOnPath(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("ailang")
	if err != nil {
		t.Skip("ailang not on PATH (run 'make quick-install' first)")
	}
	newest := newestSource(t, repoRoot(t))
	if st, err := os.Stat(path); err == nil && st.ModTime().Before(newest.mtime) {
		t.Skipf("ailang on PATH is STALE — %s, but newest Go source %s was modified %s.\n"+
			"Run 'make quick-install'. Stale binaries cause phantom stdlib errors; "+
			"see the 2026-07-10 note in internal/testutil/ailangbin.go.",
			describeStale(path, st.ModTime()), newest.path, newest.mtime.Format(time.RFC3339))
	}
	return path
}

func describeStale(path string, mtime time.Time) string {
	return fmt.Sprintf("%s (built %s)", path, mtime.Format(time.RFC3339))
}

// repoRoot walks up from the test's working directory to the directory
// containing go.mod. `go test` runs each package with cwd = the package
// source dir, so this resolves to the checkout (or worktree) under test.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("testutil: getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("testutil: no go.mod above %s — cannot locate repo root", dir)
		}
		dir = parent
	}
}

type sourceStamp struct {
	mtime time.Time // newest mtime among the binary's Go inputs
	path  string    // the file carrying that mtime, for skip messages
}

var (
	newestOnce  sync.Once
	newestCache sourceStamp
	newestErr   error
)

// newestSource returns the newest mtime among the Go inputs to the ailang
// binary: *.go under cmd/ and internal/, plus go.mod and go.sum. The scan
// runs once per test process (the root is the same for every caller).
func newestSource(t *testing.T, root string) sourceStamp {
	t.Helper()
	newestOnce.Do(func() {
		newestCache, newestErr = newestSourceUnder(root)
	})
	if newestErr != nil {
		t.Fatalf("testutil: staleness guard broken: %v", newestErr)
	}
	return newestCache
}

// newestSourceUnder is the uncached scan behind newestSource, split out so
// unit tests can point it at fixture trees.
func newestSourceUnder(root string) (sourceStamp, error) {
	var newest sourceStamp
	consider := func(path string, mtime time.Time) {
		if mtime.After(newest.mtime) {
			newest = sourceStamp{mtime: mtime, path: path}
		}
	}
	for _, f := range []string{"go.mod", "go.sum"} {
		if st, err := os.Stat(filepath.Join(root, f)); err == nil {
			consider(filepath.Join(root, f), st.ModTime())
		}
	}
	sawGo := false
	for _, dir := range []string{"cmd", "internal"} {
		base := filepath.Join(root, dir)
		err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // unreadable entries can't make the binary stale
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			sawGo = true
			consider(path, info.ModTime())
			return nil
		})
		if err != nil {
			return sourceStamp{}, fmt.Errorf("walking %s: %w", base, err)
		}
	}
	if !sawGo {
		return sourceStamp{}, fmt.Errorf("no .go files under %s/{cmd,internal} — wrong root?", root)
	}
	return newest, nil
}
