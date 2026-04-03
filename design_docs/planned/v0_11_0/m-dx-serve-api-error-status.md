# M-DX-SERVE-API-ERROR-STATUS: Map Result.Err to proper HTTP status codes

**Status**: Planned
**Target**: v0.11.0
**Priority**: P1 (blocks correct HTTP semantics for all serve-api deployments)
**Estimated**: 1 day
**Dependencies**: None
**Milestone ID**: M-DX-SERVE-API-ERROR-STATUS
**Created**: 2026-04-03

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Same Err value always maps to same HTTP status — deterministic |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | +1 | HTTP status now reflects function outcome — previously hidden behind 200 |
| A4: Explicit Authority | 0 | No authority changes |
| A5: Bounded Verification | +1 | Clients can verify success/failure from HTTP status alone |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Load balancers, monitoring, and AI agents can distinguish errors from HTTP status without parsing body |
| A8: Minimal Syntax | +1 | No new syntax — uses existing record fields and annotations |
| A9: Cost Visibility | 0 | No cost changes |
| A10: Composability | +1 | Composes with existing @raw _status pattern |
| A11: Structured Failure | +1 | Errors are now structured at both AILANG (Result type) AND HTTP (status code) levels |
| A12: System Boundary | +1 | HTTP boundary now correctly communicates error state |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): Deterministic mapping from Result variant to HTTP status
- [x] A3 (Effects): No hidden side effects — makes existing error state MORE visible
- [x] A4 (Authority): No ambient access
- [x] A7 (Machines First): Primary motivation is machine-readable error signals

## Problem Statement

`serve-api` returns HTTP 200 for ALL successful function calls, even when the AILANG function returns `Err(...)` via the `Result` type. This makes error responses indistinguishable from success at the HTTP transport level.

**Current State:**
- `callFunction` in `routes.go:440` returns `http.StatusOK` whenever `engine.Call` succeeds (no Go error)
- An AILANG function returning `Err("invalid input")` produces: `HTTP 200 {"result": {"__tag": "Err", "__fields": ["invalid input"]}, ...}`
- The `@nowrap` path at `routes.go:422` always writes `http.StatusOK`
- Only Go-level call failures (panic, wrong arity) return non-200 status codes
- The `@raw` path already supports `_status` via record fields — but this only works for `RecordValue` responses, not `Result` types

**Impact:**
- **Load balancers / health checks**: Cannot distinguish healthy vs erroring endpoints
- **Monitoring / alerting**: HTTP 200 error rate is invisible to standard monitoring
- **Client SDKs**: Must parse JSON body to detect errors instead of checking status code
- **AI agents**: Cannot use HTTP status to decide retry/fallback behavior
- **REST convention violation**: Every major framework (Express, FastAPI, Go stdlib, Rails) maps application errors to non-200 status codes

## Goals

**Primary Goal:** Map AILANG `Result.Err` returns to appropriate HTTP error status codes, with support for custom status codes.

**Success Metrics:**
- `Err("message")` returns HTTP 400 (default) instead of 200
- `Err({_status: 404, message: "not found"})` returns HTTP 404 with error body
- `Ok(value)` continues to return HTTP 200 (no change)
- Non-Result return types are unaffected (HTTP 200 as before)
- Works with both wrapped and `@nowrap` response paths

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Default HTTP status for `Err(string)` | Sets convention for all serve-api users | human | design | med |
| Custom status mechanism (record field vs annotation) | Determines DX for error responses | human | design | med |
| Whether `@nowrap` Err responses include envelope or raw error | Affects client parsing for nowrap users | agent | compile | low |
| Whether non-Result error-like values (e.g., `None`) get special treatment | Scope creep risk | human | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Default status code for `Err(string)` — **400 Bad Request** (see rationale below)
- [x] Custom status via `_status` field in Err record payload — **yes** (reuses @raw pattern)
- [x] Only `Result.Err` gets special treatment, not `Option.None` or other ADTs — **confirmed**

## Solution Design

### Overview

Intercept `Result.Err` variants in `callFunction` (routes.go) after the function call succeeds but before writing the HTTP response. Extract the error payload, determine the HTTP status code (default 400, or custom via `_status` field), and write the response with the correct status.

