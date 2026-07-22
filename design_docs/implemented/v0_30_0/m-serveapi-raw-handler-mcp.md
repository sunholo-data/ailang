# M-SERVEAPI-RAW-HANDLER-MCP: `@nomcp` exclusion + opt-in `@raw` handlers over MCP

**Status**: **✅ LANDED 2026-07-22 (mission iteration 78) — M1 shipped; M2 DROPPED → doc COMPLETE.**
`@nomcp` MCP-exclusion annotation shipped as a standalone sprint (PR #452 squash `ee04f13d0`):
planner opus → executor opus (`2d6596292`) → evaluator **sonnet** (generator≠judge) **PASS 96/100
round 1, no defects**; dev CI green per-workflow (19 checks). Closes the live docparse
`getKeyUsage`/`requestHistory` MCP capability leak. Diff confined to the parser annotation
allowlist + `internal/apiserver/` — **no eval-core change** (Minimal-Frozen-Core north star). M2
(`@raw`-over-MCP fake envelope) was **DROPPED by Mark 2026-07-20** ("ship M1 and drop M2"): `@raw`
routes simply are NOT exposed on the MCP tool surface (the honest contract; both round-2 quorum
objections dissolve with the feature). With M1 landed and M2 dropped, this doc is fully resolved.
_(Prior status: DECIDED by Mark 2026-07-20; M1 clean + unobjected both quorum rounds, no re-quorum.)_
**Target**: v0.30.0
**Priority**: P1 (Medium) — unblocks docparse quota-hardening item 5; closes a live MCP capability leak
**Estimated**: 2 days (M1 ~0.5d unchanged, M2 ~1.25d, docs/validation ~0.25d)
**Dependencies**: None (builds on the existing `@route`/`@raw`/`@noexpose`/`@mcp_name` annotation + MCP bridge in `internal/apiserver/`)

**Revision history**: Rev 2 (2026-07-19): M2 reworked to opt-in + structured-error + route-decoupled envelope per quorum round-1 objections (gpt5-6-sol authority/no-silent-fallback; gemini per-function cardinality).

## ⛔ Quorum Reblock — needs-human-review (2026-07-19, mission iteration 57)

This doc went through the QUORUM-AT-PICK gate and was **blocked twice**. The gate allows exactly one
revision pass + one re-quorum; after that it is parked for a human decision. **Both rounds objected
ONLY to M2** — **M1 (`@nomcp`) was never objected to and is independently shippable.**

**Round 1 (Rev 1) — BLOCKED** (`m-serveapi-raw-handler-mcp-2026-07-19T01-21-10Z.json`):
- *gpt5-6-sol*: M2 made every broken `@raw` route MCP-callable **by default**, silently fabricating
  empty headers/query → authority-widening + silent-fallback; the "narrows, never widens" claim was false.
- *gemini-3-1-pro*: M2 threaded a **singular** `RouteMethod`/`RoutePath` into `makeToolHandler`, but MCP
  tools register **per function**, and a function can have 0 or >1 `@route`. Structurally flawed.

**Rev 2 fix** (codex/gpt-5.6-sol designer): `@mcp` opt-in (unmodified `@raw` not registered), function-keyed
envelope (`method="MCP"`, `path="/mcp/tool/"+funcName`, route-independent), and typed unavailable-context
**sentinels** for `headers`/`query` → `MCP_TRANSPORT_CONTEXT_UNAVAILABLE` structured error.

**Round 2 (Rev 2) — STILL BLOCKED** (`m-serveapi-raw-handler-mcp-2026-07-19T02-02-59Z.json`): the sentinel
fix introduced a deeper flaw. **The `headers`/`query` fields are typed `Json`.** A non-`Json` sentinel
**type-mismatch-panics at parameter binding — before any projection** — defeating "fail only on access"
(gemini). Making the sentinel a valid `Json` value to survive binding would require modifying core
`std/json` to intercept it → a **Minimal-Frozen-Core violation** (PROGRAM.md north star). gpt5-6-sol
concurred: an unjustified expansion of the frozen evaluator core with an unverified implementation premise.

### Human decision fork (for Mark on #399)

1. **Split M1 out and ship it now** (RECOMMENDED). `@nomcp` is clean, unobjected in both rounds, and
   closes the live docparse MCP leak (`getKeyUsage`/`requestHistory`). It touches only the parser allowlist
   + `internal/apiserver/` (no eval-core change). Route M1 as its own ~0.5d sprint; keep M2 parked.
2. **Choose an M2 architecture** (both from gemini's Rev-2 alternatives; both avoid the frozen-core change):
   - **(a) Valid-`Json` provenance marker.** Populate `headers`/`query` with a recognizable `Json` object
     (e.g. `{"_transport":"MCP_UNAVAILABLE"}`) and require opted-in `@raw` handlers to branch on
     `req.method == "MCP"` before verifying signatures. No core change; burden shifts to the handler author.
   - **(b) Drop M2's fake-envelope entirely.** Rely on M1's `@nomcp` to hide broken `@raw` tools; require
     authors to write dedicated **non-`@raw`** functions for agent/MCP access. Simplest; no `@raw`-over-MCP.
3. **Keep the whole doc parked** until the orchestration-flagship work clarifies the `@raw`/MCP contract.

Everything below this section is the Rev-2 design as-written; the M2 sentinel mechanism (Solution Design
item 6, `internal/eval/` in Files-to-Modify) is the specific part the round-2 quorum rejected.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No nondeterminism introduced. `@nomcp`/`@mcp` are static annotations; the opted-in `@raw`→MCP envelope synthesis is a pure deterministic transform of the function identity and tool arguments. |
| A2: Replayability | 0 | No impact on traces. |
| A3: Effect Legibility | 0 | Handler effect rows unchanged; no hidden effects added. |
| A4: Explicit Authority | +1 | `@nomcp` explicitly withholds a route from MCP, while M2 requires an explicit parameterless `@mcp` annotation before any `@raw` export is registered as an MCP capability. The rejected default-on design widened authority; Rev 2 does not. |
| A5: Bounded Verification | 0 | No type-system change. |
| A6: Safe Concurrency | 0 | No concurrency changes. |
| A7: Machines First | +1 | MCP lists only raw tools deliberately authorized for MCP, gives opted-in tools a deterministic function-keyed envelope, and reports unavailable transport context with a stable typed error instead of misleading values or evaluator failures. |
| A8: Minimal Syntax | 0 | Adds two parameterless annotations (`@nomcp`, `@mcp`) in the existing `@raw`/`@nowrap`/`@noexpose` family. `@mcp` is the smallest syntax that can express the required positive authority grant; no grammar class, operator, or keyword is added. |
| A9: Cost Visibility | 0 | No resource-cost surface change. |
| A10: Composability | +1 | `@nomcp` remains orthogonal to HTTP annotations; `@mcp` authorizes only the raw MCP bridge and does not derive identity from any singular `@route`. Zero or multiple route annotations therefore do not change the MCP envelope. |
| A11: Structured Failure | +1 | Missing HTTP-only context is represented by typed unavailable-context sentinels. Accessing MCP-unavailable `headers` or `query` returns `MCP_TRANSPORT_CONTEXT_UNAVAILABLE` with field and provenance details, never `None`, an empty-value fallback, or an opaque eval panic. |
| A12: System Boundary | +1 | The boundary is explicit: `method="MCP"`, `path="/mcp/tool/" + funcName`, body from tool arguments, and typed unavailable sentinels for HTTP-only fields. The envelope identifies MCP provenance instead of pretending to be a real HTTP request. |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted — the old default-on M2 claim was false; Rev 2 narrows by default and widens only with explicit `@mcp` authority
- [x] A7 (Machines First): Tool discovery, provenance, and unavailable-context failures are machine-readable

### Decision Thresholds

Net +5 ≥ +2, no −1 on A1/A3/A4/A7 → **✅ Proceed to implementation.**

## Problem Statement

`serve-api --mcp-http` auto-exposes **every** exposed `@route` handler as an MCP tool at `POST /mcp/`
(`internal/apiserver/mcp.go:registerTools`). Two distinct problems fall out of this:

**Problem A — MCP capability leak (no way to keep a `@route` HTTP-only).**
The only exclusion knob today is `@noexpose`, and it is the wrong tool: it hides an export from **HTTP
too**, and `@route` explicitly overrides it (`internal/apiserver/routes.go:106` sets
`IsNoExpose = false` whenever a `@route` is present). So there is **no** annotation that means "serve
this over HTTP but do **not** register it as an MCP tool." Concretely, docparse's `getKeyUsage` and
`requestHistory` routes remain live MCP tools that an agent can call, even though they are operator/
HTTP-only surfaces — a leak the docparse quota-hardening work flagged and could not close.

**Problem B — `@raw` handlers are unusable over MCP, but automatically registering them is not safe.**
A `@raw @route` handler's single parameter is the HTTP-request envelope
`req: {body: string, headers: Json, method: string, path: string, query: Json}` (see the canonical
example [`examples/runnable/serve_api_webhook.ail:25`](examples/runnable/serve_api_webhook.ail)). On the
HTTP path, `callFunction` builds that envelope via `buildHttpRequestRecord`
(`internal/apiserver/routes_dispatch.go:51`). But the MCP bridge (`makeToolHandler`,
`internal/apiserver/mcp.go:188`) has no knowledge that a tool is `@raw`; it maps the tool arguments
straight to the declared parameter. The result is two dead-end calls:

- Natural agent call `{"req": {"args": ["key"]}}` → the arg object `{args:["key"]}` becomes the `req`
  record → handler reads `req.headers` → **`record has no field: headers`**.
- Legacy call `{"args": ["key"]}` → `"key"` (a string) is passed as `req` → handler does field access on
  a string → **`cannot access field of non-record value`**.

Retyping the handler to `req: Json` does **not** fix it: on the HTTP path serve-api passes an AILANG
`RecordValue` (not a `Json` ADT value), so `std/json` `getString`/`get` return `None` and `encode(req)`
fails with `_json_encode: expected Json, got *eval.RecordValue`. The HTTP and MCP shapes are
irreconcilable from AILANG-side code alone.

The original M2 proposed making every currently registered `@raw` export callable by fabricating an
HTTP-looking envelope with empty headers/query. That is rejected: it would turn broken registrations
into working capabilities without author authorization, and webhook/auth handlers could interpret
unavailable transport context as a genuine empty HTTP value. M2 must therefore be positive opt-in and
must preserve the distinction between MCP provenance and HTTP transport data.

**Current State:**
- `@raw` MCP tools are auto-registered but **100% fail** on any natural call shape (Problem B); Rev 2
  treats this broken registration as a leak to remove, not an implicit grant to make callable.
- No annotation can keep a `@route` on HTTP while hiding it from MCP (Problem A).
- docparse item 5 is **blocked**; docparse shipped a `mcpAccount` workaround, but the leaky
  `getKeyUsage`/`requestHistory` MCP tools remain (per docparse note `docparse-raw-handler-mcp-limitation.md`).

**Impact:**
- Any AILANG project serving `@raw` webhook/signature handlers over `--mcp-http` exposes broken tools.
- Any project with operator-only `@route`s over `--mcp-http` leaks them to the agent surface with no
  opt-out — a security/authority concern, not just ergonomics.

## Goals

**Primary Goal:** Give AILANG authors (a) a way to keep a `@route` on HTTP but off the MCP tool surface,
and (b) an explicit, provenance-safe way to authorize selected `@raw` handlers for MCP without treating
missing HTTP transport context as real empty data.

**Success Metrics:**
- A `@route @nomcp` handler is reachable over HTTP (200) and **absent** from the MCP `tools/list` and
  `tools/call` surface (M1).
- The two docparse leaks (`getKeyUsage`, `requestHistory`) are closable with a one-line annotation, with
  no HTTP behavior change (M1).
- An unmodified `@raw @route` handler is absent from MCP `tools/list` and unavailable through
  `tools/call`; adding parameterless `@mcp` explicitly authorizes registration (M2).
- An opted-in, body-only `@mcp @raw` handler executes successfully with a function-keyed MCP envelope;
  an opted-in handler that reads unavailable `headers`/`query` receives the typed structured error
  `MCP_TRANSPORT_CONTEXT_UNAVAILABLE` rather than `None`, an empty fallback, or an eval panic (M2).
- The canonical `examples/runnable/serve_api_webhook.ail` handler continues to type-check, run, and serve
  over HTTP **unchanged** (both milestones — no breaking change to the documented `@raw` record shape).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| `@nomcp` is a new **parameterless annotation** (not a `@route` option flag, not overloading `@noexpose`) | Sets the author-facing syntax + the annotation allowlist entry; parser + docs ripple | human | design | med |
| `@nomcp` hides from the **MCP tool surface only** (HTTP + OpenAPI + A2A agent card unaffected) | Defines exactly which surfaces the knob governs; wrong scope re-leaks or over-hides | human | design | med |
| M2 uses a new **parameterless `@mcp` annotation** as positive authority for `@raw` MCP registration | Exact semantics: `@raw` exports are not MCP tools unless they carry `@mcp`; `@nomcp` still wins; on non-`@raw` exports `@mcp` is accepted but redundant because their existing registration policy is unchanged | quorum Rev 2 | design | med |
| The MCP raw envelope is keyed to the **function**, never a singular route | MCP registers one tool per exported function, while a function can have zero or multiple `@route` annotations; use `method="MCP"` and `path="/mcp/tool/" + funcName` for all cardinalities | quorum Rev 2 | design | med |
| `headers` and `query` are typed **unavailable-context sentinels**, not empty `JObject`s | Prevents signature/auth/query logic from silently treating absent MCP transport data as genuine empty HTTP data; projection produces a structured typed boundary error | quorum Rev 2 | design | high |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] `@nomcp` annotation name is ratified (vs. e.g. `@mcp(false)` / `@httponly`). **Recommended: `@nomcp`** — parallels `@noexpose`, parameterless, one allowlist entry.
- [ ] `@nomcp` scope confirmed = **MCP tool surface only** (HTTP, OpenAPI `/openapi.json`, and A2A `/.well-known/agent.json` remain unaffected).
- [ ] `@mcp` semantics confirmed: parameterless positive authorization for `@raw` MCP registration;
      unannotated `@raw` exports are skipped; `@nomcp` takes precedence; non-`@raw` behavior is unchanged.
- [ ] M2 envelope contract confirmed: `req.body = encode(arguments)`, `req.method = "MCP"`,
      `req.path = "/mcp/tool/" + funcName`, and `req.headers`/`req.query` are typed unavailable sentinels
      that raise `MCP_TRANSPORT_CONTEXT_UNAVAILABLE` when projected. No route metadata participates.

## Solution Design

### Overview

Two independently shippable milestones against `internal/apiserver/` (+ one parser allowlist entry):

- **M1 — `@nomcp` exclusion annotation** (solves Problem A). New parameterless annotation that marks an
  export hidden from **MCP tool registration only**. Immediate unblock for the docparse leak.
- **M2 — opt-in `@raw` over MCP** (solves Problem B without widening authority). Add parameterless
  `@mcp`; skip all unannotated `@raw` exports during MCP registration. For opted-in raw exports,
  synthesize a function-keyed MCP-provenance envelope and use typed sentinels for unavailable HTTP-only
  context. The HTTP envelope and handler signature remain unchanged.

### Architecture

**Components:**

1. **Parser allowlist (`internal/parser/parser_decl.go`)** — the `parseAnnotation` switch is a closed
   allowlist (unknown `@name` is a hard parse error at line 47). Add a `case "nomcp"` returning
   `&ast.Annotation{Name: "nomcp", Pos: pos}`, mirroring `@noexpose`. Update both error-hint strings
   (lines 23, 48) to list `@nomcp`.

2. **Annotation extraction + `ExportInfo` (`internal/apiserver/routes.go`, `server.go`)** — add
   `IsNoMCP bool` to `ExportInfo` (`server.go:133`). Add `extractNoMCPAnnotations` (mirror of
   `extractNoExposeAnnotations`, `routes.go:212`) that sets `IsNoMCP = true` when `fn.GetAnnotation("nomcp")`
   is present. Unlike `@noexpose`, `@nomcp` is **not** overridden by `@route` — the whole point is to hide a
   `@route` from MCP. Wire the new extractor into the same load path that already calls the other
   `extract*` functions, at `internal/apiserver/module_entry.go:113-115` (right after
   `extractMCPNameAnnotations`).

3. **MCP registration filter (`internal/apiserver/mcp.go:registerTools`)** — the loop already skips
   `!isExposed(export)`. Add a second guard: `if export.IsNoMCP { continue }`. Keep `isExposed` unchanged
   so HTTP/OpenAPI/A2A are untouched.

4. **Positive raw-MCP authorization metadata (`internal/parser/parser_decl.go`,
   `internal/apiserver/routes.go`, `server.go`, M2)** — add parameterless `@mcp` to the closed annotation
   allowlist and `ExportInfo.IsMCP`. Extract `IsRaw` independently of `@route` (today it is assigned only
   inside `extractRouteAnnotations`) and extract `IsMCP` independently as well. Registration applies this
   order: existing `isExposed`; then `IsNoMCP`; then `if IsRaw && !IsMCP { continue }`. Thus an unmodified
   `@raw @route` disappears from `tools/list`, while non-raw registration remains unchanged. `@nomcp`
   wins over `@mcp` so an explicit denial cannot be overridden accidentally.

5. **Function-keyed MCP envelope (`internal/apiserver/mcp.go:makeToolHandler`, M2)** — thread only
   `IsRaw`/`IsMCP` plus the already available `funcName`; do **not** thread `RouteMethod` or `RoutePath`.
   For an authorized raw tool, bypass normal named/positional binding and call the function once with:
   `body = <MCP arguments marshalled to JSON>`, `method = "MCP"`,
   `path = "/mcp/tool/" + funcName`, and typed unavailable-context values for `headers` and `query`.
   These constants apply whether the function has zero, one, or multiple `@route` annotations.

6. **Typed unavailable-context failure (`internal/eval/` boundary value + MCP error mapping, M2)** —
   introduce an internal request-field sentinel carrying `{transport:"MCP", field:"headers"|"query"}`.
   It occupies the existing envelope slot without masquerading as `JObject([])`. If the handler projects
   one of those fields on the executed branch, evaluation stops with a typed boundary error; MCP maps it
   to an error result with code `MCP_TRANSPORT_CONTEXT_UNAVAILABLE` and structured data containing
   `transport`, `field`, `function`, and a remediation hint. Body-only handlers remain callable; security-
   relevant handlers never silently observe "no signature" or "no query".

### Implementation Plan

**Phase M1: `@nomcp` exclusion** (~4 hours)
- [ ] Parser: add `case "nomcp"` to `parseAnnotation`; update both hint strings; add a parser unit test
      (mirror the `@noexpose` test) asserting `@nomcp` parses and an unknown `@nope` still errors.
- [ ] `ExportInfo.IsNoMCP` field + `extractNoMCPAnnotations`; wire into the module load path.
- [ ] `registerTools`: `if export.IsNoMCP { continue }` guard.
- [ ] Tests: server-level test that a `@route @nomcp` export is present in HTTP routes + OpenAPI but
      absent from the MCP tool list; a `@nomcp` **without** `@route` still hidden from MCP (and, being a
      bare export, already hidden from routes-only HTTP — assert no interaction surprise).

**Phase M2: opt-in `@raw` over MCP** (~10 hours)
- [ ] Parser/extraction: add parameterless `@mcp`, `ExportInfo.IsMCP`, independent `@raw` extraction, and
      tests for `@mcp`, zero-route raw exports, and repeated `@route` annotations.
- [ ] Registration: skip `IsRaw && !IsMCP`; keep `@nomcp` as the final explicit denial; verify non-raw
      registration is unchanged.
- [ ] Add a function-keyed MCP envelope constructor with `method="MCP"` and
      `path="/mcp/tool/"+funcName`; no `RouteMethod`/`RoutePath` parameter is permitted.
- [ ] Add typed unavailable-context sentinels for `headers`/`query` plus structured MCP error mapping.
- [ ] `buildNamedInputSchema`: only opted-in `@mcp @raw` tools receive a permissive free-form object
      schema; unannotated raw exports have no MCP schema because they are not registered.
- [ ] Tests: (a) unmodified `@raw @route` absent/unavailable, (b) opted-in body-only raw handler callable,
      (c) opted-in signature/query handler returns `MCP_TRANSPORT_CONTEXT_UNAVAILABLE`, and (d) HTTP path
      for the same handler is byte-for-byte unchanged.

**Phase 3: Docs + examples + changelog** (~2 hours)
- [ ] `ailang prompt`/CLI help + `prompts/` annotation list gains `@nomcp` and the raw-only `@mcp` opt-in semantics.
- [ ] Extend `examples/runnable/serve_api_webhook.ail` (or a sibling) to show a `@nomcp` route and note the
      MCP-callable `@raw` behavior; keep it `verify-examples`-green.
- [ ] `CHANGELOG.md`; update the docparse note `docparse-raw-handler-mcp-limitation.md` to "resolved in v0.30.0".

### Files to Modify/Create

**Modified files:**
- `internal/parser/parser_decl.go` — `@nomcp` + `@mcp` allowlist cases and hint strings (~10 LOC)
- `internal/apiserver/server.go` — `ExportInfo.IsNoMCP` + `IsMCP` fields (~2 LOC)
- `internal/apiserver/routes.go` — `extractNoMCPAnnotations`, independent raw/MCP extraction, wiring (~30 LOC)
- `internal/apiserver/mcp.go` — registration guards; opted-in raw envelope; raw schema; structured error mapping (~65 LOC, M1+M2)
- `internal/eval/` — typed unavailable request-context value/error (~30 LOC, M2)
- `examples/runnable/serve_api_webhook.ail` — `@nomcp` demo + explicit raw MCP/error note (~12 LOC)
- CLI help / `prompts/` — annotation list (~4 LOC)

**New files:**
- `internal/apiserver/nomcp_test.go` — M1 exclusion tests (~120 LOC)
- `internal/apiserver/mcp_raw_test.go` — M2 authorization/provenance/structured-error tests (~220 LOC)

## Conflict Surface

**This design touches `internal/parser/` (annotation allowlist).** Required analysis:

1. **What positions does this extend?** The `@`-annotation position above an exported function
   declaration (parsed by `parseAnnotation`, consumed in the annotation loop at `parser_decl.go:213`).
   M2 additionally extends the MCP `tools/call` argument-binding position inside `makeToolHandler`.

2. **What else already lives in those positions?** The existing closed annotation allowlist:
   `@verify(...)`, `@route(...)`, `@mcp_name(...)`, `@raw`, `@nowrap`, `@noexpose`. `@nomcp` and `@mcp`
   are new siblings in the **same** parameterless family as `@raw`/`@nowrap`/`@noexpose`. The MCP
   argument-binding position currently holds legacy `{"args":[...]}`, named params `{name: value}`, and
   missing-param rejection (`M-MCP-UNIT-PARAM-BINDING`, `mcp.go:210`).

3. **How is it disambiguated?** Each annotation is an exact-string `switch` case; unknown names still
   hard-error. At registration, `@nomcp` denies MCP on every export, while `@mcp` is consulted only for
   `IsRaw`. In binding, the raw branch is reachable only for `IsRaw && IsMCP`; non-raw tools retain all
   existing argument shapes. The raw branch uses `funcName`, never route metadata.

4. **Programs that MUST still work post-change (regression fixtures):**
   - [`examples/runnable/serve_api_webhook.ail`](examples/runnable/serve_api_webhook.ail) — unmodified
     `@raw @route` with the `{body,headers,method,path,query}` record param: must type-check, `main()` must
     run, HTTP POST must serve unchanged, and the raw export must be absent from MCP until `@mcp` is added.
   - The same file's non-`@raw` `@route("GET","/health")` — must remain an MCP tool and HTTP route.
   - Any existing `@noexpose` export — must remain hidden from **both** HTTP and MCP (M1 must not entangle
     the two `Is*` flags).
   - Existing named-param and legacy-`args` MCP tool calls on **non-`@raw`** tools — unchanged (M2 gates
     solely on `IsRaw`).

5. **What deliberately changes (intentional incompatibilities):** Broken auto-registered `@raw` tools
   disappear from MCP unless the author adds `@mcp`; this is an intentional authority-tightening change,
   not a compatibility regression. For opted-in raw tools only, the schema becomes free-form and the
   argument object becomes `req.body`. Access to HTTP-only `headers`/`query` fails structurally. No
   non-`@raw` schema or binding changes. `@nomcp` remains independent and takes precedence.

**The honest answer is not "no conflicts":** the real risks are (a) entangling `IsNoMCP`/`IsMCP` with
HTTP exposure, (b) continuing to derive `IsRaw` or MCP identity from the first route, and (c) leaking a
sentinel as an ordinary `Json` value instead of mapping it to the typed MCP error. The regression fixtures
and explicit cardinality tests cover these risks.

## Examples

### Example 1: `@nomcp` — HTTP-only route (Problem A)

**Before** (no way to hide from MCP without also killing HTTP):
```ailang
-- getKeyUsage is HTTP-served AND auto-registered as an MCP tool the agent can call.
-- @noexpose would remove it from HTTP too, and @route overrides @noexpose anyway.
@route("GET", "/admin/key-usage")
export func getKeyUsage(account: string) -> Json ! {IO} { ... }
```

**After:**
```ailang
-- Served over HTTP; hidden from the --mcp-http tool surface.
@nomcp
@route("GET", "/admin/key-usage")
export func getKeyUsage(account: string) -> Json ! {IO} { ... }
```
`GET /admin/key-usage` → 200. `tools/list` → `getKeyUsage` absent. `/openapi.json` → still present.

### Example 2: opt-in `@raw` over MCP (Problem B)

**Before** (every natural MCP call fails):
```jsonc
// tools/call handleWebhook
{"req": {"args": ["key"]}}   // -> "record has no field: headers"
{"args": ["key"]}            // -> "cannot access field of non-record value"
```

**After, without modification** (authority-safe default):
```jsonc
// tools/list: handleWebhook absent
// tools/call handleWebhook: unknown/unregistered tool
```
The HTTP `POST /webhooks/example` path is unchanged.

**Explicitly opted-in body-only raw handler:**
```ailang
@mcp
@raw
@route("POST", "/events")
export func ingest(request: {body: string, headers: Json, method: string, path: string, query: Json})
  -> Result[Json, string] ! {IO} {
  decode(request.body)
}
```
```jsonc
// tools/call ingest { "event": "test" }
// req.body   = "{\"event\":\"test\"}"
// req.method = "MCP"
// req.path   = "/mcp/tool/ingest"
// req.headers/query = typed MCP-unavailable sentinels (not empty HTTP objects)
// -> handler decodes body and succeeds
```

**Canonical signature-verification handler:** adding `@mcp` authorizes registration, but its executed
`request.headers` projection returns this MCP error instead of silently using `"no-signature"`:
```json
{
  "code": "MCP_TRANSPORT_CONTEXT_UNAVAILABLE",
  "transport": "MCP",
  "field": "headers",
  "function": "handleWebhook"
}
```
An author who truly supports MCP must branch on `request.method == "MCP"` before touching unavailable
fields and document the alternate trust model; HTTP signature verification remains unchanged.

## Success Criteria

- [ ] `@nomcp` parses (parser unit test) and unknown annotations still hard-error.
- [ ] `@route @nomcp` export: HTTP 200 + present in OpenAPI, **absent** from MCP `tools/list`/`tools/call` (server test).
- [ ] `@noexpose` still hides from HTTP **and** MCP; the two flags do not interact (regression test).
- [ ] Unmodified `@raw @route` is absent from MCP and `tools/call` cannot invoke it (M2 authority test).
- [ ] Explicit `@mcp @raw` body-only handler executes with `method="MCP"` and function-keyed path (M2 test).
- [ ] Explicit `@mcp @raw` handler that projects `headers` or `query` receives structured
      `MCP_TRANSPORT_CONTEXT_UNAVAILABLE`; canonical signature verification never falls back to
      `"no-signature"` over MCP (M2 structured-failure test).
- [ ] `examples/runnable/serve_api_webhook.ail` type-checks, runs via `verify-examples`, and its HTTP envelope is unchanged.
- [ ] All tests passing; `make ci` green.
- [ ] Docs (`ailang prompt`, CLI help, changelog) list `@nomcp`; docparse note marked resolved.

## Testing Strategy

**Unit tests:**
- Parser: `@nomcp` accepted; `@nope` rejected with the updated hint.
- `extractNoMCPAnnotations`: sets `IsNoMCP`; independent of `IsNoExpose`/`RoutePath`.
- Parser/extraction: `@mcp` sets `IsMCP`; `@raw` sets `IsRaw` even with zero routes; repeated route
  annotations do not supply MCP method/path.
- M2 envelope builder: deterministic `method="MCP"`, `path="/mcp/tool/"+funcName`, JSON body, and typed
  unavailable sentinels for `headers`/`query`.
- Sentinel projection maps to `MCP_TRANSPORT_CONTEXT_UNAVAILABLE` with stable structured fields.

**Integration tests:**
- Boot a `Server` with fixture exports (`@route`, `@route @nomcp`, unmodified `@raw @route`, body-only
  `@mcp @raw`, and signature-reading `@mcp @raw`); assert tool list, successful opted-in call, structured
  unavailable-context error, and unchanged HTTP responses.
- Add zero-route and repeated-route raw fixtures; both use the same function-keyed MCP envelope when
  opted in, proving MCP registration remains per exported function rather than per route.

**Manual testing:**
- `ailang serve-api --mcp-http examples/runnable/serve_api_webhook.ail`; confirm the unmodified raw
  webhook is absent from `tools/list`, a `@nomcp` route is invisible but answers over HTTP, and an opted-in
  body-only fixture reports `method=MCP`/function path while the signature fixture returns the typed error.

## Deferred Decisions

- Exact permissive `inputSchema` for opted-in `@raw` MCP tools (free-form `object` vs. surfacing the
  declared body sub-shape via a `body` property) — **agent may choose** the least-surprising schema that
  lets a natural argument object validate; default to free-form `object`.
- Whether the MCP-only envelope constructor lives in `mcp.go` or a new `mcp_envelope.go` — **agent may
  choose** based on file-size targets. It must remain separate from the unchanged HTTP constructor.

## Non-Goals

- **Making HTTP `@raw` emit a `Json` value / retyping `req: Json`.** Rejected: it breaks the documented
  `request.body`/`request.method` record access in every existing `@raw` handler. M2 keeps the record shape.
- **Automatically enabling broken `@raw` registrations.** Rejected: no working MCP capability is granted
  without parameterless `@mcp`; unmodified raw exports are intentionally not registered.
- **Treating unavailable MCP headers/query as empty HTTP values.** Rejected: those fields are typed
  unavailable sentinels and fail structurally on access.
- **Deriving MCP method/path from `@route`.** Rejected: MCP is per function and route cardinality is not
  singular; method/path are fixed MCP-provenance constants.
- **A per-surface matrix annotation** (e.g. `@expose(http, a2a)` selecting arbitrary surface sets). Out of
  scope; `@nomcp` covers the concrete need. Revisit only if OpenAPI/A2A need independent hiding.
- **Auth/rate-limiting for `@raw` MCP tools.** Orthogonal; existing MCP rate-limiting applies.

## Timeline

**Day 1** (~8h): M1 (`@nomcp`) unchanged; M2 `@mcp`/independent raw metadata and registration tests.
**Day 2** (~8h): Function-keyed envelope, typed unavailable-context error, MCP tests, docs, examples,
changelog, and `make ci`.

**Total: ~2 days.**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| M2 changes the HTTP `@raw` envelope while adding MCP provenance | High | Keep HTTP construction untouched; use a separate MCP envelope constructor and assert the webhook HTTP record remains byte-for-byte unchanged. |
| `IsNoMCP` entangled with `IsNoExpose`/`@route` override logic → re-leak or over-hide | Med | Keep flags independent; explicit test that `@noexpose` and `@nomcp` each hide only their intended surfaces. |
| Author confusion between `@noexpose` (HTTP+MCP) and `@nomcp` (MCP-only) | Low | Doc table + example contrasting the two; CLI hint lists both. |
| `@raw` tool argument marshalling loses type fidelity (int vs float in body JSON) | Low | Body is a JSON **string** the handler decodes itself, exactly as over HTTP — same fidelity as the real transport. |
| `@mcp` accidentally overrides `@nomcp` or enables all raw exports | High | Registration precedence is explicit (`IsNoMCP` deny; raw requires `IsMCP`) with list/call negative tests. |
| Route metadata is still threaded into the MCP envelope | High | `makeToolHandler` signature must not accept `RouteMethod`/`RoutePath`; zero/one/multiple-route tests assert identical function-keyed provenance. |
| Sentinel leaks into AILANG as an ordinary `Json` and yields `None` | High | Evaluator projection raises a typed boundary error; integration test asserts exact error code/data and rejects `"no-signature"`. |

## Related Documents

**Implemented (may inform design):**
- [design_docs/implemented/v0_11_0/m-mcp-quality-sprint-plan.md](design_docs/implemented/v0_11_0/m-mcp-quality-sprint-plan.md) (0.53) — MCP tool-name generation/validation this builds on. Distinct: that hardened names; this adds surface exclusion + `@raw` binding.
- [design_docs/implemented/v0_10_0/m-serve-api-agent-enhancements-sprint-plan.md](design_docs/implemented/v0_10_0/m-serve-api-agent-enhancements-sprint-plan.md) (0.48) — original `@route`/`@raw` serve-api surface.

**Planned (check for overlap):**
- [design_docs/planned/v0_29_0/m-serve-api-live-tool-registry.md](design_docs/planned/v0_29_0/m-serve-api-live-tool-registry.md) (0.49) — dynamic tool registry. Distinct: that governs *which modules* register; this governs *whether an exposed export* becomes an MCP tool + how `@raw` binds. Complementary, non-overlapping.

## References

- [Design Axioms](/docs/references/axioms)
- Canonical `@raw` example: [`examples/runnable/serve_api_webhook.ail`](examples/runnable/serve_api_webhook.ail)
- Code anchors: `internal/apiserver/mcp.go:registerTools`/`makeToolHandler`, `internal/apiserver/routes.go:extractRouteAnnotations`/`extractNoExposeAnnotations`/`buildHttpRequestRecord`, `internal/apiserver/server.go:ExportInfo`, `internal/ast/ast_decl.go:GetAnnotation`, `internal/apiserver/routes_dispatch.go:callFunction`, `internal/parser/parser_decl.go:parseAnnotation`
- docparse note: `docparse-raw-handler-mcp-limitation.md` (workaround = `mcpAccount`; leaks = `getKeyUsage`/`requestHistory`)

## Verification Log

Claims verified against HEAD during Rev 2 design review (2026-07-19):

| # | Claim | Method | Result |
|---|-------|--------|--------|
| V1 | `@raw` HTTP path passes a `RecordValue`, not a `Json` value | Read `routes_dispatch.go:51` (`args = []interface{}{buildHttpRequestRecord(...)}`) + `routes.go:341` (`buildHttpRequestRecord` returns `map[string]interface{}` → engine converts to `RecordValue`) | **Confirmed** — matches the reported `_json_encode: expected Json, got *eval.RecordValue`. |
| V2 | `@raw` envelope is a hybrid: `body`/`method`/`path` plain, `headers`/`query` are `JObject` | Read `routes.go:341-372` (`stringMapToJObject`) + `serve_api_webhook.ail:25` param type | **Confirmed** — retyping whole `req` to `Json` would break `request.body` record access. |
| V3 | MCP bridge has no `@raw` awareness; maps args straight to param | Read `mcp.go:188-254` (`makeToolHandler` takes only `modulePath, funcName, paramNames`) | **Confirmed** — no `IsRaw` threaded; reproduces both reported errors. |
| V4 | `@noexpose` hides from HTTP **and** MCP, and `@route` overrides it | Read `routes.go:106` (`IsNoExpose=false` on `@route`), `routes.go:228 isExposed`, `mcp.go:85` (`registerTools` calls `isExposed`) | **Confirmed** — no existing "HTTP-yes / MCP-no" path; gap is real. |
| V5 | Annotation parsing is a **closed allowlist** (so `@nomcp` needs a parser change) | Read `parser_decl.go:15-49` (`default:` returns unknown-attribute error at line 47) | **Confirmed** — `@nomcp` requires a new `case`, hence the Conflict Surface section. |
| V6 | `@nomcp` / `IsNoMCP` are **unallocated** (negative existence) | `grep -rin "nomcp\|no_mcp\|NoMCP\|no-mcp" internal/ cmd/ examples/ stdlib/` | **Confirmed empty** — free to allocate; no collision. |
| V7 | Exactly one `@raw` handler ships in-repo (regression fixture scope) | `grep -rln '@raw' --include='*.ail' .` → only `examples/runnable/serve_api_webhook.ail` | **Confirmed** — single fixture must stay green; conflict surface is small. |
| V8 | `ExportInfo` has singular `RouteMethod`/`RoutePath` plus `IsRaw`/`IsNoExpose`/`MCPName`, but no MCP-exclusion or positive raw-MCP field | Read `server.go:133-147` | **Confirmed** — route properties do exist, but only as one pair per exported function; add independent `IsNoMCP` and `IsMCP`. |
| V9 | The `extract*` annotation fns are invoked together in the module load path | `grep` → `module_entry.go:113-115` (`extractRouteAnnotations`/`extractNoExposeAnnotations`/`extractMCPNameAnnotations`) | **Confirmed** — `extractNoMCPAnnotations` wires in at the same site. |
| V10 | `extractRouteAnnotations` stores route data as one pair per export and also currently derives `IsRaw` inside the route branch | Read `routes.go:81-118` | **Confirmed** — it calls `fn.GetAnnotation("route")`, writes one `RouteMethod`/`RoutePath`, and sets `IsRaw` only after a route is found. M2 must add independent raw extraction and must not rely on these route fields. |
| V11 | Function annotations can contain zero or multiple `@route` entries, while `GetAnnotation` returns only the first | Read `ast/ast_decl.go:45-78` (`Annotations []*Annotation`; first-match loop) and parser annotation accumulation | **Confirmed** — current export metadata collapses route cardinality to at most one stored pair; there is no verified singular route identity suitable for MCP. |
| V12 | MCP tools are registered per exported function, not per route | Read `mcp.go:60-187`: candidate iteration is `for _, export := range modInfo.Exports`, dedup is `export.Name + "|" + export.Type`, and one `AddTool` occurs per candidate | **Confirmed** — the MCP envelope must be keyed by `funcName`; zero/one/multiple routes all use `method="MCP"`, `path="/mcp/tool/"+funcName`. |
| V13 | No positive `@mcp` annotation or `IsMCP` metadata exists at HEAD | `rg -n 'case "mcp"|GetAnnotation\("mcp"\)|IsMCP' internal/` | **Confirmed empty** — parameterless `@mcp` is available for the explicit raw-MCP authority grant without colliding with `@mcp_name`. |

---

**Document created**: 2026-07-18
**Last updated**: 2026-07-19 (Rev 2)

DESIGN_DOC_PATH: design_docs/planned/v0_30_0/m-serveapi-raw-handler-mcp.md
