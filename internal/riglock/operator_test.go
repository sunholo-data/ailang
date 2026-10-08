package riglock

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
)

// writeOperator writes a presence marker the way the watcher does.
func writeOperator(t *testing.T, pid int, until time.Time) {
	t.Helper()
	line := fmt.Sprintf("requester=operator pid=%d until=%s\n", pid, until.UTC().Format(time.RFC3339))
	if err := os.WriteFile(operatorPath(), []byte(line), 0o644); err != nil {
		t.Fatalf("write marker: %v", err)
	}
}

func TestOperatorPresent(t *testing.T) {
	isolateYield(t)
	if OperatorPresent() {
		t.Fatal("no marker must read absent")
	}
	writeOperator(t, os.Getpid(), time.Now().Add(time.Minute))
	if !OperatorPresent() {
		t.Error("a live marker must read present")
	}
	writeOperator(t, os.Getpid(), time.Now().Add(-time.Second))
	if OperatorPresent() {
		t.Error("an expired marker must read absent — a dead watcher cannot pause the rig forever")
	}
	writeOperator(t, 999999, time.Now().Add(time.Minute))
	if OperatorPresent() {
		t.Error("a marker whose watcher pid is gone must read absent")
	}
}

// Only opted-in batch jobs yield: an attended eval must not stall because the
// operator who started it is typing.
func TestCheckpoint_OperatorIgnoredWithoutOptIn(t *testing.T) {
	isolateYield(t)
	t.Setenv(config.EnvRigYieldToOperator, "")
	ok, release, err := Acquire(NoWait)
	if err != nil || !ok {
		t.Fatalf("Acquire: ok=%v err=%v", ok, err)
	}
	defer release()
	writeOperator(t, os.Getpid(), time.Now().Add(time.Minute))

	if yielded, _ := Checkpoint("eval-suite"); yielded {
		t.Error("Checkpoint yielded to the operator without AILANG_RIG_YIELD_TO_OPERATOR=1")
	}
}

func TestCheckpoint_OptedInYieldsWhileOperatorPresent(t *testing.T) {
	isolateYield(t)
	t.Setenv(config.EnvRigYieldToOperator, "1")
	lockPath := os.Getenv(EnvLockDir)
	ok, release, err := Acquire(NoWait)
	if err != nil || !ok {
		t.Fatalf("Acquire: ok=%v err=%v", ok, err)
	}
	defer release()
	tok := CurrentLease().Token
	writeOperator(t, os.Getpid(), time.Now().Add(time.Minute))

	// Stand in for the operator: see the lock released, then go idle.
	released := make(chan bool, 1)
	go func() {
		deadline := time.Now().Add(20 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(lockPath); os.IsNotExist(err) {
				released <- true
				_ = os.Remove(operatorPath())
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
		released <- false
		_ = os.Remove(operatorPath())
	}()

	yielded, _ := Checkpoint("eval-suite")
	if !yielded {
		t.Fatal("expected Checkpoint to yield while the operator is present")
	}
	if !<-released {
		t.Error("the lock was never released while the operator was present")
	}
	if Holder() == "" {
		t.Error("Checkpoint must re-acquire the lock once the operator goes idle")
	}
	if got := CurrentLease().Token; got != tok {
		t.Errorf("lease token changed across the yield: %q -> %q", tok, got)
	}
}
