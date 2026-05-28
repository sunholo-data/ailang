# Sprint Plan: M-DOCPARSE-RESILIENCE-FIXES

**Design docs**:
- [m-ai-call-json-simple-result.md](m-ai-call-json-simple-result.md) (M1)
- [m-mcp-unit-param-binding.md](m-mcp-unit-param-binding.md) (M2)

**Target**: v0.23.0
**Status**: Approved by user (2026-05-28)
**Estimated**: ~1 day (~100 LOC impl + ~65 LOC tests + docs)
**Risk level**: Low (both localized, mirror existing patterns, no parser/type changes)

## Sprint Goal

Close the two prod-affecting gaps reported by `sunholo/docparse` via the agent messaging system:
1. Transient AI provider 5xx on the no-schema JSON path can't be retried/typed (bare HTTP 500).
2. Omitted MCP tool params crash stdlib with `Unit` instead of returning a clean error.

After this sprint, docparse can unblock its AI-resilience work AND its MCP skill's first-run auth flow stops crashing.

## Why bundle these two

- Both are P1, both prod-affecting, both ~0.5 day, both originated from the same source (docparse triage 2026-05-27/28).
- **Independent code areas** → zero merge conflict risk: M1 touches `internal/builtins/ai_step.go` + `std/ai.ail`; M2 touches `internal/apiserver/mcp.go`. Could even run as parallel sub-agents, but sequential is simple enough at this size.
- Single sprint = single review + single CHANGELOG section + one round of sprint-evaluator.

## Recent velocity

- 2026-05-27: v0.22.0 release + M-COORD-TAG-ROUTING-LASTMILE (596 LOC, evaluator PASS 81/100) — all in a day.
- This sprint is ~165 LOC total. Comfortably <1 day.

## Milestones

### M1 — `callJsonSimpleResult` (std/ai Result-variant) (~60 LOC + 30 test, ~3h)

**What:**
- [ ] Add `registerAICallJsonSimpleResult()` + `aiCallJsonSimpleResultImpl` in `internal/builtins/ai_step.go`, mirroring `registerAICallJsonResult` (NumArgs: 1, Effect: AI, Since: v0.23.0). Reuse `makeAICallResultType` (already `(string) -> Result[string, AIError] ! {AI}`) for the type builder — no new type-builder needed.
- [ ] Route provider errors through the existing `internal/ai/errors.go` AIError mapper (same as `aiCallJsonResultImpl`) — no panic on provider 5xx.
- [ ] Wire the new `register...()` into the same init sequence as `_ai_call_result` / `_ai_call_json_result`.
- [ ] Add `callJsonSimpleResult` export to `std/ai.ail` (right after `callJsonResult`).
- [ ] Regenerate `internal/pipeline/testdata/builtin_types.golden`.
- [ ] `examples/runnable/ai_call_json_simple_result.ail` + register in `examples/manifest.json`.
- [ ] Add `callJsonSimpleResult` to the active teaching prompt's AI-effect Result-variant list.

**Acceptance:**
- [ ] `callJsonSimpleResult("...")` returns `Ok(jsonString)` on success (byte-identical to `callJsonSimple`)
- [ ] Simulated provider 5xx returns `Err(AIError{retryable: true})`, no host crash
- [ ] `AIError.retryable` classification matches `callResult`'s for the same codes (shared mapper)
- [ ] New builtin in `builtin_types.golden` with signature `(string) -> Result[string, AIError] ! {AI}`
- [ ] `go test ./internal/builtins/ -run <new> -count=20` deterministic
- [ ] `examples/runnable/ai_call_json_simple_result.ail` runs under `ailang run --caps AI`

**Files:** `internal/builtins/ai_step.go` (+40), `std/ai.ail` (+6), test (+30), example (+20), `examples/manifest.json` (+3), prompt (+2)

### M2 — MCP omitted-param rejection (~40 LOC + 35 test, ~3h)

**What:**
- [ ] In `internal/apiserver/mcp.go::makeToolHandler`, replace the omit-blind `args[i] = argMap[name]` loop with presence-checked binding: collect missing/null param names, return `mcpError("missing required parameter(s): <names>")` before calling the engine. Iterate `paramNames` (stable slice) for deterministic ordering.
- [ ] **AUDIT (per design doc):** grep `internal/apiserver/routes.go::callFunction` (line 352, uses `ParamNames`/`ParamTypes`) for the same omit-blind binding. If present → fix in this milestone (unified fix per CLAUDE.md §3). If safe → document why in the commit.
- [ ] `docs/docs/guides/serve-api.md`: note omitted MCP params return a structured error (no optional-param support yet).

