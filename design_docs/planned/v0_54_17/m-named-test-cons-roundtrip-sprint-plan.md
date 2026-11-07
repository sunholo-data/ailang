# Sprint Plan: M-NAMED-TEST-CONS-ROUNDTRIP

**Design:** [Approved design](m-named-test-cons-roundtrip.md)
**Status:** Planned; implementation starts after coordinator plan approval.
**Target:** v0.54.17 · **Issue:** #1915 · **Duration:** 1 day (6 hours work + 2 hours buffer)
**Risk:** Low implementation risk; medium regression risk in source reconstruction.

## Goal and scope

Restore cons expressions in named tests and forall predicates by printing the existing two-argument cons AST as parenthesised infix source. Reuse the formatter's shape check in the harness printer. No parser, type, evaluator or bytecode semantics changes.

The approved design audits all six PrintAILANGSource callers, other call shapes, cons patterns, interpolation, and the prior assert/strip failures. It passes the systemic-analysis gate. This sprint implements that unified printer fix and pins the affected paths with regression tests.

## Current status and estimates

Inspection confirms executor_helpers.go still prints all FuncCall nodes generically, while internal/format/expr.go already handles cons. No implementation work is present. The working tree was clean at planning start.

The seven-day velocity script finds only the design commit (1579ef50) in this shallow checkout; it cannot supply a credible LOC/day average. Historical changelog counts are not recent velocity. Use the approved design's task estimates, with 25% buffer: 6 hours work plus 2 hours reserve. Estimated changed/added code and fixture/documentation volume: 300 LOC, dominated by regression coverage. Target capacity is 300 LOC/day, an estimate rather than a measured rate.

## Registry reuse audit

Search performed: `ailang pkg search 'test source printer'` returned no packages (CLI also warned the binary may be stale). No package capability is being added: this is a private Go AST printer correction and tests of the host CLI. Each milestone chooses **none**, reusing local formatter and test infrastructure instead. No registry dependency or candidate inspection is applicable.

## Milestones

### M1: Printer fix and AST tripwire (~100 LOC)

**Estimate:** 10 implementation + 90 test LOC; 90 minutes. **Dependencies:** None.
**Files:** modify internal/testing/executor_helpers.go; create internal/testing/executor_cons_test.go.

Add an exact Identifier("::")/two-argument check in the FuncCall case. Recursively print both operands as `(left :: right)`. Retain generic printing for all other calls. Unit tests parse, fold, print and reparse, checking structural cons shape while ignoring source positions. Include nested right-associated chains, explicit nested operands, call arguments and wrong arity (0/1/3) to protect shape and precedence.

- [ ] Printed two-argument cons uses parenthesised infix; output reparses to equivalent cons structure with the same operands.
- [ ] Ordinary calls, zero-argument calls and wrong-arity cons retain their existing output.
- [ ] Existing assert tripwire and focused PrintAILANGSource tests pass.

**Risk:** AST position fields prevent bytewise comparison; compare semantic structure, not positions.

### M2: Named-test and property regression matrix (~120 LOC)

**Estimate:** 95 Go test + 25 fixture LOC; 90 minutes. **Dependencies:** M1.
**Files:** create internal/testing/named_cons_integration_test.go and internal/testing/testdata/named_batch/cons.ail; reuse named_batch_test.go and property_batch_test.go helpers without expanding those files.

Cover the reporter's call argument, extra parentheses and let-binding forms; nested cons; cons pattern; concat workaround; interpolation and ordinary calls. Exercise evaluator batch and per-body fallback explicitly. Add a cons forall predicate through shared batch and the standalone property compile path using existing APIs. Use a fixed seed. Verify successful bodies and the existing mixed fixture's pass/fail pattern.

- [ ] Every new fixture body passes; successful shared execution records no NamedBatchFailures and compiles once.
- [ ] Forced per-body and standalone property execution pass, independently of the shared batch path.
- [ ] Cons forall property passes all configured cases with a fixed seed.
- [ ] Existing mixed/assert/property fixtures retain expected verdicts.

