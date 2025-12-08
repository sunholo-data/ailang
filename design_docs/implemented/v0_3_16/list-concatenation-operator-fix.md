# List Concatenation Operator Fix

**Status:** Implemented
**Target Version:** v0.3.16
**Discovered:** October 2025 (during Phase 2.6 example updates)
**Implemented:** October 2025

## Problem

The `++` operator type-checks for both strings and lists, but the builtin implementation only handles strings, causing runtime panics.

### Example Error

```ailang
-- examples/snippets/showcase/lists.ail
let numbers = [1, 2, 3] ++ [4, 5] in
print("Concatenation works for lists and strings!")
```

**Type checking:** ✅ PASSES
**Runtime:** ❌ PANIC

```
panic: interface conversion: eval.Value is *eval.ListValue, not *eval.StringValue
```

### Root Cause

**Location:** `internal/builtins/string.go:455`

The `_str_concat` builtin (which backs the `++` operator) has this implementation:

```go
func strConcatImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	s1 := args[0].(*eval.StringValue)  // ← Assumes string, panics on list!
	s2 := args[1].(*eval.StringValue)
	return &eval.StringValue{Value: s1.Value + s2.Value}, nil
}
```

But the **type system** allows `++` on lists via row polymorphism:

```
++ : forall a. a -> a -> a where a ~ String | List
```

## Impact

**Severity:** Medium
**User Impact:** Examples fail at runtime despite passing type checking
**AI Impact:** Confusing for AI - code looks correct but crashes

**Affected Files:**
- `examples/snippets/showcase/lists.ail` (currently broken)
- Any user code using `++` on lists

## Proposed Solution

### Option 1: Add List Concatenation Builtin (Recommended)

Add `_list_concat` builtin and route `++` operator appropriately:

```go
// internal/builtins/list.go
func listConcatImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	list1 := args[0].(*eval.ListValue)
	list2 := args[1].(*eval.ListValue)

	result := make([]eval.Value, 0, len(list1.Elements)+len(list2.Elements))
	result = append(result, list1.Elements...)
	result = append(result, list2.Elements...)

	return &eval.ListValue{Elements: result}, nil
}
```

**Effort:** ~1 hour
**Risk:** Low (simple append operation)

### Option 2: Runtime Type Dispatch

Modify `strConcatImpl` to check value type at runtime:

```go
func concatImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	switch v1 := args[0].(type) {
	case *eval.StringValue:
		v2 := args[1].(*eval.StringValue)
		return &eval.StringValue{Value: v1.Value + v2.Value}, nil
	case *eval.ListValue:
		v2 := args[1].(*eval.ListValue)
		result := append(v1.Elements, v2.Elements...)
		return &eval.ListValue{Elements: result}, nil
	default:
		return nil, fmt.Errorf("++ operator requires String or List")
	}
}
```

**Effort:** ~30 minutes
**Risk:** Medium (mixes type checking concerns into runtime)

### Option 3: Restrict Type System

Remove list concatenation from type system, make `++` string-only:

```
++ : String -> String -> String
```

**Effort:** ~2 hours (need to update type system, tests, examples)
**Risk:** High (breaking change, reduces language expressiveness)

## Recommendation

**Choose Option 1** (Add List Concatenation Builtin)

**Rationale:**
- Clean separation of concerns (one builtin per type)
- Matches AILANG's explicit philosophy
- Easy to test in isolation
- No breaking changes
- Follows existing builtin patterns

## Implementation Plan

1. **Add builtin** (~30 min):
   - Create `_list_concat` in `internal/builtins/list.go`
   - Register with proper type: `List[a] -> List[a] -> List[a]`
   - Mark as pure (no effects)

2. **Wire to operator** (~15 min):
   - Update operator resolution to use `_list_concat` for lists
   - Keep `_str_concat` for strings

3. **Test** (~15 min):
   - Unit tests in `internal/builtins/list_test.go`
   - Fix `examples/snippets/showcase/lists.ail`
   - Verify both string and list `++` work

## Test Cases

```ailang
-- String concatenation (existing, must still work)
let s = "hello" ++ " world" in
print(s)  -- "hello world"

-- List concatenation (currently broken, should work)
let nums = [1, 2, 3] ++ [4, 5] in
print(show(nums))  -- "[1, 2, 3, 4, 5]"

-- Nested lists
let nested = [[1], [2]] ++ [[3], [4]] in
print(show(nested))  -- "[[1], [2], [3], [4]]"

-- Empty lists
let empty = [] ++ [1, 2] in
print(show(empty))  -- "[1, 2]"
```

## Related Issues

- **Type System:** Row polymorphism for `++` operator (working correctly)
- **Runtime:** Builtin registry needs both string and list concat
- **Documentation:** Update `docs/LIMITATIONS.md` to remove this limitation once fixed

## References

- Discovery commit: c78f290 (Phase 2.6 example updates)
- Error location: `internal/builtins/string.go:455`
- Affected example: `examples/snippets/showcase/lists.ail`
- Related design: `design_docs/planned/v0_3_15/example-parity-vision-alignment.md`

---

## Implementation Report

**Implementation Approach:** Option 1 (Add List Concatenation Builtin)

