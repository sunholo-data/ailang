# M-BUG-NESTED-RECORD-ANF: ANF Completion for Let RHS

**Status**: IMPLEMENTED
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

This is a **standard ANF completion bug**: the ANF transformer generates nested lets inside expressions but forgets to run the canonical "float inner lets to statement level" step for let RHSs.

The ANF invariant requires:
> Every `let x = e in ...` has `e` as a simple expression (var, literal, simple call), not a Let itself.

The issue is in the normalization pipeline:

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

**Key Insight:** This is NOT a "nested records" bug specifically - it's a general ANF normalization oversight. Any expression that normalization represents as `Let ... in expr` can appear as a let RHS. Records are just the first concrete encounter. The same issue would occur with nested tuples, list literals with non-atomic elements, or any composite construct.

**Location:** `internal/elaborate/expressions.go:387` - `normalizeLet` function

**Impact:**
- Cannot use natural nested record construction
- Verbose workarounds required for common patterns
- Confusing error message for users
- Affects game/simulation code with nested state (like stapledons_voyage)

## Goals

**Primary Goal:** Implement ANF completion for let RHS - ensure no Let expressions appear as RHS values.

**Success Metrics:**
- `{ pos: { x: 10, y: 20 }, name: "guard" }` compiles and runs correctly
- Deeply nested records work (3+ levels)
- Field access chains work: `npc.pos.x`
- All existing tests pass (regression-free)
- Clear error messages for genuine ANF violations
- Other composite constructs (tuples, lists) also benefit

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

This transformation is:
- **Semantically sound** in a pure + effect-typed setting (preserves evaluation order)
- **Exactly what a textbook ANF pass does**: no let on the RHS, all lets "pulled out" to statement level

### Architecture Options

#### Option A: Fix in normalizeLet (Recommended for this sprint)

```go
normVal := normalize(expr.Value)

// Peel off nested Lets (only at top level, not inside lambdas)
bindings, coreVal := extractLetBindings(normVal)

// Build flattened structure: inner bindings outermost
core := body
core = coreLet(name, coreVal, core)
for i := len(bindings)-1; i >= 0; i-- {
    core = coreLet(bindings[i].name, bindings[i].value, core)
}
return core
```

**Pros:**
- Minimal diff (~50 LOC)
- Keeps "RHS must be simple" invariant enforced in one place

**Cons / Watch-outs:**
- `extractLetBindings` must NOT wander across lambda/LetRec boundaries
- Only strip `core.Let` nodes at top of expression, don't descend into subexpressions

#### Option B: Centralize "bindings+atom" Pattern (Future Refactor)

Make normalization functions return `(bindings, atom)` pairs where possible:
- Maintain a single helper `emitBindings(bindings, finalExpr)` which always builds a flat let chain
- ANF invariant "RHS is simple" follows by construction

**Trade-off:** Larger refactor, but prevents this class of bug for nested tuples, list literals, etc. Keep in mind for future ANF extensions.

**Decision:** Implement Option A for this sprint. Option B is future work if more composite constructs hit this pattern.

### Critical Implementation Notes

**Order Matters:** The helper must preserve original nesting order:
```
For: Let tmp1 = e1 in Let tmp2 = e2 in e3
Want: tmp1 outermost, then tmp2, then npc, then body
```

**Scope Safety:** Since everything is pure and ANF dictates evaluation order, moving inner lets outward doesn't change semantics as long as we don't cross lambdas/binders. Doing it inside `normalizeLet` (only rearranging a single RHS and its local body) respects this.

**Boundary Constraint:** `extractLetBindings` should ONLY:
- Strip top-level Let nodes from the expression
- NOT recursively descend inside arbitrary subexpressions
- NOT cross into lambda bodies or LetRec bindings

### Implementation Plan

**Phase 1: Implement Let-Flattening Helper** (~2 hours)
- [ ] Create `extractLetBindings(expr CoreExpr) ([]binding, CoreExpr)` helper
- [ ] Recursively extract ONLY top-level Let bindings into a flat list
- [ ] Stop at non-Let nodes, lambdas, LetRec boundaries
- [ ] Return the innermost body and all extracted bindings in correct order

**Phase 2: Modify normalizeLet** (~2 hours)
- [ ] After normalizing value, call `extractLetBindings` if value is Let
- [ ] Use extracted bindings to build flattened structure
- [ ] Maintain correct scoping order (inner bindings first, user binding last)
- [ ] Handle LetRec similarly if needed

