# Inline Tests in AILANG

Inline tests are a powerful way to specify expected behavior directly in your function definitions. This guide covers syntax, patterns, edge cases, and best practices.

## What Are Inline Tests?

Inline tests are test cases embedded in function signatures using the `tests [...]` clause. They run automatically and verify your function produces expected outputs for given inputs.

```ailang
pure func double(x: int) -> int
  tests [
    (5, 10),
    (0, 0),
    (-3, -6)
  ]
{
  x * 2
}
```

**Benefits:**
- ✅ Tests live next to implementation (single source of truth)
- ✅ Automatic execution during compilation
- ✅ Provides training data for AI code generation
- ✅ Documents expected behavior inline
- ✅ Lightweight and readable

**Limitations:**
- ❌ Only work with block-style functions `{ }`
- ❌ Cannot use expression-style `=` syntax
- ❌ File-level `test "name" { }` blocks are skipped

---

## Basic Syntax

### Block-Style Functions (✅ Works)

Inline tests work with block-style function definitions:

```ailang
pure func functionName(arg1: Type1, arg2: Type2) -> ReturnType
  tests [
    ((input1, input2), expected_output1),
    ((input3, input4), expected_output2)
  ]
{
  // function body
}
```

**Format:**
- `tests [...]` clause comes AFTER function signature
- Each test is a tuple: `(input, expected)`
- Multiple tests in square brackets, comma-separated
- Function body in curly braces `{ }`

### Expression-Style Functions (❌ Doesn't Work)

Expression-style functions do NOT support inline tests:

```ailang
// ❌ WRONG - This syntax is not supported
pure func double(x: int) -> int =
  tests [
    (5, 10)
  ]
  x * 2

// ✅ CORRECT - Use block-style instead
pure func double(x: int) -> int
  tests [
    (5, 10)
  ]
{
  x * 2
}
```

---

## Test Patterns

### Pattern 1: Nullary Functions (No Arguments)

Functions with zero arguments use an empty tuple `()` as input:

```ailang
pure func getConstant() -> int
  tests [
    ((), 42)
  ]
{
  42
}

pure func getCurrentYear() -> int
  tests [
    ((), 2025)
  ]
{
  2025
}
```

**Format:** `((), expected_value)`

**Key Point:** Even with no arguments, tests are still tuples with empty tuple as input.

### Pattern 2: Single-Argument Functions

Single-argument functions use the value directly as input:

```ailang
pure func double(x: int) -> int
  tests [
    (5, 10),
    (0, 0),
    (-3, -6)
  ]
{
  x * 2
}

pure func absolute(x: int) -> int
  tests [
    (5, 5),
    (-5, 5),
    (0, 0)
  ]
{
  if x < 0 then 0 - x else x
}

pure func negate(x: int) -> int
  tests [
    (5, -5),
    (-5, 5),
    (0, 0)
  ]
{
  0 - x
}
```

**Format:** `(input_value, expected_value)`

**Key Point:** Single arguments are NOT wrapped in extra parentheses.

### Pattern 3: Multi-Argument Functions (Tuple Format)

Multi-argument functions use nested tuples: `((arg1, arg2, ...), expected)`:

```ailang
pure func add(a: int, b: int) -> int
  tests [
    ((3, 5), 8),
    ((0, 0), 0),
    ((-1, 1), 0),
    ((100, 200), 300)
  ]
{
  a + b
}

pure func multiply(a: int, b: int) -> int
  tests [
    ((3, 4), 12),
    ((0, 5), 0),
    ((-3, -4), 12)
  ]
{
  a * b
}
```

**Format:** `((arg1, arg2), expected)`

**Key Points:**
- Arguments grouped in tuple `(arg1, arg2)`
- That tuple is the first element of the test tuple
- Expected result is second element

### Pattern 4: Three-or-More Arguments

Same pattern extends to any number of arguments:

```ailang
pure func mixedOps(a: int, b: int, c: int) -> int
  tests [
    ((2, 3, 4), 14),  // 2 + 3*4 = 14
    ((1, 2, 3), 7),   // 1 + 2*3 = 7
    ((5, 0, 3), 3)    // 5 + 0*3 = 3
  ]
{
  a + b * c
}

pure func threeWayMax(a: int, b: int, c: int) -> int
  tests [
    ((5, 3, 7), 7),
    ((10, 10, 10), 10),
    ((1, 2, 3), 3)
  ]
{
  if a >= b && a >= c then a
  else if b >= c then b
  else c
}
```

**Format:** `((arg1, arg2, arg3, ...), expected)`

---

## Working with Different Types

### Integers

```ailang
pure func addInts(a: int, b: int) -> int
  tests [
    ((5, 3), 8),
    ((0, 0), 0),
    ((-5, 3), -2)
  ]
{
  a + b
}
```

### Floats

```ailang
pure func addFloats(a: float, b: float) -> float
  tests [
    ((1.5, 2.5), 4.0),
    ((0.0, 0.0), 0.0),
    ((-1.5, 1.5), 0.0)
  ]
{
  a + b
}
```

