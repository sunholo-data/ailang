# M-SERVEAPI-ROUTE-NOT-FOUND Sprint Plan

**Design Doc**: [m-serveapi-route-not-found.md](m-serveapi-route-not-found.md)
**Target**: v0.11.0
**Duration**: 1 day (~4 hours)
**Risk**: Low
**Created**: 2026-04-09
**Source**: Agent message `566f6da6` from docparse

## Sprint Summary

**Goal**: Replace the misleading `"module X not loaded"` router 404 in `serve-api` with a typed `ROUTE_NOT_FOUND` error that tells AI agents exactly which route they meant and what routes exist.

**Key deliverables**:
1. Typed error envelope (`error_detail` field) with backward-compat flat `error` string
2. Route suggestion algorithm (longest-common-prefix)
3. 3-way discrimination in the catch-all handler (route-driven vs legacy module/func)
4. Tests + docs + reply to docparse

**Risk level**: **Low** — small, localized change in `internal/apiserver/`. Backward-compatible envelope (flat `error` field preserved). Zero changes to AILANG semantics or stdlib.

**Why one day**: All 4 phases are small, independent, well-scoped. The design doc resolved all design decisions up front. No cross-module impact.

## Current Status

**From design doc**:
- Problem identified: [internal/apiserver/handler.go:85-90](../../../internal/apiserver/handler.go#L85-L90) emits `"module %q not loaded"` on ANY 404, even when the real cause is an unmatched `@route`.
- Root cause located: the fallback `findRouteByPath` call exists but on failure reuses the module-loading error string.
- Fix scope confirmed: `internal/apiserver/` only. No parser, elaborator, runtime, or stdlib changes.

**Systemic check** (from design doc): confirmed this is part of a pattern with [m-dx-serve-api-error-status.md](m-dx-serve-api-error-status.md) and [m-dx-serve-api-coercion.md](m-dx-serve-api-coercion.md). The `error_detail` envelope introduced here will be reused by those sprints.

## Milestone Breakdown

### M1_ERROR_ENVELOPE — Typed error envelope

**Goal**: Create a shared typed error envelope that all router error sites can use.

**Estimated LOC**: ~80 impl + ~40 tests = ~120 LOC

**Files**:
- **New**: `internal/apiserver/errors.go` (~80 LOC)
- **Modified**: `internal/apiserver/handler.go` — extend `FunctionCallResponse` with `ErrorDetail *RouterErrorDetail` (~5 LOC)

**Tasks**:
- [ ] Define `RouterErrorDetail` struct with `Code`, `Message`, `Retryable`, `SuggestedFix`, `AvailableRoutes` fields
- [ ] Define constants: `ErrCodeRouteNotFound`, `ErrCodeModuleNotLoaded`, `ErrCodeFunctionNotFound`, `ErrCodeMethodNotAllowed`
- [ ] Implement `writeRouterError(w, status int, code, msg, suggestedFix string, available []string)` helper
- [ ] Extend `FunctionCallResponse` with `ErrorDetail *RouterErrorDetail` (json tag `error_detail,omitempty`)
- [ ] Unit test: envelope marshals with `error_detail` when set, omits when nil, flat `error` mirrors `error_detail.Message`

**Acceptance criteria**:
- `writeRouterError` compiles and produces JSON matching Example 1 in the design doc
- Flat `error` string is always populated (backward compat)
- `error_detail` is omitted when nil

**Dependencies**: None

**Estimated time**: ~1h

---

### M2_ROUTE_SUGGEST — Route suggestion algorithm

**Goal**: Given a request `(method, path)` and the registered route table, return a `suggested_fix` string and bounded `available_routes` list.

**Estimated LOC**: ~70 impl + ~150 tests = ~220 LOC

**Files**:
- **New**: `internal/apiserver/route_suggest.go` (~70 LOC)
- **New**: `internal/apiserver/route_suggest_test.go` (~150 LOC)

**Tasks**:
- [ ] Implement `suggestRoutes(method, path string, routes []RouteEntry) (suggestedFix string, available []string)`
- [ ] Algorithm: longest-common-prefix over `/`-split segments; a route is "close" if it shares ≥ half the request path's segments AND matches method (fall back to any method)
- [ ] Format `suggested_fix`: `"Did you mean {METHOD} {PATH}?"` for single close match, `"Did you mean one of: ..."` (up to 3) for multi
- [ ] `available`: routes sharing first 2 path segments, cap at 10; fall back to all routes (capped) if <2 match
- [ ] Unit tests:
  - `TestSuggestRoutes_ExactMissWithClose` — `/api/v1/auth/device/token` vs `/api/v1/auth/device/poll` → suggests `/poll`
  - `TestSuggestRoutes_NoCloseMatch` — totally unrelated path → empty `suggested_fix`, `available` populated
  - `TestSuggestRoutes_WrongMethod` — path matches but method doesn't → falls back to any-method suggestion
  - `TestSuggestRoutes_EmptyRegistry` → empty `suggested_fix`, empty `available`
  - `TestSuggestRoutes_SingleRoute` — one registered route → always suggests it if prefix overlaps

**Acceptance criteria**:
- All 5 unit tests pass
- `available_routes` never exceeds 10 entries
- Docparse scenario (from design doc Example 1) produces `"Did you mean POST /api/v1/auth/device/poll?"`

**Dependencies**: M1 (uses `RouteEntry` already, but no envelope dependency — can parallelize)

**Estimated time**: ~1h

---

### M3_HANDLER_REWRITE — 3-way discrimination in handler

**Goal**: Rewrite the 404 branch in `handleFunctionCall` to emit the correct typed error based on server configuration and request shape. Also convert the 2 sibling 404 sites and the `registerCustomRoutes` method-not-allowed error.

**Estimated LOC**: ~70 LOC changed

**Files**:
- **Modified**: `internal/apiserver/handler.go` — rewrite 404 branch + convert sibling errors (~60 LOC changed)
- **Modified**: `internal/apiserver/routes.go` — convert `registerCustomRoutes` method-not-allowed (~10 LOC changed)

**Tasks**:
- [ ] Rewrite [handler.go:85-90](../../../internal/apiserver/handler.go#L85) 404 branch with 3-way rule:
  - **Case A**: `s.getCustomRoutes()` returns ≥1 route AND none matched → `ROUTE_NOT_FOUND` with `suggestRoutes` output
  - **Case B**: path parsed as `/api/{module}/{func}`, module loaded, function missing → `FUNCTION_NOT_FOUND` (existing behavior, new envelope) — lines 102-122
  - **Case C**: server has zero `@route`s AND module not loaded → `MODULE_NOT_LOADED` (legacy behavior, new envelope)
- [ ] Convert [handler.go:106](../../../internal/apiserver/handler.go#L106) function-not-found error to use `writeRouterError` with `FUNCTION_NOT_FOUND` code
- [ ] Convert [handler.go:116](../../../internal/apiserver/handler.go#L116) hidden-export error to use `writeRouterError` with `FUNCTION_NOT_FOUND` code (same public behavior)
- [ ] Convert [routes.go:329](../../../internal/apiserver/routes.go#L329) method-not-allowed error to `writeRouterError` with `METHOD_NOT_ALLOWED` code
- [ ] Convert [handler.go:33](../../../internal/apiserver/handler.go#L33) top-level method-not-allowed error similarly

**Acceptance criteria**:
- `make build` passes
- Manual curl against a local serve-api reproducing the docparse scenario returns the envelope from design doc Example 1
- Legacy `/api/{module}/{func}` dispatch on a no-`@route` server returns `MODULE_NOT_LOADED` envelope (Example 2)
- No plain-text error from the router layer reaches the wire (grep for `writeJSON.*FunctionCallResponse{Error:` should find zero remaining sites in the modified files)

**Dependencies**: M1 (envelope), M2 (suggestions)

**Estimated time**: ~1h

---

### M4_TESTS_DOCS_REPLY — Integration tests, docs, and message reply

**Goal**: Lock the behavior with integration tests, document the new envelope, and acknowledge the bug report.

**Estimated LOC**: ~120 test + ~40 docs = ~160 LOC

**Files**:
- **Modified**: nearest existing handler test file (find via `grep -l handleFunctionCall internal/apiserver/*_test.go`) — ~120 LOC of new tests
- **Modified**: `docs/docs/guides/serve-api.md` — error envelope docs (~40 LOC)
- **Modified**: `CHANGELOG.md` — one line under Unreleased → Changed

**Tasks**:
- [ ] `TestHandleFunctionCall_RouteNotFound_WithSuggestion`: server with `@route`s, unmatched path close to a real route → `error_detail.code == "ROUTE_NOT_FOUND"`, `suggested_fix` populated
- [ ] `TestHandleFunctionCall_RouteNotFound_NoSuggestion`: server with `@route`s, unmatched path with no close match → empty `suggested_fix`, non-empty `available_routes`
- [ ] `TestHandleFunctionCall_ModuleNotLoaded_Legacy`: server with zero `@route`s, unmatched module → `error_detail.code == "MODULE_NOT_LOADED"`, flat `error` matches historical text
- [ ] `TestHandleFunctionCall_FunctionNotFound`: loaded module, missing function → `error_detail.code == "FUNCTION_NOT_FOUND"`
- [ ] Update `docs/docs/guides/serve-api.md`:
  - New "Error responses" subsection documenting envelope shape
  - Error codes table: `ROUTE_NOT_FOUND`, `MODULE_NOT_LOADED`, `FUNCTION_NOT_FOUND`, `METHOD_NOT_ALLOWED`
  - Note that envelope rollout is incremental (more codes coming in sibling sprints)
- [ ] Add CHANGELOG entry: `- serve-api: router 404 errors now return typed 'error_detail' envelope with 'ROUTE_NOT_FOUND' code and did-you-mean suggestions (M-SERVEAPI-ROUTE-NOT-FOUND)`
- [ ] `make test` passes
- [ ] `make verify-examples` passes
- [ ] Send reply to docparse: `ailang messages send docparse "Fixed in v0.11.0 — router now returns typed ROUTE_NOT_FOUND with did-you-mean" --title "FIXED: serve-api router 404 now typed" --from "ailang-maintainer"`
- [ ] `ailang messages ack 566f6da6-0c4e-439b-b701-05e945ebb5c8`

**Acceptance criteria**:
- All 4 integration tests pass
- `make test` and `make verify-examples` green
- Docs page renders locally without broken links
- Reply message sent and original ack'd

**Dependencies**: M1, M2, M3

**Estimated time**: ~1h

## Timeline

**Day 1** (4 hours, single session):

| Hour | Milestone | Activity |
|------|-----------|----------|
| 0:00–1:00 | M1 + M2 (parallel-safe) | Create `errors.go` and `route_suggest.go` + suggest tests |
| 1:00–2:00 | M2 finish + M3 start | Finish suggestion tests; begin handler rewrite |
| 2:00–3:00 | M3 | Handler rewrite + sibling error conversions + manual curl verification |
| 3:00–4:00 | M4 | Integration tests, docs, changelog, reply to docparse |

## Success Metrics

**Code quality**:
- [ ] `make test` passes (no regressions)
- [ ] `make verify-examples` passes
- [ ] `make lint` passes
- [ ] No new `writeJSON(w, *, FunctionCallResponse{Error: "..."})` without `ErrorDetail` in modified files

**Behavior**:
- [ ] Docparse repro (`POST /api/v1/auth/device/token`) returns new envelope with `suggested_fix` pointing to `/poll`
- [ ] Legacy module/func dispatch still works on a no-`@route` server
- [ ] Flat `error` field still populated (no client breakage)

**Documentation**:
- [ ] `docs/docs/guides/serve-api.md` updated with error codes table
- [ ] `CHANGELOG.md` updated under Unreleased → Changed
- [ ] Reply sent to docparse via `ailang messages send`
- [ ] Original message ack'd

## Dependencies & Blockers

**External**: None
**Internal sprint dependencies**: None — this sprint is independent of M-BYTECODE-MULTIMODULE (current active sprint)
**Sibling sprints** (will reuse envelope but don't block): M-DX-SERVE-API-ERROR-STATUS, M-DX-SERVE-API-COERCION

## Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Existing clients parse flat `error` string and break on shape change | Low | Flat `error` preserved, mirrors `error_detail.Message`. Covered by `TestHandleFunctionCall_ModuleNotLoaded_Legacy`. |
| Suggestion algorithm surfaces misleading suggestions | Low | Only emit `suggested_fix` when LCP ≥ half request segments. `TestSuggestRoutes_NoCloseMatch` locks this. |
| Sibling sprints (ERROR-STATUS, COERCION) want a different envelope shape | Medium | Envelope is deliberately minimal (code, message, retryable, suggested_fix, available_routes). If siblings need more fields they can extend `RouterErrorDetail` additively. |
| Hidden-export (`@noexpose`) case now leaks more info via error code | Low | Keep same `FUNCTION_NOT_FOUND` code (not `FUNCTION_HIDDEN`) so `@noexpose` remains indistinguishable from genuinely missing. Design doc decision. |

## Open Questions

None — all design decisions resolved in the design doc's Design Freeze section.

## Handoff

After user approval, send handoff to sprint-executor:

```bash
ailang agent send sprint-executor '{
  "type": "plan_ready",
  "correlation_id": "sprint_M-SERVEAPI-ROUTE-NOT-FOUND_20260409",
  "sprint_id": "M-SERVEAPI-ROUTE-NOT-FOUND",
  "plan_path": "design_docs/planned/v0_11_0/m-serveapi-route-not-found-sprint-plan.md",
  "progress_path": ".ailang/state/sprints/sprint_M-SERVEAPI-ROUTE-NOT-FOUND.json",
  "estimated_duration": "1 day (4 hours)",
  "total_loc_estimate": 570,
  "risk_level": "low"
}'
```

---

**SPRINT_PLAN_PATH**: `design_docs/planned/v0_11_0/m-serveapi-route-not-found-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-SERVEAPI-ROUTE-NOT-FOUND.json`
