# Sprint Plan: M-IFC-AUTHORITY-SCOPING

**Status:** Planned; scheduling approved by Mark on 2026-10-08; execution awaits plan approval.
**Target:** v0.53.x (v0.54.0 if the security review misses the v0.53.x window).
**Design:** [m-ifc-authority-scoping.md](../m-ifc-authority-scoping.md), merged 05a6457d1.
**Issue links:** Refs #752; Refs #1134 (sequencing dependency only).
**Duration:** 4 engineering days, approximately 24 focused hours including review buffer.
**Risk:** High — security-sensitive effect validation and IFC semantics.
**Estimate:** 720 LOC: 300 implementation, 360 tests, 60 examples/documentation.

## Goal and scope

Narrow opt-in Declassify authority to declared label sets and enforce positive parameter labels on local calls. Preserve bare Declassify's existing whole-body authority and conservative closure propagation. Reject previously accepted cross-label argument flows intentionally. No compiler implementation belongs in this planning change; no new GitHub issue is needed.

Mark's P0 triage reproduced the defect on origin/dev 658ff76a3 on 2026-10-08: a value labelled arg reaches a sqlsafe-labelled parameter without rejection, and bare Declassify still grants whole-body authority. This is supplied maintainer evidence, not a new reproduction by this planning session. The local snapshot is v0.52.5 and still has ifcSig.declassify as a bool, Check B bypass for that bool, no Check C, and invariant effectParamsCompatible. Declared record-label tracking has already landed: preserve ifc_static_type.go's deepLabel contributions and projection precision.

The design's systemic analysis covers both validation and IFC authority paths, joins, closures, parameter flows, duplicate effect atoms and legacy sink_check.go. It remains applicable. Cross-module metadata, pipeline/REPL wiring, runtime tracing (#1132), label polymorphism and blanket-authority lint are outside this sprint.

## Release and sequencing contract

