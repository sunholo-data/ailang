# M-SERVEAPI-RAW-HANDLER-MCP: `@nomcp` exclusion + `@raw` handlers callable over MCP

**Status**: Planned
**Target**: v0.30.0
**Priority**: P1 (Medium) — unblocks docparse quota-hardening item 5; closes a live MCP capability leak
**Estimated**: 1.5 days (M1 ~0.5d, M2 ~1d)
**Dependencies**: None (builds on the existing `@route`/`@raw`/`@noexpose`/`@mcp_name` annotation + MCP bridge in `internal/apiserver/`)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No nondeterminism introduced. `@nomcp` is a static annotation; the `@raw`→MCP envelope synthesis is a pure deterministic transform of the tool arguments. |
| A2: Replayability | 0 | No impact on traces. |
| A3: Effect Legibility | 0 | Handler effect rows unchanged; no hidden effects added. |
| A4: Explicit Authority | +1 | `@nomcp` lets an author explicitly withhold a route from the agent-facing tool surface, tightening the authority boundary (a route can be HTTP-only). Today the MCP surface is all-or-nothing per exposed export. |
| A5: Bounded Verification | 0 | No type-system change. |
| A6: Safe Concurrency | 0 | No concurrency changes. |
| A7: Machines First | +1 | Removes two failure modes an agent hits when calling `@raw` tools (`record has no field: headers` / `cannot access field of non-record value`), and lets authors hide non-agent routes so the tool list an agent sees is smaller and correct. |
| A8: Minimal Syntax | 0 | Adds one parameterless annotation (`@nomcp`) in the same shape as the existing `@raw`/`@nowrap`/`@noexpose` — no new grammar class, no operator, no keyword. Net-neutral: it is the minimal form for the capability. |
| A9: Cost Visibility | 0 | No resource-cost surface change. |
| A10: Composability | +1 | `@nomcp` composes orthogonally with `@route`/`@raw`/`@nowrap` (HTTP contract) and fills the gap `@noexpose` cannot (which also hides HTTP and is overridden by `@route`). |
| A11: Structured Failure | +1 | M2 replaces two raw eval panics surfaced as opaque MCP tool errors with a well-formed request envelope, so `@raw` handlers return their normal typed `Result`/response instead. |
| A12: System Boundary | +1 | Makes the HTTP↔MCP boundary crossing explicit and correct for `@raw`: the MCP transport now supplies the same `{body, headers, method, path, query}` contract the HTTP transport does. |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted — narrows the tool surface, never widens it
- [x] A7 (Machines First): Directly reduces agent-facing failure modes

### Decision Thresholds

Net +6 ≥ +2, no −1 on A1/A3/A4/A7 → **✅ Proceed to implementation.**

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

**Problem B — `@raw` handlers are unusable over MCP.**
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

**Current State:**
- `@raw` MCP tools are auto-registered but **100% fail** on any natural call shape (Problem B).
- No annotation can keep a `@route` on HTTP while hiding it from MCP (Problem A).
- docparse item 5 is **blocked**; docparse shipped a `mcpAccount` workaround, but the leaky
  `getKeyUsage`/`requestHistory` MCP tools remain (per docparse note `docparse-raw-handler-mcp-limitation.md`).

**Impact:**
- Any AILANG project serving `@raw` webhook/signature handlers over `--mcp-http` exposes broken tools.
- Any project with operator-only `@route`s over `--mcp-http` leaks them to the agent surface with no
  opt-out — a security/authority concern, not just ergonomics.

## Goals

**Primary Goal:** Give AILANG authors (a) a way to keep a `@route` on HTTP but off the MCP tool surface,
and (b) `@raw` handlers that are actually callable over MCP through the same handler code that serves HTTP.

**Success Metrics:**
- A `@route @nomcp` handler is reachable over HTTP (200) and **absent** from the MCP `tools/list` and
  `tools/call` surface (M1).
- The two docparse leaks (`getKeyUsage`, `requestHistory`) are closable with a one-line annotation, with
  no HTTP behavior change (M1).
- A `@raw @route` handler invoked via an MCP `tools/call` with a natural argument object executes the
  same handler body successfully and returns its normal response — no `has no field` / `non-record`
  errors (M2).
- The canonical `examples/runnable/serve_api_webhook.ail` handler continues to type-check, run, and serve
  over HTTP **unchanged** (both milestones — no breaking change to the documented `@raw` record shape).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| `@nomcp` is a new **parameterless annotation** (not a `@route` option flag, not overloading `@noexpose`) | Sets the author-facing syntax + the annotation allowlist entry; parser + docs ripple | human | design | med |
