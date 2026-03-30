# M-SERVE-API-AGENT-ENHANCEMENTS: Named Parameter Binding & Response Headers for Agent-Friendly APIs

**Status**: Implemented
**Target**: v0.9.2
**Priority**: P1 (High — needed for v0.9.0 agent-friendly API)
**Estimated**: 2-3 days
**Dependencies**: None (builds on existing serve-api features)
**Milestone ID**: M-SERVE-API-AGENT-ENHANCEMENTS
**Created**: 2026-03-26
**Source**: DocParse agent messages `32d46e7d`, `d6f8c5ff`, `ce694515`

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | HTTP layer is already outside the deterministic core |
| A2: Replayability | 0 | HTTP requests are inherently non-replayable; no change |
| A3: Effect Legibility | +1 | Named params make API contracts self-documenting at the HTTP boundary |
| A4: Explicit Authority | 0 | No change |
| A5: Bounded Verification | +1 | Named binding enables parameter name validation at startup |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Agents send natural JSON objects instead of opaque positional arrays |
| A8: Minimal Syntax | 0 | No new language syntax — changes are in the serve-api runtime only |
| A9: Cost Visibility | 0 | No change |
| A10: Composability | +1 | Named params compose with @route, @raw, @nowrap independently |
| A11: Structured Failure | +1 | Unknown field names produce 400 errors with field-level diagnostics |
| A12: System Boundary | +1 | HTTP↔AILANG boundary gains richer type-aware translation |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects — HTTP serving is already an effect
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Named binding is specifically FOR machine callers (agents)

---

## Systemic Audit

**Is this a one-off or part of a larger pattern?**

Yes — the **"serve-api agent ergonomics" gap**. Three separate docparse messages all describe the same root problem: serve-api was designed for simple positional calls, but agents need standard HTTP conventions.

**What already exists (do NOT re-implement):**
- `@nowrap` — **Already implemented** in `routes.go:272-281`. Skips envelope, sets `Content-Type: application/json`, adds `X-Elapsed-Ms` header. The docparse message requesting this is already satisfied.
- `_body`/`_status`/`_headers` pattern — **Already implemented** in `writeRawResponse()`. Enables custom headers + status codes on binary responses.

**What's genuinely new:**
1. **Named JSON parameter binding** — `parseArgs()` only supports `{"args": [...]}` or single-value. No way to send `{"path": "file.docx", "output_format": "blocks"}` and have it bind to function parameters by name.
2. **Custom headers on @nowrap responses** — `@nowrap` currently hardcodes `Content-Type: application/json` with no way to add headers like `X-Request-Id` or `X-RateLimit-*` without using the full `_body` pattern.

---

## Problem Statement

### Problem 1: Agents Must Know Positional Argument Order

**Current State:**
```bash
# Agent must construct positional array — fragile, order-dependent
curl -X POST /api/docparse/parseFile \
  -d '{"args": ["data/sample.docx", "blocks"]}'
```

The only alternatives are `{"args": ["val1", "val2"]}` (positional) or sending the entire body as a single argument. Neither is how agents naturally communicate — they send named JSON objects.

**Impact:**
- Every agent integration requires documentation of argument positions
- Positional order is an implementation detail that leaks through the API boundary
- OpenAPI spec shows parameters, but agents can't use them directly

### Problem 2: @nowrap Responses Can't Set Custom Headers

**Current State:**
- `@nowrap` writes `Content-Type: application/json` and `X-Elapsed-Ms` — nothing else
- To set custom headers (e.g., `X-Request-Id`, `X-RateLimit-*`), you must use the full `_body`/`_status`/`_headers` record pattern
- The `_body` pattern is verbose for simple JSON responses that just need an extra header

**Impact:**
- ~6 `@raw` endpoints in v0.9.0 could be simpler `@nowrap` endpoints if `@nowrap` supported headers
- HTTP-level tooling (proxies, load balancers) expects headers, not body fields

---

## Goals

**Primary Goal:** Make serve-api agent-friendly by supporting named JSON parameter binding and custom response headers on `@nowrap` endpoints.

