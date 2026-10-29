# Sprint: M-BENCHMARK-PERFORMANCE-REFRESH

## Summary

Ship the reviewed A performance page with overall plots, an evidence table and
versioned saved results. Add the scoped export needed to avoid pooling metrics
that the UI claims are mode/language/tier-specific. No eval or model calls.

**Design:** [M-BENCHMARK-PERFORMANCE-REFRESH](m-benchmark-performance-refresh.md)
**Duration:** Two focused days, about 10 implementation/validation hours plus
25% contingency; independent evaluation and deployment latency additional.
**Risk:** Medium, primarily provenance/missingness and historical compatibility.
**Status:** ✅ Completed 2026-10-23; independently evaluated 95/100. Production publication pending review.

## Current State and Velocity

The user selected A, requested overall plots and the AILANG style, and corrected
the undefeated sorting. Those choices are reflected in revision 3. The mockup
is not the production implementation; its limited projection has known legacy
cost/timing provenance gaps.

The velocity script was run on 2026-10-23. It exited 141 after a head/pipe stage,
and its changelog extraction sampled old releases. The seven-day path-scoped git
history contains no useful comparable frontend implementation sample. Do not
claim a measured LOC/day from it. Estimates below are engineering estimates
informed by the existing prototype/export inventory, with explicit contingency.

Total estimated changed LOC: **2,020**, including focused tests. Each implementation
milestone should fit a 60–90 minute executor slice; split on measured overrun.
No file should approach the 800-line soft limit: use companion modules for
export types/builders and React data, table, charts, details and styling.

## Registry Reuse Audit

Every milestone is `none` for new registry dependencies: this is native Go
export and React/Docusaurus integration using existing project modules and
Recharts, not a new package-like AILANG capability. Registry package search is
not applicable; no AILANG package will be created or copied into the compiler.
Reuse loader validity/dedup, existing ELO fits, benchmarkFetchWithSource, theme
tokens, Recharts, current history and existing task/research routes.

## Milestones

### ✅ M1: Scoped saved-evidence export (~420 LOC)

**Dependencies:** None. **Slice:** 60–90 minutes.
**Implementation/test split:** ~250 / 170 LOC.

Files: new `internal/eval_analysis/evidence_types.go`,
`evidence_export.go`, `evidence_export_test.go`; small wiring in `types.go` and
`export_json.go`. Follow-up companion files if aggregation outgrows the slice.

- [x] Optional versioned benchmarkEvidence preserves mode/harness/language/tier,
  exact task IDs and Core-set coverage, pass/first-attempt/repair/outcomes, reported
  versions and measurement dates.
- [x] Cost categories and timing boundary are explicit; unknown billing data
  and absent timing do not become metered dollars or instantaneous latency.
- [x] Existing schema keys and history round-trip unchanged; offline export over
  a saved fixture produces the new block without inference or bank mutation.
- [x] Tests cover 49/49 Argon, 3/3 Astra, duplicate trials, mixed versions,
  unknown cost, explicit free-local zero and missing versus measured zero.

Example: a small JSON fixture derived from the design snapshot, with omitted
provenance, two trial IDs and a legacy duration row. No `.ail` example applies.

### ✅ M2: Production comparison table and identity projection (~520 LOC)

**Dependencies:** M1. **Slice:** 60–90 minutes.
**Implementation/test split:** ~360 / 160 LOC.

Files: new `docs/src/components/BenchmarkPerformance/` shell/table/CSS modules,
plus small pure data/sort helpers under `docs/src/lib/` and focused Node tests.

- [x] One fetched snapshot powers the page; mode, language, scope, search and
  sort preserve identity and never manufacture a Python score.
- [x] Unrated undefeated group first; complete Core coverage before provisional
  samples, numeric ELO next. Astra's three AILANG runs are not 23 Core results.
- [x] Cost and name sorts remain independent of capability grouping. Rated
  agent models keep their numeric-ELO ordering even if undefeated.
- [x] Empty/missing/zero states and legacy evidence adapter have targeted tests.

Example: fixture-rendered Argon/Astra/finite-ELO/unknown-cost rows with both
languages, one missing language and a measured 0% case.

### ✅ M3: Overall saved-history plots (~420 LOC)

**Dependencies:** M2. **Slice:** 60–90 minutes.
**Implementation/test split:** ~270 / 150 LOC.

Files: new trend component and projection helper/tests; reuse Recharts and
existing saved HistoryEntry/Events. Do not port prototype handwritten chart code
if the existing chart components/library satisfy the reviewed interaction.

- [x] Main page shows pass-rate and model-ELO plots, full/recent history,
  independent clearly labeled scope and a shared keyboard snapshot inspector.
