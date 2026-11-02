# Sprint Plan: M-EFFECT-ROW-OPEN-CLOSED-RELAND (#1958 M2 re-land)

**Status: ✅ COMPLETE 2026-11-02.** M2 and M3 landed as one commit, because F-B is unsound without the guard. Results are in the design doc's "Re-land implementation report".

## Placement and Human-Decision Rules

There are **no human-decision gates**. Every decision this sprint depends on is ruled:

| Ruling | Source |
|---|---|
| D-R1: reopen F2 for **lambdas**; F-B restores the original `\c. 1` spelling | design doc, "Decisions for this re-land" (Mark, attended, 2026-11-02) |
| D-R2: an IO lambda into a **pure** `map`/`any` slot is rejected whatever its body shape; `any IO lambda in IO caller` becomes must-reject | same |
| F2 still stands for **named and eta** function values into wider slots (Phase 2b, deferred) | same |
| D4: breaking soundness change, minor bump, no opt-out flag | design doc, Design Freeze F3 |

If an executor finds a corpus file outside these rulings that flips (a status change in any sweep),
**STOP: commit current work, report `BLOCKED: unruled flip <file>`, and end the run**.

## Summary

Re-land #1958's open-vs-closed row guard (M2, `830934d23`, reverted in v0.56.1). This time, fix
the two inference defects that made it reject valid programs: the distinct-tail union drop (cause
A) and lambda width (cause B).

**Design:** `design_docs/implemented/v0_56_0/m-effect-row-open-closed-soundness.md`, section
"Revert note and re-land brief (2026-11-02)". Fix parts F-A1, F-A2 and F-B; the spike measurements
are in that section.
**Base:** `dev` `90f31bbe0` (v0.56.1 + brief). **Target:** v0.57.0 (breaking, soundness).
**Duration:** 2 days. **Risk:** Medium. The spike proved the approach, but its globals must become
per-context state, and that wiring is the risk.
**Dependencies:** none open.

## Current Status Analysis

- ✅ Phase 0 (`std/ai` row-polymorphic streaming) and the explicit-empty-callback contract
  (`8fd84a200`) are shipped and unaffected.
- ✅ Spike at `05c860aaf` with M2 restored: 0 of 810 tracked and 0 of 256 external files change
  status. The Go reds are listed in the brief.
- ⏳ F-A1, F-A2, F-B, the restored M2 guard, tests and docs.
- **Velocity:** `internal/types` saw 26 commits in 14 days. The original #1958 sprint (Phase 0+1)
  took about 2 days. The spike diff is about 90 LOC of inference code; production adds plumbing
  and tests.

## Registry Reuse Audit

| Milestone | Search | Decision |
|---|---|---|
| M1–M4 | n/a | `none`: these are changes to compiler-internal type inference (`internal/types`); no package capability is involved |

## AILANG Syntax Gate

AILANG prompt version loaded: v0.16.8 (`ailang prompt --version-active`). `.ail` code is written
only as inline Go test fixtures (programs already checked by the spike). There are no showcase
modules. Contracts: skip (test fixtures). Effects: include (every fixture declares its row). Inline
tests: skip (Go-driven).

## Proposed Milestones

### M1_UNION_RESOLVE: F-A1 + F-A2 (union operands resolved, recursive self-tails absorbed)

**Goal:** a lambda body's effect union never closes to `{}` because its operands are still bare
tails.
**Estimated:** 140 implementation + 120 tests = 260 LOC.

- Make `combineEffects` / `combineEffectList` context-aware. All 11 call sites in
  `typechecker_data.go`, `typechecker_functions.go`, `typechecker_operators.go` and
  `typechecker_patterns.go` pass a resolver taken from `ctx`. No package-level hook.
- Add a lambda frame stack to `InferenceContext`: constraint checkpoint, and the enclosing-scope
  type/row/effect-row variables snapshotted at lambda entry. Push and pop in `inferLambda`.
- In the distinct-tail case only: run `solveLocalEqualities(checkpoint)`, drop the
  enclosing-scope bindings, apply `closeIfResolved`, then recombine. If the solve fails, use
  today's behaviour.
- F-A2: add a let-rec self stack to `ctx`, holding `inferLetRec`'s `bindingTypes`. An operand
  whose fully resolved tail equals an active self binding's effect-row tail contributes only its
  labels. Only self rows are absorbed. A callback parameter's `! {e}` is never absorbed (there is
  a test for this).
- Fix the `combineEffects` doc comment so it matches the behaviour.
- Put the new code in a companion file (`typechecker_effect_union.go`), because
  `typechecker_functions.go` is near the size limit.

**Acceptance criteria:**
- [x] Unit tests: the published lambda row of `\x. { println(x); println(x) }`,
  `\d. ex(p(), 1)` and `\d. if … then q(1) else q(2)` contains `IO`.
