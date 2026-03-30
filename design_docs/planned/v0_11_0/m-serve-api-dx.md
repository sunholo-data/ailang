# M-SERVE-API-DX: serve-api Developer Experience — Custom Routes, File Upload, Binary Response, Auth

**Status**: Planned
**Target**: v0.9.4
**Priority**: P1 (High — blocks DocParse production deployment on Cloud Run)
**Estimated**: 4-5 days
**Dependencies**: None (builds on existing `internal/apiserver/`)
**Milestone ID**: M-SERVE-API-DX
**Created**: 2026-03-18
**Source**: DocParse agent message `c4c9aec0` (serve-api feature requests for production REST API)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to determinism — HTTP layer is already outside the deterministic core |
| A2: Replayability | 0 | HTTP requests are inherently non-replayable; no change |
| A3: Effect Legibility | +1 | Route annotations make HTTP effects explicit at the function declaration site |
| A4: Explicit Authority | +1 | Auth middleware makes access control explicit rather than implicit open-access |
| A5: Bounded Verification | +1 | Route annotations are locally verifiable — each function declares its own route |
| A6: Safe Concurrency | 0 | No concurrency changes — Go's `net/http` handles this |
| A7: Machines First | +1 | OpenAPI spec enriched with custom routes — better for code generators and AI agents |
| A8: Minimal Syntax | 0 | Uses existing annotation syntax (`@route`), no new grammar |
| A9: Cost Visibility | +1 | File upload size limits and binary response types make I/O costs visible |
| A10: Composability | +1 | Middleware composes — auth wraps routes, routes wrap handlers |
| A11: Structured Failure | 0 | Error handling follows existing JSON error response pattern |
| A12: System Boundary | +1 | Custom routes make system boundaries (HTTP ↔ AILANG) explicit and configurable |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): HTTP serving is already an effect; no new nondeterminism
- [x] A3 (Effects): Route annotations make effects MORE legible, not less
- [x] A4 (Authority): Auth middleware adds explicit authority checking
- [x] A7 (Machines First): Custom routes improve machine-readable API specs

---

## Systemic Audit

**Is this a one-off or part of a larger pattern?**

Yes — this is the **"serve-api production readiness" gap**:

1. **v0.8.0**: `ailang serve-api` shipped with auto-routing, OpenAPI, MCP, A2A
2. **v0.9.2**: DocParse (19 modules, 406 functions) attempted production deployment
3. **v0.9.2**: Hit 6 gaps that block any real REST API use case

**Pattern**: `serve-api` was built for introspection and development. Any project trying to deploy it as a production API hits the same walls: no custom routes, no file uploads, no binary responses, no auth. These are not DocParse-specific — they're universal REST API requirements.

**Audit of related work:**
- **M-CODEGEN-API-SERVER** (v0.10.0): Compiled Go API server. That doc's Phase 2 mentions "route configuration" — our Phase 1 here provides the annotation system that codegen can read later.
- **serve-api handler.go**: Single catch-all handler at `/api/` with 1MB JSON-only body. All 6 requested features require changes here.

---

## Problem Statement

### Immediate Problem: DocParse Can't Deploy as Production API

DocParse wants to deploy on Cloud Run as a drop-in replacement for Unstructured's document parsing API. Six gaps block this:

| Gap | Severity | Why It Blocks |
|-----|----------|---------------|
| No custom routes | CRITICAL | Must serve `POST /general/v0/general` for Unstructured compatibility |
| No file upload | CRITICAL | Document parsing API must accept DOCX/PDF/PPTX via multipart/form-data |
| No binary response | HIGH | Format conversion must return DOCX/PDF files, not JSON-wrapped bytes |
| No header access | HIGH | Must read `unstructured-api-key` and `Authorization` headers |
| No auth middleware | MEDIUM | Must validate API keys before allowing access |
| No serve-api docs | HIGH | No guide for deployment patterns, type mapping, or protocol details |

**Current State:**
- Routes auto-generated as `POST /api/{module}/{function}` — not configurable
- Request body: JSON only, 1MB limit (`io.LimitReader(r.Body, 1<<20)`)
- Response: always JSON-wrapped via `embed.ToGo()` + `json.Marshal`
- No access to HTTP headers from AILANG functions
- No authentication — all loaded functions publicly callable
- Documentation limited to `ailang serve-api --help`

