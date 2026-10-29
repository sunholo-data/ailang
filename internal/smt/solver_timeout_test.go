//go:build !windows

package smt

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

const fakeSolverSleep = 30 * time.Second

// maxRaceAttempts bounds how many times a hard-timeout test re-runs its scenario
// when the fake solver loses its startup race (ailang#602).
//
// The two tests do not share a budget, and the measurement below is the smaller
// one. TestSolve_HardTimeout gets max(Timeout, effectiveTimeout)+solverKillGrace
// = 3s; TestZ3Version_HardTimeout gets versionProbeTimeout = 5s outright, since
// solverKillGrace is only its WaitDelay and not part of its context deadline.
//
// Retrying alone did not fix the race: the delay was the first exec of a
// freshly written script, and under suite load that delay is correlated, so
// all three attempts lost together (2026-10-29, 3.0s each). The startup cost
// is now paid by a warm-up exec in writeSleepingFakeSolver (max 11ms
// afterwards). The retry stays as a second line. It is safe because a genuine
// process-group-kill regression fails assertProcessGoneWithin on every attempt.
const maxRaceAttempts = 3

func TestSolve_HardTimeout_FakeSolverIgnoringT(t *testing.T) {
	for attempt := 1; attempt <= maxRaceAttempts; attempt++ {
		fake, childPIDFile := writeSleepingFakeSolver(t)

		start := time.Now()
		result, err := Solve("(check-sat)\n", SolverConfig{
			Z3Path:  fake,
			Timeout: time.Second,
		})
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("Solve returned an error: %v", err)
		}
		if elapsed >= 10*time.Second {
			t.Errorf("Solve took %v, want less than 10s", elapsed)
		}
		if result.Status != StatusUnknown {
			t.Errorf("status = %v, want %v (output: %q)", result.Status, StatusUnknown, result.RawOutput)
		}
		if result.Error != "solver timeout" {
			t.Errorf("error = %q, want %q", result.Error, "solver timeout")
		}

		childPID, ok := readChildPIDIfPresent(t, childPIDFile)
		if !ok {
			t.Logf("attempt %d/%d inconclusive: fake solver killed before recording its child PID (ailang#602)", attempt, maxRaceAttempts)
			continue
		}
		assertProcessGoneWithin(t, childPID, 2*time.Second)
		return
	}
	t.Fatalf("all %d attempts lost the fake-solver startup race; child PID never recorded (ailang#602)", maxRaceAttempts)
}

func TestZ3Version_HardTimeout(t *testing.T) {
	for attempt := 1; attempt <= maxRaceAttempts; attempt++ {
		fake, childPIDFile := writeSleepingFakeSolver(t)
		t.Setenv("AILANG_Z3_PATH", fake)

		start := time.Now()
		version := Z3Version()
		elapsed := time.Since(start)

		if elapsed >= 8*time.Second {
			t.Errorf("Z3Version took %v, want less than 8s", elapsed)
		}
		if version != "" {
			t.Errorf("Z3Version = %q, want empty string", version)
		}

		childPID, ok := readChildPIDIfPresent(t, childPIDFile)
		if !ok {
			t.Logf("attempt %d/%d inconclusive: fake solver killed before recording its child PID (ailang#602)", attempt, maxRaceAttempts)
			continue
		}
		assertProcessGoneWithin(t, childPID, 2*time.Second)
		return
	}
	t.Fatalf("all %d attempts lost the fake-solver startup race; child PID never recorded (ailang#602)", maxRaceAttempts)
}

func writeSleepingFakeSolver(t *testing.T) (string, string) {
	t.Helper()

	dir := t.TempDir()
	childPIDFile := filepath.Join(dir, "child.pid")
	scriptPath := filepath.Join(dir, "fake-z3")
	script := fmt.Sprintf(`#!/bin/sh
[ "$1" = warm ] && exit 0
sleep %d &
echo $! > %q
sleep %d
`, int(fakeSolverSleep.Seconds()), childPIDFile, int(fakeSolverSleep.Seconds()))
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake solver: %v", err)
	}
	// Pay the first exec of the freshly written script before the timed run.
	// On macOS that first exec waits on the system's assessment of a new
	// executable, which `go test ./...` floods with fresh test binaries.
	// Measured 2026-10-29, spawn to pidfile during `go test -count=1
	// ./internal/...`, 60 trials each: fresh p95 2.6s / max 4.46s against a 3s
	// budget; warmed max 11ms (ailang#602).
	if out, err := exec.Command(scriptPath, "warm").CombinedOutput(); err != nil {
		t.Fatalf("warm fake solver: %v (%s)", err, out)
	}
	return scriptPath, childPIDFile
}

// readChildPIDIfPresent reports the fake solver's recorded child PID. ok is false
// when the pidfile is absent or empty, which means the fake solver was killed
// before it could record its child — an inconclusive trial, not a failure
// (ailang#602). A malformed pidfile is a real defect and fails loudly.
func readChildPIDIfPresent(t *testing.T, path string) (int, bool) {
	t.Helper()

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return 0, false
	}
	if err != nil {
		t.Fatalf("read child pid: %v", err)
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return 0, false
	}
	pid, err := strconv.Atoi(text)
	if err != nil {
		t.Fatalf("parse child pid %q: %v", data, err)
	}
	return pid, true
}

func assertProcessGoneWithin(t *testing.T, pid int, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		err := syscall.Kill(pid, 0)
		if errors.Is(err, syscall.ESRCH) {
			return
		}
		if err != nil {
			t.Fatalf("probe child process %d: %v", pid, err)
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Errorf("child process %d still exists after %v", pid, timeout)
}
