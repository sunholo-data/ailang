---
title: Cost-and-Speed Budgets
sidebar_position: 9
---

# Cost-and-Speed Budgets (v0.15.1+)

The eval harness uses **cost** as the primary execution gate, not wall-clock time. This lets cheap-but-slow models (like the open-source SOTA models on OpenRouter) get fair evaluation, while still preventing expensive models from running away.

## Why cost > time

Before v0.15.1, the harness enforced wall-clock timeouts (60s opencode, 90-180s benchmark-level) as a *proxy* for cost control. This systematically excluded cheap-but-slow models — DeepSeek V4 Flash, Kimi K2.6, GLM 4.7 Flash — even though they cost fractions of a cent per benchmark.

Meanwhile expensive models (`opencode-sonnet-4-6` at ~$0.20/call × multi-turn = $13.38 across 68 runs) were *not* effectively constrained by wall-clock — each call was fast, just expensive.

**Cost is the right dimension.** v0.15.1 makes it explicit.

## How it works

Each `*executor.Task` carries a `*CostBudget` (nullable). The 5 executors (claude/opencode/gemini/codex/pi) call `budget.Add(input, output)` at their natural token-tally event point:

| Executor | Hook event |
|----------|-----------|
| claude   | stream-json `usage` event |
| opencode | stream-json `tool_result` / `step_finish` event |
| gemini   | post-hoc at `result` event (no incremental usage from Gemini CLI) |
| codex    | turn boundary |
| pi       | per-turn `message_end` event |

When the running cost exceeds `MaxCostUSD`, the executor returns early with `Result.CostKilledAt = current` and stops reading the stream. This is a **distinct failure category** from `api_error`/`timeout`/`logic_error` in the dashboard.

## Per-model `budgets:` block

Override defaults per-model in `models.yml`:

```yaml
opencode-or-minimax-m2-7:
  pricing:
    input_per_1k: 0.0003
    output_per_1k: 0.0012
  budgets:
    max_cost_usd: 0.30           # generous cost ceiling — 30¢ for cheap model
    hard_timeout_secs: 600       # 10 min wall-clock SAFETY NET
    expected_ttft_secs: 30       # for "abnormally slow" alerts (advisory)
    expected_ttf_solution_secs: 90
```

When `budgets:` is omitted, defaults apply:

```
max_cost_usd      = min($0.50, input_per_1k × 64 + output_per_1k × 32)
hard_timeout_secs = 600
```

Examples of resolved defaults:

| Model | Pricing | Default `max_cost_usd` |
|-------|---------|----------------------:|
| `or-minimax-m2-7` | $0.30/$1.20 per 1M | $0.058 (formula) |
| `or-glm-5` | $0.60/$2.08 per 1M | $0.105 (formula) |
| `claude-opus-4-7` | $15/$75 per 1M | $0.50 (clipped to ceiling) |
| `ollama-codellama` (free) | $0/$0 | $0 (no enforcement) |

## Speed metrics in `Result`

The eval harness now records speed observables alongside cost:

| Field | Meaning |
|-------|---------|
| `FirstAttemptMs` | ms from task start to first solution submission (first `Write`/`Edit` tool call or first text fallback) |
| `SuccessAtMs` | ms from task start to passing solution (-1 if not measured) |
| `Turns` | agent-mode turn count (already existed, now top-level) |
| `TokensPerSec` | OutputTokens / generation_seconds |
| `CostKilledAt` | running cost at kill time (0 if not killed) |

These flow into the dashboard's `efficiency` block per model:

```json
"efficiency": {
  "median_time_to_first_attempt_ms": 8400,
  "median_time_to_success_ms":       42000,
  "median_turns_to_success":         3,
  "median_tokens_per_sec":           45.2,
  "p90_cost_per_success":            0.18,
  "speed_efficiency_score":          0.73,
  "cost_killed_count":               0
}
```

`speed_efficiency_score = success_rate / (1 + median_TTS_seconds / 60)` — a [0..1] score the dashboard uses to rank models on the cost-vs-speed Pareto frontier.

## Dashboard charts

Two new charts visualize the new metrics:

- **Speed Radar** — median time-to-success per model, outlier-clipped at 5× median (mirroring the v0.15.0 cost radar pattern)
- **Cost-Speed Frontier** — Pareto scatter (x = $/success log scale, y = sec/success log scale, marker size = success rate, color = harness). Dominated points highlighted; frontier line drawn through optimal models.

The existing `PerModelTrend` chart gains a metric dropdown: Success Rate / Time to Success / Cost per Success.

## Migration guide

