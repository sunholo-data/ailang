package eval_harness

// Tests for the generated-code memory watchdog (M-EVAL-MEM-GUARD).
//
// The allocator tests use python3 directly (not uv) so they exercise
// runGuarded with any command; they skip when python3 is not on PATH.

import (
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestEvalMaxRSSParsing(t *testing.T) {
	cases := []struct {
		env     string
		want    int64
		wantErr bool
	}{
		{"", DefaultEvalMaxRSS, false},
		{"1024", 1024, false},
		{"512M", 512 << 20, false},
		{"512MB", 512 << 20, false},
		{"8G", 8 << 30, false},
		{"8g", 8 << 30, false},
		{"16GiB", 16 << 30, false},
		{"2K", 2 << 10, false},
		{"1T", 1 << 40, false},
		{"0", 0, false},
		{"off", 0, false},
		{"OFF", 0, false},
		{"disabled", 0, false},
		{"abc", 0, true},
		{"-5", 0, true},
		{"1.5G", 0, true},
		{"G", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.env, func(t *testing.T) {
			t.Setenv(EnvEvalMaxRSS, tc.env)
			got, err := evalMaxRSS()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("evalMaxRSS(%q) = %d, want error", tc.env, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("evalMaxRSS(%q) unexpected error: %v", tc.env, err)
			}
			if got != tc.want {
				t.Errorf("evalMaxRSS(%q) = %d, want %d", tc.env, got, tc.want)
			}
		})
	}
}

func TestCategorizeRunError(t *testing.T) {
	memStderr := MemKillMarker + " process group resident memory 8.50 GiB exceeded the eval cap 8.00 GiB (AILANG_EVAL_MAX_RSS); process tree killed"

	// A memory-killed run (runtime failure + marker) banks as resource_limit.
	if got := CategorizeRunError(true, false, false, memStderr); got != ErrorCategoryResourceLimit {
		t.Errorf("mem-killed run categorized as %q, want %q", got, ErrorCategoryResourceLimit)
	}
	// Without the marker, classification is unchanged.
	if got := CategorizeRunError(true, false, false, "panic: index out of range"); got != ErrorCategoryRuntime {
		t.Errorf("runtime error categorized as %q, want %q", got, ErrorCategoryRuntime)
	}
	// A passing run never gets promoted, marker or not.
	if got := CategorizeRunError(true, true, true, memStderr); got != ErrorCategoryNone {
		t.Errorf("passing run categorized as %q, want %q", got, ErrorCategoryNone)
	}
}

func TestResourceLimitRepairHint(t *testing.T) {
	memStderr := MemKillMarker + " process group resident memory 8.50 GiB exceeded the eval cap 8.00 GiB (AILANG_EVAL_MAX_RSS); process tree killed"
	code, hint := CategorizeErrorCode(memStderr)
	if code != RES_LIMIT {
		t.Fatalf("CategorizeErrorCode = %q, want %q", code, RES_LIMIT)
	}
	if hint == nil || hint.Title == "" {
		t.Fatal("expected a non-empty repair hint for RES_LIMIT")
	}
}

func TestRunGuardedInvalidEnvFailsLoudly(t *testing.T) {
	t.Setenv(EnvEvalMaxRSS, "not-a-size")
	cmd := exec.Command("echo", "hi")
	res := runGuarded(cmd, 5*time.Second, "timed out")
	if res.ExitCode != -1 || res.RuntimeOk {
		t.Fatalf("expected failed run on invalid %s, got exit=%d runtimeOk=%v", EnvEvalMaxRSS, res.ExitCode, res.RuntimeOk)
	}
	if !strings.Contains(res.Stderr, EnvEvalMaxRSS) {
		t.Errorf("stderr should name the bad env var, got: %s", res.Stderr)
	}
}

// requirePython3 returns the python3 path or skips the test.
func requirePython3(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("memory watchdog is inert on Windows (no process-group RSS sampling)")
	}
	py, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not on PATH")
	}
	return py
}

func TestRunGuardedSmallProcessPasses(t *testing.T) {
	py := requirePython3(t)
	t.Setenv(EnvEvalMaxRSS, "1G")
	cmd := exec.Command(py, "-c", "print('ok')")
	res := runGuarded(cmd, 30*time.Second, "timed out")
	if !res.RuntimeOk || res.MemExceeded || res.TimedOut {
		t.Fatalf("small process should pass: %+v", res)
	}
	if strings.TrimSpace(res.Stdout) != "ok" {
		t.Errorf("stdout = %q, want \"ok\"", res.Stdout)
	}
}

