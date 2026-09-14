# Sprint M-EVAL-STANDARD-MODE-INPUT-FILES-GAP: gate multi-file grading benchmarks out of standard mode

Design: `design_docs/planned/v0_33_1/m-eval-standard-mode-input-files-gap.md`
Target: v0.33.1
Plan: 2 working days, approximately 8.5 hours

## Outcome and sequencing

Stop scheduling the two `grade_entrypoint`-bearing benchmarks (`markdown_reimplement`,
`docx_reimplement`) into standard mode, where their multi-file premise cannot be satisfied, and
report *why* they were skipped instead of recording a misleading compile/runtime failure. The
design is FINAL (approved design-ready; attended human ruling 2026-09-07 authorized direct
routing to planning) — this plan only sequences its existing Implementation Plan.

Sequencing follows the design's three phases, split into five independently green milestones:
M1 lays down the two leaf changes with no wiring (the `RequiresAgentWorkspace()` predicate and
the `ReasonModeIncompatible` validity constant), M2 wires the dispatch-time guard (design
component 3), M3 the scheduling-time exclusion (design component 2), M4 verifies the aggregation
exclusion end-to-end (design component 4 / Phase 3), and M5 lands the docs note and final gates.
No execution semantics change anywhere — eval-tooling only.

## Milestones

### Day 1 — M1: mode-compatibility predicate and validity reason (1.5 hours)

Files: `internal/eval_harness/spec.go`, `internal/eval_harness/validity.go`,
`internal/eval_harness/spec_test.go` (update).

Add `func (s *BenchmarkSpec) RequiresAgentWorkspace() bool { return s.GradeEntrypoint != "" }`
co-located with the `GradeEntrypoint`/`SolutionFiles` fields (spec.go:45-54), and extend that
field's doc comment to state the rule explicitly — `grade_entrypoint` implies agent-mode-only —
so the next benchmark author hits the rule at the definition site (design Risks mitigation).
Add the `ReasonModeIncompatible = "mode_incompatible"` constant to the `Reason` block in
`validity.go` (alongside `ReasonCanaryFailed`/`ReasonZeroFiles`/`ReasonConfigMismatch`/…),
with a doc comment in the file's established style explaining that a mode-incompatible run is a
"we failed to measure the subject" row, not a measurement. Extend `spec_test.go`: the predicate
is true iff `GradeEntrypoint != ""` (cover set and unset, e.g. via the two real benchmark specs).

End the milestone with `go build ./...` and `go test ./internal/eval_harness/` passing.

### Day 1 — M2: dispatch-time guard in runSingleBenchmark (2 hours)

Files: `cmd/ailang/eval_benchmark.go`, `cmd/ailang/eval_benchmark_test.go` (new).

In `runSingleBenchmark` (defined at eval_benchmark.go:31), add an early standard-mode check
(~15 LOC per design): when `agentConfig == nil` (the agent branch has already returned at
:83-84, so the guard is standard-mode-only by construction) and
`spec.RequiresAgentWorkspace()`, short-circuit before any provider dispatch — return a
structured skip result carrying BOTH the human-readable label
`error_category = "skipped_mode_incompatible"` AND the machine-aggregation marker
`Validity: eval_harness.MarkInvalid(eval_harness.ReasonModeIncompatible)`. This is the design's
defense-in-depth for direct `--benchmarks` invocations that bypass the scheduler filter.

Create `cmd/ailang/eval_benchmark_test.go` (no such file exists today): call
`runSingleBenchmark` directly against `markdown_reimplement` in standard mode with a
mocked/counted provider transport (follow the httptest provider-mock precedent of the
`configdriven_*_test.go` files in the same package) and assert (a) the skip result shape and
(b) the AI provider call counter reads zero.

End the milestone with `go build ./...` and `go test ./cmd/ailang/` passing.

### Day 1 — M3: scheduling-time exclusion (1.5 hours)

Files: `cmd/ailang/eval_helpers.go`, `cmd/ailang/eval_suite.go`.

