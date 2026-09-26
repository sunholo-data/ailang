# M-EVAL-TOKEN-HEADROOM: Equal Headroom, Visible Truncation, Labelled Thinking State

**Status**: Planned
**Target**: v0.31.0
**Priority**: P1 (Medium-High — blocks honest token-efficiency measurement across the suite)
**Estimated**: ~1 day (P1 ~1h, P2 ~2h, P3 ~4h incl. re-baseline annotation)
**Dependencies**: None. The metrics pipeline this relies on already exists (see V4–V6 below).

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Eval meta-tooling about *measurement fidelity*, not language semantics. The positive scores
concern reproducibility, machine-readable outcomes, and cost visibility.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Equal headroom removes a per-model confound; a run's outcome stops depending on an arbitrary per-entry cap |
| A2: Replayability | 0 | No change to trace capture |
| A3: Effect Legibility | 0 | No language-level effect changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | No verification-surface change |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | `finish_reason` becomes machine-readable on every standard row, so a downstream agent can separate "truncated" from "wrong" without reading prose |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | +1 | Token-efficiency comparisons become meaningful once headroom is equal and thinking state is labelled |
| A10: Composability | 0 | Reuses the existing metrics pipeline rather than adding a parallel one |
| A11: Structured Failure | +1 | Truncation stops masquerading as a capability failure — it becomes a typed, distinguishable outcome |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced — this *removes* one
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Strictly improves machine-readability of results

## Problem Statement

We are starting to measure **token efficiency** across the eval suite. Three defects make
those numbers uninterpretable today.

**Current State:**

1. **Truncation is invisible on the Anthropic rows.** `internal/ai/anthropic/client.go`'s
   `Generate` (the standard-eval path) returns an `ai.Response` with no `FinishReason`, so a
   run cut off at `max_tokens` is banked indistinguishably from a run that genuinely failed.
   The agent path does not have this gap — `step.go:136` already calls `mapStopReason`.
2. **`max_output_tokens` is wildly unequal** across `models.yml` — 26 entries at 32768,
   17 at 128000, 16 at 8192, 12+ at 65536, 10 at 64000, 3 at 16000, and **7 entries with no
   value at all** (falling back to the handler default of 4096). A model that thinks by
   default at 8192 headroom and one at 128000 are not running the same experiment.
3. **Default thinking state is unrecorded.** `models.yml` captures price, gate status and
   suite membership, but not "does this model think when we send no `thinking` field." That
   is exactly the column whose absence produced the GLM-5.2 misdiagnosis.

**Impact:**

- Any cross-model token-efficiency claim is currently confounded by unequal headroom.
- The failure mode is *silent and directional*: a thinking model under tight headroom looks
  like a weak model, which is precisely how GLM-5.2 was wrongly rejected in June 2026 (the
  regression was our 32768 cap sharing a budget with a 28–32K thinking phase; at 64K it beat
  GLM-5.1 on every axis). Kimi K3 nearly repeated it.
- The 2026-07-27 Opus 5 gate could not verify that *any* of its 76 core runs avoided
  truncation, because the field was absent from every banked result.

**Explicit non-problem — do NOT rebuild this.** The metrics pipeline is already complete and
correct end-to-end (V4–V6). This design adds a stop-reason mapping call and a data policy;
it does not add plumbing.

### Verification Log

Every claim below was checked against the code at authoring time (2026-07-27).

