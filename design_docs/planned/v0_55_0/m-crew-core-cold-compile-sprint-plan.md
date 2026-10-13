# Sprint Plan: crew-core cold compile follow-up

**Design:** [approved follow-up](m-crew-core-cold-compile-followup.md)
**Status:** Planned; coordinator review precedes execution.
**Duration:** 9 engineering days including roughly 25% investigation buffer; nine milestones, each scoped to one 60–90 minute executor run. Day estimates include profiling, review and validation outside the edit run.
**Risk:** High (typechecker correctness and retained heap).
**Scope:** Verify the shipped #1811 memory fix and reduce residual CPU cost. No union-find, record representation redesign, new CLI flags, or external repository edits.

## Current status and velocity

Repository version is v0.54.8. The approved design measures session-cold/deps-warm check at 7.0 s / 124 MB and fully cold package check at 29.1 s / 134 MB. These are design-session measurements, not fresh planner measurements. The memory fix, scheme constraint guard and profiling CLI already exist. Current code confirms rendering-based constraint dedup, clearing pooled substitution maps and full effect-row environment traversal.

The velocity script was run for seven days. This is a shallow checkout with only one visible commit (13048651); its script finds no usable recent LOC metrics. No empirical LOC/day claim is possible. Planning capacity is 140 LOC/day, 1,260 total changed/new LOC including tests and docs across nine days, based on the design's nine-day estimate and conservative validation overhead. Test/fixture work is approximately half the implementation scope. Splitting extractions from optimizations keeps executor runs bounded.

## Approval and dependencies

The handoff states prior design work is approved; treat D1 preferred narrowing with its specified fallback, D4 3x timing headroom and D5 docs-only export guidance as the approved choices. The unchecked design-freeze boxes remain evidence checkpoints: M1/M6 pin #386, M8 records the actual fixed budget, and M9 supplies the guidance. This plan does not authorize execution itself; coordinator plan approval/merge triggers the executor per its workflow. No agent message is sent before that approval.

External performance verification requires ailang-packages at 6dfc0fa, content-identical path dependencies when registry access is unavailable, and CI-grade linux x64; darwin arm64 is a second-platform verification when available. A missing platform or checkout is reported explicitly, while local deterministic guards remain runnable.

## Registry reuse audit

All M1–M9 decisions: **none**. These are compiler-internal Go algorithms, pipeline tests and repository documentation, not package-like capabilities; registry searches are not applicable and none were run. An AILANG registry dependency cannot implement inference internals. crew_core/social_dynamics are pinned external validation inputs, not new dependencies. Each milestone has this concrete rationale in JSON.

## Syntax and examples

AILANG prompt version loaded: v0.16.7 (`ailang prompt`, active prompt version); repository compiler version v0.54.8. Reload the active teaching prompt before writing generated fixture source. M1 uses test-only generated AILANG, no new public language feature or showcase module.

| Module | Contracts | Effects | Inline tests |
|---|---|---|---|
| M1 generated wide-record fixture | include: ensures on deterministic record transformations to retain contract inference cost | include: pure functions with explicit ! {} | skip: Go pipeline assertions provide the compile-cost and semantics checks |
| Existing #386 showcase fixtures | skip: preserve existing contracts unchanged | include: pin show ! {} and println ! {IO} | skip: existing Go integration tests assert effects |

Existing examples to verify: record_cons_pattern.ail, record_in_result.ail, record_list_extraction.ail, deriving_eq.ail, eq_containers.ail. Verify std/list, std/json and std/sem through existing stdlib tooling.

## Milestones

### M1: Pin baseline and semantic fixtures (~140 LOC)

**Dependencies:** None
**Files:** internal/pipeline/crew_core_compile_fixture_test.go; internal/pipeline/crew_core_compile_bench_test.go
**Executor budget:** 60–90 minutes; stop with a checkpoint if investigation exceeds it.

Build a deterministic test-only source generator matching 17-field State, 11-field Session, 24 functions and deep lets. Capture cold timing, TotalAlloc, accepted/rejected diagnostics and decoded compile artifacts. Run existing #386 and #1811 fixtures before optimizations.

