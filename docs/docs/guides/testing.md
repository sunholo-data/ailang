---
sidebar_position: 10
title: Testing Guide
description: Property-based testing for deterministic AI code synthesis in AILANG
---

# AILANG Testing Guide

**Property-based testing for deterministic AI code synthesis**

## Table of Contents
- [Quick Start](#quick-start)
- [Writing Tests](#writing-tests)
- [Property-Based Testing](#property-based-testing)
- [Running Tests](#running-tests)
- [CI/CD Integration](#cicd-integration)
- [Examples](#examples)
- [Advanced Topics](#advanced-topics)

---

## Quick Start

### Installation
```bash
# Clone and build
git clone https://github.com/sunholo/ailang
cd ailang
make install

# Verify installation
ailang test --help
```

### Your First Test
Create `hello_test.ail`:
```ailang
// Unit test (simple assertion)
test "addition works" = 1 + 1 == 2

// Property test (QuickCheck-style)
property "addition is commutative" (x: int, y: int) =
  x + y == y + x
```

Run it:
```bash
ailang test hello_test.ail
```

Output:
```
→ Running tests in hello_test.ail

Test Results
Module: All Tests

Tests:
  ✓ addition works

Properties:
  ✓ addition is commutative (100 cases)

✓ All tests passed

2 tests: 2 passed, 0 failed, 0 skipped (0.2s)
```

---

## Writing Tests

### Unit Tests

Unit tests are simple boolean assertions:

```ailang
test "name" = expression
```

**Examples:**
```ailang
// Basic assertions
test "integers equal" = 42 == 42
test "strings concat" = "hello" ++ " world" == "hello world"
test "lists append" = [1, 2] ++ [3] == [1, 2, 3]

// Function tests
test "map doubles list" =
  map(\x. x * 2, [1, 2, 3]) == [2, 4, 6]

// ADT tests
test "Some wraps value" =
  match Some(42) {
    | Some(x) -> x == 42
    | None -> false
  }
```

### Property Tests

Property tests verify invariants hold for many random inputs:

```ailang
property "name" (param: type, ...) = expression
```

**Examples:**
```ailang
// Commutativity
property "addition commutes" (x: int, y: int) =
  x + y == y + x

// Associativity
property "addition associates" (x: int, y: int, z: int) =
  (x + y) + z == x + (y + z)

// Identity
property "zero is additive identity" (x: int) =
  x + 0 == x && 0 + x == x

// Conditional properties (implications)
property "division by non-zero" (x: int, y: int) =
  y != 0 ==> (x / y) * y + (x % y) == x
```

**Supported Types:**
- `int` - Integers (configurable range)
- `float` - Floating-point numbers
- `bool` - Booleans
- `string` - Strings (configurable length/charset)
- `list(T)` - Lists of type T
- `Option(T)` - Optional values (Some/None)
- `Result(T, E)` - Results (Ok/Err)
- Custom ADTs and records

---

## Property-Based Testing

### How It Works

1. **Generation**: Create 100 random test cases
2. **Execution**: Run property on each case
3. **Shrinking**: If failure, find minimal counterexample

**Example:**
```ailang
property "all integers less than 100" (x: int) =
  x < 100
```

**Execution:**
```
→ Running property "all integers less than 100"
  Generated: -523, 17, 891, 42, ..., 234
  ✗ Failed on input: 234
  Shrinking... 234 → 117 → 100
  Minimal counterexample: 100

✗ Property failed: all integers less than 100
  Input: 100
  Expected: true
  Got: false
```

### Shrinking

When a property fails, shrinking finds the **minimal failing input**:

**Integer shrinking**: Binary search toward zero
```
1000 → 500 → 250 → 125 → 100 (minimal)
```

**List shrinking**: Remove elements, shrink elements
```
[1, 2, 100, 4, 5] → [100] → shrink 100 → [50] → ...
```

**String shrinking**: Remove chunks, characters
```
"hello world" → "hello" → "hell" → "hel" → ...
```

### Configuration

Customize generation with environment variables:

```bash
# Number of test cases (default: 100)
export AILANG_TEST_RUNS=1000

# Random seed (for reproducibility)
export AILANG_TEST_SEED=42

# Max size for collections (default: 100)
export AILANG_TEST_MAX_SIZE=50

# Integer range (default: -1000 to 1000)
export AILANG_TEST_MIN_INT=-100
export AILANG_TEST_MAX_INT=100
```

---

## Running Tests

### Command-Line Interface

```bash
# Run all tests in directory (recursive)
ailang test .

# Run tests in specific file
ailang test examples/testing_basic.ail

# Run tests in multiple files
ailang test file1.ail file2.ail

# Human-readable output (default)
ailang test --format human .

# JSON output (for CI/CD)
ailang test --format json .

# Disable colored output
ailang test --no-color .

# Show help
ailang test --help
```

### Output Formats

**Human (default)**:
```
→ Running tests in .

Test Results
Module: All Tests

Tests:
  ✓ addition works
  ✗ subtraction broken

Properties:
  ✓ addition commutes (100 cases)

──────────────────────────────────────────────────
✗ Some tests failed

2 tests: 1 passed, 1 failed, 0 skipped (0.5s)
  ✓ Passed: 1
  ✗ Failed: 1
```

**JSON** (`--format json`):
```json
{
  "module": "All Tests",
  "tests": [
    {
      "name": "addition works",
      "status": "pass",
      "duration": 0.001
    },
    {
      "name": "subtraction broken",
      "status": "fail",
      "message": "Expected true, got false",
      "duration": 0.001
    }
  ],
  "properties": [
    {
      "name": "addition commutes",
      "status": "pass",
      "cases": 100,
      "duration": 0.5
    }
  ],
  "summary": {
    "total": 2,
    "passed": 1,
    "failed": 1,
    "skipped": 0,
    "duration": 0.5
  }
}
```

---

## CI/CD Integration

### GitHub Actions

`.github/workflows/test.yml`:
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

      - name: Run tests
        run: |
          ailang test --format json --no-color . > test-results.json

      - name: Upload results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: test-results
          path: test-results.json

      - name: Check test status
        run: |
          if ! ailang test --format json --no-color . | jq -e '.summary.failed == 0'; then
            echo "Tests failed!"
            exit 1
          fi
```

### GitLab CI

`.gitlab-ci.yml`:
```yaml
stages:
  - test

test:
  stage: test
  image: golang:1.21
  before_script:
    - git clone https://github.com/sunholo/ailang
    - cd ailang && make install && cd ..
  script:
    - ailang test --format json --no-color . | tee test-results.json
  artifacts:
    reports:
      junit: test-results.json
    paths:
      - test-results.json
```

### Exit Codes

```bash
# Exit code 0: All tests passed
ailang test .
echo $?  # 0

# Exit code 1: Some tests failed
ailang test failing.ail
echo $?  # 1
```

### Pre-commit Hook

`.git/hooks/pre-commit`:
```bash
#!/bin/bash
# Run tests before commit

echo "Running tests..."
if ! ailang test --no-color .; then
    echo "Tests failed! Commit aborted."
    exit 1
fi

echo "Tests passed! Proceeding with commit."
exit 0
```

Make executable:
```bash
chmod +x .git/hooks/pre-commit
```

---

## Examples

### Example 1: List Properties

```ailang
// List reversal properties
property "reverse twice is identity" (xs: list(int)) =
  reverse(reverse(xs)) == xs

property "reverse preserves length" (xs: list(int)) =
  length(reverse(xs)) == length(xs)

property "reverse reverses order" (x: int, y: int) =
  reverse([x, y]) == [y, x]
```

### Example 2: Tree Properties

```ailang
type Tree =
  | Leaf(int)
  | Node(Tree, int, Tree)

property "tree depth is non-negative" (t: Tree) =
  depth(t) >= 0

property "tree size is positive" (t: Tree) =
  size(t) > 0

property "inorder traversal preserves size" (t: Tree) =
  length(inorder(t)) == size(t)
```

### Example 3: Conditional Properties

```ailang
// Division with preconditions
property "division identity" (x: int, y: int) =
  y != 0 ==> (x / y) * y + (x % y) == x

// List head with precondition
property "non-empty list has head" (xs: list(int)) =
  length(xs) > 0 ==> head(xs) != null

// Positive number properties
property "positive implies greater than zero" (x: int) =
  x > 0 ==> x >= 1
```

### Example 4: Algebraic Properties

```ailang
// Monoid laws
property "concatenation identity" (xs: list(int)) =
  (xs ++ [] == xs) && ([] ++ xs == xs)

property "concatenation associativity" (xs: list(int), ys: list(int), zs: list(int)) =
  (xs ++ ys) ++ zs == xs ++ (ys ++ zs)

// Functor laws
property "map identity" (xs: list(int)) =
  map(\x. x, xs) == xs

property "map composition" (f: int -> int, g: int -> int, xs: list(int)) =
  let composed = \x. f(g(x)) in
  map(composed, xs) == map(f, map(g, xs))
```

---

## Advanced Topics

### Custom Generators

For complex types, guide the generator with metadata:

```ailang
// @generator Tree: balanced tree with depth 0-5
property "balanced tree depth bound" (t: Tree) =
  depth(t) <= 5

// @generator Point: x,y in range -1000 to 1000
property "point distance non-negative" (p1: Point, p2: Point) =
  distance(p1, p2) >= 0.0
```

### Debugging Failed Properties

When a property fails, examine the minimal counterexample:

```ailang
property "controversial claim" (x: int) =
  x < 100
```

Output:
```
✗ Property failed: controversial claim
  Minimal counterexample: 100
  Expected: true
  Got: false

Shrinking path: 523 → 261 → 130 → 115 → 107 → 103 → 101 → 100
```

### Test Organization

Organize tests into files by category:

```
tests/
  ├── unit/
  │   ├── arithmetic.ail
  │   ├── strings.ail
  │   └── lists.ail
  ├── properties/
  │   ├── algebraic.ail
  │   ├── functor_laws.ail
  │   └── monad_laws.ail
  └── integration/
      ├── effects.ail
      └── modules.ail
```

Run all tests:
```bash
ailang test tests/
```

### Performance Testing

Test asymptotic behavior:

```ailang
property "map is O(n)" (xs: list(int)) =
  let doubled = map(\x. x * 2, xs) in
  length(doubled) == length(xs)

property "sort preserves length" (xs: list(int)) =
  length(sort(xs)) == length(xs)
```

---

## For AI Agents: Testing Best Practices

### 1. Property > Unit Test
**Prefer properties when possible:**
```ailang
// Weak: Only tests one case
test "addition example" = 2 + 3 == 5

// Strong: Tests 100 cases
property "addition commutes" (x: int, y: int) =
  x + y == y + x
```

### 2. Shrinking-Friendly Properties
**Write properties that shrink well:**
```ailang
// Bad: Shrinking won't help
property "complex condition" (x: int, y: int, z: int) =
  (x + y) * z % 7 == 0  // Arbitrary, hard to debug

// Good: Clear invariant
property "addition preserves ordering" (x: int, y: int) =
  x < y ==> x + 1 <= y + 1
```

### 3. Test Algebraic Laws
**Use mathematical properties:**
```ailang
// Commutativity
property "commutes" (x: T, y: T) = op(x, y) == op(y, x)

// Associativity
property "associates" (x: T, y: T, z: T) =
  op(op(x, y), z) == op(x, op(y, z))

// Identity
property "identity" (x: T) = op(x, identity) == x
```

### 4. Use Conditional Properties
**Test preconditions explicitly:**
```ailang
property "division correctness" (x: int, y: int) =
  y != 0 ==> (x / y) * y + (x % y) == x
```

### 5. CI/CD Integration Checklist
- Use `--format json --no-color` for CI
- Check exit code: `0` = pass, `1` = fail
- Upload test artifacts for debugging
- Set appropriate timeouts (properties can be slow)
- Use `AILANG_TEST_SEED` for reproducibility

---

## Resources

- **Examples**: `examples/testing_basic.ail`, `examples/testing_advanced.ail`
- **API Docs**: See `internal/testing` package documentation
- **Source**: https://github.com/sunholo/ailang
- **Issues**: https://github.com/sunholo/ailang/issues
