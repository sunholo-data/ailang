# M-AI-TOOL-LOOP: Multi-Turn AI Tool Dispatch (`std/ai.step` / `runTools`)

**Status**: Planned (refreshed 2026-05-05 against v0.15.1 codebase — most of the Motoko coordination work has already shipped through v0.15.0/v0.15.1)
**Target**: v0.17.0
**Priority**: P1 (Medium)
**Estimated**: 7 days (~42-52 hours) — slimmed from 8d after v0.15.x already landed AIError + streaming
**Dependencies**: All blocking deps shipped:
- ✅ M-UNIFIED-AI-PROVIDERS (v0.5.10)
- ✅ M-AI-OPENROUTER (v0.16.x — wired into `ailang run` 2026-05-04, commit `67254452`)
- ✅ M-AI-EFFECT-MODES (v0.15.0 — bare `!{AI}` desugars to `!{AI[mode=fixed]}`; `runTools` will inherit)
- ✅ M-AI-PROVIDER-CONFIG (v0.15.0)
- ✅ M-AI-STREAMING-HELPER (v0.15.0) and M-AI-CALL-STREAM-HELPER (v0.15.1) — `AIError` type already exists in `std/ai/streaming.ail`; `callStream` already returns `Result[string, AIError]`

**Coordinated with:** [motoko-integration-sequence.md](../motoko-integration-sequence.md) master plan (Phase 2 work — this doc IS one of the Phase 2 milestones) and [motoko-agent-v0.15.0-migration.md](../motoko-agent-v0.15.0-migration.md). The arniwesth fork is being archived as part of Phase 3; this milestone closes the last upstream gap that motoko_agent's hand-rolled `tool_contract.ail`/`tool_runtime.ail` modules currently fill.

## Framing

> **AILANG AI effects support multi-turn tool dispatch. `std/ai.step` exposes a single agent turn; `runTools` is the convenience driver. Both reuse the existing `AIError` type from `std/ai/streaming` and are deterministic and replayable from trace.**

Today `std/ai` is single-shot for non-streaming calls: `call(prompt) -> string`, `callJson(prompt, schema) -> string`. There is no way to express "model emits a tool call, host runs it, result is fed back, model continues" — the universal shape of agentic workflows. Consumers wanting that pattern (eval harness, the planned `docparse legal review` workflow in ailang-parse v0.18.0, the existing `motoko_agent` harness, anyone building Mike-class apps in AILANG) either drop into Go or roll their own tool dispatch in user-space against `_ai_call_json` (which is what motoko_agent does today in `tool_contract.ail`/`tool_runtime.ail`).

The streaming side of this story already shipped in v0.15.0/v0.15.1: `callStream(provider, model, messagesJson) -> Result[string, AIError] ! {AI, Stream, Net}` is the synchronous accumulator, with the lower-level `openaiCompatStream`/`anthropicStream` for per-token control. **Tool dispatch is the remaining gap.**

This milestone adds three things:
1. **`step`** — one model turn returning either text or tool calls. Returns `Result[StepResult, AIError]` reusing the existing `AIError` type.
2. **`runTools`** — convenience loop driver built on `step`.
3. **`callResult` / `callJsonResult`** — Result-returning variants of the existing non-streaming `call` / `callJson`. Brings the non-streaming single-shot calls into parity with `callStream` (which already returns `Result[string, AIError]`). Closes the last surface where consumers hit `panic` instead of typed errors.