**Acceptance:**
- [ ] `tools/call` omitting `apiKey` → `missing required parameter(s): apiKey, requestId`, `isError=true`, no `_str_len` crash
- [ ] All-params-present call still succeeds (regression)
- [ ] Legacy positional `{"args":[...]}` form still works (regression)
- [ ] Type-agnostic: omitted `int` param → same structured error (not a string special-case)
- [ ] Explicit JSON `null` for a param → also rejected
- [ ] `go test ./internal/apiserver/ -run <new> -count=20` deterministic (error message in `paramNames` order)
- [ ] routes.go audit outcome documented (fixed-too OR confirmed-safe)

**Files:** `internal/apiserver/mcp.go` (+12), test (+35), `docs/docs/guides/serve-api.md` (+8), POSSIBLY `internal/apiserver/routes.go` (if audit finds the same bug)

### M3 — CHANGELOG + finalize (~15 LOC, ~0.5h)

**What:**
- [ ] Single CHANGELOG entry in `changelogs/v0.10-current.md` covering M1 + M2 (note both originated from docparse triage; reference msg IDs)
- [ ] Reply to both source messages (623811a0, b08617cb) confirming the shipped fixes
- [ ] Move both design docs + this plan from `planned/v0_23_0/` to `implemented/v0_23_0/`

**Acceptance:**
- [ ] CHANGELOG entry covers both fixes
- [ ] `make test` + `make lint` clean
- [ ] Design docs moved to implemented/

## Sequencing

M1 and M2 are fully independent (no shared files). Recommended order: M1 → M2 → M3 (sequential, simplest). Parallel sub-agents are viable but overkill at ~165 LOC.

## Total estimates

| Milestone | LOC (impl+test) | Hours | Risk |
|---|---:|---:|---|
| M1 callJsonSimpleResult | 60 + 30 | 3 | Low — mechanical mirror of callJsonResult |
| M2 MCP omit-rejection | 40 + 35 | 3 | Low-Med — the routes.go audit may widen scope by ~15 LOC |
| M3 CHANGELOG + finalize | 15 | 0.5 | Low |
| **Total** | **~165 + docs** | **~6.5h** | **Low** |

## Risk management

| Risk | Likelihood | Mitigation |
|---|---|---|
| routes.go `@route` path has the same omit→Unit bug (M2 audit) | Med | If found, fix both in M2 (unified fix); +~15 LOC. If the @route path binds differently, document why and scope M2 to MCP. |
| `makeAICallResultType` signature differs subtly from what callJsonSimpleResult needs | Low | Both are `(string) -> Result[string, AIError] ! {AI}` — confirmed by inspection. If the builder bakes a name/metadata that conflicts, write a thin `makeAICallSimpleResultType` (5 LOC). |
| AIError mapper has map-iteration nondeterminism surfacing in the -count=20 test | Low | Mapper is shared with callResult/callJsonResult which already pass deterministic tests; reuse verbatim. |
| Provider-5xx simulation in tests requires a live AI call | Low | Use the same test stub/mock the existing `_ai_call_json_result` tests use (check ai_step_test.go for the pattern). |

## Success criteria for the whole sprint

- [ ] M1 + M2 + M3 acceptance criteria all met
- [ ] `make test` + `make lint` clean; new tests deterministic at -count=20
- [ ] Both design docs + plan moved to `implemented/v0_23_0/`
- [ ] Both source messages replied to
- [ ] sprint-evaluator PASS (≥70)

## Out of scope

- serve-api `E_AI_CALL_ERROR → HTTP 502` auto-mapping (M1 design doc stretch — separate doc if pursued)
- AILANG optional-parameter / default-value syntax (M2 design doc — separate, much larger language feature)
- Fixing the underlying `callJson` large-response corruption bug (separate known issue)

---

**Sprint plan created**: 2026-05-28
**Created by**: sprint-planner skill (claude-opus-4-7)
**Approved by**: user
