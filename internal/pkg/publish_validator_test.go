package pkg

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// findAilangBinary locates a usable ailang binary for the smoke runner. We
// prefer the freshly-built bin/ailang (relative to the repo root) so tests
// always exercise current code rather than whatever happens to be in PATH.
func findAilangBinary(t *testing.T) string {
	t.Helper()
	// Walk up to find the repo root (contains go.mod with module sunholo-data/ailang).
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(dir, "bin", "ailang")
		if runtime.GOOS == "windows" {
			candidate += ".exe"
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	if path, err := exec.LookPath("ailang"); err == nil {
		return path
	}
	t.Skip("ailang binary not found (run `make build` first)")
	return ""
}

func writeMinimalPackage(t *testing.T, dir string, smokeBody string) {
	t.Helper()
	manifest := `[package]
name = "test/smoke"
version = "0.1.0"
edition = "1"

[exports]
modules = ["test/smoke"]
`
	if err := os.WriteFile(filepath.Join(dir, ManifestFile), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	core := `module test/smoke

export func ping() -> int = 42
`
	if err := os.WriteFile(filepath.Join(dir, "smoke.ail"), []byte(core), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, SmokeFile), []byte(smokeBody), 0644); err != nil {
		t.Fatal(err)
	}
}

// TestRunSmokeInTempDir_Pass asserts a well-formed _smoke.ail is reported as
// passing. The smoke body just imports the local module and calls a pure func.
func TestRunSmokeInTempDir_Pass(t *testing.T) {
	bin := findAilangBinary(t)

	pkgDir := t.TempDir()
	writeMinimalPackage(t, pkgDir, `module _smoke

import std/io (println)

export func main() -> () ! {IO} =
  println("smoke ok")
`)

	res, err := RunSmokeInTempDir(pkgDir, bin, 30*time.Second)
	if err != nil {
		t.Fatalf("RunSmokeInTempDir: %v", err)
	}
	if !res.Passed {
		t.Errorf("expected smoke to pass, got exit=%d output=\n%s", res.ExitCode, res.Output)
	}
	if res.TimedOut {
		t.Error("did not expect timeout")
	}
}

// TestRunSmokeInTempDir_Fail asserts that a smoke whose main() crashes fails.
func TestRunSmokeInTempDir_Fail(t *testing.T) {
	bin := findAilangBinary(t)

	pkgDir := t.TempDir()
	// Reading a path that doesn't exist will return Err — but the smoke uses
	// the panicking variant readFile to force a hard crash, mirroring the
	// "extension call panics in empty workdir" failure mode.
	writeMinimalPackage(t, pkgDir, `module _smoke

import std/fs (readFile)

export func main() -> () ! {FS} =
  let _ = readFile("definitely-not-here.txt") in
  ()
`)

	res, err := RunSmokeInTempDir(pkgDir, bin, 30*time.Second)
	if err != nil {
		t.Fatalf("RunSmokeInTempDir: %v", err)
	}
	if res.Passed {
		t.Errorf("expected smoke to fail, but Passed=true. output=\n%s", res.Output)
	}
	if res.Output == "" {
		t.Error("expected captured output for failing smoke")
	}
}

// TestRunSmokeInTempDir_Timeout asserts that a smoke that runs longer than
// the timeout is killed and reported as TimedOut.
func TestRunSmokeInTempDir_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timeout test in -short mode")
	}
	bin := findAilangBinary(t)

	pkgDir := t.TempDir()
	// Sleep for 60s — far longer than the 1s deadline below, which should
	// kill the spawned process well before sleep returns.
	writeMinimalPackage(t, pkgDir, `module _smoke

import std/clock (sleep)

export func main() -> () ! {Clock} = sleep(60000)
`)

	start := time.Now()
	res, err := RunSmokeInTempDir(pkgDir, bin, 1*time.Second)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("RunSmokeInTempDir: %v", err)
	}
	if !res.TimedOut {
		t.Errorf("expected timeout, got Passed=%v ExitCode=%d output=\n%s", res.Passed, res.ExitCode, res.Output)
	}
	if elapsed > 5*time.Second {
		t.Errorf("smoke didn't terminate promptly: took %v (expected ~1s)", elapsed)
	}
}

