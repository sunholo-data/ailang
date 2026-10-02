# Sprint Plan: M-OPENROUTER-IMAGE-OUTPUT

## Summary
Implement the approved [design](m-openrouter-image-output.md): portraits and text through one OpenRouter key, with an optional image model override resolved under the existing provider boundary.

**Status:** Planned; implementation awaits sprint approval through the coordinator.
**Target:** v0.51.1
**Duration:** 3 days, approximately 18 engineering hours including 25% contingency.
**Estimated total:** 700 LOC across implementation, tests and documentation.
**Risk:** Medium: wire compatibility, provider enforcement and image decoding.
**Issues:** #1504 (image output), #1496 (image model choice). #1495 (TTS) excluded.

## Current Status and Velocity
The clean planning checkout is `coordinator/task-8e76b5a7`; the handed-off design originated on `coordinator/task-5871413e`. Version is v0.51.0. The approval supplied in the handoff authorizes planning; the design's unchecked approval box is stale and is not changed here.

Source inspection confirms the Generate refusal, two handler calls pinned to h.model, and direct AIContext image dispatch without resolution. The existing step resolver and provider composition are the reuse points. The design audits other providers and all image entry points; their explicit unsupported errors stay in place.

The seven-day velocity script sees only the 2026-10-02 design commit (#1504), no implementation history or usable LOC/day metrics. Its changelog scan finds historical entries, not recent throughput. Therefore 233 LOC/day is a planning capacity assumption, not measured velocity. The design estimates 2–3 days; choose 3 days with contingency and extra decoder/resolver tests. Coverage baseline will be measured during execution, not invented during planning.

## Registry Reuse Audit
Ran `ailang pkg search image` and `ailang pkg search openrouter`. The image results are document parsing, Gemini file upload, LinkedIn and Discord clients; OpenRouter returns `sunholo/decisions`, a separate decision endpoint client. Inspected `pkg info` and `pkg docs` for `sunholo/ailang_parse` and `sunholo/decisions`; neither provides the required native provider capability. None can replace the Go provider adapter, builtin-handler model resolution or standard-library documentation. No package dependency is added.

| Milestone | Decision | Reason |
|---|---|---|
| M1 | none | Native Go chat transport and AI cost/correlation plumbing must be extended in place; document parsing and decision endpoints are unrelated. |
| M2 | none | Existing Go ModelResolver is the enforcement boundary; an AILANG package cannot replace it. |
| M3 | none | This documents and verifies the builtins and repository example; no package capability is introduced. |

## Milestones

### M1: OpenRouter chat image output (~430 LOC)

**Estimate:** 130 implementation + 300 tests = 430 LOC
**Schedule:** Day 1, 7 hours
**Dependencies:** None
**Files:** internal/ai/openrouter/{types,chat,client,step,streamstep}.go; internal/ai/openrouter/images_test.go (new); client_test.go and correlation_test.go
**Example:** examples/runnable/ai_image_generation.ail (usage integrated in M3)

Extend the composed OpenRouter wire structs without modifying openai.ChatRequest. Decode the first inline image, preserve captions and billing metadata, and guard both step paths. Use httptest fixtures and golden JSON bodies; check omitted extension fields on text calls.

**Acceptance criteria:**

- [ ] Image Generate sends lowercase modalities image,text and maps aspect_ratio and PNG/JPEG output_format; text-only bodies retain byte identity.
- [ ] Inline base64 data URLs decode to ImageData/ImageMIME while caption text, usage.cost, routing and correlation remain intact.
- [ ] First image wins and ai.image_count records the returned count; zero images, remote URLs, malformed data URLs, invalid base64 and unsupported MIME options fail with typed errors.
- [ ] Step and StreamStep reject IMAGE requests before network dispatch; non-image model upstream errors remain typed ProviderError.
- [ ] Positive dispatch replaces the obsolete image-refusal test and focused OpenRouter tests pass.

**Risk and mitigation:** Response message composition could lose tool-call fields used by Step; retain every existing field and run step/stream regressions.

### M2: Per-call image model resolution (~210 LOC)

**Estimate:** 90 implementation + 120 tests = 210 LOC
**Schedule:** Day 2, 6 hours
**Dependencies:** M1
**Files:** internal/ai/{provider,handler,handler_test}.go; internal/effects/{ai,ai_test}.go
**Example:** examples/runnable/ai_image_generation.ail (per-call model variant integrated in M3)

Expose the existing image-options parser or a narrow rewrite helper. Resolve model only when supplied, preserve unknown keys and existing malformed JSON behavior, then coalesce the selected model independently in both handler calls. Use provider and handler spies to assert pre-dispatch rejection and call-local overrides.

**Acceptance criteria:**

- [ ] Both image call methods honor options.model for only that call; absent or empty model uses the bound model.
- [ ] AIContext reuses the step ModelResolver for friendly names and rejects cross-provider overrides with CodeModelNotAllowed before invoking the handler.
- [ ] Nil resolver passthrough, stub behavior, existing malformed-options behavior and unknown option keys remain compatible; rewriting preserves caller options.
- [ ] Handler requests and RequestedModel identify the selected model; a later call without an override still uses the original bound model.
- [ ] Focused internal/ai and internal/effects tests pass for both file and base64 methods.

**Risk and mitigation:** Re-marshaling parsed options could drop unknown keys or change defaults; test round-trip preservation and absent-model passthrough.

### M3: Examples documentation and regression verification (~60 LOC)

**Estimate:** 30 implementation/test support + 30 documentation = 60 LOC
**Schedule:** Day 3, 5 hours including contingency
**Dependencies:** M1, M2
**Files:** std/ai.ail; examples/runnable/ai_image_generation.ail; changelogs/v0.32-current.md
**Example:** examples/runnable/ai_image_generation.ail

Read ailang prompt, update comments and the existing runnable example, type-check and run its stub flow, then execute focused and repository regression checks. Record live smoke results only when actually run.

**Acceptance criteria:**

- [ ] std/ai.ail comments document OpenRouter and options.model; runnable image example demonstrates bound text plus per-call image model and retains Gemini usage.
- [ ] Run ailang prompt before editing AILANG, then check the example and execute its stub-backed path with documented capability flags.
- [ ] v0.51.1 changelog records image routing and model override; TTS issue 1495 remains out of scope.
- [ ] go test ./internal/ai/... ./internal/effects/... ./cmd/ailang, make test-core, make fmt, make lint and make check-boundaries pass.
- [ ] If credentials are available, explicitly opt into a paid OpenRouter portrait smoke and Gemini regression run; otherwise record live verification as skipped, without claiming success.

**Risk and mitigation:** Live APIs require credentials and incur cost; deterministic fixtures provide required acceptance, live checks are supplementary.

## Verification and Success Metrics
All milestone criteria must pass with no provider allowlist or silent fallback. Capture focused package coverage before/after with `go test -cover ./internal/ai/... ./internal/effects/...`; aim for at least 90% of new branches through explicit tests, without claiming a repository-wide coverage increase. Tests must include file-output bytes, base64 JSON and failed resolution causing zero provider invocations. Formatting must produce no unintended unrelated edits.

The executor should use `go test ./internal/ai/openrouter` for M1, `go test ./internal/ai ./internal/effects` for M2, then the M3 checks. HTTP fixtures need no secrets or paid calls. Inspect shared response structs for Step/tool-call compatibility. Do not forward additional size/quality keys in this sprint. Unsupported explicit mime_type must return an error rather than silently selecting a format. Keep the design's PNG default for an absent MIME only.

## Dependencies and Handoff
No new external dependency. M1 → M2 → M3 is the proposed sequence. Preserve effects.AIHandler signatures and the Gemini adapter. Cross-provider image choice is intentionally rejected. No open product decision blocks planning.

The coordinator should present these artifacts for sprint approval; merging the planning PR triggers sprint-executor according to the configured handoff. This planning task does not authorize implementation. Progress JSON starts `not_started`, with every milestone `passes: null`. Link #1504 and #1496; do not close #1495 or imply completion of its TTS work.

## Planning Artifact Validation
The supplied shell validator cannot run correctly because `jq` is absent (it reports invalid JSON). Python JSON parsing and equivalent structural checks pass: three real milestones, all acceptance criteria populated, valid dependencies, per-milestone registry decisions, 700 LOC total and three estimated days. `git diff --check` passes. No implementation tests were run during planning.
