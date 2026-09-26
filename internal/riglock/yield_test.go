package riglock

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// isolateYield points both the lock and the handoff marker at a temp dir.
func isolateYield(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv(EnvLockDir, filepath.Join(dir, "rig.lock.d"))
	t.Setenv(EnvHandoffFile, filepath.Join(dir, "rig.handoff"))
	t.Setenv(EnvHeld, "")
}

func TestRequestYield_RoundTrip(t *testing.T) {
	isolateYield(t)

	if _, ok := PendingYield(); ok {
		t.Fatal("expected no handoff before one is requested")
	}
	if err := RequestYield("daneel-intake", time.Minute); err != nil {
		t.Fatalf("RequestYield: %v", err)
	}
	y, ok := PendingYield()
	if !ok {
		t.Fatal("expected a pending handoff")
	}
	if y.Requester != "daneel-intake" {
		t.Errorf("requester = %q, want daneel-intake", y.Requester)
	}
	if y.PID != os.Getpid() {
		t.Errorf("pid = %d, want %d", y.PID, os.Getpid())
	}

	ClearYield("daneel-intake")
	if _, ok := PendingYield(); ok {
		t.Error("expected the handoff to be gone after ClearYield")
	}
}

func TestRequestYield_RejectsBadRequester(t *testing.T) {
	isolateYield(t)
	if err := RequestYield("", time.Minute); err == nil {
		t.Error("expected an error for an empty requester")
	}
	if err := RequestYield("two words", time.Minute); err == nil {
		t.Error("expected an error for a whitespace-bearing requester: it would corrupt the marker")
	}
}

// A second requester must not inherit the first one's grant.
func TestRequestYield_ForeignHandoffRefused(t *testing.T) {
	isolateYield(t)
	if err := RequestYield("first", time.Minute); err != nil {
		t.Fatalf("first RequestYield: %v", err)
	}
	if err := RequestYield("second", time.Minute); err == nil {
		t.Error("expected the second requester to be refused while the first handoff is in force")
	}
	// Re-requesting our own is an extension, not a conflict.
	if err := RequestYield("first", 2*time.Minute); err != nil {
		t.Errorf("re-requesting our own handoff should extend it, got %v", err)
	}
}

func TestClearYield_OnlyOwnerClears(t *testing.T) {
	isolateYield(t)
	if err := RequestYield("owner", time.Minute); err != nil {
		t.Fatalf("RequestYield: %v", err)
	}
	ClearYield("someone-else")
	if _, ok := PendingYield(); !ok {
		t.Fatal("a non-owner must not be able to clear a handoff")
	}
	ClearYield("owner")
	if _, ok := PendingYield(); ok {
		t.Error("the owner should have cleared it")
	}
}

