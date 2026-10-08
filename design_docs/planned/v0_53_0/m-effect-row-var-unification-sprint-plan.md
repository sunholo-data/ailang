# Sprint Plan: M-EFFECT-ROW-VAR-UNIFICATION

**Status:** Planned — scheduling approved; execution awaits plan approval
**Date:** 2026-10-08
**Target:** v0.53.x; v0.54.0 if the v0.53 release train has closed before integration
**Priority:** P0 — static effect soundness and actionable diagnostics
**Design:** [M-EFFECT-ROW-VAR-UNIFICATION](../v1_0_0/m-effect-row-var-unification.md)
**Issues:** Refs #616. Coordination: Refs #1326, Refs #573.
**Approval provenance:** Mark approved scheduling on 2026-10-08 following P0 issue triage.
**Planner-Lane:** opus-required, inherited from design; retain this routing requirement in coordinator dispatch/review. This artifact does not claim an Opus review occurred.
**Duration:** 5 working days + 1 day contingency (6 days reserved, approximately 36 focused hours)
**Risk:** High — shared row algebra, inference publication, validator compatibility

## Goal and scope

Restore the static promise that pure signatures perform no effects. Same-module row-polymorphic
helpers must accept pure callbacks, reject undeclared IO callbacks, and preserve effects across
repeated calls. An effect-check failure must explain the missing label, parameter mismatch, or
undischarged tail rather than present an identical suggested signature.

This is a compiler fix for existing syntax. It preserves the runtime capability backstop and
shipped effect-polymorphic APIs. No new GitHub issue is needed. The plan PR uses `Refs #616`
(and the coordination references above); only the implementation PR uses `Closes #616`.
Do not close #1326/#573 from this sprint: they belong to the separately scheduled latent-values work.

## Current status and evidence

- The scheduling request supplies a live reproduction on `origin/dev` **658ff76a3**:
  blank Suggested fix and a pure `runTwice(noisy)` passing check and printing twice.
  Treat this as maintainer/triage evidence, not a fresh reproduction performed by this planner.
- Planning checkout: **62ac2d09**, `std/VERSION` **v0.52.5**, clean before planning.
  The older design measurements at af6d56144/817bb0274 must be refreshed on the execution base.
- `internal/types/effects.go`: `UnionEffectRows` still builds a label-only union, returns nil
  for empty labels and emits `Tail: nil`. `SubsumeEffectRows` checks missing labels/params only.
- `internal/pipeline/validate_effects.go`: same-module App selection still prefers declared
  rows; `extractEffectFromType` is not a row-discharge contract. No `CallEffects` interface exists.
