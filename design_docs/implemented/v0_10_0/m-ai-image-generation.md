# M-AI-IMAGE: AI Image Generation via std/ai

**Status**: Planned
**Target**: v0.10.0
**Priority**: P1 (Medium)
**Estimated**: 3 days (10-12 hours implementation + tests + docs)
**Dependencies**: std/ai (v0.5.1 ✅), std/bytes (v0.3.13 ✅), std/fs (✅)
**Origin**: Agent message from `blog` agent (inbox `2a9d4677`) — needs banner image generation for product launches

## Related Documents

- [M-UNIFIED-AI-PROVIDERS](../../implemented/v0_5_10/m-unified-ai-providers.md) — Established the Provider interface, Request/Response types, Handler bridge
- [M-STRUCTURED-AI-OUTPUT](../../implemented/v0_7_3/m-structured-ai-output.md) — Added callJson/callJsonSimple, extended AIHandler interface, set pattern for new AI operations

---

## Problem Statement

AILANG's `std/ai` module provides three functions, all `string -> string`:

```ailang
export func call(input: string) -> string ! {AI}
export func callJson(input: string, schema: string) -> string ! {AI}
export func callJsonSimple(input: string) -> string ! {AI}
```

Image-generating AI models (e.g. Gemini `gemini-2.5-flash-image`) return binary PNG/JPEG data via `InlineData` parts with base64-encoded bytes. The current architecture has **no path for binary output** — `ai.Response` only carries `Text string`, and the `AIHandler` interface only returns `(string, error)`.

**Current State:**
- The Gemini provider already handles multimodal *input* (base64 inline data via `buildParts()`) but ignores `InlineData` in *response* parts — line 113 of `generate.go` only reads `part.Text`
- The `generationConfig` struct lacks `ResponseModalities` field needed to request image output
- No `callImage` or equivalent exists at any layer (builtin, effect, handler, provider)
- Blog agent must shell out to external Go binaries for image generation instead of writing `.ail` scripts

**Impact:**
- Blog product-launch skill cannot generate banner images natively
- Any AILANG application needing AI-generated images must escape to Go/shell
- Blocks future multimodal AI pipelines (audio, video would follow same pattern)

## Goals

**Primary Goal:** Enable AILANG scripts to generate images via AI models with a simple, type-safe API.

**Success Metrics:**
- `AI.callImage(prompt, path, options)` generates and saves an image in one call
- `AI.callImageBase64(prompt, options)` returns base64 for programmatic pipelines
- Gemini `gemini-2.5-flash-image` works end-to-end (the only provider with image gen)
- Unsupported providers return clear errors (no silent fallbacks)
- Example file runs successfully with `make verify-examples`
- Full test coverage on new code paths (stub handler + option parsing)

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Return base64 string (not bytes) from `callImageBase64` | Avoids adding bytes return to AIHandler interface; composes with existing `std/bytes.fromBase64` | human | design | high |
| `callImage` writes file directly (not separate AI + FS steps) | Determines whether AI builtins can have compound effects {AI, FS} | human | design | high |
| Options passed as JSON string (not typed record) | Matches existing `callJson` pattern; avoids blocking on optional-field records | agent | design | low |
| Gemini-only for v1 (no OpenAI DALL-E) | DALL-E uses a completely different API shape; defer to future milestone | human | design | low |
| Extend existing Provider interface (not new ImageProvider) | One Generate() method handles both text and image; keeps provider count at 4 | human | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Return type from `callImageBase64` is `string` (JSON with base64), not `bytes`
- [x] `callImage` has effect `{AI, FS}` — compound effects on builtins are acceptable
- [x] Extend existing `ai.Request`/`ai.Response` rather than creating new types
- [x] Gemini-only provider support for v1
- [ ] Options JSON schema: which fields are supported (aspect_ratio, size, mime_type)?

## Deferred Decisions

