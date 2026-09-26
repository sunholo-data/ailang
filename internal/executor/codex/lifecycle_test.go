package codex

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
)

// Tests for lifecycle.go (HealthCheck, Register); moved out of codex_test.go
// at the 800-line gate (M-V1-SIMPLIFY-S4 M4).
func TestHealthCheck_MissingBinary(t *testing.T) {
	exec, _ := New(&executor.Config{CodexPath: "/nonexistent/codex-not-here-xyz",
		CodexModel: "gpt-5-codex", // D2(a): model is required
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := exec.HealthCheck(ctx); err == nil {
		t.Error("expected error for missing codex binary")
	}
}

func TestHealthCheck_SucceedsWhenBinaryRespondsToVersion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell scripts unsupported on windows")
	}
	tmpDir := t.TempDir()
	// Fake binary that prints a version and exits 0 when invoked with --version.
	scriptPath := filepath.Join(tmpDir, "fake-codex")
	script := "#!/bin/sh\necho 'codex 0.0.1'\nexit 0\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	exec, _ := New(&executor.Config{CodexPath: scriptPath,
		CodexModel: "gpt-5-codex", // D2(a): model is required
	})
	// Generous: the fake-binary exec only needs ms, but CI runners under load have
	// blown the old 3s deadline (test-windows flake). A real hang still trips go test.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := exec.HealthCheck(ctx); err != nil {
		t.Errorf("expected HealthCheck to succeed, got %v", err)
	}
}

func TestRegister_Idempotent(t *testing.T) {
	// init() already registered; calling Register() again should not panic
	// and should not duplicate the entry.
	Register()
	Register()

	available := executor.GlobalFactory().ListAvailable()
	count := 0
	for _, n := range available {
		if n == "codex" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 'codex' to appear exactly once, got %d", count)
	}
}
