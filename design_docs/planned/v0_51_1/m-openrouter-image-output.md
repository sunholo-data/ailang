# M-OPENROUTER-IMAGE-OUTPUT: Route AI Image Generation Through OpenRouter (Chat-API Modalities + Per-Call Model Choice)

**Status**: Planned
**Target**: v0.51.1
**Priority**: P1 (Medium)
**Estimated**: 2–3 days
**Dependencies**: None new — builds on [M-AI-IMAGE](../../implemented/v0_10_0/m-ai-image-generation.md) (v0.10.0, the `callImage`/`callImageBase64` surface) and [M-AI-OPENROUTER-PROVIDER](../../implemented/v0_16_0/m-ai-openrouter-provider.md) (v0.16.0, the OpenRouter adapter whose image refusal this doc removes)

## Problem Statement

`internal/ai/openrouter/client.go` refuses every image-generation request before
it reaches the network:

```go
// internal/ai/openrouter/client.go:131-138 (v0.51.0)
if ai.RequestsImage(req) {
    return nil, ai.NewProviderError(
        "openrouter", 0,
        "image generation not supported by openrouter (use a Gemini image model)",
        nil,
    )
}
```

That refusal was a deliberate v0.16.0 scope cut, not a protocol fact:
[M-AI-OPENROUTER-PROVIDER](../../implemented/v0_16_0/m-ai-openrouter-provider.md)
recorded "OpenRouter web search / image generation features — Their preview
features … deferred. Text-in / text-out only" with "Future when needed" as the
revisit condition. **It is now needed, and the protocol now exists.** OpenRouter's
Chat Completions API (the exact endpoint this adapter already calls,
`/api/v1/chat/completions`) accepts a `modalities` request parameter
(enumerated values `text`, `image`, `audio`) and returns generated images inline
in `choices[].message.images[]` as base64 data URLs (Verification Log V1–V3).

**Who is blocked.** Stapledon's Voyage (D-8 model-neutral routing decision,
Mark, 2026-10-02) wants **text and portraits through one OpenRouter key for
cost** — one billing plane, one routing policy, one observatory trace. Today
portraits must go Gemini-direct: an AILANG program bound to an OpenRouter
handler (`--ai openrouter/z-ai/glm-5.3`-style) cannot generate an image at all;
the operator must run a second, Gemini-bound handler with a separate
`GOOGLE_API_KEY`/ADC lane, splitting billing and routing exactly the way D-8
was meant to stop.

**Second blocker, same surface (ailang#1496).** Even with the refusal gone,
image calls cannot name a model. `Handler.CallImage`/`CallImageBase64`
(`internal/ai/handler.go:237-292`) always send `h.model` — the single model the
handler was constructed with — and `parseImageOptions`
(`internal/ai/handler.go:295-315`) parses only `aspect_ratio` and `mime_type`
(V6). The multi-turn `step()` path solved this exact problem with an explicit
per-call model plus a `ModelResolver` that resolves friendly names and rejects
cross-provider routing with `ai.CodeModelNotAllowed`
(`cmd/ailang/ai_handlers.go:makeModelResolver`, `internal/effects/ai.go:resolveModel`).
The image path has no equivalent, so a handler bound to a text model could not
ask for an image model per portrait even once the refusal is removed.

**Current State:**
- OpenRouter adapter: image requests rejected client-side, never reach the API
  (V4).
- The only image-capable provider today is Gemini-direct
  (`internal/ai/gemini/generate.go:59-61` maps `ResponseModalities` to
  `generationConfig.responseModalities`; `:100-117` decodes `InlineData` parts
  into `resp.ImageData`/`resp.ImageMIME`). `anthropic`, `openai`, `ollama` and
  `configdriven` all reject image requests loudly (V5).
- `std/ai`'s `callImage`/`callImageBase64` doc comments say "Calls an image
  generation model (e.g., Gemini gemini-2.5-flash-image)"
  (`std/ai.ail:362-375`) — the doc surface is Gemini-only too.
- Image-output models exist behind one OpenRouter key today: `/api/v1/models`
  lists e.g. `google/gemini-3-pro-image`, `google/gemini-3.1-flash-image`,
  `openai/gpt-5-image`, `openai/gpt-5-image-mini` with
  `output_modalities: ["image","text"]` (V2).

**Impact:**
- Voyage (D-8) and every other AILANG consumer that wants images plus text
  cannot run on one OpenRouter key; per-call portraits are impossible on any
  provider.
- Cost visibility (Axiom A9) splits across two providers/keys — image spend
  never lands in the OpenRouter `usage.cost` plane the observatory already
  ingests.

## Goals

**Primary Goal:** Let `AI.callImage`/`AI.callImageBase64` generate images through
an OpenRouter-bound handler using the same Chat Completions call path, with an
optional per-call model so one program can keep its text model bound and name an
image model per portrait.

**Success Metrics:**
- `callImage`/`callImageBase64` succeed against an OpenRouter image-output model
  (e.g. `google/gemini-3-pro-image`) through the existing OpenRouter handler —
  one key, one billing plane, `usage.cost` visible as today.
- Text-only OpenRouter requests marshal byte-identically to before the change
  (the wire-shape property `correlation_test.go` defends).
- `{"model": "..."}` in the options JSON overrides the bound model for that
  image call only, resolved through the same `ModelResolver` semantics as
  `step()`; absent key = today's behavior exactly.
- A per-call model that resolves to a different provider fails loudly with
  `ai.CodeModelNotAllowed`, mirroring `step()`.
- Requests to non-image models surface the upstream OpenRouter error as a typed
  `ProviderError` — no silent fallback, no client-side allowlist drift.
- Existing Gemini-direct image generation is untouched (regression: the
  `examples/runnable/ai_image_generation.ail` flow still runs with
  `--ai gemini-2.5-flash-image`).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Use the Chat Completions `modalities` path, not OpenRouter's dedicated `/api/v1/images` API | One code path, one billing/correlation plane (D-8's cost goal), zero new endpoint plumbing; the dedicated Image API is a separate surface with its own discovery/params (Future Work) | human (D-8 context) | design | med |
