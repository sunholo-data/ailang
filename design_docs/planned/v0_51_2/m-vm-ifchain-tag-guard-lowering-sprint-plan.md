# Sprint Plan: M-VM-IFCHAIN-TAG-GUARD-LOWERING

## Summary

Restore strict-bytecode parity for constructor sub-pattern tag checks and guards referencing their own pattern bindings. Follow the approved [design](m-vm-ifchain-tag-guard-lowering.md); this sprint supplies prerequisites for the nested-pattern and variable-default-arm siblings, rather than implementing their scopes.

**Status:** Planned; design approval supplied by coordinator handoff. Sprint execution awaits the coordinator's plan approval gate.
**Target:** v0.51.2; retain the active version folder, no release/version bump in this sprint.
**Duration:** 2 working days, 16 hours including 3 hours of contingency (25% over approximately 12 hours of work, rounded up).
**Estimated change:** 600 LOC: 130 implementation, 380 tests/fixtures, 90 examples/docs.
**Risk:** Medium: shared lowering file, continuation duplication, lexical scope analysis, shared Go emitter.
**Tracking:** #1511 is the owning issue (design commit 07846355). #1473 and #1503 are related reports; #1355 is background, not an issue to auto-close here.

## Current Status Analysis

At planning HEAD `07846355`, `std/VERSION` is v0.51.0 and the working tree was clean. Source inspection confirms the fabricated `FieldAccess("Tag")` constructor condition and guards ANDed into conditions before body bindings. Neither fix is implemented. The design contains a systemic audit of constructor, cons, exact-list, record, tuple, variable, and switch-path cases and a 31-row verification log.

The seven-day velocity script found only the design commit and no usable LOC metrics. Git history is shallow (one visible commit); archive-wide LOC matches are historical, not current velocity. Recent v0.51.0 changelog work established canonical builtin tables, native ADT constructor names, and builtin coverage checks, all usable foundations. No empirical LOC/day or completion rate can be calculated here. Budget 300 planned LOC/day as a capacity assumption, not a measured velocity. The design's 16-hour estimate and localized source review set the schedule; tests dominate the work.

Go is absent in this runner; coverage and Go tests cannot be measured during planning. Executor needs a Go toolchain and must build a fresh binary before using strict runs (the installed CLI warns it may be stale). Do not use the design's ephemeral `/tmp/repro` files as execution dependencies: reconstruct and check permanent fixtures. Run `ailang prompt` before authoring any `.ail` file.

## Registry Reuse Audit

On 2026-10-02, `ailang pkg search pattern` returned only `sunholo/config@0.1.2`; `ailang pkg search bytecode` returned no candidates. `ailang pkg info sunholo/config` and `ailang pkg docs sunholo/config` confirm environment configuration functions, unrelated to compiler pattern lowering. M1, M2, M3 each decide **none**: these are Go compiler/VM internals and their first-party regression tests, not package-like user capabilities. Reuse the existing Statement IR, native builtin dispatch contract, and golden harness; add no package dependency or new sweep script.

## Merge Order and Scope Contract

Land this sprint first, then the nested-pattern sibling, then the variable-default-arm sibling, sequentially. All three edit `internal/gen/lower/match.go`; do not execute their edits concurrently. Both siblings must retain `_adt_tag` name comparisons and bind-before-guard assembly. The nested sibling must replace its obsolete fabricated Tag-field proposal and guard-AND assumption with these mechanisms when executing; the variable sibling's V4/AC2 must run on top of M2. Retain sibling doc paths; promoting the nested sibling's stale v0.49.1 target to the active release queue is separate planning work.

If a sibling has already landed by executor start, adapt to its actual recursive binding/condition helpers and eligibility gate instead of overwriting them. Re-run the combined shapes on the composed tree. No sibling is a prerequisite for standalone C/G acceptance. The original `a :: b :: _` failure is expected to remain until the nested sibling lands; completion of this sprint must not claim the original report resolved.

## Proposed Milestones

### M1: Native ADT tag checks (~180 LOC)