- Internal helper function naming — agent may choose
- Stub handler response format (minimal 1x1 PNG or larger test image) — agent may choose
- Whether `callImage` creates parent directories or requires they exist — agent may choose
- Error message wording for unsupported providers — agent may choose
- Test file organization (new file vs extend existing ai_test.go) — agent may choose

---

## Solution Design

### Overview

Add two new functions to `std/ai` following the established pattern from M-STRUCTURED-AI-OUTPUT. The call chain mirrors `callJson`:

```
AILANG: AI.callImage(prompt, path, options)  [std/ai.ail]
  -> _ai_call_image builtin                  [internal/builtins/ai.go]
  -> effects.Call(ctx, "AI", "callImage")    [internal/effects/ai.go]
  -> AIContext.CallImage(prompt, path, opts) [internal/effects/ai.go]
  -> Handler.CallImage(prompt, path, opts)   [internal/ai/handler.go]
  -> Provider.Generate(Request{ResponseModalities: ["IMAGE"]})
  -> Gemini API with responseModalities
  -> Response.ImageData extracted
  -> os.WriteFile(path, imageData)
  -> return path
```

### Architecture

**Key insight**: The existing `Provider` interface (`Generate(ctx, *Request) (*Response, error)`) is sufficient. We extend `Request` with `ResponseModalities` and `Response` with `ImageData`/`ImageMIME`. No new interfaces needed.

**Components:**
1. **Extended Request/Response** (`internal/ai/provider.go`): Add image fields to existing types
2. **Gemini image generation** (`internal/ai/gemini/`): Add `ResponseModalities` to config, extract `InlineData` from response parts
3. **AIHandler extension** (`internal/effects/ai.go`): Add `CallImage` and `CallImageBase64` to interface
4. **Handler bridge** (`internal/ai/handler.go`): Implement new methods, parse options, write file
5. **New builtins** (`internal/builtins/ai.go`): Register `_ai_call_image` and `_ai_call_image_base64`
6. **Effect operations** (`internal/effects/ai.go`): Register `callImage` and `callImageBase64` ops
7. **Stdlib wrapper** (`std/ai.ail`): Export `callImage` and `callImageBase64`

### New std/ai Functions

```ailang
-- callImage: Generate an image and save to file
-- Returns the output path on success.
-- Options as JSON: {"aspect_ratio": "16:9", "mime_type": "image/png"}
export func callImage(prompt: string, output_path: string, options: string) -> string ! {AI, FS} =
  _ai_call_image(prompt, output_path, options)

-- callImageBase64: Generate an image and return as base64 string
-- Returns JSON: {"base64": "...", "mime_type": "image/png"}
-- Use std/bytes.fromBase64 to decode if needed.
export func callImageBase64(prompt: string, options: string) -> string ! {AI} =
  _ai_call_image_base64(prompt, options)
```

### Options JSON

```json
{
  "aspect_ratio": "16:9",
  "mime_type": "image/png",
  "model": "gemini-2.5-flash-image"
}
```

All fields optional. Empty `"{}"` uses provider defaults.

### Go Implementation Changes

#### 1. Extend `ai.Request` and `ai.Response` (`internal/ai/provider.go`)

```go
type Request struct {
    // ...existing fields...
    ResponseModalities []string      // e.g., ["IMAGE"] for image generation
    ImageOptions       *ImageOptions // Parsed from options JSON
}

type ImageOptions struct {
    AspectRatio string // "16:9", "1:1", etc.
    MIMEType    string // "image/png", "image/jpeg"
}

type Response struct {
    // ...existing fields...
    ImageData []byte // Raw image bytes (PNG/JPEG) — nil for text responses
    ImageMIME string // e.g., "image/png" — empty for text responses
}
```

#### 2. Gemini Provider (`internal/ai/gemini/`)

Add `ResponseModalities` to `generationConfig`:
```go
type generationConfig struct {
    // ...existing fields...
    ResponseModalities []string `json:"responseModalities,omitempty"`
}
```

