package testutil

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRunBounded_CapturesOutputAndExitCode(t *testing.T) {
	t.Setenv("TESTUTIL_BOUNDED_CHILD", "output")

	stdout, stderr, exitCode := RunBounded(t, "", 5*time.Second, os.Args[0], "-test.run=^TestRunBoundedChild$")
	if stdout != "child stdout\n" {
		t.Errorf("stdout = %q, want %q", stdout, "child stdout\n")
	}
	if stderr != "child stderr\n" {
		t.Errorf("stderr = %q, want %q", stderr, "child stderr\n")
	}
	if exitCode != 7 {
		t.Errorf("exitCode = %d, want 7", exitCode)
	}
}

// Deliberately ungated: the -short flag is never passed anywhere in CI, so a
// short-mode skip would be inert — the exact defect this package exists to replace.
// The test is self-bounding (2s cap, asserts elapsed < 10s), so it is always safe to run.
func TestRunBounded_KillsHungChild(t *testing.T) {
	t.Setenv("TESTUTIL_BOUNDED_CHILD", "sleep")
	started := time.Now()

	_, _, exitCode := RunBounded(t, "", 2*time.Second, os.Args[0], "-test.run=^TestRunBoundedChild$")
	elapsed := time.Since(started)
	if elapsed >= 10*time.Second {
		t.Fatalf("RunBounded took %v, want less than 10s", elapsed)
	}
	if exitCode == 0 {
		t.Fatal("RunBounded exitCode = 0, want non-zero after killing hung child")
	}
}

func TestRunBoundedChild(t *testing.T) {
	switch os.Getenv("TESTUTIL_BOUNDED_CHILD") {
	case "output":
		fmt.Fprintln(os.Stdout, "child stdout")
		fmt.Fprintln(os.Stderr, "child stderr")
		os.Exit(7)
	case "sleep":
		time.Sleep(60 * time.Second)
	default:
		t.Skip("subprocess helper")
	}
}

func TestRunBounded_UsesDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TESTUTIL_BOUNDED_CHILD", "cwd")

	stdout, stderr, exitCode := RunBounded(t, dir, 5*time.Second, os.Args[0], "-test.run=^TestRunBoundedDirectoryChild$")
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0; stderr: %s", exitCode, stderr)
	}
	got := strings.SplitN(strings.TrimSpace(stdout), "\n", 2)[0]
	if runtime.GOOS == "windows" {
		got = strings.ToLower(got)
		dir = strings.ToLower(dir)
	}
	if got != dir {
		t.Fatalf("child cwd = %q, want %q", got, dir)
	}
}

func TestRunBoundedDirectoryChild(t *testing.T) {
	if os.Getenv("TESTUTIL_BOUNDED_CHILD") != "cwd" {
		t.Skip("subprocess helper")
	}
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprintln(os.Stdout, dir)
}
