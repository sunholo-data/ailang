# M-POLY-B: Operator Re-Linking in Monomorphization

**Status**: Planned
**Target**: v0.4.1
**Priority**: P0 (High) - Blocks M-POLY-A completion
**Estimated**: 1-2 days (10-16 hours)
**Dependencies**: M-POLY-A (v0.4.0 - Var→Lam resolution infrastructure)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | Neutral | 0 | Fixes bug, no syntax impact |
| Preserve Semantic Clarity | Positive | +1 | Makes polymorphic operators work as expected |
| Increase Determinism | Positive | +1 | Ensures correct type-specific operator dispatch |
| Lower Token Cost | Neutral | 0 | No token impact (fixes runtime behavior) |
| **Net Score** | | **+2** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Rationale**: This is a critical bug fix that makes monomorphization semantically correct. When an AI writes `let max = \x. \y. if x > y then x else y`, they expect `max(3.14)(2.71)` to work with floats. Currently it panics. Fixing this maintains semantic clarity (+1) and determinism (+1) without changing syntax.

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

**The Bug:**

When a polymorphic lambda is bound to a variable and then specialized for concrete types, operators inside the lambda still use the wrong type-specific builtins.

**Test case that fails:**
```ailang
let max = \x. \y. if x > y then x else y
print(show(max(3.14)(2.71)))
```

**Expected**: `3.14`
**Actual**: Runtime panic
```
panic: interface conversion: eval.Value is *eval.FloatValue, not *eval.IntValue
    at internal/builtins/math.go:264 (gt_Int builtin)
```

**Current State:**
- M-POLY-A (v0.4.0) infrastructure complete: Var→Lam resolution works ✅
- Specialization is triggered correctly: `isPoly=true`, `count=1` ✅
- Specialized lambda created with fresh NodeIDs ✅
- CoreTypeInfo updated with type substitutions ✅
- **BUT**: Operators (like `>`) still call `gt_Int` instead of `gt_Float` ❌

**Impact:**
- **Who**: AI agents trying to write polymorphic comparison functions
- **Severity**: Critical - monomorphization is unusable for any code with operators
- **Scope**: Affects all polymorphic functions containing operators (comparison, arithmetic, etc.)
- **Current workaround**: Inline lambdas work, but Var-bound lambdas don't

**Root Cause:**

When cloning a lambda during specialization, we apply type substitution to CoreTypeInfo but don't update the actual operator nodes. The `>` operator was linked to a specific builtin (e.g., `gt_Int`) during earlier compilation phases, and that binding persists even after type substitution.

## Goals

**Primary Goal:** Ensure specialized lambdas use type-appropriate operators (e.g., `gt_Float` instead of `gt_Int` when specialized for floats).

**Success Metrics:**
- ✅ Var-bound polymorphic lambdas execute without type errors
- ✅ Operators dispatched to correct type-specific builtins (Int vs Float vs String)
- ✅ All M-POLY-A integration tests pass
- ✅ Zero performance regression (<5% slowdown acceptable)
- ✅ No changes to surface syntax (pure compiler fix)

## Solution Design

### Overview

**Strategy:** Re-link operators during lambda cloning by analyzing the type substitution and updating operator bindings accordingly.

**Key Insight:** The evaluator already has infrastructure for operator dispatch:
1. `BinOp` evaluation via `experimentalBinopShim` (runtime type dispatch)
2. `DictApp` evaluation (dictionary-based type class dispatch)

We need to ensure cloned lambdas use the correct dispatch mechanism.

### Architecture

**Three possible approaches investigated:**

#### Approach A: Runtime Type Dispatch (Current Behavior)
- **Idea**: Keep operators as BinOp, let evaluator dispatch based on runtime types
- **Status**: Works for inline lambdas, fails for var-bound lambdas
- **Why it fails**: Type information lost during cloning

#### Approach B: Re-elaborate Operators (Recommended)
- **Idea**: After cloning, re-run dictionary elaboration on the specialized lambda body
- **How**:
  1. Clone lambda with type substitution (existing code)
  2. Build a `ResolvedConstraint` map for the substituted types
  3. Run `ElaborateWithDictionaries()` on the cloned body
  4. This transforms BinOp → DictApp with correct TypeName