**Files Changed:**
1. `internal/builtins/list.go` - New file with `concat_List` builtin
2. `internal/builtins/list_test.go` - Unit tests for list concatenation
3. `internal/builtins/register.go` - Documentation update
4. `internal/pipeline/op_table.go` - Added "List" to concat operator types
5. `internal/pipeline/op_lowering.go` - Added binding tracking and List type detection
6. `internal/pipeline/testdata/builtin_types.golden` - Updated golden file

**Key Implementation Details:**

1. **Builtin Registration:**
   - Registered `concat_List` with polymorphic type: `forall a. (List[a], List[a]) -> List[a]`
   - Pure function (no side effects)
   - Implementation concatenates elements using Go's `append`

2. **Type Detection Challenge:**
   - The ++ operator doesn't use type classes (like Num, Eq, Ord)
   - Type information not available in `resolvedConstraints` map
   - Core AST uses ANF (Administrative Normal Form), so list literals are bound to variables

3. **Solution:**
   - Track variable bindings during lowering phase
   - When encountering `++` operator with `Var` arguments, follow bindings to find actual values
   - Detect `*core.List` vs `*core.Lit` (string) to choose `concat_List` vs `concat_String`

4. **Test Coverage:**
   - Unit tests for `concat_List` implementation (6 test cases)
   - Type mismatch error handling tests (3 test cases)
   - Registration validation test
   - Integration test with `examples/snippets/showcase/lists.ail`

**Results:**
- ✅ All tests pass (`make test`)
- ✅ Linter passes (`make lint`)
- ✅ Both string and list concatenation work correctly
- ✅ Example file `examples/snippets/showcase/lists.ail` now runs successfully

**Lines of Code:**
- Builtin implementation: ~95 lines (list.go)
- Tests: ~160 lines (list_test.go)
- Op lowering changes: ~10 lines (binding tracking + list detection)
- Total new code: ~265 lines

**Limitations:**
- Binding-based type detection is brittle to future ANF transformations
- Doesn't work for function returns or complex expressions (see design_docs/planned/v0_3_17/type-guided-operator-lowering.md)

**Future Work:**
- **Type-guided lowering** (v0.3.17) - Use type information from type checker instead of value inspection
- Consider adding similar tracking for other polymorphic operators if needed
- Potential optimization: Cache binding lookups if performance becomes an issue

---

## Post-Implementation Review (Follow-up Improvements)

### Improvements Applied (October 2025)

Following code review feedback, the following enhancements were made:

#### 1. Enhanced Test Coverage ✅
Added comprehensive property-based and edge case tests:
- **Mathematical properties:** Length, left/right identity, associativity
- **Edge cases:** Empty lists, large lists (2000 elements), deeply nested lists
- **Immutability:** Verified inputs are not mutated
- **Total test coverage:** 15 test cases (up from 3)

#### 2. Immutability Documentation ✅
Added explicit comments to `listConcatImpl`:
```go
// IMPORTANT: This function does NOT mutate the input lists. It creates a new
// list with copies of the element references. The input lists remain unchanged,
// preserving referential transparency and enabling safe reuse.
```

#### 3. Operator Lowering Test ✅
Added `TestOpLowering_Concat` to lock in the lowering behavior:
- Verifies `++` lowers to `concat_String` for strings
- Verifies `++` lowers to `concat_List` for lists
- Prevents regressions in operator dispatch logic

#### 4. Type-Guided Lowering Design ✅
Documented the recommended improvement in `design_docs/planned/v0_3_17/type-guided-operator-lowering.md`:
- Replace binding-based detection with type information from type checker
- More robust, deterministic, and handles all cases
- Estimated effort: ~3 hours
- Target version: v0.3.17+

### Remaining Items (Future Work)

These items were identified but deferred to future versions:

#### 1. Type Error Tests (v0.3.17)
Add integration tests for type errors (should fail at typecheck):
```ailang
[] ++ ""           -- Error: cannot concat List and String
["a"] ++ [1]       -- Error: List[string] != List[int]
```
**Why deferred:** Type checker already handles these; tests would be redundant at builtin level.

#### 2. Documentation Updates (v0.3.17)
- Update language/operators page with `++` semantics
- Add REPL examples for list and string concat
**Why deferred:** Waiting for broader documentation overhaul.

#### 3. Naming Consistency (Not needed)
The naming convention is already consistent:
- `concat_List` (not `_list_concat`)
- Matches pattern: `<operation>_<Type>` (e.g., `add_Int`, `concat_String`)
- No changes needed ✅

### Metrics (After Improvements)

**Test Coverage:**
- Unit tests: 15 test cases (up from 3)
- Property tests: 4 mathematical properties verified
- Edge cases: 3 boundary conditions tested
- Integration tests: 2 operator lowering scenarios
- Total new test code: +150 lines

**Documentation:**
- Implementation comments: Enhanced with immutability guarantees
- Future work: Documented in separate design doc (type-guided lowering)
- Design rationale: Captured in post-implementation review

**Code Quality:**
- All tests pass ✅
- Linter clean ✅
- No regressions ✅
- Behavior locked in with golden tests ✅
