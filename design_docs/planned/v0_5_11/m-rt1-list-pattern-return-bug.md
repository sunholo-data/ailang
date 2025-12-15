# M-RT1: List Pattern Match Return Value Bug

**Status:** IMPLEMENTED
**Priority:** P0 (Blocker)
**Version:** v0.5.11
**Created:** 2025-12-15
**Implemented:** 2025-12-15
**Blocks:** S1.3 (JSON decoding), std/json accessor functions

## Implementation Summary

**Root Cause:** The bug was NOT in list pattern matching - it was in how imported nullary constructors (like `None` from `std/option`) were handled. Imported constructors were not being registered with the elaborator, causing them to be treated as variable patterns instead of constructor patterns.

**Fix Location:** `internal/pipeline/pipeline_module.go`

**Changes Made:**
1. Added `importedCtorInfos` map to track full constructor info (TypeName, Arity, TypeParamCount) during import processing
2. After the elaborator is created, call `RegisterConstructor()` for each imported constructor
3. This allows the elaborator to recognize `None` as a constructor pattern instead of a variable pattern

**Files Modified:**
- `internal/pipeline/pipeline_module.go` - Register imported constructors with elaborator

**Regression Test:**
- `tests/m_rt1_imported_constructor_pattern_test.ail` - Tests Option pattern matching with imported constructors

## Summary

When a function returns a value extracted from a list pattern match (e.g., `Some(pair.value)` where `pair` comes from `[pair, ...rest]`), the caller receives `None` instead of the expected `Some` value. This is a **runtime bug** that corrupts return values.

## Reproduction

### Minimal Reproducer

```ailang
module temp/bug_repro

import std/json (decode, Json, JObject, JString)
import std/result (Result, Ok, Err)
import std/option (Option, Some, None)

let test_json = "{\"id\":\"test\"}"

-- This function exhibits the bug
func extract_first(kvs: List[{key: string, value: Json}]) -> Option[Json] ! {IO} {
  match kvs {
    [] => None,
    [pair, ...rest] => {
      _io_println("Returning Some(pair.value)");  -- This prints!
      Some(pair.value)  -- But caller receives None
    }
  }
}

export func main() -> () ! {IO} {
  match decode(test_json) {
    Err(_) => _io_println("Parse error"),
    Ok(j) => match j {
      JObject(kvs) => match extract_first(kvs) {
        None => _io_println("Got None"),  -- <-- This is what happens
        Some(_) => _io_println("Got Some!")
      },
      _ => ()
    }
  };
  ()
}
```

**Output:**
```
Returning Some(pair.value)
Got None
```

### What Works (Contrast)

```ailang
-- Direct pattern match in caller works:
match kvs {
  [first, ...rest] => {
    _io_println(first.key);  -- Works: prints "id"
    _io_println(show(_str_eq(first.key, "id")));  -- Works: prints "true"
  },
  [] => ()
}

-- Passing value to function and returning it works:
func always_some(val: Json) -> Option[Json] {
  Some(val)  -- Caller receives Some correctly
}
```

## Diagnostic Evidence

### Test 1: List is passed correctly
```
Before passing to function:
  Direct: has key=id
Calling check_list:
  check_list: has elements
```
✅ The list has elements in both caller and callee.

### Test 2: Field access works inside function
```
  extract_first: got pair with key=id
  extract_first: returning Some(pair.value)
```
✅ `pair.key` is accessible and correct inside the function.

### Test 3: Return value is corrupted
```
Final: None
```
❌ Despite returning `Some(pair.value)`, caller sees `None`.

### Test 4: Same pattern works without list extraction
```
func always_some_json(val: Json) -> Option[Json] {
  Some(val)
}
-- Caller receives Some correctly
```
✅ `Option[Json]` returns work when not extracting from list pattern.

## Root Cause Analysis

The bug occurs specifically when:
1. A function parameter is `List[{record}]`
2. The function uses pattern matching `[head, ...tail]` to extract elements
3. The function returns a value derived from the extracted element (e.g., `Some(head.field)`)

The return value is "lost" somewhere between the function body and the caller.

### Hypotheses

1. **Continuation capture issue**: The pattern match continuation may be capturing/restoring state incorrectly, overwriting the return value.

2. **Value reference invalidation**: `pair.value` may be a reference that becomes invalid when the function returns, causing the `Some` wrapper to contain garbage that gets interpreted as `None`.

3. **Evaluator stack corruption**: The nested match (list pattern inside ADT pattern) may be corrupting the evaluation stack.

4. **Record field access thunk**: `pair.value` may be a lazy thunk that evaluates incorrectly in the return context.

## Affected Code Paths

### Currently Blocked
- `std/json.get()` - Object field access
- `std/json.findInList()` - Key lookup helper
- `std/sem.decode_frame()` - Frame deserialization
- Any user code using list pattern matching with returns

### Files to Investigate
- `internal/eval/eval.go` - Core evaluator
- `internal/eval/pattern.go` - Pattern matching implementation
- `internal/eval/value.go` - Value representation
- `internal/core/` - Core IR (pattern match lowering)

## Proposed Fix

### Phase 1: Debugging (0.5 days)
1. Add debug logging to pattern match evaluation
2. Trace value lifecycle from pattern binding through return
3. Identify exact point where value is lost/corrupted

### Phase 2: Fix (0.5-1 day)
Depends on root cause:
- If continuation issue: Fix state save/restore
- If reference issue: Ensure proper value copying
- If stack issue: Review evaluation stack management

### Phase 3: Regression Tests (0.5 days)
1. Add test case for list pattern return
2. Add test case for nested ADT extraction
3. Add test case for record field access in returns

## Acceptance Criteria

```ailang
-- This must work:
func find_key(kvs: List[{key: string, value: Json}], target: string) -> Option[Json] {
  match kvs {
    [] => None,
    [kv, ...rest] => if _str_eq(kv.key, target) then Some(kv.value) else find_key(rest, target)
  }
}

-- Caller must receive Some when key exists
match find_key([{key: "foo", value: JString("bar")}], "foo") {
  Some(v) => assert(v == JString("bar")),  -- Must pass
  None => fail("Should not be None")
}
```

## Workarounds

Until fixed, users can:

1. **Use Go-backed builtins** for critical operations (not ideal)
2. **Avoid returning values from list patterns** (restructure code)
3. **Use explicit accumulator pattern** (pass result as parameter)

## Priority Justification

**P0 Blocker** because:
- Breaks fundamental list processing operations
- Blocks JSON object access (`std/json.get`)
- Blocks semantic frame serialization (sprint S1.3)
- Makes the language unreliable for basic operations

## References

- Sprint: DX-15-MVP
- Milestone blocked: S1.3-sem-frame-json
- Related: std/json accessor functions (just uncommented, also affected)
