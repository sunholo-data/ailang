# M-SERVEAPI-OPERATOR-SURFACE: what serve-api exposes is the operator's choice

**Status**: Planned
**Target**: v0.45.0
**Priority**: P1 (a downstream consumer, Daneel's tailnet-only talk page, is blocked on two items and works around two)
**Estimated**: 1.5 days
**Dependencies**: M-SERVEAPI-WS-BRIDGE (v0.44.0, `@route("WS")`), M-STDLIB-ROOT-RESOLUTION (v0.44.x)
**Source**: four Daneel requests, 2026-09-26 (daneel `design_docs/planned/m-daneel-live-page.md`, "AILANG requests arising")

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No evaluation change. The base path becomes a pure function of the command line and the module headers (it was already deterministic, only wrong) |
| A2: Replayability | 0 | No trace-format change; the header record is an ordinary argument value |
| A3: Effect Legibility | 0 | No effect-row change |
| A4: Explicit Authority | +1 | Request headers reach a WS program only when the operator names them; credential headers can never be named. Introspection can be switched off. `std/*` stops being presented as callable surface |
| A5: Bounded Verification | 0 | — |
| A6: Safe Concurrency | 0 | — |
| A7: Machines First | +1 | The startup banner and `/api/_meta/modules` list only what is actually callable — no more 23 endpoints that answer LDR001 |
| A8: Minimal Syntax | 0 | No syntax; flags only |
| A9: Cost Visibility | 0 | — |
| A10: Composability | +1 | The header record reuses the existing WS `req` record, built field-for-field from the handler's declared type |
| A11: Structured Failure | +1 | A forbidden `--ws-pass-header`, a malformed `--static-cache`, or a `headers` field of the wrong type is a startup error naming the fix — never a silent drop |
| A12: System Boundary | +1 | The page-facing boundary (introspection, CDN-loading doc pages, cache policy, which headers cross into the program) becomes explicit and operator-set |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted — headers are opt-in per name, credentials are unnameable
- [x] A7 (Machines First): The change removes misleading machine-read surface

## Problem Statement

Daneel serves a talk page on its tailnet with `serve-api --routes-only --static … tools/daneel_serve.ail`. By design it should
expose exactly `/healthz`, `/live` (a `@route("WS")`) and static files. Four gaps, all measured by Daneel on v0.44.0:

1. **Introspection ignores `--routes-only`.** `/api/_meta/modules` lists *every* export — name, type, doc comment — including
   the ones `--routes-only` hides from dispatch; `/api/_meta/openapi.json`, `/api/_meta/docs`, `/api/_meta/redoc` and
   `/api/_health` stay up. The two doc pages load Google Fonts, unpkg and cdn.redoc.ly — external fetches from a page that is
   otherwise tailnet-only.
2. **`std/*` listed as endpoints; the base path is guessed from the wrong file.**
   (a) Without `--routes-only`, a module importing `std/stream` shows **23 `std/*` endpoints** in the banner
   (`POST /api/<embedded>/std/stream/asyncExecProcess` …). None is callable (LDR001).
   (b) `serve-api .` in a directory holding `client/wsclient.ail` took `client/` as its base path and **refused to start**
   ("1 @route-bearing module(s) dropped by under-basePath filter").
3. **No cache policy for `--static`.** Responses carry `Last-Modified` and ranges, no `Cache-Control`. Daneel's media files
   are dated and never rewritten.
4. **A WS handler cannot see who is calling.** `req` is `{path, query, origin}`. Tailscale serve stamps
   `Tailscale-User-Login` on every proxied request (and overwrites a forged one — measured by Daneel).

### Root causes (read from the code, not inferred)

| Gap | Cause | Where |
|-----|-------|-------|
| 1 | `buildRoutes` registers the meta routes unconditionally; `handleListModules`/`handleModuleDetail` serialize `ModuleInfo.Exports` raw, never through `loadedExportMember` — the one surface gateway every other enumeration (OpenAPI, MCP, A2A, banner, dispatch) already uses | `internal/apiserver/server.go` `buildRoutes`; `internal/apiserver/meta.go` |
| 2a | `registerModule` decides "local" by `filepath.Abs(loaded.File.Path)` having the base-path prefix. The embedded stdlib's display path is the synthetic `<embedded>/std/<name>.ail` (`stdlibroot.Root.DisplayPath`); `filepath.Abs` makes it `<cwd>/<embedded>/std/…`, which is under the base path whenever base = cwd. The same happens for an on-disk stdlib root under the project (the `./std` tier) | `internal/apiserver/module_entry.go:32`; `internal/stdlibroot/root.go:56` |
| 2a' | Off-base stdlib modules (on-disk root elsewhere) are recorded as *drops*, so `/api/_health` lists the resolved stdlib path of every imported `std/*` module | `module_entry.go` `recordDrop` call |
| 2b | `findFirstModuleDecl` walks the directory, takes the **first** `.ail` file in lexical order (`client/wsclient.ail` sorts before `talk.ail`), and trusts its `module` header to name the root — even when that root is *inside* the directory the operator gave | `cmd/ailang/serve_api.go:200` |
| 3 | `mux.Handle("/", http.FileServer(...))` with no header hook | `server.go` `buildRoutes` |
| 4 | `serveWSSession` builds `req` with three fixed fields | `internal/apiserver/routes_ws.go:205` |

## Goals

**Primary Goal:** every surface serve-api presents — routes, introspection, headers passed to a program, cache policy — is
one the operator chose, and what is listed is what is callable.

**Success Metrics (each is a DONE WHEN, verified with a fresh binary):**
- `serve-api --routes-only --no-introspection …`: `curl` on each of `/api/_meta/modules`, `/api/_meta/openapi.json`,
  `/api/_meta/docs`, `/api/_meta/redoc`, `/api/_health` → **404**; the `@route` endpoints and `--static` files still answer.
- `serve-api --routes-only …` (no new flag): `/api/_meta/modules` lists **only** `@route` exports (docparse keeps its docs).
- A module importing `std/stream`, served without `--routes-only`: the banner lists **0** `std/*` endpoints; `/api/_health`
  lists no `std/*` drop.
- `serve-api .` in a directory holding `main.ail` (`module main`, with an `@route`) and `client/wsclient.ail`
  (`module wsclient`): **starts**, and the route answers.
- `curl -sI` on a `--static` file: `Cache-Control: public, max-age=31536000, immutable` with `--static-cache immutable`,
  and **no** `Cache-Control` without the flag; a 404 never carries it.
- A WS client sending `Tailscale-User-Login: a@b` and `Cookie: CANARY…` to a handler under `--ws-pass-header Tailscale-User-Login`:
  the handler sees `headers == [{name: "tailscale-user-login", value: "a@b"}]`; the canary appears in none of: the handler's
  view, the deep trace, the server log.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1. `--routes-only` does **not** turn off introspection; a new `--no-introspection` does | Daneel offered either. docparse — our largest serve-api consumer — runs `--routes-only` in production **and** publishes `/api/_meta/docs` + `openapi.json` (its `deploy.sh:97-98`, `tests/test_serve_api.sh:174-179`). Folding it into `--routes-only` breaks docparse | agent (recorded for Mark) | design | low |
| D2. `/api/_meta/modules` and `/api/_meta/modules/{path}` go through `loadedExportMember` | They were the only enumeration that bypassed the gateway (`.claude/rules/api-server.md`: "discovery and invocation must consume the same authorized surface"). This is a fix, not a new policy | agent | design | low |
| D3. **Do not** make `--routes-only` the default when any `@route` is present | Breaking: `examples/runnable/mcp_tools.ail` mixes `@route` with plain exports (`escapeXml`) that are served today; a silent narrowing would 404 them. The concrete harm Daneel measured — 23 uncallable `std/*` listings — is removed by D4 on its own. Instead the banner prints one hint line when `@route` exports and auto-exposed exports coexist | agent (recommendation; Mark may overrule) | design | med if flipped later |
| D4. A stdlib module is never a local module | One rule (canonical or declared module path under `std/`) replaces the path-prefix accident; stdlib modules are neither registered nor recorded as drops | agent | design | low |
| D5. Base path: the directory given on the command line is the floor; module headers may only move it **outward** | Honours Daneel's ask while keeping `serve-api ./api/` (files declaring `module api/…`) working, which the help text documents | agent | design | med |
| D6. `--static-cache immutable\|SECONDS` | One flag, two forms; 2xx/304 only | agent | design | low |
| D7. `--ws-pass-header NAME` (repeatable), WS only; credential headers unnameable; the record is built from the handler's declared fields | See §Header semantics | agent (HTTP question recorded for Mark) | design | med |

### Design Freeze

- [x] D1 `--no-introspection` is a separate flag — measured docparse dependency
- [x] D3 no default flip — measured example dependency; open question for Mark recorded below
- [x] D5 base-path rule
- [x] D7 header semantics, including "HTTP unchanged in this milestone"

## Solution Design

### 1. `--no-introspection` (+ gateway fix)

- `Config.NoIntrospection bool`; `buildRoutes` registers `/api/_meta/*` and `/api/_health` only when it is false. The
  `builtinPaths` collision set keeps those paths **reserved** either way, so an app cannot claim `/api/_health` with a
  `@route` and change meaning when the flag flips (deliberate: the flag removes surface, it does not free paths).
  With the flag on, those paths fall to the `/api/` catch-all, which answers 404 (`module not found`).
- The banner's "Introspection:" block prints `(off: --no-introspection)` instead of the list.
- `handleListModules`/`handleModuleDetail` return a copy of each `ModuleInfo` whose `Exports` are filtered by
  `s.isExposed`; a module with no exposed exports is omitted from the list and 404s on detail (it has no surface).
- Why not self-host Swagger/ReDoc assets: out of scope; with `--no-introspection` the pages (and their CDN fetches) are gone,
  which is what a tailnet-only page needs. Recorded as future work.

### 2a. Stdlib is never local (D4)

In `registerModule`, before the base-path filter: if `loaded.Path` (canonical ID) or the declared `module` header starts
with `std/`, return `("", false, nil)` without `recordDrop`. The loader already serves these modules to the engine via
`PreloadModule` (unchanged), so imports keep working; they are simply not HTTP/MCP/A2A surface.
A user file cannot declare `module std/…` outside the stdlib and be served — that is intended: `std/` is the stdlib's namespace.

### 2b. Base path (D5)

Move the derivation from `cmd/ailang` into `apiserver.ResolveBasePath(paths []string, cwd string) string` (testable):

1. For every `.ail` file named on the command line or found under a named directory (same walk as `LoadProject`,
   skipping nested `pkg/`), compute its *implied root*: the file path minus `<declared module>.ail`, when the suffix matches.
2. Keep only implied roots that **contain every argument** (a directory argument itself; a file argument's directory).
3. If any remain, use the **outermost** (shortest). Outermost, because every kept root contains every argument, and the
   outermost one also contains every other kept root — so every local module lands under the base path.
4. Otherwise: `cwd` if it contains every argument (today's fallback), else the arguments' common directory.

Daneel's case: argument `/d`; `talk.ail` → `/d` (kept); `client/wsclient.ail` (`module wsclient`) → `/d/client`
(discarded: does not contain `/d`). Base = `/d`. The documented `serve-api ./api/` case: `api/handlers.ail`
(`module api/handlers`) → project root, which contains `./api` → kept.

### 3. `--static-cache` (D6)

`--static-cache immutable` → `Cache-Control: public, max-age=31536000, immutable`;
`--static-cache N` (integer seconds, 1…31536000) → `Cache-Control: public, max-age=N`.
Anything else, or the flag without `--static`, is a startup error. The header is set by a `ResponseWriter` wrapper at
`WriteHeader` time only for 2xx and 304 — a 404 or 416 must never be cached as immutable. Applies only to the `--static`
file server (not `--frontend`'s Vite proxy, not API routes).

### 4. `--ws-pass-header` (D7) — header semantics

| Question | Decision | Why |
|----------|----------|-----|
| Who chooses the headers | The operator, by flag, repeatable. The program cannot widen it | Daneel's ask; a page cannot smuggle a header in because only named headers are read |
| Name matching | Case-insensitive (HTTP semantics); flag values validated as RFC 7230 tokens at startup; duplicates collapse | Header names are case-insensitive on the wire |
| Name in the record | Lower-case (`tailscale-user-login`) | One spelling regardless of how the operator or the proxy wrote it — HTTP/2's wire form; the program compares one string |
| Multi-valued | One `{name, value}` entry per header *line*, in received order; a comma inside a line is not split | Splitting on commas is wrong for some headers (dates, quoted strings); the program sees what arrived |
| Absent | No entry. Present-but-empty → an entry with `value: ""` | Absence and emptiness stay distinguishable |
| Order of entries | Operator's flag order, then received order within one name | Deterministic |
| Credential headers | **Startup error** if named: `Authorization`, `Proxy-Authorization`, `Cookie`, `Sec-WebSocket-Protocol` (carries `ailang.key.<key>`), `Sec-WebSocket-Key`, and the configured `--api-key-header` | Fail loudly (principle 2): silently dropping a listed header would leave the operator believing it arrives. These are the headers serve-api itself treats as credentials (`checkWSKey`) or that carry browser session state |
| Record shape | `req.headers : [{name: string, value: string}]`. The `req` record is now built **field-for-field from the handler's declared type**: only the fields it declares (subset of `path`, `query`, `origin`, `headers`) | The bytecode VM may resolve a closed record's field by static index (`internal/vm/builtins.go:143`); handing a record with extra fields to a handler that declared fewer is exactly the layout mismatch that corrupts reads. Declaring `headers` without the flag yields `[]` |
| Validation | At startup, a WS handler whose `req` declares a field outside `{path, query, origin, headers}`, or a `headers` field not of type `[{name: string, value: string}]`, is refused with the accepted shape | Structured failure instead of a runtime crash on first connect |
| HTTP `@route` handlers | **Unchanged** in this milestone | They already receive *every* header, program-chosen (`_headers: Json`, `@raw`) — `routes_dispatch.go:115`, `routes.go:368`. Narrowing that to an operator allowlist is a breaking change to a documented feature (examples/runnable/mcp_tools.ail `secureParse`). One helper (`passHeaders`) implements the allowlist so HTTP can adopt it later with no second mechanism. Open question for Mark |

**Trace/log interaction (M-SERVEAPI-WS-BRIDGE canary).** serve-api never logs header names or values; the `[ws]`
and decision-log lines are unchanged. The values that reach the program are exactly the named ones, and credential-class
headers cannot be named, so the bound upstream credential and any cookie/API key stay out of the program's values and
therefore out of any trace of its arguments. A new canary test plants a secret in `Cookie` and `Authorization` alongside
the allowed header and asserts: the allowed value reaches the handler (positive control), the canary appears in none of
the handler's echo, the deep trace, or stderr.

**Trust note (documented, not enforced).** A passed header is only as trustworthy as the proxy in front. Tailscale serve
overwrites `Tailscale-User-Login`; a process that reaches the loopback listener directly can set anything. The guide says
so: pass identity headers only with a loopback `--bind` behind the proxy that stamps them.

### Files to Modify/Create

- `internal/apiserver/server.go` — Config fields, `buildRoutes` gating + static wrapper, banner (≈ +40)
- `internal/apiserver/meta.go` — gateway-filtered module listing (≈ +25)
- `internal/apiserver/module_entry.go` — stdlib-is-never-local rule (≈ +12)
- `internal/apiserver/basepath.go` — new, `ResolveBasePath` (≈ 110)
- `internal/apiserver/static_cache.go` — new, flag parsing + wrapper (≈ 70)
- `internal/apiserver/ws_headers.go` — new, allowlist validation, `req` record builder, shape check (≈ 150)
- `internal/apiserver/routes.go` — record WS `req` field names/types from the AST (≈ +30)
- `internal/apiserver/routes_ws.go` — use the builder; shape validation (≈ ±20)
- `cmd/ailang/serve_api.go` — flags, help, `ResolveBasePath` (≈ ±40)
- Tests: `basepath_test.go`, `static_cache_test.go`, `ws_headers_test.go`, `introspection_test.go`, stdlib case in an existing module-entry test
- `docs/docs/guides/serve-api.md`, `changelogs/v0.32-current.md`, generated CLI reference

## Examples

```bash
# Daneel's talk page: exactly the routes and the files, nothing to introspect, dated media cached forever.
ailang serve-api --routes-only --no-introspection --bind 127.0.0.1 --caps Stream \
  --static tools/live-page --static-cache immutable \
  --ws-pass-header Tailscale-User-Login tools/serve
```

```ailang
@route("WS", "/live")
export func live(client: StreamConn, req: {path: string, headers: [{name: string, value: string}]}) -> unit ! {Stream} = ...
```

## Success Criteria

- [ ] Each Goal metric verified with a fresh binary from this worktree (evidence in the PR)
- [ ] Unit tests for every decision row; key tests mutation-tested (revert the fix, watch them fail)
- [ ] `make test`, `make lint`, `make check-file-sizes`, CLI docs regenerated
- [ ] Guide + CHANGELOG updated

## Testing Strategy

**Unit:** `ResolveBasePath` table (Daneel layout, `./api` layout, file arg, no decls); stdlib module not registered and not
a drop; meta listing filtered under `--routes-only`; meta routes 404 with `--no-introspection` and reserved paths still
refused to `@route`; `--static-cache` parse table + 200/304 carry the header, 404 does not; `--ws-pass-header` validation
table (forbidden names, api-key header, bad token); `req` built from declared fields; shape refusal.
**Integration:** a WS session with `Tailscale-User-Login`, a multi-valued header, an unlisted header and a Cookie canary,
through the existing WS harness; the credential canary extended.
**Manual (DONE WHEN):** curl and a Go/`websocat`-free WS client against a fresh binary.

## Deferred Decisions

- Exact wording of banner hint and error text — implementer.
- Whether `--static-cache` also accepts `no-cache` — not now (YAGNI).

## Non-Goals

- Flipping `--routes-only` to default (D3).
- An allowlist for HTTP `@route` headers (D7, open question).
- Self-hosting Swagger/ReDoc assets.

## Open Questions for Mark

1. **Default `--routes-only` when `@route` is present?** Recommended *no* (D3). If yes, it belongs in a minor with a
   CHANGELOG "Breaking" entry and an `--expose-all` escape hatch.
2. **HTTP handlers and headers.** Today `_headers`/`@raw` see every header including `Cookie`/`Authorization`. Should
   `--ws-pass-header` become `--pass-header` and narrow HTTP too (breaking for `_headers` users)?

## Quorum

Not run: the dispatching instruction for this unit forbids evals/quorum. Triggers that would otherwise fire: #1 (D3 and
D7-HTTP are recorded as open questions for a human).

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Base-path rule changes URL paths for some layout | The rule only differs from today when today's first-file guess did **not** contain the argument — the layouts that refused to start or dropped modules. Table tests pin the documented layouts |
| A user module declared under `std/` stops being served | Intended; `std/` is the stdlib namespace. CHANGELOG notes it |
| Handler declared `req` subset previously received extra fields | Now it receives exactly what it declares — strictly safer |

## Related Documents

- `design_docs/implemented/v0_20_0/m-serveapi-surface-drops.md` — the under-basePath fail-fast this doc's 2b stops tripping falsely
- `design_docs/planned/v0_44_0/m-server-origin-policy.md` — the origin rule WS routes share
- `design_docs/planned/v0_44_0/m-stdlib-root-resolution.md` — where `<embedded>/std/` paths come from

---

**Document created**: 2026-09-27
**Last updated**: 2026-09-27
