# M-LYCEUM-PROVIDER: Lyceum as an EU-Hosted Third Route for Open-Weight Models

**Status**: Planned — Phase 0 spike COMPLETE 2026-09-03 (V16-V22 recorded; all premises
verified, one contingency carried into Phase 1). Ready for sprint planning.
**Target**: v0.34.1
**Priority**: P2 (Low) — a route-diversification hedge; no lane is blocked on it
**Estimated**: ~6-8 hours across 4 phases (Phase 0 COMPLETE — ~1 hour, 2026-09-03)
**Dependencies**: A Lyceum API key (human action, Mark — key already generated). No code dependencies.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is eval/harness infrastructure, not a language change. As with M-OLLAMA-CLOUD-PROVIDER,
most axioms are genuinely neutral; scoring them 0 is the honest answer, not an evasion.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Model sampling is already nondeterministic; routing the same weights to a different EU host does not add a new class of nondeterminism. |
| A2: Replayability | 0 | Trace shape unchanged; the provider name is recorded on banked rows via the existing provider field. |
| A3: Effect Legibility | 0 | No AILANG effect surface touched. Harness-only change. |
| A4: Explicit Authority | 0 | `LYCEUM_API_KEY` is explicit, per-machine, env-scoped — same authority shape as `OPENROUTER_API_KEY`. Missing key fails loudly (AC4). |
| A5: Bounded Verification | 0 | No verification-surface change. |
| A6: Safe Concurrency | 0 | Concurrency bounded by the existing `--parallel` semaphore; Lyceum concurrency limits unknown until Phase 0 (Risk R4). |
| A7: Machines First | 0 | No prompt or output-format change. |
| A8: Minimal Syntax | 0 | No new AILANG syntax. Config-only (`models.yml` rows) + one dispatch case. |
| A9: Cost Visibility | +1 | Adds a second real-metered price point for the same weights (the EU-price comparison is already banked in models.yml notes for OR twins), making route-cost deltas measurable rather than anecdotal. |
| A10: Composability | +1 | Reuses the existing `internal/ai/openai` transport end-to-end (the openrouter precedent) — no new provider package, no new wire protocol, no new streaming/tool/reasoning code. |
| A11: Structured Failure | 0 | Errors flow through the existing `error_category` path. AC7 requires 401/429 not be miscategorized as model failure. |
| A12: System Boundary | +1 | Route identity is explicit: `provider: "lyceum"` is a structural field in the models.yml row (not an unenforced naming suffix like the ollama `:cloud` convention), and it flows into banked output. |

**Net Score: +2** → **Decision: Move forward**

### Hard Violation Check

**These axioms cannot have −1 scores (automatic rejection):**

- [x] A1 (Determinism): No implicit nondeterminism introduced — routing does not change sampling semantics
- [x] A3 (Effects): No hidden side effects — no AILANG effect surface touched
- [x] A4 (Authority): No ambient access granted — API key is explicit env var, per-machine; unset key = loud error
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

## Problem Statement

OpenRouter is the only per-token metered route to open-weight models in `models.yml`, so every
open-weight datapoint we bank carries a single vendor's routing, pricing, and availability as a
confound. Ollama Cloud (M-OLLAMA-CLOUD-PROVIDER, v0.34.0) added a flat-rate second route but is
subscription-shaped (imputed prices, unpublished quota denominator, `list-price-equivalent`
provenance) — it is not a clean per-token price control.