- **Pros**: Reuses existing elaboration infrastructure
- **Cons**: Requires making elaboration stateless/reentrant

#### Approach C: Direct Operator Node Update (Simpler Alternative)
- **Idea**: When cloning BinOp/DictApp/DictRef, directly update the operator binding
- **How**:
  1. Clone operator node (existing code)
  2. Check if type changed (e.g., `α → Float`)
  3. Update operator name (`gt` → `gt_Float`)
  4. Update DictRef TypeName if present
- **Pros**: Minimal changes, fast
- **Cons**: Duplicates operator resolution logic

**Recommendation:** Start with **Approach B** (re-elaboration) as it's architecturally cleaner and reuses existing logic. Fall back to Approach C if re-elaboration proves too complex.

### Components

1. **Operator Linking Tracer** (Priority 1)
   - Add debug logging to understand current operator resolution flow
   - Trace `>` from Surface AST → Core BinOp → DictApp? → Builtin call
   - Compare inline vs var-bound cases side-by-side
   - Document the full transformation pipeline

2. **Reentrant Dictionary Elaboration** (Priority 2)
   - Make `ElaborateWithDictionaries()` accept partial AST (not just full Program)
   - Support re-elaboration with pre-built ResolvedConstraint map
   - Add API: `ElaborateExpr(expr CoreExpr, resolved map[uint64]*ResolvedConstraint) (CoreExpr, error)`

3. **Specialization Re-Linking** (Priority 3)
   - Modify `specializeLambda()` to re-elaborate the cloned body
   - Build ResolvedConstraint map from type substitution
   - Call `ElaborateExpr()` on the cloned lambda body
   - Update CoreTypeInfo with new node mappings

4. **Integration Tests** (Priority 4)
   - Add test cases for all operator types (comparison, arithmetic, string ops)
   - Test multiple type specializations (Int, Float, String)
   - Test nested operators and complex expressions

### Implementation Plan

**Phase 1: Understand Operator Linking** (~2-4 hours)

- [ ] Add comprehensive debug logging to trace operator lifecycle
- [ ] Compare inline lambda vs var-bound lambda execution paths
- [ ] Document findings in M-POLY-B-INVESTIGATION.md
- [ ] Verify hypothesis: operators are linked before monomorphization
- [ ] Determine if BinOp → DictApp transformation happens (and when)

**Deliverable:** Clear understanding of operator resolution mechanism

**Phase 2: Implement Re-Elaboration** (~4-8 hours)

- [ ] Create `ElaborateExpr()` function for partial elaboration
- [ ] Refactor `ElaborateWithDictionaries()` to use `ElaborateExpr()`
- [ ] Add tests for re-elaboration (ensure idempotence)
- [ ] Update `specializeLambda()` to call `ElaborateExpr()` after cloning
- [ ] Build ResolvedConstraint map from type substitution
- [ ] Update CoreTypeInfo with re-elaborated nodes

**Deliverable:** Working re-elaboration infrastructure

**Phase 3: Testing & Validation** (~2 hours)

- [ ] Add integration tests for Var-bound lambda specialization
- [ ] Test all operator types: `+`, `-`, `*`, `/`, `>`, `<`, `==`, etc.
- [ ] Test multiple type instantiations: Int, Float, String
- [ ] Verify no performance regression (benchmark against inline lambdas)
- [ ] Test edge cases: nested operators, multiple operators in one lambda

**Deliverable:** All tests passing, bug fixed

**Phase 4: Documentation & Cleanup** (~1 hour)

- [ ] Remove verbose DEBUG_MONO_VERBOSE logging
- [ ] Update CHANGELOG.md: Remove v0.4.0 limitation
- [ ] Update CLAUDE.md: Remove "Direct lambda applications only" note
- [ ] Add operator re-linking section to monomorphization.md
- [ ] Clean up test files (test_max_fixed.ail, test_inline_max.ail)

