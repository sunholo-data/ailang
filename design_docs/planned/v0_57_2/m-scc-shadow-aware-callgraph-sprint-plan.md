# Sprint Plan: M-SCC-SHADOW-AWARE-CALLGRAPH

**Design:** [Approved design](m-scc-shadow-aware-callgraph.md) · **Issue:** #2078 · **Target:** v0.57.2
**Status:** Planned; implementation starts through the coordinator approval/handoff workflow.
**Duration:** 2.5 working days, approximately 14 hours including integration/debugging buffer.
**Risk:** Medium: removing an incorrect edge must not remove genuine declaration-order dependencies.

## Goal and scope

Make the elaborator's reference walker respect lexical binders so unrelated callers no longer force a row-polymorphic recursive helper into their SCC. Preserve per-call-site effect charging and genuine mutual recursion. Production edits stay in `internal/elaborate/scc.go`; add direct graph tests and external pipeline regression tests. Do not change type inference, effect validation, ApplicationEffects, or traversal of function contracts/tests.

## Current implementation and estimate

Inspected on 2026-11-04 at `93ac5a91`, current work branch `coordinator/task-07a17e59`, repository version v0.57.1. The working tree was clean. `scc.go` is 303 lines and still scope-blind; it has no direct test file. The existing external `elaborate_test` package in `module_let_resolution_test.go` already imports pipeline and exposes `checkModule`: reuse it in `internal/elaborate/scc_effects_test.go` without creating an import cycle. `patterns.go` is 266 lines; put the new helper beside its only caller in `scc.go`.

The velocity script found only the design re-verification commit in this shallow checkout and no usable recent LOC/day measurements. Historical changelog totals are not current velocity. Budget uses the approved design's 2.5 days / 14 hours, not a claimed measured throughput. Approximately 460 added/changed LOC (120 implementation, 320 tests, 20 documentation); planning capacity 184 LOC/day. Seven bounded executor milestones each target 60–90 minutes of active work, with roughly 4 hours reserved for integration, sweep runtime and debugging. Corpus execution can run longer than active preparation; checkpoint results between runs.

Installed CLI reports v0.57.1, commit b91eb27-dirty, and warns it may be stale. Build baseline and candidate from recorded source revisions for validation; do not treat the installed binary as checkout evidence.

## Registry reuse audit

Ran `ailang pkg search callgraph`: no packages found. No package-like user capability is being introduced: these are compiler-internal AST/scoping changes and tests, inaccessible to an AILANG dependency. No candidates required `pkg info`/`pkg docs`. Every milestone records `none`, including validation/docs work, with this reason in JSON. No dependency or package publication is planned.

## Approved rulings and resolved choices

- Fix call-graph construction; leave validator and ApplicationEffects semantics intact (Design Freeze and High-Impact Decisions).
- Pattern classification mirrors `elaboratePattern`, including uppercase constructor and underscore handling; pin agreement through tests.
- Preserve module-level bodyless-letrec self-references; filter local-letrec binders in both value and body.
- Add all five missing expression variants and retain DEBUG_STRICT failure behavior.
- E2E location is resolved to `internal/elaborate/scc_effects_test.go`, package `elaborate_test`, following the existing harness. This was an agent-resolvable placement choice, not an outstanding human decision.
- Adjacent observation: `BuildCallGraph` and `isSelfRecursive` walk raw function bodies without `FuncSig.Params`. Record this boundary for follow-up rather than expanding the approved walker-only production scope into `file.go`; this sprint covers the binder forms enumerated in the design.

No human-decision gates remain. The approved design's implementation choices do not require a new ruling. Do not self-approve or start implementation from this planning task; coordinator approval/merge supplies the execution handoff.

## Milestones

### M1: Pattern binder classification (~70 LOC)

**Dependencies:** none
**Timebox:** 60–90 minutes active work.
**Files:** `internal/elaborate/scc.go`, new `internal/elaborate/scc_test.go` (same-package helper tests).

- [ ] Surface pattern binder classification agrees with elaboratePattern/corePatternBinders for identifiers, constructors, cons, lists/rest, tuples, records, wildcards and literals.
- [ ] Underscore and uppercase constructors do not bind; nested lowercase identifiers do.