In `generateContent()`, extract image data from response parts:
```go
for _, part := range result.Candidates[0].Content.Parts {
    if part.InlineData != nil && part.InlineData.Data != "" {
        imageBytes, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
        if err != nil {
            return nil, ai.NewProviderError("gemini", 0, "failed to decode image data", err)
        }
        resp.ImageData = imageBytes
        resp.ImageMIME = part.InlineData.MimeType
    }
    text += part.Text
}
```

#### 3. AIHandler Interface Extension (`internal/effects/ai.go`)

```go
type AIHandler interface {
    Call(input string) (string, error)
    CallJson(input string, schema string) (string, error)
    CallImage(prompt string, outputPath string, options string) (string, error)
    CallImageBase64(prompt string, options string) (string, error)
}
```

#### 4. Handler Bridge (`internal/ai/handler.go`)

```go
func (h *Handler) CallImage(prompt, outputPath, options string) (string, error) {
    opts := parseImageOptions(options)
    resp, err := h.provider.Generate(context.Background(), &Request{
        Model:              h.model,
        SystemPrompt:       h.systemPrompt,
        UserPrompt:         prompt,
        ResponseModalities: []string{"IMAGE"},
        ImageOptions:       opts,
    })
    if err != nil { return "", err }
    if resp.ImageData == nil {
        return "", fmt.Errorf("provider returned no image data")
    }
    if err := os.WriteFile(outputPath, resp.ImageData, 0644); err != nil {
        return "", fmt.Errorf("failed to write image: %w", err)
    }
    return outputPath, nil
}

func (h *Handler) CallImageBase64(prompt, options string) (string, error) {
    opts := parseImageOptions(options)
    resp, err := h.provider.Generate(context.Background(), &Request{
        Model:              h.model,
        SystemPrompt:       h.systemPrompt,
        UserPrompt:         prompt,
        ResponseModalities: []string{"IMAGE"},
        ImageOptions:       opts,
    })
    if err != nil { return "", err }
    if resp.ImageData == nil {
        return "", fmt.Errorf("provider returned no image data")
    }
    b64 := base64.StdEncoding.EncodeToString(resp.ImageData)
    result := fmt.Sprintf(`{"base64":"%s","mime_type":"%s"}`, b64, resp.ImageMIME)
    return result, nil
}
```

### Provider Support Matrix

| Provider | Image Generation | Notes |
|----------|-----------------|-------|
| Gemini | Supported | `gemini-2.5-flash-image` with `ResponseModalities: ["IMAGE"]` |
| OpenAI | Future | DALL-E uses different API (`/v1/images/generations`) — separate milestone |
| Anthropic | N/A | Claude doesn't generate images |
| Ollama | N/A | Local models typically don't generate images |

Unsupported providers return: `error: image generation not supported by provider "anthropic" (model: claude-sonnet-4-6)`

### Files to Modify/Create

**Modified files:**
- `internal/ai/provider.go` (+25 LOC) — Add `ResponseModalities`, `ImageOptions`, `ImageData`, `ImageMIME`
- `internal/ai/handler.go` (+80 LOC) — Add `CallImage`, `CallImageBase64`, `parseImageOptions`
- `internal/ai/gemini/types.go` (+5 LOC) — Add `ResponseModalities` to `generationConfig`
- `internal/ai/gemini/generate.go` (+25 LOC) — Extract `InlineData` from response, pass modalities
- `internal/effects/ai.go` (+60 LOC) — Extend `AIHandler`, add `CallImage`/`CallImageBase64`, register ops, extend `StubAIHandler`
- `internal/builtins/ai.go` (+100 LOC) — Register `_ai_call_image`, `_ai_call_image_base64`
- `std/ai.ail` (+15 LOC) — Export `callImage`, `callImageBase64`
- `CHANGELOG.md` (+20 LOC) — Document new functions

**New files:**
- `examples/image_generation.ail` (~20 LOC) — Working example
- `internal/ai/handler_test.go` or extend existing (~80 LOC) — Unit tests for new methods

**Estimated total: ~430 LOC**

---

## Examples

### Before (shell out to Go binary)

