# Sprint Plan: M-DX-ROUTE-CTX — HTTP Request Context for @route Handlers

## Summary

Add `@raw` annotation support so `@route` handlers can receive full HTTP request context (headers, raw body, method, path, query) as an AILANG record. This unblocks production webhook processing for external services like Stripe that send custom JSON with signature headers.

**Duration:** 2 days
**Dependencies:** None
**Risk Level:** Low — purely additive, no breaking changes

## Current Status Analysis

### Velocity
- Recent commits: ~18 in last 7 days (Debug ghost effect, package metadata, net-allow flags)
- Feature scope: ~250 LOC estimated (small, focused change)
- Typical milestone: 1-2 days for similar scope

### Key Files to Modify
- `internal/parser/parser_decl.go` — Add `@raw` annotation parsing
- `internal/apiserver/server.go` — Add `IsRaw` to `ExportInfo`
- `internal/apiserver/routes.go` — Detect `@raw`, build request record, dispatch
- `internal/apiserver/routes_test.go` — Tests for `@raw` extraction and request building
- `internal/apiserver/handler.go` — No changes (non-`@raw` paths unchanged)

## Proposed Milestones

### Milestone 1: M1_PARSER_AND_EXTRACTION
**Goal:** Parse `@raw` annotation and propagate `IsRaw` flag through to route registration
**Estimated:** 60 LOC implementation + 40 LOC tests = 100 LOC
**Duration:** 0.5 day

**Tasks:**
1. Add `@raw` case to `parseAnnotation()` switch in `parser_decl.go` — parameterless annotation returning `&ast.Annotation{Name: "raw", Args: nil}`
2. Update error message in `default` case to list `@raw` as supported
3. Add `IsRaw bool` field to `ExportInfo` struct in `server.go`
4. In `extractRouteAnnotations()` in `routes.go`, detect `@raw` annotation and set `IsRaw: true`
5. Add parser test for `@raw` annotation
6. Add `TestExtractRouteAnnotations` case with `@raw @route` combination

**Acceptance Criteria:**
- [ ] `@raw` parses without error before `@route` or after `@route`
- [ ] Unknown annotations still produce clear error
- [ ] `ExportInfo.IsRaw` correctly set when `@raw` present
- [ ] Existing `@route` and `@verify` parsing unchanged
- [ ] `make test` passes

### Milestone 2: M2_REQUEST_RECORD_DISPATCH
**Goal:** When `IsRaw` is true, build `HttpRequest` record from `*http.Request` and pass as single argument
**Estimated:** 80 LOC implementation + 50 LOC tests = 130 LOC
**Duration:** 0.5 day

**Tasks:**
1. Add `buildHttpRequestRecord(r *http.Request, body []byte)` function in `routes.go` that constructs a `map[string]interface{}` with `body`, `headers`, `method`, `path`, `query` fields
2. In `callFunction()`, read body early, then branch: if route is `@raw`, build request record and use as sole arg; otherwise proceed with existing `parseArgs(body)` flow
3. Plumb `IsRaw` into `callFunction` — either via lookup on `s.modules` or by passing it as parameter
4. Add unit test for `buildHttpRequestRecord` — verify all fields populated correctly
5. Add unit test for headers with multiple values (use first value)

**Acceptance Criteria:**
- [ ] `buildHttpRequestRecord` returns correct shape: `{body, headers, method, path, query}`
- [ ] Headers are `map[string]interface{}` with string values
- [ ] Query params are `map[string]interface{}` with string values
- [ ] Body is raw string (not parsed)
- [ ] Non-`@raw` routes completely unchanged
- [ ] `make test` passes

### Milestone 3: M3_INTEGRATION_AND_EXAMPLE
**Goal:** End-to-end integration test and example file
**Estimated:** 30 LOC example + 60 LOC test = 90 LOC
**Duration:** 0.5 day

**Tasks:**
1. Create `examples/serve_api_webhook.ail` showing `@raw @route` webhook handler pattern
2. Add integration test: POST custom JSON with custom header to `@raw` route, verify handler receives full request record
3. Add test: verify non-`@raw` route still receives parsed args (backward compat)
4. Update CHANGELOG.md with the feature
5. Update `cmd/ailang/serve_api.go` help text to document `@raw`

**Acceptance Criteria:**
- [ ] Example file parses and type-checks cleanly
- [ ] Integration test sends Stripe-like webhook (custom JSON + Stripe-Signature header)
- [ ] Handler receives `request.headers["Stripe-Signature"]` correctly
- [ ] Handler receives `request.body` as raw string
- [ ] `make verify-examples` passes
- [ ] CHANGELOG.md updated
- [ ] `make lint` passes

## Success Metrics
- All tests passing: `make test`
- Linting clean: `make lint`
- Examples verified: `make verify-examples`
- Total LOC: ~320 (implementation + tests + example + docs)
- Documentation: CHANGELOG.md, serve_api.go help text, example file

## Open Questions
- Should `@raw` work without `@route`? (Proposed: No — `@raw` only meaningful on routed functions)
- Should headers be case-normalized? (Proposed: Yes — Go's `http.Header` already canonical-cases keys)
- Body size limit for `@raw` routes? (Proposed: Keep existing 1MB default, configurable via `--max-upload-size`)