**Risk:** accidental fallback can conceal a broken batch; assert compile count and failure metadata, not only total passes.

### M3: CLI engine parity and release evidence (~80 LOC)

**Estimate:** 70 Go test + 10 changelog LOC; 90 minutes, plus 30 minutes final checks. **Dependencies:** M1, M2.
**Files:** create cmd/ailang/test_cons_test.go using test_bytecode_test.go helpers; create changelogs/unreleased/2026-10-20-named-test-cons-roundtrip.md, opening with `### `.

Use the exact reporter module as a temporary r.ail fixture. Assert check succeeds, consRun returns 2, and test reports one passing test under evaluator, --bytecode and --strict-bytecode, with no batch-fallback notice. Bytecode strictness is distinct from --strict-syntax. Record the downstream restore instructions and strict-syntax canonical-form inconsistency as follow-up information in the changelog/implementation completion report; downstream edits and publishing are separate work after release.

- [ ] Exact reporter reproduction succeeds in all three execution modes with identical verdicts and no parse/fallback diagnostic.
- [ ] Focused cmd/ailang regression tests and make test-changed pass.
- [ ] make check-file-sizes and make check-changelog pass; changed Go files are gofmt clean.
- [ ] Release fragment explains named-test/property restoration; final report identifies downstream sim/similar_test.ail restore and deferred strict-syntax issue without closing historical #590/#1821.

**Risk:** installed binary may lag source; CLI tests must build the current checkout via buildAilang(t).

## Day 1 schedule

1. M1: 0–1.5h, red regression then printer correction.
2. M2: 1.5–3h, harness and property path coverage.
3. M3: 3–5h, CLI parity, fragment and focused checks.
4. 5–6h: review acceptance evidence and resolve failures.
5. 6–8h: reserved buffer. Each milestone fits a 60–90 minute executor work unit; final validation is a separate 30-minute step.

## AILANG syntax and fixture checklist

AILANG prompt version loaded: v0.16.7 (`ailang prompt` loaded; `ailang prompt --version-active` queried). Repository release is v0.54.16; teaching-prompt version is independently numbered.

| Module | Contracts | Effects | Inline tests |
|---|---|---|---|
| testdata/named_batch/cons.ail | skip: parser/harness regression fixture, no public API contract | skip: pure test bodies and pure helpers require no named effect row | skip: named tests and forall property are the regression target |
| temporary r.ail in CLI test | skip: preserve exact reporter reproduction | skip: exact reporter pure identity/consRun signatures | skip: preserve exact reporter named test |

Executor must load `ailang prompt` again before authoring fixtures and run `ailang check` on created modules. No showcase module or new language feature is introduced; the runnable regression fixture is the example deliverable.

## Validation and completion

Run focused Go tests for PrintAILANGSource, named batch/cons and forall properties in internal/testing, then the TestTestCommand cons/bytecode cases in cmd/ailang. Run `make test-changed` (changed packages plus core), `make check-file-sizes` and `make check-changelog`; do not gate execution on the full suite. CI owns the full suite. No stdlib changes require verify-stdlib; new regression fixtures are verified directly rather than a repository-wide example sweep. executor_helpers.go is 496 lines; existing named_batch_test.go is 268 and test_bytecode_test.go is 77; dedicated companion files above keep growth bounded below 800 lines.

Success means all explicit milestone criteria pass, semantic AST round-trip is pinned, three CLI engine verdicts agree, and no shared compile regression is hidden by fallback. Coverage target is every new printer branch and affected compile route, rather than a repository coverage percentage unsupported by local data.

## Handoff and dependencies

No unresolved design decisions block planning. JSON starts not_started, all passes fields null. Coordinator approval/merge of this sprint plan triggers sprint-executor; planning does not implement code or self-approve execution. Link development commits with refs #1915; reserve Fixes #1915 for the final implementation commit. #590 and #1821 are historical references, not issues to close in this sprint.
