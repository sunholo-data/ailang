# M-DX17: Go Codegen - ConcatList Undefined + Closure Variable Scoping

**Status:** Complete ✅
**Target:** v0.6.2
**Priority:** P0 (Blocking - breaks real-world code)
**Estimated:** 4 hours
**Dependencies:** None
**Reporter:** stapledons_voyage (agent message)

---

## Problem Statement

Two codegen bugs are blocking Go compilation of AILANG modules using list concatenation and match expressions:

### Bug 1: ConcatList Undefined

The `++` (list concatenation) operator generates calls to `ConcatList()` but the runtime helper in generated Go code is named `Concat`.

**Error messages:**
```
sim_gen/interior.go:436:16: undefined: ConcatList
sim_gen/interior.go:446:15: undefined: ConcatList
sim_gen/interior.go:538:12: undefined: ConcatList
... (6 occurrences)
```

**Root cause analysis:**

1. When `++` is used on lists, `op_lowering.go` converts it to builtin `concat_List` (line 36 of `op_table.go`)
2. The codegen's `generateVarGlobal` in `codegen_expr_simple.go:141` falls through to `ToPascalCase(e.Ref.Name)`
3. `ToPascalCase("concat_List")` produces `ConcatList` (underscores become word boundaries)
4. But the runtime helper defined in `codegen_runtime_collections.go:97` is named `Concat`, not `ConcatList`

**AILANG code that triggers:**
```ailang
[{head | lastVisited: frame}] ++ tail
{state | deckStates: state.deckStates ++ [deckID]}
```

### Bug 2: Closure Variable Scoping in Match Arms

Variables from match patterns are not being captured correctly in closures or block expressions within match arm bodies.

**Error messages:**
```
sim_gen/interior.go:621:63: undefined: role
sim_gen/interior.go:621:69: undefined: mood
sim_gen/interior.go:623:83: undefined: npcID
```

**AILANG code that triggers:**
```ailang
pure func assignCrewRecursive(deckStates: [DeckState], crewInfo: [(int, string, string)]) -> [DeckState] {
    match crewInfo {
        [] => deckStates,
        (npcID, role, mood) :: rest => {
            let preferredDeck = getCrewPreferredDeck(role, mood);  -- role, mood undefined!
            let updatedStates = addCrewToDeck(deckStates, preferredDeck, npcID);  -- npcID undefined!
            assignCrewRecursive(updatedStates, rest)
        }
    }
}
```

**Root cause hypothesis:**

Looking at `codegen_match.go`, the `generatePatternCondition` function generates bindings for tuple pattern elements (line 350-384). The bindings are emitted correctly, but when the body contains a block expression `{ ... }` that's compiled to an IIFE, the IIFE closure may not be capturing the pattern-bound variables.

---

## Goals

**Primary Goal:** Make `++` operator and match pattern variables work correctly in generated Go code.

**Success Metrics:**
- [ ] `++` operator on lists compiles to valid Go (calls `Concat` not `ConcatList`)
- [ ] Match pattern variables accessible in block expressions within arm bodies
- [ ] stapledons_voyage interior.ail compiles and runs successfully
- [ ] No regressions in existing codegen tests

---

## Solution Design

### Fix 1: Rename Runtime Helper OR Add Mapping

**Option A: Rename `Concat` to `ConcatList` (simpler)**

In `internal/gen/golang/codegen_runtime_collections.go`, rename the helper:

```go
// Current (line 97):
g.writef("// Concat concatenates two lists (++ operator).\n")
g.writef("func Concat(a, b interface{}) interface{} {\n")

// Fixed:
g.writef("// ConcatList concatenates two lists (++ operator).\n")
g.writef("func ConcatList(a, b interface{}) interface{} {\n")
```

**Option B: Add builtin mapping in codegen**

In `codegen_expr_simple.go`, add mapping similar to `mapPureMathBuiltin`:

```go
func mapListBuiltinToHelper(name string) string {
    mappings := map[string]string{
        "concat_List": "Concat",
        // Add others as needed
    }
    return mappings[name]
}
```

**Recommendation:** Option A is simpler and maintains naming consistency with how other operator builtins are named (the function name matches what `ToPascalCase` produces).

### Fix 2: Investigate Block Expression IIFE Closure Capture

Need to trace through how block expressions in match arms are generated. The issue is likely in one of:

1. `codegen_block.go` - Block expression generation
2. `codegen_match.go:generateMatchIfElse` - Match arm body generation
3. IIFE wrapper around block expressions not including pattern bindings in scope

**Debugging approach:**
1. Create minimal reproduction case
2. Add `DEBUG_CODEGEN=1` output to see generated Go code
3. Identify where pattern variables go out of scope
4. Fix the scope chain to include pattern bindings

---

## Implementation Plan

### Phase 1: Fix ConcatList (~30 min) ✅ COMPLETE

- [x] Rename `Concat` to `ConcatList` in `codegen_runtime_collections.go`
- [x] Update any references to `Concat` in the codebase (none found)
- [x] Verified existing tests pass
- [x] Verified generated Go code compiles

**Implementation (2025-12-20):**
Changed `codegen_runtime_collections.go:97-98` from:
```go
g.writef("// Concat concatenates two lists (++ operator).\n")
g.writef("func Concat(a, b interface{}) interface{} {\n")
```
to:
```go
g.writef("// ConcatList concatenates two lists (++ operator).\n")
g.writef("func ConcatList(a, b interface{}) interface{} {\n")
```