This milestone does NOT add per-domain tool sets — those live in user code or packages. It does NOT add token streaming for `step` (orthogonal; consumers needing per-token tool-call streaming compose `step` with the existing `openaiCompatStream` primitives in user-space; a future `stepStream` may unify them once usage data shows demand).

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Each `step` call is deterministic given inputs + provider; the loop driver is pure AILANG so the entire conversation replays from the message list |
| A2: Replayability | +2 | Every `step` records full request (messages + tools) and response (text + tool_calls) in the trace; replay reproduces the conversation turn-by-turn |
| A3: Effect Legibility | +1 | `step` and `runTools` both carry `! {AI}`. No new effect row needed — tool dispatch is in user-space, so its effects (FS, Net, etc.) appear naturally on the dispatch callback |
| A4: Explicit Authority | +1 | The dispatch callback is a normal AILANG function with its own effects; capabilities flow through it. Tool definitions are inert data (JSON Schema strings) |
| A5: Bounded Verification | 0 | Type-checking unchanged; tool schemas are opaque strings (JSON Schema) by design |
| A6: Safe Concurrency | 0 | No concurrency changes; `step` is sequential within a conversation |
| A7: Machines First | +2 | Unblocks agentic workflows in AILANG entirely. Without this, every AI agent has to be written in Go. With it, agents are AILANG programs the same way `callJson`-based extractors already are |
| A8: Minimal Syntax | +1 | No new syntax; new module-level functions and record types only |
| A9: Cost Visibility | +1 | Each `step` records token usage + cost the same way `call` does. `runTools` aggregates across the loop |
| A10: Composability | +2 | `step` is the primitive; `runTools` composes it with a dispatch callback. Consumers can interpose budget gates, approval flows, tracing, branching — none of which require provider changes |
| A11: Structured Failure | +2 | Typed `AIError` (provider/statusCode/retryable/code) replaces string errors across `step`/`runTools` AND existing single-shot calls (via `callResult`/`callJsonResult`); enables retry logic without regex-parsing error messages |
| A12: System Boundary | +1 | Same HTTP boundary as existing `call`; no new external surface. Tool dispatch is internal AILANG code |

**Net Score: +13** → **Decision: ✅ Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): Each step is deterministic; loop is pure AILANG; replay works
- [x] A3 (Effects): Tool dispatch effects flow through the callback's effect row, naturally visible
- [x] A4 (Authority): No ambient capability; dispatch callback is a typed AILANG function
- [x] A7 (Machines First): Unblocks agentic AILANG entirely

### Decision Thresholds

Net score +12 ≥ +2, no −1 on hard-violation axioms → **Proceed**.

## Problem Statement

**Current State:**
- `std/ai` exposes single-shot calls: `call`, `callJson`, `callJsonSimple`, `callImage`, `callImageBase64`
- All AI providers (`internal/ai/openai`, `anthropic`, `gemini`, `ollama`, `openrouter`) implement `Provider.Generate(ctx, *Request) (*Response, error)` — single request, single response
- The `Request`/`Response` types have no concept of `Messages` (multi-turn conversation), `Tools` (tool schemas), or `ToolCalls` (model-emitted invocations)
- Consumers wanting an agent loop must wrap the AILANG runtime from Go and drive the loop in Go — defeating the point of effect-typed AILANG

**Impact:**
- **Agentic AILANG is impossible in pure AILANG.** Anyone building a real agent (legal-AI workflow, code-review bot, doc summarizer with retrieval) drops into Go
- **No tool-use determinism story.** Even if a Go consumer drives a loop, the per-step trace is opaque — we don't capture the tool calls or the dispatch results in the AI trace
- **Eval harness duplication.** The eval harness has its own bespoke tool-loop in Go; folding it onto the same primitive would unify two implementations
- **`docparse legal review` (planned in ailang-parse v0.18.0) is blocked** until this lands

## Goals

**Primary Goal:** Add `std/ai.step` (one model turn with optional tool calls) and `std/ai.runTools` (loop driver) as language-level primitives. Extend the AI provider interface to support multi-turn `Messages` + `Tools`. Capture tool calls and dispatch results in the trace.

**Success Metrics:**
- AILANG programs can drive a complete agent loop in pure AILANG, with all per-turn cost/token data captured in trace
- Both Claude and Gemini providers implement the new `Step` method against their native tool-use formats
- The 7-line "agent that reads a doc and answers a question" example from the design works end-to-end on both providers
- Eval harness can opt into using `std/ai.runTools` for tool-using benchmarks (migration not required in this sprint)
- `docparse legal review` (downstream, ailang-parse v0.18.0) becomes implementable

## High-Impact Decisions

Decisions resolved in v0.15.x are marked ✅ — only one open decision remains.

