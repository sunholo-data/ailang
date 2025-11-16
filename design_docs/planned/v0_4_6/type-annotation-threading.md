# Type Annotation Threading from Surface AST to Core Type Checking

**Status**: Planned
**Target**: v0.4.6
**Priority**: P1 (Medium) - Affects type safety but has runtime fallback
**Estimated**: 2-3 days (12-18 hours)
**Dependencies**: None

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | No change to syntax - internal type system fix |
| Preserve Semantic Clarity | + | +2 | Type errors caught at compile-time instead of runtime - **major clarity win** |
| Increase Determinism | + | +1 | Static type checking more deterministic than runtime panics |
| Lower Token Cost | + | +1 | AI models get immediate feedback at type-check time, not after generation |
| **Net Score** | | **+4** | **Decision: Move forward - high value for type safety** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

**Discovered during**: v0.4.5 concat operator fix (M-BUG-CONCAT-INFERENCE)

Type annotations from function signatures in Surface AST are not properly threaded through to Core type checking. This causes parameters with explicit type annotations to appear as unconstrained type variables (`α1`, `α2`) during type inference, preventing early detection of type errors.

**Current State:**
- Function parameters like `func foo(s: string, xs: [int])` are parsed correctly in Surface AST
- During elaboration (Surface → Core), type annotations are lost
- Core type checker treats all parameters as fresh type variables
- Type errors only caught when constraints unify during solving, or at runtime
- Mixed-type operations (e.g., `string ++ [int]`) pass type checking but panic at runtime

**Impact:**
- **Type safety degraded**: Errors that should be caught statically are caught at runtime
- **AI development friction**: Models generate code that type-checks but crashes
- **User confusion**: "Why did it type-check if it's wrong?"
- **Debugging cost**: Runtime panics harder to debug than compile-time errors

**Example of the bug:**
```ailang
module test/broken

export func broken(s: string, xs: [int]) -> string {
  s ++ xs  // Should fail at type-check time!
}
```

**Current behavior:**
```bash
$ ailang check test/broken.ail
✓ No errors found!  # WRONG!

$ ailang run test/broken.ail
panic: interface conversion: eval.Value is *eval.ListValue, not *eval.StringValue
```

**Expected behavior:**
```bash
$ ailang check test/broken.ail
Error: Type mismatch at line 4
  Cannot concatenate string and [int]
  Left operand:  s: string
  Right operand: xs: [int]
```

## Goals

**Primary Goal:** Thread type annotations from Surface AST function signatures through to Core type checking, enabling compile-time detection of type mismatches.

**Success Metrics:**
- Mixed-type operations caught at type-check time (not runtime)
- `TestConcatMixedTypes` test unskipped and passing
- Zero regressions in existing type checking behavior
- All benchmarks still pass
- Improved error messages showing actual vs expected types

## Solution Design

### Overview

The root cause is a gap in the elaboration pipeline. When Surface AST is elaborated to Core AST, type annotations are discarded. The type checker then treats parameters as fresh variables, losing the explicit type constraints.

**Three potential approaches:**

#### Approach 1: Extend Core AST with Type Annotations (Recommended)

Add optional type annotations to Core AST nodes (Lambda, LetRec) and thread them through elaboration.

**Benefits:**
- Clean separation: Core AST carries type info, type checker uses it
- Minimal changes to existing code
- Backwards compatible (annotations are optional)
- Aligns with how effect annotations already work

**Drawbacks:**
- Adds fields to Core AST (slight complexity increase)
- Need to update Core AST serialization if used

#### Approach 2: Side-Channel Type Hints (Like Effect Annotations)

Use a map similar to `effectAnnots: map[uint64][]string` to pass type hints from elaborator to type checker.

**Benefits:**
- No Core AST changes
- Similar pattern to existing `effectAnnots` field

**Drawbacks:**
- Type info separated from AST structure
- Harder to maintain consistency
- Less principled than Approach 1

#### Approach 3: Full Bidirectional Type Checking

Implement full bidirectional type checking where types flow both up and down the AST.

**Benefits:**
- Most correct long-term solution
- Enables better type inference overall

**Drawbacks:**
- Major refactor (weeks of work)
- High risk of regressions
- Out of scope for this fix

### Recommended: Approach 1 (Extend Core AST)

**Why:** Cleanest architecture, minimal changes, aligns with existing patterns.

### Architecture

**Components:**

1. **Core AST Extension** (internal/core/core.go)
   - Add `ParamTypes []Type` field to `Lambda` struct
   - Add `ParamTypes []Type` field to `LetRec.RecBinding` struct (for recursive functions)
   - These are optional - `nil` means no type annotations