### Phase 2: Investigate Closure Scoping ✅ COMPLETE

- [x] Create minimal reproduction case for the scoping bug
- [x] Generate debug output to see where variables go missing
- [x] Trace through codegen for block-in-match-arm pattern
- [x] Identify root cause

**Root cause (2025-12-20):**
In `codegen_match.go:generatePatternCondition`, the `ListPattern` case only handled `VarPattern` elements. When a tuple pattern like `(n, s)` appeared as the head of a cons pattern `(n, s) :: rest`, it was completely ignored and no bindings were generated.

### Phase 3: Fix Closure Scoping ✅ COMPLETE

- [x] Implement fix based on investigation
- [x] Verified generated Go code compiles
- [x] All tests pass

**Implementation (2025-12-20):**
Updated `codegen_match.go:281-328` to handle nested patterns in list pattern elements:
- Added switch statement to handle `VarPattern`, `WildcardPattern`, `TuplePattern`, and other patterns
- For non-VarPattern elements, extract into temp variable and recursively generate bindings
- Added `M-DX17` comments for traceability

**Generated code now correctly produces:**
```go
_listElem0 := ListHead(_scrutinee)
n := _listElem0.([]interface{})[0]
s := _listElem0.([]interface{})[1]
```

### Phase 4: Validation ✅ COMPLETE

- [x] Run full test suite - all pass
- [x] Minimal reproduction cases compile and run
- [x] No regressions

### Phase 5: Direct `concat` Function Calls ✅ COMPLETE (2025-12-21)

**Bug report:** stapledons_voyage reported that direct `concat(a, b)` function calls (not using `++` operator) were generating `Concat` instead of `ConcatList`.

**Root cause:** The `++` operator compiles to `concat_List` builtin which correctly maps to `ConcatList`. But direct `concat` function references fell through to `ToPascalCase("concat")` → "Concat" which doesn't exist.

**Fix:** Added `mapPureListBuiltin` function in `codegen_expr_simple.go` (similar to `mapPureMathBuiltin`):

```go
func (g *Generator) mapPureListBuiltin(name string) string {
    listMappings := map[string]string{
        "concat":       "ConcatList",
        "_list_concat": "ConcatList",
        "length":       "Length",
        "_list_length": "Length",
    }
    return listMappings[name]
}
```

Called from `generateVarGlobal` after the math builtin check.

---

## Minimal Reproduction Cases

### Test Case 1: List Concatenation

```ailang
module test

pure func concatTest(a: [int], b: [int]) -> [int] = a ++ b

export main = concatTest([1, 2], [3, 4])
```

**Expected generated Go:**
```go
func ConcatList(a, b interface{}) interface{} { ... }
// or
func Concat(a, b interface{}) interface{} { ... }

func concatTest_impl(a interface{}, b interface{}) interface{} {
    return ConcatList(a, b)  // Must match defined helper name
}
```

### Test Case 2: Match Pattern Variables in Block

```ailang
module test

pure func closureTest(pairs: [(int, string)]) -> [string] {
    match pairs {
        [] => [],
        (n, s) :: rest => {
            let result = s;  -- s must be in scope here
            [result]
        }
    }
}

export main = closureTest([(1, "hello")])
```

**Expected generated Go:**
```go
func closureTest_impl(pairs interface{}) interface{} {
    return func() interface{} {
        _scrutinee := pairs
        if ListLen(_scrutinee) >= 1 {
            n := _scrutinee.([]interface{})[0].([]interface{})[0]
            s := _scrutinee.([]interface{})[0].([]interface{})[1]
            _ = n
            _ = s
            rest := ListTail(_scrutinee)
            _ = rest
            // Block body - s MUST be visible here
            return func() interface{} {
                result := s  // s must be captured by closure
                _ = result
                return []interface{}{result}
            }()
        }
        // ...
    }()
}
```

---

## Files to Modify

| File | Change | LOC |
|------|--------|-----|
| `internal/gen/golang/codegen_runtime_collections.go` | Rename `Concat` to `ConcatList` | ~5 |
| `internal/gen/golang/codegen_match.go` | Fix pattern variable scoping (TBD) | ~20-50 |
| `internal/gen/golang/codegen_block.go` | May need scope chain fix (TBD) | ~10-20 |
| `internal/gen/golang/codegen_test.go` | Add test cases | ~50 |

**Total estimated LOC:** ~100

---

## Related Documents

- [M-CODEGEN-RECORDUPDATE](../../archive/v0_5_1_m-codegen-recordupdate.md) - Similar codegen issues
- [M-BUG-MODULE-LET-SCOPE](../../archive/v0_4_9_m-bug-module-let-scope.md) - Previous scoping bug

---

## Success Criteria

- [x] `make test` passes
- [x] Minimal reproduction cases compile and run
- [x] No undefined symbol errors in generated Go code
- [ ] stapledons_voyage interior.ail compiles successfully (pending external testing)

---

## Timeline

**Day 1 (4 hours):**
- Fix ConcatList naming (30 min)
- Investigate and fix closure scoping (3 hours)
- Validation and testing (30 min)

---

## Notes

- These are blocking bugs for external project stapledons_voyage
- Both bugs are in Go codegen, not in AILANG type-checking (which passes)
- The ConcatList fix is straightforward; closure scoping may require deeper investigation