**Acceptance criteria:**

- [ ] Network-free fixture compiles with caching disabled through the actual pipeline API.
- [ ] #386 row-generalization and application-effect fixtures and depth-12 64 MB guard pass before edits.
- [ ] Record hardware, compiler commit, Go version, three cold samples and baseline profile.
- [ ] Run make test-changed, focused touched-package tests and make check-file-sizes. Compare semantics with M1; use raw gob equality only if repeated unchanged builds prove encoding deterministic, otherwise compare decoded canonical contents and record the serialization limitation.

### M2: Use qualified constraint bookkeeping (~100 LOC)

**Dependencies:** M1
**Files:** internal/types/inference_helpers.go; internal/types/scheme_constraints_test.go
**Executor budget:** 60–90 minutes; stop with a checkpoint if investigation exceeds it.

Audit all constraint writers, then replace the whole-context class filter with qualifiedConstraints. Keep unsolved non-generalized constraints enforced at the declaration boundary.

**Acceptance criteria:**

- [ ] Constraint selection, first-wins order and errors match baseline.
- [ ] Tests cover constraints mentioning generalized and non-generalized variables and nested lets.
- [ ] Run make test-changed, focused touched-package tests and make check-file-sizes. Compare semantics with M1; use raw gob equality only if repeated unchanged builds prove encoding deterministic, otherwise compare decoded canonical contents and record the serialization limitation.

### M3: Replace rendering-based constraint deduplication (~170 LOC)

**Dependencies:** M2
**Files:** internal/types/scheme_constraints.go; internal/types/scheme_constraint_key.go; internal/types/scheme_constraint_key_test.go
**Executor budget:** 60–90 minutes; stop with a checkpoint if investigation exceeds it.

Compute a cycle-safe deterministic structural fingerprint during the relevance traversal. Resolve hash collisions by equivalence checking; hash equality alone must never discard a constraint. Preserve the existing String-key equivalence relation, including field order and variable identity.

**Acceptance criteria:**

- [ ] Corpus includes wide/open records, rows, ADTs, recursive types, aliases and class distinctions.
- [ ] Forced collisions retain distinct constraints; genuine duplicates preserve the first object.
- [ ] Artifact and diagnostic comparison and constraint-growth guard pass.
- [ ] Run make test-changed, focused touched-package tests and make check-file-sizes. Compare semantics with M1; use raw gob equality only if repeated unchanged builds prove encoding deterministic, otherwise compare decoded canonical contents and record the serialization limitation.

### M4: Window effect-row environment traversal (~130 LOC)

**Dependencies:** M3
**Files:** internal/types/env.go; internal/types/env_effect_window_test.go; internal/types/inference_helpers.go
**Executor budget:** 60–90 minutes; stop with a checkpoint if investigation exceeds it.

Mirror the existing type-variable frame-window algorithm only after establishing the base-frame invariant for effect rows. Keep inherited free rows unquantified and quantified scheme rows excluded.

**Acceptance criteria:**

- [ ] Inherited, shadowed and quantified row-variable tests preserve generalization semantics.
- [ ] #386 fixtures and semantic artifact comparisons pass; record profile delta.
- [ ] Run make test-changed, focused touched-package tests and make check-file-sizes. Compare semantics with M1; use raw gob equality only if repeated unchanged builds prove encoding deterministic, otherwise compare decoded canonical contents and record the serialization limitation.

### M5: Bound epoch-stamped substitution scratch maps (~190 LOC)

**Dependencies:** M4
**Files:** internal/types/pool.go; internal/types/substitution_visit.go; internal/types/unification_substitution.go; internal/types/substitution_visit_test.go
**Executor budget:** 60–90 minutes; stop with a checkpoint if investigation exceeds it.

Replace per-return delete churn with exclusively owned epoch-stamped scratch state. Bound retained entries and references with size-based retirement/reset, handle epoch wrap, and audit all map consumers. Preserve cycle termination.

