# std/ai — typed media parts + Gemini server-side file slicing (page_options, resolution, content_layers)

**Status**: Planned
**Target**: v0.60.7
**Priority**: P1 (Medium) — blocks docparse's `pdfBackend=ai` cost/latency ceiling today; no in-flight sprint depends on it
**Estimated**: ~2 weeks (M0 wire probe ~0.5d · M1 typed surface + Vertex ~5d · M2 Developer API + docs ~2d · testing/migration buffer ~2.5d)
**Dependencies**: None new — rides PR #2077 (Vertex regional/EU host, already merged on dev — verified below)

> **Provenance**: GitHub issue [sunholo-data/ailang#2458](https://github.com/sunholo-data/ailang/issues/2458),
> forwarded from docparse agent message `inbox_1795507642049_08f2dcfb` (consumer: `ailang-parse`
> `pdfBackend=ai`, opt-in `pdfBackend=hybrid`). Triage summary arrived via PR #2459; the triage
> summary as carried in the issue is the input to this doc (the triage file itself was not present
> on the `coordinator/task-586009a6` branch at drafting time — every load-bearing claim below was
> re-verified live instead, per the Verification Log).

> **Scope shape**: this doc settles the four open decisions named in the issue — (D1) generalize
> `ImagePart` vs. sibling type, (D2) per-part vs. per-call slicing options, (D3) transport staging
> (Developer API vs. Vertex AI), (D4) usage metadata surfacing. It is a **stdlib + provider-adapter
> change, not a parser/typechecker change** — no new syntax, no compiler-pipeline work.

## Verification Log

Every language/behaviour claim below was checked against the v0.60.6 code, not assumed.

| # | Claim | Method | Result |
|---|-------|--------|--------|
| 1 | `std/ai` exposes exactly one media-input type, `ImagePart = { source: string, mime: string }`, attached to `Message.images: [ImagePart]` | Read [`std/ai.ail:83-97`](../../../std/ai.ail#L83-L97) | **Confirmed** |
| 2 | Single-shot calls have **no typed media input** — the only file/document path today is a *stringly-typed JSON convention* inside the Gemini adapter: `{"mode":"multimodal","mimeType":…,"fileUri"\|"data":…,"prompt"\|"fileName":…}` | Read [`internal/ai/gemini/generate.go`](../../../internal/ai/gemini/generate.go) `buildParts` (comment block ~L188–230) | **Confirmed** — no `page_options`/`sheet_index`/`offset`/`content_layers` keys exist in that convention |
| 3 | No slicing controls exist anywhere in the provider layer | `grep -rin "pageOptions\|startPage\|sheetIndex\|contentLayers\|visibleCells\|startOffset\|start_page\|content_layers" internal/ai/` | **Confirmed empty** (negative-existence claim, freshly grepped this session) |
| 4 | AILANG records are **closed** — adding a required field to `Message` breaks every existing `Message` literal | Live `ailang check` on a record literal missing a declared field → `type error: … record field mismatch … Hint: AILANG records are closed — add the field(s) to the literal` (exit 1); also asserted in-code at [`internal/builtins/ai_step.go:49-53`](../../../internal/builtins/ai_step.go#L49-L53) | **Confirmed** — any `Message` field change is a deliberate breaking change (v0.30.0 precedent exists, see Related Documents) |
| 5 | A unified `MediaPart` variant (`Image({…}) \| File({…})`, record payloads) is expressible in current AILANG | Live `ailang check` + `ailang run` on a local `type MediaPart = Image({source,mime,resolution}) \| File({uri,mime,resolution,options})` with constructor-as-value, list membership, and `match` — **exit 0**, ran to `Summarize file` | **Confirmed** — the proposed type and every example snippet below compile today |
| 6 | `StepResult` already surfaces `input_tokens`/`output_tokens` (+ cache fields) on step paths | Read [`std/ai.ail`](../../../std/ai.ail) `StepResult` docs | **Confirmed** — multi-turn requirement #4 needs no step-side change |
| 7 | Single-shot `callResult`/`callJsonResult`/`callJsonSimpleResult` return **only** `Result[string, AIError]` — no usage surfaces | Read [`std/ai.ail:141-161`](../../../std/ai.ail#L141-L161) | **Confirmed** — usage on single-shot calls is a genuine gap (D4) |
| 8 | `ai.Response` already carries `InputTokens`/`OutputTokens`/`TotalTokens`/`ReasonTokens` from `usageMetadata` | Read [`internal/ai/provider.go:336-367`](../../../internal/ai/provider.go#L336-L367), [`internal/ai/gemini/generate.go:118-132`](../../../internal/ai/gemini/generate.go#L118-L132) | **Confirmed** — Go side needs plumbing, not new metering |
| 9 | PR #2077 (Vertex regional host) is merged: `vertexAIBaseURL` maps `eu`→`https://aiplatform.eu.rep.googleapis.com/v1`, `us`→`…aiplatform.us.rep…`, regions→`https://{region}-aiplatform.googleapis.com/v1`, non-regional names refused | Read [`internal/ai/gemini/client.go:28-53`](../../../internal/ai/gemini/client.go#L28-L53) | **Confirmed on this branch** — EU residency transport is available *today*; this feature must only fill it with media parts, not re-do transport |
| 10 | Gemini wire part already supports external-file references (`fileData{MimeType, FileUri}` for `gs://`) and inline (`inlineData`); parts attach to **user-role messages only** today — tool results go through `functionResponse` string content | Read [`internal/ai/gemini/types.go:63-107`](../../../internal/ai/gemini/types.go#L63-L107), [`internal/ai/gemini/step.go:141-175,261-280`](../../../internal/ai/gemini/step.go) | **Confirmed** |
| 11 | `AIError` code `CapabilityNotSupported` already exists in the canonical code enum; **no new error code is needed** | Read [`std/ai.ail`](../../../std/ai.ail) `AIError` docs (code list) | **Confirmed** — no `MODxxx`/`Exxx` allocation in this design |
| 12 | Options-as-validated-JSON-string is established std/ai idiom (strict keys, unknown key → `Err(SchemaValidation)`) | Read [`std/ai.ail`](../../../std/ai.ail) `callSpeech` docs; [`internal/builtins/ai_speech.go:41-42`](../../../internal/builtins/ai_speech.go#L41-L42) | **Confirmed** |
| 13 | **Exact Gemini wire field names for slicing controls are NOT verified** — the issue itself flags "depending on where Google wires `page_options`" (generateContent vs. `/v1beta/interactions`) | Not checkable offline | **Explicitly deferred** — M0 live probe before any wire struct is written (see Deferred Decisions); the *typed AILANG surface* below is wire-name-independent |

**Conclusion**: the issue's gap description is accurate. std/ai supports string prompts and
`ImagePart` vision input only; documents/spreadsheets/media have no typed input path, no
server-side slicing, and no single-shot usage metering. The four open decisions are settled below.

## Problem Statement

docparse (the `ailang-parse` pipeline behind `pdfBackend=ai` and opt-in `hybrid`) sends whole PDFs
to Gemini today and slices nothing. Concretely, from the issue:

1. **No typed file/document input.** `std/ai` accepts string prompts and `ImagePart {source, mime}`.
   A document reaches Gemini only through the adapter's private stringly-typed JSON convention
   (Verification Log #2) — invisible to the type checker, undocumented in `std/ai`, and unable to
   express `resolution` at all.
2. **No server-side slicing.** `page_options` (start/end page), `sheet_index`/`visible_cells`,
   `start_offset`/`end_offset`/`fps`, and `content_layers` do not exist anywhere (Verification Log
   #3), so every call ships the whole artifact and pays whole-artifact latency and token cost —
   and for EU-resident tenants, whole-artifact bytes cross the boundary that `aiplatform.eu.rep.googleapis.com`
   exists to police (Verification Log #9).
3. **No usage metering on single-shot calls.** `callJsonResult` returns text only (Verification Log #7),
   so a sliced call's savings are unmeasurable — exactly the metering a benchmark loop needs.

**Impact:** docparse (primary consumer), any AILANG agent doing document QA/spreadsheet analysis,
and the benchmark harness's per-call cost telemetry. Severity: cost/latency ceiling on a
production consumer — significant, not language-blocking.

## Goals

**Primary Goal:** let an AILANG program hand Gemini a typed, sliced file part
(`gs://` or inline, EU-resident capable) and read back its per-call token usage —
replacing the stringly multimodal JSON convention with a closed, statically typed surface.

**Success Metrics:**
- docparse can request pages 3–5 of a `gs://` PDF with `content_layers=["page_text"]` through
  `std/ai` with no JSON hand-assembly and measure the token delta vs. whole-file.
- `Message`/`MediaPart` literals are fully type-checked; a bad `content_layers` value fails as
  `Err(AIError{code:"SchemaValidation"})`, never a silent drop.
- A single-shot sliced call returns `{text, input_tokens, output_tokens}` in one result record.
- The stringly `{"mode":"multimodal",…}` convention is preserved (back-compat) but documented as
  superseded by the typed path.
- All existing vision/tool-loop/streaming fixtures keep passing (see Compatibility Surface).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **D1: Generalize `ImagePart` into a unified `MediaPart` variant** (`Image({...}) \| File({...})`) replacing `Message.images: [ImagePart]` with `Message.media: [MediaPart]` — **breaking change to `Message`** (records are closed, Verification Log #4) | Sibling `FilePart` type would force a second list field on `Message` (two attachment lists to keep consistent, order lost), and every future media kind would add another field; a variant keeps one ordered list and matches the `StreamChunk` precedent for provider-shaped data. But it re-breaks every `Message` literal in the wild, incl. external motoko agents — same trade v0.30.0 made | **human** (breaks external consumers) | design | **high** |
| **D2: Slicing options attach per-part** (`File.options`), not per-call | A multi-turn conversation can carry several files; per-call options cannot express "pages 3–5 of *this* PDF, sheet 2 of *that* XLSX". Slicing is a property of a file, not of a turn | human | design | med |
| **D3: Options are a validated JSON string** (`options: string` on the `File` payload; unknown key → `Err(SchemaValidation)`), not fully typed sub-records | AILANG closed records make every sub-field required — `FileOptions` would need sentinel defaults (`0` = unset) for `start_page`, which is exactly the silent-fallback trap CLAUDE.md forbids. The JSON string keeps optionality explicit and matches `ToolSchema.parameters` / `callSpeech` idiom (Verification Log #12) | agent | design | low |
| **D4: Transport staging — Vertex first (incl. EU multi-region), Developer API second** | `gs://` URIs are Vertex-only, and EU residency (docparse's driver) rides the already-merged regional host (Verification Log #9). Google has not settled where `page_options` lands (issue's own caveat); building against the Developer API first would gate an EU-resident consumer on an endpoint that may not carry the fields at all | human | design | med |
| **D5: Single-shot usage via a new result record** `MediaCallResult = {text, input_tokens, output_tokens}` on a new `callMediaResult` builtin — `callResult` & friends stay text-only | Changing `callResult`'s return type would break every existing caller for a benefit only media callers need; a sibling function is the established additive pattern (`stepWithCache`, `callImageWithRefs`) | agent | design | low |
| **D6: Wire field names + endpoint for slicing** (generateContent vs. `/v1beta/interactions`) locked by live probe | Wrong wire names = 400s or (worse) silently ignored fields on a metering-sensitive path | agent (with human sign-off on the probe result) | design (M0, before any wire struct) | low |

### Design Freeze

All six decisions are **settled by this doc** and must be ratified together (D1 is the only one
that breaks external source):

- [x] D1 — breaking change accepted: `Message.images: [ImagePart]` → `Message.media: [MediaPart]`;
      `ImagePart` record **remains exported** for the generation-refs path (`callImageWithRefs` /
      `callImageBase64WithRefs`, Verification Log #10/`std/ai.ail:421-433`). Migration is mechanical:
      `images: [{source, mime}]` → `media: [Image({source, mime, resolution: ""})]`; `images: []` → `media: []`.
- [x] D2 — per-part options on the `File` constructor.
- [x] D3 — `options` is a JSON string validated per media class (see Solution Design).
- [x] D4 — M1 ships Vertex (incl. `eu`/`us` multi-region); Developer API follows when Google wires
      the fields; until then a slicing request against an endpoint that lacks the fields returns
      `Err(AIError{code:"CapabilityNotSupported", retryable:false})` — never a silent whole-file send.
- [x] D5 — new additive builtin; existing single-shot returns unchanged.
- [x] D6 — M0 probe (live `curl` against a one-page `gs://` PDF on Vertex `eu`) pins the wire
      names before M1 touches `types.go`.

## Solution Design

### Overview

Replace `Message.images` with a single typed attachment list, `Message.media: [MediaPart]`, where
`MediaPart` is a two-constructor variant: `Image` (today's vision input, plus `resolution`) and
`File` (documents/spreadsheets/audio/video, plus `resolution` and a per-part `options` JSON string
carrying the server-side slicing controls). Add one single-shot builtin, `callMediaResult`,
which takes prompt + media list + call options and returns `Ok(MediaCallResult)` — text **plus**
input/output tokens. The multi-turn path needs no signature change: `step`/`stepWithCache`/
`stepWithStream` already flow `Message` records and already return `input_tokens`/`output_tokens`
on `StepResult` (Verification Log #6), so media parts and slicing ride the existing loop unchanged.

Per provider: only the Gemini adapter implements media-file slicing in M1/M2. The other four
adapters (anthropic, openai, openrouter, ollama) get `Message.Media` decoding with the same
loud-failure rule the vision rollout used: a `File` part with non-empty `options` (or a `File`
part at all, for providers without file input) → `Err(AIError{code:"CapabilityNotSupported"})`,
never a silent drop — mirroring how `callImageWithRefs` already rejects unsupported providers
(`std/ai.ail:429-431`).

### The AILANG surface (exact, verification-log-checked syntax)

```ail
-- std/ai.ail additions / changes (v0.60.7)

-- MediaPart is one media INPUT attachment (generalizes ImagePart, M-STD-AI-VISION-INPUT).
--   resolution : "" | "low" | "medium" | "high" — "" = provider default; anything
--                else is Err(SchemaValidation) at the effect boundary.
-- Image.source : base64 payload or data-URI (unchanged semantics).
-- File.uri     : gs://bucket/object | https://… | data:… | raw base64 payload.
--                gs:// and https:// map to fileData (server-side, no inline copy —
--                required for EU residency); data:/base64 map to inlineData.
-- File.options : JSON string, "" = no slicing (whole artifact). Keys are validated
--                per media class — unknown keys are Err(SchemaValidation).
export type MediaPart =
  Image({ source: string, mime: string, resolution: string })
| File({ uri: string, mime: string, resolution: string, options: string })

-- Message: `images: [ImagePart]` is REPLACED by `media: [MediaPart]` (closed records —
-- breaking, see Design Freeze D1). `media: []` is a text-only message.
export type Message = { role: string, content: string, tool_calls: [ToolCall], tool_call_id: string, media: [MediaPart] }

-- Single-shot media call WITH usage (D4/D5): text + token counts in one record.
export type MediaCallResult = { text: string, input_tokens: int, output_tokens: int }

export func callMediaResult(prompt: string, media: [MediaPart], options: string) -> Result[MediaCallResult, AIError] ! {AI} =
  _ai_call_media_result(prompt, media, options)
```

`callMediaResult`'s call-level `options` JSON carries only call-level keys today — `model`
(per-call override, same resolution rule as `callSpeech`) — and is validated with the same
unknown-key-→-`SchemaValidation` rule. Slicing lives per-part (D2), never here.

### Slicing options — per media class (D2/D3)

Validated against the `File` part's `mime`; the class is picked by MIME prefix, and an option that
does not belong to the class is `Err(SchemaValidation)` (not ignored):

| Media class (MIME examples) | Options keys | Semantics |
|---|---|---|
| Documents / slides (`application/pdf`, `application/vnd.openxmlformats-officedocument.presentationml.*`, `application/vnd.google-apps.presentation`) | `page_options: {start_page: int, end_page: int}`, `content_layers: [string]` | 1-indexed inclusive range; `start_page < 1`, `end_page < start_page` → `SchemaValidation`. `content_layers` ⊆ `{page_text, page_image}` |
| Spreadsheets (`application/vnd.openxmlformats-officedocument.spreadsheetml.*`, `application/vnd.google-apps.spreadsheet`) | `sheet_index: int`, `visible_cells: string`, `content_layers: [string]` | `visible_cells` is an A1 range (`"A1:G42"` — validated by regex, not silently passed through); `content_layers` ⊆ `{data, formulas, visuals}` |
| Audio / video (`audio/*`, `video/*`) | `start_offset: string`, `end_offset: string`, `fps: int`, `content_layers: [string]` | Offsets are ISO-8601 durations or seconds (`"PT90S"` / `"90"`); `content_layers` ⊆ `{transcript, frames, audio}` |
| Images via `File` (rare) | `content_layers: [string]` | ⊆ `{image_text, image}` — resolution is on the part, not in options |

Validation lives in the **Gemini adapter request-builder** (it is the only M1/M2 implementer and
the only place that knows the wire encoding), surfaced as typed `AIError`s by the existing
`Step`/`Generate` error-classification path — identical to how `callSpeech` rejects bad options
keys today.

### Transport (D4)

- **M1 — Vertex AI.** `gs://` URIs map to `fileData` parts server-side (no inline copy). The
  host already follows `GOOGLE_CLOUD_LOCATION` through `vertexAIBaseURL`, including
  `aiplatform.eu.rep.googleapis.com` for `eu` (Verification Log #9) — this design adds **zero**
  transport code and treats #2077 as a hard dependency, verified merged.
- **M2 — Gemini Developer API (AI Studio).** Same typed surface, enabled only once the M0 probe
  (or Google's docs) confirm the slicing fields exist on the Developer API surface. Until then,
  a `File` part with non-empty `options` against AI Studio → `CapabilityNotSupported`.
- **Everywhere** — `Image` parts and `File` parts with `options: ""` degrade to today's exact
  wire behavior (inline/base64 vision input) on every provider that already supports images.

### Implementation Plan

**M0: Wire probe + freeze** (~0.5 day)
- [ ] Live `curl` (ADC) against `https://aiplatform.eu.rep.googleapis.com/v1/…/models/<model>:generateContent`
      with a one-page `gs://` PDF + the issue's `page_options`/`content_layers` shapes; pin exact
      JSON field names + which surface (`generateContent` vs `/v1beta/interactions`) accepts them.
- [ ] Probe whether tool-result media can ride `functionResponse` parts (expected: no) — decides the
      tool-role attachment shape in Phase 2.
- [ ] Record probe output in this doc's follow-up; lock wire names in `types.go` only after that.

**Phase 1: Typed surface (M1 core)** (~3 days)
- [ ] `std/ai.ail`: `MediaPart`, `Message.media` (breaking), `MediaCallResult`, `callMediaResult`;
      keep `ImagePart` exported for generation refs.
- [ ] `internal/builtins/ai_step.go`: `mediaPartRecordType` (variant), `messageRecordType` swap,
      new `_ai_call_media_result` registration + type builder.
- [ ] `internal/effects/ai_decode.go` + `ai_encode.go`: decode `media` list (variant → `ai.MediaPart`),
      encode `MediaCallResult`; strict validation (`resolution` enum, option-key classes) here so
      every provider gets it for free.
- [ ] `internal/ai/provider.go`: `ai.MediaPart`, `ai.Message.Media`, `ai.MediaCallResult` plumbing
      on `Request`/`Response` (Response tokens already exist, Verification Log #8).

**Phase 2: Gemini adapter (M1 Vertex)** (~2 days)
- [ ] `internal/ai/gemini/step.go`: `userImageParts` → `userMediaParts` (text part + one part per
      media; `gs://`/`https` → `fileData`, `data:`/base64 → `inlineData`, per the uri rule).
- [ ] `internal/ai/gemini/generate.go`: single-shot path consumes `Message.Media` (the stringly
      `{"mode":"multimodal"}` convention stays for back-compat; typed media takes precedence).
- [ ] `types.go`: wire structs for the M0-confirmed slicing fields; capability gate for AI Studio.
- [ ] Tool-role media (requirement: "multi-turn step/tool results"): a tool result carrying media
      attaches its media parts as a **user-turn** content block after the `functionResponse`
      (our adapter encodes tool results as text-only `functionResponse` content today —
      Verification Log #10; whether Gemini's API accepts media parts inside `functionResponse`
      is expected to be "no" but is an M0 probe item, not a verified fact). Documented in the
      `std/ai.ail` comment.

**Phase 3: Stub fixtures, examples, migration, docs (M2 tail)** (~2 days)
- [ ] `internal/effects/ai_stub_fixtures.go`: `media` fixture kind keyed by media part shape
      (precedent: speech fixtures keyed by `{style, text, voice}`) + `usage` on stub responses.
- [ ] Migrate the four in-repo `Message` literal sites (Compatibility Surface) and add
      `examples/runnable/ai_file_slicing.ail` (PDF page range + spreadsheet range, stub-runnable).
- [ ] `docs/recipes/ai-token-streaming` sibling: `docs/docs/recipes/ai-file-slicing` (or extend the
      vision recipe); `llms.txt` regeneration.

### Files to Modify/Create

**New files:**
- `internal/builtins/ai_media.go` — `_ai_call_media_result` registration (~120 LOC)
- `examples/runnable/ai_file_slicing.ail` (~80 LOC)

**Modified files:**
- `std/ai.ail` (+~60/-8 LOC) — types + `callMediaResult`; `Message` field swap
- `internal/builtins/ai_step.go` (+~90/-20 LOC) — record types, Message shape, media variant
- `internal/effects/ai_decode.go` / `ai_encode.go` (+~110/-30 LOC) — media decode, result encode, validation
- `internal/effects/ai_step.go` (+~40 LOC) — new effect op wiring
- `internal/effects/ai_stub_fixtures.go` (+~80 LOC) — media/usage fixtures
- `internal/ai/provider.go` (+~70/-10 LOC) — `MediaPart`, `Message.Media`, request plumbing
- `internal/ai/gemini/{types,step,generate}.go` (+~180/-40 LOC) — wire structs, `userMediaParts`, capability gate

## Examples

### Example 1: docparse — pages 3–5 of an EU-resident PDF, single-shot with usage

**Before (stringly convention, no slicing, no metering — today):**
```ail
-- hand-assembled JSON; whole PDF crosses the wire; savings unmeasurable
let raw = AI.callJson("{\"mode\":\"multimodal\",\"mimeType\":\"application/pdf\",\"fileUri\":\"gs://eu-bucket/report.pdf\",\"prompt\":\"Extract the invoice totals\"}", schema)
```

**After (typed, sliced, metered):**
```ail
import std/ai (callMediaResult)
import std/result (Result)

export func main() -> () ! {AI} {
  let pdf = File({ uri: "gs://eu-bucket/report.pdf", mime: "application/pdf", resolution: "high",
                   options: "{\"page_options\":{\"start_page\":3,\"end_page\":5},\"content_layers\":[\"page_text\"]}" })
  match callMediaResult("Extract the invoice totals", [pdf], "") {
    Ok(r)  => -- r.input_tokens / r.output_tokens: the measured slice cost
      (),
    Err(e) => ()   -- e.g. SchemaValidation for a bad page_options key, CapabilityNotSupported off-Vertex
  }
}
```

### Example 2: multi-turn — spreadsheet analysis in a tool loop (media rides `step` unchanged)

```ail
let sheet = File({ uri: "gs://eu-bucket/q3.xlsx", mime: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
                   resolution: "", options: "{\"sheet_index\":0,\"visible_cells\":\"A1:G42\",\"content_layers\":[\"data\",\"formulas\"]}" })
let msgs: [Message] = [{ role: "user", content: "Which rows miss VAT?", tool_calls: [], tool_call_id: "", media: [sheet] }]
-- step("", msgs, tools) — StepResult.input_tokens / output_tokens already carry the usage
```

### Example 3: migration of an existing vision literal (the D1 break, mechanical)

```ail
-- before (v0.60.6)
let msgs = [{ role: "user", content: prompt, tool_calls: [], tool_call_id: "", images: [{ source: png, mime: "image/png" }] }]
-- after (v0.60.7)
let msgs = [{ role: "user", content: prompt, tool_calls: [], tool_call_id: "", media: [Image({ source: png, mime: "image/png", resolution: "" })] }]
```

## Success Criteria

- [ ] `ailang check std/ai.ail` clean; every snippet in this doc compiles (already verified, Verification Log #5)
- [ ] `callMediaResult` returns `{text, input_tokens, output_tokens}`; stub fixtures make it deterministic
- [ ] `File` part with `gs://` URI on Vertex `eu` reaches `aiplatform.eu.rep.googleapis.com` as `fileData` (no inline copy) — asserted in a `location_test.go`-style unit test plus one manual live check
- [ ] Bad option keys (`content_layers: ["page_text"]` on a spreadsheet), bad enums (`resolution: "ultra"`), bad ranges (`end_page < start_page`) → `Err(AIError{code:"SchemaValidation", retryable:false})`
- [ ] Slicing request against a provider/endpoint without the fields → `CapabilityNotSupported`, never a silent whole-file send
- [ ] All four migrated in-repo `Message` sites + full test suite passing (`make test`; `make test-core` <15s unaffected)
- [ ] `examples/runnable/ai_file_slicing.ail` runs under `--ai-stub` and against a live Vertex model
- [ ] Docs updated (`docs/docs/recipes/*`, `llms.txt`) — stringly multimodal convention documented as superseded
- [ ] All tests passing; docs updated

## Compatibility Surface (this is a stdlib + builtins change, not a parser change)

No syntax or typechecker positions change; the risk surface is the **`Message` record shape**,
which is constructed by user programs and external agents:

| Position | Existing form (v0.60.6) | New form (v0.60.7) |
|---|---|---|
| `Message` literal field `images` | `images: [ImagePart]` (required since v0.30.0) | removed — replaced by `media: [MediaPart]` (required) |
| `ImagePart` record | `{ source, mime }` | unchanged, still exported — **generation refs only** (`callImageWithRefs`) |
| Attachment list on a user message | one list, images only | one list, any media — same field count, same `[] = text-only` rule |

**Programs that MUST still work (regression fixtures):**
- `examples/runnable/ai_tool_loop.ail` — `runTools` loop + `Message` literals (migrate `images: []`→`media: []`)
- `examples/runnable/ai_caching.ail` — `stepWithCache` + `Message` literals
- `examples/runnable/ai_streaming.ail` — `stepWithStream` + `Message` literals
- `examples/runnable/ai_vision_input.ail` — vision input (migrate to `Image(...)` constructor)
- `std/ai.ail` + `std/ai/streaming.ail` — `make test-core` (verified clean today, Verification Log basis)

**What deliberately changes:** the `Message` field name/type (D1) — every `Message` literal in the
wild must add `media:` (mechanically, per Example 3). Nothing else: `step`/`runTools`/streaming/
`callImage*` signatures are byte-identical. External consumers (motoko agents) need the same
mechanical migration — called out as the cost of D1, accepted at freeze.

## Testing Strategy

**Unit tests:**
- `internal/effects`: media decode (variant→struct, uri-class routing, resolution enum, option-class
  validation per MIME), `MediaCallResult` encode, stub fixtures (keyed media + usage).
- `internal/builtins`: record-type shapes match `std/ai.ail` declarations (the
  `TestMakeOkStepResult_MessageMatchesDeclaredType` pattern).
- `internal/ai/gemini`: `userMediaParts` part shapes; `location_test.go` pattern for the EU host +
  `fileData` assertion; capability-gate matrix (provider × options-set).

**Integration tests:**
- `--ai-stub` end-to-end: `examples/runnable/ai_file_slicing.ail` (document, spreadsheet, audio class)
- Back-compat: the stringly multimodal JSON convention still parses on the single-shot path.

**Manual testing:**
- M0/M1 live probe against Vertex `eu` with a real `gs://` PDF; measured token delta
  sliced vs. whole-file recorded in the implementation report.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact Gemini wire JSON field names + endpoint (`generateContent` vs `/v1beta/interactions`) — **agent**, via the M0 probe; human signs off the probe artifact before M1 wire structs land (D6)
- Offset unit spelling for audio/video (`"PT90S"` vs `"90"`) — agent may accept both or pick one; must be documented either way
- Whether `fps` applies when `content_layers` excludes `"frames"` (validate as redundant vs. reject) — agent
- Stub fixture file format for media responses (inline in fixtures JSON vs. sibling files) — agent
- `docs` placement (new recipe vs. extending the vision recipe) — agent
- Whether tool-role media attaches as user-turn parts (planned) or is M1-restricted to user messages with M3 to follow — agent, but the restriction (if chosen) must be a typed `CapabilityNotSupported`, not a silent drop

## Non-Goals

- **Client-side slicing** (PDF page extraction, sheet re-encoding in AILANG/Go) — out of scope; the whole point is server-side slicing on the provider
- **Anthropic/OpenAI/OpenRouter/Ollama file-slicing implementations** — they get the typed decode + loud `CapabilityNotSupported`; native support lands per-provider later if their APIs grow the capability
- **Local-path media sources** (`source: "/data/report.pdf"`) — still deferred exactly as v0.30.0 deferred it; `uri` covers `gs://`/`https`/`data:`/base64
- **Changing `callResult`/`callJsonResult` return types** — D5 keeps them text-only; usage rides the new `callMediaResult`
- **Cache/breakpoint interaction with media parts** — orthogonal; nothing here touches `CacheBreakpoint`

## Timeline

**Week 1** (~6 days): M0 probe (0.5d) → Phase 1 typed surface (3d) → Phase 2 Gemini adapter starts (2.5d)
**Week 2** (~5 days): Phase 2 finish + stub fixtures (1.5d) → migration + example + docs (1.5d) → full test pass + manual live check (2d)

**Total: ~11 days** (2× the naive 5-day estimate, per skill guidance)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Google wires `page_options` differently than the issue assumes (or not at all yet) | High | M0 probe before any wire code; typed surface is wire-independent; AI Studio stays gated `CapabilityNotSupported` until M2 |
| D1 break strands external motoko agents on the old `Message` shape | High | Mechanical one-line-per-literal migration (Example 3); announce in CHANGELOG with the v0.30.0 precedent cited; v0.60.7 is a minor bump carrying an explicitly-flagged breaking stdlib change |
| Sliced responses under-report usage (partial views billed oddly) | Med | `input_tokens`/`output_tokens` come from `usageMetadata` verbatim (Verification Log #8); benchmark harness records per-call deltas in the implementation report |
| Options validation drifts between effect layer and Gemini adapter | Med | Validate once, in the effect layer decode (every provider inherits); adapter only maps wire names |
| EU residency silently violated by an inline fallback | High | uri-class routing is tested: `gs://`→`fileData` only; any inline fallback for `gs://` is a bug, asserted in unit tests |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | AI calls are already non-deterministic via `{AI}`; media adds none, and stub fixtures keep tests deterministic |
| A2: Replayability | +1 | Media parts (uri/base64 + slicing options) are captured in the `Message` record → replayable in traces like any other AI input |
| A3: Effect Legibility | 0 | Stays inside `! {AI}`; no new effects (local-path sources would add `{FS}` but are a Non-Goal) |
| A4: Explicit Authority | 0 | No ambient access — `gs://`/`https`/base64 URIs are caller-supplied strings; no FS capability implied |
| A5: Bounded Verification | +1 | Record/variant shapes are locally type-checkable (verified, Log #5); option-class validation is a closed enum check |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +1 | Typed `MediaPart` variant replaces a stringly JSON convention the type checker could not see; machines decide validity instead of parsing prose |
| A8: Minimal Syntax | +1 | Zero new syntax — a record field swap + one variant type + one builtin, mirroring `CacheBreakpoint`/`stepWithCache` additive pattern |
| A9: Cost Visibility | +1 | `MediaCallResult` makes per-call token cost a first-class return value — the metering docparse needs to prove slicing saves |
| A10: Composability | +1 | `step`/`runTools`/`stepWithCache`/`stepWithStream` signatures unchanged; media rides the existing `Message` flow |
| A11: Structured Failure | +1 | Bad options/resolution → typed `SchemaValidation`; unsupported endpoint → typed `CapabilityNotSupported` — never silent drop (the v0.30.0 "model_no_vision" discipline) |
| A12: System Boundary | +1 | Provider wire mapping (inline vs fileData, Vertex vs AI Studio) is the boundary crossing, made explicit per-endpoint with residency consequences documented |

**Net Score: +7** → **Decision: Proceed to implementation** ✅

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced
- [x] A3 (Effects): media stays inside `{AI}`; no hidden FS access (local-path sources explicitly a Non-Goal)
- [x] A4 (Authority): no ambient access granted — URIs are inert strings
- [x] A7 (Machines First): typed variant + strict key validation replaces human-convention JSON

### Decision Thresholds

Net +7 ≥ +2 → **proceed**. No −1 on A1/A3/A4/A7.

## Related Documents

The auto-search hits on the scaffold (`m-process-subcmd-allowlist` 1.00, `m-v1-simplification-s4-sprint-plan` 1.00, …) were
keyword noise on "std/ai/gemini/file/slicing" — none is topically adjacent; they are not cited. The
genuinely related documents, read during drafting:

- [design_docs/implemented/v0_30_0/m-std-ai-vision-input.md](../v0_30_0/m-std-ai-vision-input.md) — **the direct precedent**: added `Message.images: [ImagePart]` as a deliberate breaking change; its Design Freeze rationale and per-provider "typed error, never silent drop" discipline are followed here
- [design_docs/planned/ailang-core-triage/gemini-vertex-regional-host.md](../ailang-core-triage/gemini-vertex-regional-host.md) — the triage that produced PR #2077 (`vertexAIBaseURL` regional/EU host), this design's transport dependency
- [design_docs/planned/v0_53_0/ailang-parse-positioning.md](../v0_53_0/ailang-parse-positioning.md) — consumer context: `ailang-parse` `pdfBackend` escalation policy (`ai` / opt-in `hybrid`) that this feature serves
- [design_docs/implemented/v0_52_0/m-openrouter-image-output.md](../v0_52_0/m-openrouter-image-output.md) — the `callImageWithRefs`/`ImagePart` generation-refs path that this design intentionally leaves unchanged

Distinction note (duplicate gate): none of these covers typed **media input parts**, server-side
**slicing**, or **single-shot usage metering** — verified absent by grep (Verification Log #3).

## References

- **Issue**: [sunholo-data/ailang#2458](https://github.com/sunholo-data/ailang/issues/2458) — the request, incl. the four open decisions
- **Transport dependency**: PR #2077 — Vertex regional host (merged on dev; re-verified on this branch, `internal/ai/gemini/client.go:42`)
- **Consumer**: docparse agent message `inbox_1795507642049_08f2dcfb`; `ailang-parse` `pdfBackend=ai` / opt-in `hybrid`
- **Axiom reference**: [Design Axioms](/docs/references/axioms)
- **Gemini wire docs for slicing fields**: to be pinned by the M0 probe (Verification Log #13) — deliberately not cited from memory

## Future Work

- **Developer API (AI Studio) slicing** — M2, gated on Google wiring the fields there (D4)
- **Anthropic/OpenAI document-part parity** (`file`/`input_file` blocks with their own slicing semantics, if/when offered)
- **Local-path media sources** with explicit `{FS}` capability on the part — picks up v0.30.0's deferred follow-up
- **Cache breakpoints across media turns** — cached-prefix interaction with large stable attachments
- **`callJsonResultMedia`** — schema-enforced JSON output over sliced media, if the graded-extraction path (docparse `hybrid`) needs response schemas rather than prompt-level JSON contracts

---

**Document created**: 2026-11-24
**Last updated**: 2026-11-24