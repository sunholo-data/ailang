# Sprint Plan: M-SERVE-API-AGENT-ENHANCEMENTS

## Summary

Add named JSON parameter binding and @nowrap custom headers to serve-api, making it agent-friendly. Agents send natural JSON objects (`{"path": "file.docx"}`) instead of positional arrays.

**Duration:** 3 days (~20 hours)
**Dependencies:** None — builds on existing serve-api infrastructure
**Risk Level:** Low — additive changes, no breaking changes
**Design Doc:** [m-serve-api-agent-enhancements.md](m-serve-api-agent-enhancements.md)

## Current Status Analysis

### Completed Recently (serve-api work in last 14 days)
- @nowrap annotation: +94 LOC routes.go, +273 LOC tests (commit `50275eec`)
- @raw JObject headers: +LOC across routes.go (commits `f385e50a`, `d4214960`)
- Route collision guard: commit `421c70d2`
- @raw handler inline record: commit `71f60b5e`

### Velocity
- 28 commits / 14 days = 2 commits/day
- ~6,100 insertions / 14 days = ~435 LOC/day
- Serve-api specific: ~400 LOC impl + tests in ~3 days (recent @nowrap + @raw work)
- **Conservative estimate for this sprint: 150-200 LOC/day** (focused feature work is slower than mixed commits)

### Remaining from Design Doc
- Phase 1: Named parameter binding (~100 LOC impl + ~120 LOC tests)
- Phase 2: @nowrap custom headers (~30 LOC impl + ~40 LOC tests)
- Phase 3: Docs & examples (~70 LOC)

**Total: ~360 LOC** (well within 3-day capacity)

### Key Implementation Facts
- `extractRouteAnnotations()` already iterates ALL exported fns in AST — param names can be extracted here
- `ast.FuncDecl.Params` has `[]*Param` with `.Name` field — names are available
- `parseArgs()` in handler.go is the single entry point for arg parsing — one place to change
- `ExportInfo` struct needs `ParamNames []string` field added
- `callFunction()` already receives module/func info — can pass param names through
- `server.go` is 598 LOC (under 800 limit), `routes.go` is 428 LOC — room to grow
- `@nowrap` path at routes.go:272-281 is the exact insertion point for header extraction

## Proposed Milestones

### M1: PARAM_NAME_EXTRACTION — Extract and Store Parameter Names
**Goal:** Add `ParamNames` to `ExportInfo` and populate from AST during module loading
**Estimated:** ~60 LOC implementation + ~40 LOC tests = ~100 LOC
**Duration:** 0.5 days

**Tasks:**
1. Add `ParamNames []string` field to `ExportInfo` struct in server.go
2. In `extractRouteAnnotations()`, extract param names from `fn.Params` for ALL exported fns (not just @route)
3. Verify param names appear in `/api/_meta/modules` JSON output
4. Unit test: param names extracted correctly for various function signatures

**Acceptance Criteria:**
- [ ] `ExportInfo.ParamNames` populated for all exported functions
- [ ] `/api/_meta/modules` endpoint includes `param_names` in JSON
- [ ] Functions with 0 params get empty `ParamNames` slice
- [ ] All existing tests pass
- [ ] Linting clean

**Risks:**
- None — purely additive field on existing struct

### M2: NAMED_BINDING — Named JSON Parameter Binding
**Goal:** Parse `{"key": "val"}` JSON bodies and bind to function params by name
**Estimated:** ~80 LOC implementation + ~100 LOC tests = ~180 LOC
**Duration:** 1 day

**Tasks:**
1. Implement `camelToSnake()` utility in handler.go or routes.go
2. Implement `parseNamedArgs(body map[string]interface{}, paramNames []string) []interface{}`
3. Update `callFunction()` to try named binding when body is a JSON object (after `{"args": [...]}` check)
4. Wire param names through: `callFunction` receives ExportInfo → passes ParamNames to parseNamedArgs
5. Also support named binding in auto-route handler (same parseNamedArgs function)
6. Unit tests: exact match, snake_case→camelCase, no match fallback, partial match, empty body
7. Integration test: full HTTP request with named JSON body → correct invocation

**Acceptance Criteria:**
- [ ] `{"path": "file.docx", "output_format": "blocks"}` binds to `parseFile(path: string, outputFormat: string)`
- [ ] `{"args": ["file.docx", "blocks"]}` still works (backward compat)
- [ ] Unmatched JSON keys silently ignored
- [ ] Named binding works for both @route and auto-route endpoints
- [ ] `camelToSnake("outputFormat")` → `"output_format"` and vice versa
- [ ] All existing tests pass
- [ ] Linting clean

