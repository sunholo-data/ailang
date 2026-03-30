# Sprint Plan: M-SERVE-API-DX

## Summary
Make `ailang serve-api` production-ready with custom routes, file upload, binary responses, header access, API key auth, and comprehensive documentation. Enables DocParse deployment on Cloud Run as an Unstructured-compatible API.

**Duration:** 5 days (~40 hours)
**Dependencies:** None — builds on existing `internal/apiserver/`
**Risk Level:** Medium (parser annotation generalization is the main risk)
**Design Doc:** `design_docs/planned/v0_9_4/m-serve-api-dx.md`

## Current Status Analysis

### Completed Recently
- M-HASH-COLLECTIONS Phase 1: ~160 LOC in 1 day
- M-CODEGEN-REGISTRY-ONLY + COMPILE-GATE: ~300 LOC in 1 day
- M-CODEGEN-SUSTAIN M1-M5: ~800 LOC in 3 days
- M-PERF5/6/7: ~500 LOC across v0.9.2

### Velocity
- Recent average: ~500 LOC/day (5044 insertions over ~10 days)
- Sprint estimate: ~1000 LOC total (implementation + tests + docs)
- At 500 LOC/day, 5 days is conservative — buffer for docs and testing

### Remaining from Design Doc
- Phase 1: Custom route annotations (~230 LOC)
- Phase 2: File upload & Bytes type (~260 LOC)
- Phase 3: Binary response & header access (~200 LOC)
- Phase 4: Auth middleware & documentation (~540 LOC, mostly docs)

### Parser Annotation Status
The parser supports `@verify(depth: N)` but rejects all other annotations. To support `@route("POST", "/path")` we need to generalize the annotation parser. The infrastructure is solid — lexer tokenizes `@`, parser has `parseVerifyAttribute()`, AST has `FuncDecl.VerifyDepth`. We need to:
1. Add a generic `Annotations []Annotation` field to `FuncDecl`
2. Generalize `parseVerifyAttribute()` → `parseAttribute()` dispatcher
3. Keep `@verify` working, add `@route` handling

## Proposed Milestones

### M1: ANNOTATION_GENERALIZE — Generalize Parser Annotations
**Goal:** Extend the parser to support `@route("METHOD", "/path")` annotations on exported functions, keeping `@verify` working.
**Estimated:** ~120 LOC implementation + ~80 LOC tests = ~200 LOC
**Duration:** 1 day

**Tasks:**
- Add `Annotation` struct to AST: `{Name string, Args []ast.Expr, Pos token.Pos}`
- Add `Annotations []Annotation` field to `FuncDecl`
- Refactor `parseVerifyAttribute()` into generic `parseAnnotation()` that dispatches by name
- Keep `@verify` handling, add `@route` with validation (2 string args: method + path)
- Migrate `VerifyDepth` to read from `Annotations` for backward compat
- Tests: `@route("POST", "/foo")`, `@route("GET", "/bar")`, invalid args, coexistence with `@verify`

**Acceptance Criteria:**
- [ ] `@route("POST", "/general/v0/general")` parses without error
- [ ] `@verify(depth: 5)` still works (no regression)
- [ ] Multiple annotations on same function work
- [ ] Invalid annotation args produce clear error messages
- [ ] `make test` passes, `make lint` clean

**Risks:**
- Parser changes could regress other features — Mitigation: run full test suite after each change

### M2: CUSTOM_ROUTES — Route Registration & OpenAPI
**Goal:** Custom routes from `@route` annotations are served by the HTTP server and reflected in OpenAPI/A2A specs.
**Estimated:** ~130 LOC implementation + ~80 LOC tests = ~210 LOC
**Duration:** 1 day

**Tasks:**
- Add `GetAnnotations(module, func)` method to `embed.Engine`
- Create `internal/apiserver/routes.go` — `RouteEntry`, `extractRoutes()`, `makeRouteHandler()`
- Modify `buildRoutes()` to register custom routes before the catch-all handler
- Update OpenAPI generator to use custom paths for annotated functions
- Update A2A Agent Card to include custom route info
- Tests: route registration, precedence over auto-routes, OpenAPI output with custom paths

**Acceptance Criteria:**
- [ ] `@route("POST", "/general/v0/general")` serves at that exact path
- [ ] `@route("GET", "/api/v1/formats")` handles GET requests
- [ ] Auto-routes still work for non-annotated functions
- [ ] OpenAPI spec shows custom paths (not `/api/{module}/{function}`)
- [ ] `make test` passes, `make lint` clean

**Risks:**
- Go 1.22 method patterns may not be available — Mitigation: check Go version, fall back to manual method check

### M3: FILE_UPLOAD_BYTES — Bytes Type & Multipart Upload
**Goal:** Add `BytesValue` to the runtime and handle `multipart/form-data` uploads in serve-api.
**Estimated:** ~160 LOC implementation + ~100 LOC tests = ~260 LOC
**Duration:** 1.5 days