**Impact:**
- DocParse (first major AILANG production deployment) blocked
- Any AILANG project wanting a custom REST API hits same walls
- Unstructured API compatibility is a strong adoption driver

---

## Goals

**Primary Goal:** Make `ailang serve-api` production-ready for REST APIs with custom routing, file upload, binary response, header access, and API key auth.

**Success Metrics:**
- DocParse serves `POST /general/v0/general` with file upload and binary response
- Custom routes appear correctly in generated OpenAPI spec
- API key auth rejects unauthorized requests with 401
- `docs/docs/guides/serve-api.md` covers all features with examples
- Zero breaking changes to existing `serve-api` behavior (additive only)

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Route annotation syntax (`@route` vs config file) | Determines how all future AILANG APIs define routes | human | design | high |
| How file bytes enter AILANG (`Bytes` type vs `String` base64) | Affects type system, codegen, and all I/O functions | human | design | high |
| Auth mechanism (CLI flags vs middleware function) | Extensibility vs simplicity trade-off | agent | compile | med |
| Header access model (implicit context vs explicit parameter) | Purity implications — headers are an effect | human | design | high |
| Body size limit for uploads (configurable vs fixed) | Security and resource implications for Cloud Run | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Route annotation syntax: **`@route` annotations on exported functions** (see Solution Design)
- [x] File bytes representation: **`Bytes` value type** backed by `[]byte` in Go (see Phase 2)
- [x] Header access model: **`Headers` parameter type** — function declares it wants headers, server provides them (see Phase 3)
- [ ] Maximum upload size: default 50MB, configurable via `--max-upload-size` flag

---

## Solution Design

### Overview

Four phases, each independently shippable:

1. **Custom Routes** — `@route` annotations parsed from AST, route table built at startup
2. **File Upload & Bytes Type** — multipart/form-data handling, new `Bytes` runtime value
3. **Binary Response & Headers** — `Response` record type, header access via `Headers` parameter
4. **Auth & Documentation** — API key middleware, comprehensive serve-api guide

### Architecture

```
┌─────────────────────────────────────────────────┐
│                  HTTP Request                     │
├─────────────────────────────────────────────────┤
│  Auth Middleware (Phase 4)                        │
│  ├─ Check API key header against env var          │
│  └─ 401 if invalid                                │
├─────────────────────────────────────────────────┤
│  Route Dispatcher (Phase 1)                       │
│  ├─ Custom routes: @route annotations             │
│  ├─ Auto routes: POST /api/{module}/{function}    │
│  └─ Custom routes take precedence                 │
├─────────────────────────────────────────────────┤
│  Content Negotiation (Phase 2-3)                  │
│  ├─ multipart/form-data → Bytes parameter         │
│  ├─ application/json → existing JSON parsing      │
│  └─ Response type → binary/custom content-type    │
├─────────────────────────────────────────────────┤
│  Function Handler (existing)                      │
│  ├─ engine.CallPreserveFloats()                   │
│  └─ Result conversion                             │
└─────────────────────────────────────────────────┘
```

### Phase 1: Custom Route Annotations (~8 hours)

**Annotation syntax in AILANG:**

```ailang
@route("POST", "/general/v0/general")
export func partitionLegacy(content: string) -> {elements: [{type: string, text: string}]}
  parseDocument(content)

@route("POST", "/api/v1/parse")
export func parse(content: string, format: string) -> ParseResult
  parseWithFormat(content, format)

@route("GET", "/api/v1/formats")
export func listFormats() -> [string]
  supportedFormats()
```

**How annotations work:**

Annotations are already parsed by the AILANG parser as `@name(args)` before function declarations. They're stored in `ast.FuncDecl.Annotations`. We extract route annotations during module loading.

**Implementation:**