### Architecture

The fix is localized to `internal/apiserver/routes.go` in the `callFunction` method. No changes to the evaluator, type system, or AILANG language semantics.

**Status Code Resolution Order:**
1. If Err payload is a record with `_status: int` field → use that status code
2. If Err payload is anything else → use 400 (Bad Request)
3. Ok payload → 200 (unchanged)
4. Non-Result return types → 200 (unchanged)

**Why 400 as default (not 422 or 500):**

| Code | Meaning | Fit for Err |
|------|---------|-------------|
| 400 Bad Request | Client sent something wrong | Best default — most Err returns are validation/input errors |
| 422 Unprocessable Entity | Semantically invalid | Too specific — implies the request was syntactically valid |
| 500 Internal Server Error | Server bug | Wrong signal — Err is intentional, not a crash |

Frameworks comparison:
- **FastAPI**: Returns 422 for validation errors (Pydantic), but AILANG Err is broader than validation
- **Express.js**: No automatic mapping — developers set status explicitly
- **Go stdlib**: No automatic mapping — developers call `http.Error(w, msg, code)`
- **Rails**: `render json: ..., status: :bad_request` — developers choose explicitly

400 is the safest default because:
- It signals "this request didn't produce a successful result" without implying server fault
- It's the most commonly used client error code
- Custom `_status` lets users override for specific cases (404, 409, 422, 500, etc.)

### Implementation Plan

**Phase 1: Core Logic** (~2 hours)
- [ ] Add `isResultErr` helper: checks if `eval.Value` is `*eval.TaggedValue` with `CtorName == "Err"`
- [ ] Add `errStatusCode` helper: extracts `_status` from Err payload record, defaults to 400
- [ ] Modify `callFunction` in routes.go: after successful `engine.Call`, check result before `writeJSON`
- [ ] Handle both wrapped and `@nowrap` response paths

**Phase 2: Response Format** (~1 hour)
- [ ] Wrapped path: `FunctionCallResponse` with appropriate status code and error field populated
- [ ] `@nowrap` path: write Err payload directly with appropriate status code
- [ ] Ensure `@raw` path (writeRawResponse) is unaffected (already has _status support)

**Phase 3: Tests & Docs** (~1 hour)
- [ ] Unit test: Err(string) → 400
- [ ] Unit test: Err({_status: 404, message: "not found"}) → 404
- [ ] Unit test: Err({_status: 503, message: "upstream down"}) → 503
- [ ] Unit test: Ok(value) → 200 (regression)
- [ ] Unit test: non-Result return → 200 (regression)
- [ ] Unit test: @nowrap + Err → 400
- [ ] Update serve-api docs

### Files to Modify/Create

**Modified files:**
- `internal/apiserver/routes.go` — Add Result.Err detection + status code extraction (~30 LOC)

**New files:**
- `internal/apiserver/result_status_test.go` — Tests for Result → HTTP status mapping (~100 LOC)

## Examples

### Example 1: Simple Err(string) → 400

**AILANG function:**
```ailang
@route("GET", "/users/:id")
export func getUser(id: string) -> Result[string, string] ! {Net, FS} =
  match findUser(id) with
  | Some(user) -> Ok(encode(userToJson(user)))
  | None -> Err("user not found")
```

**Before (current):**
```
HTTP/1.1 200 OK
{"result": {"__type": "Result", "__tag": "Err", "__fields": ["user not found"]}, "module": "api", "func": "getUser", "elapsed_ms": 3}
```

**After:**
```
HTTP/1.1 400 Bad Request
{"error": "user not found", "module": "api", "func": "getUser", "elapsed_ms": 3}
```

### Example 2: Custom Status with _status field → 404

**AILANG function:**
```ailang
@route("GET", "/users/:id")
export func getUser(id: string) -> Result[string, {_status: int, message: string}] ! {Net, FS} =
  match findUser(id) with
  | Some(user) -> Ok(encode(userToJson(user)))
  | None -> Err({_status: 404, message: "user not found"})
```

**Response:**
```
HTTP/1.1 404 Not Found
{"error": {"message": "user not found"}, "module": "api", "func": "getUser", "elapsed_ms": 3}
```