**Success Metrics:**
- Agent sends `{"path": "file.docx", "output_format": "blocks"}` → binds to `parseFile(path: string, outputFormat: string)` automatically
- `{"args": [...]}` still works (backward compatible)
- `@nowrap` functions can return `{_headers: {...}, ...rest}` to set custom headers while returning `rest` as JSON body
- All existing serve-api behavior unchanged

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| snake_case JSON → camelCase AILANG mapping | Determines how all future APIs accept parameters | human | design | high |
| Unknown JSON fields: ignore vs 400 error | Affects backward compatibility with loose callers | agent | compile | low |
| How `@nowrap` gets custom headers | Sets pattern for all non-_body header customization | human | design | med |
| Parameter name source: AST vs iface | Determines what metadata serve-api needs access to | agent | compile | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] snake_case → camelCase: **Yes, automatic conversion** (standard JSON convention)
- [x] Unknown fields: **Ignore silently** (forward-compatible with schema evolution)
- [ ] @nowrap header mechanism: **Option A** (magic `_headers` field) vs **Option B** (new `@headers` annotation) — see Solution Design
- [x] Parameter name source: **AST** (`FuncDecl.Params[i].Name`) extracted during `extractRouteAnnotations()`

---

## Solution Design

### Overview

Two independently shippable features:

1. **Named Parameter Binding** — Parse JSON object keys as parameter names, match to function params
2. **@nowrap Custom Headers** — Detect `_headers` field in @nowrap result, extract and set as HTTP headers

### Feature 1: Named Parameter Binding

**Core change:** Modify `parseArgs()` in `handler.go` and `callFunction()` in `routes.go` to support named JSON objects.

**How it works:**

1. Client sends JSON body: `{"path": "file.docx", "output_format": "blocks"}`
2. serve-api detects it's a JSON object (not `{"args": [...]}`)
3. Looks up function's parameter names from `ExportInfo.ParamNames`
4. Maps JSON keys to parameters using snake_case → camelCase conversion
5. Orders values to match function's parameter positions
6. Falls back to single-arg if no parameter names match

**Name mapping rules:**
- `output_format` → `outputFormat` (snake_case → camelCase)
- `path` → `path` (exact match, no conversion needed)
- `outputFormat` → `outputFormat` (already camelCase, exact match)
- Matching is case-insensitive after normalization

**Data flow:**

```
JSON body: {"path": "file.docx", "output_format": "blocks"}
     ↓
parseNamedArgs(body, paramNames=["path", "outputFormat"])
     ↓
Match: "path" → param[0], "output_format" → normalize → "outputFormat" → param[1]
     ↓
args = ["file.docx", "blocks"]  (positional, in parameter order)
     ↓
engine.CallPreserveFloats(module, func, args...)
```

**Precedence (backward compatible):**
1. If body has `"args"` key with array value → positional (existing behavior)
2. If body is a JSON object → try named binding using ExportInfo.ParamNames
3. If named binding matches 0 params → fall back to single-arg (existing behavior)
4. If body is not an object → single-arg (existing behavior)

**ExportInfo changes:**

```go
type ExportInfo struct {
    Name        string   `json:"name"`
    Type        string   `json:"type"`
    Pure        bool     `json:"pure"`
    Arity       int      `json:"arity"`
    ParamNames  []string `json:"param_names,omitempty"`  // NEW: parameter names in order
    RouteMethod string   `json:"route_method,omitempty"`
    RoutePath   string   `json:"route_path,omitempty"`
    IsRaw       bool     `json:"is_raw,omitempty"`
    IsNowrap    bool     `json:"is_nowrap,omitempty"`
}
```

**Extracting parameter names:**

Parameter names come from the AST during `extractRouteAnnotations()`. The parsed `ast.File` is already available there — iterate `fn.Params` to get names.

```go
// In extractRouteAnnotations, for each exported function:
paramNames := make([]string, len(fn.Params))
for i, p := range fn.Params {
    paramNames[i] = p.Name
}
export.ParamNames = paramNames
```

