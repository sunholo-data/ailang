# M-STD-AI-VISION-INPUT — Sprint Plan

**Design doc**: [m-std-ai-vision-input.md](m-std-ai-vision-input.md)
**Target**: v0.30.0
**Risk**: Medium (record-shape wire-contract change across 5 providers; back-compat is the pivot)
**Estimated**: 3–4 days (~600 LOC impl + ~250 LOC tests)
**Created**: 2026-07-16

## Goal

Feed images into vision-capable models through the existing `std/ai` surface, backward-compatibly.
Images live on the **`Message` record**, so every `step`/`stepWithCache`/`stepWithStream` caller
(motoko included) gets vision for free; `callJsonResultVision` is the one-call grading shortcut.

## Sequencing rationale

M1 is the foundation and the highest-risk item (back-compat of the record-shape change). It MUST
land and be proven green before M2 touches any provider — a broken back-compat contract would
force rework across all 5 adapters. M2 fans out over the 5 providers (independent, parallelizable).
M3 is additive surface (builtin + example + docs) and depends only on M1.

---

## M1 — Record shape, effect bridge, provider abstraction (~1 day, ~120 LOC + 80 test)

**The pivot milestone.** Adds `images` to the AILANG `Message` record, threads it through the
effect bridge into a new `ai.Message.Images` field, and — critically — proves that omitting
`images` still type-checks and produces a byte-identical provider request.

