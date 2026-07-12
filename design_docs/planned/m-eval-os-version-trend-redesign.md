# M-EVAL-OS-VERSION-TREND-REDESIGN: local-rig version-trend, done right

**Status**: PLANNED — the old `OSVersionTrend` component was **removed** from `docs/docs/benchmarks/explorer.md` on 2026-07-12 after it consumed disproportionate effort without a reliable render. Redesign from scratch.
**Priority**: P3 — nice-to-have visualization; the underlying data is already correct and published. Do NOT reopen until higher-value eval work is done.
**Author**: Claude Opus 4.8 (with Mark, 2026-07-12 — "it's just not worth the tokens; reassess with a fresh approach")

---

## Goal
Show, on the benchmarks site, the **local-rig version-over-version trend**: for each on-device model×harness (opencode/Pi/motoko on local Qwen), its AILANG pass rate per AILANG release — "does each release move the needle for local models?"

## What went wrong (so the redesign doesn't repeat it)
The data and the component logic were both **correct the entire time**. The failure was operational, and self-inflicted:

1. **It showed nothing for weeks because the data file was never generated** — `docs/static/benchmarks/os/history.json` had no per-version rows until a backfill on 2026-07-11. That was ~90% of "it never works." (Now auto-refreshed every rotation cycle by `os-rotation-filler.sh` step 7b — keep that.)
2. **Then it was over-engineered.** The component was changed three times in one session — plain fetch → cache-busted fetch → **build-time JSON import** → back to fetch. Each change produced a *different* JS build.
3. **The interaction that actually broke it:** the site is **GitHub Pages behind Fastly**, which served a *mix* of builds across requests (a days-old bundle on one request, the latest on another). The build-time-import version had 1-model data **baked into the JS**; whenever a request landed on one of those stale bundles, it rendered 1 model regardless of the (correct) live data. The working tables were immune only because their component code never changed, so every build renders them identically.

Verified facts at removal time: `curl` and the browser's own `fetch` both returned 4 versions × 3 models; a React SSR render of the component produced 3 rows; but headless-Chrome (post-hydration) on the live site showed 1 model — an artifact of the served bundle, not the code.

## Rules for the fresh approach
1. **Copy a working table verbatim.** `OSLocalLeaderboard` reliably `fetch`es `/benchmarks/os/latest.json` and renders. Mirror its exact pattern (plain runtime `fetch`, no cache-busting, no build-time `import`). The version-trend just needs `os/history.json` instead.
2. **Never bundle changing data into the build.** Runtime-fetch only, so the component's JS chunk stays byte-identical across data updates — stale/mixed builds then render the *same* (correct) thing.
3. **Prefer folding into an existing, proven component** (a tab/toggle inside `BenchmarkExplorer` or the main dashboard) over a new standalone one — reuse a data path that already works end-to-end.
4. **Change it once, then stop.** Repeated component edits are what created the mixed-build mess. Land it once and leave it.
5. **Verify with headless Chrome post-hydration** (`--headless --dump-dom`), never SSR HTML alone. SSR looked correct throughout while the hydrated page was wrong.
6. **Build/verify on Node 20** (Docusaurus 3.10 target; the rig's Node 26 fails the build — unrelated but blocks local verification).

## Data (already correct — reuse as-is)
- `docs/static/benchmarks/os/history.json`: array of `{ ailang_version, generated, languages, rows: [{model, harness, lang:{ailang,...}}] }`, newest-first, auto-refreshed per rotation cycle (`tools/os-release-snapshot.sh`, no `--reset`).

## Out of scope
- Any change to the data pipeline — it's correct and publishing.
- Reviving the deleted component. Start clean per the rules above.
