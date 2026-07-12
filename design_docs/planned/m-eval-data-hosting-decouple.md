# M-EVAL-DATA-HOSTING-DECOUPLE: serve benchmark data off the site build

**Status**: PLANNED — 2026-07-12.
**Target**: v0.30.x (eval infrastructure).
**Priority**: P1 — this is the *root cause* of the recurring dashboard cache/stale-render pain. Fixing it retires a whole class of problems in one move.
**Author**: Claude Opus 4.8 (with Mark, 2026-07-12 — "some way of updating data elsewhere to avoid re-renders").

---

## Problem
The benchmark JSONs (`docs/static/benchmarks/latest.json`, `os/latest.json`, `os/history.json`, …) are **baked into the Docusaurus build**. So every data refresh is a full site lifecycle:

```
rig rotation (every 45 min) → git commit + push → GitHub Actions rebuilds the WHOLE site
  → GitHub Pages deploy → Fastly CDN propagation
```

That coupling is the source of everything we fought on 2026-07-11/12:
- **Mixed builds served across CDN edges** — a data push triggers a rebuild; during propagation, Fastly POPs serve *different* builds (we caught days-old bundles alongside the latest).
- **Stale-after-data-update** — `latest.json` carries `max-age=600`, so a fresh publish is invisible for up to 10 min; the newest block (uplift, local rows) shows empty on first load.
- **Build/cache churn** — the rig wants to publish every 45 min; every publish is a heavyweight rebuild + deploy of the entire docs site, and any concurrent code push muddies which build is live.
- The version-trend component was *removed* largely because chasing these artifacts cost days ([m-eval-os-version-trend-redesign](m-eval-os-version-trend-redesign.md)).

The data does not need to be in the build: **every consumer already fetches it at runtime** — the only site-side call-sites are
`fetch('/benchmarks/latest.json')`, `fetch('/benchmarks/os/latest.json')`, `fetch('/benchmarks/os/history.json')`.

## Principle
**Frequently-changing data does not belong in a statically-built, CDN-cached site.** Decouple the data plane from the code plane: the site bundle rebuilds only when *code* changes (rare); data updates are cheap object writes to a separate origin the site fetches at runtime.

## Existing infra we can use (discovered 2026-07-12)
- **GCS buckets** in `ailang-multivac-dev`: `…-ailang-artifacts`, `…-ailang-config`, `…-docparse-temp`. A dedicated `ailang-benchmarks` (public-read) bucket is the natural home.
- **Cloud Run** in `ailang-multivac-dev`: `ailang-dev-dashboard` (the Collaboration Hub), `ailang-dev-website-builder`, `ailang-dev-coordinator`, `ailang-dev-mcp`, `ailang-dev-docparse-api`, `ailang-dev-ntfy` — so serving a small data endpoint is a well-trodden path if we want one.
- The docs site itself is **GitHub Pages + Fastly** (`server: GitHub.com`); it stays as-is.

## Proposed architecture (bucket-first)
1. **Data origin = a public GCS bucket** `gs://ailang-benchmarks/` (or a `benchmarks/` prefix in an existing bucket), fronted by a CDN and a stable host (e.g. `benchmarks.ailang.sunholo.com`, or the raw `storage.googleapis.com` URL to start).
   - Objects written with a **short cache** header: `Cache-Control: public, max-age=60` (near-live, still edge-cached).
   - **CORS** allowing `https://ailang.sunholo.com` (required — cross-origin fetch).
2. **Rig writes there, not git.** The rotation's publish steps `gsutil -h "Cache-Control:public,max-age=60" cp <file> gs://ailang-benchmarks/…` each cycle. No commit, no rebuild, no deploy. (`os-rotation-filler.sh` already regenerates the files; swap the `git add/commit/push` tail for an upload.)
3. **Site fetches the bucket URL.** A single config constant (`DATA_BASE`, env-overridable) replaces the 3 `/benchmarks/...` paths. Keep the in-build copy as a **fallback** so a bucket hiccup degrades to the last release snapshot rather than an empty dashboard.
4. **Provenance snapshot on release only.** `post-release` still commits a point-in-time JSON to git (or a versioned bucket path) so history is reproducible — but routine 45-min updates never touch git.

**Cloud Run alternative** (fallback if we want cleaner caching/headers/domain control than a raw bucket): a tiny read-through service (or an endpoint on an existing service) that serves the JSON from the bucket with exact cache/CORS headers. More moving parts; adopt only if the bucket's CORS/CDN ergonomics prove annoying.

## Work items
- **W1** — Create the bucket (public read + CORS + lifecycle), pick the host/CDN. Decide `DATA_BASE` URL.
- **W2** — Rig: replace the `git add/commit/push` of `latest.json` / `os/*.json` in `os-rotation-filler.sh` (and `os-release-snapshot.sh`) with a `gsutil cp` upload (short cache). Keep `OS_FILLER_PUSH` semantics off by default.
- **W3** — Site: introduce `DATA_BASE` (one constant), repoint the 3 fetch call-sites, add fallback-to-in-build-copy on fetch error. Verify via CI build + a manual fetch (not headless — see [feedback: no headless verify]).
- **W4** — `post-release`: keep a release-time git/bucket snapshot for provenance; stop routine data commits.
- **W5** — Remove the data-churn commits from the rotation once W1–W3 are proven; document the new flow in [database-architecture](../../docs/docs/guides/database-architecture.md).

## Acceptance criteria
1. A rig rotation cycle updates the live dashboard within ~1 min **with no site rebuild and no git commit**.
2. A code-only site change still deploys via the normal GitHub Pages path, unaffected.
3. The dashboard never shows a stale mixed-build after a data update (the class of bug from 2026-07-12 cannot recur — data is not in the bundle).
4. A bucket outage degrades to the last in-build snapshot, not an empty dashboard.

## Out of scope
- Changing the docs site host (stays GitHub Pages).
- The dashboard component logic (already runtime-fetch; only the fetch URL changes).
- Reworking the eval pipeline that *produces* the JSON (unchanged — only where it's *published*).

## Risks / notes
- **CORS is mandatory** (cross-origin fetch) — a one-time bucket config.
- **Cache tuning**: `max-age=60` balances freshness vs. CDN load; the rig publishes every 45 min so even 300s would be fine.
- **Auth**: the bucket is public-read (benchmark data is already public on the site); no credentials in the browser.
- This directly unblocks re-adding richer live views (the removed version-trend, per-tier local breakdowns) without fear of the deploy-churn tax.
