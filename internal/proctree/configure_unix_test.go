//go:build !windows

package proctree_test

import (
	"context"
	"fmt"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/proctree"
)

// TestRunCtxTimeoutKillsProcessGroup validates the M-MOTOKO-RIG-WEDGE-FIX wiring
// (now the shared proctree recipe): when the run's ctx deadline expires, the
// Cancel func kills the whole process GROUP — so a hung motoko AND its
// backgrounded env-server child (the analogue that squatted port 8080 for 10h
// on 2026-06-29) both die, and cmd.Run() returns promptly instead of blocking
// forever. Models exactly every executor's setup: CommandContext(timeoutCtx)
// + Configure (Setpgid + group-kill Cancel + WaitDelay).
func TestRunCtxTimeoutKillsProcessGroup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	// A group leader (sh) that backgrounds a long-lived child, prints its PID,
	// then waits — i.e. it never exits on its own within the test window.
	cmd := exec.CommandContext(ctx, "sh", "-c", "sleep 300 & echo $! ; wait")
	proctree.Configure(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	start := time.Now()
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	var childPID int
	if _, err := fmt.Fscan(stdout, &childPID); err != nil {
		t.Fatalf("read backgrounded child pid: %v", err)
	}

	_ = cmd.Wait() // should return after ctx expiry triggers the group kill

	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("run was not bounded by the timeout: took %v (a hung motoko would block forever)", elapsed)
	}

	// The backgrounded child must also be gone — proving the GROUP was killed,
	// not just the leader (a leader-only kill would orphan the env-server on 8080).
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if syscall.Kill(childPID, 0) != nil { // ESRCH → reaped
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("backgrounded child %d survived the group kill — env-server would orphan on 8080", childPID)
}
