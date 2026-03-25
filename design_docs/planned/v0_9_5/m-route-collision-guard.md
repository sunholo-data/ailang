# M-ROUTE-COLLISION: Route Collision Guard & Optional A2A

**Status**: Planned
**Target**: v0.9.5
**Priority**: P0 (High — blocks all docparse Cloud Run deploys since March 23)
**Estimated**: 3-4 hours
**Dependencies**: None
**Milestone ID**: M-ROUTE-COLLISION
**Created**: 2026-03-25
**Source**: ailang-multivac agent message `8fc78138`

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to evaluation semantics |
| A2: Replayability | 0 | No change to traces |
| A3: Effect Legibility | +1 | Route conflicts become a compile-time warning instead of a runtime panic |
| A4: Explicit Authority | +1 | A2A protocol exposure is now opt-in via `--a2a` flag |
| A5: Bounded Verification | 0 | No change |
| A6: Safe Concurrency | 0 | No change |
| A7: Machines First | +1 | Agents can reason about which protocols are active from CLI flags |
| A8: Minimal Syntax | 0 | No new syntax — CLI flag + internal guard only |
| A9: Cost Visibility | 0 | No change |
| A10: Composability | +1 | Custom routes compose cleanly with built-in routes instead of panicking |
| A11: Structured Failure | +1 | Panics replaced with logged warnings |
| A12: System Boundary | 0 | No change |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): Explicit authority improved (A2A now opt-in)
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

## Problem Statement

Go 1.22+ `http.ServeMux` panics on duplicate route patterns. The AILANG apiserver registers built-in protocol routes (A2A, MCP, health, meta) in `buildRoutes()`, then calls `registerCustomRoutes()` which registers user-defined `@route` annotations. If a user's `.ail` file declares `@route("GET", "/.well-known/agent.json")`, the server panics on startup.

**Current State:**
- `server.go:429` registers `/.well-known/agent.json` unconditionally as a built-in A2A route
- `routes.go:111` registers custom `@route` paths with no collision check
- The docparse service in ailang-multivac declares `@route("GET", "/.well-known/agent.json")` (legacy, from before A2A was built-in)
- Result: `panic: pattern "/.well-known/agent.json" is already registered` on every deploy
- 3 consecutive terraform deploy failures in ailang-multivac-dev since March 23

**Impact:**
- All ailang-multivac Cloud Run services using `@route` on built-in paths are broken
- Any user who declares a custom route matching a built-in path will hit this
- The set of built-in paths will grow over time (OpenAPI, MCP, etc.), making collisions more likely

## Goals

**Primary Goal:** Prevent duplicate route registration panics and make A2A protocol exposure opt-in.

**Success Metrics:**
- `ailang serve-api` never panics from route collisions
- Custom `@route` on a built-in path logs a warning and is silently skipped (built-in wins)
- A2A routes (`/.well-known/agent.json`, `/a2a/`) only registered when `--a2a` flag is passed
- Existing deployments without `--a2a` continue to work (A2A routes simply absent)

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Built-in routes always win over @route | Prevents silent override of health checks, protocol endpoints | agent | design | low |
| A2A becomes opt-in via `--a2a` flag | Breaking change for anyone relying on always-on A2A | human | design | med |
| Default for `--a2a` is `false` | Conservative default — only serve protocols explicitly requested | human | design | med |
| Warning log vs error on collision | Warning is less disruptive but could be missed | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] A2A opt-in: Should `--a2a` default to `false` (conservative) or `true` (backward-compatible)?
- [ ] Should collision on built-in paths be a warning (log + skip) or a hard startup error?

## Solution Design

### Overview

Two changes:

1. **Route collision guard** in `registerCustomRoutes()` — maintain a set of built-in paths, skip any `@route` that collides, log a warning.
2. **`--a2a` CLI flag + `A2A` config field** — gate A2A route registration behind an explicit opt-in, mirroring the existing `--mcp-http` pattern for MCP.

### Architecture

**Component 1: Built-in route registry**

`buildRoutes()` already registers paths imperatively. Collect these paths into a `map[string]bool` before calling `registerCustomRoutes()`, then pass the set to the custom route registrator.