**Tasks:**
- Add `BytesValue` to `internal/eval/value.go` with `String()`, `Equals()`, `Type()` methods
- Add `Bytes` case to `canonicalKey()` in `canonical_key.go` (for hash collections compat)
- Add `Bytes` case to `valuesEqual()` in `list.go`
- Implement `parseMultipartArgs()` in `handler.go` — file fields → `BytesValue`, value fields → strings
- Add content-type detection in `handleFunctionCall` — dispatch multipart vs JSON
- Add `--max-upload-size` CLI flag (default 50MB) to `serve_api.go`
- Register builtins: `bytesLength`, `bytesToString`, `stringToBytes`, `bytesFilename`, `bytesMimeType`
- Update `embed.ToGo()` to convert `BytesValue` → `[]byte`
- Tests: multipart upload, size limit, bytes builtins, BytesValue equality

**Acceptance Criteria:**
- [ ] `curl -F "file=@test.pdf" http://localhost:8080/api/mod/func` uploads file
- [ ] File content arrives as `BytesValue` in AILANG function
- [ ] `bytesToString(file, "utf-8")` converts bytes to string
- [ ] `--max-upload-size` rejects oversized uploads with 413
- [ ] `BytesValue` works with `dedup`, `member` (canonical key support)
- [ ] `make test` passes, `make lint` clean

**Risks:**
- `BytesValue` as new `eval.Value` may break type switches — Mitigation: grep all `switch v.(type)` on `eval.Value`, add cases
- Large uploads may stress memory — Mitigation: `--max-upload-size` default 50MB, document Cloud Run sizing

### M4: RESPONSE_HEADERS_AUTH — Binary Response, Headers, Auth
**Goal:** Functions can return binary responses, read request headers, and API key auth protects endpoints.
**Estimated:** ~150 LOC implementation + ~80 LOC tests = ~230 LOC
**Duration:** 1.5 days

**Tasks:**
- Implement `writeRawResponse()` in `handler.go` — detect `_body` field, write binary with custom headers
- Implement header injection for functions with `Headers`-typed parameter
- Register builtins: `getHeader`, `hasHeader`
- Create `internal/apiserver/auth.go` — API key middleware with constant-time compare
- Add `--api-key-header` and `--api-key-env` CLI flags
- Wire auth middleware into route registration (exempt `/api/_*`, `/mcp/`, `/a2a/`)
- Update OpenAPI for binary response schemas (application/octet-stream)
- Tests: binary response, header injection, auth valid/invalid/missing

**Acceptance Criteria:**
- [ ] Function returning `{_body: bytes, _status: 200, _headers: {...}}` sends raw binary
- [ ] `Content-Type` and `Content-Disposition` headers set correctly
- [ ] `getHeader(headers, "authorization")` returns header value
- [ ] `--api-key-header "x-api-key" --api-key-env "API_KEY"` rejects invalid keys with 401
- [ ] Meta endpoints (`/api/_health`, `/api/_meta/*`) bypass auth
- [ ] `make test` passes, `make lint` clean

**Risks:**
- Auth middleware could break MCP/A2A — Mitigation: explicitly exempt protocol paths
- `_body` field convention could collide — Mitigation: document underscore convention, low real-world risk

### M5: DOCS_EXAMPLES — Documentation & Examples
**Goal:** Comprehensive serve-api guide and working examples.
**Estimated:** ~400 LOC docs + ~60 LOC examples = ~460 LOC
**Duration:** 0.5 days

**Tasks:**
- Create `docs/docs/guides/serve-api.md` — comprehensive guide (12 sections from design doc)
- Create `examples/runnable/serve_api_basic.ail` — simple API with auto + custom routes
- Create `examples/runnable/serve_api_upload.ail` — file upload example
- Update CHANGELOG.md with all M-SERVE-API-DX features
- Update design doc status

**Acceptance Criteria:**
- [ ] `docs/docs/guides/serve-api.md` covers all features with examples
- [ ] Example files parse without errors (`ailang check examples/runnable/serve_api_*.ail`)
- [ ] CHANGELOG updated with feature summary
- [ ] Design doc moved to implemented/

**Risks:**
- Minimal — documentation is low-risk

## Success Metrics
- All 5 milestones complete with tests passing
- `make test` and `make lint` clean throughout
- DocParse can define custom routes and handle file uploads
- OpenAPI spec reflects custom routes
- `docs/docs/guides/serve-api.md` exists
- CHANGELOG.md updated
- No regressions in existing serve-api behavior

## Dependencies
- None external — all work is within the AILANG codebase
- M1 blocks M2 (annotations needed for routes)
- M2 blocks M5 (routes needed for examples)
- M3 partially blocks M4 (BytesValue needed for binary responses)

## Open Questions
- None — design freeze items resolved in design doc

## Notes
- Each milestone has a natural pause point — can ship M1+M2 separately if needed
- Parser annotation generalization (M1) benefits future features beyond `@route` (e.g., `@deprecated`, `@test`, `@bench`)
- Total estimate ~1360 LOC is conservative — actual may be less since docs are counted in LOC
