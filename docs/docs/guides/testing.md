---
sidebar_position: 10
title: Testing Guide
description: Property-based testing for deterministic AI code synthesis in AILANG
reviewed: 2026-10-09
reviewBy: 2026-11-16
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
- [Debugging Failed Properties](#debugging-failed-properties)

---

## Named-test and property purity

Named-test bodies and `forall` property predicates are checked pure. All module-level
helpers remain visible, including plain `func` and `export func` declarations without
an effect annotation. An unused effectful helper is retained as a closure and does not
run merely because the module is tested.

```ailang
module helper_test
export func inc(x: int) -> int { x + 1 }
test "plain export" { inc(1) == 2 }
```

Calling a helper declared with `! {FS}`, or calling `readFile` directly, fails with
`Missing effects: FS` and the original test/property name and location. Test bodies
cannot grant capabilities. Verify effectful behavior through an exported entry instead:

```bash
ailang run --caps FS,IO examples/tests/effectful_fixture_verification.ail
```

The example reads a tracked fixture and prints its verification result; FS grants file
access and IO grants output. For deterministic named tests, use pure checks over
byte-mirror fixtures. See `examples/tests/named_helper_purity.ail` for a plain export,
a seeded property, and an unused FS helper. Invalid helper declarations still report
the compiler's module error rather than a test-body purity diagnostic.

## Quick Start

### Installation

Install the `ailang` binary as described in [Getting Started](/docs/guides/getting-started), then check it:

```bash
ailang test --help
```

### Your First Test
Create `hello_test.ail`:
```ailang
module hello_test

-- Unit test: the body must evaluate to true
test "addition works" { 1 + 1 == 2 }

-- Property test (QuickCheck-style): 100 generated cases
property "addition is commutative" {
  forall(x: int, y: int) => x + y == y + x
}
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
  ✓ addition works (89.5µs)
      at hello_test.ail:4:1

Properties:
  ✓ addition is commutative (100 cases, 413.625µs)
      at hello_test.ail:7:1

──────────────────────────────────────────────────
✓ All tests passed!

2 tests: 2 passed, 0 failed, 0 skipped (503.125µs)
  ✓ Passed: 2
  ✗ Failed: 0

Seed:
  mode: derived
  master seed: 0
  derivation: ailang-property-seed-v1
  replay: ailang test --seed 0 hello_test.ail
```

---

## Writing Tests

### Unit Tests

A unit test is a named block whose body must evaluate to `true`:

```text
test "name" { expression }
```

**Examples:**
```ailang
import std/list (map)
import std/option (Option, Some, None)

-- Basic assertions
test "integers equal" { 42 == 42 }
test "string interpolation" { "hello ${"world"}" == "hello world" }
test "lists append" { [1, 2] ++ [3] == [1, 2, 3] }

-- Function tests
test "map doubles list" { map(\x. x * 2, [1, 2, 3]) == [2, 4, 6] }

-- ADT tests
test "Some wraps value" {
  match Some(42) {
    Some(x) => x == 42,
    None => false
  }
}
```

`++` concatenates lists only; for strings use `"${…}"` interpolation, `concat` or `join`.

### Inline Tests

A function can carry its own `(input, expected)` cases in a `tests [ … ]` list between the signature and the body. A function of several parameters takes its input as a tuple:

```ailang
export pure func square(x: int) -> int
  tests [
    (0, 0),
    (3, 9),
    (-4, 16)
  ]
{
  x * x
}

export pure func add(a: int, b: int) -> int
  tests [
    ((1, 2), 3),
    ((-1, 1), 0)
  ]
{
  a + b
}
```

Each case runs as its own test, named `<function>_test_<n>` (`square_test_1`, `add_test_2`, …). See `examples/snippets/v3_3/math/gcd.ail` and `examples/inline_tests_*.ail` for more.

### Property Tests

Property tests verify that an invariant holds for many generated inputs:

```ailang
property "name" {
  forall(param: type, ...) => predicate
}
```

The predicate can call the module's pure functions. Each property is compiled once, as a function of its binders, and called with each of 100 generated cases. A failing case is shrunk to a minimal counterexample: `forall(n: int) => n < 5` reports `property failed on input: [5]`.

**Examples:**
```ailang
import std/list (length, reverse)

export pure func double(x: int) -> int = x * 2

-- Commutativity
property "addition commutes" {
  forall(x: int, y: int) => x + y == y + x
}

-- Calling module functions
property "double is additive" {
  forall(n: int) => double(n) == n + n
}

-- A precondition: there is no `where`/`==>`, so write it as an if
property "division by non-zero" {
  forall(x: int, y: int) => if y == 0 then true else (x / y) * y + (x % y) == x
}

-- Lists
property "reverse keeps length" {
  forall(xs: [int]) => length(reverse(xs)) == length(xs)
}
```

**Binder types with a generator:**
- `int`, `float`, `bool`, `string`, `()`
- lists (`[T]`), tuples (`(A, B)`) and records (`{name: string, age: int}`) of these
- ADTs declared in the same module (`type Shape = Circle(float) | Square(float)`)

A binder whose type has no generator (for example an imported `Option[int]`) skips that property with `no generator for type …`.


---

## Property-Based Testing

### How It Works

1. **Generation**: each binder gets a value from its type's generator, for 100 cases (a fixed count).
2. **Execution**: the predicate runs on each case; the first `false` stops the property.
3. **Shrinking**: the failing input is shrunk to a minimal counterexample, which is what gets reported.

**Example:**
```ailang
property "all integers less than 100" {
  forall(x: int) => x < 100
}
```

**Output** (`ailang test --no-color`, trimmed):
```
Properties:
  ✗ all integers less than 100 (2 cases, 109.792µs)
      property failed on input: [100]
      at fail_test.ail:3:1
```

`(2 cases)` is how many cases ran before the failure. The input list holds one value per binder, in order. The intermediate shrink steps are not printed; only the minimal input is.

### Shrinking

Shrinking repeatedly replaces a binder with a simpler value on which the predicate is still `false`, until nothing simpler fails:

- **int**: toward zero (tries `0`, then a binary search, then `n - 1`), so `x < 100` reports `100`.
- **float**: toward `0.0`, by halving.
- **string**: by removing characters (empty string, halves, single characters). Characters themselves are not simplified.
- **list**: by removing elements (empty list, halves, single elements), then by shrinking the first few elements. `forall(xs: [int]) => length(xs) < 3` reports `[[0, 0, 0]]`.

`bool`, `()` and tuples are not shrunk, and an ADT counterexample may come back unshrunk.

### Integer and Size Ranges

Generated `int`s lie in `-1000..1000` and `float`s in `-1000.0..1000.0`; strings and lists of scalars have at most 100 elements. These ranges and the 100-case count are fixed: there are no flags or environment variables to change them.

### Seeds and Replay

Property generation is deterministic. By default each property's seed is derived from a master seed of `0`, so the same file produces the same cases every run. Every run ends by printing the seed and a replay command:

```
Seed:
  mode: derived
  master seed: 0
  derivation: ailang-property-seed-v1
  replay: ailang test --seed 0 fail_test.ail
```

- `--seed N` sets the master seed (a signed int64).
- `--random-seed` draws a fresh master seed and prints it for replay:

```
Seed:
  mode: master
  master seed: -6170952522490737774
  derivation: ailang-property-seed-v1
  replay: ailang test --seed -6170952522490737774 fail_test.ail
```

---

## Running Tests

### Command-Line Interface

```bash
# Run all tests in the current directory (recursive)
ailang test

# Run all tests in a directory (recursive)
ailang test tests/

# Run tests in a specific file
ailang test examples/inline_tests_arithmetic.ail

# Run tests in several files (one summary)
ailang test file1.ail file2.ail

# JSON output (for CI/CD)
ailang test --json .

# Disable colored output
ailang test --no-color .

# Package mode: run every *_test.ail found via ailang.toml
ailang test --package .

# Show all flags
ailang test --help
```

### Output Formats

For this file:

```ailang
test "addition works" { 1 + 1 == 2 }
test "subtraction broken" { 5 - 3 == 1 }

property "addition commutes" {
  forall(x: int, y: int) => x + y == y + x
}
```

**Human (default)**:
```
→ Running tests in json_test.ail

Test Results
Module: All Tests

Tests:
  ✓ addition works (89.666µs)
      at json_test.ail:3:1
  ✗ subtraction broken (75.375µs)
      expected true, got false
      at json_test.ail:4:1

Properties:
  ✓ addition commutes (100 cases, 436.792µs)
      at json_test.ail:6:1

──────────────────────────────────────────────────
✗ Some tests failed

3 tests: 2 passed, 1 failed, 0 skipped (601.833µs)
  ✓ Passed: 2
  ✗ Failed: 1

Seed:
  mode: derived
  master seed: 0
  derivation: ailang-property-seed-v1
  replay: ailang test --seed 0 json_test.ail
```

**JSON** (`--json`; the `→ Running tests` line goes to stderr, so stdout is pure JSON):
```json
{
  "failed_tests": 1,
  "module_path": "All Tests",
  "passed_tests": 2,
  "properties": [
    {
      "discarded_inputs": 0,
      "duration": "432.958µs",
      "generated_inputs": 100,
      "location": "json_test.ail:6:1",
      "name": "addition commutes",
      "seed": "279323736052768828",
      "skip_kind": "",
      "status": "pass",
      "tests_run": 100
    }
  ],
  "seed": "0",
  "seed_derivation": "ailang-property-seed-v1",
  "seed_mode": "derived",
  "skipped_tests": 0,
  "success": false,
  "tests": [
    {
      "duration": "136.125µs",
      "location": "json_test.ail:3:1",
      "name": "addition works",
      "status": "pass"
    },
    {
      "duration": "78.917µs",
      "error": "expected true, got false",
      "location": "json_test.ail:4:1",
      "name": "subtraction broken",
      "status": "fail"
    }
  ],
  "total_duration": "648µs",
  "total_tests": 3,
  "vacuous_skips": 0
}
```

### How named tests are compiled

All the `test "…" { … }` blocks in a file are compiled together, once, and each test then runs on its own fresh evaluator, or on a fresh VM under `--bytecode`. A file's test run therefore costs about one compile however many tests it has. A runtime error inside a test reports a position in your file; an error inside the test body itself is reported as `<file>:<line> (test body)`.

Sometimes the shared compile fails, usually because one test body does not type-check. In that case `ailang test` prints, on stderr:

```
→ named tests in sim/x_test.ail: could not share one compile (<reason>); compiled each test separately
```

It then compiles each test on its own, so only the broken test fails, with its own error. `--json` reports the same thing under `named_test_batch_failures`. If the notice adds that every body compiles on its own, the fault is in the test harness, not your code; please report it.

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
      - uses: actions/checkout@v4

      - uses: sunholo-data/setup-ailang@v1
        with:
          version: latest
          github-token: ${{ github.token }}

      - name: Run tests
        run: ailang test --json . > test-results.json

      - name: Upload results
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: test-results
          path: test-results.json
```

The `Run tests` step fails the job on its own, because `ailang test` exits non-zero when a test fails. To inspect the JSON yourself, `jq -e '.success'` exits non-zero unless every test passed.

### Other CI Systems

Install with the `install.sh` one-liner from [Getting Started](/docs/guides/getting-started), then run `ailang test --json .`. The JSON is AILANG's own shape, not JUnit XML, so do not declare it as a JUnit report.

### Exit Codes

| Exit code | Meaning |
|-----------|---------|
| `0` | All tests passed |
| `1` | At least one test failed |
| `1` | Every test was skipped (for example, no binder had a generator); pass `--allow-skips` to exit `0` instead |

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
import std/list (length, reverse)

property "reverse twice is identity" {
  forall(xs: [int]) => reverse(reverse(xs)) == xs
}

property "reverse preserves length" {
  forall(xs: [int]) => length(reverse(xs)) == length(xs)
}

property "reverse reverses order" {
  forall(x: int, y: int) => reverse([x, y]) == [y, x]
}
```

### Example 2: Tree Properties

A binder can have an ADT type declared in the same module, including a recursive one:

```ailang
import std/list (length)

type Tree = Leaf(int) | Node(Tree, int, Tree)

export pure func size(t: Tree) -> int {
  match t {
    Leaf(_) => 1,
    Node(l, _, r) => size(l) + 1 + size(r)
  }
}

export pure func depth(t: Tree) -> int {
  match t {
    Leaf(_) => 0,
    Node(l, _, r) => {
      let dl = depth(l);
      let dr = depth(r);
      1 + (if dl > dr then dl else dr)
    }
  }
}

export pure func inorder(t: Tree) -> [int] {
  match t {
    Leaf(v) => [v],
    Node(l, v, r) => inorder(l) ++ [v] ++ inorder(r)
  }
}

property "tree depth is non-negative" {
  forall(t: Tree) => depth(t) >= 0
}

property "tree size is positive" {
  forall(t: Tree) => size(t) > 0
}

property "inorder traversal preserves size" {
  forall(t: Tree) => length(inorder(t)) == size(t)
}
```

### Example 3: Conditional Properties

There is no implication operator and no `where` clause. Write a precondition as an `if` whose other branch is `true`:

```ailang
-- Division, for non-zero divisors
property "division identity" {
  forall(x: int, y: int) => if y == 0 then true else (x / y) * y + (x % y) == x
}

-- The same shape with `not`
property "positive implies at least one" {
  forall(x: int) => if not (x > 0) then true else x >= 1
}
```

Cases the precondition rules out still count toward the 100; they simply pass.

### Example 4: Algebraic Properties

```ailang
import std/list (map)

-- Monoid laws
property "concatenation identity" {
  forall(xs: [int]) => (xs ++ [] == xs) && ([] ++ xs == xs)
}

property "concatenation associativity" {
  forall(xs: [int], ys: [int], zs: [int]) => (xs ++ ys) ++ zs == xs ++ (ys ++ zs)
}

-- Functor laws
property "map identity" {
  forall(xs: [int]) => map(\x. x, xs) == xs
}

property "map composition" {
  forall(xs: [int]) => {
    let f = \x. x + 1;
    let g = \x. x * 2;
    map(\x. f(g(x)), xs) == map(f, map(g, xs))
  }
}
```

Function-typed binders have no generator (`forall(f: int -> int, …)` is skipped with `no generator for type (int -> int)`), so fix the functions inside the predicate as above.

---

## Debugging Failed Properties

A failure prints the minimal counterexample, one value per binder:

```
  ✗ all integers less than 100 (2 cases, 109.792µs)
      property failed on input: [100]
```

To reproduce it, re-run with the `replay:` command from the `Seed:` block, for example `ailang test --seed 0 fail_test.ail`. The same seed generates the same cases. Then copy the counterexample into a unit test so it stays fixed:

```ailang
test "regression: 100 is not less than 100" { not (100 < 100) }
```

A property that reports `⊘ … no generator for type …` was skipped, not passed. Change the binder to a type with a generator, or declare the ADT in the same module.

---

## For AI Agents: Testing Best Practices

### 1. Property > Unit Test
**Prefer properties when possible:**
```ailang
-- Weak: only tests one case
test "addition example" { 2 + 3 == 5 }

-- Strong: tests 100 generated cases
property "addition commutes" {
  forall(x: int, y: int) => x + y == y + x
}
```

### 2. Shrinking-Friendly Properties
**State a clear invariant**, so the minimal counterexample says what broke:
```ailang
property "adding one preserves ordering" {
  forall(x: int, y: int) => if x < y then x + 1 < y + 1 else true
}
```

### 3. Test Algebraic Laws
**Use mathematical properties:**
```ailang
export pure func maxInt(a: int, b: int) -> int = if a > b then a else b

-- Commutativity
property "max commutes" {
  forall(x: int, y: int) => maxInt(x, y) == maxInt(y, x)
}

-- Associativity
property "max associates" {
  forall(x: int, y: int, z: int) => maxInt(maxInt(x, y), z) == maxInt(x, maxInt(y, z))
}

-- Idempotence
property "max is idempotent" {
  forall(x: int) => maxInt(x, x) == x
}
```

### 4. Write Preconditions as `if`
```ailang
property "division correctness" {
  forall(x: int, y: int) => if y == 0 then true else (x / y) * y + (x % y) == x
}
```

### 5. CI/CD Integration Checklist
- Use `--json` (and `--no-color` for human output in logs)
- Check the exit code: `0` = pass, non-zero = failure, or every test skipped
- Upload the JSON as an artifact for debugging
- Record the `seed` from the JSON (or the `replay:` line) to reproduce a failure

---

## See Also

- [Development Workflow](/docs/guides/development-workflow) - Tests are part of sprint execution
- [Debugging Guide](/docs/guides/debugging) - Debug flags for troubleshooting test failures
- [Module Execution](/docs/guides/module_execution) - Running modules with effects
- [Language Syntax](/docs/reference/language-syntax) - Complete syntax reference
- [Evaluation Framework](/docs/guides/evaluation) - AI code generation benchmarks

## Resources

- **Examples**: `examples/inline_tests_*.ail`, `examples/snippets/v3_3/math/gcd.ail`
- **Source**: `internal/testing` (generators, shrinkers, runner)
- **Issues**: https://github.com/sunholo-data/ailang/issues