### M2: Lexical reference traversal (~110 LOC)

**Dependencies:** M1
**Timebox:** 60–90 minutes active work.
**Files:** `internal/elaborate/scc.go`, `internal/elaborate/scc_test.go` (scope isolation and positive/negative graph controls).

- [ ] Lambda/FuncLit parameters, nonrecursive let values/bodies, match guards/bodies and sequential blocks obey lexical scope without leaking names across branches.
- [ ] BuildCallGraph excludes false shadow edges while preserving real module references, use-before-let edges and genuine mutual recursion.

### M3: Recursion and emission boundaries (~70 LOC)

**Dependencies:** M2
**Timebox:** 60–90 minutes active work.
**Files:** `internal/elaborate/scc.go`, `internal/elaborate/scc_test.go`, `internal/elaborate/scc_effects_test.go` (emission check). Audit `file.go` read-only; use graph entry scopes rather than expanding production scope.

- [ ] Local letrec binds in value and body; module-level bodyless letrec retains its self-edge and emits core.LetRec.
- [ ] isSelfRecursive distinguishes shadowed and genuine self-references; graph and core emission tests preserve module-level recursive bindings.

### M4: Complete expression position coverage (~70 LOC)

**Dependencies:** M3
**Timebox:** 60–90 minutes active work.
**Files:** `internal/elaborate/scc.go`, `internal/elaborate/scc_test.go` (AST-level cases for every new expression position).

- [ ] ForallExpr walks bounds outside Var scope and body inside it; Send, Recv, AssertStmt and QuasiQuote interpolations preserve genuine references.
- [ ] DEBUG_STRICT unknown-variant panic remains; existing expression positions and module-let ordering matrix pass.

### M5: Pipeline reproduction and soundness matrix (~120 LOC)

**Dependencies:** M4
**Timebox:** 60–90 minutes active work.
**Files:** New/extend `internal/elaborate/scc_effects_test.go`, external test package; reuse `checkModule`.

- [ ] All eleven design reproduction rows run through pipeline.Run ModeCheck: rows 1-9 pass; rows 10-11 fail with Missing effects on the expected function.
- [ ] Pre-fix rows 1/5/6 fail and post-fix pass; the let-underscore callback variant also passes after the fix.
- [ ] Existing TestModuleLetResolutionMatrix and TestLexicalScope_BinderShadowsGlobals pass unchanged; no internal/types or internal/pipeline production changes.

### M6: Corpus comparison (~0 LOC)

**Dependencies:** M5
**Timebox:** 60–90 minutes active work.
**Files:** Verification evidence under `.ailang/state/sprints/`; existing examples and std modules are read-only fixtures. Reuse the historical sweep harness rather than adding a new repository script.

- [ ] Identical baseline/candidate manifests and check options cover examples/, std/ and the historical 1066-file corpus; record counts, binary commits, statuses and diffs.
- [ ] There are zero status flips outside intentionally separate positive-control repro fixtures; rows 1/5/6 flip to success and rows 10/11 remain failures.
- [ ] Historical corpus and harness are located and run; missing inputs are reported as blocked rather than counted as a successful sweep.

### M7: Changelog and focused release checks (~20 LOC)

**Dependencies:** M6
**Timebox:** 60–90 minutes active work.
**Files:** New `changelogs/unreleased/2026-11-04-scc-shadow-aware-callgraph.md`; sprint verification notes.

- [ ] A changelogs/unreleased/2026-11-04-scc-shadow-aware-callgraph.md fragment opens with ### and references #2078.
- [ ] make test-changed with an explicit available baseline, make check-file-sizes, make check-changelog, make verify-examples, make verify-stdlib and make lint pass; Go edits are gofmt clean.
- [ ] Verification evidence and sweep artifact paths are recorded; no feature scope expansion or effect-validator changes.

## Execution schedule and risks

Day 1: M1–M3, then establish pre-fix failures for the integration matrix. Day 2: M4–M5, locate the historical sweep inputs and begin M6. Day 3 (half day): finish M6–M7 and use the reserved buffer. Capture a baseline binary before production edits, without switching branches or discarding local changes.

