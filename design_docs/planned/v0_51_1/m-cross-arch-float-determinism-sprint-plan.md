# Sprint Plan: M-CROSS-ARCH-FLOAT-DETERMINISM

## Summary

Freeze the 11 std/math transcendental implementations so identical float inputs produce identical bits on amd64 and arm64 through the interpreter, strict VM and generated Go. Canonical values are the approved design's amd64 reference, with no changes to language signatures.

**Design:** [m-cross-arch-float-determinism.md](m-cross-arch-float-determinism.md)
**Target:** v0.51.1
**Sprint ID:** M-CROSS-ARCH-FLOAT-DETERMINISM
**Source task:** task-5077e633; planning task task-646c8186
**Related issue:** #1472 (design commit reference)
**Status:** Planned; implementation has not started.
**Approval:** The coordinator handoff explicitly approves previous design work. Treat that as D1 approval of amd64 canonical values and arm64 golden regeneration; the unchecked design checklist is historical, not an additional approval request. Sprint execution remains the next coordinator/user stage.
**Duration:** 5 working days / 40 hours, including 8 hours of contingency; native runner provisioning or required-check administration can extend elapsed time.
**Estimate:** 2,800 source/test/documentation LOC, excluding generated golden data.
**Risk:** High: numeric compatibility, transitive algorithm dependencies and generated-helper ownership.

## Current Status and Estimate Basis

Verified on 2026-10-01: interpreter registrations, VM math wrappers and codegen Inline templates still call host math.Exp/Log. internal/mathx does not exist. All four design phases remain open. The current branch is coordinator/task-646c8186 and was clean at planning start; no branch switch is needed.

The velocity script was run for seven days. This shallow checkout exposes one design-doc commit (35ae3261), no parent diff, and historical changelog LOC examples rather than reliable recent delivery measurements. Current changelog entries describe v0.51.0 VM coverage and float literal fixes but provide no measured duration for this work. **Measured LOC/day is unavailable.** The planning capacity of 560 LOC/day is an estimate, not observed velocity; most implementation lines are vendored/copied algorithms.

The design's ~1,700 total undercounts its own file list: ~700 vendored lines + ~700 emitted-helper lines + ~250 tests + ~120 probe lines already total 1,770 before wiring, backend tests, docs and CI. This plan allocates 2,800 LOC and five days instead of claiming that four days are evidence-backed. Re-estimate after source dependency closure is known in M2.

This container has ailang and Python but no Go, make or jq and no native arm64 runner. Planning needs none of those; execution must provision the repository-pinned Go 1.26.6 toolchain, make, and native amd64/arm64 execution. No Go coverage, compiler or numerical validation has been performed during planning.

## Registry Reuse Audit

Executed `ailang pkg search math` and `ailang pkg search float`. Math returned sunholo/relativity@0.4.0 and sunholo/deontic@0.3.0; float returned no packages. Inspected both candidates with `ailang pkg info` and `ailang pkg docs`: relativity provides application-level physics and deontic provides obligation/settlement arithmetic. Neither supplies portable Go transcendental kernels, backend glue or cross-arch tests. The CLI warned it may be stale; results describe the available registry view.

| Milestone | Decision | Reason |
|---|---|---|
| M1 | none | Canonical host-bit census and replay fixtures are compiler-internal verification, absent from candidates. |
| M2 | none | Requires a dependency-free Go leaf implementation; application AILANG packages cannot replace runtime math. Reuse Go's approved BSD source with attribution. |
| M3 | none | Builtin/VM/codegen integration and self-contained emitted helpers are repository-internal infrastructure. |
| M4 | none | CI, required-check evidence, limitations and migration documentation are repository-specific. |

## Milestones

### M1: Canonical corpus and architecture census (~250 LOC)