```go
// internal/apiserver/routes.go — NEW

type RouteEntry struct {
    Method   string // "GET", "POST", etc.
    Path     string // "/general/v0/general"
    Module   string // "docparse/api"
    Function string // "partitionLegacy"
}

// extractRoutes reads @route annotations from loaded modules
func extractRoutes(engine *embed.Engine, modules map[string]*ModuleInfo) []RouteEntry {
    var routes []RouteEntry
    for modPath, mod := range modules {
        for _, exp := range mod.Exports {
            if ann := engine.GetAnnotation(modPath, exp.Name, "route"); ann != nil {
                routes = append(routes, RouteEntry{
                    Method:   ann.Args[0].(string),
                    Path:     ann.Args[1].(string),
                    Module:   modPath,
                    Function: exp.Name,
                })
            }
        }
    }
    return routes
}
```

**Route registration in `buildRoutes()`:**

```go
// Register custom routes BEFORE the catch-all
for _, route := range customRoutes {
    pattern := route.Method + " " + route.Path  // Go 1.22 method patterns
    mux.HandleFunc(pattern, s.corsWrap(s.makeRouteHandler(route)))
}

// Existing catch-all still works for non-annotated functions
mux.HandleFunc("/api/", s.corsWrap(s.handleFunctionCall))
```

**OpenAPI integration:** Custom routes appear in the generated spec with their actual paths instead of the auto-generated `/api/{module}/{function}` pattern.

### Phase 2: File Upload & Bytes Type (~10 hours)

**New runtime value: `BytesValue`**

```go
// internal/eval/value.go — ADD

type BytesValue struct {
    Data     []byte
    Filename string // original filename from upload, empty if not from upload
    MimeType string // detected or declared MIME type
}

func (v *BytesValue) String() string {
    return fmt.Sprintf("<bytes:%d:%s>", len(v.Data), v.MimeType)
}
```

**AILANG type:** `Bytes` — opaque type, not a list of ints. Builtins operate on it:

```ailang
-- New builtins for Bytes
export pure func bytesLength(b: Bytes) -> int
export pure func bytesToString(b: Bytes, encoding: string) -> string
export pure func stringToBytes(s: string, encoding: string) -> Bytes
export pure func bytesFilename(b: Bytes) -> string
export pure func bytesMimeType(b: Bytes) -> string
```

**Multipart handler in `handler.go`:**

```go
func (s *Server) handleFunctionCall(w http.ResponseWriter, r *http.Request) {
    // ... existing path parsing ...

    var args []interface{}
    contentType := r.Header.Get("Content-Type")

    if strings.HasPrefix(contentType, "multipart/form-data") {
        // Parse multipart — configurable max size
        if err := r.ParseMultipartForm(s.maxUploadSize); err != nil {
            writeError(w, http.StatusRequestEntityTooLarge, "upload too large")
            return
        }
        args, err = s.parseMultipartArgs(r)
    } else {
        // Existing JSON parsing
        args, err = s.parseJSONArgs(r)
    }
    // ... rest of handler ...
}

func (s *Server) parseMultipartArgs(r *http.Request) ([]interface{}, error) {
    var args []interface{}

    // File fields become BytesValue
    for _, fileHeaders := range r.MultipartForm.File {
        for _, fh := range fileHeaders {
            f, err := fh.Open()
            if err != nil { return nil, err }
            defer f.Close()
            data, err := io.ReadAll(io.LimitReader(f, s.maxUploadSize))
            if err != nil { return nil, err }
            args = append(args, &eval.BytesValue{
                Data:     data,
                Filename: fh.Filename,
                MimeType: fh.Header.Get("Content-Type"),
            })
        }
    }

    // Non-file fields become string args
    for _, values := range r.MultipartForm.Value {
        for _, v := range values {
            args = append(args, v)
        }
    }

    return args, nil
}
```

**CLI flag:**

```
--max-upload-size SIZE   Maximum upload size in bytes (default: 50MB)
```

### Phase 3: Binary Response & Header Access (~8 hours)

**Response type for binary/custom responses:**

When an AILANG function returns a record with special fields `_body`, `_status`, `_headers`, the server treats it as a raw HTTP response instead of JSON-wrapping:

