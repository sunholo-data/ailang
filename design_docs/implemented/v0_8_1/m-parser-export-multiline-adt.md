# M-PARSER-EXPORT: Single-Constructor ADT Cursor Bug Drops `export` Flag

**Status**: Planned
**Target**: v0.8.1
**Priority**: P0 (High) — blocks all stdlib type exports
**Estimated**: 2 hours (1h fix + 1h tests)
**Dependencies**: None
**Bug Report**: Discovered during M-STREAM-PHASE2-DX sprint

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No impact on determinism |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | No impact on effects |
| A4: Explicit Authority | 0 | No impact on capabilities |
| A5: Bounded Verification | +1 | Fixes module-level type checking |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Typed ADT exports enable machine-verifiable module interfaces |
| A8: Minimal Syntax | 0 | No syntax changes — existing `export type` syntax just needs to work |
| A9: Cost Visibility | 0 | No impact |
| A10: Composability | +1 | Cross-module ADT composition currently broken |
| A11: Structured Failure | 0 | No impact |
| A12: System Boundary | 0 | No impact |

**Net Score: +3** → **Decision: Proceed**

## Problem Statement

**Root cause**: The parser's `parseTypeDeclBody()` in `internal/parser/parser_type.go` has an off-by-one cursor position bug on the single-constructor path. After parsing `StreamConn(int)`, it calls `p.nextToken()` to consume RPAREN (line 390), leaving the cursor **past** RPAREN — one token ahead of where it should be. The main parsing loop in `parser_file.go:81` then calls `p.nextToken()` again, skipping the `export` keyword of the next declaration.

**Symptom**: Any `export type` declaration that follows a single-constructor ADT loses its `Exported=true` flag. The second type declaration is parsed as non-exported.

**Minimal reproduction**:
```ailang
export type Wrapper = Wrap(int)      -- single-constructor ADT

export type MyEvent =                 -- ← Exported=false (BUG!)
  | Foo(string)
  | Bar(int)
```

`MyEvent` is parsed with `Exported=false` because:
1. `Wrapper = Wrap(int)` parsing ends with cursor at the `export` token (one past RPAREN)
2. Main loop calls `p.nextToken()` → cursor advances to `type`
3. `parseTopLevelDecl` sees `type` (not `export`), calls `parseTypeDeclaration(false)`

**Why `parseVariant()` is correct but the single-constructor path is not**:
- `parseVariant()` (line 506): "DON'T consume RPAREN - leave it for the caller to handle"
- Single-constructor path (line 390): `p.nextToken() // consume RPAREN` ← **violates convention**

**Impact**:
- `std/stream.ail`: `StreamEvent` type cannot be imported (IMP010 error)
- `std/option.ail`: Works because `Option[a] = Some(a) | None` is a multi-constructor ADT
- Any stdlib module with `export type Foo = Bar(int)` followed by more type declarations is affected
- Blocks the M-STREAM-PHASE2-DX sprint's final milestone

## Goals

**Primary Goal:** Fix cursor positioning so single-constructor ADTs leave the cursor AT the last consumed token (RPAREN), matching the convention used by `parseVariant()`.

**Success Metrics:**
- All `export type` declarations retain `Exported=true` regardless of preceding declarations
- `import std/stream (StreamEvent, Message, Closed)` resolves correctly
- `ailang check examples/runnable/stream_websocket.ail` passes
- No regression in existing 438 parser tests

## Solution Design

### Overview

Remove the extra `p.nextToken()` at line 390 in `parseTypeDeclBody()` that consumes RPAREN on the single-constructor path. This aligns it with `parseVariant()`'s convention of leaving cursor AT RPAREN.

### The Fix (3 lines changed)

**File: `internal/parser/parser_type.go`**

Current code (lines 387-391):
```go
if !p.curTokenIs(lexer.RPAREN) {
    p.reportExpected(lexer.RPAREN, "Add ')' to close constructor fields")
} else {
    p.nextToken() // consume RPAREN  ← BUG: over-advances
}
```

Fixed code:
```go
if !p.curTokenIs(lexer.RPAREN) {
    p.reportExpected(lexer.RPAREN, "Add ')' to close constructor fields")
}
// DON'T consume RPAREN — leave cursor AT last token, matching parseVariant() convention
// The main parsing loop (parser_file.go:81) calls nextToken() to advance between declarations
```