- Current Core inference is `CoreTypeChecker.inferApp` in
  `internal/types/typechecker_functions.go`, not just the surface `InferenceContext.Infer`.
  It already locally solves subtree equalities, filters enclosing-scope bindings, intentionally
  delays callee-row closure to protect recursive multi-effect functions, and asserts a
  single-tail App invariant (#386). Reuse and test that machinery; do not restore eager closure.
- Production validator entry points are `pipeline_single.go` and `pipeline_module_compile.go`.
  Existing test helpers also invoke `ValidateEffects` directly and need deliberate migration.
- The design's 2026-09-08 correction is incorporated: V14/V15 prove behavior only for
  **declared** cross-module row-polymorphic callees. Inferred effect-transparent combinators
  and the #1091 pure-row over-generalization regression need their own controls.
- The design includes systemic analysis: union, subsumption/diff, both union consumers,
  diagnostics, inference/validation boundary, lambda validation, ghost erasure, budgets,
  recursion, and imported interfaces. The first milestone refreshes that census.

### Estimate basis and velocity limits

Ran `.agents/skills/sprint-planner/scripts/analyze_velocity.sh 7`. This is a shallow checkout
with one visible 2026-10-08 documentation commit; parent diff is unavailable. The script's
LOC samples are historical changelog entries, not measured last-week velocity. No defensible
recent LOC/day can be calculated. The current changelog confirms v0.52.5 on 2026-10-07,
but supplies no directly comparable compiler milestone throughput.

Use the design's 4–5 day estimate plus approximately 25% contingency for current #386 machinery
and shared-validator integration: **6 reserved days**, **1,100 estimated changed LOC**
(390 production + 650 tests + 60 examples/docs). Capacity assumption: about 183 changed LOC/day;
this is a scheduling assumption, not an observed productivity claim. Re-estimate after M1 if
resolving the App constraint requires a new representation or general instantiation redesign.
No current coverage percentage is claimed; record changed-package baseline at execution start.

## Coordination and ordering with latent function values

Related design: [M-EFFECT-LATENT-FUNCTION-VALUES](../v0_48_0/m-effect-latent-function-values.md),
Refs #1326, Refs #573, sent separately for planning. No companion sprint plan exists in this checkout.
The bugs have separate acceptance criteria but share `collectRequiredEffects`, App selection,
`extractEffectFromType`, row cloning/union, and the annotation/inference boundary.

**Proposed integration order:**

1. Land the approved latent-function-values implementation first; it preserves annotated
   function effects and closes concrete callback/record-field leaks.
2. Rebase this sprint onto that commit and refresh both repro matrices before shared-file edits.
3. Implement this sprint serially through M1 → M2 → M3 → M4. Parallel exploration/tests can be
   isolated, but do not concurrently edit the validator or shared row algebra across sprints.
4. Run the combined matrix before merging this implementation PR. Record companion base SHA
   and results in sprint JSON notes and PR validation.

This is a shared-file integration dependency, not a claim that #616 logically needs #1326
resolved to be understood. Planning and M1 analysis can proceed while the companion is pending.
If the companion is delayed and Mark changes the order, record the ruling and give the second
sprint the same rebase + joint-test obligation. Do not silently treat a pending sibling as complete.

**Joint done-gate:** passing a function value does not cause this sprint to erase its call-time
latent effects; invoking a row-polymorphic helper with a pure callback is accepted, with an IO
callback under pure/FS declarations is rejected naming IO, and under IO is accepted. Exercise
same-module name, imported name, let alias, inline lambda, parameter call and record-field call.
Include nested callback-return and constructor/record-update surfaces covered by the companion.
Do not paper over the pure case by universally charging every callback's possible effects.

## Registry reuse audit

Ran `ailang pkg search 'effect'` on 2026-10-08: seven application packages (logging,
billing_entitlements, external_backend, email, gmail, decisions, world/core). No candidate
implements Go compiler inference, effect-row algebra, or validation. None warrants pkg info/docs
inspection for this scope. The CLI warned that its binary may be stale; the search is an advisory
reuse inventory, not compiler behavior evidence. These are core compiler responsibilities that
cannot be replaced with a source-language package.

| Milestone | Decision | Package | Reason |
|---|---|---|---|
| M1 | none | none | Compiler constraint analysis and base-red regressions; reuse existing Go inference/test harnesses |
| M2 | none | none | Shared Go row algebra and diagnostic formatting, below package execution |
| M3 | none | none | Core type-checker publication and pipeline contract, inaccessible to AILANG packages |
| M4 | none | none | Compiler integration guards, runnable examples and release documentation |

## Milestones

### M1: Refresh constraint mechanism and combined baseline (~150 LOC)

**Estimate:** 0 production + 150 tests; 6 hours, day 1.
**Dependencies:** None for analysis; merged latent-values base required before finalizing shared-file implementation baseline.
**Files:** `internal/types/effect_call_publication_test.go` (new),
`internal/pipeline/effect_rowvar_discharge_test.go` (new), existing
`typechecker_functions.go`, `typechecker_substitution.go`, `typechecker_core.go`,
`unification_types.go`, `row_unification.go`, `types_v2.go` (read/trace).
**Examples:** inventory `examples/runnable/effectful_list_t1_mapE_basic.ail`; select the new
pure-caller and mixed-callback example paths used in M4. New .ail content requires `ailang prompt` first.

Trace parameter/result row identity through scheme instantiation, App equality, local solve,
whole-program substitution/defaulting and generalization. Distinguish solved empty rows from
unowned metavariables; use Core node IDs rather than names for independent occurrences.
The old doc's inference that parameter/result variables were never shared is a hypothesis to
verify, not permission to duplicate instantiation logic already keyed by variable name.

- [ ] Refresh base-red tests for the pure single call, wrong-row IO, mixed two calls, and intra-helper runTwice on fresh temp paths/content; bank exit codes/messages and base SHA.
- [ ] Pin #386 nested show/println and recursive multi-effect controls, #1091 closed pure exports, and companion pure/IO callbacks before modification.
- [ ] Record the exact constraint/substitution boundary that prevents publication of the solved call effect; separate callee invocation effects from argument-evaluation effects.
- [ ] Define return-only default-empty eligibility and enclosing-owned open-tail behavior with tests. Never default an argument-linked or enclosing-owned tail to empty.
- [ ] Record current caller census for union, subsumption, typed-App outputs and validation entry points; update estimate if a wider architecture is necessary.

**Risk:** old measurements locate the wrong layer. Mitigation: source trace plus per-occurrence
unit assertions and downstream check outcomes. If a new join representation or generalized
signature redesign is required, stop that expansion for a design amendment; keep completed artifacts.

### M2: Preserve tails and make diagnostics actionable (~300 LOC)

**Estimate:** 130 production + 170 tests; 6 hours, day 2.
**Dependencies:** M1.
**Files:** `internal/types/effects.go`, `effect_subsumption.go`, `effects_test.go`,
`effects_budget_test.go`, `internal/pipeline/validate_effects_rows.go`,
`validate_effects.go`, `effect_rowvar_discharge_test.go`.
**Examples:** source regressions mirror runTwice and pure-caller paths; negative programs stay
in Go test fixtures, outside runnable manifest.

Normalize only empty-label **closed** rows to pure. Preserve one tail and identical-tail unions;
surface distinct-tail conflicts loudly rather than drop/pick a tail. Choose an explicit error path
or existing invariant mechanism after auditing callers; production-facing errors must retain
callee/App context where available. Preserve params, budgets and provenance semantics.

- [ ] Same-tail union survives both required-row collection and suggested-row construction (design AC4/AC7); nil/closed-empty normalization is tested in both orders.
- [ ] Distinct-tail conflicts never fabricate purity; parameter conflicts and budget composition retain existing behavior.
- [ ] Diff/subsumption account for undischarged tails, compare canonical substituted identity, and preserve concrete effect checks; a declared e alone never absorbs concrete IO (AC10/AC11).
- [ ] The runTwice helper with a pure outer signature is rejected; adding its proper row declaration accepts valid pure/IO instantiations after M3 (AC4).
- [ ] Diagnostics name the undischarged tail or missing effect, force an invariant error for empty-diff failure, and omit identical Suggested fix text; closed concrete/lambda errors remain actionable (AC10).

**Risk:** union changes influence two production consumers and error formatting. Test both;
do not implement subsumption as function-type compatibility (function effects stay invariant).

### M3: Publish solved per-App invocation effects and consume them (~430 LOC)

**Estimate:** 230 production + 200 tests; 12 hours, days 3–4.
**Dependencies:** M1, M2; companion merged/rebased integration base.
**Files:** `internal/types/typechecker_core.go`, `typechecker_functions.go`,
`typechecker_substitution.go`, `typechecker_effect_row_issue386.go` (only if necessary),
`effect_call_publication_test.go`; `internal/pipeline/pipeline_single.go`,
`pipeline_module_compile.go`, `validate_effects.go`, `effect_soundness_test.go`,
`validate_effects_test.go`, `effect_rowvar_discharge_test.go`.
**Examples:** exercise new pure-caller and mixed-callback shapes through single-file and imported module fixtures.

Capture App invocation rows at the inference boundary that owns the determining constraint;
finalize `CallEffects[appID]` using full substitution/defaulting before validation. Reuse existing
solvers without eager closure of enclosing recursion variables. This map describes the callee's
invocation effects, not the complete TypedApp effect union: validation still separately combines
argument-expression effects, avoiding double counting budgets/params.
Every successfully typed App has an entry, including an explicit closed-empty value for purity.
Use map membership to distinguish absent data from present purity. Owned open tails require a
verifiable ownership contract; ordinary unresolved CoreTI tails do not satisfy it.

Consume publication for the same-module tail-bearing declaration path, keeping declared concrete
labels authoritative and the contamination-safe concrete path. Thread data through both compile
paths and direct validator test helpers. Preserve the companion's field/parameter/alias handling;
this sprint does not replace its helper extraction changes. No CoreTI fallback for discharge.

- [ ] Pure single-call App publishes closed empty and checks clean (AC1); two occurrences publish independently as empty and IO (AC3/AC6).
- [ ] Wrong FS declaration and pure mixed-call laundering reject naming IO; a correct IO declaration accepts (AC2/AC3/AC11).
- [ ] Return-only row in the eligible concrete pure caller publishes empty; legitimate generic context publishes its owned tail (AC5/AC8).
- [ ] Missing map/entry, wrong App key, malformed row and unowned tail fail with callee and App ID; deliberately omitted publication cannot fall back to CoreTI (AC8).
- [ ] Both pipeline entry points and recursive/imported-module paths satisfy publication coverage; no stale declaration-level row is reused across Apps.
- [ ] Nested show/println, recursive multi-effect inference and annotation/owned-tail invariants remain green; no callee-row eager-default workaround is introduced.

**Risk:** A3 alone could publish an unsolved row. M1 pins its derivation; tests assert solved data
and downstream behavior. Minimal constraint repair within the existing model is in scope; new
instantiation architecture requires design review.

### M4: Joint regression matrix, examples and release evidence (~220 LOC)

**Estimate:** 30 production + 130 tests + 60 examples/docs; 6 hours day 5, day 6 reserved for failures.
**Dependencies:** M2, M3.
**Files:** `effect_rowvar_discharge_test.go`, `effect_call_publication_test.go`, relevant
companion effect tests; new `examples/runnable/effect_row_var_pure_caller.ail` and
`examples/runnable/effect_row_var_mixed_callbacks.ail`, `examples/manifest.json`,
`changelogs/v0.32-current.md` (linked by CHANGELOG.md), `docs/LIMITATIONS.md` if applicable,
and source design's completion metadata at close-out.

- [ ] AC11 full matrix passes: arms a/d/e/k; mixed IO+e accepted with IO and rejected without; concrete IO under e rejected; declared imported mapE pure accepted and forEachE wrong-row rejected; let alias, inline lambda, Debug+e and 71b610d68 recursion controls.
- [ ] Inferred effect-transparent std/option, std/result and std/list combinators and #1091 pure-export guards pass independently of declared-row controls.
- [ ] Companion same-module/imported/inline/field/parameter/constructor/record-update matrix remains sound, including pure callback acceptance and IO callback rejection.
- [ ] Both new runnable examples check and run with expected results/caps; manifest is consistent. runTwice(noisy) under a pure signature fails check and cannot reach the old two-print execution.
- [ ] Mutation checks kill tail stripping, declaration-level publication, missing-entry CoreTI fallback, tail-name-presence matching and removed blank-diff/suggestion guards; record an unreachable mutation rationale if type invariants prevent construction.
- [ ] Focused package tests and full implementation gates pass (AC9); changelog explains accept-to-reject migration by declaring real effects, and LIMITATIONS entry is corrected if present (AC12).
- [ ] Implementation PR body contains Refs #616 and Closes #616, links validation evidence and companion integration SHA; plan PR contains only Refs references. Closing/commenting occurs with authorized implementation merge, not planning.

**Risk:** suite-green without reaching the defect. Require the base-red controls, outcome-level
mutation checks and cache-safe source matrix together with the green regression suite.

## Day-by-day execution

| Day | Work | Evidence required |
|---|---|---|
| 1 | M1 baseline refresh + constraint trace | SHA, base-red arms, existing solver contracts, publication semantics |
| 2 | M2 row algebra + diagnostics | runTwice rejection, both union consumers, actionable diagnostic tests |
| 3 | M3 capture/finalize invocation rows | independent App IDs, pure/IO and return-only map values |
| 4 | M3 pipeline consumers + contract failures | both compile paths, named missing/malformed-entry errors, recursion controls |
| 5 | M4 combined matrix + examples/docs + gates | no over-rejection, companion soundness, verified runnable examples |
| 6 | Contingency only | resolve regression/gate failures, reevaluate changed work; no new feature scope |

## Validation and success metrics

At execution start build a source-matched binary with `make build`; run `ailang prompt` before
writing/editing .ail. Source probes use new temp path **and content/module identity** per design
V38. Pair passing probes with failing controls; cached silence is not publication evidence.
Use existing pipeline test helpers, inspecting their cache behavior first.

Inner loop: `go test ./internal/types/ ./internal/pipeline/ -count=1`, adding
`./internal/elaborate/ ./internal/iface/` for the joint integration surface. Record initial
changed-package coverage and require coverage of tail branches, publication ownership/membership
and diagnostic invariant paths, without claiming an unmeasured repository percentage.

Final implementation gates: `make test`, `make lint`, `make fmt-check`, `make check-boundaries`,
`make check-file-sizes`, `make verify-examples`, `make verify-stdlib` and the existing changed-package
coverage workflow. Re-run relevant focused checks after any fix. Every failing gate is diagnosed;
manifest drift does not excuse a red verify-examples result. Keep large-file guards in view:
use a small type-checker publication companion file if existing files exceed limits.

Done means all design AC1–AC12 are mapped above, both new runnable examples pass, the negative
programs fail naming their cause, concrete and polymorphic regressions stay green, and the
implementation is evaluated with the sprint-evaluator skill after execution. No runtime behavior
or capability policy changes are needed.

## Approval and resumption

The scheduling approval authorizes this plan; it does not self-approve implementation.
Machine state: `.ailang/state/sprints/sprint_M-EFFECT-ROW-VAR-UNIFICATION.json`, with real
milestones and null progress fields. Coordinator plan approval/merge is the handoff point;
attended execution requires the user to say "execute sprint" per AGENTS.md. Do not dispatch an
executor before that gate. The coordinator must retain the design's opus-required routing
requirement and must surface an unavailable required model rather than silently downgrade.

Resume from the JSON, verify companion status/base, run M1 discovery gates, then proceed in
milestone order. If the release target has changed, update both plan and JSON to v0.54.0 before
implementation review. No new issue, issue closure, code implementation or runtime probe was
performed during planning.
