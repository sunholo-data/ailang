# M-EVAL-REASONING-MODEL-FAIRNESS: Fair, correctly-measured reasoning models in the eval harness

**Status**: PLANNED — surfaced 2026-07-11 while verifying the "weird" GLM-5.2 vs GLM-5.1 result in the v0.29.2 baseline.
**Target**: v0.30.x (eval infrastructure; no language surface)
**Priority**: P2 (data trust — reasoning models are a growing share of the roster and their results are currently untrustworthy-looking and possibly unfair)
**Estimated**: 1–2 days
**Dependencies**: none in AILANG core. Touches `internal/eval_harness/` (standard path) + `internal/ai/` provider adapters (OpenRouter). Agent path (`agent_runner*`) is secondary.
**Author**: Claude Opus 4.8 (requested by Mark, 2026-07-11)

---

## Problem

Always-reasoning models (GLM-5.2, Kimi K2.7 Code, and cloud thinking models) come out of the eval harness with **broken metrics** and **possibly depressed scores**. Investigating GLM-5.2's v0.29.2 standard result (40/56 AILANG vs GLM-5.1's 48/56 — a real-looking "core regression") surfaced three distinct issues:

1. **Token accounting goes 0 or negative.** `output_tokens` came back `0`, `-2`, `-23` on GLM-5.2 responses that *did* generate and run code (`compile_ok: True`, cost ~$0.04). A reasoning model's `completion_tokens` vs `completion_tokens_details.reasoning_tokens` is being mis-combined somewhere in the OpenRouter usage parse → the completion count can go negative.
2. **Generated code isn't captured.** The result JSON's `code` field is **empty** for several GLM-5.2 rows even though `compile_ok: True` proves real code was compiled and executed. The persisted `code` is sourced from `resp.Text` via `extractCodeFromMarkdown` (`ai_provider.go:125`); for a reasoning response the parseable answer can land in a `reasoning`/structured field, leaving `Text` empty → no code logged, results look broken, and failures are un-debuggable.
3. **Reasoning may not be requested or budgeted (the fairness question).** `ai_provider.go:generate` builds `ai.Request{Model, SystemPrompt, UserPrompt, MaxTokens, Attribution}` — **no `reasoning`/`effort`/`include_reasoning` field**. So for OpenRouter reasoning models we neither explicitly enable reasoning nor reserve a separate reasoning-token budget. If `MaxTokens` bounds *total* output, the reasoning phase can crowd out the answer; if reasoning isn't requested at the provider level, we may be measuring these models in a mode they weren't built for. **We may be penalising reasoning models for a harness limitation, not a capability gap.**

### Evidence (GLM-5.2, v0.29.2 standard, AILANG)
| Benchmark | error_category | compile_ok | output_tokens | note |
|---|---|---|---|---|
| `commonmark_emphasis` | logic_error | **true** | 2430 | code ran, `code` field empty |
| `gauntlet_10` | logic_error | **true** | **0** | code ran, 0 tokens logged |
| `legal_obligation_engine` | logic_error | **true** | **−23** | negative tokens |
| `quine` | constraint_violation | false | **−2** | negative tokens |
| `effect_txn_rollback` | logic_error | true | (clean 2070-char module) | **genuine** wrong-answer |

**Important:** the pass/fail *verdict* is computed correctly (from `compile_ok` + `stdout_ok`), so GLM-5.2's AILANG regression is at least partly real — this doc does **not** claim the numbers are wrong, it fixes the *measurement + fairness* so the residual gap is trustworthy. Confirmed **not** a routing bug (distinct `z-ai/glm-5.2` slug, distinct outputs, real per-call cost) and **not** budget-killed (cost far under the $0.30 cap).

## Goals
1. **Correct, non-negative token accounting** for reasoning models — surface `reasoning_tokens` as a first-class field alongside `output_tokens` (completion), never derive a negative.
2. **Always capture the executed code** into the result JSON, decoupled from `resp.Text` parsing (log what actually compiled/ran).
3. **Provably fair reasoning** — explicitly enable + budget reasoning for models flagged as reasoning, and record that we did (so a low score is a capability result, not a truncation artifact).
4. A **fairness re-run** of the affected models (GLM-5.2, Kimi K2.7) to re-establish trustworthy numbers.