// An expired marker must be reported absent AND removed: left in place it would
// refuse every NoWait acquirer until a human noticed.
func TestPendingYield_ExpiredIsRemoved(t *testing.T) {
	isolateYield(t)
	if err := RequestYield("stale", -time.Second); err != nil {
		// negative windows fall back to the default, so write one by hand
		t.Fatalf("RequestYield: %v", err)
	}
	path := os.Getenv(EnvHandoffFile)
	past := time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
	if err := os.WriteFile(path, []byte("requester=stale pid=1 until="+past+"\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, ok := PendingYield(); ok {
		t.Fatal("an expired handoff must be reported absent")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("an expired handoff must be removed, not just ignored")
	}
}

func TestPendingYield_UnparseableIsRemoved(t *testing.T) {
	isolateYield(t)
	path := os.Getenv(EnvHandoffFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// No until= at all: an unbounded grant we must never honour.
	if err := os.WriteFile(path, []byte("requester=nodeadline pid=1\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, ok := PendingYield(); ok {
		t.Fatal("a handoff with no deadline must be reported absent")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("an unparseable handoff must be removed")
	}
}

// A requester that dies between asking and acquiring must not strand the holder.
func TestPendingYield_DeadRequesterIsRemoved(t *testing.T) {
	isolateYield(t)
	path := os.Getenv(EnvHandoffFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	future := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	// PID 0x7FFFFFFE is not a live process on any platform we run on.
	if err := os.WriteFile(path, []byte("requester=ghost pid=2147483646 until="+future+"\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, ok := PendingYield(); ok {
		t.Fatal("a handoff whose requester is dead must be reported absent")
	}
}

// The whole point of the marker living outside the lock dir: the gap a yielding
// holder opens belongs to the named requester, not to whoever polls fastest.
func TestAcquireAs_HandoffRefusesOthersButAdmitsRequester(t *testing.T) {
	isolateYield(t)
	if err := RequestYield("daneel-intake", time.Minute); err != nil {
		t.Fatalf("RequestYield: %v", err)
	}

	// The background filler acquires anonymously and must be refused.
	ok, _, err := AcquireAs(NoWait, "")
	if err != nil {
		t.Fatalf("AcquireAs: %v", err)
	}
	if ok {
		t.Error("an anonymous NoWait acquirer must be refused while a handoff is in force")
	}

	// The named requester is exactly who the gap is for.
	ok, release, err := AcquireAs(NoWait, "daneel-intake")
	if err != nil {
		t.Fatalf("AcquireAs(requester): %v", err)
	}
	if !ok {
		t.Fatal("the named requester must be admitted")
	}
	release()
}

// Checkpoint must be a no-op for a process that does not hold the lock —
// otherwise a cloud-only or --no-rig-lock run would release someone else's.
func TestCheckpoint_NoopWhenNotHolding(t *testing.T) {
	isolateYield(t)
	if err := RequestYield("someone", time.Minute); err != nil {
		t.Fatalf("RequestYield: %v", err)
	}
	if yielded, _ := Checkpoint("eval-suite"); yielded {
		t.Error("Checkpoint must not yield when we do not hold the lock")
	}
	if _, ok := PendingYield(); !ok {
		t.Error("a non-holder's Checkpoint must leave the handoff alone")
	}
}

func TestCheckpoint_NoopWhenNothingPending(t *testing.T) {
	isolateYield(t)
	ok, release, err := Acquire(NoWait)
	if err != nil || !ok {
		t.Fatalf("Acquire: ok=%v err=%v", ok, err)
	}
	defer release()

	start := time.Now()
	yielded, waited := Checkpoint("eval-suite")
	if yielded {
		t.Error("Checkpoint must not yield with no handoff pending")
	}
	if waited != 0 {
		t.Errorf("waited = %v, want 0", waited)
	}
	if time.Since(start) > time.Second {
		t.Error("the no-handoff path must be cheap — it runs between every benchmark")
	}
	// The lock must still be ours.
	if Holder() == "" {
		t.Error("Checkpoint must not have released the lock")
	}
}

// Our own request must not make us yield to ourselves.
func TestCheckpoint_IgnoresOwnRequest(t *testing.T) {
	isolateYield(t)
	ok, release, err := Acquire(NoWait)
	if err != nil || !ok {
		t.Fatalf("Acquire: ok=%v err=%v", ok, err)
	}
	defer release()
	if err := RequestYield("eval-suite", time.Minute); err != nil {
		t.Fatalf("RequestYield: %v", err)
	}
	if yielded, _ := Checkpoint("eval-suite"); yielded {
		t.Error("a holder must not yield to its own handoff request")
	}
}

// The real handoff: holder releases, requester runs, holder re-acquires.
func TestCheckpoint_YieldsAndReacquires(t *testing.T) {
	isolateYield(t)
	lockPath := os.Getenv(EnvLockDir)

	ok, release, err := Acquire(NoWait)
	if err != nil || !ok {
		t.Fatalf("Acquire: ok=%v err=%v", ok, err)
	}
	defer release()

	if err := RequestYield("daneel-intake", time.Minute); err != nil {
		t.Fatalf("RequestYield: %v", err)
	}

	// Stand in for the short job: wait for the holder to let go, take the lock,
	// do its work, release, and clear the handoff.
	took := make(chan bool, 1)
	go func() {
		deadline := time.Now().Add(20 * time.Second)
		for time.Now().Before(deadline) {
			if _, statErr := os.Stat(lockPath); os.IsNotExist(statErr) {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		if mkErr := os.Mkdir(lockPath, 0o755); mkErr != nil {
			took <- false
			ClearYield("daneel-intake")
			return
		}
		_ = os.RemoveAll(lockPath)
		took <- true
		ClearYield("daneel-intake")
	}()

	yielded, waited := Checkpoint("eval-suite")
	if !yielded {
		t.Fatal("expected Checkpoint to yield")
	}
	if !<-took {
		t.Error("the requester never got the lock — the gap was not actually opened")
	}
	if waited <= 0 {
		t.Errorf("waited = %v, want > 0", waited)
	}
	if _, ok := PendingYield(); ok {
		t.Error("the handoff should have been cleared by the requester")
	}
	// And we must hold it again afterwards.
	if Holder() == "" {
		t.Error("Checkpoint must re-acquire the lock before returning")
	}
}
