# M-AI-STRUCTURED-STEP — Sprint Plan

**Design doc**: [m-ai-structured-step.md](m-ai-structured-step.md)
**Target**: v0.31.0
**Risk**: Medium-High (Anthropic forced-tool structured output collides with the existing tool-loop; schema-on-Step must not regress the no-schema wire path across 5 providers)
**Estimated**: 4–6 days (~350 LOC impl + ~200 LOC tests)
**Created**: 2026-07-16
**Dependencies**: M-STD-AI-VISION-INPUT (v0.30.0, done) — schema composes with images on `step`.

## Goal

Make provider-native structured output an **orthogonal option on `step`**, so messages + tools +
images + schema compose in one call. `stepWithOptions(model, messages, tools, StepOptions)` where
`StepOptions = {response_format, response_schema, cache_breakpoints}`. Subsumes the deferred
`callJsonResultVision`. JSON always lands in `StepResult.message.content`.

Design Freeze is RESOLVED (see design doc): StepOptions record; Anthropic schema+user-tools →
typed `AIError`; response normalized into `message.content`.

## Sequencing rationale

M1 (surface + threading) is the foundation and blocks M2 — no provider can honor a schema on Step
until the schema reaches the Request there. M2 fans out per provider (independent packages, like the
vision M2), with **Anthropic as the heavy, serial item** (forced-tool + extraction + collision
guard) — do not parallelize Anthropic with the mechanical providers; it needs care and owns the
response-normalization contract. M3 is additive (example + docs), depends on M1+M2.

---

## M1 — StepOptions surface + effect bridge + handler threading (~1.5 days, ~120 LOC + 60 test)

**The foundation.** Adds the `StepOptions` record, the `stepWithOptions` builtin, and threads
`response_format`/`response_schema` from the AILANG surface through the effect bridge and handler
onto the outgoing `ai.Request` for the Step path. No provider behavior yet — just plumbing + the
no-schema regression guarantee.

**Tasks:**
- [ ] `std/ai.ail`: `StepOptions` type `{response_format: string, response_schema: string, cache_breakpoints: [CacheBreakpoint]}`; `stepWithOptions(model, messages, tools, options)`; refactor `step`/`stepWithCache` as thin wrappers (keep their existing signatures — additive).
- [ ] `internal/builtins/ai_step.go`: register `stepWithOptions` + `stepOptionsRecordType`.
- [ ] `internal/effects/ai_step.go`: new effect op that reads options and sets `Request.ResponseFormat`/`ResponseSchema` (+ existing cache breakpoints).
- [ ] `internal/ai/handler.go`: thread schema onto the Step/StepWithCache/StepWithStream Request.

**Acceptance criteria:**
- `stepWithOptions` type-checks and runs; `step`/`stepWithCache` still work unchanged (wrappers).
- Empty `response_schema` ⇒ Request carries no schema ⇒ Step wire request byte-for-byte identical to today (golden/handler test).
- `go build ./...` clean; `go test ./internal/builtins/ ./internal/effects/ ./internal/ai/` green.
- Builtin-types golden regenerated for the new builtin.

**Risk:** `StepOptions` refactor of `stepWithCache` could regress cache callers. Mitigation: keep `stepWithCache` signature; wrapper delegates to `stepWithOptions`. BLOCKER GATE for M2.

---

## M2 — Per-provider Step schema + Anthropic forced-tool + normalization (~2.5 days, ~180 LOC + 110 test)

Each provider honors `ResponseSchema` on its Step path; JSON is normalized into
`StepResult.message.content`. **Mechanical providers are parallelizable; Anthropic is serial.**

**Tasks (mechanical — parallelizable):**
- [ ] OpenAI Step (`internal/ai/openai/step.go`): set `response_format:{type:"json_schema"}` when schema present, reusing `chat.go`'s `ensureStrictSchemaCompliance`.
- [ ] Gemini Step (`internal/ai/gemini/step.go`): set `GenerationConfig.responseSchema`+`responseMimeType`, reusing `generate.go` logic.
- [ ] Ollama Step: already wired (`step.go:381`) — **regression test only**.
- [ ] OpenRouter: inherits via shared `openai.BuildChatStepRequest` — **regression test only**.
- [ ] configdriven Step: typed `AIError` when a schema is requested (unsupported in v1 `[[ai_provider]]` schema).