// allocatorSrc allocates ~300MB resident and then sleeps far past the test's
// expectations, so only the watchdog (not natural exit) can end it.
const allocatorSrc = `a = "x" * (300 * 1024 * 1024)
print("allocated", flush=True)
import time
time.sleep(120)`

func TestMemWatchdogKillsAllocator(t *testing.T) {
	py := requirePython3(t)
	t.Setenv(EnvEvalMaxRSS, "64M")

	start := time.Now()
	cmd := exec.Command(py, "-c", allocatorSrc)
	res := runGuarded(cmd, 60*time.Second, "timed out")
	elapsed := time.Since(start)

	if !res.MemExceeded {
		t.Fatalf("expected MemExceeded, got: %+v (elapsed %v)", res, elapsed)
	}
	if res.RuntimeOk || res.ExitCode == 0 {
		t.Errorf("mem-killed run must not be RuntimeOk: %+v", res)
	}
	if res.TimedOut {
		t.Error("mem kill must not be reported as a timeout")
	}
	if !strings.Contains(res.Stderr, MemKillMarker) {
		t.Errorf("stderr missing %q marker: %s", MemKillMarker, res.Stderr)
	}
	// The watchdog polls every 2s; the kill must land long before the 120s
	// sleep or the 60s timeout.
	if elapsed > 30*time.Second {
		t.Errorf("watchdog took %v to kill, expected a few poll intervals", elapsed)
	}
	// Categorization end-to-end: this stderr banks as resource_limit.
	if got := CategorizeRunError(res.CompileOk, res.RuntimeOk, false, res.Stderr); got != ErrorCategoryResourceLimit {
		t.Errorf("categorized as %q, want %q", got, ErrorCategoryResourceLimit)
	}
}

// TestMemWatchdogKillsGrandchild pins the uv-shaped scenario: the process we
// start is a small wrapper and the allocation happens in ITS child. Group RSS
// accounting must see the grandchild, and the group kill must take down the
// whole tree.
func TestMemWatchdogKillsGrandchild(t *testing.T) {
	py := requirePython3(t)
	t.Setenv(EnvEvalMaxRSS, "64M")

	wrapper := `import subprocess, sys
subprocess.run([sys.executable, "-c", '''` + allocatorSrc + `'''])`

	cmd := exec.Command(py, "-c", wrapper)
	res := runGuarded(cmd, 60*time.Second, "timed out")

	if !res.MemExceeded {
		t.Fatalf("expected MemExceeded via grandchild allocation, got: %+v", res)
	}

	// The whole process group must be gone shortly after the kill — a
	// surviving grandchild is exactly the orphan-allocator failure mode.
	pid := cmd.Process.Pid
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, found := ProcessGroupRSS(pid); !found {
			return // group fully reaped
		}
		if time.Now().After(deadline) {
			t.Fatal("process group still has live members 10s after mem kill")
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func TestRunGuardedTimeoutStillWorks(t *testing.T) {
	py := requirePython3(t)
	cmd := exec.Command(py, "-c", "import time; time.sleep(60)")
	res := runGuarded(cmd, 1*time.Second, "custom timeout message")
	if !res.TimedOut || res.MemExceeded {
		t.Fatalf("expected timeout, got: %+v", res)
	}
	if res.Stderr != "custom timeout message" {
		t.Errorf("stderr = %q, want the lane's timeout message", res.Stderr)
	}
}

func TestProcessGroupRSSSamplesGroup(t *testing.T) {
	py := requirePython3(t)
	cmd := exec.Command(py, "-c", "import time; time.sleep(30)")
	SetProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = KillProcessGroup(cmd.Process.Pid)
		_ = cmd.Wait()
	}()

	// Give the interpreter a beat to map its memory, then sample.
	time.Sleep(500 * time.Millisecond)
	rss, found := ProcessGroupRSS(cmd.Process.Pid)
	if !found {
		t.Fatal("ProcessGroupRSS did not find the live process group")
	}
	if rss <= 0 {
		t.Errorf("rss = %d, want > 0", rss)
	}
}
