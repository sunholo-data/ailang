# OpenAI-Step Refusal / content_filter Classification (issue #2123)

**Status**: Planned (design only — not sprint authorization)
**Target**: v0.57.2 (planning folder only — no ancestry claim about any released version, see "Baseline Distinction")
**Priority**: P1
**Estimated**: 2–3 days
**Dependencies**: None. Must compose with (not duplicate) the in-band SSE error-envelope work done on unmerged source `2063f4ceb` — see "Baseline Distinction".

**Scope note**: this change touches `internal/ai/` (Go provider adapters) only — no parser/typechecker/eval surface, so the Conflict Surface section required for language internals does not apply. The compatibility analysis that replaces it lives in "Scope & Compatibility Audit" below.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Provider-side blocking is deterministic per prompt; classification adds no nondeterminism. |
| A2: Replayability | 0 | Traces unchanged in shape; new span attributes are additive. |
| A3: Effect Legibility | 0 | No new effects. |
| A4: Explicit Authority | 0 | No capability surface touched. |
| A5: Bounded Verification | +1 | Classification is testable with synthetic httptest/SSE fixtures — zero paid API calls needed. |
| A6: Safe Concurrency | 0 | No concurrency changes. |
| A7: Machines First | +1 | Replaces "empty text + generic FinishReason + nil error" (a string-matching trap) with a machine-decidable error code, exactly the failure mode A7 exists to prevent. |
| A8: Minimal Syntax | 0 | No language syntax. |
| A9: Cost Visibility | +1 | Blocked paid calls stop being retried into the same deterministic block, and their usage is recorded in span attributes instead of vanishing with a dropped response. |
| A10: Composability | +1 | One shared classifier used by streaming and non-streaming paths; OpenRouter and ollama /v1 inherit it by delegation, not by copy. |
| A11: Structured Failure | +1 | A named, non-retryable typed failure replaces a silent generic one — the core of the issue. |
| A12: System Boundary | 0 | No new boundary crossings. |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1: no implicit nondeterminism introduced
- [x] A3: no hidden side effects
- [x] A4: no ambient access granted
- [x] A7: not optimizing for human convenience over machine analysis

## Problem Statement