**Goal:** Constructor conditions in if-chain positions compare ADT constructor names without treating values as records.
**Estimated:** 25 implementation + 155 unit/golden fixture LOC; 4 hours on day 1.
**Dependencies:** None.
**Files:** `internal/bytecode/builtin_names.go`, `internal/vm/builtins.go`, new `internal/vm/builtins_adt_test.go`, `internal/gen/lower/match.go`, `internal/gen/lower/lower_match_test.go`, `tests/golden/bytecode/golden_test.go`, new `tests/golden/codegen/ifchain_tag_guard.ail`.

Append `_adt_tag` to canonical names and the corresponding native dispatch entry in lockstep; never insert among existing native entries. Return the stored constructor name for an ADT; explicitly reject wrong arity and non-ADT values. Replace only the if-chain ConstructorPattern tag projection. Keep positional field bindings and switch dispatch intact. Test two constructors sharing an ordinal across distinct ADTs to pin name-based comparison.

The design proposes a fixture under `tests/golden/bytecode/`, but `golden_test.go:232` loads named specs from `../codegen`; place the new shared fixture there and register its spec in the bytecode harness. Do not change fixture discovery for this fix.

- [ ] `_adt_tag` returns constructor names (including nullary constructors) and rejects wrong arity/non-ADT arguments with explicit errors; table agreement and index-cap checks pass.
- [ ] Lowered constructor conditions use `_adt_tag`, with no fabricated Tag record access; genuine record Tag fields still use record access.
- [ ] C1 cons-head and C2 exact-list constructors return hand-computed, evaluator-equal results in all three CLI modes; None/empty-list near misses select the default without unsafe projections.
- [ ] Existing switch-path ADT fixtures and native/adapted builtin coverage tests pass.

**Risk:** Shared Statement IR also feeds Go emitters. Inspect and probe affected emitter routes during M3; do not silently add broad emitter work to the sprint.

### M2: Bind pattern variables before guards (~280 LOC)

**Goal:** Guard evaluation sees the current arm's bindings and false guards resume the remaining arms in source order.
**Estimated:** 105 implementation + 175 test/fixture LOC; 5 hours across day 1 and day 2.
**Dependencies:** M1 for serial integration; the guard fix is technically independent.
**Files:** `internal/gen/lower/match.go`, new `internal/gen/lower/match_guard_test.go` (or extend `lower_match_test.go`), `tests/golden/codegen/ifchain_tag_guard.ail`, `tests/golden/bytecode/golden_test.go`.

Build the remaining-arm continuation from right to left. For referencing guards emit structural condition, then arm bindings, then a nested guard If with REST on false. Keep both structural-failure and guard-failure continuations correct. Preserve the current flat condition for guards that reference no variables bound by that arm. Inspect/reuse lexical free-variable traversal patterns, accounting for lambda and let shadowing and every reachable Core expression form; an unhandled node must not silently classify the guard as independent. Collect pattern variables recursively even though recursive projection/binding support remains sibling work. Cover single-arm, guarded irrefutable final-arm, and refutable-final-arm branches. Preserve the documented pre-existing unit fall-through for terminal no-match; this sprint does not redesign exhaustiveness.

- [ ] G1 exact-list, G2 cons, G3 mixed guarded var, G4 record, and G5 tuple matches agree with evaluator and hand-computed outputs in all three modes for guard true, guard false, and structural near misses.
- [ ] IR assertions place bindings inside the structural match and before the nested guard; no projection executes for an unmatched structural condition.
- [ ] False guards reach the next arm in source order; single-arm, last-arm and guarded irrefutable-arm paths honor guards.
- [ ] Guards using only enclosing variables retain the flat path; switch-path guards are unchanged; lambda/let shadowing and same names in later arms are tested.
- [ ] Three consecutive referencing guards compile with valid register allocation and choose the correct later/default arm.

**Risk:** Remaining-arm duplication is worst-case exponential, not globally bounded. Keep the independence fast path, test three guarded arms, validate prototypes, and record code/register size; any resource overflow must fail loudly rather than silently fall back or drop an arm.

### M3: Integration, corpus baseline, examples and documentation (~140 LOC)

**Goal:** Demonstrate standalone parity and no regressions while recording sibling-dependent gaps accurately.
**Estimated:** 50 tests/fixtures + 90 examples/docs LOC; 3 hours on day 2, plus shared contingency of 4 hours across the sprint.
**Dependencies:** M1, M2.
**Files:** `tests/golden/bytecode/golden_test.go`, `tests/golden/codegen/ifchain_tag_guard.ail`, new `examples/runnable/vm_ifchain_tag_guard.ail`, `docs/docs/reference/language-syntax.md`, `changelogs/v0.32-current.md`.

