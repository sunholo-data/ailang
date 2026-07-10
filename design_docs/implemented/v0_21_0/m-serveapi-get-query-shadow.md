# M-SERVEAPI-GET-QUERY-SHADOW — GET query args silently shadowed by zero-value padding

**Status**: Planned
**Target**: v0.19.3 (patch) — too critical to wait for v0.21
**Priority**: P0 — production data corruption; every v0.19.2 GET handler with declared params is affected
**Estimated**: ~2h (~40 LOC fix + ~120 LOC tests + provenance plumbing + docs)
**Dependencies**: None (pure fix to `internal/apiserver/`)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Same inputs still produce same outputs; fix is deterministic |
| A2: Replayability | 0 | No trace impact |
| A3: Effect Legibility | 0 | No effect-row changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | **+1** | Arg-resolution now has a single explicit precedence; easier to reason about locally |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | **+1** | Handlers no longer silently receive empty strings for declared params; agents reading traces see real input or a real error, not zero-padded ghost values |
| A8: Minimal Syntax | 0 | No syntax surface |
| A9: Cost Visibility | 0 | Compile-time only |
| A10: Composability | 0 | No composition impact |
| A11: Structured Failure | **+2** | Direct application of "NO SILENT FALLBACKS." Today the padding silently shadows the query-arg fallback and the handler runs with `["",""]`. After this change, declared params are populated from the actual request or the request is rejected — never silently zero-padded when a real source was available. |
| A12: System Boundary | 0 | No boundary change |

**Net Score: +4** → **Decision: Proceed**

### Hard Violation Check

- [x] A1 (Determinism): Pure refactor of an existing decision tree
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access
- [x] A7 (Machines First): Eliminates a silent-failure mode that produces all-zero responses indistinguishable from a populated success

## Problem Statement

In v0.19.2, two well-intentioned safety nets in the serve-api request pipeline interact destructively:

1. **`parseArgsWithNames` (handler.go:248-262)** pads empty JSON bodies with type-zero-values to match declared arity, so handlers run their normal validation instead of crashing inside builtins on `UnitValue`. This was the M-SERVE-API-EMPTY-BODY-NAMED-ZEROPAD fix (msg_7425b152).
2. **`routes.go:438-441`** falls back to `parseQueryArgs(r.URL.Query())` when body args are empty — i.e., for GET requests that carry their inputs in the URL.

After fix (1) landed, the second safety net stopped firing for any handler with declared params. The padding produces `["", ""]` instead of `nil`/`[]`, so the `len(args) == 0` guard in (2) is false, and `?uid=X&period=Y` is silently discarded.

### Real production incident (May 2026)

`docparse` chased this for **five rounds of messaging** before locating it. Repro:

```bash
$ curl 'https://.../api/v1/debug/usage?args=smoke-test-claude&args=2026_04'
{"uid":"","period":"","getSubDoc_direct":{"status":"err","error":"Firestore GET 400: ...usage//periods/..."},"getUsage_wrapped":{"status":"ok","requests":0,"unitsCompleted":0}}
```

The handler `debugUsage(uid: string, period: string)` was invoked with `uid=""`, `period=""`. Firestore returned 400 (empty path segments). `usage_repo.getUsage` then swallowed the Err to `emptyUsage()` (the *separate* anti-pattern that M-CHECK-STRICT-FALLBACKS targets), so the dashboard rendered `0 requests / 1000` with no error signal.

**Blast radius:** every `@route("GET", "/...")` handler with declared params on v0.19.2 without a JSON body. Confirmed broken in production: `/billing/me/entitlements` (dashboard usage gauge), `/api/v1/debug/usage`. Pre-v0.12.0 binaries did NOT have this regression — the padding landed somewhere between v0.12.0 and v0.19.2.

**Bug history (immutable refs):** ailang messages `msg_e1814c9f` → `msg_597f3ae9` → `msg_952d6ef0` → `msg_9044f0cf` → `msg_3375cbf5` → `msg_8154269d` → `msg_79c86f8e` → `msg_e60442d7` → `msg_20260516_014642_dbf01230` (root-cause pinpoint).

