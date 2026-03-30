# M-SERVE-API-ENDPOINT-FILTERING: Endpoint Filtering & JSON Auto-Unwrap for serve-api

**Status**: Planned
**Target**: v0.9.5
**Priority**: P1 (High — blocks DocParse production deployment)
**Estimated**: 1-2 days
**Dependencies**: M-ROUTE-NOWRAP (implemented — @nowrap annotation exists)
**Milestone ID**: M-SERVE-API-ENDPOINT-FILTERING
**Created**: 2026-03-29
**Source**: DocParse agent messages `64d52a2e`, `ec758118`, `52c85477`

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to evaluation semantics — HTTP layer only |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | +1 | `@noexpose` makes endpoint visibility explicit at declaration site |
| A4: Explicit Authority | +1 | `--routes-only` prevents accidental exposure of internal functions |
| A5: Bounded Verification | +1 | Endpoint exposure is locally verifiable per-function |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | AI agents reading OpenAPI spec see only intentional API surface, not 133 internal functions |
| A8: Minimal Syntax | 0 | Reuses existing annotation mechanism, no new grammar |
| A9: Cost Visibility | 0 | No cost impact |
| A10: Composability | +1 | `--routes-only`, `@noexpose`, `@nowrap` compose independently |
| A11: Structured Failure | 0 | No failure path changes |
| A12: System Boundary | +1 | Separates module visibility (`export`) from HTTP exposure — two distinct boundaries |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): `@noexpose` makes effects MORE legible, not less
- [x] A4 (Authority): Reduces ambient access (functions no longer auto-exposed)
- [x] A7 (Machines First): Cleaner OpenAPI spec is better for machine consumption

---

## Systemic Audit

**Is this a one-off or part of a larger pattern?**

Yes — this is the **"export conflates two concerns" gap**:

1. `export` in AILANG means "visible to other modules" (needed for cross-module imports)
2. `serve-api` treats ALL exports as HTTP endpoints
3. Any project with internal helper modules hits this: exported-for-import functions leak as API endpoints

**Pattern**: DocParse has 133 exported functions (for cross-module use) but only 21 intentional `@route` endpoints. The gap exposes security-sensitive functions (`generateApiKey`, `validateApiKey`, `recordUsage`, `apiKeyGenHexParts`) alongside the public API. This affects any non-trivial AILANG project deploying via `serve-api`.

**Related gap**: @nowrap correctly strips the response envelope but double-encodes JSON strings. Functions returning `encode(jo([...]))` produce valid JSON strings that get JSON-encoded again as string literals. This is a separate but related DX issue reported in the same batch of messages.

**Audit of related work:**
- **M-ROUTE-NOWRAP** (v0.9.5, implemented): `@nowrap` exists and works for envelope stripping
- **M-SERVE-API-DX** (v0.9.4, partially implemented): Custom routes, auth, file upload
- **M-SERVE-API-GET-ARGS** (v0.10.0): GET route query params — orthogonal, no conflict

---

## Problem Statement

### Problem 1: `export` Conflates Module Visibility and HTTP Exposure

DocParse exports 133 functions for cross-module imports. `serve-api` exposes ALL of them as HTTP endpoints at `/api/{module}/{function}`. This leaks internal functions — including security-sensitive ones — alongside the 21 intentional `@route` endpoints.

**Current State:**
- Only `IsExport == true` functions become endpoints (no filtering beyond that)
- No `--routes-only` flag exists
- No `@noexpose` annotation exists
- OpenAPI spec includes all 133 functions mixed with the 21 intentional routes
- AI agents reading the spec see internal helpers alongside the public API

**Impact:**
- Security: internal functions like `generateApiKey`, `apiKeyGenHexParts` callable via HTTP
- DX: consumers and AI agents confused by 133 endpoints when only 21 are the real API
- Any multi-module AILANG project deploying via `serve-api` hits this

### Problem 2: @nowrap Double-Encodes JSON Strings

Functions returning `encode(jo([kv("status", js("healthy"))]))` produce the string `{"status":"healthy"}`. When `@nowrap` writes this via `json.Encoder.Encode()`, the string gets JSON-encoded as a string literal:

```
"{"status":"healthy"}"     ← what @nowrap returns (double-encoded)
{"status": "healthy"}      ← what consumers expect
```

