# M-ROUTE-NOWRAP: @nowrap annotation for unwrapped @route responses

**Status**: Planned
**Target**: v0.9.5
**Priority**: P1 (High — blocks A2A protocol compliance for custom agent cards)
**Estimated**: 1-2 hours
**Dependencies**: M-ROUTE-COLLISION (merged)
**Milestone ID**: M-ROUTE-NOWRAP
**Created**: 2026-03-26
**Source**: docparse agent message `5e45fe0a`

---

## Axiom Compliance

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to evaluation semantics |
| A2: Replayability | 0 | No change to traces |
| A3: Effect Legibility | +1 | Response format now declared at function definition site |
| A4: Explicit Authority | 0 | No change |
| A5: Bounded Verification | 0 | No change |
| A6: Safe Concurrency | 0 | No change |
| A7: Machines First | +1 | A2A clients get protocol-compliant raw JSON without envelope noise |
| A8: Minimal Syntax | 0 | Reuses existing annotation mechanism, no new syntax |
| A9: Cost Visibility | 0 | No change |
| A10: Composability | +1 | @nowrap composes with @route and @raw independently |
| A11: Structured Failure | 0 | Errors still returned as JSON with status codes |
| A12: System Boundary | 0 | No change |

**Net Score: +3** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

## Problem Statement

`callFunction()` always wraps results in `FunctionCallResponse{result, module, func, elapsed_ms}`. This is correct for the standard `/api/` catch-all, but breaks protocol compliance when custom `@route` handlers need to return raw JSON — e.g., an A2A Agent Card at `/.well-known/agent.json` or an OpenID Connect discovery document.

**Current State:**
- `@raw` annotation controls the *input* side (passes `HttpRequest` record instead of parsed args)
- The `_body` field convention controls the *output* side (raw HTTP response with custom headers/status)
- But there's no lightweight way to say "return my result as-is, without the envelope"
- The `_body` pattern requires wrapping the return in `{_body: ..., _status: 200, _headers: {...}}` — verbose for simple JSON responses

**Impact:**
- A2A protocol compliance: agent cards must be raw JSON, not envelope-wrapped
- Any REST-style API where the consumer expects a specific schema (webhooks, OAuth, health endpoints)

## Goals

**Primary Goal:** Add `@nowrap` annotation that tells the apiserver to write the function's return value directly as JSON, skipping the `FunctionCallResponse` envelope.

**Success Metrics:**
- `@nowrap @route("GET", "/path")` returns raw JSON response
- `@nowrap` composes with `@raw` (raw input + raw output)
- Existing @route behavior unchanged when @nowrap absent
- Built-in A2A handler unaffected

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Annotation name `@nowrap` | Must be intuitive and not clash | agent | design | low |
| @nowrap affects output only, not input | Clear separation of concerns with @raw | agent | design | low |

### Design Freeze

- [x] Annotation name: `@nowrap`
- [x] @nowrap is output-only (does not change arg parsing)

## Solution Design

### Overview

Add a `@nowrap` annotation recognized by the apiserver. When present on a `@route` function, `callFunction()` skips the `FunctionCallResponse` envelope and writes the Go-converted result directly as JSON with `Content-Type: application/json`.

### Architecture

**1. Parse @nowrap in extractRouteAnnotations** — already handles @raw, add @nowrap the same way.

**2. Carry `IsNowrap` through RouteEntry and ExportInfo** — mirror the `IsRaw` pattern.

**3. Pass to callFunction and skip envelope** — add a `nowrap` parameter (or encode in the existing `isRaw` variadic, or add a new variadic). When nowrap is set, after getting the result, convert to Go and write directly instead of wrapping in `FunctionCallResponse`.

### Implementation Plan

**Phase 1: Wire @nowrap through** (~30 min)
- [ ] Add `IsNowrap bool` to `ExportInfo` and `RouteEntry`
- [ ] Detect `@nowrap` in `extractRouteAnnotations()`
- [ ] Pass `IsNowrap` from `registerCustomRoutes()` to handler

**Phase 2: Skip envelope in callFunction** (~30 min)
- [ ] Add `nowrap` option to `callFunction` (new bool param or options struct)
- [ ] When nowrap: convert result to Go, write as raw JSON (no FunctionCallResponse)
- [ ] Error responses still use JSON with appropriate status codes

**Phase 3: Tests** (~30 min)
- [ ] Test: @nowrap route returns raw JSON (no envelope)
- [ ] Test: @nowrap + @raw composes correctly
- [ ] Test: non-@nowrap route still returns envelope
- [ ] Update CHANGELOG

### Files to Modify

- `internal/apiserver/routes.go` — Add IsNowrap to structs, detect annotation, skip envelope (~+25 LOC)
- `internal/apiserver/routes_test.go` — Tests for @nowrap (~+40 LOC)
- `internal/apiserver/protocol_test.go` — Optional: verify A2A + nowrap interaction
- `changelogs/v0.9-current.md` — Changelog entry

## Examples

### Example 1: Custom A2A Agent Card

```ailang
@nowrap
@route("GET", "/.well-known/agent.json")
export func agentCard() -> {name: string, skills: List[string]} ! {IO} =
  {name: "DocParse", skills: ["parse", "extract"]}
```

**Response (raw, no envelope):**
```json
{"name": "DocParse", "skills": ["parse", "extract"]}
```

### Example 2: Without @nowrap (default, unchanged)

```ailang
@route("GET", "/api/v1/status")
export func status() -> string ! {IO} = "ok"
```

**Response (envelope-wrapped):**
```json
{"result": "ok", "module": "myapp/api", "func": "status", "elapsed_ms": 1}
```

### Example 3: @nowrap + @raw (full control)

```ailang
@nowrap
@raw
@route("POST", "/webhooks/stripe")
export func handle(req: HttpRequest) -> {received: bool} ! {IO} =
  {received: true}
```

## Success Criteria

- [ ] @nowrap @route returns raw JSON (no FunctionCallResponse envelope)
- [ ] @nowrap composes with @raw
- [ ] Default @route behavior unchanged
- [ ] All existing tests pass
- [ ] CHANGELOG updated

## Non-Goals

- Changing the built-in A2A handler (already returns raw JSON)
- Custom HTTP status codes from @nowrap (use `_body`/`_status` pattern for that)
- Streaming responses

## Related Documents

- [design_docs/planned/v0_9_5/m-route-collision-guard.md](design_docs/planned/v0_9_5/m-route-collision-guard.md) — Route collision guard (prerequisite)
- [design_docs/planned/v0_9_5/m-dx-route-request-context.md](design_docs/planned/v0_9_5/m-dx-route-request-context.md) — @raw annotation design

---

**Document created**: 2026-03-26
**Last updated**: 2026-03-26
