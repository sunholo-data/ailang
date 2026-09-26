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
// v0_85_1/tool_use.ndjson is three assistant turns. The live capture (ollama) reports
// zero cacheWrite, so the per-turn cache buckets are PATCHED in (patchAssistantUsage)
// with the values the retired 0.70.2 Claude capture carried, keeping the ratio that
// makes these tests matter:
//
//	turn   input  cacheWrite  output   cumulative TokensProcessed
//	1        812        4555      83                        5450
//	2        903           0      12                        6365
//	3        920        4709       3                       11997
//
// The run's whole input+output is 2,733. Cache creation is 9,264 — 77% of the work.
// Under the old `inputTokens + outputTokens` guard this run weighed 2,733, so the
// 3,000-token cap below could never fire.
//
// TestPiTokenBudgetIsEnforced already covers the kill MECHANISM using fizzbuzz.ndjson;
// that fixture reports zero cache, so it cannot see this. These two tests cover the
// QUANTITY the mechanism is given.

const (
	piFixtureInput         = 2635
	piFixtureCacheCreation = 9264
	piFixtureOutput        = 98
	piFixtureProcessed     = piFixtureInput + piFixtureCacheCreation + piFixtureOutput
)

var piCachePatches = []assistantUsagePatch{{CacheWrite: 4555}, {CacheWrite: 0}, {CacheWrite: 4709}}

// TestPiSumsPerTurnUsageIncludingCacheWrites asserts pi SUMS its per-turn deltas, and that
// newly written cache lands in the canonical total.
func TestPiSumsPerTurnUsageIncludingCacheWrites(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip(skipWindows)
	}
	result := runPiFixtureWithCap(t, "v0_85_1/tool_use.ndjson", 0)

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
// 3,000 is above the run's entire input+output (2,733) and well below its real 11,997,
// so this passing means the guard is measuring work rather than chat.
func TestPiCapCountsCacheCreation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip(skipWindows)
	}
	const cap = 3000
	result := runPiFixtureWithCap(t, "v0_85_1/tool_use.ndjson", cap)

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
	writeFakePi(t, dir, patchAssistantUsage(t, loadFixtureLines(t, fixture), piCachePatches))
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
