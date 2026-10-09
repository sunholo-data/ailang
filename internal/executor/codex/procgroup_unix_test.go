//go:build !windows

package codex

import (
	"context"
	"errors"
	"fmt"
	"github.com/sunholo-data/ailang/internal/executor"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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

func TestCodexExitedLeaderStillStopsOwnedChildBeforeReturn(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "fake-codex")
	pidPath := filepath.Join(dir, "child.pid")
	script := "#!/bin/sh\nsleep 60 </dev/null >/dev/null 2>&1 &\necho $! > \"$CHILD_PID_PATH\"\nexit 1\n"
	if err := os.WriteFile(binary, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	e, err := New(&executor.Config{CodexPath: binary, CodexModel: "test", TimeoutSeconds: 5})
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.Execute(context.Background(), &executor.Task{Directive: "fixture", Workspace: dir, Timeout: 5 * time.Second, ExtraEnv: map[string]string{"CHILD_PID_PATH": pidPath}})
	if errors.Is(err, ErrProcessTerminationUnconfirmed) {
		t.Fatal(err)
	}
	data, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatal(err)
	}
	child, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.Kill(child, syscall.SIGKILL)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if syscall.Kill(child, 0) != nil {
			return
		}
		// Containers may retain a terminated orphan as a zombie; it cannot refresh.
		state, _ := exec.Command("ps", "-p", strconv.Itoa(child), "-o", "stat=").Output()
		if strings.HasPrefix(strings.TrimSpace(string(state)), "Z") {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("owned child remained live after leader exit and executor return")
}
