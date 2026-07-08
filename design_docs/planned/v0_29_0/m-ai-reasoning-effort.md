---
title: "M-AI-REASONING-EFFORT — Cross-provider request-side reasoning control"
status: Planned
target: v0.17.0
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

## Framing

> Reasoning/thinking tokens are no longer a Gemini-3-only concern. **Response-side**, AILANG already abstracts this cleanly: every provider populates `ai.Response.ReasonTokens` with the same field name and semantics. **Request-side**, only OpenAI is wired — via the freeform `req.Options["reasoning_effort"]` map — while Gemini, Anthropic, and OpenRouter have no way to influence reasoning at all. This doc promotes reasoning effort to a first-class typed field on `ai.Request` and maps it consistently across all four providers.

## Why this changed

Earlier draft of this design proposed adding `Request.ThinkingBudget *int` as a Gemini-only feature, motivated by docparse wanting `thinkingBudget: 0` for `gemini-3-flash-preview` PDF parses. Re-reading the codebase showed that's the wrong shape:

1. **Response side is already universal** — [provider.go:139-140](../../../internal/ai/provider.go#L139-L140) defines `ReasonTokens int` once; [gemini/generate.go:138](../../../internal/ai/gemini/generate.go#L138), [openai/responses.go:142](../../../internal/ai/openai/responses.go#L142), [openai/chat.go:130](../../../internal/ai/openai/chat.go#L130), and [openrouter/chat.go:146](../../../internal/ai/openrouter/chat.go#L146) all funnel through it.
2. **OpenAI request side already has a knob** — [openai/responses.go:49](../../../internal/ai/openai/responses.go#L49) reads `req.Options["reasoning_effort"]` as a string. It's documented nowhere on the `Request` type, lives in a freeform map, and is silently ignored by every other provider.
3. **At least three providers expose request-side controls now**: OpenAI o-series + GPT-5 (`reasoning_effort`), Gemini 3 (`thinkingConfig.thinkingBudget`), Anthropic Claude Opus 4 / Sonnet 4 (`thinking.budget_tokens`).

Shipping a Gemini-specific field on `Request` would entrench the inconsistency: OpenAI uses `Options[...]`, Gemini would use a typed field, Anthropic would have a third path. Promoting it to a typed first-class field is the right factoring.

## Problem

Three concrete shortfalls today:

| Provider | Request-side reasoning control | Status |
|----------|-------------------------------|--------|
| OpenAI Responses (`responses.go`) | `Options["reasoning_effort"]` (`"low"\|"medium"\|"high"`) | Wired, undocumented on `Request` |
| OpenAI Chat (`chat.go`) | None | Field exists in API but not forwarded |
| Gemini (generate + step) | None | API supports `thinkingConfig`; not in `generationConfig` struct ([types.go:75-83](../../../internal/ai/gemini/types.go#L75-L83)) |
| Anthropic | None | API supports `thinking.budget_tokens`; not in our request shape |
| OpenRouter | None | Relays to underlying provider; needs pass-through |

Concrete downstream impact: `docparse` evaluated `gemini-3-flash-preview` for PDF parsing. The model produces good output but spends thinking tokens unnecessarily — there's no way for docparse (or any caller) to say "skip thinking, this is a parse workload." Same will hit any caller wanting bounded latency on Anthropic Claude 4 thinking models.

## Goals

1. **Promote `reasoning_effort` to a typed field** on `ai.Request`.
2. **Wire all four providers** to honour it (OpenAI Responses + Chat, Gemini, Anthropic, OpenRouter).
3. **Preserve back-compat** — when the field is unset, request bodies are byte-identical to today's. OpenAI continues to honour `Options["reasoning_effort"]` if set, with a deprecation note in the Request docstring.
4. **Document in `prompts/`** so std/ai callers know the option exists at the AILANG language level (likely a follow-up; see Out-of-Scope).

## Non-Goals

- No std/ai surface changes in this sprint. The new field is plumbed through `ai.Request` first; surfacing in `std/ai` (e.g. `callWithReasoning(prompt, "off")`) is a follow-up gated on this landing.
- No automatic per-model defaults. Defaults belong in callers where workload context is known.
- No replay-store invalidation. Unset field = byte-identical request body, so existing replay hashes are unaffected.
- Not aiming to expose **chain-of-thought content** (Gemini `includeThoughts`, Anthropic `thinking` block content). Token-budget control only.

## Design

### The unified field

```go
// internal/ai/provider.go
type Request struct {
    // ...existing fields...

    // ReasoningEffort controls per-call reasoning/thinking-token spend on
    // providers that expose such a knob (currently: OpenAI o-series + GPT-5,
    // Gemini 3+, Anthropic Claude 4 thinking models). Providers that do not
    // expose a knob ignore the field. Pre-thinking models on supporting
    // providers also ignore it server-side (verified: gemini-2.5-flash
    // accepts thinkingConfig as a no-op).
    //
    // Values:
    //   ""        — provider default (current implicit behaviour, fully back-compat)
    //   "off"     — disable reasoning where supported
    //   "low"     — minimum reasoning budget
    //   "medium"  — moderate reasoning budget
    //   "high"    — maximum reasoning budget
    //
    // Each provider maps these to its native parameter (table below).
    // Quantitative budgets (Gemini, Anthropic) are not exposed directly to
    // keep the abstraction small; callers needing exact token counts should
    // use Options["thinking_budget_tokens"] (provider-specific escape hatch).
    ReasoningEffort string
}
```

### Provider mapping

| Effort   | OpenAI (Responses + Chat) | Gemini             | Anthropic                              | OpenRouter |
|----------|---------------------------|--------------------|----------------------------------------|------------|
| `""`     | omit (server default)     | omit `thinkingConfig` | omit `thinking`                       | passthrough |
| `"off"`  | `reasoning.effort: "minimal"` (or omit + log) | `thinkingBudget: 0` | omit `thinking` (no "off" semantic; defaults to disabled) | passthrough |
| `"low"`  | `reasoning.effort: "low"` | `thinkingBudget: 1024` | `thinking: {type: "enabled", budget_tokens: 1024}` | passthrough |
| `"medium"` | `reasoning.effort: "medium"` | `thinkingBudget: 4096` | `thinking: {type: "enabled", budget_tokens: 4096}` | passthrough |
| `"high"` | `reasoning.effort: "high"` | `thinkingBudget: 16384` | `thinking: {type: "enabled", budget_tokens: 16384}` | passthrough |

Notes:
- OpenAI lacks a true "off" — `minimal` is the floor; we may also accept `"off"` as a synonym and log a one-line warning. To be decided in implementation.
- Anthropic doesn't model "off" — absence of `thinking` is the off state.
- Gemini token budgets above are seed defaults; tunable via constants in `internal/ai/gemini/`.
- OpenRouter relays whichever underlying provider; it forwards `reasoning` per [its docs](https://openrouter.ai/docs/use-cases/reasoning-tokens) using the OpenAI-compatible shape.

### Escape hatch for exact budgets

Callers needing exact token counts (e.g. "cap Gemini at exactly 256 thinking tokens") use `Options`:

```go
req.Options = map[string]any{
    "thinking_budget_tokens": 256,  // Gemini + Anthropic; ignored by OpenAI
}
```

When `Options["thinking_budget_tokens"]` is set on a provider that supports it, it overrides whatever bucket `ReasoningEffort` maps to. This keeps the typed surface small without locking out precision use cases.

### Back-compat for OpenAI

OpenAI Responses currently reads `req.Options["reasoning_effort"]` ([responses.go:49](../../../internal/ai/openai/responses.go#L49)). New rule:

1. If `req.ReasoningEffort != ""`, use it.
2. Else if `req.Options["reasoning_effort"]` is set, use it (with a one-line `_ = ` log to encourage migration).
3. Else default to `"medium"` (current behaviour).

Existing eval baselines and recorded calls that set `Options["reasoning_effort"]` continue to work unchanged.

### Wiring

Touched files:

- [internal/ai/provider.go](../../../internal/ai/provider.go) — add `ReasoningEffort string` field with full doc comment.
- [internal/ai/gemini/types.go](../../../internal/ai/gemini/types.go) — add `thinkingConfig` struct, `ThinkingConfig *thinkingConfig` on `generationConfig`.
- [internal/ai/gemini/generate.go](../../../internal/ai/gemini/generate.go) + [internal/ai/gemini/step.go](../../../internal/ai/gemini/step.go) — translate `req.ReasoningEffort` (and `Options["thinking_budget_tokens"]` override) to `thinkingConfig`.
- [internal/ai/openai/responses.go](../../../internal/ai/openai/responses.go) — accept new field, fall back to `Options["reasoning_effort"]`.
- `internal/ai/openai/chat.go` — newly accept reasoning effort (currently it ignores it; needs the same translation).
- `internal/ai/anthropic/` — add `thinking` block to request shape; honour the field on supported models.
- [internal/ai/openrouter/](../../../internal/ai/openrouter) — translate to OpenAI-compatible `reasoning: {effort: ...}` body field.

A small shared helper `internal/ai/reasoning.go` could centralise the bucket→tokens mapping so Gemini and Anthropic stay in sync; tradeoff is one more file vs. inline constants in each provider. Lean toward the helper.

## Risks and considerations

- **Anthropic API surface stability**: `thinking` block requires `anthropic-beta: extended-thinking-2024-12-19` (or current header). Add gated by model name or always send for thinking-capable models. Implementation detail; smoke-test before merge.
- **Token budget tuning**: 1024/4096/16384 are reasonable seeds but may want per-provider tuning. Pick conservative defaults; revisit after eval-harness data.
- **Mapping lossiness**: there's no perfect bijection between OpenAI's qualitative scale and Gemini/Anthropic's quantitative budgets. We're choosing a useful coarse-grained surface; the `Options` escape hatch handles precision cases.
- **Docs drift risk**: with 4 providers, mapping table is easy to fall out of sync. Mitigate by table-driven tests in `internal/ai/reasoning_test.go` that assert each bucket → expected JSON for each provider.
- **OpenAI `"off"` ambiguity**: punt to implementation — either accept and map to `"minimal"`, or reject with a clear error. Document whichever choice is made.

## Acceptance criteria

1. `Request.ReasoningEffort string` added with full docstring listing all five values.
2. Helper `mapReasoningEffort(effort) (tokens int, ok bool)` exists in `internal/ai/reasoning.go` (or per-provider equivalent), table-tested.
3. **Gemini**: request with `ReasoningEffort: "off"` produces JSON containing `"thinkingConfig":{"thinkingBudget":0}`; unset request produces no `thinkingConfig` key. Live smoke test against `gemini-3-flash-preview` returns `thoughtsTokenCount == 0`.
4. **OpenAI Responses**: request with `ReasoningEffort: "high"` produces `"reasoning":{"effort":"high"}`. Falling back to `Options["reasoning_effort"]` still works (test).
5. **OpenAI Chat**: request with `ReasoningEffort: "low"` is honoured (currently ignored).
6. **Anthropic**: request with `ReasoningEffort: "medium"` produces `"thinking":{"type":"enabled","budget_tokens":4096}` against a thinking-capable model.
7. **OpenRouter**: request with `ReasoningEffort: "high"` produces an OpenAI-compatible `"reasoning":{"effort":"high"}` body field; relayed to underlying model.
8. **Escape hatch**: `Options["thinking_budget_tokens"] = 256` on Gemini overrides `ReasoningEffort: "low"` and produces `thinkingBudget: 256`.
9. **Back-compat**: unset field produces byte-identical request bodies for all four providers (table-driven golden test).
10. CHANGELOG entry under v0.16.0: "AI providers now accept `Request.ReasoningEffort` (off/low/medium/high) for cross-provider reasoning-token control. OpenAI's existing `Options[\"reasoning_effort\"]` continues to work."
11. Notify ailang-parse (docparse) via `ailang messages` once shipped so they can adopt for `gemini-3-flash-preview` parse calls.

## Estimated effort

| Task | Hours |
|------|-------|
| Add `Request.ReasoningEffort` + `internal/ai/reasoning.go` helper + tests | 1.5 |
| Gemini wiring (types, generate, step) + tests | 2.0 |
| OpenAI Responses (refactor existing) + Chat (new) + tests | 2.0 |
| Anthropic wiring (request shape, header gating) + tests | 2.5 |
| OpenRouter wiring + tests | 1.5 |
| Live smoke tests across all four providers | 1.5 |
| CHANGELOG, downstream message, doc updates | 1.0 |
| Buffer | 2.0 |
| **Total** | **~14h** |

## Out-of-scope follow-ups

- Surface `ReasoningEffort` in `std/ai` (e.g. `callWithReasoning(prompt, effort)` builtin).
- Per-model default policies (e.g. auto-`off` for parse-mimetype calls).
- Expose chain-of-thought content (Gemini `includeThoughts`, Anthropic `thinking` block content) — only useful with deliberate audit/eval use case.
- Eval-harness baseline columns for `ReasoningEffort` (already records `ReasonTokens` on the response side).
