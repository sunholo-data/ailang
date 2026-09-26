# Sprint Plan: M-SERVEAPI-BIND-HOST-CORS

**Design doc**: [m-serveapi-bind-host-cors.md](m-serveapi-bind-host-cors.md)
**Sprint JSON**: `.ailang/state/sprints/sprint_m_serveapi_bind_host_cors.json`
**Target**: v0.44.0 (minor: two default flips)
**Duration**: 1–1.5 days · **Risk**: medium (default flips reach Cloud Run consumers; all are audited in the design doc)
**Created**: 2026-09-25

## Decisions ratified (Mark, 2026-09-25)

- F1: the flag is `--bind`, with no `--host` alias.
- F3/F4/Q2: with no CORS flags, CORS is off and there is no default same-origin enforcement. The 403 for unlisted origins applies only when `--cors-origin` is set.
- Q3: `ailang server`'s CORS `*` and any-origin WebSocket are a documented follow-up, not this sprint.
- M3 includes the ollama-tap loopback default, `Dockerfile.mcp` explicit `--bind 0.0.0.0`, and `web_api_demo/test.sh --cors`.

## Current state

- `internal/apiserver/server.go` is 750 LOC. The new CORS code goes in a new `cors.go` and the bind/listen code in `listen.go`, so server.go does not grow.
- `ailang server` already implements the bind rule (defaulting to `localhost`, not `127.0.0.1`); `isPortInUse` probes the wildcard.
- `PORT` is already registered (`internal/config/coordinator.go`), so no new env var is needed.

## Milestones

### M1: bind policy + eager listen ✅ (~90 LOC impl, ~90 LOC tests)
Files: `internal/config/coordinator.go` (`DefaultBindHost()`, `EnvPort` description), `internal/apiserver/listen.go` (new: bind resolution, eager listen), `internal/apiserver/server.go` (`Config.Bind`, `Start` listens before the banner and calls `Serve(ln)`), `cmd/ailang/serve_api.go` (`--bind`), `cmd/ailang/server.go` (shared getter, exact-address `isPortInUse`).
Acceptance:
- [x] `TestListenAddress_DefaultLoopback`, `TestListenAddress_PortEnvSelectsWildcard`, `TestListenAddress_ExplicitBindWins` (IPv6 skip), `TestListen_PortInUseFailsBeforeBanner`.
- [x] `config.DefaultBindHost` unit test (PORT set/unset), plus `TestIsPortInUse_ExactAddress`.
- [x] Manual: `lsof` shows `TCP 127.0.0.1:N (LISTEN)`.

### M2: CORS modes + Origin enforcement ✅ (~80 LOC impl, ~150 LOC tests)
Files: `internal/apiserver/cors.go` (new: `ValidateCORSOrigins`, three-mode `corsWrap`), `internal/apiserver/server.go` (`Config.CORSOrigins`), `cmd/ailang/serve_api.go` (`--cors` default false, repeatable `--cors-origin`, conflict/malformed validation), `internal/apiserver/server_test.go` (replace `TestCORSHeaders`).
Acceptance (all [x], mutation-checked): `TestCORS_OffByDefault`, `TestCORS_AllowlistPreflightDenied`, `TestCORS_AllowlistPreflightAllowed`, `TestCORS_AllowlistBlocksSimplePost` (handler not invoked), `TestCORS_NoOriginPasses`, `TestCORS_AnyMode`, `TestCORS_FlagConflict` (+ malformed origin). Manual: a preflight from an unlisted origin has no ACAO; the text/plain POST gets 403.

### M3: consumers + docs ✅ (~40 LOC + docs)
Files: `docker/Dockerfile.mcp`, `examples/web_api_demo/test.sh`, `tools/ollama-tap/main.go`, `docs/docs/guides/serve-api.md`, serve-api help text, `CHANGELOG.md` (Breaking defaults), the CLI surface snapshot if it is tracked, and the design doc's decision section + follow-up row.
Acceptance: [x] `make test` and `make lint` green; [x] `make fmt` clean; [x] file sizes within limits; [x] `examples/web_api_demo/test.sh` 17/17 with the worktree binary; [ ] `docker run` of the MCP image not run (no docker on this host).

## Out of scope / deferred
- `ailang server` CORS `*` + WS `CheckOrigin: true` (follow-up).
- Default-mode same-origin enforcement (Q2; needs a measurement of the headers `tailscale serve` passes).
- `docker build`/`docker run` of the MCP image (run only if docker is available; the change is a one-token ENTRYPOINT addition).

## Success metrics
- Every M1/M2 named test passes, and fails when the fix is reverted (mutation check).
- Daneel's DONE-WHEN verified with a binary built from this worktree.
