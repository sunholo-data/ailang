# Sprint Plan: M-ITERATIVE-LIST-REMAINING

## Summary

Complete and verify the approved remaining-list design against the current checkout: prove the already implemented iterative searches/right fold, replace extrema callback folds with the design's six direct monomorphic builtins, and finish fallback visibility, cost documentation, and regression evidence.

**Design:** [m-iterative-list-remaining.md](m-iterative-list-remaining.md)  
**Status:** planned; design approved by coordinator handoff, sprint approval pending coordinator review/merge. No implementation authorized by this planning artifact alone.  
**Duration:** 5 working days, approximately 30 focused hours including 6 hours contingency.  
**Target:** v0.51.3; current std/VERSION is v0.51.0.  
**Risk:** medium (engine parity, float extrema semantics, stale authoring binary).  
**Lane:** AILANG fix, per PROGRAM.md.  
**Estimated additional work:** 750 LOC: 180 implementation, 410 tests/benchmarks/examples, 160 documentation. Generated trace data excluded.

## Current Status Analysis

Planning baseline: clean branch `coordinator/task-72823b01`, HEAD `b5899eb1`, 2026-10-02. The handoff design was authored against older tree `07846355`; its negative builtin-existence claims and recursion audit are historical, not current implementation gaps.

Already implemented in committed source:

- `internal/builtins/list_iterative_search.go` registers `_list_any`, `_list_findIndex`, `_list_foldr`, and `_list_mapAccumL`; the first three are this sprint's scope.
- `std/list.ail` delegates any/findIndex/foldr; findIndexHelper is removed. `internal/vm/builtins_hof_list.go`, bytecode HOF names and VM tables contain native ports. Search/fold Go codegen specs exist.
- `internal/builtins/list_iterative_search_test.go`, `internal/vm/list_hof_parity_test.go`, and `tests/stdlib/list_iterative_test.ail` provide reusable regressions. Review their coverage before adding cases.
- All six extrema now call iterative foldr seeded with the last element. Thus their recursive-depth and quadratic-traversal defects are already addressed in source. Direct raw-value extrema builtins required by design D4 remain absent; implement these as a completion/constant-factor improvement, without claiming the extrema are still recursive.
- Fallback warning remains inside `if !params.Quiet` in `internal/runner/entrypoint.go`. VM DefaultMaxStack remains 1000 with the stale evaluator-parity comment. Trace sample still contains findIndexHelper events.
- contains/nth/last/member/dedup already delegate; no changes needed. Iterative dedup does not imply linear equality/set-search cost.

### Velocity and verification limits

Ran the existing analyze_velocity.sh for 7 days. This is a shallow clone with one visible commit; HEAD~1 is unavailable. The script's changelog sampling includes old entries and its pipe warning is not evidence of current velocity. No defensible observed LOC/day or completion-rate estimate can be computed. Use **150 planned LOC/day**, not measured velocity; 750 LOC over 5 days includes 25% time contingency over 24 focused hours.

`go` is absent. Installed `ailang` reports a stale-binary warning. No fresh-build coverage, tests, benchmarks, or quorum result is claimed in this plan. M1 requires a build-capable executor; approval permits planning despite optional quorum unavailability, but never permits claiming unrun verification.

## Registry Reuse Audit

Executed `ailang pkg search list`: returned sunholo/a2ui, sunholo/duckdb, sunholo/decisions. None supplies language-runtime list HOF ports, monomorphic builtin registration, VM routing, or runner engine diagnostics; no plausible candidates required pkg info/docs inspection. There is no package-like API being introduced: these are stdlib/runtime internals, for which a package cannot replace the VM primitive path.

All milestones record **none**, with reuse of existing in-repository registry, adapter, HOF, parity, and example tooling. No package dependency or contribution notification required.

## Milestones and Day-by-Day Work

### M1: Fresh-build baseline and inherited implementation verification (~180 LOC)

**Goal:** verify current committed work and recover reproducible evidence rather than reimplement it.  
**Estimate:** 0 implementation + 180 tests/benchmark/example LOC; Day 1, 6 hours.  
**Dependencies:** none; Go toolchain and normal build dependencies required on execution runner.

Tasks:

