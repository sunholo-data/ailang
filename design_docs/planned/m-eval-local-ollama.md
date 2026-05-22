# M-EVAL-LOCAL-OLLAMA — Optimize eval-suite for local Ollama models

**Status**: Planned — Investigation in progress (overnight run 2026-05-22 → 2026-05-23)
**Target**: v0.22.0 (tentative; depends on refactor scope)
**Priority**: P1 — Strategic infrastructure for cost-free continuous eval
**Estimated**: 3-5 days (after overnight measurement phase resolves open questions)
**Dependencies**: None (existing Ollama provider + opencode executor already wired)

## TL;DR

The team purchased a 128 GB Apple M4 Max Mac Studio specifically to run open-source models locally for **continuous 24/7 eval**, replacing pay-per-token OpenRouter routing for OS models. The machine is dedicated to this workload — not a one-off overnight test, but the **canonical OS-model eval rig** going forward. gemma4:26b is the starting point; the rig should comfortably absorb a rotation of comparable-or-smaller OS models (Qwen3, GLM, DeepSeek, Llama families).

Investigation on 2026-05-22 found the AILANG eval-suite is *technically* capable of doing this today, but several rough edges prevent it from being practical:

1. **`-agent-parallel` is a dead field** — set, printed in banner, never read. Only `-parallel` actually gates concurrency.
2. **Default timeouts are tuned for cloud APIs** — 3-min idle / 180s generation timeout kills local thinking models mid-reasoning.
3. **No Ollama autoconfig** — eval-suite does not adjust `OLLAMA_NUM_PARALLEL` / `OLLAMA_MAX_LOADED_MODELS` to match requested `-parallel N`.
4. **High output variance per benchmark** — gemma4:26b agent-mode fizzbuzz: 1m57s (clean) vs 3m57s + 2.6M tokens of thrashing on a re-run. Need to understand why.

Goal of this doc: capture decisions + measurement data so we can pick the optimal default config for the Mac Studio rig and decide which (if any) of the rough edges merits a refactor PR.

## Problem Statement

**Why a 128 GB local rig:**

Open-source frontier-class models (Gemma 4 26B, DeepSeek V4, Qwen3 235B, GLM 5) cost $0 to run locally vs $0.05–$1.50 per benchmark via OpenRouter. With ~50 benchmarks × ~10 model variants × continuous re-runs as we iterate on stdlib/prompts, that's the difference between a free overnight run and a multi-hundred-dollar one. The local rig is purpose-built to absorb that workload.

**Current state (gemma4:26b smoke, 2026-05-22):**

- Hardware: Apple M4 Max, 16 cores (12P + 4E), 40 GPU cores, 128 GB unified memory
- Software: Ollama 0.24.0, opencode 1.15.7, AILANG v0.21.0-4-gdf2ed8de-dirty
- Model: `gemma4:26b` — 25.8B params, Q4_K_M, 17 GB on disk, 25.76 GB VRAM resident, 262K context

| Run mode | fizzbuzz | adt_option | csv_to_json |
|---|---|---|---|
| Standard (single-shot) | 35s → PAR_001 fail | not run | not run |
| Agent (clean) | **1m57s → PASS** (298K tokens) | not run | not run |
| Agent (re-run, 600s timeout) | 3m57s → fail (compile_err, **2.6M tokens**) | 7m25s → fail (logic_err, 148K tokens) | not run |
| Agent (180s timeout, default) | timeout | 3-min idle timeout | 3-min hard timeout |

Massive output variance. The model can succeed cheaply (1m57s) or thrash (3m57s + 2.6M tokens) on the same task with the same seed. Investigating overnight.

**Impact:**

- We can't yet trust local-Ollama eval results because variance dominates signal.
- We can't yet maximize throughput on the rig because parallelism flag is a no-op and Ollama defaults to `MAX_LOADED_MODELS=1`.
- The eval-suite UX has a footgun (dead `-agent-parallel` flag) that misled even me during this investigation.

## Goals

**Strategic goal (multi-month):** Turn this Mac Studio into AILANG's canonical OS-model eval lab. Continuous background eval generates the empirical data that drives:
1. **Roadmap prioritization** — "what's the next stdlib/prompt change that would help OS models pass smoke?"
2. **OS-model rotation tracking** — as Gemma 4 → Gemma 5, Qwen 3.5 → Qwen 4 etc., we measure improvement on the SAME benchmarks over time.
3. **Fine-tuning data foundation** — every eval run produces (prompt, model_output, pass/fail, error_category) tuples. After 6–12 months of continuous evals, this is the dataset to fine-tune AILANG-specific models from.
4. **Public benchmark surface** — the same data becomes the "how do OS models compare on AILANG?" leaderboard.

**This-sprint goal:** Make `make eval-suite MODELS=opencode-gemma4-26b ...` produce *useful, repeatable, throughput-maximized* results without per-run hand-tuning. Establish the eval rotation infrastructure so the rig can run unattended for weeks.

**Success Metrics (this sprint):**

1. **Throughput**: ≥3 benchmarks completing concurrently on local gemma4:26b without quality degradation.
2. **Stability**: 3 consecutive runs of the same benchmark return matching pass/fail (variance ≤1 of 3).
3. **No footguns**: `-agent-parallel` either works as advertised or is removed.
4. **Defaults that work**: A new user can `make eval-suite MODELS=opencode-gemma4-26b -agent` and get a usable result without reading source.
5. **Documentation**: One clear page in `docs/docs/guides/evaluation/local-ollama.md` describing the workflow.
6. **Rotation infrastructure**: Cron-or-equivalent schedule running smoke tier nightly across the Tier-1 model rotation, results land in time-series-queryable form.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Wire `-agent-parallel` to a real semaphore, or delete the flag | User-visible UX; affects all agent-mode runs | human | design | low |
| Default timeouts for `provider: "ollama"` entries | All Ollama models thrash on cloud-tuned defaults | human | design | low |
| Whether eval-suite should set `OLLAMA_NUM_PARALLEL` automatically | Surprises if it does; broken throughput if it doesn't | human | design | med |
| Whether `OLLAMA_MAX_LOADED_MODELS` should be raised on multi-model runs | Risk of OOM on smaller boxes; throughput on big rigs | human | design | med |
| Whether to keep gemma4:26b dense vs MoE variant as canonical | Affects what we test going forward | human | design | low |
| Whether to add per-provider concurrency caps for mixed local+cloud runs | Avoids local Ollama bottlenecking cloud parallelism | agent | design | med |

### Design Freeze

- [ ] `-agent-parallel` resolved (wire or delete)
- [ ] Per-Ollama-model timeout defaults agreed (probably `generation_timeout: 600`, `ttft_timeout: 480` for thinking models)
- [ ] Decision on Ollama env autoconfig (yes/no)
- [ ] Canonical gemma4 tag (`gemma4:26b` dense vs `gemma4:26b-a4b-it-q4_K_M` MoE) — currently using dense per user pull

### Deferred Decisions

- Per-provider concurrency caps (option 3 in the refactor list) — only worth it if mixed-mode runs become common.
- Auto-pulling missing Ollama models if requested but not present — convenience, not blocking.
- Promoting `ollama-gemma4-26b` (direct, standard-mode) to a default suite — depends on whether OS models ever pass standard-mode smoke.

## Findings from 2026-05-22 Investigation

### 1. Dead `-agent-parallel` Field

```
cmd/ailang/eval_suite.go:122   agentMaxConcurrent := fs.Int("agent-parallel", 10, ...)
cmd/ailang/eval_suite.go:568   AgentBenchmarkConfig.MaxConcurrent: *agentMaxConcurrent
internal/eval_harness/agent_runner.go:20   MaxConcurrent int   // never read
```

