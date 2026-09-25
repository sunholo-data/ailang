# M-SERVEAPI-BIND-HOST-CORS: serve-api binds loopback by default, CORS is opt-in and allowlistable

**Status**: Implemented (2026-09-25; sprint plan [m-serveapi-bind-host-cors-sprint-plan.md](m-serveapi-bind-host-cors-sprint-plan.md))
**Target**: v0.44.0 (minor, because it flips two defaults)
**Priority**: P1 (security-ish; it blocks a real deployment, Daneel's tailnet-only page)
**Estimated**: 1–1.5 days
**Dependencies**: None
**Created**: 2026-09-25
**Requested by**: Daneel (`inbox_1790359271083_90811127-companion`, msg `d00e4a75`, 2026-09-25)
**Quorum**: triggers 1 (design-freeze items: two default flips plus the flag name) and 2 (overrides serve-api defaults that docparse and the MCP image depend on) both fire. `ailang design-quorum` was **not run**: Mark ruled on every freeze item directly (see "Decisions" below), which is the input a quorum would have fed.

## Problem Statement

`ailang serve-api` has two defaults that are wrong for any host that isn't a container:

1. **It listens on every interface.** `internal/apiserver/server.go:511` builds `httpAddr := fmt.Sprintf(":%s", s.port)`. There is no flag to pick the host.
2. **It allows cross-origin requests from every origin by default.** `cmd/ailang/serve_api.go:20` declares `--cors` with default `true`. `corsWrap` (`internal/apiserver/server.go:619-632`) then sets `Access-Control-Allow-Origin: *` on every wrapped route and answers every `OPTIONS` with 204.

**Repro (verified on this machine, installed `ailang v0.43.1-4-gab08ce876`, 2026-09-25, scratch dir `/private/tmp/claude-501/serveapi-repro`):**

```
$ cat ping.ail            # module ping / export pure func ping(x: int) -> int = x
$ ailang serve-api --port 18973 .
$ lsof -nP -iTCP:18973 -sTCP:LISTEN
ailang 52656 ... IPv6 ... TCP *:18973 (LISTEN)
$ curl -i -X OPTIONS -H 'Origin: https://evil.example' -H 'Access-Control-Request-Method: POST' \
       http://127.0.0.1:18973/api/ping/ping
HTTP/1.1 204 No Content
Access-Control-Allow-Origin: *
$ curl -i -X POST -H 'Origin: https://evil.example' -H 'Content-Type: application/json' \
       -d '{"args":[7]}' http://127.0.0.1:18973/api/ping/ping
HTTP/1.1 200 OK
Access-Control-Allow-Origin: *
{"result":7,...}
```

**A second finding: turning CORS off does not stop cross-origin *execution*.** With `--cors=false`, the preflight gets a 405 and no CORS headers. But a CORS "simple request" (a POST with `Content-Type: text/plain`, which a browser sends without any preflight) still runs the function:

```
$ ailang serve-api --cors=false --port 18974 .
$ curl -i -X POST -H 'Origin: https://evil.example' -H 'Content-Type: text/plain' \
       -d '{"args":[7]}' http://127.0.0.1:18974/api/ping/ping
HTTP/1.1 200 OK
{"result":7,...}
```

`handleFunctionCall` (`internal/apiserver/handler.go:36`) never checks `Content-Type` or `Origin`. CORS only stops the attacking page from **reading** the response. It does not stop the call itself. For Daneel's endpoint, which spends Vertex money and speaks as Daneel, the call is the harm.

**Who is affected:** Daneel's voice/avatar page is published tailnet-only with `tailscale serve --https=<port> http://127.0.0.1:<port>`. That setup is only a boundary if the backend listens on loopback alone. Today it doesn't: the page is reachable from the LAN, and only the macOS application firewall happens to block it. Any page open in a tailnet user's browser can also call it.

**Related bug this closes for the default case:** `design_docs/planned/ailang-core-triage/serve-api-port-collision.md` describes this problem: the coordinator holds `127.0.0.1:8765` and serve-api then binds the wildcard `:8765`. macOS allows both, and requests split between the two listeners. Once both bind the same loopback address, the second bind fails with EADDRINUSE. This doc also makes that failure loud (see M1, eager listen).

## Goals

**Primary goal:** one bind-and-CORS policy for every AILANG HTTP listener: loopback unless told otherwise, and no cross-origin access unless it is granted.

**Success metrics:**
- `ailang serve-api --bind 127.0.0.1 --port N .` → `lsof` shows `TCP 127.0.0.1:N (LISTEN)`. The default without `--bind` and without `PORT` is the same.
- A preflight from an origin not on the allowlist gets no `Access-Control-Allow-Origin` header.
- With an allowlist set, a cross-origin state-changing request from an origin not on the list is refused with 403 **before the function runs**.
- Cloud Run deploys (docparse API, billing, MCP image) keep working with no config change, or with a one-token change recorded in this doc.
- If the port is already taken, serve-api exits non-zero **before** it prints the startup banner.

## Design Freeze (Mark to ratify)

| # | Decision | Recommendation | Alternative |
|---|----------|----------------|-------------|
| F1 | Flag name | **`--bind ADDR`**. It matches `ailang server --bind` (`cmd/ailang/server.go:48-51`) and `COORDINATOR_BIND_ADDR`, so there's one name for one concept (simplicity program). | `--host` (Daneel's wording). Adding both would give two names for one flag, so no. |
| F2 | Default bind | `127.0.0.1`. `0.0.0.0` when `PORT` is set (`config.Port()`, `internal/config/coordinator.go:83`). `--bind` always wins. This is exactly the rule `ailang server` already follows (`cmd/ailang/server.go:29-38`). | Always loopback with no PORT auto-select. That would break Cloud Run images, which inject `PORT` but pass no `--bind`. |
| F3 | Default CORS | **Off.** `--cors` stays a bool meaning "all origins (`*`)" but its default becomes `false`. New repeatable `--cors-origin ORIGIN` sets an exact-match allowlist. Passing both is a startup error. | Keep `--cors` defaulting to true. That keeps the hole open. |
| F4 | Server-side Origin enforcement | Only when `--cors-origin` is set: a request whose `Origin` header is present, not on the allowlist, and whose method isn't GET/HEAD/OPTIONS gets **403 before dispatch**. Requests with no `Origin` (curl, server-to-server, MCP clients) pass. | Also enforce a same-origin check by default. That is deferred as an open question, because behind `tailscale serve` the `Host` header the backend sees is an external-system detail we haven't measured. |

