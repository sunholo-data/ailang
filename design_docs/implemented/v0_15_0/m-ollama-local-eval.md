# M-OLLAMA-LOCAL-EVAL: Local Model Eval Harness Adaptation

**Status**: Implemented (v0.15.0)
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

**Initial eval run (all 4 failed):**

| Run | Benchmark | Failure mode | Actual timeout applied |
|-----|-----------|-------------|------------------------|
| 1 | recursion_fibonacci (ailang) | Hard timeout | 90 s (YAML spec) |
| 2 | fizzbuzz (python) | Idle timeout | 180 s (3 m default) |
| 3 | recursion_fibonacci (python) | Hard timeout | 90 s (YAML spec) |
| 4 | fizzbuzz (ailang) | Idle timeout | 180 s (3 m default) |

Total wall time: 9 m 32 s — hitting limits every time, not running to completion.

**Actual timing measurements (gemma4:e4b, model warm, 24 GB Mac):**

Prompt sizes: AILANG system prompt = **72 KB / ~18,200 tokens**. Python system prompt = **1.5 KB / ~380 tokens** (48× smaller).

| Language | Benchmark | Prompt tokens | TTFT | Generation | Total | Current limit | Verdict |
|----------|-----------|--------------|------|------------|-------|---------------|---------|
| Python | fizzbuzz | ~76 | 20 s | 110 s | 130 s | 300 s (default) | ✓ passes |
| Python | recursion_fibonacci | ~60 | 66 s | 91 s | 157 s | 90 s (YAML) | ✗ TTFT alone > limit |
| AILANG | fizzbuzz | ~18,200 | 55 s | 43 s | 98 s | 300 s (default) | ✓ passes |
| AILANG | recursion_fibonacci | ~18,200 | **241 s** | 68 s | **309 s** | 90 s (YAML) | ✗ TTFT alone > 2× limit |

Key findings:
1. **TTFT is driven by prompt size, not task complexity.** Python fizzbuzz (76 tokens) = 20 s TTFT; AILANG recursion_fibonacci (18,200 tokens) = 241 s TTFT. Same model, same hardware.
2. **Generation time is fast and consistent** — 43–110 s once the model starts. This is the actual quality-measurable work.
3. **The idle timeout conflates two unrelated waits**: prefill latency (hardware/prompt-size-dependent) and per-token generation idle (quality-dependent). They should be measured and limited separately.

## Root Cause

The per-benchmark YAML `timeout` field and the executor's `idleTimeout` default conflate two distinct phases of model execution that have completely different latency profiles:

| Phase | Cloud API | Local Ollama (large prompt) | What limits it |
|-------|-----------|----------------------------|----------------|
| **Prefill / TTFT** | <1 s | 20–241 s (scales with prompt tokens) | Hardware + prompt size |
| **Generation** | 30–60 s | 43–110 s (scales with task complexity) | Model quality + hardware |

The current `idleTimeout` (3 min default) fires on *any* 3-minute gap in stdout events — including the prefill window before the first token. For a cloud model with sub-second TTFT this is fine. For a local model processing 18k tokens of AILANG syntax, the prefill alone takes 241 s and fires the idle timeout 80 s before the model would have responded.

The benchmark `timeout` (e.g. 90 s for recursion_fibonacci) was designed as a total-run SLA. It was never intended to include prefill wait, which is purely a hardware cost, not a measure of model quality.

## Design Decisions

### Decision A: Split TTFT timeout from generation timeout ✅ Recommended architecture

The cleanest fix is to split the current single `idleTimeout` into two distinct limits that reflect the two phases measured above:

- **`ttft_timeout`** — how long to wait for the *first* event after process start. Covers prefill latency. Cloud default: 10 s. Local Ollama: 60–300 s depending on prompt size.
- **`generation_timeout`** — per-token idle window *after* the first event arrives. Covers stuck-mid-generation. Stays tight for all models (60–120 s). This is the actual quality SLA.

The benchmark `timeout` YAML field (e.g. `timeout: 90`) would be reinterpreted as **generation budget only** — the clock starts from first event, not from process start. This preserves the quality SLA without penalising models for slow hardware prefill.

**Measured values needed per model in models.yml:**
```yaml
opencode-gemma4-e4b:
  agent_cli: "opencode"
  agent_model_name: "ollama/gemma4:e4b"
  pricing: 0.0
  ttft_timeout: 300      # prefill budget: up to 5 min (measured 241s on 24 GB Mac)
  generation_timeout: 120  # per-token idle after first event
```

Cloud models use defaults (ttft_timeout: 10 s, generation_timeout: 60 s) — no YAML change needed.

**Where applied**: `executor.Task` gains `TTFTTimeout time.Duration`. The executor starts a separate `ttftTimer` that fires only if no events arrive before it expires. Once the first event arrives, the ttftTimer stops and the existing `idleCheck` (now generation_timeout) takes over.

```go
// executor/opencode.go — conceptual change
ttftTimer := time.NewTimer(task.TTFTTimeout)   // new: prefill budget
idleCheck  := time.NewTimer(task.IdleTimeout)  // existing: per-token gap

// In the event loop:
case line := <-stdoutCh:
    if !firstEventSeen {
        ttftTimer.Stop()   // prefill done, cancel prefill budget
        firstEventSeen = true
    }
    idleCheck.Reset(task.IdleTimeout)  // reset per-token idle
    // ...

case <-ttftTimer.C:
    // model never responded — hardware/connectivity issue
    return error("ttft timeout: no output after %v", task.TTFTTimeout)

case <-idleCheck.C:
    // model got stuck mid-generation
    return error("generation idle for %v", task.IdleTimeout)
```

