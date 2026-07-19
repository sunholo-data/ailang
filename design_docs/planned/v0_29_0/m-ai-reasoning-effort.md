---
title: "M-AI-REASONING-EFFORT — Cross-provider request-side reasoning control"
status: PARKED needs-human-review (2026-07-19, iter 61 — quorum-at-pick R1 revised → R2 re-quorum blocked on 2 narrow converging fixes; bounded gate consumed)
target: v0.31.0
priority: P2
estimated: ~10-14h
owner: ailang-core
dependencies:
  - none (additive change to ai.Request and four provider clients)
consumers:
  - ailang-parse (docparse) — wants `reasoning=off` for Gemini 3 PDF parse calls
  - eval-harness — already records ReasonTokens; gains request-side knob
  - any future caller wanting deterministic latency budgets
---

# M-AI-REASONING-EFFORT — Cross-Provider Request-Side Reasoning Control

## ⛔ Quorum Record (mission iteration 61, 2026-07-19) — PARKED needs-human-review

This pre-quorum backlog doc was picked by mission-control. Per QUORUM-AT-PICK it ran the text
quorum (`gpt5-6-sol` + `gemini-3-1-pro`, controller verdict PASS both rounds). The bounded gate
(one revision + one re-quorum) is now **consumed** and the doc is **PARKED for human review**.

**Round 1 (original doc) — BLOCKED.** Two objections, both resolved by Rev-1 (codex designer):
1. `gpt5-6-sol` — the "ignore/omit+log" fallback policy and OpenAI `"off"→"minimal"` mapping
   violated AILANG's no-silent-fallback + determinism axioms. → **Resolved**: Rev-1 added a
   fail-loud contract (5 typed sentinel errors, capability-table gating where unknown-model =
   error, `"off"` rejected not weakened, deterministic precedence + conflict errors).
2. `gemini-3-1-pro` — missing Conflict Surface vs the existing `MaxTokens` field (Anthropic
   requires `max_tokens > thinking.budget_tokens`; `"high"`=16384 with unset MaxTokens → 400).
   → **Resolved**: Rev-1 added a full `## Conflict Surface` with pre-dispatch `MaxTokens > budget`
   rules and caught that `anthropic/client.go:160-168` silently substitutes `MaxTokens=4096`
   (validation must precede that defaulting). Version metadata aligned to v0.31.0; Verification
   Log added.

**Round 2 (Rev-1) — BLOCKED on 2 NEW, NARROWER, converging fixes** (NOT the fmt-phase2 pattern of
ever-deeper premise gaps — these are shallow completeness/refinement fixes):
1. `gpt5-6-sol` — the shared resolver omits a **4th input**, OpenRouter's existing (code-verified)
   `Options["reasoning_max_tokens"]`. It is retained but absent from the precedence/conflict/
   validation matrix and test acceptance, so combining it with typed effort or
   `thinking_budget_tokens` has undefined behavior. **Fix**: add `reasoning_max_tokens` to the
   resolver as a deprecated OpenRouter-only input; all other providers reject its presence; any
   effort+max_tokens combination on OpenRouter is `ErrConflictingReasoningConfig` unless a
   documented equivalence is proven; both numeric-budget option names together is always a
   conflict.
2. `gemini-3-1-pro` — the Gemini Conflict-Surface rule **over-reaches**: it forces `MaxTokens` to
   be explicitly set even for `B=0` (`"off"`), but a zero thinking budget consumes no output
   tokens. This is a hostile constraint that breaks the **target consumer docparse** (wants
   reasoning=off for PDF parse without truncating output). Anthropic already exempts `B=0`; Gemini
   should too. **Fix**: exempt `B=0`/`"off"` from Gemini's mandatory-MaxTokens check — require
   `MaxTokens > B` only for enabled thinking (`B > 0`).

