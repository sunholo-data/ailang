# M-DASHBOARD-MINIMAL-SURFACE: two screens, a four-package runtime

**Status**: Planned
**Target**: v0.53.0
**Priority**: P2 (maintenance load and security noise, not a user-facing defect)
**Estimated**: 3–4 days across three phases (revised after design quorum, see Review Record)
**Dependencies**: None blocking. Continues [dashboard recovery plan](../dashboard-recovery-plan-2026-09-08.md) (deletion sprint A shipped in `bd06e6534`); independent of its Work-API contract sprint (step 3), which this doc neither needs nor replaces.

## Problem Statement

The dashboard UI (`ui/`, ~1% of the repository) produces a large share of the
repository's dependency and security-alert traffic, and almost none of that traffic
concerns code a user ever runs.

**Current state (measured 2026-10-06 at origin/dev unless marked):**

- **Alert and PR volume.** All-time Dependabot alerts: 220. `docs/` (Docusaurus) 115,
  `ui/` 59, Go 36, other 10. Dependabot PRs: `ui/` 56, `docs/` 68. Since deletion
  sprint A (2026-09-08), 7 of the 10 commits touching `ui/` are dependency bumps.
- **The alerts are in tooling, not the shipped app.** Of the 59 `ui/` alerts, 28 come
  from the eslint family (`eslint-plugin-react`, `@typescript-eslint/*`,
  `eslint-plugin-react-hooks`: brace-expansion ×13, minimatch ×6, …), 16 from the
  vite/vitest toolchain, and 15 from `firebase`. All 15 firebase alerts (protobufjs
  ×10 incl. 1 critical, `@grpc/grpc-js` ×4, `@protobufjs/utf8` ×1) arrive only
  through `@firebase/firestore`, which the UI never imports (`ui/src` imports only
  `firebase/app` and `firebase/auth`). The shipped bundle contains no
  `grpc`/`protobuf` strings. **Zero** alerts trace to a dependency that actually
  ships in the browser (`react`, `react-dom`, `react-markdown`, `highlight.js`).
  57 of the 59 are already `fixed`; the 2 open ones are both `@grpc/grpc-js` via
  Firestore (V8).
- **Dependency bumps never reach production.** Every `ailang` binary embeds a
  committed bundle (`//go:embed dist`, `internal/server/server.go:95`), last
  refreshed 2026-09-08 (`bd06e6534`). `docker/Dockerfile.dashboard` has a
  `ui-builder` stage (`npm ci && npm run build`) that no later stage copies from, and
  the release and CI binaries are built with a plain `go build` (V14). So every `ui/`
  Dependabot PR costs a CI run and a review, and changes nothing that is deployed.
  `ui/src` changes since that date also have not shipped. Nothing detects that the
  committed bundle is stale.
- **Tooling blocks its own upgrades.** `eslint-plugin-react@7.37.5` declares peer
  `eslint "^3 || … || ^9.7"`, so the eslint 10 bump fails `npm ci` with ERESOLVE.
  That happened in #371 (pinned back in #1363) and again in #1591 (open, failing the
  now-required `UI build gate`).
- **Install footprint.** `ui/package-lock.json`: 494 packages (178 runtime, 316 dev).
  `eslint-plugin-react` alone reaches 209.
- **Usage is small and concentrated.** Prod Cloud Run logs, 30 days: apart from the
  2026-09-08 audit session, about 6 browser sessions from about 4 IPs. They hit
  approvals (`/api/approvals`, task events, diff), work inspection
  (`/api/controlplane/exec-hierarchy`, `/api/chains/by-task`), header stats and auth.
  Dev had no browser API traffic. No product analytics exist, so this is a lower
  bound on interest, not a census.
- **Dead weight inside the app.** `ui/src` is 30,284 lines (about 12.5k CSS). 1,935 TS
  LOC sit in modules nothing uses (V13). 976 of them are in 10 files the TypeScript
  resolver cannot reach from `main.tsx` (`ApprovalPanel`, `ChatPreview`,
  `SessionsNav`, `BudgetStatus`, `CliCommandDisplay`, `displayNames`, three barrel
  `index.ts`). 959 are in 5 modules reachable only through barrel re-exports whose
  every export is unused (`useExecHierarchyState`, `useSessionsData`,
  `FilterIndicator`, `GlobalStats`, `CommandBar`). On top of that, nine hooks in
  `useObservatory.ts` are never called. The agent `DetailPanel` can never open
  (`handleAgentClick` / `handleNodeSelect` are defined at `ControlPlane.tsx:357,366`
  and never passed to an element), yet `useTopologyData` polls
  `/api/controlplane/exec-hierarchy` every 5 s to feed it.
