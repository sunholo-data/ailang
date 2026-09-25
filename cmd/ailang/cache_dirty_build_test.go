package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// TestCompileCache_DirtyBuildsDoNotShareVerdicts is #1275 as a test. Two
// builds of ONE commit used to share cache keys, so a rebuilt (fixed) type
// checker was served its predecessor's verdicts: `check` printed "No errors"
// for a program it rejects cold, and `run` executed it.
//
// A dirty build's identity now includes an executable fingerprint, so a
// rebuilt binary misses (it recompiles), while a CLEAN build (same commit, no
// dirty marker) still hits across copies of itself. `cp` stands in for a
// rebuild: same bytes and a new file, which is exactly what `go install` does
// to the fingerprint.
func TestCompileCache_DirtyBuildsDoNotShareVerdicts(t *testing.T) {
	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	build := func(version string) string {
		bin := filepath.Join(t.TempDir(), "ailang")
		ldflags := "-X github.com/sunholo-data/ailang/internal/version.Commit=REPRO" +
			" -X github.com/sunholo-data/ailang/internal/version.Version=" + version
		// -buildvcs=false: the test controls "dirty" through Version alone,
		// whatever state the working tree is in.
		_, stderr, code := testutil.RunBounded(t, projectRoot, 180*time.Second,
			"go", "build", "-buildvcs=false", "-ldflags", ldflags, "-o", bin, "./cmd/ailang")
		if code != 0 {
			t.Fatalf("go build: %s", stderr)
		}
		return bin
	}
	copyBin := func(src string) string {
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond) // distinct mtime even on coarse clocks
		dst := filepath.Join(t.TempDir(), "ailang")
		if err := os.WriteFile(dst, data, 0o755); err != nil {
			t.Fatal(err)
		}
		return dst
	}
	// check runs `ailang check --debug-compile` in dir and returns the
	// [CACHE] verdict for the module (SKIP = served from cache).
	check := func(bin, dir string, env ...string) string {
		ctx, cancel := testutil.HangGuardContext(t, 60*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, bin, "check", "--debug-compile", "--relax-modules", "m.ail")
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), append([]string{"AILANG_NO_CACHE="}, env...)...)
		out, err := cmd.CombinedOutput()
		// A crash or timeout must not read as "no cache line" (evaluator
		// finding): the check itself has to succeed.
		if err != nil || !strings.Contains(string(out), "No errors found") {
			t.Fatalf("check failed (err=%v):\n%s", err, out)
		}
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(line, "[CACHE] m:") {
				return strings.TrimSpace(strings.TrimPrefix(line, "[CACHE] m:"))
			}
		}
		return "none"
	}
	project := func() string {
		dir := t.TempDir()
		src := "module m\n\nexport func main() -> int = 1 + 2\n"
		if err := os.WriteFile(filepath.Join(dir, "m.ail"), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		return dir
	}

	t.Run("dirty rebuild misses", func(t *testing.T) {
		dir := project()
		a := build("v9.9.9-dirty")
		if v := check(a, dir); v != "MISS" {
			t.Fatalf("first check: %q, want MISS", v)
		}
		if v := check(a, dir); !strings.HasPrefix(v, "SKIP") {
			t.Fatalf("same binary again: %q, want SKIP (a hit)", v)
		}
		if v := check(copyBin(a), dir); v != "MISS" {
			t.Fatalf("rebuilt dirty binary: %q, want MISS; it was served the other build's entry", v)
		}
	})

	t.Run("clean build still hits across copies", func(t *testing.T) {
		dir := project()
		a := build("v9.9.9")
		check(a, dir)
		if v := check(copyBin(a), dir); !strings.HasPrefix(v, "SKIP") {
			t.Fatalf("clean build copy: %q, want SKIP (clean identity is the commit)", v)
		}
	})

	t.Run("AILANG_NO_CACHE is honored by check", func(t *testing.T) {
		dir := project()
		a := build("v9.9.9")
		check(a, dir)
		if v := check(a, dir, "AILANG_NO_CACHE=1"); v != "none" {
			t.Fatalf("check with AILANG_NO_CACHE=1 consulted the cache: %q", v)
		}
	})
}