## Design

### D1 — Reasoning-aware usage parsing (fixes the negative tokens)
In the OpenRouter/provider adapter (`internal/ai/…`) parse the full usage object: `prompt_tokens`, `completion_tokens`, and `completion_tokens_details.reasoning_tokens`. Populate the harness result with three explicit fields — `input_tokens`, `output_tokens` (completion, **clamped ≥ 0**), `reasoning_tokens` — and make `total_tokens = input + output + reasoning`. Never compute `output = completion − reasoning`. Add a `reasoning_tokens` column to result JSON + the dashboard token views.

### D2 — Capture the executed code (fixes empty `code`)
Persist the exact source the harness wrote to the `.ail`/`.py` file (the artifact that `compile_ok`/`stdout_ok` were computed against) into the result's `code` field, independent of how `resp.Text` was parsed. If the answer arrived via a reasoning-wrapped response, extract the final code block robustly (handle `<think>…</think>` prefixes, `reasoning` fields, and fenced/unfenced answers) — see `extractCodeFromMarkdown` (`ai_provider.go:161`), which currently only strips leading fences.

### D3 — Reasoning enablement + budget (the fairness fix)
- Add an optional `reasoning:` block to a model's `models.yml` entry (e.g. `reasoning: { enabled: true, effort: high, max_tokens: 16000 }`). GLM-5.2 and Kimi already carry a `# Reasoning model (always-thinking…)` note — promote that to structured config.
- Thread it into `ai.Request` (a `Reasoning` field) and the OpenRouter request (`reasoning: {...}` / `include_reasoning: true`) so reasoning is explicitly requested and returned.
- Ensure `MaxTokens` reserves headroom for reasoning (extend the existing `ai_provider.go:28-30` / `ai_agent.go:44` headroom logic to *all* reasoning models, keyed off the new flag, not a hardcoded model list).
- **Record fairness provenance** in the result: `reasoning_requested`, `reasoning_tokens`, and whether the response was truncated mid-reasoning (`finish_reason == length`), so a bad row is attributable.

### D4 — Fairness re-run + report
Re-run GLM-5.2 and Kimi K2.7 (standard + agent, AILANG) with D1–D3 landed. Compare pre/post: if the AILANG gap to GLM-5.1 narrows materially, the earlier "core regression" was partly a harness artifact; if it holds, the regression is real (reputation is a Python phenomenon — GLM-5.2 topped the v0.29.2 Python ELO at ~2528). Either way the number is now defensible. Update `benchmarks/CURATION.md` if the re-gate changes GLM-5.2's suite standing.

## Risks / unknowns
- **Provider variance**: OpenRouter's usage schema for reasoning tokens differs by upstream (z.ai vs DeepSeek vs Moonshot). D1 must be defensive (missing fields → 0, never negative) and covered by a table test per provider shape.
- **Cost**: enabling + budgeting reasoning legitimately raises reasoning models' cost/latency. That's correct (it's what the model is), but re-check the per-model `max_cost_usd` caps so we don't swap a truncation artifact for a budget-kill artifact (GLM-5.2's $0.30 cap may need raising — mirror the luna budget question).
- **Not every "reasoning" model is always-on**: some (GPT-5, Gemini, Claude) reason conditionally on `effort`. D3's config must express "off / low / high", not just a boolean.

## Acceptance criteria
1. No result row has negative `output_tokens`; `reasoning_tokens` is populated (and non-zero) for reasoning models.
2. Every row with `compile_ok: true` has a non-empty `code` field.
3. Reasoning models carry `reasoning_requested: true` and adequate budget headroom; no silent mid-reasoning truncation goes unrecorded.
4. GLM-5.2 + Kimi re-run banked; a short written verdict on whether the AILANG regression survives the fixes.
5. Table tests for the usage parser across ≥2 OpenRouter reasoning-response shapes; existing eval tests still pass.

## Out of scope
- Changing the pass/fail grading logic (it's already correct — this is measurement + fairness only).
- The agent-mode CLI harnesses' own reasoning handling (opencode/codex/claude manage their own token budgets) beyond making sure the harness records `reasoning_tokens` when the CLI reports it. Standard mode is the priority.
- Re-gating models other than GLM-5.2 / Kimi (do those opportunistically if the parser touches their rows).