## The class of bug: "padding shadows fallback"

This sprint is also about preventing the **class** of bug recurring, not just patching one symptom. The pattern:

- Function A produces a "safe default" value when it can't produce a real one.
- Function B has a fallback gated on "A produced nothing."
- B's guard checks the *shape* of A's output (`len(args) == 0`) rather than its *provenance* (was this from a real source or synthesized?).
- A's defensive defaults silently disable B.

This is exactly the silent-fallback class CLAUDE.md warns against, but at the **call-graph composition** layer, not the data layer. The strict-fallbacks check (M-CHECK-STRICT-FALLBACKS) catches `Ok(emptyDefault)` inside a single function; it does NOT catch "fn A pads, fn B's guard against emptiness now never fires."

### Why this keeps biting us

The serve-api dispatch path has grown three argument sources (raw body, multipart, JSON+named) and two fallback layers (zero-pad for arity, query-args for GET). Each fix landed in isolation, each correct in isolation, each tested against its own niche. There is no single function or test that surveys "given (method=GET, body=empty, query=present, paramTypes=non-empty), where do the args come from and is that the right choice?"

The 5-round bisect docparse ran is the cost of having no such survey.

### Prevention measures (chosen)

1. **Provenance, not shape**: `parseArgsWithNames` returns `(args, source)` where source ∈ `{Real, ZeroPadded, None}`. The fallback in routes.go checks `source != Real`, not `len(args) == 0`. Future padding fixes can't accidentally shadow future fallbacks.
2. **Single arg-resolution function**: `resolveArgs(r, opt) -> ([]interface{}, error)` becomes the one place where the precedence decision lives. Body-vs-multipart-vs-query-vs-pad lives here. Callers consume its result.
3. **Combinatorial regression test**: `TestArgResolution_AllSources` exercises the cross-product (method ∈ {GET, POST}) × (body ∈ {empty, JSON-object-matching, JSON-object-non-matching, JSON-array}) × (query ∈ {none, positional, named}) × (paramTypes ∈ {empty, single, multi}). One test, ~20 cases, fails loudly if any future fallback shadows another.
4. **Debug-trace breadcrumb**: when zero-padding fires, emit a `[apiserver] zero-padded args for %s/%s (no body, no query)` log at INFO. Makes the silent path *not* silent for operators.

(3) is the meta-prevention — it's the test that would have caught this bug at landing time.

## Goals

**Primary Goal:** Restore correct GET-with-query-args behavior on v0.19.2 for handlers with declared params, and prevent the bug class (padding shadows fallback) from recurring via a combinatorial regression test.

**Success Metrics:**
- `curl 'host/api/v1/foo?args=X&args=Y'` to `foo(a: string, b: string)` calls the handler with `("X", "Y")`. Today it calls `("", "")`.
- `curl 'host/api/v1/foo?a=X&b=Y'` to `foo(req: {a: string, b: string})` calls the handler with `({a: "X", b: "Y"})`. Today it calls `(zero-record)`.
- POST `{}` to `foo(a: string, b: string)` still calls handler with `("", "")` (preserves the M-SERVE-API-EMPTY-BODY-NAMED-ZEROPAD fix).
- POST `{"a": "X"}` to `foo(a: string, b: string)` still calls handler with `("X", "")` (partial body match preserved).
- `TestArgResolution_AllSources` table-driven test exercises the cross-product of {method, body, query, paramTypes} and would have failed on v0.19.2 with the bug present.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Surface provenance (`ArgSource` enum) vs in-place "is this padded?" check | Provenance makes the precedence rule explicit and survives future additions; in-place check is shorter but the next padding-style fix re-introduces the same shadowing class | human (this doc) | design | medium (API of internal helper) |
| Refactor to single `resolveArgs` vs surgical fix in routes.go | Single function consolidates the precedence rule. Surgical fix is faster but leaves the multi-layer fallback architecture that caused the bug | human (this doc) | design | medium |
| Patch into v0.19.3 vs roll into v0.21 | Production incident; every GET handler with declared params is broken. v0.19.3 must ship. | human (this doc) | release | low |

