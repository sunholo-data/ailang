---
layout: page
title: AI Prompt - testing_guide_ai
parent: AI Prompts
nav_order: 1
---

# AILANG Testing Guide for AI Agents

**How to write and use property-based tests for autonomous code synthesis**

---

## Quick Reference

### Test Syntax
```ailang
// Unit test (simple assertion)
test "name" = boolean_expression

// Property test (100 random cases)
property "name" (param: type, ...) = boolean_expression
```

### Running Tests
```bash
ailang test file.ail              # Human output
ailang test --format json file.ail # JSON for parsing
```

---

## When to Write Tests

### ✅ ALWAYS Write Tests For:
1. **Pure functions** - Properties are perfect
2. **Data transformations** - Test algebraic laws
3. **Parsing/serialization** - Round-trip properties
4. **Mathematical operations** - Commutativity, associativity, identity
5. **List/collection operations** - Length preservation, order properties

### ⚠️ Consider Tests For:
1. **Complex business logic** - Unit tests for specific cases
2. **Edge cases** - Boundary conditions
3. **Error handling** - Result/Option unwrapping

### ❌ Skip Tests For:
1. **Trivial assignments** - `let x = 42`
2. **Type definitions** - `type Point = { x: int, y: int }`
3. **Simple wrappers** - One-line delegations

---

## Property-Based Testing Patterns

### Pattern 1: Algebraic Laws