```ailang
@route("POST", "/api/v1/convert")
export func convertToDocx(content: Bytes, targetFormat: string) -> {_body: Bytes, _status: int, _headers: {string: string}}
  let result = convert(content, targetFormat)
  {
    _body = result,
    _status = 200,
    _headers = {
      "Content-Type" = "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
      "Content-Disposition" = "attachment; filename=\"output.docx\""
    }
  }
```

**Detection in handler:**

```go
func (s *Server) writeResult(w http.ResponseWriter, result eval.Value) {
    // Check if result is a Response record (has _body field)
    if rec, ok := result.(*eval.RecordValue); ok {
        if body, hasBody := rec.Fields["_body"]; hasBody {
            s.writeRawResponse(w, rec, body)
            return
        }
    }
    // Default: JSON-wrapped response (existing behavior)
    s.writeJSON(w, http.StatusOK, result)
}

func (s *Server) writeRawResponse(w http.ResponseWriter, rec *eval.RecordValue, body eval.Value) {
    // Set custom headers
    if headers, ok := rec.Fields["_headers"].(*eval.RecordValue); ok {
        for k, v := range headers.Fields {
            if sv, ok := v.(*eval.StringValue); ok {
                w.Header().Set(k, sv.Value)
            }
        }
    }
    // Set status code
    status := 200
    if s, ok := rec.Fields["_status"].(*eval.IntValue); ok {
        status = int(s.Value)
    }
    w.WriteHeader(status)
    // Write body
    switch b := body.(type) {
    case *eval.BytesValue:
        w.Write(b.Data)
    case *eval.StringValue:
        w.Write([]byte(b.Value))
    }
}
```

**Header access via `Headers` parameter type:**

Functions that want HTTP headers declare a `Headers` parameter:

```ailang
@route("POST", "/general/v0/general")
export func partitionLegacy(file: Bytes, headers: Headers) -> {elements: [Element]}
  let apiKey = getHeader(headers, "unstructured-api-key")
  -- ...
```

The server detects `Headers` in the function's type signature and injects a `RecordValue` with lowercase header keys:

```go
// If function signature includes a Headers-typed parameter, inject request headers
if paramIsHeaders(funcInfo, paramIdx) {
    headerFields := make(map[string]eval.Value)
    for k, vs := range r.Header {
        headerFields[strings.ToLower(k)] = &eval.StringValue{Value: vs[0]}
    }
    args = append(args, &eval.RecordValue{Fields: headerFields})
}
```

**New builtins:**
```ailang
export pure func getHeader(h: Headers, name: string) -> Option[string]
export pure func hasHeader(h: Headers, name: string) -> bool
```

`Headers` is actually just a `RecordValue` with string values — no new type needed in the type system. The "magic" is that the server injects it.

### Phase 4: Auth Middleware & Documentation (~6 hours)

**API key auth via CLI flags:**

```bash
ailang serve-api app.ail \
  --api-key-header "unstructured-api-key" \
  --api-key-env "DOCPARSE_API_KEY"
```

**Implementation:**

```go
// internal/apiserver/auth.go — NEW

func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
    if s.apiKeyHeader == "" || s.apiKeyEnv == "" {
        return next // no auth configured
    }
    expected := os.Getenv(s.apiKeyEnv)
    if expected == "" {
        log.Fatalf("serve-api: --api-key-env %q is not set", s.apiKeyEnv)
    }
    return func(w http.ResponseWriter, r *http.Request) {
        // Skip auth for meta/health endpoints
        if strings.HasPrefix(r.URL.Path, "/api/_") {
            next(w, r)
            return
        }
        key := r.Header.Get(s.apiKeyHeader)
        if key == "" {
            key = r.Header.Get("Authorization")
            if strings.HasPrefix(key, "Bearer ") {
                key = strings.TrimPrefix(key, "Bearer ")
            }
        }
        if subtle.ConstantTimeCompare([]byte(key), []byte(expected)) != 1 {
            writeError(w, http.StatusUnauthorized, "invalid or missing API key")
            return
        }
        next(w, r)
    }
}
```

**Documentation:** Create `docs/docs/guides/serve-api.md` covering:

