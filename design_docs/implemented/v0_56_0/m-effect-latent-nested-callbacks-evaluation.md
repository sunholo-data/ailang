# PR #1963 independent sprint evaluation — PASS 93/100

Reviewed head: `86d3a5f8e946e12967c38b446e5fb854a884a42e` against integrated dev `fcb95e0b1`.
Sprint: M-EFFECT-LATENT-NESTED-CALLBACKS. Independent evaluator; no code or sprint JSON edited. Existing repair handoff remains `pr1963-repair-report.md`.

**PASS 93/100 at exact head `86d3a5f8e946e12967c38b446e5fb854a884a42e`, gate_source=local.** The documented explicit local-evaluation path supplies the native full test gate; successful exact-head remote checks supply independent Linux/Windows/lint/security/examples/launchd evidence. Two hosted macOS build jobs remain queued. No merge, coordinator action, approval, or GitHub comment was performed by the evaluator.

## Scope and findings

Reviewed the full approved design and sprint plan, completed sprint JSON, original implementation plus repair diff, regression tests, all three PR review endpoints, and the executor's captured red/green logs. The only historical review demand is the demonstrated depth-17 erase-without-charge failure at old head `0860b0781`; it is covered directly by this repair.

`dataArrowsWith` in `internal/types/typechecker_latent_data.go` visits only data positions, sorts record labels for deterministic traversal, and expands alias heads at each level. Active source and expanded node identities terminate cycles without a global visited set suppressing reachable sibling callbacks. Function arrows are leaves, preserving the approved exclusion of arrow parameters and returns.

The four mask/charge/ownership/application sites share the walker. Callback effects are recomputed after substitution; concrete contract tails close only in cloned rows; unresolved distinct non-owned tails still fail loudly through the existing validator. Pure callback storage remains accepted. This matches the approved conservative call-site charging model.

The former silent depth-16 cut-off is gone. Finite depths 16/17/32/256 remain charged. At the defensive 1000-level ceiling, traversal throws only a typed private marker; the public InferWithConstraints and CheckCoreExpr wrappers return an ordinary type error. Unrelated panics rethrow. Overflow cannot publish a partial successful row. Tests also exercise an expanding generic alias.

Evaluator and strict VM integration fixtures include accepted deep pure callbacks, accepted explicitly IO callers, and rejected pure callers before the callback's output. The negative test checks actual performed output rather than matching a quoted source excerpt. No emit-go parity is required by this approved sprint.

## Acceptance evidence

| Criteria | Evidence |
| --- | --- |
| AC1 nested leak shapes | `TestLatentNested_LeaksRejected`: R1–R9, including cross-module record alias |
| AC2 generic container definitions and uses | `TestLatentNested_AnnotatedGenericContainers`: R10/R11 definitions accepted, pure effectful uses rejected, declared IO accepted |
| AC3 concrete annotation tails | `TestLatentNested_ConcreteTailsClosed`: S1–S5 and reduced revokeFamily fixture; captured mcp_oauth 0.1.0/0.1.1 package sweep succeeds |
| AC4 unchanged accepted controls | `TestLatentNested_Controls`: pure lambdas, storage, length, row-polymorphic calls, two-callback forwarding, cross-module accepted program |
| AC5 unchanged rejected controls | Same controls plus mixed IO/FS in both orders; `TestLatentNested_DistinctDeclaredTailsFailLoud` preserves explicit invariant |
| AC6 shared traversal | `TestDataArrows_Positions`, `TestDataArrows_AliasExpansionAndCycles`, deep/cycle tests; inspected all production call sites |
| AC7 erase subset of charge | `TestLatentEraseWithinCharge`, generated container/arrow/alias combinations; repair extends depth coverage; captured site-(a) mutation turns the assertion red |
| AC8 corpus compatibility | Approved Conflict Surface enumerates affected positions/other valid constructs and named consumers; original recorded 515-file examples+std and 45-package+ailang-parse differential has zero unexplained flips, mcp_oauth improvements are documented |
| AC9 required gates | Complete native `GOFLAGS=-p=1 make test` passed at the exact clean head outside TMPDIR, plus successful exact-head remote Linux/Windows test, lint, build, checks/examples, security and launchd gates; local sizes/lint passed |
| AC10 documentation/prompt | Breaking migration fragment, limitation entries resolved, runnable example+manifest, new v0.16.9 sentence and synchronized mirrors |

