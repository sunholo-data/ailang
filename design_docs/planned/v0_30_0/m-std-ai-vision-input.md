# M-STD-AI-VISION-INPUT — Image (vision) input on std/ai Message

**Status**: Planned
**Target**: v0.30.0
**Priority**: P1 (Medium) — unblocks a whole class of benchmark items; not blocking any in-flight sprint
**Estimated**: 3–4 days
**Dependencies**: None (extends existing std/ai effect surface)

> **Provenance**: Feature request from `stx-bench` / `cph-uni-stx-bench`
> (agent message `msg_20260716_070552_f9104997`, 2026-07-16). Verified live against
> the v0.29.2 codebase before drafting — see Verification Log.

## Verification Log

Every language/behaviour claim below was checked against the code, not assumed.

| Claim | Method | Result |
|-------|--------|--------|
| `Message` record is text-only `{role, content: string, tool_calls, tool_call_id}` | Read [`internal/builtins/ai_step.go:38-45`](../../../internal/builtins/ai_step.go#L38-L45) | **Confirmed** — `messageRecordType` has `content: T.String()`, no image field |
| `callImage`/`callImageBase64` GENERATE images (output), not vision input | Read [`internal/builtins/ai.go:195-281`](../../../internal/builtins/ai.go#L195-L281) | **Confirmed** — both write/return a *generated* image; neither feeds an image into a model |
| `step`/`stepWithCache`/`stepWithStream` take `list[Message]` with string content | Read [`internal/builtins/ai_step.go:259-500`](../../../internal/builtins/ai_step.go#L259-L500) | **Confirmed** — all consume the text-only `messageRecordType` |
| Provider abstraction `ai.Message` is text-only | Read [`internal/ai/provider.go:200-205`](../../../internal/ai/provider.go#L200-L205) | **Confirmed** — `struct { Role, Content string; ToolCalls; ToolCallID }` |
| Effect bridge reads only `content` string off the record | Read [`internal/effects/ai_step.go:468`](../../../internal/effects/ai_step.go#L468) | **Confirmed** — `Content: getStringField(rec, "content")` |
| 5 providers present (anthropic, openai, gemini, openrouter, ollama) | `ls internal/ai/` | **Confirmed** |
| No existing vision-input design doc | `ailang docs search --neural` (top match 0.31, unrelated) + grep | **Confirmed** — not a duplicate |
| New error code `model_no_vision` unallocated | `grep -rn "model_no_vision\|CodeModelNoVision" internal/ cmd/` | **TODO at sprint start** — must grep-verify free before wiring (see Deferred) |

**Conclusion**: the request is accurate. AILANG has no image-INPUT path today; the only image
builtins are output/generation. The gap is real and blocks graded benchmark items.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | AI calls are already non-deterministic; image input adds no new non-determinism beyond the existing `AI` effect |
| A2: Replayability | +1 | Image bytes/URIs are captured in the Message record → replayable in traces like any other AI-call input |
| A3: Effect Legibility | +1 | Vision stays inside the existing `! {AI}` effect; local-path image sources also surface `! {FS}` for reads — nothing hidden |
| A4: Explicit Authority | +1 | No new ambient authority; base64/data-URI needs only `AI`, local-path sources need explicit `FS` cap |
| A5: Bounded Verification | 0 | Record-shape extension is locally type-checkable; no new inference |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +1 | Structured `ImagePart` record (source+mime) is machine-legible; typed `model_no_vision` error beats silent drop |
| A8: Minimal Syntax | +1 | Zero new syntax — a new record field + new builtin, mirroring how `CacheBreakpoint` was added |
| A9: Cost Visibility | 0 | Image tokens flow through existing Usage accounting |
| A10: Composability | +1 | Composes with `step`/`callJsonResult`/tool-loop; empty `images` list = today's exact behaviour |
| A11: Structured Failure | +1 | Non-vision models return typed `AIError{code:"model_no_vision"}`, not a silent drop or panic |
| A12: System Boundary | +1 | Provider image-block mapping is the boundary crossing; made explicit per-provider |

**Net Score: +9** → **Decision: Proceed to implementation**

### Hard Violation Check
- [x] A1 (Determinism): no new implicit nondeterminism
- [x] A3 (Effects): vision stays in `{AI}`; local-path reads surface `{FS}`
- [x] A4 (Authority): no ambient access; local-path source requires `FS` cap
- [x] A7 (Machines First): structured record + typed error, not human-convenience shortcut

## Problem Statement

std/ai has **no image-input (vision) path**. `Message` is `{role, content: string, tool_calls,
tool_call_id}` — text only. `callResult`/`callJsonResult`/`callJsonSimpleResult` take a `string`;
`step`/`stepWithCache`/`stepWithStream` take `[Message]` whose `content` is a `string`.
`callImage`/`callImageBase64` **generate** images (output); they do not feed an image **into** a
vision model.

**Result:** any task requiring a model to *look at* an image is impossible in AILANG today.

**Current State:**
- `messageRecordType` ([ai_step.go:38-45](../../../internal/builtins/ai_step.go#L38-L45)) —
  `content: String()`, no image field.
- Provider `ai.Message` ([provider.go:200](../../../internal/ai/provider.go#L200)) — `Content string` only.
- Effect bridge ([ai_step.go:468](../../../internal/effects/ai_step.go#L468)) reads only the `content` string.

**Impact:**
- **stx-bench (STX Fysik-A exam benchmark):** the panel holds/abstains on every plot- and
  graph-reading item because the figure can't be shown to the model. OCR-to-text
  (`docparse --describe`) recovers axis labels but never the curve, so those items are
  un-gradable — a whole class of exam questions is excluded from the leaderboard.
- Any AILANG program doing document understanding, chart reading, screenshot QA, or
  visual grading is blocked.

## Goals

**Primary Goal:** Let an AILANG program feed one or more images into a vision-capable model
through the existing `std/ai` surface, backward-compatibly.

**Success Metrics:**
- `step(model, [msg_with_images], tools)` forwards images as each provider's native image block
  for all 5 providers (anthropic, openai, gemini, openrouter, ollama).
- A vision-aware structured-output call — `callJsonResult`-equivalent accepting `(input, images, schema)` —
  returns graded JSON in one call, no manual `step()` plumbing.
- Non-vision models return `AIError{code:"model_no_vision"}` rather than silently dropping the image.
- The STX Fysik-A plot/graph items that currently abstain become gradable (measured by the
  benchmark panel actually answering them).
- **Zero regression**: an empty `images` list produces byte-for-byte the same provider wire request as today.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **Shape (a) parallel `images` field vs (b) typed content-part list** | Determines the Message wire contract for every provider + every caller | human | design | high |
| **ImagePart source encoding** (data-URI/base64 string + `mime`, plus optional local-path) | Affects effect requirements (`FS` cap for path reads) and provider mapping | human | design | med |
| **New builtin for vision structured-output** (`callJsonResultVision` or overload) | Primary grading path; naming is a public surface | human | design | med |
| **`model_no_vision` error code + detection** (capability map vs provider-reported) | Determines whether we ship a static model→vision table or trust provider errors | agent | compile | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **Shape decision: (a) parallel `images: [ImagePart]` field.** Recommended — smaller,
      backward-compatible (empty list = today's exact wire shape), mirrors how `CacheBreakpoint`
      was added. Shape (b) (content becomes `[Part]` with `Text | Image`) is a breaking change to
      every existing caller and every provider adapter — rejected unless a reviewer overrides.
- [ ] **ImagePart shape: `{ source: string, mime: string }`** where `source` is a data-URI/base64
      string OR a local path; `mime` e.g. `"image/png"`. Confirm whether v1 supports local-path
      sources (adds `FS` cap requirement) or base64/data-URI only (pure `AI`).
- [ ] **New vision structured-output builtin name.** Recommended `callJsonResultVision(input, images, schema)`.

## Solution Design

### Overview

Add an optional `images` field to the `Message` record (shape (a)), thread it through the effect
bridge into a new `Images []ImagePart` field on the provider `ai.Message`, and map it to each
provider's native image block in the 5 adapters. Add one vision-aware structured-output builtin.
Empty `images` = current behaviour, exactly.

**Key design consequence — vision on `step`, not just a special builtin.** Because `images` lives
on the **`Message` record** (not only on the new `callJsonResultVision` convenience builtin),
*every* consumer of `step`/`stepWithCache`/`stepWithStream` gains vision input automatically:

- **motoko** (the agentic coding harness) — its multi-turn tool-loop already builds and feeds
  `Message` values through `step`; it can now attach images to any user/tool message with no new API.
- **Any AILANG application** doing its own `step`-based conversation gets the same.
- `callJsonResultVision` is then just the ergonomic one-call shortcut for the grading path — the
  raw `step` path is the general mechanism (see Example 2).

This is the reason shape (a) puts images on `Message` rather than adding a second, parallel
vision-only entry point: one record change lights up every existing step-based caller.

### Architecture

```
AILANG Message record                internal/effects/ai_step.go        internal/ai/provider.go
{ role, content,          ──►  reads content + NEW images list  ──►  ai.Message{ ..., Images []ImagePart }
  tool_calls,tool_call_id,                                                         │
  images: [ImagePart] }   ◄── NEW field on messageRecordType                       ▼
                                                              per-provider adapter maps ImagePart to:
                                                                Anthropic: {type:"image", source:{type:"base64",media_type,data}}
                                                                OpenAI:    {type:"image_url", image_url:{url}} (data-URI ok)
                                                                Gemini:    inline_data{mime_type, data}
                                                                OpenRouter: dispatch by model prefix (as step/cache today)
                                                                Ollama:    images:[base64] on the message
```

**Components:**
1. **AILANG record shape** — add `images: [ImagePart]` to `messageRecordType`; `ImagePart = {source: string, mime: string}`.
2. **Effect bridge** — extend the record→`ai.Message` conversion in `internal/effects/ai_step.go`
   to read the `images` list (default empty for back-compat).
3. **Provider abstraction** — add `Images []ImagePart` to `ai.Message` in `internal/ai/provider.go`.
4. **5 provider adapters** — map `Images` to each provider's native image block.
5. **Vision structured-output builtin** — `callJsonResultVision(input, images, schema)` returning `Result[..., AIError]`.
6. **Capability guard** — non-vision model + non-empty images → `AIError{code:"model_no_vision", retryable:false}`.

### Conflict Surface

This change touches `internal/effects/` and the `Message` *record wire contract* consumed by
every `std/ai` call path — so a Conflict Surface analysis is required even though no
parser/lexer grammar changes.

1. **What positions does this extend?** The `Message` record shape (a shared type used by
   `step`, `stepWithCache`, `stepWithStream`, and — via history — the tool-loop `callResult`
   family) and the provider `ai.Message` struct.
2. **What else lives in those positions?** Every existing caller constructs `Message` literals
   `{role, content, tool_calls, tool_call_id}`. The tool-loop feeds assistant/tool-role messages
   back into `step`. Row-polymorphic record typing means adding a field changes the record's
   principal type.
3. **Disambiguation / back-compat:** AILANG records are structurally typed. **Open question the
   sprint MUST resolve:** does adding a required `images` field break existing literals that omit
   it? Two options — (i) make `images` behave as optional/defaulted at the builtin boundary so
   `{role, content, tool_calls, tool_call_id}` still type-checks (preferred, matches "empty list =
   today's shape"), or (ii) require all callers to pass `images:[]`. Option (i) is the back-compat
   promise; verify against the record unifier before committing. This is the single highest-risk
   item.
4. **Programs that MUST still work post-change (regression fixtures — verify each exists at sprint start):**
   - Existing text-only `step`/tool-loop programs under `examples/` that build `Message` literals.
   - `stepWithCache` cache-breakpoint programs (the `CacheBreakpoint` precedent).
   - `stepWithStream` streaming programs.
   - Non-vision provider calls (configdriven, ollama text models) with empty/absent images.
   - **Action:** the sprint-planner must enumerate the actual fixture files with `grep -rln "step(" examples/`
     and confirm they still type-check and run — do NOT cite fixtures unseen.
5. **Deliberate changes:** `ai.Message` gains an `Images` field; `messageRecordType` gains an
   `images` field. Callers that opt in pass images; nothing else changes.

### Implementation Plan

**Phase 1: Record shape + effect bridge (~1 day)**
- [ ] Add `images: [ImagePart]` to `messageRecordType`; define `ImagePart = {source, mime}` record.
- [ ] Resolve back-compat: confirm omitting `images` still type-checks (Conflict Surface item 3).
- [ ] Extend record→`ai.Message` conversion in `internal/effects/ai_step.go` to read the images list.
- [ ] Add `Images []ImagePart` to `ai.Message` in `internal/ai/provider.go`.

**Phase 2: Provider adapters (~1.5 days)**
- [ ] Anthropic: `{type:"image", source:{type:"base64", media_type, data}}`.
- [ ] OpenAI: `{type:"image_url", image_url:{url}}` (data-URI accepted).
- [ ] Gemini: `inline_data{mime_type, data}` (file_data for uploaded refs deferred).
- [ ] OpenRouter: dispatch by model prefix (reuse existing step/cache routing).
- [ ] Ollama: `images:[base64]` on the message.
- [ ] Capability guard: non-vision model + images → `AIError{code:"model_no_vision"}`.

**Phase 3: Vision structured-output builtin + docs (~1 day)**
- [ ] `callJsonResultVision(input, images, schema)` builtin (thin pass-through, mirrors `callJsonResult`).
- [ ] `examples/ai_vision_input.ail` — read a local image, ask a vision model a structured question.
- [ ] std/ai doc update; CHANGELOG entry; reply to stx-bench with the new surface.

### Files to Modify/Create

**Modified files:**
- `internal/builtins/ai_step.go` — add `images` field to `messageRecordType` + `imagePartRecordType`, ~40 LOC.
- `internal/effects/ai_step.go` — read images list in record→ai.Message conversion, ~30 LOC.
- `internal/ai/provider.go` — add `Images []ImagePart` + `ImagePart` struct to abstraction, ~20 LOC.
- `internal/ai/anthropic/*.go` — image content block mapping, ~30 LOC.
- `internal/ai/openai/*.go` — image_url mapping, ~25 LOC.
- `internal/ai/gemini/*.go` — inline_data mapping, ~30 LOC.
- `internal/ai/openrouter/*.go` — prefix dispatch reuse, ~20 LOC.
- `internal/ai/ollama/*.go` — images:[base64] mapping, ~20 LOC.
- `internal/builtins/ai.go` — register `callJsonResultVision`, ~50 LOC.

**New files:**
- `examples/ai_vision_input.ail` — runnable vision-input example.

## Examples

### Example 1: Vision structured-output grading (primary path)

**Before** (impossible today):
```
-- No way to show the model a figure; benchmark abstains.
```

**After:**
```ailang
module examples/ai_vision_input
import std/ai (callJsonResultVision, ImagePart)
import std/fs as FS

export func main() -> () ! {AI, FS, IO} {
  let img = { source: FS.readImageBase64("plot.png"), mime: "image/png" }
  match callJsonResultVision(
    "Read the velocity-time graph. Return {answer: string, unit: string}.",
    [img],
    "{\"type\":\"object\",\"properties\":{\"answer\":{\"type\":\"string\"},\"unit\":{\"type\":\"string\"}}}"
  ) {
    Ok(json)  => println(json),
    Err(e)    => println("vision failed: ${e.code}")   -- e.g. "model_no_vision"
  }
}
```

### Example 2: Image on a raw step() Message

**After:**
```ailang
let msg = {
  role: "user",
  content: "What is shown in this figure?",
  tool_calls: [],
  tool_call_id: "",
  images: [{ source: dataUri, mime: "image/png" }]
}
match step(model, [msg], []) {
  Ok(r)  => println(r.content),
  Err(e) => println(e.code)
}
```

## Success Criteria

- [ ] `images: [ImagePart]` field on `Message`; omitting it still type-checks (back-compat).
- [ ] All 5 providers forward images as their native image block (unit test per provider adapter).
- [ ] `callJsonResultVision(input, images, schema)` returns structured JSON from a vision model.
- [ ] Non-vision model + non-empty images → `AIError{code:"model_no_vision"}` (typed, not silent).
- [ ] Empty `images` list produces identical provider wire request to pre-change (golden test).
- [ ] `examples/ai_vision_input.ail` runs green.
- [ ] All tests passing; std/ai doc + CHANGELOG updated; stx-bench notified.

## Testing Strategy

**Unit tests:**
- Per-provider adapter: `ImagePart` → correct native image block JSON.
- Back-compat: `Message` literal without `images` still type-checks and produces unchanged wire request (golden).
- Capability guard: non-vision model + images → `model_no_vision` AIError.

**Integration tests:**
- `callJsonResultVision` end-to-end against configdriven/mock vision provider.
- `step` with an image on a user Message → provider request contains the image block.

**Manual testing:**
- Run `examples/ai_vision_input.ail` against a live Anthropic + Gemini vision model.
- stx-bench re-runs a previously-abstaining STX Fysik-A plot item and confirms it grades.

## Deferred Decisions

- **Image source: local-path support in v1** — agent may choose to ship base64/data-URI only
  first (pure `AI` effect) and defer local-path reads (`FS` cap) to a follow-up. If path support
  ships, the read MUST surface `{FS}` in the signature.
- **Gemini `file_data` (uploaded file refs)** — agent may defer; v1 ships `inline_data` only.
- **Vision-detection strategy** — agent decides between a static model→vision capability table vs
  trusting provider-reported "no vision" errors. **Must grep `model_no_vision` unallocated first.**
- **Multiple images per message ordering** — agent may choose provider-native ordering; document it.

## Non-Goals

- **Image generation** — already exists (`callImage`/`callImageBase64`, v0.10.0); this doc is
  strictly the INPUT direction. See Related Documents.
- **Audio/video/PDF-native input** — out of scope; images only.
- **Content-part list refactor (shape b)** — rejected in favour of the backward-compatible
  parallel-field shape (a) unless a design reviewer overrides.
- **Provider-side image resizing/optimization** — forward bytes as-is; caller controls size.

## Timeline

**Days 1** — Phase 1 (record shape + effect bridge + back-compat proof).
**Days 2–3** — Phase 2 (5 provider adapters + capability guard).
**Day 4** — Phase 3 (vision structured-output builtin, example, docs, stx-bench reply).

**Total: ~3–4 days.**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Adding `images` field breaks existing `Message` literals (record typing) | High | Conflict Surface item 3 — prove optional/defaulted behaviour before Phase 2; golden back-compat test |
| Provider image-block formats drift / one adapter wrong | Med | Per-provider unit test with recorded native JSON; forensic check against provider docs |
| Silent image drop on non-vision model | Med | Typed `model_no_vision` AIError is an explicit success criterion, not best-effort |
| `model_no_vision` code collides with existing error code | Low | Grep-verify unallocated at sprint start (Verification Log TODO) |

## Related Documents

**Implemented (may inform design):**
- [design_docs/implemented/v0_10_0/m-ai-image-generation.md](../../implemented/v0_10_0/m-ai-image-generation.md) (0.43) —
  image **generation** (`callImage`). This doc is the **input** counterpart; the two are
  complementary, not overlapping. Reuse its provider-dispatch patterns.

**Planned (checked — no overlap):**
- Neural top match was 0.31 (`m-oracle-adequacy`), unrelated — confirms this is not a duplicate.

## References

- [Design Axioms](/docs/references/axioms)
- Feature request: agent message `msg_20260716_070552_f9104997` (stx-bench, 2026-07-16)
- Related DX report: `msg_20260715_074403_ab7dfaa9` (per-call `step(model)` friendly-name
  resolution) — separate issue, not addressed here.
- [`internal/builtins/ai_step.go`](../../../internal/builtins/ai_step.go),
  [`internal/ai/provider.go`](../../../internal/ai/provider.go),
  [`internal/effects/ai_step.go`](../../../internal/effects/ai_step.go)

## Future Work

- Local-path and uploaded-file (Gemini `file_data`) image sources.
- Audio/video/PDF-native multimodal input.
- Image-token cost surfacing in the Usage record.

---

**Document created**: 2026-07-16
**Last updated**: 2026-07-16
