# M-BUG-NESTED-RECORD-ANF: Fix Nested Record Literal ANF Verification

**Status**: Planned
**Target**: v0.5.0
**Priority**: P1 (Medium) - Affects common functional programming patterns
**Estimated**: 4-6 hours
**Dependencies**: None
**Reported by**: stapledons_voyage (agent inbox, 2025-11-30)

## AI-First Alignment Check

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Allows natural nested record construction |
| Preserve Semantic Clarity | + | +1 | Inline nested records are semantically clear |
| Increase Determinism | 0 | 0 | No change to determinism |
| Lower Token Cost | + | +1 | Avoids verbose intermediate let bindings |
| **Net Score** | | **+3** | **Decision: Move forward** |

## Problem Statement

Inline nested record literals fail ANF verification with error: "let bindings are not simple calls".

**Reproduction (2025-12-01):**
```ailang
-- FAILS: Inline nested record
let npc = { pos: { x: 10, y: 20 }, name: "guard" }
npc.pos.x
-- ERROR: ANF verification error: declaration 0: let binding 'npc' value: let bindings are not simple calls

-- WORKS: Intermediate let binding (workaround)
let pos = { x: 10, y: 20 }
let npc = { pos: pos, name: "guard" }
npc.pos.x  -- Returns: 10
```

**Root Cause Analysis:**

The issue is in the normalization pipeline, specifically how `normalizeLet` handles values that contain nested let bindings:

1. When normalizing `{ pos: { x: 10, y: 20 }, name: "guard" }`:
   - `normalizeRecord` calls `normalizeToAtomic` on each field value
   - For `{ x: 10, y: 20 }`, `normalize` returns a `*core.Record`
   - Since `*core.Record` is NOT atomic, `normalizeToAtomic` creates a binding:
     - `$tmp1 = Record{x: 10, y: 20}`
     - Returns `Var($tmp1)` as the atomic reference
   - `wrapWithBindings` wraps the outer record in a Let:
     - `Let $tmp1 = inner in Record{pos: $tmp1, name: "guard"}`

2. When normalizing `let npc = {...}`:
   - `normalizeLet` calls `normalize(let.Value)` which returns the Let from step 1
   - It creates: `Let npc = (Let $tmp1 = ... in ...) in body`

3. ANF verification fails because:
   - A Let value must be a "simple call" (see `verify.go:282-283`)
   - A Let expression is NOT a simple call
   - Error: "let bindings are not simple calls"

**Location:** `internal/elaborate/expressions.go:387` - `normalizeLet` function

**Impact:**
- Cannot use natural nested record construction
- Verbose workarounds required for common patterns
- Confusing error message for users
- Affects game/simulation code with nested state (like stapledons_voyage)

## Goals

**Primary Goal:** Allow inline nested record literals without ANF verification errors.

**Success Metrics:**
- `{ pos: { x: 10, y: 20 }, name: "guard" }` compiles and runs correctly
- Deeply nested records work (3+ levels)
- Field access chains work: `npc.pos.x`
- All existing tests pass (regression-free)
- Clear error messages for genuine ANF violations

## Solution Design

### Overview

Implement **let-flattening** during normalization. When a let binding's value is itself a Let expression, extract the inner bindings and flatten them:

**Before (current - fails):**
```
Let npc = (Let $tmp1 = inner in outer) in body
```

**After (fixed - flattened):**
```
Let $tmp1 = inner in Let npc = outer in body
```

### Architecture

**The fix needs to modify `normalizeLet` to:**
1. Normalize the value
2. If the value is a `*core.Let`, extract bindings recursively
3. Create flattened structure with inner bindings outermost

**Key insight:** This is a form of **let-floating** or **let-hoisting** - a standard transformation in ANF conversion.

### Implementation Plan

**Phase 1: Implement Let-Flattening Helper** (~2 hours)
- [ ] Create `extractLetBindings(expr CoreExpr) ([]binding, CoreExpr)` helper
- [ ] Recursively extract nested Let bindings into a flat list
- [ ] Return the innermost body and all extracted bindings

**Phase 2: Modify normalizeLet** (~2 hours)
- [ ] After normalizing value, call `extractLetBindings` if value is Let
- [ ] Use extracted bindings to build flattened structure
- [ ] Maintain correct scoping order (inner bindings first)
- [ ] Handle LetRec similarly if needed

