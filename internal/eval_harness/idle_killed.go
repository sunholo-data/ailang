package eval_harness

import "github.com/sunholo-data/ailang/internal/executor"

// CarryFailedRunSpend copies what a failed run really spent — wall time,
// tokens, money, turns — onto its banked row. It is spend, not a grade: the
// row stays a failure. It reports whether there was anything to carry.
//
// Before 2026-10-16 every executor failure banked 0 ms, 0 tokens and $0: the
// 9 idle-killed v0.54.9 qwen3.8 motoko runs, and runs stopped at their
// wall-clock bound (a glm-5.3-flash motoko run that streamed 41 steps for 60
// minutes, 2026-10-16) — so cost tallies under-counted and per-run medians
// read the slowest runs as instant.
func CarryFailedRunSpend(m *RunMetrics, res *AgentBenchmarkResult) bool {
	if m == nil || res == nil || (res.DurationMS == 0 && res.Cost == 0 && res.NumTurns == 0) {
		return false
	}
	m.DurationMs = int64(res.DurationMS)
	u := res.Usage
	m.InputTokens = u.InputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens
	m.OutputTokens = u.OutputTokens
	m.CacheReadInputTokens = u.CacheReadInputTokens
	m.CacheCreationInputTokens = u.CacheCreationInputTokens
	m.TotalTokens = u.InputTokens + u.OutputTokens + u.ReasonTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens
	m.CostUSD = res.Cost
	m.CostProvenance = res.CostProvenance
	m.AgentTurns = res.NumTurns
	m.AgentToolCalls = res.ToolCallCount
	if res.FinishReason != "" {
		m.FinishReason = res.FinishReason
	}
	return true
}

// MarkIdleKilled turns the banked row of a run our idle watchdog killed into
// what it is: a harness failure (invalid, so aggregates skip it and
// --skip-existing retries it). It reports whether res was an idle kill; other
// rows are untouched. A wall-clock timeout is NOT this: the model needed more
// time, which is a valid outcome.
//
// Before 2026-10-16 these rows banked as valid `timeout` model failures: 9 of
// 39 v0.54.9 qwen3.8 motoko runs, every one killed mid-generation on the
// buffered ollama path, which pulled motoko's measured pass rate from 97%
// (runs that completed) to 74%.
func MarkIdleKilled(m *RunMetrics, res *AgentBenchmarkResult) bool {
	if m == nil || res == nil || res.FinishReason != executor.FinishIdleKilled {
		return false
	}
	v := MarkInvalid(ReasonHarnessError)
	v.Detail = "killed by the executor's idle watchdog (session log stopped growing) — a harness verdict, not a model outcome"
	m.Validity = v
	m.FinishReason = res.FinishReason
	return true
}
