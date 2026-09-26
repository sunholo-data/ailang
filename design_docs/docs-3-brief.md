# docs-3 sprint brief — wire the existing fallback-source badge into the 4 benchmark pages that silently omit it

**Not a design doc.** Per `docs-mission.md`'s Guardrails ("most items here need no design doc at
all — prefer a Gate-2 reality-check straight into a sprint"), this is a routing declaration only —
it exists so `tools/launchd/derive-planner-lane.sh` can route the planner instead of failing closed
to opus for a missing `Planner-Lane` field. It carries no new design claims (the mechanism this
sprint wires up already exists and is already proven on one page) and needs no quorum.

**Planner-Lane**: codex-ok

(All touched files are under `docs/src/**`, inside `MISSION_PLANNER_ALLOWLIST`'s `docs/*` pattern —
verified per D-2's control: `case` globs match `/`, so nested paths are reachable, and this is the
cheap codex lane, not opus.)

## Task — clause 6 (benchmark report maintenance), queue item docs-3, "benchmark surface audit"

Controller audit findings, both confirmed live against current HEAD (`grep`/`cat`, not assumed):

### Finding: 4 of 5 DataProvenance-carrying pages can never show the "stale data" badge they render

`docs/src/components/DataProvenance/index.jsx`'s own header comment records the mission this
machinery came from (M-EVAL-ROLLING-ELO M4): *"Before this, 5 of the benchmark pages (Model
Leaderboard, ELO, Explorer, Value, Gallery) rendered numbers with NO version or date, and 12 of 13
component fetch sites degraded silently to the in-build static copy... with no indicator."* That
mission built two pieces of machinery to fix it — `benchmarkFetchWithSource()` (returns
`{response, source}`, `source` is `'gcs'` or `'fallback'`) and `<DataProvenance source={...}>` (its
`isFallback` prop renders a `⚠ Fallback / stale data` badge) — and wired **both** into exactly ONE
of the five pages, `ValueDashboard/index.jsx` (tracks `dataSource` state via `benchmarkFetchWithSource`,
passes `source={dataSource}` into `DataProvenance`).

The other four pages already render `<DataProvenance version={...} timestamp={...} />` but never
pass a `source` prop, so `DataProvenance`'s `isFallback` is `source === 'fallback'` — always
`false` for these four, structurally. They still fetch via the plain `benchmarkFetch()` (silent
fallback, per that helper's own doc comment: falls back to the in-build static copy on any
network/CORS error or non-2xx with **no signal to the caller at all**). Net effect: these four
pages can silently serve the in-build static copy — which the same doc comment says "can be several
releases behind" — while displaying a `DataProvenance` strip that looks identical to a fresh read,
because the one thing that would flip it (`isFallback`) is wired to a prop that is never passed.
This is a live instance of clause 6's own bar ("current and honest") and of Critical Principle 2
("no silent fallbacks") on the exact surface that principle was written for — release/benchmark
claims a reader uses to form beliefs about the project.

Confirmed which four, live grep against HEAD:
```
$ grep -rn "<DataProvenance" docs/src/
docs/src/components/BenchmarkExplorer/index.jsx:527:      <DataProvenance version={data?.version} timestamp={data?.timestamp} />
docs/src/components/EloLeaderboard/index.jsx:170:      <DataProvenance version={data?.version} timestamp={data?.timestamp} />
docs/src/components/ValueDashboard/index.jsx:70:      <DataProvenance version={data?.version} timestamp={data?.timestamp} source={dataSource} />
docs/src/components/BenchmarkStandaloneGallery/index.jsx:23:      <DataProvenance version={data.version} timestamp={data.timestamp} />
docs/src/components/BenchmarkDashboard/index.jsx:297:      <DataProvenance version={version} timestamp={data?.timestamp} />
```
Only `ValueDashboard` carries `source={dataSource}`. The other four import `benchmarkFetch` (not
`benchmarkFetchWithSource`) and have no `dataSource`/equivalent state at all.

### Fix — replicate ValueDashboard's exact, already-proven pattern in the other four files

For each of the four files below: swap the `benchmarkFetch` import for `benchmarkFetchWithSource`,
add a `dataSource` state (`useState(null)`), and wire it through the fetch chain and into
`<DataProvenance source={dataSource} ... />`. Match `ValueDashboard/index.jsx`'s existing code
exactly in shape (same helper, same prop name, same `.then(({ response, source }) => ...)`
destructuring) — do not invent a new pattern.

1. **`docs/src/components/EloLeaderboard/index.jsx`** (line ~96) — single `benchmarkFetch('latest.json')`
   call, structurally identical to ValueDashboard's. Direct swap.
