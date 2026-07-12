# M-DYNAMIC-DATA-RUNTIME-PLANE: generalize the runtime data plane to AILANG's other dynamic datasets

**Status**: PLANNED — **BLOCKED on precondition** (see callout). Follow-up to
[m-eval-data-hosting-decouple](m-eval-data-hosting-decouple.md) (that doc's **W6**).
**Target**: v0.31.x (dashboard / eval infrastructure).
**Priority**: P2 — the *acute* pain (mixed-build / stale-cache on data refresh) is already retired by the
benchmark decouple. This doc generalizes the reusable **plumbing** once and routes each remaining dynamic
dataset to the treatment it actually warrants. None is as urgent as the benchmark case was; see the honest
reframe under **Problem**.
**Author**: Claude Opus 4.8 (with Mark, 2026-07-12).

---

> ## ⛔ PRECONDITION (blocking — do not start until both are true)
> This work is **W6** of [m-eval-data-hosting-decouple](m-eval-data-hosting-decouple.md) and is explicitly
> gated on the benchmark case being proven *and promoted to prod*:
> 1. **Benchmark decouple green end-to-end on dev** — a full rig rotation cycle updates the live dashboard
>    within ~1 min with no site rebuild and no git commit (that doc's Acceptance criteria 1–4), and its
>    **W5** (retire the routine data-churn commits) has landed.
> 2. **Route + `DATA_BASE` promoted to prod** — the `/benchmarks/` read-through is deployed to the prod
>    `ailang-dashboard` service and `DATA_BASE` points at `dashboard.ailang.sunholo.com` (the benchmark
>    doc's *Custom-domain adoption* + *Generalization* sections). Until then the data plane runs on the
>    **dev** run.app URL only, and widening it to more datasets would spread a dev-only dependency.
>
> Rationale: proving one dataset on the real prod domain de-risks the shared handler, the CORS/cache
> headers, and the fallback contract before we point three more consumers at them.

---

## Problem

The benchmark decouple ([m-eval-data-hosting-decouple](m-eval-data-hosting-decouple.md)) proved a clean
pattern for frequently-changing data:

```
producer → gsutil cp JSON to a private GCS bucket
        → dashboard Cloud Run read-through  GET /benchmarks/<path>.json  (path-validated, global * CORS, Cache-Control max-age=60)
        → docs fetch at RUNTIME via benchmarkFetch.js  (DATA_BASE + in-build fallback)
```

That killed a whole class of bug: a data refresh no longer triggers a Docusaurus rebuild + GitHub Pages
deploy + Fastly propagation, so there is no mixed-build window and no stale-after-update.

The benchmark doc's *Generalization* section named three follow-on datasets — **eval run details**,
**telemetry / chain rollups**, **coordinator status** — as candidates for the same shape. This doc scopes
that migration.

### Honest reframe (audit, 2026-07-12)

An audit of the three named datasets found that the premise "migrate the *other* datasets off the static
site build" is only literally true for **one** of them, and that **none of the three currently forces a
docs rebuild**:

- The docs deploy workflow triggers only on `docs/**`, `internal/**`, `cmd/**`, `prompts/**`, `llms.txt`,
  `CHANGELOG.md`, `web/**`, `go.{mod,sum}` (`.github/workflows/docusaurus-deploy.yml`). The rebuild
  coupling was specific to the **aggregate benchmark JSONs under `docs/static/benchmarks/`** — the exact
  files the benchmark decouple already moved.
- **Eval run details** (`eval_results/baselines/**`) *are* git-committed, but at the **repo root**, and
  are **not** on the docs data path (nothing fetches them; they are the raw source that `eval-report`
  folds into the already-migrated aggregates). Their cost is **repo bloat + `dev`-branch churn**, not a
  site rebuild.
- **Telemetry / chain rollups** and **coordinator status** are **already off the docs build** — they live
  in the Collaboration Hub's runtime plane (Cloud Run Go server reading SQLite directly). Nothing baked,
  committed, or fetched by Docusaurus.

So this doc's real job is narrower and clearer than "migrate 3 more datasets off the build":

1. **Generalize the plumbing once** (one read-through handler gated by a prefix allowlist + one
   `dataFetch` helper), so adding any future dataset is a one-line allowlist edit plus a producer publish
   step — *not* a new handler.
2. **Apply it where it pays off**, per the treatment each dataset actually warrants (below). For two of the
   three, the payoff is a *new public / rebuild-free / DB-offloading surface*, **not** un-baking existing
   data. The doc says so plainly rather than implying a rebuild-churn win that isn't there.

## Principle (inherited)

**Frequently-changing data does not belong in a statically-built, CDN-cached site** — and, more broadly,
does not belong in **git** (repo bloat, churn commits) or behind a **live DB read** for public/high-fanout
views when a cheap object write + read-through gives near-live, rebuild-free, cache-friendly delivery.
Decouple the data plane from both the code plane and the interactive control plane.

## The proven pattern (recap — reuse verbatim)

| Layer | Where it lives today |
|-------|----------------------|
| **Bucket** | `gs://ailang-multivac-dev-benchmarks` (private; org policy `storage.publicAccessPrevention` enforced), env override `BENCHMARKS_BUCKET` |
| **Read-through route** | `internal/server/handlers_benchmarks.go` — `handleBenchmarks` streams `gs://<bucket>/<path>` for `GET /benchmarks/<path>.json`; safe-path regex `^[a-zA-Z0-9_./-]+\.json$` (:27), no `..`/leading-`/`; registered at `internal/server/server.go:490` |
| **Headers** | global `*` CORS (`server.go:722`) + per-handler `Cache-Control: public, max-age=60` |
| **Runtime fetch** | `docs/src/lib/benchmarkFetch.js` — `DASHBOARD_BASE`/`DATA_BASE` + remote→in-build fallback; returns a `Response` so call sites are unchanged |
| **Producer sync** | `tools/launchd/os-rotation-filler.sh` step 9 (:305–326) — `gsutil -h "Cache-Control:public,max-age=60" cp` each cycle, independent of `AUTOPUSH`, opt-out `BENCH_BUCKET_SYNC=0` |

## Dataset inventory (audited 2026-07-12)

| # | Dataset | Producer | Current publish path | On docs build? | Forces rebuild? | Treatment |
|---|---------|----------|----------------------|:--:|:--:|-----------|
| 0 | **Aggregate benchmark JSONs** (`latest.json`, `os/latest.json`, `os/history.json`) | `ailang eval-report` / `eval-publish` → rig | **MIGRATED** (bucket + `benchmarkFetch`, git as fallback) | was yes | was yes | — (reference row) |
| 1 | **Eval run details** (`eval_results/baselines/**` ≈ 17.4k JSON + `eval_results/performance_tables/*.json`) | `internal/eval_harness/metrics.go:222` per-run writer, driven by `ailang eval-suite` (`make eval`); rig `os-rotation-filler.sh`, `nightly-eval.sh` | **git-committed at repo ROOT** (`.gitignore:86–88` allowlist `!eval_results/baselines/`); folded into aggregates by `eval-report` | no | **no** (not in deploy paths) | **A — retire git-churn → bucket** |
| 2 | **`axiom_scorecard.json`** | `ailang axioms` (`cmd/ailang/axioms.go` `//go:embed`), synced by `make/build.mk:16` into `docs/static/benchmarks/` | baked into build; raw `fetch('/benchmarks/axiom_scorecard.json')` (`AxiomScorecard.jsx:12`); **not** in the migrated `benchmarkFetch` set, **not** bucket-synced | **yes** | **yes** (release cadence) | **B — fold into existing data plane (quick win)** |
| 3 | **`codebase_stats.json`** | `tools/generate_codebase_stats.sh` at CI deploy time (`docusaurus-deploy.yml:109`) | regenerated fresh each build; raw `fetch('/codebase_stats.json')` (`CodebaseStats.jsx:13`) | yes | build-coupled, but **regenerated fresh → no stale-churn** | **C — low priority (optional)** |
| 4 | **Telemetry / chain rollups** | `ailang chains stats --json` (`cmd/ailang/chains_stats.go`), `trace`, `observatory` → STDOUT; Hub `GET /api/chains/stats`, `/api/controlplane/stats`, `/api/observatory/*` reading `collaboration.db` (`server.go:315`) | **not on docs build** — Hub-live, DB-direct | **no** | no | **D — net-new public snapshot (contingent)** |
| 5 | **Coordinator status** | `ailang coordinator status --json` (`coordinator_lifecycle.go:293`); Hub `GET /api/coordinator/status` (`server.go:519` → `handlers_monitor.go:256`), daemon HTTP `/status` — reads `coordinator.db` | **not on docs build** — Hub-live, DB-direct | **no** | no | **D — net-new public snapshot (contingent)** |
| 6 | `microrag_ab.jsonl` | appended by `nightly-eval.sh:203` into `docs/static/benchmarks/` | committed, **never fetched** by any component | yes (orphan) | yes | **cleanup — delete or wire a consumer** |
| 7 | Package registry | `docs/scripts/sync-registry.sh` (build snapshot) + live validator API | already runtime: `useRegistryData.js` live API + static-snapshot fallback | snapshot only | no | — (already decoupled, different pattern) |

## Shared plumbing (do this once — W1)

### Server: widen the handler's prefix allowlist (NOT per-dataset handlers)

Per the directive, generalize the **one** existing handler rather than adding a handler per dataset. The
safe-path regex already accepts nested paths (`eval/…`, `telemetry/…`, `coordinator/…`); the only things
hardcoded to "benchmarks" are the route string and the (implicit) single prefix.

- Rename/generalize `handleBenchmarks` → `handleData`, gated by an **allowlist of top-level dataset
  segments** served from the same bucket:

  ```go
  // internal/server/handlers_benchmarks.go (→ handlers_data.go)
  var dataDatasets = map[string]bool{
      "benchmarks": true, "eval": true, "telemetry": true, "coordinator": true,
  }
  // GET /data/<dataset>/<subpath>.json  →  gs://<bucket>/<dataset>/<subpath>.json
  // first path segment must be in dataDatasets; rest must match the existing safe regex.
  ```

- Register **one** umbrella route `mux.HandleFunc("/data/", s.handleData)` alongside the existing
  `/benchmarks/` registration (`server.go:490`). **Keep `/benchmarks/` as a back-compat alias** — the
  already-shipped `benchmarkFetch.js` and the in-build fallback URLs both call
  `.../benchmarks/latest.json`, and benchmark objects live at the **bucket root** (not under a
  `benchmarks/` prefix), so the alias maps `benchmarks/<path>` → bucket-root object. New datasets get
  their own top-level prefixes (`eval/`, `telemetry/`, `coordinator/`) in the **same** bucket. (Folding
  benchmarks under `/data/benchmarks/` is a later cosmetic cleanup, not required.)
- Unchanged: same bucket + `BENCHMARKS_BUCKET` env, same regex, same global `*` CORS, same
  `Cache-Control: max-age=60`, same GET/HEAD-only.
- Result: adding dataset N is a **one-line allowlist edit** + a producer publish step. (Acceptance #5.)

### Frontend: generalize `benchmarkFetch` → `dataFetch`

Promote `docs/src/lib/benchmarkFetch.js` into a general helper:

```js
// dataFetch('eval/v0.31.0/summary.json', { fallback: false })
export async function dataFetch(path, { fallback = true, cacheBust = false, ...opts } = {}) {
  const rel = String(path).replace(/^\/+/, '');
  const q = cacheBust ? `?v=${Date.now()}` : '';
  try { const r = await fetch(`${DATA_BASE}/data/${rel}${q}`, opts); if (r.ok) return r; } catch {}
  return fallback ? fetch(`/${rel}${q}`, opts)      // in-build copy (benchmarks, codebase_stats)
                  : new Response(null, { status: 503 }); // no baked copy → caller renders empty state
}
export const benchmarkFetch = (rel, opts) => dataFetch(`benchmarks/${...}`, opts); // thin wrapper, unchanged call sites
```

Datasets **with** an in-build copy (benchmarks, `codebase_stats`) keep `fallback: true` → degrade to the
last build snapshot on outage (Acceptance #4). Datasets **without** one (telemetry, coordinator) pass
`fallback: false` → the component renders an explicit empty / "live data unavailable" state, never a hard
error.

### Producer side: one publish step

Factor the "publish JSON to the data plane" step (gsutil cp + the cache header, skip-if-no-gsutil,
never-fail-the-job) that `os-rotation-filler.sh` step 9 already implements into a reusable snippet/target,
so `eval-suite`, the rig, and any rollup job publish identically.

## Work items

- **W1 — Shared plumbing.** Generalize the server handler (allowlist) + register `/data/`; promote
  `benchmarkFetch.js` → `dataFetch`; factor the producer publish step. Land on dev, then **promote to prod
  with the benchmark route** (single deploy). *No behavior change for benchmarks.*
- **W2 — `axiom_scorecard.json` (Treatment B, the quick win).** Repoint `AxiomScorecard.jsx:12` from raw
  `fetch('/benchmarks/axiom_scorecard.json')` → `benchmarkFetch('axiom_scorecard.json')`; add
  `axiom_scorecard.json` to the rig/`post-release` bucket sync. Closes the **last un-migrated
  `docs/static/benchmarks/` raw-fetch** on the rebuild-critical path. Smallest change, highest leverage —
  do it first after W1.
- **W3 — Eval run details (Treatment A).** Have `eval-suite`/the rig publish per-run detail to bucket
  prefix `eval/<version>/…` via the shared step; **stop (or thin) the `eval_results/baselines/**` git
  commits** (they cost ~17.4k tracked files + `dev` churn and never reach the site). Keep a **release-time
  provenance snapshot** in git/versioned bucket path (mirror benchmark **W4**) so history stays
  reproducible. *Optional net-new:* a `BenchmarkExplorer` drill-down that fetches per-run detail from
  `eval/` (today it can only show what's embedded in the aggregate).
- **W4 — Telemetry rollups (Treatment D, contingent).** Only if a **public** telemetry surface (or a
  DB-offload for high-fanout reads) is wanted: a periodic job runs `ailang chains stats --json` /
  observatory rollups → `telemetry/rollup.json` → bucket; a docs component fetches via
  `dataFetch('telemetry/rollup.json', { fallback: false })`. Otherwise leave telemetry in the Hub
  (interactive/authed views stay DB-direct). **Requires redaction** (see Open questions).
- **W5 — Coordinator status (Treatment D, contingent).** Only if a **public status page** is wanted: a
  periodic job snapshots `ailang coordinator status --json` (public-safe / redacted) →
  `coordinator/status.json` → bucket; net-new docs status page fetches it. **Requires redaction** — the
  raw status carries task content, file paths, and cost figures.
- **W6 — Cleanup + docs.** Delete the orphan `microrag_ab.jsonl` (or wire a consumer); document the
  generalized `/data/` plane + `dataDatasets` allowlist in
  [database-architecture](../../docs/docs/guides/database-architecture.md).

## Acceptance criteria (mirror the benchmark doc)

1. **~1 min, no rebuild, no commit** — a producer update (eval run, rollup snapshot, axiom rescore) shows
   on the live site within ~1 min with **no site rebuild and no git commit**.
2. **Code-only changes unaffected** — a code-only site change still deploys via the normal GitHub Pages
   path.
3. **Class of bug cannot recur** — no dataset in scope is in the Docusaurus bundle, so the mixed-build /
   stale-after-update failure mode is structurally impossible for it.
4. **Graceful fallback** — a bucket / Cloud Run outage degrades gracefully: datasets with an in-build copy
   fall back to it; datasets without one render an explicit empty / "unavailable" state, never a hard
   error or a broken page.
5. **One handler, allowlist-gated** — the server exposes a **single** generalized read-through handler;
   adding a dataset is a one-line allowlist edit + a producer publish step, **not** a new handler
   (verified by review of `handlers_data.go` + `server.go`).
6. **No provenance loss** — retiring the `eval_results/baselines/**` git commits (W3) preserves a
   reproducible release-time snapshot (git or versioned bucket path).

## Design decisions / open questions

- **One bucket + prefixes vs bucket-per-concern** → **Recommend one bucket, prefixes.** Reuse
  `ailang-multivac-dev-benchmarks` (add `eval/`, `telemetry/`, `coordinator/`; benchmarks stay at root
  behind the legacy alias). A cosmetic rename to `…-dataplane` is *not* worth breaking existing objects
  and the shipped `DATA_BASE`.
- **Redaction for public coordinator / telemetry (W4/W5)** → **OPEN.** These carry task content, file
  paths, and cost figures. A public snapshot needs a redaction/whitelist producer. **Default: keep private
  (Hub only)** unless a public surface is explicitly requested — hence W4/W5 are contingent.
- **Prod `DATA_BASE` / custom domain** → inherits the benchmark doc's decision: flip `DATA_BASE` to
  `dashboard.ailang.sunholo.com` once the route is on prod (this doc's precondition).
- **Is a public eval-detail drill-down actually wanted?** → the aggregate already carries per-benchmark /
  per-model detail; W3's drill-down is optional and demand-driven. The non-optional half of W3 is the
  git-churn retirement.

## Out of scope

- Changing the docs host (stays GitHub Pages) or the **Hub** architecture — interactive/authed control-plane
  views stay Cloud Run + DB-direct; the data plane is for **public, read-only, high-fanout** snapshots.
- Reworking the **producers** of any dataset (eval pipeline, telemetry ingestion, coordinator daemon) — only
  *where the data is published* changes.
- Package registry (already runtime) and `badges/examples.json` (consumed by an external shields.io
  endpoint, not a docs component).

## Risks / notes

- **Don't oversell.** Two of the three named datasets are already off the build; W4/W5's payoff is a *new
  public / DB-offloading surface*, not rebuild-churn relief. Only W2 touches the rebuild-critical path, and
  the eval-detail win (W3) is a repo/git concern.
- **Redaction risk** for any public coordinator/telemetry snapshot — treat as a gate, not a footnote.
- **Never regress to empty** — preserve the in-build fallback for anything that had one; the `fallback:
  false` path must render a designed empty state, not a blank/error page.
- This unblocks re-adding richer live views (per-tier local breakdowns, a public mission/status surface)
  without the deploy-churn tax, exactly as the benchmark decouple did for the leaderboard.