**Note on Float Precision:** Floating-point arithmetic can have precision issues. Simple tests like `0.1 + 0.2 = 0.3` may fail due to binary representation.

### Strings

```ailang
pure func repeatString(s: string, n: int) -> string
  tests [
    (("hi", 3), "hihihi"),
    (("x", 1), "x"),
    (("a", 0), "")
  ]
{
  // Implementation would use string concatenation
  s
}
```

### Booleans

```ailang
pure func isEven(x: int) -> bool
  tests [
    (4, true),
    (3, false),
    (0, true)
  ]
{
  x % 2 == 0
}
```

### Algebraic Data Types (ADTs)

```ailang
type Color = Red | Green | Blue

pure func isRed(c: Color) -> bool
  tests [
    (Red, true),
    (Green, false),
    (Blue, false)
  ]
{
  match c {
    Red => true,
    Green => false,
    Blue => false
  }
}

pure func colorToString(c: Color) -> string
  tests [
    (Red, "red"),
    (Green, "green"),
    (Blue, "blue")
  ]
{
  match c {
    Red => "red",
    Green => "green",
    Blue => "blue"
  }
}
```

---

## Edge Cases

### Float Precision

Floating-point arithmetic can have precision issues:

```ailang
// ⚠️ CAUTION: Precision issues with binary representation
pure func problematicFloat(a: float, b: float) -> float
  tests [
    ((0.1, 0.2), 0.3)  // May fail! 0.1 + 0.2 ≠ 0.3 in binary
  ]
{
  a + b
}

// ✅ BETTER: Test with values that are exact in binary
pure func safeFloat(a: float, b: float) -> float
  tests [
    ((0.5, 0.25), 0.75),   // Exact in binary
    ((1.5, 2.5), 4.0),     // Exact in binary
    ((0.0, 0.0), 0.0)
  ]
{
  a + b
}
```

### Recursive Functions

Test recursive functions with both base cases and recursive cases:

```ailang
pure func fibonacci(n: int) -> int
  tests [
    (0, 0),      // Base case: fib(0) = 0
    (1, 1),      // Base case: fib(1) = 1
    (5, 5),      // fib(5) = 5
    (6, 8),      // fib(6) = 8
    (10, 55)     // fib(10) = 55
  ]
{
  if n <= 1 then n else fibonacci(n-1) + fibonacci(n-2)
}

pure func factorial(n: int) -> int
  tests [
    (0, 1),       // 0! = 1
    (1, 1),       // 1! = 1
    (5, 120),     // 5! = 120
    (6, 720)      // 6! = 720
  ]
{
  if n <= 1 then 1 else n * factorial(n - 1)
}
```

### Pattern Matching

Test functions that use pattern matching:

```ailang
type Tree = Leaf(int) | Node(Tree, int, Tree)

pure func treeSize(t: Tree) -> int
  tests [
    (Leaf(5), 1),
    (Node(Leaf(1), 2, Leaf(3)), 3)
  ]
{
  match t {
    Leaf(_) => 1,
    Node(left, _, right) => 1 + treeSize(left) + treeSize(right)
  }
}
```

### List Operations

```ailang
pure func listLength(xs: [int]) -> int
  tests [
    ([], 0),
    ([1], 1),
    ([1, 2, 3], 3)
  ]
{
  // Implementation using recursion or builtin
  let rec len = (lst: [int], acc: int) -> int =>
    match lst {
      [] => acc,
      _ :: rest => len(rest, acc + 1)
    }
  in
    len(xs, 0)
}
```

---

## Best Practices

### 1. Test Coverage Guidelines

**Every function should have at least 3 tests:**

```ailang
// ✅ GOOD: Tests cover multiple cases
pure func absolute(x: int) -> int
  tests [
    (5, 5),       // Positive
    (-5, 5),      // Negative
    (0, 0)        // Edge case: zero
  ]
{
  if x < 0 then 0 - x else x
}

// ⚠️ INCOMPLETE: Only one test
pure func absolute(x: int) -> int
  tests [
    (5, 5)
  ]
{
  if x < 0 then 0 - x else x
}
```

### 2. Include Edge Cases

**Test boundaries and special values:**

```ailang
pure func divide(a: int, b: int) -> int
  tests [
    ((10, 2), 5),      // Normal case
    ((0, 5), 0),       // Zero numerator
    ((5, 1), 5),       // Divisor of 1
    ((5, 5), 1),       // Equal values
    ((10, 3), 3)       // Integer division
  ]
{
  a / b
}
```

### 3. Use Descriptive Comments

```ailang
pure func gcd(a: int, b: int) -> int
  tests [
    ((48, 18), 6),   // 48 = 6*8, 18 = 6*3, gcd = 6
    ((7, 7), 7),     // Same values
    ((0, 5), 5),     // Zero case
    ((100, 50), 50)  // Power of 2 relationship
  ]
{
  if b == 0 then a else gcd(b, a % b)
}
```

### 4. Organize by Category