**Current flow:**
1. AILANG function returns string value `{"status":"healthy"}`
2. `embed.ToGo()` converts to Go `string`
3. `json.NewEncoder(w).Encode(goResult)` encodes the string → double-quoted JSON string literal

**Impact:**
- Every `@nowrap` endpoint returning JSON via `encode(jo(...))` is broken
- Consumers must `JSON.parse(response)` to unwrap the extra encoding layer
- DocParse has switched all 20 endpoints to `@nowrap` but still hits this

---

## Goals

**Primary Goal:** Give `serve-api` users control over which exported functions become HTTP endpoints, and fix JSON string double-encoding in `@nowrap` responses.

**Success Metrics:**
- `--routes-only` flag limits endpoints to `@route`-annotated functions only
- `@noexpose` hides individual exported functions from HTTP exposure
- OpenAPI spec respects both filtering mechanisms
- `@nowrap` returns `{"status":"healthy"}` not `"{\"status\":\"healthy\"}"` for JSON string results
- Zero breaking changes to existing behavior (both features are opt-in)

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| `--routes-only` as CLI flag vs config | Determines deployment ergonomics | agent | design | low |
| `@noexpose` annotation name | Must be intuitive, not clash with existing annotations | agent | design | low |
| Auto-unwrap scope: `@nowrap` only vs global | Affects all existing serve-api users if global | human | design | med |
| JSON detection strategy (try-parse vs content-type hint) | Performance and correctness trade-off | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] `--routes-only` as CLI flag (not config file) — matches existing `serve-api` flag pattern
- [x] `@noexpose` annotation name — clear, negative, matches `@nowrap` naming convention
- [x] Auto-unwrap scope: **@nowrap only** — safer, opt-in, no surprise behavior changes for existing users

---

## Solution Design

### Overview

Two independent features, each shippable separately:

1. **Endpoint Filtering** — `--routes-only` flag + `@noexpose` annotation to control which exports become HTTP endpoints
2. **JSON Auto-Unwrap** — Detect valid JSON strings in `@nowrap` responses and write as raw bytes instead of double-encoding

### Architecture

```
┌─────────────────────────────────────────────────┐
│              Module Loading                       │
│  ├─ Extract exports (existing)                    │
│  ├─ Extract @route annotations (existing)         │
│  ├─ Extract @noexpose annotations (NEW)           │
│  └─ Filter: --routes-only / @noexpose (NEW)       │
├─────────────────────────────────────────────────┤
│              Route Registration                   │
│  ├─ Custom @route endpoints (existing)            │
│  ├─ Auto /api/{mod}/{func} endpoints              │
│  │   └─ SKIP if --routes-only or @noexpose (NEW)  │
│  └─ OpenAPI spec generation                       │
│      └─ SKIP filtered functions (NEW)             │
├─────────────────────────────────────────────────┤
│              Response Writing                     │
│  ├─ Default: FunctionCallResponse envelope        │
│  ├─ @nowrap: raw JSON                             │
│  │   └─ Auto-unwrap JSON strings (NEW)            │
│  └─ _body: raw HTTP response                      │
└─────────────────────────────────────────────────┘
```

### Phase 1: Endpoint Filtering (~4 hours)

**1a. `--routes-only` CLI flag**

Add flag to `serve_api.go` that restricts auto-generated `/api/{module}/{function}` endpoints to only `@route`-annotated functions. Custom `@route` endpoints are always registered regardless of this flag.

```go
// cmd/ailang/serve_api.go — add flag
routesOnly := flags.Bool("routes-only", false, "Only expose @route-annotated functions as HTTP endpoints")

// internal/apiserver/server.go — Config field
type Config struct {
    // ...existing fields...
    RoutesOnly bool
}
```

**1b. `@noexpose` annotation**

Mark individual exported functions that should NOT become HTTP endpoints. Parsed alongside `@route`, `@raw`, `@nowrap`.

```ailang
-- Exported for cross-module import, but NOT an HTTP endpoint
@noexpose
export func generateApiKey(userId: string) -> string ! {IO}
  -- ...
```

```go
// internal/apiserver/routes.go — add to ExportInfo
type ExportInfo struct {
    // ...existing fields...
    IsNoExpose bool // @noexpose annotation
}

// In extractRouteAnnotations():
isNoExpose := fn.GetAnnotation("noexpose") != nil
```

