# M-EFFECT-LATENT-FUNCTION-VALUES sprint plan

**Status**: Planned — scheduling authorized; execution awaits sprint-plan approval.
**Date**: 2026-10-08
**Target**: v0.53.0; move to v0.54.0 if the v0.53.x train is closed. Do not ship this acceptance-changing fix as a v0.52.x patch.
**Design**: [approved design](../v0_48_0/m-effect-latent-function-values.md), merged in #1378.
**Tracking**: Refs #1326, Refs #573. Sequencing: Refs #616.
**Duration**: 5 working days (approximately 30 hours, including 25% contingency).
**Estimated LOC**: 750 total: 270 implementation, 430 tests, 50 example/documentation.
**Risk**: High — shared type inference and effect validation; over-rejection is the dominant risk.

## Goal and current evidence

Reject pure callers that perform latent effects through HOF arguments and rowless callers that invoke effectful record fields. Preserve storage-only callbacks, polymorphic combinators, and binding shadowing. Implement design Phases 0–2 and 4 with L1-open; no runtime or stdlib signature changes.

Mark approved scheduling on 2026-10-08. P0 triage verified both repros live on origin/dev 658ff76a3: check accepts them and execution performs IO. This checkout is 62ac2d09, std/VERSION v0.52.5; inspection confirms function annotations still receive anonymous empty-label tails and there is no LatentParamMask/CallEffects publication. No implementation is included here. The approved design's prototype results are historical, not a current validation run.

The velocity script found one documentation commit in this shallow seven-day history and no usable LOC metrics. An empirical LOC/day rate cannot be claimed. Estimate from the approved design's phase breakdown, adding interface plumbing, regression coverage and 25% contingency: 5 days, 150 estimated LOC/day. Re-estimate after M1 if the current implementation differs materially.

## Merge order and ownership

1. **This latent-function-values implementation lands first.** L0 canonicalizes effect-label payloads; L1 preserves concrete annotation labels; L2 charges known latent labels. It does not depend on solving row-variable tails and must leave UnionEffectRows, DiffEffectRows and extractEffectFromType tail semantics unchanged.
2. **#616 lands second**, after rebasing onto this fix. Its design at [M-EFFECT-ROW-VAR-UNIFICATION](../v1_0_0/m-effect-row-var-unification.md) proposes CallEffects[appID] and affects application constraints and tail algebra. Extend a single per-App publication to carry both the latent-parameter mask and resolved call row; retain the mask's pre-instantiation provenance and fail-loud invariant. Do not reconstruct it from instantiated CoreTypeInfo or create two independent authorities.
3. Serialize edits to internal/types, internal/pipeline/validate_effects.go and pipeline compiler plumbing. Before #616 merge, run this sprint's full arm matrix plus #616 AC1–AC12 against the combined tree, especially concrete-row recursive contamination, stored callbacks, and cross-module accept/reject pairs. A green suite alone is insufficient.

#616's historical parked/D-10 notes are not a new authorization for its implementation; its separately routed plan must reconcile those decisions. This sprint neither implements nor closes #616. If #616 merges unexpectedly first, rebase and reassess metadata and row payload assumptions before executing M2; do not mechanically port this plan.

## Milestones and daily tasks

### M1: Binding-aware resolution and canonical effect labels (~150 LOC)

**Day 1**, 6 hours. Dependencies: none. Approximately 50 implementation + 100 test LOC.

Update internal/pipeline/validate_effects.go so declaredEffects resolves top-level bindings, including argument lookup; track lexical bindings through lambdas, lets and patterns. Update internal/types/builder.go to Unit() payloads and internal/types/row_unification.go to skip payload unification only for EffectRow; preserve effect parameter/budget compatibility and record-row payload checks. Add focused tests in those packages and internal/pipeline/effect_latent_function_values_test.go.

- [ ] AC9: shadowed applyStep accepted; direct IO call R0 remains rejected.
- [ ] AC10: DOM callback collision is removed; single_agent_replay keeps its baseline status.
- [ ] Canonical payload and unifier behavior tested independently; record payload mismatches still reject, effect budget/parameter checks remain active.

Example controls: examples/cognitive_os/single_agent_replay.ail; new runnable example deferred to M4. Risk: incomplete lexical traversal or weakening record unification.

### M2: Preserve function-annotation labels (~170 LOC)

**Day 2**, 6 hours. Dependencies: M1. Approximately 40 implementation + 130 test LOC.

Update internal/elaborate/file_funcs.go FuncType conversion using existing effect elaboration helpers: preserve concrete labels, params and budgets, use the written named tail where present, and retain fresh tails for unannotated arrows. Use L1-open for concrete annotations. Test parameter/return/let annotations, named and inline records, aliases, ADT payloads and record updates in elaborate and pipeline tests.

- [ ] AC5–AC6: R3 and all field/parameter/alias/ADT/update variants reject at the offending caller, naming IO; FS-only caller blames rowless rather than main.
- [ ] AC7: stored-never-called hooks, closed-annotated storage-only callback and all package-ceiling controls stay accepted.
- [ ] Missing-effect rejection requires effect-check diagnostic and correct function, not an unrelated parse/type error.

