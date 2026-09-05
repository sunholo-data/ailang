# M-EVAL-STANDARD-MODE-INPUT-FILES-GAP: gate multi-file grading benchmarks out of standard mode

**Status**: Planned
**Target**: v0.33.1
**Priority**: P1 — actively suppresses measured AILANG capability on 2 frontier-tier benchmarks, every release
**Estimated**: 1-2 days
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to execution semantics, only to which benchmarks standard mode schedules |
| A2: Replayability | 0 | No trace/replay impact |
| A3: Effect Legibility | 0 | Not touched |
| A4: Explicit Authority | 0 | Not touched |
| A5: Bounded Verification | +1 | Removes a class of eval results that were never actually testing what they claimed to test |
| A6: Safe Concurrency | 0 | Not touched |
| A7: Machines First | +1 | Fixes a measurement instrument the harness's own consumers (curation cycles, tier scoring) rely on to route real fixes |
| A8: Minimal Syntax | 0 | No language syntax change — this is eval-tooling only |
| A9: Cost Visibility | +1 | Stops silently spending API budget on 0-shot calls that cannot possibly pass, for every model, every release |
| A10: Composability | 0 | Not touched |
| A11: Structured Failure | +1 | Skipped benchmarks report a clear `skip_reason` instead of a misleading `compile_error`/`runtime_error` |
| A12: System Boundary | 0 | Not touched |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Improves machine-analyzable eval data, not human convenience

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |

## Verification Log

Claims in this doc verified against the live codebase (2026-08-04, dev @ v0.33.0, post-v0.32.0-release):