**1c. Filter in route registration**

```go
// internal/apiserver/server.go — in the catch-all registration loop
for _, exp := range mod.Exports {
    // Skip if @noexpose
    if exp.IsNoExpose {
        continue
    }
    // Skip non-@route functions when --routes-only is set
    if s.routesOnly && exp.RouteMethod == "" {
        continue
    }
    // ...register auto-route as before...
}
```

**1d. Filter in OpenAPI spec generation**

Apply the same filtering logic in `buildOpenAPISpec()` so the generated spec only includes exposed endpoints.

**Implementation tasks:**
- [ ] Add `--routes-only` flag to `serve_api.go` and `Config` struct
- [ ] Add `IsNoExpose` field to `ExportInfo` struct
- [ ] Parse `@noexpose` in `extractRouteAnnotations()`
- [ ] Filter auto-route registration by `--routes-only` and `@noexpose`
- [ ] Filter OpenAPI spec generation to match
- [ ] Filter A2A Agent Card skills to match
- [ ] Filter `/api/_meta/modules` response to indicate filtered functions
- [ ] Add startup log line: `"routes-only mode: N endpoints exposed (M filtered)"`
- [ ] Tests for all filtering combinations

### Phase 2: JSON Auto-Unwrap in @nowrap (~2 hours)

When `@nowrap` is active and `goResult` is a `string`, check if it's valid JSON. If so, write it as raw bytes instead of re-encoding through `json.Encoder`.

**Current code** (`routes.go:322-346`):
```go
if opt.Nowrap {
    w.Header().Set("Content-Type", "application/json")
    // ...header extraction...
    enc := json.NewEncoder(w)
    _ = enc.Encode(goResult)  // ← double-encodes strings!
    return
}
```

**Fixed code:**
```go
if opt.Nowrap {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("X-Elapsed-Ms", fmt.Sprintf("%d", elapsed))

    // Extract _headers from Go result map (existing logic)
    if m, ok := goResult.(map[string]interface{}); ok {
        if headersVal, ok := m["_headers"]; ok {
            // ...set HTTP headers, delete _headers...
        }
    }

    w.WriteHeader(http.StatusOK)

    // Auto-unwrap: if result is a string containing valid JSON,
    // write it as raw bytes instead of double-encoding
    if s, ok := goResult.(string); ok && isValidJSON(s) {
        w.Write([]byte(s))
        w.Write([]byte("\n"))  // match json.Encoder behavior
        return
    }

    enc := json.NewEncoder(w)
    enc.SetIndent("", "  ")
    _ = enc.Encode(goResult)
    return
}
```

**JSON validation helper:**
```go
// isValidJSON checks if a string is a valid JSON object or array.
// Only unwraps objects ({...}) and arrays ([...]) — not bare strings, numbers, etc.
func isValidJSON(s string) bool {
    s = strings.TrimSpace(s)
    if len(s) < 2 {
        return false
    }
    // Only unwrap JSON objects and arrays, not primitives
    if s[0] != '{' && s[0] != '[' {
        return false
    }
    return json.Valid([]byte(s))
}
```

**Implementation tasks:**
- [ ] Add `isValidJSON()` helper to `routes.go`
- [ ] Add auto-unwrap check in `@nowrap` response path
- [ ] Tests: JSON object string → raw JSON, JSON array string → raw JSON
- [ ] Tests: plain string → still JSON-encoded (no unwrap)
- [ ] Tests: invalid JSON string → still JSON-encoded
- [ ] Tests: non-string result → unchanged behavior

### Files to Modify/Create

**Modified files:**
- `cmd/ailang/serve_api.go` — Add `--routes-only` flag (~+5 LOC)
- `internal/apiserver/server.go` — Add `RoutesOnly` to Config, filter auto-route registration (~+15 LOC)
- `internal/apiserver/routes.go` — Add `IsNoExpose` to ExportInfo, parse `@noexpose`, auto-unwrap JSON in @nowrap (~+30 LOC)
- `internal/apiserver/openapi.go` — Filter spec generation by routes-only/noexpose (~+10 LOC)
- `internal/apiserver/a2a.go` — Filter Agent Card skills (~+5 LOC)
- `internal/apiserver/meta.go` — Annotate filtered functions in module list (~+5 LOC)

