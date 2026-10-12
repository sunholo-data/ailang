# Sprint Plan: M-IFC-AUTHORITY-SCOPING

**Status:** Implemented; scheduling approved by Mark on 2026-10-08; Declassify syntax ruled by Mark on 2026-10-08 (single label, see below). Merging this plan dispatches the executor; there is no further human gate before coding.
**Target:** v0.53.x (v0.54.0 if the security review misses the v0.53.x window).
**Design:** [m-ifc-authority-scoping.md](m-ifc-authority-scoping.md), merged 05a6457d1.
**Issue links:** Refs #752; Refs #1134 (sequencing dependency only).
**Duration:** 4 engineering days, approximately 23 focused hours including review buffer.
**Risk:** High — security-sensitive effect validation and IFC semantics.
**Estimate:** 660 LOC: 270 implementation, 330 tests, 60 examples/documentation.

## Goal and scope

Narrow opt-in Declassify authority to a single declared label (`Declassify[label=email]`) and enforce positive parameter labels on local calls. Preserve bare Declassify's existing whole-body authority and conservative closure propagation. Reject previously accepted cross-label argument flows intentionally. No compiler implementation belongs in this planning change; no new GitHub issue is needed.

Mark's P0 triage reproduced the defect on origin/dev 658ff76a3 on 2026-10-08: a value labelled arg reaches a sqlsafe-labelled parameter without rejection, and bare Declassify still grants whole-body authority. This is supplied maintainer evidence, not a new reproduction by this planning session. The local snapshot is v0.52.5 and still has ifcSig.declassify as a bool, Check B bypass for that bool, no Check C, and invariant effectParamsCompatible. Declared record-label tracking has already landed: preserve ifc_static_type.go's deepLabel contributions and projection precision.

The design's systemic analysis covers both validation and IFC authority paths, joins, closures, parameter flows and legacy sink_check.go. It remains applicable. Cross-module metadata, pipeline/REPL wiring, runtime tracing (#1132), label polymorphism, multi-label Declassify and blanket-authority lint are outside this sprint.

## Release and sequencing contract