| Per-call model rides the existing options JSON string (`{"model": "..."}`), not a new builtin signature or `AIHandler` interface method | Keeps the `callImage(prompt, output_path, options)` / `callImageBase64(prompt, options)` signatures and `effects.AIHandler` interface frozen; matches the `callJson` options-as-JSON precedent | agent | design | low |
| Per-call model resolution reuses the step() `ModelResolver` (friendly names + bound-provider enforcement → `CodeModelNotAllowed`) | One resolver, one enforcement point, identical error semantics across step/image; prevents accidental cross-provider routing under D-8's one-key model | agent | design | med |
| No client-side image-model allowlist — OpenRouter is the authority; a non-image model surfaces the upstream 4xx as a typed `ProviderError` | An allowlist drifts as the catalog changes and re-creates the very refusal this doc removes; fail loud per Critical Principle 2 | agent | design | low |
| `message.images` entries that are not `data:` URLs fail loudly (typed error), never a silent fetch | A remote-URL fetch is an unrequested second network hop with its own auth/timeout semantics; explicit is better (A11) | agent | design | low |
| `aspect_ratio` maps to the verified wire field `image_config.aspect_ratio`; `mime_type` maps to `image_config.output_format` (`image/png`→`png`) | Expresses options on the wire instead of dropping or prompt-hacking them; no silent fallback of a caller-specified option | agent | design | low |
| OpenRouter `Step`/`StreamStep` reject `ResponseModalities` containing `IMAGE` loudly | Today Step silently ignores modalities and would return text; under the new regime that is a silent fallback (Critical Principle 2) | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Chat Completions `modalities` path chosen (D-8 context; dedicated Image API
  deferred to Future Work)
- [x] Per-call model via options JSON key, resolved through the existing
  `ModelResolver` — no interface growth
- [ ] Human approval of this design doc (the normal Feature gate: this doc →
  approval → sprint-planner)

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact shape of the response-side image structs in
  `internal/ai/openrouter/types.go` — the doc pins the wire contract
  (`choices[].message.images[].image_url.url`, V1); whether to model the choice
  as a composed `openai.ChatChoice` extension or a local struct is free as long
  as `correlation_test.go`'s byte-identity property holds for text requests.
- Multiple returned images: the doc pins "first image wins" + an OTEL span
  attribute `ai.image_count`; whether to also note the count in the error path
  is the implementer's call.
- Whether `image_config` also forwards the `quality`/`size` keys if callers put
  them in options (the options JSON currently defines only `aspect_ratio`,
  `mime_type`, and the new `model`); unknown keys in options stay silently
  ignored as today (that is the current, tested behavior of
  `parseImageOptions`) — extending the forwarded set is allowed but must be
  documented in `std/ai.ail`.
- Test file placement: new `internal/ai/openrouter/images_test.go` vs extending
  `client_test.go` — implementer's choice, one golden-wire test per request
  shape is mandatory either way.

## Solution Design

### Overview

Two coordinated changes, one per blocker:

1. **OpenRouter image output (Generate path).** Remove the client-side refusal;
   when `ai.RequestsImage(req)` is true, the existing `generateChat` request
   gains two OpenRouter extension fields — `modalities: ["image","text"]` and
   (when the caller passed options) `image_config` — and the response decoder
   harvests `choices[0].message.images[]` data URLs into
   `resp.ImageData`/`resp.ImageMIME`. Everything else (routing-policy
   translation, reasoning, correlation, usage/cost plumbing, spans) is reused
   unchanged.