Wrong binder classification or scope leakage can silently drop a real edge (#327): mitigate with positive controls, nested scopes, use-before-let, sibling branches and the module-let ordering matrix. False self-recursion can change emission: explicitly inspect core Let/LetRec shapes. A missing expression case must still fail loudly under DEBUG_STRICT. Tests should assert graph edges and pipeline behavior rather than brittle node IDs or reference ordering.

Keep each new test file below 800 lines; if the direct tests approach that limit, use companion `internal/elaborate/scc_scope_test.go` for the scope matrix. The projected production file remains well below 800 lines.

## Examples and AILANG syntax gate

AILANG prompt version loaded: v0.16.11 (actual `ailang prompt --version-active` output); full teaching prompt requested using `ailang prompt`. This prompt version differs from repository/CLI release numbering and is recorded as returned.

No new standalone .ail example is needed for this correction. The exact reported source and its variants become embedded .ail pipeline fixtures in M5, checked against the source-built compiler. Keep existing examples `inline_tests_recursive.ail`, `record_cons_pattern.ail`, `first_non_repeat.ail` and `std/list.ail` unchanged and verify them. Do not modernize the issue syntax while reproducing the bug.

| Fixture group | Contracts | Effects | Inline tests |
|---|---|---|---|
| Accepted repro / controls (rows 1–9, underscore variant) | skip: graph/effect validation is the behavior under test | include: recursive `each[e]` declares `! {e}`, IO caller declares `! {IO}`; pure/mutual controls retain design signatures | skip: Go table assertions drive ModeCheck and inspect diagnostics/grouping |
| Rejected soundness guards (rows 10–11) | skip: intentionally invalid effect programs | include: declared `! {e}` with a genuinely introduced closed IO callback must fail | skip: invalid modules cannot run inline tests; negative pipeline checks assert expected function and Missing effects |
| Existing example modules | skip: unchanged existing examples | skip: preserve existing signatures and effect rows | skip: preserve existing inline tests; run existing verification targets |

## Validation and completion evidence

Run focused Go tests for new graph/scope/pipeline matrices while implementing, including existing `TestModuleLetResolutionMatrix` and `TestLexicalScope_BinderShadowsGlobals`. Final executor gate is `make test-changed` (includes `make test-core`), with `TEST_CHANGED_BASE` explicitly set to the recorded pre-sprint commit when origin/dev is absent or unsuitable. Run `make check-file-sizes`, `make check-changelog`, `make verify-examples`, `make verify-stdlib`, and `make lint`; format only edited Go files. CI owns the full test suite.

For M6, locate the #2074/#1958 sweep harness and its 1,066-file manifest before claiming this acceptance criterion. Repository search found no checked-in copy. Compare baseline/candidate statuses on identical source manifests/options, including failures that are expected to remain failures; bank paths, counts, commands, revisions and status diffs. Exclude the newly added positive controls from the zero-flip legacy corpus and report their expected flips separately. The historical 20,046-file sweep is background impact evidence, not a newly promised rerun.

Missing corpus/harness is an external validation dependency, not an open design decision. If it cannot be recovered, **STOP: commit current work, report `BLOCKED: historical 1,066-file sweep corpus/harness unavailable`, and end the run**. Do not silently replace it with examples-only coverage or mark M6 passing. This allows useful implementation milestones to complete before reporting the exact validation limitation.

Success requires all seven milestones, the 11-row matrix and graph controls, zero unintended corpus status flips, unchanged soundness rejection, unchanged effect-validator code, and a changelog fragment referencing #2078. Coverage target is all binder forms and five missing expression variants, rather than an unmeasured package percentage. Record DEBUG_EFFECTS evidence showing the recursive call retains an open row while the IO caller has its concrete site row. Do not require stable node IDs.

## Handoff

Plan and JSON are ready for coordinator review. Merge/approval of this planning task triggers sprint-executor per the coordinator workflow; no duplicate direct execution message is needed before that approval. Execution should reference #2078; only the final implementation completion should use Fixes #2078. This plan does not close or implement the issue.
