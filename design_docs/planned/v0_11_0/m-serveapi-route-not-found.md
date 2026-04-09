# M-SERVEAPI-ROUTE-NOT-FOUND: Typed ROUTE_NOT_FOUND error for unmatched serve-api paths

**Status**: Planned
**Target**: v0.11.0
**Priority**: P1 (AI agent DX — agents get derailed debugging wrong layer)
**Estimated**: 0.5 days
**Dependencies**: None
**Milestone ID**: M-SERVEAPI-ROUTE-NOT-FOUND
**Created**: 2026-04-09
**Source**: Agent message `566f6da6` from docparse (2026-04-09)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No determinism impact |
| A2: Replayability | 0 | No trace impact |
| A3: Effect Legibility | +1 | Surface the actual failure layer (routing) instead of misattributing to module loading |
| A4: Explicit Authority | 0 | No authority changes |
| A5: Bounded Verification | +1 | Error response carries `available_routes` — client can locally verify which URL to call |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +2 | Primary motivation: AI agents debug from structured errors. Current message misdirects them to imports/lock files |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | 0 | No cost changes |
| A10: Composability | +1 | Reuses existing typed-error envelope shape from other serve-api errors |
| A11: Structured Failure | +2 | Converts a plain-text misleading error into a typed, code-bearing, self-describing failure |
| A12: System Boundary | +1 | HTTP boundary now correctly communicates "not a route" vs "module missing" |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

- [x] A1: No nondeterminism introduced (route table is deterministic snapshot at startup)
- [x] A3: Makes failure cause MORE visible, not less
- [x] A4: No ambient access
- [x] A7: Core motivation is machine-readable error signals

## Problem Statement