**Tasks:**
- [ ] Add `imagePartRecordType` = `{source: string, mime: string}` and an `images: [ImagePart]`
      field to `messageRecordType` — [`internal/builtins/ai_step.go:37-45`](../../../internal/builtins/ai_step.go#L37-L45).
- [ ] **Back-compat proof (BLOCKER for M2):** confirm a `Message` literal WITHOUT `images` still
      type-checks. If AILANG record typing requires the field, resolve via optional/defaulted
      handling at the builtin boundary (Conflict Surface item 3). Do not proceed to M2 until a
      test asserts the old 4-field literal compiles.
- [ ] Add `ImagePart struct { Source, Mime string }` and `Images []ImagePart` to `ai.Message`
      — [`internal/ai/provider.go:200-205`](../../../internal/ai/provider.go#L200-L205).
- [ ] Extend `decodeMessages` / record→`ai.Message` conversion to read the images list (default
      empty) — [`internal/effects/ai_step.go`](../../../internal/effects/ai_step.go) (`getStringField` neighbor at :468).

**Acceptance criteria:**
- `Message` literal without `images` type-checks and runs unchanged (regression test).
- Golden test: empty/absent `images` → identical `ai.Message` (no `Images` populated).
- `ImagePart` decodes correctly from an AILANG record with `source`+`mime`.
- `go build ./...` clean; `go test ./internal/builtins/ ./internal/effects/` green.

**Risk:** record-typing back-compat (High). Mitigation: this is the milestone's own blocker gate.

---

## M2 — Provider adapters + capability guard (~1.5 days, ~150 LOC + 100 test)

Map `ai.Message.Images` to each provider's native image block. Non-vision model + non-empty
images → `AIError{code:"model_no_vision"}`. Each provider is independent — parallelizable.

**Tasks (one per provider — each in its `step.go`/`streamstep.go`/`cache.go` message builder):**
- [ ] **Anthropic** — `{type:"image", source:{type:"base64", media_type, data}}` in the
      `contentBlock` array — [`internal/ai/anthropic/step.go`](../../../internal/ai/anthropic/step.go), reuse `client.go:123` `contentBlock`.
- [ ] **OpenAI** — `{type:"image_url", image_url:{url}}` (data-URI ok) — [`internal/ai/openai/step.go`](../../../internal/ai/openai/step.go), `types.go`.
- [ ] **Gemini** — `inline_data{mime_type, data}` (file_data deferred) — [`internal/ai/gemini/step.go`](../../../internal/ai/gemini/step.go), `types.go`.
- [ ] **OpenRouter** — dispatch by model prefix, reuse existing step/cache routing — [`internal/ai/openrouter/step.go`](../../../internal/ai/openrouter/step.go).
- [ ] **Ollama** — `images:[base64]` on the message — [`internal/ai/ollama/step.go`](../../../internal/ai/ollama/step.go).
- [ ] **Capability guard** — new error code `model_no_vision` (grep-verify unallocated first:
      `grep -rn "model_no_vision\|CodeModelNoVision" internal/ cmd/`); non-vision model + images
      → typed non-retryable `AIError` — [`internal/ai/errors.go`](../../../internal/ai/errors.go).

**Acceptance criteria:**
- Per-provider unit test: `ImagePart` → correct native image-block JSON (recorded fixture).
- Empty `Images` → each provider emits its exact pre-change wire request (golden, no regression).
- Non-vision model + images → `model_no_vision` AIError (not silent drop, not panic).
- `go test ./internal/ai/...` green.

**Risk:** per-provider format drift (Med). Mitigation: recorded-JSON fixtures checked against each
provider's current API docs; forensic diff on one live call per provider where creds available.

---

## M3 — Vision structured-output builtin + example + docs (~1 day, ~120 LOC + 70 test)

The ergonomic grading path and user-facing surface.

**Tasks:**
- [ ] Register `callJsonResultVision(input: string, images: [ImagePart], schema: string)` →
      `Result[..., AIError]`, thin pass-through mirroring `callJsonResult` —
      [`internal/builtins/ai.go`](../../../internal/builtins/ai.go), effect op in [`internal/effects/ai_step.go`](../../../internal/effects/ai_step.go).
- [ ] `examples/ai_vision_input.ail` — read a local image, ask a vision model a structured
      question, print the JSON (runnable; the CLAUDE.md "every feature needs an example" rule).
- [ ] std/ai doc update (`ailang docs std/ai` surface + website reference) documenting `images`,
      `ImagePart`, `callJsonResultVision`, and the `model_no_vision` error.
- [ ] CHANGELOG entry under `[Unreleased]`.
- [ ] Reply to stx-bench (msg_20260716_070552_f9104997) with the shipped surface. *(already sent
      a triage ack; send a "landed" follow-up when green.)*

**Acceptance criteria:**
- `callJsonResultVision` returns structured JSON end-to-end against a mock/configdriven vision provider.
- `examples/ai_vision_input.ail` runs green (`make verify-examples` passes it).
- Docs + CHANGELOG updated; `ailang docs std/ai` shows the new surface.

**Risk:** Low (additive).

---

## Success Metrics

- [ ] All 3 milestones' acceptance criteria met.
- [ ] Zero regression: existing text-only `step`/tool-loop/cache/stream programs unchanged (golden).
- [ ] `examples/ai_vision_input.ail` created and verified (`make verify-examples`).
- [ ] `make test` green; `make build` clean.
- [ ] All 5 providers forward images; non-vision → typed `model_no_vision`.
- [ ] Docs (std/ai + website) + CHANGELOG updated.

## Deferred (per design doc — agent latitude)

- Local-path image sources (`FS` cap) — v1 may ship base64/data-URI only.
- Gemini `file_data` uploaded refs — v1 ships `inline_data` only.
- Vision-detection strategy (static table vs provider-reported) — agent decides; must grep the
  new error code unallocated first.

## Open Questions

- **Back-compat mechanism (M1):** does adding `images` to `messageRecordType` require every
  existing literal to add `images:[]`, or can the builtin boundary default it? This decides
  whether M1 is a clean additive change or needs a record-defaulting shim. Resolve in M1 before M2.
- **`model_no_vision` code casing:** design doc uses `model_no_vision`; existing codes are
  PascalCase (`ModelNotFound`). Align to the existing convention at implementation time
  (likely `ModelNoVision`) — cosmetic, agent decides.