**New files:**
- `internal/apiserver/filtering_test.go` — Tests for endpoint filtering (~100 LOC)

---

## Examples

### Example 1: DocParse with --routes-only

**Before** (133 endpoints exposed):
```bash
ailang serve-api docparse/
# Exposes: /api/docparse/handlers/generateApiKey  ← security-sensitive!
#          /api/docparse/handlers/validateApiKey   ← security-sensitive!
#          /api/docparse/handlers/recordUsage      ← internal
#          /api/docparse/handlers/apiKeyGenHexParts ← internal
#          ... 129 more internal functions ...
#          Plus 21 @route endpoints
```

**After** (21 endpoints exposed):
```bash
ailang serve-api docparse/ --routes-only
# Exposes: ONLY @route-annotated functions
# /general/v0/general  ← @route("POST", "/general/v0/general")
# /api/v1/parse        ← @route("POST", "/api/v1/parse")
# ... 19 more @route endpoints
# Internal functions NOT accessible via HTTP
```

### Example 2: @noexpose for fine-grained control

```ailang
module docparse/billing

-- This is a public API endpoint
@nowrap
@route("GET", "/api/v1/usage")
export func getUsage(userId: string) -> Json ! {IO}
  encode(jo([kv("requests", ji(lookupUsage(userId)))]))

-- Exported for cross-module import, NOT an HTTP endpoint
@noexpose
export func generateApiKey(userId: string) -> string ! {IO}
  apiKeyGenHexParts(userId, timestamp())

-- Also exported for import, also hidden from HTTP
@noexpose
export func validateApiKey(key: string) -> bool ! {IO}
  checkKeyInStore(key)
```

### Example 3: @nowrap auto-unwrap fix

**Before** (double-encoded):
```bash
curl http://localhost:8080/api/v1/health
# Returns: "{\"status\":\"healthy\",\"version\":\"1.0\"}"
#          ^ string literal, needs JSON.parse()
```

**After** (auto-unwrapped):
```bash
curl http://localhost:8080/api/v1/health
# Returns: {"status":"healthy","version":"1.0"}
#          ^ raw JSON object, ready to use
```

### Example 4: Combining all features

```ailang
module myapp/api

import myapp/internal/auth (checkToken)  -- checkToken is @noexpose in auth module

@nowrap
@route("GET", "/api/v1/health")
export func health() -> string ! {}
  encode(jo([kv("status", js("healthy"))]))
  -- Returns: {"status":"healthy"}  (auto-unwrapped)

@nowrap
@route("POST", "/api/v1/data")
export func getData(query: string) -> string ! {IO}
  encode(jo([kv("results", ja(queryDB(query)))]))
  -- Returns: {"results": [...]}  (auto-unwrapped)
```

```bash
ailang serve-api myapp/ --routes-only --port 8080
# Only /api/v1/health and /api/v1/data exposed
# myapp/internal/auth functions NOT exposed despite being exported
```

---

## Success Criteria

- [ ] `--routes-only` flag limits auto-generated endpoints to @route functions only
- [ ] `@noexpose` prevents individual exported functions from becoming HTTP endpoints
- [ ] `@noexpose` functions still importable by other AILANG modules
- [ ] OpenAPI spec excludes filtered functions
- [ ] A2A Agent Card excludes filtered functions
- [ ] `@nowrap` returns `{"status":"healthy"}` not `"{\"status\":\"healthy\"}"` for JSON string results
- [ ] `@nowrap` with non-JSON strings still works (returns JSON-encoded string)
- [ ] `@nowrap` with record/list results still works (no change)
- [ ] All existing tests pass
- [ ] CHANGELOG updated
- [ ] Startup log shows filtered endpoint count

---

## Testing Strategy

**Unit tests:**
- `isValidJSON()`: JSON objects, arrays, strings, numbers, invalid JSON, empty string
- ExportInfo filtering: `IsNoExpose` correctly set from annotation
- Route registration: filtered with `--routes-only`, filtered with `@noexpose`, both combined