`grep -rn '\.MaxConcurrent' internal/ cmd/` returns zero readers of the field outside its declaration. Only `-parallel` (via `runBenchmarksParallel`'s `chan struct{}` semaphore at [eval_parallel.go:38](cmd/ailang/eval_parallel.go#L38)) actually gates concurrency, and it does so uniformly for both standard and agent modes.

**Action:** Either wire `MaxConcurrent` to a real per-CLI semaphore *inside* the agent path, OR remove the flag. The current state misleads users (it misled me).

### 2. Timeout defaults are cloud-tuned

[opencode.go:168](internal/executor/opencode/opencode.go#L168) defaults idle timeout to 3 minutes if `task.IdleTimeout == 0`. The hard timeout defaults to `task.Timeout` which comes from each model's `generation_timeout`. Pre-investigation, `opencode-gemma4-26b` had `generation_timeout: 180`. That's fine for Claude Sonnet 4.6 (which solves csv_to_json in ~43s) but kills local thinking models mid-reasoning.

**Already changed in this branch** (commit pending):

```yaml
opencode-gemma4-26b:
  ttft_timeout: 480
  generation_timeout: 600  # was 180; local thinking model needs 5–10min headroom
```

**Open question:** what should the idle timeout be for local thinking models? gemma4:26b can spend >3 min in pure thinking-token streaming. Need to either:
- Bump idle timeout when `provider: "ollama"`, OR
- Make thinking tokens count as activity (requires opencode-side change — out of scope here).

### 3. Ollama parallelism

Ollama 0.24 defaults (no env vars set on this box):

| Var | Default | Effect |
|---|---|---|
| `OLLAMA_MAX_LOADED_MODELS` | 1 | One model resident at a time |
| `OLLAMA_NUM_PARALLEL` | 4 (auto-tuned) | Up to 4 concurrent requests per loaded model |
| `OLLAMA_MAX_QUEUE` | 512 | Backlog before 429 |

With these defaults, Ollama is ready to serve 4 concurrent gemma4:26b requests. The eval-suite gates at `-parallel 1` (my conservative choice during the investigation) — so the bottleneck is *our* harness, not Ollama. Running `-parallel 4` should yield ~4× wall-clock throughput per-benchmark-suite, modulo GPU compute serialization (one GPU still serves all 4 streams).

**Memory math for gemma4:26b at -parallel 4:**

- Weights: 25.76 GB (shared across all concurrent requests — only loaded once)
- KV cache per request: depends on context length actually used. gemma4 supports 262K context but most benchmarks stay <20K tokens. Estimated 1–4 GB per concurrent slot.
- 4 concurrent: ~26 + 4 × 3 = ~38 GB total. Well within 128 GB.

**Implication:** We could likely run `-parallel 4` safely on this rig without touching Ollama config. `-parallel 8` would need either KV quantization or bumping context limit. Will measure overnight.

### 4. Variance is the real problem

Same fizzbuzz benchmark, same seed (42), same model, three different outcomes in three runs:

- Run 1: PASS in 117s, 298K total tokens
- Run 2: FAIL (compile_err) in 237s, **2.6M total tokens** (model thrashed on parse errors)
- Run 3: timed out (180s old hard cap)

Output is fundamentally non-deterministic in agent-mode because each opencode session can take a different number of turns depending on what the model emits on turn 1. This is a property of stochastic LLM agents, not a bug — but it means **single-run pass/fail is not a reliable signal for local-Ollama models.**

**Implication:** For local Ollama, we likely need N≥3 trial runs per benchmark and a "best-of-N" or "median-of-N" pass criterion. This is a meaningful eval-harness behavior change. Worth a separate sub-design before committing.

## Solution Design

### Overview

Three categories of change, listed in increasing cost:

1. **Configuration only (no code):** Per-model timeout bumps in `models.yml` for all `provider: "ollama"` entries. Already done for `opencode-gemma4-26b`. Apply same pattern to `opencode-gemma4-e4b`, future `ollama-*` direct entries.

2. **Small code changes (~50 LOC):** Fix the `-agent-parallel` dead field (either wire to a per-CLI semaphore or delete). Optionally: emit a warning if `-parallel N` is set higher than detected `OLLAMA_NUM_PARALLEL` for ollama-provider models.

3. **Larger changes (~200 LOC, deferred):** N-trial evaluation for high-variance models; per-provider concurrency caps; auto-detect-and-pull missing Ollama models.

### Architecture

No architectural changes proposed. All work fits within existing components:

- `internal/eval_harness/models.yml` — per-model config knobs
- `cmd/ailang/eval_parallel.go` — concurrency control
- `cmd/ailang/eval_suite.go` — flag wiring
- `internal/executor/opencode/opencode.go` — idle/hard timeouts
- `internal/ai/ollama/client.go` — connection check, error messages

### Implementation Plan

**Phase 1 (overnight 2026-05-22 → 2026-05-23):** Measurement-only. No code changes.

- [x] Capture serial baseline (fizzbuzz + adt_option, `-parallel 1`)
- [ ] Capture parallel-2 result (same benchmarks, `-parallel 2`)
- [ ] Capture parallel-4 result (same benchmarks, `-parallel 4`)
- [ ] Capture parallel-2 with `OLLAMA_NUM_PARALLEL=2` explicitly set
- [ ] Run smoke tier (15 benchmarks) at best discovered setting
- [ ] Investigate variance: 3 sequential runs of fizzbuzz, log token counts + outcomes

**Phase 2 (post-measurement, 1 day):** Trivial config fixes.

- [ ] Apply timeout pattern to all Ollama models in `models.yml`
- [ ] Document recommended `OLLAMA_*` env vars in a new `docs/docs/guides/evaluation/local-ollama.md`
- [ ] Update `model-manager` skill to include local-Ollama smoke recipe

**Phase 3 (post-measurement, 1–2 days):** `-agent-parallel` resolution + optional warning.

- [ ] Decision: wire or delete `-agent-parallel`. (Recommendation: delete; the outer `-parallel` is sufficient.)
- [ ] If deleting: remove flag, update CHANGELOG, update tests
- [ ] If wiring: implement per-CLI semaphore in `internal/eval_harness/agent_runner.go`
- [ ] Optionally: warn if `-parallel N` > Ollama's configured `NUM_PARALLEL` for Ollama-provider models

**Phase 4 (deferred, separate design):** N-trial best-of-N evaluation.

- Probably belongs in its own milestone (M-EVAL-MULTITRIAL or similar).

### Files to Modify

| File | LOC delta | Why |
|---|---|---|
| `internal/eval_harness/models.yml` | ~30 (config) | Bump timeouts on all Ollama entries |
| `cmd/ailang/eval_suite.go` | -10 or +30 | Delete or wire `-agent-parallel` |
| `internal/eval_harness/agent_runner.go` | -1 or +20 | Field deletion or semaphore |
| `docs/docs/guides/evaluation/local-ollama.md` | +150 (new) | User-facing guide |
| `.claude/skills/model-manager/SKILL.md` | +30 | Local-Ollama smoke recipe |
| `Makefile` and/or `make/eval.mk` | +5 | Optional: `eval-local` shortcut target |

**Total estimated impact:** ~250 LOC. Mostly config + docs.

## Conflict Surface

This is not a parser/typechecker/codegen change, so no conflict surface in the language-semantics sense. However:

**eval-suite flag surface:**

| Position | Existing meaning | This change |
|---|---|---|
| `-parallel N` | Outer semaphore for all benchmark concurrency | Unchanged |
| `-agent-parallel N` | Currently a no-op | Either wired (per-CLI sub-semaphore) or removed |
| `-agent-timeout S` | Per-benchmark hard cap (agent mode) | Unchanged |
| Per-model `generation_timeout` | Opencode hard timeout | Bumped for Ollama entries |
| Per-model `ttft_timeout` | First-byte budget | Bumped for Ollama entries |

**Existing benchmark specs that already set timeouts:**
- `benchmarks/csv_to_json_converter.yml` has `timeout: 90s` baked in — overrides per-model defaults. Not touched here.

**Programs that must still work post-change:**
1. `make eval-suite` with default flags — full-suite cloud-model runs
2. `make eval-smoke MODELS=claude-sonnet-4-6 EXTRA='-agent'` — current happy path
3. `make eval-baseline EVAL_VERSION=v0.21.0` — release workflow
4. The 6 existing `ollama-*` model entries (`ollama-codellama`, `ollama-deepseek-coder`, `ollama-qwen-coder`, `ollama-llama3-2`, `ollama-gemma3`, `ollama-granite3-2-vision`) — already broken by the `GuessProvider` dispatcher bug fixed in this branch; the timeout bump for Ollama entries restores them.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Doesn't change determinism story; agent-mode is already stochastic by nature |
| A2: Replayability | 0 | Trace recording unchanged |
| A3: Effect Legibility | 0 | No effect-system changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | Type system unchanged |
| A6: Safe Concurrency | +1 | Fixing the dead `-agent-parallel` field removes a footgun |
| A7: Machines First | +1 | Reduces friction for AI-driven continuous eval |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Local-Ollama eval is $0; doc captures the trade-off explicitly |
| A10: Composability | 0 | Existing executor contract unchanged |
| A11: Structured Failure | +1 | Timeout fixes turn opaque hangs into clear "model thrashed" signals |
| A12: System Boundary | 0 | Ollama already a system boundary |

**Net Score: +4** → **Proceed to implementation** after measurement phase concludes.

**Hard violation check:** None. No `-1` on A1/A3/A4/A7.

## Success Criteria

- [ ] `make eval-smoke MODELS=opencode-gemma4-26b -agent` works out-of-the-box on a fresh 128 GB Mac Studio with no manual flag tuning beyond `-parallel`.
- [ ] `-agent-parallel` either does what it says, or doesn't exist.
- [ ] Three consecutive runs of fizzbuzz against gemma4:26b return matching outcome ≥2/3.
- [ ] Overnight smoke tier (~15 benchmarks) completes within 4 hours wall-clock on this rig.
- [ ] `docs/docs/guides/evaluation/local-ollama.md` exists and matches reality.
- [ ] `models.yml` Ollama entries have appropriate timeouts.
- [ ] CHANGELOG entry for whatever ships.

## Timeline

| Day | Work |
|---|---|
| 2026-05-22 night | Measurement phase: parallelism sweep, variance characterization, smoke-tier overnight |
| 2026-05-23 AM | Review results, finalize design decisions in the checkboxes above |
| 2026-05-23-24 | Phase 2: config fixes + docs |
| 2026-05-25 | Phase 3: `-agent-parallel` resolution |
| 2026-05-26 | Buffer / CHANGELOG / release prep |

## Related Documents

From auto-search (neural + simhash):

- [m-arch3-task-classification](v0_13_0/m-arch3-task-classification.md) — task classification architecture (low relevance to this design)
- [m-arch5-error-handling-strategy](v0_13_0/m-arch5-error-handling-strategy.md) — error categorization in eval results
- [m-locobench-long-context-benchmark](v0_13_0/m-locobench-long-context-benchmark.md) — long-context benchmark methodology

Higher-relevance manual references:

- `internal/executor/opencode/README.md` — opencode + Ollama provider config (already documents the `~/.config/opencode/opencode.jsonc` block)
- `.claude/skills/model-manager/SKILL.md` — smoke-test gate (3-benchmark cross-tier rule), used as the standard for assessing new models
- [M-EVAL-COST-AND-SPEED-BUDGETS (v0.15.1)](../implemented/v0_15_1/) — precedent for tuning per-benchmark hard timeouts
- [M-MOTOKO-EXECUTOR-ADAPTER (v0.18.0)](../implemented/v0_18_0/) — context on why `motoko-*` entries route via OpenRouter, not local Ollama (motoko itself is OpenRouter-only by design)

## Concrete Implementation Diffs (Phase 2 + 3)

### Phase 1 (NEW): Fix the budget-vs-spec-timeout precedence bug (3 lines)

**Discovered 2026-05-22**: the eval suite already has cost/wall-clock budget infrastructure (`Budgets.HardTimeoutSecs`, `ResolvedHardTimeoutSecs()`, M-EVAL-COST-AND-SPEED-BUDGETS v0.15.1/v0.16.0). It works for cloud models because they have cost as the primary gate. It's broken for local Ollama because:

1. Local Ollama models have `pricing: 0` → cost budget is `min($0.50, 0×... + 0×...) = 0` → cost never enforced
2. The wall-clock `HardTimeoutSecs` could be the gate, BUT...
3. Every benchmark YAML has a `timeout:` field (cloud-tuned at 90–180s), and the precedence chain at [agent_runner_multi.go:190-198](../../internal/eval_harness/agent_runner_multi.go#L190-L198) lets `spec.Timeout` **veto** the model's `HardTimeoutSecs`.

The comment in the code even says it explicitly:

> `Per-benchmark spec.Timeout (e.g. csv_to_json's 180s override) wins.`

That precedence was correct for cloud models (where individual benchmarks legitimately had per-task budgets in mind). It's wrong for local Ollama where the *model* knows its own slowness, not the benchmark spec.

**The fix is 3 lines** — replace "spec.Timeout vetoes" with "spec.Timeout and model.HardTimeoutSecs are floors, take the max":

```go
// agent_runner_multi.go around line 194-198
// BEFORE:
if spec.Timeout == 0 {
    if hardSecs := cfg.ResolvedHardTimeoutSecs(); hardSecs > 0 {
        task.Timeout = time.Duration(hardSecs) * time.Second
    }
}

// AFTER:
specT := spec.Timeout
hardSecs := cfg.ResolvedHardTimeoutSecs()
effective := specT
if hardSecs > effective {
    effective = hardSecs
}
if effective > 0 {
    task.Timeout = time.Duration(effective) * time.Second
}
```

Same change at [agent_runner_streaming.go:94-95](../../internal/eval_harness/agent_runner_streaming.go#L94-L95).

**Behavioral impact:**

| Model | Has model budget? | Has spec timeout? | Before fix | After fix |
|---|---|---|---|---|
| Cloud Sonnet 4.6 / csv_to_json | yes (600s) | yes (180s) | 180s wall, $0.30 cost gate | 600s wall, $0.30 cost gate (no regression — cost still primary) |
| Cloud Sonnet 4.6 / balanced_parens | yes (600s) | yes (90s) | 90s wall, $0.30 cost gate | 600s wall, $0.30 cost gate (cost still primary) |
| Local gemma4:26b / csv_to_json | yes (settable to 1800s) | yes (180s) | 180s wall (killed local) | 1800s wall (local can iterate) |
| Local gemma4:26b / balanced_parens | yes (1800s) | yes (90s) | **90s wall (killed local)** | 1800s wall (local can iterate) |
| Any model / unknown benchmark | no budget set | no spec timeout | falls back to CLI default | unchanged |

No regression on cloud — cost was already the primary gate; wall-clock was the safety net, and bumping the safety net higher doesn't change behavior because cost trips first.

**Apply with**: bump `opencode-gemma4-26b.budgets.hard_timeout_secs` to 1800 in `models.yml`. Then the fix takes effect.

**Estimated diff**: 3 lines in agent_runner_multi.go + 3 lines in agent_runner_streaming.go + 4 lines in models.yml. Plus a regression test that proves cloud cost-gate still trips first.

### Phase 2: Apply timeout pattern to all Ollama models in `models.yml`

For every entry with `provider: "ollama"`, ensure these fields are set (currently most are missing them, which means they fall through to cloud-tuned defaults):

```yaml
max_output_tokens: 8192     # higher than 4096 default; thinking models need headroom
ttft_timeout: 480           # first-byte budget — model loads + thinks before first token
generation_timeout: 600     # hard timeout for whole run — local thinking is slow
```

Existing Ollama entries needing this update (per `grep "provider: \"ollama\"" models.yml`):

- `ollama-codellama` — currently has none (will be deleted with Llama drop)
- `ollama-deepseek-coder` — currently has none
- `ollama-qwen-coder` — currently has none
- `ollama-llama3-2` — will be deleted with Llama drop
- `ollama-gemma3` — currently has none
- `ollama-gemma4-26b` — already updated in this branch
- `ollama-granite3-2-vision` — currently has none (vision-only; may not need eval)
- `ollama-paddleocr-vl` — vision-only
- `opencode-gemma4-e4b` — partial (`ttft_timeout: 300`, `generation_timeout: 120` — needs bump)
- `opencode-gemma4-26b` — already updated in this branch

**Estimated diff**: ~40 LOC in `internal/eval_harness/models.yml`. No code changes.

### Phase 3: Delete the dead `-agent-parallel` flag

Five-call-site removal:

```diff
--- a/cmd/ailang/eval_suite.go
+++ b/cmd/ailang/eval_suite.go
@@ -119,7 +119,6 @@
   agent := fs.Bool("agent", false, "Use agent-based evaluation (Claude Code or Gemini CLI)")
   agentModel := fs.String("agent-model", "", "Override agent CLI model ...")
-  agentMaxConcurrent := fs.Int("agent-parallel", 10, "Max concurrent agent sessions (agent mode only)")
   agentRequestsPerSecond := fs.Int("agent-rate", 1, "API requests per second (agent mode only)")
   agentTimeout := fs.Int("agent-timeout", 60, "Timeout per benchmark in seconds (agent mode only)")
@@ -525,7 +524,6 @@
       fmt.Printf("🤖 Agent mode ENABLED\n")
       fmt.Printf("  - Models: %v\n", modelNames)
       fmt.Printf("  - Agent CLI model: ...\n")
-      fmt.Printf("  - Parallel sessions: %d\n", *agentMaxConcurrent)
       fmt.Printf("  - Rate limit: %d req/sec\n", *agentRequestsPerSecond)
@@ -565,7 +563,6 @@
       agentConfig = &eval_harness.AgentBenchmarkConfig{
-          MaxConcurrent:      *agentMaxConcurrent,
           RequestsPerSecond:  *agentRequestsPerSecond,
           TimeoutSeconds:     *agentTimeout,

--- a/internal/eval_harness/agent_runner.go
+++ b/internal/eval_harness/agent_runner.go
@@ -17,7 +17,6 @@
 type AgentBenchmarkConfig struct {
-    MaxConcurrent      int           // Max parallel Claude sessions
     RequestsPerSecond  int           // API rate limit
     TimeoutSeconds     int           // Timeout per benchmark
@@ -34,7 +33,6 @@
 func DefaultAgentConfig() AgentBenchmarkConfig {
     return AgentBenchmarkConfig{
-        MaxConcurrent:     10,
         RequestsPerSecond: 1,
         TimeoutSeconds:    300,
```

**CHANGELOG entry:**

```markdown
### Removed

- `ailang eval-suite --agent-parallel` flag. It was never actually wired to gate
  concurrency; only `--parallel` controls concurrency in both standard and agent
  modes. Anyone using `--agent-parallel N` should switch to `--parallel N`.
```

**Estimated diff**: ~10 LOC removed across two files. Tests already don't depend on `MaxConcurrent` (verified by `grep -rn MaxConcurrent internal/eval_harness/*_test.go` returning nothing).

**Alternative considered (and rejected):** Wire `MaxConcurrent` to a per-CLI sub-semaphore so `--parallel` controls overall benchmark concurrency and `--agent-parallel` further caps agent sessions. Rejected because:
1. We have no current evidence that a sub-semaphore is needed — the outer `--parallel` is sufficient.
2. Two-level concurrency control is harder to reason about; a single knob (`--parallel`) is clearer.
3. Per-provider caps (option 3 in the original refactor list) would be more useful than per-CLI caps if we ever need fine-grained control — and that's a deferred decision in its own right.

## Recommended Configuration (provisional, pending smoke-tier validation)

Based on the 6 measurement runs to date, here's what the rotation should use. Will be finalized once the 17-benchmark smoke tier completes.

### Recommended command for canonical local-Ollama smoke

```bash
make eval-smoke \
  MODELS=opencode-gemma4-26b \
  EXTRA='-agent -langs ailang \
    -benchmarks fizzbuzz,adt_option,balanced_parens,binary_tree_sum,canonical_convergence,canonical_normalization,dense_operator_program,explicit_state_threading,gcd_lcm,immutable_data_structures,inline_tests,nested_records,numeric_modulo,record_update,records_book,recursion_fibonacci,type_safe_record_access \
    -output eval_results/rotation/$(date +%Y-%m-%d)/$(date +%H%M)_gemma4-26b_smoke \
    -parallel 4 \
    -agent-timeout 1800'
```

### Recommended Ollama env config

`launchctl setenv` or in the Ollama.app preferences:

```bash
# Default values for 128 GB M4 Max running gemma4:26b (and similar 26–32B class)
launchctl setenv OLLAMA_MAX_LOADED_MODELS 1     # one model resident; weights reuse
launchctl setenv OLLAMA_NUM_PARALLEL 4          # matches our eval -parallel default
launchctl setenv OLLAMA_MAX_QUEUE 64            # back-pressure if eval queue gets ahead
```

### Recommended models.yml deltas

For all `provider: "ollama"` entries currently in `models.yml`, add (or update):

```yaml
# Apply to: ollama-codellama (DELETE per Llama drop),
#           ollama-deepseek-coder, ollama-qwen-coder,
#           ollama-llama3-2 (DELETE), ollama-gemma3, ollama-granite3-2-vision,
#           ollama-paddleocr-vl, opencode-gemma4-e4b,
#           opencode-gemma4-26b (already updated)
budgets:
  hard_timeout_secs: 1800   # 30 min — wall-clock safety net for local thinking models
max_output_tokens: 8192
ttft_timeout: 480
generation_timeout: 600
```

(Cost budget is implicit: free local models have `pricing: 0`, so `ResolvedMaxCostUSD()` returns 0 — wall-clock is the only cap.)

### Continuous rotation: launchd plist (v1, no daemon needed)

`~/Library/LaunchAgents/dev.ailang.eval-rotation-gemma4.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>dev.ailang.eval-rotation-gemma4</string>
    <key>WorkingDirectory</key>
    <string>/Users/voightkampff/dev/sunholo-data/ailang</string>
    <key>ProgramArguments</key>
    <array>
        <string>/bin/bash</string>
        <string>-lc</string>
        <string>cd /Users/voightkampff/dev/sunholo-data/ailang &amp;&amp; PATH=$HOME/go/bin:/opt/homebrew/bin:$PATH make eval-smoke MODELS=opencode-gemma4-26b EXTRA="-agent -langs ailang -benchmarks fizzbuzz,adt_option,balanced_parens,binary_tree_sum,canonical_convergence,canonical_normalization,dense_operator_program,explicit_state_threading,gcd_lcm,immutable_data_structures,inline_tests,nested_records,numeric_modulo,record_update,records_book,recursion_fibonacci,type_safe_record_access -output eval_results/rotation/$(date +%Y-%m-%d)/$(date +%H%M)_gemma4-26b_smoke -parallel 4 -agent-timeout 1800" &gt;&gt; /tmp/ailang_eval_rotation.log 2&gt;&amp;1</string>
    </array>
    <key>StartCalendarInterval</key>
    <dict>
        <key>Hour</key>
        <integer>0</integer>
        <key>Minute</key>
        <integer>0</integer>
    </dict>
    <key>StandardOutPath</key>
    <string>/tmp/ailang_eval_rotation.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/ailang_eval_rotation.log</string>
</dict>
</plist>
```

Load with: `launchctl load ~/Library/LaunchAgents/dev.ailang.eval-rotation-gemma4.plist`

For multi-model rotation, duplicate this plist with different Label, different Hour/Minute, and different `MODELS=...`. See the rotation manifest example earlier in this doc.

### Footgun: agent mode requires explicit `--benchmarks`

Discovered while running this investigation: `ailang eval-suite --agent` refuses to run without explicit `--benchmarks list` ("Agent mode is expensive and time-consuming. You must explicitly specify..."). This is correct behavior for accidentally-launched cloud runs, but it's a paper cut for the daily rotation. The plist above handles this by computing the list inline. A future `ailang eval-rotation` daemon would handle this more cleanly.

## Monitoring & Observability (Phase 4)

**→ Split into sibling milestone: [M-EVAL-LOCAL-OBSERVABILITY](v0_22_0/m-eval-local-observability.md)**

The 2026-05-22 investigation found a concrete bug (FK constraint at the OTLP receiver) that drops opencode's per-step spans before they land in observatory.db. Once that's fixed, plus chain-stage labeling and a new `ailang chains live <id>` command, the monitoring gap closes. That work has its own design doc because it's a separate scope with its own implementation plan (~350 LOC, 2–3 days).

Quick recap of what was found (full detail in the sibling doc):
- opencode emits OTEL spans (both AILANG-side `opencode.execute`/`opencode.step` and npm-side via `@effect/opentelemetry`)
- They reach our OTLP receiver
- They're rejected by SQLite FOREIGN KEY constraint on `spans.task_id REFERENCES tasks(id)` because eval-suite never inserts a parent `tasks` row
- Fix: ~15 LOC to set `task_id=NULL` instead of failing the INSERT

### Original Phase 4 content (now superseded by the sibling doc)

The 24/7 rotation needs in-flight visibility so we can tell "model is thinking" from "model is stuck" without `ps`+`tail -f` gymnastics. State of the world today:

### What works now

| Method | Command | Shows |
|---|---|---|
| **Chains (high-level)** | `ailang chains view --spans <id>` | 4 stages running, no per-benchmark labels |
| **Chains diagnose** | `ailang chains diagnose <id>` | Stage health; currently flags "No session ID recorded" as an issue |
| **opencode session DB** | `sqlite3 ~/.local/share/opencode/opencode.db ...` | Per-session message count = live turn count |
| **Ollama API** | `curl localhost:11434/api/ps` | VRAM, model status, expires-at |
| **Process state** | `ps -eo etime,command \| grep opencode` | Subprocess count + wall time |

### What doesn't work / would help

| Gap | Why it matters | Fix size |
|---|---|---|
| Observatory.db has 0 spans (telemetry disabled) | We'd see per-turn token counts, per-tool-call timing | **Run `make services-start`** to launch the local OTLP receiver; no code change |
| Chain stages don't have benchmark name attached | Can't tell which stage is fizzbuzz vs adt_option | ~15 LOC: pass benchmark_id into chain_stages.attributes when registering the stage |
| "No session ID recorded" warning on every active chain | Diagnose flags it as a problem; it's actually just a timing thing | ~5 LOC: write session_id to chain_stages as soon as opencode subprocess returns it |
| No `ailang chains live` command | Have to manually join SQLite tables | **~80 LOC** new subcommand: joins chain_stages + opencode messages + ollama /api/ps for one-page live view |
| Cloud Trace not configured | Can't see traces across machines if rig is ever clustered | Add to local launchd: `OTLP_GOOGLE_CLOUD_PROJECT=multivac-internal-prod` if we want it |

### Recommended monitoring config for the rotation (no code changes)

Start the local AILANG server + coordinator at boot (it runs the OTLP receiver):

```bash
make services-start                       # one-time, on boot
# Or via launchd:
# ~/Library/LaunchAgents/dev.ailang.server.plist (RunAtLoad: true, KeepAlive: true)
```

Then export to that local endpoint from eval-suite subprocesses by setting in shell init:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:1957/v1/traces
```

Once enabled, this gives us:
- Per-turn span timing
- Token counts per agent turn
- Tool-call breakdown
- Live `ailang trace list --hours 1` data

### Phase 4 proposed work (~150 LOC + a launchd plist)

1. **Wire benchmark_id into chain_stages** (~20 LOC) so `ailang chains view` shows what each stage is actually running
2. **`ailang chains live <id>`** (~80 LOC) — single-page TUI/text view that refreshes every 2 sec, showing: which benchmark in which slot, turn count, last DB-write age, VRAM, Ollama queue depth
3. **`ailang server` launchd plist** (~50 lines XML) so the OTLP receiver runs on boot
4. **`ailang trace status --local` flag** that doesn't require GCP credentials (currently the command errors with "GOOGLE_CLOUD_PROJECT not set" even if the local OTLP receiver is running)

None of this is required for the rotation to *function*. It's all "quality of life when we're running 16 evals/day and want to spot regressions fast."

## Open Questions

To be answered overnight:

1. What is the optimal `-parallel N` for gemma4:26b on M4 Max + 128 GB? Hypothesis: 4.
2. Does explicit `OLLAMA_NUM_PARALLEL=4` outperform the default auto-tuned value?
3. What is the per-run variance σ for fizzbuzz across N=3 trials?
4. Does `OLLAMA_MAX_LOADED_MODELS=2` enable productive multi-model parallelism, or does GPU compute serialize them anyway?
5. What is the largest concurrency level before per-benchmark wall time degrades >2× vs serial?
6. For smoke tier (15 benchmarks), what % pass agent-mode against local gemma4:26b? (Baseline: most OS models pass 0–2 of 3 in the model-manager smoke set.)

## Model Sizing Matrix (Mac Studio M4 Max, 128 GB unified memory)

What model classes the rig can practically run, with concurrency estimates. **Bold rows are measured on this exact box.** Unbolded rows are extrapolations from gemma4:26b memory profile + Ollama's reported VRAM use.

**Note**: Llama family (3.2/3.3, CodeLlama) is excluded — Meta has effectively stopped advancing its open models in favor of OpenAI/Anthropic partnerships, and recent benchmarks (May 2026) show Qwen / DeepSeek / Gemma / GLM consistently beating Llama 3.3 70B at smaller sizes. The Llama era of OS leadership is over; not worth our eval budget.

| Model class | Example tag (verified) | Disk (Q4) | Per-instance VRAM | Concurrent on 128 GB | Notes |
|---|---|---|---|---|---|
| 7–8B | `qwen2.5-coder:7b`, `granite-code:8b` | ~4–5 GB | ~6 GB | 15+ | Practical floor for AILANG eval |
| 14–16B | `phi4:14b`, `phi4-reasoning:14b`, `deepseek-coder-v2:16b`, `starcoder2:15b` | ~8–9 GB | ~12 GB | 8–10 | |
| 24B | `mistral-small:24b` | ~13 GB | ~18 GB | 5–6 | |
| **26B dense Q4** | **`gemma4:26b`** | **17 GB** | **25.76 GB measured** | **3–4** | Current baseline |
| 27–35B | `gemma3:27b`, `qwen3:32b`, `qwen3.5:35b`, `qwen2.5-coder:32b`, `granite-code:34b`, `nemotron-3-nano:30b` | ~15–20 GB | ~22–28 GB | 3–4 | Peer comparison set |
| 70B+ | `deepseek-r1:70b`, `qwen3:72b` (if released) | ~39 GB | ~48 GB | 2 (or 1 + headroom) | Force `MAX_LOADED_MODELS=1` |
| 235B–671B | `qwen3:235b-a22b`, `deepseek-r1:671b`, `deepseek-coder-v2:236b` | 130–370 GB | 140–390 GB | **0 (won't fit)** | Stays on cloud routing |

**Implication**: the 26B–32B range is the sweet spot for this rig — large enough to be a serious capability test, small enough to run 3–4 concurrent and still leave system headroom. 70B class is single-instance-only. 235B+ models stay on OpenRouter regardless.

**KV cache memory growth**: weights load once and are shared; the per-instance cost above is roughly weights + a 1–4 GB KV-cache budget for context. gemma4:26b at 4 concurrent benchmarks (8K–20K tokens each) = ~26 + 4×3 ≈ **38 GB total VRAM**, well within budget. The 39.6 GB free we observed *during* the current `-parallel 2` run confirms this.

## Candidate Model Rotation (verified from Ollama library + OpenRouter, May 2026)

Cross-referenced live with [ollama.com/library](https://ollama.com/library) and [openrouter.ai/api/v1/models](https://openrouter.ai/api/v1/models) on 2026-05-22. Earlier draft included models that don't exist on Ollama (GLM, MiniMax, Kimi are cloud-only) — corrected below.

### Tier 1: Direct peer comparison to gemma4:26b (15–30 GB class — runs 2–4 concurrent on this rig)

| Model | Ollama tag | Disk | Why this one |
|---|---|---|---|
| Gemma 4 26B (current) | `gemma4:26b` | 17 GB | Baseline |
| **Gemma 4 31B** | `gemma4:31b` | ~18 GB | Larger sibling; tests size-vs-capability in same family |
| Gemma 3 27B | `gemma3:27b` | ~15 GB | Prior-gen comparison |
| Qwen 2.5 Coder 32B | `qwen2.5-coder:32b` | ~18 GB | Coding-specialized; strongest contender in class |
| Qwen3 32B | `qwen3:32b` | ~20 GB | Newer Qwen with reasoning + tools |
| Qwen3.5 35B | `qwen3.5:35b` | ~20 GB | Newest Qwen (May 2026 release) |
| Mistral Small 24B | `mistral-small:24b` | ~13 GB | Efficient performer |
| Nemotron 3 Nano 30B | `nemotron-3-nano:30b` | ~17 GB | NVIDIA's agentic-tuned model |
| Granite Code 34B | `granite-code:34b` | ~19 GB | IBM, code-specialized |

### Tier 2: Large (70B class — single-instance only on this rig)

| Model | Ollama tag | Disk | Notes |
|---|---|---|---|
| DeepSeek R1 70B | `deepseek-r1:70b` | ~39 GB | Reasoning-tuned DeepSeek |
| Qwen3 72B | `qwen3:72b` (if released; check) | ~40 GB | Largest Qwen3 dense |

These force `MAX_LOADED_MODELS=1` and cap at ~2 concurrent inference slots. Useful for "does scale alone matter for AILANG?" question. (Llama 3.3 70B excluded — see note above.)

### Tier 3: Small (7–14B — fast iteration, lots of concurrency, weaker results)

| Model | Ollama tag | Disk | Notes |
|---|---|---|---|
| Qwen 2.5 Coder 7B | `qwen2.5-coder:7b` | 4 GB | Already in `models.yml` as `ollama-qwen-coder` |
| DeepSeek Coder v2 16B | `deepseek-coder-v2:16b` | ~9 GB | Newer than `deepseek-coder:6.7b` we have |
| Phi-4 14B | `phi4:14b` | ~8 GB | Microsoft, frontier-class for its size |
| Phi-4 Reasoning 14B | `phi4-reasoning:14b` | ~8 GB | Reasoning variant; relevant for AILANG's typed errors |
| Starcoder 2 15B | `starcoder2:15b` | ~8 GB | Code-specialized |
| Granite Code 8B | `granite-code:8b` | ~5 GB | IBM smaller variant |

(CodeLlama excluded — Llama family is dead.)

### Tier 4: Cloud-only OS (already curated in models.yml via OpenRouter — NOT on Ollama)

These cannot run locally; they remain on OpenRouter routing. Listed for completeness — they're the cloud OS peers gemma4:26b is competing against:

| Model | OR ID | Status in current AILANG suite |
|---|---|---|
| GLM 5 (z.ai) | `z-ai/glm-5` | PASS 3/3 smoke (`opencode-or-glm-5`) |
| GLM 5.1 (newer, May 2026) | `z-ai/glm-5.1` | **NOT YET in models.yml** — should add |
| GLM 4.7 Flash | `z-ai/glm-4.7-flash` | PASS 3/3 with budgets (`opencode-or-glm-4-7-flash`) |
| MiniMax M2.7 | `minimax/minimax-m2.7` | PASS 3/3 (`opencode-or-minimax-m2-7`) |
| DeepSeek V4 Flash | `deepseek/deepseek-v4-flash` | NEAR-MISS, but **free tier exists** (`:free` suffix) |
| Kimi K2.6 | `moonshotai/kimi-k2.6` | NEAR-MISS |
| Qwen 3.6 35B-A3B (MoE, newer than 3.5) | `qwen/qwen3.6-35b-a3b` | **NOT YET in models.yml** — should add |

**Action items for cloud OS rotation** (independent of local-Ollama work):
- Add `opencode-or-glm-5-1` (GLM 5.1 supersedes GLM 5)
- Add `opencode-or-qwen3-6-35b-a3b` (newest Qwen MoE)
- Consider switching `opencode-or-deepseek-v4-flash` to the `:free` tier for cost-free baseline

### Local-rig comparison strategy

Each Tier 1 candidate gets the same 3-benchmark smoke gate (fizzbuzz, adt_option, csv_to_json_converter) at the optimal `-parallel N` we discover from the gemma4:26b sweep. Output: a table comparing local-OS models in agent-mode against the cloud-OS baselines we've already characterized.

**User-driven**: I will NOT pull these without explicit go-ahead — Tier 1 alone is ~150 GB of disk and several hours of downloads. Listed here for discussion.

## Quantization Strategy

Ollama defaults to Q4_K_M for most "default tag" pulls (e.g. `ollama pull gemma4:26b` gives you Q4_K_M without asking). This is the right default for 99% of our eval work, but worth being explicit about when *not* to use it.

| Quant | VRAM (vs FP16) | Quality loss | When to use |
|---|---|---|---|
| **Q4_K_M (default)** | **~25% of FP16** | Small (~1–3%) | **Default for all eval work.** Best perf/quality ratio on Apple Silicon. |
| Q5_K_M | ~31% of FP16 | Minimal (~0.5–1%) | When Q4_K_M smoke is borderline 2/3 and you suspect quantization is the cause |
| Q8_0 | ~50% of FP16 | Near-zero | Sanity check: "is this model really capable of solving AILANG benchmark X?" |
| FP16/BF16 | 100% | Reference | Only when investigating quantization-specific degradation |
| Q3_K_M / Q2_K | ~19% / 12% | Substantial (~5–15%) | Avoid for eval; not enough capacity |

**Rule**: AILANG eval rotation is Q4_K_M unless we have a specific reason to bump higher. The point of running OS models is to know how they'd perform in the field, and the field uses Q4_K_M. If a benchmark fails at Q4 we may quickly verify against Q8 to attribute (model capability vs quantization), then return to Q4 for the rotation.

**Disk math at Q4_K_M** for the full Tier 1 rotation (9 models): ~17 GB × 9 ≈ **150 GB**. Easy on a 2 TB Mac Studio. At Q8: ~300 GB. Still fine.

## Continuous Eval Rotation (the 24/7 plan)

### Rotation manifest format

Proposed `eval_rotation.yml` shape (one file checked into the repo, used by both the human-facing docs and the daemon):

```yaml
# eval_rotation.yml — canonical AILANG continuous-eval rotation
# Read by: ailang eval-rotation (daemon) or launchd plist
# Updated when: new models added/removed, schedule tuned, or pause needed

version: 1

defaults:
  langs: [ailang]
  parallel: 4                # discovered optimum from M-EVAL-LOCAL-OLLAMA
  agent_timeout: 700         # per-benchmark hard cap
  output_root: eval_results/rotation
  # Output directory layout: <output_root>/<YYYY-MM-DD>/<HHMM>_<model>_<tier>/

# Daily rotation: each row runs once per day at the given time
# All times in local timezone (`launchctl` default)
daily:
  - { time: "00:00", model: opencode-gemma4-26b,       tier: smoke }
  - { time: "00:30", model: opencode-qwen3-32b,        tier: smoke }
  - { time: "01:00", model: opencode-qwen2-5-coder-32b, tier: smoke }
  - { time: "01:30", model: opencode-gemma3-27b,       tier: smoke }
  - { time: "02:00", model: opencode-mistral-small-24b, tier: smoke }
  - { time: "02:30", model: opencode-granite-code-34b, tier: smoke }
  - { time: "03:00", model: opencode-phi4-14b,         tier: smoke }
  - { time: "03:30", model: opencode-nemotron-30b,     tier: smoke }
  - { time: "04:00", model: opencode-deepseek-coder-v2-16b, tier: smoke }
  - { time: "04:30", model: opencode-starcoder2-15b,   tier: smoke }
  - { time: "05:00", model: opencode-gemma4-26b,       tier: core }
  - { time: "05:45", model: <yesterday_top_2nd>,       tier: core, dynamic: true }
  - { time: "07:00", model: opencode-gemma4-26b,       tier: stretch }
  - { time: "08:00", model: opencode-gemma4-26b,       tier: variance, repeats: 5, benchmarks: [fizzbuzz, adt_option, csv_to_json_converter] }
  - { time: "20:00", model: opencode-deepseek-r1-70b,  tier: smoke, force_serial: true }

# Weekly rotation: heavier work, once a week
weekly:
  - { day: sunday, time: "10:00", model: opencode-gemma4-26b, tier: full }   # all 50 benchmarks
  - { day: sunday, time: "14:00", model: <yesterday_top_2nd>, tier: full }

# Pause control: set to true to halt the rotation without removing entries
paused: false

# Notification: if a smoke run drops below this pass-rate, send agent-inbox alert
alert:
  smoke_pass_rate_min: 0.5
  via: agent-inbox
```

### Schedule architecture

The rig should run unattended on a rotation. Proposed structure (subject to refinement after the parallelism sweep concludes):

```
Hour  Job                                    Models                       Duration
──────────────────────────────────────────────────────────────────────────────────
00:00 Smoke tier (15 benchmarks)             gemma4:26b                   ~30 min
00:30 Smoke tier                             qwen3:32b                    ~30 min
01:00 Smoke tier                             qwen2.5-coder:32b            ~30 min
01:30 Smoke tier                             gemma3:27b                   ~30 min
02:00 Smoke tier                             mistral-small:24b            ~30 min
02:30 Smoke tier                             granite-code:34b             ~30 min
03:00 Smoke tier                             phi4:14b (+ phi4-reasoning)  ~30 min
03:30 Smoke tier                             nemotron-3-nano:30b          ~30 min
04:00 Smoke tier                             deepseek-coder-v2:16b        ~30 min
04:30 Smoke tier                             starcoder2:15b               ~30 min
05:00 Core tier (~20)                        gemma4:26b (deep dive)       ~45 min
05:45 Core tier                              top-2 Tier-1 winner          ~45 min
07:00 Stretch tier (~8)                      gemma4:26b                   ~45 min
08:00 Variance characterization              random 3 benchmarks × N=5    ~60 min
09:00 [idle / catch-up / human-driven work]
...
20:00 Full nightly: 70B class                deepseek-r1:70b              ~3 hr
23:00 [idle until midnight rotation]
```

Daily throughput at this cadence: each Tier 1 model gets a fresh smoke result every day, two get deep-dive core runs, one gets stretch coverage, gemma4:26b gets variance data. After 7 days we have a full week-over-week trend for every model.

### Implementation options

1. **launchd** (macOS native) — `~/Library/LaunchAgents/dev.ailang.eval-rotation.plist`. Simplest. Sticky across reboots.
2. **A simple Go daemon** (`ailang eval-daemon`) that reads `eval_rotation.yml` and dispatches via existing `ailang eval-suite`. More structured; integrates with chains DB; can be paused/inspected via existing CLI.
3. **GitHub Actions on self-hosted runner** — overkill for a local-only rig, but if we ever want public visibility of results, this is the path.

**Recommendation**: launchd for v1 (no code changes needed), `ailang eval-rotation` for v2 (after the rotation is stable enough to justify the wrapper).

### `ailang eval-rotation` CLI sketch (v2)

```
ailang eval-rotation start [--manifest eval_rotation.yml] [--dry-run]
ailang eval-rotation status                      # next 5 scheduled runs + current pause state
ailang eval-rotation pause [--for 2h | --indefinite]
ailang eval-rotation resume
ailang eval-rotation history [--days 7]          # recent runs + outcomes
ailang eval-rotation logs <run-id>               # tail of a specific run
```

Sketch — actual implementation deferred until rotation has been validated by launchd in production for ≥2 weeks. Core loop:

```go
// internal/eval/rotation/daemon.go (sketch)
func (d *Daemon) Run(ctx context.Context) error {
    for {
        next := d.nextScheduledRun()
        wait := time.Until(next.Time)
        select {
        case <-time.After(wait):
            // Skip if paused
            if d.manifest.Paused { continue }
            d.dispatch(ctx, next)            // shells out to `ailang eval-suite`
        case <-ctx.Done():
            return ctx.Err()
        }
    }
}
```

Estimated implementation: ~300 LOC including manifest parsing, scheduler, dispatcher, and CLI.

### `ailang eval-trend` CLI sketch

For time-series analysis once data accumulates:

```
ailang eval-trend pass-rate --model opencode-gemma4-26b --benchmark fizzbuzz --days 30
ailang eval-trend cost --model opencode-gemma4-26b --benchmark fizzbuzz --days 30
ailang eval-trend wall-time --model opencode-gemma4-26b --tier smoke --days 30
ailang eval-trend compare --models opencode-gemma4-26b,opencode-qwen3-32b --tier smoke --days 7
ailang eval-trend regression --model opencode-gemma4-26b --baseline 2026-05-01  # flag recent regressions
```

Output is ASCII chart by default (suitable for terminal), optional `--format=json` or `--format=csv` for piping into other tools.

Estimated implementation: ~250 LOC including the chart rendering. Could lean on an existing Go library (e.g. `gonum/plot` for sparklines).

### Results storage

All eval-suite results already write to `eval_results/` directories with structured JSON. To turn this into time-series:

1. Each rotation run goes to `eval_results/rotation/YYYY-MM-DD/<hour>_<model>_<tier>/`
2. A new `ailang eval-trend` command (~100 LOC) aggregates across dates, model-or-benchmark filters, and produces:
   - Pass rate over time per (model, benchmark) pair
   - Token cost drift (does the model thrash more or less week-over-week?)
   - Wall-clock drift (is gemma4:26b getting faster? or our box getting slower?)
3. Eventually backed into the `ailang dashboard` UI as a "longitudinal" view alongside the existing baseline matrix.

## Time-Series Progress Tracking

Once the rotation is running, we get a daily data point per (model, benchmark). After 30 days that's enough to plot trends:

```
Pass rate of gemma4:26b on smoke tier over 30 days
    ┌──────────────────────────────────────────────┐
1.0 │                              ●               │
0.9 │                       ●  ●      ●  ●         │
0.8 │              ●  ●  ●                   ●  ●  │  ← stdlib v0.22 shipped
0.7 │     ●  ●                                     │
0.6 │  ●                                           │
    └──────────────────────────────────────────────┘
       day 0       day 15           day 30
```

Two improvement directions move this line up:
1. **Stdlib + prompt improvements** (our side) — visible as discrete step-ups on the date a change ships
2. **Newer model releases** (their side) — visible when we swap `gemma4:26b` → `gemma5:26b` in the rotation

The combined plot tells us which lever (ours vs theirs) is moving faster. Critical for roadmap prioritization: if every Gemma release jumps us +0.1 pass rate but stdlib changes give +0.03 each, we should focus on model-rotation hygiene; if it's the inverse, double down on stdlib.

## Fine-Tuning Data Foundation

Every eval run produces a tuple worth keeping:

```
{
  "model": "gemma4:26b",
  "benchmark_id": "fizzbuzz",
  "prompt": "<full teaching prompt + task>",
  "generated_code": "module fizzbuzz\n...",
  "compile_ok": true,
  "runtime_ok": true,
  "stdout_ok": false,
  "error_category": "logic_error",
  "agent_turns": 12,
  "tool_calls": [...],
  "total_tokens": 113000
}
```

After 6–12 months of continuous eval, this becomes a high-quality fine-tuning dataset:

- **Positive examples**: successful runs become "good" examples for SFT
- **Negative examples + error categories**: failed runs become RLHF/DPO pairs (the model output vs a corrected version)
- **Agent trajectories**: multi-turn opencode sessions are exactly the format a fine-tuned agent model needs

**Initial schema decisions to make now (so the data is fine-tuning-ready):**

- [ ] Always log the full system prompt + user prompt, not just a hash
- [ ] Always log the full agent transcript (turns, tool calls, tool results), not just the final answer
- [ ] Always log the canonical "correct answer" alongside (so DPO pairs are constructable)
- [ ] Tag each run with prompt version, stdlib version, model version, hardware fingerprint

Most of these are already in our chains/observatory schema. Worth auditing once the rotation is running.

The fine-tuning dataset doesn't need to be built up-front; it's a free byproduct of running the rotation. We just need to not throw the data away.

## Time-to-Completion Estimates

**Per-benchmark time on gemma4:26b agent-mode** (single benchmark, single concurrent slot):

| Benchmark class | Observed range | Likely median |
|---|---|---|
| Trivial (fizzbuzz, balanced_parens) | 2–5 min | 3 min |
| Easy (adt_option, simple records) | 4–8 min | 6 min |
| Medium (csv_to_json, contract benchmarks) | 6–15 min, often timeout | 10 min |
| Hard (Boyer-Moore, BST, graph algos) | not measured | est. 15+ min |

**Full-suite extrapolations on gemma4:26b** (assuming `-parallel 2` and ~6 min median):

| Suite | Benchmark count | Serial (-parallel 1) | -parallel 2 | -parallel 4 (projected) |
|---|---|---|---|---|
| 3-bench smoke gate | 3 | ~18 min | ~10 min | ~6 min |
| Smoke tier | 15 | ~90 min | ~50 min | ~30 min |
| Core tier | ~20 | ~2 hr | ~70 min | ~40 min |
| Full suite | ~50 | ~5 hr | ~2.5 hr | ~1.5 hr |
| Stretch tier (hard) | ~8 | ~2 hr | ~70 min | ~40 min |

**Implication for 24/7 operation**: at `-parallel 4` the full ~50-benchmark suite finishes in **~90 min**. Across a 24-hour day that's **~16 full-suite runs** of one model, or one full run across **~16 different OS models** in rotation. The rig can absorb the workload comfortably.

## Measurement Log

Live findings appended below as runs progress.

### 2026-05-22 ~15:30 (run 1, serial baseline, -parallel 1)

Command: `make eval-smoke MODELS=opencode-gemma4-26b EXTRA='-agent -langs ailang -benchmarks fizzbuzz,adt_option -output /tmp/agent_smoke_v2 -parallel 1 -agent-timeout 700 -agent-parallel 1'`

| Benchmark | Outcome | Wall | Tokens | Notes |
|---|---|---|---|---|
| adt_option | FAIL logic_error | 7m25s | 148K | compiled + ran, wrong stdout |
| fizzbuzz | FAIL compile_error | 3m57s | 2.6M (!) | model thrashed on parse errors |
| **Total** | 0/2 pass | **~11m** | 2.75M | gemma reloaded fresh between benchmarks |

Compared to first-ever clean fizzbuzz (1m57s, 298K tokens, PASS) — **same model, same seed, 13× token blowup on retry**. Variance dominates signal. This is the #1 finding tonight.

### 2026-05-22 ~15:51 (run 2, -parallel 2)

Command: `make eval-smoke MODELS=opencode-gemma4-26b EXTRA='-agent -langs ailang -benchmarks fizzbuzz,adt_option -output /tmp/agent_smoke_p2 -parallel 2 -agent-timeout 700'`

| Benchmark | Outcome | Wall | Tokens | Notes |
|---|---|---|---|---|
| adt_option | **PASS** | 6m49s | 113K | Compiled + correct stdout |
| fizzbuzz | **PASS** | 7m29s | 110K | Compiled + correct stdout |
| **Total wall** | **2/2 pass** | **7m30s** | **223K** | Concurrent execution |

**Comparison serial baseline → parallel-2:**

| Metric | Serial (-parallel 1) | Parallel-2 | Ratio |
|---|---|---|---|
| Total wall clock | ~11 min | 7m30s | **0.68× (1.47× faster)** |
| Per-benchmark time | 4–7 min | 6.8–7.5 min | ~1.3× slower per benchmark |
| Total tokens | 2.75M (thrashing) | 223K (clean) | 0.08× — **12× fewer** |
| Success rate | 0/2 | **2/2** | quality also improved |

### Answer: GPU is NOT a serialization bottleneck for same-model concurrency

The empirical result directly contradicts the "single GPU = serialized = no benefit" framing:
- Per-benchmark time only grew ~1.3× under 2× concurrency (not 2× as serialization would imply)
- Aggregate wall clock dropped 32%
- This matches **batched-inference theory**: Ollama feeds 2 concurrent request streams through the same gemma4:26b weights in one batched forward pass, getting near-linear aggregate throughput at modest per-request slowdown

**Caveat on the quality improvement (0/2 → 2/2):** likely confounded with variance. The serial baseline run hit a thrash mode where fizzbuzz blew 2.6M tokens on parse errors; the parallel-2 runs completed in normal turn counts. Cannot attribute the quality jump to parallelism per se until we have N=3+ replicates at each setting (Open Question 3, still queued).

### Implications for Open Question 1 (optimal -parallel N)

Strong evidence that `-parallel 2` beats `-parallel 1` on this rig for both throughput AND stability. `-parallel 4` is the next test — if it scales similarly, it should hit ~4–4.5 min total wall (vs 7m30s now). At that point we'll know whether 4 is the sweet spot or we should push higher.

### Open Questions Status

- [x] Open Question 1 partial answer: `-parallel 2` is strictly better than `-parallel 1`. Need `-parallel 4`/`-parallel 8` data to find ceiling.
- [ ] Open Question 2 (explicit `OLLAMA_NUM_PARALLEL` vs auto): still queued
- [ ] Open Question 3 (variance across N≥3 trials): still queued — critical to understanding the 0/2 vs 2/2 jump
- [ ] Open Question 4 (`MAX_LOADED_MODELS=2`): pending model rotation discussion
- [ ] Open Question 5 (per-benchmark wall-time degradation curve): partial — 1.3× at p=2, need data at p=4, p=8
- [ ] Open Question 6 (full smoke tier pass rate): pending

### 2026-05-22 ~16:00 (run 3, -parallel 4 — **methodological flaw, inconclusive**)

Command: `make eval-smoke MODELS=opencode-gemma4-26b EXTRA='-agent -langs ailang -benchmarks fizzbuzz,adt_option -output /tmp/agent_smoke_p4 -parallel 4 -agent-timeout 700'`

| Benchmark | Outcome | Wall | Tokens | Notes |
|---|---|---|---|---|
| adt_option | PASS | 7m10s | 147K | |
| fizzbuzz | PASS | 7m56s | 221K | |
| **Total wall** | 2/2 pass | **7m57s** | 368K | |

**Problem with this run:** with only 2 benchmarks in the queue, `-parallel 4` effectively behaves identically to `-parallel 2` — slots 3 and 4 sit idle the entire time. The wall-clock difference vs p=2 (7m30s → 7m57s, ~6%) is within run-to-run variance, not a parallelism effect.

**Variance observation (more evidence for Open Question 3):**

| Run | fizzbuzz tokens | adt_option tokens |
|---|---|---|
| Serial baseline | 2.6M (thrashed) | 148K |
| -parallel 2 | 110K | 113K |
| -parallel 4 | 221K | 147K |

Same task, same model, same seed (42), three wildly different token counts for fizzbuzz alone (110K to 2.6M, a 24× spread). This is the variance problem — single-run pass/fail is unreliable signal.

### 2026-05-22 ~16:11 (run 4, real -parallel 4 with 4 benchmarks)

Command: `make eval-smoke MODELS=opencode-gemma4-26b EXTRA='-agent -langs ailang -benchmarks fizzbuzz,adt_option,balanced_parens,binary_tree_sum -output /tmp/agent_smoke_p4_real -parallel 4 -agent-timeout 700'`

| Benchmark | Outcome | Wall | Tokens | Error |
|---|---|---|---|---|
| adt_option | **PASS** | 8m27s | 112K | — |
| fizzbuzz | **PASS** | 9m47s | 188K | — |
| balanced_parens | **TIMEOUT** | (90s spec) | 0 | benchmark spec hard cap |
| binary_tree_sum | **TIMEOUT** | (120s spec) | 0 | benchmark spec hard cap |
| **Total wall** | 2/4 pass | **9m48s** | — | |

### Critical new finding: benchmark-spec timeouts are cloud-tuned

The two failures were NOT model failures — they were killed by **per-benchmark `timeout` fields in the benchmark YAML files** (90s for `balanced_parens.yml`, 120s for `binary_tree_sum.yml`). These were tuned for Claude Sonnet 4.6 speeds. Local gemma4:26b in agent mode under p=4 concurrency cannot solve them in that window.

Precedent in the codebase: `csv_to_json_converter.yml` was already bumped 90s → 180s in May 2026 for OS models. Same pattern needs applying to the rest of the suite for local-OS eval to work.

**Adds to Phase 2 of the implementation plan: bump all benchmark-spec timeouts** so per-model-defaults can govern. Or, better, refactor benchmark spec timeouts to be a *floor* (cloud baseline) while a per-model multiplier kicks in for slow providers. Initial estimate: ~30 lines across 50 benchmark YAML files + a small change to how the spec timeout is read.

### Parallelism ceiling analysis

Per-benchmark wall time across the sweep:

| Concurrency | Per-benchmark median wall | Best case |
|---|---|---|
| p=1 (serial) | ~6 min (with high variance — sometimes thrashes to 4+ min/turn) | 1m57s (clean fizzbuzz) |
| p=2 | ~7 min | 6m49s (adt_option) |
| p=4 | ~9 min | 8m27s (adt_option) |

Per-benchmark slowdown vs serial: **~1.2× at p=2, ~1.5× at p=4**. This matches the predicted batched-inference curve (sub-linear cost growth).

**Aggregate throughput** (benchmarks completed per minute of wall clock, assuming no spec-timeout failures):

| Concurrency | Throughput (benchmarks/min) | Speedup vs serial |
|---|---|---|
| p=1 | 1/6 = 0.17 | 1.0× (baseline) |
| p=2 | 2/7.5 = 0.27 | 1.6× |
| p=4 | 4/10 ≈ 0.40 (projected if all 4 had completed) | 2.4× |

**Tentative conclusion**: `-parallel 4` IS the right setting for a 128 GB rig running gemma4:26b — provided benchmark-spec timeouts are bumped to accommodate slower per-request throughput. The aggregate wall-clock savings (2.4× over serial) outweigh the per-benchmark slowdown.

**`-parallel 8` worth testing next?** Maybe. The Ollama default is `NUM_PARALLEL=4`. Going higher than that would queue inside Ollama (no GPU benefit) unless we also raise the env var. Will test after the benchmark-spec timeout fix lands.

### 2026-05-22 ~16:41 (run 5, -parallel 4 + budget fix APPLIED)

After applying the 3-line precedence fix in `agent_runner_multi.go` + setting `opencode-gemma4-26b.budgets.hard_timeout_secs: 1800`, re-ran the same 4-benchmark p=4 test.

Command: `make eval-smoke MODELS=opencode-gemma4-26b EXTRA='-agent -langs ailang -benchmarks fizzbuzz,adt_option,balanced_parens,binary_tree_sum -output /tmp/agent_smoke_p4_fixed -parallel 4 -agent-timeout 1800'`

| Benchmark | Outcome | Wall | Tokens |
|---|---|---|---|
| adt_option | **PASS** | 14m21s | 110K |
| balanced_parens | **PASS** | 23m56s | 302K |
| binary_tree_sum | **PASS** | 24m38s | 349K |
| fizzbuzz | **PASS** | 16m42s | 110K |
| **Total wall** | **4/4 (100%)** | **24m39s** | 871K |

**Budget fix conclusively validated**: benchmarks that previously died at 90s/120s spec timeouts (balanced_parens, binary_tree_sum) now complete in 23–24 minutes of agent iteration and pass.

### Revised parallelism conclusion: p=2 likely better than p=4 on this rig

Per-benchmark wall-clock comparison (same workload):

| Benchmark | Tokens | p=2 wall | p=4 wall | Slowdown |
|---|---|---|---|---|
| adt_option | 110–113K | 6m49s | 14m21s | **2.1×** |
| fizzbuzz | 110K | 7m29s | 16m42s | **2.2×** |

Same token count → same actual work. The 2.1–2.2× per-request slowdown at p=4 means batched-inference efficiency on Apple Silicon is **significantly worse than the theoretical 1.5× I projected**. Aggregate throughput math:

| Setting | Per-benchmark | Aggregate (benchmarks/min) |
|---|---|---|
| p=2 | ~7 min | 2 / 7 = **0.29** |
| p=4 | ~15 min (for ~110K-token benchmarks) | 4 / 15 = **0.27** |

**p=4 is essentially flat with p=2 on aggregate throughput** — sometimes worse. The wall-clock per-benchmark slowdown wipes out the slot doubling. The Apple Silicon Metal/MLX backend in Ollama 0.24 doesn't batch as efficiently as NVIDIA CUDA stacks.

**Tentative recommendation**: `-parallel 2` is the sweet spot for gemma4:26b on this M4 Max. Maybe `-parallel 3` is worth testing, but `-parallel 4` is past the knee.

This contradicts my earlier projection in the time-to-completion table. Updating that section after the smoke-tier run confirms.

### 2026-05-22 ~17:08 (run 6, fair p=2 same 4 benchmarks)

Command: `make eval-smoke MODELS=opencode-gemma4-26b EXTRA='-agent ... -benchmarks fizzbuzz,adt_option,balanced_parens,binary_tree_sum -output /tmp/agent_smoke_p2_fair -parallel 2 -agent-timeout 1800'`

| Benchmark (order) | Outcome | Wall | Tokens |
|---|---|---|---|
| binary_tree_sum | PASS | 20m04s | 416K |
| fizzbuzz | **FAIL compile_error** | 6m51s | **1.76M (thrashed)** |
| adt_option | PASS | 13m11s | 254K |
| balanced_parens | PASS | 6m08s | 110K |
| **Total wall** | **3/4 (75%)** | **26m12s** | 2.54M |

### The variance finding is now the headline result

Three runs with overlapping benchmarks, all under the budget fix:

| Run | Setting | Wall | Pass rate | fizzbuzz tokens | fizzbuzz duration |
|---|---|---|---|---|---|
| #5 | p=4, 4 benchmarks | 24m39s | 4/4 | 110K | 16m42s |
| #6 | p=2, 4 benchmarks | 26m12s | 3/4 | **1,758K** | 6m51s → compile_error |
| (earlier #2) | p=2, 2 benchmarks | 7m30s | 2/2 | 110K | 7m29s |

**Same task, same seed (42), 16× token spread on fizzbuzz between runs.** The variance is the dominant signal. Single-run p=X comparisons cannot answer "what's optimal" — variance > parallelism effect.

### Token-rate observation suggests p=4 may be *protective* against thrashing

Tokens-per-second of generation by setting:

| Setting | fizzbuzz tokens/sec |
|---|---|
| p=4 (passed) | 110K / 1002s = **110 tok/s** |
| p=2 (thrashed) | 1,758K / 411s = **4280 tok/s** |

At p=4, each session gets a smaller GPU slice → physically can't generate tokens fast enough to "thrash 1.76M tokens of wrong-AILANG in 7 minutes." The forced slowdown acts as a *think-before-emitting* governor. Model has more time per token, fewer opportunities to spiral into rapid-fire compile errors.

**Tentative refined conclusion**: `-parallel 4` may be the right choice for two reasons:
1. Aggregate throughput roughly equivalent to p=2 (24–26 min for 4 benchmarks)
2. **Stability**: forces slower token rate, suppressing thrash-mode failures

This is a fascinating side effect. Needs N≥3 trials at each setting to confirm — but the directional evidence is real.

### 2026-05-22 ~18:28 (run 7, full smoke tier at p=4, 17 benchmarks)

Command: `make eval-smoke MODELS=opencode-gemma4-26b EXTRA='-agent -langs ailang -benchmarks <17 smoke benchmarks> -output /tmp/agent_smoke_tier_p4 -parallel 4 -agent-timeout 1800'`

**Wall clock: 1h17m55s. Headline pass rate: 7/17 (41.2%).**

| Benchmark | Outcome | Wall | Tokens | Real cause |
|---|---|---|---|---|
| adt_option | PASS | 16m41s | 147K | — |
| canonical_normalization | PASS | 14m13s | 111K | — |
| explicit_state_threading | PASS | 20m37s | 191K | — |
| gcd_lcm | PASS | 20m25s | 184K | — |
| immutable_data_structures | PASS | 14m06s | 110K | — |
| inline_tests | PASS | 23m44s | 223K | — |
| record_update | PASS | 13m17s | 110K | — |
| balanced_parens | FAIL | 0s | 0 | **TTFT 8m timeout** (prefill, hit at startup) |
| canonical_convergence | FAIL | 0s | 0 | TTFT 8m timeout |
| nested_records | FAIL | 0s | 0 | TTFT 8m timeout |
| records_book | FAIL | 0s | 0 | TTFT 8m + "non-agentic (1 turn, 0 tools)" |
| type_safe_record_access | FAIL | 0s | 0 | TTFT 8m timeout |
| binary_tree_sum | FAIL | 0s | 0 | Model-level timeout (likely 18m exceeded) |
| fizzbuzz | FAIL | 0s | 0 | Model-level timeout |
| recursion_fibonacci | FAIL | 0s | 0 | Model-level timeout |
| dense_operator_program | FAIL | 23m50s | 148K | **Real**: compile_error (genuine AILANG gap) |
| numeric_modulo | FAIL | 18m57s | 72K | **Real**: logic_error (wrong stdout) |

### The headline number is misleading

```
                     ┌──────────┬───────┐
True model failures  │ compile  │   1   │
                     │ logic    │   1   │
                     ├──────────┼───────┤
Infrastructure       │ TTFT     │   5   │  ← these can be fixed by bumping ttft_timeout
                     │ idle     │   3   │  ← bumping idle/hard timeout
                     │ no-agent │   1   │  ← model decided to one-shot
                     ├──────────┼───────┤
PASS                 │          │   7   │
                     └──────────┴───────┘
```

**If TTFT/idle timeouts were appropriately tuned for p=4 load, projected pass rate: ~14/17 = 82%.** That puts gemma4:26b at smoke-pass parity with cloud GLM 5 (3/3 on the small smoke gate) — a strong result for a free, local model.

### Critical lessons

1. **`ttft_timeout: 480` (8 min) is too tight at p=4 concurrency.** When 4 concurrent requests share the GPU, prefill (the initial forward pass through the prompt) takes longer because all 4 prefills compete. Need to bump to ~900s or higher, OR reduce concurrency.

2. **`hard_timeout_secs: 1800` (30 min) is borderline at p=4 for some benchmarks.** Three benchmarks hit it (fizzbuzz, recursion_fibonacci, binary_tree_sum). Bump to 2400 (40 min).

3. **The "non-agentic result" check is a footgun.** opencode rejects results where the model produced a complete solution in 1 turn without tool calls (gemma sometimes one-shots). This is fine for cloud models that always iterate, but local models with longer thinking phases sometimes skip the iteration. Should be a soft warning, not a hard fail.

4. **Per-benchmark wall time at p=4 ranges 13–24 min** (median ~16 min). For the rotation this means smoke tier = ~80 min per model. Doable, but suggests we want p=3 or p=2 for tighter SLAs.

### Updated recommended config

Bump per-model defaults for `opencode-gemma4-26b`:

```yaml
opencode-gemma4-26b:
  ttft_timeout: 900            # was 480 — p=4 prefill needs more
  generation_timeout: 1200     # was 600 — opencode hard timeout per session
  budgets:
    hard_timeout_secs: 2400    # was 1800 — fizzbuzz can need 40+ min at p=4
```

### 2026-05-22 ~19:50 (run 8, smoke tier with bumped timeouts)

Same 17 benchmarks at p=4 after bumping `ttft_timeout: 480→900`, `generation_timeout: 600→1200`, `hard_timeout_secs: 1800→2400`, `agent-timeout 1800→2400`.

**Result: 12/17 PASS (70.6%) in 1h41m43s** — recovered 5 of the 9 infra-failures from run 7.

| Benchmark | Run 1 (orig) | Run 2 (bumped) | Notes |
|---|---|---|---|
| adt_option | **PASS** 16m | **FAIL** compile | **flipped** — variance |
| balanced_parens | FAIL ttft | FAIL ttft | still infrastructure-killed; needs even bigger TTFT |
| binary_tree_sum | FAIL ttft | **PASS** 36m | recovered by hard_timeout bump |
| canonical_convergence | FAIL ttft | **PASS** 16m | recovered by ttft bump |
| canonical_normalization | PASS 14m | PASS 17m | stable |
| dense_operator_program | FAIL compile | FAIL compile (2.68M tokens!) | real gap — model thrashes hard |
| explicit_state_threading | PASS 21m | PASS 25m | stable |
| fizzbuzz | FAIL ttft | **PASS** 13m | recovered |
| gcd_lcm | PASS 20m | PASS 20m | stable |
| immutable_data_structures | PASS 14m | PASS 11m | stable |
| inline_tests | PASS 24m | **FAIL** ttft | **flipped** — was passing, now ttft |
| nested_records | FAIL ttft | **PASS** 12m | recovered |
| numeric_modulo | FAIL logic | **PASS** 13m | flipped to pass |
| record_update | PASS 13m | **FAIL** logic | flipped |
| records_book | FAIL ttft+non-agent | **PASS** 13m | recovered |
| recursion_fibonacci | FAIL ttft | **PASS** 12m | recovered |
| type_safe_record_access | FAIL ttft | **PASS** 12m | recovered |

### Variance is even worse than feared

**5 of 17 benchmarks flipped outcome between two consecutive runs** of the same model, same seed, same config. That's a 29% outcome-instability rate. Same model, same task, sometimes pass, sometimes fail with totally different failure modes.

| Type of flip | Count |
|---|---|
| PASS → FAIL | 3 (adt_option, inline_tests, record_update) |
| FAIL → PASS | 5 (binary_tree_sum, canonical_convergence, fizzbuzz, numeric_modulo, ...) |
| Stable | 9 |

This makes **single-trial pass rate unreliable as a signal**. For the rotation we need:
- N≥3 trials per benchmark for any reportable pass rate
- OR a wider success criterion ("passes at least once out of N")
- OR explicitly model the variance and report mean ± σ

### Stable failures (likely real model gaps)

Two benchmarks failed BOTH runs with the same failure mode:
- **dense_operator_program**: compile_error, both runs. 2.68M tokens of thrashing in run 2. Model can't synthesize correct AILANG for this task. Real gap worth investigating.
- **balanced_parens**: TTFT timeout, both runs. May be a benchmark-specific prompt length issue (could be a large input). Worth pulling apart separately.

### Monitoring reliability assessment (answering the user question)

| Signal | What it tells us | Reliable? |
|---|---|---|
| `ailang chains view <id>` | Stage count, status | ✓ at high level |
| `ailang chains view --spans` | Per-span detail | ✗ no spans (telemetry disabled) |
| `opencode.db` msg count | Live turn count | ⚠ misleading during long thinking phases |
| `ps` opencode subproc CPU% | "Working" signal | ✗ orchestrator is always ~0% |
| `ollama runner` CPU% | GPU activity | ✓ 20-30% = generating, <1% = stuck |
| Result file count | Completion count | ✓ reliable |
| `pgrep -fc opencode run` | Active session count | ✓ reliable |

**Honest take:** for a 24/7 rotation we are blind to per-benchmark progress between session start and completion. The two timeouts we hit (balanced_parens) could not be distinguished from "model is thinking really hard" until they expired. This must be fixed before the rotation goes live (see Phase 4 above — observatory.db with live spans).

### Path to reliable rotation

1. **Lock in current config** (timeouts bumped, budget precedence fixed). Pass rate stabilizes around 70%.
2. **Variance reduction**: switch from single-trial to N=3 with median-pass criterion. Doubles to triples run time but eliminates 5-of-17 outcome flips.
3. **Monitoring** (Phase 4 in this doc): enable OTLP receiver via `make services-start`, get per-turn span data, build `ailang chains live <id>` for in-flight visibility.
4. **Per-benchmark real-gap fixes**: dense_operator_program needs stdlib/prompt investigation (consistent compile errors).