**For existing baselines (v0.14.x, v0.15.0):**
- Pre-v0.15.1 result JSONs lack `cost_killed_at`/`first_attempt_ms`/`success_at_ms`/`tokens_per_sec` fields → all default to 0
- `median_time_to_success_ms` falls back to `duration_ms` for these baselines (`ComputeEfficiency` uses `DurationMs` when `SuccessAtMs <= 0`)
- Dashboard Speed Radar will show realistic data for v0.15.1+ baselines; pre-v0.15.1 entries appear with full DurationMs (post-hoc) instead of true TTS

**For new evaluations:**
- No action needed — the eval harness automatically populates `Task.Budget` from each model's resolved `MaxCostUSD`
- Pass `--agent-parallel N` as before; budgets enforce per-Task, parallelism unchanged
- Watch for `cost_killed` rows in the Top Error Codes table after large baseline runs

## Disabling cost enforcement

To run with legacy wall-clock-only behaviour (e.g., debugging a stuck eval):

```yaml
some-model:
  budgets:
    max_cost_usd: 0    # disable enforcement; passive token tally only
    hard_timeout_secs: 60   # restore original 60s ceiling
```

`MaxCostUSD == 0` makes `CostBudget.Add()` always return `exceeded=false`. Useful for back-compat testing and replay verification.

## Reading the sweet-spot report (v0.19.0+, M-EVAL-SWEET-SPOT)

`ailang eval-sweet-spot <results_dir>` consumes the same per-result speed and cost metrics described above and answers a different question: **where is the cost-vs-time-vs-success Pareto frontier across all the models that ran in this directory?**

```bash
ailang eval-sweet-spot eval_results/v0_18_5_core_3harness
ailang eval-sweet-spot eval_results/standard --format=csv --slow-ms=30000 > sweet.csv
ailang eval-sweet-spot eval_results/standard --format=mdx     # for blog posts / docs inlining
ailang eval-sweet-spot eval_results/standard --format=json    # programmatic consumption
```

### Failure-category taxonomy

To make the buckets meaningful, v0.19.0 introduced 5 typed `ErrorCategory` values that replace catch-all `api_error` attribution where the cause is identifiable:

| Category | What it means | Capability signal? |
|---|---|---|
| `timeout` | Wall-clock or context deadline exceeded | ✅ yes — model couldn't solve in budget |
| `quota_exhausted` | Provider account/key cap reached (e.g. OpenRouter monthly limit) | ❌ no — provider noise |
| `rate_limit` | 429 / transient throttling | ❌ no — provider noise |
| `cost_killed` | Eval-side $ budget exceeded (motoko `cost_exhausted`, future executor caps) | ✅ yes — operator could raise budget |
| `step_exhausted` | Agent ran out of turns / step budget | ✅ yes — operator could raise budget |
| `api_error` | Catch-all when no more specific cause is detectable | partial — kept excluded during legacy-JSON transition |

The eval harness assigns these via `CategorizeAgentError(err, finishReason)`, which is also offline-recomputable from `stderr` strings on existing result JSONs.

### Sweet-spot buckets

For each `(model × harness × benchmark)`, the report assigns exactly one bucket:

- **fast_pass** — model passes within the slow threshold (default 60s)
- **slow_pass** — passes but slower than the threshold
- **budget_blocked** — failed with `cost_killed` or `step_exhausted`; raising the budget might flip this to a pass
- **capability_blocked** — `compile_error` / `runtime_error` / `logic_error` / `timeout`; the model genuinely couldn't solve the task within reasonable scope
- **provider_blocked** — `quota_exhausted` / `rate_limit` / `api_error`; excluded from pass-rate denominators (not the model's fault)

The per-benchmark "Cheapest / Fastest pass" footer names the model with the lowest cost-per-success and the lowest TTS for each benchmark, considering only runs where `stdout_ok == true`.

### Example

See [`examples/eval_sweet_spot_example.md`](https://github.com/sunholo-data/ailang/blob/dev/examples/eval_sweet_spot_example.md) for a real CLI output snippet against `eval_results/v0_18_5_core_3harness`.

## Reading the dashboard sweet-spot view (v0.19.0+, M-EVAL-SWEET-SPOT-WEBSITE-INTEGRATION)

The same sweet-spot data that drives `ailang eval-sweet-spot` is now embedded in `docs/static/benchmarks/latest.json` and surfaced by 3 new components on the public benchmark dashboard. All three read pre-computed values from the Go exporter — there is **no client-side recomputation** of bucket assignment or Pareto frontier membership. A CI test (`sweet_spot_parity_test`) asserts numerical equality between the CLI output and the dashboard JSON, so anywhere you see a number on the dashboard, the CLI will produce the same value to 4 decimal places.

### `$/Pass Economics` table

The headline economic comparison. For each model, shows:

- **$/pass** — total cost divided by number of passing runs (default-sort ascending)
- **Pass rate**
- **Runs** — total runs feeding this row
- **Total spend** — actual $ across all runs
- **Frontier** — ✓ when no other model has both lower $/win AND lower median TTS

A "Show as ratio vs cheapest" toggle replaces the $/pass column with `model$ / cheapest$`. This surfaces the large absolute spreads — e.g. validation runs showed gemma-4-26b at **12.4× the $/pass of deepseek-v4-flash** on the same benchmarks.

### `Cheapest / Fastest Pass per Benchmark` table

Per-benchmark champions. For each benchmark in `sweet_spot_global.champions[]`:

- **Cheapest model** + cost — the model that solved it for the lowest $
- **Fastest model** + TTS — the model that solved it in the lowest median time

When the same model wins both columns, the fastest cell collapses to "↑ (same)" to reduce visual noise.

Operator question this answers: "I want to run benchmark X, which model is cheapest / fastest?" — direct answer in one row.

### `Failure Modes` stacked bars

Per-model breakdown of every `(model × benchmark)` outcome into 4 family colors:

- **Success** (green): `fast_pass` + `slow_pass` (the model solved it)
- **Budget** (orange): `cost_killed` + `step_exhausted` — operator could raise budget and possibly flip these to passes
- **Capability** (red): `compile_error` / `runtime_error` / `logic_error` / `timeout` — the model couldn't write working code in scope
- **Provider** (gray): `quota_exhausted` / `rate_limit` / `api_error` — provider-side noise, **excluded from capability scoring**

Operator question: "Did this model fail because it can't code, because we capped it too low, or because the provider 429'd us?" — visible at a glance.

### Data flow (latest.json schema additions)

The Go exporter (`ExportBenchmarkJSON` in `internal/eval_analysis/export_json.go`) embeds:

```json
{
  "models": {
    "motoko-or-deepseek-v4-flash": {
      "aggregates": { ... },
      "efficiency": { ... },
      "sweet_spot": {
        "pass_rate": 0.857,
        "median_tts_ms": 46700,
        "p90_cost_per_success": 0.0422,
        "dollars_per_pass": 0.038,
        "pareto_frontier": true,
        "buckets": { "fast_pass": 4, "slow_pass": 5, "budget_blocked": 0, "capability_blocked": 1, "provider_blocked": 0 },
        "finish_reasons": { "stop": 13 },
        "error_categories": { "cost_killed": 0, "step_exhausted": 0, "timeout": 0, "quota_exhausted": 0, "rate_limit": 0, "api_error": 1 }
      }
    }
  },
  "sweet_spot_global": {
    "champions": [
      { "benchmark_id": "lambda_calc", "cheapest_model": "motoko-or-deepseek-v4-flash", "cheapest_cost_usd": 0.0159, "fastest_model": "claude-haiku-4-5", "fastest_tts_ms": 29000 }
    ],
    "slow_threshold_ms": 60000,
    "total_runs": 56
  }
}
```

Pre-v0.19.0 baselines that don't carry `finish_reason` or typed `error_category` values still produce a valid (mostly-zero) `sweet_spot` block — the dashboard components gate-check for presence and render a "regenerate with v0.19.0+" hint when missing.

### Known limitations

- **Tier filter on QualityScatter**: not yet wired. The frontier-inversion phenomenon (deepseek dominates hard tier, claude+gemma share stretch — see the M-EVAL-SWEET-SPOT validation report) is invisible in the current aggregate Pareto chart. Adding a tier filter requires per-tier `sweet_spot.tiers[tier]` data, which depends on M-BENCHMARK-DATA-INTEGRITY Issue #5. Tracked as a follow-up.
- **Sweet-spot data only as accurate as the input run set**: aggregate `sweet_spot` is per-model not per-(model × tier × prompt-version). Filtering the input result set before regenerating gives a sliced view.

## See also

- [Design doc](https://github.com/sunholo-data/ailang/blob/dev/design_docs/implemented/v0_15_1/m-eval-cost-and-speed-budgets.md) — full architectural rationale + axiom scoring
- [Sprint plan](https://github.com/sunholo-data/ailang/blob/dev/design_docs/implemented/v0_15_1/m-eval-cost-and-speed-budgets-sprint-plan.md) — milestone breakdown
- [Sweet-spot design doc](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_19_0/m-eval-sweet-spot.md) — v0.19.0 follow-on for failure-category disambiguation + sweet-spot CLI
- [Model configuration guide](./model-configuration.md) — full `models.yml` schema