```go
// In buildRoutes():
builtinPaths := map[string]bool{
    "/api/_meta/modules":      true,
    "/api/_meta/modules/":     true,
    "/api/_health":            true,
    "/api/_meta/openapi.json": true,
    "/api/_meta/docs":         true,
    "/api/_meta/redoc":        true,
    "/api/":                   true, // catch-all
}
if s.a2aEnabled {
    builtinPaths["/.well-known/agent.json"] = true
    builtinPaths["/a2a/"] = true
}
if s.mcpEnabled {
    builtinPaths["/mcp/"] = true
}
```

**Component 2: Collision guard in registerCustomRoutes**

```go
func (s *Server) registerCustomRoutes(mux *http.ServeMux, builtinPaths map[string]bool) {
    routes := s.getCustomRoutes()
    for _, route := range routes {
        if builtinPaths[route.Path] {
            log.Printf("  WARNING: @route %s %s collides with built-in route, skipping",
                route.Method, route.Path)
            continue
        }
        // ... existing registration logic
    }
}
```

**Component 3: A2A opt-in flag**

Mirror the `--mcp-http` pattern exactly:

```go
// cmd/ailang/serve_api.go
a2aFlag := fs.Bool("a2a", false, "Enable A2A protocol endpoints (/.well-known/agent.json, /a2a/)")

// internal/apiserver/server.go
type Config struct {
    // ...existing fields...
    A2A bool // enable A2A endpoints
}

type Server struct {
    // ...existing fields...
    a2aEnabled bool
}
```

### Implementation Plan

**Phase 1: Route collision guard** (~1 hour)
- [ ] Add `builtinPaths` set construction in `buildRoutes()`
- [ ] Pass set to `registerCustomRoutes()`
- [ ] Skip + warn on collision
- [ ] Add unit test: custom route on `/.well-known/agent.json` does not panic

**Phase 2: A2A opt-in flag** (~1 hour)
- [ ] Add `A2A bool` to `Config` struct
- [ ] Add `a2aEnabled bool` to `Server` struct
- [ ] Wire `--a2a` flag in `serve_api.go`
- [ ] Gate A2A route registration behind `s.a2aEnabled`
- [ ] Update startup banner to show A2A status
- [ ] Update help text

**Phase 3: Tests & docs** (~1 hour)
- [ ] Test: collision guard skips conflicting route
- [ ] Test: A2A routes absent when `a2aEnabled=false`
- [ ] Test: A2A routes present when `a2aEnabled=true`
- [ ] Update `docs/docs/guides/serve-api.md` route table
- [ ] Update CHANGELOG.md
- [ ] Notify ailang-multivac to add `--a2a` flag where needed and remove stale `@route` from docparse

### Files to Modify/Create

**Modified files:**
- `internal/apiserver/server.go` — Add `a2aEnabled` field, gate A2A registration, build `builtinPaths` set (~+20 LOC)
- `internal/apiserver/routes.go` — Accept `builtinPaths` param, add collision check (~+10 LOC)
- `cmd/ailang/serve_api.go` — Add `--a2a` flag, wire to config (~+5 LOC)
- `internal/apiserver/auth_test.go` — Update test to set `a2aEnabled=true` (~+2 LOC)
- `internal/apiserver/protocol_test.go` — Update A2A tests to set `a2aEnabled=true` (~+2 LOC)
- `docs/docs/guides/serve-api.md` — Document `--a2a` flag (~+10 LOC)

**New files:**
- `internal/apiserver/routes_test.go` — Collision guard tests (~50 LOC)

## Examples

### Example 1: Route collision (current — panic)

```
$ ailang serve-api ./docparse/services/
panic: pattern "/.well-known/agent.json" is already registered
goroutine 1 [running]:
net/http.(*ServeMux).register(...)
```

### Example 2: Route collision (after — warning + skip)

```
$ ailang serve-api --a2a ./docparse/services/
  WARNING: @route GET /.well-known/agent.json collides with built-in route, skipping
  Custom route: POST /api/parse -> docparse/services/api_server/parse_document
  ...
  Listening on :8080
```