| `@nomcp` hides from the **MCP tool surface only** (HTTP + OpenAPI + A2A agent card unaffected) | Defines exactly which surfaces the knob governs; wrong scope re-leaks or over-hides | human | design | med |
| M2 bridges `@raw` over MCP by **synthesizing the HTTP-request envelope** (body ← JSON-encoded tool args, headers/query ← empty `JObject`, method/path ← route) rather than retyping `req` to `Json` | Keeps the documented `{body,headers,method,path,query}` record contract → **no breaking change** to existing `@raw` handlers; the alternative (make HTTP emit `Json`) breaks `request.body` record access | agent (within this envelope shape) | design | high |
| `@raw` tools remain **auto-exposed to MCP by default** (opt out via `@nomcp`), rather than hidden by default | A default flip would silently drop tools projects may rely on; opt-out is the safe, reversible choice | human | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] `@nomcp` annotation name is ratified (vs. e.g. `@mcp(false)` / `@httponly`). **Recommended: `@nomcp`** — parallels `@noexpose`, parameterless, one allowlist entry.
- [ ] `@nomcp` scope confirmed = **MCP tool surface only** (HTTP, OpenAPI `/openapi.json`, and A2A `/.well-known/agent.json` remain unaffected).
- [ ] M2 envelope-synthesis contract confirmed: MCP `tools/call` arguments object → `req.body = encode(arguments)` (JSON string), `req.headers = JObject([])`, `req.query = JObject([])`, `req.method = route.Method`, `req.path = route.Path`.

## Solution Design

### Overview

Two independently shippable milestones against `internal/apiserver/` (+ one parser allowlist entry):

- **M1 — `@nomcp` exclusion annotation** (solves Problem A). New parameterless annotation that marks an
  export hidden from **MCP tool registration only**. Immediate unblock for the docparse leak.
- **M2 — `@raw` over MCP** (solves Problem B). Teach the MCP tool handler that a tool is `@raw`, and for
  those tools synthesize the same `{body,headers,method,path,query}` request envelope the HTTP path builds,
  so one handler body serves both transports with no retype and no breaking change.

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