2. **Elaboration Updates** (internal/elaborate/)
   - When elaborating `ast.FuncDecl`, extract parameter type annotations
   - Convert `ast.Type` to `types.Type` using existing `astTypeToType()` helper
   - Store in Core Lambda's `ParamTypes` field
   - Thread through `normalizeLambda()` and related functions

3. **Type Checker Updates** (internal/types/typechecker_functions.go)
   - In `inferLambda()`, check if `lam.ParamTypes` is non-nil
   - If yes, use annotated types instead of fresh type variables
   - Still add to environment for body type checking
   - Unify with inferred types to catch mismatches

4. **Error Reporting** (internal/types/errors.go)
   - Improve error messages to show annotated vs inferred types
   - Example: "Expected string (from annotation), got [int] (inferred)"

### Implementation Plan

**Phase 1: Core AST Extension** (~4 hours)
- [ ] Add `ParamTypes []types.Type` field to `core.Lambda` struct
- [ ] Add `ParamTypes []types.Type` field to `core.RecBinding` struct
- [ ] Update Core AST constructors to accept optional param types
- [ ] Add unit tests for Core AST with type annotations

**Phase 2: Elaboration Threading** (~5 hours)
- [ ] Extract type annotations from `ast.FuncDecl.Params` in `elaborateFuncDecl()`
- [ ] Convert `ast.Type` to `types.Type` using existing helpers
- [ ] Thread param types through `normalizeLambda()` to Core Lambda
- [ ] Handle polymorphic type annotations (type parameters)
- [ ] Add tests for elaboration with type annotations

**Phase 3: Type Checker Integration** (~6 hours)
- [ ] Modify `inferLambda()` to use `lam.ParamTypes` if present
- [ ] Replace fresh type vars with annotated types for params
- [ ] Add constraint: annotated type ~ inferred type
- [ ] Improve error messages for annotation mismatches
- [ ] Add tests for type checking with annotations

**Phase 4: Testing & Documentation** (~3 hours)
- [ ] Unskip `TestConcatMixedTypes` and verify it fails correctly
- [ ] Add comprehensive tests for parameter type annotations
- [ ] Test edge cases: polymorphic types, recursive functions, nested lambdas
- [ ] Run full test suite and benchmarks
- [ ] Update CHANGELOG.md
- [ ] Update this design doc with implementation notes

### Files to Modify/Create

**Modified files:**
- `internal/core/core.go` (+10 LOC) - Add ParamTypes fields
- `internal/elaborate/file.go` (+30 LOC) - Extract and thread annotations
- `internal/elaborate/expressions.go` (+20 LOC) - Thread through normalizeLambda
- `internal/types/typechecker_functions.go` (+40 LOC) - Use annotations in inferLambda
- `internal/types/errors.go` (+20 LOC) - Better error messages
- `internal/pipeline/concat_operator_test.go` (~10 LOC) - Unskip TestConcatMixedTypes

**New test files:**
- `internal/elaborate/type_annotation_test.go` (~80 LOC) - Test elaboration threading
- `internal/types/param_annotation_test.go` (~100 LOC) - Test type checker integration

**Total estimated changes:** ~310 LOC (implementation: ~120 LOC, tests: ~190 LOC)

## Examples

### Example 1: Mixed-Type Concatenation (The Bug Case)

**Before (v0.4.5):**
```ailang
module test/broken

export func broken(s: string, xs: [int]) -> string {
  s ++ xs
}
```

```bash
$ ailang check test/broken.ail
✓ No errors found!  # Type annotations ignored!

$ ailang run test/broken.ail
panic: interface conversion: eval.Value is *eval.ListValue, not *eval.StringValue
```

**After (v0.4.6):**
```bash
$ ailang check test/broken.ail
Error: Type mismatch in function 'broken' at line 4
  Cannot concatenate string and list

  Left operand:  s: string (from parameter annotation)
  Right operand: xs: [int] (from parameter annotation)

  The ++ operator requires both operands to be the same type (string or list).
```

### Example 2: Correct Type Inference (Should Still Work)

**Code:**
```ailang
module test/correct

export func greet(name: string) -> string {
  "Hello, " ++ name  # Both are strings - works!
}

export func concat[a](xs: [a], ys: [a]) -> [a] {
  xs ++ ys  # Both are lists - works!
}
```

```bash
$ ailang check test/correct.ail
✓ No errors found!

$ ailang run test/correct.ail
✓ All tests pass
```

### Example 3: Polymorphic Functions (Still Infer Correctly)

**Code:**
```ailang
module test/polymorphic

export func identity[a](x: a) -> a {
  x  # Polymorphic type annotation preserved
}

export func apply[a, b](f: a -> b, x: a) -> b {
  f(x)  # Higher-order function with type params
}
```