2. **`docs/src/components/BenchmarkStandaloneGallery/index.jsx`** (line ~11) — single
   `benchmarkFetch('latest.json')` call. Direct swap.
3. **`docs/src/components/BenchmarkDashboard/index.jsx`** (line ~137) — single
   `benchmarkFetch('latest.json')` call feeding the Model Leaderboard. Direct swap.
4. **`docs/src/components/BenchmarkExplorer/index.jsx`** (lines ~484-489) — the one non-trivial
   case: TWO fetches combined via `Promise.all` (`latest.json` as `base`, `os/latest.json` as
   optional `os`, unioned by `mergeOSData`). `data?.version`/`data?.timestamp` passed to
   `DataProvenance` come from `base` (the `latest.json` read) — track and pass the SOURCE of that
   `latest.json` fetch only; leave the `os/latest.json` fetch's existing optional/fallback-to-null
   handling untouched (it is a different, already-graceful degradation, not this defect).

Do NOT touch `benchmarkFetch.js` or `DataProvenance/index.jsx` themselves — both already do exactly
what this fix needs; only the four call sites are missing the wiring. Do NOT touch the other
`benchmarkFetch` call sites found in the same audit (`OSReleaseTrend`, `AgentUpliftTable`,
`osHistory.js`, `OSLocalLeaderboard`, `ModelComparisonTable`) — none of those five render
`DataProvenance` at all, so wiring a `source` through them has no visible effect and is a
separate, un-scoped change (M-EVAL-ROLLING-ELO M4's own stated scope was these five
DataProvenance-bearing pages, not every benchmark fetch site in the tree).

### Acceptance

- All four files import `benchmarkFetchWithSource` (not `benchmarkFetch`) and pass a live
  `source` prop into their `DataProvenance` call — grep `source={` on all five `<DataProvenance`
  call sites must return 5 (currently 1).
- Manual/mutation check: force one file's `benchmarkFetchWithSource` call to resolve
  `{ source: 'fallback' }` (e.g. by temporarily making the remote fetch reject) and confirm the
  `⚠ Fallback / stale data` badge renders on that page — do this for at least one of the four,
  proving the wiring is live rather than merely present (a prop that is passed but never actually
  varies would be the same vacuous-pass shape as the bug this fixes). Restore afterward.
- `make docs-build` still green (no build-time regression from the added state/prop).
- No change to `benchmarkFetch.js`, `DataProvenance/index.jsx`, or any of the five
  DataProvenance-less fetch sites named above (diff scope check).
- CHANGELOG entry not required (internal consistency fix to already-shipped, non-public-API
  frontend plumbing — same scoping precedent as docs-1's router).

## Files

- `docs/src/components/EloLeaderboard/index.jsx`
- `docs/src/components/BenchmarkStandaloneGallery/index.jsx`
- `docs/src/components/BenchmarkDashboard/index.jsx`
- `docs/src/components/BenchmarkExplorer/index.jsx`

## Deferred, not in this sprint (recorded so it is not rediscovered as new)

A second, larger clause-6 finding from the same audit: `os/history.json`'s time-series trend
charts (`PerModelTrend.jsx`, `ModelDeltaTrend.jsx`, both via `osHistory.js`/`useEvents.js`) plot
the `local` provider line continuously across all 9 committed history entries (2026-07-12 through
2026-08-24), which spans all **three** known non-poolable local-model measurement boundaries this
mission's own charter warns about (07-21..08-03 untuned flags, 08-13 budgets+num_ctx, 08-17 ollama
0.32.1→0.32.14 — see `docs-mission.md` clause 6 text and memory
`project_ollama_split_brain_eval_confound`). `benchmarks/events.yml`'s annotation schema already
supports dashed `ReferenceLine` markers on these exact charts, but only for `kind: benchmark_add |
benchmark_remove | taxonomy | prompt` — there is no `kind` for a measurement-methodology
discontinuity, so the three boundaries render with no visual signal at all, unlike suite-composition
changes which already get one. Fixing this means extending `events.yml`'s schema and adding three
entries, plus a new `kind` case in `useEvents.js`'s `annotationColor` — `benchmarks/events.yml` sits
outside `MISSION_PLANNER_ALLOWLIST` (repo-root `benchmarks/`, not `docs/*`), so it plans on opus, not
the cheap codex lane, and is a distinct enough change (new schema field, three data entries, a
color-mapping case) to warrant its own queue item rather than folding into this one. Queue as
**docs-3b** or fold into a future clause-6 sprint.
