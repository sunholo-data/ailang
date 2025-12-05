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

**The Invariant We Need to Enforce:**

> **If a parameter `x` is annotated as type `τ` in the Surface AST, then in Core and in the type environment, `x` must have type `τ` (modulo renaming of type variables), and any attempt to use it inconsistently is a type error.**

This is not currently enforced. Type annotations are effectively decorative, which violates user expectations and reduces type safety.

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

## The Role of Type Annotations in AILANG

**AILANG Philosophy:**

Type annotations in AILANG are **hard constraints**, not optional hints. When a programmer (or AI model) writes `x: string`, they are making an explicit assertion about the program's semantics. The type system must enforce this assertion.

**Contrast with other languages:**
- **Haskell**: Annotations are optional hints; inference can narrow or specialize them
- **TypeScript**: Annotations are assertions but can be overridden by inference in some cases
- **Rust**: Annotations are hard constraints when provided
- **AILANG**: Follows the Rust model - annotations are non-negotiable contracts

**Why this matters for AI code generation:**
1. **Determinism**: AI models need predictable feedback - no "sometimes annotations matter, sometimes they don't"
2. **Early errors**: Catch mismatches at type-check time, not after code generation completes
3. **Trust**: If code type-checks, it should run without type-related panics
4. **Composability**: Annotations serve as verified documentation for function boundaries

This fix moves AILANG from "annotations are decorative" (current buggy state) to "annotations are contracts" (intended behavior).

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

**Core Principle: Annotations as Hard Constraints**

When a parameter has a type annotation, that annotation **replaces** the fresh type variable that would normally be created. The annotated type is the parameter's type - not a constraint on it, not a hint, but the actual type.

**Implementation semantics:**
```go
// Current (buggy) behavior
param_type := ctx.freshTypeVar()  // Always create fresh var
// Annotation ignored!

// New (correct) behavior
if param.Annotation != nil {
    param_type := elaborateType(param.Annotation)  // Use annotation
} else {
    param_type := ctx.freshTypeVar()  // Fall back to inference
}
```

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
   - If yes, **replace** fresh type variables with the annotated types (not layer on top)
   - Add annotated types directly to environment for body type checking
   - Parameter types are fixed at annotation values - no further unification needed
   - If usage conflicts with annotation, constraint solving will fail with error

4. **Error Reporting** (internal/types/errors.go)
   - Improve error messages to show parameter annotations in context
   - Example: "Cannot concatenate string and [int]. Left operand: s: string (annotated), Right operand: xs: [int] (annotated)"
   - Focus on clarity: what went wrong, what types were involved, where they came from

### Handling Polymorphism

**Type parameters require careful handling:**

When a function uses type parameters (e.g., `func concat[a](xs: [a], ys: [a]) -> [a]`), the parameter annotations contain type variables (e.g., `a`). These must be handled consistently:

1. **Type parameter scope**: Type variables `a`, `b`, etc. are scoped to the function declaration
2. **Consistent mapping**: All occurrences of `a` in parameter annotations map to the same fresh type variable during elaboration
3. **Elaboration creates mapping**: When elaborating `[a]`, create fresh type var `α1` and map `a → α1`
4. **Reuse across parameters**: Second occurrence of `[a]` reuses `α1`, not `α2`
5. **Type checking proceeds normally**: Constraint solving unifies `α1` across all uses

**Example elaboration:**
```ailang
// Surface AST
func concat[a](xs: [a], ys: [a]) -> [a]

// Elaboration phase
// Create mapping: a → α1 (fresh)
// Convert [a] → TList(α1)
// Result: ParamTypes = [TList(α1), TList(α1)]

// Core AST
Lambda {
  ParamTypes: [TList(TVar(α1)), TList(TVar(α1))],
  Body: ...
}
```

**Key insight**: Type parameters are still type variables (not concrete types), so constraint solving still works. The difference from current behavior is that *all* occurrences of a type parameter map to the *same* type variable, enforcing consistency.

### Interaction with v0.4.5 Expected-Type Fix

**This fix complements the v0.4.5 concat operator fix:**

**v0.4.5 added:**
- `expectedType` field to `InferenceContext`
- Expected type threaded to Match arm bodies (tail position only)
- `++` operator checks expected type if both operands are type variables

**v0.4.6 adds:**
- Parameter type annotations threaded to Core type checking
- Annotations become hard constraints in the type environment

**How they interact:**

1. **Parameters get concrete types from annotations** (v0.4.6)
   ```ailang
   func join(sep: string, xs: [int]) -> string { ... }
   // sep: string (annotated), xs: [int] (annotated)
   ```

2. **Match arms get expected type from return annotation** (v0.4.5)
   ```ailang
   match xs {
     x :: rest => show(x) ++ sep ++ join(sep, rest)
     // Expected type: string (from return annotation)
   }
   ```

3. **`++` operator has concrete left operand, type variable right operand**
   ```ailang
   show(x) ++ sep
   // Left: string (inferred from show)
   // Right: string (annotated parameter)
   // Result: string concat ✓

   sep ++ join(sep, rest)
   // Left: string (annotated parameter)
   // Right: α1 (recursive call, not yet resolved)
   // Expected type: string (from Match arm context)
   // v0.4.5 fix: Use expected type → string concat ✓
   ```