**Commutativity** (order doesn't matter):
```ailang
property "addition commutes" (x: int, y: int) =
  x + y == y + x

property "max commutes" (x: int, y: int) =
  max(x, y) == max(y, x)
```

**Associativity** (grouping doesn't matter):
```ailang
property "addition associates" (x: int, y: int, z: int) =
  (x + y) + z == x + (y + z)

property "list concat associates" (xs: list(int), ys: list(int), zs: list(int)) =
  (xs ++ ys) ++ zs == xs ++ (ys ++ zs)
```

**Identity** (neutral element):
```ailang
property "zero is additive identity" (x: int) =
  x + 0 == x && 0 + x == x

property "empty list is concat identity" (xs: list(int)) =
  xs ++ [] == xs && [] ++ xs == xs
```

**Inverse** (cancellation):
```ailang
property "subtraction inverts addition" (x: int, y: int) =
  (x + y) - y == x

property "reverse inverts itself" (xs: list(int)) =
  reverse(reverse(xs)) == xs
```

### Pattern 2: Invariant Preservation

**Length preservation**:
```ailang
property "map preserves length" (xs: list(int), f: int -> int) =
  length(map(f, xs)) == length(xs)

property "reverse preserves length" (xs: list(int)) =
  length(reverse(xs)) == length(xs)
```

**Order preservation**:
```ailang
property "filter preserves order" (xs: list(int), p: int -> bool) =
  isOrdered(xs) ==> isOrdered(filter(p, xs))

property "map preserves sorted" (xs: list(int), f: int -> int) =
  isSorted(xs) ==> isSorted(map(f, xs))
```

**Type preservation**:
```ailang
property "Option map preserves None" (f: int -> int) =
  map(f, None) == None

property "Result map preserves Err" (f: int -> int, err: string) =
  mapResult(f, Err(err)) == Err(err)
```

### Pattern 3: Round-Trip Properties

**Serialization/deserialization**:
```ailang
property "JSON round-trip" (x: int) =
  let json = toJSON(x) in
  fromJSON(json) == Ok(x)

property "string round-trip" (x: int) =
  let str = toString(x) in
  fromString(str) == Ok(x)
```

**Encoding/decoding**:
```ailang
property "base64 round-trip" (s: string) =
  decode(encode(s)) == s

property "compress round-trip" (data: bytes) =
  decompress(compress(data)) == data
```

**Construction/destruction**:
```ailang
property "list cons/head/tail" (x: int, xs: list(int)) =
  let list = x :: xs in
  head(list) == x && tail(list) == xs
```

### Pattern 4: Conditional Properties (Implications)

Use `==>` for preconditions:

```ailang
property "division by non-zero" (x: int, y: int) =
  y != 0 ==> (x / y) * y + (x % y) == x

property "non-empty list has head" (xs: list(int)) =
  length(xs) > 0 ==> head(xs) != null

property "positive numbers are >= 1" (x: int) =
  x > 0 ==> x >= 1
```

### Pattern 5: Functor/Monad Laws

**Functor laws**:
```ailang
// Identity: map(id, xs) = xs
property "map identity" (xs: list(int)) =
  map(\x. x, xs) == xs

// Composition: map(f ∘ g, xs) = map(f, map(g, xs))
property "map composition" (f: int -> int, g: int -> int, xs: list(int)) =
  let composed = \x. f(g(x)) in
  map(composed, xs) == map(f, map(g, xs))
```

**Monad laws** (if applicable):
```ailang
// Left identity: return(x) >>= f = f(x)
property "monad left identity" (x: int, f: int -> Option(int)) =
  flatMap(f, Some(x)) == f(x)

// Right identity: m >>= return = m
property "monad right identity" (mx: Option(int)) =
  flatMap(Some, mx) == mx

// Associativity: (m >>= f) >>= g = m >>= (\x. f(x) >>= g)
property "monad associativity" (mx: Option(int), f: int -> Option(int), g: int -> Option(int)) =
  let left = flatMap(g, flatMap(f, mx)) in
  let right = flatMap(\x. flatMap(g, f(x)), mx) in
  left == right
```

---

## Shrinking: Understanding Minimal Counterexamples

### How Shrinking Works

When a property fails, AILANG automatically finds the **minimal failing input**:

```ailang
property "all integers less than 100" (x: int) =
  x < 100
```

**Execution**:
1. **Generate**: `x = 523` (random)
2. **Test**: `523 < 100` → `false` ✗
3. **Shrink**: Try simpler values
   - `0 < 100` → `true` ✓
   - `261 < 100` → `false` ✗ (keep shrinking)
   - `130 < 100` → `false` ✗
   - `100 < 100` → `false` ✗ (minimal!)

**Result**: Minimal counterexample is `100`

### Shrinking Strategies

**Integers**: Binary search toward zero
```
1000 → 500 → 250 → 125 → 100
-500 → -250 → -125 → -100
```

**Strings**: Remove chunks, then characters
```
"hello world" → "" (passes)
"hello world" → "hello" (test)
"hello" → "hell" (test)
```

**Lists**: Remove elements, then shrink elements
```
[1, 2, 100, 4] → [] (passes)
[1, 2, 100, 4] → [100] (test)
[100] → [50] → [25] → ...
```

**ADTs**: Shrink fields independently
```
Some(100) → Some(0) (test)
Node(left, 100, right) → Node(left, 0, right) (test field)
```

### Writing Shrink-Friendly Properties

**✅ Good** (shrinking helps debug):
```ailang
property "numbers less than 100" (x: int) =
  x < 100
// Shrinks to: x = 100 (clear boundary)

property "list length under 50" (xs: list(int)) =
  length(xs) < 50
// Shrinks to: xs = [1,1,...] (50 elements)
```

**❌ Bad** (shrinking doesn't help):
```ailang
property "complex modulo" (x: int, y: int, z: int) =
  (x + y) * z % 17 == 0
// Shrinks to: x=0, y=0, z=0 (trivial, not informative)

property "hash collision" (s1: string, s2: string) =
  hash(s1) != hash(s2)
// Shrinks to: s1="", s2="" (not useful)
```

**Rule**: Properties should have clear failure boundaries that shrinking can find.

---

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Install AILANG
        run: |
          make install
          echo "$HOME/go/bin" >> $GITHUB_PATH

      - name: Run tests with JSON output
        run: ailang test --format json --no-color . > test-results.json

      - name: Parse and fail on errors
        run: |
          if jq -e '.summary.failed > 0' test-results.json; then
            echo "Tests failed!"
            exit 1
          fi

      - name: Upload results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: test-results
          path: test-results.json
```

### JSON Output Schema

```json
{
  "module": "string",
  "tests": [
    {
      "name": "string",
      "status": "pass" | "fail" | "skip",
      "duration": 0.001,
      "message": "optional error message"
    }
  ],
  "properties": [
    {
      "name": "string",
      "status": "pass" | "fail" | "skip",
      "cases": 100,
      "duration": 0.5,
      "counterexample": "optional minimal failing input"
    }
  ],
  "summary": {
    "total": 10,
    "passed": 8,
    "failed": 2,
    "skipped": 0,
    "duration": 1.5
  }
}
```

### Exit Codes

```bash
# Exit 0: All tests passed
ailang test .
echo $?  # 0

# Exit 1: Some tests failed
ailang test failing.ail
echo $?  # 1
```

---

## Common Mistakes to Avoid

### ❌ Mistake 1: Testing Implementation Details

```ailang
// BAD: Tests internal implementation
test "list uses cons cells" =
  let xs = [1, 2, 3] in
  isCons(xs) == true

// GOOD: Tests behavior
property "list preserves order" (xs: list(int)) =
  let indexed = zip(xs, [0..length(xs)]) in
  all(\(x, i). xs[i] == x, indexed)
```

### ❌ Mistake 2: Overly Specific Properties

```ailang
// BAD: Tests one case (not a property!)
property "specific addition" (x: int, y: int) =
  x == 5 && y == 3 ==> x + y == 8

// GOOD: Tests general law
property "addition commutes" (x: int, y: int) =
  x + y == y + x
```

### ❌ Mistake 3: Non-Deterministic Properties

```ailang
// BAD: Uses randomness inside property
property "random behavior" (x: int) =
  random() < 0.5 ==> x > 0

// GOOD: Deterministic check
property "absolute value is non-negative" (x: int) =
  abs(x) >= 0
```

### ❌ Mistake 4: Ignoring Preconditions

```ailang
// BAD: Will fail on y=0
property "division" (x: int, y: int) =
  (x / y) * y == x

// GOOD: Explicit precondition
property "division" (x: int, y: int) =
  y != 0 ==> (x / y) * y + (x % y) == x
```

### ❌ Mistake 5: Tautologies

```ailang
// BAD: Always true (useless)
property "tautology" (x: int) =
  x == x

property "obvious" (xs: list(int)) =
  length(xs) >= 0

// GOOD: Actually tests something
property "reverse involution" (xs: list(int)) =
  reverse(reverse(xs)) == xs
```

---

## Code Generation Checklist for AI Agents

When generating AILANG code, include tests following this checklist:

### Phase 1: Write Core Implementation
```ailang
// Implement function
let myFunction = \x y. implementation
```

### Phase 2: Add Unit Tests
```ailang
// Test specific cases
test "myFunction example 1" = myFunction(1, 2) == expected
test "myFunction example 2" = myFunction(0, 0) == expected
test "myFunction edge case" = myFunction(-1, 100) == expected
```

### Phase 3: Add Property Tests
```ailang
// Test general laws
property "myFunction commutes" (x: T, y: T) =
  myFunction(x, y) == myFunction(y, x)

property "myFunction associates" (x: T, y: T, z: T) =
  myFunction(myFunction(x, y), z) == myFunction(x, myFunction(y, z))
```

### Phase 4: Verify in CI
```bash
ailang test --format json --no-color file.ail
```

---

## Performance Considerations

### Test Execution Time

Property tests run **100 cases by default**:
- Fast properties: `<1s`
- Medium properties: `1-5s`
- Slow properties: `>5s`

**Optimization strategies:**
1. Reduce test cases: `export AILANG_TEST_RUNS=10`
2. Use smaller data: `export AILANG_TEST_MAX_SIZE=20`
3. Filter expensive cases: `expensive(x) ==> property`

### Shrinking Time

Shrinking can take time for complex failures:
- Simple types (int, string): Fast (`<1s`)
- Lists: Medium (`1-5s`)
- Trees/recursive: Slow (`>5s`)

**Bounded by 100 iterations max** to prevent infinite loops.

---

## Examples by Category

### Mathematical Functions
```ailang
property "sqrt squares" (x: float) =
  x >= 0 ==> abs(sqrt(x) * sqrt(x) - x) < 0.0001

property "log exp" (x: float) =
  abs(log(exp(x)) - x) < 0.0001
```

### String Operations
```ailang
property "toUpper is idempotent" (s: string) =
  toUpper(toUpper(s)) == toUpper(s)

property "split join" (s: string, delim: string) =
  delim != "" ==> join(split(s, delim), delim) == s
```

### List Operations
```ailang
property "sort is idempotent" (xs: list(int)) =
  sort(sort(xs)) == sort(xs)

property "filter reduces length" (xs: list(int), p: int -> bool) =
  length(filter(p, xs)) <= length(xs)
```

### Tree Operations
```ailang
property "tree size equals traversal length" (t: Tree) =
  length(inorder(t)) == size(t)

property "tree depth bounds size" (t: Tree) =
  size(t) >= depth(t)
```

---

## Summary: Key Principles

1. **Properties > Unit Tests** - More coverage, less code
2. **Test Laws, Not Examples** - Algebraic properties are robust
3. **Use Shrinking** - Minimal counterexamples are debuggable
4. **CI/CD Integration** - JSON output + exit codes
5. **Fast Feedback** - Properties should run in <5s

**Golden Rule**: If you can express it as an algebraic law, write a property. Otherwise, write a unit test.

---

## Resources

- **User Guide**: `docs/TESTING.md`
- **Examples**: `examples/testing_basic.ail`, `examples/testing_advanced.ail`
- **API**: `internal/testing` package
- **CI/CD**: See `docs/TESTING.md#cicd-integration`