## Solution Design

### Bind policy (one helper, two callers)

Add `config.DefaultBindHost() string` to `internal/config` next to `Port()`. It returns `"0.0.0.0"` when `PORT` is set, otherwise `"127.0.0.1"`. This is a new getter over an **already-registered** variable (`EnvPort`, `internal/config/coordinator.go:9,44`), so there's **no new env var** and forbidigo is satisfied. Update the `EnvPort` registry description (`coordinator.go:44`) to say that serve-api binds `0.0.0.0` when it is set.

- **serve-api:** `cmd/ailang/serve_api.go` adds `--bind` (default `""`, which means `config.DefaultBindHost()`) and passes it through `apiserver.Config.Bind`. `Server.Start` builds the address with `net.JoinHostPort(bind, port)`. A plain `fmt.Sprintf("%s:%s")` would break `::1`, which is why `JoinHostPort` is used.
- **Eager listen:** `Start` calls `net.Listen("tcp", addr)` **before** `printStartupBanner()`, then `srv.Serve(ln)`. Today `srv.ListenAndServe()` at `server.go:553` runs after the banner has already printed (`server.go:551`). The banner prints `ln.Addr()` instead of the hard-coded `http://localhost:%s` at `server.go:661`. A bind failure then exits non-zero with `bind 127.0.0.1:N: address already in use` and no banner, which fixes the "looks like a normal launch" half of the port-collision triage doc.
- **`ailang server`:** its bind rule is already correct. Make it call `config.DefaultBindHost()` so there is one implementation instead of two. Also fix `isPortInUse` (`cmd/ailang/server.go:295-303`), which probes the wildcard `":port"` while the server binds `localhost:port`. That is the same mismatched pair described in the triage doc, so the probe must test the exact bind address.

### CORS policy (apiserver)

Replace `corsWrap` (`server.go:619-632`). It is applied at `server.go:560-572,604` and `routes.go:337`, and it stays a single wrapper with three modes:

| Mode | Trigger | Response headers | OPTIONS | Cross-origin non-safe request |
|------|---------|------------------|---------|-------------------------------|
| off (new default) | neither flag | none | falls through to the handler, as today (405 on function routes) | runs (unchanged; see open question Q2) |
| any | `--cors` | `ACAO: *` plus the current Methods/Headers | 204 | runs |
| allowlist | `--cors-origin X` (repeatable) | `ACAO: <echoed origin>` and `Vary: Origin`, only when the origin matches exactly | 204 if on the list, **403 with no ACAO** if not | **403** if `Origin` is present and not on the list |

