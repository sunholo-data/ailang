# List Concatenation Operator Fix

**Status:** Planned
**Target Version:** v0.3.15+
**Discovered:** October 2025 (during Phase 2.6 example updates)

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