```bash
# Must use external Go binary for image generation
./voyage-cli generate-image --prompt "banner" --output banner.png
```

### After: Basic Image Generation

```ailang
module examples/image_generation

import std/ai as AI
import std/io (println)

export func main() -> string ! {AI, FS, IO} =
  let path = AI.callImage(
    "A serene mountain landscape at golden hour, photorealistic",
    "output/landscape.png",
    "{}"
  )
  let _ = println("Generated image: " ++ path)
  path
```

### After: Programmatic Pipeline (base64)

```ailang
module examples/image_pipeline

import std/ai as AI
import std/bytes (fromBase64)
import std/json (decode, get)
import std/fs (writeFile)
import std/io (println)

export func main() -> string ! {AI, FS, IO} =
  let raw = AI.callImageBase64(
    "Logo for a tech startup called 'Nexus'",
    "{\"aspect_ratio\": \"1:1\"}"
  )
  let json = decode(raw)
  let b64 = get(json, "base64")
  match fromBase64(b64) {
    Some(data) ->
      let _ = writeFile("output/logo.png", data)
      let _ = println("Saved logo.png")
      "ok"
    None ->
      "error: invalid base64 data"
  }
```

### Blog Skill (Original Use Case)

```ailang
module blog/generate_banner

import std/ai as AI

export func main() -> string ! {AI, FS} =
  AI.callImage(
    "Modern minimalist banner for a product launch, blue gradient, abstract geometric shapes",
    "assets/banner.png",
    "{\"aspect_ratio\": \"16:9\"}"
  )
```

---

## Implementation Plan

### Phase 1: Provider Infrastructure (~4 hours)

- [ ] Extend `ai.Request` with `ResponseModalities` and `ImageOptions`
- [ ] Extend `ai.Response` with `ImageData` and `ImageMIME`
- [ ] Add `ResponseModalities` field to Gemini `generationConfig`
- [ ] Update Gemini `generateContent()` to extract `InlineData` from response parts
- [ ] Add "not supported" errors for OpenAI, Anthropic, Ollama providers
- [ ] Unit tests for Gemini response parsing with image data

### Phase 2: Effect System & Builtins (~4 hours)

- [ ] Extend `AIHandler` interface with `CallImage` and `CallImageBase64`
- [ ] Implement `CallImage` and `CallImageBase64` on `Handler`
- [ ] Implement `parseImageOptions` helper
- [ ] Extend `StubAIHandler` with image stub responses
- [ ] Register `callImage` and `callImageBase64` effect operations
- [ ] Register `_ai_call_image` and `_ai_call_image_base64` builtins
- [ ] Update `std/ai.ail` with new function exports
- [ ] Unit tests for builtins and effect dispatch

### Phase 3: Examples, Docs & Polish (~2 hours)

- [ ] Create `examples/image_generation.ail`
- [ ] Run `make verify-examples` to confirm
- [ ] Update CHANGELOG.md
- [ ] Run `make test` and `make lint`
- [ ] Update this design doc status

---

## Testing Strategy

**Unit tests:**
- Stub handler returns known 1x1 PNG for any image request
- `callImage` writes file to disk with correct PNG content
- `callImageBase64` returns valid JSON with base64 and mime_type
- Options parsing (aspect_ratio, mime_type pass through correctly)
- Empty options `"{}"` uses defaults
- Unsupported providers return descriptive error

**Integration tests:**
- Gemini live call (gated by `GOOGLE_API_KEY` env var): generate real image
- Effect checking: `callImage` without `FS` capability fails with effect error
- Example file runs end-to-end with `--ai-stub`

**Manual testing:**
- Run `examples/image_generation.ail` with real Gemini API key
- Verify generated image is valid PNG/JPEG (open in viewer)

---

## Success Criteria

