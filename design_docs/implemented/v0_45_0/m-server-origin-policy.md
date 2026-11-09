# M-SERVER-ORIGIN-POLICY: `ailang server` becomes same-origin by default, and its WebSocket checks Origin

**Status**: Planned (implemented alongside this doc, same PR)
**Target**: v0.44.x (it flips a security default)
**Priority**: P1 (security default)
**Estimated**: 0.5 day
**Dependencies**: M-SERVEAPI-BIND-HOST-CORS (#1313, `design_docs/implemented/v0_44_0/m-serveapi-bind-host-cors.md`). This closes its audit row #4.
**Created**: 2026-09-26
**Quorum**: not run. Trigger 1 (a default flip) fires, but the orchestrating session fixed the policy in the task (same-origin, an explicit allowlist, reuse of #1313's code). Trigger 4 fires on the dashboard's cross-origin callers, which are external systems. Those premises were checked by reading their source (V5–V8). The residual questions are listed for Mark below instead of being spent on a review round.

## Problem

`ailang server` (the Collaboration Hub/dashboard) exposes the coordinator (approve/reject tasks),
messages and observatory APIs. It has two holes:

1. `corsMiddleware` (`internal/server/server.go:651-678` before this change) set
   `Access-Control-Allow-Origin: *` on **every** route and answered every `OPTIONS` with 200.
2. The hub WebSocket upgrader (`internal/websocket/server.go:34-40`) had
   `CheckOrigin: return true // TODO: In production, validate origin header`. The only check was a
   token, and only when `COORDINATOR_API_KEY` was set, which it never is locally.

The server binds loopback locally, but any web page open in the operator's browser can read the API,
POST to it, and open `/ws` or `/ws/observatory` (cross-site WebSocket hijacking). Measured on the
installed `v0.44.0-9` binary (V1):

```
OPTIONS /api/version  Origin: https://evil.example  → 200, Access-Control-Allow-Origin: *
GET     /api/version  Origin: https://evil.example  → 200, Access-Control-Allow-Origin: *
GET /ws (Upgrade)     Origin: https://evil.example  → 101 Switching Protocols
```

## Who legitimately calls it cross-origin

| Caller | How it reaches the server | Origin seen | Effect of same-origin default | Opt-in |
|--------|---------------------------|-------------|-------------------------------|--------|
| Embedded dashboard UI (`//go:embed dist`, local and Cloud Run) | relative `/api`, `window.location.host` for `/ws` (V5) | same-origin | none | none |
| Vite dev server (`cd ui && npm run dev`, :3000) | proxies `/api` and `/ws` to :1957 (V6) | `http://localhost:3000`. `/ws` keeps `Host: localhost:3000`, but `/api` had `changeOrigin: true`, so `Host` became `localhost:1957` | every POST looks cross-origin → **403** (measured, V9) | **fixed in-repo**: drop `changeOrigin` on `/api`, so `Host` matches `Origin`. Alternatively `--cors-origin http://localhost:3000` |
| Docs site (GitHub Pages) → `GET /benchmarks/*` on the dev dashboard run.app (V7) | browser `fetch`, cross-origin | `https://ailang.sunholo.com` | would lose `ACAO` → falls back to the stale in-build copy | **route exception**: `/benchmarks/*` keeps `ACAO: *` for GET/HEAD/OPTIONS only. It is public read-only data. |
| website-builder sidecar (ailang-demos `website_builder/portal/server.js:1439-1452`) → `wss://…dashboard…/ws?token=` (V8) | Node `ws` client, server-side | none | none (missing Origin admits) | none |
| Documented browser integration (`ailang-demos/website_builder/docs/ailang-cloud-integration.md:877`: `new WebSocket(${DASHBOARD_URL}?token=…)`) | browser, cross-origin, with token | foreign | **kept**: a valid `?token=` admits any origin. That token check is the existing external-client contract, and a hostile page cannot know the token. | none |
| CLIs (`ailang dashboard`, `eval_events`, coordinator `http_broadcaster`), Claude Code HTTP hooks, OTLP/Broadcast exporters | Go/HTTP clients | none | none | none |
| Collaboration hub = this server. VS Code/LSP does not call it (V10). | — | — | — | — |

No deployment config needs to change: the `ailang-multivac` dashboard service passes no CORS setting,
and its callers are covered by the rows above (V7, V8). **Who breaks:** a browser page on another
origin that called the hub without the token, e.g. a custom UI on another port or a notebook. It
opts in with `--cors-origin <origin>`.

## Policy

The origin logic is **one implementation**, `internal/platform/originpolicy`, extracted from #1313's
`internal/apiserver/cors.go` (CLAUDE.md principle 3). Both servers call it. The implementation, not
an alternative:

- `Validate(any, origins)` checks the exact `scheme://host[:port]` form (moved unchanged).
- `Apply(w, r) bool` handles REST. A request is admitted when it has no Origin, is same-origin
  (the Origin's host:port equals `r.Host`), or is listed (the origin is echoed back with
  `Vary: Origin`). Any other origin gets GET/HEAD without ACAO, and **403 before dispatch** for
  everything else, preflight included. `--cors` sets `*`.
- `CheckWebSocket(r, allowMissing) error` admits same-origin and listed origins. `null` and foreign
  origins are refused. `--cors` is not a WebSocket grant.

| Server | Mode wiring | Missing Origin on WS |
|--------|-------------|----------------------|
| `ailang server` | always `Apply`. `--cors-origin` (repeatable) is the new flag and carries the same name as serve-api's, so one concept has one name | **allowed** (non-browser clients) |
| `ailang serve-api` | unchanged: off mode skips `Apply`, `--cors` / `--cors-origin` call it | refused (unchanged from #1313) |

The hub WebSocket runs the origin check first and admits on a valid `?token=`. After that, the
existing token rule still applies (same-origin is exempt from the token).

**One deliberate serve-api change:** in `--cors-origin` mode, a same-origin request now passes
without being listed. Before this, a page served by serve-api itself got a 403 on POST unless its
own origin was on the list.

## Files

- `internal/platform/originpolicy/originpolicy.go` (new, ~150 LOC) + tests
- `internal/apiserver/cors.go`: now a thin adapter. `routes_ws.go` uses `CheckWebSocket(r, false)`
- `internal/server/origin.go` (new): the middleware plus the `/benchmarks/` exception and `WithCORSOrigins`. `server.go` builds the policy and hands it to the WS hub
- `internal/websocket/server.go`: `SetOriginPolicy`, the check before `Upgrade`, and a constant-time token compare
- `cmd/ailang/server.go`: `--cors-origin`, validated at startup
- `ui/vite.config.ts`: `/api` proxy without `changeOrigin`
- docs: `collaboration-hub.md`, `serve-api.md`. Changelog: `changelogs/v0.32-current.md`

## Acceptance (all tested, mutation-checked)

- [x] Foreign-origin REST: no ACAO on GET/HEAD, 403 on POST/PUT/DELETE/OPTIONS before dispatch (`TestHubOrigin_ForeignRefusedByDefault`)
- [x] Foreign-origin and `null` WS upgrade refused with 403 (`TestWSOrigin_ForeignRefused`, `TestWSOrigin_DefaultPolicyIsSameOrigin`)
- [x] Same-origin and allowlisted origins admitted on REST and WS. The allowlist reaches the WS hub through `NewServer` (`TestHubOrigin_AllowlistedOriginGranted`, `TestWSOrigin_SameOriginListedAndCLIAdmitted`)
- [x] No-Origin (CLI) admitted (`TestHubOrigin_SameOriginAndCLIPass`)
- [x] Token contract kept (`TestWSOrigin_TokenContract`). `/benchmarks/*` stays public for reads only (`TestHubOrigin_BenchmarksStayPublic`)
- [x] 10 mutations, all killed (see PR)

## Verification Log

| # | Claim | Evidence |
|---|-------|----------|
| V1 | Old binary sends `ACAO: *` and accepts a foreign WS | curl against `ailang v0.44.0-9-g94eeb7d8a` on :19574 (output above) |
| V2 | `internal/server` had exactly one CORS site, and it wraps the whole mux | `grep -rn Access-Control internal/server` → only `server.go:667-669`. `handler := s.corsMiddleware(mux)` |
| V3 | Two gorilla upgraders exist: the hub (`internal/websocket`) and `platform/streamws` (serve-api, caller-checked) | `grep -rn websocket.Upgrader` |
| V4 | No other in-repo browser client of the hub | `grep -rn "localhost\|1957" ui/src` → only relative URLs and `window.location.host` |
| V5 | UI uses same-origin URLs | `ui/src/hooks/useObservatory.ts:415`, `ui/src/features/controlplane/hooks/useEventQueue.ts:196` |
| V6 | Vite proxy config | `ui/vite.config.ts` (`/api` `changeOrigin: true`, `/ws` no changeOrigin) |
| V7 | Docs site fetches `/benchmarks/*` cross-origin | `docs/src/lib/benchmarkFetch.js:18,25` (`DASHBOARD_BASE` = dev run.app) |
| V8 | Dashboard Cloud Run passes no CORS config. The sidecar is a Node client with a token | `ailang-multivac/terraform/cloud_run.tf` dashboard env block. `ailang-demos/website_builder/portal/server.js:1439-1452` |
| V9 | Vite `changeOrigin: true` makes same-origin POSTs 403. Dropping it fixes that | Two Vite instances on :3998 (old config) → 403 and :3999 (new) → the handler's 404, `/ws/observatory` 101 |
| V10 | LSP/VS Code do not call :1957 | `grep -rln 1957` lists no LSP or extension code |

## Open questions (for Mark)

- **Q1 DNS rebinding.** Same-origin is judged from `Origin` vs `Host`. A rebinding attack makes both
  name the attacker's domain, and no Origin policy can stop that. A loopback `Host` allowlist
  (`localhost`, `127.0.0.1`, `[::1]` when bound to loopback) would. Is that worth a follow-up for
  both servers?
- **Q2 GET reads.** Foreign-origin GETs still *execute* (without ACAO). The hub has no state-changing
  GETs we know of, and the response can't be read. Refusing foreign GETs outright would also break
  `<img>`/navigation, so this is left as is.
- **Q3 OTLP ingest** on the prod dashboard remains unauthenticated by your 08-13 decision. This change
  does not touch non-browser ingest.
