# Sprint Plan: M-SCC-SHADOW-AWARE-CALLGRAPH

**Design:** [Approved design](m-scc-shadow-aware-callgraph.md)  
**Issue:** #2078  
**Target:** v0.57.2  
**Status:** Planned; implementation awaits coordinator plan approval.  
**Duration:** 2.5 working days, approximately 7.5 focused implementation hours plus investigation, corpus execution and review buffer.  
**Risk:** Medium: dropping genuine ordering edges can reintroduce false undefined-variable errors.

## Goal and frozen decisions

Remove false SCC edges from locally shadowed identifiers while preserving genuine recursion, module declaration ordering and per-call effect validation. Change production code only in internal/elaborate/scc.go. Keep internal/types and internal/pipeline production behavior untouched. Follow the approved lowercase pattern classification, local versus bodyless module letrec split, and complete the five missing expression cases. Contracts and test expressions outside function bodies remain out of scope.

There are no human-decision gates inside the approved design. Representation and test organization are implementer choices. E2E placement is resolved: internal/elaborate/scc_shadow_effects_test.go uses package elaborate_test, matching module_let_resolution_test.go, avoiding the pipeline-to-elaborate import cycle. Implementation starts only after normal coordinator sprint-plan approval.

## Current status and estimates

Working tree was clean at planning start. Repository version is v0.57.0. scc.go is 303 lines and its current walker is scope-blind. BuildCallGraph and isSelfRecursive share that walker; module lets are already graph nodes. The design has audited related binder forms and genuine effectful recursion, satisfying the systemic analysis requirement.

The seven-day velocity script found only the design commit c96b7180, no implementation delta and no usable recent LOC metric. Historical changelog matches are not a measured velocity. The 510 LOC estimate (roughly 100 production, 380 tests, 30 documentation) is a planning assumption, yielding 204 LOC/day over 2.5 days. Each milestone is scoped to about 60–90 minutes of focused editing; corpus runtime and investigation are budgeted separately. If traversal work exceeds that window, checkpoint before continuing in a new executor run.

## Registry reuse audit

All five milestones: none. This is compiler-internal scope analysis and regression testing, not a package-like capability; registry searches and candidate inspections are not applicable. Reuse the existing AST, scope classification, checkModule pipeline harness, make targets and corpus harness. Do not add a package dependency or an ad hoc sweep tool.

## Syntax and examples

AILANG prompt version loaded: v0.16.11 (reported by the installed CLI; repository std/VERSION is v0.57.0). The full current prompt was loaded during planning. Executor must build/use the repository binary and reload its prompt before writing AILANG fixtures; the installed CLI is not the authoritative baseline for v0.57.0 behavior.

No standalone showcase module is added. The original issue module and variants live as temporary .ail sources in the Go regression matrix. Their checklist is:

| Concern | Decision |
|---|---|
| contracts | skip: compiler acceptance/rejection fixtures do not need runtime contracts |
| effects | include: each[e] declares ! {e}; sibling t declares ! {IO}; soundness controls introduce concrete IO at recursive call sites |
| inline tests | skip: Go pipeline matrix asserts compile outcomes and diagnostics, including intentionally rejected fixtures |

Check accepted fixtures with the candidate CLI as well as pipeline tests. Preserve expected rejection of negative fixtures.

## Milestones

### M1: Pattern binder classification (~100 LOC)

**Duration:** 60–90 minutes focused work.  
**Dependencies:** none.  
**Files:** internal/elaborate/scc.go; internal/elaborate/scc_test.go.  
**Registry reuse:** none: compiler-internal AST scope and SCC analysis plus regression coverage; no package-like capability or runtime dependency is introduced. Registry search is not applicable.

- [ ] astPatternBinders agrees with corePatternBinders after elaboratePattern for lowercase, uppercase, wildcard, nested constructor, cons, list/rest, tuple and record patterns.
- [ ] Constructor and record field names never become binders; nil patterns are harmless.

**Validation:** Run targeted internal/elaborate tests for this milestone; preserve all previously passing controls.

### M2: Scope-aware reference traversal (~160 LOC)

**Duration:** 60–90 minutes focused work.  
**Dependencies:** M1.  
**Files:** internal/elaborate/scc.go; internal/elaborate/scc_test.go.  
**Registry reuse:** none: compiler-internal AST scope and SCC analysis plus regression coverage; no package-like capability or runtime dependency is introduced. Registry search is not applicable.

- [ ] Lambda and FuncLit parameters, match guards/bodies, local let/letrec, sequential blocks and forall bodies filter locally bound names without scope leakage.
- [ ] Let values and forall bounds retain outer references; bodyless module letrec retains its self-edge.
- [ ] ForallExpr, Send, Recv, AssertStmt and QuasiQuote interpolations retain genuine references; DEBUG_STRICT still panics for an unhandled expression variant.

