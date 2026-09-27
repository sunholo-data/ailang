# Sprint plan: M-SERVEAPI-OPERATOR-SURFACE

**Design doc**: [m-serveapi-operator-surface.md](m-serveapi-operator-surface.md)
**Target**: v0.45.0 · **Duration**: 1.5 days (one session) · **Risk**: low-medium
**Progress file**: `.ailang/state/sprints/sprint_m_serveapi_operator_surface.json`

## Summary

Four Daneel requests, one unit: what serve-api exposes is the operator's choice. Five milestones, ordered so each is
independently testable and the riskiest (base path, WS header record) come after the cheap fixes are green.
Velocity reference: the M-SERVEAPI-WS-BRIDGE sprint (same package, 2026-09-25) landed ~2,270 LOC in 4 milestones
in one day; this sprint is ~900 LOC.

## Milestones

### M1 — Stdlib is never local (D4) · ~60 LOC + tests
- `registerModule`: canonical ID or declared path under `std/` → not registered, not a drop.
- Test: a module importing `std/stream` (embedded root, base = cwd layout) registers 0 `std/*` modules and records 0 drops.
- **Accept**: banner/`GetModules` hold no `std/*`; test fails with the rule reverted.

### M2 — Introspection: gateway filter + `--no-introspection` (D1, D2) · ~120 LOC + tests
- `meta.go` listing/detail filtered through `isExposed`; `Config.NoIntrospection`; `buildRoutes` gating; banner.
- **Accept**: `--routes-only` `/api/_meta/modules` lists only `@route` exports; with the flag every meta path and
  `/api/_health` 404s while routes still answer; `@route` on a reserved path is still refused.

### M3 — Base path from the command line (D5) · ~150 LOC + tests
- New `apiserver.ResolveBasePath`; `cmd/ailang/serve_api.go` calls it; `findFirstModuleDecl` removed.
- **Accept**: table test — Daneel layout (`module wsclient` in `client/`) → the given dir; `./api` with
  `module api/handlers` → project root; single file; no module decls → cwd/common dir.

### M4 — `--static-cache` (D6) · ~110 LOC + tests
- Parse `immutable|SECONDS`; wrapper sets header on 2xx/304 only; flag without `--static` is an error.
- **Accept**: 200 and 304 carry the header, 404 does not, no header without the flag.

### M5 — `--ws-pass-header` + declared-shape `req` (D7) · ~350 LOC + tests
- Flag (repeatable) → `WSConfig.PassHeaders`; validation (token syntax, forbidden credential names incl. `--api-key-header`).
- `ExportInfo.WSReqFields` from the AST; `ValidateWSRoutes` refuses unknown fields / wrong `headers` type.
- `serveWSSession` builds `req` from declared fields only; `headers` = allowlisted, lower-cased, one entry per line.
- Canary test: `Cookie`/`Authorization` secrets never reach handler, trace, log; allowed header does (positive control).
- **Accept**: handler echo shows exactly the allowed header(s); multi-valued header yields two entries; subset-declared
  handler still reads `path` correctly.

### M6 — Docs, CHANGELOG, CLI reference, DONE WHEN on a fresh binary · ~100 LOC docs
- `docs/docs/guides/serve-api.md`, `changelogs/v0.32-current.md` [Unreleased], `make check-cli-docs` / regenerate.
- curl evidence for M1–M4; a WS client with a custom header for M5.

## Example files

`examples/serveapi_ws_bridge.ail` stays valid (declares `req: {path, query}`, a subset). A new WS header example is
not added as a separate file: the guide documents the record shape, and the unit fixture exercises it (the flag is
an operator surface, not a language feature).

## Open questions (carried to the PR)

1. Default `--routes-only` when `@route` present — recommended no.
2. Operator allowlist for HTTP `_headers`/`@raw` — deferred, breaking.