Example controls: existing DOM replay and effectful_list_t1_mapE_basic.ail. Risk: named tails must retain source identity without adding #616 tail-solving semantics.

### M3: Publish pre-instantiation masks and charge latent arguments (~330 LOC)

**Days 3–4**, 12 hours. Dependencies: M1, M2. Approximately 180 implementation + 150 test LOC.

Update internal/types/typechecker_functions.go, typechecker_literals.go and the CoreTypeChecker metadata owner (typechecker_core.go as needed) to publish the callee scheme's callback openness per successfully typed App. Handle local/global/lambda/field/computed callee paths explicitly. Thread the lookup through internal/pipeline/pipeline_single.go and pipeline_module_compile.go to ValidateEffects. At open callback positions, charge concrete latent labels using binding-aware same-module declarations or value types. Keep existing imported/lambda handling and ghost-effect erasure.

- [ ] AC1–AC4: local HOF, sortBy and all six leaky std HOFs, cross-module recursive/forwarding/std-forwarding shapes and wrong FS row reject naming IO at the pure caller.
- [ ] AC7–AC8: storage-only callback controls, inferred-polymorphic combinator, pure-importer genuine-effect rejection and mapE example preserve their expected outcomes.
- [ ] Missing publication for a successfully typed function App raises an internal invariant error; a false mask and absent entry are distinct. Tests cover pre-instantiation openness surviving concrete instantiation.
- [ ] Same-module concrete-row recursive calls retain contamination-safe declared rows; no tail union/diff changes.

Example: HOF accept arm to be added in M4. Risk: scheme provenance can be lost before inferApp; record it at lookup rather than infer openness from the instantiated type.

### M4: End-to-end evidence, runnable example and migration docs (~100 LOC)

**Day 5**, 6 hours. Dependencies: M1, M2, M3. Approximately 50 tests + 50 example/documentation LOC.

Create examples/runnable/effect_latent_function_values.ail and its examples/manifest.json entry. Show accepted declared-effect HOF and field invocation and pure storage-only hook construction; negative programs stay in Go tests. Read ailang prompt before writing AILANG. Update CHANGELOG.md with Breaking — soundness and signature migration, docs/LIMITATIONS.md where applicable, and the active teaching prompt through the repository prompt workflow. Correct the misleading mechanism comment in internal/pipeline/effect_pure_row_overgeneralization_test.go.

- [ ] AC11: tree-walk and bytecode run reject R1/R3 before execution; no side-effect marker appears.
- [ ] AC12: compare fresh base/fixed example pass/fail sets and std import-probe sets; no unintended status changes. Recount current corpus, rather than asserting historical 425/49 counts.
- [ ] New example checks and runs with required IO capability and expected output; pure storage-only construction checks without IO.
- [ ] Mutation evidence and package/full repository checks below pass; report coverage and changed outcomes, then run sprint-evaluator after execution.

## Validation and success metrics

Use AILANG_NO_CACHE=1 and fresh temporary module paths for every arm. Rebuild the binary from the implementation checkout and record its exact SHA; do not trust installed version stamps. First bank current expected exit codes/output and the current corpus baseline. Run focused go test ./internal/types ./internal/elaborate ./internal/pipeline ./internal/iface, then make test, make lint, make check-boundaries, and make verify-examples using available repository targets. Compare any pre-existing example failures rather than attributing them to this fix. No benchmark run is required for this compile-time correction.

All design AC1–AC12 are mandatory. AC13 applies only to the deferred closed-row phase. Each modified rule needs a direct positive and negative control; collect changed-package coverage without inventing a repository baseline or claiming historical prototype coverage. Revert each component temporarily and restore it after recording results: L2 removal makes AC1–AC4 red while annotation cases stay green; L1 removal makes AC5–AC6 red; removing the open-parameter gate makes storage controls red; removing D4 makes shadowing red. For L0, revert canonicalization and unifier protection together for the DOM integration mutation, and test each half independently because either can mask the other's removal.

## Registry reuse

Ran ailang pkg search effects on 2026-10-08: returned sunholo/billing_entitlements and world/core, neither implements compiler inference or effect validation. No package-like runtime capability is being introduced. M1–M4 each use action none: changes belong to existing compiler internals, integration tests and repository documentation; registry packages cannot replace this machinery. Persist this decision per milestone in JSON.

## Scope decisions and handoff

Plan recommendation is to ship L1-open in the next minor release with no opt-out. The approved design still has unchecked human freezes for L1-open versus holding for closed rows, and the breaking-change posture. Sprint-plan approval must explicitly ratify this release scope before implementation; scheduling approval alone is not recorded as that ruling. Closed rows/open-on-use and V13 width enforcement are excluded from this 750-LOC estimate. Record V13 as a limitation linked to the existing issues; do not create another GitHub issue in this task.

The planning PR body must contain Refs #1326, Refs #573 and Refs #616, with no closing keywords. Only the implementation PR, after AC1–AC12 pass, should contain Closes #1326 and Closes #573; retain Refs #616. Do not close #616 here. No GitHub issue is created. Plan/JSON remain not_started for the coordinator's approval path; merging the approved planning PR is the coordinator handoff, or an attended user says execute sprint. Do not dispatch execution before that gate.
