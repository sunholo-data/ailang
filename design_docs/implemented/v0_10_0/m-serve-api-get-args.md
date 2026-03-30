# M-SERVE-API-GET-ARGS: GET Route Function Arguments via Query Parameters

**Status**: Planned
**Target**: v0.10.0
**Priority**: P2 (Enhancement — not blocking, improves DX)
**Estimated**: 0.5 days
**Dependencies**: None

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to execution semantics |
| A2: Replayability | 0 | Same args produce same results |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | No verification changes |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | GET with query params is standard REST — easier for agents to discover and call |
| A8: Minimal Syntax | 0 | No language syntax changes |
| A9: Cost Visibility | 0 | No cost impact |
| A10: Composability | +1 | GET routes become composable via standard URL conventions |
| A11: Structured Failure | 0 | No failure path changes |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +2** → **Decision: Move forward**

## Problem Statement

The `@route("GET", "/path")` annotation system already exists (`internal/apiserver/routes.go`). Custom GET routes are registered via `registerCustomRoutes` and correctly dispatched to `callFunction`. However, `callFunction` (`routes.go:110`) only extracts arguments from the **request body** (JSON or multipart). For GET requests the body is empty, so `args` is always `nil`.

This means:
- `@route("GET", "/search")` works for zero-arity functions
- But any GET-annotated function with parameters receives no arguments — query params are ignored
- The catch-all `/api/` handler (`handleFunctionCall`) additionally rejects non-POST entirely (`handler.go:30`)

### Existing Infrastructure

- `@route("GET", "/path")` annotations → parsed in `extractRouteAnnotations` (`routes.go:40-63`)
- `registerCustomRoutes` → registers GET handlers that call `callFunction` (`routes.go:87-106`)
- `callFunction` → shared caller, but only reads body, never query params (`routes.go:110-223`)

### Current Behavior

```
# Custom GET route — dispatched but args lost:
@route("GET", "/search")
export func search(query: string) -> string ! {IO}

GET /search?query=hello
→ 200 but search("") — query param ignored, function receives no args

# Catch-all — rejected entirely:
GET /api/mymod/greet?name=Alice
→ 405 Method Not Allowed: "use POST with JSON body to call functions"
```

### Desired Behavior

```
GET /search?query=hello
→ 200 {"result": "Results for: hello", ...}

GET /api/mymod/add?args=3&args=5
→ 200 {"result": 8, ...}
```

## Related: CORS Authorization Header Fix (Done)

The CORS issue (commit `8fb51252`) is resolved. The `corsWrap` handler in `server.go:402` now includes `Authorization` in `Access-Control-Allow-Headers`, allowing Firebase Auth bearer tokens from cross-origin dashboard requests.

## Solution Design

### Approach: Add Query Parameter Parsing to `callFunction`

The fix is in `callFunction` (`routes.go:110`), the shared caller used by both custom routes and the catch-all. When the request is GET (or body is empty), fall back to query parameters.

**Two query parameter conventions:**

1. **Positional parameters**: `?args=val1&args=val2` → passed as positional args `["val1", "val2"]`
2. **Named parameters**: `?name=Alice&age=30` → passed as a single record argument `{name: "Alice", age: "30"}`

### Implementation

**Change 1: `callFunction` in `routes.go` — query param fallback**

In the arg-parsing section of `callFunction`, after the existing body parsing, add a fallback:

```go
// After body parsing produces nil args, try query parameters (for GET requests)
if len(args) == 0 && len(r.URL.Query()) > 0 {
    args = parseQueryArgs(r.URL.Query())
}
```

**Change 2: Add `parseQueryArgs` to `handler.go`**

```go
func parseQueryArgs(query url.Values) []interface{} {
    // Convention 1: positional via ?args=...&args=...
    if positional, ok := query["args"]; ok {
        result := make([]interface{}, len(positional))
        for i, a := range positional {
            result[i] = tryParseJSON(a)
        }
        return result
    }

    // Convention 2: named params → single record arg
    record := make(map[string]interface{})
    for key, values := range query {
        if len(values) == 1 {
            record[key] = tryParseJSON(values[0])
        } else {
            parsed := make([]interface{}, len(values))
            for i, v := range values {
                parsed[i] = tryParseJSON(v)
            }
            record[key] = parsed
        }
    }
    if len(record) > 0 {
        return []interface{}{record}
    }
    return nil
}
```

**Change 3: Allow GET in `handleFunctionCall` catch-all (`handler.go`)**

```go
if r.Method != "POST" && r.Method != "GET" {
    writeJSON(w, http.StatusMethodNotAllowed, FunctionCallResponse{
        Error: "use GET with query params or POST with JSON body",
    })
    return
}
```

### Files to Modify

- `internal/apiserver/routes.go` — Add query param fallback in `callFunction` (~5 LOC)
- `internal/apiserver/handler.go` — Add `parseQueryArgs`, allow GET in catch-all (~35 LOC)
- `internal/apiserver/handler_test.go` — Test GET with query params (~40 LOC)
- `cmd/ailang/serve_api.go` — Update help text to document GET support (~5 LOC)
- `internal/apiserver/openapi.go` — Generate GET operations with query params in OpenAPI spec

### Implementation Plan

**Phase 1: Basic GET support** (~2 hours)
- [ ] Allow GET method in `handleFunctionCall`
- [ ] Add `parseQueryArgs` for positional (`?args=`) and named (`?key=val`) conventions
- [ ] Route to `callFunction` same as POST
- [ ] Tests for both conventions

**Phase 2: OpenAPI + docs** (~1 hour)
- [ ] Update OpenAPI generator to emit GET operations for functions
- [ ] Update help text and startup banner
- [ ] Update Swagger UI to show GET endpoints

## Success Criteria

- [ ] `GET /api/mod/func?args=1&args=2` calls func with positional args
- [ ] `GET /api/mod/func?name=Alice` calls func with record arg `{name: "Alice"}`
- [ ] `GET /api/mod/func` with no query params calls zero-arity func
- [ ] POST continues to work unchanged
- [ ] OpenAPI spec includes GET operations
- [ ] All existing tests pass

## Non-Goals

- Path parameters (`:id` in `/users/:id`) — separate feature, needs route pattern matching
- Content negotiation (Accept header) — out of scope
- Caching headers (ETag, Cache-Control) — separate concern

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Query param type ambiguity (string "3" vs int 3) | Low | `tryParseJSON` attempts JSON parse first, falls back to string |
| GET with side effects violates HTTP semantics | Med | Document that GET should only be used for pure functions; warn in docs |
| URL length limits for large args | Low | POST remains available for complex payloads |

---

**Document created**: 2026-03-24
**Last updated**: 2026-03-24