- [ ] `AI.callImage` generates and saves image file via Gemini
- [ ] `AI.callImageBase64` returns valid base64 JSON
- [ ] Stub handler works for deterministic testing
- [ ] Unsupported providers fail with clear error messages
- [ ] `examples/image_generation.ail` works with `--ai-stub`
- [ ] All tests passing (`make test`)
- [ ] Lint clean (`make lint`)
- [ ] CHANGELOG.md updated
- [ ] `make verify-examples` passes

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | AI image generation is inherently nondeterministic, but wrapped in explicit `AI` effect |
| A2: Replayability | 0 | No impact — effect traces capture the call |
| A3: Effect Legibility | +1 | `callImage` explicitly requires `{AI, FS}` — both effects visible in signature |
| A4: Explicit Authority | +1 | Requires `--caps AI,FS` at CLI — no ambient file write access |
| A5: Bounded Verification | 0 | No impact |
| A6: Safe Concurrency | 0 | Single-threaded effect handler, no shared state |
| A7: Machines First | +1 | Structured options JSON; base64 variant enables machine-processable pipeline |
| A8: Minimal Syntax | 0 | No new syntax — just new builtin functions following existing pattern |
| A9: Cost Visibility | +1 | Same AI budget tracking; token counts captured in Response |
| A10: Composability | +1 | `callImageBase64` composes with `std/bytes.fromBase64` and `std/json.decode` |
| A11: Structured Failure | +1 | Clear errors for unsupported providers; `Option[bytes]` for decode failures |
| A12: System Boundary | 0 | Uses existing AI effect boundary |

**Net Score: +6** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): Nondeterminism is explicit via `AI` effect — no violation
- [x] A3 (Effects): Both `AI` and `FS` effects declared in signature
- [x] A4 (Authority): Requires explicit `--caps` flags at CLI invocation
- [x] A7 (Machines First): JSON options, base64 output — machine-friendly throughout

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Gemini API changes image response format | Medium | Integration test catches regressions; `InlineData` format is stable |
| Large images cause memory pressure in eval | Medium | Stream to disk in `callImage`; warn on >10MB base64 in `callImageBase64` |
| Adding methods to `AIHandler` interface breaks existing implementations | High | Only 2 implementations exist (Handler, StubAIHandler) — both in our codebase |
| Options JSON is stringly-typed | Low | Matches existing `callJson` pattern; future: typed record when AILANG has optional fields |

---

## Non-Goals

- **Audio/video generation** — Same pattern, future milestone after image gen proves the approach
- **Image-to-image transformation** — Inpainting, style transfer require different API shapes
- **Multi-image generation** — Single image per call for v1; batch in future
- **`std/image` module** — No resize/crop/compose; use external tools
- **OpenAI DALL-E support** — Different API shape (`/v1/images/generations`), separate milestone
- **Streaming binary output** — Unnecessary for image gen (whole image or nothing)

## Future Work

- OpenAI DALL-E provider support (separate API shape)
- Audio generation following same pattern (`callAudio`)
- `std/image` module for basic manipulation (resize, format conversion)
- Multi-image generation (batch API)
- Typed record options when AILANG supports optional fields

---

## References

- [std/ai.ail](../../../std/ai.ail) — Current AI stdlib module (3 functions)
- [std/bytes.ail](../../../std/bytes.ail) — Existing bytes support with `fromBase64`/`toBase64`
- [internal/builtins/ai.go](../../../internal/builtins/ai.go) — AI builtin registrations
- [internal/ai/provider.go](../../../internal/ai/provider.go) — Provider interface, Request/Response types
- [internal/ai/handler.go](../../../internal/ai/handler.go) — Handler bridge (Call, CallJson)
- [internal/ai/gemini/generate.go](../../../internal/ai/gemini/generate.go) — Gemini provider with multimodal input
- [internal/effects/ai.go](../../../internal/effects/ai.go) — AIHandler interface, effect ops
- [M-UNIFIED-AI-PROVIDERS](../../implemented/v0_5_10/m-unified-ai-providers.md) — Provider architecture
- [M-STRUCTURED-AI-OUTPUT](../../implemented/v0_7_3/m-structured-ai-output.md) — Pattern for adding AI operations
- Origin message: Agent inbox `2a9d4677` from `blog` agent