4. **`@raw` MCP envelope synthesis (`internal/apiserver/mcp.go:makeToolHandler`, M2)** — thread the
   export's `IsRaw`, `RouteMethod`, `RoutePath` into the closure (currently only `modulePath, funcName,
   paramNames`). When `IsRaw`, bypass the named/positional/`args` binding and instead build the request
   record: reuse the **same** `buildHttpRequestRecord`-shaped constructor with `body = <MCP arguments
   marshalled to a JSON string>`, `headers = stringMapToJObject({})`, `query = stringMapToJObject({})`,
   `method = RouteMethod`, `path = RoutePath`; call the function with that single record arg. Extract the
   shared envelope builder so HTTP and MCP cannot drift.

### Implementation Plan

**Phase M1: `@nomcp` exclusion** (~4 hours)
- [ ] Parser: add `case "nomcp"` to `parseAnnotation`; update both hint strings; add a parser unit test
      (mirror the `@noexpose` test) asserting `@nomcp` parses and an unknown `@nope` still errors.
- [ ] `ExportInfo.IsNoMCP` field + `extractNoMCPAnnotations`; wire into the module load path.
- [ ] `registerTools`: `if export.IsNoMCP { continue }` guard.
- [ ] Tests: server-level test that a `@route @nomcp` export is present in HTTP routes + OpenAPI but
      absent from the MCP tool list; a `@nomcp` **without** `@route` still hidden from MCP (and, being a
      bare export, already hidden from routes-only HTTP — assert no interaction surprise).

**Phase M2: `@raw` over MCP** (~6 hours)
- [ ] Extract a shared `httpRequestEnvelope(body []byte, headers, query map[string][]string, method, path string)`
      constructor used by both `buildHttpRequestRecord` (HTTP) and the MCP path (guards against drift).
- [ ] Thread `IsRaw`/`RouteMethod`/`RoutePath` into `makeToolHandler`; when `IsRaw`, marshal
      `req.Params.Arguments` to a JSON string body and synthesize the envelope; call with the single record.
- [ ] `buildNamedInputSchema`: for `@raw` tools, emit a permissive object schema (free-form `object`,
      `additionalProperties: true`) instead of the `{req: {…record…}}` shape the agent cannot satisfy.
- [ ] Tests: `tools/call` on a `@raw` tool with `{"event":"test"}` reaches the handler; `request.body`
      decodes to that object; `getString(request.headers, …)` returns `None` (empty headers) without
      panicking; the HTTP path for the same handler is byte-for-byte unchanged.

**Phase 3: Docs + examples + changelog** (~2 hours)
- [ ] `ailang prompt`/CLI help + `prompts/` annotation list gains `@nomcp`.
- [ ] Extend `examples/runnable/serve_api_webhook.ail` (or a sibling) to show a `@nomcp` route and note the
      MCP-callable `@raw` behavior; keep it `verify-examples`-green.
- [ ] `CHANGELOG.md`; update the docparse note `docparse-raw-handler-mcp-limitation.md` to "resolved in v0.30.0".

### Files to Modify/Create

**Modified files:**
- `internal/parser/parser_decl.go` — `@nomcp` allowlist case + 2 hint strings (~6 LOC)
- `internal/apiserver/server.go` — `ExportInfo.IsNoMCP` field (~1 LOC)
- `internal/apiserver/routes.go` — `extractNoMCPAnnotations` + call-site wiring (~15 LOC)
- `internal/apiserver/mcp.go` — `registerTools` skip guard; `makeToolHandler` `@raw` envelope path; `buildNamedInputSchema` raw schema (~40 LOC, M1+M2)
- `internal/apiserver/routes_dispatch.go` — extract shared envelope constructor (~15 LOC, M2)
- `examples/runnable/serve_api_webhook.ail` — `@nomcp` demo + MCP note (~10 LOC)
- CLI help / `prompts/` — annotation list (~4 LOC)

**New files:**
- `internal/apiserver/nomcp_test.go` — M1 exclusion tests (~120 LOC)
- `internal/apiserver/mcp_raw_test.go` — M2 `@raw`-over-MCP tests (~150 LOC)

## Conflict Surface

**This design touches `internal/parser/` (annotation allowlist).** Required analysis:

1. **What positions does this extend?** The `@`-annotation position above an exported function
   declaration (parsed by `parseAnnotation`, consumed in the annotation loop at `parser_decl.go:213`).
   M2 additionally extends the MCP `tools/call` argument-binding position inside `makeToolHandler`.

2. **What else already lives in those positions?** The existing closed annotation allowlist:
   `@verify(...)`, `@route(...)`, `@mcp_name(...)`, `@raw`, `@nowrap`, `@noexpose`. `@nomcp` is a new
   sibling in the **same** parameterless family as `@raw`/`@nowrap`/`@noexpose`. The MCP argument-binding
   position currently holds three shapes: legacy `{"args":[...]}`, named params `{name: value}`, and the
   missing-param rejection (`M-MCP-UNIT-PARAM-BINDING`, `mcp.go:210`).

3. **How is it disambiguated?** The annotation is an exact-string `switch` case — `nomcp` cannot collide
   with any existing case, and unknown names still hard-error (the allowlist is preserved, not widened to
   accept-all). For M2, the `IsRaw` branch is taken **before** any of the three existing binding shapes, so
   it does not perturb non-`@raw` tools' argument handling.

4. **Programs that MUST still work post-change (regression fixtures):**
   - [`examples/runnable/serve_api_webhook.ail`](examples/runnable/serve_api_webhook.ail) — `@raw @route`
     with the `{body,headers,method,path}` record param: must type-check, `main()` must run, HTTP POST must
     serve unchanged. (M2 must not alter the HTTP envelope this handler receives.)
   - The same file's non-`@raw` `@route("GET","/health")` — must remain an MCP tool and HTTP route.
   - Any existing `@noexpose` export — must remain hidden from **both** HTTP and MCP (M1 must not entangle
     the two `Is*` flags).
   - Existing named-param and legacy-`args` MCP tool calls on **non-`@raw`** tools — unchanged (M2 gates
     solely on `IsRaw`).

5. **What deliberately changes (intentional incompatibilities):** For `@raw` tools **only**, the MCP
   `inputSchema` changes from the (unsatisfiable) `{req: {record}}` shape to a permissive free-form object,
   and the argument object is now routed into `req.body` rather than into `req` directly. This is the
   defect being fixed; no non-`@raw` tool's schema or binding changes. `@nomcp` on a `@route` deliberately
   does **not** get the `@route`-overrides-`@noexpose` treatment — it is an independent flag.

**The honest answer is not "no conflicts":** the real risk is (a) entangling `IsNoMCP` with the existing
`IsNoExpose`/`@route` override logic, and (b) M2's `IsRaw` branch accidentally changing the HTTP envelope
by refactoring `buildHttpRequestRecord`. Both are covered by the regression fixtures above.

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

### Example 2: `@raw` callable over MCP (Problem B)

**Before** (every natural MCP call fails):
```jsonc
// tools/call handleWebhook
{"req": {"args": ["key"]}}   // -> "record has no field: headers"
{"args": ["key"]}            // -> "cannot access field of non-record value"
```

**After** (same handler body, no retype):
```jsonc
// tools/call handleWebhook  { "event": "test" }
// serve-api synthesizes: req = { body: "{\"event\":\"test\"}", headers: JObject([]),
//                                method: "POST", path: "/webhooks/example", query: JObject([]) }
// handler runs: decode(request.body) => Ok({event:"test"}); getString(request.headers,"X-Signature") => None
// -> "received webhook with signature: no-signature body length: 16"
```
The HTTP `POST /webhooks/example` path is unchanged.

## Success Criteria

- [ ] `@nomcp` parses (parser unit test) and unknown annotations still hard-error.
- [ ] `@route @nomcp` export: HTTP 200 + present in OpenAPI, **absent** from MCP `tools/list`/`tools/call` (server test).
- [ ] `@noexpose` still hides from HTTP **and** MCP; the two flags do not interact (regression test).
- [ ] `@raw` MCP `tools/call` with a natural argument object executes the handler and returns its normal response (no `has no field`/`non-record` error) (M2 test).
- [ ] `examples/runnable/serve_api_webhook.ail` type-checks, runs via `verify-examples`, and its HTTP envelope is unchanged.
- [ ] All tests passing; `make ci` green.
- [ ] Docs (`ailang prompt`, CLI help, changelog) list `@nomcp`; docparse note marked resolved.

## Testing Strategy

**Unit tests:**
- Parser: `@nomcp` accepted; `@nope` rejected with the updated hint.
- `extractNoMCPAnnotations`: sets `IsNoMCP`; independent of `IsNoExpose`/`RoutePath`.
- M2 envelope builder: HTTP and MCP produce the identical record shape for equal inputs.

**Integration tests:**
- Boot a `Server` with a fixture module (`@route`, `@route @nomcp`, `@raw @route`); assert the MCP tool
  list, an MCP `tools/call` on the `@raw` tool, and the HTTP responses for all three.

**Manual testing:**
- `ailang serve-api --mcp-http examples/runnable/serve_api_webhook.ail`; `curl` HTTP + an MCP `tools/call`
  against `/mcp/`; confirm a `@nomcp` route is invisible to `tools/list` but answers over HTTP.

## Deferred Decisions

- Exact permissive `inputSchema` for `@raw` MCP tools (free-form `object` vs. surfacing the declared body
  sub-shape via a `body` property) — **agent may choose** the least-surprising schema that lets a natural
  argument object validate; default to free-form `object`.
- Whether the shared envelope constructor lives in `routes_dispatch.go` or a new `envelope.go` — **agent may
  choose** based on file-size targets.

## Non-Goals

- **Making HTTP `@raw` emit a `Json` value / retyping `req: Json`.** Rejected: it breaks the documented
  `request.body`/`request.method` record access in every existing `@raw` handler. M2 keeps the record shape.
- **Hiding `@raw` tools from MCP by default.** Rejected: a silent default flip could drop tools projects
  rely on; `@nomcp` is the opt-out.
- **A per-surface matrix annotation** (e.g. `@expose(http, a2a)` selecting arbitrary surface sets). Out of
  scope; `@nomcp` covers the concrete need. Revisit only if OpenAPI/A2A need independent hiding.
- **Auth/rate-limiting for `@raw` MCP tools.** Orthogonal; existing MCP rate-limiting applies.

## Timeline

**Day 1** (~6h): M1 (`@nomcp`) parser + extraction + registration filter + tests; start M2 envelope extraction.
**Day 2** (~6h): M2 `@raw`-over-MCP handler + schema + tests; docs, example, changelog; `make ci`.

**Total: ~1.5 days.**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| M2 refactor of `buildHttpRequestRecord` silently changes the HTTP `@raw` envelope | High | Shared constructor + a test asserting HTTP and MCP produce identical records; `serve_api_webhook.ail` regression fixture. |
| `IsNoMCP` entangled with `IsNoExpose`/`@route` override logic → re-leak or over-hide | Med | Keep flags independent; explicit test that `@noexpose` and `@nomcp` each hide only their intended surfaces. |
| Author confusion between `@noexpose` (HTTP+MCP) and `@nomcp` (MCP-only) | Low | Doc table + example contrasting the two; CLI hint lists both. |
| `@raw` tool argument marshalling loses type fidelity (int vs float in body JSON) | Low | Body is a JSON **string** the handler decodes itself, exactly as over HTTP — same fidelity as the real transport. |

## Related Documents

**Implemented (may inform design):**
- [design_docs/implemented/v0_11_0/m-mcp-quality-sprint-plan.md](design_docs/implemented/v0_11_0/m-mcp-quality-sprint-plan.md) (0.53) — MCP tool-name generation/validation this builds on. Distinct: that hardened names; this adds surface exclusion + `@raw` binding.
- [design_docs/implemented/v0_10_0/m-serve-api-agent-enhancements-sprint-plan.md](design_docs/implemented/v0_10_0/m-serve-api-agent-enhancements-sprint-plan.md) (0.48) — original `@route`/`@raw` serve-api surface.

**Planned (check for overlap):**
- [design_docs/planned/v0_29_0/m-serve-api-live-tool-registry.md](design_docs/planned/v0_29_0/m-serve-api-live-tool-registry.md) (0.49) — dynamic tool registry. Distinct: that governs *which modules* register; this governs *whether an exposed export* becomes an MCP tool + how `@raw` binds. Complementary, non-overlapping.

## References

- [Design Axioms](/docs/references/axioms)
- Canonical `@raw` example: [`examples/runnable/serve_api_webhook.ail`](examples/runnable/serve_api_webhook.ail)
- Code anchors: `internal/apiserver/mcp.go:registerTools`/`makeToolHandler`, `internal/apiserver/routes.go:extractNoExposeAnnotations`/`buildHttpRequestRecord`, `internal/apiserver/routes_dispatch.go:callFunction`, `internal/parser/parser_decl.go:parseAnnotation`
- docparse note: `docparse-raw-handler-mcp-limitation.md` (workaround = `mcpAccount`; leaks = `getKeyUsage`/`requestHistory`)

## Verification Log

Claims verified against the code at design time (v0.29.2 worktree):

| # | Claim | Method | Result |
|---|-------|--------|--------|
| V1 | `@raw` HTTP path passes a `RecordValue`, not a `Json` value | Read `routes_dispatch.go:51` (`args = []interface{}{buildHttpRequestRecord(...)}`) + `routes.go:341` (`buildHttpRequestRecord` returns `map[string]interface{}` → engine converts to `RecordValue`) | **Confirmed** — matches the reported `_json_encode: expected Json, got *eval.RecordValue`. |
| V2 | `@raw` envelope is a hybrid: `body`/`method`/`path` plain, `headers`/`query` are `JObject` | Read `routes.go:341-372` (`stringMapToJObject`) + `serve_api_webhook.ail:25` param type | **Confirmed** — retyping whole `req` to `Json` would break `request.body` record access. |
| V3 | MCP bridge has no `@raw` awareness; maps args straight to param | Read `mcp.go:188-254` (`makeToolHandler` takes only `modulePath, funcName, paramNames`) | **Confirmed** — no `IsRaw` threaded; reproduces both reported errors. |
| V4 | `@noexpose` hides from HTTP **and** MCP, and `@route` overrides it | Read `routes.go:106` (`IsNoExpose=false` on `@route`), `routes.go:228 isExposed`, `mcp.go:85` (`registerTools` calls `isExposed`) | **Confirmed** — no existing "HTTP-yes / MCP-no" path; gap is real. |
| V5 | Annotation parsing is a **closed allowlist** (so `@nomcp` needs a parser change) | Read `parser_decl.go:15-49` (`default:` returns unknown-attribute error at line 47) | **Confirmed** — `@nomcp` requires a new `case`, hence the Conflict Surface section. |
| V6 | `@nomcp` / `IsNoMCP` are **unallocated** (negative existence) | `grep -rin "nomcp\|no_mcp\|NoMCP\|no-mcp" internal/ cmd/ examples/ stdlib/` | **Confirmed empty** — free to allocate; no collision. |
| V7 | Exactly one `@raw` handler ships in-repo (regression fixture scope) | `grep -rln '@raw' --include='*.ail' .` → only `examples/runnable/serve_api_webhook.ail` | **Confirmed** — single fixture must stay green; conflict surface is small. |
| V8 | `ExportInfo` has `IsRaw`/`IsNoExpose`/`MCPName` but no MCP-exclusion field | Read `server.go:133-147` | **Confirmed** — add `IsNoMCP`. |
| V9 | The `extract*` annotation fns are invoked together in the module load path | `grep` → `module_entry.go:113-115` (`extractRouteAnnotations`/`extractNoExposeAnnotations`/`extractMCPNameAnnotations`) | **Confirmed** — `extractNoMCPAnnotations` wires in at the same site. |

---

**Document created**: 2026-07-18
**Last updated**: 2026-07-18

DESIGN_DOC_PATH: design_docs/planned/v0_30_0/m-serveapi-raw-handler-mcp.md