- [x] Missing values break paths, measured zeros remain, and changing model
  identities or anchors are not silently connected.
- [x] Cohort counts/events and snapshot publication versus measurement meaning
  are visible. An absent fixed-panel index is unavailable, not invented.
- [x] Saved history is preserved across a publication without new eval runs.

Example: short irregular release history with missing language, true zero,
anchor change, carried-forward values and an older unrated snapshot.

### ✅ M4: Model evidence details and history access (~320 LOC)

**Dependencies:** M2, M3. **Slice:** 60–90 minutes.
**Implementation/test split:** ~220 / 100 LOC.

Files: selected-model detail component, snapshot access and links; tests for
metric scope/missingness. Keep the full download as the actual fetched artifact,
not the mockup's reduced dataset.

- [x] First/final pass, repair, tokens, outcome categories, cost provenance,
  reported versions and timing boundaries remain reachable from each row.
- [x] Whole-model detail does not pretend to inherit a tier or language filter;
  each metric uses its actual declared scope.
- [x] All saved version snapshots and original published fields remain
  inspectable/downloadable; existing Tasks/ELO/Value/Explorer pages stay linked.

Example: a dual-mode model with different mode costs, multiple measured versions
and missing generation latency; prove that none are borrowed across scopes.

### ✅ M5: Route replacement, documentation and theme integration (~180 LOC)

**Dependencies:** M3, M4. **Slice:** 60–90 minutes.
**Implementation/test split:** ~140 / 40 LOC.

Files: `docs/docs/benchmarks/performance.md`, `overview.md`, small component
integration changes, and `changelogs/unreleased/2026-10-23-benchmark-performance-refresh.md`
opening with `### `. Use current docs theme variables and original SVG.

- [x] Performance route mounts the new page; removes duplicate radar/chart/KPI
  composition from this route without deleting evidence or breaking other views.
- [x] Inter/Montserrat, docs palette, focus states and light/dark behavior match
  the reviewed AILANG direction. No marketing-only global CSS is copied in.
- [x] Latest publication and per-row measured version labels remain distinct;
  a visible static-copy fallback does not automatically imply stale data.
- [x] Documentation describes rolling evidence reuse and has no full-baseline
  or every-release eval requirement.

Example: render the performance route against both new and legacy fixture shapes.

### ✅ M6: Rendered verification, evaluation and publication preparation (~160 LOC)

**Dependencies:** M5. **Slice:** 60–90 minutes plus build/review latency.
**Implementation/test split:** ~40 / 120 LOC (validation fixtures/checks).

Files: focused integration/visual fixtures and the sprint evaluation record.
Keep a dedicated preview checkout/output if docs generation would overwrite
unrelated dirty work. Do not stash, discard or switch the dirty main checkout.

- [x] Targeted tests, relevant repository gates and production docs build pass.
- [x] Verify actual rendered desktop/mobile and light/dark states, Argon/Astra,
  missing Python, cost sort, empty filters, model detail, historical plots and
  static fallback. Record browser-tool limits rather than claiming JSON is UI QA.
- [x] Independent sprint-evaluator scores against the design and resolves
  blocking findings before production publication.
- [x] Prepare the concrete final diff, preview and validation report for the
  final publication decision. Do not publish the new UI under a prior data-only
  deployment instruction.

Example: rendered evidence-table screenshot plus trend/inspector interaction,
not a screenshot of the JSON endpoint.

## Two-Day Execution Order

Day 1: M1 and M2, then M3. Establish typed evidence and projection tests before
presentation. Day 2: M4 and M5, then M6; reserve contingency for imported legacy
data and real rendered verification. No parallel agents are required.

## Validation

- Native scope: `make test-changed`, targeted
  `go test ./internal/eval_analysis -run 'Evidence|Ratings|Export'`, then applicable
  `make check-file-sizes`, `make check-changelog` and `make check-boundaries`.
- Frontend: use the built-in Node test runner for pure helpers and existing
  dependencies for fixture rendering; no new test framework solely for this page.
- Build: `npm run build` from `docs/` (contains the established sync steps), using
  an isolated output/check-out if required to preserve unrelated generated edits.
- Run `make docs-build` when needed for WASM/assets integration; it is broader
  than the frontend-only build and must not be rerun without new concerns.
- CI handles the full Go suite. Do not require `make test` as an executor gate.
- Never run eval-suite, access probes, paid quorum, post-release evals or provider
  inference to validate a UI change. Work only with saved fixtures and exports.

## Execution Authorization and Completion

The user explicitly said **execute sprint** on 2026-10-23. All six milestones
are implemented and verified. Independent evaluation passed 95/100. No model
calls or new eval trials were used. The private preview is ready for the final
publication decision; this execution did not publish the production UI.