| Claim | Method | Result |
|-------|--------|--------|
| Standard-mode prompt never includes `InputFiles` content | Read [spec.go:229-311](../../../internal/eval_harness/spec.go) (`PromptForLanguage`/`PromptPartsForLanguage`) | Confirmed — only `getDefaultPrompt(lang)` + raw `TaskPrompt` string are concatenated; `s.InputFiles` is never referenced in either function |
| `InputFiles` is used ONLY for post-generation workspace writes | `grep -n "InputFiles" internal/eval_harness/runner.go` | Two hits: [runner.go:139](../../../internal/eval_harness/runner.go) (Python), [runner.go:263](../../../internal/eval_harness/runner.go) (AILANG) — both inside `LanguageRunner.Run(code, timeout)`, called *after* the model has already returned `code` |
| Standard-mode AILANG execution always targets a fixed `benchmark/solution.ail`, ignoring `SolutionFiles`/`GradeEntrypoint` | Read [runner.go:227-279](../../../internal/eval_harness/runner.go) (`AILANGRunner.Run`) | Confirmed — model's `code` is unconditionally written to `workspace/benchmark/solution.ail` (line 254-259); `spec.SolutionFiles`/`spec.GradeEntrypoint` are never read in this function |
| `GradeEntrypoint`/`SolutionFiles` grading is explicitly documented as agent-mode-only | Read [spec.go:45-54](../../../internal/eval_harness/spec.go) | Doc comment: *"When GradeEntrypoint is set (agent mode, ailang), the harness grades the agent's PRESERVED workspace... Unset → legacy single-file solution.ail grading."* |
| A prior milestone explicitly scoped standard-mode grading changes OUT | Read [design_docs/implemented/v0_26_1/m-eval-reliable-grading.md:97-99](../../../design_docs/implemented/v0_26_1/m-eval-reliable-grading.md) | Non-Goal: *"Reworking standard (0-shot) single-file grading — it is correct for its model."* — true for single-file benchmarks, but never re-checked against benchmarks that ALSO set `GradeEntrypoint` |
| Exactly which benchmarks are exposed | `grep -l "^solution_files:" benchmarks/*.yml` and `grep -l "^grade_entrypoint:" benchmarks/*.yml` | Identical result both queries: `docx_reimplement.yml`, `markdown_reimplement.yml` — no others |
| Other `input_files`-using benchmarks are unaffected | Read `benchmarks/cli_args.yml`, `benchmarks/programbench_probe_wc_l.yml`, `benchmarks/programbench_probe_cat.yml` | All three ship plain data files (`numbers.txt`, `data.txt`, `a.txt`) read via `readFile` at runtime, no `solution_files`/`grade_entrypoint`, no cross-module import — the existing "write input_files to workspace disk before execution" path is sufficient for these; they do not need this fix |
| Standard-mode dispatch has zero mode-compatibility gate today | Read [eval_benchmark.go:80-154](../../../cmd/ailang/eval_benchmark.go) | `runSingleBenchmark` builds the prompt and dispatches unconditionally for whatever `benchmarkID` it is given; no check against `spec.GradeEntrypoint` anywhere in the standard-mode path |
| Failure signature matches the hypothesis | Read `eval_results/baselines/v0.32.0/standard/markdown_reimplement_ailang_*.json` (13 files) | 8/13 models across every provider family (gemini, gpt5-4-mini/luna/sol/terra, or-deepseek-v4-pro, or-kimi-k2-7-code, or-minimax-m3) fail identically: `undefined variable: emptyParseState at benchmark/solution.ail:9:NN` — the model calls a helper it was told exists but never shown |
| `docx_reimplement` shows the same signature | `jq` scan of `eval_results/baselines/v0.32.0/standard/docx_reimplement_ailang_*.json` `error_category` | 15/N `runtime_error` results share the identical `WARNING MOD010 (relaxed): module 'docparse/services/docx_parser' does not match canonical path 'benchmark/solution'` signature that also appears on every `markdown_reimplement` failure — strong but not per-model-confirmed evidence of the same root cause (see Non-Goals) |
| Duplicate design docs | `create_planned_doc.sh` auto-search (SimHash + neural) | Top matches (`m-codegen-list-sprint-plan.md`, `m-three-camps-sprint-plan.md`, `m-verify-stdlib-stale-path.md`, `m-nightly-sustained-failure-label-sprint-plan.md`, `m-effect-refinement.md`) are all keyword-coincidence false positives — none discuss `input_files`, `solution_files`, or standard-mode prompt construction. Not a duplicate. |
| The standard-mode benchmark-enumeration call site (2026-09-05, dev @ `461e0949a`, re-verified for the mission-docs quorum round-1 objection) | `grep -rn "discoverBenchmarks" cmd/ailang/*.go` | Exactly one definition and one call site: `discoverBenchmarks()` is DEFINED at [eval_helpers.go:36](../../../cmd/ailang/eval_helpers.go) and CALLED at [eval_suite.go:302](../../../cmd/ailang/eval_suite.go), inside the branch guarded by `!*agent && *benchmarks == ""` — the comment on the line immediately above the call already reads `// Auto-discover benchmarks from benchmarks/ directory (standard mode only)`. This is the scheduling-time exclusion point named in Solution Design component 2: filter `discoverBenchmarks()`'s return value (or gate inside it) on `!RequiresAgentWorkspace()` when `evalMode == "standard"`. |
| Whether downstream `error_category` consumers (eval-elo, confidence-gating, curation-cycle) would silently corrupt scoring on the new `skipped_mode_incompatible` value (2026-09-05, dev @ `461e0949a`, mission-docs quorum round-2 objection) — SUPERSEDED by the round-3 finding below, kept for the trail | `grep -rn "ErrorCategory\|error_category" cmd/ailang/eval_elo.go internal/observatory/ratings.go` (zero direct hits in those two files) | `eval-elo`/`observatory/ratings.go` never read `ErrorCategory` directly — but `eval_elo.go:122` loads via `eval_analysis.LoadResults`, which DOES filter through machinery that reads categorical signal (see next row). An `ErrorCategory`-only fix (e.g. `ShouldExcludeFromCapability`) would NOT reach `eval-elo`'s fit at all, since `fitLang` (`cmd/ailang/eval_elo.go:254-280`) computes `ok := r.CompileOk && r.RuntimeOk && r.StdoutOk` directly with ZERO category filtering of any kind — not even the pre-existing `quota_exhausted`/`rate_limit`/`api_error` exclusions apply there. `ShouldExcludeFromCapability` only protects `internal/eval_analysis`'s dashboard/export reporting path, a DIFFERENT pipeline. |
| The correct, single exclusion point that covers BOTH pipelines (2026-09-05, mission-docs quorum round 3, root-caused after the above) | Read `internal/eval_harness/validity.go` (`Validity`/`MarkInvalid`/`IsValid`) and `internal/eval_analysis/validity_filter.go` (`FilterValidResults`) | `internal/eval_analysis.LoadResults` (→ `LoadResultsFromDirs`) ALWAYS applies `FilterValidResults`, which drops any row where `!r.IsValid()` (i.e. `Validity.Valid == false`) BEFORE it reaches either consumer — `eval_elo.go:122` calls `eval_analysis.LoadResults` directly, and the `internal/eval_analysis` export/dashboard functions consume the same filtered set. `Validity` exists FOR EXACTLY THIS CASE per its own doc comment: distinguishing "the subject was measured and did badly" from "we failed to measure the subject" — a `skipped_mode_incompatible` result is definitionally the second kind (the model was never invoked). Marking it `Validity: MarkInvalid(ReasonModeIncompatible)` (a new `Reason` constant, alongside the existing `ReasonCanaryFailed`/`ReasonZeroFiles`/etc. in `validity.go`) excludes it from EVERY aggregate — ELO fitting, confidence-gating, capability stats, dashboard exports — via the one mechanism already built and already trusted for "not a real measurement," while the row itself is RETAINED on disk (never deleted) exactly like every other invalid-but-informative row this system already produces. Supersedes the narrower `ShouldExcludeFromCapability`-only fix from round 2/3 — that switch is left untouched (no `skipped_mode_incompatible` case needed there, since a `Valid:false` row never reaches it at all). |

