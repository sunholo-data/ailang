# Sprint Plan: docs-3 — benchmark provenance source wiring

## Overview

- Routing brief: `design_docs/docs-3-brief.md` (routing declaration; no design doc required).
- Mission: docs clause 6, benchmark report maintenance.
- Goal: make all five benchmark pages expose whether benchmark data came from GCS or the in-build fallback, so the existing stale-data badge is truthful.
- Size: S, approximately one focused day.
- Scope: exactly four files under `docs/src/components/{EloLeaderboard,BenchmarkStandaloneGallery,BenchmarkDashboard,BenchmarkExplorer}/index.jsx`.
- No commits or git staging; the controller finalizes the diff.
- Explicitly out of scope: `benchmarks/events.yml` annotation/schema work and all deferred trend-boundary work in the brief.

## Current Status Analysis

The controller audit found that `ValueDashboard/index.jsx` is the only one of five
`<DataProvenance>` call sites passing `source={dataSource}`. The other four pages use the
plain `benchmarkFetch()` helper and cannot surface `DataProvenance`'s existing
`source === 'fallback'` badge. `benchmarkFetchWithSource()` and
`DataProvenance/index.jsx` already implement the required behavior and must not change.

Recent repository activity is active, but the generic historical velocity script does not
provide a useful comparable LOC metric for this mechanical docs-side task. Estimate from the
known pattern and call-site count instead: roughly 20–30 changed lines across four files,
plus verification-only work, with one day of executor capacity.

## Proposed Milestones

### Milestone 1: Wire the three single-fetch benchmark pages

**Goal:** Replicate the ValueDashboard wiring in EloLeaderboard, BenchmarkStandaloneGallery,
and BenchmarkDashboard.

**Estimated:** 15 LOC implementation + 0 LOC tests = 15 LOC  
**Duration:** 0.5 day

**Tasks:**

- Replace each `benchmarkFetch` import with `benchmarkFetchWithSource`.
- Add `const [dataSource, setDataSource] = useState(null);` beside the existing component state.
- Change the fetch promise to `.then(({ response, source }) => { ... })`, preserve each page's existing response validation/data handling, and call `setDataSource(source)`.
- Add `source={dataSource}` to each existing `<DataProvenance>` call without changing the component implementation.
- Run a focused grep/diff review to confirm only the three named files changed in this milestone.

**Acceptance Criteria:**

- Each of the three files imports `benchmarkFetchWithSource`, tracks `dataSource`, and passes `source={dataSource}`.
- Existing data/error behavior remains intact.
- No change is made to `benchmarkFetch.js`, `DataProvenance/index.jsx`, or any DataProvenance-less fetch site.

**Risks:**

- The pages have slightly different promise formatting and response handling. Mitigation: preserve local logic and copy only the proven ValueDashboard shape.

### Milestone 2: Wire BenchmarkExplorer and complete verification gates

**Goal:** Track the source of the baseline `latest.json` fetch through the existing `Promise.all`
chain while leaving optional OS data handling unchanged, then prove the badge and build remain
correct.

**Estimated:** 10 LOC implementation + 0 LOC tests = 10 LOC  
**Duration:** 0.5 day

**Tasks:**

- Replace the Explorer import and add `dataSource` state.
- In the `Promise.all`, destructure the baseline fetch result as `{ response, source }`, retain the existing `os/latest.json` optional rejection/non-OK handling, and pass the baseline response JSON plus source to the existing merge path.
- Set `dataSource` from the `latest.json` source only; do not represent the optional OS fetch as the page provenance source.
- Add `source={dataSource}` to Explorer's existing `<DataProvenance>` call.
- Confirm `grep -rn 'source={'` across the five DataProvenance call sites returns 5.
- Perform a live mutation check for at least one of the four repaired pages by forcing `benchmarkFetchWithSource` to resolve with `source: 'fallback'` (or temporarily forcing the remote request to reject), verify the rendered text `⚠ Fallback / stale data`, and restore the mutation.
- Run `make docs-build`.
- Run a diff-scope check proving the helper/component and the five named DataProvenance-less fetch sites are untouched.

**Acceptance Criteria:**

- All four repaired files use the source-aware helper and pass a live source prop; the five-call-site grep count is exactly 5.
- The fallback mutation check observes the rendered `⚠ Fallback / stale data` badge, not merely a passed prop.
- `make docs-build` exits successfully.
- The final diff contains no changes to `docs/src/lib/benchmarkFetch.js`, `docs/src/components/DataProvenance/index.jsx`, `docs/src/components/OSReleaseTrend/index.jsx`, `docs/src/components/AgentUpliftTable/index.jsx`, `docs/src/components/BenchmarkDashboard/osHistory.js`, `docs/src/components/OSLocalLeaderboard/index.jsx`, or `docs/src/components/BenchmarkDashboard/ModelComparisonTable.jsx`.
- No event annotation/schema work is included.

**Risks:**

- `Promise.all` can accidentally make optional OS provenance affect baseline provenance. Mitigation: source only the `latest.json` result and leave the OS branch's `null` fallback untouched.

## Success Metrics

- `source={` appears at all 5 `<DataProvenance>` call sites.
- At least one repaired page visibly renders `⚠ Fallback / stale data` under a forced fallback result.
- `make docs-build` is green.
- Only the four specified component files are modified; no helper, badge component, or five DataProvenance-less fetch sites change.
- No CHANGELOG entry, design doc, benchmark rerun, git staging, or commit is required.

## Dependencies

- Existing `benchmarkFetchWithSource` behavior in `docs/src/lib/benchmarkFetch.js`.
- Existing `DataProvenance` fallback badge behavior in `docs/src/components/DataProvenance/index.jsx`.
- ValueDashboard's current source-tracking implementation as the exact reference pattern.

## Open Questions

- None. The brief fixes the pattern, files, acceptance criteria, and out-of-scope work.

## Notes

- The executor should restore any temporary mutation before reporting completion and leave only the four intended call-site files in the working diff.
- This is a mechanical consistency fix to already-shipped frontend plumbing; no new design claims or API changes are planned.
