package riglock

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// The handoff marker is a CONTRACT between three implementations — this
// package, tools/launchd/rig-lock.sh, and daneel's tools/daneel — that must all
// read and write the same line. A format drift would not fail loudly; it would
// look exactly like "nobody ever asks for a yield", which is the silent
// starvation this whole mechanism exists to end. So the contract is tested
// across the language boundary, not just within Go.

func shellFixture(t *testing.T) (lockDir, handoff, script string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the shell half of the protocol is POSIX-only")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the test file")
	}
	repo := filepath.Join(filepath.Dir(thisFile), "..", "..")
	script = filepath.Join(repo, "tools", "launchd", "rig-lock.sh")
	if _, err := os.Stat(script); err != nil {
		t.Skipf("rig-lock.sh not found: %v", err)
	}
	dir := t.TempDir()
	lockDir = filepath.Join(dir, "rig.lock.d")
	handoff = filepath.Join(dir, "rig.handoff")
	t.Setenv(EnvLockDir, lockDir)
	t.Setenv(EnvHandoffFile, handoff)
	t.Setenv(EnvHeld, "")
	return lockDir, handoff, script
}

// runShell evaluates body with rig-lock.sh sourced and the fixture's paths set.
func runShell(t *testing.T, script, lockDir, handoff, body string) (string, error) {
	t.Helper()
	cmd := exec.Command("bash", "-c", "source "+script+"\n"+body)
	cmd.Env = append(os.Environ(),
		"RIG_LOCK_DIR="+lockDir,
		"RIG_HANDOFF_FILE="+handoff,
	)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// Format first, liveness second. A marker written by a shell that has since
// exited is CORRECTLY discarded (dead requester), so reading one through
// PendingYield cannot tell a format drift from a working expiry rule. Parse the
// raw line to test the format on its own.
func TestHandoff_ShellMarkerFormat(t *testing.T) {
	lockDir, handoff, script := shellFixture(t)

	if out, err := runShell(t, script, lockDir, handoff,
		`rig_lock_request_yield daneel-intake 120 || { echo "request failed"; exit 1; }`); err != nil {
		t.Fatalf("shell request: %v (%s)", err, out)
	}
	raw, err := os.ReadFile(handoff)
	if err != nil {
		t.Fatalf("shell wrote no marker: %v", err)
	}
	y, err := parseYield(string(raw))
	if err != nil {
		t.Fatalf("Go cannot parse a marker the shell wrote (%q): %v", strings.TrimSpace(string(raw)), err)
	}
	if y.Requester != "daneel-intake" {
		t.Errorf("requester = %q, want daneel-intake", y.Requester)
	}
	if y.PID <= 0 {
		t.Errorf("pid = %d, want the shell's pid", y.PID)
	}
	if remaining := time.Until(y.Until); remaining <= 0 || remaining > 121*time.Second {
		t.Errorf("until is %v away, want a little under 120s", remaining)
	}
}

// And end to end, with the requester still alive — which is the real shape:
// daneel writes the marker and then waits for the gap in the same process.
func TestHandoff_ShellWritesGoReads(t *testing.T) {
	lockDir, handoff, script := shellFixture(t)

	cmd := exec.Command("bash", "-c", "source "+script+`
rig_lock_request_yield daneel-intake 120 || exit 1
echo ready
sleep 30`)
	cmd.Env = append(os.Environ(), "RIG_LOCK_DIR="+lockDir, "RIG_HANDOFF_FILE="+handoff)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()
	buf := make([]byte, len("ready\n"))
	if _, err := io.ReadFull(stdout, buf); err != nil {
		t.Fatalf("requester never signalled ready: %v", err)
	}

	y, ok := PendingYield()
	if !ok {
		t.Fatal("Go could not read a LIVE handoff the shell wrote — the marker format has drifted")
	}
	if y.Requester != "daneel-intake" {
		t.Errorf("requester = %q, want daneel-intake", y.Requester)
	}
	if y.PID != cmd.Process.Pid {
		t.Errorf("pid = %d, want the requester's pid %d", y.PID, cmd.Process.Pid)
	}
}

func TestHandoff_GoWritesShellReads(t *testing.T) {
	lockDir, handoff, script := shellFixture(t)

	if err := RequestYield("eval-suite", 2*time.Minute); err != nil {
		t.Fatalf("RequestYield: %v", err)
	}

	// An anonymous caller (the background filler) must see it as in force...
	out, err := runShell(t, script, lockDir, handoff,
		`rig_yield_pending && echo PENDING || echo ABSENT`)
	if err != nil {
		t.Fatalf("shell: %v (%s)", err, out)
	}
	if out != "PENDING" {
		t.Errorf("shell read %q, want PENDING — it cannot see a handoff Go wrote", out)
	}

	// ...and must therefore be refused the lock.
	out, err = runShell(t, script, lockDir, handoff,
		`rig_lock_acquire nowait && echo ACQUIRED || echo REFUSED`)
	if err != nil {
		t.Fatalf("shell: %v (%s)", err, out)
	}
	if out != "REFUSED" {
		t.Errorf("filler got %q, want REFUSED — it would race into the holder's gap", out)
	}

	// The named requester is exactly who the gap is for.
	out, err = runShell(t, script, lockDir, handoff,
		`rig_lock_acquire nowait eval-suite && echo ACQUIRED || echo REFUSED`)
	if err != nil {
		t.Fatalf("shell: %v (%s)", err, out)
	}
	if out != "ACQUIRED" {
		t.Errorf("named requester got %q, want ACQUIRED", out)
	}
}

// The whole mechanism, across the boundary: Go holds the lock and checkpoints
// between units of work; a shell requester asks, gets in, and hands it back.
func TestHandoff_GoHolderYieldsToShellRequester(t *testing.T) {
	lockDir, handoff, script := shellFixture(t)

	ok, release, err := Acquire(NoWait)
	if err != nil || !ok {
		t.Fatalf("Acquire: ok=%v err=%v", ok, err)
	}
	defer release()

	// The shell side, as daneel's rig_lock_try drives it: ask, wait for the gap,
	// take it, do the work, release, and withdraw the request.
	done := make(chan string, 1)
	go func() {
		out, shErr := runShell(t, script, lockDir, handoff, `
rig_lock_request_yield daneel-intake 60 || { echo "REQUEST-REFUSED"; exit 0; }
waited=0
while [ "$waited" -lt 30 ]; do
  sleep 1; waited=$((waited+1))
  if mkdir "$RIG_LOCK_DIR" 2>/dev/null; then
    printf '%s %s daneel-intake pid=%s\n' "$$" "$(date -u +%FT%TZ)" "$$" > "$RIG_LOCK_DIR/holder"
    rm -rf "$RIG_LOCK_DIR"
    rig_lock_clear_yield daneel-intake
    echo "GOT-IT-AFTER-${waited}s"
    exit 0
  fi
done
rig_lock_clear_yield daneel-intake
echo "NEVER-GOT-IT"`)
		if shErr != nil {
			done <- "shell error: " + shErr.Error() + " / " + out
			return
		}
		done <- out
	}()

	// Poll a checkpoint the way eval-suite does between benchmarks.
	var yielded bool
	var waited time.Duration
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if yielded, waited = Checkpoint("eval-suite"); yielded {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !yielded {
		t.Fatal("the Go holder never yielded to the shell requester")
	}

	result := <-done
	if !strings.HasPrefix(result, "GOT-IT-AFTER-") {
		t.Fatalf("shell requester result = %q, want GOT-IT-AFTER-<n>s", result)
	}
	if waited <= 0 {
		t.Errorf("waited = %v, want > 0", waited)
	}
	if _, still := PendingYield(); still {
		t.Error("the handoff should have been withdrawn by the requester")
	}
	if Holder() == "" {
		t.Error("the holder must have the lock back after the handoff")
	}
}

// riglock decides whether a crashed holder can be reclaimed by reading the
// FIRST field of the holder file as a pid. Daneel used to write "daneel intake
// pid=N ..." there, which parses as "not a number" -> assume alive -> no
// Go-side job could reclaim a dead Daneel lock for the full 6h window. Both
// writers must lead with the bare pid.
func TestHolderFile_LeadsWithPID(t *testing.T) {
	_, _, _ = shellFixture(t)
	dir := os.Getenv(EnvLockDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// A dead pid in the shape daneel now writes.
	line := "2147483646 " + time.Now().UTC().Format(time.RFC3339) + " daneel-intake pid=2147483646\n"
	if err := os.WriteFile(filepath.Join(dir, "holder"), []byte(line), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if holderAlive(dir) {
		t.Error("a dead holder written in daneel's format must be recognised as dead and reclaimable")
	}
	ok, release, err := Acquire(NoWait)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if !ok {
		t.Fatal("expected to reclaim the lock from a dead daneel holder")
	}
	release()
}

// rig-lock.sh is sourced by nightly-eval.sh, which runs `set -euo pipefail`.
// Under that, a command substitution whose pipeline returns non-zero — a
// missing handoff file, or SIGPIPE when `head -1` closes the pipe early — is a
// failing simple command and EXITS THE SHELL. It cost a debugging round on
// 2026-09-11: an ordinary rig_lock_release on a free rig killed the process
// outright, and the symptom (a scheduled verb that simply stopped) named
// nothing. Every substitution in the yield block carries `|| true`; this is the
// check that says so.
func TestShellYield_SurvivesSetE(t *testing.T) {
	lockDir, handoff, script := shellFixture(t)

	// No handoff file, no lock: the paths where every substitution comes back
	// empty and non-zero.
	for _, call := range []string{
		"rig_yield_pending || true",
		"rig_lock_clear_yield nobody",
		"rig_lock_request_yield probe 60",
		"rig_lock_clear_yield probe",
		"rig_lock_acquire nowait probe || true",
	} {
		t.Run(call, func(t *testing.T) {
			_ = os.RemoveAll(lockDir)
			_ = os.Remove(handoff)
			out, err := runShell(t, script, lockDir, handoff,
				"set -euo pipefail\n"+call+"\necho SURVIVED")
			if err != nil {
				t.Fatalf("%s exited the shell under set -e: %v (%s)", call, err, out)
			}
			if !strings.Contains(out, "SURVIVED") {
				t.Errorf("%s exited the shell under set -e; output: %q", call, out)
			}
		})
	}
}
