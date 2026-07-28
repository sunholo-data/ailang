# v0.3.17 Prompt Improvement Recommendations

Based on analysis of v0.3.16 eval baseline results (145 AILANG failures).

## Executive Summary

**Key Finding**: 17 benchmarks have 0% success rate (all 6 models fail). Major issue is **missing or inadequate examples** for common patterns, not just false limitations.

**Error Breakdown**:
- 62% compile_error (90 failures) - Wrong syntax, models guess incorrectly
- 31% logic_error (46 failures) - Correct syntax but wrong algorithm
- 6% runtime_error (9 failures) - Stack overflow, capability errors

**Top Error Codes**:
- PAR_001 (68 occurrences) - Parse errors from wrong syntax
- WRONG_LANG (10) - Generating Python/other languages
- IMPERATIVE (7) - Using assignment statements
- CAP_001 (3) - Missing capability grants

## Critical Missing Examples

### 1. Higher-Order Functions (0% success, 6/6 models fail)

**What models generate**:
```ailang
-- ❌ WRONG - func(b) -> c is not a valid type syntax
export func compose[a, b, c](f: func(b) -> c, g: func(a) -> b) -> func(a) -> c {
  func(x: a) -> c { f(g(x)) }
}
```

**What works**:
```ailang
-- ✅ CORRECT - Use lambdas, not type annotations for functions
export func main() -> () ! {IO} =
  let compose = \f. \g. \x. f(g(x)) in
  let double = \x. x * 2 in
  let addOne = \x. x + 1 in
  let result = compose(addOne)(double)(5) in
  print(show(result))  -- Prints: 11
```

**Issue**: AILANG doesn't have first-class function **type** syntax (`func(T) -> U`). Functions are passed as lambda values.

**Recommendation**: Add higher-order function section with compose, map, filter examples using lambda syntax.

### 2. JSON Encoding (0% success, 6/6 models fail)

**What models generate**:
```ailang
-- ❌ WRONG - Imperative style with mutations
obj = {
  "name": "Alice",
  "age": 30
}
json = encode_json(obj)
print(json)
```

**What works** (already in v0.3.17 but needs emphasis):
```ailang
-- ✅ CORRECT - Use std/json builder functions
module benchmark/solution

import std/json (encode, jo, kv, js, jnum, jbool)

export func main() -> () ! {IO} {
  let json = encode(jo([
    kv("name", js("Alice")),
    kv("age", jnum(30.0)),
    kv("active", jbool(true))
  ]));
  print(json)
}
```

**Issue**: Models don't know AILANG has no JSON literal syntax. Must use builder functions.

**Recommendation**: Already added in v0.3.17 (line 164), but needs more visibility. Consider adding to "Common Patterns" section.

### 3. List Operations (0% success, 6/6 models fail - runtime errors)

**What models generate** (looks correct but fails):
```ailang
-- Generated code that compiles but has runtime error
export func sum_list(xs: [int]) -> int {
  match xs {
    [] => 0,
    Cons(x, rest) => x + sum_list(rest)
  }
}

let my_list = [1, 2, 3, 4, 5];
let total = sum_list(my_list);
```

**Issue**: Unknown - code looks correct. Possible issues:
- List pattern matching runtime behavior
- List literal handling
- Need to investigate actual runtime error

**Recommendation**: Test this pattern, investigate runtime error, add working example.

### 4. List Comprehension (0% success, 6/6 models fail)

**What models generate**:
```python
# ❌ WRONG - Python list comprehension syntax
result = [x * 2 for x in [1, 2, 3, 4, 5]]
```

**What works**:
```ailang
-- ✅ CORRECT - Use recursion or std/list functions
import std/list (map)

export func main() -> () ! {IO} =
  let nums = [1, 2, 3, 4, 5] in
  let doubled = map(\x. x * 2)(nums) in
  print(show(doubled))
```

**Issue**: AILANG has no list comprehension syntax. Models default to Python.

**Recommendation**: Add explicit "NO list comprehensions" limitation + recursive alternative example.

### 5. Numeric Modulo (0% success, 6/6 models fail)

**Check**: Does `%` operator work? Prompt says it does (line 108).

**Recommendation**: Examine generated code, verify operator works, add example if missing.

### 6. Record Update (0% success, 6/6 models fail)

**Prompt claims** (line 108): ✅ Record updates - `{base | field: value}`

**Check**: Why are all models failing? Is syntax documented?

**Recommendation**: Verify record update syntax, add concrete example with multiple fields.

## Secondary Issues

### 7. Float Equality (0% success)
### 8. Exhaustive Pattern Matching (0% success)
### 9. Effect Pure Separation (0% success)
### 10. Pipeline Operations (0% success)
### 11. CLI Arguments (0% success)
### 12. Canonical Normalization (0% success)
### 13. Deterministic List Transform (0% success)
### 14. Print Missing Effect (0% success)
### 15. No Runtime Crashes Option (0% success)

**Recommendation**: Examine each, identify pattern, add example.

## Prompt Structure Issues

### Problem: Prompt Too Long (1237 lines)

**Impact**: Models struggle to find relevant information.

