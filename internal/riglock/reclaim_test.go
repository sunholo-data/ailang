package riglock

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// holdDirAsLiveHolder creates the lock directory with a holder line naming this
// (live) process, so no acquirer may steal it.
func holdDirAsLiveHolder(t *testing.T) string {
	t.Helper()
	dir := os.Getenv(EnvLockDir)
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(dir, "holder"),
		[]byte(fmt.Sprintf("%d %s", os.Getpid(), time.Now().UTC().Format(time.RFC3339))), 0o644)
	return dir
}

func TestReclaim_RefusesOthersButAdmitsTheLender(t *testing.T) {
	isolateYield(t)
	if err := writeReclaim("eval-suite"); err != nil {
		t.Fatal(err)
	}
	if ok, _, _ := AcquireAs(NoWait, ""); ok {
		t.Fatal("a third party took the lock while the lender's reclaim claim stood")
	}
	ok, release, err := acquireDir(NoWait, nil) // the lender re-acquiring
	if err != nil || !ok {
		t.Fatalf("the lender could not take its own gap back: ok=%v err=%v", ok, err)
	}
	release()
	clearReclaim()
	if _, pending := PendingReclaim(); pending {
		t.Error("clearReclaim left the claim in force")
	}
}

// While the handoff is live the gap is the borrower's, reclaim claim or not.
func TestReclaim_BorrowerAdmittedDuringHandoff(t *testing.T) {
	isolateYield(t)
	if err := writeReclaim("eval-suite"); err != nil {
		t.Fatal(err)
	}
	if err := RequestYield("daneel-intake", time.Minute); err != nil {
		t.Fatal(err)
	}
	if ok, _, _ := AcquireAs(NoWait, "other"); ok {
		t.Fatal("a non-borrower took a gap opened for daneel-intake")
	}
	ok, release, err := AcquireAs(NoWait, "daneel-intake")
	if err != nil || !ok {
		t.Fatalf("the named borrower was refused: ok=%v err=%v", ok, err)
	}
	release()
}

// A crashed lender must not leave the rig refusing everyone.
func TestReclaim_DeadOrExpiredClaimIsIgnoredAndRemoved(t *testing.T) {
	isolateYield(t)
	cmd := exec.Command("true")
	if err := cmd.Run(); err != nil {
		t.Skipf("cannot spawn a short-lived process: %v", err)
	}
	dead := cmd.Process.Pid
	cases := map[string]string{
		"dead pid": fmt.Sprintf("holder=eval-suite pid=%d until=%s\n", dead, time.Now().Add(time.Hour).UTC().Format(time.RFC3339)),
		"expired":  fmt.Sprintf("holder=eval-suite pid=%d until=%s\n", os.Getpid(), time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)),
		"garbled":  "holder= pid=x\n",
	}
	for name, line := range cases {
		if err := os.WriteFile(reclaimPath(), []byte(line), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, pending := PendingReclaim(); pending {
			t.Errorf("%s: claim reported in force", name)
		}
		if _, err := os.Stat(reclaimPath()); !os.IsNotExist(err) {
			t.Errorf("%s: claim not removed", name)
		}
		ok, release, err := AcquireAs(NoWait, "")
		if err != nil || !ok {
			t.Errorf("%s: acquire refused: ok=%v err=%v", name, ok, err)
			continue
		}
		release()
	}
}

// The regression (2026-11-25). A waiter that was already queued in Wait mode when
// the holder yielded must not take the gap. Before the fix the guard ran once, up
// front, so a waiter sleeping in the mkdir loop walked straight into it.
func TestAcquireAs_QueuedWaiterCannotTakeTheGap(t *testing.T) {
	isolateYield(t)
	prev := waitPollInterval
	waitPollInterval = 20 * time.Millisecond
	t.Cleanup(func() { waitPollInterval = prev })
	dir := holdDirAsLiveHolder(t)

	type res struct {
		ok      bool
		release Release
	}
	got := make(chan res, 1)
	go func() {
		ok, release, _ := AcquireAs(Wait, "")
		got <- res{ok, release}
	}()
	time.Sleep(150 * time.Millisecond) // the waiter is now queued in the retry loop

	// The holder lends: claim first, then open the gap.
	if err := writeReclaim("eval-suite"); err != nil {
		t.Fatal(err)
	}
	_ = os.RemoveAll(dir)
	time.Sleep(400 * time.Millisecond) // ~20 retries' worth of opportunity

	select {
	case r := <-got:
		if r.release != nil {
			r.release()
		}
		t.Fatal("a queued waiter took the gap while the lender's reclaim claim stood")
	default:
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("someone holds the lock during the gap")
	}

	clearReclaim() // the lender is done; the waiter may now proceed
	select {
	case r := <-got:
		if !r.ok {
			t.Fatal("waiter did not acquire after the claim cleared")
		}
		r.release()
	case <-time.After(5 * time.Second):
		t.Fatal("waiter never acquired after the claim cleared")
	}
}

