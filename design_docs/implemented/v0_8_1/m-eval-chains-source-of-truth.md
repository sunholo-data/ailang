# M-EVAL-CHAINS: Chains as Single Source of Truth for Agent Evals

**Status:** Planned
**Version:** v0.8.1
**Created:** 2026-02-14
**Milestone:** M-EVAL-CHAINS

## Problem

AILANG's eval harness stores agent evaluation results in two places:

1. **JSON files on disk** (`eval_results/agent/*.json`) — current source of truth for reporting
2. **Observatory spans** (`observatory.db`) — partial telemetry (cost, tokens, turns as span attributes)

This dual storage creates problems:
- Assessment data (compile_ok, runtime_ok, stdout_ok) lives only in JSON files, not queryable via `ailang chains`
- No per-benchmark chain stages — the entire eval suite is one flat Task with orphaned spans
- Can't use `ailang chains view` to investigate individual benchmark failures
- Dashboard reporting reads from filesystem, not from the structured database
- Agent eval data is disconnected from the coordinator chain system that tracks all other agent work
- Tool usage and conversation history captured only as flat transcript strings, not structured data

## Design

### Architecture: One Chain Per Suite, One Stage Per Benchmark

```
execution_chains (source_type = "eval_suite")
├── source_ref: "v0.8.0/agent/baseline"
├── status: completed
├── total_cost: $12.34
│
├── chain_stage 1: fizzbuzz / claude-haiku / ailang
│   ├── agent_id: "eval-agent"
│   ├── cost, turns, tool_calls (denormalized metrics)
│   ├── session_id → session_tools (structured tool calls)
│   ├── session_id → chat_messages (turn-by-turn conversation)
│   └── eval_assessment: { compile_ok: true, runtime_ok: true, stdout_ok: true, ... }
│
├── chain_stage 2: fizzbuzz / gemini-flash / ailang
│   └── ...
└── ... (one stage per benchmark × model × language × condition)
```

### Schema Change

Add `eval_assessment TEXT` JSON column to `chain_stages` table (migration v10).

JSON column chosen over individual columns because:
- chain_stages serves coordinator workflows too — 20+ eval-specific columns would pollute it
- New eval metrics can be added without migrations
- SQLite `json_extract()` enables filtered queries

### EvalAssessment JSON Schema

```json
{
  "benchmark_id": "fizzbuzz",
  "model": "claude-haiku-4-5",
  "language": "ailang",
  "condition": "baseline",
  "eval_mode": "agent",
  "executor": "claude",
  "compile_ok": true,
  "runtime_ok": true,
  "stdout_ok": true,
  "error_category": "none",
  "first_attempt_ok": true,
  "verify_ok": true,
  "verify_verified": 3,
  "verify_counterexample": 0,
  "prompt_version": "v0.3.24",
  "code_hash": "a1b2c3d4"
}
```

### Structured Tool & Conversation Capture

During streaming NDJSON parsing, an `ObservatoryWriter` stores:
- **Tool calls** → `session_tools` table (name, input JSON, output JSON, timing)
- **Conversation turns** → `chat_messages` table (role, content, per-turn tokens)

This makes eval data queryable and investigable through existing `ailang chains` CLI.

### Scope

- **In scope**: Agent-based evals (Claude Code, Gemini CLI)
- **Out of scope**: Standard API evals (0-shot + self-repair) — stay file-based

### Report Integration

New `LoadResultsFromChain(chainID)` returns same `[]*BenchmarkResult` type as file-based loader, so downstream pipeline (GenerateMatrix, ExportBenchmarkJSON, FormatComparison) works unchanged.

CLI flags: `--from-chain <id>`, `--from-latest-chain` on `eval-report` and `eval-compare`.

## Files Modified

- `internal/observatory/migrate.go` — migration v10
- `internal/observatory/models_chains.go` — EvalAssessment type
- `internal/observatory/store_chains.go` — store methods
- `cmd/ailang/eval_suite.go` — chain creation
- `cmd/ailang/eval_benchmark.go` — stage creation, assessment storage
- `cmd/ailang/eval_parallel.go` — threading store/chainID
- `internal/eval_analysis/loader_chains.go` — chain-based loader (NEW)
- `internal/eval_harness/agent_runner_streaming.go` — ObservatoryWriter
- `cmd/ailang/chains.go` — eval_assessment display

## Success Criteria

1. `ailang chains list --source eval_suite` shows eval chains
2. `ailang chains view <id>` shows per-benchmark assessment results
3. `ailang eval-report --from-chain <id>` generates dashboard from chain data
4. Tool calls queryable via `session_tools` for each benchmark stage
5. Conversation history queryable via `chat_messages` per stage