## Problem Statement

Two frontier-tier AILANG benchmarks — `markdown_reimplement` and `docx_reimplement` — are multi-file
"fill in the stub" tasks. Their `task_prompt` text assumes the model can see a companion file
containing already-implemented helper functions ("the helpers in the SAME file are implemented —
use them: `emptyParseState()`, `mdProcessLine(state,line)`, `flushState(state)`"), and their spec
sets `grade_entrypoint`/`solution_files` so a PRESERVED multi-file workspace can be graded correctly.

That grading path is real and works — but only in **agent mode**. In **standard mode** (the 0-shot,
direct-API-call path used for every model, every release), none of this is honored:

- The model's prompt is built from `PromptPartsForLanguage` (spec.go), which never includes
  `InputFiles` content — the model never sees the companion file it's told to reuse.
- Execution always writes the model's single-file output to a fixed `benchmark/solution.ail` and
  runs that directly — `SolutionFiles`/`GradeEntrypoint` are never consulted, so there is no
  mechanism to splice the model's answer into the correct multi-file position even if it tried.

**Current State:**
- v0.32.0 standard-mode results: 8/13 models (every provider family represented — Gemini, GPT-5
  variants, DeepSeek, Kimi, MiniMax) fail `markdown_reimplement` with the identical
  `undefined variable: emptyParseState` compile error — a 62% failure rate driven entirely by a
  prompt that references content the model was never shown.
- `docx_reimplement` (same docparse family, also frontier tier) shows the matching failure
  signature across 15 results.
- This is not new to v0.32.0 — the benchmarks and the harness gap both predate this release. Every
  prior release's frontier-tier standard-mode score has been depressed by the same mechanism.

**Impact:**
- AILANG's frontier-tier standard-mode pass rate (currently ~23-24%, see
  [[project_v0320_gating_composition_effect]]) is partly an artifact of running two
  agent-mode-only benchmarks through a mode that cannot pass them, not a measure of real
  frontier-tier difficulty.
- Every standard-mode run burns real API budget calling every model against 2 benchmarks with a
  ceiling near 0% regardless of model capability — pure waste under the M-EVAL-STANDARD-CONFIDENCE-GATING
  budget discipline.
- Curation-cycle promote/demote decisions (CURATION.md §5) read standard-mode pass rates per
  benchmark; these two entries are structurally noise, not signal, until fixed.

## Goals

**Primary Goal:** Stop scheduling `GradeEntrypoint`-bearing benchmarks into standard mode, where
their multi-file premise cannot be satisfied, and report *why* they were skipped instead of
recording a misleading compile/runtime failure.

**Success Metrics:**
- `markdown_reimplement` and `docx_reimplement` no longer appear in standard-mode results with
  `error_category: compile_error`/`runtime_error`; they appear as an explicit skip (or are simply
  absent from the standard-mode benchmark set), with the reason machine-readable.