- **Shipped-config bug found on the way.** Firebase web config is compiled in at UI
  build time (`ui/src/firebase/config.ts:9-14`, `VITE_FIREBASE_*` with `ailang-dev`
  defaults). The committed bundle therefore carries the **ailang-dev** config
  (`authDomain` `ailang-dev.firebaseapp.com`), and prod serves it. The Go server
  already takes its Firebase project at runtime (`WithFirebaseAuth(projectID)`,
  `internal/server/server.go:342`), so browser and server can disagree about which
  project they are in.

**Impact:** maintainers and agents triage roughly two `ui/` alerts and PRs a week for
code that is not deployed. Security signal is diluted: a critical `protobufjs` alert
reads as a dashboard vulnerability when no protobuf code ships. The dashboard's real
jobs (approve agent work; inspect what a task did) are buried in a 30k-line app that
mostly renders empty analytics.

## Goals

**Primary goal:** reduce the dashboard UI to the two surfaces people use, Approvals
and Work, with a four-package runtime (react, react-dom, @firebase/app, @firebase/auth) and a dev toolchain small
enough that Dependabot traffic for `ui/` falls to near zero. Make what is in `ui/`
the thing that ships.

**Success metrics:**
- `ui/package-lock.json` ≤ 150 installed packages (from 494).
- Runtime deps are exactly four packages: `react`, `react-dom`, `@firebase/app` and
  `@firebase/auth`. No `firebase` umbrella, `react-markdown` or `highlight.js`.
- Open Dependabot alerts on `ui/` = 0 after Phase 2 (today: 2 open, both
  `@grpc/grpc-js` via Firestore; the other 57 are already fixed). Over the following
  60 days, ≤ 1 `ui/` Dependabot PR per month (grouped).
- `ui/src` ≤ 12k lines (from 30.3k), with no unused files or exports (enforced in CI).
- The committed bundle can never silently diverge from `ui/`. A CI gate fails when
  the bundle's recorded source hash does not match `ui/`. Every binary build path,
  whether release, CI, local or image, keeps a real UI, and no build ever serves a stub.
- Firebase web config is served at runtime from the same source as the server's
  project ID. One image is promoted unchanged from dev to test to prod.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|---|---|---|---|---|
| D1. Retained surface = Approvals + Work (+ header status) | Defines everything deleted. Analytics, outliers, topology, sessions and aggregation facets go | human | design | high |
| D2. Lint toolchain: drop `eslint-plugin-react` and `@typescript-eslint/*`; type-check with `tsc --noEmit`; keep `eslint` + `eslint-plugin-react-hooks` only, or replace eslint entirely with Biome (one binary) | 28 of 59 alerts and the eslint-10 blocker live here | human | design | med |
| D3. How the shipped bundle tracks `ui/`: keep the committed bundle plus a source-hash freshness gate (recommended), vs build the UI in every binary build path | Release/CI binaries use plain `go build` (V14), and so does local `ailang serve`. An untracked bundle would ship a stub UI in all of them | human | design | med |
| D7. Firebase web config: served by the Go server at runtime from its environment, vs compiled into the bundle | Compiled-in config breaks one-image promotion (dev → test → prod copies images) and is how the dev config reached prod | human | design | med |
| D4. Markdown/diff rendering without `react-markdown`/`highlight.js` | Approval detail renders agent summaries and diffs. Plain `<pre>` with line-level +/- colouring vs a ~1 KB in-repo renderer | agent | design | low |
| D5. Auth: modular `@firebase/auth` (keep Firebase sign-in) vs moving auth to the server (IAP or Go-side OAuth) and dropping Firebase from the browser | Firebase is 87 packages and all 15 firebase alerts. Server-side auth removes the dependency but changes the deploy | human | design | high |
| D6. Remove Go dashboard API routes with no caller (≈80 routes) in this milestone or a follow-up | Shrinks the exposed surface scanners already probe; touches `internal/server` broadly | human | design | med |

### Design Freeze

- [ ] D1: retained surface is Approvals + Work + header status (task count, budget, sign-in, LIVE).
- [ ] D2: lint toolchain choice (recommended: `tsc --noEmit` + eslint core + `eslint-plugin-react-hooks`; Biome as the alternative).
- [ ] D3: committed bundle + source-hash freshness gate (recommended), not untracking `dist/`.
- [ ] D7: runtime config from the server (recommended).
- [ ] D5: auth path (recommended for this milestone: modular `@firebase/auth`; server-side auth as Future Work).
- [ ] D6: dead-route removal now or follow-up (recommended: follow-up milestone, which produces the full route inventory; Future Work lists the candidates found so far).

