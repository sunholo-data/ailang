# Sprint Plan: M-AI-IMAGE

## Summary
Add AI image generation to `std/ai` with `callImage` (file-based) and `callImageBase64` (programmatic) functions, extending the provider infrastructure through the full call chain: provider types, Gemini implementation, effect system, builtins, and stdlib.

**Duration:** 2 days (~10 hours)
**Dependencies:** None (all prerequisite infrastructure exists)
**Risk Level:** Low — follows established pattern from M-STRUCTURED-AI-OUTPUT
**Design Doc:** [m-ai-image-generation.md](m-ai-image-generation.md)

## Current Status Analysis

### Completed Recently (Last 7 Days)
- Named JSON parameter binding for serve-api: ~1,253 LOC in 1 day
- Cross-package stdlib resolution: ~981 LOC in 1 day
- @raw route annotation: ~767 LOC in 1 day
- @nowrap annotation: ~324 LOC in <1 day

### Velocity
- Recent average: ~500-800 LOC/day (feature implementation)
- Estimated capacity: ~430 LOC for this sprint (well within range)

### Remaining from Design Doc
- All work is new — no partial implementation exists
- Provider types, Gemini support, effects, builtins, stdlib, examples all needed

## Proposed Milestones

### M1: Provider Infrastructure & Gemini Implementation
**Goal:** Extend `ai.Request`/`ai.Response` with image fields, implement Gemini image generation, add unsupported-provider errors
**Estimated:** ~55 LOC implementation + ~50 LOC tests = ~105 LOC
**Duration:** ~3 hours

**Tasks:**
1. Add `ResponseModalities []string` and `ImageOptions *ImageOptions` to `ai.Request`
2. Add `ImageData []byte` and `ImageMIME string` to `ai.Response`
3. Add `ResponseModalities` to Gemini `generationConfig` struct
4. Update Gemini `generateContent()` to pass `ResponseModalities` and extract `InlineData` from response parts
5. Add "image generation not supported" error returns in OpenAI, Anthropic, Ollama providers
6. Unit test: Gemini response parsing with InlineData parts

**Acceptance Criteria:**
- [ ] `ai.Request` has `ResponseModalities` and `ImageOptions` fields
- [ ] `ai.Response` has `ImageData` and `ImageMIME` fields
- [ ] Gemini provider sets `responseModalities` in API request
- [ ] Gemini provider decodes base64 InlineData into `Response.ImageData`
- [ ] Non-Gemini providers return clear error for image requests
- [ ] Unit tests pass for image response parsing

**Risks:**
- Gemini InlineData response format may differ from docs — Mitigation: test with real API call if key available

### M2: Effect System, Builtins & Stdlib
**Goal:** Wire image generation through the full AILANG call chain: AIHandler interface, Handler methods, effect operations, builtins, and std/ai.ail
**Estimated:** ~200 LOC implementation + ~80 LOC tests = ~280 LOC
**Duration:** ~4 hours
**Dependencies:** M1

**Tasks:**
1. Extend `AIHandler` interface with `CallImage(prompt, path, options string) (string, error)` and `CallImageBase64(prompt, options string) (string, error)`
2. Implement `CallImage` on `Handler` — parse options, build Request with `ResponseModalities: ["IMAGE"]`, call provider, write file
3. Implement `CallImageBase64` on `Handler` — same but return base64 JSON
4. Add `parseImageOptions(json string) *ImageOptions` helper
5. Extend `StubAIHandler` with stub image responses (1x1 PNG)
6. Register `callImage` and `callImageBase64` effect operations in `init()`
7. Implement `aiCallImage` and `aiCallImageBase64` effect op functions
8. Register `_ai_call_image` and `_ai_call_image_base64` builtins with metadata
9. Update `std/ai.ail` with `callImage` and `callImageBase64` exports
10. Unit tests for builtins, effect dispatch, and stub handler

**Acceptance Criteria:**
- [ ] `AIHandler` interface has `CallImage` and `CallImageBase64` methods
- [ ] `Handler.CallImage` generates image and writes to file
- [ ] `Handler.CallImageBase64` returns JSON with base64 and mime_type
- [ ] `StubAIHandler` returns valid test PNG for image requests
- [ ] Effect ops `callImage` and `callImageBase64` are registered
- [ ] Builtins `_ai_call_image` and `_ai_call_image_base64` registered with metadata
- [ ] `std/ai.ail` exports both functions with correct effect signatures
- [ ] `make test` passes
- [ ] `make lint` passes

**Risks:**
- Extending AIHandler interface requires updating all implementations — Mitigation: only 2 exist (Handler, StubAIHandler), both in our codebase

### M3: Examples, Docs & Verification
**Goal:** Create working example, update CHANGELOG, verify everything passes
**Estimated:** ~45 LOC
**Duration:** ~1 hour
**Dependencies:** M2

**Tasks:**
1. Create `examples/image_generation.ail` with `--ai-stub` compatible example
2. Update CHANGELOG.md with new `std/ai` functions
3. Run `make verify-examples` to confirm example passes
4. Run full `make test` and `make lint`

**Acceptance Criteria:**
- [ ] `examples/image_generation.ail` exists and compiles
- [ ] `make verify-examples` passes
- [ ] CHANGELOG.md updated
- [ ] `make test` passes
- [ ] `make lint` passes

## Success Metrics
- Test coverage: >80% on new code
- Examples passing: `image_generation.ail` verified
- Documentation: CHANGELOG.md updated
- All tests passing
- All linting passing

## Open Questions
- Options JSON schema: which fields beyond `aspect_ratio` and `mime_type` should be supported? (Design freeze item — agent can resolve with minimal set)

## Notes
- This sprint follows the exact pattern established by M-STRUCTURED-AI-OUTPUT (v0.7.3) which added `callJson`/`callJsonSimple` through the same layers
- The Gemini provider already handles InlineData for multimodal *input* — extending to *output* is straightforward
- `std/bytes.fromBase64` already exists, so the base64 pipeline composes naturally