// TestRunSmokeInTempDir_Isolation asserts the smoke runs from a temp dir, so
// it cannot read files that exist alongside the package source.
func TestRunSmokeInTempDir_Isolation(t *testing.T) {
	bin := findAilangBinary(t)

	pkgDir := t.TempDir()
	writeMinimalPackage(t, pkgDir, `module _smoke

import std/fs (fileExists)
import std/io (println)

export func main() -> () ! {FS, IO} =
  if fileExists("secret.txt") then println("LEAK") else println("isolated")
`)

	// Drop a file alongside the package source. If the smoke ran in pkgDir,
	// it would see the file and print "LEAK". Because we copy only the
	// publishable file set into a fresh temp dir, secret.txt is invisible.
	if err := os.WriteFile(filepath.Join(pkgDir, "secret.txt"), []byte("nope"), 0644); err != nil {
		t.Fatal(err)
	}

	res, err := RunSmokeInTempDir(pkgDir, bin, 30*time.Second)
	if err != nil {
		t.Fatalf("RunSmokeInTempDir: %v", err)
	}
	if !res.Passed {
		t.Errorf("expected smoke to pass, output=\n%s", res.Output)
	}
	if strings.Contains(res.Output, "LEAK") {
		t.Errorf("smoke leaked across isolation boundary; output=\n%s", res.Output)
	}
	if !strings.Contains(res.Output, "isolated") {
		t.Errorf("expected 'isolated' marker in output, got=\n%s", res.Output)
	}
}

// TestRunSmokeInTempDir_RespectsCustomTimeout asserts that callers can pass
// a short timeout (e.g. derived from [smoke].timeout_seconds in ailang.toml)
// and the runner honours it. Pairs with TestRunSmokeInTempDir_Timeout which
// exercises the default 30s case.
func TestRunSmokeInTempDir_RespectsCustomTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timeout test in -short mode")
	}
	bin := findAilangBinary(t)

	pkgDir := t.TempDir()
	writeMinimalPackage(t, pkgDir, `module _smoke

import std/clock (sleep)

export func main() -> () ! {Clock} = sleep(30000)
`)

	// A 500ms timeout against a 30s sleep — verifies the per-call timeout
	// argument is the controlling knob, not just DefaultSmokeTimeout.
	start := time.Now()
	res, err := RunSmokeInTempDir(pkgDir, bin, 500*time.Millisecond)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("RunSmokeInTempDir: %v", err)
	}
	if !res.TimedOut {
		t.Errorf("expected timeout, got Passed=%v ExitCode=%d", res.Passed, res.ExitCode)
	}
	// Custom 500ms timeout should fire well under the default 30s. Give a
	// generous ceiling for slow CI but reject "took the full default".
	if elapsed > 5*time.Second {
		t.Errorf("custom timeout not honoured: took %v (expected ~500ms)", elapsed)
	}
}

// TestRunSmokeInTempDir_NoSmokeFile reports an infrastructure error when the
// package has no _smoke.ail. (The publish wrapper in cmd/ailang interprets
// this as "no smoke = warn but allow", but the runner itself draws a
// distinction between absent and failing.)
func TestRunSmokeInTempDir_NoSmokeFile(t *testing.T) {
	pkgDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(pkgDir, ManifestFile), []byte(`[package]
name = "test/empty"
version = "0.1.0"
edition = "1"
`), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := RunSmokeInTempDir(pkgDir, "ailang", 30*time.Second)
	if err == nil {
		t.Error("expected error when _smoke.ail is missing")
	}
}
