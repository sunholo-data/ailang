# M-CODEGEN-FLAT-IF-ELSE: Flatten Nested Closure Chains in If-Else Codegen

**Status**: Planned
**Target**: v0.5.9
**Priority**: P0 (High - causes OOM and system freezes)
**Estimated**: 6 hours
**Dependencies**: None
**Bug Report**: `msg_20251209_202946_f10751f5` from stapledons_voyage

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | AILANG syntax unchanged; Go output cleaner |
| Preserve Semantic Clarity | 0 | 0 | Same semantics, different implementation |
| Increase Determinism | + | +1 | Predictable linear output vs exponential nesting |
| Lower Token Cost | + | +1 | 400 lines → ~50 lines for 25-branch if-else |
| **Compilation Success** | ++ | +2 | Fixes OOM crashes, enables large if-else chains |
| **Net Score** | | **+4** | **Decision: Move forward (critical fix)** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

## Problem Statement

The Go code generator wraps every if-else expression in an IIFE (Immediately Invoked Function Expression). When if-else expressions are chained, this creates deeply nested closures that cause:

1. **Go compiler OOM** - 25-branch chain uses 2GB+ RAM, often killed by OS
2. **Runtime GC pressure** - Each closure allocates heap memory; hot paths (26K calls/sec in games) cause system freezes

**Current Implementation** (`internal/gen/golang/codegen_expr_control.go:11`):
```go
func (g *Generator) generateIf(ifExpr *core.If) error {
    g.writef("func() %s {\n", returnType)
    g.indent++
    g.writef("if ")
    // ... condition ...
    g.write(" {\n")
    g.writef("return ")
    // ... then branch ...
    g.writef("return ")
    // ... else branch (may be another If, creating nesting!) ...
    g.indent--
    g.write("}()")
    return nil
}
```

**AILANG Source** (simple 25-branch if-else):
```ailang
pure func getTileType(distSq: int) -> int {
    if distSq < 0 then 0
    else if distSq < 4 then 1
    else if distSq < 9 then 2
    else if distSq < 16 then 3
    -- ... 21 more branches ...
    else 25
}
```

**Current Go Output** (~400 lines, 25 levels deep):
```go
func GetTileType(distSq int64) int64 {
    return func() int64 {
        if distSq < int64(0) { return int64(0) }
        return func() int64 {
            if distSq < int64(4) { return int64(1) }
            return func() int64 {
                if distSq < int64(9) { return int64(2) }
                return func() int64 {
                    // ... 22 more levels of nesting ...
                    return int64(25)
                }()
            }()
        }()
    }()
}
```

**Impact:**
- **stapledons_voyage game** had to close apps to compile, then froze at runtime
- **Any long if-else chain** (common in parsers, state machines, tile maps) is affected
- **Workaround applied**: Use squared distance comparisons to avoid isqrt, reducing branches

## Goals

**Primary Goal:** Generate flat Go if-else statements instead of nested closures for if-else chains

**Success Metrics:**
- 25-branch if-else generates ~50 lines instead of ~400 lines
- Go compiler memory usage stays under 500MB for typical if-else chains
- No runtime heap allocations for pure if-else chains
- All existing codegen tests pass unchanged

## Solution Design

### Overview

Detect when the `else` branch of an if-expression is another if-expression, and generate a flat `if-else if-else` chain instead of nested IIFEs.

**Key Insight:** Go's `if-else if-else` is a statement, but we need expressions. Solution: wrap the entire chain in ONE IIFE at the top level, then generate flat if-else statements inside.

### Desired Go Output

```go
func GetTileType(distSq int64) int64 {
    return func() int64 {
        if distSq < int64(0) { return int64(0) }
        if distSq < int64(4) { return int64(1) }
        if distSq < int64(9) { return int64(2) }
        if distSq < int64(16) { return int64(3) }
        // ... 21 more simple if statements ...
        return int64(25)
    }()
}
```