Group tests logically within the test list:

```ailang
pure func compare(a: int, b: int) -> int
  tests [
    // Equal values
    ((5, 5), 0),
    ((0, 0), 0),

    // First greater
    ((10, 3), 1),
    ((1, -1), 1),

    // Second greater
    ((3, 10), -1),
    ((-1, 1), -1)
  ]
{
  if a > b then 1 else if a < b then -1 else 0
}
```

### 5. Performance Considerations

**Keep tests fast** - avoid expensive computations:

```ailang
// ✅ GOOD: Quick tests
pure func isEven(n: int) -> int
  tests [
    (4, true),
    (7, false)
  ]
{
  n % 2 == 0
}

// ⚠️ SLOW: Expensive computation in test
pure func slowFunction(n: int) -> int
  tests [
    (10000, 1000000)  // Slow!
  ]
{
  // Expensive computation
  n
}
```

---

## Common Pitfalls

### Pitfall 1: Expression-Style Syntax

```ailang
// ❌ WRONG: Expression-style doesn't support tests
pure func double(x: int) -> int =
  tests [
    (5, 10)
  ]
  x * 2

// ✅ CORRECT: Use block-style
pure func double(x: int) -> int
  tests [
    (5, 10)
  ]
{
  x * 2
}
```

### Pitfall 2: Missing Tuple Nesting for Multiple Arguments

```ailang
// ❌ WRONG: Arguments not wrapped in tuple
pure func add(a: int, b: int) -> int
  tests [
    (3, 5, 8)      // Missing parentheses around (3, 5)
  ]
{
  a + b
}

// ✅ CORRECT: Arguments wrapped in tuple
pure func add(a: int, b: int) -> int
  tests [
    ((3, 5), 8)    // Arguments in nested tuple
  ]
{
  a + b
}
```

### Pitfall 3: Wrong Format for Nullary Functions

```ailang
// ❌ WRONG: Forgot empty tuple
pure func getConstant() -> int
  tests [
    (42)           // Should be ((), 42)
  ]
{
  42
}

// ✅ CORRECT: Use empty tuple as input
pure func getConstant() -> int
  tests [
    ((), 42)
  ]
{
  42
}
```

### Pitfall 4: Type Mismatches

```ailang
// ❌ WRONG: Expected value has wrong type
pure func double(x: int) -> int
  tests [
    (5, 10.0)      // Expected int, got float
  ]
{
  x * 2
}

// ✅ CORRECT: Types match
pure func double(x: int) -> int
  tests [
    (5, 10)        // Both int
  ]
{
  x * 2
}
```

### Pitfall 5: File-Level Tests Don't Run

```ailang
// ❌ SKIPPED: File-level tests are not executed
test "verify add function" {
  add(5, 3) == 8
}

// ✅ USE INLINE TESTS INSTEAD
pure func add(a: int, b: int) -> int
  tests [
    ((5, 3), 8)
  ]
{
  a + b
}
```

---

## Advanced Examples

### Testing Polymorphic Functions

```ailang
// Identity function works with any type
pure func identity[T](x: T) -> T
  tests [
    (5, 5),
    ("hello", "hello")
  ]
{
  x
}
```

### Testing Functions with Effects

```ailang
// Only pure functions can have inline tests
// (Functions with ! effects cannot be tested inline)
pure func validateEmail(email: string) -> bool
  tests [
    ("user@example.com", true),
    ("invalid-email", false)
  ]
{
  // Email validation logic
  email
}
```

### Testing with Complex Data Structures

```ailang
type Person = { name: string, age: int }

pure func canVote(p: Person) -> bool
  tests [
    ({ name = "Alice", age = 21 }, true),
    ({ name = "Bob", age = 17 }, false)
  ]
{
  p.age >= 18
}
```

---

## Testing Workflow

### 1. Run Tests During Development

```bash
# Tests run automatically during compilation
ailang run --entry main my_file.ail

# Check specific file
ailang check my_file.ail
```

### 2. Verify Examples

```bash
# Run all example files (including inline tests)
make verify-examples

# Check specific example
ailang check examples/tests/inline_tests_arithmetic.ail
```

### 3. Test Coverage

```bash
# Run full test suite
make test

# Check coverage
make test-coverage
```

---

## Resources

- **[AILANG Teaching Prompt](../prompts/)** - Complete language syntax guide
- **[Examples Directory](../examples/tests/)** - Runnable inline test examples
- **[Testing Guide](testing.md)** - General testing documentation
- **[LIMITATIONS.md](../LIMITATIONS.md)** - Known issues and workarounds

---

## Summary

Inline tests provide a lightweight, effective way to:
- ✅ Specify expected behavior in function definitions
- ✅ Document functions with examples
- ✅ Ensure functions work correctly
- ✅ Provide training data for AI code generation

**Remember:**
- Use block-style functions `{ }` for tests
- Follow the tuple format: `(input, expected)`
- Include edge cases and boundary values
- Keep tests fast and focused
- Comment tests with expected outcomes
