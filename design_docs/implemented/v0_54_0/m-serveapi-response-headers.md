# serve-api Response Headers: Security Headers on `--static` + Hyphenated Route Headers

**Status**: Implemented; independent local evaluation passed (95/100); full suite pending coordinator PR CI
**Target**: v0.54.0 (next release; the `--static` security-header half is P0 and ships first — see Splitting below)
**Priority**: P0 (security #1597; silent-failure bug #1609)
**Estimated**: 1 week (Phase 1: 1 day · Phase 2: 2–3 days · Phase 3: 1 day · buffer)
**Dependencies**: None
**Refs**: #1597, #1609 (existing issues; triage docs `design_docs/planned/ailang-core-triage/serve-api-static-security-headers.md` and `design_docs/planned/ailang-core-triage/route-headers-hyphenated-names.md`, which asked for exactly this one design covering both)

**Scheduling**: approved by Mark, 2026-10-08, from the P0 issue triage. Defect verified live by triage on
origin/dev `658ff76a3`; re-verified live for this doc (see Verification Log) against `internal/apiserver/routes_dispatch.go`
at HEAD `62ac2d09` with an `ailang serve-api` binary (v0.52.5, commit `7200786`).

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Header application is order-independent (`Set` replaces per name); the Json form iterates the declared `JObject` list in program order. No new nondeterminism |
| A2: Replayability | 0 | No trace changes; headers are derived deterministically from the response value |
| A3: Effect Legibility | 0 | No effect-row changes; response shaping stays inside the serve-api boundary |
| A4: Explicit Authority | +1 | The operator explicitly owns `--static` response headers; a program explicitly declares route response headers; a dropped security header (believed sent, actually absent) is an authority failure this removes |
| A5: Bounded Verification | 0 | No verification surface change |
| A6: Safe Concurrency | 0 | Handler wrappers are per-request, as today |
| A7: Machines First | +1 | Fail-loudly converts a browser-silent wrong header name into a startup/dispatch error an agent can read and fix; the accepted `_headers` shapes are quoted in every refusal (the WS `wsReqShape` pattern) |
| A8: Minimal Syntax | +1 | No new syntax: reuses the existing record and `std/json` forms; the fix is remapping + acceptance, not new grammar |
| A9: Cost Visibility | 0 | No resource accounting changes |
| A10: Composability | +1 | The Json form composes with the existing `std/json` builders (`jo`/`kv`/`js`) already used for request headers |
| A11: Structured Failure | +1 | An unusable `_headers` value (wrong shape, non-string value, invalid header name) becomes a named error instead of a silent drop — the exact A11 ask in the operator-surface doc |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +5** → **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted (static defaults are operator-escapable by flag)
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

## Problem Statement

Two issues, one root: **serve-api has no way to put a security header on the wire.**

### 1. `--static` sends no security headers (#1597, security)

`serve-api --static` serves files through `staticCacheHandler(s.staticCache, http.FileServer(...))`
(`internal/apiserver/server.go:650`) and sets only `Cache-Control` — and only when `--static-cache` is
given. No `X-Frame-Options`, no `Content-Security-Policy` `frame-ancestors`, no `X-Content-Type-Options`,
no `Referrer-Policy`, anywhere in the static path (grep across `internal/`, `cmd/`, `serveapi/`: the only
`nosniff` in the tree is on MCP protocol envelopes, `serveapi/protocol/envelope.go:38` and
`serveapi/protocol/mcphttp/wire.go:67`). RFC 9700 §4.16 requires an authorization server to stop its
consent page being framed; the MCP OAuth package (`sunholo/mcp_oauth`, planned
`design_docs/planned/v0_52_0/m-mcp-oauth-package.md`) steers adopters to host their sign-in page on
`serve-api --static`, so those consent pages are framable. The published workaround
(`docs/docs/guides/mcp-connectors.md:333-340`) is a synchronous JS anti-framing script copied into every
page — defense the page must run itself, in content an attacker partially controls.

### 2. `@route` handlers cannot express hyphenated header names (#1609)

Record field labels in AILANG cannot contain `-` (verified: a hyphenated label in a record type is
`PAR_FIELD_NAME_EXPECTED`; in a literal it parses but cannot be given a declared type, so it cannot be
a typed handler return). So the documented `_headers` record form can only spell `x_frame_options`,
which the server sends as `X_frame_options` (Go canonicalises the first letter only) — a name browsers
ignore. The `Json` form can spell the name, but is silently dropped on one of the two dispatch paths.
Live matrix (fresh `serve-api`, four handlers — full transcripts in the Verification Log):

| Handler form | `@nowrap` path (`routes_dispatch.go:239-253`) | `_body` raw path (`writeRawResponse`, `routes_dispatch.go:317-330`) |
|---|---|---|
| `_headers: {x_frame_options: "DENY"}` (record) | `X_frame_options: DENY` — wrong name, browsers ignore it | `X_frame_options: DENY` — same wrong name |
| `_headers: Json` (`jo([kv("X-Frame-Options", js("DENY"))])`) | **`X-Frame-Options: DENY` — works today** (the `JObject` unwraps to `map[string]interface{}` in `embed.ToGo`, `internal/embed/convert.go:238-242,435`) | **silently dropped** — `writeRawResponse` only matches `*eval.RecordValue`, a `Json` is a `*eval.TaggedValue` |

This refines #1609's report: the Json form is dropped on the raw/binary path, not on `@nowrap`.
The outcome is the same — the only form that can spell a security header is unusable on the path that
serves HTML, and nothing anywhere tells the program its headers never went out.

Compounding both: **the guide's own examples do not compile.** `docs/docs/guides/serve-api.md:1166-1178`
and `:1371-1382` show `_headers = { "X-Request-Id" = ... }` — `=` in record literals is `PAR016`, and
string-keyed record fields cannot be declared in the return type. The documented mechanism has never
worked as documented. And **no test covers the real path**: `TestWriteRawResponse`
(`internal/apiserver/auth_test.go:122-125`) is an empty stub ("covered by integration tests" — there are
none for `_headers`), and `TestNowrapHeaders_ExtractsFromGoMap` (`named_args_test.go:491`) tests a
hand-copied re-implementation of the loop, not the production code.

**Impact:** every `serve-api` deployment (AI-generated services steered to this hosting) ships without
anti-framing/nosniff defaults; any service that needs `X-Frame-Options`, `Content-Security-Policy`,
`Referrer-Policy`, `ETag`-style or any hyphenated header on the raw path cannot send it and is not told.
RFC 9700 §4.16 non-compliance is a hard blocker for MCP directory submission of OAuth-backed services.

## Goals

**Primary Goal:** a `serve-api` operator can put the four RFC 9700-aligned security headers on every
`--static` response by default (flags only, zero program change), and an AILANG handler can send any
valid HTTP header name on both response paths — with every unusable `_headers` value reported loudly.

**Success Metrics:**
- `curl -sI` on a `--static` file shows `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`,
  `Content-Security-Policy: frame-ancestors 'none'`, `Referrer-Policy: no-referrer` with no flags given
- `--static-header "X-Frame-Options: SAMEORIGIN"` overrides exactly that default; `--no-static-security-headers`
  drops the default set; `--static-header` without `--static` is a startup error
- A handler returning `{_headers: {x_frame_options: "DENY", content_security_policy: "frame-ancestors 'none'"}}`
  puts `X-Frame-Options` and `Content-Security-Policy` on the wire on BOTH the `@nowrap` and `_body` paths
- A handler returning `{_headers: jo([kv("X-Frame-Options", js("DENY"))])}` does the same (exact names, no remap)
- An `_headers` value with a non-string field, a non-`JString` Json entry, or an invalid HTTP header name
  produces a named error (500 + log on dispatch; registration refusal for a declared bad type on `@route`)
  — never a silent drop
- Wire-level tests (real `httptest` through the server) for `X-Frame-Options` + `Content-Security-Policy`
  on both paths — the reporter's test ask
- The two broken guide examples are replaced with checked, runnable ones (extracted snippets `ailang check`ed)

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1. Static security headers are **on by default** (opt-out via `--no-static-security-headers`), not opt-in | Safe-by-default is the reporter's ask and the RFC 9700 driver; but it is a behavior change for any deployment that legitimately frames its own static HTML — must be a documented default + escape hatch, not an implementation detail | human — **ruled YES, Mark 2026-10-08** | design | med |
| D2. Default set = `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Content-Security-Policy: frame-ancestors 'none'`, `Referrer-Policy: no-referrer`, on every `--static` response incl. 404 | Choosing `DENY`/`'none'` over `SAMEORIGIN`/`'self'` blocks self-framing too; error pages can be framed and sniffed as much as content, so all statuses | human | design | low |
| D3. Override shape: repeatable `--static-header 'Name: value'`, last-wins per name, validated at startup (RFC 9110 token, non-empty value); no per-path map | One flag covers add + override + (with D1's kill switch) removal; per-path maps are YAGNI until a deployment asks | agent | design | low |
| D4. Record `_headers` labels are remapped `_`→`-`; Json `_headers` names are exact | A record label `x_frame_options` currently produces the wire name `X_frame_options` — remapping changes what an existing (if broken) program sends: a public-surface change. The Json form keeps exact-name control as the escape hatch for any name a record cannot spell | human — **ruled YES, Mark 2026-10-08** | design | med |
| D5. Extraction happens on the `eval.Value` **before** Go conversion, one shared helper for both dispatch sites | Post-`ToGo` the record and `JObject` forms are indistinguishable (`map[string]interface{}`), so D4's two rules cannot be applied; pre-`ToGo` mirrors `resultErrStatus` and fixes both sites with one mechanism | agent | design | med |
| D6. Unusable `_headers` fails loudly: dispatch-time 500 + ERROR log naming the accepted shapes; `@route` registration additionally refuses a declared `_headers` type that is neither a string-valued record nor `Json` (the `WSReqIssue` pattern) | Defines new startup/type-error behaviour; the silent drop is the bug class | agent | design | med |
| D7. Server-owned headers — `X-Elapsed-Ms`, every `Access-Control-*` header and `Vary`, plus the message-framing headers `Content-Length`, `Transfer-Encoding`, `Connection`, `Keep-Alive`, `Upgrade`, `Trailer`, `TE` and `Proxy-*` (review 2026-10-09) — are set last (or refused from `_headers`) on both paths, so a program cannot overwrite them; program `Content-Type` wins over the `_body` default, which requires setting that default **before** `WriteHeader` (it is not sent today — see Review notes 2026-10-08) | `@nowrap` today lets `_headers` overwrite the timing header (order: `:236-237` before `:239`); `corsWrap` sets the operator's CORS headers before the handler runs (`cors.go:28-37`), so program `_headers` can overwrite them today | agent | design | low |
| D8. `ailang mcp check` grows a framing probe: send a well-formed dummy authorization request to the discovered authorization endpoint and judge only a final 2xx HTML response; FAIL (anthropic/both targets) when neither `X-Frame-Options` nor `frame-ancestors` is present | New check semantics in a submission gate; what counts as FAIL vs WARN is a rules decision | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] D1: defaults-on for `--static` security headers — **RULED YES by Mark, 2026-10-08** (the one
      deployment-breaking default in this doc; escape hatches `--static-header` / `--no-static-security-headers`)
- [x] D4: `_`→`-` remap of record `_headers` labels, with the Json form as the exact-name escape hatch
      — **RULED YES by Mark, 2026-10-08** (public-surface change to the `_headers` record format introduced in
      `m-serve-api-agent-enhancements.md`)
- [x] D5: pre-Go-conversion extraction, one shared helper at both sites
- [x] D6: loud failure at dispatch + registration-time type refusal for `@route`
- [x] D2/D3/D7/D8 as specified above (agent-decidable, ratified by design review)

## Deferred Decisions

- Exact error message wording and error detail codes — implementer (quote the accepted shapes, follow the
  `wsReqShape` convention)
- Whether the static security wrapper composes with `staticCacheHandler` as a separate wrapper or one
  combined writer — implementer (both are correct if `Cache-Control` stays 2xx/304-only and security
  headers are all-status)
- Whether the `mcp check` framing probe runs as WARN under `--target openai` — implementer (FAIL under
  anthropic/both is fixed by D8)
- Whether to log the effective static header set in the startup banner and in what order — implementer
- Test fixture organisation (table-driven vs one test per case) — implementer

## Solution Design

### Overview

Two halves that share only the motivation — deliberately split so the security half can ship first:

- **Phase 1 (#1597)**: operator flags put security headers on every `--static` response. Pure
  serve-api/CLI change, no language surface touched, no program needs rewriting. Clones the
  `--static-cache` pattern (`static_cache.go`: parse-at-startup, `ResponseWriter` wrapper) exactly.
- **Phase 2 (#1609)**: the `_headers` response convention gets its two usable forms (record with `_`→`-`
  remap; `Json` with exact names), at both dispatch sites, with loud failures for unusable values.
- **Phase 3**: `ailang mcp check` framing probe + documentation (guide examples that currently do not
  compile, plus retiring the JS workaround note).

**Splitting (per scheduling instruction):** Phase 1 touches only `internal/apiserver/static_*.go`,
`server.go` and `cmd/ailang/serve_api.go` — zero overlap with Phase 2's `routes_dispatch.go`/`routes.go`.
Ship Phase 1 as its own PR and release it even if Phase 2 slips; Phase 2 + 3 follow as a second PR.

### Phase 1: `--static` security headers (#1597)

**New file `internal/apiserver/static_headers.go`** (~120 LOC), beside `static_cache.go`:

1. **Defaults.** An ordered slice, applied to every response from the `--static` file server:

   | Header | Value | Why |
   |---|---|---|
   | `X-Content-Type-Options` | `nosniff` | MIME sniffing turns a mislabeled upload into script execution |
   | `X-Frame-Options` | `DENY` | RFC 9700 §4.16: the consent page must not be framable; legacy-browser half of the belt-and-braces |
   | `Content-Security-Policy` | `frame-ancestors 'none'` | The modern half; ignored by legacy browsers that need `X-Frame-Options` |
   | `Referrer-Policy` | `no-referrer` | The authorize request's query string carries `client_id`, `state`, scopes; it must not leak to third-party links on the page |

   All statuses — a 404 can be framed and sniffed exactly like content. (Different rule than
   `Cache-Control`'s 2xx/304: caching a 404 is a correctness bug, framing it is a security bug.)

2. **`ParseStaticHeader(v string) (name, value string, err error)`** — splits on the first `":"`;
   trims; validates the name with the existing `isHTTPToken` (`ws_headers.go:67-68`, RFC 9110 token);
   rejects a name that is empty or a control character is in the value; error names both the flag and
   the expected `Name: value` form (fail loudly, principle 2).

3. **`--static-header` (repeatable, D3)** and **`--no-static-security-headers`** in
   `cmd/ailang/serve_api.go`, using the existing `multiFlag` type (`serve_api.go:20-21`,
   `--cors-origin` precedent). Effective header map = defaults, then operator entries last-wins per
   name — so `--static-header "X-Frame-Options: SAMEORIGIN"` overrides one default, and the kill
   switch removes the default set wholesale (explicit, greppable, no empty-value magic).

4. **Startup validation** (mirror `--static-cache`): `--static-header` or the kill switch without
   `--static` is an error. Banner logs the effective set (`server.go:746-751` area).

5. **`staticSecurityHandler(headers []staticHeader, next http.Handler)`** — wraps
   `http.FileServer`, composing under `staticCacheHandler` (or beside it; see Deferred). Sets every
   header before delegating, so they land on all statuses. Wire into `server.go:648-651`:
   `mux.Handle("/", staticCacheHandler(s.staticCache, staticSecurityHandler(s.staticHeaders, http.FileServer(...))))`.

   Not applied to: `--frontend` (Vite dev proxy), API/MCP/A2A routes (JSON surfaces; the MCP envelope
   already sets `nosniff`; `@route` handlers get their own control in Phase 2). The `--static`-only
   scope matches `--static-cache` exactly.

### Phase 2: hyphenated route response headers (#1609)

**New file `internal/apiserver/route_headers.go`** (~150 LOC):

1. **One shared helper** (D5), operating on the `eval.Value` **before** `embed.ToGo`, used by both sites:

   ```
   applyResponseHeaders(w http.ResponseWriter, v eval.Value) error
   ```

   Accepted forms (quoted in every error message, the `wsReqShape` convention):

   - **Record** (`*eval.RecordValue`), string fields only: label with `_`→`-` remapped (D4).
     `{x_frame_options: "DENY"}` → `X-Frame-Options: DENY`.
   - **`Json`** (`*eval.TaggedValue`, `std/json`): must be a `JObject` whose values are `JString`:
     names exact as written (Go's `Set` canonicalises case). `jo([kv("X-Frame-Options", js("DENY"))])`
     → `X-Frame-Options: DENY`. This is the escape hatch for any name, and it is already the
     request-side convention (`_headers: Json` binds exact canonicalised names —
     `docs/docs/guides/serve-api.md:419`).

   Every name must pass `isHTTPToken` — else a loud error, never an invalid header on the wire.
   Every value must be a string (`*eval.StringValue` / `JString`) — else a loud error.
   Every value must be free of CR, LF and other control characters (same rule as Phase 1's
   `ParseStaticHeader`) — else a loud error, never a silently stripped or split header.
   Names in the server-owned set (D7: `X-Elapsed-Ms`, `Access-Control-*`, `Vary`) are refused
   loudly or overwritten last — the sprint picks one and pins it with a wire test.
   **Review 2026-10-09:** the `_` → `-` label mapping also let a route set message-framing headers
   (`Transfer-Encoding: "identity, chunked"` put two TE headers on the wire; `Content-Length: "3"`
   truncated an 11-byte body). D7's refused set therefore also covers, case-insensitively,
   `Content-Length`, `Transfer-Encoding`, `Connection`, `Keep-Alive`, `Upgrade`, `Trailer`, `TE`
   and every `Proxy-*` name — refused loudly (structured 500 + ERROR log) on both the record and
   the Json/`@nowrap` paths.
   Loud error = the handler's response becomes a 500 with a structured error naming the accepted
   shapes and the offending field, plus an `[API]` ERROR log (the existing failure channel,
   `routes_dispatch.go:168-176`).

2. **`@nowrap` site** (`routes_dispatch.go:239-253`, replaced): after the `Result.Ok` unwrap and
   before `embed.ToGo` of the body, if the (record) result has a `_headers` field, call the helper on
   the raw value and strip the field from a shallow copy before body serialisation (the
   `resultErrStatus` strip pattern, `routes_dispatch.go:296-305`). `X-Elapsed-Ms` is set after
   program headers (D7 — today the program can overwrite it).

3. **`writeRawResponse` site** (`routes_dispatch.go:320-330`, replaced): same helper on
   `rec.Fields["_headers"]`. **Correction (review 2026-10-08):** today the `_body` default
   `Content-Type` is set *after* `w.WriteHeader(status)` (`routes_dispatch.go:341` vs `:347-359`), so
   it is never sent and Go sniffs the type instead — the "program `Content-Type` still wins over the
   `_body` default" behaviour is not current behaviour. Phase 2 moves the default-`Content-Type`
   selection (program value if set, else the `_body`-type default) **before** `WriteHeader`, pinned
   by a wire test per body type (bytes, string, JSON fallback). `X-Elapsed-Ms` and the other
   server-owned headers (D7) are set last, still before `WriteHeader`.

4. **Registration-time type check (D6)**: when reading `@route` annotations
   (`routes.go:93-131`), if the declared return type is a record with a `_headers` field whose
   declared type is neither a record of `string` fields nor `Json`, refuse the route with the
   accepted shapes (store as `RouteEntry`-adjacent issue like `WSReqIssue`/`WSReq` at
   `export_info.go:45-46`, surface like `extractWSReq` at `ws_headers.go:82`). This catches
   `{_headers: int}` and friends at startup. The catch-all `/api/` endpoints have no per-function
   registration — those are covered by the dispatch-time failure only (documented limitation, same
   as today's `_status` handling).

5. **What deliberately changes** (see Regression Surface):
   - `{x_frame_options: "DENY"}` sends `X-Frame-Options` instead of `X_frame_options`. The old wire
     name was browser-ignored; there is no working migration path to preserve. Anyone needing the
     literal underscore name uses the Json form.
   - `@nowrap` programs can no longer overwrite `X-Elapsed-Ms` via `_headers`.
   - Silently-ignored non-string `_headers` values (and the raw-path Json drop) become 500s. A
     program that "worked" while its headers silently never went out now fails loudly — that is the
     point (A11); the error names the fix.

### Phase 3: `ailang mcp check` framing probe + docs

1. **Probe (D8)**: `internal/mcpcheck` already discovers the authorization server
   (`checkAuthorizationServer`, `mcpcheck.go:298`) and fetches well-known metadata. After that, GET
   the authorization endpoint with a **well-formed dummy authorization request**
   (`response_type=code`, a dummy `client_id`, `redirect_uri`, `state`, and a PKCE
   `code_challenge`/`code_challenge_method=S256`) and follow redirects — the final HTML page is what
   gets framed. A bare GET earns a 400 without security headers and would raise false FAILs, so the
   probe judges **only a final 2xx HTML response**; any other outcome is a WARN naming the status
   seen, never a FAIL. When that final 2xx HTML response carries neither `X-Frame-Options` nor a `Content-Security-Policy` with a
   `frame-ancestors` directive, emit a finding: FAIL under `--target anthropic|both` (RFC 9700 is
   the connector rule), WARN under `openai` (Deferred). Unreachable/`--target`-less behaviour
   unchanged; a server without OAuth metadata skips the probe silently (nothing to frame).
2. **Guide fixes**: `docs/docs/guides/serve-api.md` — replace the two non-compiling `_headers`
   examples (`:1166-1178`, `:1371-1382`, verified PAR016) with the two accepted forms from this doc;
   add the `--static` security-headers section (flags, defaults, escape hatches).
   `docs/docs/guides/mcp-connectors.md:333-340` — replace "Copy that pattern until #1597 is fixed"
   with the flags (keep the JS pattern as a note for pinned old versions).
   `changelogs/v0.32-current.md` entries for both halves.

### Files to Modify/Create

**New files:**
- `internal/apiserver/static_headers.go` (~120 LOC) — defaults, `ParseStaticHeader`, `staticSecurityHandler`
- `internal/apiserver/route_headers.go` (~150 LOC) — `applyResponseHeaders`, accepted-shape errors, `_`→`-` remap, token validation
- `internal/apiserver/static_headers_test.go` (~150 LOC) — parse table, all-status application, override/kill-switch, without-`--static` error
- `internal/apiserver/route_headers_test.go` (~220 LOC) — the live matrix of this doc as wire-level tests: both forms × both paths, loud failures, remap table, token refusal

**Modified files:**
- `cmd/ailang/serve_api.go` (+30) — `--static-header` (repeatable `multiFlag`), `--no-static-security-headers`, validation, help text (`:230` block)
- `internal/apiserver/server.go` (+15) — `Config.StaticHeaders`/`NoStaticSecurityHeaders`, wire wrapper at `:648-651`, banner at `:746-751`
- `internal/apiserver/routes_dispatch.go` (+25/−25) — both sites call `applyResponseHeaders`; `@nowrap` extraction moved pre-`ToGo`; `X-Elapsed-Ms` set last
- `internal/apiserver/routes.go` (+20) — declared-`_headers`-type check at `@route` registration (`WSReqIssue` pattern)
- `internal/apiserver/export_info.go` (+4) — carry the `_headers` type issue
- `internal/mcpcheck/mcpcheck.go` (+60) — framing probe after authorization-server discovery
- `docs/docs/guides/serve-api.md`, `docs/docs/guides/mcp-connectors.md`, `changelogs/v0.32-current.md` — Phase 3

## Examples

**Operator, #1597 (Phase 1) — flags only:**

```bash
# Defaults on, nothing to do:
ailang serve-api --mcp-http --routes-only --static /app/static docparse_api/
# curl -sI localhost:8080/oauth-login.html
#   X-Content-Type-Options: nosniff
#   X-Frame-Options: DENY
#   Content-Security-Policy: frame-ancestors 'none'
#   Referrer-Policy: no-referrer

# An app that frames its own static page:
ailang serve-api --static ./dist --static-header "X-Frame-Options: SAMEORIGIN" app.ail

# Escape the defaults wholesale:
ailang serve-api --static ./dist --no-static-security-headers app.ail
```

**Handler, #1609 (Phase 2) — both forms, both paths:**

```ailang
module app/pages
import std/json (Json, jo, kv, js)

-- Record form: labels remap '_' -> '-'. The only form that needs no import.
@route("GET", "/page")
export func page() -> {_body: string, _status: int, _headers: {x_frame_options: string, content_security_policy: string}} ! {IO} =
  {_body: "<html>ok</html>", _status: 200,
   _headers: {x_frame_options: "DENY", content_security_policy: "frame-ancestors 'none'"}}

-- Json form: exact names, arbitrary spellings (the escape hatch).
@nowrap
@route("GET", "/data")
export func data() -> {items: [string], _headers: Json} ! {IO} =
  {items: [],
   _headers: jo([kv("X-Frame-Options", js("DENY")), kv("X-RateLimit-Remaining", js("99"))])}
```

Wire result of both: `X-Frame-Options: DENY` (and the CSP) on the response — verified end-to-end today
for the `@nowrap`+Json combination (it works by accident of `ToGo`); the other three cells of the
matrix are what this design fixes.

**Loud failure (Phase 2):**

```ailang
-- Registration refusal (declared type is neither string-record nor Json), for @route handlers:
--   ERROR: _headers must be {x_y: string, ...} (labels remapped _ -> -) or Json (JObject of strings);
--   field x_retry_after is declared int
@route("POST", "/bad")
export func bad() -> {_body: string, _headers: {x_retry_after: int}} ! {IO} = ...

-- Dispatch-time 500 + ERROR log (value shape does not match an accepted form, any handler):
--   _headers JObject entry "retry-after" is JNumber, not JString
export func worse() -> {_headers: Json} ! {IO} = {_headers: jo([kv("retry-after", jnum(5.0))])}
```

## Success Criteria

- [ ] The Goals metrics verified with a fresh binary from this worktree (evidence in the PR)
- [ ] Phase 1 mergeable and shippable independently of Phase 2 (self-contained first commits on one branch, per approved sprint plan)
- [ ] Wire-level tests: `X-Frame-Options` + `Content-Security-Policy` on both dispatch paths, both forms
- [ ] Wire test: the `_body` default `Content-Type` is actually sent (set before `WriteHeader`), and a program `Content-Type` wins
- [ ] Wire test: program `_headers` cannot overwrite `X-Elapsed-Ms`, `Access-Control-*` or `Vary`
- [ ] A route header value containing CR/LF or another control character fails loudly
- [ ] Every loud-failure mode has a test (non-string value, non-JObject Json, invalid token, declared-bad type at registration)
- [ ] Mutation-tested: revert the remap / the raw-path Json acceptance, watch tests fail
- [ ] Regression fixtures (below) still pass unmodified
- [ ] `make test`, `make lint`, `make check-file-sizes`, `make check-cli-docs`, `make check-changelog`
- [ ] Guide examples replaced and every AILANG snippet `ailang check`ed (the extraction gate the docs already run)

## Regression Surface

This design does not touch `internal/parser`, `internal/types`, `internal/eval` or any pipeline pass —
all changes live in `internal/apiserver`, `cmd/ailang`, `internal/mcpcheck` and docs. But it changes a
public surface (the `_headers` convention), so the equivalent analysis is required:

**Positions touched:** the `_headers` field of a handler's return value, consumed at exactly two sites
(`routes_dispatch.go:239-253` and `:317-330`). Request-side `_headers` (the `Json` parameter bound from
request headers, `routes_dispatch.go:115-124`) is a different, untouched mechanism that shares only the
name.

**Programs that MUST still work (regression fixtures):**
1. `examples/runnable/mcp_tools.ail` — request-side `_headers: Json` param (`secureParse`)
2. `examples/runnable/serve_api_mcp_header_auth.ail` — request-side `_headers: Json` (`whoami`)
3. `internal/apiserver/mcp_header_auth_test.go` — the full request-side suite incl. the forging guard
4. `internal/apiserver/routes_test.go` `TestNowrapResponse`/`TestNonNowrapResponse` — `@nowrap` without `_headers`
5. The `@nowrap`+Json response path that works today (Verification Log row 5) — pinned as a wire test

**What deliberately changes:** listed in Solution Design, Phase 2 item 5 (remap, `X-Elapsed-Ms` ownership,
silent drops becoming 500s). Anything outside that list that breaks is a regression, not a change.

## Testing Strategy

**Unit:** `ParseStaticHeader` table (valid, missing colon, invalid token, control char, empty value);
effective-header-map construction (defaults, override, kill switch, both flags without `--static`);
`applyResponseHeaders` table (record remap, Json exact, non-string value, non-JObject, invalid token);
declared-type refusal table at registration.

**Integration (wire-level, the reporter's ask):** through `httptest` + the real server: the live matrix
of this doc (both forms × `@nowrap` × `_body`), `X-Frame-Options`/`CSP` on a `--static` file incl. a
404, override + kill switch on the wire, `X-Elapsed-Ms` not overwritable, request-side `_headers`
fixtures untouched.

**Regression:** one test per fixture above, pinning current behaviour.

**Manual (DONE WHEN):** `curl -sI` against a fresh binary for each Goals metric; `ailang mcp check`
against a local serve-api with/without the defaults.

## Non-Goals

- Per-path static header maps (`--static-header /admin/*:...`) — YAGNI until a deployment asks
- Security-header defaults on API routes, MCP, A2A, or `--frontend` — routes get program control (Phase 2); MCP envelopes already set `nosniff`; Vite proxy is dev-only
- An operator allowlist for **response** headers set by programs — the documented feature is program-chosen response headers; narrowing it is the operator-surface doc's separate open question
- New record-label syntax (hyphens in labels) — the Json form already spells any name; remap covers the rest
- `--static-cache` gaining more forms — unchanged
- Removing the JS anti-framing workaround from deployed pages — theirs to retire

## Timeline

**Day 1** — Phase 1: `static_headers.go` + flags + wiring + tests (shippable PR #1, closes #1597)
**Days 2–4** — Phase 2: `route_headers.go`, both dispatch sites, registration check, the wire-level matrix (PR #2, closes #1609)
**Day 5** — Phase 3: `mcp check` probe + guide/changelog
**Day 6** — Buffer: mutation tests, `make ci-quick`, review fixes

**Total: ~6 working days** (estimates doubled from instinct, per house rule)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Defaults break a deployment that frames its own static HTML | Medium | `--static-header` per-name override + `--no-static-security-headers` kill switch; CHANGELOG entry names the flags; D1 frozen at design review |
| `_`→`-` remap changes what an existing program sends | Medium | The old wire name was browser-ignored (verified); Json form preserves exact names; D4 frozen at design review; deliberate-change list |
| `X-Content-Type-Options: nosniff` breaks a mislabeled static file that relied on sniffing | Low | That reliance is the vulnerability; per-name override exists |
| Moving `@nowrap` extraction pre-`ToGo` misses a wrapper (`Result.Ok`, tagged) the post-Go path handled by accident | Medium | Extraction placed after the existing Ok-unwrap; the wire matrix pins all four cells |
| `mcp check` probe false-FAILs on an AS behind a bot-challenging edge | Medium | Probe failure is `WARN`-downgradable; findings quote the response headers seen; `--target` composition documented |
| One more wrapper on the static path costs latency | Low | Two map writes per response; no measurable cost in bench |

## Review notes 2026-10-08

**Maintainer rulings (Mark, 2026-10-08):** D1 = YES (static security headers on by default);
D4 = YES (record `_headers` labels remap `_`→`-`). Both ticked in the Design Freeze.

Review findings, folded into the sections above:

1. **`_body` default `Content-Type` is never sent.** `writeRawResponse` calls `w.WriteHeader(status)`
   (`internal/apiserver/routes_dispatch.go:341`) before setting the default `Content-Type`
   (`:347-359`); header writes after `WriteHeader` are ignored, so Go's sniffer picks the type. The
   doc's earlier "program `Content-Type` still wins over the `_body` default (current behaviour)"
   claim is corrected in Phase 2 item 3; Phase 2 sets defaults before `WriteHeader`, pinned by a wire test.
2. **Program `_headers` can overwrite the operator's CORS headers.** `corsWrap` applies the origin
   policy before the handler runs (`internal/apiserver/cors.go:28-37`), and both `_headers` sites then
   `Set` whatever the program returns. D7's server-owned set is extended from `X-Elapsed-Ms` to every
   `Access-Control-*` header and `Vary`.
3. **D8 framing probe must not judge a bare GET.** A GET of the authorization endpoint without
   parameters gets a 400 without security headers on most servers — false FAILs. The probe sends a
   well-formed dummy authorization request and judges only a final 2xx HTML response (Phase 3 item 1).
4. **Route header values reject CR/LF and other control characters loudly**, the same rule Phase 1
   applies to `--static-header` (Phase 2 item 1).
5. **Line refs corrected:** `isHTTPToken` is at `ws_headers.go:67-68`; `multiFlag`/`--cors-origin`
   at `cmd/ailang/serve_api.go:20-21`; `TestNowrapHeaders_ExtractsFromGoMap` at `named_args_test.go:491`.

## Verification Log

Every load-bearing claim in this doc, checked against the code or a live run (2026-10-08, HEAD `62ac2d09`,
binary `ailang` v0.52.5 commit `7200786`; the four live outcomes follow line-by-line from
`routes_dispatch.go` + `internal/embed/convert.go` at HEAD, so they hold for the source under design):

| # | Claim | Method | Result |
|---|-------|--------|--------|
| 1 | Record **type** labels cannot contain `-` | `ailang check` on `{_headers: {"X-Frame-Options": string}}` | `PAR_FIELD_NAME_EXPECTED` — Confirmed |
| 2 | Record **literal** string keys parse but cannot be typed into a handler return | `ailang check` on `-> {_headers: {string: string}} = {_headers: {"X-Request-Id": "abc"}}` | type error: field `'X-Request-Id'` not found — Confirmed (usable neither way) |
| 3 | `{x_frame_options: "DENY"}` compiles | `ailang check` | ✓ No errors — Confirmed |
| 4 | Json `_headers` construction compiles (`jo`/`kv`/`js`) | `ailang check` on `{_headers: jo([kv("X-Frame-Options", js("DENY"))]), data: "ok"}` | ✓ No errors — Confirmed |
| 5 | Live matrix (record/Json × @nowrap/raw), 4 handlers, `@route`+`@nowrap` on a real `serve-api`, `curl -i` | full transcripts in this doc's Problem Statement | rec→`X_frame_options`, nowrap-Json→`X-Frame-Options`+CSP correct, raw-Json→dropped, raw-rec→`X_frame_options` — Confirmed (corrects #1609's "dropped on @nowrap": only the raw path drops) |
| 6 | Static path sets no security headers; only `Cache-Control` when flagged | grep `nosniff|X-Frame-Options|frame-ancestors|X-Content-Type-Options|Referrer-Policy` over `internal/ cmd/ serveapi/` (non-test) | hits only in `serveapi/protocol/{envelope.go:38, mcphttp/wire.go:67}` (MCP envelopes, not static) — Confirmed |
| 7 | No CLI flag exists for static response headers | `ailang serve-api --help` + `serve_api.go` flag list | no such flag; `--static-header` unallocated — Confirmed |
| 8 | `writeRawResponse` has no `_headers` test | read `auth_test.go:122-125` | empty stub, comment only — Confirmed |
| 9 | `TestNowrapHeaders_ExtractsFromGoMap` tests a re-implementation, not production | read `named_args_test.go:491` | loop hand-copied in the test body — Confirmed |
| 10 | Guide `_headers` examples do not compile | `ailang check` of their syntax (`=` in record literal; string-keyed fields) | `PAR016` (row 3's sibling) + row 2 — Confirmed |
| 11 | `isHTTPToken` exists for name validation | read `ws_headers.go:67-68` | reusable, unexported, same package — Confirmed |
| 12 | `FuncDecl.ReturnType` is available at route registration for the declared-type check; AST shape validation has precedent | read `ast/ast_decl.go:49`, `routes.go:93-131` (WS precedent `extractWSReq` at `ws_headers.go:82` reading `fn.Params[1].Type`) | Confirmed |
| 13 | Repeatable-flag precedent exists (`multiFlag`) | read `serve_api.go:20-21` (`--cors-origin`) | Confirmed |
| 14 | `--static-cache` wrapper pattern to clone | read `static_cache.go` (parse, handler, writer, `Unwrap`) | Confirmed |
| 15 | `mcp check` already discovers the authorization server and fetches well-known | read `internal/mcpcheck/mcpcheck.go:298-333` (`checkAuthorizationServer`, `wellKnown`, `getJSON`) | Confirmed — probe hook point exists |
| 16 | Request-side `_headers: Json` is documented and exemplified (untouched surface) | read `docs/docs/guides/serve-api.md:419,1095-1113`, `examples/runnable/mcp_tools.ail:22`, `serve_api_mcp_header_auth.ail:34` | Confirmed |
| 17 | No new error codes proposed | this doc proposes plain startup errors / 500 + log, no `PARxxx`/`TCxxx`/`MODxxx` allocations | n/a — no grep needed |
| 18 | Historical design target was v0.53.0 while the installed binary was v0.52.5. Execution found std/VERSION v0.53.0 and retargeted to v0.54.0 (see execution target update). | std/VERSION and execution checkout | Retargeted |

## References

- **Triage (this doc's mandate):** `design_docs/planned/ailang-core-triage/route-headers-hyphenated-names.md` (#1609), `design_docs/planned/ailang-core-triage/serve-api-static-security-headers.md` (#1597) — both recommend one design covering both; this is it
- **Prior art for `_headers`:** `design_docs/implemented/v0_10_0/m-serve-api-agent-enhancements.md` (introduced the `_body`/`_status`/`_headers` pattern and the `@nowrap` `_headers` field; cannot rule on its own format — superseded here on names)
- **Pattern to clone:** `design_docs/implemented/v0_46_0/m-serveapi-operator-surface.md` (D6 `--static-cache`, D7 header semantics, fail-loudly axiom A11, `wsReqShape` quoting)
- **Consumer:** `design_docs/planned/v0_52_0/m-mcp-oauth-package.md` (OAuth surface this unblocks); `docs/docs/guides/mcp-connectors.md` (the workaround to retire)
- **RFC 9700** §4.16 (framing of authorization endpoints); RFC 9110 §5.1 (token = valid header name), §12.5.4 (`X-Content-Type-Options`)
- **Issues:** Refs #1597, Refs #1609

## Future Work

- Adopting `passHeaders`-style operator allowlists for **response** headers, if an operator ever needs
  to cap what programs may set (the operator-surface doc's open question)
- A `--route-security-headers` default for `@route` responses, if deployments want the Phase 1 defaults
  without per-handler code (YAGNI: handlers can now set them)
- `frame-ancestors` with origins (not just `'none'`) behind a flag, if a multi-frame portal asks

## Execution target update (2026-10-09)

`std/VERSION` is now v0.53.0. Retargeted this design and its approved sprint to v0.54.0. The approved sprint supersedes the earlier separate-PR scheduling: M1 is the self-contained first commit, available for cherry-picking; the coordinator raises the implementation PR. Refs #1597. Refs #1609.

## Implementation evidence

Implemented in self-contained static-first commits on `coordinator/task-62c68b6a`. See the companion sprint plan and `docs/sprint-retros/M-SERVEAPI-RESPONSE-HEADERS-retro.md` for wire evidence, mutation checks and explicit baseline/CI limitations.
