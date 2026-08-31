package coordinator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// M-MESSAGE-PLANE-FAIL-LOUD M1 (decision D1: fatal exit).
//
// A daemon that cannot initialize task processing used to log ONE Warning and
// then run forever in "standby mode (no message processing)". Standby is
// indistinguishable from idle from the outside: `coordinator status` reports
// "running", the health port answers, and the log prints "Checking for new
// tasks..." on every tick.
//
// Measured 2026-08-26 on the rig: 20 standby startups against 11 healthy ones,
// first on 2026-05-25. The cause was that initTaskProcessing builds a default
// worktree manager with an empty repo path, which resolves the git root from
// CWD — and launchd's WorkingDirectory was $HOME. The same binary was healthy
// when launched from a terminal inside the repo and inert under launchd, which
// is why a manual end-to-end verification could pass while production claimed
// nothing for three months.
//
// D1: fail loudly. launchd's KeepAlive restarts it and ThrottleInterval bounds
// the loop, so a misconfigured daemon crash-loops visibly instead of idling.
func TestRun_FailsLoudlyWhenTaskProcessingCannotInit(t *testing.T) {
	// initTaskProcessing resolves the default worktree manager's repo from CWD.
	// Point CWD at a directory that is not a git repository to reproduce exactly
	// the launchd-in-$HOME condition.
	// The daemon loads its agent registry from AILANG_CONFIG, falling back to
	// ~/.ailang/config.yaml — the DEVELOPER'S REAL CONFIG. Without this the test
	// asserts on whatever agents that machine happens to declare, and it was
	// measured green on CI and red on a workstation for exactly that reason: a
	// local `coordinator` agent whose workspace is a real repo gives
	// initTaskProcessing a worktree manager, so the failing fallback this test
	// depends on is never reached and Run() proceeds instead of bailing out.
	// Point it at an empty file so the reproduction is the launchd condition and
	// nothing else.
	t.Setenv("AILANG_CONFIG", filepath.Join(t.TempDir(), "empty-config.yaml"))

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	notARepo := t.TempDir()
	if err := os.Chdir(notARepo); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWD) })

	tmpDir := t.TempDir()
	cfg := &Config{
		PollInterval: time.Second,
		MaxWorktrees: 2,
		LogFile:      filepath.Join(tmpDir, "logs", "coordinator.log"),
		PIDFile:      filepath.Join(tmpDir, "state", "coordinator.pid"),
		StateDir:     filepath.Join(tmpDir, "state"),
	}

	daemon, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("failed to create daemon: %v", err)
	}
	defer daemon.Close()

	// Bounded: pre-fix, Run() does not return at all on an init failure — it
	// falls through to the poll loop and blocks forever, which is the defect
	// itself. Assert on a deadline so the regression reads as a failure rather
	// than a hung test.
	errCh := make(chan error, 1)
	go func() { errCh <- daemon.Run() }()

	var runErr error
	select {
	case runErr = <-errCh:
	case <-time.After(10 * time.Second):
		daemon.cancel() // unblock the goroutine so the test binary can exit
		t.Fatal("Run() did not return within 10s on an init failure: the daemon is reachable but cannot process messages — this is the three-month standby regression")
	}

	if runErr == nil {
		t.Fatal("Run() returned nil on an init failure: a daemon that cannot process messages must not report success")
	}
	// The message must name the fix, not just the failure: an operator reading
	// it in a launchd log has no other context.
	if !strings.Contains(runErr.Error(), "task processing") {
		t.Errorf("error should name what could not initialize; got: %v", runErr)
	}
}

// Guard: the standby code path must not come back. This matches the EMITTED log
// literal, not the phrase — the surrounding comment necessarily discusses standby
// at length, and a guard that trips on its own documentation is a guard nobody
// keeps.
func TestRun_NoStandbyModeRemains(t *testing.T) {
	src, err := os.ReadFile("daemon.go")
	if err != nil {
		t.Fatalf("read daemon.go: %v", err)
	}
	// The old path was: d.logger.Println("Daemon running in standby mode ...")
	if strings.Contains(string(src), `Println("Daemon running in standby`) {
		t.Error("daemon.go still logs the standby path; D1 ratified fatal exit instead")
	}
}
