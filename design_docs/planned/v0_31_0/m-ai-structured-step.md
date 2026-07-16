# M-AI-STRUCTURED-STEP — Structured output as an orthogonal option on `step`

**Status**: Planned
**Target**: v0.31.0
**Priority**: P1 (Medium) — unblocks vision+JSON grading (stx-bench) and aligns std/ai with the industry request shape
**Estimated**: 4–6 days
**Dependencies**: [M-STD-AI-VISION-INPUT](../v0_30_0/m-std-ai-vision-input.md) (v0.30.0, vision on `step`) — done. This doc composes with it.

> **Provenance**: fell out of the M-STD-AI-VISION-INPUT sprint. `callJsonResultVision`
> was deferred once we found the real problem is structural, not vision-specific:
> AILANG split structured-output and multi-turn/vision into **separate builtins**
> (`callJson` vs `step`) that cannot compose. Every mainstream provider keeps them as
> **one composable request**. This doc closes that gap.

## Verification Log

Every provider claim below was read from the code, not assumed.

| Claim | Method | Result |
|-------|--------|--------|
| `callJson` (schema) uses `Generate` with `UserPrompt` string — no `Messages`, no images | Read `internal/ai/handler.go` `CallJson` | **Confirmed** — builds `Request{UserPrompt, ResponseFormat:"json", ResponseSchema}` |
| `step` builds `Request{Messages, Tools}` with **no** schema | Read `internal/ai/handler.go` `Step`/`StepWithCache` | **Confirmed** — no `ResponseSchema`/`ResponseFormat` set |
| OpenAI does schema via `response_format:{type:"json_schema"}` on **Generate/chat only** | Read `internal/ai/openai/chat.go:60-72`, `responses.go:56` | **Confirmed** — `openai/step.go` sets neither (grep = 0) |
| Gemini does schema via `responseSchema`+`responseMimeType` on **Generate only** | Read `internal/ai/gemini/generate.go:48-51`, `types.go:86-87` | **Confirmed** — `gemini/step.go` sets neither (grep = 0) |
| Ollama sets `Format` on **both** Generate and Step | Read `internal/ai/ollama/client.go:157`, `step.go:381-383` | **Confirmed** — Ollama Step ALREADY honors schema |
| OpenRouter Step delegates to `openai.BuildChatStepRequest` | Read `internal/ai/openrouter/step.go:51`, `streamstep.go:50` (from vision sprint) | **Confirmed** — inherits whatever the OpenAI Step builder does |
| **Anthropic has NO `response_format`** — structured output = a forced `respond` tool | Read `internal/ai/anthropic/client.go` (`toolDef{Name:"respond", InputSchema}` + `ToolChoice`) | **Confirmed** — this is the load-bearing nuance |
| `StepResult.message.content` is the text field; `ai.Response.Text` feeds it | Read `std/ai.ail:154-162`, `internal/ai/provider.go` `Response.Text` | **Confirmed** |

**Conclusion**: schema is honored on the single-shot `Generate` path for 4/5 providers and on
Step only for Ollama. Threading it through Step is mostly mechanical for OpenAI/Gemini/OpenRouter,
but **Anthropic is structurally different** (tool_use, not a response-format field), and its
structured-output mechanism **collides with `step`'s existing user-tool support**. That collision
is the core design problem, not the plumbing.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | AI calls already non-deterministic; schema does not add nondeterminism |
| A2: Replayability | +1 | Schema is part of the request → captured in traces, replayable |
| A3: Effect Legibility | 0 | Stays inside `{AI}` |
| A4: Explicit Authority | 0 | No new authority |
| A5: Bounded Verification | +1 | Native schema enforcement yields provider-guaranteed-valid JSON — stronger than prompt-injection, less runtime parse-failure handling |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +1 | Guaranteed-schema JSON is the machine-legible grading path; composes with tools + vision |
| A8: Minimal Syntax | +1 | Reuses the existing `Request.ResponseSchema` field; a single new builtin/option, not per-combination builtins |
| A9: Cost Visibility | 0 | Tokens flow through existing Usage |
| A10: Composability | +1 | **The whole point** — schema composes with messages, tools, images, cache in one call |
| A11: Structured Failure | +1 | Malformed schema / provider rejection → typed AIError |
| A12: System Boundary | +1 | Per-provider structured-output mapping is the explicit boundary |