| Decision | Why High Impact | Status |
|----------|-----------------|--------|
| Adopt the existing v0.15.0 `AIError` shape `{code, message, retryable}` (vs extend with `provider`/`statusCode`) | Locks typed-error contract across single-shot + streaming + tool-loop | ✅ **Resolved**: reuse v0.15.0 shape as-is. Provider/statusCode deferred to v0.17.x if motoko_agent or eval harness telemetry need them — record fields are additive |
| `step` returns `tool_calls: list[ToolCall]` and finishes; loop driver in user-space | User-space loop is more composable (budget/approval gates) but slightly more verbose | ✅ **Recommended**: user-space `runTools`, same composability rationale that `callStream` already validated |
| Tool schema as JSON-Schema string (opaque) vs typed AILANG record | Opaque is provider-portable today; typed enables compile-time tool validation later | ✅ **Recommended**: opaque JSON Schema string, same as existing `ResponseSchema` field |
| Extend `Request`/`Response` vs add `StepRequest`/`StepResponse` | Touches every provider; affects whether `call`/`callJson` and `step` share or diverge | ✅ **Recommended**: extend existing types, additive only. OpenRouter already proved this pattern works with its `Routing` field addition |
| Provider parity policy for `Step` | All providers tool-capable, or unsupported ones raise typed `tools_not_supported`? | ✅ **Recommended**: Claude + Gemini + OpenRouter + OpenAI ship real `Step` (all four already on tool-capable HTTP shapes); Ollama returns `AIError{code: "tools_not_supported"}` |
| Streaming `step` (in-step chunk callback vs separate `stepStream` follow-up) | Affects the `step` signature | ✅ **Resolved**: ship `step` non-streaming in v0.17.0; defer streaming entirely. Consumers needing per-token streaming compose `step` with the existing `openaiCompatStream` primitives in user-space. Re-evaluate `stepStream` only if motoko_agent or another consumer requests it — current `callStream` covers the streaming use case for non-tool-using calls |
| ✅ **`dispatch` callback signature** | M0 read motoko_agent's `tool_contract.ail` + `tool_runtime.ail` (2026-05-05). Confirmed: `dispatch: (ToolCall) -> string` is sufficient. Notes: (a) AILANG callbacks are effect-polymorphic by default — surface syntax `(T) -> U` compiles to `(T) -> U ! ε`, so motoko_agent's `! {FS, Process}` dispatch propagates through `runTools`'s effect row (proven via `std/stream.onEvent`'s iface.json). (b) motoko_agent uses `Json` for arguments; bridging is one `decode()` call at the dispatch boundary. (c) Their `ToolResultItem` variants get JSON-stringified to a `ToolResultEnvelope` string before going back to the model — matches our `string` return. | ✅ Resolved |

## Proposed API

### AILANG-side (`std/ai.ail`)

```ailang
module std/ai

-- AIError is the typed error already shipped in std/ai/streaming.ail
-- (v0.15.0). This milestone re-exports / reuses it from std/ai so the
-- non-streaming surface gains parity. Existing shape:
--
--   export type AIError = {
--     code: string,        -- "auth_failed" | "rate_limit" | "timeout" | ...
--     message: string,
--     retryable: bool
--   }
--
-- Note: provider and statusCode are intentionally NOT in v1 of this type
-- (decided when callStream shipped in v0.15.1 — minimal shape covers
-- the dominant retry-vs-surface decision). If consumers (motoko_agent,
-- eval harness) report needing them for telemetry, extend in v0.17.x —
-- the shape is additive-extensible without breakage since AIError is a
-- record.

import std/ai/streaming (AIError)

-- Result-returning single-shot calls (typed errors, no exceptions).
-- The existing call/callJson/callJsonSimple stay as-is for back-compat;
-- new code should prefer the Result variants. These bring non-streaming
-- single-shot calls to parity with callStream which already returns
-- Result[string, AIError].
export func callResult(input: string) -> Result[string, AIError] ! {AI} =
  _ai_call_result(input)

export func callJsonResult(input: string, schema: string)
  -> Result[string, AIError] ! {AI} =
  _ai_call_json_result(input, schema)

-- Tool-dispatch types:

export type ToolSchema = {
  name:        string,
  description: string,
  parameters:  string                  -- JSON Schema as a string
}

export type ToolCall = {
  id:        string,
  name:      string,
  arguments: string                    -- JSON-encoded arguments
}

export type Message = {
  role:         string,                -- "user" | "assistant" | "tool" | "system"
  content:      string,
  tool_calls:   list[ToolCall],        -- assistant only; empty for other roles
  tool_call_id: Option[string]         -- "tool" role only; matches a prior tool_call.id
}

export type StepResult = {
  message:       Message,              -- assistant message produced this step
  tool_calls:    list[ToolCall],       -- empty = loop terminates
  finish_reason: string,               -- "stop" | "tool_calls" | "length" | "error"
  input_tokens:  int,
  output_tokens: int
}

-- Single step: feed messages + tools, get an assistant message back.
-- If tool_calls is non-empty, the host dispatches them and calls step
-- again with the results appended. Loop is in user-space.
export func step(
  model: string,
  messages: list[Message],
  tools: list[ToolSchema]
) -> Result[StepResult, AIError] ! {AI} = _ai_step(model, messages, tools)

-- Convenience driver: loops step + dispatch until finish_reason != "tool_calls"
-- or step_budget exhausted. Dispatch is a callback the caller provides.
-- The dispatch callback's effects flow through to runTools' effect row.
export func runTools(
  model: string,
  messages: list[Message],
  tools: list[ToolSchema],
  dispatch: (ToolCall) -> string,
  step_budget: int
) -> Result[list[Message], AIError] ! {AI}
```