## Solution Design

### Overview

Three phases, each self-consistent at merge time. Phase 1 lands the bundle-freshness
gate first, then deletes UI that is unused or not retained. Phase 2 cuts the
dependency tree to what the retained UI needs. Phase 3 moves Firebase config to
runtime and tightens Dependabot. **Every phase ends by committing a rebuilt bundle
(`make ui-deploy`), and from Phase 1 onward the gate refuses a stale one.** No
merge point leaves the committed `dist/` out of step with `ui/`.

### Conflict surface: who consumes `internal/server/dist`

| Consumer | Build path | Effect of this design |
|---|---|---|
| Cloud Run dashboard image | `cloudbuild-dev.yaml:120` / `cloudbuild-release.yaml:148` → `Dockerfile.dashboard` `go-builder` | Unchanged: embeds the committed bundle |
| Release binaries (GitHub releases) | `.github/workflows/release.yml:211`, plain `go build` | Unchanged: embeds the committed bundle. This is why `dist/` must stay committed |
| CI build matrix | `.github/workflows/build.yml:84`, plain `go build` | Unchanged |
| Local `ailang serve` / `make install` | plain `go build` | Unchanged; sign-in is disabled when no Firebase config is supplied (already the behaviour of `isFirebaseConfigured`) |
| Ingest clients (`/v1/*`, `/api/exec/*`, hooks) | n/a | Unaffected: no route removal in this milestone |

### Retained surface (D1)

| Surface | Keeps | Endpoints |
|---|---|---|
| **Header** | Task/budget tiles, sign-in, approvals badge, LIVE indicator | `/api/controlplane/stats` (poll ≥ 60 s), `/api/budget/status`, `/api/auth/whoami`, `/api/workspaces`, `/ws` |
| **Approvals** | Pending queue, detail modal, diff, event log, approve/reject/cancel with feedback | `/api/approvals*`, `/api/coordinator/tasks/{id}/events`, `/api/coordinator/tasks/{id}/diff` |
| **Work** | Inbox feed (left), chain list + detail (`ChainExplorer`), selected task's span tree and transcript (`ExecHierarchy` tree mode, `ChatHistory`, `TraceWaterfall`) | `/api/inbox`, `/api/chains*`, `/api/observatory/spans*`, `/api/observatory/traces`, `/api/observatory/tasks/{id}/hierarchy`, `/api/claude-history/by-span`, `/session`, `/search` |

**Removed:** `AggregationNav` + `useBreakdownData` (breakdown returns empty data on
the cloud backend per audit F3), `EventDetail` outliers, `DetailPanel` +
`useTopologyData` (unreachable, 5 s poll), `ExecHierarchy` graph mode, the 1,935 TS LOC of
unreachable modules, the unused hooks in `useObservatory.ts`, the hardcoded
`v0.6.4` footer (replace with `/api/version`), and CSS that only these used.

### Dependency target (Phase 2)

| Now | Target | Why |
|---|---|---|
| `firebase` (umbrella, 87 pkgs incl. firestore → protobufjs/grpc) | `@firebase/app` + `@firebase/auth` | UI uses only app and auth. Removes all 15 firebase alerts and the Firestore SDK |
| `react-markdown` (88 pkgs) | in-repo minimal renderer or escaped `<pre>` (D4) | Used in one modal |
| `highlight.js` | none; diff lines coloured by prefix | Used only in `DiffViewer` |
| `eslint-plugin-react` (209), `@typescript-eslint/eslint-plugin` + `parser`, `globals`, `@eslint/js` | `tsc --noEmit` + `eslint` + `eslint-plugin-react-hooks` (D2) | 28 alerts, the eslint-10 peer block. Type errors are caught by `tsc` |
| `vite`, `@vitejs/plugin-react`, `vitest`, `typescript`, `@types/react*` | unchanged | Needed. vite 8.3.2 bundles with `rolldown` 1.2.11, and the lock has no `esbuild`/`rollup` packages (V15). All 16 historical vite/vitest alerts are already fixed |

### Bundle freshness and runtime config (Phase 3, D3 + D7)

**Freshness gate (lands first, in Phase 1).** `internal/server/dist` stays committed,
because every binary build path embeds it (see Conflict surface). `npm run build`
also writes `dist/.source-hash`, a SHA-256 over two inputs:

- the LF-normalised contents of `ui/src/**`, `ui/index.html`, `ui/vite.config.ts` and
  `ui/tsconfig*.json`;
