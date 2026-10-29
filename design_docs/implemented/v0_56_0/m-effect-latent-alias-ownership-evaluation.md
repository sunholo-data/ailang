# #2008 alias ownership follow-up — independent evaluation

**PASS 95/100**, plus regression-surface coverage 10/10 reported separately. Exact head `d5f783585084ff5eb791a7db34fc9de8899b2fc9`. Evaluation scope is **R1/R2 alias ownership only**; the original parent round1 PASS93 is unchanged.

All 26 PR checks are terminal: 19 SUCCESS, 7 SKIPPED, none pending or failed. The full CI workflow completed successfully on that exact head. The documented evaluator reports `CI_GATE=pass`; Linux/Windows tests, builds, Mac/arm64 checks, lint/structure, docs and Sonar passed. Actual Sonar new coverage is100% against the unchanged80 threshold; every gate condition is OK. No gate-policy waiver or CI retry was used.

The two repaired paths match the approved AC6/fwd_two contract: implicit ownership expands named aliases through the existing data-arrow walker, and every implicit-owned tail is included in general publication ownership before solver re-anchoring. Declared/unknown rows retain their existing obligations and checked-union failures. Caller charging remains conservative across the full parameter; unused IO/FS callbacks are still charged. No rowguard, explicit-empty, mask, solver or VM semantics were redesigned.

All nine R1/R2 criteria are supported by inspected source/tests and measured artifacts. The unchanged social_dynamics tasks source fails app1070,arg1 on actual80793 and checks clean on the repair. Exact delivered consumer083ef437, tree067e8990, passes fresh-empty-cache `gmake check_core`:60/60 core,9/9 extension boots,204 default runtime guard assertions. The actual80793 paired530-file corpus has446 passes and84 rejects in each arm, with zero status/diagnostic changes and zero invariants. The final metadata head is production/AIL/std/Docker/prompt/golden equivalent to tested integration15afcc12.

Independent small checks passed: alias publication, solver renaming, declared-tail loudness and erase-subset-charge (0.355s). Eight additional local annotated-let alias programs preserve IO/FS caller charging in either callback position: undeclared caller rejects with its missing effect; matching declared caller accepts. No source worktree edits or repeated broad local tests were performed.

Strict-FS qualification remains explicit. Two initial invoked-FS strict positive failures are preserved as the existing Phase2E evaluator-only builtin/interop limitation. Final invoked-FS positives use normal bytecode interop and assert VM dispatch with no top-level evaluator fallback. Missing-effect negatives, executed IO and unused-FS acceptance retain strict mode. Strict executed-FS parity is not claimed; this repair changes no VM code.

| Category | Score |
|---|---:|
| Full tests |20/20|
| Lint |10/10|
| R1/R2 acceptance criteria |30/30|
| Code quality |10/15|
| Documentation |15/15|
| Design fidelity |10/10|

Deductions:−2 for new `TestAliasOwnershipCLI` (85 lines, measured with Go AST; rubric includes tests), and−3 for the new `alias_ownership_review_repair.integration` field still saying pending integration despite completed R2 in the same object. Normalize that metadata in the completion audit while preserving this evaluated snapshot. No production respin is required. Documentation uses the existing applicable runnable example and new real regression programs; this repair introduces no new feature requiring a showcase module. Inherited parent target/archive state is not rescored here.

Original parent PASS93, raw red social/alias logs, the initial strict-FS verification mismatch, and provisional review remain preserved. All three PR feedback endpoints were empty at the final exact-head check. Root owns merge and final archive; no merge or external comment was performed by this reviewer.

Evidence under `/tmp/ailang-backlog-20261029`: `pr2008-final-{pr,checks,ci-run,sonar-gate}.json`, `pr2008-final-ci-gate.log`, `alias-ownership-independent-{types.log,let-controls.json,function-sizes.txt}`, and executor `alias-ownership-R2-*` logs. Criterion-by-criterion detail is in the companion JSON.

EVALUATION_SCORE: 95/100
EVALUATION_ROUND: 2
EVALUATION_VERDICT: PASS score=95

## Archive acceptance

Merged in [PR2008](https://github.com/sunholo-data/ailang/pull/2008) at749045717c484299b3103008023bf27a826df844. R1/R2 only:95/100; original parent round1 PASS93 unchanged. [Validation](m-effect-latent-alias-ownership-validation.md), [executor evidence](m-effect-latent-alias-ownership-evidence.json), [machine evaluation](../../../.ailang/state/evaluations/eval_M-EFFECT-LATENT-NESTED-CALLBACKS_round_2.json). Target0.56.0 remains unreleased.