#### Provider-error → `AIError.code` mapping

Reuses the same code vocabulary already established by `callStream`'s `StreamErrorKind → AIError` flattening (see [std/ai/streaming.ail](../../../std/ai/streaming.ail) lines 134–150). New codes for the tool-loop surface are additive:

| Condition | `code` | `retryable` | Status |
|---|---|---|---|
| HTTP 401/403 | `AuthFailed` | false | already in v0.15.1 |
| HTTP 429 | `rate_limit` | true | new — was bundled into `internal` pre-v0.17 |
| HTTP 5xx | `internal` | true | new |
| Network timeout / context cancel | `Timeout` | true | already in v0.15.1 |
| TCP/TLS/connection-reset | `ConnectionFailed` | true | already in v0.15.1 |
| Provider says "context length exceeded" | `context_length` | false | new |
| Provider rejects schema (callJson) | `schema_validation` | false | new |
| Tools requested on a tools-less provider | `tools_not_supported` | false | new |
| AI cap budget overflow | `BudgetExhausted` | false | already in v0.15.1 |
| Provider name not registered | `ProviderNotFound` | false | already in v0.15.1 |
| Streaming requested but disabled in spec | `CapabilityNotSupported` | false | already in v0.15.1 (irrelevant for non-streaming `step`) |
| Malformed upstream response | `ProtocolError` | false | already in v0.15.1 |
| Anything else | `internal` | true (conservative) | new |

The classifier extends `internal/effects/stream.go`'s existing `wrapErrAsAIError` (or factors a shared `internal/ai/errors.go` if call sites multiply) — single source of truth so callStream and step return identical error shapes for identical underlying conditions.

### Go-side (`internal/ai/provider.go`)

```go
// New fields on Request (additive, non-breaking):
type Request struct {
    // ... existing fields ...

    // Messages, when non-empty, supersedes SystemPrompt + UserPrompt.
    // Used for multi-turn conversations and tool dispatch.
    Messages []Message

    // Tools advertises tool schemas the model may call. Empty = no tools.
    Tools []ToolSchema
}

type Message struct {
    Role       string     // "user" | "assistant" | "tool" | "system"
    Content    string
    ToolCalls  []ToolCall // assistant only
    ToolCallID string     // tool-result message only
}

type ToolSchema struct {
    Name        string
    Description string
    Parameters  string // JSON Schema
}

type ToolCall struct {
    ID        string
    Name      string
    Arguments string // JSON-encoded
}

// New fields on Response (additive):
type Response struct {
    // ... existing fields ...

    // ToolCalls, when non-empty, indicates the model wants the host to
    // dispatch these tools and feed results back via a follow-up Step call.
    ToolCalls []ToolCall

    // FinishReason: "stop" | "tool_calls" | "length" | "error"
    FinishReason string
}

// New method on Provider:
type Provider interface {
    Generate(ctx context.Context, req *Request) (*Response, error)
    // Step is multi-turn aware. Same Request/Response types; uses
    // req.Messages and req.Tools, returns resp.ToolCalls and FinishReason.
    // Providers that don't support tools return ErrToolsNotSupported when
    // len(req.Tools) > 0.
    Step(ctx context.Context, req *Request) (*Response, error)
    Name() string
}
```

A new builtin `_ai_step(model, messages, tools) -> StepResult` wires AILANG's Message/ToolCall record types to the Go `Provider.Step`.

### Worked example (target end-state)