Origins must be `scheme://host[:port]` and are validated at startup. A malformed origin, or `--cors` together with `--cors-origin`, is an error. `cmd/registry-validator/handlers_api.go:62-88` is the in-repo precedent for exact-match echo.

`/mcp/` (`server.go:575-578`) and static/Vite routes (`server.go:607-615`) are not wrapped today, and this doc does not change that.

### Files to modify

- `internal/config/coordinator.go`: `DefaultBindHost()` getter and `EnvPort` description (~10 LOC)
- `internal/apiserver/server.go`: `Config.Bind`, `Config.CORSOrigins`, eager listen, banner, new `corsWrap` (~60 LOC)
- `cmd/ailang/serve_api.go`: `--bind`, `--cors-origin`, `--cors` default false, validation, help text at `:120-121` (~30 LOC)
- `cmd/ailang/server.go`: use `config.DefaultBindHost()`, exact-address `isPortInUse` (~10 LOC)
- `internal/apiserver/server_test.go`: replace `TestCORSHeaders` (`:375-404`) with the tests below
- `docker/Dockerfile.mcp`: add `"--bind", "0.0.0.0"` to the ENTRYPOINT at `:79-83`
- `examples/web_api_demo/test.sh`: the preflight check at `:128-131` expects 204, so pass `--cors` at `:32`
- `tools/ollama-tap/main.go`: default listen `":11435"` becomes `"127.0.0.1:11435"` at `:63`
- `docs/docs/guides/serve-api.md`: flags at `:513` and the example at `:557`, plus a "Binding & CORS" section
- `CHANGELOG.md`: a **Breaking defaults** entry

## Audit: every listener and CORS site in the repo (CLAUDE.md principle 3)

Found with `git grep -E 'ListenAndServe|net\.Listen\(|Access-Control-Allow-Origin|Sprintf\(":%|\.listen\(|http\.server|0\.0\.0\.0'` outside tests and vendored JS.