Exclude `RequiresAgentWorkspace()` benchmarks from the standard-mode auto-discovery results
(~15-30 LOC per design). `discoverBenchmarks()` (eval_helpers.go:36) takes no arguments and has
exactly one call site — eval_suite.go:302, inside the `!*agent && *benchmarks == ""` branch
where `evalMode` is in scope — so the smaller diff is filtering the returned list at the call
site (the design allows either shape: "filter `discoverBenchmarks()`'s return value (or gate
inside it)"). Agent-mode scheduling must be untouched: per the design's own verification log,
agent mode never passes through `discoverBenchmarks` (it requires an explicit `--benchmarks`
list, eval_suite.go:288-292), so assert the explicit-list path is unfiltered.

Verify from the CLI:
- `ailang eval-suite --dry-run --tier frontier` (standard) no longer lists
  `docx_reimplement`/`markdown_reimplement` in the printed `Benchmarks:` line;
- `ailang eval-suite --dry-run --agent --benchmarks docx_reimplement,markdown_reimplement`
  still plans both.

Executor note (observation V12, within the design's stated defense-in-depth, no scope change):
the `--benchmarks-by-confidence` scheduling path (`selectBenchmarksByConfidence`,
eval_confidence.go:33, feeds `benchmarkList` at eval_suite.go:286) is *not* component 2 — any
benchmark reaching `runSingleBenchmark` from any path is caught by M2's dispatch-time guard.
Record the observed behavior in implementation notes; do not add a second scheduler filter
without controller approval.

End the milestone with `go build ./...` and `go test ./cmd/ailang/` passing.

### Day 2 — M4: aggregation exclusion verified, real-data re-run (2 hours)

Files: `internal/eval_analysis/validity_filter_test.go` (update). No production code expected.

Extend `validity_filter_test.go`: a result row with `Validity.Valid == false`,
`Reason == ReasonModeIncompatible` is dropped by `FilterValidResults` (validity_filter.go:26).
Then confirm the downstream non-impact claims by reading the load path (verification only, no
edits): `cmd/ailang/eval_elo.go:122` loads via `eval_analysis.LoadResults` →
`FilterValidResults`, so `fitLang` (eval_elo.go:254) never sees the skip row in its trials;
confidence-gating ratings are fit from the already-filtered set;
`ShouldExcludeFromCapability` stays untouched (a `Valid:false` row is filtered upstream of it).
Old result rows keep their historical `error_category` with no `Validity` field (absent =
valid, per `IsValid()`'s doc comment) — never retroactively edit historical result files.

Manual (design's manual-testing item): re-run `markdown_reimplement`/`docx_reimplement` in
standard mode post-fix — or confirm on the next scheduled run — and record that they appear as
absent/labelled-skipped, not as compile/runtime failures.

End the milestone with `go build ./...` and `go test ./internal/eval_analysis/
./internal/eval_harness/` passing.

### Day 2 — M5: mode-compatibility docs note, changelog, final gates (1.5 hours)

Files: `design_docs/PROGRAM.md` **or**
`docs/docs/guides/evaluation/harness-setup.md` (pick one author-facing location — the design
says "or"), `changelogs/v0.32-current.md`.

Document the mode-compatibility rule where benchmark authors will read it: a benchmark setting
`grade_entrypoint` is agent-mode-only and is excluded from standard-mode scheduling. Add the
release note that pre-fix standard-mode data for these 2 benchmarks should be treated as
unreliable per the design doc — in `changelogs/v0.32-current.md`, NOT the root `CHANGELOG.md`
(the root file is an index-only file enforced by the `make check-changelog` CI gate).

Final gates (all must pass from this worktree): `go build ./...`, `go test ./...`,
`make fmt-check`, `make lint`, `make check-boundaries`, `make check-file-sizes`, and
`git diff --name-only` matching the approved file list below.

## Approved file list (M5 diff target)

| File | Change |
|------|--------|
| `internal/eval_harness/spec.go` | add `RequiresAgentWorkspace()` + doc-comment rule (~10 LOC) |
| `internal/eval_harness/validity.go` | add `ReasonModeIncompatible` constant (~3 LOC) |
| `internal/eval_harness/spec_test.go` | update: predicate unit tests |
| `cmd/ailang/eval_benchmark.go` | dispatch-time guard in `runSingleBenchmark` (~15 LOC) |
| `cmd/ailang/eval_benchmark_test.go` | new: skip result + zero-provider-calls test |
| `cmd/ailang/eval_helpers.go` | scheduling-time exclusion (with its eval_suite.go:302 call site) (~15-30 LOC) |
| `cmd/ailang/eval_suite.go` | call-site filter wiring only |
| `internal/eval_analysis/validity_filter_test.go` | update: skip-row dropped by `FilterValidResults` |
| `design_docs/PROGRAM.md` or `docs/docs/guides/evaluation/harness-setup.md` | mode-compatibility rule note (one location) |
| `changelogs/v0.32-current.md` | historical-data unreliability note |

Hard unchanged: `internal/eval_harness/runner.go`, `internal/eval_harness/agent_validation.go`,
`internal/eval_analysis/validity_filter.go` (and `ShouldExcludeFromCapability`), `benchmarks/*.yml`,
`eval_results/**` (no retroactive edits), root `CHANGELOG.md` (index-only gate).

## Acceptance checklist

- [ ] `ailang eval-suite --dry-run` for standard mode never lists a `GradeEntrypoint`-bearing
      benchmark (diff the `Benchmarks:` line against the known 2-benchmark exclusion list:
      `docx_reimplement`, `markdown_reimplement`).
- [ ] `runSingleBenchmark` called directly against `markdown_reimplement`/`docx_reimplement` in
      standard mode returns a skip result
      (`error_category: skipped_mode_incompatible`, `Validity.Reason: mode_incompatible`) and
      makes zero provider API calls (mock call-counter assertion).
- [ ] Agent-mode scheduling is unchanged: `--agent --dry-run` with an explicit `--benchmarks`
      list still plans both benchmarks (agent mode never routes through `discoverBenchmarks`).
- [ ] A skip row (`Validity.Valid == false`, `Reason == ReasonModeIncompatible`) is dropped by
      `FilterValidResults`; no per-model ELO fit, confidence-gating rating, or
      capability/success-rate statistic moves.
- [ ] All tests passing (`go build ./...`, `go test ./...`).
- [ ] The mode-compatibility rule is noted in `design_docs/PROGRAM.md` or the eval guide;
      `changelogs/v0.32-current.md` notes pre-fix standard-mode rows are unreliable.
- [ ] Historical result files untouched; `ShouldExcludeFromCapability` untouched; no
      standard-mode execution path (`runner.go`) modified.

## Verification Log

All load-bearing claims below were re-checked in this worktree (2026-09-07, read-only) before
planning; the IDs are referenced by the machine-readable sprint JSON. The design doc's own
Verification Log line numbers were spot-checked and confirmed.

| ID | Command | Observed output |
|---|---|---|
| V1 | `nl -ba internal/eval_harness/spec.go \| sed -n '45,54p'` | Agent-mode grading doc comment at 45-52; `GradeEntrypoint` field at line 53, `SolutionFiles` at line 54 — exactly as the design cites. |
| V2 | `grep -n "PromptForLanguage\|PromptPartsForLanguage" internal/eval_harness/spec.go` | `PromptForLanguage` at :215, `PromptPartsForLanguage` at :229 — inside the design's cited 229-311 region. |
| V3 | `grep -n "func discoverBenchmarks" cmd/ailang/eval_helpers.go` | Defined at line 36 (doc comment on 35). |
| V4 | `grep -n "discoverBenchmarks" cmd/ailang/eval_suite.go` | Exactly one call site at line 302, under the comment "Auto-discover benchmarks from benchmarks/ directory (standard mode only)". |
| V5 | `grep -n "func runSingleBenchmark" cmd/ailang/eval_benchmark.go` | Defined at line 31; agent branch delegates to `runSingleBenchmarkAgent` and returns at :83-84; standard-mode chain-stage setup begins :87+ — the guard insertion point is after spec load, before :87. |
| V6 | `grep -n '"tier"\|"dry-run"\|evalMode :=' cmd/ailang/eval_suite.go` | `--tier` flag :101, `--dry-run` flag :114, `evalMode := "standard"` :277 (agent :279), and the dry-run printer emits the `Benchmarks:` list at :387. |
| V7 | `grep -n "Reason\|func MarkInvalid\|IsValid" internal/eval_harness/validity.go` | `Validity` type :22, existing constants `ReasonCanaryFailed` :41 … `ReasonTreatmentUnproven` :58, `MarkInvalid` :74 (empty reason falls back to `ReasonHarnessError`), `IsValid` :89; `ReasonModeIncompatible` absent (to be added). |
| V8 | `grep -rn "func FilterValidResults" internal/eval_analysis/` + `sed -n '14,49p' internal/eval_analysis/loader.go` | `FilterValidResults` at validity_filter.go:26; `LoadResults` :14 → `LoadResultsFromDirs` :49 filters invalid rows by default (`LoadResultsIncludingInvalid` :23 opts back in) — matches design component 4. |
| V9 | `ls internal/eval_harness/spec_test.go cmd/ailang/eval_benchmark_test.go internal/eval_analysis/validity_filter_test.go` | `spec_test.go` exists (12.8KB, update); `validity_filter_test.go` exists (5.7KB, update); `cmd/ailang/eval_benchmark_test.go` does NOT exist (new file). |
| V10 | `grep -l "^grade_entrypoint:" benchmarks/*.yml` and `grep -l "^solution_files:" benchmarks/*.yml` | Identical result for both queries: exactly `benchmarks/docx_reimplement.yml` and `benchmarks/markdown_reimplement.yml`. |
| V11 | `sed -n '120,124p' cmd/ailang/eval_elo.go; grep -n "func fitLang" cmd/ailang/eval_elo.go` | `eval_analysis.LoadResults(resultsDir)` at ~:122; `func fitLang` at :254 — the design's aggregation-exclusion claim holds. |
| V12 | `grep -rn "func selectBenchmarksByConfidence" cmd/ailang/` + `sed -n '275,292p' cmd/ailang/eval_suite.go` | A third scheduling path exists: `--benchmarks-by-confidence` → `selectBenchmarksByConfidence` (eval_confidence.go:33) sets `benchmarkList` at :286. Outside design component 2 by scope; covered by component 3's dispatch-time guard. Recorded as an executor note in M3. |
| V13 | `sed -n '288,292p' cmd/ailang/eval_suite.go` | "SAFETY: Agent mode requires explicit benchmark list" — agent mode errors out rather than auto-discovering, so the agent-mode regression criterion is exercised via explicit `--benchmarks` (never through `discoverBenchmarks`). |
| V14 | `grep -rln "httptest" cmd/ailang/*_test.go \| head` | Provider-mock precedent exists in-package (`configdriven_callstream_test.go`, `configdriven_dispatch_test.go`, `configdriven_harvest_test.go`, …) — the pattern M2's zero-provider-calls test follows. |
| V15 | `git status --porcelain=v1` | Empty; worktree clean before planning. |
| V16 | `make help-health` + `head -5 CHANGELOG.md` | CI gates `fmt-check`, `lint`, `check-boundaries`, `check-file-sizes` exist; root `CHANGELOG.md` is index-only pointing at `changelogs/v0.32-current.md` (enforced by `make check-changelog`) — release notes go in the changelogs/ file. |

## Handoff

This plan is ready for controller approval and handoff to sprint-executor. The planner does not
stage, commit, or implement any milestone. Worktree is read/plan-only per mission instructions.