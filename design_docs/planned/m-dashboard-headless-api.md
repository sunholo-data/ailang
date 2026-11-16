# M-DASHBOARD-HEADLESS-API: the dashboard becomes a headless API with a stability contract

**Status**: Planned (decision taken; PRs in flight)
**Target**: v0.59.0
**Priority**: P1
**Estimated**: 3 PRs (UI removal, server surface reduction, stability contract)
**Dependencies**: the dashboard API auth gate (#2197, `internal/server/read_auth.go`) — kept as is.
**Planner-Lane**: opus-required
**Supersedes**: [m-dashboard-minimal-surface.md](m-dashboard-minimal-surface.md) (two screens over a work-ledger API; D8–D10 were never ratified)

## Decision

**Mark, 2026-11-10: option (a) — zero screens.** `ailang serve` (Cloud Run `ailang-dashboard`,
`ailang-dev-dashboard`, `ailang-test-dashboard`; the rig's launchd `dev.ailang.server`) keeps
serving its machine API and stops serving a UI. A small UI may come back later, built on the
authenticated API; nothing in this design blocks that, and nothing here builds toward it.

The dashboard stays in the D-64 **Internal** tier. It is not added to the public stability
promise (`docs/docs/reference/stability.md`); "stable" below means engineered stability — a
documented, tested route surface and a deploy smoke check — not a semver promise.

## Problem Statement

Usage over 30 days (2026-10-10 → 11-09, Cloud Run request logs, all three services; source:
[HANDOVER-v1-codebase-health-2026-11-09.md](HANDOVER-v1-codebase-health-2026-11-09.md) §3):

- **The UI is unused.** One identifiable human browser session in 30 days. UI static
  requests were overwhelmingly crawlers; the UI's data APIs saw ≈150 requests in prod and
  none in dev or test. Local GETs are not logged, but no browser WebSocket has connected to the
  rig server since 2026-11-02.
- **The service is load-bearing.** OTLP ingest (≈55k prod, ≈86k dev, incl. ≈31k OpenRouter
  Broadcast), observatory reads from the rig (≈840), `/benchmarks/` for the docs site (≈377 from
  291 distinct IPs), secret-approval intake for cloud jobs, the release gate's `/health` probe,
  and ≈80k `/api/hooks/claude` POSTs a week to the rig's local server (they feed `ailang chains`).
- **The UI costs more than it is worth.** ≈30k lines of TS/TSX/CSS in `ui/`, ≈10 Dependabot
  PRs a month, a per-PR CI gate, a committed build (`internal/server/dist`) that Dependabot
  bumps never reached, an unused `ui-builder` Docker stage, and ≈7.7k lines of Go handlers
  whose only caller was the UI.

The previous plan (two screens over a new work-ledger API) assumed a human operator using the
dashboard. The usage data says there is none; the operator works through the CLI.

## Goals

**Primary goal:** the dashboard serves exactly the routes something outside the UI calls, each
one classified, contract-tested and smoke-checked after deploy.

**Success metrics:**
- `ui/`, `internal/server/dist`, the `ui-builder` stage, the UI CI workflow (bar the ruleset stub
  below) and the `/ui` Dependabot ecosystem are gone.
- Every registered route has a row in the table below and a consumer; `TestReadAuth_EveryRouteClassified`
  stays complete.
- A contract test per kept route (method, auth outcome without and with credentials, status, and
  the response shape a CLI or script parses).
- `scripts/dashboard_smoke.sh <base-url>` passes against dev and test after each deploy.

## Kept route surface

Auth classes (`internal/server/read_auth.go`): **public** answers without a credential;
**own-auth** is public at the gate but the handler checks its own credential; **gated** needs
`Authorization: Bearer <COORDINATOR_API_KEY>` (or a Firebase Viewer/Approver token) on a
non-loopback bind, 401 without, 503 when the server has neither configured. A loopback-bound
server (the rig) is exempt from the gate.

| Route | Method | Auth class | Consumer (verified by reading the client) |
|---|---|---|---|
| `/` | GET, HEAD | public | none — one-line plain-text banner naming the API and `/health`; every unrouted path is a plain 404 (no SPA fallback) |
| `/health` | GET | public | `cloudbuild-release.yaml` smoke gate 4/4; ailang-multivac `scripts/test-cloud.sh`; `make services-status`; `ailang chains dashboard health`; `scripts/dashboard_smoke.sh` |
| `/benchmarks/*` | GET | public | docs site `docs/src/lib/benchmarkFetch.js` (dev instance) |
| `/v1/traces`, `/v1/logs`, `/v1/metrics` | POST | public (optional `AILANG_OTLP_INGEST_TOKEN`) | OpenRouter Broadcast; `OTEL_EXPORTER_OTLP_ENDPOINT` of the cloud coordinator and every Cloud Run job (ailang-multivac `cloud_run.tf`, `cloud_run_jobs.tf`); rig Claude Code |
| `/api/version` | GET | public | `tools/launchd/rig-watchdog.sh`, `nightly-eval.sh`, `nightly-lang-eval.sh`, `os-rotation-filler.sh` |
| `/api/hooks/claude` | POST | public (optional `AILANG_HUB_TOKEN`) | Claude Code HTTP hooks (`scripts/hooks/claude_settings.json`, `internal/executor/claude_settings.json`) |
| `/api/observatory/hooks` | POST | public | `internal/executor/claude_telemetry.sh`, `docker/resident/lib/observatory.mjs` |
| `/api/exec/sessions`, `/api/exec/events` | POST | public | `internal/executor/claude_telemetry.sh`, `docker/resident/lib/observatory.mjs` |
| `/api/coordinator/events` | POST | public | `internal/coordinator/http_broadcaster.go`, `cmd/ailang/eval_events.go` |
| `/api/telemetry` | POST | public | `internal/eval_harness/telemetry_reporter.go`, `internal/pipeline/metrics.go` |
| `/api/approvals` | POST | own-auth (Google ID token / `AILANG_APPROVAL_TOKEN`) | `internal/secrets/cloud_approver.go` via `AILANG_APPROVAL_URL` (ailang-multivac `cloud_run_jobs.tf`) |
| `/api/approvals/{id}` | GET | own-auth | `internal/secrets/cloud_approver.go` status poll |
| `/api/approvals/{id}/approve`, `/reject` | POST | public with a signed `?token=`, else gated | ntfy Approve/Deny actions (`internal/coordinator/secret_notification.go`) |
| `/ws` | GET (upgrade) | gated (bearer, or `?token=`) | ailang-demos `website_builder/portal/server.js` WebSocket proxy (`DASHBOARD_URL`, ailang-multivac `cloud_run_website.tf`) |
| `/api/inbox` | GET | gated | `ailang chains dashboard inbox` (`cmd/ailang/dashboard_spans.go`) |
| `/api/controlplane/exec-hierarchy` | GET | gated | `ailang chains dashboard hierarchy` (`cmd/ailang/dashboard_tools.go`) |
| `/api/controlplane/stats/breakdown` | GET | gated | `ailang chains dashboard stats` (`cmd/ailang/dashboard_tools.go`) |
| `/api/coordinator/tasks/{id}/events` | GET | gated | `cmd/ailang/coordinator_browse.go` (local server) |
| `/api/observatory/*` | per pattern | gated | `cmd/ailang/eval_paired.go`, `scripts/check_pi_wire_budget.sh`, `ailang chains dashboard spans|traces|sessions|tools`, `cmd/ailang/eval_observatory.go` (workspace/task/agent writes), skill scripts |
| `/api/chains/*`, `/api/spans/{id}` | GET | gated | chains readers (the `ailang chains` family reads the stores directly; the HTTP surface is kept for remote reads) |
| `/api/auth/whoami` | — | — | removed: the Firebase sign-in UI was its only caller |

The authoritative list is the code: `internal/server/routes.go` registers, `read_auth_test.go`
classifies, and the contract tests pin behaviour. This table is the human-readable index.

## What is deleted

- **PR 1 — frontend:** `ui/`; `internal/server/dist` and its `//go:embed`; the SPA file server
  on `/` (replaced by the banner + plain 404); the `ui-builder` stage of
  `docker/Dockerfile.dashboard`; `.github/ui-build-paths.txt`; the body of
  `.github/workflows/dashboard-ui-build.yml`; the `/ui` Dependabot ecosystem; the `/ui/`
  CODEOWNERS line; `make ui-deploy`; `ailang serve` opening a browser; the notify daemon's
  click-through link to the dashboard root; UI sections of the guides and skills.
- **PR 2 — server surface:** every route in `internal/server` with no consumer outside the UI
  (threads, thread messages, workspaces, statistics, agents and process monitor, metrics,
  hierarchy, coordinator status/pending/approve/reject/running, the `/api/tasks/` alias,
  `/api/exec/sessions/{id}` reads, inbox mutations, select-folder, telemetry config, the
  remaining control-plane analytics, budget, Claude history, `/ws/observatory`, whoami), their
  handlers and the helpers only they used. Each removal's consumer check is recorded in the PR.

**Ruleset stub.** The `dev` branch ruleset required a check named `UI build gate`. Deleting the
workflow would have blocked every PR, and the ruleset is an owner setting, so the workflow was
reduced to a one-step always-pass job with that name. **Done 2026-11-12:** Mark removed
`UI build gate` from dev's required checks (now `test, lint, build, docs-gate, checks`) and the
stub workflow was deleted.

## Stability contract

1. **Every route is classified.** `TestReadAuth_EveryRouteClassified` fails on a registered but
   unclassified route and on a classified but unregistered one. A new route is gated by default.
2. **Every kept route has a contract test** (`internal/server/contract_test.go`): method allowed,
   anonymous outcome (public reached / gated 401 with the dashboard realm), credentialed outcome,
   status, and — for reads a CLI or script parses — the JSON keys that client reads.
3. **Deploy smoke** (`scripts/dashboard_smoke.sh <base-url> [--expect-benchmarks]`): status
   codes only, never a response body — `/health` 200, `/` a `text/plain` banner (not a UI page),
   `/api/version` 200, a gated read 401 without a token (200 on a loopback URL; 200 with
   `$AILANG_DASHBOARD_TOKEN` when set), OTLP `/v1/traces` accepts an empty POST, `/benchmarks/`
   registered and GET-only (and `latest.json` 200 with `--expect-benchmarks`), an unrouted path
   404. Wired into `cloudbuild-release.yaml` (smoke gate 4/4, against **test**) and
   `cloudbuild-dev.yaml` (`smoke-dashboard`, after the dev roll, with `--expect-benchmarks`).
   Prod is promoted from a test-verified build; to check it by hand:
   `scripts/dashboard_smoke.sh https://dashboard.ailang.sunholo.com`.
4. **A route is added only with a consumer.** The PR adding it names the client that calls it,
   adds the table row above, classifies it, and adds its contract test.

## Non-goals

- Pruning `internal/observatory`'s own API (`/api/observatory/*`) or the chains API: both have
  live non-UI consumers and are outside `internal/server`'s UI-only code.
- Renaming the service, the `ailang serve` command or the Cloud Run services (no Terraform change).
- Changing OTLP ingest auth (stays off by decision, `.claude/rules/cloud-endpoints.md`).

## Follow-ups outside this repo (owner / ops)

- ~~Remove `UI build gate` from the `dev` ruleset, then delete `.github/workflows/dashboard-ui-build.yml`.~~
  Done 2026-11-12 (Mark removed the required check; the stub workflow was deleted).
- ailang-multivac `scripts/test-cloud.sh` probes `/api/workspaces` and an anonymous `/api/inbox`;
  replace those probes with `scripts/dashboard_smoke.sh`.
- ailang-multivac `cloudbuild-images.yaml` comments the dashboard build as "Go + Node"; it is Go only.
- Artifact Registry `dashboard:cache` manifests have no retention rule.
