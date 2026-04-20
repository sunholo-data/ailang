# M-CODEGEN-V3: Binding Hoisting for VALUE-Position Lets

**Status**: Planned (Follow-up to M-CODEGEN-V2)
**Target: v0.6.0 or v0.6.0
**Priority**: P2 (Enhancement - not blocking production)
**Estimated**: 2-3 days
**Dependencies**: M-CODEGEN-V2 complete
**Reporter**: Discovered during M-CODEGEN-V2 implementation

---

## TL;DR

**Problem**: After M-CODEGEN-V2, VALUE-position let chains still generate 2-3 levels of IIFEs.

**Solution**: Hoist inner let bindings to the enclosing function body level.

**Risk**: Evaluation order and variable capture semantics require careful handling.

---

## Background

M-CODEGEN-V2 successfully eliminated O(n) nesting from function bodies by flattening top-level let chains. However, let chains that appear in **value position** (e.g., the RHS of a variable binding) still generate nested IIFEs.

### Current State (After M-CODEGEN-V2)

```ailang
fn isWalkable(state, x, y) =
    let dSq = let tmp7 = state.discCenterX in
              let tmp8 = state.discCenterY in
              distSq(x, y, tmp7, tmp8)
    in let radiusSq = state.discRadius * state.discRadius
    in dSq < radiusSq
```

Generates:
```go
func isWalkable_impl(state, x, y interface{}) interface{} {
    var dSq interface{} = func() interface{} {
        var tmp7 interface{} = FieldGet(state, "discCenterX")
        return func() interface{} {
            var tmp8 interface{} = FieldGet(state, "discCenterY")
            return distSq_impl(x, y, tmp7, tmp8)
        }()
    }()
    var radiusSq interface{} = MulInt(FieldGet(state, "discRadius"), FieldGet(state, "discRadius"))
    return LtInt(dSq, radiusSq)
}
```

The function body is flat (`var dSq`, `var radiusSq`, `return`), but `dSq`'s value has 2-level IIFE nesting.

### Target State (With Binding Hoisting)

```go
func isWalkable_impl(state, x, y interface{}) interface{} {
    // Hoisted bindings from dSq's value expression
    var tmp7 interface{} = FieldGet(state, "discCenterX")
    var tmp8 interface{} = FieldGet(state, "discCenterY")
    var dSq interface{} = distSq_impl(x, y, tmp7, tmp8)
    var radiusSq interface{} = MulInt(FieldGet(state, "discRadius"), FieldGet(state, "discRadius"))
    return LtInt(dSq, radiusSq)
}
```

**Zero IIFEs** - all bindings are flat at function body level.

---

## Problem Statement

### Why VALUE-Position Lets Need IIFEs (Currently)

Go doesn't have let-expressions. In AILANG, `let x = e1 in e2` is an expression that:
1. Binds `x` to `e1`
2. Evaluates `e2` in scope with `x`
3. Returns the value of `e2`

When this appears in value position (e.g., `var foo = <let-expr>`), we must somehow evaluate both `e1` and `e2` and return `e2`. An IIFE accomplishes this:

```go
var foo = func() T {
    var x = e1
    return e2
}()
```

### The Opportunity: Hoisting

If the let-expression appears inside a function body, we can hoist the bindings up:

```go
// Instead of:
var foo = func() T {
    var x = e1
    return e2
}()

// We can emit:
var x = e1
var foo = e2
```