### Design Freeze

- [x] **Provenance enum: `ArgSource { Real, ZeroPadded, None }`** returned from `parseArgsWithNames` and `parseMultipartArgsWithNames`. Real = parsed from body/multipart fields that matched a declared param. ZeroPadded = synthesized to satisfy arity. None = no args resolved.
- [x] **Single arg-resolution function: `resolveArgs(r, opt) ([]interface{}, error)`** in `internal/apiserver/argresolve.go`. Implements the precedence: raw → multipart → body-with-names → query (when body source != Real) → zero-pad. Callers (`routes.go`) consume only its result.
- [x] **Ship in v0.19.3.** Bug is too severe to wait. Doc moves from `planned/v0_21_0/` to `implemented/v0_19/` after merge.
- [x] **Debug breadcrumb at INFO when zero-pad fires** (not behind DEBUG_* flag) — silent-pad is the bug; making it visible to operators is part of the fix.

## Conflict Surface

This change touches the apiserver request-dispatch path. Per the design-doc-creator's required Conflict Surface analysis:

### What syntactic positions does this change extend?

None at the AILANG language level. Pure Go-side refactor of `internal/apiserver/`. The annotation surface (`@route`, `@raw`, `@nowrap`, etc.) is unchanged.

### What OTHER valid constructs already live in those positions?

- Existing route handlers — `@route("GET", ...)`, `@route("POST", ...)`, `@raw`, `@nowrap` — all dispatch through `handleFunctionCall` / `callFunction` in `routes.go`. The argument-resolution branch (raw vs multipart vs JSON) is the section being refactored.
- `parseArgs` (positional `{"args": [...]}` body) and `parseNamedArgs` (JSON-object → named binding) — both preserved unchanged; their outputs feed into the new `resolveArgs`.

### How does the dispatcher disambiguate?

- The order of consideration is: `@raw` → multipart Content-Type → JSON body. This change adds an explicit "if body source is not Real, try query" step *between* JSON body parsing and zero-padding. The branching structure is otherwise unchanged.

### Programs that MUST still work post-change

1. **POST with empty body to typed handler** — `parseArgsWithNames(nil, ...)` still returns `["", ""]` so existing user validation runs unchanged. `resolveArgs` returns that result with `source = ZeroPadded`; if query is also empty (typical for POST), the zero-padded args flow through to the handler.
2. **POST `{}` with non-matching keys** — `parseArgsWithNames(body={}, ...)` returns zero-padded. Query is typically empty; zero-padded args reach handler. Existing behavior preserved.
3. **POST `{"a": "X"}` to `foo(a, b)`** — `parseArgsWithNames` returns `["X", ""]` with `source = Real` (some keys matched). Even if query has `?b=Y`, body wins because source is Real. Existing partial-body behavior preserved.
4. **`@raw` handlers** — unchanged. Bypass argument parsing entirely.
5. **Multipart handlers** — unchanged. `parseMultipartArgsWithNames` already populates from form fields; the new `resolveArgs` calls it identically.
6. **Existing `?args=X&args=Y` POST** (unusual but legal) — handler routed via POST with query also present. Today: body is empty, args is zero-padded, query is ignored, handler runs with zeros. Post-fix: source = ZeroPadded, query fallback runs, handler runs with query values. **This is a behavior change**, but it's the *intended* behavior — if the client provided real data in the query, use it. No production code relies on the buggy POST-with-query-but-empty-body case.

### What deliberately changes (intentional incompatibilities)

- v0.19.2 `GET /foo?args=X&args=Y` to handler with declared params returns zero-padded results. Post-fix it returns query-arg results. **This is the fix.**
- v0.19.2 `POST /foo?args=X&args=Y` with empty body to handler with declared params returns zero-padded results. Post-fix it returns query-arg results. (Edge case; no known callers.)

### Fixture tests (added in sprint)