**Phase 3: Testing** (~2 hours)
- [ ] Add test: simple nested record
- [ ] Add test: deeply nested record (3+ levels)
- [ ] Add test: nested record with field access
- [ ] Add test: record update with nested value
- [ ] Add test: list of nested records
- [ ] Add test: multiple non-atomic fields (ordering)
- [ ] Add test: nested record inside call argument
- [ ] Add test: effects inside nested record (ordering)
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

## Extended Test Cases (From Expert Review)

### Test 5: Multiple Non-Atomic Fields (Ordering)

```ailang
pure func computeX() -> int { 10 }
pure func computeY() -> int { 20 }
pure func baseHp() -> int { 100 }
pure func baseMp() -> int { 50 }

let npc = {
  pos: { x: computeX(), y: computeY() },
  stats: { hp: baseHp() + 10, mp: baseMp() }
}
-- Verify: multiple temps generated, correct let order, no nested-let RHS
```

### Test 6: Nested Record Inside Call Argument

```ailang
pure func processWorld(w: {pos: {x: int, y: int}, name: string}) -> int {
    w.pos.x
}

let result = processWorld({ pos: { x: 10, y: 20 }, name: "test" })
-- Verify: don't flatten lets across call boundary
```

### Test 7: Interaction with Record Updates

```ailang
let npc = { pos: { x: 0, y: 0 }, name: "guard" }
let npc2 = { npc with pos: { x: 5, y: npc.pos.y } }
-- Verify: let-flattening doesn't break record update syntax
```

### Test 8: Effects Inside Nested Record (Evaluation Order)

```ailang
func log(msg: string) -> int ! IO {
    _io_print(msg)
    1
}

func test() -> {pos: {x: int, y: int}, name: int} ! IO {
    { pos: { x: log("x"), y: log("y") }, name: log("name") }
}
-- Verify: ANF preserves evaluation order (x, y, name)
```

## Success Criteria

- [ ] `{ pos: { x: 10, y: 20 }, name: "guard" }` compiles and runs
- [ ] Deeply nested records (3+ levels) work
- [ ] Field access chains work: `npc.pos.x`
- [ ] Nested records in lists work: `[{a: {b: 1}}, {a: {b: 2}}]`
- [ ] Multiple non-atomic fields generate correct temp ordering
- [ ] Nested record in call argument doesn't break
- [ ] Record updates with nested values work
- [ ] Effects maintain correct evaluation order
- [ ] All existing tests pass
- [ ] `examples/nested_records.ail` added and verified
- [ ] CHANGELOG.md updated

## Interim Error Message Improvement (Optional)

While the bug exists, consider adding a targeted diagnostic in the ANF verifier:

```go
// In verify.go, when detecting nested Let in RHS:
case *core.Let:
    return fmt.Errorf("nested let in RHS (not yet supported): " +
        "inline nested records require intermediate let bindings. " +
        "Try: let inner = {...} then let outer = {field: inner, ...}")
```

This gives users the workaround while the fix is pending. Low priority if fix ships in v0.5.0.

## Testing Strategy

**Unit tests** (`internal/elaborate/expressions_test.go`):
```go
func TestNestedRecordNormalization(t *testing.T) {
    src := `let npc = { pos: { x: 10, y: 20 }, name: "guard" }`
    // Verify normalization succeeds
    // Verify flattened structure
}

func TestMultipleNonAtomicFields(t *testing.T) {
    // Test ordering of generated temps
}

func TestNestedRecordInCallArg(t *testing.T) {
    // Ensure we don't flatten across call boundaries
}
```

**Integration tests:**
- Run `examples/nested_records.ail` through full pipeline
- Verify REPL handles nested records correctly

**Manual testing:**
- Test stapledons_voyage actual use case
- Verify game state structures work

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Scoping issues with flattened lets | High | Careful ordering: inner bindings first |
| Flattening across lambda boundaries | High | extractLetBindings stops at lambdas |
| Performance regression | Low | Linear transformation, minimal overhead |
| Breaking existing elaboration | High | Comprehensive regression tests |
| LetRec interaction | Medium | Test recursive cases separately |

## Future Work

- **Option B Refactor**: Centralize "bindings+atom" pattern for all composite constructs
- Record spread syntax: `{ ...base, newField: value }`
- Nested record pattern matching: `match r { {a: {b: x}} => x }`
- Record type inference improvements

## References

- stapledons_voyage bug report (agent inbox, 2025-11-30)
- `internal/elaborate/expressions.go:336-404` - Current `normalizeLet` implementation
- `internal/elaborate/core.go:154-190` - `normalizeToAtomic` and `wrapWithBindings`
- `internal/elaborate/verify.go:282-283` - ANF verification error location
- ANF (A-Normal Form) transformation - Standard compiler technique

---

**Document created**: 2025-12-01
**Last updated**: 2025-12-01 (incorporated expert review feedback)
