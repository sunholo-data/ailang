//go:build !windows

package smt

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

const fakeSolverSleep = 30 * time.Second

func TestSolve_HardTimeout_FakeSolverIgnoringT(t *testing.T) {
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

	childPID := readChildPID(t, childPIDFile)
	assertProcessGoneWithin(t, childPID, 2*time.Second)
}

func TestZ3Version_HardTimeout(t *testing.T) {
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

	childPID := readChildPID(t, childPIDFile)
	assertProcessGoneWithin(t, childPID, 2*time.Second)
}

func writeSleepingFakeSolver(t *testing.T) (string, string) {
	t.Helper()

	dir := t.TempDir()
	childPIDFile := filepath.Join(dir, "child.pid")
	scriptPath := filepath.Join(dir, "fake-z3")
	script := fmt.Sprintf(`#!/bin/sh
sleep %d &
echo $! > %q
sleep %d
`, int(fakeSolverSleep.Seconds()), childPIDFile, int(fakeSolverSleep.Seconds()))
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake solver: %v", err)
	}
	return scriptPath, childPIDFile
}

func readChildPID(t *testing.T, path string) int {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read child pid: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("parse child pid %q: %v", data, err)
	}
	return pid
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