- Positive: GET with positional `?args=X&args=Y` populates declared params from query.
- Positive: GET with named `?a=X&b=Y` to single-record-param handler populates the record.
- Negative: POST with empty body and no query still produces zero-padded args (preserves M-SERVE-API-EMPTY-BODY-NAMED-ZEROPAD).
- Negative: POST with partial body match (`{"a": "X"}`) and conflicting query (`?b=Y`) uses body, not query.

## Solution Design

### Overview

Two-part change:

1. **`internal/apiserver/argresolve.go` (new, ~80 LOC)** — single `resolveArgs(r, opt) ([]interface{}, error)` function that owns the precedence decision. Returns args and emits a debug log when zero-padding fires.

2. **`internal/apiserver/handler.go`** — `parseArgsWithNames` and `parseMultipartArgsWithNames` return `(args, source ArgSource, err)`. ZeroPadded vs Real is determined by whether any input key matched a declared param.

3. **`internal/apiserver/routes.go`** — JSON-body branch (lines 403-424) and query-fallback (lines 438-441) collapse into one call to `resolveArgs`.

### Precedence rule (single source of truth)

```
1. If @raw      → HttpRequest record  (source = Real)
2. If multipart → parseMultipartArgsWithNames (source = Real if any field matched, else ZeroPadded)
3. Else (JSON body):
   3a. parseArgsWithNames(body, paramNames, paramTypes)
       → (args, source)
   3b. If source != Real AND len(query) > 0:
       → args = parseQueryArgs(query); source = Real
   3c. If source == ZeroPadded (still): log INFO "[apiserver] zero-padded args for %s/%s (no body, no query)"
```

This makes the bug class structurally impossible: the only way `parseQueryArgs` doesn't run is if `source == Real`, i.e., the body had real data. Adding another padding-style fix later cannot regress this because the guard is provenance, not shape.

### `ArgSource` enum

```go
type ArgSource int

const (
    ArgSourceNone ArgSource = iota   // no parsing happened (e.g., raw)
    ArgSourceReal                     // at least one arg came from request data
    ArgSourceZeroPadded               // all args synthesized to satisfy arity
)
```

### Files to Modify/Create

**New:**
- `internal/apiserver/argresolve.go` — `resolveArgs` function (~80 LOC)
- `internal/apiserver/argresolve_test.go` — combinatorial `TestArgResolution_AllSources` table test (~150 LOC)

**Modified:**
- `internal/apiserver/handler.go` — `parseArgsWithNames` returns `(args, ArgSource, error)`; `parseNamedArgs` returns `(args, matchedAny bool)`; `parseMultipartArgsWithNames` returns same shape (~30 LOC delta)
- `internal/apiserver/routes.go` — JSON-body branch and query fallback collapse into `resolveArgs` call (~25 LOC delta)
- `internal/apiserver/named_args_test.go` — update existing tests to assert source returned (~20 LOC delta)
- `internal/apiserver/routes_test.go` — add HTTP-level regression test for the exact docparse repro (~40 LOC)
- `changelogs/v0.10-current.md` — `[Unreleased]` entry

### Stretch: telemetry counter

Optional follow-up (not in this sprint): expose `apiserver_zero_padded_calls_total{module,func}` so operators can spot handlers that frequently zero-pad — usually a sign of misuse or a real bug.

## Examples

### Example 1: The docparse repro (the production bug)

**Before (v0.19.2 — broken):**

```bash
$ curl 'host/api/v1/debug/usage?args=smoke-test-claude&args=2026_04'
{"uid":"","period":"",...}  ← uid and period silently empty
```

**After (v0.19.3 — fixed):**

```bash
$ curl 'host/api/v1/debug/usage?args=smoke-test-claude&args=2026_04'
{"uid":"smoke-test-claude","period":"2026_04",...}
```

No code change in the AILANG handler; only the dispatcher precedence changed.

### Example 2: POST with empty body still zero-pads (preserved)

```bash
$ curl -X POST 'host/api/v1/billing/me/entitlements'  # no body, no query
```