**Comparison**:
- v0.3.9: 656 lines (higher success rate?)
- v0.3.17: 1237 lines (nearly double)

**Recommendations**:

1. **Move detailed sections to separate "advanced" prompt**
   - Keep core syntax, common patterns, critical examples in main prompt
   - Move effect system details, state threading, advanced patterns to "advanced" prompt
   - Models can request "advanced prompt" if needed

2. **Create "Common Patterns" section near top** (after imports):
   - HTTP POST with JSON (✅ already added line 164)
   - Higher-order functions (compose, map, filter)
   - List operations (sum, length, filter with recursion)
   - JSON encoding (nested objects, arrays)
   - Record updates (with examples)
   - Pattern matching (exhaustive, guards)

3. **Add "What AILANG Does NOT Have" section**:
   - ❌ NO list comprehensions → use map/filter/recursion
   - ❌ NO function type syntax `func(T) -> U` → use lambdas
   - ❌ NO JSON literals `{"key": "value"}` → use jo/kv builder functions
   - ❌ NO mutable variables → use let bindings
   - ❌ NO for/while loops → use recursion
   - ❌ NO assignment `x = y` → use let

## Recommended Changes for v0.3.18

### High Priority (Address 0% success benchmarks)

1. **Add Higher-Order Functions section**
   - Compose, map, filter examples with lambda syntax
   - Explicitly state NO `func(T) -> U` type syntax
   - Show currying pattern

2. **Enhance JSON Encoding section**
   - More prominent placement (already at line 164)
   - Add nested objects example
   - Add arrays with ja() example
   - Show all builder functions: jo, ja, kv, js, jnum, jbool, jnull

3. **Add List Operations section**
   - Recursive sum, length, filter examples
   - Pattern matching on lists
   - Explicitly state NO list comprehensions
   - Show std/list functions (map, filter, fold)

4. **Add "Common Patterns" section** (lines 200-400)
   - HTTP POST with JSON (✅ done)
   - Higher-order functions
   - List operations
   - Record updates
   - Pattern matching

5. **Investigate runtime errors**
   - Why does list_operations fail with runtime_error?
   - Test generated code locally
   - Add working examples

### Medium Priority

6. **Add explicit "What AILANG Does NOT Have" section**
7. **Shorten prompt** (target: <1000 lines)
8. **Move advanced topics** to separate prompt

### Low Priority

9. Examine remaining 0% benchmarks
10. Add more negative examples (❌ WRONG / ✅ CORRECT pairs)

## Testing Strategy

After implementing changes:

1. **Test with one model first** (gpt5 or claude-sonnet-4-5)
2. **Focus on previously failing benchmarks**:
   - higher_order_functions
   - json_encode
   - list_operations
   - list_comprehension
3. **Measure improvement**:
   - Target: 0% → 50%+ on critical benchmarks
   - Overall: 31% → 45%+ AILANG success rate
4. **Iterate if needed**

## Example Code to Add

### Higher-Order Functions (Complete Example)

```ailang
module benchmark/solution

-- Compose: apply g, then f
export func main() -> () ! {IO} =
  let compose = \f. \g. \x. f(g(x)) in
  let double = \x. x * 2 in
  let addOne = \x. x + 1 in
  let addOneThenDouble = compose(double)(addOne) in
  let result = addOneThenDouble(5) in
  print(show(result))  -- Prints: 12

-- Map over list
export func mapExample() -> () ! {IO} =
  let map = \f. \xs.
    match xs {
      [] => [],
      Cons(x, rest) => Cons(f(x), map(f)(rest))
    } in
  let square = \x. x * x in
  let nums = [1, 2, 3, 4, 5] in
  let squared = map(square)(nums) in
  print(show(squared))  -- Prints: [1, 4, 9, 16, 25]
```

### List Operations (Complete Example)

```ailang
module benchmark/solution

-- Recursive sum
export func sum(xs: List[int]) -> int =
  match xs {
    [] => 0,
    Cons(x, rest) => x + sum(rest)
  }

-- Recursive length
export func length[a](xs: List[a]) -> int =
  match xs {
    [] => 0,
    Cons(_, rest) => 1 + length(rest)
  }

-- Recursive filter
export func filter[a](pred: a -> bool, xs: List[a]) -> List[a] =
  match xs {
    [] => [],
    Cons(x, rest) =>
      if pred(x)
      then Cons(x, filter(pred)(rest))
      else filter(pred)(rest)
  }

export func main() -> () ! {IO} =
  let nums = [1, 2, 3, 4, 5] in
  let total = sum(nums) in
  let count = length(nums) in
  print("Sum: " ++ show(total) ++ ", Length: " ++ show(count))
```

## Conclusion

The v0.3.17 prompt improvements (httpRequest documentation) are good, but **not sufficient**. The main issues are:

1. **Missing examples for common patterns** (higher-order functions, JSON, lists)
2. **Prompt too long** (1237 lines, hard for models to navigate)
3. **Insufficient guidance on what AILANG does NOT have**

Addressing the 17 benchmarks with 0% success rate should significantly improve overall AILANG success rate from 31% to 45-50%+.