**Properties:**
- Single IIFE wrapper (constant overhead)
- Linear if statements (O(n) code, not O(n) nesting depth)
- Early returns mean no else-if needed
- Same semantics, dramatically simpler output

### Architecture

**Algorithm:**
1. When generating an if-expression, check if we're at the "chain root" (not already inside a flattened chain)
2. If so, detect the chain length by walking the else branches
3. Generate single IIFE wrapper
4. Generate flat `if { return } if { return } ... return default` inside

**Chain Detection:**
```go
func isIfElseChain(expr core.CoreExpr) bool {
    ifExpr, ok := expr.(*core.If)
    if !ok {
        return false
    }
    // Check if else branch is another If
    _, elseIsIf := ifExpr.Else.(*core.If)
    return elseIsIf
}

func collectIfChain(ifExpr *core.If) []chainBranch {
    var branches []chainBranch
    current := ifExpr
    for {
        branches = append(branches, chainBranch{
            Cond: current.Cond,
            Then: current.Then,
        })
        if nextIf, ok := current.Else.(*core.If); ok {
            current = nextIf
        } else {
            // Final else branch
            branches = append(branches, chainBranch{
                Cond: nil, // No condition = default case
                Then: current.Else,
            })
            break
        }
    }
    return branches
}
```

**Components:**
1. **Chain Detector**: `isIfElseChain()` - identifies chains worth flattening
2. **Chain Collector**: `collectIfChain()` - extracts all branches
3. **Flat Generator**: `generateIfChain()` - emits flat Go code
4. **Context Flag**: `inFlatChain` field to prevent re-wrapping nested ifs

### Implementation Plan

**Phase 1: Chain Detection & Collection** (~2 hours)
- [ ] Add `chainBranch` struct type
- [ ] Implement `isIfElseChain()` helper
- [ ] Implement `collectIfChain()` helper
- [ ] Add unit tests for chain detection