// Checkpoint must hold the claim for the whole time it lends, and drop it after.
func TestCheckpoint_ClaimsTheGapWhileLending(t *testing.T) {
	isolateYield(t)
	lockPath := os.Getenv(EnvLockDir)
	ok, release, err := Acquire(NoWait)
	if err != nil || !ok {
		t.Fatalf("Acquire: ok=%v err=%v", ok, err)
	}
	defer release()
	if err := RequestYield("daneel-intake", time.Minute); err != nil {
		t.Fatal(err)
	}
	var claimedDuringLoan bool
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer ClearYield("daneel-intake")
		for i := 0; i < 400; i++ {
			if _, statErr := os.Stat(lockPath); os.IsNotExist(statErr) {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		_, claimedDuringLoan = PendingReclaim()
		if os.Mkdir(lockPath, 0o755) == nil {
			_ = os.RemoveAll(lockPath)
		}
	}()
	if yielded, _ := Checkpoint("eval-suite"); !yielded {
		t.Fatal("expected Checkpoint to yield")
	}
	wg.Wait()
	if !claimedDuringLoan {
		t.Error("no reclaim claim was in force while the GPU was lent")
	}
	if _, pending := PendingReclaim(); pending {
		t.Error("the reclaim claim outlived the loan")
	}
	if Holder() == "" {
		t.Error("Checkpoint must re-acquire the lock before returning")
	}
}

// The shell half reads the claim Go writes: a NoWait shell acquirer is refused.
func TestReclaim_ShellRefusesWhileClaimed(t *testing.T) {
	lockDir, handoff, script := shellFixture(t)
	if err := writeReclaim("eval-suite"); err != nil {
		t.Fatal(err)
	}
	out, _ := runShell(t, script, lockDir, handoff, `rig_lock_acquire nowait && echo ACQUIRED || echo REFUSED`)
	if out != "REFUSED" {
		t.Fatalf("shell acquirer got %q while the reclaim claim stood, want REFUSED", out)
	}
	clearReclaim()
	out, _ = runShell(t, script, lockDir, handoff, `rig_lock_acquire nowait && echo ACQUIRED || echo REFUSED`)
	if out != "ACQUIRED" {
		t.Fatalf("shell acquirer got %q after the claim cleared, want ACQUIRED", out)
	}
}

// The regression on the shell side: rig_lock_acquire wait, already queued, must
// not take the gap. This is the exact shape of 2026-11-25.
func TestReclaim_ShellQueuedWaiterCannotTakeTheGap(t *testing.T) {
	lockDir, handoff, script := shellFixture(t)
	if err := os.Mkdir(lockDir, 0o755); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	var mu sync.Mutex
	cmd := exec.Command("bash", "-c", "source "+script+"\nrig_lock_acquire wait && echo ACQUIRED")
	cmd.Env = append(os.Environ(), "RIG_LOCK_DIR="+lockDir, "RIG_HANDOFF_FILE="+handoff, "RIG_LOCK_POLL_SEC=1")
	cmd.Stdout = lockedWriter{&out, &mu}
	cmd.Stderr = lockedWriter{&out, &mu}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() })
	time.Sleep(1500 * time.Millisecond) // queued in the retry loop

	if err := writeReclaim("eval-suite"); err != nil {
		t.Fatal(err)
	}
	_ = os.RemoveAll(lockDir)
	time.Sleep(3 * time.Second) // three retries' worth of opportunity
	mu.Lock()
	early := out.String()
	mu.Unlock()
	if strings.Contains(early, "ACQUIRED") {
		t.Fatal("a queued shell waiter took the gap while the lender's reclaim claim stood")
	}

	clearReclaim()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		s := out.String()
		mu.Unlock()
		if strings.Contains(s, "ACQUIRED") {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("shell waiter never acquired after the claim cleared")
}

type lockedWriter struct {
	b  *bytes.Buffer
	mu *sync.Mutex
}

func (w lockedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.Write(p)
}

// The check-then-mkdir window: a claim that appears between the guard check and
// the mkdir must still win. acquireDir re-checks after mkdir and hands the
// directory back. Driven with a guard that changes its answer mid-attempt,
// because no timing test can hit a microsecond window reliably.
func TestAcquireDir_RechecksTheGuardAfterMkdir(t *testing.T) {
	isolateYield(t)
	calls := 0
	admit := func() bool { calls++; return calls == 1 } // admitted, then a claim lands
	ok, release, err := acquireDir(NoWait, admit)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		release()
		t.Fatal("acquired although the guard refused right after mkdir")
	}
	if _, statErr := os.Stat(os.Getenv(EnvLockDir)); !os.IsNotExist(statErr) {
		t.Error("the directory won in the gap was not handed back")
	}
}