**Net Score: +7** → **Proceed.** No hard-axiom violations.

## Problem Statement

Structured output and multi-turn/vision/tools do not compose in std/ai:

- `callJson(input, schema)` — schema-enforced JSON, but takes a **string** input: no
  conversation, no tools, no images.
- `step(model, messages, tools)` — carries conversation + tools + (as of v0.30.0) images, but
  has **no schema** knob, so it cannot request guaranteed-valid JSON.

**Impact:**
- **stx-bench (primary grading path):** wants vision + structured JSON in one call. Today that's
  impossible; the interim is `step` with a "respond only in JSON" **prompt instruction**, which is
  not provider-guaranteed and flakes a small % of the time — unacceptable noise in a grading
  leaderboard.
- **Any agent** wanting a tool-loop that also returns a final structured result must drop to
  prompt-level JSON.
- The API is drifting toward one bespoke builtin per capability-combination
  (`callJson`, `callJsonResultVision`, `callJsonResultVisionWithTools`, …). Every mainstream
  provider instead treats structured-output as an **orthogonal request option** that composes with
  everything else.

## Goals

**Primary Goal:** Let a `step`-family call request provider-native structured output (a JSON
Schema), so vision + tools + schema compose in one request, with guaranteed-valid JSON where the
provider supports it.

**Success Metrics:**
- A single `step`-family call with a schema returns schema-conforming JSON in
  `StepResult.message.content` for all 5 providers.
- Vision + schema in ONE call returns graded JSON (stx-bench's plot-reading items become gradable
  with guaranteed-valid output, no prompt-injection).
- Native enforcement where the provider offers it (OpenAI `json_schema` strict, Gemini
  `responseSchema`, Anthropic forced-tool, Ollama `format`); a clear typed error where a schema is
  requested but unsupported.
