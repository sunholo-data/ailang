package opencode

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
)

// TestExecuteStreaming_CacheTokens pins the cache-token accounting that dev's
// fixture test does not cover. Before cache wiring, opencode summed only
// input/output, so a run that consumed ~52K tokens banked InputTokens=5 —
// opencode reports cache counters EXCLUSIVE of input, so the uncached
// remainder was all that survived.
func TestExecuteStreaming_CacheTokens(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip(skipWindows)
	}
	_, thisFile, _, _ := runtime.Caller(0)
	fixture := filepath.Join(filepath.Dir(thisFile), "testdata", "opencode_response.jsonl")
	f, err := os.Open(fixture)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer func() { _ = f.Close() }()

	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		if line := sc.Text(); line != "" {
			lines = append(lines, line)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan fixture: %v", err)
	}

	dir := t.TempDir()
	_ = writeFakeOpenCode(t, dir, lines)

	e, err := New(&executor.Config{
		OpenCodePath:   filepath.Join(dir, "opencode"),
		OpenCodeModel:  "anthropic/claude-haiku-4-5",
		TimeoutSeconds: 10,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	result, err := e.ExecuteStreaming(context.Background(), &executor.Task{
		ID:        "test-cache-tokens",
		Directive: "create a file",
		Workspace: dir,
		Timeout:   10 * time.Second,
	}, &executor.NoOpEventHandler{})
	if err != nil {
		t.Fatalf("ExecuteStreaming: %v", err)
	}
	if result == nil {
		t.Fatal("nil result")
	}

	// Summed across the fixture's three step_finish events:
	//   input   1 +   1 +     3 =     5
	//   output 26 + 133 +    25 =   184
	//   write  17213 + 17223 + 147 = 34583
	//   read       0 +     0 + 17223 = 17223
	const (
		wantInput      = 5
		wantOutput     = 184
		wantCacheWrite = 34583
		wantCacheRead  = 17223
	)
	if result.InputTokens != wantInput {
		t.Errorf("InputTokens = %d, want %d", result.InputTokens, wantInput)
	}
	if result.OutputTokens != wantOutput {
		t.Errorf("OutputTokens = %d, want %d", result.OutputTokens, wantOutput)
	}
	if result.CacheCreationInputTokens != wantCacheWrite {
		t.Errorf("CacheCreationInputTokens = %d, want %d", result.CacheCreationInputTokens, wantCacheWrite)
	}
	if result.CacheReadInputTokens != wantCacheRead {
		t.Errorf("CacheReadInputTokens = %d, want %d", result.CacheReadInputTokens, wantCacheRead)
	}

	// Cache counters are exclusive of input, so the four sum to the fixture's
	// own reported per-step totals (17240 + 17357 + 17398). This is the
	// invariant that makes summing them additive rather than double-counting.
	const wantTotal = 17240 + 17357 + 17398
	gotTotal := result.InputTokens + result.OutputTokens +
		result.CacheCreationInputTokens + result.CacheReadInputTokens
	if gotTotal != wantTotal {
		t.Errorf("token total = %d, want %d (cache counters must be exclusive of input)", gotTotal, wantTotal)
	}
}
