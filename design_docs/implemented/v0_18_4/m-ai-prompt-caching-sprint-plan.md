# Sprint Plan: M-AI-PROMPT-CACHING (v0.18.4)

**Sprint ID**: M-AI-PROMPT-CACHING
**Target Version**: v0.18.4
**Design Doc**: [m-ai-prompt-caching.md](m-ai-prompt-caching.md)
**Estimated**: ~6 hours, ~210 LOC
**Risk Level**: Medium (touches `internal/ai/` request path; back-compat is the main risk surface)
**Created**: 2026-05-08

---

## Sprint Goal

Add a model-independent `cache_breakpoints` field to AILANG's `ai.Request` so callers can opt in to provider-side prompt caching. Anthropic stamps `cache_control` markers; OpenAI/Gemini emit a structured one-shot warning (their caching is automatic or async). Default behavior (`cache_breakpoints: []`) is bit-for-bit identical to current.

**Out of scope** (per design doc Non-Goals):
- motoko-internal config bug (issue #225 — BashExec storm)
- motoko-internal path-validation bug (msg `8ae01ad5` — sibling to #225)
- motoko opt-in itself (cross-repo change, separate PR after this lands)
- Gemini Context Caching API integration (Phase 2 / future design)

---

## Velocity Context

Last 3 days: 20 commits, large mixed-content changes (motoko hardening sprints v0.18.1-3 + sandbox diagnostics). Focused-feature commits typical: ~50-100 LOC + tests per milestone, completed in 1-2 hours each.

This sprint is short (210 LOC across 4 milestones) — should comfortably fit in a single working session.

---

## Milestone Breakdown

The design doc has 6 phases. Collapsing per user instruction to 4 sprint milestones:
- **M1** = Phase 1 (AILANG type + Request plumbing)
- **M2** = Phase 2 (Anthropic provider — the substantive work)
- **M3** = Phases 3+4 (OpenRouter routing + OpenAI/Gemini/Ollama no-op + warnings)
- **M4** = Phase 6 (Examples + docs + CHANGELOG; Phase 5 motoko opt-in deferred to cross-repo PR)

### M1 — Request type + plumbing

**Estimated**: 1 hour, ~40 LOC

**Description**: Add `CacheBreakpoint` type and `cache_breakpoints` field to both `std/ai.ail` (AILANG-side) and `internal/ai.Request` (Go-side). Thread through `_ai_step` builtin → `internal/runtime/ai_handler.go` → provider Request. The field defaults to `[]` (back-compat).

**Files**:
- `std/ai.ail` (+15 LOC) — `CacheBreakpoint` type, optional `cache_breakpoints` field, `stepWithRequest` wrapper
- `internal/ai/provider.go` (+10 LOC) — `CacheBreakpoint` Go type, `CacheBreakpoints []CacheBreakpoint` field on Request
- `internal/runtime/ai_handler.go` or wherever `_ai_step` lives (+10 LOC) — pass cache_breakpoints through

**Acceptance criteria**:
- `internal/ai.Request` has `CacheBreakpoints []CacheBreakpoint` field
- `std/ai.ail` exports `CacheBreakpoint` type with `position: string` and `ttl: string` fields
- `step(model, messages, tools)` 3-arg form still works unchanged (back-compat)
- New 4-arg / Request-builder form (`stepWithRequest(req)`) accepts cache_breakpoints
- Field round-trips through builtin → handler → provider Request
- All existing tests pass: `go test ./internal/ai/... ./internal/runtime/...`

**Dependencies**: none

---

### M2 — Anthropic provider implementation

**Estimated**: 2 hours, ~80 LOC

**Description**: When `req.CacheBreakpoints` contains a `{position:"system"}` entry, restructure the Anthropic system prompt from a string to a content-array with `cache_control: {type:"ephemeral"}`. Extract mapper to its own file so OpenRouter (M3) can reuse it. Phase 1 placement = system-only; tool_result and last_user deferred per design.

**Files**:
- `internal/ai/anthropic/cache.go` (+50 LOC, new) — `applyCacheHints(apiReq, breakpoints)` mapper
- `internal/ai/anthropic/step.go` (+30 LOC) — `stepContentBlock` gets optional `CacheControl`; `buildStepRequest` calls `applyCacheHints` after building messages; `stepMessagesRequest.System` becomes `interface{}` (string OR `[]systemBlock`) so the wire shape stays string when no hints
- `internal/ai/anthropic/cache_test.go` (+50 LOC, new) — unit tests: empty breakpoints → string system; system breakpoint → content-array with cache_control; verify wire JSON

**Acceptance criteria**:
- `applyCacheHints` with `[]` produces wire bytes identical to today (golden snapshot test)
- `applyCacheHints` with `{position:"system", ttl:"ephemeral"}` produces a request with `system: [{type:"text", text:"...", cache_control:{type:"ephemeral"}}]`
- Unknown position strings are warned-and-ignored, not errors
- `system: ""` (empty system prompt) + cache breakpoint = no-op (don't mark empty content)
- All existing `internal/ai/anthropic/*_test.go` tests pass without modification
- Specifically the v0.18.3 hybrid-tool tests in motoko_agent must still pass against the new code (verify via `make test` here covers any AILANG-side anthropic tests; full motoko verification deferred to motoko opt-in PR)

**Dependencies**: M1 (CacheBreakpoints field must exist on Request)

**Regression fixtures** (per Conflict Surface section of design doc):
1. `internal/ai/anthropic/step_test.go` — all unchanged, must pass
2. Snapshot test: `Request{...no cache_breakpoints}` → wire bytes match golden file

---

### M3 — Multi-provider routing + no-op + warnings

**Estimated**: 1.5 hours, ~70 LOC

**Description**: OpenRouter dispatches based on model-prefix (`anthropic/...` calls Anthropic mapper; `openai/...`/`google/...`/other = no-op + warning). OpenAI, Gemini, Ollama gain no-op handling with structured one-shot warnings via a shared `cache_warnings.go` helper. Ollama is silent (local model).

**Files**:
- `internal/ai/cache_warnings.go` (+30 LOC, new) — `WarnOnceCacheHintIgnored(provider, reason)` using `sync.Map[string]struct{}`
- `internal/ai/openrouter/step.go` (+25 LOC) — model-prefix sniffing, dispatch to Anthropic mapper or no-op-warn
- `internal/ai/openai/step.go` (+5 LOC) — call `WarnOnceCacheHintIgnored("openai", "auto_cache")` if hints non-empty
- `internal/ai/gemini/step.go` (+5 LOC) — call `WarnOnceCacheHintIgnored("gemini", "no_explicit_api")` if hints non-empty
- `internal/ai/ollama/step.go` (+5 LOC) — silent no-op
- Tests: `cache_warnings_test.go` (concurrent calls produce 1 warning), `openrouter/cache_routing_test.go` (prefix dispatch correctness)

**Acceptance criteria**:
- Concurrent `WarnOnceCacheHintIgnored("openai", "auto_cache")` from N goroutines emits exactly 1 log line
- OpenRouter model `anthropic/claude-3-5-haiku` + system breakpoint → wire body contains `cache_control` (matches Anthropic-direct)
- OpenRouter model `openai/gpt-4o-mini` + system breakpoint → wire body does NOT contain `cache_control`, warning emitted once
- OpenRouter model with unrecognized prefix → no-op + once-per-session warning
- All existing `internal/ai/{openai,gemini,openrouter,ollama}/*_test.go` tests pass without modification
- `make test` clean

**Dependencies**: M2 (Anthropic mapper must be importable from `internal/ai/anthropic/cache.go`)

---

### M4 — Examples + docs + CHANGELOG

**Estimated**: 1 hour, ~30 LOC + docs

**Description**: Runnable example, docs guide update, CHANGELOG entry, design doc status flip. Acceptance gate verification (does NOT include motoko-side opt-in, which is a separate cross-repo PR).

**Files**:
- `examples/runnable/ai_caching.ail` (+40 LOC, new) — 3-turn loop demonstrating cache hits (gated behind `ANTHROPIC_API_KEY` env var; falls back to no-op message if unset)
- `examples/manifest.json` (+5 LOC) — register the new example
- `docs/docs/guides/custom-ai-providers.md` (+30 LOC) — `cache_breakpoints` documentation + per-provider behavior table
- `changelogs/v0.10-current.md` (+15 LOC) — v0.18.4 entry
- `design_docs/planned/v0_18_4/m-ai-prompt-caching.md` → moved to `design_docs/implemented/v0_18_4/` with status flip (handled by `finalize_sprint.sh`)

**Acceptance criteria**:
- `ailang run examples/runnable/ai_caching.ail` runs cleanly with `ANTHROPIC_API_KEY` set
- `make verify-examples` passes
- CHANGELOG entry under v0.18.4 lists the user-visible API additions
- Docs guide contains the per-provider behavior table from design doc
- Design doc moved to `implemented/v0_18_4/` with status updated to "Implemented"

**Dependencies**: M1, M2, M3

---

## Day-by-Day Plan

**Single-day sprint (target: today, 2026-05-08, evening session)**

| Time | Milestone | Work |
|------|-----------|------|
| Hour 1 | M1 | Request type plumbing + tests |
| Hour 2-3 | M2 | Anthropic mapper + cache_test.go + golden-snapshot back-compat test |
| Hour 4-5 | M3 | OpenRouter routing + warnings + 4 provider no-op patches + tests |
| Hour 6 | M4 | Example + docs + CHANGELOG + finalize |

**Total: ~6 hours focused work**

---

## Success Metrics

- All 4 milestones pass acceptance criteria
- `make test` and `make lint` clean
- `make verify-examples` passes (new ai_caching.ail example runs)
- 4 new test files (`cache_test.go`, `cache_warnings_test.go`, `cache_routing_test.go`, plus the snapshot test)
- All existing `internal/ai/*/step_test.go` tests pass without modification (the back-compat assertion)
- Documentation: `docs/docs/guides/custom-ai-providers.md` updated with cache_breakpoints section
- CHANGELOG entry under v0.18.4

**Post-sprint validation gate** (NOT a sprint milestone — separate cross-repo work):
- motoko opt-in lands in `motoko_agent/src/core/agent_loop_v2.ail`
- Re-run 3-harness smoke comparison shows nonzero `cache_read` for motoko sessions
- This is what closes the loop on the design-doc acceptance gate; the AILANG sprint itself is complete once M1-M4 pass

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Anthropic system content-array rejected by older `apiVersion` | Medium | Add a defensive check at request build time; fall back to string form + warning if `apiVersion < "2023-06-01"` |
| Bedrock validates strict; we test only Anthropic-direct | Low | Phase 1 only places cache_control on `system` (the stable surface); skip tool_result placement until v0.18.5+ Bedrock-explicit testing |
| Existing tests have hidden golden-bytes dependencies | Medium | M2 includes a snapshot test that pins the no-cache wire bytes byte-for-byte before any other change |
| OpenRouter changes its model-string prefix convention | Low | Single greppable place to update; once-per-session warning surfaces unknown prefixes loudly |
| Sprint time underestimated due to AI-handler plumbing complexity | Low | M1 estimate of 1h includes the runtime handler; if it balloons, reduce M4 example scope (drop runnable, keep docs) |

---

## References

- Design doc: [m-ai-prompt-caching.md](m-ai-prompt-caching.md)
- Conflict Surface section: explicitly lists 5 fixture programs that must keep working
- Cross-repo follow-up (after sprint completes): motoko_agent PR adding `cache_breakpoints` opt-in to `agent_loop_v2.ail`
- Related external bugs (NOT in scope here): arniwesth/motoko_agent #225, motoko-explore msg `8ae01ad5`

---

**Document created**: 2026-05-08