**Acceptance criteria:**

- [ ] Stale entries miss across passes; recursive ADTs and mu-types terminate with unchanged results.
- [ ] Epoch wrap, repeated distinct compilations, reentrant traversal and race-focused checks pass.
- [ ] Retained heap stays bounded; benchmark allocation and CPU deltas support the change.
- [ ] Run make test-changed, focused touched-package tests and make check-file-sizes. Compare semantics with M1; use raw gob equality only if repeated unchanged builds prove encoding deterministic, otherwise compare decoded canonical contents and record the serialization limitation.

### M6: Extract and pin application substitution behavior (~110 LOC)

**Dependencies:** M5
**Files:** internal/types/typechecker_functions.go; internal/types/application_substitution.go; internal/types/application_substitution_test.go
**Executor budget:** 60–90 minutes; stop with a checkpoint if investigation exceeds it.

Move the application substitution helper into a companion file without behavior change before narrowing. The current file is 780 lines; avoid crossing the 800-line limit. Pin closed show {}, println {IO}, nested applications, recursive self calls and declaration-boundary constraint replay.

**Acceptance criteria:**

- [ ] Extraction preserves diagnostics and decoded compile artifacts.
- [ ] Focused fixtures exercise withheld enclosing variables, replayed equalities and subtree CoreTypeInfo consumers.
- [ ] Run make test-changed, focused touched-package tests and make check-file-sizes. Compare semantics with M1; use raw gob equality only if repeated unchanged builds prove encoding deterministic, otherwise compare decoded canonical contents and record the serialization limitation.

### M7: Reduce application substitution work (~150 LOC)

**Dependencies:** M6
**Files:** internal/types/application_substitution.go; internal/types/application_substitution_test.go
**Executor budget:** 60–90 minutes; stop with a checkpoint if investigation exceeds it.

Apply approved D1 narrowing only to proven necessary operand effect rows and locally bound types. If fixtures show narrowing unsafe, use the approved per-local-solve memoization alternative; invalidate on substitution changes and do not share entries across solves.

**Acceptance criteria:**

- [ ] All #386 fixtures, recursive self-call withholding and declaration-boundary replay pass unchanged.
- [ ] No unresolved type is left for downstream CoreTypeInfo consumers.
- [ ] Re-profile deps-cached session and package check; report goals achieved or remaining bottleneck honestly.
- [ ] Run make test-changed, focused touched-package tests and make check-file-sizes. Compare semantics with M1; use raw gob equality only if repeated unchanged builds prove encoding deterministic, otherwise compare decoded canonical contents and record the serialization limitation.

### M8: Install compile time and allocation regression guards (~150 LOC)

**Dependencies:** M7
**Files:** internal/pipeline/compile_time_guard_test.go; internal/pipeline/crew_core_compile_fixture_test.go
**Executor budget:** 60–90 minutes; stop with a checkpoint if investigation exceeds it.

Calibrate a fixed committed wall budget with 3x headroom from post-change CI-grade measurements; never derive the budget dynamically in the test. Measure time and TotalAlloc together with clear failure output. Keep the original depth-12 allocation guard unchanged.

**Acceptance criteria:**

- [ ] Guard runs without registry/network and asserts both time and allocation.
- [ ] Attempt the proposed 64 MB wide-fixture budget; if baseline cannot fit, report evidence and request a design revision rather than silently weakening it.
- [ ] Fixed budget and calibration environment are recorded; two-week stability is tracked after merge, not claimed during execution.
- [ ] Run make test-changed, focused touched-package tests and make check-file-sizes. Compare semantics with M1; use raw gob equality only if repeated unchanged builds prove encoding deterministic, otherwise compare decoded canonical contents and record the serialization limitation.

### M9: Document measurements and downstream retirement (~120 LOC)

**Dependencies:** M8
**Files:** docs/docs/guides/cold-compile-performance.md; design_docs/planned/v0_55_0/m-crew-core-cold-compile-followup.md; changelogs/unreleased/2026-10-13-crew-core-cold-compile.md
**Executor budget:** 60–90 minutes; stop with a checkpoint if investigation exceeds it.

