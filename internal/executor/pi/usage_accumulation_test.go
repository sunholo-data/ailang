package pi

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
)

// The pi row of the per-harness accumulation table (executor/tokens_processed.go), pinned
// against a recorded stream instead of trusted.
//
// tool_use.ndjson is four turns whose per-turn usage carries a real cacheWrite bucket:
//
//	turn   input  cacheWrite  output   cumulative TokensProcessed
//	2         10        4555     128                        4693
//	4         13        4709      84                        9499
//
// The run's whole input+output is 235. Everything else — 9,264 tokens, 97.5% of the work —
// is cache creation. That ratio is the reason these tests exist: under the old
// `inputTokens + outputTokens` guard this entire run weighed 235, so no cap below 235
// could distinguish it from a trivial one, and the 3,000-token cap below could never fire.
//
// TestPiTokenBudgetIsEnforced already covers the kill MECHANISM using fizzbuzz.ndjson;
// that fixture reports zero cache, so it cannot see this. These two tests cover the
// QUANTITY the mechanism is given.

const (
	piFixtureInput         = 23
	piFixtureCacheCreation = 9264
	piFixtureOutput        = 212
	piFixtureProcessed     = piFixtureInput + piFixtureCacheCreation + piFixtureOutput
)

// TestPiSumsPerTurnUsageIncludingCacheWrites asserts pi SUMS its per-turn deltas, and that
// newly written cache lands in the canonical total.
func TestPiSumsPerTurnUsageIncludingCacheWrites(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip(skipWindows)
	}
	result := runPiFixtureWithCap(t, "tool_use.ndjson", 0)

	if !result.Success {
		t.Fatalf("uncapped run failed: %q", result.Error)
	}
	if result.InputTokens != piFixtureInput {
		t.Errorf("InputTokens = %d, want %d", result.InputTokens, piFixtureInput)
	}
	if result.OutputTokens != piFixtureOutput {
		t.Errorf("OutputTokens = %d, want %d", result.OutputTokens, piFixtureOutput)
	}
	if result.CacheCreationInputTokens != piFixtureCacheCreation {
		t.Errorf("CacheCreationInputTokens = %d, want %d — per-turn cacheWrite is not being summed",
			result.CacheCreationInputTokens, piFixtureCacheCreation)
	}
	if got := result.TokensProcessed(); got != piFixtureProcessed {
		t.Errorf("TokensProcessed() = %d, want %d", got, piFixtureProcessed)
	}
}

// TestPiCapCountsCacheCreation fires a cap that is reachable ONLY through cache creation.
// 3,000 is an order of magnitude above the run's entire input+output (235) and well below
// its real 9,499, so this passing means the guard is measuring work rather than chat.
func TestPiCapCountsCacheCreation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip(skipWindows)
	}
	const cap = 3000
	result := runPiFixtureWithCap(t, "tool_use.ndjson", cap)

	if result.Success {
		t.Fatal("run succeeded despite exceeding its token cap")
	}
	if result.FinishReason != executor.FinishThrashAborted {
		t.Errorf("FinishReason = %q, want %q", result.FinishReason, executor.FinishThrashAborted)
	}
	if result.ThrashKilledAt < cap {
		t.Errorf("ThrashKilledAt = %d, want >= cap %d — the cap cannot be seeing cache writes",
			result.ThrashKilledAt, cap)
	}
	if result.ThrashKilledAt > piFixtureProcessed {
		t.Errorf("ThrashKilledAt = %d exceeds the whole run's %d",
			result.ThrashKilledAt, piFixtureProcessed)
	}
}

func runPiFixtureWithCap(t *testing.T, fixture string, maxTokens int) *executor.Result {
	t.Helper()
	dir := t.TempDir()
	writeFakePi(t, dir, loadFixtureLines(t, fixture))
	e, err := New(&executor.Config{
		PiPath:  filepath.Join(dir, "pi"),
		PiModel: "anthropic/claude-haiku-4-5",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	result, err := e.ExecuteStreaming(context.Background(), &executor.Task{
		ID:                "pi-usage-fixture",
		Directive:         "replay",
		Workspace:         dir,
		Timeout:           30 * time.Second,
		MaxTokensPerBench: maxTokens,
	}, &collectingHandler{})
	if err != nil {
		t.Fatalf("ExecuteStreaming: %v", err)
	}
	if result == nil {
		t.Fatal("ExecuteStreaming returned no result")
	}
	return result
}
