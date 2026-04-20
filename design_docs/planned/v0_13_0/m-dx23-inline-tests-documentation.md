# Inline Tests Documentation & Examples (M-DX23)

**Status:** Planned
**Target Version:** v0.8.0
**Priority:** P1 (High - impacts developer experience)
**Estimated:** 2 days
**Dependencies:** None (feature already implemented, documentation gap only)

---

## Problem Statement

The inline tests feature (`tests [...]` clause in function signatures) is not well documented on https://ailang.sunholo.com, despite being fully implemented and working. Developers discover this feature only through:
- Manual search through AILANG repo examples
- Trial and error experimentation
- Peer knowledge sharing

**Current State:**
- Feature implemented in v0.3+ (fully functional)
- Syntax works for block-style functions `{ }` but NOT expression-style `=`
- Multiple patterns supported: nullary functions `()`, single args, multi-arg tuples
- File-level `test "name" { }` syntax exists but is skipped
- **Documentation gap:** Zero public documentation on syntax, usage, or patterns

**Impact:**
- Developers write fewer tests (don't know feature exists)
- Code quality inconsistency (some functions well-tested, others untested)
- Onboarding friction (new developers must reverse-engineer syntax)
- Lost opportunity for better AI training data (inline tests provide execution traces)

**Key Findings from User Discovery:**
1. Inline tests ONLY work with block-style functions `{ }`, NOT expression-style `=`
2. Nullary functions use `()` as input (empty tuple)
3. Multi-arg functions use tuple format: `((arg1, arg2), expected_result)`
4. File-level `test "name" { expr }` syntax doesn't run (shows as skipped)

---

## Goals

**Primary Goal:** Document inline tests feature comprehensively on website with syntax, patterns, and best practices.

**Success Metrics:**
1. ✅ Website docs cover all 4 inline test patterns (nullary, single-arg, multi-arg, tuple format)
2. ✅ 8+ runnable examples demonstrating each pattern
3. ✅ Edge cases documented (type constraints, float precision, ADTs)
4. ✅ Developers can discover and use feature without reverse-engineering
5. ✅ Test coverage maintained at 100% for examples (all tests pass)

---

## Solution Design

### Overview

Create a three-part documentation package:

1. **Design Doc** (`design_docs/planned/v0_8_0/m-dx23-inline-tests-documentation.md`)
   - This document - rationale, patterns, best practices

2. **Guide Document** (`docs/guides/inline_tests.md`)
   - Comprehensive guide with syntax, patterns, edge cases
   - Website-ready markdown

3. **Example Files** (`examples/tests/inline_tests_*.ail`)
   - Focused examples demonstrating each pattern
   - All examples verified to work

### Architecture

**Documentation Structure:**

```
docs/guides/inline_tests.md
├── What are Inline Tests?
├── Basic Syntax
│   ├── Block-style functions (✅ works)
│   └── Expression-style functions (❌ doesn't work)
├── Test Patterns
│   ├── Nullary functions (no args)
│   ├── Single-argument functions
│   ├── Multi-argument functions
│   └── Tuple format for multi-args
├── Edge Cases
│   ├── Float precision
│   ├── ADT pattern matching
│   ├── Type constraints
│   └── Recursive functions
├── Best Practices
│   ├── Test coverage guidelines
│   ├── Naming conventions
│   └── Performance considerations
├── Common Pitfalls
│   ├── Expression-style limitation
│   ├── Type mismatches
│   └── Infinite recursion
└── Advanced Examples
    ├── Testing polymorphic functions
    ├── Testing with effects
    └── Property-based testing (future)
```

**Example File Organization:**

```
examples/tests/
├── inline_tests_nullary.ail          # Functions with no arguments
├── inline_tests_unary.ail            # Single-argument functions
├── inline_tests_multiarg.ail         # Multi-argument functions with tuples
├── inline_tests_edge_cases.ail       # Float precision, ADTs, etc.
├── inline_tests_recursive.ail        # Recursive function testing
└── inline_tests_polymorphic.ail      # Polymorphic function testing
```

### Implementation Plan

**Phase 1: Documentation (6 hours)**
- [ ] Create `docs/guides/inline_tests.md` with comprehensive guide
- [ ] Document all 4 test patterns with clear examples
- [ ] Add edge cases and best practices sections
- [ ] Document known limitations (expression-style functions)
- [ ] Add website integration (update sidebar navigation)

**Phase 2: Examples (4 hours)**
- [ ] Create `examples/tests/inline_tests_nullary.ail`
- [ ] Create `examples/tests/inline_tests_unary.ail`
- [ ] Create `examples/tests/inline_tests_multiarg.ail`
- [ ] Create `examples/tests/inline_tests_edge_cases.ail`
- [ ] Create `examples/tests/inline_tests_recursive.ail`
- [ ] Verify all examples with `make verify-examples`

**Phase 3: Website Integration (2 hours)**
- [ ] Add inline_tests.md to website sidebar
- [ ] Import example files into guide using raw-loader
- [ ] Test website builds with new documentation
- [ ] Validate all example imports work

**Phase 4: Testing & Validation (2 hours)**
- [ ] Run `make verify-examples` on all inline test files
- [ ] Update README.md implementation status
- [ ] Run `make test-coverage` to ensure tests pass
- [ ] Manual verification on https://ailang.sunholo.com

### Files to Create

| File | Type | LOC | Purpose |
|------|------|-----|---------|
| `docs/guides/inline_tests.md` | Markdown | 250-300 | Comprehensive guide |
| `examples/tests/inline_tests_nullary.ail` | AILANG | 40-50 | Nullary function tests |
| `examples/tests/inline_tests_unary.ail` | AILANG | 60-80 | Single-arg function tests |
| `examples/tests/inline_tests_multiarg.ail` | AILANG | 80-100 | Multi-arg tuple tests |
| `examples/tests/inline_tests_edge_cases.ail` | AILANG | 100-120 | Edge cases (floats, ADTs) |
| `examples/tests/inline_tests_recursive.ail` | AILANG | 60-80 | Recursive function tests |

**Files to Modify:**

| File | Changes | Impact |
|------|---------|--------|
| `docs/sidebars.js` | Add inline_tests guide link | Navigation |
| `README.md` | Update implementation status | Public docs |
| `CHANGELOG.md` | Document documentation improvement | Release notes |

---

## Test Examples

### Pattern 1: Nullary Functions (No Arguments)

```ailang
pure func getConstant() -> int
  tests [
    ((), 42)
  ]
{
  42
}
```

**Format:** `((), expected_value)` - empty tuple as input

### Pattern 2: Single-Argument Functions

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

**Format:** `(input, expected_value)` - direct value

### Pattern 3: Multi-Argument Functions (Tuple Format)

```ailang
pure func add(a: int, b: int) -> int
  tests [
    ((3, 5), 8),
    ((0, 0), 0),
    ((-1, 1), 0)
  ]
{
  a + b
}
```

**Format:** `((arg1, arg2, ...), expected_value)` - nested tuples

### Pattern 4: Edge Cases

```ailang
pure func gcd(a: int, b: int) -> int
  tests [
    ((48, 18), 6),
    ((0, 5), 5),
    ((7, 7), 7)
  ]
{
  if b == 0 then a else gcd(b, a % b)
}

pure func safeDivide(a: float, b: float) -> float
  tests [
    ((10.0, 2.0), 5.0),
    ((9.0, 3.0), 3.0),
    ((0.0, 5.0), 0.0)
  ]
{
  a / b
}
```

---

## Edge Cases & Limitations

### Known Limitations

1. **Expression-style functions don't support inline tests**
   ```ailang
   // ❌ DOESN'T WORK
   pure func double(x: int) -> int =
     tests [...]
     x * 2

   // ✅ WORKS
   pure func double(x: int) -> int
     tests [...]
   {
     x * 2
   }
   ```

2. **File-level test syntax is not executed**
   ```ailang
   // ❌ Shows as "skipped"
   test "verify implementation" {
     add(5, 3) == 8
   }

   // ✅ Use inline tests instead
   pure func add(a: int, b: int) -> int
     tests [
       ((5, 3), 8)
     ]
   { ... }
   ```

### Edge Case Handling

**Float Precision:**
```ailang
pure func addFloats(a: float, b: float) -> float
  tests [
    ((0.1, 0.2), 0.3),        // May have precision issues
    ((1.5, 2.5), 4.0),        // Exact representation
    ((0.0, 0.0), 0.0)
  ]
{ a + b }
```

**Recursive Functions:**
```ailang
pure func fibonacci(n: int) -> int
  tests [
    (0, 0),
    (1, 1),
    (5, 5),
    (10, 55)
  ]
{
  if n <= 1 then n else fibonacci(n-1) + fibonacci(n-2)
}
```

**ADT Pattern Matching:**
```ailang
type Color = Red | Green | Blue

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

## Success Criteria

- [ ] Design doc created and reviewed
- [ ] `docs/guides/inline_tests.md` created with 250+ lines
- [ ] 6 focused example files created in `examples/tests/`
- [ ] All examples pass `make verify-examples`
- [ ] Website documentation is accessible and searchable
- [ ] README.md updated with inline tests documentation status
- [ ] No regression in existing tests (100% test suite passes)
- [ ] Documentation includes real, runnable examples

---

## Timeline

**Week 1:**
- Mon: Create guide document, gather examples
- Tue-Wed: Create focused example files, test syntax
- Thu: Website integration, build verification
- Fri: Final review, testing, commit

---

## Related Documents

- [M-DX1: Builtin Developer Experience](design_docs/implemented/v0_3_10/M-DX1-FINAL-SUMMARY.md) - Similar documentation effort
- [Testing Guide](docs/guides/testing.md) - Complementary testing documentation
- [AILANG Syntax Guide](docs/guides/syntax.md) - General language documentation

---

## Notes

- This is primarily a documentation task, not a feature implementation
- The inline tests feature is already fully functional (v0.3+)
- No code changes to the evaluator or compiler needed
- Focus on making the feature discoverable through clear documentation
- Inline tests are valuable for AI training (execution traces, test data)