`resolveArgs` returns `("", "")` with `source = ZeroPadded`. Debug log fires: `[apiserver] zero-padded args for billing_service_api/entitlements (no body, no query)`. Handler runs with empty strings, hits its declared `uid == ""` validation branch, returns `Err("missing uid")`. Same as v0.19.2 (no behavior change).

### Example 3: POST partial body wins over query (preserved)

```bash
$ curl -X POST 'host/foo?b=Y' -d '{"a":"X"}'
```

`parseArgsWithNames` matches key `a` → `args = ["X", ""]`, `source = Real`. Query fallback skipped because source is Real. Handler runs with `("X", "")`. Same as v0.19.2 — partial body match preserved.

## Success Criteria

- [ ] `resolveArgs` implements the precedence rule with explicit `ArgSource` provenance
- [ ] `parseArgsWithNames` and `parseMultipartArgsWithNames` return `(args, ArgSource, error)`
- [ ] `routes.go` JSON-body branch + query fallback collapse into one `resolveArgs` call
- [ ] `TestArgResolution_AllSources` table test exercises ≥20 (method, body, query, paramTypes) combinations and fails on the v0.19.2 buggy behavior
- [ ] HTTP-level regression test reproduces the docparse curl exactly and asserts handler receives non-empty args
- [ ] `[apiserver] zero-padded args for %s/%s` INFO log fires when zero-pad is the final source
- [ ] All existing tests in `named_args_test.go` and `routes_test.go` pass with the new return signature
- [ ] `make ci` passes
- [ ] CHANGELOG entry under v0.19.3
- [ ] Doc moves from `planned/v0_21_0/` to `implemented/v0_19/` after merge

## Testing Strategy

**Unit tests** (`internal/apiserver/argresolve_test.go`):

Table-driven `TestArgResolution_AllSources`:

| method | body              | query              | paramTypes      | expected args                | expected source |
|--------|-------------------|--------------------|--------------------|------------------------------|-----------------|
| GET    | nil               | `?args=X&args=Y`   | [string, string]   | `["X", "Y"]`                 | Real            |
| GET    | nil               | `?a=X&b=Y`         | [record]           | `[{a:"X", b:"Y"}]`           | Real            |
| GET    | nil               | none               | [string, string]   | `["", ""]`                   | ZeroPadded      |
| POST   | nil               | none               | [string, string]   | `["", ""]`                   | ZeroPadded      |
| POST   | `{}`              | none               | [string, string]   | `["", ""]`                   | ZeroPadded      |
| POST   | `{"a":"X"}`       | none               | [string, string]   | `["X", ""]`                  | Real            |
| POST   | `{"a":"X"}`       | `?b=Y`             | [string, string]   | `["X", ""]` (body wins)      | Real            |
| POST   | `{"args":["X"]}`  | none               | [string]           | `["X"]`                      | Real            |
| POST   | nil               | `?args=X`          | [string]           | `["X"]` (query wins ZP)      | Real            |
| GET    | nil               | none               | []                 | `nil` / empty                | None            |

Plus ~10 more for edge cases (single-record-arg, JSON array body, etc).

**HTTP-level regression test** (`routes_test.go`):

```go
func TestServeAPI_GETQueryArgs_PopulateDeclaredParams(t *testing.T) {
    // Regression for docparse msg_20260516_014642_dbf01230:
    // v0.19.2 silently zero-padded declared params on GET requests,
    // discarding ?args=...&args=... entirely.
    // ...spin up server with a handler taking (uid: string, period: string)
    // ...issue GET /api/test/api/v1/debug/usage?args=smoke&args=2026_04
    // ...assert response includes uid="smoke", period="2026_04"
}
```

**Acceptance test:** the AILANG eval suite for serve-api handlers continues to pass post-change.

## Deferred Decisions

