# M-PARSER-BLOCK-TRAILING-RECORD: Sprint Plan

**Sprint ID**: M-PARSER-BLOCK-TR
**Target Version**: v0.21.0
**Estimated Duration**: ~3 hours
**Estimated LOC**: ~180 (60 prod + 120 tests/example/docs)
**Risk Level**: low
**Created**: 2026-05-15

## Sprint Goal

Fix the parser bug where `{ stmt; ...; trailingRecordLiteral }` block expressions fail with `PAR_UNEXPECTED_TOKEN`. The parser currently treats the trailing `{` as opening a new nested block instead of a record literal. Same bug class as the implemented [M-DX16](../../implemented/v0_6_1/m-dx16-inline-record-match-arms.md) (which fixed match arms), but in `parseBlockOrExpression`'s inner-expression position.

Reported by `cli` agent in `sunholo/ailang-parse` (inbox `c8813647`). Design doc: [m-parser-block-trailing-record-literal.md](m-parser-block-trailing-record-literal.md).

## Background & Recent Velocity

- Today's M-DX26-P5 sprint shipped Phases 5 + 5.1 + 5.2 in ~5 hours, ~570 LOC. Three tight focused commits, all green.
- This sprint is comparable scale: ~180 LOC, ~3 hours. One commit per milestone.
- Risk profile: low. The fix mirrors a known-working pattern (M-DX16). No type system, no runtime semantics — pure parser dispatch routing.

## Out of Scope (Tracked Separately)

- ❌ **Consolidating `parseBlockOrExpression` and `parseRecordLiteral` into one entry function.** Design doc Alternative 2. Defer to a follow-up if a third variant of the bug appears.
- ❌ **Improving the misleading `expected else, got }` error message.** Diagnostic improves automatically once the parse succeeds.
- ❌ **`requires`/`ensures` contract block parsing.** Has its own `{ ... }` parser, not affected by this bug.

## Milestones

### M0: Lock Failing Tests (~45 min, ~80 LOC)

Capture the bug as failing tests **before** touching production code. The design doc identifies five shapes that should all parse — write all five as failing tests first.

**Files to create:**

| File | Change | LOC |
|------|--------|-----|
| `internal/parser/record_block_arms_test.go` | New file: 5 table-driven failing test cases (let-RHS, top-level, nested, function body, match-arm regression) | ~80 |

**Acceptance criteria:**
- [ ] New test file `internal/parser/record_block_arms_test.go` exists with 5 distinct test functions covering: block as `if`-then-else branch with leading stmt + trailing record; same shape at top level (statement position in function body); nested blocks `{ s; { s; rec } }`; function body `{ stmt; rec }`; match arm `{ stmt; rec }` (M-DX16 regression check).
- [ ] Running `go test ./internal/parser/ -run TestBlockTrailingRecord -v` shows all 5 tests **failing** with `PAR_UNEXPECTED_TOKEN` errors (this is the "before" state).
- [ ] No production code changed yet — this milestone is purely the failing test corpus.

**Risks:**
- Tests may need slight tweaks if AILANG syntax around `if/then/else` or `match` differs from what the design doc shows. Verify each test's source against `ailang prompt` before declaring shape correct.

### M1: Fix the Dispatch in `parseBlockOrExpression` (~1 hour, ~50 LOC)

Apply the same architectural insight from M-DX16: `IDENT COLON` / `IDENT PIPE` lookahead for record-vs-block disambiguation must run at the **inner-expression position** in `parseBlockOrExpression`'s `;`-separated loop, not just at the leading position.

**Files to modify:**

| File | Change | LOC |
|------|--------|-----|
| `internal/parser/parser_expr.go` | Refactor `parseBlockOrExpression`: extract leading record/record-update detection (lines 346–356) into a helper, call it from both leading position and inner-loop position. Or simpler: in the inner loop at line 374, check if `curToken == LBRACE` and dispatch the same way. | ~50 |

**Acceptance criteria:**
- [ ] All 5 tests from M0 pass.
- [ ] No regression in `internal/parser/record_match_arms_test.go` (the M-DX16 test corpus).
- [ ] No regression in any other parser tests (`go test ./internal/parser/...`).
- [ ] `make test` passes; `make lint` clean.
- [ ] DEBUG_PARSER=1 trace on the original repro shows the inner LBRACE going through the record-literal path, not recursing into a nested block.