2. **Per-call model choice for image calls (all providers).** `ai.ImageOptions`
   gains a `Model` field parsed from the options JSON `"model"` key.
   `Handler.CallImage`/`CallImageBase64` use it in place of `h.model` when set.
   `effects.AIContext.CallImage`/`CallImageBase64` run the model string through
   the injected `ModelResolver` first (same as `Step`), so friendly names
   resolve to api_names and cross-provider requests fail with
   `CodeModelNotAllowed`.

### Architecture

**Component 1 — OpenRouter wire extension (`internal/ai/openrouter`).**

- `types.go`: `chatRequest` gains
  `Modalities []string \`json:"modalities,omitempty"\`` and
  `ImageConfig map[string]any \`json:"image_config,omitempty"\``. Both are
  OpenRouter extensions on the *composed* `chatRequest` — deliberately NOT on
  `openai.ChatRequest`, which has neither field today (V7) — so an unset
  request keeps marshaling byte-identically to the plain OpenAI body, the
  property `correlation_test.go` defends.
- `types.go`: response side gains the assistant-images shape. Per the verified
  OpenRouter schema (V1): each item is
  `{ "image_url": { "url": "<URL or data:...;base64,...>" } }` (required:
  `image_url.url`; description "Generated images from image generation
  models"). Since `openai.ChatMessage` is `{Role, Content}` only (V7), the
  response choice must be modeled locally — e.g. a `chatChoice` with an
  extended message carrying `Images []chatAssistantImage`.
- `chat.go` (`generateChat`): when `ai.RequestsImage(req)`:
  - `apiReq.Modalities = []string{"image", "text"}` — lowercase per the
    OpenRouter enum; note `ai.Request.ResponseModalities` uses the Gemini-style
    uppercase `"IMAGE"` (`provider.go:430-436`), so the adapter translates
    rather than forwards.
  - `apiReq.ImageConfig` from `req.ImageOptions`: `aspect_ratio` →
    `aspect_ratio`; `mime_type` → `output_format` (`image/png`→`png`,
    `image/jpeg`→`jpeg`).
  - After the existing decode: harvest
    `result.Choices[0].Message.Images`. First image wins; its `url` must start
    with `data:` — strip the `data:<mime>;base64,` prefix, decode into
    `resp.ImageData`, set `resp.ImageMIME` from the prefix (defaulting
    `image/png` like the handler does). Zero images → typed `ProviderError`
    ("openrouter: model %q returned no image data") — the Handler layer already
    errors on `resp.ImageData == nil`, but the adapter message names the model
    and modality request, which is the diagnosable form. A non-`data:` URL →
    typed `ProviderError` naming the model and stating remote URLs are not
    supported by this adapter. Record `ai.image_count` on the OTEL span.
  - Text in the same response (`message.content`) flows into `resp.Text` as
    today — image-output models routinely caption.
- `client.go`: delete the `RequestsImage` refusal; `Generate` dispatches image
  requests to the same `generateChat` (after reasoning resolution — image
  models are not reasoning-controlled, and `ResolveReasoning` fails loud on
  invalid input regardless). Update the `Client` doc comment, which currently
  asserts "Image generation is not supported".
- `step.go`: add the loud guard this path never had:
  `if ai.RequestsImage(req) { return typed error "image output not supported on the Step path; use callImage" }` (V8 for the negative existence).

**Component 2 — Per-call model (`internal/ai` + `internal/effects`).**

- `provider.go`: `ImageOptions` gains
  `Model string \`json:"model"\`` — "per-call model override; api_name or a
  models.yml-resolvable friendly name; empty = the handler's bound model".
- `handler.go`: `parseImageOptions` parses `"model"`; `CallImage` and
  `CallImageBase64` compute `model := h.model; if opts != nil && opts.Model != ""
  { model = opts.Model }` and send `Model: model` on the `ai.Request` (keeping
  `RequestedModel` semantics: the adapter reports `RequestedModel: req.Model`
  already, so routing metadata stays truthful).
- `effects/ai.go`: `AIContext.CallImage`/`CallImageBase64` extract the model key
  before dispatch (the `ai` package must expose the parse — e.g. export
  `ParseImageOptions`, today an unexported `parseImageOptions`, V6), run it
  through `c.resolveModel` — the SAME resolver `Step` uses
  (`internal/effects/ai.go:170-177`, wired in
  `cmd/ailang/ai_handlers.go:makeModelResolver`) — and hand the handler either
  an options string carrying the resolved api_name or (implementer's choice, see
  Deferred Decisions) dispatch through a small exported rewrite helper. A nil
  resolver or empty model key = passthrough, preserving today's behavior for
  direct `ai.Handler` users and tests.
- `cmd/ailang/ai_handlers.go`: no change required — `setupAIHandlerFromConfig`
  already attaches the resolver via `WithModelResolver` (V9).

**Component 3 — surface docs.**

- `std/ai.ail:362-375`: update the `callImage`/`callImageBase64` doc comments —
  options JSON gains `"model"`, provider set is no longer "e.g., Gemini".
- `examples/runnable/ai_image_generation.ail`: add a per-call-model variant
  (options carrying `{"model": "google/gemini-3-pro-image"}`) alongside the
  existing Gemini-direct usage lines.
- `changelogs/v0.32-current.md`: entry under v0.51.1.

### What does NOT change

- The `effects.AIHandler` interface and both builtin signatures —
  `callImage(prompt: string, output_path: string, options: string) -> string ! {AI, FS}`
  and `callImageBase64` keep their types.
- The Gemini-direct image path (`generate.go`) — untouched.
- `anthropic`/`openai`/`ollama`/`configdriven` image refusals — still loud, still
  correct (their APIs don't expose these models behind one chat call).
- Routing-policy translation, correlation, broadcast, reasoning, usage/cost
  parsing in the OpenRouter adapter.
- The `"IMAGE"` value convention on `ai.Request.ResponseModalities`.

### Implementation Plan

**Phase 1: OpenRouter image output** (~1 day)
- [ ] `types.go`: `Modalities`, `ImageConfig` request fields; response
      `chatChoice`/message extension with `Images`
- [ ] `chat.go`: modality translation + `image_config` mapping + image harvest
      (data-URL decode, first-image, typed errors, `ai.image_count` span
      attribute)
- [ ] `client.go`: remove refusal, update doc comment
- [ ] `step.go`: loud `RequestsImage` guard
- [ ] `images_test.go` (new): golden wire body (`modalities` present and
      lowercase, `image_config` mapped), byte-identity for text-only requests,
      data-URL decode, zero-images error, remote-URL error, multi-image
      first-wins, Step guard
- [ ] Rewrite `TestClient_Generate_RejectsImageRequests`
      (`client_test.go:297-323`) into a positive dispatch test — this is the one
      intentional test-text change; the test currently asserts the server "should
      not be called" and the "image generation not supported" message (V10)

**Phase 2: Per-call model choice** (~1 day)
- [ ] `provider.go`: `ImageOptions.Model`
- [ ] `handler.go`: parse + coalesce in both Call methods
- [ ] `effects/ai.go`: resolver pass-through in `CallImage`/`CallImageBase64`
      (export `ParseImageOptions` from `ai`)
- [ ] Tests: `internal/ai` handler-level (override applies; absent key =
      `h.model`; RequestedModel truthful) and `internal/effects` resolver-level
      (friendly name resolves; cross-provider → `CodeModelNotAllowed`; nil
      resolver passthrough; stub handler unchanged)

**Phase 3: docs + regression sweep** (~0.5 day)
- [ ] `std/ai.ail` comments, example update, changelog entry
- [ ] `make test-core`, `make fmt`, `make lint`, `make check-boundaries`
- [ ] Manual smoke (if a key is available): one portrait via
      `--ai openrouter/<text-model>` + per-call `google/gemini-3-pro-image`,
      checking the observatory span carries `modalities` in `rawRequest` and
      `usage.cost`

### Files to Modify/Create

**New files:**
- `internal/ai/openrouter/images_test.go` — wire-shape + decode + error tests (~250 LOC)

**Modified files:**
- `internal/ai/openrouter/client.go` — remove refusal, doc comment (~−10 LOC)
- `internal/ai/openrouter/chat.go` — modality translation, image_config, image harvest (~+70 LOC)
- `internal/ai/openrouter/types.go` — request/response extension fields (~+35 LOC)
- `internal/ai/openrouter/step.go` — loud image guard (~+6 LOC)
- `internal/ai/openrouter/client_test.go` — rewrite the rejection test into a dispatch test (~±40 LOC)
- `internal/ai/provider.go` — `ImageOptions.Model` (~+4 LOC)
- `internal/ai/handler.go` — parse + coalesce + export parse helper (~+20 LOC)
- `internal/effects/ai.go` — resolver pass-through in both Call methods (~+25 LOC)
- `internal/effects/ai_test.go` — resolver tests (~+60 LOC)
- `std/ai.ail` — doc comments (~+8 LOC)
- `examples/runnable/ai_image_generation.ail` — per-call-model variant (~+10 LOC)
- `changelogs/v0.32-current.md` — v0.51.1 entry (~+10 LOC)

## Examples

### Example 1: Voyage portraits through the one OpenRouter key (D-8)

**Before** (today, two providers/two keys):
```console
$ ailang run --caps AI,FS,IO --ai gemini-2.5-flash-image --entry main voyage_portraits.ail
# text calls elsewhere in the same program need a SECOND, OpenRouter-bound handler;
# a handler bound to openrouter/z-ai/glm-5.3 rejects callImage outright:
#   "image generation not supported by openrouter (use a Gemini image model)"
```

**After:**
```ailang
-- handler bound once: ailang run --caps AI,FS,IO --ai openrouter/z-ai/glm-5.3 ...
-- text calls use the bound model; each portrait names an image model per call:
let portraitPath =
  AI.callImage("portrait of the navigator, warm rim light",
               "img/navigator.png",
               "{\"model\": \"google/gemini-3-pro-image\", \"aspect_ratio\": \"1:1\"}")
```
The request goes out over the same `/chat/completions` call with
`"modalities": ["image","text"]`, one key, one `usage.cost` line in the
observatory plane.

### Example 2: options stay backwards compatible

**Before/After (identical behavior):** `AI.callImageBase64(prompt, "{}")` — no
`model` key means the handler's bound model, exactly as v0.10.0–v0.51.0.

### Example 3: cross-provider per-call model fails loud, same as step()

```ailang
-- bound to --ai openrouter/z-ai/glm-5.3:
AI.callImage(prompt, out, "{\"model\": \"gemini-2.5-flash-image\"}")
-- AIError CodeModelNotAllowed: per-call model "gemini-2.5-flash-image" resolves to
-- provider "google", but the bound --ai handler is "openrouter"; use --ai to switch.
```

## Success Criteria

- [ ] `openrouter.Generate` with `ResponseModalities: ["IMAGE"]` reaches the
      test server (no client-side refusal) and the request body contains
      `"modalities":["image","text"]` — golden JSON assertion (AC-1)
- [ ] A response carrying `choices[0].message.images[0].image_url.url =
      "data:image/png;base64,..."` yields non-nil `resp.ImageData` with correct
      decoded bytes and `resp.ImageMIME = "image/png"`; through the Handler,
      `CallImage` writes the file and `CallImageBase64` returns
      `{"base64":...,"mime_type":...}` (AC-2)
- [ ] A text-only OpenRouter request body is byte-identical to pre-change
      output (no `modalities`, no `image_config` keys) (AC-3)
- [ ] `aspect_ratio` lands as `image_config.aspect_ratio` and `mime_type` as
      `image_config.output_format` on the wire (AC-4)
- [ ] Zero images in the response → typed `ProviderError` naming the model;
      remote-URL image → typed `ProviderError`; `Step`/`StreamStep` with image
      modalities → typed error (AC-5)
- [ ] Options `{"model": M}` overrides the request model; `{}` keeps `h.model`;
      resolver applies (friendly → api_name) and cross-provider M fails with
      `ai.CodeModelNotAllowed` (AC-6)
- [ ] `examples/runnable/ai_image_generation.ail` still type-checks and the
      Gemini-direct flag form in its header comment remains accurate (AC-7)
- [ ] `make test-core`, `make lint`, `make check-boundaries` green; `make fmt`
      clean (AC-8)
- [ ] `std/ai.ail` comments + changelog updated (AC-9)

## Conflict Surface

This change touches `internal/ai/*` and `internal/effects/ai.go` — not the
parser/typechecker core — but the shared surfaces below carry real conflict
risk, so they are enumerated per the skill's requirement.

**Positions extended:**
1. `ai.Request.ResponseModalities` consumers: every provider's `Generate` gate
   (`ai.RequestsImage`). OpenRouter flips from reject→support; the other four
   providers keep their rejections.
2. The image options JSON string (`callImage`/`callImageBase64` third/second
   parameter): gains a `"model"` key. Existing keys: `aspect_ratio`,
   `mime_type`.
3. `internal/ai/openrouter/chatRequest` JSON body: gains `modalities` +
   `image_config` — the same wire body every OpenRouter text call, broadcast
   correlation, and golden/byte-identity test marshals.
4. `effects.AIContext` per-call dispatch: gains image-side model resolution
   alongside the existing step-side resolution.

**What else lives there:**
1. `configdriven` also gates on `ai.RequestsImage` via
   `capabilities.vision` (`internal/ai/configdriven/provider.go:87-93`) —
   unaffected; its gate stays.
2. `internal/ai/gemini` consumes `ImageOptions.AspectRatio`/`MIMEType`
   differently (generationConfig); the new `Model` field must be ignored
   gracefully there — it is, because per-call model resolution happens in the
   Handler/effects layers before Gemini sees `Request.Model`.
3. `mission/quorum/call.go` constructs handlers directly via `ai.NewHandler`
   with no resolver — the passthrough path must keep working (no resolver = raw
   model string honored).
4. Wire-body consumers: `correlation_test.go` asserts byte-identical marshaling
   for requests without the new fields; Broadcast observatory spans carry
   `rawRequest` — a `modalities` key will now appear there for image calls only.

**Disambiguation:** none needed at the language level — no parser or type
surface changes. At the provider level, "does this provider support image
output" remains a per-`Generate` runtime fact, decided by each adapter's gate
exactly as today.

**Programs that MUST still work (regression fixtures, all verified to exist —
V10, V11):**
1. `examples/runnable/ai_image_generation.ail` — the existing Gemini-direct
   image flow (`--ai gemini-2.5-flash-image`), runnable per its header.
2. `std/ai.ail:368-375` — `callImage`/`callImageBase64` exported surface and
   effect rows (`{AI, FS}` / `{AI}`).
3. `internal/ai/openrouter/correlation_test.go` — byte-identical text-request
   bodies (must stay green with zero changes).
4. `internal/ai/gemini/generate_finishreason_test.go` + the Gemini
   `ResponseModalities` → `generationConfig` mapping — untouched path.
5. `internal/effects/ai_step_test.go` — step-side model resolution semantics,
   which the new image-side resolution must mirror, not alter.

**What deliberately changes:**
- `TestClient_Generate_RejectsImageRequests` — rewritten from refusal to
  dispatch; its old assertion text ("image generation not supported") dies with
  the refusal. This is the single intentional test change; every other listed
  test must pass unmodified.
- The OpenRouter `Client` doc comment that promises "Image generation is not
  supported; callers should use a Gemini image model".

## Testing Strategy

**Unit tests** (`internal/ai/openrouter/images_test.go`, new):
- Golden request body: `ResponseModalities:["IMAGE"]` → body contains
  `"modalities":["image","text"]` (exact JSON, lowercase) and nothing else new.
- Byte-identity: text-only request body has no `modalities`/`image_config` keys —
  assert against a literal expected body string so key-order regressions also
  surface.
- `image_config` mapping: `aspect_ratio:"16:9"` + `mime_type:"image/png"` →
  `{"aspect_ratio":"16:9","output_format":"png"}`.
- Response decode: one image (PNG data URL) → ImageData/ImageMIME; two images →
  first wins + span attr; `message.content` still lands in `resp.Text`.
- Errors: zero choices / zero images / remote URL / Step-with-modalities — all
  typed `ProviderError`s with model-naming messages.

**Unit tests** (`internal/ai` handler + `internal/effects`):
- `parseImageOptions` parses `"model"`; coalesce logic (override vs bound
  default) in `CallImage`/`CallImageBase64` via a stub provider.
- `AIContext` with resolver: friendly name → api_name rewrite; cross-provider →
  `CodeModelNotAllowed`; nil resolver → passthrough; no model key → no
  resolver call.

**Integration/manual:**
- If a live key is available in the sprint environment: one real portrait call
  via `google/gemini-3-pro-image`, then check the observatory span's
  `rawRequest` shows `modalities` and `usage.cost` is non-zero. Otherwise the
  httptest golden bodies are the contract and the live check is a release-notes
  TODO.

## Non-Goals

**Not attempted in this feature:**
- TTS / audio output — separate workstream (ailang#1495).
- OpenRouter's dedicated `/api/v1/images` API (model discovery, `n` up to 10,
  `input_references`, SSE partial images) — the D-8 ask is satisfied by the
  chat path; the Image API is a candidate Future Work follow-up if per-model
  parameter depth is ever needed.
- Streaming image output on Step/StreamStep — image output is Generate-path
  only; Step rejects it loudly.
- Fetching remote-URL images — typed error instead (see High-Impact Decisions).
- Config-driven (`[[ai_provider]]`) image support — their `capabilities.vision`
  gate is unchanged.
- Multiple-image return (`n>1`) beyond first-wins + count telemetry.
- Any parser/typechecker surface change — none exists.

## Timeline

**Week 1** (~2.5 days at 2× the naive estimate):
- Day 1: Phase 1 (adapter + wire tests)
- Day 2: Phase 2 (per-call model + resolver tests)
- Day 3 (half): Phase 3 (docs, changelog, lint/boundaries, smoke)

**Total: ~2.5 focused days**, one sprint.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| OpenRouter changes/retires the chat `modalities` path (it sits next to a newer dedicated Image API) | Med | The wire contract is pinned by golden-body tests; a break surfaces as a typed upstream error, not silent text. The refusal this doc removes was itself a snapshot of 2026-era OpenRouter; the tests now document the live contract. |
| Image-output models charge per image, not per token — runaway cost under D-8's one key | Med | `usage.cost` already flows to the observatory plane (A9); routing-policy `PriceCap` translation is unchanged and applies to image calls; consider a follow-up per-call image price cap. |
| `image_config` is documented as "provider-specific" — some models reject unknown keys | Med | We forward only `aspect_ratio`/`output_format` and only when the caller set them; upstream 4xx surfaces typed. |
| Options-JSON "model" collides with a future richer options schema | Low | Key is documented in `std/ai.ail` now; the options string is versioned by convention the same way `callJson`'s schema is. |
| Per-call model on Gemini-direct handler bypasses models.yml (no resolver in direct path) | Low | Passthrough semantics identical to step()'s pre-resolver behavior; `makeModelResolver`'s unknown-name branch already treats raw api_names as legal. |

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Same request → same wire body (golden-tested); first-image-wins is a deterministic rule |
| A2: Replayability | +1 | Full request/response already banked via Broadcast correlation; image calls join the same trace |
| A3: Effect Legibility | +1 | No new effects; `callImage` keeps `{AI, FS}`, `callImageBase64` keeps `{AI}` |
| A4: Explicit Authority | +1 | Cross-provider per-call models rejected loudly, mirroring step(); no ambient provider switch |
| A5: Bounded Verification | 0 | Provider behavior is not locally verifiable beyond wire-shape tests |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Removes a hand-written workaround (second Gemini handler) that human operators had to reason about; errors stay machine-parseable typed AIErrors |
| A8: Minimal Syntax | +1 | Zero language/builtin surface growth; options JSON key only |
| A9: Cost Visibility | +1 | The core D-8 motivation: image spend lands in the OpenRouter `usage.cost` plane instead of a second provider's billing |
| A10: Composability | +1 | Reuses Resolver, routing policy, correlation, spans; new code is two extension fields + one decode path |
| A11: Structured Failure | +1 | Every new failure (no images, remote URL, non-image model, cross-provider model) is a typed error naming the model |
| A12: System Boundary | 0 | Boundary unchanged — same HTTP egress point, same key |

**Net Score: +9** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): not optimizing for human convenience over machine analysis

## Verification Log

Every load-bearing claim, with the command or read that proves it. Environment:
repo at `4460d91b` (dev), version `v0.51.0` (`std/VERSION`).

| # | Claim | Verification |
|---|---|---|
| V1 | OpenRouter Chat Completions accepts a request `modalities` parameter, enum `text`/`image`/`audio`, example `[text, image]` | Fetched `https://openrouter.ai/docs/api/api-reference/chat/create-a-chat-completion.md` (2026-10-02): "Output modalities for the response. Supported values are 'text', 'image', and 'audio'." with `example: [text, image]` |
| V2 | Image-output models are served by OpenRouter behind one key | `curl https://openrouter.ai/api/v1/models` → `google/gemini-3-pro-image`, `google/gemini-3.1-flash-image`, `openai/gpt-5-image`, `openai/gpt-5-image-mini`, `openai/gpt-5.4-image-2` all list `output_modalities: ['image','text']` |
| V3 | Images come back as base64 in `message.images` | Same API reference: assistant message has `images: ChatAssistantImages`, described "Generated images from image generation models", items `{ image_url: { url } }`, example `data:image/png;base64,iVBORw0KGgo...`; `image_url.url` = "URL or base64-encoded data" |
| V4 | The OpenRouter adapter refuses image requests client-side at v0.51.0 | Read `internal/ai/openrouter/client.go:131-138` (quoted in Problem Statement) |
| V5 | Every other provider rejects image requests loudly today | `grep -n "RequestsImage" internal/ai/{gemini,anthropic,openai,ollama}/client.go internal/ai/configdriven/provider.go` → gemini has no gate (supports), anthropic `:196`, openai `:82`, ollama `:120`, configdriven `:90` all return typed errors |
| V6 | Image calls have no per-call model today; options parse only aspect_ratio/mime_type | Read `internal/ai/handler.go:237-315`: both Call methods send `Model: h.model`; `parseImageOptions` unmarshal struct has exactly `aspect_ratio`, `mime_type` fields |
| V7 | `openai.ChatRequest` has no `Modalities` and `openai.ChatMessage` has no `Images` (negative existence — the extension must live on openrouter's composed structs) | Read `internal/ai/openai/types.go:8-20` (ChatRequest fields: Model, Messages, MaxTokens, MaxCompletionTokens, Temperature, Seed, ResponseFormat, ReasoningEffort — no Modalities) and `:36-39` (ChatMessage = Role, Content only); `grep -i modalit internal/ai/openai/` → 0 hits |
| V8 | OpenRouter Step has no image guard (negative existence — the silent-fallback claim) | `grep -rn "RequestsImage" internal/ai/openrouter/step.go` → 0 hits (rc=1); Step builds from Messages/Tools only, so a Request with `ResponseModalities:["IMAGE"]` would return text silently today |
| V9 | Per-call model machinery exists for step() and is wired from the CLI | Read `internal/effects/ai.go:112-177` (ModelResolver contract, `WithModelResolver`, `resolveModel` empty-model/nil-resolver passthrough) and `cmd/ailang/ai_handlers.go:makeModelResolver` (bound-provider check → `ai.CodeModelNotAllowed`); `setupAIHandlerFromConfig` attaches `.WithModelResolver(makeModelResolver(...))` |
| V10 | The test pinning the OLD behavior exists and what it actually asserts | Read `internal/ai/openrouter/client_test.go:297-323` (`TestClient_Generate_RejectsImageRequests`): asserts the test server is never called (`t.Fatalf("server should not be called")`) and the message contains "image generation not supported" |
| V11 | Regression fixtures exist | `ls`: `examples/runnable/ai_image_generation.ail`, `std/ai.ail`, `internal/ai/openrouter/correlation_test.go`, `internal/ai/gemini/generate_finishreason_test.go`, `internal/effects/ai_step_test.go` — all present; read `examples/runnable/ai_image_generation.ail` header (Gemini-direct usage line) and `std/ai.ail:362-375` (exported functions + doc comments quoted in this doc) |
| V12 | Byte-identity of text-only bodies is an existing, tested property | Read `internal/ai/openrouter/types.go:1-14` package comment ("a request with none of them set marshals byte-identically to a plain OpenAI body — the property the golden-body tests defend") and `correlation_test.go:15` ("marshal byte-identically to one built before these fields existed") |
| V13 | `aspect_ratio`/`output_format` are expressible on the chat wire via `image_config` | Same API reference as V1: request field `image_config` ($ref ImageConfig), "Provider-specific image configuration options", example `{aspect_ratio: '16:9', quality: high}`; the server-tool config section enumerates "all image_config params (aspect_ratio, quality, size, background, output_format, output_compression, moderation, etc.)" |
| V14 | The refusal was a deliberate v0.16.0 deferral, now revisitable | Read `design_docs/implemented/v0_16_0/m-ai-openrouter-provider.md`: "OpenRouter web search / image generation features — Their preview features … deferred. Text-in / text-out only." and the non-goals table "Future when needed" |
| V15 | models.yml has no image model entry today (Gemini-direct image runs via the GuessProvider fallback path) | `grep -i image internal/modelreg/models.yml` → 0 hits; read `internal/ai/config.go:48-100` (GuessProvider: "google/…" vendor prefix → OpenRouter; bare "gemini…" → Google direct) |
| V16 | No new error code is proposed (namespace check n/a) | This doc introduces no `MOD/PAR/TC/EFF` code; it reuses existing typed errors (`ProviderError`, `ai.CodeModelNotAllowed`, `ai.CodeSchemaValidation`); `grep` of proposed messages shows they carry model names, not codes |
| V17 | Related-doc duplicate gate: no existing/planned doc covers OpenRouter image output | `ailang docs search --stream implemented/planned --limit 5 "openrouter image output"` → top hits are M-AI-IMAGE (v0.10.0, the original Gemini-only callImage surface — this doc extends it, no overlap in the OpenRouter routing ask) and M-AI-OPENROUTER-PROVIDER (v0.16.0, whose deferral this doc discharges, V14); no hit ≥ 0.65 neural on this topic |

## Related Documents

<!-- Auto-populated by search on "openrouter image output" (SimHash; neural unavailable — embeddings backend fell back to SimHash, 2026-10-02) -->

**Implemented (may inform design):**
- [design_docs/implemented/v0_10_0/m-ai-image-generation.md](../../implemented/v0_10_0/m-ai-image-generation.md) (0.95) — the `callImage`/`callImageBase64` surface, options-as-JSON precedent, "unsupported providers return clear errors" principle
- [design_docs/implemented/v0_16_0/m-ai-openrouter-provider.md](../../implemented/v0_16_0/m-ai-openrouter-provider.md) (0.90) — the adapter this doc extends; recorded the image-feature deferral this doc discharges

**Planned (check for overlap):**
- [design_docs/planned/v0_38_0/m-openrouter-eu-routing.md](../../planned/v0_38_0/m-openrouter-eu-routing.md) (0.95) — EU in-region routing; orthogonal (region pinning, not modalities)

## References

- **D-8 model-neutral routing decision** (Mark, attended 2026-10-02): text AND portraits through one OpenRouter key for cost — the motivating directive for this doc
- **Issues**: ailang#1496 (per-call model choice), ailang#1495 (TTS — explicitly separate), this task's GitHub issue (OpenRouter image output)
- **Wire contract**: [OpenRouter Chat Completions API reference](https://openrouter.ai/docs/api/api-reference/chat/create-a-chat-completion.md) (`modalities`, `image_config`, `ChatAssistantImages`; fetched 2026-10-02), [Image Generation guide](https://openrouter.ai/docs/guides/overview/multimodal/image-generation.md)
- **Prior art in this repo**: [M-AI-IMAGE](../../implemented/v0_10_0/m-ai-image-generation.md) (v0.10.0), [M-AI-OPENROUTER-PROVIDER](../../implemented/v0_16_0/m-ai-openrouter-provider.md) (v0.16.0), step()'s per-call model + resolver (`internal/effects/ai.go:112-177`)
- **Axiom reference**: [Design Axioms](/docs/references/axioms)

## Future Work

- **OpenRouter dedicated Image API** (`/api/v1/images`): model discovery,
  `n` up to 10, `input_references` (image-to-image), SSE partial images — a
  richer surface if per-model parameters ever outgrow `image_config`.
- **Per-call image price cap**: D-8's cost visibility would benefit from a
  routing-policy-style per-image price guard once real portrait spend is
  measurable in the observatory.
- **TTS / audio modality** (`modalities: ["audio"]`) — ailang#1495, explicitly
  out of scope here.
- **Remote-URL image fetch** (typed-error today) if an upstream ever returns
  hosted URLs by default.

---

**Document created**: 2026-10-02
**Last updated**: 2026-10-02
