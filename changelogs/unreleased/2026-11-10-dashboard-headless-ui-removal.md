### Changed — the dashboard is a headless API: the web UI is removed (2026-11-10)

`ailang serve` (Cloud Run `ailang-dashboard`, `ailang-dev-dashboard`, `ailang-test-dashboard`,
and the rig's local server) no longer serves a UI. Usage over 30 days showed one human browser
session while OTLP ingest, observatory reads, `/benchmarks/`, approvals intake and `/health`
carried the load, so the UI goes and the API stays
([m-dashboard-headless-api](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/m-dashboard-headless-api.md),
superseding m-dashboard-minimal-surface).

- Removed `ui/` (React/Vite, ≈39k lines incl. the lockfile), the committed `internal/server/dist`
  bundle and its `//go:embed`, the unused `ui-builder` stage of `docker/Dockerfile.dashboard`,
  `make ui-deploy`, the `/ui` Dependabot ecosystem and the UI build CI job.
- `/` now answers a one-line plain-text banner (`AILANG dashboard API (headless). Health: /health`)
  and any path no route claims is a plain-text 404 — there is no SPA fallback, so a mistyped API
  path no longer answers 200 with an HTML page.
- `ailang serve` no longer opens a browser on start (or when it finds a server already running).
- Notify-daemon notifications no longer carry a click-through link to the dashboard root.
- Every API route, its auth, and the hook/OTLP ingest paths are unchanged.
- The `UI build gate` check is kept as an always-pass stub because the `dev` ruleset requires it;
  it goes when the ruleset entry does.
