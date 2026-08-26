//go:build !windows

package codex

import (
	"context"
	"fmt"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestCodexCancellationKillsMCPProcessGroup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", "sleep 300 & echo $! ; wait")
	configureProcessTree(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	var childPID int
	if _, err := fmt.Fscan(stdout, &childPID); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if syscall.Kill(childPID, 0) != nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("MCP-like child %d survived Codex cancellation", childPID)
}
