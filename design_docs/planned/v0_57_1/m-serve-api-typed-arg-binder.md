# M-SERVE-API-TYPED-ARG-BINDER: One Declared-Type Argument Binder for MCP, A2A, and REST

**Status**: Planned
**Target**: v0.57.1
**Priority**: P0 (High — every AILANG MCP service with an `int` param is broken today)
**Estimated**: 2 days (~5h binder + 4h surface integration + 4h tests + 1h docs, 2x'd from a 4.5h initial guess)
**Dependencies**: Builds on M-SERVE-API-ZERO-VALUE-PADDING (implemented v0.9.5) — reuses its `ExportInfo.ParamTypes`/`ParamZeros` plumbing and its `param_zero.go` traversal
**Milestone ID**: M-SERVE-API-TYPED-ARG-BINDER
**Created**: 2026-11-03
**Source**: CLI feedback via `ailang messages` (task-d8420071); triage note [`design_docs/planned/ailang-core-triage/serveapi-mcp-int-args-bind-float64.md`](../ailang-core-triage/serveapi-mcp-int-args-bind-float64.md)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Binding is a pure function of (declared param types, JSON input); no nondeterminism added or removed |
| A2: Replayability | +1 | Identical JSON now produces identical typed bindings on every surface — previously the surface chosen (MCP vs REST) changed the value type |
| A3: Effect Legibility | 0 | No effect-row changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Argument conformance is checked against the declared signature at the boundary, before the engine call — a local, decidable check |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Machine callers (MCP clients, A2A agents) get the *declared* type contract honoured plus a structured, param-named rejection instead of an opaque evaluator dispatch error |
| A8: Minimal Syntax | 0 | No language syntax changes; serve-api runtime only |
| A9: Cost Visibility | 0 | No resource accounting changes |
| A10: Composability | +1 | One binder composes with named binding, `@optional` zeros, `@mcp_secret`, `@mcp_file`, `_headers` injection, `@raw`, multipart — each surface keeps its own resolution step and shares the ONE conversion step |
| A11: Structured Failure | +1 | "2.5 for an int param" becomes a structured invalid-argument error naming the parameter, instead of a 500 / in-band evaluator dispatch string |
| A12: System Boundary | +1 | The JSON↔AILANG boundary translation becomes typed by the declared interface — the boundary stops guessing (whole-number heuristic) or refusing (preserve-floats) |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced — binder is a pure mapping
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): The fix is *for* machine callers; no human convenience traded

---

## Problem Statement

serve-api binds JSON arguments to AILANG function parameters through **two different code paths that disagree about numeric types**, and each path is broken in the other's direction. Go's `encoding/json` decodes every JSON number into `float64`; what happens next depends on which surface received the request:

- **MCP tools/call** (`internal/apiserver/mcp.go:456`), **A2A** (`internal/apiserver/a2a.go:226`), and **MCP UI serve** (`internal/apiserver/mcp_ui_serve.go:103`) call `engine.CallPreserveFloats`, which keeps `float64` as `FloatValue` "even for whole numbers". A param declared `int` therefore receives a `FloatValue` and the call fails.
- **REST routes** (`internal/apiserver/routes_dispatch.go:146`) call `engine.Call`, whose `FromGo` converts any whole-number `float64` to `IntValue`. An `int` param works — but a `float` param receiving an integral JSON number binds `IntValue` and fails.

**Current state, live-verified on this worktree** (ailang v0.57.0, commit `b23cadc`, full transcripts in the Verification Log):

| Case | MCP tools/call | REST POST |
|---|---|---|
| `int` param, JSON `21` | ❌ `function call failed: numeric dictionary method "mul" received an int and a float…` | ✅ `{"result":42}` |
| `float` param, JSON `2` (whole) | ✅ `2` | ❌ same `mul`/`div` dispatch error |
| `int` record field, JSON `5` | ❌ same failure | ✅ `6` |
| `float` record field, JSON `2` (whole) | ✅ `6` | ❌ same failure |
| `@optional int` param, omitted | ❌ binds `FloatValue(0)` → same failure | ✅ `IntValue(0)` |
| `int` param, JSON `2.5` (non-integral) | ❌ opaque runtime failure | ❌ HTTP 500 with opaque error |