- [x] D-R2 matrix: all 7 pure-slot probes from the brief (single, block, block2, if, any-block,
  any-single, named) are rejected in a `! {IO}` caller.
- [x] `debug_effect.ail`, `debug_concurrent.ail` and `TestRecursiveConditionalStillRejectsRealEffect`
  pass. A recursive function that calls a `! {e}` callback still charges `e`.
- [x] A full sweep against `90f31bbe0` changes the status of only D-R2-class files, and the spike
  measured 0. *(Measured against `05c860aaf`, which has identical code: 0 flips.)*
- [x] `make test-changed` and `make check-file-sizes` pass.

### M2_LAMBDA_OPEN_ON_USE: F-B (Phase 2a)

**Goal:** an unannotated lambda whose latent row is closed fits a wider slot.
**Estimated:** 90 implementation + 80 tests = 170 LOC.

- In `inferLambda`: if the lambda is unannotated and its row is closed, use `{L | ρ}` with fresh
  `ρ`, and record `ρ` in a per-`InferenceContext` opened-tail set (not a global).
- Before both `applyApplicationSubstitution(sub)` calls (`typechecker_core.go`), bind every
  opened chain's terminal tail that is still unbound to closed `{}`. Register these through the
  `ApplicationEffects` closing and ownership model, so the publication invariant
  (`validate_effects_latent.go`) holds.
- Opened tails never appear in diagnostics, for example
  `No instance for Eq[[int -> int]]` (`TestEqContainersNegatives/fn_elements`).

**Acceptance criteria:**
- [x] `TestEqContainersNegatives`, `TestLatentNested_DistinctDeclaredTailsFailLoud` and
  `TestPkgConfinementMatrix` pass in a **full** package run, not only in isolation.
- [x] A full sweep shows 0 status changes compared with M1.
- [x] `make test-changed` passes.

### M3_RESTORE_GUARD: put M2 back, flip the guards

**Estimated:** 20 implementation + 140 tests = 160 LOC.

- Re-apply `830934d23`'s two guards in `row_unification.go`, and recover
  `internal/types/row_unification_open_closed_test.go` from `830934d23`.
- Unskip `TestEffectRowOpenClosedM2Deferred` and the deferred cases in
  `cmd/ailang/effect_row_soundness_test.go`.
- Make `any IO lambda in IO caller` must-reject (D-R2). Restore the two `EffectCeiling` subtests to
  `\c. 1` (D-R1). Update `TestLatentNested_IntendedFlips/map_concrete_callback_is_type_error` to
  accept the guard's wording.
- Add must-reject coverage for the named and eta width class (`runWide(loud, …)`), pinned as the
  Phase 2b deferral with a comment.

**Acceptance criteria:**
- [x] All 11 guard cases and the M2 rejections pass. The M2-only arm reproduces as `executor.ail`
  rejected (positive control).
- [x] Sweep over all `git ls-files '*.ail'` plus `ailang-packages` and `ailang-parse` from their
  package roots, using `perl -e 'alarm 90; exec @ARGV'` and the positive control: 0 unexplained
  flips.
- [x] Mutation, on a scratch copy: removing the guards turns R1–R5 green-accepted; removing F-A1
  turns the three cause-A guard cases red; *(with F-B present they stay green; the leak tests go red instead, see the report)* removing F-B turns `pure lambda into wider slot` red.
- [x] `executor_tools/executor.ail` checks clean.
- [x] `go test ./internal/... ./cmd/...` is green, except environmental tests named with their
  reason.

### M4_DOCS_PERF: changelog, docs, perf

**Estimated:** 80 LOC (docs).

- Add a changelog fragment, `changelogs/unreleased/2026-11-02-effect-row-open-closed-reland.md`,
  marked **Breaking (soundness)**. It covers the guard, D-R2's migration
  (`mapE`/`filterE`/`foldlE`/`forEachE`; there is no `anyE`) and the restored lambda width.
- Perf: record `make test-core` time and a type-check timing of the largest tracked modules,
  before and after. Flag anything over 10%.
- Add an implementation report to the design doc and update its status line.

**Acceptance criteria:**
- [x] `make check-changelog`, `make lint`, `make check-file-sizes`, `make verify-examples` and
  `make test-core` pass, run unpiped.
- [x] Perf numbers are recorded in the design doc.

## Success Metrics

- R1–R5 are rejected before execution on both engines. The four revert shapes plus the
  `EffectCeiling` originals are accepted.
- 0 unexplained flips across 810 + 256 files. `executor.ail` is clean.
- No global state added to `internal/types`.

## Risks

- **The local solve at each union costs time.** It runs only on distinct-tail unions. Measured in
  M4.
- **Per-context state misses a nested path, such as monomorphisation re-checks or the REPL.**
  Mitigation: full Go suites, and the spike's order-dependent reds become the regression tests.
- **External corpora move during the sprint.** Record the SHAs used in the sweep.
