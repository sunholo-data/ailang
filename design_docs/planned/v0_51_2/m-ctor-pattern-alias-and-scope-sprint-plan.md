# Sprint Plan: M-CTOR-PATTERN-ALIAS-AND-SCOPE

## Summary

Make constructor import aliases work in expressions and patterns, reject unknown constructor patterns, and preserve transitive constructor matching. Implement the approved [design](m-ctor-pattern-alias-and-scope.md) in the shared compilation pipeline.

**Status:** Planning complete; implementation not started. Design approval supplied by the task handoff. Sprint execution follows coordinator approval of this plan.
**Target:** v0.51.2
**Duration:** 3 days (2 implementation days plus 1 validation/buffer day), approximately 18–24 engineering hours.
**Estimated total:** 480 LOC: 135 implementation, 300 tests, 45 examples/documentation.
**Risk:** Medium — constructor identity, import precedence and transitive compatibility.
**Dependencies:** Existing M-MATCH-ADT-XCHECK machinery; no external package dependency.
**Source task:** task-60429428; planning worktree: coordinator/task-c949fdac.

## Current Status and Velocity

Source inspection confirms explicit constructor imports still pass `sym`, rather than the computed alias binding, to `resolveConstructorImport`. Imported constructor metadata lacks a canonical-name field; `compileFreshModule` registers constructors using the binding key. The typechecker has a populated transitive diagnostic constructor map, while constructor-pattern checking silently ignores names absent from the direct map. No implementation milestone is complete.