## Prompt preservation

Independently compared all source/embedded/website v0.16.7 and v0.16.8 bytes against integrated dev: unchanged. New v0.16.9 copies v0.16.8 except its version header and the approved one-sentence nested callback rule. All three v0.16.9 copies have SHA256 `6197fe5f4bf1c112737c550e08d557eea5c92db9b85839efaac8d46accd28be1`. Prior registry content hashes remain intact, only the formerly active v0.16.8 freeze metadata and new active version are changed semantically. No historical prompt was rewritten.

## Scoring notes for final gate

Final base rubric 93/100: tests 20, lint 10, acceptance 30, code quality 13, documentation 10, fidelity 10. Code quality deduction: new dataArrowsWith exceeds 50 lines. Documentation deduction: design remains Planned with unchecked acceptance boxes; parent is explicitly assigned to move/update design and companion plan after independent PASS. This is completion housekeeping and does not change implementation acceptance. Regression Surface Coverage separately meets all 10/10 conditional criteria; no hard failure and no performance goal/profiling requirement applies. Bonus is reported separately rather than inflating the base 100-point score.

The sprint JSON contains a clearly dated review_repair append superseding historical depth-16 and in-place prompt notes; those earlier notes are audit history, not current implementation claims. All milestones have passes, timestamps and notes; plan milestones are completed.

## Final gates and preserved failures

The parent explicitly selected the documented EVAL_LOCAL_GATES=1 evaluation path. Its initial concurrent native full run failed four deadline-sensitive tests: TestCodexExitedLeaderStillStopsOwnedChildBeforeReturn, TestHealthCheck_WithFakeBinary, TestOpenCodeTerminationKillsProcessGroup, and TestSolve_HardTimeout_FakeSolverIgnoringT. All four passed isolated serial rechecks, and the subsequent **complete** `GOFLAGS=-p=1 make test` passed with no test exclusions or source edits. Initial output remains `1963-native-evaluation.log`; the final full log is `1963-native-make-test-serial.log`. Reviewer independently verified the native worktree clean at the exact reviewed SHA. The serial run resolves the failed full gate; isolated tests alone were not substituted for it.

A supplemental concurrent native launchd run reported49/50 suites because one worker died without a verdict (`1963-native-launchd.log`); this is preserved as an incomplete local attempt. The actual exact-head remote launchd job later passed. Final remote metadata is saved in `pr1963-final-gate.json`: Linux and Windows full tests, lint, build, checks/examples, docs, vulnerability gate, CodeQL, SonarCloud, float determinism and launchd all succeed. Two hosted macOS build jobs remain **queued**; no remote all-checks-success claim is made. No required check was force-completed, cancelled, or bypassed by this reviewer.

## Parent handoff

This is independent source acceptance on the exact SHA under the explicitly selected local-gate path. Parent owns normal GitHub integration and the separate completion-audit PR moving design and companion plan to implemented/v0_56_0 with **Implemented — unreleased** status. The 5-point documentation deduction remains until that archival completion. Historical GitHub NEEDS-FIX evidence remains preserved. Any future semantic source change requires rechecking this evaluation's applicability. No further whole-suite rerun is warranted without a new delta or failure.


Archival clarification: original v0.55.0 target is historical. Because v0.55.0 has shipped, the parent assigns the breaking soundness change to the next minor v0.56.0, implemented/unreleased; no version bump or tag is performed by this review.