### Example 3: A2A disabled (default)

```
$ ailang serve-api ./myapp/
  GET  /api/_health                     Health check
  GET  /api/_meta/modules               List modules
  GET  /api/_meta/openapi.json          OpenAPI spec
  POST /api/{module}/{function}         Function call
  Listening on :8080
```

### Example 4: A2A enabled

```
$ ailang serve-api --a2a ./myapp/
  GET  /api/_health                     Health check
  GET  /.well-known/agent.json          (A2A Agent Card)
  POST /a2a/                            (A2A JSON-RPC)
  POST /api/{module}/{function}         Function call
  Listening on :8080
```

## Success Criteria

- [ ] `ailang serve-api` with conflicting `@route` does NOT panic (logs warning, skips)
- [ ] `ailang serve-api` without `--a2a` does NOT register `/.well-known/agent.json` or `/a2a/`
- [ ] `ailang serve-api --a2a` registers A2A routes as before
- [ ] Existing `--mcp-http` behavior unchanged
- [ ] All existing tests pass (updated for `a2aEnabled` where needed)
- [ ] Documentation updated

## Testing Strategy

**Unit tests:**
- `registerCustomRoutes` with a conflicting built-in path → skipped with warning
- `registerCustomRoutes` with a non-conflicting path → registered normally
- `buildRoutes` with `a2aEnabled=false` → no A2A paths in mux
- `buildRoutes` with `a2aEnabled=true` → A2A paths present

**Integration tests:**
- Full server startup with conflicting `@route` module — no panic, 200 on health
- A2A Agent Card returns 404 when `--a2a` not set
- A2A Agent Card returns 200 when `--a2a` set

**Manual testing:**
- Deploy docparse in ailang-multivac-dev with `--a2a` flag
- Verify no panic, service starts cleanly

## Deferred Decisions

- Prefix-based collision detection (e.g., `/api/` catch-all vs `/api/foo` custom) — agent may choose simplest approach (exact match first, expand later if needed)
- Whether to emit a structured warning vs plain log line — agent may choose

## Non-Goals

- Allowing custom routes to **override** built-in routes — built-in always wins
- Route priority/ordering system — out of scope, not needed
- A2A authentication or authorization — separate concern
- Removing the stale `@route` from docparse — that's an ailang-multivac fix

## Timeline

Single sprint, estimated 3-4 hours total:
- Phase 1: Route collision guard (1h)
- Phase 2: A2A opt-in flag (1h)
- Phase 3: Tests & documentation (1-2h)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Breaking change: existing deploys lose A2A | Med | Document in CHANGELOG, notify ailang-multivac to add `--a2a` to terraform |
| Prefix collisions (e.g., `/api/` vs `/api/foo`) | Low | Start with exact-match guard; prefix detection is a future enhancement |
| Users unaware their `@route` was skipped | Med | Warning log is visible in Cloud Run logs; could add startup summary |

## Related Documents

**Implemented (may inform design):**
- [design_docs/implemented/v0_9_0/m-protocol-support.md](design_docs/implemented/v0_9_0/m-protocol-support.md) — Original A2A/MCP implementation

**Planned (check for overlap):**
- [design_docs/planned/v0_9_5/m-dx-route-request-context.md](design_docs/planned/v0_9_5/m-dx-route-request-context.md) — Route handler DX improvements
- [design_docs/planned/v0_10_0/m-serve-api-get-args.md](design_docs/planned/v0_10_0/m-serve-api-get-args.md) — GET arg extraction for @route
- [design_docs/planned/v0_9_4/m-serve-api-dx.md](design_docs/planned/v0_9_4/m-serve-api-dx.md) — General serve-api DX

## References

- [Design Axioms](/docs/references/axioms)
- Go 1.22 ServeMux: panics on duplicate pattern registration (by design)
- Agent message `8fc78138` from ailang-multivac describing the production failure

## Future Work

- Route conflict detection at `ailang check` time (compile-time warning before deploy)
- `@override` annotation to explicitly replace a built-in route
- Structured route table introspection endpoint (`/api/_meta/routes`)

---

**Document created**: 2026-03-25
**Last updated**: 2026-03-25
