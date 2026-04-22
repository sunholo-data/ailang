# M-OLLAMA-LOCAL-EVAL: Local Model Eval Harness Adaptation

**Status**: Planned
**Target**: v0.15.x
**Priority**: P2 (Low) — zero cost, quality-of-life
**Estimated**: 2 days
**Dependencies**: M-EXEC-EXPAND (v0.15.0) — opencode executor + Ollama model entries in models.yml

## Problem Statement

The AILANG eval harness can route benchmarks through local Ollama models via the opencode executor (added in M-EXEC-EXPAND). However, the harness was designed around cloud API latencies and a first run against `opencode-gemma4-e4b` produced 0/4 passes with two distinct failure modes:

**Failure 1 — Hard benchmark timeout (recursion_fibonacci)**
`recursion_fibonacci.yml` has `timeout: 90` (90 seconds), which is the right SLA for a cloud API. A local 4B model (gemma4:e4b, 9.6 GB) running on a 24 GB laptop under memory pressure is slow — it was producing output, just not finishing in 90 s. Error: `opencode exceeded hard timeout (1m30s)`.

**Failure 2 — Idle timeout (fizzbuzz)**
The default idle timeout is 3 minutes. `fizzbuzz` has no per-benchmark YAML timeout, so the hard ceiling was 300 s. But the idle timeout fired first because the model emitted **no output** for 3 full minutes. The model itself responds quickly to a tiny prompt (`ollama run gemma4:e4b 'write a fizzbuzz'` completes fast). The issue is **prompt prefill latency**: the eval harness sends a large system prompt (full AILANG language guide + tool schemas ≈ thousands of tokens). Time-to-first-token (TTFT) for a large prompt on a memory-pressured local model can exceed 3 minutes. Error: `opencode idle for 3m0s (no output)`.

**What worked**: The Ollama warmup/teardown (`keep_alive=60m` via `tools/ollama_eval.sh`) correctly kept the model pinned in RAM. The TTFT problem is prompt size, not cold load.

**Note**: This is a P2 / nice-to-have. Local Ollama eval is zero cost so some leeway in latency is acceptable. The goal is correctness measurement, not latency SLA.

## Investigation Data

| Run | Benchmark | Failure mode | Actual timeout applied |
|-----|-----------|-------------|------------------------|
| 1 | recursion_fibonacci (ailang) | Hard timeout | 90 s (YAML spec) |
| 2 | fizzbuzz (python) | Idle timeout | 180 s (3 m default) |
| 3 | recursion_fibonacci (python) | Hard timeout | 90 s (YAML spec) |
| 4 | fizzbuzz (ailang) | Idle timeout | 180 s (3 m default) |

Total wall time: 9 m 32 s = (90 + 180) × 2 — hitting limits every time, not running to completion.

Direct comparison: `ollama run gemma4:e4b 'write a fizzbuzz program'` → completes in seconds.

## Root Cause

The per-benchmark YAML `timeout` field and the executor's `idleTimeout` default were both designed with cloud API latencies in mind (sub-second TTFT, fast generation). Local models differ in two ways:

1. **TTFT scales with prompt size** — prefilling 2000+ tokens on a 4B model under memory pressure takes minutes. Cloud APIs pipeline this in milliseconds.
2. **Generation throughput** — local models generate tokens at ~5–20 tok/s vs cloud APIs at 50–100+ tok/s. A benchmark that completes in 30 s on a cloud model takes 5–10× longer locally.

## Design Decisions

### Decision A: `timeout_scale` in models.yml ✅ Recommended

Add an optional `timeout_scale` field per model. When the eval harness picks up a model with `timeout_scale > 1`, it multiplies both the benchmark hard timeout and the executor idle timeout by that factor.

```yaml
opencode-gemma4-e4b:
  agent_cli: "opencode"
  agent_model_name: "ollama/gemma4:e4b"
  pricing: 0.0
  timeout_scale: 5    # 90s → 450s, idle 3m → 15m
```

**Where applied**: `agent_runner_multi.go` reads `timeout_scale` from `GlobalModelsConfig` after resolving the model. Before building the `executor.Task`, it scales:
```go
if scale := modelCfg.TimeoutScale; scale > 1.0 {
    timeoutSeconds = int(float64(timeoutSeconds) * scale)
    // pass IdleTimeout to task as well
    task.IdleTimeout = time.Duration(float64(defaultIdleTimeout) * scale)
}
```

**Tradeoff**: Scaled-timeout results are not directly comparable to cloud results on timed benchmarks. Must be flagged in reports (see Decision C).

### Decision B: Lite prompt mode for local models (Deferred)