**Risks:**
- snake_case conversion edge cases (acronyms like `httpURL`) — mitigate with comprehensive test cases

### M3: NOWRAP_HEADERS — Custom Headers on @nowrap Responses
**Goal:** Extract `_headers` field from @nowrap results and set as HTTP response headers
**Estimated:** ~30 LOC implementation + ~40 LOC tests = ~70 LOC
**Duration:** 0.5 days

**Tasks:**
1. In the `@nowrap` path (routes.go:272-281), detect `_headers` field in result record BEFORE ToGo conversion
2. Extract header key/value pairs and call `w.Header().Set()`
3. Remove `_headers` from record fields before JSON encoding
4. Unit test: @nowrap with _headers, @nowrap without _headers (unchanged), non-record result
5. Integration test: verify headers appear in HTTP response

**Acceptance Criteria:**
- [ ] `@nowrap` function returning `{data: "x", _headers: {"X-Request-Id": "abc"}}` sets `X-Request-Id` header
- [ ] `_headers` field excluded from JSON response body
- [ ] @nowrap without `_headers` works unchanged
- [ ] Non-record @nowrap results work unchanged
- [ ] All existing tests pass
- [ ] Linting clean

**Risks:**
- Must extract `_headers` from `eval.RecordValue` BEFORE `embed.ToGo()` conversion — order matters

### M4: DOCS_AND_OPENAPI — Documentation, OpenAPI, and Examples
**Goal:** Update docs, OpenAPI spec, and add example files
**Estimated:** ~80 LOC docs + ~30 LOC code = ~110 LOC
**Duration:** 0.5 days

**Tasks:**
1. Update OpenAPI generator to include parameter names in request body schema
2. Add "Named Parameter Binding" section to `docs/docs/guides/serve-api.md`
3. Add "@nowrap Custom Headers" section to serve-api guide
4. Create `examples/runnable/serve_api_named_params.ail` example
5. Update CHANGELOG.md with both features
6. Reply to docparse inbox messages with implementation status

**Acceptance Criteria:**
- [ ] OpenAPI spec shows parameter names in request body schema
- [ ] serve-api guide has named binding section with curl examples
- [ ] Example file runs successfully with `ailang serve-api`
- [ ] CHANGELOG updated
- [ ] Docparse messages acknowledged with status

**Risks:**
- OpenAPI parameter name integration may need schema changes — mitigate by keeping it simple (just add names to description)

## Day-by-Day Breakdown

### Day 1 (8 hours)
- **Morning:** M1 (param name extraction) — 3 hours
- **Afternoon:** M2 (named binding implementation + tests) — 5 hours

### Day 2 (8 hours)
- **Morning:** M2 continued (integration tests, edge cases) — 3 hours
- **Afternoon:** M3 (nowrap headers) — 3 hours
- **Late afternoon:** M4 start (OpenAPI, docs) — 2 hours

### Day 3 (4 hours)
- **Morning:** M4 continued (examples, changelog, inbox replies) — 3 hours
- **Final:** Full test pass, `make ci`, verify examples — 1 hour

## Success Metrics
- All tests passing: `make test` clean
- Linting clean: `make lint` clean
- File sizes: `make check-file-sizes` passes (server.go < 800, routes.go < 800)
- Named binding works for both @route and auto-route endpoints
- Backward compatible: `{"args": [...]}` unchanged
- Documentation: serve-api guide updated
- Examples: `examples/runnable/serve_api_named_params.ail` exists and verifies

## Dependencies
- None — all prerequisite serve-api features (@route, @nowrap, @raw) are already implemented

## Open Questions
- **@nowrap `_status` support**: Should we also extract `_status` from @nowrap results? (Currently only `_body` pattern supports custom status codes.) Recommendation: add if trivial (~5 LOC), defer otherwise.

## Notes
- `server.go` at 598 LOC is approaching the 800 LOC limit. If M1 pushes it past 650, consider extracting `extractModuleInfo()` into a separate file.
- The `camelToSnake` function should be well-tested — it's the most likely source of subtle bugs.
- Named binding for auto-routed functions requires passing param names through the catch-all handler path, not just the custom route path.