- a canonical, sorted `name@version` list read from `package-lock.json`, not from
  `npm ls` output, whose JSON shape varies across npm versions. The list covers the
  runtime dependencies **and** the toolchain that determines emitted bytes: `vite`,
  `rolldown`, `@vitejs/plugin-react`, `typescript`.

`make ui-deploy` copies the hash into `internal/server/dist/`. The `UI build gate` job
recomputes it from the PR's tree and fails, naming `make ui-deploy`, when it differs
from the committed one. Consequences:

- Lint and test tooling bumps (eslint, vitest, knip) are outside the hash and pass on
  their own.
- A change to `ui/src`, a runtime dependency or the bundling toolchain cannot merge
  without a rebuilt bundle, which closes the "changes never ship" gap. Dependabot
  cannot rebuild, so a grouped monthly bump touching hashed packages needs one
  `make ui-deploy` commit pushed onto it. If the output happens to be byte-identical,
  that commit only updates `.source-hash`. Deliberate and rare.
- `Dockerfile.dashboard`'s `ui-builder` stage stays. It is the gate's build target
  (`dashboard-ui-build.yml:110`), and BuildKit skips it when building the image,
  because it builds only the stages the final stage depends on (V22).

**Runtime Firebase config.** The server serves `/ui-config.js`
(`window.__AILANG_UI_CONFIG__ = {...}`) from its environment: the same project it
passes to `WithFirebaseAuth`, plus `AILANG_FIREBASE_API_KEY`,
`AILANG_FIREBASE_AUTH_DOMAIN` and `AILANG_FIREBASE_APP_ID`. These are public web
config values, not secrets. `index.html` loads it before the bundle.
`ui/src/firebase/config.ts` reads it, and keeps `VITE_FIREBASE_*` only for `npm run
dev`. No config means sign-in stays disabled, the current local behaviour. The
values are set per environment in ailang-multivac terraform (`terraform/cloud_run.tf`,
dashboard service env), so the image promoted from test to prod is byte-identical.

**Dependabot (`.github/dependabot.yml`, `/ui`):** move from `weekly` to `monthly`.
Use one group for all dev dependencies and one for runtime. Ignore semver-major bumps
of dev tooling (taken deliberately, not by bot). Security updates still arrive
immediately, because Dependabot security PRs ignore the schedule.

### Implementation Plan

**Phase 1: Freshness gate, then delete what is not retained** (~1.5 days)
- [ ] `ui/scripts/source-hash.mjs` (inputs above); `npm run build` writes `dist/.source-hash`; `make ui-deploy` copies it; `UI build gate` recomputes and compares. Prove it on throwaway PRs: fails on a `ui/src`-only edit, passes on an eslint-only bump.
- [ ] Delete the 15 unused modules and nine unused hooks (V13 lists), re-running `npx knip@5` after each removal batch to catch newly orphaned code.
- [ ] Add `knip` as the one new dev dependency, run from `npm run build` (unused files and exports fail the build). The zero-dependency alternative, a `tsc --listFilesOnly` reachability script, catches unused files but not unused exports (see Deferred Decisions).
- [ ] Remove `AggregationNav`, outliers, `DetailPanel`/topology, graph mode, and the 60 s/30 s/10 s/5 s polls they own. Keep one stats poll at ≥ 60 s. Footer reads `/api/version` (existing handler, V21) instead of the hardcoded `v0.6.4`.
- [ ] Delete CSS that only removed components used: whole `.module.css` files go with their component (knip reports them as unused files); rules inside the shared `ControlPlane.module.css` are pruned by class-name grep against the retained TSX.
- [ ] `make ui-deploy` and commit the rebuilt bundle, which also replaces the stale 2026-09-08 bundle. Record bundle size and `ui/src` LOC before/after.

