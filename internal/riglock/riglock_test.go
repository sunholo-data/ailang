package riglock

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// isolate points the lock at a temp dir and clears inherited env so tests are
// hermetic regardless of any real rig lock on the dev box.
func isolate(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "rig.lock.d")
	t.Setenv(EnvLockDir, dir)
	t.Setenv(EnvHeld, "")
	return dir
}

func TestAcquire_FreeThenContended(t *testing.T) {
	isolate(t)

	// First acquire succeeds.
	ok, release, err := Acquire(NoWait)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if !ok {
		t.Fatal("expected to acquire a free lock")
	}

	// Holder is reported. (Acquire set EnvHeld=1; a same-process second Acquire
	// would short-circuit as ancestor-held, so test contention via a cleared env.)
	if h := Holder(); h == "" {
		t.Error("expected a non-empty holder after acquire")
	}

	// Simulate a *different* process attempting NoWait acquire: clear the
	// ancestor sentinel so we exercise the real mkdir-contended path.
	t.Setenv(EnvHeld, "")
	ok2, _, err := Acquire(NoWait)
	if err != nil {
		t.Fatalf("second Acquire: %v", err)
	}
	if ok2 {
		t.Fatal("expected NoWait acquire to FAIL while the lock is held")
	}

	// Release frees it; a subsequent acquire succeeds again.
	release()
	t.Setenv(EnvHeld, "")
	ok3, release3, err := Acquire(NoWait)
	if err != nil {
		t.Fatalf("third Acquire: %v", err)
	}
	if !ok3 {
		t.Fatal("expected to re-acquire after release")
	}
	release3()
}

func TestAcquire_AncestorHeldIsNoop(t *testing.T) {
	dir := isolate(t)
	t.Setenv(EnvHeld, "1") // pretend a launchd wrapper already holds it

	ok, release, err := Acquire(NoWait)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if !ok {
		t.Fatal("ancestor-held should report acquired (no-op), not blocked")
	}
	// No lock dir should have been created — the ancestor owns it.
	if _, statErr := os.Stat(dir); statErr == nil {
		t.Error("ancestor-held Acquire must NOT create the lock dir")
	}
	release() // safe no-op
}

func TestAcquire_StealsStaleLock(t *testing.T) {
	dir := isolate(t)
	t.Setenv(EnvStaleMin, "1") // 1-minute staleness window

	// Pre-create a "held" lock and backdate it well past the window.
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("seed lock: %v", err)
	}
	old := time.Now().Add(-10 * time.Minute)
	if err := os.Chtimes(dir, old, old); err != nil {
		t.Fatalf("backdate: %v", err)
	}

	ok, release, err := Acquire(NoWait)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if !ok {
		t.Fatal("expected to steal a stale lock")
	}
	release()
}

func TestAcquire_StealsDeadHolderLock(t *testing.T) {
	dir := isolate(t)

	// Pre-create a "held" lock with a fresh mtime (inside the staleness window)
	// whose recorded holder PID cannot exist (above macOS/Linux pid limits) —
	// the holder exited via os.Exit and skipped its deferred release.
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("seed lock: %v", err)
	}
	holder := []byte("999999999 2026-07-11T08:56:41Z")
	if err := os.WriteFile(filepath.Join(dir, "holder"), holder, 0o644); err != nil {
		t.Fatalf("seed holder: %v", err)
	}

	ok, release, err := Acquire(NoWait)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if !ok {
		t.Fatal("expected to steal a lock whose holder PID is dead")
	}
	release()
}

func TestAcquire_KeepsFreshLockWithUnreadableHolder(t *testing.T) {
	dir := isolate(t)

	// A fresh lock with no holder file must NOT be stolen — stealing is only
	// safe when the holder is positively dead.
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("seed lock: %v", err)
	}

	ok, _, err := Acquire(NoWait)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if ok {
		t.Fatal("must not steal a fresh lock just because the holder file is missing")
	}
}

func TestHolder_FreeLock(t *testing.T) {
	isolate(t)
	if h := Holder(); h != "" {
		t.Errorf("Holder on a free lock = %q, want empty", h)
	}
}