When a request hits `serve-api` at a path that matches no registered `@route` and no valid `/api/{module}/{function}` dispatch, the catch-all handler at [internal/apiserver/handler.go:85-90](internal/apiserver/handler.go#L85-L90) returns:

```json
{"error":"module \"v1/auth/device\" not loaded","module":"v1/auth/device","func":"token","elapsed_ms":0}
```

**This is actively misleading.** The failure was "no route matches POST /api/v1/auth/device/token" but the error surfaces as a **module loading** problem — a completely different layer. AI agents seeing this error start debugging imports and lock files instead of fixing the URL.

**Current control flow** ([internal/apiserver/handler.go:42-91](internal/apiserver/handler.go#L42-L91)):
1. Strip `/api/` prefix → `v1/auth/device/token`
2. Split on last `/` → `modulePath="v1/auth/device"`, `funcName="token"`
3. `findModuleByRelPath(modulePath)` fails
4. Fallback: `findRouteByPath(r.URL.Path)` also fails
5. Return `"module %q not loaded"` — **but the failure is that neither path matched any route**

**Concrete repro** (from message 566f6da6):
```bash
curl -X POST https://ailang-dev-docparse-api-ejjw6zt3bq-ew.a.run.app/api/v1/auth/device/token \
  -H 'Content-Type: application/json' -d '{}'
# → {"error":"module \"v1/auth/device\" not loaded",...}
# Real endpoint is /api/v1/auth/device/poll
```

**Impact**:
- **AI agents**: docparse user reported an agent called `/token` (RFC 8628 OAuth convention), got "module not loaded", went down a wrong rabbit hole before finding `/poll`. Cost: multiple wasted turns.
- **General**: Any serve-api app that exclusively uses `@route` will hit this whenever a client typos a path or follows convention from a different framework.
- **DX inconsistency**: docparse (and other serve-api consumers) return typed AI-first errors (`{"error": {"code": "...", "message": "...", "retryable": false, "suggested_fix": "..."}}`) for every other failure mode. This router-level 404 bypasses that envelope entirely because it comes from the Go layer, not user code.

### Systemic Check

This is **not** a one-off. It's the same pattern as [M-DX-SERVE-API-ERROR-STATUS](m-dx-serve-api-error-status.md) (HTTP status for Err) and [M-DX-SERVE-API-COERCION](m-dx-serve-api-coercion.md) (input coercion errors): **serve-api's transport layer produces low-level errors that leak implementation details (module/func dispatch internals) when they should produce typed, self-describing errors at the HTTP boundary.**

**Unified principle**: every error emitted by the serve-api router layer should be a structured AI-first error with `code`, `message`, `retryable`, `suggested_fix`. No plain-text errors from router internals should reach the wire.

**Audit target** during implementation: search `internal/apiserver/handler.go` and `internal/apiserver/routes.go` for all `writeJSON(w, http.StatusXXX, FunctionCallResponse{Error: "..."})` call sites. Ensure each either (a) has a meaningful typed code in this sprint or (b) is explicitly scheduled for a sibling sprint.

## Goals

**Primary Goal**: Replace the misleading `"module X not loaded"` router 404 with a typed `ROUTE_NOT_FOUND` error that tells AI agents exactly what went wrong and how to fix it.

**Success Metrics**:
- Hitting an unmatched path returns `error_detail.code == "ROUTE_NOT_FOUND"` with `available_routes` populated.
- When there is a similar registered route (longest-common-prefix match), the response includes a `suggested_fix` naming the closest route(s).
- The word "module" no longer appears in the error text for paths that were targeting an `@route`.
- Zero regressions: existing `/api/{module}/{func}` dispatch on a legacy (no-`@route`) server still returns a `MODULE_NOT_LOADED`-coded error.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Error envelope shape — add nested `error_detail` while keeping flat `error` | All serve-api error sites will converge on this. Sets precedent for sibling sprints (ERROR-STATUS, COERCION). | human | design | high |
| When to emit `ROUTE_NOT_FOUND` vs `MODULE_NOT_LOADED` | Determines whether existing `/api/{mod}/{func}` users see a behavior change | human | design | med |
| Suggestion algorithm — Levenshtein vs longest-common-prefix vs list-all | Affects response size and suggestion quality | agent | design | low |
| Cap on `available_routes` (initial: 10) | Response size vs debuggability | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **Error envelope**: adopt nested `{error_detail: {code, message, retryable, suggested_fix, available_routes}}` for this sprint. **Backward compat**: keep top-level `error` string populated with `error_detail.message` so existing clients don't break. Sibling sprints (ERROR-STATUS, COERCION) will reuse this envelope.
- [ ] **Dispatch discrimination rule**: emit `ROUTE_NOT_FOUND` when the request path does not match any registered `@route` AND the server has at least one `@route` registered (i.e., this is a route-driven deployment). Emit `MODULE_NOT_LOADED` only when the server has zero `@route`s registered (legacy module/func-only deployment) OR when the parsed module path is a valid `findModuleByRelPath` lookup shape but the module isn't loaded.

## Solution Design

### Overview

Restructure the catch-all handler's 404 path in [internal/apiserver/handler.go:85](internal/apiserver/handler.go#L85) to:
1. Detect whether the request was targeting an `@route` or a legacy module/function dispatch.
2. Emit a typed error with the correct `code` and a suggestion derived from the registered route table.

### Architecture

**Components**:

1. **Typed error envelope** (new): shared helper `writeRouterError(w, status, code, msg, suggestedFix, availableRoutes)` in `internal/apiserver/errors.go` (new file). Produces:
   ```json
   {
     "error": "No route registered for POST /api/v1/auth/device/token",
     "error_detail": {
       "code": "ROUTE_NOT_FOUND",
       "message": "No route registered for POST /api/v1/auth/device/token",
       "retryable": false,
       "suggested_fix": "Did you mean POST /api/v1/auth/device/poll?",
       "available_routes": [
         "POST /api/v1/auth/device",
         "POST /api/v1/auth/device/poll",
         "POST /api/v1/auth/device/approve"
       ]
     },
     "module": "",
     "func": "",
     "elapsed_ms": 0
   }
   ```
   The flat `error` field stays (mirrors `error_detail.message`) for backward compatibility.

2. **Route suggestion** (new): `suggestRoutes(method, path string, routes []RouteEntry) (suggestedFix string, available []string)` in `internal/apiserver/route_suggest.go`.
   - **Algorithm**: longest-common-prefix over path segments. A route is "close" if it shares at least half the request path's segments as a prefix AND matches the method (fall back to any method if no same-method matches).
   - **`suggested_fix`**: populated only when at least one close match exists. Format: `"Did you mean {METHOD} {PATH}?"` (single) or `"Did you mean one of: ..., ..."` (multi, up to 3).
   - **`available`**: all routes sharing the first 2 path segments (e.g., `/api/v1/auth/*`), capped at 10. If fewer than 2 segments match, fall back to all routes (still capped at 10).

3. **Handler dispatch fix** ([handler.go:85-90](internal/apiserver/handler.go#L85)): replace the single fallback branch with a 3-way discrimination:
   - **Case A** — server has `@route`s registered, and the path matches none → `ROUTE_NOT_FOUND` with suggestions.
   - **Case B** — path parses as `/api/{module}/{func}`, module loaded, function missing → existing "function not found" path, now wrapped in `error_detail: {code: "FUNCTION_NOT_FOUND", ...}`.
   - **Case C** — server has zero `@route`s (legacy dispatch-only), module not loaded → `MODULE_NOT_LOADED` with the existing message, wrapped in the new envelope.

### Implementation Plan

**Phase 1: Typed error envelope** (~1h)
- [ ] Create `internal/apiserver/errors.go` with `RouterErrorDetail` struct and `writeRouterError(w, status int, code, msg, suggestedFix string, available []string)` helper.
- [ ] Define error code constants: `ErrCodeRouteNotFound`, `ErrCodeModuleNotLoaded`, `ErrCodeFunctionNotFound`, `ErrCodeMethodNotAllowed`.
- [ ] Extend `FunctionCallResponse` in [handler.go:21](internal/apiserver/handler.go#L21) with optional `ErrorDetail *RouterErrorDetail` field (`json:"error_detail,omitempty"`).

**Phase 2: Route suggestion** (~1h)
- [ ] Create `internal/apiserver/route_suggest.go` with `suggestRoutes`.
- [ ] Longest-common-prefix over segments; cap `available` at 10.
- [ ] Unit tests with fixtures: exact miss w/ close match, exact miss w/o close match, wrong method, no registered routes, single registered route.

**Phase 3: Handler rewrite** (~1h)
- [ ] Rewrite the 404 branch in `handleFunctionCall` ([handler.go:85](internal/apiserver/handler.go#L85)) using the 3-way discrimination rule. Use `s.getCustomRoutes()` (already exists, [routes.go:284](internal/apiserver/routes.go#L284)) to detect case A vs C.
- [ ] Convert the sibling 404s in the same handler (function-not-found at [handler.go:106](internal/apiserver/handler.go#L106), hidden-export at [handler.go:116](internal/apiserver/handler.go#L116)) to use `writeRouterError` with `FUNCTION_NOT_FOUND` code.
- [ ] Audit `routes.go` `registerCustomRoutes` handler ([routes.go:329](internal/apiserver/routes.go#L329)) — method-not-allowed error uses plain text; convert to envelope with `METHOD_NOT_ALLOWED` code.

**Phase 4: Tests + docs** (~1h)
- [ ] Add `TestHandleFunctionCall_RouteNotFound` in the nearest existing handler test file. Cases:
  - Server with `@route`s only → unmatched path → `ROUTE_NOT_FOUND` + suggestions.
  - Server with close match → `suggested_fix` populated.
  - Server with no close match → `suggested_fix` empty, `available_routes` still populated.
- [ ] Add `TestHandleFunctionCall_ModuleNotLoaded_Legacy` locking the legacy behavior on a server with zero `@route`s.
- [ ] Update `docs/docs/guides/serve-api.md` with the new error envelope shape and error codes table.
- [ ] Add a line to `CHANGELOG.md` under Unreleased → Changed.
- [ ] Send a reply `ailang messages` to docparse acknowledging the fix.

### Files to Modify/Create

**New files**:
- `internal/apiserver/errors.go` — typed error envelope + codes (~80 LOC)
- `internal/apiserver/route_suggest.go` — suggestion algorithm (~70 LOC)
- `internal/apiserver/route_suggest_test.go` — unit tests (~150 LOC)

**Modified files**:
- `internal/apiserver/handler.go` — rewrite 404 branch + convert sibling errors (~60 LOC changed)
- `internal/apiserver/routes.go` — convert `registerCustomRoutes` method-not-allowed (~10 LOC)
- `internal/apiserver/handler_test.go` (or nearest existing test file) — new tests (~120 LOC)
- `docs/docs/guides/serve-api.md` — error envelope docs (~40 LOC)
- `CHANGELOG.md` — one line under Unreleased → Changed

## Examples

### Example 1: Unmatched path with close match (the docparse repro)

**Before**:
```
POST /api/v1/auth/device/token
→ 404
{"error":"module \"v1/auth/device\" not loaded","module":"v1/auth/device","func":"token","elapsed_ms":0}
```

**After**:
```
POST /api/v1/auth/device/token
→ 404
{
  "error": "No route registered for POST /api/v1/auth/device/token",
  "error_detail": {
    "code": "ROUTE_NOT_FOUND",
    "message": "No route registered for POST /api/v1/auth/device/token",
    "retryable": false,
    "suggested_fix": "Did you mean POST /api/v1/auth/device/poll?",
    "available_routes": [
      "POST /api/v1/auth/device",
      "POST /api/v1/auth/device/poll",
      "POST /api/v1/auth/device/approve"
    ]
  },
  "module": "",
  "func": "",
  "elapsed_ms": 0
}
```

### Example 2: Legacy module/func dispatch (unchanged semantics, new envelope)

**Before**:
```
POST /api/ecommerce/api/handlers/nonexistent
→ 404
{"error":"module \"ecommerce/api/handlers\" not loaded",...}
```

**After**:
```
POST /api/ecommerce/api/handlers/nonexistent
→ 404
{
  "error": "module \"ecommerce/api/handlers\" not loaded",
  "error_detail": {
    "code": "MODULE_NOT_LOADED",
    "message": "module \"ecommerce/api/handlers\" not loaded",
    "retryable": false,
    "suggested_fix": "Ensure the module is reachable from an --entry file or passed via --load"
  },
  "module": "ecommerce/api/handlers",
  "func": "nonexistent",
  "elapsed_ms": 0
}
```

## Success Criteria

- [ ] `TestHandleFunctionCall_RouteNotFound` passes: unmatched path with registered routes returns `error_detail.code == "ROUTE_NOT_FOUND"`.
- [ ] `TestHandleFunctionCall_RouteNotFound_Suggestion` passes: close match triggers `suggested_fix`.
- [ ] `TestHandleFunctionCall_ModuleNotLoaded_Legacy` passes: bare module/func dispatch on a server with no `@route`s still returns `MODULE_NOT_LOADED`.
- [ ] `TestSuggestRoutes_*` unit tests pass (≥4 cases: exact miss, prefix match, wrong method, empty registry).
- [ ] Manual curl against a local serve-api with the docparse scenario returns the new envelope.
- [ ] `make test` and `make verify-examples` pass.
- [ ] `docs/docs/guides/serve-api.md` documents `ROUTE_NOT_FOUND` and the error envelope shape.
- [ ] Reply message sent to docparse via `ailang messages send`.

## Testing Strategy

**Unit tests**:
- `suggestRoutes` — exact miss, single close match, multiple close matches, method mismatch, empty registry.
- Error envelope marshaling — `error_detail` omitted when nil, populated when set, flat `error` mirrors `error_detail.message`.

**Integration tests** (in `handler_test.go` or nearest existing):
- Server with only `@route` endpoints → unmatched path → `ROUTE_NOT_FOUND` with suggestions.
- Server with only module/func dispatch (no `@route`) → unmatched module → `MODULE_NOT_LOADED` (legacy behavior, new envelope).
- Server with both → unmatched path with no close route match → `ROUTE_NOT_FOUND` with `available_routes` but empty `suggested_fix`.
- Mixed server → loaded module, missing function → `FUNCTION_NOT_FOUND` (existing path, new envelope).

**Manual testing**:
- Rebuild serve-api locally, curl the docparse scenario path, verify envelope shape matches Example 1.

## Deferred Decisions

- **Levenshtein vs longest-common-prefix for suggestions**: agent may choose. LCP is the default; Levenshtein may be added later if LCP misses obvious typos.
- **Cap on `available_routes`** (currently 10): agent may tune based on typical app sizes.
- **Whether to emit `error_detail` on non-error response paths**: out of scope — only error paths get the envelope in this sprint.
- **Migration of 200-OK-with-Err cases** to use `error_detail`: deferred to [M-DX-SERVE-API-ERROR-STATUS](m-dx-serve-api-error-status.md).

## Non-Goals

- **Full error envelope migration across ALL serve-api error paths**: this sprint covers router/dispatch 404s and the adjacent MethodNotAllowed. Coercion errors, runtime panics, and `Result.Err` HTTP status are separate sprints.
- **Removing the legacy `/api/{module}/{func}` dispatch**: out of scope. It's still supported for non-`@route` users.
- **Structured error codes in AILANG user source** (i.e., letting user code emit `ROUTE_NOT_FOUND`): this sprint adds codes at the Go router layer only.
- **Backward-breaking error response changes**: the flat `error` string is preserved as an alias for `error_detail.message`.

## Timeline

**Day 1** (~4 hours):
- Phase 1: Error envelope (1h)
- Phase 2: Suggestion algorithm + tests (1h)
- Phase 3: Handler rewrite (1h)
- Phase 4: Integration tests + docs + message reply (1h)

**Total: ~4 hours, single-day sprint**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Existing serve-api clients parse the flat `error` string and break on shape change | High | Keep flat `error` populated with `error_detail.message`. No breaking change. |
| Suggestion algorithm surfaces misleading suggestions | Med | Only emit `suggested_fix` when LCP covers ≥ half the request path's segments; otherwise emit only `available_routes`. |
| Adjacent error sites (coercion, runtime) look inconsistent until their sprints land | Low | Document in `serve-api.md` that envelope rollout is incremental; code table will expand. |
| The 3-way discrimination misclassifies an edge case (e.g., server has both `@route`s and module/func users) | Med | Explicit test fixtures for each case; default to `ROUTE_NOT_FOUND` (better UX) when a server has any `@route`s registered. |

## Related Documents

**Implemented (may inform design)**:
- [design_docs/implemented/v0_10_0/m-route-nowrap.md](../../implemented/v0_10_0/m-route-nowrap.md) — `@nowrap` response shape
- [design_docs/implemented/v0_9_4/m-serve-api-json-decode-taggedvalue.md](../../implemented/v0_9_4/m-serve-api-json-decode-taggedvalue.md) — existing serve-api error conventions

**Planned (sibling sprints)**:
- [m-dx-serve-api-error-status.md](m-dx-serve-api-error-status.md) — Result.Err → HTTP status. **Closely related**: shares the "typed error envelope" theme. Coordinate rollout.
- [m-dx-serve-api-coercion.md](m-dx-serve-api-coercion.md) — input coercion errors. Another envelope consumer.
- [m-serve-api-dx.md](m-serve-api-dx.md) — general serve-api DX umbrella.

## References

- Source: agent message `566f6da6` from docparse — read via `ailang messages read 566f6da6-0c4e-439b-b701-05e945ebb5c8`
- [.claude/rules/api-server.md](../../../.claude/rules/api-server.md) — serve-api annotation + filtering conventions
- Current handler: [internal/apiserver/handler.go:31-126](../../../internal/apiserver/handler.go#L31-L126)
- Route registry: [internal/apiserver/routes.go:258-306](../../../internal/apiserver/routes.go#L258-L306)

## Future Work

- Migrate remaining serve-api error sites to `error_detail` envelope (runtime errors, coercion errors, panic recovery).
- Consider exposing the route registry as a debug endpoint (`GET /api/_routes`) so agents can self-discover routes without triggering a 404.
- Promote `error_detail` to top-level (drop flat `error` string) in a future major version once clients have migrated.

---

**Document created**: 2026-04-09
**Last updated**: 2026-04-09

DESIGN_DOC_PATH: design_docs/planned/v0_11_0/m-serveapi-route-not-found.md