**Together, these fixes enable the recursive string join case:**
- v0.4.6 gives parameters their annotated types
- v0.4.5 resolves ambiguous `++` calls using expected-type context
- Constraint solving unifies everything

**Important:** These are orthogonal fixes:
- v0.4.6 works even without v0.4.5 (parameter types are still enforced)
- v0.4.5 works even without v0.4.6 (expected-type threading still helps inference)
- Together, they cover more cases than either alone

### Implementation Plan

**Phase 1: Core AST Extension** (~4 hours)
- [ ] Add `ParamTypes []types.Type` field to `core.Lambda` struct
- [ ] Add `ParamTypes []types.Type` field to `core.RecBinding` struct
- [ ] Update Core AST constructors to accept optional param types
- [ ] Add unit tests for Core AST with type annotations

**Phase 2: Elaboration Threading** (~5 hours)
- [ ] Extract type annotations from `ast.FuncDecl.Params` in `elaborateFuncDecl()`
- [ ] Convert `ast.Type` to `types.Type` using existing helpers
- [ ] Handle polymorphic type annotations: create mapping from type param names (`a`, `b`) to fresh type vars (`α1`, `α2`)
- [ ] Ensure consistent mapping: all occurrences of `a` map to same `α1` across all parameters
- [ ] Thread param types through `normalizeLambda()` to Core Lambda
- [ ] Add tests for elaboration with type annotations (both monomorphic and polymorphic)

**Phase 3: Type Checker Integration** (~6 hours)
- [ ] Modify `inferLambda()` to check if `lam.ParamTypes` is non-nil
- [ ] If annotations present: use annotated types directly (don't create fresh type vars)
- [ ] Add annotated types to environment for body type checking
- [ ] No explicit unification needed - constraint solving will catch conflicts
- [ ] Improve error messages to show parameter annotations in conflict messages
- [ ] Add tests for type checking with annotations (monomorphic and polymorphic)

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
Error: Type mismatch at line 4, column 3
  Cannot concatenate string and list

  Expression: s ++ xs
  Left:  s: string
  Right: xs: [int]

  The ++ operator requires both operands to have the same type (both string or both list).
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

## Scope and Risk Assessment

**What's in scope:**
- Function parameter type annotations only
- Both monomorphic (concrete) and polymorphic (type parameter) annotations
- Minimal changes to Core AST, elaboration, and type checker
- Error message improvements for parameter-related type errors

**What's explicitly out of scope:**
- Let-binding annotations (`let x: int = ...`) - Deferred to future work
- Full bidirectional type checking - Too large, would require major refactor
- General error message improvements - Separate UX effort
- Type annotations on match patterns - Deferred to future work
- Type annotations on lambda parameters (inline lambdas) - Deferred to future work

**Risk level: LOW**

This is a well-scoped, surgical change:
1. **Minimal surface area**: Only function parameters, not all bindings
2. **Existing patterns**: Similar to how effect annotations work
3. **Testable**: Easy to write comprehensive tests for parameter annotations
4. **Backwards compatible**: Annotations are optional, existing code unchanged
5. **Orthogonal to other systems**: Doesn't interact with effects, modules, or evaluation

**Estimated effort: 18 hours** (conservative, includes comprehensive testing)

## Non-Goals

**Not in this feature:**
- Full bidirectional type checking - Major refactor, weeks of work
- Type inference improvements for unannotated code - Already works via Hindley-Milner
- General error message system overhaul - Separate UX improvement effort
- Let-binding annotations (`let x: int = ...`) - Deferred to v0.5.0+
- Return type annotation checking - Already works (return type is part of function signature)
- Lambda parameter annotations - Deferred to future work (less common use case)

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

## Follow-Up Questions for Future Versions

**This fix deliberately leaves some questions open for future work:**

1. **Let-binding annotations** (v0.5.0+)
   - Should `let x: int = 42` be supported?
   - If yes, same "hard constraint" semantics as function parameters?
   - What about pattern bindings? (`let (x: int, y: string) = ...`)

2. **Full bidirectional type checking** (v0.5.0+)
   - Should expected types flow down for *all* expressions, not just Match arms?
   - What's the right balance between inference and annotation requirements?
   - Can we maintain "AI-friendly" error messages with bidirectional typing?

3. **Lambda parameter annotations** (v0.5.0+)
   - Should inline lambdas support `\(x: int). x + 1`?
   - Less common in ML-style languages, but could reduce verbosity
   - What's the syntax? Parentheses required for annotated params?

4. **Annotation inference** (Future)
   - Can we infer "obvious" annotations to reduce boilerplate?
   - Example: `let f = \x. x + 1` → infer `x: int` from `+` operator?
   - Risk: Makes type checking less deterministic

**These questions are deliberately deferred** to keep v0.4.6 focused and low-risk. We'll revisit based on user feedback and AI code generation patterns.

---

**Document created**: 2025-11-16
**Last updated**: 2025-11-16
**Status**: Planned (pending architectural review approval)

**Changelog:**
- 2025-11-16: Initial version
- 2025-11-16: Incorporated architectural feedback
  - Added "Role of Annotations in AILANG" section
  - Clarified "hard constraint" vs "hint" semantics
  - Added polymorphism handling section
  - Added interaction with v0.4.5 expected-type fix
  - Added scope and risk assessment
  - Added follow-up questions for future versions
  - Improved error message examples