**Phase 2: Cut the dependency tree** (~1 day)
- [ ] Swap `firebase` → `@firebase/app` + `@firebase/auth`; fix imports (3 files).
- [ ] Replace `react-markdown` and `highlight.js` (D4) in `ApprovalDetailModal` and `DiffViewer`. Escaping must stay safe: never `dangerouslySetInnerHTML` agent text.
- [ ] Apply D2's lint toolchain; `npm run build` runs `tsc --noEmit` then lint.
- [ ] Regenerate the lock with npm 10; confirm ≤ 150 packages and 0 open `ui/` alerts.
- [ ] `make ui-deploy` and commit the rebuilt bundle (the gate enforces this).
- [ ] Close the superseded Dependabot PRs (e.g. #1591).

**Phase 3: Runtime config and quiet the bot** (~1 day)
- [ ] `/ui-config.js` handler in `internal/server`, `config.ts` reads it, `index.html` loads it; Go test for the handler with and without config. `make ui-deploy` and commit.
- [ ] ailang-multivac: dashboard env vars per environment in terraform (dev → test → prod by branch), then verify each environment serves its own `authDomain`.
- [ ] Update `.github/dependabot.yml` for `/ui` as above.
- [ ] Deploy to dev, check sign-in + one approval detail + one chain detail in a browser, then promote through the normal release path.

### Files to Modify/Create

- `ui/src/features/controlplane/ControlPlane.tsx` — drop removed panels, unused handlers and polls (~−400 LOC)
- `ui/src/features/controlplane/ControlPlane.module.css` — drop rules for removed panels (~−2,000 LOC)
- `ui/src/hooks/useObservatory.ts` — remove unused hooks (~−450 LOC)
- `ui/src/components/DiffViewer/DiffViewer.tsx` — prefix-coloured diff without highlight.js (~±80 LOC)
- `ui/src/features/approvals/ApprovalDetailModal/ApprovalDetailModal.tsx` — replace react-markdown (~±60 LOC)
- `ui/src/firebase/` — modular `@firebase/app`/`@firebase/auth` imports, env-driven config (~±30 LOC)
- `ui/knip.json` — knip config (entry `src/main.tsx`) (~10 LOC)
- `ui/package.json`, `ui/package-lock.json`, `ui/eslint.config.js` — dependency and lint changes; `build` writes `dist/.source-hash`
- `ui/index.html` — load `/ui-config.js` before the bundle (~1 LOC)
- `ui/scripts/source-hash.mjs` — hash computation shared by build and gate (~40 LOC)
- `internal/server/ui_config.go` + `_test.go` — `/ui-config.js` handler (~60 + ~60 LOC)
- `internal/server/dist/` — refreshed committed bundle including `.source-hash`
- `make/services.mk` — `ui-deploy` copies `.source-hash` (~2 LOC)
- `.github/workflows/dashboard-ui-build.yml` — freshness comparison step (~15 LOC)

- `.github/dependabot.yml` — `/ui` schedule and groups (~15 LOC)
- Deleted: ~25 unreachable/retired files under `ui/src` (1,935 TS LOC unused per V13, plus retired panels)

**Cross-repo (sunholo-data/ailang-multivac, deployed by branch push):** per-environment Firebase web-config env vars on the dashboard service in terraform (`cloud_run.tf` + each environment's tfvars, ~20 LOC). Land it before the Phase 3 image reaches each environment.

## Examples

### Example 1: a Dependabot alert on a dev tool

**Before:** `protobufjs` critical alert on `ui/package-lock.json` → maintainer triages
a "critical dashboard vulnerability" → PR merges → nothing deployed changes.

**After:** `protobufjs` is not in the tree (no Firestore SDK). A remaining dev-tool
advisory produces at most one grouped monthly PR, and the gate builds the same bundle
that ships.

### Example 2: approving agent work

**Before and after (unchanged behaviour):** sign in → approvals badge → detail modal
shows summary, file tree, diff, event log → approve with feedback. Only rendering
changes: plain-text summary, prefix-coloured diff.

## Success Criteria

- [ ] `ui/package-lock.json` ≤ 150 packages; runtime deps limited to react, react-dom, @firebase/app, @firebase/auth
- [ ] 0 open Dependabot alerts on `ui/` after Phase 2 (`gh api repos/sunholo-data/ailang/dependabot/alerts?state=open` filtered to `ui/`)
- [ ] Import-closure test passes (no orphan modules), UI tests and `UI build gate` green
- [ ] `ui/src` ≤ 12k lines
- [ ] Freshness gate fails a PR that edits `ui/src` without `make ui-deploy`, and passes a dev-tooling-only bump (both demonstrated on throwaway PRs)
- [ ] Release-path binary (`go build ./cmd/ailang`) serves the real UI, not a stub
- [ ] Each environment serves its own Firebase `authDomain` from `/ui-config.js`, and the image digest promoted test → prod is unchanged
- [ ] Browser check on dev: sign-in, approval detail with diff, approve/reject reachable, chain detail with span tree
- [ ] Ingest endpoints unaffected: `/v1/traces`, `/v1/metrics`, `/api/exec/*`, `/api/hooks/claude`, `/api/observatory/hooks`, `/benchmarks/` still served (these carry 86% of sampled prod requests)
- [ ] All tests passing; changelog fragment added

## Testing Strategy

**Unit tests:** knip in the build; `DiffViewer` renders +/− lines with classes and
escapes HTML; approval summary renders `<script>` payloads as text; `config.ts`
reads `window.__AILANG_UI_CONFIG__` and stays signed-out without it; Go tests for
`/ui-config.js` with and without env.

**Integration tests:** source-hash script gives the same hash on macOS and Linux CI
for the same tree; the freshness gate's pass and fail cases; existing Go server tests
for ingest routes unchanged.

**Manual testing:** dev deployment browser pass (criteria above). No real approval
is executed; use a fixture approval on dev.

## Deferred Decisions

- D4 rendering approach (escaped `<pre>` vs tiny in-repo markdown subset) — agent may choose, provided agent text is never injected as HTML.
- knip vs the zero-dependency `tsc --listFilesOnly` check for the CI unused-code gate — agent may choose; knip is recommended because it also catches unused exports (V13 shows 959 LOC that only an export-level check finds).
- Exact Dependabot groups and ignore rules — agent may choose within "monthly, grouped, majors of dev tooling ignored".
- Whether `ExecHierarchy` tree mode and `ChainExplorer` merge into one Work detail — left to deletion sprint B of the recovery plan, after its Work-API contract. This doc only removes what is unreachable or not retained.

## Non-Goals

- **Repairing cloud query contracts** (pagination, stage counts, missing indexes, audit F3/F4). That is recovery-plan step 3; this doc removes UI, it does not fix data.
- **Retiring the dashboard Cloud Run service.** It is also the OTLP/hooks ingest endpoint (`/v1/*` = 43,026 of a 50,000-request prod sample). Cost is already about $0.02–0.10 per service-week since prod moved to min-instances 0.
- **The docs site.** `docs/` (Docusaurus) is the larger alert source (115 all-time, 104 via `@docusaurus/*`), but it is a different build with different users. It deserves its own doc (Future Work).
- **New dashboard features or a framework change.**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| A removed panel was used by someone not visible in 30 days of logs | Low | Logs show ≈6 sessions, all on retained endpoints. Retired views stay in git history; restoring one is a revert |
| Dropping react-markdown/highlight.js opens an HTML-injection path | Med | Render agent text as text only; unit test with `<script>` payloads |
| Source hash differs across machines (line endings, npm resolution) | Med | Hash normalised file contents (LF) and `npm ls` resolved versions, not lockfile bytes; cross-platform test |
| Runtime-dependency Dependabot PRs fail the freshness gate | Low | Intended. Grouped monthly; an agent or maintainer adds the `make ui-deploy` commit |
| `/ui-config.js` missing or misconfigured in an environment | Med | Sign-in disables cleanly (no crash); success criterion checks `authDomain` per environment before promote |
| Removing `@typescript-eslint` loses type-aware lint rules | Low | `tsc --noEmit` covers type errors. Hooks rules are kept, which are the React rules that catch real bugs |

## Verification Log

| # | Claim | How verified | Result |
|---|---|---|---|
| V1 | `ui-builder` output is not copied by any later stage | `grep -n "FROM\|COPY --from" docker/Dockerfile.dashboard` | Only `COPY --from=go-builder` (lines 49, 57, 58); confirmed |
| V2 | Server embeds a committed bundle | `grep -n go:embed internal/server/server.go` → `:95 //go:embed dist`; `git log -1 -- internal/server/dist` → `bd06e6534` 2026-09-08 | Confirmed |
| V3 | UI imports only `firebase/app` and `firebase/auth` | `grep -rhn "from 'firebase\|@firebase" ui/src` → 3 imports, app/auth only | Confirmed |
| V4 | No grpc/protobuf in the shipped bundle | string search of served `index-*.js` | 0 matches; confirmed |
| V5 | `DetailPanel` handlers never wired | `grep -n "handleAgentClick\|handleNodeSelect" ControlPlane.tsx` → definitions at :357, :366 only | Confirmed |
| V6 | react-markdown / highlight.js have one consumer each | `grep -rln` → `ApprovalDetailModal.tsx`, `DiffViewer.tsx` | Confirmed |
| V7 | eslint-plugin-react blocks eslint 10 | lockfile `peerDependencies` = `^3 … ^9.7`; #1591 `npm ci` ERESOLVE | Confirmed |
| V8 | Alert attribution (59 ui alerts: 28 eslint, 16 vite/vitest, 15 firebase, 0 shipped) | Dependabot alerts API (all states) + package-lock dependency walk | Measured |
| V9 | Usage ≈6 browser sessions / 30 days in prod outside the audit day; 0 in dev | Cloud Run request logs (`gcloud logging read`, 30 d, per-path counts) | Measured; lower bound (no analytics) |
| V10 | Ingest endpoints carry most dashboard traffic | prod log sample: `/v1/*` 43,026 / 50,000 | Measured (capped sample) |
| V11 | Dev Firebase config shipped to prod | `authDomain` string in served bundle | Confirmed |
| V12 | `ui/` Dependabot config is weekly | `.github/dependabot.yml:83-87` | Confirmed |
| V13 | 10 files unreachable from `main.tsx` (976 LOC); 5 more modules (959 LOC) reachable only via barrels with every export unused; nine unused `useObservatory.ts` hooks | `tsc --listFilesOnly` with `files: [src/main.tsx]` (69 reachable of 79 non-test files) at `b7028a7cb`; cross-checked with `npx knip@5` (5.88.1) "Unused files (10)" + "Unused exports" | Confirmed by two independent resolvers |
| V14 | Release and CI binaries are built with plain `go build`, so they embed the committed `dist/` | `.github/workflows/release.yml:211`, `.github/workflows/build.yml:84`; the image path is `cloudbuild-dev.yaml:120` / `cloudbuild-release.yaml:148` → `Dockerfile.dashboard` | Confirmed |
| V15 | vite 8 uses rolldown, no esbuild/rollup in the lock | `package-lock.json` keys: `vite` 8.3.2, `rolldown` 1.2.11 + platform bindings, no `esbuild`/`rollup` entries | Confirmed |
| V16 | All 15 firebase alerts trace to `@firebase/firestore` | alerts by package: protobufjs 10, `@grpc/grpc-js` 4, `@protobufjs/utf8` 1; all three appear in the lock only under the firestore subtree | Confirmed |
| V17 | Firebase web config is build-time with dev defaults; server project is runtime | `ui/src/firebase/config.ts:9-14`; `internal/server/server.go:342` `WithFirebaseAuth(projectID)` | Confirmed |
| V18 | The retained UI needs no third-party runtime package beyond the four | every non-relative import in non-test `ui/src`: `react` (55), `react-dom/client`, `react-dom/server`, `firebase/app`, `firebase/auth`, `react-markdown` (1, replaced), `highlight.js/*` (12, replaced); `vitest` only in tests | Confirmed |
| V19 | Lock baseline 494 packages = 178 runtime + 316 dev | `package-lock.json` `packages` entries by `dev` flag | Confirmed |
| V20 | 7 of the 10 commits touching `ui/` since and including `bd06e6534` are dependency bumps | `git log bd06e6534^..origin/dev -- ui/`: #1587, #1590, #1537, #1368, #1363, #1120, #1117 | Confirmed |
| V21 | `/api/version` exists | `internal/server/server.go:576` `mux.HandleFunc("/api/version", s.handleVersion)` | Confirmed |
| V22 | Image builds do not run the `ui-builder` stage; the gate does | gate targets it (`dashboard-ui-build.yml:110`); cloudbuild uses `docker buildx` with no `--target`, and BuildKit builds only the stages the target depends on (Docker docs, "multi-stage builds") | Confirmed in config; BuildKit behaviour by documentation |
| V23 | Dashboard spend ≈ $0.02–0.10 per service-week since 2026-09-28 | GCP billing export, per-service weekly sum ÷ 6.415 DKK/USD | Measured |
| V24 | Dependabot security updates are not held to the version-update schedule | GitHub docs, "About Dependabot security updates": raised when an alert fires | By documentation (external) |

Quorum trigger: #1 fires (design-freeze items D1–D3, D5–D7).

## Axiom Compliance

| Axiom | Score | Justification |
|---|---|---|
| A1: Determinism | 0 | No language or runtime semantics touched |
| A2: Replayability | 0 | No trace format change; trace inspection retained |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | +1 | Removes the browser-side Firestore SDK and unreachable UI paths. Auth stays explicit (sign-in, server role check). D6 follow-up shrinks unauthenticated routes |
| A5: Bounded Verification | 0 | N/A |
| A6: Safe Concurrency | 0 | N/A |
| A7: Machines First | +1 | Retained surfaces mirror CLI/API (`ailang chains`, approvals API). No UI-only state is introduced, and the UI stops polling endpoints for views that cannot open |
| A8: Minimal Syntax | 0 | N/A |
| A9: Cost Visibility | 0 | Budget tile retained; empty breakdown analytics (which showed $0 on the cloud backend, audit F5) removed rather than kept misleading |
| A10: Composability | 0 | N/A |
| A11: Structured Failure | 0 | Unchanged |
| A12: System Boundary | +1 | The shipped bundle is provably built from the committed `ui/` (freshness gate), and environment identity crosses into the browser at runtime from the same source the server uses, replacing a compiled-in dev config |

**Net Score: +3** → **Decision: Move forward**

- [x] A1: no nondeterminism introduced
- [x] A3: no hidden side effects
- [x] A4: no ambient access granted (reduced)
- [x] A7: not optimising for human convenience over machine analysis

## Review Record

Design quorum, 2026-10-06 (`.ailang/state/mission-quorum/m-dashboard-minimal-surface-2026-10-06T06-48-47Z.json`):
`oc-glm-5-3`, `oc-kimi-k3` and `gemini-3-1-pro` rejected; `gpt6-1-sol` was absent (provider out of credits). Every objection was accepted:

| Objection | Revision |
|---|---|
| Unreachability claim had no verification row; bespoke import walker is circular | V13 from two independent resolvers (tsc, knip). knip replaces the hand-rolled walker; the figure is corrected from 2,716 to 1,935 TS LOC + nine hooks |
| Untracked `dist/` + placeholder would silently ship a stub UI | Placeholder dropped. Every binary path embeds `dist/` (V14), so D3 reversed: committed bundle + source-hash freshness gate |
| Build-time Firebase args break one-image promotion | D7: runtime `/ui-config.js` from the server's environment |
| vite/vitest alerts vs "0 open" criterion; firebase alert attribution; vite-8 aside; "three" vs four runtime packages | V8/V15/V16 added (57/59 already fixed, 2 open both firestore); title and metric say four packages |

Round 2 (re-quorum, used once): `oc-glm-5-3`, `oc-kimi-k3` and `gemini-3-1-pro` rejected again on narrower points, all accepted:

| Objection | Revision |
|---|---|
| Phases 1–2 would merge with a stale bundle; the gate only arrived in Phase 3 | Gate moved to the start of Phase 1; every phase ends with a committed `make ui-deploy` |
| Hash excluded the bundling toolchain, so a vite bump could silently change output | Hash now includes `vite`, `rolldown`, `@vitejs/plugin-react`, `typescript` from the lockfile, plus `tsconfig*`, in canonical `name@version` form |
| Four-package runtime closure unverified | V18: full third-party import list of `ui/src` |
| `ui-builder` stage called dead | Kept: it is the gate's target; image builds skip it (V22) |
| Unlogged secondary claims (package counts, commit ratio, `/api/version`, cost, Dependabot schedule, D6 inventory) | V19–V24 added; D6 freeze item reworded |

The re-quorum guardrail is spent, so per the design-doc-creator rule this goes to a
human for ratification (design-freeze items D1–D3, D5–D7) rather than a third round.

## Related Documents

- [Dashboard recovery plan (2026-09-08)](../dashboard-recovery-plan-2026-09-08.md) — parent plan; deletion sprint A shipped; this doc is a dependency- and surface-focused continuation, ordered before its deletion sprint B
- [Dashboard live audit (2026-09-08)](../dashboard-live-audit-2026-09-08.md) — F3/F5 justify removing the empty analytics facets
- [M-DASHBOARD-DELETE-ANALYTICS sprint](m-dashboard-delete-analytics-sprint-plan.md) — removed recharts/d3; 1,265 kB → 720 kB JS
- [m-dashboard-simplification (archived)](../archive/v0_29_0/m-dashboard-simplification.md) — earlier attempt
- [M-COST2 dashboard Firestore optimisation](../implemented/v0_9_8/m-cost2-dashboard-firestore-optimization.md)

## Future Work

- **D6 dead-route removal**: ≈80 Go routes with no UI, CLI or agent caller (`/api/threads*`, `/api/statistics`, most `/api/observatory/*` write routes, `/api/controlplane/heatmap|topology*|task-evolution|usage-timeseries|token-distribution`, `/api/claude-history/db/*`, `/api/select-folder`, …). Inventory: the 2026-10-06 research notes. Keep CLI-used routes (`/api/observatory/workspaces|agents|sessions`) and all ingest routes.
- **Server-side auth (D5 alternative)**: IAP or Go-side Google OAuth would remove Firebase from the browser entirely.
- **Docs-site dependency diet**: `@docusaurus/theme-mermaid` and `preset-classic` each reach ~1,200–1,300 of 1,416 packages; 104 of 115 docs alerts trace there.
- **Recovery-plan deletion sprint B**: one Work detail replacing `ExecHierarchy` + `ChainExplorer`, after the Work-API contract.

---

**Document created**: 2026-10-06
**Last updated**: 2026-10-06