**Deliverable:** Clean, documented implementation

### Files to Modify/Create

**New files:**
- `internal/elaborate/reentrant.go` - Reentrant elaboration API (~150 LOC)
- `M-POLY-B-INVESTIGATION.md` - Operator linking investigation notes (~500 lines)

**Modified files:**
- `internal/pipeline/specialize.go` - Call re-elaboration in specializeLambda() (~50 LOC)
- `internal/elaborate/dictionaries.go` - Refactor to support partial elaboration (~100 LOC)
- `internal/pipeline/specialize_integration_test.go` - Add operator tests (~200 LOC)
- `CHANGELOG.md` - Remove v0.4.0 limitation (~10 LOC)
- `CLAUDE.md` - Update monomorphization section (~20 LOC)

**Total new/modified LOC:** ~1030 LOC

## Examples

### Example 1: Polymorphic Comparison (Currently Broken)

**Before (v0.4.0 - Panics):**
```ailang
let max = \x. \y. if x > y then x else y
print(show(max(3.14)(2.71)))
# Panic: interface conversion: *FloatValue, not *IntValue
```

**After (v0.4.1 - Works):**
```ailang
let max = \x. \y. if x > y then x else y
print(show(max(3.14)(2.71)))  # 3.14
print(show(max(42)(17)))      # 42
print(show(max("zebra")("apple")))  # "zebra"
```

**What changed:**
- Specialized `max$Float` uses `gt_Float` operator
- Specialized `max$Int` uses `gt_Int` operator
- Specialized `max$String` uses `gt_String` operator

### Example 2: Polymorphic Arithmetic (Currently Broken)

**Before (v0.4.0 - Panics):**
```ailang
let add_double = \x. x + x
print(show(add_double(3.14)))
# Panic: interface conversion: *FloatValue, not *IntValue
```

**After (v0.4.1 - Works):**
```ailang
let add_double = \x. x + x
print(show(add_double(3.14)))    # 6.28
print(show(add_double(42)))      # 84
print(show(add_double("hello"))) # "hellohello" (string concat)
```

### Example 3: Inline Lambda (Already Works)

**Current behavior (v0.4.0 - Works):**
```ailang
print(show((\x. \y. if x > y then x else y)(3.14)(2.71)))  # 3.14
```

**After v0.4.1 (Still works, same mechanism):**
```ailang
print(show((\x. \y. if x > y then x else y)(3.14)(2.71)))  # 3.14
```

**Why it works:** Type inference happens before elaboration, so operators get correct types immediately.

## Success Criteria

- [ ] Var-bound polymorphic lambdas execute without type errors
- [ ] `let max = \x. \y. if x > y then x else y; max(3.14)(2.71)` returns `3.14`
- [ ] `let max = \x. \y. if x > y then x else y; max(42)(17)` returns `42`
- [ ] `let add_double = \x. x + x; add_double(3.14)` returns `6.28`
- [ ] All comparison operators work: `<`, `>`, `<=`, `>=`, `==`, `!=`
- [ ] All arithmetic operators work: `+`, `-`, `*`, `/`, `%`
- [ ] String concatenation works: `++`
- [ ] All M-POLY-A integration tests pass (19/19)
- [ ] No performance regression (<5% slowdown)
- [ ] Documentation updated (CHANGELOG.md, CLAUDE.md)
- [ ] Example files cleaned up

## Testing Strategy

**Unit tests:**
- `TestElaborateExpr_Reentrant` - Verify re-elaboration works
- `TestElaborateExpr_Idempotent` - Ensure re-elaborating twice gives same result
- `TestBuildResolvedConstraints` - Verify constraint map building from type subst

**Integration tests (in `specialize_integration_test.go`):**
- `TestSpecializeVarBoundLambda_FloatComparison` - Main bug fix
- `TestSpecializeVarBoundLambda_IntComparison` - Verify Int still works
- `TestSpecializeVarBoundLambda_StringComparison` - Verify String works
- `TestSpecializeVarBoundLambda_FloatArithmetic` - Test `+`, `-`, etc.
- `TestSpecializeVarBoundLambda_MultipleOperators` - Test `if x > 0 then x * 2 else x`
- `TestSpecializeVarBoundLambda_NestedOperators` - Test `(x + y) * (x - y)`