**Tasks (Anthropic — serial, owns normalization):**
- [ ] `internal/ai/anthropic/step.go`: when `ResponseSchema` set AND no user tools → inject a forced `respond` tool (`input_schema` = schema, `tool_choice` forces it).
- [ ] **Collision guard**: `ResponseSchema` set AND `len(Tools) > 0` → typed `AIError` (`CodeCapabilityNotSupported` or a new `CodeSchemaWithToolsUnsupported`; grep-verify before allocating) with a clear message.
- [ ] **Extraction**: pull the returned `tool_use` block's `input` JSON into `Response.Text` so it normalizes into `message.content` like every other provider.

**Acceptance criteria:**
- Per-provider unit test: schema on Step produces the correct native structured-output request (recorded wire JSON).
- Empty schema ⇒ each provider's Step wire request byte-for-byte identical to today (golden, no regression).
- Anthropic: schema-only → forced `respond` tool + extracted JSON in `message.content`; schema+user-tools → typed error (tested).
- configdriven: schema requested → typed error.
- `go test ./internal/ai/...` green.

**Risk:** Anthropic extraction breaks the existing tool-loop, OR schema regresses the no-schema path. Mitigation: strict `ResponseSchema==""` guard; golden tests on the no-schema tool-loop + per-provider empty-schema wire.

---

## M3 — Composable example + docs + notify (~1 day, ~50 LOC + 30 test)

**Tasks:**
- [ ] `examples/runnable/ai_structured_step.ail`: structured output via `stepWithOptions` (schema), and a vision+schema variant showing the stx-bench grading path (guaranteed JSON, no markdown fence). Register in `examples/manifest.json`.
- [ ] Update the vision example's comment (the "one-call structured helper is a follow-up" note now points to the shipped `stepWithOptions`).
- [ ] std/ai docs (CLI `ailang docs std/ai` surface + website) + CHANGELOG: `StepOptions`, `stepWithOptions`, per-provider schema support, Anthropic schema+tools limitation.
- [ ] Notify stx-bench: composable vision+JSON grading path is live; show the one-call pattern.

**Acceptance criteria:**
- `ai_structured_step.ail` type-checks + runs live: `stepWithOptions` returns schema-valid JSON (no markdown fence) from a real provider.
- Vision + schema in one call returns graded JSON (live against a vision model).
- Docs + CHANGELOG updated; example in manifest; stx-bench notified.

**Risk:** Low (additive).

---

## Success Metrics

- [ ] `stepWithOptions` requests native structured output on all 5 providers; JSON in `message.content`.
- [ ] Vision + schema compose in one call (live-verified) — guaranteed-valid JSON, no prompt-injection.
- [ ] Anthropic schema-only works; schema+user-tools returns a documented typed error.
- [ ] Zero regression: no-schema Step path byte-for-byte identical (golden per provider); `step`/`stepWithCache` wrappers unchanged.
- [ ] `callJsonResultVision` NOT added (subsumed).
- [ ] `make test` green; example + docs + CHANGELOG; stx-bench notified.

## Deferred (per design doc)

- Anthropic schema+tools once Anthropic ships a native response-format.
- Schema on `stepWithStream` may be deferred if provider streaming + structured output conflict.
- Deprecation of `callJson`/`callJsonResult` — human decides later; out of scope.

## Open Questions (resolve at sprint start)

- **New error code for schema+tools collision?** Grep-verify whether `CodeCapabilityNotSupported`
  suffices or a dedicated `CodeSchemaWithToolsUnsupported` is warranted (allocate only if free).
- **StepOptions field naming** — `response_format`/`response_schema` (mirrors `ai.Request`) vs
  shorter `format`/`schema`. Agent decides; keep consistent with the record on the Go side.