- Standard-mode API spend no longer includes calls to these 2 benchmarks × every model.
- `ailang eval-suite --dry-run` for standard mode shows these 2 benchmarks routed to agent-mode-only.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Fix direction: gate these benchmarks OUT of standard mode vs. build real multi-file standard-mode support (inline `InputFiles` into the prompt + splice the model's answer into `SolutionFiles` + run `GradeEntrypoint`) | The gate is ~1 day, low-risk, and matches the existing M-EVAL-RELIABLE-GRADING Non-Goal decision ("standard single-file grading is correct for its model"). Building real multi-file standard-mode support is a much larger, riskier lift for exactly 2 benchmarks and would re-litigate a decision already made deliberately in v0.26.1 | human | design | high |
| Where the gate lives: `discoverBenchmarks`/`eval-suite` scheduling (skip before dispatch) vs. inside `runSingleBenchmark` (skip at dispatch, still shows up as a result row with a skip status) | Affects whether skipped benchmarks are invisible (cleaner totals) or visible-but-labelled (better for auditing why coverage dropped) | human | design | med |
| Whether `docx_reimplement`'s 15 `runtime_error` results are confirmed to be the SAME root cause, or need independent per-model verification before being folded into this fix's justification | The design doc currently treats this as strong-but-unconfirmed evidence (see Verification Log); the fix itself doesn't depend on the answer (both benchmarks share `grade_entrypoint`, so both qualify for the gate regardless), but the Problem Statement's framing does | agent | compile | low |

### Design Freeze

- [x] Confirm fix direction is "gate out of standard mode" (not "build multi-file standard-mode support") before implementation starts — this doc assumes the gate; if the human decides otherwise, Solution Design below needs a rewrite, not an edit. Unresolved as of doc creation; resolved by mission-docs controller 2026-09-05 in favor of the gate (Solution Design's own Architecture section already committed to it, and the High-Impact Decisions row's Change Cost of "high" for the alternative is reason enough not to default into the larger lift)
- [x] Confirm gate location (scheduling-time vs. dispatch-time skip) — resolved as BOTH (defense in depth), per Architecture components 2+3, which the doc already specified; call sites for both now verified (see Verification Log)

## Solution Design

### Overview

Add a mode-compatibility check: any benchmark with `spec.GradeEntrypoint != ""` is agent-mode-only.
Standard-mode scheduling (wherever `discoverBenchmarks`/`eval-suite` enumerates benchmarks × models
for the standard run) excludes these benchmarks, and if one is explicitly requested via
`--benchmarks` for standard mode, `runSingleBenchmark` short-circuits with a labelled skip rather
than dispatching a doomed API call.

### Architecture

**Components:**
1. **`BenchmarkSpec` helper**: a small method, e.g. `func (s *BenchmarkSpec) RequiresAgentWorkspace() bool { return s.GradeEntrypoint != "" }`, co-located with the existing `GradeEntrypoint`/`SolutionFiles` fields in spec.go so the mode-compatibility rule lives next to the fields that define it.
2. **Scheduling-time exclusion**: `discoverBenchmarks()` ([eval_helpers.go:36](../../../cmd/ailang/eval_helpers.go)), called from the standard-mode auto-discovery branch at [eval_suite.go:302](../../../cmd/ailang/eval_suite.go) (verified by grep, see Verification Log) — filter out benchmarks where `RequiresAgentWorkspace()` is true when the target mode is standard.
3. **Dispatch-time guard**: in `runSingleBenchmark` (`cmd/ailang/eval_benchmark.go`), an early check that returns a structured skip result — `error_category = "skipped_mode_incompatible"` for human/log readability, AND `Validity: eval_harness.MarkInvalid(eval_harness.ReasonModeIncompatible)` for machine aggregation — if somehow called for a `RequiresAgentWorkspace()` benchmark in standard mode. Defense in depth for direct `--benchmarks` invocations that bypass the scheduler filter (component 2); in the normal scheduled path component 2 already excludes the benchmark, so no result row is written at all (per the doc's own Success Metrics, "or are simply absent") and this component never fires.
4. **Validity exclusion (the load-bearing half, root-caused in round 3 — see Verification Log)**: add `ReasonModeIncompatible` to the `Reason` constants in `internal/eval_harness/validity.go`, alongside `ReasonCanaryFailed`/`ReasonZeroFiles`/etc. Setting `Validity.Valid = false` on the skip row is what actually excludes it from `eval-elo` fitting, confidence-gating, capability stats, and every dashboard export — all of them ultimately load through `eval_analysis.LoadResults` → `FilterValidResults`, which drops any `!IsValid()` row before any of those consumers ever sees it. `ShouldExcludeFromCapability` is NOT touched by this fix — verified unnecessary, since a `Valid:false` row is filtered out upstream of it.

### Implementation Plan

**Phase 1: Add the compatibility predicate and dispatch-time guard** (~2 hours)
- [ ] Add `RequiresAgentWorkspace()` to `BenchmarkSpec` in spec.go, next to the `GradeEntrypoint`/`SolutionFiles` doc comment
- [ ] Add the early-return guard in `runSingleBenchmark` (eval_benchmark.go), emitting a clear skip result with a machine-readable reason
- [ ] Unit test: a spec with `GradeEntrypoint` set, dispatched via `runSingleBenchmark` in standard mode, returns the skip result and makes zero AI provider calls

**Phase 2: Scheduling-time exclusion** (~3 hours)
- [ ] Exclude `RequiresAgentWorkspace()` benchmarks from `discoverBenchmarks()`'s standard-mode results (`cmd/ailang/eval_helpers.go:36`, called from `cmd/ailang/eval_suite.go:302` — call site confirmed, see Verification Log)
- [ ] Confirm agent-mode enumeration is untouched (these 2 benchmarks must still run in agent mode)
- [ ] `ailang eval-suite --dry-run --tier frontier` (standard mode) no longer lists `markdown_reimplement`/`docx_reimplement`; the same `--dry-run` for agent mode still lists them

**Phase 3: Verify against real data** (~2 hours)
- [ ] Re-run `markdown_reimplement`/`docx_reimplement` in standard mode post-fix and confirm they're absent/skipped, not failing
- [ ] Unit test: a skip row (`Validity.Valid == false`, `Reason == ReasonModeIncompatible`) is dropped by `eval_analysis.FilterValidResults`
- [ ] Confirm `eval-elo`'s `fitLang` never sees the skip row in its `trials` slice (it loads via `eval_analysis.LoadResults`, which already filters); confirm confidence-gating (`selectBenchmarksByConfidence` → `observatory.LoadBenchmarkRatings`) is unaffected since ratings are fit from the already-filtered set — old rows keep their historical `error_category`/no `Validity` field (absent = valid, per `IsValid()`'s doc comment), only new runs use the skip path

### Files to Modify/Create

**Modified files:**
- `internal/eval_harness/spec.go` - add `RequiresAgentWorkspace()` method, ~10 LOC
- `cmd/ailang/eval_benchmark.go` - dispatch-time guard in `runSingleBenchmark`, ~15 LOC
- `cmd/ailang/eval_helpers.go` (`discoverBenchmarks`, called from `cmd/ailang/eval_suite.go:302`) - scheduling-time exclusion, ~15-30 LOC
- `internal/eval_harness/validity.go` - add `ReasonModeIncompatible` constant, ~3 LOC
- `internal/eval_harness/spec_test.go` / `cmd/ailang/eval_benchmark_test.go` / `internal/eval_analysis/validity_filter_test.go` - new tests, ~50 LOC

## Examples

### Example 1: Standard-mode dry-run before/after

**Before:**
```
$ ailang eval-suite --dry-run --tier frontier --mode standard
Benchmarks: bytecode_vm_trace, docx_reimplement, lfu_cache_trace, markdown_reimplement, ...
```

**After:**
```
$ ailang eval-suite --dry-run --tier frontier --mode standard
Benchmarks: bytecode_vm_trace, lfu_cache_trace, ...
(2 benchmarks skipped: agent-workspace-only — docx_reimplement, markdown_reimplement)
```

## Success Criteria

- [ ] `ailang eval-suite --dry-run` for standard mode never lists a `GradeEntrypoint`-bearing benchmark (acceptance test: dry-run output diffed against the known 2-benchmark exclusion list)
- [ ] `runSingleBenchmark` called directly against `markdown_reimplement`/`docx_reimplement` in standard mode returns a skip result and makes zero provider API calls (acceptance test: mock provider call-counter assertion)
- [ ] Agent-mode scheduling for these 2 benchmarks is unchanged (regression test: agent-mode dry-run still lists both)
- [ ] A skip row (`Validity.Valid == false`, `Reason == ReasonModeIncompatible`) is excluded by `FilterValidResults` (unit test); does not move any per-model ELO fit, confidence-gating rating, or capability/success-rate statistic
- [ ] All tests passing
- [ ] `design_docs/PROGRAM.md` or the eval guide notes the mode-compatibility rule so a future benchmark author knows `grade_entrypoint` implies agent-mode-only

## Testing Strategy

**Unit tests:**
- `RequiresAgentWorkspace()` returns true iff `GradeEntrypoint != ""`
- `runSingleBenchmark` dispatch-time guard short-circuits before any provider call

**Integration tests:**
- Standard-mode `eval-suite --dry-run` excludes both known benchmarks
- Agent-mode `eval-suite --dry-run` still includes both

**Manual testing:**
- Run the real `markdown_reimplement`/`docx_reimplement` benchmarks once in standard mode post-fix and confirm the skip path fires against a live model, not just the mock

## Deferred Decisions

- Whether the skip result should count toward or be excluded from a model's "coverage" denominator used for tier-composition analysis (`[[project_v0320_gating_composition_effect]]`) — RESOLVED for capability/reliability scoring (excluded, via `ShouldExcludeFromCapability`, see Verification Log); the narrower question of tier-composition denominators specifically is still the executor's call, documented at implementation time, since it affects future composition-effect analyses differently than per-model capability stats do

## Non-Goals

- **Building real multi-file standard-mode support** (inlining `InputFiles` into the prompt, splicing the model's answer into `SolutionFiles`, running `GradeEntrypoint`) - this doc treats that as the larger, unchosen alternative (see High-Impact Decisions); only pursue if a human explicitly wants standard-mode data for these specific benchmarks badly enough to justify the bigger lift
- **Independently re-verifying `docx_reimplement`'s per-model failure causes** - the Verification Log notes this as strong-but-unconfirmed; not required because the fix (gating on `GradeEntrypoint`) applies regardless of the precise per-model cause
- **Auditing other benchmarks for similar prompt/execution-model mismatches** - this doc is scoped to the 2 benchmarks confirmed via `grep -l "^solution_files:"`; a broader audit is future work if this pattern recurs

## Timeline

**Day 1** (5 hours): Phase 1 + Phase 2
**Day 2** (2 hours): Phase 3, verification, docs note

**Total: ~1-2 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Scheduling-time exclusion accidentally also filters agent-mode runs | High — would remove real coverage of the 2 benchmarks entirely | Explicit regression test asserting agent-mode dry-run is unchanged (Success Criteria item 3) |
| Historical standard-mode rows for these 2 benchmarks silently confuse downstream tools (ELO fitting, curation cycle) that expect every benchmark to have a mix of pass/fail | Medium — could skew historical trend analysis | Don't retroactively edit old result files; only new runs use the skip path. Note in CHANGELOG that pre-fix standard-mode data for these 2 benchmarks should be treated as unreliable per this doc |
| A future benchmark author adds `grade_entrypoint` without realizing it implies agent-mode-only, reintroducing the gap for a new benchmark | Low-Medium | The doc comment change in Phase 1 makes this explicit at the field definition; consider a `check_benchmark.sh` lint rule as follow-up (not in scope here) |

## Related Documents

**Implemented (informs this design):**
- [design_docs/implemented/v0_26_1/m-eval-reliable-grading.md](../../implemented/v0_26_1/m-eval-reliable-grading.md) — built the `GradeEntrypoint`/`SolutionFiles` agent-mode grading this doc gates standard mode away from; its own Non-Goals section assumed standard-mode single-file grading was universally correct, which this doc corrects for the 2 benchmarks where that assumption doesn't hold
- [design_docs/implemented/v0_32_0/m-eval-standard-confidence-gating.md](../../implemented/v0_32_0/m-eval-standard-confidence-gating.md) — the confidence-gating/budget system whose spend this fix stops wasting on 2 unwinnable benchmarks

**Planned (distinct, no overlap found):**
- [design_docs/planned/m-eval-validity-discipline.md](../m-eval-validity-discipline.md) — addresses cross-cohort comparison validity (coverage gating, like-for-like deltas); this doc addresses a different problem (a benchmark that cannot pass in a given mode at all, not a comparison-validity issue), reviewed for overlap and found genuinely distinct

## References

- [Design Axioms](/docs/references/axioms)
- `benchmarks/markdown_reimplement.yml`, `benchmarks/docx_reimplement.yml`

## Future Work

- If real multi-file standard-mode grading is ever wanted, this doc's Verification Log and Solution Design's rejected alternative are the starting point: inline `InputFiles` in `PromptPartsForLanguage`, splice the model's single-shot answer into the `SolutionFiles` path instead of a fixed `benchmark/solution.ail`, and run `GradeEntrypoint` — reusing as much of the existing agent-mode grading logic (spec.go:45-54, `agent_validation.go`) as possible rather than building a parallel path
- A lint rule (`check_benchmark.sh` or `ailang doctor benchmarks`) that flags any benchmark setting `grade_entrypoint` without an explicit `languages`/mode restriction, so this class of gap can't reappear silently for a new benchmark

---

**Document created**: 2026-08-04
**Last updated**: 2026-08-04