**Impact:**
- **Every AILANG MCP service with an `int` parameter is unusable.** MCP is the primary machine-calling surface (docparse's `createUpload` had to declare `count` as `float` as a workaround — docparse PR #229).
- The REST mirror bug (`float` param + integral JSON) is unreported but live: any client sending `{"x": 2}` to `func half(x: float)` gets a 500.
- The two paths have already diverged once in this exact family (commit 39dd37990 / M-SERVE-API-ZERO-VALUE-PADDING: omitted record params got `{}` on REST). Each surface fixing the numeric half it happens to hit is the incremental-special-casing anti-pattern; this doc is the unified fix.

**Reporter's mechanism claim, corrected:** the report says "the REST binder converts per declared type". It does not — REST uses the whole-number *heuristic* inside `FromGo` (`internal/embed/convert.go`, `f == float64(int(f))` check, regression-tested as `TestFromGoWholeFloatBecomesInt`). That is why REST fails the float-param case. No surface binds per declared type today; that is precisely what this feature adds.

---

## Goals

**Primary Goal:** ONE shared argument binder, typed by the declared parameter types, used by MCP tools/call, A2A, MCP UI serve, and REST, so that identical JSON binds identically on every surface and type mismatches are rejected as structured invalid-argument errors before the engine call.

**Success Metrics:**
- All 6 rows of the matrix above pass on BOTH surfaces (12 acceptance cases, tested on each surface).
- Non-integral number for an `int` param → structured, param-named invalid-argument error on every surface (not an evaluator dispatch error / 500).
- `@optional int` omitted on MCP binds `IntValue(0)` (currently binds `FloatValue(0)` and fails).
- Exactly ONE code path converts resolved JSON args to AILANG values in `internal/apiserver` (grep: no `CallPreserveFloats` callers left in the package).
- MCP test coverage gains its first *successful* `int` tools/call (today no test in the repo performs one — see Verification Log V8).

---

## Systemic Audit

**Is this a one-off or part of a pattern?** Pattern — the third divergence in the same family:

1. REST switched from `CallPreserveFloats` to `Call` (routes_dispatch.go comment, M-SERVE-API-ZERO-VALUE-PADDING era) to fix int-typed record fields — leaving MCP/A2A on the broken path.
2. MCP grew presence-checks and typed record zeros (M-MCP-UNIT-PARAM-BINDING, `param_zero.go`) — REST kept its own padding in `parseNamedArgs`/`resolveArgs`.
3. This report: MCP int params fail; REST float params fail. Same root, opposite directions.

**Similar code paths audited (all `CallPreserveFloats`/`Call` call sites in `internal/apiserver`):**

| Call site | Binds JSON numbers? | Affected |
|---|---|---|
| `mcp.go:456` tools/call | Yes (named + positional) | **Yes** |
| `a2a.go:226` message/task send | Yes (`args` data/text parts) | **Yes** |
| `routes_dispatch.go:146` REST routes | Yes (body, query, zeros) | **Yes** (mirror bug) |
| `mcp_ui_serve.go:103` UI resource | No (nils/zeros only) | No numeric bug, but must adopt the binder so its zeros are the shared ones |
| `mcp_auth.go:207` token verifier | No (one string param) | No |
| `routes_ws.go:219` WS routes | No (StreamConn + req record, structured values) | Out of scope |

`internal/embed`'s `CallPreserveFloats`/`FromGoPreserveFloats` remain for **external Go embedders** (their public API contract is out of scope); only `internal/apiserver` stops using the float-preserving variant.

**What already exists (do NOT re-implement):**
- `ExportInfo.ParamNames` / `ParamTypes` / `ParamZeros` extraction from the AST (`internal/apiserver/routes.go` `extractParamInfo`, `paramTypeToString`).
- `param_zero.go`: declared-record zero extraction (`extractParamZeros`, `zeroFromAST`, `maxZeroDepth` alias-recursion bound) — already produces *typed* zeros for record fields (`int(0)`, `&eval.FloatValue{0}`, `""`, `false`).
- MCP named-param presence checks, `@optional` zero substitution, `@mcp_secret` zero-binding, `@mcp_file` object binding, `_headers` injection (mcp.go makeToolHandler).
- REST `resolveArgs` body→query→zero precedence and `parseNamedArgs` snake_case matching.
- `FromGo`'s identity path: a value that is already an `eval.Value` passes through unconverted (`internal/embed/convert.go:37-41`) — this is the seam the binder uses to hand pre-typed values to `engine.Call`.

**What's genuinely new:**
1. A recursive per-param **declared type schema** (today `ParamTypes` collapses records to `"record"` and lists to `"list"` — not enough to type record fields or list elements).
2. The binder itself: JSON `any` → `eval.Value`, per declared type, with structured rejection.
3. Typed scalar zeros (`zeroValueForType("int")` currently returns `float64(0)` — handler.go:180 — which is exactly why an omitted `@optional int` binds `FloatValue(0)` on MCP).

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **Binder lives in `internal/apiserver`**, not inside `internal/embed` conversion | The engine never sees declared param types (they live in `ExportInfo`, extracted from the AST at serve-api load); pushing them into `embed` would couple the generic Go↔AILANG converter to serve-api's interface model and change `FromGo` semantics for every embedder | human | design | high |
| **Binder emits `eval.Value` directly** (per declared type); all four apiserver call sites switch to `engine.Call`; `FromGo`'s whole-number heuristic is bypassed via its existing eval.Value identity path | This is what makes float-with-whole-JSON correct (a Go `float64(2)` emitted for a float param would be re-converted to `IntValue` by `FromGo` and re-break); also lets `CallPreserveFloats` retire from apiserver | human | design | med |
| **Non-integral number for an `int` param → structured invalid-argument error** (param-named, rejected before the engine call) | Public-surface behavior change: today it is an opaque evaluator error (HTTP 500 / in-band MCP error); tomorrow a client-error-class rejection. Clients that relied on the 500 have none to rely on | human | design | low |
| **Strict scalar typing on `bool`/`string` too** (a JSON number for a `string` param is rejected, not silently bound) | Today that binds `IntValue`/`FloatValue` into a string param and crashes deep in stdlib; rejecting is the same class of change as the int rule and leaving it out would re-open the "binder is a special case" pattern | human | design | med |
| **Record binding types declared fields only; fields without declared types (row-polymorphic/open records, `unknown` param types) fall back to the `FromGo` heuristic** | Full strictness for open rows would reject valid programs that pass extra fields; partial strictness keeps the fix total for the *declared* contract without inventing a closed-record rule the language doesn't have | agent (per this doc's rule) | design | low |
| **Typed zeros**: `zeroValueForType` and `paramZero` emit `eval.Value` zeros (`IntValue(0)` for int) instead of `float64(0)`/Go values | Fixes the omitted-`@optional`-int case on MCP; zeroes are already half-typed for record fields (param_zero.go does this) — this finishes the job for scalars | agent | implementation | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Binder in `internal/apiserver` (not `internal/embed`) — ratified by this doc's approval
- [x] Binder output is `eval.Value`, consumed via `engine.Call`'s identity path; `CallPreserveFloats` leaves apiserver
- [x] Non-integral-for-int and scalar type mismatches are structured invalid-argument errors (public behavior change, enumerated under "What deliberately changes")
- [x] Open records / unknown types keep the `FromGo` heuristic (no closed-record strictness)
- [ ] Human approval of this design doc (the pipeline gate this document exists to satisfy)