Land and evaluate this sprint before implementing m-ifc-cross-module-labels (#1134). The companion plan (sprint id M-IFC-CROSS-MODULE) can be prepared concurrently, but compiler edits in the shared IFC checker must be serialized. Rebase its implementation onto the authority-scoping implementation commit; run this sprint's complete regression suite again after imported-call wiring.

The companion must replace its stale declassify: bool example with a deterministic label list: [] means no authority, ["*"] means bare/all authority, and a sorted unique list means scoped authority (with the v1 single-label ruling, a scoped Declassify serializes as a one-element list). Its imported signatures must retain positive parameter labels and feed the same Check A/Check C logic and result-label policy as local signatures. Reserve wildcard for summary encoding; do not interpret a user label as wildcard accidentally. Do not build a second checker or ship a bool-based IFC cache schema as an interim representation. This sprint owns local semantics and reusable authority/check helpers; the companion owns serialization, digest/cache invalidation, import resolution and CLI/REPL integration.

Planning PR title/body and milestone commits use Refs #752 and Refs #1134. Only the implementation PR uses the closing keyword for #752; #1134 remains open until its own companion implementation PR. Do not close the runtime tracing issue as a side effect.

## Maintainer ruling: single-label Declassify in v1 (Mark, 2026-10-08)

Declassify takes a **single** label in v1: `Declassify[label=email]`. There is no parser change and no repeated atoms. The parser keeps rejecting duplicate effect names (`internal/parser/parser_effect.go:91-95`, `PAR_EFF001_DUP`), so `! {Declassify[label=a], Declassify[label=b]}` remains a parse error, and that is intended. A function that must relabel more than one label declares bare `! {Declassify}` in v1. Multi-label scoped authority is future work and needs its own design. The ruling is recorded as a dated note in the design doc, which also fixes its `Declassify[label=email]]` typo and its "comma-separated label list" wording.

parseEffectParams already accepts `label=email` as an identifier key=value pair and rejects duplicate keys, so the existing grammar suffices. Internally the authorized labels stay a set (⊤ or exactly one label in v1) so the covering rule and the companion's label-list summary need no reshaping when multi-label lands. Empty values, unknown keys (for example `Declassify[mode=x]`), malformed atoms and a user-supplied reserved wildcard (`label=*` or equivalent) must fail loudly.

Effect validation subsumption is directional: SubsumeEffectRows(requiredBody, declaredAuthority) requires the declaration to cover the body's required set. Function-value effect rows remain invariant in Unifier.unifyRows; do not weaken them by globally replacing symmetric compatibility with a covering test. Audit DiffEffectRows, all effectParamsCompatible callers, row merging and diagnostic paths before choosing the helper boundary.

## Current status and velocity

Ran sprint-planner/scripts/analyze_velocity.sh 7 and read CHANGELOG.md plus changelogs/v0.32-current.md. The latter records v0.52.5 fixes and v0.52.4 additions on 2026-10-07, but supplies no reliable comparable IFC LOC/day measurements. This checkout is shallow with one visible documentation commit, no HEAD~1 and no usable seven-day implementation delta. The script's historical LOC grep and N/A diff are not a measured daily velocity.

Use the design's three-day baseline plus one day (33%) for the effect-direction audit, compatibility audit and security review. Planning capacity is 180 LOC/day, an assumption rather than observed throughput. M1 was re-estimated down from 240 to 180 LOC after the single-label ruling removed repeated-atom normalization; re-estimate after M1 if the directional-coverage audit needs broader row changes. No coverage build was run for a docs-only plan; record baseline package coverage during execution rather than claiming an unmeasured percentage.

## Registry reuse audit

Ran ailang pkg search ifc on 2026-10-08; the sole candidate was sunholo/linkedin@0.5.1. Inspected pkg info and attempted pkg docs: it has no AGENT.md. It is an IFC consumer (LinkedIn client), not a compiler checker or effect schema implementation. Every milestone therefore records action none: these changes must live in compiler internals, tests and compiler examples. No package dependency or package contribution is appropriate. No registry notification was sent.

## Milestones and daily tasks

### M1: Scoped effect representation and validation (~180 LOC)

**Day 1; 5 hours.** 80 implementation + 100 tests. Dependencies: none (syntax settled by the 2026-10-08 ruling above).

Audit effects.go's schema, validation, defaults, elaboration, row merge/formatting and compatibility call sites. Add only the Declassify.label open-vocabulary exception (single label per atom). Separate directional authority coverage from invariant function-value equality. Table-test bare/scoped/none, same/different single-label scopes, bare covering scoped, malformed and empty values, reserved wildcard input, and that a duplicated Declassify atom is still rejected by the parser with PAR_EFF001_DUP. Check inference does not erase scoped requirements when joining callee effects.

**Files:** internal/types/effects.go, effects_test.go and existing effect schema/row tests; internal/types/row_unification.go only if the audit identifies a required invariant-preserving normalization adjustment. Inspect internal/parser/parser_effect.go and parser_effect_params_test.go; no parser change (add at most a regression test pinning PAR_EFF001_DUP for duplicate Declassify atoms).
**Examples:** no new example in M1; M4 supplies scoped examples after validation and checking agree.

- [x] Scoped single-label atoms validate; unknown keys and malformed/empty values fail loudly.
- [x] Duplicate Declassify atoms (scoped or mixed bare/scoped) still fail with PAR_EFF001_DUP; no parser grammar change; a reserved wildcard label is rejected.
- [x] Bare covers any scope and a scope covers itself; a scoped declaration cannot cover bare or a different label; a missing effect still fails.
- [x] Rand/AI defaults and exact parameter invariance, budgets, row variables and function-value effect invariance retain existing behavior.

**Risk:** weakening shared effect comparisons. Mitigate with a separate directional helper and regression tests for every affected caller.

### M2: Check B with scoped authority (~220 LOC)

**Day 2; 6 hours.** 100 implementation + 120 tests. Dependencies: M1.

Replace the boolean projection with explicit none/scoped/all authority. Always evaluate return-label coverage; permit only leaked constituent labels authorized by the declaration. Preserve declared-result semantics for authorized declassifiers and conservative transparent/closure flows. Extend DeclassifyRequiredError diagnostics with stable sorted scopes. Audit legacy CheckDeclassify and its tests: remove or adapt the redundant helper and record the choice, with no divergent bool-only enforcement path.

**Files:** internal/types/ifc_check.go, ifc_check_test.go, ifc_closure_test.go, sink_check.go and sink_check_test.go as required by the audit.
**Examples:** M4 adds a passing scoped declassifier and tests retain rejected variants.

- [x] Authorized relabel passes; unrelated secret and mixed joins containing an unauthorized constituent fail and name the label and scope.
- [x] A caller declaring only its callee's scope cannot relabel unrelated secret data; widening must be explicit.
- [x] Scoped authority cannot launder a secret-carrying closure; no-authority rejection and bare-authority legacy acceptance remain intact.
- [x] Existing declared record/alias/ADT deep-label and projection tests pass; legacy helper has one documented disposition.

**Risk:** return-label over-approximation or loss of nested labels. Mitigate with nested-field, join and closure controls, retaining surface-AST IFC and HM behavior.

### M3: Positive parameter call-site coverage (~160 LOC)

**Day 3; 5 hours.** 70 implementation + 90 tests. Dependencies: M2.

Add Check C to the local resolved-callee path, using each already-computed argument label and LabelSubsumes constituent coverage. Add ParamLabelCoverError in errors.go with argument label, parameter name/type, source position and explicit remedy. Reuse shared checking structure so #1134 can supply imported signatures later. Suppress diagnostic emission during silent intrinsic-label walks consistently with Check A.

**Files:** internal/types/ifc_check.go, errors.go, ifc_check_test.go and existing static-type IFC tests.
**Examples:** M4 adds a passing positive-label call; failing arg-to-sqlsafe programs remain test fixtures.

- [x] Maintainer arg-to-sqlsafe repro and design secret-to-email repro fail with ParamLabelCoverError at the call edge.
- [x] Bottom/unlabelled, literals and matching-label arguments pass; joins with any uncovered constituent fail.
- [x] Check A still enforces negative refinements, Check B still rejects unauthorized return relabels, and transparent result-label joining remains unchanged.
- [x] Argument labels include declared nested types and closure bodies; intrinsic-label walks do not duplicate diagnostics.

**Risk:** Check C falsely implies positive labels are proof of sanitization. Document that bottom passes and these are lattice coverage constraints, not provenance guarantees.

### M4: Examples, documentation and integration verification (~100 LOC)

**Day 4; 7 hours including review buffer.** 20 implementation cleanup + 20 integration tests + 60 examples/docs. Dependencies: M3.

Obtain the current ailang prompt before writing any .ail files. Add examples/runnable/secrets/scoped_declassify.ail and positive_label_call.ail; type-check using a freshly built binary and run positive examples with their declared capabilities. Keep negative cases in test fixtures. Update examples/runnable/secrets/README.md, docs/docs/guides/ifc-labels.mdx and the current changelog. Inspect teaching-prompt claims; if edits are needed, note them in the sprint JSON notes as follow-up rather than expanding this sprint silently. Record the scoped-authority handoff contract for #1134 in the sprint JSON notes. Run the compatibility audit below and record each actual verdict next to the expected one.

**Files:** the two named new examples, README.md, ifc-labels.mdx, changelogs/v0.32-current.md, existing IFC integration test harnesses.

- [x] Both new examples check and run as documented; gated_secret and secret_demo retain clean verdicts and leak_attempt retains its expected single SinkRefinementError.
- [x] Design V1–V8 mechanisms have regression coverage, including intentional Check C rejection, scoped propagation and bare compatibility.
- [x] Compatibility audit verdicts below match expectations (any deviation is explained in notes, not silently accepted).
- [x] `go test ./internal/types/... ./internal/parser/...` and `make test-core` pass, plus make lint and make check-boundaries. Do not run the full `make test` locally (it has crashed executors with SIGBUS in RAM-backed /tmp); CI runs the full suite on the PR.
- [x] Documentation explains opt-in scopes, bottom acceptance, blanket-authority hazard and deliberate rejection of cross-label calls; no runtime IFC or cross-module completion is claimed.

**Risk:** stale binary masks results. Build before CLI verification and record exact tested revision and command results.

### Compatibility audit (M4, expected verdicts)

Run with a freshly built binary. "Before" is origin/dev; "after" is this sprint. None of these files uses scoped Declassify, and every positive-labelled parameter they call receives either a matching label or an unlabelled literal (⊥), so Check C should not fire on any of them.

| File | Command | Expected before | Expected after | Why |
|------|---------|-----------------|----------------|-----|
| examples/runnable/contracts/inbox_injection_v2.ail | `ailang verify` | 5 functions: 3 verified, 2 violations (injectedForward, attemptLaunder) | unchanged | sanitizeBody keeps bare Declassify (whole-body, legacy semantics); its `<email>` param receives `<email>` args; main passes a literal (⊥) |
| examples/runnable/contracts/inbox_v2_app.ail | `ailang verify` | 5 functions: 3 verified, 2 violations | unchanged | sanitizeBody is local; `m.body` is `<email>` from the imported Mail type and covers the `<email>` param; imported-callee Check C is the companion's scope |
| benchmarks/prompt_injection/expected_ailang_safe.ail | `ailang verify` | 3 verified, 0 violations | unchanged | bare Declassify; `<email>` arg into `<email>` param |
| benchmarks/prompt_injection/expected_ailang_injected.ail | `ailang verify` | 2 verified, 1 violation (injectedForward) | unchanged | the violation is the Z3 ensures failure; main's literal arg is ⊥ |
| std/secret.ail | `ailang check` | clean | unchanged | `secret` has unlabelled params and no Declassify; its `<secret>` result is unaffected. Importers (examples/runnable/secrets: gated_secret, secret_demo clean; leak_attempt one SinkRefinementError) keep their verdicts per the first M4 criterion |

If any verdict changes, stop and record it: either the change is the intended #752 rejection (then update the file's header comment and the guide in the same milestone) or it is a regression to fix.

## Success and handoff

All acceptance criteria above are required. Track internal/types coverage before/after; do not reduce IFC branch coverage, and exercise new authorization and coverage branches explicitly rather than relying only on a global percentage. M4 integration review checks no TLabelled changes reach CoreTI/codegen and no module layer boundary changes are introduced.

JSON progress is .ailang/state/sprints/sprint_M-IFC-AUTHORITY-SCOPING.json; all milestone passes remain null. Merging this plan PR is the execution authorization: the coordinator dispatches sprint-executor on merge, followed by sprint-evaluator. There is no separate pre-coding approval step. The executor works on its own branch and opens a PR; it does not push to dev or merge, and the landed commit is recorded by the coordinator, not by the executor.

## Execution outcome — 2026-10-08

All four milestones completed on `coordinator/task-8ffaca1d`. Types/parser tests,
core tests, lint (zero issues), formatting, architecture boundaries, file sizes,
and the example gate passed. Examples: 234 passed, nine existing skips, no
failures; 213 manifest modules checked with no drift. Types coverage increased
from 52.7% to 52.9%. The full suite remains assigned to CI per this plan.

Compatibility audit: both inbox programs retain three verified/two violations;
expected_ailang_safe retains three verified/zero violations; injected retains
two verified/one violation. Each also reports main without contracts. std/secret,
gated_secret and secret_demo check clean; leak_attempt has one sink error.
Both new examples check and run successfully.

Independent review round 1 exposed scoped-call laundering through unlabelled
formals. Regression failed before the fix; scoped results now preserve all
unauthorized full argument constituents before type hand-off, including nested
labels and closures. The design records this necessary adjustment. Scoped
unlabelled returns are checked too, and lexical bindings override module
function signatures. Removed the unused bool-only CheckDeclassify helper.

Fresh-container checks required temporary jq/make, Zig for SQLite CGO, Z3 and
lint tooling. Memory caps and reduced concurrency resolved build/lint OOM.
No tooling or resource-limit changes are shipped in the repository.

The companion #1134 must rebase onto this implementation and reuse authority
none=[], bare=["*"], scoped=[single label] plus the full-actual result policy.
Teaching-prompt additions remain a recorded follow-up. No runtime IFC or
cross-module completion is claimed. GitHub authentication is unavailable;
PR text is staged in .ailang/state/sprints/M-IFC-AUTHORITY-SCOPING-pr.md.