```ailang
module example/agent

import std/ai (step, runTools, ToolCall, Message, ToolSchema, AIError)
import std/io (println)
import std/option (Some, None)
import std/result (Result, Ok, Err)

let read_doc_schema: ToolSchema = {
  name: "read_doc",
  description: "Read the named document.",
  parameters: "{\"type\":\"object\",\"properties\":{\"name\":{\"type\":\"string\"}},\"required\":[\"name\"]}"
}

func dispatch(call: ToolCall) -> string =
  if call.name == "read_doc"
    then "[document body for ${call.arguments}]"
    else "unknown tool: ${call.name}"

export func main() -> () ! {AI, IO} {
  let messages: list[Message] = [
    {role: "user", content: "Summarize doc 'nda.docx'.", tool_calls: [], tool_call_id: None}
  ]
  match runTools("claude-sonnet-4-6", messages, [read_doc_schema], dispatch, 8) {
    Ok(final_messages) => println(final_messages),
    -- Typed error: agent can decide to retry, surface, or escalate based
    -- on e.retryable / e.code, no string parsing required.
    Err(e) =>
      if e.retryable
        then println("transient ${e.code}, would retry: ${e.message}")
        else println("fatal ${e.code}: ${e.message}")
  }
}
```

## Motoko coordination

This sprint is Phase 2 of [motoko-integration-sequence.md](../motoko-integration-sequence.md), the master plan tracking the multi-release sequence that lets `motoko_agent` drop its AILANG fork dependency. Most of the coordination has already happened upstream of this doc:

- ✅ **Token streaming** shipped in v0.15.0 (M-AI-STREAMING-HELPER) and v0.15.1 (M-AI-CALL-STREAM-HELPER) — `callStream` returns `Result[string, AIError]`, the synchronous accumulator that closed motoko_agent's hand-rolled streaming boilerplate gap.
- ✅ **Custom OpenAI base-URL routing** is now expressible as `[[ai_provider]]` config (M-AI-PROVIDER-CONFIG, v0.15.0).
- ✅ **OpenRouter prefix routing** is the built-in `openrouter` provider (M-AI-OPENROUTER, v0.16.x).
- ✅ **`AIError` typed-error shape** is the v0.15.0 `std/ai/streaming.AIError` `{code, message, retryable}`. This sprint reuses it (no new shape) — every additional Result-returning AI surface inherits the same error type.

What this sprint adds for Motoko:
- `step` / `runTools` lets motoko_agent retire its `tool_contract.ail` + `tool_runtime.ail` user-space tool dispatch (currently ~200 LOC of agent-loop boilerplate built over `_ai_call_json`).
- `callResult` / `callJsonResult` lets motoko_agent retire any `try`-style wrapping around `_ai_call` / `_ai_call_json` — non-streaming calls now return `Result[string, AIError]` matching the streaming surface.

After this sprint, [motoko-agent-v0.15.0-migration.md](../motoko-agent-v0.15.0-migration.md) (the cross-repo migration plan, currently focused on streaming + provider config) extends to cover the tool-loop swap as well — the eventual Phase 3 PR collapses to "swap one streaming site, one tool-loop site, drop the fork."

## Out-of-scope follow-ups

- **Streaming `step`** — token streaming for incremental tool-call UI. Compose `step` with `openaiCompatStream` in user-space today; promote to `stepStream` only if a real consumer asks for it.
- **`AIError.provider` and `AIError.statusCode` fields** — extend the v0.15.0 shape if motoko_agent or eval harness telemetry need them. Record fields are additive; safe to defer.
- **Typed tool ADT.** Tool schemas as AILANG records with compile-time validation, instead of opaque JSON Schema strings.
- **Eval harness migration.** Switch the eval harness's bespoke Go tool-loop onto `std/ai.runTools`. Not blocking; can come in a later sprint. `motoko_agent` becoming an executor (per [m-bench-motoko-executor.md](m-bench-motoko-executor.md)) is the more direct dogfooding path.
- **Parallel tool calls.** Some providers emit multiple tool calls per turn. v1 dispatches them sequentially; parallel dispatch is a future opt-in.
- **Provider-side caching declarations** (Anthropic prompt caching, Gemini context caching) on Messages. Useful for long agent loops; defer until consumers ask.

## Acceptance criteria