**Manual testing:**
- Run all examples in examples/ directory
- Verify no regressions in existing working code
- Test REPL behavior (should remain unchanged)
- Test file compilation vs REPL (both should work)

**Performance testing:**
- Benchmark `specializeLambda()` before/after
- Compare inline vs var-bound specialization times
- Ensure cache hit rate remains high

## Non-Goals

**Not in this feature:**
- **Cross-module specialization** - Deferred to v0.5.0 (M-POLY-C)
- **User-defined operators** - Out of scope (AILANG doesn't support custom operators)
- **Higher-kinded type specialization** - Deferred (requires type system extension)
- **Specialization of recursive functions** - Already handled (skipped in v0.4.0)
- **Whole-program optimization** - Out of scope
- **Pipeline visualization tools** - Separate feature (DX improvement)

## Timeline

**Day 1** (6-8 hours):
- Phase 1: Operator linking investigation (2-4h)
- Phase 2: Start re-elaboration implementation (4h)

**Day 2** (4-8 hours):
- Phase 2: Finish re-elaboration implementation (0-4h)
- Phase 3: Testing & validation (2h)
- Phase 4: Documentation & cleanup (1h)
- Buffer for unexpected issues (1h)

**Total: ~10-16 hours across 1-2 days**

**Milestone schedule:**
- ✅ Day 1 EOD: Operator linking understood, re-elaboration started
- ✅ Day 2 EOD: Bug fixed, tests passing, docs updated
- ✅ Day 3: Code review, merge to dev, tag v0.4.1

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Re-elaboration breaks existing code | High | Run full test suite, add regression tests for inline lambdas |
| Performance regression from double elaboration | Medium | Benchmark, optimize if >5% slowdown, consider caching |
| Dictionary elaboration not reentrant | High | Refactor to stateless, add idempotence tests |
| Complexity explosion in constraint building | Medium | Start simple (map type vars to concrete types), iterate |
| Corner cases in nested operators | Low | Comprehensive integration tests, manual testing |

## References

- [M-POLY-A-DAY5-FINDINGS.md](../../../M-POLY-A-DAY5-FINDINGS.md) - Initial investigation and root cause analysis
- [M-POLY-A design doc](../../implemented/v0_4_0/monomorphization.md) - Monomorphization infrastructure (Days 1-4)
- [Dictionary elaboration code](../../../internal/elaborate/dictionaries.go) - Current elaboration implementation
- [REPL elaboration](../../../internal/repl/repl_eval.go#L92) - Example of elaboration usage
- [Type class system](../../../internal/types/constraints.go) - Constraint resolution
- [Evaluation shim](../../../internal/eval/eval_operations.go#L182) - Runtime operator dispatch

**Related issues:**
- Inline lambdas work (v0.4.0) - proves the mechanism works
- Dictionary elaboration missing from file pipeline (potential future cleanup)
- Type system has TVar and TVar2 (already fixed in M-POLY-A Day 5)

## Future Work

**v0.4.2+:**
- **Pipeline visualization tool** (`ailang debug pipeline`) - See M-POLY-A-DAY5-FINDINGS.md DX feedback
- **AST inspection improvements** (`ailang debug ast --node-id X`) - Better debugging
- **Consolidate dictionary elaboration** - Move into file pipeline explicitly

**v0.5.0 (M-POLY-C):**
- **Cross-module specialization** - Specialize imported polymorphic functions
- **Specialization optimization** - Reduce code size via sharing

**v0.6.0+:**
- **Higher-kinded type specialization** - Specialize type constructors (List, Result, etc.)
- **Monomorphization-directed optimization** - Use type information for better codegen

---

**Document created**: 2025-10-22
**Last updated**: 2025-10-22
**Related Sprint**: M-POLY-A (Monomorphization)
**Milestone**: M-POLY-B (completes M-POLY-A)