**Named arg parsing function:**

```go
// parseNamedArgs maps JSON object keys to function parameter names.
// Returns positional args array in parameter order, or nil if no match.
func parseNamedArgs(body map[string]interface{}, paramNames []string) []interface{} {
    if len(paramNames) == 0 {
        return nil
    }
    args := make([]interface{}, len(paramNames))
    matched := 0
    for i, name := range paramNames {
        // Try exact match first
        if val, ok := body[name]; ok {
            args[i] = val
            matched++
            continue
        }
        // Try snake_case version of camelCase param name
        snake := camelToSnake(name)
        if val, ok := body[snake]; ok {
            args[i] = val
            matched++
            continue
        }
    }
    if matched == 0 {
        return nil // no matches, fall back to single-arg
    }
    return args
}
```

### Feature 2: @nowrap Custom Headers

**Problem:** `@nowrap` always returns the full result as JSON. There's no way to add custom headers without using the verbose `_body`/`_status`/`_headers` pattern.

**Proposed solution:** If a `@nowrap` function returns a record containing a `_headers` field, extract those as HTTP headers and exclude `_headers` from the JSON response body.

```ailang
@nowrap
@route("POST", "/api/v1/parse")
export func parseFile(path: string) -> {data: string, _headers: {string: string}} ! {IO}
  {
    data = parse(path),
    _headers = {
      "X-Request-Id" = "req_abc123",
      "X-RateLimit-Remaining" = "99"
    }
  }
```

**Response:**
```
HTTP/1.1 200 OK
Content-Type: application/json
X-Request-Id: req_abc123
X-RateLimit-Remaining: 99

{"data": "parsed content"}
```

The `_headers` field is stripped from the JSON body and applied as HTTP headers. The `_` prefix convention is already established by `_body`/`_status`/`_headers`.

**Implementation in `callFunction()`:**

```go
if opt.Nowrap {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("X-Elapsed-Ms", fmt.Sprintf("%d", elapsed))

    // Check for _headers in result record
    if rec, ok := result.(*eval.RecordValue); ok {
        if headersVal, ok := rec.Fields["_headers"]; ok {
            if headersRec, ok := headersVal.(*eval.RecordValue); ok {
                for k, v := range headersRec.Fields {
                    if sv, ok := v.(*eval.StringValue); ok {
                        w.Header().Set(k, sv.Value)
                    }
                }
            }
            // Remove _headers from result before JSON encoding
            delete(rec.Fields, "_headers")
        }
    }

    w.WriteHeader(http.StatusOK)
    enc := json.NewEncoder(w)
    enc.SetIndent("", "  ")
    _ = enc.Encode(goResult)
    return
}
```

### Implementation Plan

**Phase 1: Named Parameter Binding** (~12 hours)

- [ ] Add `ParamNames []string` to `ExportInfo` struct
- [ ] Extract param names from AST in `extractRouteAnnotations()` for @route functions
- [ ] Extract param names in `extractModuleInfo()` from `iface.Exports` for auto-routed functions
- [ ] Implement `parseNamedArgs(body, paramNames)` function
- [ ] Implement `camelToSnake()` utility function
- [ ] Update `callFunction()` to pass `ExportInfo` (with param names) through to arg parsing
- [ ] Update auto-route handler to also support named binding
- [ ] Update OpenAPI generator to include parameter names in spec
- [ ] Tests: named binding, snake_case conversion, fallback to positional, mixed scenarios

**Phase 2: @nowrap Custom Headers** (~4 hours)

- [ ] Detect `_headers` field in @nowrap result records
- [ ] Extract and set HTTP headers from `_headers` record
- [ ] Remove `_headers` from JSON response body
- [ ] Tests: @nowrap with _headers, @nowrap without _headers (unchanged), header values

**Phase 3: Documentation & Examples** (~2 hours)

- [ ] Update `docs/docs/guides/serve-api.md` with named binding section
- [ ] Add example: `examples/runnable/serve_api_named_params.ail`
- [ ] Update CHANGELOG.md
- [ ] Reply to docparse inbox messages with implementation status