1. Quick start — `ailang serve-api myapp.ail`
2. How exports map to endpoints (auto-routing)
3. Custom routes with `@route` annotations
4. Request/response formats (JSON, multipart, binary)
5. Type mapping (AILANG types → JSON Schema → OpenAPI)
6. File upload handling with `Bytes` type
7. Binary responses with `_body`/`_status`/`_headers`
8. Header access with `Headers` parameter
9. Authentication with `--api-key-*` flags
10. Protocol support (MCP, A2A)
11. Deployment patterns (Docker, Cloud Run)
12. Examples: DocParse-style API, simple CRUD, webhook handler

---

## Implementation Plan

### Phase 1: Custom Route Annotations (~8 hours)

- [ ] Add `GetAnnotation(module, func, name)` to `embed.Engine` to expose parsed annotations
- [ ] Create `internal/apiserver/routes.go` — `extractRoutes()` and `RouteEntry` type
- [ ] Modify `buildRoutes()` in `server.go` to register custom routes before catch-all
- [ ] Create `makeRouteHandler()` that binds a route to its module/function
- [ ] Update OpenAPI generator to use custom paths for annotated functions
- [ ] Update A2A Agent Card to include custom route paths
- [ ] Tests: custom route registration, precedence over auto-routes, OpenAPI output
- [ ] Test: multiple annotations on different functions, same module

### Phase 2: File Upload & Bytes Type (~10 hours)

- [ ] Add `BytesValue` to `internal/eval/value.go` with `String()`, `Equals()`, `Type()`
- [ ] Add `Bytes` to type system (`internal/types/`) as opaque primitive
- [ ] Implement `parseMultipartArgs()` in handler
- [ ] Add content-type detection in `handleFunctionCall` — dispatch to multipart vs JSON
- [ ] Add `--max-upload-size` CLI flag (default 50MB)
- [ ] Register builtins: `bytesLength`, `bytesToString`, `stringToBytes`, `bytesFilename`, `bytesMimeType`
- [ ] Update `embed.ToGo()` to handle `BytesValue` → `[]byte` conversion
- [ ] Tests: multipart upload, size limit enforcement, bytes builtins

### Phase 3: Binary Response & Header Access (~8 hours)

- [ ] Implement `writeRawResponse()` — detect `_body` field, write binary/custom response
- [ ] Implement header injection for `Headers` parameter type
- [ ] Register builtins: `getHeader`, `hasHeader`
- [ ] Update OpenAPI generator for binary response types (application/octet-stream)
- [ ] Tests: binary response with custom headers, header injection, Content-Disposition

### Phase 4: Auth & Documentation (~6 hours)

- [ ] Create `internal/apiserver/auth.go` — API key middleware
- [ ] Add `--api-key-header` and `--api-key-env` CLI flags
- [ ] Wire auth middleware into route registration (wraps all non-meta routes)
- [ ] Create `docs/docs/guides/serve-api.md` — comprehensive guide
- [ ] Create `examples/runnable/serve_api_basic.ail` — simple API example
- [ ] Create `examples/runnable/serve_api_upload.ail` — file upload example

### Files to Modify/Create

**New files:**
- `internal/apiserver/routes.go` (~100 LOC) — Custom route extraction and registration
- `internal/apiserver/routes_test.go` (~120 LOC) — Route tests
- `internal/apiserver/auth.go` (~60 LOC) — API key middleware
- `internal/apiserver/auth_test.go` (~80 LOC) — Auth tests
- `internal/builtins/bytes.go` (~80 LOC) — Bytes builtins
- `internal/builtins/bytes_test.go` (~100 LOC) — Bytes tests
- `docs/docs/guides/serve-api.md` (~400 LOC) — Comprehensive guide
- `examples/runnable/serve_api_basic.ail` (~30 LOC)
- `examples/runnable/serve_api_upload.ail` (~30 LOC)

**Modified files:**
- `internal/eval/value.go` (~+40 LOC) — Add `BytesValue`
- `internal/apiserver/server.go` (~+50 LOC) — Custom route registration, auth wiring, max upload config
- `internal/apiserver/handler.go` (~+80 LOC) — Multipart parsing, binary response, header injection
- `internal/apiserver/openapi.go` (~+30 LOC) — Custom route paths, binary response schemas
- `internal/apiserver/a2a.go` (~+10 LOC) — Custom route paths in Agent Card
- `cmd/ailang/serve_api.go` (~+20 LOC) — New CLI flags
- `internal/embed/embed.go` (~+10 LOC) — `GetAnnotation()` method, `ToGo()` for BytesValue