| # | Site | Bind today | CORS today | Verdict |
|---|------|-----------|-----------|---------|
| 1 | `internal/apiserver/server.go:511,553` (serve-api) | wildcard `:port` | `*` by default (`serve_api.go:20`, `server.go:622`) | **FIX: this doc** (M1, M2) |
| 2 | `cmd/ailang/server.go:29-38,120` (`ailang server` UI) | `localhost`; `0.0.0.0` when PORT is set; `--bind` | see #3 | **Compliant.** This is the reference rule. Route it through the shared getter (M1). |
| 3 | `cmd/ailang/server.go:295-303` `isPortInUse` | probes wildcard `:port` | n/a | **FIX (M1).** Probe the exact bind address. |
| 4 | `internal/server/server.go:648,653-675` (`ailang server` HTTP) plus `internal/websocket/server.go:34-40` | loopback (from #2) | `ACAO: *` on every route; WS `CheckOrigin` returns `true` (`// TODO: In production, validate origin header`) | **FOLLOW-UP, not this sprint.** It is loopback-bound, but any web page in the operator's browser can read its API and open its WebSocket. The UI is served from the same origin, so `*` looks unnecessary, but the OTLP/telemetry ingest paths (`server.go:655-664`) need a per-route check before tightening. File this as a separate triage row. |
| 5 | `internal/coordinator/daemon_http.go:108-126` | `127.0.0.1`; `0.0.0.0` in cloud mode; `COORDINATOR_BIND_ADDR` | none | **Compliant.** |
| 6 | `cmd/registry-validator/main.go:96` + `handlers_api.go:62-88` | wildcard `":"+port` | exact-match allowlist | **Compliant.** Cloud Run service where wildcard is required. Its allowlist is the model for `--cors-origin`. |
| 7 | `tools/ollama-tap/main.go:63,122` | wildcard `:11435` (`OLLAMA_TAP_LISTEN`) | none | **FIX (M3, one line).** A dev proxy that logs full model request bodies should not be on the LAN. Default it to `127.0.0.1:11435`. The env override stays. It lives in `tools/`, which has its own `main`, and uses an existing variable. |
| 8 | `docker/resident/server.mjs:203` | `0.0.0.0` | none | **Compliant.** It is a container entrypoint and Cloud Run requires the wildcard. |
| 9 | `docker/resident/m0-spike/{m0-bench,mem-probe}.sh` | `python3 -m http.server --bind 0.0.0.0` | none | **Ignore.** In-container spike scripts, not shipped. |
| 10 | `/mcp/` route, `internal/apiserver/server.go:575-578` | (serve-api bind) | not wrapped | **No change.** It inherits the M1 bind. MCP clients aren't browsers. |

## Backward compatibility: who relies on the current defaults

| Consumer | Evidence | Effect of the flip | Action |
|----------|----------|--------------------|--------|
| docparse API image (Cloud Run) | `~/dev/sunholo-data/docparse/Dockerfile:77` `ENV PORT=8080`; `:127-134` `serve-api ... --port ${PORT} --cors ...` | PORT is set, so it binds `0.0.0.0`. `--cors` is passed explicitly, so it still sends `*`. | **None** |
| docparse billing image (Cloud Run) | `docparse/Dockerfile.billing:66` `ENV PORT=8080`; `:77-81` `--cors` | same as above | **None** |
| ailang MCP image (Cloud Run, built by `cloudbuild-dev.yaml:151`, `cloudbuild-release.yaml:182`) | `docker/Dockerfile.mcp:79-83`: no `ENV PORT`, no `--cors` | On Cloud Run, PORT is injected, so it binds `0.0.0.0`. A local `docker run -p 8080:8080` without `-e PORT` would bind loopback inside the container and be unreachable. It has no CORS today via corsWrap for `/mcp/`, but the meta routes lose `*`. | Add `--bind 0.0.0.0` explicitly (M3) |
| `make mcp-local` | `Makefile:356-359` | Loopback, and it prints `localhost:8080` | None |
| `examples/web_api_demo/test.sh` | `:32` start, `:128-131` expects OPTIONS → 204 | It would get 405 | Pass `--cors` (M3) |
| `tools/test-concurrency.sh` | curls `http://localhost:$PORT` (`:49,58,70`) | Works over loopback. curl falls back from `::1` to `127.0.0.1`. | None |
| ailang-demos ecommerce UI | `ailang-demos/ecommerce/ui/src/App.tsx:40` uses relative `/api/...` behind the Vite proxy | Same-origin, so no CORS is needed | None |
| OAuth callback servers (Daneel calendar grant, linkedin demo, `sunholo/oauth`) | Daneel `history/…-archive-to-17-sept.md:287`; `ailang-demos/linkedin/scripts/oauth_server.ail:2` | The callback goes to localhost, so it works over loopback | None |
| Anyone who reached a dev serve-api from another device on the LAN | (not grep-able) | That breaks silently from the client's side | Changelog "Breaking defaults" entry: `--bind 0.0.0.0` restores it |
| `TestCORSHeaders` and other apiserver tests | `server_test.go:57`, `cold_start_test.go:281,366`, `nomcp_test.go:162` construct `Config{CORS: true}` directly | The package-level default doesn't change (the flip is in the CLI flag) | Replace `TestCORSHeaders` (M2) |
| `tools/cli_surface_snapshot.sh` (records the serve-api surface, `:178`) | two new flags | The snapshot diff is expected | Re-bank the snapshot. `make simplicity-audit` flag count rises by 2. |

## Implementation Plan (milestones)

**M1: bind policy and eager listen (~0.5 day)**
- `config.DefaultBindHost()`; `--bind` on serve-api; `Config.Bind`; `net.JoinHostPort`; `net.Listen` before the banner; the banner prints `ln.Addr()`.
- `ailang server` uses the shared getter; `isPortInUse` probes the exact address.
- Acceptance:
  - [ ] `TestListenAddress_DefaultLoopback`: with `PORT` unset, after `srv.listen()` the listener's `Addr().(*net.TCPAddr).IP.IsLoopback()` is true.
  - [ ] `TestListenAddress_PortEnvSelectsWildcard`: `t.Setenv("PORT", "0")` makes the resolved host `0.0.0.0`.
  - [ ] `TestListenAddress_ExplicitBindWins`: `Bind: "::1"` with `PORT` set binds `[::1]` (skip if IPv6 is unavailable).
  - [ ] `TestListen_PortInUseFailsBeforeBanner`: pre-bind `127.0.0.1:p`; `Start` returns an EADDRINUSE error and nothing is written to the banner log.
  - [ ] Manual check: `ailang serve-api --bind 127.0.0.1 --port N .` then `lsof -nP -iTCP:N -sTCP:LISTEN` shows `TCP 127.0.0.1:N (LISTEN)` (Daneel's DONE-WHEN).

**M2: CORS modes and Origin enforcement (~0.5 day)**
- `--cors` default false; `--cors-origin` repeatable; startup validation; new `corsWrap` with three modes.
- Acceptance (`httptest` against `buildRoutes()`):
  - [ ] `TestCORS_OffByDefault`: `OPTIONS /api/ping/ping` with `Origin: https://evil.example` returns no `Access-Control-Allow-Origin`.
  - [ ] `TestCORS_AllowlistPreflightDenied`: with allowlist `[https://daneel.ts.net]`, a preflight from `https://evil.example` gets 403 and no ACAO.
  - [ ] `TestCORS_AllowlistPreflightAllowed`: a preflight from the listed origin gets 204, `ACAO` equal to that exact origin, and `Vary: Origin`.
  - [ ] `TestCORS_AllowlistBlocksSimplePost`: `POST` with `Content-Type: text/plain` and an unlisted `Origin` gets **403, and the handler is not invoked** (assert with a counting test export). This is the regression test for the execution finding above.
  - [ ] `TestCORS_NoOriginPasses`: the same POST without an `Origin` header gets 200 (curl and MCP clients are unaffected).
  - [ ] `TestCORS_AnyMode`: `--cors` keeps today's behavior (`*`, preflight 204). This replaces `TestCORSHeaders`.
  - [ ] `TestCORS_FlagConflict`: `--cors --cors-origin X` is a startup error, and so is a malformed origin.
  - [ ] Manual check: the Problem Statement repro with `--cors-origin https://x.example` shows no ACAO for `evil.example`.

**M3: consumers and docs (~0.25 day)**
- `Dockerfile.mcp` gets `--bind 0.0.0.0`; `web_api_demo/test.sh` gets `--cors`; ollama-tap defaults to loopback; update `docs/docs/guides/serve-api.md` and the help text; add the CHANGELOG "Breaking defaults" entry; re-bank the CLI surface snapshot.
- Acceptance: [ ] `examples/web_api_demo/test.sh` passes · [ ] `make test`, `make lint` green · [ ] `docker build -f docker/Dockerfile.mcp` then `docker run -p 8080:8080` (no `-e PORT`) answers on host `:8080`.

## Axiom Compliance

This doesn't touch the language, so there's no Conflict Surface section (no parser, typechecker, codegen or eval files). The relevant axioms: **explicit effects / no ambient authority**. Today serve-api grants network reach and cross-origin invocation that nobody asked for. After this change, exposure is declared explicitly (`--bind`, `--cors`, `--cors-origin`), which matches "all non-determinism and side effects explicit". **Fail loudly** (CLAUDE.md principle 2): a bind collision and conflicting CORS flags both error at startup. No axiom is violated. The net effect is positive.

## Verification Log

| # | Claim | How verified |
|---|-------|--------------|
| V1 | serve-api binds the wildcard | read `internal/apiserver/server.go:511`; live `lsof` shows `TCP *:18973` (repro above) |
| V2 | `--cors` defaults to true, meaning all origins | `cmd/ailang/serve_api.go:20`; `server.go:621-625`; live preflight returned `ACAO: *` |
| V3 | With CORS off, a text/plain cross-origin POST still executes | live repro on :18974 returned `200 {"result":7}`; `handler.go:36` has no Content-Type/Origin check (`git grep 'Content-Type' internal/apiserver` shows it is only read for multipart at `routes_dispatch.go:53`) |
| V4 | `ailang server` defaults to localhost, uses 0.0.0.0 when PORT is set, and has `--bind` | `cmd/ailang/server.go:29-38,48-51,72` |
| V5 | `isPortInUse` probes the wildcard | `cmd/ailang/server.go:295-303` |
| V6 | PORT is already in the config Registry, so no new env var is needed | `internal/config/coordinator.go:9,44,83` |
| V7 | No shared bind-host helper exists today (negative claim) | `git grep -E "func .*(BindAddr\|BindHost\|bindHost\|ListenAddr)"` → only `config.CoordinatorBindAddr` (`coordinator.go:80`), which reads the env var and resolves nothing |
| V8 | The banner prints before `ListenAndServe` | `server.go:551` then `:553` |
| V9 | The docparse images set PORT and pass `--cors` explicitly | `docparse/Dockerfile:77,127-134`; `Dockerfile.billing:66,77-81` |
| V10 | `Dockerfile.mcp` sets no PORT and passes no `--bind`/`--cors` | `docker/Dockerfile.mcp:79-83`; built by `cloudbuild-dev.yaml:151`, `cloudbuild-release.yaml:182` |
| V11 | `/mcp/` is not CORS-wrapped | `server.go:575-578` |
| V12 | `ailang server` sends CORS `*` and the WS accepts any origin | `internal/server/server.go:667`; `internal/websocket/server.go:37-40` |
| V13 | registry-validator uses an exact-match allowlist | `cmd/registry-validator/handlers_api.go:62-88` |
| V14 | The web_api_demo test asserts preflight 204 | `examples/web_api_demo/test.sh:128-131` |
| V15 | Existing apiserver tests set `CORS: true` explicitly | `server_test.go:57`, `cold_start_test.go:281,366`, `nomcp_test.go:162` |
| V16 | Every server started for verification was killed | `lsof` on 18973 and 18974 after `pkill` shows no listener |

External premise, not verifiable in-repo: Cloud Run requires the container to listen on `0.0.0.0` and injects `PORT`. The coordinator relies on the same premise (`daemon_http.go:103-106`).

## Decisions (Mark, 2026-09-25)

Every freeze item and open question was resolved with this doc's recommendation.

| # | Ruling |
|---|--------|
| F1 / Q1 | The flag is **`--bind`**, matching `ailang server --bind`. No `--host` alias. Daneel's DONE-WHEN check becomes `--bind 127.0.0.1`. |
| F2 | Default bind is `127.0.0.1`, or `0.0.0.0` when `PORT` is set; `--bind` wins. One getter, `config.DefaultBindHost()`. |
| F3 / F4 / Q2 | With no CORS flags, CORS is **off**, and there is **no** default same-origin enforcement. The server-side 403 for unlisted origins applies only when `--cors-origin` is set. Default-mode execution of a cross-origin `text/plain` POST stays as it is and is documented in the guide. |
| Q3 | `ailang server`'s CORS `*` and any-origin WebSocket (audit row 4) are **out of scope**: a documented follow-up (see below). |
| M3 | Include the ollama-tap loopback one-liner, `Dockerfile.mcp` explicit `--bind 0.0.0.0`, and `web_api_demo/test.sh --cors`. |

**Follow-up (not done here):** audit row 4. `internal/server/server.go` sets `Access-Control-Allow-Origin: *` on every route, and `internal/websocket/server.go` `CheckOrigin` returns `true`. Tightening needs a per-route check of the OTLP/telemetry ingest paths first.

## Open Questions for Mark (resolved; see Decisions above)

1. **F1, the flag name:** `--bind` (consistent with `ailang server`) or `--host` (Daneel's wording, used in his DONE-WHEN)? This doc recommends `--bind`, and Daneel's check becomes `--bind 127.0.0.1`.
2. **Default-mode execution hole:** with no CORS flags, a cross-origin `text/plain` POST still runs the function (V3). Should the default mode also refuse non-safe requests whose `Origin` doesn't match the request's own origin? That is the right fix, but it depends on what `Host` and `X-Forwarded-Host` `tailscale serve` passes through, which is unmeasured. Recommendation: ship F4 (allowlist-only enforcement) now, and Daneel runs with `--cors-origin https://<his tailnet name>`.
3. **Row 4 (`ailang server` CORS `*` and WS `CheckOrigin: true`):** confirm it is a separate follow-up and not part of this sprint.

## Non-Goals

- Authentication. `--api-key-header` already exists, and CORS/bind are not auth.
- Restructuring the `/mcp/` and static/Vite routes.
- The coordinator/serve-api port-sharing remedies in the triage doc beyond "bind the same address, fail loud" (holder identification, moving the oauth example port).

## Related Documents

- `design_docs/planned/ailang-core-triage/serve-api-port-collision.md`: the wildcard-vs-loopback pair. M1 resolves the default case and makes a collision fail loud.
- `design_docs/implemented/v0_10_12/m-serveapi-unify.md`, `design_docs/implemented/v0_11_0/m-dx-serve-api-coercion.md`: neural matches at 0.43–0.49. Both are about routing and coercion, not binding, so they are distinct.
- `design_docs/planned/v0_29_0/m-serve-api-live-tool-registry.md`: 0.47, MCP registry. Distinct.