- Read builtin-developer and use-ailang skills before builtin work or writing fixtures; obtain `ailang prompt` before editing any .ail file. Build via make build and record binary commit/version, OS/CPU, exact flags.
- Run existing targeted builtin/VM parity tests and stdlib fixtures. Review short-circuit call counts, Option results, non-commutative foldr order, malformed arguments and indexed callback errors; add only missing meaningful cases.
- Add `examples/runnable/list_iterative_remaining.ail` as a parameterized pure entrypoint covering searches, right fold and extrema. Use pure return assertions for strict VM runs: reporter println requires an EvalOnly bridge, so an IO repro alone cannot establish strict-native execution. Update example manifest using existing tooling.
- Record separate interpreter/strict-VM results for 20k/40k/80k integers and 320k records, and 200k right fold. Keep setup/list creation outside microbenchmark timing where possible; separate traversal from callback work.

**Files:** existing search test, VM parity test, stdlib iterative fixture, new runnable example, example manifest; optional benchmark additions adjacent to search tests. Reuse existing list_helpers example.

Acceptance:

- [ ] Freshly built binary identity and baseline commands/results are recorded; installed stale binary is not used as HEAD evidence.
- [ ] any/findIndex return equal results at 20k on interpreter and strict VM; first-match callback counts and None/Some values are pinned.
- [ ] foldr completes at 200k with non-commutative result and right-to-left callback order on interpreter and strict VM.
- [ ] Missing callback/type-error cases are covered without duplicating existing tests; no builtin or HOF table reimplementation unless a measured failure requires repair within the approved design.

**Risk:** inherited code may fail fresh tests. Use contingency for localized repair; revise scope if the failure needs architecture changes.

### M2: Direct extrema builtins with exact semantic parity (~330 LOC)

**Goal:** finish D4 using six pure monomorphic raw-value builtins and guarded Option wrappers.  
**Estimate:** 150 implementation + 180 tests = 330 LOC; Days 2–3, 12 hours.  
**Dependencies:** M1.

Tasks:

- Add registration and implementations in `internal/builtins/list_extrema.go`; use the existing type builder and metadata convention. Add corresponding GoCodegenSpecs to registry_codegen_list.go, maintaining existing stdlib export types.
- Wrappers in std/list.ail check `_list_length` and return None for empty; only nonempty inputs call `_list_maximumInt`, `_list_minimumInt`, `_list_maximumFloat`, `_list_minimumFloat`, `_list_maximumString`, `_list_minimumString`. Raw builtin invocation on empty or malformed input returns a descriptive error, never a fabricated zero.
- Preserve current behavior by seeding from the last value and scanning right-to-left with strict `>`/`<`, retaining the accumulator on ties/unordered comparisons. This satisfies design D5's later-wins intent while correcting its proposed left-to-right >=/<= algorithm for float NaN. No float normalization or new comparison policy.
- Reuse automatic adapter routing; do not add six manual VM implementations. Test registration/type schemes, adapter classification, pure-builtin coverage ratchet and index bounds. Register codegen mappings for all six.
- Test empty/single/repeated values, negative and ordinary values, strings, infinities, NaN at each position, and signed zero (compare bits where needed). Compare against the current foldr definitions, including exact floating-point identity, across evaluator, VM adapter and generated Go.

**Files:** new list_extrema.go/list_extrema_test.go, registry_codegen_list.go, std/list.ail, existing adapter/parity test modules, tests/codegen-harness/list_ops.ail, runnable example from M1.

Acceptance:

- [ ] Six pure monomorphic raw-value builtins are registered and all exported Option signatures remain unchanged.
- [ ] Empty wrappers return None without calling a raw builtin; invalid raw inputs fail explicitly.
- [ ] Int/string ties and float NaN/infinity/signed-zero results match the existing right-fold implementation on all three engines.
- [ ] Extremes complete on 200k elements without recursive frames; VM adapter eligibility and index limits pass and unportedPure does not grow.
- [ ] Go codegen builds and executes extrema examples with parity; no callback fold remains in these wrappers.

**Risk:** comparison-order changes can silently alter NaN results. Use reference fold tests before replacing wrapper bodies. Shared comparison code must retain monomorphic registry types.

### M3: Engine visibility and truthful list documentation (~120 LOC)

**Goal:** finish approved design Phase 4, retaining the severable D7 behavior change for review.  
**Estimate:** 30 implementation + 50 tests + 40 docs = 120 LOC; Day 4, 4 hours.  
**Dependencies:** M2 (final complexity docs describe final implementation).

Tasks:

- Emit the VM-to-evaluator fallback warning even under --quiet, preserving strict-bytecode failure behavior. Correct DefaultMaxStack's comment without changing the constant or introducing flags.
- Add a runner regression with a deterministic VM failure and successful evaluator path; verify captured stderr under quiet and nonquiet, strict failure without fallback, and no warning on a successful VM run.
- Update std/list header and per-helper costs: O(n) traversal/O(1) helper stack, predicate short-circuit, foldr right-to-left. Callback and cons/concat costs are additional; a consing fold remains potentially quadratic.
- Use CLI-doc-maintainer skill for changed quiet behavior/help if applicable, and existing runner test infrastructure; update locally available reference docs describing bytecode fallback.

**Files:** internal/runner/entrypoint.go, existing/new runner fallback tests, internal/vm/vm.go, std/list.ail, affected CLI help/reference source discovered by executor. Example: reuse M1 pure example for successful native run and a runner test fixture for forced fallback.

Acceptance:

- [ ] Engine fallback is visible on stderr under --quiet; strict mode fails without evaluator retry, and native success emits no fallback warning.
- [ ] DefaultMaxStack stays 1000 with an accurate comment; CLI help matches final behavior.
- [ ] `ailang docs std/list` reflects the iterative helper set and costs without falsely promising linear callback/cons or dedup costs.

**Risk:** D7 is an observable CLI change. Sprint review may sever only warning behavior and its test; then update both plan/JSON explicitly before execution. Default plan includes it under approved design handoff.

### M4: Performance evidence, traces and completion gates (~120 LOC)

**Goal:** bank end-to-end proof and complete documentation artifacts.  
**Estimate:** 0 implementation + 120 documentation LOC; Day 5, 2 hours plus 6 hours contingency.  
**Dependencies:** M1, M2, M3.

Tasks:

- Repeat matched benchmarks after final changes; record startup separately and compare medians of sufficiently long 20k/40k/80k (or larger) runs. Target doubling ratio 1.5–3 for traversal; noisy/startup-dominated points require larger inputs, not weakened correctness tests. Record absolute reporter targets (<1s at 80k; single-digit seconds at 320k records) with hardware context rather than imposing universal flaky wall-clock CI thresholds.
- Regenerate examples/traces/list_helpers.jsonl using existing trace tooling, checking predicate call count/order and absence of findIndexHelper. Wrapper-frame removal is intentional; do not assert byte-identical timestamped traces.
- Run make test, make verify-examples, make lint, make check-boundaries, make simplicity-audit, and make test-coverage-badge; run targeted generated-Go harness cases. Resolve regressions before handoff.
- Update changelogs/v0.32-current.md via CHANGELOG conventions, and reconcile/move the approved design to implemented/v0_51_3 only after all criteria pass. Record the plan's actual outcomes, including source drift and any severed D7 item.

**Files:** examples/traces/list_helpers.jsonl, changelogs/v0.32-current.md, design doc on completion; reuse M1 runnable example and existing codegen harness.

Acceptance:

- [ ] Final interpreter, strict VM and Go-codegen correctness matrix passes; 200k folds/extrema and 320k-record searches complete without unintended fallback.
- [ ] Scaling and absolute timing evidence are recorded with machine/flags/sample methodology and callback-cost caveats.
- [ ] Trace sample is current and preserves callback event order/count; all required test/example/lint/boundary/simplicity/coverage checks pass.
- [ ] Design status, changelog and executor progress reflect verified results; no milestone marked passing on source inspection alone.

## Success Metrics and Dependencies

750 estimated LOC across four milestones; JSON LOC sum and 5-day duration must match. New/changed extrema implementation targets at least 90% focused branch coverage where tooling supports it; no repository-wide coverage regression. Existing runnable list_helpers and the new list_iterative_remaining example must type-check/run; strict runs use a pure entrypoint. No changes to effectful combinators, SMT rules, list representation/cons substrate, teaching prompts, VM frame-depth policy, or unrelated mapAccumL behavior.

Coordinate std/list comment edits with m-foldl-cons-cost-model and reuse existing TCE behavior. A build-capable runner is mandatory for execution, not for plan authoring. Optional design quorum remains unrun; the handoff says prior design work is approved. No additional architectural decision blocks planning.

## Handoff

Machine state: `.ailang/state/sprints/sprint_M-ITERATIVE-LIST-REMAINING.json`, all passes null and status not_started. Prior design approval does not imply sprint approval. The coordinator publishes this plan for review; merge/approval of the sprint-plan task triggers sprint-executor under resources/coordinator.md. Do not manually launch implementation or send a duplicate execution request before that gate. Executor starts with M1 and loads required implementation skills; sprint-evaluator runs after completion against the design and criteria above.
