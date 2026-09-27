package riglock

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/proctree"
)

// deadPID is above the macOS/Linux pid ceiling, so it can never name a live
// process. Matches the constant style used in riglock_test.go.
const deadPID = "999999999"

// seedHeldLock creates a lock directory with a fresh mtime and the given
// holder line, standing in for a lock another process took.
func seedHeldLock(t *testing.T, dir, holder string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("seed lock: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "holder"), []byte(holder), 0o644); err != nil {
		t.Fatalf("seed holder: %v", err)
	}
}

// sleeper starts a real child process that outlives the test body unless
// killed, standing in for an orphaned agent still on the GPU. It returns the
// command (so a test can Wait on how it died), its pid, and the binary path it
// was started from.
func sleeper(t *testing.T) (*exec.Cmd, int, string) {
	t.Helper()
	bin, err := exec.LookPath("sleep")
	if err != nil {
		t.Skipf("sleep not available: %v", err)
	}
	cmd := exec.Command(bin, "60")
	// Its own process-group leader, exactly as every agent we spawn is
	// (proctree.Configure), so the reap exercises the real KillGroup path.
	proctree.SetGroup(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait() // already-waited is fine; this is best-effort teardown
	})
	return cmd, cmd.Process.Pid, bin
}

// TestAcquire_RefusesStealWhileRegisteredChildLives is the regression test for
// the two-tenant bug: a supervisor killed outright leaves its agent running on
// the GPU, and the lock must not read as free.
func TestAcquire_RefusesStealWhileRegisteredChildLives(t *testing.T) {
	dir := isolate(t)
	seedHeldLock(t, dir, deadPID+" 2026-09-22T09:53:09Z")

	_, pid, bin := sleeper(t)
	writeChildren(t, dir, pid, filepath.Base(bin))

	ok, _, err := Acquire(NoWait)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if ok {
		t.Fatal("stole the lock while a registered GPU child was still running — " +
			"this admits a second tenant onto a single-GPU rig")
	}
}

// TestAcquire_StealsAndReapsOrphanedChild covers the recovery half: once the
// steal does go ahead, the orphan must be killed rather than left on the rig.
func TestAcquire_StealsAndReapsOrphanedChild(t *testing.T) {
	dir := isolate(t)

	// A lock whose mtime is old enough to be stale AND whose holder is dead.
	t.Setenv(EnvStaleMin, "1") // 1-minute staleness window
	seedHeldLock(t, dir, deadPID+" 2026-09-22T09:53:09Z")

	cmd, pid, bin := sleeper(t)
	writeChildren(t, dir, pid, filepath.Base(bin))

	// Backdate past the window — after seeding, since writing files inside the
	// directory bumps its mtime back to now.
	old := time.Now().Add(-10 * time.Minute)
	if err := os.Chtimes(dir, old, old); err != nil {
		t.Fatalf("backdate: %v", err)
	}

	ok, release, err := Acquire(NoWait)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if !ok {
		t.Fatal("expected to steal a stale lock whose holder is dead")
	}
	defer release()

	// This test process is the child's parent, so a SIGKILLed child lingers as
	// a zombie until we Wait for it and pidAlive would still report it alive.
	// Assert on how it died instead. A real orphan is reparented to launchd,
	// which reaps it immediately.
	if err := cmd.Wait(); err == nil {
		t.Error("orphaned child exited cleanly — it was not reaped by the steal")
	}
}

// TestLiveChildren_IgnoresRecycledPID is the PID-reuse guard. A recorded PID
// now running something else must count as gone, so a stranger neither holds
// the rig hostage nor gets SIGKILLed by reapChildren.
func TestLiveChildren_IgnoresRecycledPID(t *testing.T) {
	dir := isolate(t)
	seedHeldLock(t, dir, deadPID+" 2026-09-22T09:53:09Z")

	_, pid, _ := sleeper(t)
	// Register the live pid under a DIFFERENT binary name, as if the pid had
	// been recycled since registration.
	writeChildren(t, dir, pid, "opencode")

	if got := liveChildren(dir); len(got) != 0 {
		t.Fatalf("liveChildren = %v, want none (pid runs a different binary)", got)
	}

	ok, release, err := Acquire(NoWait)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if !ok {
		t.Fatal("expected to steal: the recorded child pid belongs to something else now")
	}
	defer release()

	if !pidAlive(pid) {
		t.Error("reaped an unrelated process that merely reused a recorded PID")
	}
}

// TestRegisterChild_NoopWithoutHeldLock keeps the call safe for every executor:
// a run that never took the rig lock must write nothing.
func TestRegisterChild_NoopWithoutHeldLock(t *testing.T) {
	dir := isolate(t)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// EnvHeld is cleared by isolate, so this process holds no lock.
	RegisterChild(4242, "/usr/bin/opencode")

	if _, err := os.Stat(filepath.Join(dir, childrenFile)); !os.IsNotExist(err) {
		t.Error("RegisterChild wrote a children file without a held rig lock")
	}
}

// TestRegisterChild_RecordsUnderHeldLock is the round trip through the real
// Acquire path: the entry lands and reads back as live.
func TestRegisterChild_RecordsUnderHeldLock(t *testing.T) {
	dir := isolate(t)

	ok, release, err := Acquire(NoWait)
	if err != nil || !ok {
		t.Fatalf("Acquire: ok=%v err=%v", ok, err)
	}
	defer release()

	_, pid, bin := sleeper(t)
	RegisterChild(pid, bin) // Acquire set EnvHeld=1

	live := liveChildren(dir)
	if len(live) != 1 || live[0].pid != pid {
		t.Fatalf("liveChildren = %v, want one entry for pid %d", live, pid)
	}
}

// writeChildren seeds the children record directly, bypassing the EnvHeld
// guard that RegisterChild applies.
func writeChildren(t *testing.T, dir string, pid int, name string) {
	t.Helper()
	line := strconv.Itoa(pid) + " " + name + "\n"
	if err := os.WriteFile(filepath.Join(dir, childrenFile), []byte(line), 0o644); err != nil {
		t.Fatalf("seed children: %v", err)
	}
}