- **Telemetry counter for zero-padded calls.** A `apiserver_zero_padded_calls_total{module,func}` metric would let operators alert on handlers that pad too often (a smell). Punt to a separate small sprint.
- **Per-handler "strict" mode that errors on zero-pad** (`@strict_args` annotation). Would let critical handlers refuse to run with synthesized args. Promising follow-up; out of scope for v0.19.3 patch.
- **Replacing the `?args=X&args=Y` positional convention** with a typed body for all GETs. Bigger API discussion; out of scope.

## Non-Goals

- **Generic dispatcher framework.** This is a targeted refactor of one precedence decision, not a plugin system.
- **Catching `usage_repo.getUsage`'s `Err(_) => Ok(emptyUsage())` swallow** — that's M-CHECK-STRICT-FALLBACKS' job at the AILANG layer.
- **Renaming or removing the zero-padding behavior.** It's load-bearing for the empty-POST case; we preserve it.

## Timeline

**Total: ~2 hours**

- 0:00–0:20 — Add `ArgSource` enum + update `parseArgsWithNames` / `parseNamedArgs` / `parseMultipartArgsWithNames` signatures
- 0:20–0:40 — Write `argresolve.go::resolveArgs`
- 0:40–1:00 — Update `routes.go` JSON-body branch to call `resolveArgs`; remove old fallback
- 1:00–1:30 — Add `TestArgResolution_AllSources` table test (≥20 cases)
- 1:30–1:50 — Add HTTP-level regression test mirroring docparse repro
- 1:50–2:00 — Update existing tests, CHANGELOG, `make ci`

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Signature change to `parseArgsWithNames` breaks external callers (if any) | Low | All callers are inside `internal/apiserver/` — `grep` confirms one call site in routes.go and tests |
| `ArgSource = Real` granularity insufficient (e.g., "Real for 1 of 2 params") | Low | Treating "any match" as Real is the right call — partial body match means user provided data; we should respect it, not overwrite with query |
| Existing edge case (POST with query and empty body) shifts behavior | Low | Documented intentional change; no production callers known. CHANGELOG flags it. |
| Debug log spam in tests | Low | Log goes through `log.Printf` like existing apiserver logs; test runs already filter these |

## Related Documents

**Planned (related sprints):**
- [m-check-strict-fallbacks.md](m-check-strict-fallbacks.md) — Static check for `Ok(emptyDefault)` at the AILANG layer. **Complementary**: that doc catches the *downstream* swallow in `usage_repo`; this doc catches the *upstream* shadow in the dispatcher. The May 2026 incident required both layers to lie.
- [m-serveapi-surface-drops.md](m-serveapi-surface-drops.md) — same incident-driven family. Surface-drops handles loud failure at module-load time; this doc handles loud failure at request-dispatch time. Both encode "NO SILENT FALLBACKS."

**Bug history (motivation):**
- ailang messages thread: `msg_e1814c9f` → `msg_597f3ae9` → `msg_952d6ef0` → `msg_9044f0cf` → `msg_3375cbf5` → `msg_8154269d` → `msg_79c86f8e` → `msg_e60442d7` → `msg_20260516_014642_dbf01230`
- M-SERVE-API-EMPTY-BODY-NAMED-ZEROPAD (msg_7425b152) — the original padding fix this doc partially constrains

## References

- [Design Axioms A11](/docs/references/axioms#a11-structured-failure) — Structured Failure
- [CLAUDE.md "NO SILENT FALLBACKS"](../../../CLAUDE.md) — applied at the call-graph composition layer
- [.claude/rules/api-server.md](../../../.claude/rules/api-server.md) — apiserver discipline this sprint reinforces

## Future Work

- **`@strict_args` annotation** — let critical handlers refuse to run with `ArgSource == ZeroPadded`, fail the request with 400. Promising follow-up.
- **Zero-pad counter** — Prometheus-style metric. Operator-visible signal for "this handler pads a lot."
- **Argument-source visible in `/api/_meta/trace`** — surface `source: zero_padded` in request traces so post-hoc debugging doesn't require 5 rounds of bisecting.

---

**Document created**: 2026-05-16
**Last updated**: 2026-05-16

DESIGN_DOC_PATH: design_docs/planned/v0_21_0/m-serveapi-get-query-shadow.md