---

## Solution Design

### Overview

Add ONE function — `bindCallArgs` — that converts resolved JSON-decoded arguments into AILANG values typed by the *declared* parameter types, plus the per-param declared type schema that makes record fields and list elements typeable. Every surface keeps its own argument *resolution* (which JSON key maps to which param, presence checks, secrets, file objects, headers, query fallback) and hands the resolved `[]any` to the shared binder, which is also the single place that types omitted-param zeros. All four call sites then go through `engine.Call` (whose `FromGo` already passes `eval.Value` through unconverted).

### Architecture

**Components:**

1. **`paramSchema`** (new, `internal/apiserver/param_schema.go, ~80 LOC`): a minimal recursive type descriptor extracted from the AST at registration —
   `type paramSchema struct { kind string; fields []struct{name string; schema *paramSchema}; elem *paramSchema }`
   with kinds `int, float, bool, string, record, list, array, unknown`. Extracted in `extractParamInfo` (routes.go) alongside `ParamTypes`/`ParamZeros`, using the same `recordAliases(file)` alias map and the same alias-recursion bound as `zeroFromAST` (`maxZeroDepth`, param_zero.go). Stored on `ExportInfo` as `ParamSchemas []paramSchema` (`json:"-"` — runtime-only, like `ParamZeros`).
