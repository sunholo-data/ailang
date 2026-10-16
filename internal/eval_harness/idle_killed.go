package eval_harness

import "github.com/sunholo-data/ailang/internal/executor"

// MarkIdleKilled turns the banked row of a run our idle watchdog killed into
// what it is: a harness failure (invalid, so aggregates skip it and
// --skip-existing retries it) that nonetheless spent real time, tokens and
// money. It reports whether res was an idle kill; other rows are untouched.
//
// Before 2026-10-16 these rows banked as valid `timeout` model failures with
// 0 ms, 0 tokens and $0: 9 of 39 v0.54.9 qwen3.8 motoko runs, every one killed
// mid-generation on the buffered ollama path, which pulled motoko's measured
// pass rate from 97% (runs that completed) to 74%.
func MarkIdleKilled(m *RunMetrics, res *AgentBenchmarkResult) bool {
	if m == nil || res == nil || res.FinishReason != executor.FinishIdleKilled {
		return false
	}
	v := MarkInvalid(ReasonHarnessError)
	v.Detail = "killed by the executor's idle watchdog (session log stopped growing) — a harness verdict, not a model outcome"
	m.Validity = v
	m.DurationMs = int64(res.DurationMS)
	u := res.Usage
	m.InputTokens = u.InputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens
	m.OutputTokens = u.OutputTokens
	m.CacheReadInputTokens = u.CacheReadInputTokens
	m.CacheCreationInputTokens = u.CacheCreationInputTokens
	m.TotalTokens = u.InputTokens + u.OutputTokens + u.ReasonTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens
	m.CostUSD = res.Cost
	m.AgentTurns = res.NumTurns
	m.AgentToolCalls = res.ToolCallCount
	m.FinishReason = res.FinishReason
	return true
}
