//go:build !windows

package proctree_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/executor/claude"
	"github.com/sunholo-data/ailang/internal/executor/codex"
	"github.com/sunholo-data/ailang/internal/executor/pi"
)

// Real adapters run local shell fixtures: no credentials or providers are used.
// The child inherits the adapter's pipes, reproducing MCP/server orphan hangs.
func TestAdaptersStopOwnedDescendants(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	for _, adapter := range []string{"claude", "pi", "codex"} {
		for _, reason := range []string{"cancel", "deadline", "timeout", "tokens", "cost"} {
			t.Run(adapter+"/"+reason, func(t *testing.T) {
				dir := t.TempDir()
				event := `{"type":"stream_event","event":{"type":"message_delta","usage":{"input_tokens":100,"output_tokens":100}}}`
				if adapter == "pi" {
					event = `{"type":"message_end","message":{"role":"assistant","usage":{"input":100,"output":100}}}`
				} else if adapter == "codex" {
					event = `{"type":"turn.completed","usage":{"input_tokens":100,"output_tokens":100}}`
				}
				script := "#!/bin/sh\nsleep 60 &\necho $! > child.pid\n"
				if reason == "tokens" || reason == "cost" {
					script += "echo '" + event + "'\n"
				}
				script += "wait\n"
				binary := filepath.Join(dir, "fake-agent")
				if err := os.WriteFile(binary, []byte(script), 0700); err != nil {
					t.Fatal(err)
				}
				cfg := &executor.Config{ClaudePath: binary, ClaudeModel: "test", PiPath: binary, PiModel: "anthropic/test", CodexPath: binary, CodexModel: "test"}
				var agent executor.Executor
				var err error
				switch adapter {
				case "claude":
					agent, err = claude.New(cfg)
				case "pi":
					agent, err = pi.New(cfg)
				case "codex":
					agent, err = codex.New(cfg)
				}
				if err != nil {
					t.Fatal(err)
				}
				unrelated := exec.Command("sleep", "60")
				if err := unrelated.Start(); err != nil {
					t.Fatal(err)
				}
				defer func() { _ = unrelated.Process.Kill(); _ = unrelated.Wait() }()
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if reason == "deadline" {
					var stop context.CancelFunc
					ctx, stop = context.WithTimeout(ctx, 5*time.Second)
					defer stop()
				}
				task := &executor.Task{Directive: "fixture", Workspace: dir, Timeout: 12 * time.Second}
				if reason == "timeout" {
					task.Timeout = 5 * time.Second
				}
				if reason == "tokens" {
					task.MaxTokensPerBench = 1
					task.Budget = executor.NewCostBudget(100, 1, 1)
				}
				if reason == "cost" {
					task.Budget = executor.NewCostBudget(0.001, 1, 1)
				}
				done := make(chan *executor.Result, 1)
				go func() { result, _ := agent.Execute(ctx, task); done <- result }()
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
				defer syscall.Kill(child, syscall.SIGKILL)
				if reason == "cancel" {
					cancel()
				}
				select {
				case result := <-done:
					if result == nil || result.Success {
						t.Errorf("expected failed execution, got %+v", result)
					}
					if reason == "tokens" && result != nil && result.ThrashKilledAt <= 1 {
						t.Error("token guard did not fire")
					}
					if reason == "cost" && result != nil && result.CostKilledAt <= 0 {
						t.Error("cost guard did not fire")
					}
				case <-time.After(16 * time.Second):
					t.Error("adapter did not return promptly")
				}
				until = time.Now().Add(time.Second)
				for time.Now().Before(until) && processRunning(child) {
					time.Sleep(10 * time.Millisecond)
				}
				if processRunning(child) {
					t.Error("owned descendant survived termination")
				}
				if err := unrelated.Process.Signal(syscall.Signal(0)); err != nil {
					t.Errorf("unrelated process killed: %v", err)
				}
			})
		}
	}
}

func processRunning(pid int) bool {
	if syscall.Kill(pid, 0) != nil {
		return false
	}
	if runtime.GOOS != "linux" {
		return true
	}
	// Linux containers can retain killed orphans as zombies until PID 1 reaps.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	b, err := exec.CommandContext(ctx, "ps", "-o", "stat=", "-p", strconv.Itoa(pid)).Output()
	return err != nil || !strings.HasPrefix(strings.TrimSpace(string(b)), "Z")
}