Add a runnable example covering constructor cons/exact-list conditions and binding-dependent guards, with deterministic matching and rejection results. Validate with `ailang check`, evaluator, bytecode, and strict bytecode. Pin independent expected outputs rather than accepting agreement between two wrong engines. Add a clearly named expected-failure cross-shape row for guarded nested cons, with the sibling path and flip ownership; no xfail is allowed for the C/G standalone cases. Whichever sibling lands second removes that expected failure and reruns ADT-in-cons plus depth-three nested cases.

Before edits, executor captures a baseline for all compilable functions in `std/` and `examples/`, including EvalOnly names/reasons, and all runnable entries with explicit args/caps. After edits, replay the same manifest and diff statuses and outputs. Use existing compiler/disassembly and corpus verification tooling; module-only files are inspected for compilation status rather than claimed runnable. Keep baseline and after-results in execution artifacts. Existing unsupported entries remain explicit baseline failures; zero *new* EvalOnly and zero changed previously-correct outputs are required, not universal strict support.

- [ ] Every standalone C/G/control golden passes without skip/xfail and the runnable example passes `ailang check` plus all three execution modes.
- [ ] Regression controls include `examples/pattern_matching_adt.ail`, std/json get/asString wrappers, `examples/runnable/recursion_quicksort.ail`, `examples/runnable/list_pattern_cons.ail`, an enclosing-parameter guard, and switch Some(x) guard.
- [ ] Corpus before/after manifests show zero new EvalOnly functions and zero changed previously-correct outputs; existing unsupported and sibling-dependent cases are named.
- [ ] Go emitter probes for C/G and controls have recorded results. A newly introduced regression blocks completion and is escalated with a concrete minimal fix/design amendment; pre-existing unsupported routes are recorded rather than claimed fixed.
- [ ] `make fmt`, `make test`, `make lint`, and `make check-boundaries` pass on the fresh source build; focused lowering/VM/compiler/golden tests pass.
- [ ] Reference and Unreleased changelog describe the two fixes and their scope; sibling composition checks/expected-failure flip responsibility are documented.

## Day-by-Day Execution

**Day 1 (8h):** 1h build, baseline and red fixtures; 3h complete M1; 3h begin M2, scope helper and assembly; 1h contingency.
**Day 2 (8h):** 2h finish M2 and guard matrix; 3h M3 example, corpus comparison, Go probes, docs and full checks; 3h contingency for integration/tooling or same-file conflicts.

The baseline hour is included in M1's 4h. Work totals 12h plus 4h contingency = 16h; the buffer exceeds 25% after scheduling rounding. If registry/compiler/emitter compatibility needs a semantic redesign, report the exact blocking result and update scope instead of expanding implementation silently.

## Validation and Success Metrics

Focused command: `go test ./internal/gen/lower ./internal/bytecode/... ./internal/vm ./tests/golden/bytecode`.
Required final checks: `make build`, `make fmt`, `make test`, `make lint`, `make check-boundaries`.
CLI parity protocol: use a fresh built binary and the same entry/args/caps under evaluator, `--bytecode`, and `--bytecode --strict-bytecode`; assert output and successful exit. All new `.ail` sources must be checked before running.

Cover every changed branch with meaningful unit or parity tests: builtin success/error, referencing/independent guards, matched/unmatched conditions, all arm positions and false-guard continuation. Record affected-package coverage during execution; no fabricated current percentage or arbitrary whole-repo target. Zero new EvalOnly and zero output regressions are release gates. No opcode, parser/type/effect change, fallback-policy change, sibling recursive rewrite, variable dispatch-gate fix, or release operation is authorized by this plan.

## Handoff

Machine progress: `.ailang/state/sprints/sprint_M-VM-IFCHAIN-TAG-GUARD-LOWERING.json`; all milestones start pending. Coordinator plan approval/merge triggers sprint-executor through the normal pipeline, then sprint-evaluator assesses this design's AC1-AC7. This planning task does not start code execution or self-approve the plan. External quorum was controller-only with all reviewers absent, as disclosed by the approved design; do not describe it as externally cleared.