Note: `_status` is extracted for HTTP status and stripped from the JSON body (same convention as `@raw`).

### Example 3: @nowrap + Err

**AILANG function:**
```ailang
@route("POST", "/payments")
@nowrap
export func processPayment(amount: float) -> Result[string, string] ! {Net} =
  if amount <= 0.0
    then Err("amount must be positive")
    else Ok(encode(jo([kv("status", js("paid"))])))
```

**Response:**
```
HTTP/1.1 400 Bad Request
"amount must be positive"
```

### Example 4: Ok(value) — unchanged

```ailang
export func hello(name: string) -> Result[string, string] =
  Ok("Hello, " ++ name)
```

**Response (unchanged):**
```
HTTP/1.1 200 OK
{"result": "Hello, Mark", "module": "greet", "func": "hello", "elapsed_ms": 1}
```

## Success Criteria

- [ ] `Err("msg")` returns HTTP 400 with error in response body
- [ ] `Err({_status: N, ...})` returns HTTP N with remaining fields in body
- [ ] `Ok(value)` returns HTTP 200 (no regression)
- [ ] Non-Result types return HTTP 200 (no regression)
- [ ] `@nowrap` path respects Err status codes
- [ ] `@raw` path unaffected (has its own _status mechanism)
- [ ] All existing serve-api tests passing
- [ ] Documentation updated (serve-api guide)

## Testing Strategy

**Unit tests:**
- `result_status_test.go`: Test `isResultErr` and `errStatusCode` helpers with various payloads
- Test matrix: Err(string), Err(int), Err(record with _status), Err(record without _status), Ok(value), non-Result

**Integration tests:**
- Extend existing serve-api test suite with Result-returning functions
- Test both wrapped and @nowrap paths

**Manual testing:**
- `curl` against serve-api with Result-returning endpoints
- Verify monitoring tools see non-200 responses

## Deferred Decisions

- Whether `_status` field is stripped from response body or kept — agent may choose (recommend strip, matching @raw convention)
- Exact error response format for `@nowrap` Err — agent may choose
- Whether to log Result.Err differently than Ok — agent may choose

## Non-Goals

- **Mapping `Option.None` to 404** — Tempting but scope creep; None is not an error, it's absence. Users who want 404 should use `Result` with `Err({_status: 404, ...})`
- **`@status` annotation on functions** — Over-engineering for this use case; `_status` in Err payload is more flexible and composable
- **Changing the Result type itself** — This is a serve-api transport concern, not a language concern
- **Custom status codes for Ok variants** — Use `@raw` with `_status` for this (already works)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Breaking existing clients that check for 200 + parse error from body | Med | Document in CHANGELOG as breaking change; 200→400 for errors is the correct fix |
| Err payload type varies (string, int, record) — extraction logic gets complex | Low | Keep it simple: check for record with _status, otherwise default 400 |
| @nowrap Err might confuse clients expecting raw success format | Low | Set Content-Type to application/json and include error field |

## Related Documents

**Implemented (informs design):**
- [m-dx-serve-api-coercion.md](../../../implemented/v0_11_0/m-dx-serve-api-coercion.md) — Type coercion in serve-api cross-package calls
- [m-serve-api-get-args.md](../../../implemented/v0_10_0/m-serve-api-get-args.md) — GET arg parsing, query params

**Planned (check for overlap):**
- [m-dx-serve-api-coercion.md](m-dx-serve-api-coercion.md) — Cross-package int/float coercion (separate bug, same area)
- [m-arch5-error-handling-strategy.md](m-arch5-error-handling-strategy.md) — Broader error handling strategy (complementary, not conflicting)

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [serve-api guide](docs/docs/guides/serve-api.md) - Current serve-api documentation
- [RFC 9110 - HTTP Semantics](https://httpwg.org/specs/rfc9110.html#status.codes) - HTTP status code definitions

## Future Work

- `@status(default: 422)` annotation to override default error status per-function
- Structured error responses with error codes (e.g., `{code: "USER_NOT_FOUND", message: "..."}`)
- OpenAPI spec generation that reflects error status codes in responses section
- Middleware hooks for custom error formatting

---

**Document created**: 2026-04-03
**Last updated**: 2026-04-03
