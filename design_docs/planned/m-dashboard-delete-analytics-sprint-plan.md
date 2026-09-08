# M-DASHBOARD-DELETE-ANALYTICS

Status: In progress. Authorized by Mark’s “yes continue” after the deletion sequence in the dashboard recovery plan. Work directly on dev; preserve unrelated changes.

Design: [Recovery plan](../dashboard-recovery-plan-2026-09-08.md), deletion sprint A. Scope: remove optional analytics/heatmap panel and evolution-tree mode, including exclusive dependencies. Keep Work/chain explorer, approvals, message date filters, timeline and chat inspection. No backend changes or deployment.

Estimate: one engineering day; at least 5,860 candidate lines plus exclusive support files. Prior sprint was a backend repair and does not provide meaningful UI deletion velocity.

## M1 — Remove optional views
- Remove analytics panel and heatmap polling, exclusive selection handlers and split layout.
- Remove evolution mode and its exclusive tree code/styles.
- Preserve chain explorer, timeline/chat evidence, approval detail and message date filters.

## M2 — Remove dependency closure
- Delete analytics-only hooks, unused chart files/styles and barrel exports.
- Remove recharts, d3-hierarchy and its types after verifying no retained imports.
- UI tests and production build pass; record before/after bundle sizes.

## M3 — Review and record
- Independently review retained navigation and data-fetch reachability; fix regressions.
- Update changelog and recovery plan with actual deletion counts and remaining data gaps.
- Record browser/full-suite limitations honestly; no production stability or deployment claim.

Validation: existing UI tests/build baseline first; lint via UI build, import closure checks, retained component rendering where feasible. This is removal of optional UI, not a language feature; no .ail examples or compiler tests added. Do not invoke inbox-mutating sprint scripts; construct documented JSON and use its validator. No Go changes: prior full Go suite environment limitations remain separate from UI validation.

## Execution evidence

- Baseline: 142 UI tests pass; JS 1,265.62kB (gzip 354.66kB), CSS 266.86kB (gzip 42.14kB).
- Final: 147 UI tests pass (four files), including actual controlled date-input handlers and rendered event-queue UTC midnight/end-of-day fixtures.
- Production build and lint: pass; JS 720.26kB (gzip 200.82kB), CSS 203.20kB (gzip 32.08kB), about 43% less JS and 24% less CSS. Vite retains the pre-existing >500kB chunk warning.
- Removed 19 source files; approximately 11,200 net source lines removed, excluding generated assets/lockfile. Deleted orphan Outliers chart and heatmap date detail after consumer checks; retained TraceWaterfall because timeline uses it.
- `recharts`, `d3-hierarchy`, `@types/d3-hierarchy` and exclusive transitive packages removed. Offline npm uninstall used `--legacy-peer-deps` for the existing ESLint peer-version mismatch; no package upgrades requested.
- Independent review caught the lost heatmap date-input path, local/UTC mismatch and narrow-sidebar wrapping; all corrected. Native browser inspection unavailable (`cgWindowNotFound`); browser interaction and production rollout remain unverified.
- `make ui-deploy` here only builds/copies checked-in assets; no running service restart or cloud deploy. Architecture boundaries pass. Full Go suite was not rerun for this frontend-only change; prior environmental limits are not relabeled passing.
- Logs: `/tmp/dashboard-delete-baseline-test.log`, `/tmp/dashboard-delete-baseline-build.log`, `/tmp/dashboard-delete-test-v3.log`, `/tmp/dashboard-delete-ui-assets-final.log`, `/tmp/dashboard-delete-boundaries.log`.