This is valid because:
1. `x` was only visible inside the let-expression anyway
2. Evaluation order is preserved (`e1` before `e2`)
3. `x` doesn't escape the function

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Recursive `LowerDeep` vs single-level hoisting | Recursive approach eliminates all nested IIFEs but increases complexity; single-level only removes one layer at a time | agent | design | med |
| Variable name collision strategy: detect-and-rename vs rely on elaborator uniqueness | If elaborator guarantees uniqueness, no rename needed; if not, silent collisions corrupt generated code | compiler | compile | high |
| Keep original `Lower()` as fallback or replace entirely | Keeping both means maintenance burden but provides safety net for debugging regressions | human | design | low |
| Hoist only in function bodies vs all expression contexts | Function-body-only is safe and covers most cases; all-context hoisting is more complete but risks edge cases in nested lambdas | human | design | med |
| Effect ordering: trust ANF guarantee vs add explicit ordering checks | ANF should guarantee correct order, but if the invariant is ever violated, hoisting silently reorders effects | compiler | compile | high |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Confirm elaborator uniqueness guarantee for tmp variables (determines whether rename logic is needed)
- [ ] Decide whether `LowerDeep` replaces `Lower` or coexists as an alternative path
- [ ] Decide scope of hoisting: function bodies only (Phase 1) or all expression contexts
- [ ] Verify ANF invariant holds for all effect-bearing expressions in current codebase
- [ ] Define IIFE count threshold for CI gate (stretch goal — needs a baseline number)

---

## Solution Design

### Block IR Extension

Extend the Block IR to support recursive lowering of value expressions:

```go
// Lower now recursively processes Value expressions
func LowerDeep(expr core.CoreExpr) *Block {
    var stmts []Stmt
    current := expr

    for {
        let, ok := current.(*core.Let)
        if !ok {
            break
        }

        // Recursively lower the value expression
        valueBlock := LowerDeep(let.Value)
        stmts = append(stmts, valueBlock.Stmts...)

        // The binding uses the final expression from value lowering
        stmts = append(stmts, Stmt{
            Name:  let.Name,
            Value: valueBlock.FinalExpr,
        })

        current = let.Body
    }

    return &Block{Stmts: stmts, FinalExpr: current}
}
```

### Example Transformation

**Input Core AST:**
```
Let dSq = (Let tmp7 = state.discCenterX in
           Let tmp8 = state.discCenterY in
           distSq(x, y, tmp7, tmp8))
in Let radiusSq = (MulInt(state.discRadius, state.discRadius))
in LtInt(dSq, radiusSq)
```

**After LowerDeep:**
```
Block {
    Stmts: [
        { Name: "tmp7", Value: state.discCenterX },
        { Name: "tmp8", Value: state.discCenterY },
        { Name: "dSq", Value: distSq(x, y, tmp7, tmp8) },
        { Name: "radiusSq", Value: MulInt(...) },
    ],
    FinalExpr: LtInt(dSq, radiusSq)
}
```

---

## Risks and Mitigations

### Risk 1: Variable Name Collisions

**Problem**: Hoisting `tmp7` from inside `dSq`'s value could collide with an outer `tmp7`.

**Mitigation**:
- AILANG's elaborator already generates unique temporary names (`tmp{N}`)
- Add a check during hoisting to detect collisions
- If collision detected, rename the hoisted binding

### Risk 2: Evaluation Order with Effects

**Problem**: If `e1` has effects and `e2` doesn't, hoisting changes when effects execute.

**Example:**
```ailang
let x = print("a") in
let y = let z = print("b") in z in
print("c")
```

**Analysis**: AILANG's Core IR is in ANF (A-normal form) - all arguments are already atomic. The elaborator ensures effects are sequenced correctly. Hoisting preserves the left-to-right evaluation order.

**Mitigation**: Add tests to verify effect ordering is preserved.

### Risk 3: Variable Capture in Closures

**Problem**: If a hoisted binding is captured by a lambda, hoisting could change semantics.

**Example:**
```ailang
let f = let x = 1 in (\() -> x) in f()
```

**Analysis**: This is safe because:
1. `x` is bound before the lambda is created
2. Hoisting `x` doesn't change the closure capture
3. Go closures capture by reference, so `x` is still accessible

**Mitigation**: Add tests for closure capture scenarios.

### Risk 4: Increased Complexity

**Problem**: Recursive lowering is more complex than simple let-chain extraction.

**Mitigation**:
- Implement incrementally (function bodies first, then recursive)
- Comprehensive test suite
- Keep simple Lower() as fallback for debugging