Record fully cold, dependency-warm/session-cold and fully warm check plus both execution engines against packages commit 6dfc0fa. Document downstream CI warm-up removal and exported warm-cache build pattern, compiler/source/dependency cache invalidation and fail-on-missing-cache guidance. External game edits are outside this sprint.

**Acceptance criteria:**

- [ ] Deps-cached session check <=2.5 s and fully cold package check <=12 s with peak RSS <=300 MB, or explicitly report unmet goals and propose a separate follow-up.
- [ ] Both engines run bankLimit successfully; explain encodeHostError argument decoding limitation.
- [ ] Guide documents two concurrent cold runs and memory measurement without claiming an unperformed game CI run.
- [ ] Changelog fragment opens with ###; implementation report includes profile deltas and semantic comparisons.
- [ ] Run make test-changed, focused touched-package tests and make check-file-sizes. Compare semantics with M1; use raw gob equality only if repeated unchanged builds prove encoding deterministic, otherwise compare decoded canonical contents and record the serialization limitation.

## Day-by-day schedule

| Day | Work |
|---|---|
| 1 | M1 baseline, fixtures and profiling |
| 2 | M2 constraint source audit and optimization |
| 3 | M3 structural key and collision tests |
| 4 | M4 effect environment window and comparison |
| 5 | M5 epoch pool, retained heap and cycle checks |
| 6 | M6 helper extraction and #386 consumer audit |
| 7 | M7 narrowing or memoization and CPU profile |
| 8 | M8 fixed guard calibration and allocation checks |
| 9 | M9 external measurements, docs and final validation |

## Validation and success metrics

Executor gates are `make test-changed` (changed packages plus core), focused types/pipeline tests, `make check-file-sizes`, `make check-boundaries`, `make check-changelog`, `make verify-stdlib` and `make verify-examples`. Run gofmt on changed Go files and the relevant existing lint target. CI owns the full suite; it is not an executor acceptance gate. Collect race checks specifically for changed scratch-pool ownership. Preserve existing coverage and cover new collision, stale-epoch, wrap and generalization branches rather than inventing an unmeasured repository coverage percentage.

Measurements use three comparable cold samples and report median, range, peak RSS, exact CPU/hardware and Go/compiler versions. Distinguish total package-cold from session-only cold. Never delete a user's cache: use a disposable pinned checkout and targeted cache invalidation. RSS <=300 MB is distinct from TotalAlloc <=64 MB; record both, and reject epoch designs retaining historical type graphs. Session target <=2.5 s and package target <=12 s remain performance deliverables; a miss requires a follow-up decision, not union-find in this sprint. The 3x CI time budget provides headroom, not a substitute for those deliverables.

Semantic acceptance compares diagnostics, constraint selection/order, resolved type/effect results and deterministic decoded artifacts after each optimization. Raw gob bytes may encode maps nondeterministically; establish same-build determinism before demanding raw-byte equality. Any semantic difference is a regression.

Two consecutive weeks without guard flake is a post-merge observation owned by the maintainer, with review after 14 days; sprint completion cannot claim future stability. Downstream warm-up retirement and darwin measurements are recorded as performed or pending. #1811 is already closed; use refs #1811 and refs #386 for provenance and do not automatically re-close these prior issues.

## Risks and stop conditions

Hash collisions must not change equivalence. Frame-window assumptions must not quantify an inherited free effect row. Epoch stamping must not retain an unbounded history of type graphs. Application narrowing requires full downstream consumer evidence; use the design fallback when unsafe. If the allocation budget or time target is unattainable with this scope, preserve measurements and request a separate design revision. Follow-on union-find and record-layout changes remain deferred.

## Handoff

Coordinator artifacts are the plan and sprint JSON below. All milestones start uncompleted. Plan review/merge supplies the execution gate; the executor loads these artifacts and the sprint-executor skill, then sprint-evaluator reviews the implemented result. No compiler source was changed by planning.
