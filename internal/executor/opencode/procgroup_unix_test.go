//go:build !windows

package opencode

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
)

// M-V1-SIMPLIFY-S1 M2: opencode forks a server (fixed port 8080) and MCP
// children. Before this fix every kill site was a bare cmd.Process.Kill(), so a
// timeout killed the leader and orphaned the grandchildren — the port-8080
// zombie class. The fake binary below reproduces the shape: it backgrounds a
// grandchild, publishes its pid, and blocks. Each reason must leave that
// grandchild dead within the proctree WaitDelay.
func TestOpenCodeTerminationKillsProcessGroup(t *testing.T) {
	for _, reason := range []string{"timeout", "cancel"} {
		t.Run(reason, func(t *testing.T) {
			dir := t.TempDir()
			script := "#!/bin/sh\nsleep 60 &\necho $! > child.pid\nwait\n"
			binary := filepath.Join(dir, "opencode")
			if err := os.WriteFile(binary, []byte(script), 0o700); err != nil {
				t.Fatal(err)
			}
			e, err := New(&executor.Config{
				OpenCodePath:   binary,
				OpenCodeModel:  "ollama/test",
				TimeoutSeconds: 10,
			})
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			task := &executor.Task{Directive: "fixture", Workspace: dir, Timeout: 10 * time.Second}
			if reason == "timeout" {
				task.Timeout = 500 * time.Millisecond
			}

			done := make(chan *executor.Result, 1)
			go func() { result, _ := e.Execute(ctx, task); done <- result }()

			var child int
			until := time.Now().Add(8 * time.Second)
			for time.Now().Before(until) {
				b, err := os.ReadFile(filepath.Join(dir, "child.pid"))
				if err == nil {
					child, _ = strconv.Atoi(strings.TrimSpace(string(b)))
					if child > 0 {
						break
					}
				}
				time.Sleep(10 * time.Millisecond)
			}
			if child <= 0 {
				t.Fatal("fixture never started")
			}
			defer func() { _ = syscall.Kill(child, syscall.SIGKILL) }()
			if syscall.Kill(child, 0) != nil {
				t.Fatalf("grandchild %d not alive before termination — fixture is broken", child)
			}

			if reason == "cancel" {
				cancel()
			}
			select {
			case result := <-done:
				if result == nil || result.Success {
					t.Errorf("expected failed execution, got %+v", result)
				}
			case <-time.After(16 * time.Second):
				t.Fatal("opencode executor did not return promptly")
			}

			until = time.Now().Add(2 * time.Second)
			for time.Now().Before(until) {
				if err := syscall.Kill(child, 0); err == syscall.ESRCH {
					return
				}
				time.Sleep(20 * time.Millisecond)
			}
			t.Fatalf("grandchild %d survived opencode %s — process group was not killed", child, reason)
		})
	}
}

// The bare cmd.Process.Kill() is the exact regression: it kills only the
// leader. The executor must route every kill through proctree so the group
// dies. Guarding the source keeps a future kill site from reintroducing it.
func TestOpenCodeSourceHasNoBareProcessKill(t *testing.T) {
	src, err := os.ReadFile("opencode.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(src), "Process.Kill") {
		t.Fatal("opencode.go calls cmd.Process.Kill() — use proctree.Kill(cmd) so grandchildren die too")
	}
	if !strings.Contains(string(src), "proctree.Configure(cmd)") {
		t.Fatal("opencode.go does not call proctree.Configure(cmd) on the run command")
	}
}
