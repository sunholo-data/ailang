# M-MOTOKO-OBS-TOOLCALLS: Surface `agent_tool_calls` in Eval Result JSON

**Status**: Implemented (v0.25.x, 2026-06-17)
**Target**: v0.25.x
**Priority**: P1 for the motoko mission — observability (backlog item #1)
**Estimated**: ~15 LOC + test; <0.5 day
**Mission item**: #1 in [motoko-mission.md](motoko-mission.md)

## Problem

The motoko investigation hinges on one metric: **tool-call count** — the failure
signature was literally *"1 turn, 0 tool calls, 0 code"*. Yet per-run eval result JSON
records `agent_tool_calls = None`:

```
$ jq '.agent_turns, .agent_tool_calls' <a motoko rolling result>
6
null            # ← dropped, despite the agent making tool calls
```

The value is captured everywhere **except** the per-run JSON:

| hop | carries tool-call count? |
|---|---|
| `internal/executor/motoko/parser.go:404` → `executor.Result.ToolCallCount` | ✅ set |
| → `eval_harness.AgentBenchmarkResult.ToolCallCount` (`agent_runner_multi.go:350`) | ✅ carried |
| → chain DB via `UpdateStageMetrics(...)` (`eval_benchmark.go:353`) | ✅ stored (`chains view` shows "3 tool calls") |
| → **`eval_harness.RunMetrics`** (the per-run JSON) | ❌ **no field** — dropped at `eval_benchmark.go:278` |
| → `eval_analysis.RunMetrics` (file + chain loaders) | ❌ no field |

So `ailang chains view` shows the count but the rolling JSON (what analysis sweeps read)
does not. Recovering it requires per-run chain queries — too slow for fleet analysis, and
invisible to `eval-report`/`eval-matrix` aggregation.

This was the original justification for backlog item #1 ("see qwen's actual tool-call
output"). The /v1 fix incidentally restored most session retention (failing runs now keep
`code`, `agent_turns`, `finish_reason`, `stderr`), but the **count itself** is still
dropped.

## Approach

Add an `AgentToolCalls int` field and wire the existing value through the two dropped hops.
No new measurement — purely plumbing an already-captured value to the JSON.

1. `internal/eval_harness/metrics.go` — add to `RunMetrics`, after `AgentTurns`:
   ```go
   AgentToolCalls int `json:"agent_tool_calls,omitempty"` // Tool invocations (agent mode; validates agentic behavior)
   ```
2. `cmd/ailang/eval_benchmark.go` (~278) — map it alongside `AgentTurns`:
   ```go
   AgentToolCalls: result.ToolCallCount,
   ```
3. `internal/eval_analysis/types.go` — mirror the field on the analysis `RunMetrics`
   (so the **file-based** loader reads it from JSON automatically via the struct tag).
4. `internal/eval_analysis/loader_chains.go` (~173) — map `stage.ToolCalls` for the
   **chain-based** loader:
   ```go
   AgentToolCalls: stage.ToolCalls,
   ```

`omitempty` keeps pre-existing JSON (and standard-mode rows) unchanged.

## Acceptance criteria
- [x] `RunMetrics` (both `eval_harness` and `eval_analysis`, + `ResultJSON` export) carry
      `AgentToolCalls`.
- [x] Chain-loaded results populate `AgentToolCalls` from `stage.ToolCalls`
      (`TestStageToResult` asserts `ToolCalls:7 → 7`).
- [x] `internal/...` builds; `go test ./internal/eval_harness/... ./internal/eval_analysis/...`
      green; `TestRunMetrics_AgentToolCalls` asserts serialize/round-trip + omitempty.
- [x] No change to standard-mode rows (field omitted when 0 — test asserts).
- [~] A fresh motoko agent run writes `"agent_tool_calls": N` (N>0) into its result JSON —
      **deferred to next rotation tick** (file-writer hop is the proven-adjacent
      `AgentTurns` mapping; manual GPU runs starved the in-flight rotation). Spot-check on
      next cycle.

## Out of scope
- Aggregating avg-tool-calls into `eval-report`/dashboard (separate, additive — can follow).
- Retaining the full turn-by-turn transcript on failure (motoko `SessionLog` population) —
  tracked as a follow-up note in the analysis log; the count is the high-value 80%.

## References
- Drop point: `cmd/ailang/eval_benchmark.go:278` (maps `AgentTurns` only).
- Source: `internal/executor/motoko/parser.go:404` (`res.ToolCallCount`).
- Chain has it: `internal/observatory/models_chains.go:114` (`ChainStage.ToolCalls`).