Issue [#2123](https://github.com/sunholo-data/ailang/issues/2123). When an OpenAI-compatible provider blocks a call in-band — HTTP 200, no error envelope, `finish_reason="content_filter"` and/or `message.refusal`/`delta.refusal` set (Case B) — both `Client.Step` and `Client.StreamStep` (direct `openai`, and via shared parsers `openrouter`, ollama `/v1`) return an **empty `Response.Text`, `FinishReason="error"`, and a nil error**. Verified on pinned baseline `8fc51e76`:

- `internal/ai/openai/streamstep.go:169-175` — `ChatStepStreamDelta` has **no `refusal` field**, so `delta.refusal` is silently dropped by the JSON decoder.
- `internal/ai/openai/step.go:298-304` — `ChatStepRespMessage` has **no `refusal` field** either.
- `internal/ai/openai/step.go:677-690` — `MapChatFinishReason("content_filter")` → generic `"error"`.
- `internal/ai/openai/step_test.go:408-429` — `TestStep_FinishReason_ContentFilter` asserts the current behavior: **nil error**, `FinishReason=="error"`.
- The reporter's synthetic fixture (httptest/SSE, no paid calls): `refusal="blocked fixture"` + `finish_reason="content_filter"` → empty `Response.Text`, `FinishReason="error"`, usage 2/0, nil error.

**Impact:** agent loops that drive `Step` cannot distinguish "model declined / provider blocked" from "transient empty response", so they retry into the same deterministic block and then bank a clean run summary — the exact failure class the Anthropic adapter already named when it added its refusal handling (`internal/ai/anthropic/client.go:335-346`: *"surface it distinctly so eval scoring can separate 'declined to answer' from 'API broke'"*). Downstream taxonomy already agrees this is its own category: `executor.FinishContentFilter = "content_filter"` (`internal/executor/executor.go:382`) and `piFinishReason` maps `refusal`/`safety`/`contentFilter`/`content_filter` → `FinishContentFilter` (`internal/executor/pi/events.go:239-240`).

### Baseline Distinction (do not conflate)

Two adjacent-but-distinct cases:

- **Case A — in-band error envelope in SSE.** Fixed on source `2063f4ceb` (`classifyStreamChunkError` / `chunk.Error` handling), **which is NOT an ancestor of the pinned review baseline `8fc51e76`**: grep of `internal/ai/` on this baseline returns no `classifyStreamChunkError`/`chunk.Error` hits, and `2063f4ceb` does not appear in this clone's history. This doc makes **no claim that the Case A fix is present in this baseline, no claim about which release (if any) carries it, and no release-version promise** — ancestry was not checked because the commit is not in this clone.
- **Case B — refusal / content_filter without an error envelope.** The bounded unmet request this doc designs. The two fixes must compose: see "Precedence Rule" under Solution Design.

## Goals

**Primary Goal:** Make refusal/content_filter a preserved-cause, explicitly non-retryable, typed failure in both streaming and non-streaming OpenAI-compatible Step, consistent by construction (one classifier, two call sites), with direct-Anthropic behavior as the audited parity control.

**Success Metrics:**
1. A synthetic refusal-only SSE/JSON fixture yields a typed, non-retryable error whose message carries the refusal text and `finish_reason`; `ai.ShouldRetry` returns false for it.
2. The same fixture through `openrouter.Step`/`StreamStep` produces the identical classification with no OpenRouter-side code (inheritance via shared parsers, verified).
3. Ordinary `stop`, `length`, `tool_calls`, usage accounting, and Case-A in-band error behavior are byte-for-byte unchanged (control tests).
4. Zero paid API calls: all tests are `httptest`/SSE synthetic fixtures.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Refusal/content_filter (Case B) becomes a **typed non-retryable error** from Step/StreamStep (previously: response + nil error). | Public Go API behavior change; observable through the AILANG `AIError` record and every retry loop. | **human** | design | med |
| D2: Return shape on the blocked path: **nil response + typed error** (Anthropic-Generate parity) vs novel "response + error" dual return. | Dual-return is a contract no existing internal/ai caller understands; nil-resp matches every existing error path. | **human** | design | med |
| D3: New error code `CodeContentFilter = "ContentFilter"` and its registration in **both** retry tables (`ai.IsRetryable`, cmd `isAICodeRetryable`). | The error-code vocabulary is a shared, observable namespace (mirrors std/ai `AIError` record); a code missing from the cmd table silently defaults retryable=true — a no-silent-fallbacks hazard. | **human** | design | low |
| D4: Anthropic Step/StreamStep refusal alignment (today they map refusal → generic `"error"` FinishReason with nil error — see audit row 7/8) — include in scope or record as follow-up? | Systemic-fix principle: retry loops are provider-agnostic; a provider-dependent classification difference recreates the bug on the Anthropic lane. | **human** | design | low |
| D5: Precedence rule vs the Case-A envelope fix when it lands: explicit error envelope wins; refusal classification is the fallback. | Prevents double-classification conflict at merge time without either fix knowing about the other. | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved (D1–D4 are the public-semantics choices the issue routes through design approval; this doc is that vehicle, not an implementation authorization):

- [ ] D1 — typed non-retryable failure for Case B (recommended: yes)
- [ ] D2 — nil response + typed error, no dual-return contract (recommended: yes)
- [ ] D3 — code name `ContentFilter` (recommended: yes; free per verification row 11)
- [ ] D4 — Anthropic Step/StreamStep alignment in-scope as Phase 3 (recommended: yes, it is ~20 lines)
- [ ] D5 accepted as the composition rule with the unmerged Case-A work

## Solution Design

### Overview

Add the missing wire fields (`refusal` on both the buffered and delta message shapes), and one shared classifier applied at both parse choke points, producing a new typed `AIError` (`CodeContentFilter`, `Retryable=false`) whose message preserves the provider's refusal text and finish reason. Because OpenRouter and ollama `/v1` both delegate to the OpenAI parsers (verified rows 9–10), the fix reaches every OpenAI-compatible lane without per-provider code.

### Components

1. **Wire fields** (`internal/ai/openai/step.go`, `streamstep.go`):
   - `ChatStepRespMessage.Refusal string \`json:"refusal,omitempty"\``
   - `ChatStepStreamDelta.Refusal string \`json:"refusal,omitempty"\``
   These mirror the OpenAI Chat Completions spec fields (`message.refusal`, `delta.refusal`) that exist on the wire today and are currently dropped.

2. **Shared classifier** (new, in `internal/ai/openai/step.go`, next to `MapChatFinishReason`):

   ```go
   // ClassifyRefusal returns a typed, non-retryable error when the response
   // was blocked in-band (Case B): refusal text and/or finish_reason
   // "content_filter" with no error envelope. nil when neither signal is
   // present — ordinary stop/length/tool_calls pass through untouched.
   func ClassifyRefusal(provider, finishReason, refusalText string) *ai.AIError
   ```

   Message format: `<provider>: model refused (finish_reason=content_filter, refusal="<text>")` — refusal fragment omitted when empty, finish reason included verbatim when the block arrived as `refusal` with a different finish reason (e.g. `stop`). Cause is preserved verbatim in the message (machine-first: no ellipsis, no truncation).

3. **Non-streaming integration** — `ParseChatStepResponse` (step.go:534): after decoding `choice`, before Hermes recovery: `if e := ClassifyRefusal("openai", choice.FinishReason, choice.Message.Refusal); e != nil { return nil, e }`. Classifying before recovery guarantees a refused (empty-text) response can never be mis-lifted into `tool_calls`.

4. **Streaming integration** — `ParseChatStepSSEStream` (streamstep.go:208+): accumulate `choice.Delta.Refusal` into a `refusalParts` slice alongside `contentParts` (refusal deltas fire **no** `onChunk` callback — they are not content). After the scan loop, alongside `out.FinishReason = MapChatFinishReason(finishReason)`: `if e := ClassifyRefusal("openai", finishReason, strings.Join(refusalParts, "")); e != nil { return nil, e }`. Split refusal text across N chunks reassembles exactly like content does.

5. **Error code + retry tables**:
   - `internal/ai/errors.go`: `CodeContentFilter = "ContentFilter"` constant; add to the `IsRetryable` false-list (so `ShouldRetry` never retries it — `classify` must also short-circuit on the refusal message signature before the generic `CodeInternal` catch-all, mirroring how `IsQuotaExhausted` is checked first; see audit row 13 for why the catch-all is not enough on the Anthropic path).
   - `cmd/ailang/configdriven_streaming.go:isAICodeRetryable`: add `"ContentFilter"` → false. Without this row the AILANG-side wrap silently defaults the new code to retryable=true (verified row 12) — exactly the no-silent-fallbacks trap.

6. **Telemetry parity** (Anthropic is the control): on the blocked path, record span attributes before returning — `error.message`, `error.category="refused"`, `ai.finish_reason`, `ai.tokens_in/out` — matching `internal/ai/anthropic/client.go:341-345`. This is the only place the blocked call's usage survives, since D2 drops the response.

7. **Precedence rule (D5)**: an explicit in-band error envelope (Case A, handled by the `2063f4ceb` work once it merges) always wins — it returns per-chunk and immediately, so the end-of-stream refusal classifier never runs. This doc's classifier only fires when the stream completed without an envelope error. Both fixes are therefore order-independent at merge time; neither needs to name the other in code.

8. **Responses API path** (`internal/ai/openai/responses.go:95-113`, audit row 15): a `content.Type == "refusal"` output item is currently skipped, and a refusal-only response falls through to the misleading generic `"no text output in response"`. Bounded fix: recognize the `refusal` item and return the same typed `CodeContentFilter` error. The `status:"incomplete"` + `incomplete_details.reason:"content_filter"` surface is **not** decoded today (`responsesOutputItem` has `Status` but no incomplete-details struct, audit row 16) and stays future work.

### Phase 3 (gated on D4): direct-Anthropic Step-level parity

The audited control itself is inconsistent today: Anthropic **Generate** returns a typed provider error on `stop_reason=="refusal"` (client.go:335-346), but Anthropic **Step** (step.go:98) and **StreamStep** (streamstep.go:391) map `refusal` through `mapStopReason` → generic `"error"` FinishReason with nil error — the same shape as the OpenAI bug. Aligning them (~20 lines: classify after parse / on `message_delta` stop_reason) makes the parity real at the layer where the retry loops live, and migrates Generate's `ProviderError` to the same code — its non-retryability today rests on the `known=false` accident in `classify` (audit row 13), not on a typed code.

### Implementation Plan

**Phase 1: Core classification** (~4h)
- [ ] Wire fields on `ChatStepRespMessage` + `ChatStepStreamDelta`
- [ ] `ClassifyRefusal` helper + doc
- [ ] Integrate into `ParseChatStepResponse` and `ParseChatStepSSEStream`
- [ ] `CodeContentFilter` in `errors.go` + `IsRetryable`/`classify` + cmd `isAICodeRetryable`
- [ ] Span attributes on the blocked path (both streaming and non-streaming)

**Phase 2: Test matrix (all synthetic)** (~4h)
- [ ] Non-streaming: refusal-only, content_filter-only, refusal+`finish_reason:"stop"`, positive `stop`/`length`/`tool_calls` controls
- [ ] Streaming: same matrix, plus **split SSE chunks** (refusal text across 2+ `data:` chunks; `content_filter` arriving as finish_reason on a later chunk than the refusal text; usage block after the blocked choice)
- [ ] In-band error-envelope composition fixture (Case-A shape): envelope error wins, refusal classifier does not double-fire
- [ ] Inheritance proofs: one refusal fixture each through `openrouter.Step`, `openrouter.StreamStep` (assert identical classification, no OR-side code)
- [ ] Retry-table tests: `IsRetryable("ContentFilter")==false`, `ShouldRetry(refusalErr)==false`, cmd `isAICodeRetryable("ContentFilter")==false`
- [ ] Update `TestStep_FinishReason_ContentFilter` (step_test.go:408) — the **one** intentional existing-test change (verified row 4): it now expects the typed error instead of nil error

**Phase 3: Anthropic alignment (gated on D4)** (~3h)
- [ ] `anthropic.Step` + `StreamStep`: refusal → same typed error; Generate migrated off `ProviderError` onto `CodeContentFilter` (span attrs preserved)
- [ ] Anthropic refusal fixtures (buffered + streamed `message_delta`)

### Files to Modify/Create

**Modified files:**
- `internal/ai/errors.go` — new code constant + classify short-circuit, ~15 LOC
- `internal/ai/openai/step.go` — `Refusal` field, `ClassifyRefusal`, parse integration, ~35 LOC
- `internal/ai/openai/streamstep.go` — `Refusal` field, accumulation + integration, ~20 LOC
- `internal/ai/openai/responses.go` — refusal content item, ~10 LOC
- `cmd/ailang/configdriven_streaming.go` — one switch case, ~2 LOC
- `internal/ai/openai/step_test.go`, `streamstep_test.go`, `errors_test.go` — matrix + one intentional update, ~200 LOC
- (Phase 3) `internal/ai/anthropic/step.go`, `streamstep.go`, `client.go` — ~20 LOC + tests

**New files:** none.

## Examples

### The reporter's fixture, before → after

Non-streaming response body (httptest canned, no paid API):

```json
{"id":"chatcmpl-x","object":"chat.completion","model":"gpt-5",
 "choices":[{"index":0,"message":{"role":"assistant","content":"","refusal":"blocked fixture"},"finish_reason":"content_filter"}],
 "usage":{"prompt_tokens":2,"completion_tokens":0,"total_tokens":2}}
```

**Before** (`8fc51e76`): `resp.Text=""`, `resp.FinishReason=="error"`, `err==nil` → downstream retries, run banks clean.

**After**: `resp==nil`, `err = &ai.AIError{Code:"ContentFilter", Message:"openai: model refused (finish_reason=content_filter, refusal=\"blocked fixture\")", Retryable:false}`; span carries `error.category="refused"`, `ai.finish_reason="content_filter"`, `ai.tokens_in=2`, `ai.tokens_out=0`. `ai.ShouldRetry(err)` → false; the AILANG-side `_ai_call_result` surfaces `Err(AIError{code:"ContentFilter", retryable:false})` — the existing `{code, message, retryable}` record shape, unchanged.

## Scope & Compatibility Audit

The audit the issue requested, with each claim's evidence in the Verification Log:

| Path | Current behavior (baseline `8fc51e76`) | Disposition |
|---|---|---|
| `openai.Step` (Chat Completions) | refusal dropped; `content_filter` → `"error"`, nil error | **In scope** — fix at parse choke point |
| `openai.StreamStep` (SSE) | same, refusal text silently undecoded | **In scope** — same classifier |
| `openrouter.Step` / `StreamStep` | delegates to `openai.ParseChatStepResponse` / `ParseChatStepSSEStream` (openrouter/step.go:145, streamstep.go:164) | **Inherits** — no OR-side code; proven by delegation test. OR's own non-2xx envelope translation is Case-A territory, out of scope |
| ollama `/v1` streaming | wraps an `openai.Client` (ollama/streamstep.go:261) | **Inherits**. Ollama *native* API has no finish-reason vocabulary (step.go:479-489 synthesizes `stop`/`tool_calls`) — no change possible or needed |
| `openai` Responses API (`responses.go` Generate) | `refusal` content items skipped → misleading `"no text output in response"` | **In scope, bounded** — refusal item → typed error; `incomplete_details` not decoded today, stays future work |
| `chatgpt` adapter (chatgpt/client.go, `/responses`) | shares the Responses shape; refusal items ignored there too | rides the `responses.go` wire-type change if it reuses `responsesContent`; executor verifies at implementation time (deferred decision 2) |
| direct Anthropic **Generate** | typed `ProviderError` on `stop_reason=="refusal"` (client.go:335-346) | **Audited control** — the parity reference; its own non-retryability is accidental today (row 13), Phase 3 migrates it to the shared code |
| direct Anthropic **Step/StreamStep** | refusal → generic `"error"` FinishReason, nil error (step.go:98, streamstep.go:391) | **Audit finding** — the control is inconsistent at the Step layer; Phase 3 aligns (D4) |
| Gemini Step vs Generate | Step maps `SAFETY`→`"error"` (step.go:388-391) but Generate maps `SAFETY`→`"content_filter"` (generate.go:172) — internally inconsistent, and `FinishReason:"content_filter"` already exists de-facto in the vocabulary despite provider.go's 4-value doc comment | **Audit finding only** — out of bounded scope, recorded as future work; this doc deliberately does NOT add `"content_filter"` to `ai.Response.FinishReason`'s documented values (under D2 the blocked path returns no response, so the mapping change would be inert) |
| executor / eval harness | `executor.FinishContentFilter` exists and pi events already classify refusal-family stop reasons into it (events.go:239-240) | **Confirms downstream readiness** — the taxonomy already has the bucket this error feeds |
| in-repo callers branching on `FinishReason=="error"` | none found outside the executor's own vocabulary (row 17) | no in-repo breakage; the only in-repo assertion of old behavior is step_test.go:408 (row 4) |

**Compatibility statement:** the Go API change is additive (two new struct fields, one new constant, one new exported helper). The *behavior* change — Case B now returns an error where it returned nil — is the requested semantic change and is routed through this design doc for approval per the work-routing gate. The AILANG-level `AIError` record shape `{code, message, retryable}` is unchanged; the new code string is additive and explicitly registered in the cmd retry table so no silent default applies.

## Success Criteria

- [ ] Refusal-only fixture (buffered + streamed): typed `CodeContentFilter` error, refusal text verbatim in message, `ShouldRetry` false (acceptance: Phase 2 tests)
- [ ] content_filter-only fixture (no refusal text): same typed error, finish reason in message
- [ ] Streaming refusal split across multiple SSE chunks reassembles correctly
- [ ] Positive controls unchanged: ordinary `stop`, `length`, `tool_calls` (with usage) still return response + nil error, byte-equal FinishReason mapping
- [ ] Case-A-style error-envelope fixture wins over refusal classification (composition test)
- [ ] OpenRouter delegation produces identical classification with no OR-side code
- [ ] (Phase 3, if D4 approved) Anthropic Step/StreamStep refusal → same typed error
- [ ] All tests passing; no paid API call anywhere in the suite
- [ ] `provider.go` doc comment on `FinishReason` cross-references the typed refusal error (documentation updated)

## Testing Strategy

**Unit tests (all synthetic httptest/SSE):**
- The Phase 2 matrix above; split-chunk refusal fixtures are the regression guard for the accumulation logic.

**Integration tests:**
- OpenRouter + ollama `/v1` inheritance fixtures through their public `Step`/`StreamStep` entry points.

**Manual testing:** none required — no live paid reproduction (explicit non-goal). The reporter's archived fixture from the issue thread is the reference shape for the canned bodies.

## Deferred Decisions

1. Whether a `StreamRefusalDelta` `StreamChunk` variant should fire live during streaming (for UIs that render refusal as it arrives) — currently refusal fires no callback and rides only in the terminal error. Agent may propose; the callback interface is public, so adding a variant needs its own small approval. Default: do not add.
2. Whether the `chatgpt` adapter reuses the updated `responsesContent` handling or needs its own decode — executor decides after reading chatgpt/client.go's parse (agent latitude).
3. Exact span attribute set on the blocked path beyond the Anthropic-parity minimum (agent latitude).

## Non-Goals

- **No provider prompt manipulation** — nothing tries to work around the filter; classification only.
- **No live paid reproduction** — synthetic fixtures only.
- **No Case-A (in-band error envelope) work** — that is the unmerged `2063f4ceb` change; this doc neither duplicates, claims, nor re-implements it; only the precedence rule (D5) is specified.
- **No OpenRouter non-2xx error translation changes.**
- **No Gemini alignment** — audit finding recorded as future work; the bounded request is OpenAI-compatible Step.
- **No release-version promise** — ancestry of `2063f4ceb` was not checked (not in this clone); targeting the v0_57_2 planning folder says nothing about what any released version contains.
- **No implementation** — this request authorizes a reviewable design only; the issue stays open.

## Timeline

**Week 1** (~11h): Phase 1 core (4h), Phase 2 test matrix (4h), Phase 3 Anthropic alignment pending D4 (3h).

**Total: ~11 hours across 1 week** (estimates doubled from naive guesses per skill guidance).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| A provider emits partial text **and** `content_filter` mid-generation (Azure-style truncation); nil-response drops the partial text | Med | Deterministic filters make retry pointless, so typed error is still correct; partial-text preservation via a dual-return contract is future work if a consumer reports a concrete need (per the errors.go precedent note on additive record extension) |
| New code string leaks to AILANG programs that switch on `code` exhaustively | Low | Additive by construction; `std/ai`'s `AIError` code is a `string` field, and unknown-code defaults are already defined per-surface (cmd table updated in Phase 1) |
| Merge conflict with the Case-A envelope fix when it lands | Low | D5 precedence rule makes the two order-independent; refusal classifier only runs on streams that completed without an envelope error |
| Anthropic Phase 3 changes observable behavior on a lane that currently "works" | Med | Gated on human decision D4; its own doc comment (client.go:335) already declares refusal "a MODEL BEHAVIOR, not an infrastructure failure" — the alignment implements the adapter's stated intent |

## Related Documents

- [m-ai-step-streaming-thinking](../v0_18_8/m-ai-step-streaming-thinking.md) — established the SSE chunk/delta parse layer this extends (implemented v0.18.8)
- [m-ai-openrouter-reasoning-field](../v0_18_9/m-ai-openrouter-reasoning-field.md) — precedent for adding a delta field with dual direct/OpenRouter semantics (implemented v0.18.9)
- Issue [#2123](https://github.com/sunholo-data/ailang/issues/2123) — reporter's fixtures and downstream retry/clean-summary evidence

No planned or implemented design doc covers refusal/content_filter classification (duplicate gate: script search returned none; manual grep of `design_docs/` for refusal/content_filter found only unrelated hits — see Verification Log row 19).

## Verification Log

| # | Claim | Method | Result |
|---|---|---|---|
| 1 | `ChatStepStreamDelta` has no `refusal` field | read `internal/ai/openai/streamstep.go:169-175` | Confirmed — fields are Role/Content/ReasoningContent/Reasoning/ToolCalls only |
| 2 | `ChatStepRespMessage` has no `refusal` field | read `internal/ai/openai/step.go:298-304` | Confirmed |
| 3 | `MapChatFinishReason("content_filter")` → `"error"` | read `step.go:677-690` | Confirmed |
| 4 | Existing test expects nil error for content_filter | read `step_test.go:408-429` body: `if err != nil { t.Fatalf(...) }` then asserts `FinishReason=="error"` | Confirmed — the one intentional test change |
| 5 | Case-A fix (`classifyStreamChunkError`/`chunk.Error`) absent from baseline | `grep -rn "classifyStreamChunkError\|chunk.Error" internal/ai/` → no hits; `git log 2063f4ceb` → commit not in clone | Confirmed absent from `8fc51e76`; no ancestry claim made |
| 6 | Anthropic Generate surfaces refusal as provider error | read `anthropic/client.go:335-346` (enclosing func `Generate`, client.go:190) | Confirmed |
| 7 | Anthropic Step maps refusal → `"error"`, nil error | read `anthropic/step.go:98` + `mapStopReason` (step.go:420-436: only end_turn/stop_sequence/tool_use/max_tokens/"" named) | Confirmed — audit finding |
| 8 | Anthropic StreamStep same | read `anthropic/streamstep.go:391` (`out.FinishReason = mapStopReason(ev.Delta.StopReason)`) | Confirmed |
| 9 | OpenRouter Step/StreamStep reuse the openai parsers | read `openrouter/step.go:145`, `openrouter/streamstep.go:164` | Confirmed — inheritance, no OR-side code |
| 10 | Ollama `/v1` streaming wraps an openai client; native path synthesizes finish | read `ollama/streamstep.go:261`, `ollama/step.go:479-489` | Confirmed |
| 11 | Error code `ContentFilter` unallocated in the `internal/ai` vocabulary | `grep -rn "ContentFilter" internal/ai/` → empty (the `FinishContentFilter` hits are `internal/executor`, a different namespace) | Confirmed free |
| 12 | cmd `isAICodeRetryable` defaults unknown codes to retryable=true | read `cmd/ailang/configdriven_streaming.go:489-502` | Confirmed — table update is mandatory, not optional |
| 13 | Anthropic's refusal `ProviderError` (status 0) is non-retryable only accidentally | read `errors.go:265-309`: status-0 ProviderError skips the HTTP branch, matches no substring, lands in `CodeInternal` with `known=false`; `ShouldRetry` requires `known` | Confirmed — Phase 3 migration justified |
| 14 | Gemini Generate vs Step map SAFETY differently | read `gemini/generate.go:165-176` (`"content_filter"`) vs `gemini/step.go:377-395` (`"error"`) | Confirmed — audit finding, future work |
| 15 | Responses API ignores `refusal` content items | read `openai/responses.go:95-121` — only `output_text` collected; empty text → `"no text output in response"` ProviderError | Confirmed |
| 16 | `responsesOutputItem` does not decode `incomplete_details` | read `openai/types.go:135-147` — `Status` only | Confirmed — stays future work |
| 17 | No in-repo non-test caller branches on `ai.Response.FinishReason=="error"` | grep `FinishReason ==` across `internal/ cmd/` — hits are executor/eval-harness paths using the executor's own `Finish*` vocabulary, or provider-internal | Confirmed — no in-repo breakage |
| 18 | Downstream taxonomy already has a content-filter bucket | read `executor/executor.go:381-382`, `executor/pi/events.go:239-240` | Confirmed |
| 19 | No existing/planned design doc covers this topic | create-script SimHash+neural search (none found) + manual grep of `design_docs/` for refusal/content_filter | Confirmed — duplicate gate passed |
| 20 | `std/ai` AIError record shape is `{code, message, retryable}` and additive codes are safe | read `errors.go:50-76` doc comment | Confirmed — no shape change proposed |

## References

- [Design Axioms](/docs/references/axioms)
- OpenAI Chat Completions API — `message.refusal` / `delta.refusal` / `finish_reason: content_filter` (as exercised by the issue's archived fixture)
- Issue [#2123](https://github.com/sunholo-data/ailang/issues/2123)

## Future Work

- Azure-style `content_filter` with partial text: a dual-return (response + typed error) contract to preserve the partial output — only if a downstream consumer reports a concrete need (per the errors.go additive-extension precedent).
- Gemini Step alignment on the same typed error (audit row 14).
- Responses API `status:"incomplete"` + `incomplete_details.reason` decode.
- `StreamRefusalDelta` callback variant for live refusal rendering (Deferred Decision 1).
- Reconciling `provider.go`'s `FinishReason` doc comment with Gemini Generate's de-facto `"content_filter"` value — separate from this doc, which adds no new FinishReason value.

---

**Document created**: 2026-11-04
**Last updated**: 2026-11-04
