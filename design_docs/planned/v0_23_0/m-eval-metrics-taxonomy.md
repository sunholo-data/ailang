# M-EVAL-METRICS-TAXONOMY: Beyond Pass/Fail — Metrics for Continuous LLM Eval

**Status**: Planned
**Target**: v0.24.0
**Priority**: P1
**Estimated**: 2 days (1 day schema + summary.json extension, 1 day eval-publish/eval-trend report rendering)
**Dependencies**: M-EVAL-OS-LONGITUDINAL (rotation infrastructure, already shipped)

## Problem statement

Pass/fail is necessary but not sufficient for judging an LLM's fitness for a language. Two models can both score 50% on the AILANG smoke tier with **wildly different reasons**:

- Model A: passes the easy 50%, fails the hard 50% with structurally-valid code that has small type errors (would pass with better error messages or more iteration budget)
- Model B: passes a random 50% by chance, fails the rest with garbage syntax (model just doesn't know AILANG)

Same headline number; opposite trajectory. Model A is one prompt iteration away from 80%; Model B needs a finetune.

Today's eval pipeline shipped in M-EVAL-OS-LONGITUDINAL collects far more than pass/fail per trial (input_tokens, output_tokens, duration_ms, error_category, first_attempt_ok, repair_used, etc.) but `eval-publish` only renders `pass_rate (n=N)` in the per-release leaderboard. We're discarding the data that distinguishes Model A from Model B.

We also can now afford the **full comparison matrix** (model × language × harness × prompt strategy × sampling config) at $0 incremental cost per cell because the rig runs continuously on local Ollama. Previously this was too expensive to run with cloud APIs; now it's just disk space. The metrics taxonomy needs to support that cross-product, not just per-model rollups.

## Goals

Establish a versioned schema of metrics that:

1. Captures **why** a model passes or fails, not just whether
2. Distinguishes **model capability** ceilings from **prompt/harness/error-quality** ceilings
3. Supports **cross-axis comparison** — model × language × harness × prompt × sampling config
4. Feeds the published leaderboard so external readers can answer richer questions than "what's the pass rate"
5. Is **trend-friendly** — every metric has a "moving up means better" semantic so cross-release deltas are interpretable

## Metric categories

### 1. Model-capability metrics

These tell us about the model itself — how it handles AILANG given the rest of the rig is fixed.

| Metric | What it measures | Computed from |
|---|---|---|
| **`pass_rate`** | Headline outcome — solution compiles + runs + produces correct stdout | aggregate of `compile_ok && runtime_ok && stdout_ok` |
| **`first_attempt_pass_rate`** | Without self-repair, how often does the model nail it cold? | `first_attempt_ok` field already in result JSON |
| **`convergence_rate`** | Of failed first attempts, what % eventually pass via self-repair? | `(repair_ok && !first_attempt_ok) / !first_attempt_ok` |
| **`median_iterations_to_pass`** | How many write→check→fix cycles before convergence? | count of `tool=write` parts per session, conditional on PASS |
| **`token_efficiency`** | Tokens spent per successful trial | `total_tokens / 1 if pass else NaN`; report median |
| **`generation_throughput_tps`** | Sustained model generation speed (sanity check on rig health) | `output_tokens / generation_duration` from ollama call log |
| **`fail_token_thrash`** | When the model fails, how many tokens does it burn trying? | median `total_tokens` conditional on FAIL — high values = the model iterates hard but never converges (an "error-quality victim") |

### 2. Self-iteration quality metrics

These tell us how well the agent uses the harness's feedback loop.

| Metric | What it measures | Computed from |
|---|---|---|
| **`error_recovery_rate`** | % of compile errors that led to a successful next attempt | join `tool=bash` (running ailang check/run) with subsequent `tool=write` outcomes; chain DB |
| **`error_recovery_by_category`** | Recovery rate broken down by error category (compile_error, type_error, runtime_error, logic_error) | same as above, grouped by parsed error_category |
| **`tool_discovery_usage`** | % of trials where the agent invoked discovery tools (MCP stdlib_search, ailang docs, ailang examples) | count of MCP / discovery-CLI tool calls per session |
| **`mcp_vs_cli_usage`** | When the agent looked up syntax, did it use MCP (structured) or CLI (text)? | classify each discovery call by source |
| **`gave_up_iteration_count`** | When a trial FAILS, how many iterations had the agent tried? | count `tool=write` parts in failed sessions — proxy for "did the model give up after 2 attempts vs 8?" |

### 3. Error-quality metrics (NEW — informs the AILANG team)

These measure how *actionable* AILANG's error messages were for the agent. This is the new dimension from the 2026-05-23 list-pattern incident.

| Metric | What it measures | How to compute |
|---|---|---|
| **`error_actionable_rate`** | % of errors that include file:line:column + a "did you mean" / suggestion | parse stderr; check for `:N:N` patterns + presence of "Suggestion:" or "Did you mean" sections |
| **`error_internal_leakage_rate`** | % of errors that expose Go-internal type names (`*types.TList` etc.) — should be 0 | grep for `*types.`, `internal/types`, raw Go type signatures |
| **`error_to_recovery_correlation`** | For each error_code, what's the recovery rate next-attempt? | join error_code with subsequent-attempt outcome |

This category turns the rig into a **lens on AILANG itself** — every persistently-failing benchmark with a known error_code suggests a specific error-message improvement that would lift small-model pass rates.

### 4. Harness/rig metrics

These ensure the harness isn't the bottleneck. Stay-the-same numbers across rotations = healthy infra.

| Metric | What it measures | Source |
|---|---|---|
| **`median_call_duration_s`** | Per-LLM-call median across the rotation | ollama call log distribution |
| **`p99_call_duration_s`** | Worst-case per-call latency (catches queue oversubscription, sampling collapse) | same |
| **`long_tail_call_rate`** | % of calls > 2 min (should be near 0% with good config) | same |
| **`session_no_thrash_rate`** | % of sessions that did NOT hit pathological commands (find /, ls -R /) | tool call inspection |
| **`stdout_warn_rate`** | % of trials with stderr warnings (stdlib version mismatch, etc.) | parse stderr |
| **`vram_stability`** | Did VRAM allocation stay flat across the rotation? | ollama `/api/ps` snapshots |

### 5. Cross-axis comparison enablers

These aren't metrics per se — they're the **grouping dimensions** for the metric table above. Each rotation should record:

| Dimension | Example values | Why it matters |
|---|---|---|
| **model_id** | `opencode-gemma4-26b`, `opencode-qwen3-coder-30b`, `claude-sonnet-4-6` | obvious — the model under test |
| **api_name** | `gemma4:26b`, `gemma4:26b-ailang`, `claude-sonnet-4-6` | distinguishes a model from its Modelfile variant |
| **lang** | `ailang`, `python`, `go`, `javascript` | comparison against language baselines |
| **agent_cli** | `opencode`, `claude`, `codex`, `motoko` | which harness drove the model |
| **prompt_version** | `v0.9.0` (embedded), `v0.10.0-slim` (MCP-driven), `v0.16.1` (full teaching) | which prompt strategy |
| **sampling_config** | `default`, `ailang-tuned` (Modelfile + opencode options) | which sampling profile |
| **release_tag** | `v0.23.0`, `v0.24.0` | for trend deltas across releases |

The leaderboard then displays per cell of the cross-product: `metric@(model, lang, harness, prompt, sampling, release)`.

## Schema extension

Add `EvalMetricsTaxonomy v1` to `internal/eval_harness/rotation_summary.go`:

```go
// In addition to the existing BenchmarkSummary fields:
type BenchmarkSummary struct {
    // ... existing fields ...

    // M-EVAL-METRICS-TAXONOMY v1 (added 2026-05-23):
    FirstAttemptPassRate   float64 `json:"first_attempt_pass_rate"`
    ConvergenceRate        float64 `json:"convergence_rate"`        // 0 if no failures had repair
    MedianIterationsToPass int     `json:"median_iterations_to_pass"`
    TokenEfficiency        float64 `json:"token_efficiency"`        // median tokens / pass
    FailTokenThrashMedian  float64 `json:"fail_token_thrash_median"`
    ErrorActionableRate    float64 `json:"error_actionable_rate"`
    ErrorInternalLeakRate  float64 `json:"error_internal_leak_rate"`
    ToolDiscoveryUsageRate float64 `json:"tool_discovery_usage_rate"`
    MCPVsCLIRatio          float64 `json:"mcp_vs_cli_ratio"`        // MCP calls / total discovery calls

    // Per-error-code breakdown (top 5 error codes by frequency)
    ErrorCodeBreakdown []ErrorCodeStat `json:"error_code_breakdown"`
}

type ErrorCodeStat struct {
    Code         string  `json:"code"`           // e.g. "TYPE_UNIFY_LIST_PATTERN"
    Count        int     `json:"count"`
    RecoveryRate float64 `json:"recovery_rate"`  // % of trials where next attempt fixed it
}

type RotationSummary struct {
    // ... existing fields ...

    // M-EVAL-METRICS-TAXONOMY v1:
    Dimensions RotationDimensions `json:"dimensions"`

    // Roll-up of per-(model, lang, harness, prompt, sampling) cell, computed
    // from the BenchmarkSummary list above.
    CellMetrics map[string]CellMetrics `json:"cell_metrics"`
}

type RotationDimensions struct {
    PromptVersion   string `json:"prompt_version"`     // e.g. "v0.10.0-slim"
    SamplingConfig  string `json:"sampling_config"`    // e.g. "ailang-tuned"
    AgentHarness    string `json:"agent_harness"`      // e.g. "opencode"
    ReleaseTag      string `json:"release_tag"`        // e.g. "v0.23.0"
}

type CellMetrics struct {
    // Aggregate of the 5 main metrics across all benchmarks in this cell.
    // Easy enough to extend with the full set above when needed.
    PassRate              float64 `json:"pass_rate"`
    FirstAttemptPassRate  float64 `json:"first_attempt_pass_rate"`
    MedianWallSec         float64 `json:"median_wall_sec"`
    MedianTokens          int     `json:"median_tokens"`
    NTrials               int     `json:"n_trials"`
}
```

Existing `summary.json` files remain readable — new fields are additive (zero-valued for old rotations).

## eval-publish rendering

The leaderboard markdown gets a new section showing the richer metrics. For a single release page:

```markdown
## Per-benchmark performance (v0.23.0, gemma4:26b-ailang, opencode, slim prompt)

| Benchmark | Pass | 1st-shot | Iter→pass | Tokens (pass) | Tokens (fail) |
|---|---|---|---|---|---|
| fizzbuzz   | 67% (2/3) | 33% (1/3) | 2 | 117k | 660k |
| balanced_parens | 0% (0/3) | 0% | n/a | n/a | 660k thrash |
```

For cross-release trend (the existing trend-delta table gets extended):

```markdown
## Trend deltas (v0.23.0 vs v0.22.0)

| Benchmark | Model | Pass v0.22 → v0.23 | 1st-shot v0.22 → v0.23 | Iters v0.22 → v0.23 |
|---|---|---|---|---|
| fizzbuzz | gemma4:26b | 0% → 67% ▲ +67pp | 0% → 33% ▲ +33pp | n/a → 2 |
```

The "1st-shot" column is the killer signal: it tells you whether the model itself got smarter, vs the rig got more forgiving (self-repair budget bigger, retry logic better).

## eval-trend candidates extension

`ailang eval-trend candidates` should grow an `--include-error-quality` flag that surfaces the new error-quality metrics:

```bash
$ ailang eval-trend candidates --rotation 2026-05-23 --include-error-quality

  Benchmark         Model              Lang     Top error        Recovery rate    Suggestion?
  ─────────────────────────────────────────────────────────────────────────────────────────────
  balanced_parens   gemma4:26b-ailang  ailang   TYPE_UNIFY_LIST  0/3 (0%)         ✗ no suggestion
  ...
```

This makes the "this error has 0% recovery rate AND no suggestion" cases jump out — those are exactly the AILANG error-message improvements that would pay back the most.

## Implementation plan (2-day sprint)

**Day 1**:
- Extend `BenchmarkSummary` / `RotationSummary` structs in `internal/eval_harness/rotation_summary.go` (additive, no schema break)
- Add the per-metric computation: most are simple aggregates of fields already in `RunMetrics`
- The harder ones (`error_recovery_rate`, `mcp_vs_cli_usage_ratio`) need to join with the chain DB — pull from opencode.step / part tables
- Unit tests: synthetic fixture rotation with known metrics, assert computed values match

**Day 2**:
- Extend `eval-publish` to render the new tables (per-benchmark + cross-release trend)
- Extend `eval-trend candidates` with `--include-error-quality`
- Update the os-model-leaderboard docs explaining each metric (one-line description per column header so the published page is self-documenting)
- E2E verify: regenerate the v0.23.0 page from the current rotation data, confirm the rich metrics appear

## Conflict surface

This is purely additive to existing JSON schemas (new fields, all defaulting to zero/empty for old data). Touches:

- `internal/eval_harness/rotation_summary.go` (struct extension)
- `cmd/ailang/eval_publish.go` (rendering)
- `cmd/ailang/eval_trend.go` (filter flag)

Does NOT touch parser, typechecker, codegen, runtime — pure harness/reporting work.

## Where evidence comes from (sanity)

Every metric in the taxonomy has a verified source:

- The `first_attempt_ok`, `repair_used`, `repair_ok` fields are already in `RunMetrics` (set in `cmd/ailang/eval_benchmark.go`)
- Token counts come from opencode.execute spans in `~/.ailang/state/observatory.db` (verified live: input/output/cost present)
- Tool call types are in opencode `part` rows in `~/.local/share/opencode/opencode.db` (verified: `bash`, `write`, `read`, `ailang-docs_stdlib_search`)
- Per-call ollama durations are in `/tmp/ollama-serve-launchd.log` and parseable via the same patterns we use today
- Error categories are already classified in `internal/eval_harness/error_categorizer.go`

So no new instrumentation is needed; we just need to aggregate what we have.

## Open questions

1. **MCP usage attribution**: the chain DB shows `ailang-docs_stdlib_search` calls but doesn't yet structure them as a distinct category. Should we add a "discovery_tool" classification to `error_categorizer.go` / chain instrumentation? Or is grepping the part table's `tool` field enough?
2. **Cross-language comparisons**: the rotation defaults to AILANG-only on the local rig (per cost-efficiency goals), but the schema supports cross-language. Should the v0.23.0 leaderboard generate a python-vs-ailang column for the same model, or leave that to a later sprint?
3. **Per-prompt-version cells**: when we ship the v0.10.0-slim A/B, the leaderboard could show both cells side by side per model. Worth doing in eval-publish v2 vs leaving as a manual diff?

## Why this matters now (the user's framing)

Cloud-API eval was always pass/fail because every additional metric meant another expensive API call burn. The local-Ollama rig changes the economics: **once a benchmark trial has been run, computing 30 derived metrics from the result JSON is free**. Adding the metric definitions is the LIMITING resource, not the compute.

This sprint locks in the richer measurement vocabulary while the rig is fresh in our heads. Future model assessments — qwen3-coder, deepseek, custom AILANG-finetuned models, larger 70B+ open-source releases — can be evaluated against the same taxonomy without re-inventing what to look at. The full matrix (model × language × harness × prompt × sampling × release) becomes a single coherent dataset rather than a stack of one-off comparisons.

## What "done" looks like

1. `summary.json` for any new rotation contains all the metrics from §1-4 above
2. `ailang eval-publish` renders the per-benchmark + cross-release trend tables with at least pass rate, first-attempt pass rate, median iterations, and token efficiency
3. `ailang eval-trend candidates --include-error-quality` lists the top-5 error codes by frequency with their recovery rates and "has suggestion?" flag
4. Documentation in `docs/docs/reference/os-model-leaderboard/index.md` explains each metric and why it's there (one-paragraph each)
5. The first published page (v0.24.0 release) uses the richer schema end-to-end