Land and evaluate this sprint before implementing m-ifc-cross-module-labels (#1134). The separately dispatched companion plan can be prepared concurrently, but compiler edits in the shared IFC checker must be serialized. Rebase its implementation onto the authority-scoping implementation commit; run this sprint's complete regression suite again after imported-call wiring.

The companion must replace its stale declassify: bool example with a deterministic label list: [] means no authority, ["*"] means bare/all authority, and a sorted unique list means scoped authority. Its imported signatures must retain positive parameter labels and feed the same Check A/Check C logic and result-label policy as local signatures. Reserve wildcard for summary encoding; do not interpret a user label as wildcard accidentally. Do not build a second checker or ship a bool-based IFC cache schema as an interim representation. This sprint owns local semantics and reusable authority/check helpers; the companion owns serialization, digest/cache invalidation, import resolution and CLI/REPL integration.

Planning PR title/body and milestone commits use Refs #752 and Refs #1134. Only the implementation PR uses the closing keyword for #752; #1134 remains open until its own companion implementation PR. Do not close the runtime tracing issue as a side effect.

## Design clarification required before execution

The source design has an extra closing bracket in one syntax example and says a label key accepts a comma-separated label list. Actual parseEffectParams accepts only identifier key=value pairs and rejects duplicate keys; a comma introduces a new key. No parser grammar extension was approved.

Recommended resolution: use repeated existing scoped effect atoms to express multiple labels, normalize their label values into a sorted unique set, and make a bare atom dominate regardless of order. Both IFC signature construction and effect-row elaboration must use the same normalization, avoiding today's last-write-wins parameter map. Single-label syntax remains the existing Declassify[label=email] form. Before coding, record maintainer approval of this interpretation in the design/plan; if comma-list syntax is required instead, revise the parser design and estimate through the design gate. Do not silently invent syntax or restrict the approved set semantics to one label. Empty values, unknown keys, malformed atoms and reserved wildcard inputs must fail loudly.

Effect validation subsumption is directional: SubsumeEffectRows(requiredBody, declaredAuthority) requires the declaration to cover the body's required set. Function-value effect rows remain invariant in Unifier.unifyRows; do not weaken them by globally replacing symmetric compatibility with a covering test. Audit DiffEffectRows, all effectParamsCompatible callers, row merging and diagnostic paths before choosing the helper boundary.

## Current status and velocity

Ran sprint-planner/scripts/analyze_velocity.sh 7 and read CHANGELOG.md plus changelogs/v0.32-current.md. The latter records v0.52.5 fixes and v0.52.4 additions on 2026-10-07, but supplies no reliable comparable IFC LOC/day measurements. This checkout is shallow with one visible documentation commit, no HEAD~1 and no usable seven-day implementation delta. The script's historical LOC grep and N/A diff are not a measured daily velocity.

Use the design's three-day baseline plus one day (33%) for normalization, effect-direction audit and security review. Planning capacity is 180 LOC/day, an assumption rather than observed throughput. Re-estimate after M1 if the approved representation needs broader row changes. No coverage build was run for a docs-only plan; record baseline package coverage during execution rather than claiming an unmeasured percentage.

## Registry reuse audit

Ran ailang pkg search ifc on 2026-10-08; the sole candidate was sunholo/linkedin@0.5.1. Inspected pkg info and attempted pkg docs: it has no AGENT.md. It is an IFC consumer (LinkedIn client), not a compiler checker or effect schema implementation. Every milestone therefore records action none: these changes must live in compiler internals, tests and compiler examples. No package dependency or package contribution is appropriate. No registry notification was sent.

## Milestones and daily tasks

### M1: Scoped effect representation and validation (~240 LOC)

**Day 1; 6 hours.** 110 implementation + 130 tests. Dependencies: approved syntax/normalization clarification above.

Audit effects.go's schema, validation, defaults, elaboration, row merge/formatting and compatibility call sites. Add only the Declassify.label open-vocabulary exception. Normalize repeated atoms consistently; separate directional authority coverage from invariant function-value equality. Table-test bare/scoped/none, subset/superset/incomparable scopes, malformed values, duplicates and order independence. Check inference does not erase scoped requirements when joining callee effects.

**Files:** internal/types/effects.go, effects_test.go and existing effect schema/row tests; internal/types/row_unification.go only if the audit identifies a required invariant-preserving normalization adjustment. Inspect internal/parser/parser_effect.go and parser_effect_params_test.go; no new grammar planned.
**Examples:** no new example in M1; M4 supplies scoped examples after validation and checking agree.

- [ ] Scoped single-label atoms validate; unknown keys and malformed/empty values fail loudly.
- [ ] Approved multi-label representation normalizes without last-write-wins loss; mixed bare/scoped atoms yield all authority in either order.
- [ ] Declared superset covers required subset; scoped declaration cannot cover bare or an unrelated scope; a missing effect still fails.
- [ ] Rand/AI defaults and exact parameter invariance, budgets, row variables and function-value effect invariance retain existing behavior.

**Risk:** weakening shared effect comparisons. Mitigate with a separate directional helper and regression tests for every affected caller.

### M2: Check B with scoped authority (~220 LOC)

**Day 2; 6 hours.** 100 implementation + 120 tests. Dependencies: M1.

Replace the boolean projection with explicit none/scoped/all authority. Always evaluate return-label coverage; permit only leaked constituent labels authorized by the declaration. Preserve declared-result semantics for authorized declassifiers and conservative transparent/closure flows. Extend DeclassifyRequiredError diagnostics with stable sorted scopes. Audit legacy CheckDeclassify and its tests: remove or adapt the redundant helper and record the choice, with no divergent bool-only enforcement path.

**Files:** internal/types/ifc_check.go, ifc_check_test.go, ifc_closure_test.go, sink_check.go and sink_check_test.go as required by the audit.
**Examples:** M4 adds a passing scoped declassifier and tests retain rejected variants.

- [ ] Authorized relabel passes; unrelated secret and mixed joins containing an unauthorized constituent fail and name the label and scope.
- [ ] A caller declaring only its callee's scope cannot relabel unrelated secret data; widening must be explicit.
- [ ] Scoped authority cannot launder a secret-carrying closure; no-authority rejection and bare-authority legacy acceptance remain intact.
- [ ] Existing declared record/alias/ADT deep-label and projection tests pass; legacy helper has one documented disposition.

**Risk:** return-label over-approximation or loss of nested labels. Mitigate with nested-field, join and closure controls, retaining surface-AST IFC and HM behavior.

### M3: Positive parameter call-site coverage (~160 LOC)

**Day 3; 5 hours.** 70 implementation + 90 tests. Dependencies: M2.

Add Check C to the local resolved-callee path, using each already-computed argument label and LabelSubsumes constituent coverage. Add ParamLabelCoverError in errors.go with argument label, parameter name/type, source position and explicit remedy. Reuse shared checking structure so #1134 can supply imported signatures later. Suppress diagnostic emission during silent intrinsic-label walks consistently with Check A.

**Files:** internal/types/ifc_check.go, errors.go, ifc_check_test.go and existing static-type IFC tests.
**Examples:** M4 adds a passing positive-label call; failing arg-to-sqlsafe programs remain test fixtures.

- [ ] Maintainer arg-to-sqlsafe repro and design secret-to-email repro fail with ParamLabelCoverError at the call edge.
- [ ] Bottom/unlabelled, literals and matching-label arguments pass; joins with any uncovered constituent fail.
- [ ] Check A still enforces negative refinements, Check B still rejects unauthorized return relabels, and transparent result-label joining remains unchanged.
- [ ] Argument labels include declared nested types and closure bodies; intrinsic-label walks do not duplicate diagnostics.

**Risk:** Check C falsely implies positive labels are proof of sanitization. Document that bottom passes and these are lattice coverage constraints, not provenance guarantees.

### M4: Examples, documentation and integration verification (~100 LOC)

**Day 4; 7 hours including review buffer.** 20 implementation cleanup + 20 integration tests + 60 examples/docs. Dependencies: M3.

Obtain the current ailang prompt before writing any .ail files. Add examples/runnable/secrets/scoped_declassify.ail and positive_label_call.ail; type-check using a freshly built binary and run positive examples with their declared capabilities. Keep negative cases in test fixtures. Update examples/runnable/secrets/README.md, docs/docs/guides/ifc-labels.mdx and the current changelog. Inspect teaching-prompt claims; if edits are needed, route through prompt-manager rather than expanding this sprint silently. Record the scoped-authority handoff contract for #1134 and the landed commit after approval/merge.

**Files:** the two named new examples, README.md, ifc-labels.mdx, changelogs/v0.32-current.md, existing IFC integration test harnesses.

- [ ] Both new examples check and run as documented; gated_secret and secret_demo retain clean verdicts and leak_attempt retains its expected single SinkRefinementError.
- [ ] Design V1–V8 mechanisms have regression coverage, including intentional Check C rejection, scoped propagation and bare compatibility.
- [ ] Focused Go tests for internal/types and internal/parser pass; make test, make lint and make check-boundaries pass using the rebased implementation tree.
- [ ] Documentation explains opt-in scopes, bottom acceptance, blanket-authority hazard and deliberate rejection of cross-label calls; no runtime IFC or cross-module completion is claimed.

**Risk:** stale binary masks results. Build before CLI verification and record exact tested revision and command results.

## Success and handoff

All acceptance criteria above are required. Track internal/types coverage before/after; do not reduce IFC branch coverage, and exercise new authorization and coverage branches explicitly rather than relying only on a global percentage. M4 integration review checks no TLabelled changes reach CoreTI/codegen and no module layer boundary changes are introduced.

JSON progress is .ailang/state/sprints/sprint_M-IFC-AUTHORITY-SCOPING.json; all milestone passes remain null. Scheduling approval does not itself authorize execution. Submit plan and populated JSON for review; after plan approval and explicit execution authorization, hand off to sprint-executor then sprint-evaluator. The coordinator's plan-PR merge workflow may supply that authorization; do not dispatch code work before it. No separate outbound message is necessary while the coordinator consumes the artifact markers.