**Decision for Mark (#399):** both R2 fixes are small, concrete, and converging — one is a direct
correction of Rev-1's own over-reach. **Recommended:** (1) authorize ONE more bounded revision
round (add `reasoning_max_tokens` to the resolver matrix + exempt Gemini `B=0`), then re-quorum —
this doc is close to green, unlike fmt-phase2. Alternatives: (2) amend scope (drop the
`reasoning_max_tokens` unification — keep it strictly OpenRouter-internal and out of the typed
resolver); (3) keep parked. Metered quorum+designer spend this iteration ≈ $0.23.

## Framing

> Reasoning/thinking tokens are no longer a Gemini-only concern. **Response-side**, AILANG already abstracts them as `ai.Response.ReasonTokens`. **Request-side**, only OpenAI Responses is wired, through the freeform `req.Options["reasoning_effort"]` map. This design promotes reasoning effort to a typed `ai.Request` field while preserving one narrow compatibility guarantee: an unset field and absent legacy options produce the same bytes each provider emits today. Any explicit reasoning request is validated before network dispatch and either honored exactly or rejected with a typed error. It is never silently weakened, omitted, or ignored.

## Why this changed

An earlier draft proposed `Request.ThinkingBudget *int` as a Gemini-only feature. That is the wrong factoring:

1. **Response-side accounting is already universal** — `ReasonTokens` is defined once on the shared response type.
2. **OpenAI already has an untyped request knob** — [openai/responses.go:46-53](../../../internal/ai/openai/responses.go#L46-L53) reads `Options["reasoning_effort"]`, accepts any string, defaults it to `"medium"`, and emits a Responses API reasoning block.
3. **The provider controls differ in shape, not intent** — qualitative effort for OpenAI/OpenRouter and absolute thinking budgets for Gemini/Anthropic.
4. **Latency bounds are business logic** — per AILANG's no-silent-fallback axiom, an explicit `"off"` request cannot become `"minimal"`, provider default, or a logged no-op.

## Problem

| Provider | Current request-side behavior | Failure surface |
|----------|-------------------------------|-----------------|
| OpenAI Responses | Reads untyped `Options["reasoning_effort"]`; defaults to `"medium"` | Invalid strings reach the API; no exact `"off"` contract |
| OpenAI Chat | No reasoning field in the request shape | Explicit effort would currently be ignored |
| Gemini | No `thinkingConfig` in `generationConfig` | Cannot request or validate exact thinking budgets |
| Anthropic | No `thinking` block; silently substitutes `MaxTokens=4096` when unset | A mapped thinking budget can collide with output-token limits |
| OpenRouter | Supports only `reasoning.max_tokens` through a separate option | No typed effort or exact-disable guarantee |

The dangerous case is not merely unsupported syntax. A caller may request `"off"` to enforce a latency policy and receive a response whose provider silently reasoned anyway. The client must make that outcome impossible.

## Goals

1. Promote reasoning effort to a typed field on `ai.Request`.
2. Define deterministic validation and precedence across the typed field and both legacy escape hatches.
3. Validate provider/model capability before marshaling or network dispatch.
4. Honor every explicit request exactly or return a typed, non-retryable validation error.
5. Preserve byte-identical provider request bodies only when all reasoning controls are unset.

## Non-Goals

- No `std/ai` surface change in this sprint; a language-level helper follows after the provider contract lands.
- No automatic per-model defaults or silent token-budget selection.
- No chain-of-thought content exposure; this controls token spend only.
- No assumption that an unknown model accepts a provider's reasoning parameter. Unknown capability plus an explicit request is an error.

## Design

### Typed field

```go
// ReasoningEffort controls per-call reasoning/thinking-token spend.
// Valid values are "", "off", "low", "medium", and "high".
// Empty preserves the provider's current request body exactly. Every non-empty
// value must be honored exactly by the selected provider/model or request
// construction returns ErrUnsupportedReasoningEffort before network dispatch.
ReasoningEffort string
```

The field is intentionally qualitative. Exact token counts remain available through `Options["thinking_budget_tokens"]`, but that escape hatch is subject to the same type, range, capability, conflict, and `MaxTokens` validation as the typed field.

### Fail-loud request contract

All provider entry points that construct requests (`Generate`, `Step`, and streaming variants) MUST run the same validation before JSON marshaling and before creating or dispatching an HTTP request.

Typed sentinel errors, wrapped by the existing provider/`AIError` structure with a non-retryable schema-validation code, are required:

- `ErrInvalidReasoningEffort` — a present effort is not one of `""`, `"off"`, `"low"`, `"medium"`, or `"high"`, or the legacy option is not a string.
- `ErrUnsupportedReasoningEffort` — the selected provider/model cannot honor the requested semantic exactly, including exact disablement.
- `ErrConflictingReasoningConfig` — two reasoning controls are present but disagree.
- `ErrInvalidThinkingBudget` — `thinking_budget_tokens` has an invalid Go type or provider-specific range.
- `ErrReasoningBudgetExceedsMaxTokens` — an absolute thinking budget is not strictly below `Request.MaxTokens`, or `MaxTokens` is required but unset.

Validation behavior for every effort value:

| Value | Contract |
|-------|----------|
| `""` / unset | Preserve the provider's current wire body exactly. This is the sole compatibility default and is not a fallback. OpenAI Responses therefore retains its current implicit `"medium"` block when no reasoning control is supplied; other current bodies remain unchanged. |
| `"off"` | Require exact disablement. If the provider/model cannot guarantee no reasoning, return `ErrUnsupportedReasoningEffort`; never map to `"minimal"`, omit and warn, or rely on a server-side no-op. |
| `"low"` | Emit the documented low control only for a capability-registered provider/model; otherwise return `ErrUnsupportedReasoningEffort`. |
| `"medium"` | Emit the documented medium control only for a capability-registered provider/model; otherwise return `ErrUnsupportedReasoningEffort`. |
| `"high"` | Emit the documented high control only for a capability-registered provider/model; otherwise return `ErrUnsupportedReasoningEffort`. |
| Any other string | Return `ErrInvalidReasoningEffort`. |

Capability checks MUST use an explicit provider/model capability table or equivalent deterministic predicate. Sending a field to an older model because the server might ignore it is forbidden. Unknown models are unsupported for non-empty controls until verified and registered.

### Deterministic precedence and conflicts

The three inputs are resolved in this order, but precedence never permits disagreement:

1. Validate every present input independently; an invalid lower-precedence option is still an error.
2. Resolve `Request.ReasoningEffort` and `Options["reasoning_effort"]`:
   - If only one is non-empty, use it.
   - If both are present and identical, use the typed field and emit the same wire body as one value alone.
   - If both are present and differ, return `ErrConflictingReasoningConfig`.
3. Resolve `Options["thinking_budget_tokens"]` for Gemini and Anthropic only:
   - It must have Go type `int`; floats, JSON-number wrappers, strings, unsigned integers, and other numeric types are rejected rather than coerced.
   - Gemini accepts integers `>= 0`; Anthropic accepts `0` for exact disablement or integers `>= 1024` for enabled thinking. Negative values and Anthropic values `1..1023` return `ErrInvalidThinkingBudget`.
   - If an effort is also resolved, the exact budget must equal that provider's mapped budget. A different value returns `ErrConflictingReasoningConfig`; the numeric option does not silently override the typed request.
   - On OpenAI and OpenRouter, presence of `thinking_budget_tokens` returns `ErrUnsupportedReasoningEffort`; it is never ignored.

`Options["reasoning_effort"]` remains a deprecated compatibility input. It receives the same value and capability validation as the typed field. There is no log-only failure mode.

### Provider mapping and honorable values

| Effort | OpenAI Responses + Chat | Gemini thinking-capable models | Anthropic opt-in-thinking models | OpenRouter capability-registered models |
|--------|--------------------------|---------------------------------|-----------------------------------------|-----------------------------------------|
| `""` | Preserve today's body: Responses keeps its current implicit `reasoning.effort: "medium"`; Chat omits reasoning | Omit `thinkingConfig` | Omit `thinking` | Preserve today's body |
| `"off"` | **Unsupported initially** because the supported OpenAI surface has no verified exact-off semantic; return `ErrUnsupportedReasoningEffort` | `thinkingBudget: 0` | Omit `thinking`, but only for models whose capability entry confirms thinking is opt-in; mandatory-thinking models reject | Send normalized exact-disable form `reasoning: {"effort":"none"}` only for models verified to honor it; otherwise reject |
| `"low"` | Responses: `reasoning: {"effort":"low"}`; Chat: native `reasoning_effort: "low"` | `thinkingBudget: 1024` | `thinking: {"type":"enabled","budget_tokens":1024}` | `reasoning: {"effort":"low"}` |
| `"medium"` | Responses: `reasoning: {"effort":"medium"}`; Chat: native `reasoning_effort: "medium"` | `thinkingBudget: 4096` | `thinking: {"type":"enabled","budget_tokens":4096}` | `reasoning: {"effort":"medium"}` |
| `"high"` | Responses: `reasoning: {"effort":"high"}`; Chat: native `reasoning_effort: "high"` | `thinkingBudget: 16384` | `thinking: {"type":"enabled","budget_tokens":16384}` | `reasoning: {"effort":"high"}` |

The table states the desired contract, not a license to send fields optimistically. Each non-empty row is gated by the provider/model capability table. In particular, OpenAI `"off"` MUST NOT be translated to `"minimal"`.

### Conflict Surface

`ai.Request.MaxTokens` is the maximum response/output token count and currently uses `0` to request a provider default ([provider.go:46-47](../../../internal/ai/provider.go#L46-L47)). Absolute thinking budgets consume or constrain the same request envelope on Anthropic and create the same unsafe overcommit class on Gemini. Therefore validation uses the caller's explicit `Request.MaxTokens`, not a provider client's internal default.

Exact pre-dispatch rules:

- **Gemini:** whenever a non-empty effort or `thinking_budget_tokens` resolves to an absolute budget `B` (including `B=0` for `"off"`), `Request.MaxTokens` MUST be explicitly set and MUST satisfy `MaxTokens > B`. If `MaxTokens == 0` or `B >= MaxTokens`, return `ErrReasoningBudgetExceedsMaxTokens`. Do not synthesize a max-output value.
- **Anthropic:** for enabled thinking (`B >= 1024`), `Request.MaxTokens` MUST be explicitly set and MUST satisfy the API's strict rule `MaxTokens > B`. If `MaxTokens == 0` or `B >= MaxTokens`, return `ErrReasoningBudgetExceedsMaxTokens`. `"off"` or exact budget `0` omits the `thinking` block and does not require `MaxTokens`, because no absolute thinking budget is sent.
- **OpenAI/OpenRouter:** qualitative effort does not participate in this absolute-budget check. Existing output-token validation remains in force. `thinking_budget_tokens` is rejected as unsupported rather than compared or ignored.

This deliberately rejects cases such as Anthropic `ReasoningEffort: "high"` with `MaxTokens: 4096`; the client returns a typed validation error instead of dispatching a request that the provider will reject. Current Anthropic code silently replaces unset `MaxTokens` with `4096` ([anthropic/client.go:160-168](../../../internal/ai/anthropic/client.go#L160-L168)); reasoning validation MUST occur before that defaulting and MUST NOT use the substituted value to make an explicit reasoning request appear valid.

### Wiring

- [internal/ai/provider.go](../../../internal/ai/provider.go) — add `ReasoningEffort` and typed sentinel errors.
- Shared reasoning helper — canonicalize values, resolve conflicts, map buckets, and validate capabilities/budgets for all request paths.
- [internal/ai/gemini/types.go](../../../internal/ai/gemini/types.go) plus generate/step/stream paths — add `thinkingConfig` and fail-loud absolute-budget validation.
- [internal/ai/openai/responses.go](../../../internal/ai/openai/responses.go) — preserve the unset body, validate the legacy option, and use the resolved typed value.
- [internal/ai/openai/chat.go](../../../internal/ai/openai/chat.go) — add the distinct Chat payload field and reject unsupported models.
- [internal/ai/anthropic/](../../../internal/ai/anthropic) — add `thinking`, required headers/model gating, and strict `MaxTokens > budget` validation.
- [internal/ai/openrouter/](../../../internal/ai/openrouter) — extend the normalized reasoning block with effort while retaining the existing exact-max-token control; capability-gate exact semantics.

## Verification Log

| Claim | Status | Evidence / required verification |
|-------|--------|----------------------------------|
| Gemini older-model `thinkingConfig` no-op handling | **NEEDS-LIVE-SMOKE** | Current `generationConfig` has no `thinkingConfig` field ([gemini/types.go:80-89](../../../internal/ai/gemini/types.go#L80-L89)), so the repository cannot verify the claimed server no-op. The design no longer relies on it: unregistered models reject explicit reasoning controls. Metered smoke tests may be parked, but capability entries cannot be added until verified. |
| Anthropic request currently lacks thinking configuration and beta headers | **CODE-VERIFIED** | The Generate request contains `model`, `max_tokens`, messages, temperature, and tools only ([anthropic/client.go:71-80](../../../internal/ai/anthropic/client.go#L71-L80)); dispatch sets `anthropic-version` but no thinking beta header ([anthropic/client.go:225-227](../../../internal/ai/anthropic/client.go#L225-L227)). |
| Anthropic thinking header/model constraints and strict `max_tokens > budget_tokens` behavior | **NEEDS-LIVE-SMOKE** | These are external API constraints not established by current code. Verify supported model IDs, required header/version, minimum budget, exact-off behavior, and boundary cases `max_tokens == budget_tokens` / `max_tokens == budget_tokens+1` before enabling capability entries. Metered smoke may be parked; fail-loud gating remains mandatory. |
| OpenAI Responses and Chat use different current payload shapes | **CODE-VERIFIED** | Responses has nested `reasoning` with `effort` ([openai/types.go:88-94](../../../internal/ai/openai/types.go#L88-L94), [openai/types.go:117-120](../../../internal/ai/openai/types.go#L117-L120)); Chat's request type has no reasoning field ([openai/types.go:7-16](../../../internal/ai/openai/types.go#L7-L16)). |
| OpenAI Chat native `reasoning_effort` support by model | **NEEDS-LIVE-SMOKE** | Verify the planned top-level Chat shape and model allowlist. Until verified, Chat models without a capability entry return `ErrUnsupportedReasoningEffort`. |
| OpenRouter currently passes a normalized reasoning block | **CODE-VERIFIED** | The request contains `reasoning` and currently models only `max_tokens` ([openrouter/types.go:26-37](../../../internal/ai/openrouter/types.go#L26-L37)); construction populates it only from `Options["reasoning_max_tokens"]` ([openrouter/chat.go:59-66](../../../internal/ai/openrouter/chat.go#L59-L66)). |
| OpenRouter effort and exact-off pass-through reach the selected upstream model unchanged | **NEEDS-LIVE-SMOKE** | Verify `low`/`medium`/`high`, exact-disable spelling, and routed-provider behavior per model. Unknown or dynamically ambiguous routes reject explicit effort until a deterministic capability contract exists. |

## Risks and considerations

- **Capability drift:** provider model support changes. Keep capability entries explicit and tested; unknown is an error, not an optimistic pass-through.
- **Mapping lossiness:** qualitative buckets cannot perfectly match absolute budgets. The exact-budget escape hatch remains, but conflicting controls fail rather than override.
- **Header/API drift:** Anthropic thinking headers and model constraints require live verification before enablement.
- **Compatibility:** the only body-preservation promise is the all-controls-unset case. Explicit invalid legacy options now fail loudly by design.
- **Replay determinism:** validation and conflict resolution must be shared across Generate/Step/streaming so identical requests cannot produce provider-path-dependent bodies.

## Acceptance Criteria

1. `Request.ReasoningEffort string` is documented with exactly five valid values: `""`, `"off"`, `"low"`, `"medium"`, and `"high"`.
2. Request construction validates reasoning controls before marshaling and before network dispatch on Generate, Step, and streaming paths.
3. Unsupported provider/model/value combinations return a typed `ErrUnsupportedReasoningEffort`; specifically, OpenAI `"off"` is rejected rather than mapped to `"minimal"` or omitted.
4. Invalid effort strings and non-string `Options["reasoning_effort"]` values return typed `ErrInvalidReasoningEffort` errors.
5. Invalid `Options["thinking_budget_tokens"]` types and provider-specific ranges return typed `ErrInvalidThinkingBudget` errors; unsupported providers reject the option instead of ignoring it.
6. Precedence is deterministic: identical typed/legacy values are accepted, differing values return `ErrConflictingReasoningConfig`, and a numeric exact budget must equal any simultaneously supplied effort mapping.
7. Gemini emits `thinkingBudget: 0/1024/4096/16384` only for capability-registered models. Unsupported/unknown models return a typed error rather than relying on server no-op behavior.
8. Anthropic emits enabled thinking only for capability-registered models and uses the mapped `1024/4096/16384` budgets; exact off omits thinking only where opt-in semantics are verified.
9. Gemini absolute budgets require explicit `MaxTokens > budget`; `MaxTokens == 0` or `budget >= MaxTokens` returns typed `ErrReasoningBudgetExceedsMaxTokens` before dispatch.
10. Anthropic enabled-thinking budgets require explicit `MaxTokens > budget`; `MaxTokens == 0` or `budget >= MaxTokens` returns typed `ErrReasoningBudgetExceedsMaxTokens` before dispatch.
11. OpenAI Responses emits nested `reasoning.effort`; OpenAI Chat emits its separately verified native field. Both reject unregistered models and invalid values before dispatch.
12. OpenRouter emits normalized effort only for capability-registered model/routing combinations; exact `"off"` is rejected unless live verification proves exact disablement.
13. With `ReasoningEffort == ""` and neither legacy reasoning option present, golden tests prove byte-identical request bodies for every provider and request path.
14. Table-driven tests cover all five effort values, invalid values, invalid option types/ranges, every precedence combination, unsupported models, `MaxTokens == 0`, `budget == MaxTokens`, and `budget == MaxTokens-1`.
15. CHANGELOG entry is added under v0.31.0: "AI provider requests now accept validated `Request.ReasoningEffort` controls and fail loudly when a provider/model cannot honor the requested reasoning semantic."
16. Notify ailang-parse (docparse) via `ailang messages` once shipped so it can adopt exact `"off"` for verified Gemini parse models.

## Estimated Effort

| Task | Hours |
|------|-------|
| Shared field, typed errors, resolver, capability table, and tests | 2.5 |
| Gemini wiring and conflict/MaxTokens tests | 2.0 |
| OpenAI Responses refactor and Chat support/tests | 2.0 |
| Anthropic request/header/model/budget validation and tests | 2.5 |
| OpenRouter effort extension and capability tests | 1.5 |
| Metered live smoke verification | 1.5 |
| CHANGELOG, downstream message, and docs | 1.0 |
| Buffer | 1.0 |
| **Total** | **~14h** |

## Out-of-Scope Follow-Ups

- Surface `ReasoningEffort` in `std/ai`, for example `callWithReasoning(prompt, effort)`.
- Per-workload caller policies such as explicitly selecting `"off"` for parse calls.
- Expose chain-of-thought content; response-side token accounting remains sufficient for this milestone.
- Eval-harness baseline columns for the requested effort value.