- `callJsonResultVision` is NOT added — it is subsumed by "step with a schema."
- Zero regression: a `step` call with no schema is byte-for-byte identical to today.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **API surface: new `stepStructured` builtin vs a `StepOptions` record vs schema param** | Public std/ai surface for every future request knob | human | design | high |
| **Anthropic mechanism when user tools AND a schema are both present** | tool_use is Anthropic's ONLY structured path; it collides with user tools | human | design | high |
| **Response normalization: where the JSON lands** (message.content vs a tool_call) | Callers must find the JSON in one predictable place across providers | human | design | med |
| **Unsupported-combination behavior** (schema requested, provider/model can't) | Silent prompt-fallback vs typed error | agent | compile | med |

### Design Freeze

- [x] **API surface — RESOLVED 2026-07-16 (user sign-off): `StepOptions` record.** A single new
      `stepWithOptions(model, messages, tools, options)` where
      `options = { response_format: string, response_schema: string, cache_breakpoints: [CacheBreakpoint] }`.
      This folds the existing `stepWithCache` breakpoints AND the new schema into ONE extensible
      options record, so future knobs don't spawn new builtins. `step`/`stepWithCache`/`stepWithStream`
      stay as thin wrappers over it. (Rejected: a dedicated `stepStructured` builtin — smaller now
      but re-introduces per-combination builtin drift.)
- [x] **Anthropic + user-tools + schema collision — RESOLVED 2026-07-16 (user sign-off): typed
      error.** If a schema is set AND user tools are non-empty on Anthropic, return a typed
      `AIError` (schema+tools not simultaneously supported on Anthropic) rather than silently
      dropping one. Schema-only (no user tools) uses the forced `respond` tool and extracts its
      input into `message.content`. OpenAI/Gemini/Ollama compose schema+tools natively. Revisit once
      Anthropic ships a native response-format.
- [x] **Response normalization — RESOLVED.** The structured JSON MUST land in
      `StepResult.message.content` as a string for ALL providers (including Anthropic's tool_use
      extraction), so callers decode one place. `finish_reason` reflects structured completion.

## Solution Design

### Overview

Thread `ResponseFormat`/`ResponseSchema` (already on `ai.Request`) through the **Step** path of
every provider, mirroring what each already does on its Generate path, and normalize the result so
the JSON always appears in `message.content`. Expose it via one extensible `StepOptions` record.

### Architecture

```
AILANG: stepWithOptions(model, msgs, tools, {response_format, response_schema, cache_breakpoints})
   │
   ▼ effect bridge (internal/effects/ai_step.go) — new op; sets Request.ResponseFormat/ResponseSchema
   ▼ handler.Step* (internal/ai/handler.go) — thread schema onto the outgoing Request
   ▼ per-provider Step builder:
        OpenAI     → response_format:{type:"json_schema", strict} (reuse chat.go logic in step.go)
        Gemini     → GenerationConfig.responseSchema + responseMimeType (reuse generate.go logic)
        Ollama     → Format field (ALREADY wired on Step — no change)
        OpenRouter → inherits via shared openai Step builder
        Anthropic  → forced `respond` tool (input_schema=schema, tool_choice) IF no user tools;
                     extract tool_use.input JSON → Response.Text; else typed AIError
   ▼ StepResult.message.content = the JSON string (Anthropic: extracted from the tool_use block)
```

### Conflict Surface

Touches `internal/ai/*` (all provider Step builders), `internal/ai/handler.go`,
`internal/effects/ai_step.go`, `internal/builtins/ai_step.go`, `std/ai.ail`.

1. **Positions extended:** the Step request path (schema now flows through it) and the `StepResult`
   normalization.
2. **What else lives there:** the existing tool-loop (`step` + user `tools` + tool_use responses),
   cache breakpoints (`stepWithCache`), and streaming (`stepWithStream`). A schema must compose with
   cache + streaming and NOT disturb the no-schema path.
3. **Disambiguation / the load-bearing collision:** on **Anthropic**, structured output IS a
   tool_use, so "schema + user tools" is a genuine two-tools-forced conflict. v1 rule (above):
   schema+user-tools on Anthropic → typed error; schema-only → forced `respond` tool, extracted to
   content. OpenAI/Gemini/Ollama have a separate response-format field, so schema + tools compose
   natively there.
4. **Programs that MUST still work (verify each at sprint start):** every current `step`/
   `stepWithCache`/`stepWithStream` program with NO schema (empty `response_schema` ⇒ identical wire
   request — golden test per provider); the tool-loop examples (`examples/runnable/ai_tool_loop.ail`);
   the cache + streaming examples. Enumerate with `grep -rln "step(" examples/`.
5. **Deliberate changes:** Step requests may now carry a schema; `StepResult.message.content` may be
   a provider-enforced JSON string.

### Implementation Plan

**Phase 1: Surface + effect bridge + handler threading (~1.5 days)**
- [ ] Resolve API surface (Design Freeze) — `StepOptions` record + `stepWithOptions` builtin.
- [ ] `std/ai.ail`: `StepOptions` type; `stepWithOptions`; refactor `step`/`stepWithCache` as wrappers.
- [ ] Effect op sets `Request.ResponseFormat`/`ResponseSchema`; `handler.go` threads them onto the Step Request.

**Phase 2: Per-provider Step schema (~2 days)**
- [ ] OpenAI Step: set `response_format` (reuse `chat.go`'s `ensureStrictSchemaCompliance`).
- [ ] Gemini Step: set `GenerationConfig.responseSchema`+`responseMimeType`.
- [ ] Ollama Step: no-op (already wired) — add regression test only.
- [ ] OpenRouter: inherits via shared builder — add regression test.
- [ ] Anthropic Step: forced `respond` tool when schema set & no user tools; typed error on schema+tools; extract tool_use.input → `Response.Text`.
- [ ] configdriven: typed `AIError` when a schema is requested (unsupported in v1 schema).

**Phase 3: Response normalization + example + docs (~1.5 days)**
- [ ] Normalize: JSON in `StepResult.message.content` for all providers (esp. Anthropic extraction).
- [ ] `examples/runnable/ai_structured_step.ail` (and fold vision+schema into the vision example).
- [ ] Update std/ai docs + website + CHANGELOG; notify stx-bench with the composable pattern.

### Files to Modify

- `std/ai.ail` — `StepOptions` type, `stepWithOptions`, wrapper refactor (~60 LOC)
- `internal/builtins/ai_step.go` — register `stepWithOptions` + `stepOptions` record type (~50)
- `internal/effects/ai_step.go` — new effect op; set schema on Request (~40)
- `internal/ai/handler.go` — thread schema through Step/StepWithCache/StepWithStream (~30)
- `internal/ai/openai/step.go` — response_format on Step (~30)
- `internal/ai/gemini/step.go` — responseSchema on Step (~30)
- `internal/ai/anthropic/step.go` — forced `respond` tool + tool_use extraction + collision guard (~70)
- `internal/ai/ollama/step.go`, `internal/ai/openrouter/*` — regression tests only
- `internal/ai/configdriven/step.go` — typed error on schema (~10)

## Examples

### Vision + structured output in ONE call (the stx-bench grading path)

```ailang
let img = { source: base64Png, mime: "image/png" };
let schema = "{\"type\":\"object\",\"properties\":{\"answer\":{\"type\":\"string\"},\"unit\":{\"type\":\"string\"}},\"required\":[\"answer\",\"unit\"]}";
let msgs = [{ role:"user", content:"Read the velocity-time graph.", tool_calls:[], tool_call_id:"", images:[img] }];
match stepWithOptions("", msgs, [], { response_format:"json", response_schema:schema, cache_breakpoints:[] }) {
  Ok(r)  => println(r.message.content),   -- guaranteed schema-valid JSON
  Err(e) => println("failed: ${e.code}")
}
```

## Success Criteria

- [ ] `stepWithOptions` requests native structured output on all 5 providers; JSON lands in `message.content`.
- [ ] Vision + schema compose in one call (example runs green against a live vision model).
- [ ] Anthropic: schema-only works via forced tool; schema+user-tools returns a typed error (documented).
- [ ] No-schema Step path is byte-for-byte identical to today (golden per provider).
- [ ] `callJsonResultVision` NOT added — subsumed.
- [ ] Docs + CHANGELOG updated; stx-bench notified. `make test` green.

## Deferred Decisions

- **Anthropic schema+tools once Anthropic ships a native response-format** — agent may revisit.
- **Streaming + schema interaction** — agent may defer schema on `stepWithStream` to a follow-up if
  provider streaming + structured output conflict.
- **Whether `callJson`/`callJsonResult` (string-input) are eventually deprecated** in favor of
  `stepWithOptions` — human decides later; not in scope.

## Non-Goals

- Deprecating or removing `callJson`/`callJsonResult` (they stay; this is additive).
- A new AILANG type for schemas — schema stays a JSON-Schema **string**, as today.
- Tool-calling improvements beyond the schema+tools collision rule.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Anthropic forced-tool extraction breaks the existing tool-loop | High | Collision guard (schema+user-tools → typed error); golden tests on the no-schema tool-loop |
| Schema on Step regresses the no-schema wire request | High | Strict `response_schema==""` guard + per-provider golden tests |
| `StepOptions` record is a breaking change to `stepWithCache` callers | Med | Keep `stepWithCache` as a wrapper with its current signature; `stepWithOptions` is additive |
| Provider strict-schema quirks (OpenAI strict mode rejects some schemas) | Med | Reuse existing `ensureStrictSchemaCompliance`; typed error on rejection |

## Related Documents

- [M-STD-AI-VISION-INPUT](../v0_30_0/m-std-ai-vision-input.md) — vision on `step` (v0.30.0). This
  doc is the composability follow-up that makes vision+JSON one call.
- M-AI-TOOL-LOOP (v0.17.0) — introduced `step` + tools + `StepResult`; this extends that request
  with a schema knob.
- M-AI-PROMPT-CACHING (v0.18.4) — `stepWithCache`; its breakpoints fold into the new `StepOptions`.

## References

- [Design Axioms](/docs/references/axioms)
- Industry shape (verified against provider docs at design time): vision-input is a message-content
  dimension; structured-output is a request dimension; both compose in one call. Anthropic uses
  forced tool_use for structure (no response_format); OpenAI uses `response_format:json_schema`
  (strict/guaranteed); Gemini uses `responseSchema`+`responseMimeType`.

---

**Document created**: 2026-07-16
**Last updated**: 2026-07-16