- [ ] `std/ai.step` callable from AILANG; returns a `Result[StepResult, AIError]` with text + optional `tool_calls`
- [ ] `std/ai.runTools` drives the loop end-to-end against the worked example above
- [ ] `std/ai.callResult` and `callJsonResult` exist alongside legacy `call`/`callJson` and return `Result[string, AIError]`
- [ ] All four functions reuse the existing `std/ai/streaming.AIError` type — no new error type introduced
- [ ] Claude (`internal/ai/anthropic`), Gemini (`internal/ai/gemini`), OpenAI (`internal/ai/openai`), and OpenRouter (`internal/ai/openrouter`) implement `Step` against their native tool-use formats and populate `AIError.code`/`retryable` per the mapping table
- [ ] Ollama returns `AIError{code: "tools_not_supported", retryable: false}` when tools are present, falls through to existing single-shot path otherwise
- [ ] Per-step trace records: messages-in, tools-advertised, tool_calls-emitted, dispatch-results, tokens, cost, AND any `AIError` on failure
- [ ] Replay of a `runTools` trace reproduces the conversation turn-by-turn (deterministic given the same dispatch callback)
- [ ] Test corpus: single-tool happy path, two-tool sequential, dispatch-error propagation, step-budget exhaustion, tools-not-supported provider, malformed tool_call from model, AND each new `AIError.code` value produced under controlled fixture (rate_limit, context_length, schema_validation, tools_not_supported)
- [ ] motoko_agent compatibility check: starting from `arniwesth/motoko_agent`, swap one tool-loop site from custom `tool_runtime.ail` dispatch to `std/ai.runTools`; build and existing tests pass. Diff committed as the migration example. Coordinate landing with the broader [motoko-agent-v0.15.0-migration.md](../motoko-agent-v0.15.0-migration.md) PR
- [ ] Linting clean: `make lint`, `make test`, `make verify-examples` all pass

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|-----------|
| Provider tool-call formats diverge in subtle ways (Anthropic IDs vs Gemini functionCall names) | High | Med | Normalize at the adapter boundary; document the mapping; HTTP-fixture tests per provider |
| Adding `Messages` on `Request` muddies the existing `SystemPrompt` + `UserPrompt` shape | Med | Low | Document precedence: `Messages` non-empty → use it; otherwise legacy path |
| Trace size balloons on long loops | Med | Low | Each step is a separate trace event; existing per-trace span budget (`AILANG_TRACE_MAX_SPANS`) already handles this |
| Loop driver in user-space increases boilerplate vs builtin loop | Low | Low | `runTools` covers the common case; users only drop to `step` when they need budget/approval interposition |

## Total effort

Slimmed from the original 8d estimate now that v0.15.0/v0.15.1 shipped `AIError`, `callStream`, OpenRouter, and the provider-config infrastructure that this milestone composes onto.

| Component | Estimate | Notes |
|-----------|----------|-------|
| M0: dispatch-callback signature confirmation (read motoko_agent's `tool_contract.ail`/`tool_runtime.ail`) | 0.25 day | AIError shape no longer in scope — already shipped |
| Provider interface + Request/Response extension + extend `wrapErrAsAIError` for new codes | 0.75 day | New codes `rate_limit`, `context_length`, `schema_validation`, `tools_not_supported` |
| Anthropic adapter `Step` + tests | 1 day | |
| Gemini adapter `Step` + tests | 1 day | |
| OpenAI / OpenRouter / Ollama parity + tests | 0.5 day | OpenAI gets real Step; OpenRouter is passthrough; Ollama returns `tools_not_supported` |
| `callResult` / `callJsonResult` builtins + AILANG wrappers | 0.5 day | Reuses existing `_ai_call` / `_ai_call_json` paths |
| `_ai_step` builtin + AILANG record types (`Message`, `ToolCall`, `ToolSchema`, `StepResult`) | 1 day | |
| `std/ai.step` / `runTools` AILANG impl + tests + worked example | 1 day | |
| Trace schema additions (mirroring existing `streamCall` span shape) + replay verification | 0.5 day | |
| motoko_agent compat swap (one site) + coordinate with [motoko-agent-v0.15.0-migration.md](../motoko-agent-v0.15.0-migration.md) | 0.5 day | |
| Docs + CHANGELOG + design-doc move to implemented/ | 0.25 day | |
| **Total** | **~7 days** | (vs 8d in pre-refresh estimate) |

Sprint plan in [m-ai-tool-loop-sprint-plan.md](m-ai-tool-loop-sprint-plan.md).