**Validation:** Run targeted internal/elaborate tests for this milestone; preserve all previously passing controls.

### M3: Call graph and emission controls (~100 LOC)

**Duration:** 60–90 minutes focused work.  
**Dependencies:** M2.  
**Files:** internal/elaborate/scc_test.go; internal/elaborate/scc_shadow_effects_test.go.  
**Registry reuse:** none: compiler-internal AST scope and SCC analysis plus regression coverage; no package-like capability or runtime dependency is introduced. Registry search is not applicable.

- [ ] BuildCallGraph has no false each-to-t edge for pattern, lambda and let shadows; non-shadow controls keep genuine edges.
- [ ] isSelfRecursive ignores shadowed self names; bodyless module letrec still emits core.LetRec; genuine mutual recursion still groups.
- [ ] TestModuleLetResolutionMatrix and TestLexicalScope_BinderShadowsGlobals pass unchanged.

**Validation:** Run targeted internal/elaborate tests for this milestone; preserve all previously passing controls.

### M4: Effect regression matrix (~120 LOC)

**Duration:** 60–90 minutes focused work.  
**Dependencies:** M3.  
**Files:** internal/elaborate/scc_shadow_effects_test.go.  
**Registry reuse:** none: compiler-internal AST scope and SCC analysis plus regression coverage; no package-like capability or runtime dependency is introduced. Registry search is not applicable.

- [ ] All eleven design reproduction rows run through pipeline ModeCheck using elaborate_test and existing checkModule helper.
- [ ] Rows 1 through 9 pass, including original issue and let-discard variants; rows 10 and 11 reject with Missing effects and IO.
- [ ] Baseline binary rejects positive controls 1, 5 and 6; candidate accepts them; no internal/types or internal/pipeline production changes.

**Validation:** Run targeted internal/elaborate tests for this milestone; preserve all previously passing controls.

### M5: Corpus comparison and release evidence (~30 LOC)

**Duration:** 60–90 minutes focused work.  
**Dependencies:** M4.  
**Files:** changelogs/unreleased/2026-11-04-scc-shadow-aware-callgraph.md.  
**Registry reuse:** none: compiler-internal AST scope and SCC analysis plus regression coverage; no package-like capability or runtime dependency is introduced. Registry search is not applicable.

- [ ] Pre-fix and candidate check results have zero unexpected status flips across examples, std and the design-referenced 1,066-file corpus; positive controls flip as expected.
- [ ] make test-changed, make check-file-sizes, make check-changelog and make verify-examples pass; formatting and lint checks pass.
- [ ] Changelog fragment opens with ### and references #2078; evidence records binary commits, corpus manifest, commands and results.

**Validation:** Run targeted internal/elaborate tests for this milestone; preserve all previously passing controls.

## Day-by-day execution

Day 1: Save a base binary and commit identity before any production edit; capture positive and negative baseline results. Complete M1 and M2, prioritizing nonrecursive let values, module letrec self-edges and nested scope isolation.

Day 2: Complete M3 and M4. Exercise all binder and reference positions, not only the issue's cons pattern. Run unchanged scope/module ordering matrices and the new effect matrix.

Day 3 (half day): Complete M5, collect differential corpus evidence, run focused gates, and prepare evaluator handoff.

## Validation and corpus dependency

Use make test-changed (touched packages plus test-core), make check-file-sizes, make check-changelog and make verify-examples. Run repository formatting and lint targets. Do not require make test; CI owns the full suite. No touched existing file is near 800 lines; keep new graph tests and external pipeline tests separate, splitting a companion scc_scope_test.go if needed.

Locate the existing #2074/#1958 corpus harness and exact 1,066-file manifest before M5. A tools filename search during planning did not locate it; it may be an external artifact. Capture baseline and candidate results against identical files, working directories, stdlib and flags; compare status per path, preserving pre-existing failures. Never report zero flips from aggregate pass counts or an empty corpus. Save evidence under .ailang/state/sprints/ alongside sprint progress. If the harness/corpus cannot be obtained, run the repository examples/std checks, mark the external sweep unverified, leave M5 incomplete and report the missing artifact; do not substitute a smaller corpus silently.

## Completion and handoff

Success requires every criterion above, 510 estimated LOC total, all binder/reference branches covered by explicit controls, unchanged genuine recursion and validator rejection behavior, and complete differential sweep evidence. No global percentage coverage target is invented; require branch-specific scope and soundness coverage.

Use refs #2078 for planning and intermediate commits. Fixes #2078 is permitted only in a final commit that changes code. The plan itself must never close the issue. The coordinator consumes the plan and JSON artifacts for review; merging the sprint-plan PR is the documented executor trigger. No executor is launched by this planning stage. After approval, executor loads the JSON, implements milestones and hands results to sprint-evaluator against the approved design.
