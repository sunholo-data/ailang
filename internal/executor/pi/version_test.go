package pi

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
)

// M-PI-HARNESS-UPGRADE M1: the harness version is CAPTURED, not discarded.
// HealthCheck already runs `pi --version`; before this milestone its stdout
// went nowhere, so no banked row could say which pi produced it.

func TestVersion_CapturedFromBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip(skipWindows)
	}
	dir := t.TempDir()
	_ = writeFakePi(t, dir, nil)
	e, _ := New(&executor.Config{PiPath: filepath.Join(dir, "pi"), PiModel: "anthropic/claude-haiku-4-5", TimeoutSeconds: 5})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := e.HealthCheck(ctx); err != nil {
		t.Fatalf("HealthCheck: %v", err)
	}
	// The fake prints "0.70.2"; the identity is <cli>@<observed version> — what
	// the binary REPORTS, never what the build believed it installed.
	if got := e.Version(ctx); got != "pi@0.70.2" {
		t.Fatalf("Version() = %q, want %q", got, "pi@0.70.2")
	}
}

func TestVersion_MissingBinaryIsUnmeasured(t *testing.T) {
	e, _ := New(&executor.Config{PiPath: "/nonexistent/pi-does-not-exist", PiModel: "anthropic/claude-haiku-4-5"})
	if got := e.Version(context.Background()); got != "" {
		t.Fatalf("Version() on a missing binary = %q, want empty (absent => unmeasured, never a guess)", got)
	}
}

func TestExecuteStreaming_StampsExecutorVersion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip(skipWindows)
	}
	dir := t.TempDir()
	_ = writeFakePi(t, dir, loadFixtureLines(t, "fizzbuzz.ndjson"))
	e, _ := New(&executor.Config{PiPath: filepath.Join(dir, "pi"), PiModel: "anthropic/claude-haiku-4-5", TimeoutSeconds: 10})
	res, err := e.ExecuteStreaming(context.Background(), &executor.Task{
		ID: "v", Directive: "x", Workspace: dir, Timeout: 10 * time.Second,
	}, &executor.NoOpEventHandler{})
	if err != nil {
		t.Fatalf("ExecuteStreaming: %v", err)
	}
	if res.ExecutorVersion != "pi@0.70.2" {
		t.Fatalf("Result.ExecutorVersion = %q, want pi@0.70.2", res.ExecutorVersion)
	}
}
