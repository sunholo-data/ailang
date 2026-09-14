package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
)

// The claude token contract, replayed from a RECORDED stream.
//
// Until 2026-09-14 this tree had no claude-code stream fixture, so the harness's
// accumulation semantics were documented by inference rather than asserted, and the
// in-flight token cap could not be tested at all. testdata/claude_stream_partial.ndjson
// is a real capture under the executor's own flags (--output-format stream-json
// --include-partial-messages --verbose), six turns, five Read calls. Tool-result payloads
// are trimmed and $HOME is rewritten; every usage-bearing line is byte-exact.
//
// What the capture SETTLES, against what was previously believed:
//
//   - message_delta.usage DOES carry cache_creation_input_tokens. usage_cap.go recorded
//     the opposite ("outside the usage block the message_delta handler reads"), which is
//     why the in-flight kill was given up on. It is readable in flight.
//   - the counters are PER TURN and reset each turn, so they must be SUMMED across turns.
//     Assigning keeps only the final turn: on this capture that is 8 input and 54 output
//     against a true 50 / 706.
//
// Both are arithmetic, not opinion: the per-turn sums equal the result event exactly,
// which is what TestClaudeStreamUsageIsPerTurnAndSums pins.

const (
	// From the capture. Cumulative TokensProcessed after each of the six turns:
	// 224, 12093, 22212, 31936, 42135, 49970.
	fixtureInputTokens         = 50
	fixtureCacheCreationTokens = 49214
	fixtureCacheReadTokens     = 315648
	fixtureOutputTokens        = 706
	fixtureTokensProcessed     = fixtureInputTokens + fixtureCacheCreationTokens + fixtureOutputTokens
)

// TestClaudeStreamUsageIsPerTurnAndSums pins the accumulation row for claude in the
// per-harness table (executor/tokens_processed.go). It reads the fixture directly rather
// than through the executor, so it states the PROVIDER's contract independently of how we
// happen to consume it — if claude-code ever switches to run-cumulative usage, this fails
// first and names the reason.
func TestClaudeStreamUsageIsPerTurnAndSums(t *testing.T) {
	var summed struct{ in, cacheCreate, cacheRead, out int }
	turns := 0
	var final map[string]any

	for _, line := range loadClaudeFixtureLines(t, claudeStreamFixture) {
		var env map[string]any
		if err := json.Unmarshal([]byte(line), &env); err != nil {
			t.Fatalf("fixture line is not JSON: %v", err)
		}
		switch env["type"] {
		case "stream_event":
			ev, _ := env["event"].(map[string]any)
			if ev == nil || ev["type"] != "message_delta" {
				continue
			}
			u, _ := ev["usage"].(map[string]any)
			if u == nil {
				t.Fatal("message_delta carried no usage block")
			}
			turns++
			summed.in += jsonInt(u, "input_tokens")
			summed.cacheCreate += jsonInt(u, "cache_creation_input_tokens")
			summed.cacheRead += jsonInt(u, "cache_read_input_tokens")
			summed.out += jsonInt(u, "output_tokens")
		case "result":
			final, _ = env["usage"].(map[string]any)
		}
	}

	if turns != 6 {
		t.Fatalf("expected 6 turns in the fixture, got %d", turns)
	}
	if final == nil {
		t.Fatal("fixture has no result event")
	}

	// Summing per-turn deltas must reproduce the result event EXACTLY. This is the whole
	// claim: per-turn, not run-cumulative.
	for _, c := range []struct {
		field  string
		summed int
	}{
		{"input_tokens", summed.in},
		{"cache_creation_input_tokens", summed.cacheCreate},
		{"cache_read_input_tokens", summed.cacheRead},
		{"output_tokens", summed.out},
	} {
		if got := jsonInt(final, c.field); got != c.summed {
			t.Errorf("%s: per-turn sum %d != result event %d — claude's usage is no longer per-turn deltas",
				c.field, c.summed, got)
		}
	}

	// And the cache-creation bucket must be present in flight, not only at the result —
	// the premise the in-flight kill depends on.
	if summed.cacheCreate == 0 {
		t.Error("no cache_creation_input_tokens seen in any message_delta: the in-flight cap is blind again")
	}
}