---

## Implementation Plan

### Phase 1: Recursive Block Lowering (Day 1)

**Goal**: Extend Block IR to recursively lower value expressions.

**Files to Modify**:
- `internal/gen/block/lower.go` - Add `LowerDeep()` function
- `internal/gen/block/lower_test.go` - Add recursive lowering tests

**Tests**:
- `TestLowerDeep_SingleNested` - One let inside value
- `TestLowerDeep_DeeplyNested` - 3+ levels of nesting
- `TestLowerDeep_MixedChain` - Some values have lets, some don't

### Phase 2: Integration with Codegen (Day 2)

**Goal**: Use `LowerDeep()` in `generateFlatBody()`.

**Files to Modify**:
- `internal/gen/golang/codegen_block.go` - Use LowerDeep

**Tests**:
- Verify no IIFEs in function bodies
- Effect ordering tests
- Closure capture tests

### Phase 3: Validation (Day 3)

**Goal**: Verify on stapledons_voyage.

**Metrics to Verify**:
- IIFEs should drop from 182 to <50
- No semantic changes (game behavior identical)
- Performance improvement measurable

---

## Success Criteria

### Hard Line

1. **Zero IIFEs in function bodies** for pure let chains
2. **All existing tests pass** - no regressions
3. **stapledons_voyage works** - game runs correctly

### Stretch Goals

1. **Handle all expression contexts** (not just function bodies)
2. **Add CI invariant** - fail if IIFE count exceeds threshold

---

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact rename strategy for variable collisions (prefix, suffix, counter increment) — [agent may resolve]
- Whether to emit comments in generated Go marking which bindings were hoisted (aids debugging but adds noise) — [agent may resolve]
- Performance benchmark methodology for measuring IIFE reduction impact — [agent may resolve]
- Whether `LowerDeep` should bail out and fall back to IIFE for specific edge cases (e.g., deeply nested closures) — [agent may resolve]
- CI threshold value for IIFE count gate — depends on baseline measurement from stapledons_voyage — [human may resolve]

## Non-Goals

- Full typed codegen (M-DX24)
- Native arithmetic operators
- Direct field access
- Eliminating `_impl` wrapper pattern

These are explicitly out of scope for this milestone.

---

## Decision: Implement or Defer?

**Recommendation**: Defer to v0.5.10 or v0.6.0.

**Rationale**:
1. M-CODEGEN-V2 fixes the critical OOM issue (P0 achieved)
2. Remaining IIFEs (182) are manageable for Go compiler
3. Binding hoisting has subtle risks (variable capture, effects)
4. Better to ship working fix first, optimize later

**Trigger for implementation**:
- If stapledons_voyage still has GC pressure issues
- If other projects report similar IIFE bloat
- When typed codegen (M-DX24) is in progress (synergies)

---

## References

- [M-CODEGEN-V2](m-codegen-v2-flat-output.md) - Parent design doc
- [Block IR](../../../internal/gen/block/) - Current implementation
- [codegen_block.go](../../../internal/gen/golang/codegen_block.go) - Flat body generation

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Binding order becomes explicit and deterministic |
| A2: Replayability | 0 | No impact on replay |
| A3: Effect Legibility | +1 | Effect ordering preserved and made explicit |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Simpler code structure enables local verification |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Flatter Go code is easier for tools to analyze |
| A8: Minimal Syntax | 0 | No AILANG syntax changes |
| A9: Cost Visibility | 0 | No cost tracking impact |
| A10: Composability | +1 | Builds on M-CODEGEN-V2 infrastructure |
| A11: Structured Failure | 0 | No error handling changes |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +5** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): Evaluation order explicitly preserved
- [x] A3 (Effects): Effects sequenced correctly via ANF
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Generates simpler Go for tooling

---

## Changelog

| Date | Change |
|------|--------|
| 2025-12-10 | Initial design document (discovered during M-CODEGEN-V2 validation) |