### Systemic Audit

Per CLAUDE.md Section 6 (Systemic Fixes), searched for similar cursor bugs:

1. **`parseVariant()` (line 506)**: Already correct — explicit "DON'T consume RPAREN" comment
2. **`parseConstructorField()` (line 520+)**: Returns after parsing type expression — correct
3. **Nullary constructors** (line 401-409 leadingPipe path): `p.nextToken()` after name — needs review but different path (no RPAREN involved)
4. **Sum type with first constructor having fields** (lines 369-396): This IS the buggy path

No other instances of the same bug pattern found. The fix is isolated to the single-constructor-with-fields path.

### Files to Modify

**Modified files:**
- `internal/parser/parser_type.go` — Remove `p.nextToken()` at line 390 (~3 LOC change)
- `internal/parser/type_test.go` — Add regression test (~30 LOC)

**No new files needed.**

## Examples

### Before (broken):
```ailang
-- std/stream.ail
export type StreamConn = StreamConn(int)    -- Exported=true ✓

export type StreamEvent =                    -- Exported=false ✗ (BUG)
  | Message(string)
  | Closed(int, string)
  | StreamError(StreamErrorKind)

-- Importing module:
import std/stream (StreamEvent)
-- Error: IMP010: symbol 'StreamEvent' not exported by 'std/stream'
```

### After (fixed):
```ailang
-- std/stream.ail
export type StreamConn = StreamConn(int)    -- Exported=true ✓

export type StreamEvent =                    -- Exported=true ✓ (FIXED)
  | Message(string)
  | Closed(int, string)
  | StreamError(StreamErrorKind)

-- Importing module:
import std/stream (StreamEvent, Message, Closed, StreamError)
-- ✓ No errors found!
```

## Success Criteria

- [ ] `export type Wrapper = Wrap(int)` followed by `export type Event = | A | B` both have `Exported=true`
- [ ] `ailang check examples/runnable/stream_websocket.ail` passes
- [ ] `ailang check examples/runnable/stream_sse.ail` passes
- [ ] All existing parser tests pass (438+ tests)
- [ ] `make test` passes
- [ ] `make lint` clean

## Testing Strategy

**Unit tests** (`internal/parser/type_test.go`):
- Test: single-constructor ADT followed by multi-line ADT — both exported
- Test: single-constructor ADT followed by exported function — function exported
- Test: multiple single-constructor ADTs in sequence — all exported

**Integration tests**:
- `ailang check` on `std/stream.ail` (which has this exact pattern)
- `ailang check` on example files importing stream types

**Regression guard**:
- Parser golden test with the minimal reproduction case

## Non-Goals

- Refactoring all cursor positioning in the parser — only fix this specific bug
- Changing the `parseVariant()` convention — it's already correct
- Adding general parser position validation framework

## Timeline

**Single session** (~2 hours):
- Fix the cursor bug (15 min)
- Write regression tests (30 min)
- Run full test suite (15 min)
- Verify stream examples (15 min)
- Complete M-STREAM-PHASE2-DX milestone M4 (30 min)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Removing nextToken breaks other single-constructor patterns | High | Run all 438 parser tests + stream examples before merge |
| Nullary constructors may have similar cursor issue | Medium | Included in systemic audit (line 401-409) — different code path, no RPAREN |

## Related Documents

- [design_docs/planned/v0_8_1/m-stream-phase2-dx.md](m-stream-phase2-dx.md) — Sprint that discovered this bug
- [design_docs/implemented/v0_7_0/m-gap2-lambda-arity-path-dependent-bug.md](../../implemented/v0_7_0/m-gap2-lambda-arity-path-dependent-bug.md) — Similar parser position bug

## References

- CLAUDE.md "Lexer/Parser Architecture — NEWLINE Tokens Don't Exist!" section
- CLAUDE.md parser convention: "Parser leaves cursor AT last token (not after)"
- `internal/parser/parser_type.go:506` — Correct convention: "DON'T consume RPAREN"

---

**Document created**: 2026-02-17
**Last updated**: 2026-02-17