---

## Examples

### Example 1: DocParse Unstructured-Compatible API

```ailang
module docparse/api

import docparse/parser (parseDocument)
import docparse/formats (supportedFormats, convert)

@route("POST", "/general/v0/general")
export func partitionGeneral(file: Bytes, headers: Headers) -> {elements: [{type: string, text: string}]}
  let _ = getHeader(headers, "unstructured-api-key")  -- validated by middleware
  let text = bytesToString(file, "utf-8")
  parseDocument(text)

@route("POST", "/api/v1/parse")
export func parse(file: Bytes) -> ParseResult
  let text = bytesToString(file, "utf-8")
  let format = bytesMimeType(file)
  parseWithFormat(text, format)

@route("POST", "/api/v1/convert")
export func convertFormat(file: Bytes, target: string) -> {_body: Bytes, _status: int, _headers: {string: string}}
  let result = convert(file, target)
  {
    _body = result,
    _status = 200,
    _headers = {
      "Content-Type" = "application/octet-stream",
      "Content-Disposition" = concat(["attachment; filename=\"output.", target, "\""])
    }
  }

@route("GET", "/api/v1/formats")
export func listFormats() -> [string]
  supportedFormats()
```

**Deployment:**

```bash
ailang serve-api docparse/ \
  --port 8080 \
  --api-key-header "unstructured-api-key" \
  --api-key-env "DOCPARSE_API_KEY" \
  --max-upload-size 104857600  # 100MB
```

### Example 2: Simple API with Auto + Custom Routes

```ailang
module myapp

-- Auto-route: POST /api/myapp/greet
export pure func greet(name: string) -> string
  concat(["Hello, ", name, "!"])

-- Custom route: GET /health
@route("GET", "/health")
export pure func healthCheck() -> {status: string, version: string}
  {status = "ok", version = "1.0.0"}
```

Both routes work simultaneously — custom routes are registered first, auto-routes fill in the rest.

---

## Success Criteria

- [ ] `@route("POST", "/general/v0/general")` serves at that exact path
- [ ] `curl -F "file=@test.docx" http://localhost:8080/api/v1/parse` uploads and parses file
- [ ] Binary response returns raw bytes with correct Content-Type and Content-Disposition
- [ ] `getHeader(headers, "authorization")` returns the header value in an AILANG function
- [ ] `--api-key-header` + `--api-key-env` rejects requests without valid key (401)
- [ ] OpenAPI spec reflects custom routes with correct paths and methods
- [ ] All existing `serve-api` behavior unchanged (auto-routes, MCP, A2A, Swagger UI)
- [ ] `docs/docs/guides/serve-api.md` exists with deployment examples
- [ ] All tests passing, lint clean

---

## Testing Strategy

**Unit tests:**
- Route extraction from annotations (mock AST)
- Multipart parsing with various file types and sizes
- Binary response detection (`_body` field) and writing
- Header injection into function arguments
- Auth middleware: valid key, invalid key, missing key, missing env var
- `hashKey` for `BytesValue` (needed for set operations on bytes)

**Integration tests:**
- Full HTTP request/response cycle with custom routes
- File upload → parse → JSON response
- File upload → convert → binary download
- Auth middleware + custom routes + file upload combined
- OpenAPI spec generation with custom routes
- A2A Agent Card with custom routes

**Manual testing:**
- DocParse end-to-end on Cloud Run
- `curl` examples from documentation
- Swagger UI with file upload form

---

## Deferred Decisions

The following are intentionally left open for the implementer:

- **Multipart field ordering** — how to map multipart fields to positional function args when there are multiple files + values. Agent may choose simplest approach (files first, then values in form order).
- **Header key casing** — whether to normalize to lowercase or preserve original. Agent may choose (recommendation: lowercase for consistency).
- **`_body`/`_status`/`_headers` field naming** — underscore prefix is suggested to avoid collision with user fields. Agent may adjust if a better convention exists.
- **Bytes display format** — how `BytesValue.String()` renders. Agent may choose.

