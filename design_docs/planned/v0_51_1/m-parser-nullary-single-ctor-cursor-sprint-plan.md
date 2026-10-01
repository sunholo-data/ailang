# Sprint Plan: M-PARSER-NULLARY-SINGLE-CTOR-CURSOR

## Summary
Restore declaration boundaries after a leading-pipe single nullary constructor, preserving following exports and pure-function flags. Implement only the approved guarded cursor advance; the no-pipe spelling remains a type alias.

**Design:** [Approved design](m-parser-nullary-single-ctor-cursor.md)
**Status:** Planned; awaiting sprint approval/execution gate.
**Target:** v0.51.1
**Duration:** 1 working day, 4 hours work + 1 hour contingency.
**Risk:** Low implementation complexity, medium regression impact because exports and purity are affected.
**Dependencies:** Approved design received in coordinator handoff task-c0dad6ad; sprint execution remains a separate stage.

## Current Status Analysis

The current v0.51.0 source still unconditionally advances in the nullary leadingPipe branch of internal/parser/parser_type_decl.go. The with-fields branch already guards advancement, and parseTypeDeclBody's caller handles deriving both at and after the cursor. The design includes systemic analysis, prior v0.8.1 history, a token conflict matrix, and an 18-row verification log. No implementation milestones are complete.

The seven-day analyze_velocity.sh run found no usable LOC metrics; this shallow checkout exposes only one recent documentation commit (35ae3261). A measured historical LOC/day rate cannot be calculated. Use a bottom-up budget of 240 changed/added LOC for one day, dominated by regression tests; this is planning capacity, not measured velocity. Five hours includes 25% contingency on the four-hour task budget.

## Registry Reuse Audit

Ran `ailang pkg search parser` on 2026-10-01. It returned sunholo/gemini_live@0.5.0 (protocol parsers) and sunholo/external_backend@0.2.0 (process wrappers), neither a compiler parser implementation. No relevant candidate needs info/docs inspection. The CLI warned its binary may be stale; registry results are discovery evidence only, not proof of compiler behavior.

| Milestone | Decision | Reason |
|---|---|---|
| M1 | none | Compiler cursor positioning and Go AST tests must change in internal/parser; registry protocol helpers cannot replace this code. |
| M2 | none | Repository package-import regression and verification use existing loader/CLI infrastructure; no package-like capability is added. |

## Proposed Milestones

### M1: Guard cursor advance and pin declaration boundaries (~150 LOC)

**Goal:** Restore the cursor convention and lock the affected grammar surface.
**Estimated:** 10 implementation/comment LOC + 140 test LOC = 150 LOC.
**Duration:** 2 hours. **Dependencies:** None.
**Files:** internal/parser/parser_type_decl.go; new internal/parser/type_decl_cursor_test.go (or existing type_test.go if that fits better).
**Examples:** Two-declaration sources embedded in parser tests; no new language feature or standalone example required.

Tasks: Write failing follower tests first. Assert declaration count, AST type, constructor name/fields and follower Exported/IsPure flags. Apply the PIPE-or-DERIVING peek guard only in the nullary leadingPipe branch and document the declaration-loop convention. Pin unchanged forms, including named alias, record, fielded constructor, multi-variant and deriving forms. Include EOF and annotated-function cases using existing annotation fixtures as syntax authority. Compare exported-symbol sets for std/stream.ail, std/regex.ail, std/dom.ail, std/option.ail and std/env.ail.

Acceptance criteria:
- [ ] Before the fix, new follower regressions fail for the reported cursor corruption; after the fix, they pass.
- [ ] Followers export type, export pure func, pure func, func and an annotated exported function parse without errors and retain source flags.
- [ ] Single nullary constructors at EOF parse cleanly; multi-variant, deriving, fielded, record and no-pipe alias AST shapes remain unchanged.
- [ ] The five named std modules retain their declared export sets.
- [ ] Only the approved leadingPipe nullary production branch changes; go test ./internal/parser passes.

Risk: Over-advancing or under-advancing breaks variant/deriving entry. Mitigation: AST assertions on each continuation token and both current-/peek-deriving paths.

### M2: Verify package exports and complete regression checks (~90 LOC)

**Goal:** Prove the user's IMP010 failure disappears through the package execution path.
**Estimated:** 80 integration-test/fixture LOC + 10 release-note LOC = 90 LOC.
**Duration:** 2 hours, followed by 1 hour shared contingency. **Dependencies:** M1.
**Files:** new cmd/ailang/nullary_ctor_package_test.go using established CLI integration helpers and temporary package fixtures; alternatively extend internal/loader/loader_intra_pkg_test.go for export assertions while retaining a real CLI run. Add the entry to the existing v0.51.1 changelog location, following repository release-note organization.
**Examples:** Temporary two-module rx/p fixture reproducing c.ail and b.ail from the approved design; persist source strings within the regression test, not an unrelated example tree.

Tasks: Obtain `ailang prompt` before authoring any AILANG fixture. Reuse package manifest and CLI build helpers already in tests. Import both W and mk, assert a successful run returning 2 and absence of IMP010. Use a freshly built local binary for the design's isolation matrix, not the stale installed binary. Add the release note because the design's success checklist requires it, despite its earlier statement deferring changelog work to release time.

Acceptance criteria:
- [ ] The rx/p package fixture imports W and mk successfully and main returns 2 using --quiet --package-dir and --entry main.
- [ ] The regression is persistent and exercises package import/export behavior, beyond parser flags alone.
- [ ] Rebuilt-binary isolation cases from the design verification log produce their expected outputs; formerly failing cases pass.
- [ ] make fmt, make test-core, make test and make lint pass; report any pre-existing/environment failure with evidence rather than marking it green.
- [ ] A v0.51.1 changelog entry describes the cursor/export fix without claiming no-pipe constructor syntax is supported.

Risk: Package setup can obscure the parser regression. Mitigation: Follow existing package tests and keep fixture/module paths identical to the reporter's case.

## Day-by-Day Execution

Day 1, hours 0–2: M1 red/green parser tests, minimal fix, focused parser suite.
Day 1, hours 2–4: M2 package reproduction, release note, rebuilt-binary matrix and required checks.
Day 1, hour 4–5: Contingency for integration harness or verification failures. Escalate a grammar change instead of expanding the approved scope.

## Success Metrics

Total estimated LOC: 240 (10 fix, 220 tests/fixtures, 10 release note). Both milestones pass with evidence; all affected follower flags and conflict forms have assertions. No global coverage percentage is asserted without a baseline. The package example returns 2 and full/core suites and lint pass. No std semantics, alias disambiguation, diagnostics or general cursor framework changes.

## Handoff and Open Questions

No blocking design questions remain. Fixture organization is delegated to the executor within the approved design. JSON tracks two milestones with passes/started/completed unset. No GitHub issue number was supplied or explicitly referenced in the design, so github_issues is empty rather than inferred from unrelated issues.

The coordinator should surface these artifacts for sprint approval and route the approved plan to sprint-executor, then sprint-evaluator. Design approval authorizes planning; execution awaits its own gate. No implementation or external executor message is sent by this planning stage.