### Files to Modify/Create

**Modified files:**
- `internal/apiserver/server.go` (~+15 LOC) — Add `ParamNames` to `ExportInfo`, extract in `extractModuleInfo()`
- `internal/apiserver/routes.go` (~+60 LOC) — Extract param names from AST, `parseNamedArgs()`, `camelToSnake()`, @nowrap header extraction
- `internal/apiserver/handler.go` (~+30 LOC) — Named binding in auto-route handler
- `internal/apiserver/openapi.go` (~+10 LOC) — Parameter names in OpenAPI spec
- `docs/docs/guides/serve-api.md` (~+50 LOC) — Named binding documentation

**New files:**
- `internal/apiserver/named_args_test.go` (~120 LOC) — Tests for named parameter binding
- `examples/runnable/serve_api_named_params.ail` (~20 LOC) — Example

---

## Examples

### Example 1: Named Parameter Binding (Agent-Friendly)

**Before (positional — fragile):**
```bash
curl -X POST http://localhost:8080/api/docparse/parseFile \
  -H "Content-Type: application/json" \
  -d '{"args": ["data/sample.docx", "blocks"]}'
```

**After (named — self-documenting):**
```bash
curl -X POST http://localhost:8080/api/docparse/parseFile \
  -H "Content-Type: application/json" \
  -d '{"path": "data/sample.docx", "output_format": "blocks"}'
```

**AILANG function (unchanged):**
```ailang
export func parseFile(path: string, outputFormat: string) -> string ! {IO, FS}
  -- implementation
```

### Example 2: @nowrap with Custom Headers

```ailang
@nowrap
@route("POST", "/api/v1/parse")
export func parseFile(path: string) -> {data: string, count: int, _headers: {string: string}} ! {IO}
  let result = parse(path)
  {
    data = result.text,
    count = result.elementCount,
    _headers = {
      "X-Request-Id" = generateId(),
      "X-RateLimit-Remaining" = "99"
    }
  }
```

**Response:**
```
HTTP/1.1 200 OK
Content-Type: application/json
X-Request-Id: req_abc123
X-RateLimit-Remaining: 99
X-Elapsed-Ms: 42

{
  "data": "parsed content...",
  "count": 15
}
```

### Example 3: Backward Compatibility (positional still works)

```bash
# Old positional format — still works
curl -d '{"args": ["data/sample.docx", "blocks"]}' ...

# New named format — also works
curl -d '{"path": "data/sample.docx", "output_format": "blocks"}' ...

# Single arg — still works
curl -d '"data/sample.docx"' ...
```

---

## Success Criteria

- [ ] `{"path": "file.docx", "output_format": "blocks"}` binds correctly to `parseFile(path: string, outputFormat: string)`
- [ ] `{"args": ["file.docx", "blocks"]}` still works (backward compat)
- [ ] `camelToSnake` mapping: `outputFormat` → `output_format` matches
- [ ] Unmatched JSON fields are silently ignored
- [ ] Missing required params get zero values (empty string, 0, etc.)
- [ ] @nowrap with `_headers` field sets HTTP headers and excludes from body
- [ ] @nowrap without `_headers` field works unchanged
- [ ] OpenAPI spec includes parameter names
- [ ] All existing serve-api tests pass
- [ ] Documentation updated in serve-api guide

---

## Testing Strategy

**Unit tests:**
- `parseNamedArgs()` with exact match, snake_case match, no match, partial match
- `camelToSnake()` edge cases: single word, acronyms, already snake_case
- @nowrap header extraction: present, absent, non-record value

**Integration tests:**
- Full HTTP request with named JSON body → correct function invocation
- Named binding + @route custom path
- Mixed: some params named, some missing (zero-value fill)
- Backward compat: `{"args": [...]}` still takes precedence

**Manual testing:**
- Agent-style curl requests against DocParse endpoints
- Verify OpenAPI spec reflects parameter names

---

## Deferred Decisions

The following are intentionally left open for the implementer:

- **Partial match behavior** — When some JSON keys match params and others don't, agent may choose whether to bind partial matches or require all-or-nothing
- **iface parameter name extraction** — For auto-routed (non-@route) functions, the iface may or may not expose param names. Agent may choose to support named binding only for @route functions initially.
- **@nowrap `_status` support** — Whether to also extract `_status` from @nowrap results (like `_headers`). Agent may add if trivial.

---

## Non-Goals

**Not attempted in this feature:**
- **New `HttpResponse` return type in the type system** — The `_headers`/`_body`/`_status` convention works without type system changes
- **Effect-based header setting** — `Http.setHeader()` would require a new effect type; too heavy for this use case
- **`@header` annotation** — Static annotations can't set dynamic values like request IDs
- **Rate limiting implementation** — Use proxies/load balancers; we only provide the header convention
- **WASM package vendoring** — Separate feature, separate design doc (from message `edc29247`)

---

## Timeline

**Day 1** (~8 hours):
- Phase 1a: ExportInfo changes, param name extraction, `camelToSnake()`
- Phase 1b: `parseNamedArgs()` implementation and unit tests

**Day 2** (~8 hours):
- Phase 1c: Wire named binding into `callFunction()` and auto-route handler
- Phase 1d: OpenAPI integration, integration tests

**Day 3** (~4 hours):
- Phase 2: @nowrap custom headers
- Phase 3: Documentation, examples, changelog

**Total: ~20 hours across 3 days**

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| iface doesn't expose parameter names for auto-routed functions | Med | Start with @route functions only (AST available); extend to auto-routes in follow-up |
| snake_case conversion ambiguity (e.g., `httpURL` → `http_url` vs `http_u_r_l`) | Low | Use Go's standard `camelToSnake` pattern; document edge cases |
| `_headers` convention conflicts with user field names | Low | `_` prefix convention already established; document clearly |
| Named binding changes arg parsing for all endpoints | Med | `{"args": [...]}` check comes FIRST (backward compat); named binding only activates for plain objects |

---

## Related Documents

**Directly relevant (already implemented — do NOT re-implement):**
- [M-ROUTE-NOWRAP](../../planned/v0_9_5/m-route-nowrap.md) — @nowrap annotation (IMPLEMENTED in v0.9.4)
- [M-SERVE-API-DX](../../planned/v0_9_4/m-serve-api-dx.md) — Custom routes, file upload, binary response, auth (IMPLEMENTED)
- [M-SERVE-API-CONCURRENCY](../../implemented/v0_9_4/m-serve-api-concurrency.md) — Per-request evaluator isolation

**Planned (check for overlap):**
- [M-SERVE-API-GET-ARGS](../../planned/v0_10_0/m-serve-api-get-args.md) — GET query param support (our named binding design should align)
- [M-CODEGEN-API-SERVER](../../planned/v0_10_0/m-codegen-api-server.md) — Compiled Go server (should consume our param metadata)

**Inbox messages (source):**
- `32d46e7d` — Named JSON parameter binding request
- `d6f8c5ff` — Custom HTTP response headers request
- `ce694515` — @nowrap annotation request (already satisfied)
- `edc29247` — WASM vendoring (out of scope, separate doc needed)

---

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- [serve-api guide](docs/docs/guides/serve-api.md) — Existing documentation
- `internal/apiserver/routes.go` — Current @nowrap and _body/_headers implementation
- `internal/apiserver/handler.go:parseArgs()` — Current arg parsing (to be extended)
- `internal/ast/ast_decl.go:FuncDecl.Params` — AST parameter names source

---

## Future Work

- **GET request named parameters** — Query string `?path=file.docx&output_format=blocks` with same binding logic (M-SERVE-API-GET-ARGS)
- **Request validation** — Validate named params against function types at the HTTP boundary
- **WASM package vendoring** — `ailang vendor --target wasm` (separate design doc from message `edc29247`)
- **Per-route middleware** — `@auth`, `@rateLimit` annotations for per-endpoint configuration

---

**Document created**: 2026-03-26
**Last updated**: 2026-03-26