---

## Non-Goals

**Not attempted in this feature:**
- **Streaming responses** — Server-Sent Events or chunked transfer. Separate concern, separate design doc.
- **WebSocket support** — Different protocol, different design.
- **Rate limiting** — Use Cloud Run's built-in or a reverse proxy for now.
- **JWT/OAuth** — API key auth is sufficient for v0.9.4. Full auth framework is future work.
- **Request validation** — Beyond JSON Schema. OpenAPI spec enables external validation.
- **Custom middleware in AILANG** — Only built-in auth middleware. User-defined middleware is Phase 2+.
- **Bytes in the type checker** — `Bytes` is initially a runtime-only type. Full type system integration (e.g., `Bytes` in pattern matching, Hashable instance) is future work.

---

## Timeline

**Day 1-2** (~16 hours):
- Phase 1: Custom route annotations — extraction, registration, OpenAPI, tests

**Day 3** (~10 hours):
- Phase 2: File upload & Bytes type — BytesValue, multipart handler, builtins

**Day 4** (~8 hours):
- Phase 3: Binary response & header access — writeRawResponse, header injection

**Day 5** (~6 hours):
- Phase 4: Auth middleware & documentation — auth.go, serve-api guide, examples

**Total: ~40 hours across 5 days**

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `@route` annotations not yet used in AILANG — parser support may be incomplete | High | Verify annotation parsing exists; if not, add minimal parser support (~2h) |
| `BytesValue` as new eval.Value may break type switches in evaluator | Med | Audit all `switch v.(type)` on `eval.Value` — add `BytesValue` cases |
| Multipart field-to-arg mapping ambiguity (multiple files, mixed fields) | Med | Document convention: files mapped by name matching parameter name, or positionally |
| Auth middleware may interfere with MCP/A2A protocol handlers | Med | Exempt `/mcp/` and `/a2a/` from API key auth (they have their own auth) |
| Binary response detection via `_body` field could collide with user data | Low | Underscore prefix convention; document clearly |
| Large file uploads may OOM on small Cloud Run instances | Med | `--max-upload-size` flag with sensible default (50MB); stream-to-disk for Phase 2+ |

---

## Related Documents

**Directly relevant (API serving):**
- [M-CODEGEN-API-SERVER](../../planned/v0_10_0/m-codegen-api-server.md) — Compiled Go API server. Our custom routes design should be compatible with future codegen.

**Implemented (inform design):**
- [design_docs/implemented/v0_6_0/semantic-caching-complete.md](../../implemented/v0_6_0/semantic-caching-complete.md) — Prior serve-api infrastructure work
- [design_docs/implemented/v0_7_0/m-otel-enhanced-tracing-dx.md](../../implemented/v0_7_0/m-otel-enhanced-tracing-dx.md) — Tracing DX patterns to follow

**Planned (check for overlap):**
- [design_docs/planned/v0_10_0/m-arch5-error-handling-strategy.md](../../planned/v0_10_0/m-arch5-error-handling-strategy.md) — Error handling may affect HTTP error responses

---

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- DocParse agent message `c4c9aec0` — Original feature request with full context
- [Unstructured API docs](https://docs.unstructured.io/) — The API DocParse wants to be compatible with
- [Go 1.22 ServeMux patterns](https://pkg.go.dev/net/http#hdr-Patterns) — Method-based routing used in Phase 1

---

## Future Work

- **Streaming responses** — SSE for long-running operations (e.g., large document parsing progress)
- **User-defined middleware** — Allow AILANG functions as middleware (auth, logging, rate limiting)
- **Request/Response types in type system** — First-class `Request` and `Response` types
- **Bytes in pattern matching** — `match bytes { ... }` for binary protocol handling
- **GraphQL endpoint** — Auto-generated from module types
- **Compiled API server** — M-CODEGEN-API-SERVER builds on these features for native Go binaries

---

**Document created**: 2026-03-18
**Last updated**: 2026-03-18
