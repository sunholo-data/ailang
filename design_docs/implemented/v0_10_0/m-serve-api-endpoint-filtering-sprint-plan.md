# Sprint Plan: M-SERVE-API-ENDPOINT-FILTERING

## Summary
Add endpoint filtering (`--routes-only`, `@noexpose`) and fix JSON double-encoding in `@nowrap` responses for serve-api, unblocking DocParse production deployment.

**Duration:** 1 day (2 milestones)
**Dependencies:** M-ROUTE-NOWRAP (already implemented)
**Risk Level:** Low (additive features, no breaking changes)

## Current Status Analysis

### Completed Recently
- M-STDLIB-CRYPTO-JWT: RSA + JWT support (~600 LOC)
- M-BRAIN-CONTEXT: contextual brain injection (~400 LOC)
- Swagger UI fix, elaborator alias fix, validator pattern bindings

### Velocity
- Recent average: ~420 LOC/day (5,900 LOC across ~14 days)
- This sprint estimated: ~170 LOC total (well within 1-day capacity)

### Remaining from Design Doc
- Phase 1: Endpoint filtering (~100 LOC)
- Phase 2: JSON auto-unwrap (~70 LOC)

## Proposed Milestones

### Milestone 1: M1_ENDPOINT_FILTERING — `--routes-only` flag + `@noexpose` annotation
**Goal:** Control which exported functions become HTTP endpoints
**Estimated:** ~70 LOC implementation + ~100 LOC tests = ~170 LOC
**Duration:** ~4 hours

**Tasks:**
1. Add `--routes-only` flag to `cmd/ailang/serve_api.go` and `RoutesOnly` to `apiserver.Config`
2. Add `IsNoExpose` field to `ExportInfo`, parse `@noexpose` in `extractRouteAnnotations()`
3. Filter auto-route registration in `server.go` (skip `@noexpose` + skip non-@route when `--routes-only`)
4. Filter OpenAPI spec generation in `openapi.go` to match
5. Filter A2A Agent Card skills in `a2a.go` to match
6. Add startup log line with filtered endpoint count
7. Write tests: filtering combinations, OpenAPI output, @noexpose + @route interaction

**Acceptance Criteria:**
- `--routes-only` limits auto-generated endpoints to @route functions only
- `@noexpose` prevents individual exported functions from becoming HTTP endpoints
- `@noexpose` functions still importable by other AILANG modules (no compile-time effect)
- OpenAPI spec excludes filtered functions
- A2A Agent Card excludes filtered functions
- Startup banner shows endpoint count (exposed vs filtered)
- All existing tests pass
- `make lint` clean

**Risks:**
- None significant — additive feature, off by default

### Milestone 2: M2_JSON_AUTO_UNWRAP — Fix @nowrap double-encoding of JSON strings
**Goal:** `@nowrap` endpoints returning `encode(jo(...))` produce clean JSON, not double-encoded strings
**Estimated:** ~30 LOC implementation + ~70 LOC tests = ~100 LOC
**Duration:** ~2 hours

**Tasks:**
1. Add `isValidJSON()` helper to `routes.go` (detect JSON objects/arrays only, not primitives)
2. Add auto-unwrap check in `@nowrap` response path: if `goResult` is a string containing valid JSON object/array, write as raw bytes
3. Write tests: JSON object string auto-unwrapped, JSON array string auto-unwrapped, plain string still JSON-encoded, invalid JSON string still JSON-encoded, non-string result unchanged
4. Update CHANGELOG

**Acceptance Criteria:**
- `@nowrap` returns `{"status":"healthy"}` not `"{\"status\":\"healthy\"}"` for JSON string results
- `@nowrap` with non-JSON strings still returns JSON-encoded string
- `@nowrap` with record/list results unchanged (no regression)
- `@nowrap` with bare numbers/booleans NOT unwrapped (only objects and arrays)
- All existing tests pass
- `make lint` clean

**Risks:**
- Behavior change for existing @nowrap users returning JSON strings — but current behavior is a bug (double-encoding), so this is a fix

## Success Metrics
- All existing `make test` passing
- All existing `make lint` passing
- New tests for filtering and auto-unwrap
- CHANGELOG updated
- `make verify-examples` passing

## Dependencies
- None — M-ROUTE-NOWRAP already implemented, @nowrap annotation exists

## Open Questions
- None — all design decisions frozen in the design doc

## Notes
- Both features are opt-in with zero breaking changes to existing behavior
- The JSON auto-unwrap only applies to @nowrap responses (not the default envelope)
- `@noexpose` has no effect on the type checker or module system — purely an apiserver concern