```bash
$ ailang check test/polymorphic.ail
✓ No errors found!  # Polymorphism still works
```

## Success Criteria

- [ ] `TestConcatMixedTypes` unskipped and passing (type error detected)
- [ ] Mixed-type operations fail at type-check time, not runtime
- [ ] Error messages show both annotated and inferred types
- [ ] All existing tests passing (zero regressions)
- [ ] All benchmarks passing (no performance degradation)
- [ ] Polymorphic type annotations still work correctly
- [ ] Recursive functions with annotations work correctly
- [ ] Documentation updated (CHANGELOG, design doc)
- [ ] Examples added showing before/after behavior

## Testing Strategy

**Unit tests (elaboration):**
- Extract type annotations from `ast.FuncDecl` correctly
- Convert `ast.Type` to `types.Type` accurately
- Thread annotations through `normalizeLambda()`
- Handle polymorphic type parameters (`[a]`, `[a, b]`)
- Handle missing annotations (backwards compatibility)

**Unit tests (type checking):**
- Use annotated types instead of fresh type vars
- Unify annotated types with inferred types
- Detect mismatches between annotation and usage
- Polymorphic annotations don't over-constrain
- Recursive functions with annotations

**Integration tests:**
- Mixed-type concatenation fails at type-check time
- Correct type annotations pass
- Polymorphic functions still infer correctly
- Error messages are clear and actionable

**Regression tests:**
- All existing concat tests still pass
- All existing type checking tests still pass
- Benchmarks: `string_manipulation`, `list_operations`, `higher_order_functions`

**Manual testing:**
```bash
# Test the reproduction case
cat > /tmp/test_mixed.ail <<'EOF'
module tmp/test_mixed

export func broken(s: string, xs: [int]) -> string {
  s ++ xs
}
EOF

ailang check /tmp/test_mixed.ail
# Expected: Type error (not "No errors found!")
```

## Non-Goals

**Not in this feature:**
- Full bidirectional type checking - Too large, separate effort
- Type inference for unannotated parameters - Already works via Hindley-Milner
- Better error messages in general - Separate UX improvement
- Support for type annotations on let-bindings - Deferred to future work
- Return type annotation checking - Already works (via function signature)

## Timeline

**Day 1** (6 hours):
- Phase 1: Core AST extension
- Phase 2: Elaboration threading (partial)

**Day 2** (6 hours):
- Phase 2: Elaboration threading (complete)
- Phase 3: Type checker integration (partial)

**Day 3** (6 hours):
- Phase 3: Type checker integration (complete)
- Phase 4: Testing & documentation
- Final verification

**Total: ~18 hours across 3 days (or 2 focused sessions)**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Breaking polymorphic type inference | High | Comprehensive test suite for polymorphic functions; only use annotations for monomorphic cases |
| Performance regression in type checking | Medium | Benchmark before/after; annotation check is O(1) per param |
| Core AST changes break serialization | Low | Update serialization if used; add version check |
| Existing code relies on annotations being ignored | Low | Backwards compatible - annotations are optional |
| Error messages too verbose | Low | Keep messages concise; show annotation only when it conflicts |

## References

- **Bug Discovery**: v0.4.5 concat operator fix (M-BUG-CONCAT-INFERENCE)
  - See CHANGELOG.md v0.4.5 entry
  - See `internal/pipeline/concat_operator_test.go:116` (skipped test)
- **Existing Patterns**:
  - Effect annotations: `internal/types/typechecker_core.go:72` (`effectAnnots` map)
  - AST type conversion: `internal/types/typechecker.go:189` (`astTypeToType()`)
- **Related Issues**:
  - Expected-type context threading (partial solution in v0.4.5)
  - Bidirectional type checking (future work)
- **Design Philosophy**: [AI-first DX](../v0_3_15/example-parity-vision-alignment.md)

## Future Work

**Building on this fix:**
- Full bidirectional type checking (v0.5.0+)
  - Expected types flow down, inferred types flow up
  - Better inference for nested expressions
  - Fewer type annotations needed
- Type annotations on let-bindings
  - `let x: int = ...`
  - Currently only function parameters supported
- Better error messages system-wide
  - Show code snippets with errors
  - Suggest fixes
  - Link to documentation
- Type-directed code generation
  - Use annotations to optimize compiled code
  - Specialize polymorphic functions at compile time

**Dependencies for future work:**
- This fix is a prerequisite for full bidirectional typing
- Establishes pattern for threading Surface AST info to type checker
- Proves value of compile-time vs runtime type checking

---

**Document created**: 2025-11-16
**Last updated**: 2025-11-16