2. **The binder** (new, `internal/apiserver/arg_bind.go, ~180 LOC`):
   `func bindCallArgs(export ExportInfo, args []any) ([]any, error)`
   Walks `args[i]` against `export.ParamSchemas[i]`:
   - `int`: JSON number (or `json.Number`/Go int) that is integral → `&eval.IntValue{n}`; non-integral → `fmt.Errorf("invalid argument for parameter %q: %v is not an int", name, val)`.
   - `float`: any JSON number → `&eval.FloatValue{x}` (whole numbers stay float — this is the REST mirror-bug fix).
   - `bool` → `BoolValue`, `string` → `StringValue`; JSON type mismatch → same structured rejection.
   - `record`: `map[string]any` → `RecordValue`, recursing per declared field schemas; **fields not in the declared schema pass through `embed.FromGo`** (heuristic) so open records and extra fields keep working.
   - `list`/`array`: recurse per element schema when known, else per-element `FromGo`.
   - `unknown` (ADTs, type vars, labelled refinements the extractor can't name) and `nil`: current behavior — `FromGo`/Unit, unchanged.
   - **Already-`eval.Value` args pass through identically** (multipart `*eval.BytesValue`, injected `_headers` JObject, `@mcp_file` records, zeros) — same contract as `FromGo`.
   - Argument-position errors are collected per param in declaration order (deterministic, matching the existing missing-param error style).
3. **Typed zeros** (modify `handler.go` `zeroValueForType` + `param_zero.go` `paramZero`): `int` → `&eval.IntValue{0}`, `float` → `&eval.FloatValue{0}`, `string`/`bool`/`list`/`record` similarly typed. The `float64(0) // JSON numbers are float64` comment and behavior at handler.go:180 is the MCP omitted-`@optional`-int bug and dies with it.
4. **Surface integration** — one call each:
   - `mcp.go` makeToolHandler: after named/positional resolution, secrets, file binding, headers injection → `bindCallArgs` → `engine.Call`. On binder error: `mcpError(...)` (IsError, in-band — same channel as the existing missing-param rejection).
   - `a2a.go`: same binder before `engine.Call`; binder error → JSON-RPC `-32602` (the existing invalid-params code used by a2a.go for unknown functions).
   - `routes_dispatch.go` callFunction: after `resolveArgs`/multipart → binder; error → HTTP 400 `FunctionCallResponse{Error: "invalid arguments: …"}` (reuses the existing "invalid arguments:" prefix at routes_dispatch.go:104).
   - `mcp_ui_serve.go`: binder (zeros only today, but adopts the shared path so it cannot re-diverge).
   - `mcp_auth.go`, `routes_ws.go`: unchanged (string-only / structured args — audited, out of scope).

### Implementation Plan

**Phase 1: Schema extraction + binder core** (~4 hours)
- [ ] `paramSchema` type + `extractParamSchemas` (AST → schemas, alias-bounded), wired into `extractParamInfo`; `ExportInfo.ParamSchemas`
- [ ] `bindCallArgs` in `arg_bind.go` with the semantics table above; unit tests for every kind + rejection + eval.Value identity + open-record passthrough

**Phase 2: Typed zeros + surface integration** (~4 hours)
- [ ] `zeroValueForType`/`paramZero` emit `eval.Value` zeros; existing zero tests updated (m-serve-api-zero-value-padding fixtures must still pass)
- [ ] mcp.go, a2a.go, routes_dispatch.go, mcp_ui_serve.go call the binder; `CallPreserveFloats` gone from apiserver
- [ ] Surface error mapping (MCP in-band / REST 400 / A2A -32602)

**Phase 3: Parity tests + docs** (~4 hours)
- [ ] The full matrix as parity tests on MCP and REST (and A2A for the int case), incl. first-ever successful MCP `int` tools/call
- [ ] Regression: multipart, `@mcp_file`, `_headers`, `@raw`, legacy `{"args":[...]}` positional, query-arg fallback, `@optional` records
- [ ] `make test`, `make check-boundaries`, docs/changelog entry

### Files to Modify/Create

**New files:**
- `internal/apiserver/arg_bind.go` (~180 LOC) — the binder
- `internal/apiserver/param_schema.go` (~80 LOC) — declared type schema extraction
- `internal/apiserver/arg_bind_test.go` (~300 LOC) — unit + parity matrix tests

**Modified files:**
- `internal/apiserver/export_info.go` (+3 LOC) — `ParamSchemas` field
- `internal/apiserver/routes.go` (+10 LOC) — extract schemas in `extractParamInfo`
- `internal/apiserver/handler.go` (+10/-10 LOC) — typed zeros in `zeroValueForType`
- `internal/apiserver/param_zero.go` (+10/-5 LOC) — typed zeros at the shared entry
- `internal/apiserver/mcp.go` (+8/-4 LOC) — binder + `engine.Call`
- `internal/apiserver/a2a.go` (+6/-2 LOC) — same
- `internal/apiserver/mcp_ui_serve.go` (+4/-2 LOC) — same
- `internal/apiserver/routes_dispatch.go` (+8/-2 LOC) — same

---

## Examples

### Example 1: The report's repro, on both surfaces

```ailang
module api/t
-- Double a number.
@mcp_title("Twice")
export func twice(n: int) -> int = n * 2
```

**Before** (live, v0.57.0 `b23cadc`):
```
MCP  tools/call {"name":"twice","arguments":{"n":21}}  → isError, "function call failed: numeric dictionary method \"mul\" received an int and a float…"
REST POST /api/api/t/twice {"args":[21]}              → {"result":42}
```

**After** (both surfaces):
```
MCP  tools/call {"name":"twice","arguments":{"n":21}}  → "42"
REST POST /api/api/t/twice {"args":[21]}               → {"result":42}
MCP  tools/call {"name":"twice","arguments":{"n":2.5}} → isError, "invalid argument for parameter \"n\": 2.5 is not an int"
REST POST /api/api/t/twice {"args":[2.5]}             → HTTP 400 {"error":"invalid arguments: argument for parameter \"n\": 2.5 is not an int"}
```

### Example 2: The REST mirror bug the report didn't hit

```ailang
export func half(x: float) -> float = x / 2.0
```

**Before:** `REST {"args":[2]}` → 500 (`div` dispatch error; `FromGo` whole-number heuristic bound `IntValue`). **After:** `{"result":1.0}` on REST and MCP alike — the binder types by the *declared* `float`, not by the JSON's shape.

### Example 3: Omitted `@optional int` on MCP

```ailang
@optional("n")
export func opt(n: int) -> int = n + 1
```
**Before:** MCP `{}` → `zeroValueForType("int")` = `float64(0)` → `CallPreserveFloats` → `FloatValue(0)` → dispatch error (live-verified). **After:** binds `IntValue(0)`, returns `1` — same as REST today.

---

## Success Criteria

- [ ] All 12 parity-matrix cases pass on both MCP and REST (int/float params × whole/non-integral numbers × both surfaces, int/float record fields, omitted `@optional int`)
- [ ] Non-integral-for-int rejected as a structured, param-named invalid-argument error on MCP (IsError), REST (400), and A2A (-32602)
- [ ] `grep -rn CallPreserveFloats internal/apiserver/` → no hits
- [ ] First successful MCP `int` tools/call test exists and is deterministic
- [ ] Regression suites green: multipart, `@mcp_file`, `_headers`, `@raw`, legacy positional, query fallback, zero-value-padding fixtures, `TestFromGoWholeFloatBecomesInt` (embed — unchanged semantics)
- [ ] `make test`, `make check-boundaries` pass
- [ ] Documentation updated (serve-api MCP/REST arg-binding docs + CHANGELOG entry)

## Testing Strategy

**Unit tests (`arg_bind_test.go`):**
- Per-kind conversion matrix; non-integral→int, number→string, string→int rejections (message text pinned)
- Record recursion (declared fields typed, extra fields heuristic-passthrough), list element typing, alias-bound recursion (reuse `maxZeroDepth` fixture pattern)
- `eval.Value` identity passthrough (BytesValue, TaggedValue headers, typed zeros); nil → Unit
- Typed zeros: `zeroValueForType("int")` yields `*eval.IntValue{0}`

**Integration / parity tests:**
- MCP via the existing `makeToolHandler` harness (`mcp_param_binding_test.go` pattern; the `testServer` fixture already ships `add(x: int, y: int) -> int` — V8): successful int call, whole-float-for-float, 2.5 rejection, omitted `@optional` int
- REST via `callFunction`/httptest for the same matrix (`named_args_test.go` pattern)
- A2A: int arg through the message-parts path

**Regression-surface tests:** every row of the Systemic Audit table that keeps its behavior (multipart bytes, file objects, headers, WS unaffected, token verifier unaffected).

**Manual:** run the report's exact REPRO (`serve-api --mcp-http --port 18991`) and the Examples above.

## Deferred Decisions

- Exact error message wording (beyond pinning the `invalid argument for parameter %q` shape) — agent may choose
- Whether the binder also coerces query-string values ("5" → 5 for an int param) instead of rejecting — agent may choose, but rejecting (current proposal) is the no-silent-fallback default; coercion would be a follow-up
- Internal helper names / file split between `arg_bind.go` and `param_schema.go` — agent may choose
- Whether `serveapi/` (World's SDK-free seam, which delegates binding to the embedder's `Invoker`) should later grow a reusable binder export — human at review; explicitly out of scope now

## Non-Goals

- **Changing `FromGo`/`FromGoPreserveFloats` semantics** — they are public embedder API; apiserver simply stops relying on the heuristic
- **WebSocket routes and the token verifier** — no JSON-number binding (audited, V10/V11)
- **`serveapi/` World seam** — takes `json.RawMessage` and delegates binding to the embedder; untouched
- **Closed-record strictness / rejecting extra fields** — open records stay open (heuristic passthrough)
- **Coercing query strings / multipart form fields per declared type** — possible follow-up; binder rejects type-mismatched values instead of guessing
- **`std/json` `Json` ADT params** — flow through the existing tagged-value path, unchanged

## Timeline

**Week 1** (12 hours):
- Phase 1: schema + binder core + unit tests (4h)
- Phase 2: typed zeros + surface integration (4h)
- Phase 3: parity/regression tests + docs (4h)

**Total: ~12 hours across 1 week** (initial 6h estimate, 2x'd)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Strict scalar rejection breaks a live client that sends, e.g., numbers for string params (previously a 500, now a 400) | Low | Same error class or better; enumerated under "What deliberately changes"; rejection messages name the param so clients can adapt |
| Record schema extraction misses a type form (labelled refinements, type apps) | Med | Unknown/unextractable → `unknown` kind → heuristic passthrough = today's behavior, never worse; unit tests pin the extractable forms |
| Binder changes zero values' Go type and breaks multipart padding consumers | Med | Phase 2 runs the full existing m-serve-api-zero-value-padding + multipart suites before surface switch |
| A2A/MCP error-channel differences hide binder rejections | Low | One integration test per surface pins the error channel (IsError / 400 / -32602) |

## Conflict Surface

Not a parser/typechecker/codegen change — the language is untouched; `internal/apiserver` only. But the *binding* position each surface shares is enumerated (the skill's discipline applied to the protocol layer):

**Positions touched:** the resolved-argument slot between surface-specific resolution and the engine call.

**What else already lives in that position (must keep working):**
- `@mcp_file` bound file records (mcp.go `bindFileArgs`), `_headers` injected JObject (`stringMapToJObject` → `*eval.TaggedValue`), `@mcp_secret` zero-bound params, listed-surface secrets
- REST: `@raw` HttpRequest record, multipart `*eval.BytesValue` args, legacy `{"args":[...]}` positional form, `?args=` query fallback, body→query→zero precedence (`resolveArgs`)
- Unit params (nil→Unit), zero-arity functions, `unknown`-typed params (ADTs, generics)

**What deliberately changes:**
1. Non-integral number for an `int` param (and any JSON-type mismatch for `bool`/`string`): opaque evaluator error → structured invalid-argument rejection before the call. Clients see 400/-32602/IsError instead of 500/in-band dispatch text.
2. Integral JSON numbers for `float` params on REST now bind `FloatValue` (previously `IntValue` → runtime failure) — a fix, but an observable value-type change in the only case that previously *crashed*.
3. Integral JSON numbers for `int` params on MCP now bind `IntValue` (previously `FloatValue` → failure) — the headline fix.
4. Query-string values for typed params that previously crashed the call now get a structured rejection (unless the coercion deferred-decision is taken).

**Programs that MUST still work** (regression fixtures, all existing): `internal/apiserver/mcp_file_test.go` (file objects), `internal/apiserver/mcp_param_binding_test.go` (presence checks, string success, legacy positional), `internal/apiserver/named_args_test.go` (named/snake_case binding, zero padding), multipart upload tests, `internal/embed/embed_test.go` `TestFromGoWholeFloatBecomesInt` (embed semantics unchanged).

## Verification Log

Every load-bearing claim above was checked against code or a live run on this worktree (ailang v0.57.0, commit `b23cadc`, binary `/usr/local/bin/ailang`):

| # | Claim | How verified |
|---|---|---|
| V1 | MCP tools/call with int param fails | Live: module `api/t` `twice(n: int)`, `tools/call {"n":21}` → `isError` "function call failed: numeric dictionary method \"mul\" received an int and a float…" (error text evolved since v0.52.5's "expected float arguments"; same root cause — the param binds `FloatValue`) |
| V2 | REST int param works | Live: same module → `{"result":42}` |
| V3 | REST **float** param with integral JSON fails (report's "REST converts per declared type" is wrong) | Live: `half(x: float)`, `{"args":[2]}` → 500 dispatch error; `{"args":[2.5]}` → `{"result":1.25}` |
| V4 | MCP float param + integral JSON works | Live: `{"x":4}` → `2` |
| V5 | Record fields: MCP int-field fails / REST float-field fails | Live: `countIt(r:{count:int})` MCP → error; `scaleIt(r:{factor:float})` REST `{"factor":2}` → error, `{"factor":2.5}` → `7.5` |
| V6 | Omitted `@optional int` on MCP binds float zero and fails | Live: `opt` `{}` → "add received an int and a float" error; code: `zeroValueForType("int")` returns `float64(0)` (handler.go:180) + `CallPreserveFloats` keeps it `FloatValue` |
| V7 | Mechanism locus: `CallPreserveFloats` keeps float64 as FloatValue "even for whole numbers" | Code: internal/embed/embed.go:269-272; callers in apiserver are exactly mcp.go:456, a2a.go:226, mcp_ui_serve.go:103 (grep V9); REST comment routes_dispatch.go:138-141 records the deliberate opposite choice |
| V8 | No existing test performs a successful MCP int tools/call | Read: `mcp_param_binding_test.go` covers rejections + a string success (`hello`); the `testServer` fixture's `add(x:int,y:int)` (server_test.go:38) is used only for missing-param rejection — the coverage hole that let this ship |
| V9 | No other apiserver `CallPreserveFloats` callers | `grep -n CallPreserveFloats internal/apiserver/*.go` → mcp.go:126 (comment), a2a.go:226, mcp.go:456, mcp_ui_serve.go:103 (+ param_zero.go:59 comment) — the audit table is exhaustive |
| V10 | WS routes and token verifier don't bind JSON numbers | Read: routes_ws.go:217-219 (StreamConn + wsReqRecord), mcp_auth.go:207 (single string param) |
| V11 | `serveapi/` World seam is out of scope | Read: mcp_handler.go:21 + serveapi.go:92 — Invoker receives `json.RawMessage`; binding belongs to the embedder; no in-repo consumer constructs it |
| V12 | `ParamTypes` strings are collapsed names, insufficient for typed binding | Read: `paramTypeToString` (routes.go:68-85) → "int"/"record"/"list"/"unknown" |
| V13 | `FromGo` has an eval.Value identity path (the binder's delivery seam) | Read: internal/embed/convert.go:37-41; exercised today by multipart `*eval.BytesValue` |
| V14 | Error-channel formats cited exist | Read: mcpError in-band pattern (mcp.go, "missing required parameter(s)"), REST `"invalid arguments: %v"` 400 (routes_dispatch.go:104), A2A `-32602` (a2a.go:196) |
| V15 | No new error-code namespace needed | Rejected values surface as protocol-level strings (MCP IsError text / REST 400 body / JSON-RPC -32602); no MODxxx/TCxxx code allocated |

## Related Documents

**Implemented (may inform design):**
- [`design_docs/implemented/v0_9_5/m-serve-api-zero-value-padding.md`](../../implemented/v0_9_5/m-serve-api-zero-value-padding.md) — the omitted-param half of this family (typed zeros); this doc is its present-argument complement. Distinct topic confirmed: that doc covers *missing* params; this one covers *present* args' numeric typing, and the two share `ParamTypes`/`ParamZeros` plumbing.

**Planned (check for overlap):**
- [`design_docs/planned/ailang-core-triage/serveapi-mcp-int-args-bind-float64.md`](../ailang-core-triage/serveapi-mcp-int-args-bind-float64.md) — the triage note that recommended this design doc (its mechanism claims are re-verified and extended above, including the REST mirror bug it missed)
- [`design_docs/planned/ailang-core-triage/serveapi-protocol-mcp-dispatch.md`](../ailang-core-triage/serveapi-protocol-mcp-dispatch.md) — SDK-free MCP dispatch for World (different layer; the binder stays in `internal/apiserver`, Non-Goal here)

**Related implemented behavior:** `TestFromGoWholeFloatBecomesInt` (internal/embed/embed_test.go:503) — the REST-side heuristic's regression test, which stays green because embed semantics are unchanged.

## References

- **Motivation:** CLI feedback message (task-d8420071), docparse `createUpload` float workaround (docparse PR #229)
- **Prior family:** commit 39dd37990 / M-SERVE-API-ZERO-VALUE-PADDING (REST/MCP binding divergence #1)
- **Axiom reference:** [Design Axioms](/docs/references/axioms)

## Future Work

- Query-string / multipart-field coercion per declared type (deferred decision)
- A reusable binder export for `serveapi/` embedders if World needs one
- Surfacing declared param types in MCP `inputSchema` strictness (e.g. `additionalProperties`) — schema is already `integer` for int today; binder now honours it

---

**Document created**: 2026-11-03
**Last updated**: 2026-11-03
