# Sprint Plan: M-STRUCTURED-AI-OUTPUT

## Summary
Add `callJson` and `callJsonSimple` to `std/ai` so AI responses can return validated JSON (`Json` type) instead of raw strings. This eliminates the fragile `stripCodeFences` → `decode` → retry pattern seen in downstream applications.

**Duration:** 3 days (10-12 hours implementation)
**Dependencies:** None — all provider infrastructure exists; `Json` type already in `std/json`
**Risk Level:** Medium (4 providers with different structured output APIs)

## Current Status Analysis

### Completed Recently (v0.7.2 → v0.7.3 WIP)
- Effectful list combinators (`mapE`, `filterE`, `foldlE`, `flatMapE`, `forEachE`): parser + typechecker + stdlib changes
- ZIP archive builtins: ~315 LOC impl + ~388 LOC tests
- XML parsing builtins: ~530 LOC impl + ~530 LOC tests
- `readFileBytes` for std/fs

### Velocity
- Recent average: ~300-500 LOC/day for feature work (excluding docs/infra)
- ZIP milestone: ~700 LOC in ~1.5 days
- XML milestone: ~1060 LOC in ~2 days
- Estimated capacity: ~900-1500 LOC for a 3-day sprint

### Remaining from Design Doc
- Phase 1: Provider support (~80 LOC across 4 providers + handler)
- Phase 2: Builtin + effect wiring (~85 LOC)
- Phase 3: Stdlib wrapper (~20 LOC)
- Phase 4: Docs + examples (~50 LOC)
- Tests: (~80 LOC)
- **Total: ~315 LOC** (design doc estimate ~290; revised after codebase review)

## Proposed Milestones

### Milestone 1: Core Types + Provider Support
**Goal:** Add `ResponseFormat`/`ResponseSchema` fields to `ai.Request` and wire them through all 4 providers.
**Estimated:** 100 LOC implementation + 60 LOC tests = 160 LOC
**Duration:** 1 day

**Tasks:**
1. Add `ResponseFormat` and `ResponseSchema` fields to `Request` struct in `internal/ai/provider.go`
2. Add `CallJson(input, schema string) (string, error)` method to `Handler` in `internal/ai/handler.go`
3. **Gemini** (`internal/ai/gemini/types.go` + `generate.go`): Add `responseMimeType` + `responseSchema` to `generationConfig`; pass through when `ResponseFormat == "json"`
4. **OpenAI** (`internal/ai/openai/types.go` + `client.go`): Add `response_format` with `json_schema` nested struct; wire through for both Chat and Responses APIs
5. **Anthropic** (`internal/ai/anthropic/client.go`): Implement tool-use pattern — add single tool with schema as `input_schema`, set `tool_choice: {type: "tool", name: "respond"}`, extract tool result from response
6. **Ollama** (`internal/ai/ollama/client.go`): Add `format: "json"` to request options (no schema enforcement — Ollama limitation)
7. Unit tests for each provider's request building (mock HTTP, verify JSON payload)

**Acceptance Criteria:**
- [ ] `Request.ResponseFormat = "json"` causes Gemini to set `responseMimeType: "application/json"` and `responseSchema`
- [ ] OpenAI request includes `response_format.json_schema` when schema provided
- [ ] Anthropic request includes tool with schema as `input_schema`
- [ ] Ollama request includes `format: "json"`
- [ ] `Handler.CallJson()` passes format/schema to provider
- [ ] All existing `Call()` behavior unchanged (no regressions)
- [ ] All tests passing, linting clean

**Risks:**
- Anthropic tool-use pattern is more complex than other providers — Mitigation: well-documented pattern in Anthropic's API docs; extract tool result from content blocks
- Ollama has no schema enforcement — Mitigation: document that Ollama only guarantees valid JSON, not schema conformance

### Milestone 2: Effects + Builtins + Stdlib
**Goal:** Wire `callJson`/`callJsonSimple` from AILANG stdlib through builtins/effects to the handler.
**Estimated:** 90 LOC implementation + 30 LOC tests = 120 LOC
**Duration:** 0.5 days