| # | Claim | Method | Result |
|---|-------|--------|--------|
| V1 | Anthropic `Generate` sets no `FinishReason`/`ReasonTokens` | read the `return &ai.Response{...}` in `client.go` | **Confirmed** — only Text/Input/Output/Total/Model |
| V2 | `mapStopReason` already exists and maps `max_tokens`→`"length"` | read `internal/ai/anthropic/step.go:444` | **Confirmed** — no new mapper needed |
| V3 | The agent path already banks `finish_reason` | read `step.go:136` | **Confirmed** — gap is standard-mode only |
| V4 | `RunMetrics` already has both fields | read `metrics.go:21` (`reason_tokens,omitempty`), `metrics.go:118` (`finish_reason,omitempty`) | **Confirmed** — absent from JSON only because `omitempty` + zero |
| V5 | Standard path wires provider → metrics | read `ai_provider.go:154`, `repair.go:240` (`populateMetrics`), `repair.go:160` (repair branch) | **Confirmed** — fully wired |
| V6 | Only Anthropic + Ollama fail to set `ReasonTokens` | `grep -rn "ReasonTokens:" internal/ai/*/*.go` | **Confirmed** — gemini, openai, openrouter all set it |
| V7 | `max_output_tokens` spread + 7 unset entries | `grep -c` over `models.yml`; 110 model entries vs 103 `max_output_tokens:` keys | **Confirmed** |
| V8 | Unset ⇒ 4096 | read `models.go:24` (`0 = handler default 4096`) | **Confirmed** |
| V9 | Anthropic reports no separate thinking-token count | read `anthropicUsage` in `client.go`; cross-checked vendor docs | **Confirmed** — thinking is billed inside `output_tokens` |
| V10 | OpenRouter third-party upstreams ignore reasoning params | probe recorded 2026-07-19 | **Confirmed** — headroom is the only lever there |

## Goals

**Primary Goal:** Make token exhaustion a *measurable signal* rather than a silent confound,
by equalising headroom, banking the stop reason, and labelling each model's thinking state.

**Success Metrics:**
- 100% of standard-mode results carry a non-empty `finish_reason` for providers that report one
- Every `models.yml` entry has an explicit `max_output_tokens` (no silent 4096 fallback)
- Every entry records its default thinking state, with the ceiling-limited rows flagged
- A `finish_reason: "length"` row is visually distinguishable in eval reporting from a
  capability failure

### The reframe this design encodes

Two apparently conflicting positions are both correct, and the distinction is *headroom equality*:

