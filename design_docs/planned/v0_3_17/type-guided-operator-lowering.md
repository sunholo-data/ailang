# Type-Guided Operator Lowering

**Status:** Planned
**Priority:** Medium
**Target Version:** v0.3.17+
**Effort:** ~3 hours
**Origin:** Feedback on v0.3.16 list concatenation fix

## Problem

Currently, operator lowering for polymorphic operators (like `++`) uses **value-based detection** by tracking ANF bindings and inspecting actual values (`*core.List` vs `*core.Lit`).

**Current approach (v0.3.16):**
```go
// Track bindings during traversal
l.bindings[varName] = boundValue

// Later, when lowering concat:
if v, ok := arg.(*core.Var); ok {
    if binding, exists := l.bindings[v.Name]; exists {
        arg = binding  // Follow binding to actual value
    }
}
if _, ok := arg.(*core.List); ok {
    typeSuffix = "List"  // Choose concat_List
} else {
    typeSuffix = "String"  // Choose concat_String
}
```

**Issues:**
1. **Brittle to transformations** - Future ANF optimizations might change binding structure
2. **Incomplete** - Doesn't work if binding isn't a direct literal (e.g., function return)
3. **Redundant** - Type checker already computed the types; why re-infer?
4. **Non-deterministic** - Behavior depends on ANF structure, not semantics

## Proposed Solution

### Option 1: Plumb Types from Type Checker (Recommended)

Store the inferred type for each Core AST node during type checking, then use it during lowering.

**Implementation:**

#### Step 1: Extend CoreTypeChecker to track expression types

```go
// internal/types/typechecker_core.go
type CoreTypeChecker struct {
    // ... existing fields ...
    exprTypes map[uint64]Type  // NodeID → inferred type
}

// After type checking each expression, store its type:
func (tc *CoreTypeChecker) inferExpr(expr core.CoreExpr) Type {
    typ := tc.infer(expr)  // existing logic
    tc.exprTypes[expr.ID()] = typ  // NEW: store type
    return typ
}
```

#### Step 2: Pass types to OpLowerer

```go
// internal/pipeline/pipeline.go
lowerer := NewOpLowerer(cfg.TypeEnv)
lowerer.SetResolvedConstraints(typeChecker.GetResolvedConstraints())
lowerer.SetExprTypes(typeChecker.GetExprTypes())  // NEW
```

#### Step 3: Use types in lowering

```go
// internal/pipeline/op_lowering.go
case core.OpConcat:
    // Look up the type of the first argument
    if argType, ok := l.exprTypes[intrinsic.Args[0].ID()]; ok {
        typeSuffix = getTypeSuffixFromType(argType)
        // argType is List[a] → typeSuffix = "List"
        // argType is String → typeSuffix = "String"
    } else {
        // Fallback (should never happen if type checker succeeded)
        return fmt.Errorf("no type info for concat argument")
    }
```

**Benefits:**
- ✅ **Deterministic** - Based on type system, not ANF structure
- ✅ **Robust** - Works regardless of ANF transformations
- ✅ **Complete** - Handles all cases (function returns, complex expressions, etc.)
- ✅ **Efficient** - No binding traversal needed
- ✅ **Correct** - Uses the same types the type checker verified

**Effort:** ~3 hours
- 1 hour: Extend type checker to store expression types
- 1 hour: Wire types to lowering phase
- 1 hour: Update lowering logic and tests

### Option 2: Typed Core AST

Create a parallel `TypedCore` AST where every node carries its type.

**Benefits:**
- ✅ All type information available during any traversal
- ✅ Enables future optimizations (type-directed inlining, etc.)
- ✅ Better error messages (can show types in diagnostics)

**Drawbacks:**
- ❌ Large refactor (~20 hours)
- ❌ Duplicates AST structure
- ❌ Overkill for current needs

**Recommendation:** Defer to v0.4.0+ when implementing reflection system.

## Implementation Plan (Option 1)

### Phase 1: Type Storage (~1 hour)

1. Add `exprTypes map[uint64]Type` to `CoreTypeChecker`
2. Modify `inferExpr()` to store types after inference
3. Add `GetExprTypes() map[uint64]Type` accessor

### Phase 2: Wiring (~1 hour)

1. Add `SetExprTypes(map[uint64]Type)` to `OpLowerer`
2. Update pipeline to call `SetExprTypes` after type checking
3. Extend `getTypeSuffixFromType()` to handle all type constructors

### Phase 3: Lowering Logic (~30 min)

1. Replace binding-based detection with type lookup
2. Remove `bindings map[string]core.CoreExpr` from `OpLowerer`
3. Update error messages to reference types

### Phase 4: Testing (~30 min)

1. Verify all existing tests still pass
2. Add test for function return: `f() ++ g()` where f, g return lists
3. Add test for complex expressions: `(if cond then [1] else [2]) ++ [3]`

## Migration Strategy

**Backward compatibility:** This is an internal change; no user-facing changes.

**Deprecation path:**
1. v0.3.17: Implement type-guided lowering alongside binding-based
2. v0.3.17: Add feature flag to switch between approaches
3. v0.3.18: Make type-guided default, keep binding-based as fallback
4. v0.3.19: Remove binding-based code entirely

**Rollout:** Can be done incrementally; start with `++`, then extend to other polymorphic operators.

## Testing Strategy

### Unit Tests

```go
func TestTypeGuidedLowering(t *testing.T) {
    tests := []struct {
        name     string
        expr     string
        wantType string
    }{
        {"list concat", `[1] ++ [2]`, "List"},
        {"string concat", `"a" ++ "b"`, "String"},
        {"function return", `f() ++ g()`, "List"}, // NEW: can't detect with bindings!
        {"conditional", `(if true then [1] else [2]) ++ [3]`, "List"}, // NEW
    }
    // ...
}
```

### Integration Tests

Add examples that break binding-based detection but work with type-guided:

```ailang
-- examples/tests/concat_complex.ail
export func getList() -> List[int] = [1, 2, 3]
export func test() -> List[int] = getList() ++ [4, 5]  -- Should work!
```

## Performance Impact

**Expected:** Neutral or slight improvement
- Type lookup: O(1) map access
- Binding traversal: O(depth) for nested bindings
- Preallocation: Already optimal

**Measurement:** Benchmark operator-heavy workloads before/after.

## Future Extensions

Once type information is plumbed to lowering:

1. **Better error messages:**
   ```
   Error: Operator '+' not defined for String
   At: "hello" + 42
       ^^^^^^^ String
                ^^ Int
   ```

2. **Type-directed optimizations:**
   - Inline pure functions when types known
   - Specialize polymorphic code

3. **Reflection support:**
   - `typeOf(expr)` can query the type map
   - Enables runtime type introspection

## Related Work

- **Haskell:** Uses typed Core (System F_C) for optimization
- **OCaml:** Typed intermediate representation (Lambda)
- **Rust MIR:** Typed MIR enables borrow checking and optimization

AILANG should follow this pattern for AI-friendliness: **types guide lowering and optimization**.

## References

- Current implementation: `internal/pipeline/op_lowering.go:297-313`
- Type checker: `internal/types/typechecker_core.go`
- Feedback: v0.3.16 implementation review
- Related: `docs/architecture/ANF.md` (to be created)
