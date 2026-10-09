# Sprint Plan: M-NAMEDTEST-STRIP-STRING-BLIND-BRACES

## Summary

Make named tests and properties strip according to parser-recorded closing positions so brace characters in literals, comments and names cannot corrupt the module shared by every test.

**Design:** [Approved design](m-namedtest-strip-string-blind-braces.md)
**Target:** v0.53.1
**Status:** Plan ready for coordinator review; implementation not started.
**Duration:** 2 working days, 10 hours budget (8 hours work plus 2 hours contingency).
**Estimated change:** 300 added/modified LOC, including tests and fixtures; approximately 30 obsolete scanner lines deleted separately.
**Dependencies:** None external; M1 → M2 → M3.
**Risk:** Low overall; medium regression sensitivity at parser cursor and source-line mapping boundaries.

## Current Status and Velocity

The current tree has neither end-position field. Both parser return points already sit at RBRACE. The string-blind scanner remains in `internal/testing/source_strip.go`. It serves both `stripWithLineMap` and `stripTestBlocks`; named/property batching and evaluator/VM execution consume those paths. Keep both wrapper signatures unchanged: only the internal range helper loses its source-lines argument. The design's reference to dropping wrapper plumbing does not require changing their source input.

The checkout is shallow and exposes only the 2026-10-09 design commit (#1739). The supplied velocity script found no recent LOC metrics and no usable diff baseline. Historical LOC/day and completion rates therefore cannot be measured. Recent changelog entries show compile-once batching and helper retention as regression context, not velocity evidence. The estimate uses the approved design's 8-hour breakdown with 25% contingency, scheduled across two days; 150 LOC/day is planned capacity, not observed velocity. No implementation milestone is complete.

Installed CLI: v0.52.5, commit 7200786; repository std/VERSION: v0.53.0. Build the repository CLI for execution checks rather than using the installed binary as evidence of the fix. Coverage percentage was not collected during planning; acceptance is behavioral coverage of both declarations and both strip consumers.

## Registry Reuse Audit

Ran `ailang pkg search test` and `ailang pkg search parser`. Inspected `sunholo/testing_utils@0.1.1` with `pkg info` and `pkg docs`: it exports pure AILANG assertion helpers, not Go parser metadata or source stripping. Other hits are pipeline smoke packages or application protocol helpers. For M1, M2 and M3 the decision is **none**: extend existing repository APIs without a registry dependency. The JSON records a concrete decision per milestone.

## Technical Decisions and Scope

Choose additive `End ast.Pos` fields on `TestDecl` and `PropertyDecl`, paired with their existing `Pos`. The sole new consumer needs the closing position; a duplicate start in a Span adds no value here. Record the closing token without advancing it. Leave String()/Position(), grammar, lexer, generated-source escaping, batching and function retention unchanged. Delete the textual scan outright; an unset or inverted end collapses to the start line without lexical fallback.

The reported escaped-quote fragment alone passes on the current tree according to the approved design. Preserve it as a control, and regress unbalanced opening/closing braces in strings, comments and names. Cover properties as well as tests, nested real blocks, adjacency, and remaining-line maps.

## Proposed Milestones

### M1: Record authoritative closing positions on test and property declarations (~70 LOC)

**Duration:** 2.5 hours. **LOC split:** 6 implementation + 64 tests.
**Dependencies:** None.
**Files and examples:** `internal/ast/ast_decl.go`, `internal/parser/parser_test_decl.go`; add `internal/parser/parser_test_decl_test.go`. No standalone example: parser cases are exercised by the M2 fixture.

**Tasks:** Implement the milestone behavior with failing regression cases first, then run the relevant package tests.

**Acceptance criteria:**

- [ ] TestDecl and PropertyDecl expose additive End ast.Pos fields; String() and Position() stay unchanged.
- [ ] Both parsers capture p.curPos() at the verified closing RBRACE without advancing the cursor.
- [ ] Parser cases cover single-line, multiline, nested real braces, strings, comments, brace-bearing names and adjacent declarations for both node types.

**Risk and mitigation:** Cursor drift: capture only after the existing closing-brace check and assert the next declaration still parses.

### M2: Replace character scanning in both source stripping paths (~130 LOC)

**Duration:** 3.5 hours. **LOC split:** 20 implementation + 110 tests/fixture.
**Dependencies:** M1.
**Files and examples:** `internal/testing/source_strip.go`, `internal/testing/source_strip_test.go`; create `internal/testing/testdata/strip/brace_literals.ail` covering both declaration kinds. Reuse existing strip and named-helper-retention fixtures unchanged.

**Tasks:** Implement the milestone behavior with failing regression cases first, then run the relevant package tests.

**Acceptance criteria:**

- [ ] testAndPropertySkipRanges accepts only the AST file and uses Pos.Line through End.Line; all character scanning is deleted.
- [ ] Both stripWithLineMap and stripTestBlocks call the new helper; their public signatures and function-retention rules stay unchanged.
- [ ] Unit tests assert exact stripped source and line maps for tests and properties with opening and closing braces in strings, comments and names.
- [ ] Unset and inverted End positions collapse to the declaration start line for both hand-built declaration types; existing strip and helper-retention tests pass.

**Risk and mitigation:** Off-by-one ranges: assert complete stripped text and every retained original line; cover invalid ends without scanning.

### M3: Pin command behavior across engines and document the fix (~100 LOC)

**Duration:** 2 hours. **LOC split:** 85 command regression + 15 documentation.
**Dependencies:** M2.
**Files and examples:** Create `cmd/ailang/test_strip_braces_test.go` using buildAilang/runAilangBin and temporary modules; reuse the M2 fixture. Update `changelogs/v0.32-current.md` under Unreleased; update the design completion notes. No additional public demo is needed for a harness bug.

**Tasks:** Implement the milestone behavior with failing regression cases first, then run the relevant package tests.

**Acceptance criteria:**

- [ ] A freshly built CLI checks the new fixture and runs three trigger tests plus a clean sibling with their own expected outcomes.
- [ ] Default, --bytecode and --strict-bytecode agree on a VM-compatible fixture; no shared-compile failure or harness-bug notice appears.
- [ ] A deliberately false sibling is reported as an assertion outcome, never a module parse failure; property-batch regression coverage also passes.
- [ ] make test-core, go test ./internal/testing ./internal/parser ./internal/format ./cmd/ailang, make lint and make check-boundaries pass.
- [ ] changelogs/v0.32-current.md Unreleased documents the fix and the escaped-quote distinction; design status is updated on completion, moved to implemented on ship.

**Risk and mitigation:** Engine differences or stale binaries: build once with the existing CLI test helpers; compare reports after removing duration fields and use expressions already supported by the VM.

## Day-by-day Execution

| Day | Work | Budget |
|---|---|---|
| 1 | M1: AST/parser positions and parser regressions; begin M2 strip replacement and fixtures | 2.5h + 2.5h |
| 2 | Finish M2 exact-range/line-map cases; M3 engine and CLI regression, changelog and gates; contingency | 1h + 2h + 2h |

Use the existing Go tests and build helpers; no new automation script is required. Preserve the existing declaration-range, named-batch, property-batch, engine-parity and bytecode-flag fixtures. Run focused package tests while implementing, then the combined final gates once. If CI or tooling fails, record the concrete failure rather than treating a skipped gate as passing.

## AILANG Syntax and Fixture Checklist

AILANG prompt version loaded: v0.16.6 (`ailang prompt --version-active`). Loaded the installed CLI's full teaching prompt with `ailang prompt`; this prompt version differs from the binary release version. Reload the repository-built CLI's prompt before writing fixtures, and validate new modules with `ailang check` before running them. No new AILANG source was authored in this planning stage.

| Module | Contracts | Effects | Inline tests |
|---|---|---|---|
| `internal/testing/testdata/strip/brace_literals.ail` | skip: source stripping and parser positions are checked in Go, no public helper contract | include: any helper uses `pure func ... -> bool ! {}`; property predicates stay pure | skip: named test/property declarations are the syntax under regression; separate function inline tests would not exercise stripping |
| Temporary CLI regression modules in `test_strip_braces_test.go` | skip: command outcomes are asserted by the Go integration test | include: pure bool expressions and any helper explicitly use `! {}`; no capabilities required | skip: named tests are the regression target; Go asserts each outcome across engines |

## Success Metrics and Verification

- Every trigger test executes its own body, with a clean sibling unaffected and an intentional false sibling reported normally.
- Test and property declarations get authoritative closing positions; both strip paths remove exact ranges and preserve line mappings.
- Existing strip/batch/helper-retention/parity/formatter/CLI cases pass without fixture changes.
- `make test-core` and `go test ./internal/testing ./internal/parser ./internal/format ./cmd/ailang` pass.
- `make lint` and `make check-boundaries` pass; touched Go files are gofmt-clean.
- Inspect `testAndPropertySkipRanges` to confirm character-level brace counting and the fallback scanner are absent.
- New fixtures pass repository-built `ailang check`; the CLI regression covers evaluator, bytecode and strict bytecode.
- Changelog names braces in literals/comments/names and explains why escaped quotes alone were not the reproduced cause.

## Assumptions, Deferred Work and Handoff

The external stapledons-godot checkout is not available; local fixtures pin the reported failure class. Its 29-test suite can be an additional downstream smoke check when available, but is not a local acceptance dependency. No issue number was provided for the original bug; #1739 is the merged design PR, so `github_issues` remains empty instead of inventing a bug link.

Parser end positions are the approved systemic fix; no second lexer, lexical fallback, full AST-span sweep, release bump or unrelated harness redesign belongs in this sprint. Track progress in `.ailang/state/sprints/sprint_M-NAMEDTEST-STRIP-STRING-BLIND-BRACES.json`; all passes fields start null.

Coordinator handoff: return the plan and JSON artifact markers for review. Under the coordinator workflow, approval/merge of the sprint-plan PR triggers sprint-executor; this planning stage does not start implementation. Sprint-evaluator assesses the completed work against the criteria above.