// TestClaudeTokenCapKillsInFlight is the reason the fixture exists. The fake claude holds
// the result event back behind a sleep and touches a marker just before emitting it, so
// "was this killed mid-stream?" is answered by a file's existence rather than by timing.
//
// Before the fix the guard summed nothing and saw 62 tokens against a real 49,970, so no
// cap could ever fire in flight — the marker was always written.
func TestClaudeTokenCapKillsInFlight(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake-binary tests need a POSIX shell")
	}

	const cap = 20000 // crossed during turn 3 (22,212), well before the run ends

	dir := t.TempDir()
	marker := filepath.Join(dir, "result-was-emitted")
	fake := writeFakeClaude(t, dir, marker, 3)

	handler := &MockEventHandler{}
	result := runFakeClaude(t, fake, dir, handler, cap)

	if _, err := os.Stat(marker); err == nil {
		t.Errorf("the run was NOT killed in flight: claude emitted its result event "+
			"despite processing %d tokens against a cap of %d", fixtureTokensProcessed, cap)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat marker: %v", err)
	}

	if result.ThrashKilledAt < cap {
		t.Errorf("ThrashKilledAt = %d, want >= cap %d", result.ThrashKilledAt, cap)
	}
	if result.ThrashKilledAt > fixtureTokensProcessed {
		t.Errorf("ThrashKilledAt = %d exceeds the whole run's %d",
			result.ThrashKilledAt, fixtureTokensProcessed)
	}
	if result.FinishReason != executor.FinishThrashAborted {
		t.Errorf("FinishReason = %q, want %q — a thrash kill must be attributable as one",
			result.FinishReason, executor.FinishThrashAborted)
	}
	// The path that REPORTS a token kill must itself report the tokens honestly.
	if result.CacheCreationInputTokens == 0 {
		t.Error("killed Result carries no CacheCreationInputTokens — it under-reports the " +
			"very quantity it was killed for")
	}
	if got := result.TokensProcessed(); got < cap {
		t.Errorf("killed Result.TokensProcessed() = %d, want >= cap %d", got, cap)
	}
}

// TestClaudeUncappedRunReportsFixtureTotals is the control: with no cap the stream runs to
// completion and the totals are the result event's, unchanged by the accumulation fix.
func TestClaudeUncappedRunReportsFixtureTotals(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake-binary tests need a POSIX shell")
	}

	dir := t.TempDir()
	marker := filepath.Join(dir, "result-was-emitted")
	fake := writeFakeClaude(t, dir, marker, 0)

	handler := &MockEventHandler{}
	result := runFakeClaude(t, fake, dir, handler, 0)

	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("uncapped run did not reach its result event: %v", err)
	}
	if !result.Success {
		t.Fatalf("uncapped run failed: %q", result.Error)
	}
	if result.ThrashKilledAt != 0 {
		t.Errorf("ThrashKilledAt = %d on an uncapped run", result.ThrashKilledAt)
	}
	if result.InputTokens != fixtureInputTokens {
		t.Errorf("InputTokens = %d, want %d", result.InputTokens, fixtureInputTokens)
	}
	if result.OutputTokens != fixtureOutputTokens {
		t.Errorf("OutputTokens = %d, want %d", result.OutputTokens, fixtureOutputTokens)
	}
	if result.CacheCreationInputTokens != fixtureCacheCreationTokens {
		t.Errorf("CacheCreationInputTokens = %d, want %d",
			result.CacheCreationInputTokens, fixtureCacheCreationTokens)
	}
	if result.CacheReadInputTokens != fixtureCacheReadTokens {
		t.Errorf("CacheReadInputTokens = %d, want %d",
			result.CacheReadInputTokens, fixtureCacheReadTokens)
	}
	if got := result.TokensProcessed(); got != fixtureTokensProcessed {
		t.Errorf("TokensProcessed() = %d, want %d", got, fixtureTokensProcessed)
	}
	// Six message_start events, five Read calls — proves the stream was really parsed and
	// not merely tolerated.
	if len(handler.TurnStarts) != 6 {
		t.Errorf("TurnStarts = %d, want 6", len(handler.TurnStarts))
	}
	if len(handler.ToolUses) != 5 {
		t.Errorf("ToolUses = %d, want 5", len(handler.ToolUses))
	}
}

// --- helpers ---

const claudeStreamFixture = "claude_stream_partial.ndjson"

