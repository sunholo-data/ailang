# M-PARSER-BLOCK-TRAILING-RECORD: Block Expression Greedily Consumes Trailing `{` as New Block Instead of Record Literal

**Status**: ✅ Implemented in v0.20.0 (commit 7afb69d2, sprint M-PARSER-BLOCK-TR, 2026-05-15)
**Target**: v0.20.0 (shipped alongside the LSP sprint; previously misfiled as v0.21.0)

> **Implementation note (2026-05-15):** The actual root cause was *not* in `parseBlockOrExpression`'s inner-loop dispatch (as the design doc hypothesised below). It was in **`parseRecordLiteral`'s IDENT-branch block parser** at [internal/parser/parser_literals.go:478](../../../internal/parser/parser_literals.go#L478). That code used `curTokenIs(RBRACE)` for loop termination, so after parsing an inner record literal (which leaves cur at the inner `}`), the loop incorrectly treated the inner `}` as the BLOCK's `}` and exited — leaving the outer `}` unconsumed. The if-then-else parser then failed to find `else`, producing the misleading `expected else, got }` error. Fix: rewrite the loop using peek-based termination, mirroring `parseBlockOrExpression`. ~15 LOC. The dispatch in `parseBlockOrExpression` itself was already correct — the bug lived in the OTHER LBRACE entry point (`parseRecordLiteral`, the prefix function). Sprint plan: [m-parser-block-trailing-record-literal-sprint-plan.md](m-parser-block-trailing-record-literal-sprint-plan.md).
**Priority**: P2 (Medium — clean workaround exists, but trips real codebases on idiomatic record-returning blocks)
**Estimated**: 3–4 hours
**Dependencies**: None. Same bug class as the implemented [M-DX16](../../implemented/v0_6_1/m-dx16-inline-record-match-arms.md), but in a different parser path.
**Discovered**: 2026-05-15 by `cli` agent in `sunholo/ailang-parse` (msg `c8813647`)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No semantic change; parser fix only. |
| A2: Replayability | 0 | No trace surface change. |
| A3: Effect Legibility | 0 | No effect handling change. |
| A4: Explicit Authority | 0 | No capability change. |
| A5: Bounded Verification | 0 | No verification surface change. |
| A6: Safe Concurrency | 0 | No concurrency change. |
| A7: Machines First | +1 | Removes a pattern AIs frequently emit and have to manually rewrite into helper functions. |
| A8: Minimal Syntax | +1 | No new syntax — makes existing syntax parse where humans/AIs already expect it to. |
| A9: Cost Visibility | 0 | No cost model change. |
| A10: Composability | +1 | Block expressions and record literals now compose as expected (just like match arms after M-DX16). |
| A11: Structured Failure | +1 | Eliminates a confusing `expected next token to be else, got } instead` error that points at the wrong line. |
| A12: System Boundary | 0 | No FFI / no boundary change. |

**Net Score: +4** → **Decision: Move forward.**

### Hard Violation Check

- [x] A1 (Determinism): no nondeterminism introduced
- [x] A3 (Effects): no effect change
- [x] A4 (Authority): no ambient access
- [x] A7 (Machines First): improves machine analysis by removing a syntactic pothole

## Problem Statement

A block expression whose final position is a record literal preceded by one or more `;`-separated statements fails to parse. The parser treats the trailing `{` as opening a *new* block expression instead of a record literal, then errors with a misleading `expected next token to be else, got } instead`.

**Minimal repro** ([reproduced on v0.19.2 — binary md5 `af124e1f30221cb274869cbcc414ab0a`, commit `24fd623d`](../../../CHANGELOG.md)):

```ailang
module test/bug

import std/io (println)

export func main() -> int ! {IO} {
  let parsed = if true
    then {
      println("a");
      {a: 1, b: 0}
    }
    else {
      println("b");
      {a: 2, b: 1}
    }
  in parsed.a
}
```

```
PAR_UNEXPECTED_TOKEN at bug.ail:11:5: expected next token to be else, got } instead
PAR_NO_PREFIX_PARSE at bug.ail:11:5: unexpected token in expression: }
```

**Trigger isolated by triangulation:**

| Pattern | Result |
|---------|--------|
| `{ println("a"); {a: 1, b: 0} }` | ✗ fails |
| `{ {a: 1, b: 0} }` (no leading stmt) | ✓ ok |
| `{ println("a"); 1 }` (non-record final expr) | ✓ ok |
| Same broken pattern at top level (not inside `let`) | ✗ fails |

The trigger is **not** specific to `let RHS` or to `if`-then/else; it is any `parseBlockOrExpression` call where the block contains `;`-separated expressions and the final expression is a record literal `{IDENT: ...}`.

**Original report:**
- Sender: `cli` (sunholo/ailang-parse), msg `c8813647-3852-4745-a8a4-b037f0504246`
- Reporter narrowed the trigger to "if as RHS of let" but also observed that "multi-statement if-then-else at top-level works fine in the same module". The wider triangulation in this design doc shows the actual trigger is broader than that.

**Impact:**
- Idiomatic AILANG — "do an effect, then return a record" inside a block — does not parse.
- Common shape in real code: `if ext == ".docx" then { logChoice(); {fmt: "docx", needsAi: true} } else { ...; {fmt: "txt", needsAi: false} }` (the actual `docparse/main.ail` pattern that prompted the report).
- Workaround (extract each branch to a named helper, or `let r = {a: 1} in r`) is mechanical but adds boilerplate AIs have to learn to emit defensively.

## Root Cause

In [internal/parser/parser_expr.go:325-396](../../../internal/parser/parser_expr.go#L325-L396), `parseBlockOrExpression`:

1. Lines 346–356 dispatch to `parseRecordLiteralContent` / `parseRecordUpdateContent` only **when the cursor is at the very first token after `{`**. This handles the leading-record case `{a: 1, b: 0}` correctly.
2. Lines 358–374 then enter the `;`-separated expression loop:
   ```go
   exprs := []ast.Expr{}
   exprs = append(exprs, p.parseExpression(LOWEST))
   for p.peekTokenIs(lexer.SEMICOLON) {
       p.nextToken() // SEMICOLON
       if p.peekTokenIs(lexer.RBRACE) { break }
       p.nextToken() // past SEMICOLON
       exprs = append(exprs, p.parseExpression(LOWEST))
   }
   ```
3. When the next expression is `{a: 1, b: 0}`, control passes to `p.parseExpression(LOWEST)`, which dispatches the LBRACE prefix to `parseRecordLiteral` ([parser_literals.go:282](../../../internal/parser/parser_literals.go#L282)) — which itself does have IDENT-COLON lookahead for record vs block.

Empirically the parse still fails, which means one of:

- (a) `parseRecordLiteral` is not the registered prefix in this code path (e.g. it dispatches to the old block parser instead, or is shadowed); or
- (b) `parseRecordLiteral` consumes the record correctly but the *outer* `parseBlockOrExpression` mishandles the resulting cursor position and walks past the closing `}` of the block; or
- (c) The error reporting is misleading and the actual failure is upstream — the surrounding `if … then BLOCK else …` continuation expects to find `else` but the block parser has not yet returned.

The error `expected next token to be else, got } instead` strongly suggests **(c)**: the parser believes the `then`-branch already ended at the inner `{` of the record literal and now expects `else`, but the buffer still has the record's closing `}`. That means `parseExpression(LOWEST)` at line 374, when applied to a `{`-starting expression *inside* a `;`-separated block, is going through a different LBRACE handler than `parseRecordLiteral` — most likely `parseBlockOrExpression` recursively, which then parses `{a: 1, b: 0}` and immediately returns, leaving the outer block looking for its `}`.

This needs to be confirmed under `DEBUG_PARSER=1` during implementation, but the failure shape is consistent with the parser dispatching the inner `{` through `parseBlockOrExpression` (which fails the IDENT-COLON check on entry at line 346 because `curToken` is still the outer block's content) rather than through the LBRACE prefix.

This is the **same bug class as [M-DX16](../../implemented/v0_6_1/m-dx16-inline-record-match-arms.md)** — that fix made `parseCase` go through the prefix parser dispatch instead of unconditionally calling `parseBlockOrExpression`. The same architectural fix needs to be applied here, in the inner-expression position of the block parser.

## Goals

**Primary Goal:** A block expression `{ stmt; ...; finalExpr }` parses `finalExpr` through the normal expression dispatch, so a `{IDENT: ...}` record literal is recognised regardless of whether it is the only expression or the last expression in a `;`-separated block.

**Success Metrics:**

- The minimal repro above parses and runs without error.
- All four reporter shapes from msg `c8813647` parse:
  - `let parsed = if cond then { stmt; record } else { stmt; record } in parsed.f`
  - The same shape at top level
  - The same shape in a `match` arm (regression check against M-DX16)
  - The same shape inside another block (`{ s1; { s2; rec } }`)
- No regression in existing record vs block disambiguation tests, in particular [internal/parser/record_match_arms_test.go](../../../internal/parser/record_match_arms_test.go).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Fix in `parseBlockOrExpression`'s inner loop vs. fix in the LBRACE prefix dispatch | Determines whether the fix is local to the block parser (small surface) or refactors the LBRACE entry points to a single function (M-DX16-style consolidation, larger surface but eliminates the bug class). | human | design | low vs. med |
| Whether to also collapse `parseBlockOrExpression` and `parseRecordLiteral` into one entry function | M-DX16 already noted "two LBRACE entry points" as a smell. A consolidation would prevent the next variant of this bug. But it touches function bodies, match arms, lambdas — wider blast radius. | human | design | high |
| Test corpus: minimal repros only, or sweep every `{` context in the grammar | Sweeping prevents regressions in lambda bodies, function bodies, etc.; takes longer to write. | agent | implementation | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **Confirm the dispatch path** under `DEBUG_PARSER=1` for the minimal repro. Specifically: which function handles the inner `{` of the trailing record literal? `parseRecordLiteral` (the LBRACE prefix) or `parseBlockOrExpression` (recursively)?
- [ ] **Pick fix scope**: local fix in `parseBlockOrExpression`'s inner-expression call, or consolidate the two LBRACE entry points (M-DX16 follow-up). Recommend local fix for v0.21.0 + a follow-up doc for the consolidation.
- [ ] **Lock the test corpus** before changing parser code. Add the four success-metric shapes as table-driven tests in `record_match_arms_test.go` (or a new `record_block_arms_test.go`) so the fix is regression-protected.

## Solution Design

### Overview

Locate the LBRACE dispatch inside `parseBlockOrExpression`'s `;`-separated loop. Ensure the inner expression goes through the same record-vs-block disambiguation as the leading-expression case (lines 346–356). The cleanest implementation is to factor out the IDENT-COLON / IDENT-PIPE lookahead and call it from both positions.

### Architecture

**Components:**

1. **[internal/parser/parser_expr.go](../../../internal/parser/parser_expr.go)** — refactor `parseBlockOrExpression`:
   - Extract the leading record/record-update detection (lines 346–356) into a helper `tryParseRecordOrUpdateAtBrace(startPos) ast.Expr` that returns nil if the current `{`-context isn't a record.
   - Call that helper from **both** the leading position and the post-semicolon position in the inner loop.
   - Alternative (smaller diff): inside the inner loop at line 374, before calling `parseExpression(LOWEST)`, check if `curToken == LBRACE` and apply the same record dispatch.

2. **[internal/parser/record_match_arms_test.go](../../../internal/parser/record_match_arms_test.go) or new `record_block_arms_test.go`** — add table-driven tests for:
   - Block as `if`-then-else branch with trailing record + leading stmt
   - Same shape at top-level statement position
   - Nested blocks `{ s; { s; rec } }`
   - Function body `{ s; rec }` (lambdas and `func`)
   - Match arm with `{ s; rec }` body (regression for M-DX16)

3. **[examples/parser_block_trailing_record.ail](../../../examples/parser_block_trailing_record.ail)** — single executable example demonstrating the now-supported pattern.

### Out of scope for v1

- Consolidating `parseBlockOrExpression` and `parseRecordLiteral` into one entry function. File a follow-up `M-PARSER-LBRACE-CONSOLIDATION` if/when a third variant of this bug appears.
- Reworking the misleading `expected next token to be else, got } instead` error. The diagnostic improves automatically once the parse succeeds; targeted error-message work belongs in a separate DX doc.
- Anything in the `requires` / `ensures` / contract parser. Contract blocks have their own `{ ... }` parser ([parser_contracts.go](../../../internal/parser/parser_contracts.go)) and are not affected by this bug.

## Implementation Plan

### Phase 1: Lock Tests First (1 hour)

- [ ] Capture the four success-metric shapes as failing tests in `internal/parser/record_block_arms_test.go`.
- [ ] Run `make test` — all new tests should fail with the same `PAR_UNEXPECTED_TOKEN` shape, confirming the bug surface.
- [ ] Run with `DEBUG_PARSER=1` on one minimal repro and capture the dispatch trace in a comment in the test file.

### Phase 2: Fix the Dispatch (1 hour)

- [ ] Refactor `parseBlockOrExpression` to call the record/record-update detection from both the leading and inner-loop positions.
- [ ] Run `make test` — the new tests pass; existing parser tests still pass.

### Phase 3: Example + Docs (30 min)

- [ ] Add `examples/parser_block_trailing_record.ail` with the now-working shape.
- [ ] Add an entry to CHANGELOG.md under "Fixed" for v0.21.0.
- [ ] No website doc change needed (this is a "now matches the docs" bugfix, not a new feature).

### Phase 4: Validate Against Original Reporter (30 min)

- [ ] Re-run `ailang-parse`'s `docparse/main.ail` against the new binary; confirm the workaround branch can be removed.
- [ ] Send `ailang messages` reply to the `cli` sender of `c8813647` confirming the fix and the version it landed in.

## Test Cases

Add to `internal/parser/record_block_arms_test.go`:

```go
func TestBlockTrailingRecord_LetIfRhs(t *testing.T) {
    src := `
module t
import std/io (println)
export func main() -> int ! {IO} {
  let parsed = if true
    then { println("a"); {a: 1, b: 0} }
    else { println("b"); {a: 2, b: 1} }
  in parsed.a
}
`
    requireParses(t, src)
}

func TestBlockTrailingRecord_TopLevel(t *testing.T) { ... }
func TestBlockTrailingRecord_Nested(t *testing.T)   { ... }
func TestBlockTrailingRecord_FuncBody(t *testing.T) { ... }
func TestBlockTrailingRecord_MatchArm(t *testing.T) { ... }  // M-DX16 regression
```

## Success Criteria

- [ ] All five test shapes parse successfully.
- [ ] No regression in `internal/parser/record_match_arms_test.go` (M-DX16) or any other parser test.
- [ ] `examples/parser_block_trailing_record.ail` runs and produces the expected record value.
- [ ] Reporter's original `docparse/main.ail` pattern parses without the helper-extraction workaround.
- [ ] CHANGELOG.md entry under v0.21.0 "Fixed".

## Files to Modify

| File | Changes | LOC |
|------|---------|-----|
| `internal/parser/parser_expr.go` | Refactor record/block dispatch in `parseBlockOrExpression` | ~30 |
| `internal/parser/record_block_arms_test.go` | New table-driven tests (5 shapes) | ~120 |
| `examples/parser_block_trailing_record.ail` | New example | ~25 |
| `CHANGELOG.md` | "Fixed" entry under v0.21.0 | ~5 |
| **Total** | | ~180 |

## Related Issues

- **Inbox message**: `c8813647-3852-4745-a8a4-b037f0504246` (ailang-core inbox, sender `cli` from `sunholo/ailang-parse`)
- **Earlier message** (same content): `msg_20260515_140511_7d6a84b2`

## Related Documents

- [M-DX16: Inline Record Literals in Match Arms](../../implemented/v0_6_1/m-dx16-inline-record-match-arms.md) — same bug class, fixed for `match` arms in v0.6.1. This doc applies the same architectural insight to block expressions.

## Alternatives Considered

### Alternative 1: Document as a known limitation, leave the parser alone

Add the trigger pattern to `docs/LIMITATIONS.md` and tell users to extract helpers.

**Pros:** zero parser risk.
**Cons:** the workaround is mechanical and AIs trained on the corpus will keep regenerating the broken shape; M-DX16 already established the precedent that this class of bug should be fixed.

**Rejected.**

### Alternative 2: Consolidate `parseBlockOrExpression` and `parseRecordLiteral` now

Replace both LBRACE handlers with a single function that handles records, record updates, blocks, and unit (`{}`) at every entry point.

**Pros:** eliminates the bug class entirely; simpler grammar.
**Cons:** wider blast radius (function bodies, match arms, lambdas, top-level expressions); needs its own design pass; risk of subtle behavioural changes.

**Deferred** to a follow-up doc if a third variant surfaces.

## Notes

- The reporter's hypothesis ("parser greedily consumes the trailing record literal inside the then-block") was correct in spirit. The actual mechanism is more general than they observed: it triggers in any block context where a `;`-separated statement precedes the trailing record.
- The misleading error message (`expected else, got }`) is what made the reporter assume this was an `if`-as-`let`-RHS issue. The error reporting will improve automatically once the parse succeeds.
- Same binary md5 across reporter and current `dev` HEAD (`af124e1f30221cb274869cbcc414ab0a` at commit `24fd623d`), so the bug is current and reproducible without any version drift.