**Phase 3: Testing** (~2 hours)
- [ ] Add test: simple nested record
- [ ] Add test: deeply nested record (3+ levels)
- [ ] Add test: nested record with field access
- [ ] Add test: record update with nested value
- [ ] Add test: list of nested records
- [ ] Verify all existing tests pass

### Files to Modify

**Modified files:**
- `internal/elaborate/expressions.go` - Modify `normalizeLet`, add helper (~30 LOC)
- `internal/elaborate/core.go` - Add `extractLetBindings` helper (~20 LOC)

**New files:**
- `examples/nested_records.ail` - Working examples (~25 LOC)

**Total estimated LOC:** ~75 LOC

## Examples

### Example 1: Basic Nested Record (Reported Case)

**Current (FAILS):**
```ailang
let npc = { pos: { x: 10, y: 20 }, name: "guard" }
npc.pos.x
-- ERROR: ANF verification error: let bindings are not simple calls
```

**After Fix:**
```ailang
let npc = { pos: { x: 10, y: 20 }, name: "guard" }
npc.pos.x  -- Returns: 10
```

**Internal transformation (conceptual):**
```
-- User writes:
let npc = { pos: { x: 10, y: 20 }, name: "guard" }

-- Elaborated (before flattening):
Let npc = (Let $tmp1 = {x: 10, y: 20} in {pos: $tmp1, name: "guard"})

-- After flattening:
Let $tmp1 = {x: 10, y: 20}
in Let npc = {pos: $tmp1, name: "guard"}
```

### Example 2: Deeply Nested Records

```ailang
let game = {
    player: {
        pos: { x: 0, y: 0 },
        stats: { hp: 100, mp: 50 }
    },
    level: 1
}
game.player.stats.hp  -- Returns: 100
```

### Example 3: Nested Record in Function

```ailang
pure func createNPC(x: int, y: int, name: string) -> {pos: {x: int, y: int}, name: string} {
    { pos: { x: x, y: y }, name: name }
}

let guard = createNPC(10, 20, "guard")
guard.pos.x  -- Returns: 10
```

### Example 4: Current Workaround (For Reference)

```ailang
-- WORKAROUND: Use intermediate let bindings
let pos = { x: 10, y: 20 }
let npc = { pos: pos, name: "guard" }
npc.pos.x  -- Returns: 10

-- WORKAROUND: Flatten manually
let x = 10
let y = 20
let npc = { pos: { x: x, y: y }, name: "guard" }
npc.pos.x  -- Returns: 10
```

## Success Criteria

- [ ] `{ pos: { x: 10, y: 20 }, name: "guard" }` compiles and runs
- [ ] Deeply nested records (3+ levels) work
- [ ] Field access chains work: `npc.pos.x`
- [ ] Nested records in lists work: `[{a: {b: 1}}, {a: {b: 2}}]`
- [ ] All existing tests pass
- [ ] `examples/nested_records.ail` added and verified
- [ ] CHANGELOG.md updated

## Testing Strategy

**Unit tests** (`internal/elaborate/expressions_test.go`):
```go
func TestNestedRecordNormalization(t *testing.T) {
    src := `let npc = { pos: { x: 10, y: 20 }, name: "guard" }`
    // Verify normalization succeeds
    // Verify flattened structure
}
```

**Integration tests:**
- Run `examples/nested_records.ail` through full pipeline
- Verify REPL handles nested records correctly

**Manual testing:**
- Test stapledons_voyage actual use case
- Verify game state structures work

## Timeline

**Day 1** (~4-6 hours):
- Phase 1: Implement let-flattening helper
- Phase 2: Modify normalizeLet
- Phase 3: Testing and documentation

**Total: 4-6 hours**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Scoping issues with flattened lets | High | Careful ordering: inner bindings first |
| Performance regression | Low | Linear transformation, minimal overhead |
| Breaking existing elaboration | High | Comprehensive regression tests |
| LetRec interaction | Medium | Test recursive cases separately |

## References

- stapledons_voyage bug report (agent inbox, 2025-11-30)
- `internal/elaborate/expressions.go:336-404` - Current `normalizeLet` implementation
- `internal/elaborate/core.go:154-190` - `normalizeToAtomic` and `wrapWithBindings`
- `internal/elaborate/verify.go:282-283` - ANF verification error location
- ANF (A-Normal Form) transformation - Standard compiler technique

## Future Work

- Record spread syntax: `{ ...base, newField: value }`
- Nested record pattern matching: `match r { {a: {b: x}} => x }`
- Record type inference improvements

---

**Document created**: 2025-12-01
**Last updated**: 2025-12-01