**Integration tests:**
- Full HTTP cycle: `--routes-only` blocks auto-route, allows @route
- Full HTTP cycle: `@noexpose` blocks specific function
- Full HTTP cycle: @nowrap with JSON string → raw JSON response
- Full HTTP cycle: @nowrap with plain string → JSON-encoded string response
- OpenAPI spec generation respects both filters
- A2A Agent Card respects both filters

**Manual testing:**
- DocParse deployment with `--routes-only`
- `curl` endpoints to verify filtered vs exposed
- Swagger UI shows only exposed endpoints

---

## Deferred Decisions

The following are intentionally left open for the implementer:

- **Startup log format** for filtered endpoint count — agent may choose
- **Meta endpoint behavior** — whether `/api/_meta/modules` should list filtered functions with a `"hidden": true` marker or omit them entirely. Agent may choose (recommendation: omit them to match the filtering intent)
- **@noexpose on non-exported functions** — whether this is a warning or silently ignored. Agent may choose (recommendation: silently ignore)

---

## Non-Goals

**Not attempted in this feature:**
- **Visibility keyword (`public` vs `export`)** — The root cause is that `export` conflates two concerns. A proper fix would be separate keywords (`export` for modules, `public` for HTTP). This is a language-level change deferred to v1.0.
- **Per-route auth** — Different API keys for different endpoints. Use reverse proxy for now.
- **Auto-unwrap in the default envelope** — Only `@nowrap` gets auto-unwrap. The default `FunctionCallResponse.result` field keeps string-encoded JSON for backward compatibility.
- **Content-Type negotiation** — `@nowrap` always returns `application/json`. Custom content types use the `_body`/`_headers` pattern.

---

## Timeline

**Day 1** (~6 hours):
- Phase 1: Endpoint filtering — `--routes-only`, `@noexpose`, OpenAPI, A2A, tests

**Day 2** (~3 hours):
- Phase 2: JSON auto-unwrap in @nowrap — detection, writing, tests
- CHANGELOG, documentation updates

**Total: ~9 hours across 2 days**

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `--routes-only` breaks existing deployments that rely on auto-routes | Med | Flag is opt-in, off by default. No breaking change. |
| JSON auto-unwrap changes @nowrap behavior for existing users | Low | Only affects string results that happen to be valid JSON objects/arrays. This is the correct behavior — the current double-encoding is a bug. |
| `@noexpose` parsed but not understood by older ailang versions | Low | Unknown annotations are already silently ignored by the parser |
| `isValidJSON()` performance on large strings | Low | `json.Valid()` is a fast scanner, no allocation. Only called for @nowrap string results. |

---

## Related Documents

**Directly relevant (serve-api):**
- [M-ROUTE-NOWRAP](m-route-nowrap.md) — @nowrap annotation design (prerequisite, implemented)
- [M-SERVE-API-DX](../v0_9_4/m-serve-api-dx.md) — Custom routes, auth, file upload (partially implemented)
- [M-SERVE-API-GET-ARGS](../v0_10_0/m-serve-api-get-args.md) — GET route query params (planned, orthogonal)
- [M-SERVE-API-TRANSITIVE-IMPORTS](../v0_9_4/m-serve-api-transitive-imports.md) — Transitive module loading

**Implemented (inform design):**
- [M-PROTOCOL-SUPPORT](../../implemented/v0_9_0/m-protocol-support.md) — MCP/A2A protocol support
- [M-SERVE-API-AGENT-ENHANCEMENTS](../v0_9_2/m-serve-api-agent-enhancements.md) — Agent-oriented API improvements

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- DocParse agent messages: `64d52a2e` (--routes-only + @noexpose), `ec758118` (auto-unwrap), `52c85477` (correction: @nowrap works, auto-unwrap needed), `20d5d3a2` (clarification: @raw is input-only)

---

## Future Work

- **`public` keyword** — Separate module visibility (`export`) from HTTP exposure (`public`) at the language level. Eliminates need for `@noexpose`.
- **Route groups** — `@routeGroup("/api/v1")` to prefix multiple routes without repeating the path.
- **Auto-unwrap in default envelope** — Extend JSON auto-unwrap to `FunctionCallResponse.result` field (breaking change, needs migration path).
- **Per-function middleware** — `@auth("admin")` annotation for per-endpoint auth requirements.

---

**Document created**: 2026-03-29
**Last updated**: 2026-03-29