**Phase 2: Flat Code Generation** (~2 hours)
- [ ] Add `inFlatChain` field to Generator
- [ ] Implement `generateIfChain()` method
- [ ] Modify `generateIf()` to detect and delegate to chain generator
- [ ] Handle mixed chains (some branches are ifs, some aren't)

**Phase 3: Testing & Edge Cases** (~2 hours)
- [ ] Test single if-else (should still work)
- [ ] Test 2-branch chain (minimal chain)
- [ ] Test 25-branch chain (original bug case)
- [ ] Test nested chains (if-else inside then branch)
- [ ] Test mixed expressions (non-if in middle of chain)
- [ ] Verify existing codegen tests pass
- [ ] Memory profiling: verify Go compiler uses <500MB

### Files to Modify/Create

**Modified files:**
- `internal/gen/golang/codegen_expr_control.go` - Add chain detection and flat generation (~80 LOC)
- `internal/gen/golang/generator.go` - Add `inFlatChain` context field (~5 LOC)

**New files:**
- `internal/gen/golang/codegen_expr_control_test.go` - Chain detection and generation tests (~150 LOC)

## Examples

### Example 1: Simple If-Else Chain (25 branches)

**AILANG:**
```ailang
pure func getTileType(distSq: int) -> int {
    if distSq < 0 then 0
    else if distSq < 4 then 1
    else if distSq < 9 then 2
    else if distSq < 16 then 3
    else if distSq < 25 then 4
    else 5
}
```

**Before (nested closures - ~100 lines):**
```go
func GetTileType(distSq int64) int64 {
    return func() int64 {
        if distSq < int64(0) { return int64(0) }
        return func() int64 {
            if distSq < int64(4) { return int64(1) }
            return func() int64 {
                if distSq < int64(9) { return int64(2) }
                return func() int64 {
                    if distSq < int64(16) { return int64(3) }
                    return func() int64 {
                        if distSq < int64(25) { return int64(4) }
                        return int64(5)
                    }()
                }()
            }()
        }()
    }()
}
```

**After (flat if-else - ~15 lines):**
```go
func GetTileType(distSq int64) int64 {
    return func() int64 {
        if distSq < int64(0) { return int64(0) }
        if distSq < int64(4) { return int64(1) }
        if distSq < int64(9) { return int64(2) }
        if distSq < int64(16) { return int64(3) }
        if distSq < int64(25) { return int64(4) }
        return int64(5)
    }()
}
```

### Example 2: Nested If in Then Branch (should not flatten inner)

**AILANG:**
```ailang
pure func classify(x: int, y: int) -> int {
    if x < 0 then
        if y < 0 then 1 else 2  -- Inner if should remain as IIFE
    else if x < 10 then 3
    else 4
}
```

**After:**
```go
func Classify(x int64, y int64) int64 {
    return func() int64 {
        if x < int64(0) {
            return func() int64 {
                if y < int64(0) { return int64(1) }
                return int64(2)
            }()
        }
        if x < int64(10) { return int64(3) }
        return int64(4)
    }()
}
```

### Example 3: Single If-Else (no change needed)

**AILANG:**
```ailang
pure func sign(x: int) -> int {
    if x < 0 then -1 else 1
}
```

**Output (unchanged, single IIFE is fine):**
```go
func Sign(x int64) int64 {
    return func() int64 {
        if x < int64(0) { return int64(-1) }
        return int64(1)
    }()
}
```

## Success Criteria

- [ ] 25-branch if-else generates <60 lines of Go code
- [ ] Go compiler uses <500MB RAM for stapledons_voyage build
- [ ] No runtime closures allocated for pure if-else chains
- [ ] Existing codegen tests pass without modification
- [ ] New test: `examples/if_else_chain.ail` with 25+ branches compiles
- [ ] Benchmark: stapledons_voyage game runs without GC freezes
- [ ] All tests passing
- [ ] Documentation updated (CHANGELOG.md)

## Testing Strategy

**Unit tests:**
- Chain detection: 2-branch, 5-branch, 25-branch chains
- Non-chain detection: single if-else, if with complex else
- Mixed chains: some branches have nested ifs
- Type preservation: verify return types are correct

**Integration tests:**
- Compile and run 25-branch if-else
- Verify same runtime behavior as before
- Memory profiling during Go compilation

**Manual testing:**
- Rebuild stapledons_voyage with fix
- Verify compilation completes without OOM
- Verify game runs without GC freezes

## Non-Goals

**Not in this feature:**
- **Match expression optimization** - Different AST structure, separate fix needed
- **Let-in-if optimization** - Separate issue (M-FIX-IF-ELSE-LET)
- **Tail call optimization** - Out of scope for this fix
- **Switch statement generation** - Future optimization, not required for correctness

## Timeline

**Day 1** (4 hours):
- Phase 1: Chain detection and collection
- Phase 2: Flat code generation

**Day 2** (2 hours):
- Phase 3: Testing and edge cases
- Documentation and cleanup

**Total: ~6 hours**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Break existing codegen behavior | High | Comprehensive test suite, run all examples |
| Miss edge case in chain detection | Medium | Test nested ifs, mixed chains, single if-else |
| Performance regression for simple ifs | Low | Single if-else still uses simple IIFE (unchanged) |
| Type inference breaks in flat chain | Medium | Preserve return type logic from original generateIf |

## References

- Bug report: `ailang messages read msg_20251209_202946_f10751f5`
- Current implementation: [codegen_expr_control.go:11](internal/gen/golang/codegen_expr_control.go#L11)
- Related issue: M-FIX-IF-ELSE-LET (implicit blocks in if-else branches)
- stapledons_voyage project (external consumer)

## Future Work

- **Match expression flattening** - Same pattern could apply to match with many branches
- **Switch statement generation** - For integer comparisons against literals, generate Go switch
- **Dead branch elimination** - Detect unreachable branches and skip codegen

---

**Document created**: 2025-12-09
**Last updated**: 2025-12-09
