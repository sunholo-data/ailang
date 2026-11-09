# HANDOVER — v1.0 codebase-health assessment (2026-11-09)

**Status:** assessment, not a plan. Measured in an attended session on 2026-11-09 at
`origin/dev` ≈ `bdbe023e3`, for whoever (human or agent) decides the work. Nothing below
has been changed in code except where a commit is cited.
**Context:** v1.0 bar v3 (`v1-mission.md`, ledger `D-64`/`D-65`) is down to one gate
item, G5 (simplification metrics in the release notes). Mark asked, before tagging, what
else would make the codebase easier to work with, plus getting SonarCloud and GitHub
alerts back under control. No v1.0.0 tag yet (Mark, 2026-11-09).

Every number below says the command that produced it, so it can be re-measured rather
than trusted.

## 1. Simplicity audit (`make simplicity-audit-fast`, vs snapshot 2026-11-06)

| Metric | 11-06 | 11-09 | Note |
|---|---|---|---|
| `tracked_files` (gated) | 30,219 | 30,431 | **+212, the one regression.** Records, not code: 82 `design_docs/fleet-mission-evidence/`, 23 changelog fragments, 19 `tools/finetune/`. Evidence directories have no retention bound. |
| `internal_loc` | 366,001 | 367,911 | +1,910 |
| `cmd_ailang_loc` | 68,220 | 68,312 | +92 |
| `planned_design_docs` | 540 | 363 | −177 (#2186 moved shipped docs) |
| everything else | | | unchanged: 17 top-level commands, 1 backend switch, 22 duplicate symbol names, 14 packages without a doc comment, 0 core→platform leaks |

**Instrument fix (`d979a8d30`).** The audit's "skills citing a missing command" check had
been reporting *every* command (incl. `ailang check`) since the S5 CLI reshape, because it
grepped `case "x"` labels that `cmd/ailang/main.go` no longer has. It now reads the
`Name`/`Aliases` rows in `cmd/ailang/commands_*.go` plus the `pkgLegacyTopLevel` bare
spellings, scans all skill markdown (resources included), and counts command-position
citations only. Controls: `check`/`coordinator`/`docs`/`pkg-docs` resolve; a planted
`nosuchcmd` is flagged. First real result: **64 citations of the retired `ailang agent`**,
almost all in `.claude/skills/headless-runner/resources/` (`agent_workflows.md` 56,
`troubleshooting.md` 11). The skill's own quick example was migrated; its resources were
not. `headless-runner` does not appear in the session skill listing, so deleting the
skill may be the better fix than rewriting 67 lines.

## 2. What the audit does not see — measured

### 2a. 70% of production Go is platform, not language

Tracked non-test `.go` under `internal/`, split by the core list in
`.claude/rules/architecture.md` (`parser types eval core elaborate effects builtins lexer
ast pipeline runtime link iface`):

| | prod LOC | test LOC |
|---|---|---|
| language core | 109,849 | 113,385 |
| platform | **258,075** | 248,179 |

Largest packages (prod / test): coordinator 30,117 / 33,374 · builtins 27,633 / 22,417 ·
types 21,987 / 15,102 · observatory 19,456 / 13,542 · effects 19,202 / 26,978 · gen
15,802 / 9,692 · executor 14,696 / 16,651 · eval_harness 13,520 / 13,887 · server 13,218 /
4,268 · ai 13,069 / 18,221 · pipeline 11,946 / 21,207 · mission 11,239 / 12,779.

Most of the platform share is the tier `D-64` declared **Internal** (coordinator, missions,
approvals, dashboards, eval tooling): outside the 1.0 promise, yet in the one binary and in
every contributor's search results.

### 2b. Unreachable code

`deadcode -test=false ./cmd/...` (golang.org/x/tools/cmd/deadcode v0.28.0): **1,614
unreachable functions.** 348 are in vendored `third_party/goopus` (28,053 lines; used by
exactly one builtin, `internal/builtins/audio_ogg.go`). The remaining ~1,266 by package:
types 149 · eval_harness 110 · gen 86 · ast 72 · eval 71 · coordinator 70 · browser 60 ·
server 59 · messaging 37 · testing 35 · core 35 · observatory 34 · effects 34 · repl 31.
Hottest files: `internal/eval/eval_typed.go` 37, `internal/gen/stmt/stmt.go` 32,
`internal/core/core.go` 30, `internal/ast/ast_expr.go` 28, `internal/messaging/client.go` 23.

**Caveat, binding on whoever acts:** `-test=false` counts test-only helpers as dead, and
reachability cannot see data coupling or reflection. `coding-standards.md` forbids deleting
on "unused" alone — each removal needs a `-test=true` cross-check and `git log -S`.

### 2c. SonarCloud (public API, no token)

`curl 'https://sonarcloud.io/api/qualitygates/project_status?projectKey=sunholo-data_ailang'`

- **Quality gate: ERROR** — `new_coverage` 78.3 < 80. All other new-code conditions OK
  (duplication 0.3%, hotspots 100% reviewed, ratings A).
- Overall: **security rating E (5.0), 203 vulnerabilities**; **reliability D (4.0), 7
  bugs**; 3,081 code smells; duplication 3.1%; cognitive complexity 68,159; coverage
  70.9%; 325,755 ncloc. Maintainability A.
- Triage tooling exists: `.claude/skills/sonarcloud-triage/` (needs `SONAR_PAT`).

### 2d. GitHub security alerts

- **Dependabot (5 open):** critical + low in `github.com/ollama/ollama` (go.mod, **no
  patched version**) — imported only for API types in 5 files (`internal/ai/ollama/{client,
  step,tokens}.go`, `internal/builtins/ollama_embed.go`, `internal/messaging/embedder.go`);
  replacing it with local HTTP types clears both. High: `zod` in
  `tools/pi-extensions/sandbox` (fix ≥ 4.6.6). Medium: `minimatch` in `ui/` (fix ≥ 11.0.1;
  disappears with the UI). Low: `deepmerge` in `docs/` (no fix).
- **Code scanning: 400 open** — CodeQL 266 (**30 high**, 236 medium), Scorecard 134 (5
  high, 129 medium).
- **Secret scanning is disabled** on the repository (API 404). Repo setting — owner's call.

## 3. Dashboard: drop the UI, keep the API

Read-only usage audit, 30 days (2026-10-10 → 11-09), Cloud Run request logs for
`ailang-dashboard` / `ailang-dev-dashboard` / `ailang-test-dashboard` (europe-west1).

**The UI is effectively unused; the service is load-bearing.**

| Path class (2xx only) | prod | dev | test |
|---|---|---|---|
| OTLP `/v1/traces` + `/v1/metrics` | 54,974 (≈31k OpenRouter Broadcast) | 86,221 | 1,458 |
| `/api/observatory/*` reads | 840 (821 from the rig) | 0 | 3 |
| `/benchmarks/` (docs site live data) | — | 377 (291 distinct IPs) | — |
| UI static | 2,711, overwhelmingly scanners/crawlers | 22 (scanners) | 0 |
| UI data APIs | ≈150 | 0 | 0 |

One identifiable human browser session in 30 days (prod, 2026-10-22). Local `ailang serve`
on the rig (`dev.ailang.server`) receives ≈80k `/api/hooks/claude` POSTs a week plus OTLP —
it feeds `ailang chains`; no browser WebSocket has connected since 2026-11-02.

**What depends on the service (not the UI):** OpenRouter Broadcast ingest; the cloud OTLP
sink for the coordinator and every job (`ailang-multivac` `cloud_run.tf:177`,
`cloud_run_jobs.tf`, the egress allowlist `agent_egress.tf:63-66`); dev secret approvals
(`AILANG_APPROVAL_URL`, `cloud_run_jobs.tf:66,270`); the docs site's benchmark fetch
(`docs/src/lib/benchmarkFetch.js:18`, dev instance); the release gate's `/health` probe
(`cloudbuild-release.yaml:529`); a Terraform reference from website-builder
(`cloud_run_website.tf:115`).

**Coordinator approvals do not need the UI:** `ailang coordinator approve|reject --remote
gcp` uses the same processor (`cmd/ailang/coordinator_approvals_remote.go:181`); the UI's
approve/reject routes had 0 hits in 30 days. The notify daemon's links to the dashboard
are decorative (`internal/daemon/handlers.go:180`).

**Recommended (option b):** delete `ui/` (≈30k TS/TSX/CSS), the committed
`internal/server/dist`, the Dockerfile UI stage, `.github/workflows/dashboard-ui-build.yml`,
the Dependabot `/ui` ecosystem, `make/services.mk` UI targets, ≈7.7k lines of UI-only
handlers in `internal/server`, `internal/dashboard_transforms`; keep `/v1/*`, the hook and
exec ingest routes, `/api/observatory/*` and `/api/chains` reads (authenticated),
`/benchmarks/`, approvals intake, `/health`. ≈40k LOC across 2–3 PRs; no Terraform change if
the service name stays. Cost is not the reason (≈$2/month Cloud Run): ≈10 Dependabot PRs a
month, a per-PR CI gate, and attack surface are.

**Conflicts with an existing plan:** [m-dashboard-minimal-surface.md](m-dashboard-minimal-surface.md)
(target v0.53.0) plans *two screens over a work-ledger API*. The usage numbers above argue
for zero screens; that doc's D8–D10 are unratified. Reconcile before either is executed.

**Side findings:** the `ui-builder` stage in `docker/Dockerfile.dashboard` is never copied
into the runtime image — prod serves the committed `internal/server/dist` (last changed
2026-10-22), so the npm build in Cloud Build is wasted work. ≈168 GiB (summed, before layer
dedup) of `dashboard:cache` buildx manifests sit in Artifact Registry with no retention rule.

**Security:** a dashboard security item was found during this audit and is being fixed
(auth on the dashboard's `/api/` routes, default-deny with an explicit ingest/public
allow-list and a router-wide regression test). Details are in the private ops repo
(`ailang-multivac`, `docs/ops/incidents/2026-11-09-dashboard-api-auth.md`), not here.

## 4. Proposed workstreams (awaiting Mark's go, except the security fix)

| # | Workstream | Shape | State |
|---|---|---|---|
| S | Dashboard API auth hotfix | 1 PR + a prod deploy (prod dashboard deploys are release-gated) | **in flight** |
| 1 | Security alerts | drop the ollama Go module for local types; bump zod; triage the 30 high CodeQL; secret scanning = owner setting | proposed |
| 2 | Sonar back to green | new-code coverage ≥ 80; triage 203 vulns / 7 bugs with `sonarcloud-triage` | proposed |
| 3 | Dead code | per-package deletion batches, each cross-checked with `deadcode -test=true` + `git log -S`, `make test` between batches | proposed |
| 4 | Dashboard UI removal (option b) | 2–3 PRs, after S lands; reconcile with m-dashboard-minimal-surface first | proposed |
| 5 | Audit becomes practical | add tracked metrics: unreachable-function count, Sonar gate + ratings, CodeQL-high + Dependabot counts, a per-service "used or not" check (request classes per Cloud Run service) | proposed |
| — | Platform split | move the Internal tier behind a module/binary boundary so the language is the small, clean part of the repo | needs a design doc; larger than everything above |

## 5. Open questions for Mark

1. Fix path to prod for S: a v0.57.5 patch release, or a manual prod rollout of the dashboard image?
2. Go on workstreams 1, 2, 3, 5?
3. Was the 2026-10-22 prod browser session his, and does he use `localhost:1957` locally (local GETs are not logged, so local UI use is unmeasured)?
4. Delete `headless-runner` (unlisted skill, 67 stale lines) or migrate it?
5. Enable GitHub secret scanning?