The large system prompt is the root cause of slow TTFT. A "lite" mode would strip the verbose AILANG syntax guide and reduce the prompt to just the task description + file paths. This would genuinely improve TTFT and is probably more useful for small models anyway (a 4B model can't effectively use a 2000-token language spec).

**Deferred** because: requires auditing what the agent prompt template generates, understanding which sections a small model actually uses, and measuring the quality impact. Worth a separate sprint. For now, `timeout_scale: 10` buys time without changing prompt content.

### Decision C: Result labelling

When `timeout_scale != 1.0`, the eval result JSON should include:
```json
{
  "timeout_scale": 5.0,
  "effective_timeout_seconds": 450
}
```

`ailang eval-summary` and `ailang eval-matrix` should display a `⏱×5` marker on scaled-timeout results so they are not naively compared to cloud results in reports.

### Decision D: `ollama_suite` vs `agent_suite`

Keep Ollama models **out of `agent_suite`** for now. `agent_suite` is the cross-harness comparison set used in post-release baselines — mixing scaled-timeout local results with cloud results would muddy those reports. Instead:

- `agent_suite`: cloud models only (claude, gemini, codex, opencode-cloud)
- `ollama_suite`: local Ollama models (new composite, opt-in)

```yaml
composites:
  ollama_suite:
    models: [opencode-gemma4-e4b, opencode-gemma4-26b]
    description: "Local Ollama agent models (zero cost, relaxed timeouts)"
```

Run with: `ailang eval-suite --agent --models ollama_suite`

### Decision E: Acceptable leeway

Given zero cost, the following are acceptable for Ollama eval results:
- Hard timeout scaled up to 10× the cloud SLA
- Idle timeout scaled up to 10× (default 30 min for 4B models on 24 GB RAM)
- Total benchmark wall time of 30–60 min for a 6-benchmark suite
- Results labelled clearly as `timeout_scale=N` in reports

Not acceptable (would make results meaningless):
- Removing timeouts entirely (infinite hang potential)
- Skipping correctness checks or graders
- Separate benchmark definitions just for local models

## Solution Design

### Files to modify

| File | Change | ~LOC |
|------|--------|------|
| `internal/eval_harness/models.go` | Add `TimeoutScale float64` to `ModelConfig` struct | +5 |
| `internal/eval_harness/agent_runner_multi.go` | Apply `timeout_scale` to task.Timeout and task.IdleTimeout | +15 |
| `internal/eval_harness/models.yml` | Add `timeout_scale: 5` to `opencode-gemma4-e4b` and `opencode-gemma4-26b`; add `ollama_suite` composite | +8 |
| `internal/eval_harness/metrics.go` | Add `TimeoutScale` and `EffectiveTimeoutSeconds` to result JSON | +5 |
| `cmd/ailang/eval_report.go` | Display `⏱×N` marker for scaled-timeout results | +10 |

**Total: ~45 LOC**

### Timeout scale values

| Model | Recommended `timeout_scale` | Rationale |
|-------|----------------------------|-----------|
| `opencode-gemma4-e4b` | 5 | 4B model, ~5× slower than cloud API; fits 24 GB Mac |
| `opencode-gemma4-26b` | 8 | 26B MoE, slower throughput; needs 128 GB Mac |

These are initial estimates. After implementing `timeout_scale`, re-run the benchmark suite and tune based on actual completion times.

### Implementation plan

1. Add `TimeoutScale float64` to `ModelConfig` in `models.go`
2. Parse `timeout_scale` from `models.yml` YAML
3. In `agent_runner_multi.go`, after resolving executor+model, read `TimeoutScale` and scale both `timeoutSeconds` and `task.IdleTimeout`
4. Add `timeout_scale` + `effective_timeout_seconds` fields to result JSON in `metrics.go`
5. Add `ollama_suite` composite to `models.yml`
6. Add `⏱×N` display in `eval-summary` / `eval-matrix`
7. Update `tools/ollama_eval.sh` to use `--models ollama_suite` as the default

## Success Criteria

- [ ] `./tools/ollama_eval.sh --models opencode-gemma4-e4b --benchmarks fizzbuzz,recursion_fibonacci` completes at least 1/4 (correctness, not speed)
- [ ] Result JSON includes `timeout_scale: 5` and `effective_timeout_seconds`
- [ ] `ailang eval-matrix` shows `⏱×5` marker on scaled results
- [ ] `ollama_suite` composite resolves and runs end-to-end
- [ ] `agent_suite` unchanged — no Ollama models included
- [ ] `timeout_scale` field in models.yml is documented in eval guide

## Non-Goals

- Fixing TTFT by reducing prompt size (Decision B — separate sprint)
- Supporting Ollama models in the post-release baseline automatically
- GPU acceleration or Ollama performance tuning
- Infinite timeout mode
- Per-benchmark YAML changes for local models

## Deferred Decisions

The sprint-executor can make these calls without further review:
- Exact `timeout_scale` values (5 for e4b, 8 for 26b) — tune during verification step
- Whether `⏱×N` appears in `eval-matrix` row header or column footnote — pick whichever is cleaner

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Scale factor is deterministic, recorded in output |
| A2: Replayability | +1 | `timeout_scale` in models.yml makes runs reproducible |
| A3: Effect Legibility | 0 | No change to effect system |
| A4: Explicit Authority | 0 | No change to capability model |
| A5: Bounded Verification | 0 | No change to type system |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Zero-cost local eval broadens machine accessibility |
| A8: Minimal Syntax | 0 | No language syntax changes |
| A9: Cost Visibility | +1 | `timeout_scale` + `effective_timeout_seconds` in results makes cost/time visible |
| A10: Composability | +1 | `ollama_suite` composite follows existing composite pattern |
| A11: Structured Failure | 0 | Failure modes unchanged |
| A12: System Boundary | 0 | No new boundary crossings |

**Net score: +5** (well above threshold of +2, no hard violations)

## Related Documents

- [M-EXEC-EXPAND](../../implemented/v0_15_0/m-exec-expand-codex-opencode.md) — opencode executor + Ollama model entries
- [M-AI-OLLAMA (v0.7.0)](../../implemented/v0_7_0/m-eval-ollama-local-models.md) — prior Ollama work (AI provider layer, not executor layer)
- [Evaluation guide](../../../docs/docs/guides/evaluation/model-configuration.md) — multi-harness eval configuration
- [tools/ollama_eval.sh](../../../tools/ollama_eval.sh) — warmup/teardown wrapper for local eval runs