Lyceum ([dashboard](https://dashboard.lyceum.technology/inference)) offers **EU-hosted**
inference on the same open-weight models with OpenAI-compatible per-token billing — a genuine
third route with a different vendor, jurisdiction, and price list.

**Current State:**

- `models.yml` carries **146** rows across providers `openai`/`anthropic`/`google`/`ollama`/`openrouter` (V10).
- Per-token open-weight access = OpenRouter only. Single-vendor confound on every open-weight row.
- Day-one upstream congestion is REAL in practice: the `or-qwen3-8-flash` smoke gate was
  invalidated 19/23 by Alibaba-pool 429s (2026-08-31, banked in that row's notes). There is no
  second metered route to re-run against.
- Mark holds a Lyceum API key (generated 2026-09-03) and wants the option evaluated.

**Price reality check (dashboard 2026-09-03 vs our OR rows verified 2026-08-27..31):**
Lyceum is more expensive than OpenRouter on 7 of the 8 shared models — GLM-5.3-Flash ~2.7x
($0.20/$0.50 vs $0.075/$0.25), Kimi-K2.7-Code ~1.9x on input, Qwen3.8-27B ~6% *cheaper*
($0.40/$2.40 vs $0.425/$2.55), Kimi-K3 identical ($3.00/$15.00). Lyceum's GLM-5.2 listing
($1.50/$4.50) sits near OR's *launch* price, suggesting their price list lags OR's cuts. So the
honest framing is the same as M-OLLAMA-CLOUD-PROVIDER's: **not new capability — a second price
and a second route for models we already run, plus EU data residency OR cannot offer.**

**Impact:**

- **Who**: the eval rotation (429 fallbacks), future EU-residency workloads, and the
  route-vs-route confound question (same weights, two vendors, two prices).
- **How significant**: cheap to build (one thin dispatch case — the openai transport already
  exists); cheap to try (a full smoke tier at Lyceum GLM-5.3-Flash prices ≈ $0.16).

## Goals

**Primary Goal:** Reach Lyceum's EU-hosted models from the standard-mode eval harness and
`--ai` dispatch without adding a provider package, with real metered pricing banked correctly.

**Success Metrics:**

- A models.yml row with `provider: "lyceum"` completes an eval-suite smoke tier through the
  standard-mode harness with **zero** changes to `internal/ai/openai/` (Phase 2 exit gate).
- Banked rows show real token counts, `cost_usd` from the dashboard prices, and
  `CostProvenance = metered` (never `free-local`).
- Reasoning tokens are captured on a thinking model (GLM-5.3-Flash probe) — the GLM-5.2
  truncation lesson: a route that hides reason_tokens cannot be compared to OR twins.
- Missing `LYCEUM_API_KEY` produces a loud, named error; no silent fallback.
- A route A/B (lyceum twin vs OR twin, same bench set) produces a banked comparison and a
  keep/drop decision for the row set.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **D1**: Transport = reuse `internal/ai/openai` via `WithBaseURL`; **no `internal/ai/lyceum/` package** | OpenRouter is itself a thin client over the openai package (V5) — Lyceum's API is the same OpenAI-compatible chat-completions wire format (V12). A new package would duplicate the streaming, tool-dispatch, `reasoning_content`, and cache work the openai transport already landed. | agent | design | low |
| **D2**: Base URL = hardcoded constant `https://api.lyceum.technology/openai/v1` + optional `LYCEUM_BASE_URL` env override for testing | models.yml has **no** `base_url` field (0 occurrences across 146 rows, V10; M-OLLAMA-CLOUD-PROVIDER D6/V13 reached the same finding) and adding a generic endpoint field is a schema decision this feature does not need. A per-provider env override preserves testability without a schema change. | agent | design | low |
| **D3**: Auth = `LYCEUM_API_KEY`, required; missing key = loud named error | Critical Principle #2 (no silent fallbacks). Matches every other cloud provider's shape. | agent | design | low |
| **D4**: Seed row set and their provenance labelling | Rows bank cost data; wrong/imputed prices would corrupt cost-per-verified-success. Seed prices come from the **dashboard screenshot** (not yet wire-verified) and must be labelled as such until Phase 0 reconciles them against actual usage billing. Mark ratifies the final row set. | **human (Mark)** | design | med |
| **D5**: `lyceum` added to the `SupportsStandardEval` cloud-provider list | Without it, lyceum rows classify agent-only and standard-mode runs are blocked (the 2026-05-23 incident class: 102 trials, total_tokens=0). With it, rows are dual-mode. | agent | compile | low |
| **D6**: Model slug namespace resolved by Phase 0 probe, not assumed | The dashboard shows display names (`Kimi-K3`); the sample curl uses `moonshotai/kimi-k3` — OpenRouter-slug-shaped, but whether `z-ai/glm-5.3-flash` and friends resolve is **unverified**. models.yml rows are written only after slugs are live-verified (the `or-kimi-k3` discipline: "OpenRouter ID verified live"). | agent | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] **D4 RATIFIED (Mark, attended session 2026-09-03)** — seed row set accepted:
      `lyceum-glm-5-3-flash`, `lyceum-qwen3-8-flash`, `lyceum-kimi-k3`. Rows still land
      only for slugs that pass the Phase 0 V16 live-verification gate.

D1/D2/D3/D5/D6 are not freeze items: all are agent-resolvable and D6 is resolved by the Phase 0
gate before any row is written.

## Solution Design

### Overview

**The integration is one dispatch case + models.yml rows, not a provider package.** Lyceum
exposes an OpenAI-compatible endpoint (`POST https://api.lyceum.technology/openai/v1/chat/completions`,
Bearer auth, standard chat params — V12). The `openai` transport already implements everything
the harness needs on that wire format: chat + streaming + tool dispatch, `reasoning_content`
parsing (step + stream paths), `completion_tokens_details.reasoning_tokens` accounting, the
`reasoning_effort` dial, and a `WithBaseURL` client option.

The `openrouter` package proves the pattern: it reuses `internal/ai/openai`'s
`BuildChatStepRequest`/`ParseChatStepResponse` (V5) and adds only what is OpenRouter-specific
(attribution headers, routing, correlation). Lyceum-specific surface is even thinner: a base URL
and an env var. No attribution headers exist in Lyceum's documented API (V12), so none are sent.

### Architecture

**Components:**

1. **Provider enum + auth** (`internal/ai/config.go`): `ProviderLyceum ProviderType = "lyceum"`,
   a `ProviderFromString` case, `EnvVarForProvider` → `"LYCEUM_API_KEY"`, and a `GetAPIKey` case
   (V2 for current shape).

2. **Dispatch case** (`cmd/ailang/ai_handlers.go`, `setupAIHandlerFromConfig`): mirror the
   existing `openai` case (V4) but with the Lyceum constant:

   ```go
   case ai.ProviderLyceum:
       if apiKey == "" {
           return fmt.Errorf("%s environment variable required for model %s", model.EnvVar, aiModel)
       }
       client := openai.NewClient(apiKey, openai.WithBaseURL(lyceumBaseURL))
       handler = client.NewHandler(model.APIName, opts...)
   ```

   plus the same case in `setupAIHandlerDirect`/`exec.go` dispatch (V8) so `--provider lyceum`
   works, and `"lyceum"` added to the hardcoded valid-provider sets (V8).

3. **Registry reservation** (`internal/ai/registry.go:36-42`): add `"lyceum"` to
   `builtInProviderNames` so config-driven `[[ai_provider]]` blocks cannot shadow it (V7).

4. **Standard-eval classification** (`internal/modelreg/models.go:382-388`): add `"lyceum"` to
   the dual-mode provider list (V9, D5).

5. **Model rows** (`internal/modelreg/models.yml`): three seed rows, provider prefix convention
   (`lyceum-*`, matching the `or-*` / `motoko-cloud-*` conventions), real dashboard prices with
   provenance comments, `default_thinking` set per Phase 0 probe results, and the standard
   65536 max-output headroom (the GLM-5.2 truncation lesson).

**Cost treatment (contrast with M-OLLAMA-CLOUD-PROVIDER D1):** Lyceum rows are per-token
metered with real prices — `AuthLaneForModel` defaults to `AuthLaneBilled` → `CostMetered`
with **zero code** (V15). The one trap: `ResolveCostProvenance` maps pricing `0/0` to
`CostFreeLocal` (V15), so seed rows MUST carry real prices — a 0/0 Lyceum row would bank a
positively false "on-device, free" claim. The Phase 2 checklist enforces this.

### Implementation Plan

**Phase 0: Live API verification spike — falsify the premises** (~1 hour, GATES EVERYTHING)

Run against the live API with Mark's key (2026-09-03): all seven probes executed, results
recorded in the Verification Log (V16-V22). Summary:

- [x] **V16**: ✅ PASSED — `GET /openai/v1/models` returns a 39-model catalogue; slugs follow
      the OpenRouter convention. Seed slugs live-verified: `moonshotai/kimi-k3` ✅,
      `z-ai/glm-5.3-flash` ✅, **`qwen/qwen3.8-flash-next`** ✅ (correction: the `-next`
      suffix IS the slug — `qwen/qwen3.8-flash` 404s). Catalogue also carries models our rig
      cannot host (`qwen/qwen3.8-2.4t-a95b`, `qwen/qwen3.5-397b-a17b`,
      `deepseek/deepseek-v4-pro`, `nvidia/nemotron-3-ultra-550b-a55b`) — relevant to the
      Phase 3 decision note. Vendor router models exist (`lyceum/complex|reasoning|simple`);
      not registered.
- [x] **V17**: ✅ PASSED — kimi-k3 and glm-5.3-flash both `finish=stop` with content at
      max_tokens 2048 (at 32 the whole budget went to thinking — headroom matters, as always).
- [x] **V18**: ✅ PASSED — standard SSE (`chat.completion.chunk`), `delta.reasoning_content`
      thinking chunks, `data: [DONE]` sentinel, AND a final usage-bearing chunk sent WITHOUT
      `stream_options.include_usage` being set (our client sets it anyway — harmless).
- [x] **V19**: ✅ PASSED with one caveat — GLM-5.3-flash returns `reasoning_content`
      (message field) AND `usage.completion_tokens_details.reasoning_tokens`; Kimi-K3 returns
      thinking in a `reasoning` field (NOT `reasoning_content`) and NO reasoning_tokens in
      usage. Our `ChatStepRespMessage` already parses BOTH fields (step.go:317-324) — zero
      code needed for the thinking text. CAVEAT: `ChatStepUsage` (step.go:336) and the
      streamstep usage struct do NOT parse `completion_tokens_details` — only the Generate
      path (types.go:58-66) does. If banked reason_tokens come back 0 at the Phase 2 smoke
      gate, extend both structs with `CompletionTokensDetails` (3 lines each, same shape as
      types.go chatUsage) — added to Phase 1 as a contingency task.
- [x] **V20**: ✅ PASSED — standard OpenAI usage shape; Lyceum extras: Kimi carries
      `prompt_tokens_details.cached_tokens` + `created_cache_tokens`; GLM carries
      `text_tokens` alongside `reasoning_tokens` (they sum to completion_tokens on a clean
      finish; they DIVERGED on the finish=length probe — 32 completion vs 34 reasoning — do
      not assume they sum). ALSO NOTED: kimi-k3 counted `prompt_tokens=93` for the same
      6-word prompt GLM counted as 17 — possible hidden system prompt or chat-template
      accounting; flag for the A/B (Lyceum kimi input costs may read high vs OR).
- [x] **V21**: ✅ PASSED — standard OpenAI error envelopes: 401
      `{"error":{"type":"unauthorized_error"}}`, 404
      `{"error":{"type":"not_found_error","param":"model"}}`. Map cleanly onto the
      existing error-category path (AC7).
- [x] **V22**: ⚠️ PARTIAL — no self-serve billing endpoints exist under `/openai/v1`
      (usage/billing/credits/user all 404). Reconciliation goes via the dashboard, manually.
      Probe spend recorded for reconciliation: <1,000 tokens total (kimi ~287, glm ~345,
      rest error/empty) — expected cost < $0.001 at dashboard prices. Mark can eyeball the
      dashboard to confirm before the smoke gate.

**Phase 1: Provider plumbing** (~3 hours)

- [ ] `internal/ai/config.go`: enum + `ProviderFromString` + `EnvVarForProvider` + `GetAPIKey`.
- [ ] `cmd/ailang/ai_handlers.go`: dispatch cases in `setupAIHandlerFromConfig` and
      `setupAIHandlerDirect` (lyceum models are models.yml-first; the direct path just needs a
      sane error, not GuessProvider support).
- [ ] `cmd/ailang/exec.go`: valid-provider sets (lines ~135, ~139) + dispatch case near the
      openrouter case (~479).
- [ ] `internal/ai/registry.go`: `builtInProviderNames` reservation.
- [ ] `internal/modelreg/models.go`: `SupportsStandardEval` list.
- [ ] `cmd/ailang/help.go` / CLI docs: `LYCEUM_API_KEY` env var + provider mention (the
      cli-doc-maintainer convention: help.go is the single source of truth).
- [ ] CONTINGENCY (from V19): if banked reason_tokens are 0 on the GLM probe at smoke,
      extend `ChatStepUsage` (step.go:336) and the streamstep usage struct with
      `CompletionTokensDetails{ReasoningTokens}` — 3 lines each, mirroring the Generate
      path's chatUsage (types.go:58-66).
- [ ] Unit tests: `ProviderFromString("lyceum")` round-trip; dispatch with a stub server
      (the openai package's httptest pattern) proves the base URL lands on the wire; missing
      key → error naming `LYCEUM_API_KEY`; existing `OPENAI_BASE_URL` path untouched.

**Phase 2: Seed rows + smoke gate** (~1 hour)

- [ ] Write the three rows (only for Phase 0-verified slugs) with dashboard prices labelled
      `verified via dashboard 2026-09-03, wire-reconciled V22` and `budgets.max_cost_usd`
      scaled to Lyceum prices (flash rows keep 0.30; kimi-k3 keeps 0.60 per the OR precedent).
- [ ] `modelreg` validation passes (V11: no enum check on `provider:`, so the real gate is the
      dispatch test + a smoke run).
- [ ] Smoke gate: `ailang eval-suite --models lyceum-glm-5-3-flash --tier smoke` — expect
      ~23 benchmarks at ≈ 2-2.7x OR flash cost (≈ $0.06-0.15 total).
- [ ] Confirm banked rows: provider=lyceum, real tokens, cost>0, provenance=metered,
      reasoning tokens present (V19).

**Phase 3: Route A/B + decision note** (~1-2 hours)

- [ ] Head-to-head vs the OR twin: `ailang eval-suite --models
      lyceum-glm-5-3-flash,or-glm-5-3-flash --tier smoke` (same weights, same benches, two
      routes, two prices), then `--tier core` only if smoke shows the route is faithful
      (same failure shape class, no systematic truncation).
- [ ] Decision note in the rows: does Lyceum earn a standing seat (429-fallback lane,
      EU-residency lane) or stay opt-in? Roster rules apply: no suite displacement without
      core evidence + Mark ratification.

### Files to Modify/Create

**New files:**
- None (no provider package — that is the point; D1)

**Modified files:**
- `internal/ai/config.go` — `ProviderLyceum` constant + 3 switch cases (~15 LOC)
- `cmd/ailang/ai_handlers.go` — dispatch cases in both setup paths (~15 LOC)
- `cmd/ailang/exec.go` — valid-provider sets + dispatch case (~10 LOC)
- `internal/ai/registry.go` — `builtInProviderNames` entry (1 LOC)
- `internal/modelreg/models.go` — `SupportsStandardEval` list (1 LOC)
- `internal/modelreg/models.yml` — 3 seed rows (~100 LOC with provenance notes)
- `cmd/ailang/help.go` + `docs/docs/reference/` — env var + provider docs (~20 LOC)
- Tests: extend `internal/ai/config_test.go`, `cmd/ailang` dispatch tests (~80 LOC)

## Examples

### Example 1: models.yml seed row (shape)

```yaml
  lyceum-glm-5-3-flash:
    api_name: "z-ai/glm-5.3-flash"        # V16-verified live 2026-09-03
    provider: "lyceum"
    default_thinking: "always_on"          # V19 probe: reasoning_content on every response
    description: "GLM-5.3-Flash via Lyceum (EU-hosted) — same weights as or-glm-5-3-flash, different route/jurisdiction/price."
    env_var: "LYCEUM_API_KEY"
    max_output_tokens: 65536               # GLM-5.2 truncation lesson — headroom, not a cap
    pricing:
      input_per_1k: 0.0002   # $0.20 per 1M — dashboard 2026-09-03, wire-reconcile V22
      output_per_1k: 0.0005  # $0.50 per 1M — dashboard 2026-09-03, wire-reconcile V22
    budgets:
      max_cost_usd: 0.30
      hard_timeout_secs: 600
```

### Example 2: smoke command

```bash
export LYCEUM_API_KEY=...
ailang eval-suite --models lyceum-glm-5-3-flash --tier smoke --langs ailang
ailang eval-suite --models lyceum-glm-5-3-flash,or-glm-5-3-flash --tier smoke --langs ailang
```

## Success Criteria

- [ ] AC1: A `lyceum-*` row completes the smoke tier in standard mode — zero changes to
      `internal/ai/openai/` (Phase 2 exit gate)
- [ ] AC2: Banked row shows provider=lyceum, real token counts, cost>0, `CostProvenance=metered`
- [ ] AC3: Reasoning tokens captured on the GLM probe (or explicitly flagged in the row if the
      upstream does not expose them — V19)
- [ ] AC4: `LYCEUM_API_KEY` unset → loud named error on dispatch; no silent fallback
- [ ] AC5: All existing provider dispatch tests pass; `OPENAI_BASE_URL` global override
      behaviour unchanged
- [ ] AC6: `SupportsStandardEval("lyceum-glm-5-3-flash")` returns true (no agent-clamp)
- [ ] AC7: 401/429 from Lyceum categorize as `api_error` (cause: infra), not model failure
- [ ] AC8: Route A/B banked with a keep/drop decision recorded in the row notes
- [ ] All tests passing
- [ ] Documentation updated (help.go, env vars reference)

## Testing Strategy

**Unit tests:**
- Provider enum round-trip, env-var mapping, key-required error
- Dispatch constructs an openai client pointed at the Lyceum base URL (httptest stub asserts
  the request host + Authorization header)
- `builtInProviderNames` contains "lyceum" (config-driven shadowing rejected)

**Integration tests:**
- Smoke tier through the real API (Phase 2) — this is the acceptance gate
- Error-shape checks: 401 → loud auth error; 429 → `api_error` category

**Manual testing:**
- Phase 0 probes (V16-V22) recorded in the doc/rows before rows are written
- Billing reconciliation against Lyceum's dashboard (V22)

## Verification Log

| # | Claim | Method | Result |
|---|-------|--------|--------|
| V1 | "lyceum" appears nowhere in the codebase | `grep -rn "lyceum\|Lyceum\|LYCEUM" internal/ cmd/ internal/modelreg/models.yml` | empty — confirmed 2026-09-03 |
| V2 | Provider enum/auth shape | `internal/ai/config.go:13-17` (5 constants), `ProviderFromString` (~:57-96), `EnvVarForProvider` (:104-120), `GetAPIKey` (:123+) | read 2026-09-03 |
| V3 | openai client supports custom base URL | `internal/ai/openai/client.go:18` (`defaultBaseURL`), `:33-36` (`WithBaseURL`) | read 2026-09-03 |
| V4 | Dispatch switch shape | `cmd/ailang/ai_handlers.go:119-130` (`setupAIHandlerFromConfig`), openai case reads global `OPENAI_BASE_URL` | read 2026-09-03 |
| V5 | OpenRouter is a thin client over openai | `internal/ai/openrouter/step.go:13,23` reuses `openai.BuildChatStepRequest`/`ParseChatStepResponse`; `openrouter/client.go:18` own `defaultBaseURL` | read 2026-09-03 |
| V6 | Reasoning works on the openai transport | `internal/ai/openai/step.go:317-324` + `streamstep.go:156-172,273-281` (`reasoning_content` parsed, thinking deltas streamed); `types.go:58-66` (`reasoning_tokens` usage); `chat.go:60-63` (`reasoning_effort` dial) | read 2026-09-03 |
| V7 | Built-in name reservation map | `internal/ai/registry.go:36-42` | read 2026-09-03 |
| V8 | Hardcoded provider sets + dispatch in exec path | `cmd/ailang/exec.go:135,139,467-484` | read 2026-09-03 |
| V9 | `SupportsStandardEval` hardcoded cloud list | `internal/modelreg/models.go:382-388` (`anthropic, openai, google, gemini, vertex, openrouter`; unknown → agent-only) | read 2026-09-03 |
| V10 | No `base_url` field in models.yml | `grep -c base_url` → 0 (146 rows); matches M-OLLAMA-CLOUD-PROVIDER D6/V13 | confirmed 2026-09-03 |
| V11 | models.yml provider names are not enum-validated | `internal/modelreg/validate.go:43` checks only non-empty | read 2026-09-03 |
| V12 | Lyceum API shape | Mark-provided curl: OpenAI-compatible `/openai/v1/chat/completions`, Bearer auth, standard params, slug `moonshotai/kimi-k3`; no attribution headers documented | user-provided 2026-09-03 |
| V13 | Lyceum price list | Dashboard screenshot 2026-09-03 (Kimi-K3 $3/$15; GLM-5.2 $1.50/$4.50; GLM-5.3 $1.75/$4.50; GLM-5.3-Flash $0.20/$0.50; Qwen3.8-Flash-Next $0.20/$0.50; Qwen3.8-27B $0.40/$2.40; K2.7-Code $1.25/$4.50 — row partially cut off in screenshot) | dashboard 2026-09-03 |
| V14 | OR twin prices in our rows | models.yml pricing blocks verified 2026-08-27..31 (see or-* rows) | read 2026-09-03 |
| V15 | `ResolveCostProvenance` maps 0/0 → `CostFreeLocal`; default lane for unknown models is `AuthLaneBilled` → `CostMetered` | `internal/executor/cost.go:140-153,217-222` | read 2026-09-03 |
| V16 | Slug resolution | `GET /openai/v1/models` + per-slug completions | ✅ 39-model catalogue; `moonshotai/kimi-k3`, `z-ai/glm-5.3-flash`, `qwen/qwen3.8-flash-next` all live-verified 2026-09-03 (`qwen/qwen3.8-flash` 404s — `-next` is part of the slug) |
| V17 | Completion round-trip | max_tokens 2048 probes, both models | ✅ finish=stop with content; at 32 tokens the budget went entirely to thinking |
| V18 | Streaming | SSE capture, head+tail | ✅ standard `chat.completion.chunk`, `delta.reasoning_content`, `data: [DONE]`, unprompted final usage chunk |
| V19 | Reasoning exposure | GLM-5.3-flash + kimi-k3 probes + step.go:317-324 read | ✅ GLM: `reasoning_content` + `usage.completion_tokens_details.reasoning_tokens`; Kimi: `reasoning` field (both parsed by ChatStepRespMessage) but NO reasoning_tokens in kimi usage. ⚠️ step/stream usage structs lack `completion_tokens_details` — contingency task added to Phase 1 |
| V20 | Usage shape | both models' usage blocks | ✅ standard shape + Lyceum extras (kimi: `cached_tokens`/`created_cache_tokens`; glm: `text_tokens`). kimi counted prompt_tokens=93 vs glm 17 for the same prompt — A/B flag |
| V21 | Error shapes | bad-key + bad-model probes | ✅ 401 `unauthorized_error`, 404 `not_found_error` — standard OpenAI envelopes |
| V22 | Billing reconciliation | probed `/usage`,`/billing/usage`,`/credits`,`/user` — all 404 | ⚠️ PARTIAL: no self-serve endpoints; reconcile via dashboard. Probe spend <1,000 tokens, expected cost < $0.001 |

## Conflict Surface

Harness/dispatch change (not parser/typechecker/codegen), but the dispatch switch is itself a
shared position — here is what else lives there:

1. **Extended position**: the built-in provider dispatch switch(es) in
   `ai_handlers.go`/`exec.go` and the `ProviderType` string space.
2. **Existing occupants**: five built-ins (openai, anthropic, google/gemini, ollama,
   openrouter) + config-driven `[[ai_provider]]` packages (dispatch falls through to
   `LookupConfigDrivenProvider` / the registry on unknown names, V2/V4).
3. **Disambiguation**: built-in dispatch wins on collision with registry entries
   (`registry.go` D4 semantics); `"lyceum"` is added to the reserved map so a package cannot
   register a shadowing provider (V7).
4. **Programs/configs that MUST still work post-change**: all 146 existing models.yml rows
   (no switch-case reordering); the `OPENAI_BASE_URL` global override path for `provider:
   openai` rows (V4); `--provider openrouter` routing-flag validation
   (`routing_flags.go:113` — unchanged, lyceum does not support `--routing-*` flags, same
   error path as other non-openrouter providers); config-driven providers on the fallback
   path; the ollama GPU-lock clamp (keys off `SupportsAgentEval && !SupportsStandardEval`,
   not provider name — V9).
5. **Deliberately changed**: nothing breaks. Additive only: one enum value, one dispatch case
   per switch, one list entry each in two hardcoded sets, three new rows.

## Deferred Decisions

The following are intentionally left open for the implementer:

- `LYCEUM_BASE_URL` override semantics (exact name, whether it also gates a `--lyceum-base-url`
  flag) — agent may choose, mirroring the `OPENAI_BASE_URL` precedent
- Exact error-message wording — agent may choose, keep the existing provider-error style
- Whether `GuessProvider` learns `lyceum:` prefixes — agent may choose; models.yml-first means
  the direct path only needs a sane error

## Non-Goals

**Not attempted in this feature:**

- Agent-mode (`agent_cli`) twins for opencode/pi/motoko — standard mode first; agent twins are
  the M-OLLAMA-CLOUD-PROVIDER pattern and can follow if the route earns a seat
- OpenRouter-style attribution/routing headers — Lyceum's documented API has none (V12)
- A generic `base_url`/`endpoint` field in models.yml — schema decision, not needed here (D2)
- Automatic OpenRouter→Lyceum failover on 429s — a future route-policy feature; this doc only
  makes the second route *reachable*
- models.yml rows for all 8 dashboard models — only live-verified slugs for the ratified seed
  set (D6)

## Timeline

**Day 1** (~4-5 hours):
- Phase 0 verification spike (1h)
- Phase 1 plumbing + unit tests (3h)

**Day 2** (~2-3 hours):
- Phase 2 seed rows + smoke gate (1h)
- Phase 3 A/B + decision note (1-2h)

**Total: ~6-8 hours across 2 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| R1: Slug namespace differs from OpenRouter's | ~~Med~~ **RESOLVED (V16)** | Slugs follow the OpenRouter convention; all three seed slugs live-verified (`qwen3.8-flash-next` naming correction banked) |
| R2: Reasoning tokens not exposed by Lyceum's upstreams | ~~Med~~ **RESOLVED (V19)** | GLM exposes reasoning_content + reasoning_tokens; Kimi exposes thinking text but no reasoning-token count — kimi rows get a `reason_tokens unavailable` note |
| R3: Dashboard prices drift from actual billing | Low | V22 reconciliation before the smoke gate; rows carry verification date + provenance comments like the OR rows |
| R4: Unknown concurrency limits / throttling on a smaller vendor | Med | Lyceum rows start opt-in (not in any suite); run them serially (`--parallel 1`) until Phase 0 observes the throttle behaviour |
| R5: `0/0` pricing row banks as `free-local` (false provenance) | Med | Phase 2 checklist: every lyceum row carries real dashboard-reconciled prices (V15) |
| R6: Vendor availability/SLA unknown | Low | Keep rows opt-in; no suite displacement without core evidence + Mark ratification |

## Related Documents

<!-- Auto-populated by Ollama neural search on "lyceum provider"; duplicate gate PASSED
     (max neural 0.28, below the 0.45 warn band) -->

**Implemented (informs design):**
- [m-ollama-cloud-provider.md](design_docs/implemented/v0_34_0/m-ollama-cloud-provider.md) —
  the closest prior art: second route for open-weight models, same D1-style cost-provenance
  problem (solved differently: imputed `list-price-equivalent` vs Lyceum's real metered prices),
  same Phase-0-falsifies-premises discipline, and the D6 finding that models.yml has no
  route/endpoint field (re-verified V10)
- [m-unified-ai-providers.md](design_docs/implemented/v0_5_10/m-unified-ai-providers.md) —
  the built-in vs config-driven provider split this design stays inside (built-in path chosen
  for the features the harness needs)
- [m-ollama-v1-streaming-idle-timeout.md](design_docs/planned/m-ollama-v1-streaming-idle-timeout.md) —
  the streaming work the openai transport already carries (why no new package)

**Planned (checked for overlap — distinct):**
- [m-eval-data-hosting-decouple.md](design_docs/planned/m-eval-data-hosting-decouple.md) (0.26) —
  data hosting, not inference routing
- [m-dynamic-data-runtime-plane.md](design_docs/planned/m-dynamic-data-runtime-plane.md) (0.27) —
  runtime data plane, unrelated

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [Lyceum inference dashboard](https://dashboard.lyceum.technology/inference) - price list (V13)
- [M-OLLAMA-CLOUD-PROVIDER](design_docs/implemented/v0_34_0/m-ollama-cloud-provider.md) - route-diversification precedent
- `internal/ai/openai/` - the transport this design reuses

## Future Work

- Agent-mode twins (opencode/pi/motoko `agent_cli` rows) if the route earns a seat
- Route policy: automatic OR→Lyceum fallback on upstream 429s, quota-aware lane selection
- A generic `endpoint` field in models.yml if a fourth OpenAI-compatible vendor appears
  (three would make the pattern a schema, not a convention)

---

**Document created**: 2026-09-03
**Last updated**: 2026-09-03