**Risks:**
- Risk of breaking unrelated parser cases that legitimately use `{ stmt; { newBlock } }`. Mitigation: M0's nested-blocks test pins this behavior — if the fix breaks legitimate nested blocks, that test fails immediately.
- The fix path may require choosing between two approaches: refactor into a shared helper, or duplicate the dispatch inline at the inner position. Pick whichever leaves `parseBlockOrExpression` more readable; both work.

### M2: Acceptance Test + Reporter Validation (~30 min, ~40 LOC)

Promote the bug shape to a runnable example and validate against the original reporter's pattern.

**Files to create/modify:**

| File | Change | LOC |
|------|--------|-----|
| `examples/runnable/parser_block_trailing_record.ail` | New: minimal working example demonstrating the now-supported pattern (the exact shape from inbox `c8813647`) | ~25 |
| `examples/manifest.json` | Add the new example | ~5 |
| `CHANGELOG.md` (changelogs/v0.10-current.md) | "Fixed" entry under `[Unreleased]` referencing M-PARSER-BLOCK-TR + inbox `c8813647` | ~10 |

**Acceptance criteria:**
- [ ] `examples/runnable/parser_block_trailing_record.ail` runs cleanly via `ailang run` and produces the expected record value.
- [ ] The reporter's exact `docparse/main.ail` shape (from msg `c8813647`) parses without the helper-extraction workaround. Verify by reproducing the minimal repro from the design doc and confirming it now parses.
- [ ] CHANGELOG entry under v0.21.0 `[Unreleased]` references M-PARSER-BLOCK-TR, inbox `c8813647`, and the new example.

**Risks:** None significant.

### M3: Inbox Reply + Doc Cleanup (~15 min, ~5 LOC)

Notify the reporter and finalize design doc tracking.

**Files to modify:**

| File | Change | LOC |
|------|--------|-----|
| Inbox reply | Send `ailang messages` reply to `cli` (sender of `c8813647`) confirming the fix | (no LOC) |
| `design_docs/planned/v0_21_0/m-parser-block-trailing-record-literal.md` | Add a brief "Implemented" footer noting the commit + version | ~5 |

**Acceptance criteria:**
- [ ] Reply sent to `cli@sunholo/ailang-parse` confirming the fix and recommending they remove the helper-extraction workaround in `docparse/main.ail`.
- [ ] Design doc has an "Implemented" note at the top with the commit SHA + target version.
- [ ] (Optional) Move design doc + sprint plan to `design_docs/implemented/v0_21_0/` if all work is complete.

## Implementation Order (TDD)

1. **M0 (45 min)**: Write failing tests first. Run `go test ./internal/parser/ -run TestBlockTrailingRecord` → all 5 fail with `PAR_UNEXPECTED_TOKEN`. Commit.
2. **M1 (1h)**: Refactor `parseBlockOrExpression`. Run tests → all 5 pass. No regressions. Commit.
3. **M2 (30 min)**: Add example + CHANGELOG. Verify with `ailang run`. Commit.
4. **M3 (15 min)**: Inbox reply + finalize doc. Commit.

## Success Metrics

- [ ] All 5 new tests in `record_block_arms_test.go` pass.
- [ ] `examples/runnable/parser_block_trailing_record.ail` runs cleanly.
- [ ] No regression in `record_match_arms_test.go` (M-DX16 still works) or any other parser test.
- [ ] `make test` clean; `make lint` clean.
- [ ] CHANGELOG entry under v0.21.0 `[Unreleased]` "Fixed".
- [ ] Inbox reply sent to `cli` reporter.

## Dependencies

- None. The fix is local to `internal/parser/parser_expr.go` and a new test file.

## Open Questions (resolved before sprint start)

- ✅ **Refactor vs inline dispatch?** Decide during M1 based on which leaves `parseBlockOrExpression` more readable. Both achieve the same correctness.
- ✅ **Consolidate the two LBRACE handlers (`parseBlockOrExpression` + `parseRecordLiteral`)?** Out of scope for this sprint. File a follow-up `M-PARSER-LBRACE-CONSOLIDATION` if a third variant surfaces.
- ✅ **Should we also improve the misleading `expected else, got }` error message?** No — it's a symptom of the bug; diagnostic improves automatically once parse succeeds.

## Related

- **Design doc**: [M-PARSER-BLOCK-TRAILING-RECORD](m-parser-block-trailing-record-literal.md) — complete root-cause analysis + alternatives.
- **Same bug class, fixed for match arms**: [M-DX16](../../implemented/v0_6_1/m-dx16-inline-record-match-arms.md) (v0.6.1).
- **Inbox message**: `c8813647-3852-4745-a8a4-b037f0504246` from `cli@sunholo/ailang-parse`.