**Tradeoff vs `timeout_scale`**: More surgical — the benchmark quality SLA (generation time) stays unchanged for all models. Only the prefill wait is relaxed, and only for models that need it. Results remain directly comparable for generation quality; TTFT is reported separately as a hardware metric.

### Decision A2: `timeout_scale` in models.yml — simpler fallback

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

### Decision B: Prompt strategy for local models — three options

The 72 KB AILANG teaching prompt adds 140 s to TTFT on recursion_fibonacci (4B model). There are three approaches, not mutually exclusive:

**B1 — timeout_scale only (papering over it)**
Keep the full prompt, just allow more time. `timeout_scale: 5` brings recursion_fibonacci from 90 s → 450 s, which covers the 309 s actual. Simple, no prompt changes.  
Tradeoff: wastes the model's limited context window (8k–32k) on syntax docs it may not use.

**B2 — Lite prompt (strip the AILANG teaching guide)**
Send only the task description + workspace instructions. Drops from ~18k tokens to ~200 tokens. Measured TTFT improvement: 241 s → ~50 s for recursion_fibonacci (projection from short-prompt timing). The 4B model probably can't effectively use an 18k-token language spec anyway — it will hallucinate AILANG syntax regardless.  
Tradeoff: model gets no syntax reference at all; quality may drop for AILANG benchmarks (but may already be poor given context limits).

**B3 — Minimal prompt + on-demand context via μRAG tool (recommended for investigation)**
Send a minimal system prompt that tells the model it can call a `fetch_ailang_docs` tool to look up specific syntax sections when needed. The μRAG infrastructure (added in M-EXEC-EXPAND / M7A) already provides `ailang micro-rag context`. The model fetches only what it needs, keeping prefill tiny while giving access to the full spec.  
Tradeoff: requires wiring a new tool into the eval harness; more complex. But this is the right long-term architecture — models should retrieve rather than ingest.

**Verdict**: Implement B1 (timeout_scale) to unblock testing immediately. Prototype B3 in the next sprint using the existing μRAG infrastructure. B2 is useful as a baseline to measure how much the teaching prompt actually helps.

The design doc should be updated with results from all three approaches once measured.

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

### Preferred approach: split TTFT + generation timeouts

| File | Change | ~LOC |
|------|--------|------|
| `internal/executor/executor.go` | Add `TTFTTimeout time.Duration` to `Task` struct | +3 |
| `internal/executor/opencode/opencode.go` | Add `ttftTimer`; stop on first event; separate error messages | +20 |
| `internal/executor/codex/codex.go` | Same TTFT timer pattern | +20 |
| `internal/eval_harness/models.go` | Add `TTFTTimeout int`, `GenerationTimeout int` to `ModelConfig` | +6 |
| `internal/eval_harness/agent_runner_multi.go` | Read per-model TTFT/generation timeouts, set on task | +15 |
| `internal/eval_harness/models.yml` | Add `ttft_timeout: 300`, `generation_timeout: 120` to Ollama models; add `ollama_suite` composite | +10 |
| `internal/eval_harness/metrics.go` | Add `ttft_seconds`, `generation_seconds` to result JSON | +8 |

**Total: ~80 LOC** (vs ~45 for timeout_scale — worth the extra 35 LOC for correct semantics)

### Per-model timeout values (data-driven, 24 GB Mac)

| Model | `ttft_timeout` | `generation_timeout` | Benchmark hard limit | Rationale |
|-------|---------------|---------------------|----------------------|-----------|
| cloud models | 10 s (default) | 60 s (default) | unchanged | sub-second TTFT |
| `opencode-gemma4-e4b` | **300 s** | 120 s | unchanged (90 s from YAML) | measured TTFT 241 s + 25% headroom |
| `opencode-gemma4-26b` | **480 s** | 180 s | unchanged | not yet measured; estimate 2× e4b |

With split timeouts: the 90 s `recursion_fibonacci` hard limit now means "model must finish generating within 90 s of first token" — a fair quality SLA. The 241 s prefill wait is allowed by `ttft_timeout: 300` and recorded as a hardware metric in results.

### Result JSON additions

```json
{
  "ttft_seconds": 241,
  "generation_seconds": 68,
  "total_seconds": 309,
  "ttft_timeout_seconds": 300,
  "generation_timeout_seconds": 120
}
```

### Implementation plan

1. Add `TTFTTimeout time.Duration` to `executor.Task`
2. Implement split timer in opencode and codex executors (ttftTimer stops on first event)
3. Add `TTFTTimeout`, `GenerationTimeout` fields to `ModelConfig`; parse from YAML
4. In `agent_runner_multi.go`, set `task.TTFTTimeout` and `task.IdleTimeout` from per-model config
5. Reinterpret benchmark `timeout` YAML field as generation budget (clock from first event)
6. Add `ttft_seconds` / `generation_seconds` to result JSON
7. Add `ollama_suite` composite to `models.yml` with Ollama models
8. Update `tools/ollama_eval.sh` to default to `--models ollama_suite`

## Success Criteria

- [ ] `./tools/ollama_eval.sh --models opencode-gemma4-e4b --benchmarks fizzbuzz,recursion_fibonacci` completes at least 1/4 runs without timing out
- [ ] `recursion_fibonacci` (ailang) no longer fails with "idle timeout" or "hard timeout before first token"
- [ ] Result JSON includes `ttft_seconds`, `generation_seconds`, `ttft_timeout_seconds`
- [ ] `ollama_suite` composite resolves and runs end-to-end
- [ ] `agent_suite` unchanged — no Ollama models included
- [ ] Cloud model benchmarks unaffected (TTFTTimeout defaults to 10 s, no behaviour change)
- [ ] `ttft_timeout` / `generation_timeout` documented in eval guide

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