func runFakeClaude(t *testing.T, fakePath, workspace string, handler executor.EventHandler, maxTokens int) *executor.Result {
	t.Helper()
	// Guarantee writeCredentialsFile() is a no-op: with a token set it would overwrite the
	// developer's real ~/.claude/.credentials.json.
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("AILANG_AUTH_MODE", "")

	e, err := New(&executor.Config{ClaudePath: fakePath, ClaudeModel: "haiku"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	result, err := e.ExecuteStreaming(context.Background(), &executor.Task{
		ID:                "claude-usage-fixture",
		Directive:         "replay",
		Workspace:         workspace,
		Timeout:           60 * time.Second,
		MaxTokensPerBench: maxTokens,
	}, handler)
	if err != nil {
		t.Fatalf("ExecuteStreaming: %v", err)
	}
	if result == nil {
		t.Fatal("ExecuteStreaming returned no result")
	}
	return result
}

// writeFakeClaude emits the recorded stream, then holds the final result event behind a
// sleep and a marker file. If the executor kills the process for exceeding its cap, the
// marker is never written — which is the difference between killing a run in flight and
// merely noticing afterwards that it was too big.
func writeFakeClaude(t *testing.T, dir, marker string, holdSeconds int) string {
	t.Helper()
	fixture := claudeFixturePath(t, claudeStreamFixture)
	script := filepath.Join(dir, "claude")
	body := fmt.Sprintf(`#!/bin/sh
case "$1" in
  --version) echo "2.0.0 (fake)" ; exit 0 ;;
esac
FIX=%q
N=$(wc -l < "$FIX" | tr -d ' ')
sed -n "1,$((N-1))p" "$FIX"
sleep %d
: > %q
sed -n "${N}p" "$FIX"
exit 0
`, fixture, holdSeconds, marker)
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatalf("writeFakeClaude: %v", err)
	}
	return script
}

func claudeFixturePath(t *testing.T, name string) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "testdata", name)
}

func loadClaudeFixtureLines(t *testing.T, name string) []string {
	t.Helper()
	data, err := os.ReadFile(claudeFixturePath(t, name))
	if err != nil {
		t.Fatalf("open fixture %s: %v", name, err)
	}
	var lines []string
	for _, l := range strings.Split(string(data), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

func jsonInt(m map[string]any, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

// TestUsageGuardSumsTurnsAndKillsOnce exercises the accounting on its own, without a
// process or a stream. The loop-local version of this logic could only ever be tested by
// running a whole fake binary, which is why it went five months without anyone noticing it
// compared one turn's tail against a whole run's cap.
func TestUsageGuardSumsTurnsAndKillsOnce(t *testing.T) {
	kills := 0
	g := newUsageGuard(&executor.Task{MaxTokensPerBench: 10000}, func() { kills++ })

	turn := func(in, cacheCreate, cacheRead, out int) map[string]interface{} {
		return map[string]interface{}{
			"input_tokens":                float64(in),
			"cache_creation_input_tokens": float64(cacheCreate),
			"cache_read_input_tokens":     float64(cacheRead),
			"output_tokens":               float64(out),
		}
	}

	// Turn 1 opens, grows, and ends well inside the cap.
	g.fold(turn(10, 3000, 500, 1))
	g.fold(turn(10, 3000, 500, 200)) // same turn: cumulative, not additive
	if got := g.inputTokens(); got != 10 {
		t.Fatalf("within a turn the counters are cumulative: inputTokens = %d, want 10", got)
	}
	if g.thrashKilled {
		t.Fatalf("killed at %d, inside a cap of 10000", g.thrashKilledAt)
	}

	// Turn 2 starts from zero and pushes the RUN total over the cap.
	g.endTurn()
	g.fold(turn(8, 8000, 4000, 50))

	if !g.thrashKilled {
		t.Fatal("run total 10+3000+200 + 8+8000+50 = 11,268 did not breach a cap of 10,000")
	}
	if kills != 1 {
		t.Errorf("kill called %d times, want exactly 1", kills)
	}
	if want := 10 + 3000 + 200 + 8 + 8000 + 50; g.thrashKilledAt != want {
		t.Errorf("thrashKilledAt = %d, want %d", g.thrashKilledAt, want)
	}
	// Cache READS are excluded on purpose — they are the re-sent prefix, so counting them
	// measures conversation length rather than work done.
	if g.cacheReadTokens() != 4500 {
		t.Errorf("cacheReadTokens = %d, want 4500 (tracked, but not in the cap)", g.cacheReadTokens())
	}

	// A second breach must not re-kill.
	g.endTurn()
	g.fold(turn(9, 9000, 0, 9))
	if kills != 1 {
		t.Errorf("kill called %d times after a second breach, want 1", kills)
	}
}