**Dependencies:** None. Native runners and pinned toolchain are execution prerequisites.
**Effort:** 4 hours; 120 probe implementation + 100 corpus tests + 30 report LOC.
**Files:** internal/mathx/probe/main.go, internal/mathx/golden/*.json, internal/mathx/corpus_test.go, implementation census report alongside this plan.
**Examples:** Plan the new examples/runnable/cross_arch_float_determinism.ail fixture; preserve examples/runnable/math_trig.ail and examples/float_nan.ail. Obtain `ailang prompt` before writing any .ail source and type-check it.

Use deterministic input bit patterns, a fixed sweep seed and binary arguments for atan2/pow. For each of the 11 functions include approximately 200 boundary/special cases and at least 10,000 sweep cases. Encode input/output uint64 bits as hexadecimal strings, not JSON numbers. Record toolchain, GOOS/GOARCH, seed and corpus hash. Generate expected outputs once on native amd64; arm64 consumes exactly that corpus and reports host deltas without regenerating expectations. NaN payload/sign behavior must be recorded explicitly; if portable bit preservation cannot match canonical bits, escalate rather than quietly weakening assertions.

Audit eval/VM/builtins arithmetic for multi-operation float expressions and run existing arithmetic suites on both architectures. The census is evidence of host behavior, not proof of the assumed FMA mechanism.

- [ ] Repro input and expected exp result 1.2293173989217931 are in the canonical corpus.
- [ ] All 11 functions have the specified special/boundary and deterministic sweep coverage, including two-argument cases.
- [ ] Native amd64 and arm64 host census reports identify differing functions, counts and representative ULP deltas with provenance.
- [ ] Corpus serialization round-trips every input/output bit pattern including signed zeros and non-finite values.
- [ ] Arithmetic audit records findings and existing interpreter/VM arithmetic suite results on both architectures.

**Risk:** Native arm runner availability; provision the M4 CI runner early without treating CI completion as already achieved.

### M2: Portable math core (~1100 LOC)

**Dependencies:** M1.
**Effort:** 8 hours; 800 vendored implementation/dependency closure + 300 unit tests.
**Files:** internal/mathx/{exp,log,log10,sin,cos,tan,asin,acos,atan,atan2,pow}.go plus necessary reduction/kernel helpers, mathx_test.go and BSD attribution/notice.
**Examples:** Use the M1 corpus and report fixture; no API additions.

Inventory the pinned Go pure-Go source dependency graph before copying: trig argument reduction, shared kernels and pow's dependencies must remain inside the portable closure wherever they perform transcendental work. Do not call host transcendental functions indirectly through a copied wrapper. Preserve source coefficients, branches, special values and source revision/license; force intermediate rounding at every multiplication feeding addition/subtraction, including nested Horner expressions and shared helpers. Exact constants and permitted primitive math operations stay unchanged.

- [ ] All 11 exports pass the same golden-bit assertions on native amd64 and arm64.
- [ ] Exp repro matches the committed expected bits; signed zero, subnormals, infinities and NaNs have explicit bit-level tests.
- [ ] Portable-vs-amd64-host comparison has zero deltas over the full corpus; any delta pauses M3 and re-escalates D1 with evidence.
- [ ] Dependency closure and fusion-proofing review finds no indirect host transcendental or unrounded multiply-add paths.
- [ ] Source provenance and BSD notices are retained; mathx remains a leaf package within documented architecture boundaries.

**Risk:** Design's source-size estimate omits reduction helpers; use the contingency and revise the estimate rather than dropping functions or tests. Do not claim the approved scope guarantees all arm64 shifts are at most one ULP until the census measures them.

### M3: Integrate interpreter, VM and self-contained codegen (~1250 LOC)

**Dependencies:** M2.
**Effort:** 12 hours; 850 implementation/emitted algorithms + 350 differential/codegen tests + 50 example LOC.
**Files:** internal/builtins/math_trig.go, registry_codegen_math.go; internal/vm/builtins_math.go and new math tests; internal/gen/golang/codegen_math_helpers.go, codegen.go, codegen_runtime.go, codegen_expr_simple.go, codegen_math_test.go and differential tests in the appropriate existing integration package.
**Example:** Create examples/runnable/cross_arch_float_determinism.ail with the original exp/log chain and ordinary-domain calls covering all 11 functions; check and run through interpreter and strict VM, then compiled Go. Preserve math_trig.ail and float_nan.ail.

Rewire all registrations and codegen function paths; keep constants and sqrt/floor/ceil/round/abs semantics intact. Emit portable helper bodies with their full dependency closure. Check names against actual user identifier mangling and test collisions. Test registry Inline and fallback paths, math-constant-only programs, lazy emission and multi-file skipRuntimeHelpers ownership. Multi-file output must contain each required helper once, without duplicate definitions or unresolved calls.

Compare backend result bits over the full committed corpus using existing embed/runner/codegen APIs and batched generated programs, not one compile per sample. Construct non-finite values through test APIs or bit reconstruction rather than unsupported .ail literals. Add a separate text assertion for the ordinary finite report case; bit equality and text formatting are separate checks.

- [ ] Interpreter, strict VM and compiled Go match canonical bits for all 11 functions over the full corpus on amd64 and arm64.
- [ ] Emitted-helper outputs bit-match mathx, guarding the two algorithm copies.
- [ ] Original finite exp/log chain text is 1.2293173989217931 and 0.20645905511071927 on both architectures through interpreter and strict VM; generated-Go finite formatting is checked without broadening show semantics.
- [ ] Registry/fallback emission, helper-name collisions, lazy emission, constants-only and multi-file compilation tests pass; generated programs remain stdlib-only.
- [ ] Existing tolerant builtin tests pass unchanged; codegen text expectations change deliberately; both existing conflict-surface examples and the new fixture pass.

**Risk:** Runtime helper order and multi-file emission are more complex than replacing mathFunctions strings. Batch corpus evaluation to keep CI cost bounded without reducing corpus coverage.

### M4: Native CI guard and release evidence (~200 LOC)

**Dependencies:** M3; runner setup can begin during M1.
**Effort:** 8 hours; 50 CI implementation + 50 CI/test integration + 100 documentation/report LOC.
**Files:** .github/workflows/ci.yml, docs/LIMITATIONS.md, changelogs/v0.32-current.md (current changelog referenced by CHANGELOG.md), implementation report alongside this plan.
**Examples:** Wire the new fixture and existing math fixtures into validation using current repository targets/harnesses.

Add ubuntu-24.04-arm execution with the pinned Go toolchain. Run mathx, VM, builtins, generated-helper/backend differential tests and make test-core; include the codegen/integration package hosting parity tests explicitly. The amd64 job must consume the identical corpus. If path filtering is used, include mathx, all callers/codegen, shared test harnesses, Go/toolchain config and workflow changes; required-check reporting must not hang on skipped jobs. Verify runner availability before relying on the label.

Record the compiled-mode user-arithmetic FMA limitation separately from transcendental guarantees. Publish measured before/after deltas, provenance and representative host-vs-portable timing using existing benchmark tooling; do not imply arbitrary simulation code is universally deterministic. Prepare a consumer migration note for arm64 golden regeneration and collapsing per-arch transcendental goldens after validation. Deliver it via the coordinator/issue handoff; sending a separate consumer message requires an authorized recipient.

- [ ] Native amd64 and arm64 CI assert the same corpus and execute all three backend comparisons successfully.
- [ ] Arm64 guard is included in required checks, or repository-admin configuration is identified as an outstanding release blocker with evidence; do not mark M4 passing while that criterion remains unmet.
- [ ] make test, make lint, make check-boundaries and make simplicity-audit pass; any audit regression is resolved or escalated before completion.
- [ ] LIMITATIONS documents compiled user-arithmetic fusion; current changelog and migration note accurately describe canonical values and measured arm64 changes.
- [ ] Implementation report includes corpus provenance, architecture census, zero-amd64-delta result, native CI runs and performance measurements.

**Risk:** Required-check configuration may need maintainer access. Configuration evidence is mandatory; a cross-build alone is insufficient.

## Daily Schedule

| Day | Work | Hours |
|---|---|---:|
| 1 | Provision runners/toolchain, M1 corpus/census; begin M2 dependency inventory | 8 |
| 2 | Finish M2 portable closure, bits and amd64 compatibility gate; start M3 | 8 |
| 3 | M3 backend wiring, emitted-helper closure and multi-file tests | 8 |
| 4 | Finish M3 full-corpus parity; M4 CI, documentation and required-check evidence | 8 |
| 5 | Contingency for source closure/runner failures, complete M4 checks and evaluator handoff | 8 |

Milestone estimates sum to 32 hours; the remaining eight hours are explicit contingency, not added scope. Do not progress past a failed numeric compatibility gate.

## Completion and Handoff

All four JSON features start with passes=null and no started/completed timestamps. Success requires every acceptance criterion, not just tolerant numerical tests. All 11 functions receive golden and special-value coverage; a numeric coverage percentage is not substituted for bit-level correctness. New example syntax is validated during execution, not claimed validated here.

Out of scope: compiled user-arithmetic fusion fixes, correctly-rounded replacement semantics, new math APIs, float formatting changes and unrelated tooling defects. The executor chooses corpus file layout and collision-safe helper names within the approved design.

Use `.ailang/state/sprints/sprint_M-CROSS-ARCH-FLOAT-DETERMINISM.json` to resume. Next stage is sprint-executor, followed by sprint-evaluator against this plan and the design. Execution commits should reference #1472; close it only with completed acceptance evidence. This planning stage does not implement the feature or authorize an automatic execution launch.

The installed CLI lacks `ailang agent`; the skill's legacy handoff command cannot run here. Return the artifact markers to the coordinator so it can route the next stage through its supported workflow. JSON is populated directly in the existing schema because the scaffold script requires unavailable jq and performs message import/read side effects. Validate syntax, dependency closure, estimates and registry entries before handoff.