- **Under unequal/arbitrary headroom**, truncation is a **harness artifact** producing a wrong
  capability verdict. (The GLM-5.2 lesson; model-manager's rule "leave default thinking ON.")
- **Under equal, declared headroom**, running out of tokens is **real efficiency signal** about
  the model. (Mark, 2026-07-27.)

Equalising headroom is therefore precisely the change that converts truncation from noise into
signal. The two rules do not conflict — the second becomes true *because of* this work. Both
require the stop reason to be visible, which is why P1 gates the rest.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Common headroom target (proposed: 65536) | Sets the budget every model is judged under; too low re-creates the GLM-5.2 trap, too high inflates cost on runaway rows | human | design | high |
| Equalisation is `min(declared_ceiling, target)`, not a flat value | Some models cannot reach the target (Haiku 4.5 caps at 64K; Ollama rows are bound by `num_predict`/VRAM) — a flat value would be a lie | agent | design | med |
| Re-baseline is annotated, not silently redrawn | Post-change numbers are not comparable to pre-change ones; hiding that corrupts the trend line | human | design | high |
| `ReasonTokens` stays 0 for Anthropic (not estimated) | Fabricating a split would violate no-silent-fallback and make cost math wrong | agent | design | low |
| Ollama/local rows are out of scope for equalisation | Their cap is VRAM-bound, not policy-bound | agent | design | low |

### Design Freeze

- [ ] Common headroom target confirmed (proposed **65536** — the current modal value among
      thinking-capable rows, and the value that fixed GLM-5.2)
- [ ] Re-baseline annotation approach confirmed: annotate in `models.yml` + changelog, keep
      pre-change baselines but mark the discontinuity

## Solution Design

### Overview

Three sequential phases, smallest-first. P1 is a two-line change that unblocks the other two by
making the thing we care about observable.

### Architecture

**Components:**
1. **Stop-reason capture (P1)** — `Generate` calls the existing `mapStopReason(result.StopReason)`.
2. **Thinking-state labels (P2)** — a new `models.yml` field per entry, plus the annotation of
   which rows are ceiling-limited.
3. **Headroom equalisation (P3)** — set `max_output_tokens` to `min(declared_ceiling, 65536)`
   for every cloud entry; record the discontinuity.

### Implementation Plan

**Phase 1: Make truncation visible** (~1 hour)
- [ ] `internal/ai/anthropic/client.go`: set `FinishReason: mapStopReason(result.StopReason)` in `Generate`
- [ ] Table-driven test: `end_turn`→`stop`, `max_tokens`→`length`, `refusal`→(see Deferred)
- [ ] Confirm `finish_reason` appears in a banked standard-mode result (one cheap live run)

**Phase 2: Label thinking state** (~2 hours) — **DONE 2026-07-27**
- [x] Add `default_thinking` to `ModelConfig` + all 110 entries. Vocabulary gained a fifth
      value, **`unknown`**, not in the original plan: forcing a label on every row would have
      meant guessing for models whose default is undocumented, and a wrong label silently
      corrupts an efficiency comparison whereas an admission withholds the row from one.
- [x] Populated **29 rows from citable evidence** (16 Anthropic from the generation table in
      `internal/ai/reasoning_anthropic.go`; GLM 5.x + Kimi K3 + 5 Gemini/OpenAI rows from
      notes already in `models.yml`). **81 remain `unknown`** — including `gpt5-6-sol`, a
      `benchmark_suite` flagship. Reducing that count is follow-up work, not a blocker.
- [x] Test asserting every entry has an explicit, in-vocabulary value
      (`TestModels_DefaultThinkingIsExplicit`), mutation-verified

**Phase 3: Equalise headroom** (~4 hours) — **DONE 2026-07-27**
- [x] ~~Set `max_output_tokens` explicitly on all 7 entries currently falling back to 4096~~
      **DROPPED — this task was wrong.** All 7 unset entries turned out to be `ollama`
      (local) rows, which this same doc scopes OUT as VRAM-bound. Nothing to do.
- [x] Apply `min(declared_ceiling, 65536)` to cloud entries; annotate each ceiling-limited row
      — 53 entries changed, 15 local rows skipped
- [x] Test asserting the policy (`TestModels_CloudHeadroomEqualised`), mutation-verified
- [x] Changelog entry recording the baseline discontinuity

**Measured ceilings (OpenRouter endpoints API, 2026-07-27).** Only two models have a hard
cap below target across *all* upstreams — `deepseek/deepseek-chat` and
`qwen/qwen-2.5-72b-instruct`, both 16384. The min/max spread across upstreams is otherwise
large (e.g. `z-ai/glm-5.2`: 65535 → 1048576 across 28 endpoints), confirming the
routing-roulette caveat; the target is safe because `or-kimi-k2-7-code` (upstream min 16384)
and `or-glm-5-2` (min 65535) have both been running at 65536 without error.

**Anthropic is held at 64000, not raised to the target.** The repo encodes the Claude 4.5+
non-streaming cap as 64000, the client is non-streaming, and the Opus 5 gate ran clean there.
The residual 2.3% gap is immaterial next to the 8192↔128000 spread this phase removes, and
closing it could 400 every Anthropic run — unverifiable while the API key is out of quota.

### Files to Modify/Create

**Modified files:**
- `internal/ai/anthropic/client.go` — +1 line in `Generate` (~1 LOC)
- `internal/ai/anthropic/reasoning_test.go` (or a new `client_test.go`) — stop-reason table (~40 LOC)
- `internal/eval_harness/models.go` — `DefaultThinking` field (~5 LOC)
- `internal/eval_harness/models.yml` — thinking labels + headroom equalisation (~110 entries touched)
- `internal/eval_harness/models_test.go` — schema-completeness assertions (~50 LOC)
- `changelogs/v0.18-current.md` — baseline-discontinuity annotation

## Examples

### Example 1: A truncated Anthropic run

**Before** (indistinguishable from a capability failure):
```json
{ "model": "claude-opus-5", "compile_ok": false, "error_category": "compile_error",
  "output_tokens": 64000 }
```

**After:**
```json
{ "model": "claude-opus-5", "compile_ok": false, "error_category": "compile_error",
  "output_tokens": 64000, "finish_reason": "length" }
```
`finish_reason: "length"` at the cap says the model was cut off mid-answer. Under **equal**
headroom that is a token-efficiency finding; under unequal headroom it was a harness artifact.

### Example 2: Interpreting the Opus 5 gate afterwards

The 2026-07-27 gate measured Opus 5 at 1.9× the output tokens of Opus 4.8 (52,675 vs 28,063 on
an N=3 core sweep) because Opus 5 thinks by default. With P2 landed, that row reads
`default_thinking: on` vs `off` and the ratio is interpretable rather than surprising. With P1
landed we could also assert no run hit `length` — which the gate could not verify.

## Success Criteria

- [ ] A standard-mode Anthropic result JSON contains `finish_reason` (live-verified, not mocked)
      — ⚠️ **BLOCKED until 2026-08-01**: the Anthropic API key hit its configured usage cap
      immediately after the Opus 5 gate consumed $37.61 on 2026-07-27
      (`400 ... You will regain access on 2026-08-01 at 00:00 UTC`). P1 is unit-verified
      (`TestAnthropic_Generate_BanksFinishReason`) but **not** end-to-end verified. Do not
      mark this criterion met on unit tests alone — the whole point of the field is that it
      shows up in a banked result.
- [ ] `mapStopReason` coverage test passes for every documented Anthropic stop reason
- [ ] Zero `models.yml` entries lack `max_output_tokens`; test enforces this
- [ ] Zero entries lack `default_thinking`; test enforces this
- [ ] No entry's `max_output_tokens` exceeds its documented provider ceiling
- [ ] Changelog records the baseline discontinuity with the commit that introduces it
- [ ] All tests passing
- [ ] Documentation updated

## Testing Strategy

**Unit tests:**
- `mapStopReason` table: `end_turn`/`stop_sequence`→`stop`, `tool_use`→`tool_calls`,
  `max_tokens`→`length`, `""`→`""`, unknown→`error`
- `Generate` populates `FinishReason` from a stubbed `messagesResponse` (httptest server)
- `models.yml` schema completeness (both new invariants), in the style of the existing
  `TestAnthropicThinkingStyle_CoversCapabilityTable` guard

**Integration tests:**
- One live cheap standard run (e.g. `fizzbuzz`, `claude-haiku-4-5`) asserting a non-empty
  `finish_reason` lands in the banked JSON

**Manual testing:**
- Re-run one previously-banked benchmark at the new headroom and confirm the delta is
  attributable, not mysterious

## Deferred Decisions

- **~~`refusal` mapping~~ — RESOLVED during P1.** `Generate` already short-circuits
  `stop_reason == "refusal"` into a typed "model refused" error at `client.go:357`, so it never
  reaches `mapStopReason` on the standard path. Pinned by
  `TestAnthropic_Generate_RefusalIsAnError`. Only `model_context_window_exceeded` remains
  folded into `"error"`. *Agent may choose whether that deserves its own value; flag if it
  changes existing agent-path classification.*
- **Whether `default_thinking` should gate suite membership** — recording it is in scope,
  acting on it is not. *Human decides later.*
- **Report-layer surfacing of `finish_reason: "length"`** — banking it is in scope; whether
  `eval-report`/dashboard highlights it is deferred. *Agent may propose.*

## Non-Goals

- **Estimating Anthropic thinking tokens.** The API does not report them separately (V9), so
  `ReasonTokens` stays 0 for Anthropic rather than being fabricated. Cross-provider
  *decomposition* of output tokens is therefore permanently asymmetric — record it, don't fake it.
- **Enforcing a thinking effort per tier.** Blocked upstream: `reasoningCapabilities` ships
  empty (fail-loud by design) and OpenRouter third-party upstreams ignore reasoning params
  entirely (V10). A separate design doc if we ever want it.
- **Equalising Ollama/local rows.** Their ceiling is VRAM-bound, not a policy choice.
- **Re-running historical baselines.** Pre-v0.30.0 results carry no `reason_tokens` and cannot
  be re-judged for truncation — only re-run, which is out of scope here.

## Timeline

**Day 1** (~7 hours):
- P1 implementation + tests + one live verification (~1h)
- P2 schema + labels + tests (~2h)
- P3 equalisation + tests + changelog annotation (~4h)

**Total: ~1 day**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Raising headroom raises cost on runaway rows | Med | Cost is bounded by the model actually using the budget; the Opus 5 gate showed input tokens dominate at core sizes ($13.35 vs $13.59 despite 1.9× output) |
| Post-change baselines not comparable to pre-change | High | Annotate the discontinuity in changelog + `models.yml`; do not silently redraw the trend line |
| A model's declared ceiling is below the common target | Med | `min(ceiling, target)` + explicit per-row annotation, so a ceiling-limited row is never read as an efficiency finding |
| Non-streaming Anthropic calls may time out at very high `max_tokens` | Med | The Anthropic client is non-streaming; keep the 64000-class values for those rows rather than pushing to 128000 |

## Related Documents

<!-- Auto-populated by Ollama neural search on "eval token headroom" -->

**Implemented (may inform design):**
- [design_docs/implemented/v0_15_1/m-eval-cost-and-speed-budgets.md](../../implemented/v0_15_1/m-eval-cost-and-speed-budgets.md) (0.43) — per-model cost/timeout budgets; this doc is the *token* analogue
- [design_docs/implemented/v0_30_0/m-mission-cost-chains.md](../../implemented/v0_30_0/m-mission-cost-chains.md) (0.41)
- [design_docs/implemented/v0_19_0/m-eval-sweet-spot-sprint-plan.md](../../implemented/v0_19_0/m-eval-sweet-spot-sprint-plan.md) (0.40) — introduced `FinishReason` on the agent path

**Planned (checked for overlap — none found, top neural match 0.42 < 0.45 threshold):**
- [design_docs/planned/v1_1_0/m-eval-trust-signals.md](../v1_1_0/m-eval-trust-signals.md) (0.41) — external *credibility* of results (receipts, replay); this doc is internal *measurement fidelity*. Distinct.
- [design_docs/planned/v0_29_0/m-eval-stream-health-retry.md](../v0_29_0/m-eval-stream-health-retry.md) (0.41) — transport health, not token budgets
- [design_docs/planned/v0_30_0/m-arch-boundaries-eval-exclusion-tighten.md](../v0_30_0/m-arch-boundaries-eval-exclusion-tighten.md) (0.42)

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- `changelogs/v0.18-current.md` → "Anthropic thinking control is now model-generation aware"
  (the generation table this design's thinking labels align with)
- `changelogs/v0.18-current.md` → "claude-opus-5 replaces claude-opus-4-8" (the gate that
  surfaced the missing `finish_reason`)
- `.claude/skills/model-manager/SKILL.md` § "Reasoning-Model Check" — the GLM-5.2 truncation
  case study that motivates equal headroom

## Future Work

- Enforced thinking-effort tiers once `reasoningCapabilities` is populated by live smoke (M7),
  accepting that `or-*` rows can never participate (V10)
- A token-efficiency KPI (verified successes per 1K output tokens) — only meaningful once
  headroom is equal, which is what this doc delivers

---

**Document created**: 2026-07-27
**Last updated**: 2026-07-27
