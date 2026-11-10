### Added — dashboard API contract tests and a deploy smoke check (2026-11-10)

The headless dashboard API now has an engineered stability contract
([m-dashboard-headless-api](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/m-dashboard-headless-api.md)):

- `internal/server/contract_test.go` — one row per kept route, run through the real handler chain
  on a non-loopback bind: the method its client uses, the outcome without and with the dashboard
  credential (and which check refused it), the status, and the JSON keys the CLI or script that
  reads it parses. Also the secret-approval intake → poll round trip, the loopback exemption the
  rig relies on, and a guard that every contract row names a registered route.
- `scripts/dashboard_smoke.sh <base-url> [--expect-benchmarks]` — post-deploy check by status code
  only (never a response body): `/health`, the `/` banner, `/api/version`, a gated read refused
  anonymously, OTLP ingest accepting an empty POST, `/benchmarks/`, and a 404 control. It replaces
  the bare `/health` probe as gate 4/4 of the release smoke gate (`cloudbuild-release.yaml`, against
  test) and runs after every dev roll (`cloudbuild-dev.yaml`, `smoke-dashboard`).

### Fixed — `GET /api/chains/stats` answered 500 when no chain matched (2026-11-10)

`SUM(CASE …)` over zero rows is SQL NULL, which the store could not scan into an int, so an empty
chains table — or a `?hours=` window with no chains in it — made `/api/chains/stats` (and
`GetChainStats`, and the span error count of the filtered metrics summary) fail instead of
reporting zeros. The aggregates are now `COALESCE`d. Found by the new contract test.