**Tasks:**
1. Extend `AIHandler` interface in `internal/effects/ai.go` with `CallJson(input string, schema string) (string, error)`
2. Add `callJson` and `callJsonSimple` effect operations in `AIContext` (parallel to existing `call`)
3. Register `_ai_call_json` builtin in `internal/builtins/ai.go`:
   - Module: `std/ai`, Arity: 2, Params: `(prompt: string, schema: string)`, Returns: `string`, Effect: `AI`
4. Register `_ai_call_json_simple` builtin:
   - Module: `std/ai`, Arity: 1, Params: `(prompt: string)`, Returns: `string`, Effect: `AI`
5. Add `callJson` and `callJsonSimple` wrappers in `std/ai.ail` that call builtins and `decode()` the result to `Json`
6. Update module manifest/exports for `std/ai`
7. Tests with `MockEffContext` for builtin registration and effect dispatch

**Acceptance Criteria:**
- [ ] `_ai_call_json(prompt, schema)` registered as builtin with correct type signature
- [ ] `_ai_call_json_simple(prompt)` registered as builtin
- [ ] `callJson(prompt, schema)` in `std/ai.ail` returns `Json ! {AI}`
- [ ] `callJsonSimple(prompt)` in `std/ai.ail` returns `Json ! {AI}`
- [ ] Invalid JSON from provider produces error (not silent fallback)
- [ ] Budget counts each call as 1 AI unit
- [ ] `ailang doctor builtins` passes

**Risks:**
- Effect dispatch for new operations must follow existing pattern exactly — Mitigation: copy `call` pattern, modify for 2 args

### Milestone 3: AI Stub Support + Examples + Docs
**Goal:** Ensure `--ai-stub` mode returns valid JSON for testing; create example files; update teaching prompt and CHANGELOG.
**Estimated:** 50 LOC implementation + 40 LOC examples/docs = 90 LOC
**Duration:** 0.5 days

**Tasks:**
1. Update `--ai-stub` handler to return valid JSON for `callJson`/`callJsonSimple` (e.g., `{"stub": true}` or schema-conforming stub)
2. Create `examples/runnable/structured_ai_basic.ail` — basic `callJsonSimple` usage
3. Create `examples/runnable/structured_ai_schema.ail` — `callJson` with schema
4. Update teaching prompt (`prompts/v0.7.3.md`) with `callJson`/`callJsonSimple` examples
5. Update `CHANGELOG.md` with structured output feature
6. Verify examples work with `--ai-stub` mode

**Acceptance Criteria:**
- [ ] `ailang run --ai-stub --caps AI --entry main structured_ai_basic.ail` succeeds
- [ ] `ailang run --ai-stub --caps AI --entry main structured_ai_schema.ail` succeeds
- [ ] Teaching prompt documents `callJson` and `callJsonSimple`
- [ ] CHANGELOG updated
- [ ] `make verify-examples` passes with new examples
- [ ] `make lint` clean
- [ ] `make test` all passing

**Risks:**
- AI stub must produce JSON that parses to valid `Json` ADT — Mitigation: use `_json_decode` builtin output format

## Success Metrics
- Test coverage: Maintained (no regression)
- New examples: 2 runnable examples created and verified
- Documentation: Teaching prompt + CHANGELOG updated
- All tests passing
- All linting clean
- `ailang doctor builtins` passes with new builtins

## Dependencies
- `std/json` `Json` ADT and `decode` function (already exists)
- `_json_decode` builtin (already exists)
- Provider API keys for manual end-to-end testing (optional — unit tests use mocks)

## Open Questions
1. **Anthropic response extraction:** Confirm that tool-use content block contains the JSON as a string (not a pre-parsed object). Need to extract and return as raw string for `decode()`.
2. **Schema validation level:** Design doc says "provider enforces schema" — should we add client-side schema validation as a fallback? (Recommend: no, per design doc. Provider does enforcement; we just validate it's valid JSON.)

## Notes
- The design doc's LOC estimate (~290) is realistic — our review confirms ~315 LOC including tests
- This sprint is well-scoped: no new syntax, no type system changes, just new builtins + provider wiring
- The `Request.Options` field mentioned in the design doc already exists but the design recommends dedicated `ResponseFormat`/`ResponseSchema` fields instead, which is cleaner
- Anthropic provider is the most complex (tool-use pattern) — tackle in M1 to de-risk early