The velocity script was run for seven days. This shallow checkout contains only commit 233ebd4e, the design-doc addition (#1480); it supplies no implementation LOC/day evidence. The script's archived changelog hits are not recent velocity. The v0.51.0 changelog records related VM parity fixes, but no reliable duration/LOC pairs. Therefore 160 LOC/day is a planning capacity assumption, not a measured rate. The design's two-day estimate plus a third day provides validation capacity and uncertainty buffer.

The design includes a systemic audit of nullary/argument patterns, expressions, transitive imports, foreign ADTs and clashes (V1–V25). Its optional quorum attempted both reviewers but neither was available; it is not quorum-cleared. The supplied approval permits planning; reviewer availability is not an execution dependency.

## Registry Reuse Audit

Executed `ailang pkg search constructor`: no results. Executed `ailang pkg search pattern`: only `sunholo/config@0.1.2`. Inspected it with `ailang pkg info sunholo/config` and `ailang pkg docs sunholo/config`: it handles environment-variable configuration, not compiler name resolution. The binary emitted a stale-build warning; these results are registry evidence only, not semantic verification.

- **M1: none.** Constructor alias metadata and elaboration belong to the existing Go compiler; no registry package supplies this capability.
- **M2: none.** Pattern typechecking and structured compiler diagnostics must extend `internal/types`; the candidate package is unrelated.
- **M3: none.** Runtime parity fixtures and compiler documentation validate the existing compiler; no package implementation is needed.

## Milestones

### M1: Bind aliases and emit canonical constructor names (~165 LOC)

**Estimate:** 45 implementation + 120 tests. **Duration:** Day 1, 6–8 hours.
**Dependencies:** None.

Update `internal/pipeline/pipeline_module_imports.go`, `internal/pipeline/pipeline_module_phases.go`, `internal/elaborate/core.go`, and `internal/elaborate/patterns.go`. Add canonical identity to imported metadata; bind explicit aliases to canonical factories; register alias keys without losing canonical constructor names. Canonicalize both nullary and argument pattern branches. Preserve auto-imported canonical bindings, first-wins imports, and local declaration precedence.

Add `internal/pipeline/ctor_alias_pattern_test.go` and elaborator cases alongside `internal/elaborate/patterns_nullary_test.go`. Test nullary aliases, aliases with fields, expression construction, arity errors, polymorphic payloads, and aliased imports alongside a same-named local constructor. Audit `convertConstructors`, `buildConstructorFactoryTypes`, interface publication and backend constructor tables: consumers deriving factory identity from a map key must not accidentally create alias factories or overwrite canonical schemes. Extend only those consumers proven to need correction; preserve canonical identity end-to-end.

**Example:** Plan `examples/runnable/constructor_import_alias.ail`, completed in M3, covering both alias arities and a local-name clash.

**Acceptance criteria:**

- [ ] `None as Nada` matches None and constructs None in expressions; `Some as S` matches fields and `S(3)` constructs Some.
- [ ] Elaborated constructor patterns use canonical `None`/`Some` names, and factory references retain canonical identity.
- [ ] Bare non-nullary aliases still fail arity validation; canonical names remain usable.
- [ ] Local declarations and duplicate imports retain existing precedence; a separately aliased imported constructor remains usable beside a local clash.
- [ ] Focused pipeline/elaborator tests and `make test-core` pass.

**Risk:** Binding names leaking into factory or interface identity. Mitigate with Core assertions and collision fixtures before moving to M2.

### M2: Reject unknown patterns with tiered resolution (~220 LOC)

**Estimate:** 90 implementation + 130 tests. **Duration:** Day 2, 6–8 hours.
**Dependencies:** M1.

Update `internal/types/typechecker_patterns.go` and `internal/types/errors.go`. Retain direct-scope checking; consult `diagnosticCtorTypes` on direct misses; apply the existing foreign-ADT error when a transitive constructor provably belongs to another ADT. Preserve the existing unconstrained acceptance for transitive names when the scrutinee ADT is unresolved. Reject names found nowhere with `match_unknown_constructor` / `TC_MATCH_001`, source position and actionable suggestions. Sort/deduplicate suggestion candidates so map iteration cannot change diagnostics.

Extend the new pipeline suite and add targeted structured-error tests in `internal/types/ctor_pattern_scope_test.go`. Include nullary and argument unknowns, nested patterns, transitive Option patterns using only `std/list`, transitive Err against Option, local ADTs, and unresolved-scrutinee compatibility.

**Example files:** Positive transitive case in the M3 runnable example or a companion `examples/runnable/constructor_pattern_transitive.ail`; negative fixtures are isolated temporary test modules, not files that the runnable-example gate must accept.

**Acceptance criteria:**

- [ ] Unknown `Bogus` and `Bogus(v)` fail checking with the structured kind/code, constructor spelling and location.
- [ ] Suggested constructors are deterministic and useful; known transitive constructors are distinguished from typos.
- [ ] Only importing `std/list (nth)` still permits matching its Option result with Some/None (#323).
- [ ] Transitive Err against concrete Option fails with the existing foreign-ADT diagnostic.
- [ ] Existing direct foreign-ADT and nullary-pattern suites remain green; wildcard and lowercase variable patterns retain their behavior.
- [ ] Focused types/pipeline tests and `make test-core` pass.

**Risk:** Overly strict scope checks breaking #323. Mitigate with a named positive transitive test and an unresolved-scrutinee test before adding the rejection branch.

### M3: Verify routes and document the contract (~95 LOC)

**Estimate:** 50 tests + 45 examples/documentation. **Duration:** Day 3, 6–8 hours including buffer.
**Dependencies:** M1, M2.

Add route-parity tests in `internal/pipeline/ctor_alias_pattern_test.go` or existing runner integration suites. Create the runnable examples listed above, register required manifest metadata using the existing example workflow, and update `docs/docs/reference/modules.md` plus `changelogs/v0.32-current.md`. Get `ailang prompt` before writing any .ail fixtures and validate with a freshly built binary.

Split positive alias and negative unknown repros into separate modules: the original combined file will correctly fail whole-module checking because `unknown()` exists, even when `aliased` is selected as entry. Likewise isolate each negative scenario so another error cannot mask it.

Run focused package tests, `make test-core`, `make test`, `make lint`, `make check-boundaries`, `make verify-examples`, and `make verify-examples-toplevel`. Use the existing example gates, recording any pre-existing failures and comparing them against the baseline. Check std/option, std/result, and the documented ADT fixtures. Verify the shared rejection gate also applies to `ailang compile`; no Go-codegen feature expansion is planned.

**Acceptance criteria:**

- [ ] Isolated alias repro returns 1 under interpreter, `--bytecode`, and `--bytecode --strict-bytecode`; argument aliases also agree.
- [ ] Isolated unknown patterns fail `check`, all run routes and `compile` before execution.
- [ ] Unaliased clash patterns/expressions retain the existing foreign-ADT/unification errors on the applicable routes.
- [ ] New runnable examples check and run successfully; existing #323, foreign-ADT and documented ADT fixtures pass.
- [ ] Required test, lint, boundary and example gates pass with no new failures against baseline.
- [ ] Module documentation explains constructor aliases and the direct/transitive/unknown resolution rule; changelog records the corrected behavior.

**Risk:** Bytecode fallback masking parity. Mitigate by including strict bytecode explicitly and using a fresh local build.

## Success Metrics and Execution Handoff

All V1–V9 in-scope behaviors have named regression coverage, including expressions and transitive matching. Preserve V10/V11 errors; V12 qualified patterns and V13 type aliases remain documented non-goals rather than implementation acceptance targets. Tests cover every new resolution branch; measure focused package coverage without inventing an unavailable baseline percentage.

Progress artifact: `.ailang/state/sprints/sprint_M-CTOR-PATTERN-ALIAS-AND-SCOPE.json`. All milestones start uncompleted. Execute M1 → M2 → M3; do not execute the sibling variable-default-arm sprint or add exhaustiveness checking. Issue #323 is historical regression context, not an issue to auto-close; #1480 is the merged design PR, not a bug issue. No current bug issue number was supplied, so `github_issues` stays empty.

Coordinator should route the reviewed plan and JSON to sprint-executor after its approval gate, then sprint-evaluator against this plan and the design. No implementation or executor dispatch is performed during planning. Optional quorum may be reattempted if reviewer routes recover; record absences honestly and do not claim clearance.
