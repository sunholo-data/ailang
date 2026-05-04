# M-AI-TOOL-LOOP: Multi-Turn AI Tool Dispatch (`std/ai.step` / `runTools`)

**Status**: Planned
**Target**: v0.17.0
**Priority**: P1 (Medium)
**Estimated**: 7-9 days (~40-54 hours) — includes typed `AIError` upstreaming from Motoko fork
**Dependencies**: M-UNIFIED-AI-PROVIDERS (v0.5.10, complete). Lands cleanly on top of M-AI-OPENROUTER (v0.16.0) but does not strictly require it.
**Coordinated with:** [arniwesth/ailang motoko branch](https://github.com/arniwesth/ailang/tree/motoko) (typed `AIError`, streaming AI) and [sunholo-data/motoko_agent](https://github.com/sunholo-data/motoko_agent) (~5,200 LOC AILANG agent harness — currently rolls its own tool dispatch in `src/core/tool_contract.ail` + `tool_runtime.ail`).

## Framing

> **AILANG AI effects support multi-turn tool dispatch with typed errors. `std/ai.step` exposes a single agent turn; `runTools` is the convenience driver. Both are deterministic and replayable from trace.**

Today `std/ai` is single-shot: `call(prompt) -> string`, `callJson(prompt, schema) -> string`. There is no way to express "model emits a tool call, host runs it, result is fed back, model continues" — the universal shape of agentic workflows. Consumers wanting that pattern (eval harness, the planned `docparse legal review` workflow in ailang-parse v0.18.0, the existing `motoko_agent` harness, anyone building Mike-class apps in AILANG) either drop into Go or roll their own tool dispatch in user-space against `_ai_call_json` (which is what motoko_agent does today in `tool_contract.ail`/`tool_runtime.ail`).

This milestone adds three things:
1. **`AIError`** — a typed error record (lifted from the Motoko fork's `std/ai_motoko.AIError`) with `provider`, `statusCode`, `retryable`, `code`. Becomes the error type for `step`, `runTools`, and new `callResult`/`callJsonResult` variants of the existing single-shot calls. String-error returns are inadequate for agent loops that must decide retry-vs-surface based on `retryable` and `statusCode`.
2. **`step`** — one model turn returning either text or tool calls.
3. **`runTools`** — convenience loop driver built on `step`.

This milestone does NOT add per-domain tool sets — those live in user code or packages. It does NOT bake in token-streaming — `step` is request/response, but a streaming variant is scoped as an in-release follow-up (see "Streaming decision" below).

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

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Extend `Request`/`Response` vs add `StepRequest`/`StepResponse` | Touches every provider; affects whether `call`/`callJson` and `step` share or diverge | human | design | high |
| Tool schema as JSON-Schema string (opaque) vs typed AILANG record | Opaque is provider-portable today; typed enables compile-time tool validation later but is a larger surface | human | design | med |
| `step` returns `tool_calls: list[ToolCall]` and finishes; loop driver in user-space | vs. baking the loop into the builtin. User-space loop is more composable (budget/approval gates) but slightly more verbose | human | design | med |
| Provider parity policy for `Step` | Do all providers support tools, or do unsupported providers raise at typecheck/runtime? | human | design | low |
| Adopt Motoko `AIError` shape verbatim, or extend it | Locks the typed-error contract for the whole std/ai surface (single-shot + tool-loop). Motoko has shipped + validated against motoko_agent. Verbatim adoption deletes the std/ai vs std/ai_motoko fork; extension delays harmonization | human + arniwesth | design | high |
| Streaming: in-`step` chunk callback vs separate `stepStream` follow-up | Affects the `step` signature shape (callback param vs not). Motoko already has `callStreamResult`; their usage data should drive the choice | human + arniwesth | design | med |

## Recommended decisions (for human ratify)

- **Extend existing `Request`/`Response`** with optional `Messages`, `Tools`, `ToolCalls`, `ToolCallID`. Single shape avoids API split. `call`/`callJson` remain as-is — they construct a single-message Request internally.
- **Tool schemas are opaque JSON Schema strings.** Same shape as `ResponseSchema` already is. Defer typed AILANG tool ADT to a later milestone.
- **`step` ends after one assistant turn.** Loop is user-space. Provide `runTools` as a convenience built on `step`. Composability beats brevity here.
- **Provider parity:** Claude, Gemini, OpenRouter implement `Step` in this sprint. OpenAI and Ollama follow if their adapters need it; otherwise they return `ErrToolsNotSupported` (mapped to `AIError{retryable: false, code: "tools_not_supported"}`) when `len(req.Tools) > 0`.
- **Adopt Motoko `AIError` verbatim** as the typed error for `step`/`runTools`. Add `callResult`/`callJsonResult` (also returning `Result[string, AIError]`) so the entire std/ai surface gains typed errors in one release. This deletes the need for the parallel `std/ai_motoko` namespace going forward — the motoko fork's `ai_motoko.ail` becomes a re-export shim during transition.
- **Streaming via separate `stepStream` follow-up in v0.17.x.** Ship `step` non-streaming in v0.17.0 to keep the API surface minimal; ship `stepStream` (and a streaming `runTools` variant) immediately after. Rationale: a streaming callback parameter on `step` complicates the signature for the 80% of consumers who don't need it, and Motoko's `callStreamResult` already returns chunks as a list (not a callback) — that pattern transposes cleanly to a `stepStream` returning chunks alongside the assistant message. Open to flipping if arniwesth prefers in-step callback.

## Proposed API

### AILANG-side (`std/ai.ail`)

```ailang
module std/ai

-- Typed error (lifted verbatim from Motoko fork's std/ai_motoko.AIError).
-- Used by step, runTools, and the new Result-returning variants of the
-- single-shot calls (callResult, callJsonResult).
export type AIError = {
  message:    string,                  -- human-readable summary
  provider:   string,                  -- "anthropic" | "gemini" | "openai" | "openrouter" | "ollama"
  statusCode: int,                     -- HTTP status (0 if not applicable)
  retryable:  bool,                    -- true for 429, 5xx, timeouts; false for 4xx
  code:       string                   -- "auth_failed" | "rate_limit" | "context_length"
                                       -- | "tools_not_supported" | "timeout" | "transport"
                                       -- | "schema_validation" | "model_not_found" | "internal"
}

-- Result-returning single-shot calls (typed errors, no exceptions).
-- The existing call/callJson/callJsonSimple stay as-is for back-compat;
-- new code should prefer the Result variants.
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

The mapping is normalized at the adapter boundary so consumers don't depend on per-provider taxonomies:

| Condition | `code` | `retryable` |
|---|---|---|
| HTTP 401/403 | `auth_failed` | false |
| HTTP 404 model not found | `model_not_found` | false |
| HTTP 429 | `rate_limit` | true |
| HTTP 5xx | `internal` | true |
| Network timeout / context cancel | `timeout` | true |
| TCP/TLS/connection-reset | `transport` | true |
| Provider says "context length exceeded" | `context_length` | false |
| Provider rejects schema (callJson) | `schema_validation` | false |
| Tools requested on Ollama (or other tools-less provider) | `tools_not_supported` | false |
| Anything else | `internal` | true (conservative) |

Adapters fill `provider`, `statusCode`, `retryable`, and `code`; `message` is the provider's human-readable error verbatim.

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
    -- Typed error: agent can decide to retry, surface, or escalate based on
    -- e.retryable / e.code, no string parsing required.
    Err(e) =>
      if e.retryable
        then println("transient ${e.code} from ${e.provider}, would retry: ${e.message}")
        else println("fatal ${e.code}: ${e.message}")
  }
}
```

## Motoko coordination

This sprint is co-designed with the Motoko ecosystem:

- **Upstream from [arniwesth/ailang motoko branch](https://github.com/arniwesth/ailang/tree/motoko):** the `AIError` shape (`provider`/`statusCode`/`retryable`/`code`) is adopted verbatim from `std/ai_motoko.AIError`. This deletes the API-design fork — once v0.17.0 ships, `std/ai_motoko.callResult` becomes a one-line re-export of `std/ai.callResult` and the parallel namespace can be retired.
- **Validated against [sunholo-data/motoko_agent](https://github.com/sunholo-data/motoko_agent):** before sprint kickoff, M0 (new) reads `motoko_agent/src/core/tool_contract.ail` + `tool_runtime.ail` to confirm the proposed `dispatch: (ToolCall) -> string` callback signature is sufficient. If motoko_agent's existing tool dispatch needs richer inputs (e.g. per-call timeout, conversation context) or richer outputs (e.g. structured error from the tool), the signature widens before M1.
- **Streaming alignment:** `stepStream` (v0.17.x follow-up) lifts Motoko's `AIStreamChunk` shape so consumers get a uniform streaming contract across single-shot and multi-turn calls.

The motoko fork's `_motoko` files become deletable in waves:
- v0.17.0: `std/ai_motoko.callResult` / `callJsonResult` → re-export shim
- v0.17.x: `std/ai_motoko.callStreamResult` → re-export shim once `stepStream` lands
- Local-endpoint URI parsing (`openai://host:port/model`) and OpenRouter/OpenAI prefix routing in the motoko fork are adjacent but separate concerns — track in a follow-up.

## Out-of-scope follow-ups

- **`stepStream` and streaming `runTools`** — token streaming for incremental UI rendering. Lifted from Motoko's `AIStreamChunk`/`AIStreamResult` shapes; ships in v0.17.x immediately after the non-streaming primitive lands.
- **Typed tool ADT.** Tool schemas as AILANG records with compile-time validation, instead of opaque JSON Schema strings.
- **Eval harness migration.** Switch the eval harness's bespoke Go tool-loop onto `std/ai.runTools`. Not blocking; can come in a later sprint.
- **Parallel tool calls.** Some providers emit multiple tool calls per turn. v1 dispatches them sequentially; parallel dispatch is a future opt-in.
- **Provider-side caching declarations** (Anthropic prompt caching, Gemini context caching) on Messages. Useful for long agent loops; defer until consumers ask.
- **Local OpenAI-compatible endpoint URI parsing** (`openai://host:port/model`). Already implemented in motoko fork's `endpoint_motoko.go`; upstream as a separate doc — orthogonal to tool dispatch.

## Acceptance criteria

- [ ] `std/ai.step` callable from AILANG; returns a `Result[StepResult, AIError]` with text + optional `tool_calls`
- [ ] `std/ai.runTools` drives the loop end-to-end against the worked example above
- [ ] `std/ai.callResult` and `callJsonResult` exist alongside legacy `call`/`callJson` and return `Result[string, AIError]`
- [ ] `AIError` shape matches `std/ai_motoko.AIError` byte-for-byte; motoko fork's parallel namespace functions become re-export one-liners
- [ ] Claude provider (`internal/ai/anthropic`) implements `Step` against Anthropic's tool-use format and populates `AIError.code`/`retryable` per the mapping table
- [ ] Gemini provider (`internal/ai/gemini`) implements `Step` against Gemini's `functionCall`/`functionResponse` format and populates `AIError.code`/`retryable`
- [ ] OpenRouter provider passes tools through unchanged (`Step` calls Generate with tool-aware request) and maps OpenRouter error envelopes to `AIError`
- [ ] Per-step trace records: messages-in, tools-advertised, tool_calls-emitted, dispatch-results, tokens, cost, AND any `AIError` (with `code`/`retryable`/`statusCode`) on failure
- [ ] Replay of a `runTools` trace reproduces the conversation turn-by-turn (deterministic given the same dispatch callback)
- [ ] OpenAI and Ollama either implement `Step` or return `AIError{code: "tools_not_supported", retryable: false}` when tools are present
- [ ] Test corpus: single-tool happy path, two-tool sequential, dispatch-error propagation, step-budget exhaustion, tools-not-supported provider, malformed tool_call from model, AND each `AIError.code` value produced under controlled fixture (rate_limit, auth_failed, timeout, context_length, schema_validation, transport, internal)
- [ ] motoko_agent compatibility check: starting from a checkout of `sunholo-data/motoko_agent`, swap one tool-loop site from custom `tool_runtime.ail` dispatch to `std/ai.runTools`; build and existing tests pass
- [ ] Linting clean: `make lint`, `make test`, `make verify-examples` all pass

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|-----------|
| Provider tool-call formats diverge in subtle ways (Anthropic IDs vs Gemini functionCall names) | High | Med | Normalize at the adapter boundary; document the mapping; HTTP-fixture tests per provider |
| Adding `Messages` on `Request` muddies the existing `SystemPrompt` + `UserPrompt` shape | Med | Low | Document precedence: `Messages` non-empty → use it; otherwise legacy path |
| Trace size balloons on long loops | Med | Low | Each step is a separate trace event; existing per-trace span budget (`AILANG_TRACE_MAX_SPANS`) already handles this |
| Loop driver in user-space increases boilerplate vs builtin loop | Low | Low | `runTools` covers the common case; users only drop to `step` when they need budget/approval interposition |

## Total effort

| Component | Estimate |
|-----------|----------|
| M0: Motoko coordination read (tool_contract.ail / tool_runtime.ail review, AIError verbatim check) | 0.5 day |
| `AIError` type + adapter error-mapping table impl | 0.5 day |
| Provider interface + Request/Response extension | 0.5 day |
| Anthropic adapter `Step` impl + AIError mapping + tests | 1 day |
| Gemini adapter `Step` impl + AIError mapping + tests | 1 day |
| OpenRouter passthrough + AIError mapping + tests | 0.5 day |
| `callResult` / `callJsonResult` Result-returning variants + builtins | 0.5 day |
| `_ai_step` builtin + AILANG types | 1 day |
| `std/ai.step` and `std/ai.runTools` AILANG impl + tests | 1 day |
| Trace schema additions (incl. AIError fields) + replay verification | 0.5 day |
| motoko_agent compatibility swap + verification | 0.5 day |
| Worked example + docs + CHANGELOG | 0.5 day |
| **Total** | **~8 days** |

Sprint plan in [m-ai-tool-loop-sprint-plan.md](m-ai-tool-loop-sprint-plan.md).
