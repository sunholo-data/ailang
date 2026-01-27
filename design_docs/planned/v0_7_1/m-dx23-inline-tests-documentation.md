# M-DX23: Comprehensive Inline Tests Documentation

**Status**: Planned
**Target**: v0.7.1
**Priority**: P1 - Critical for user-facing DX
**Estimated**: 2-3 days
**Dependencies**: M-TESTING-INLINE (v0.4.7, complete)

## Problem Statement

**The inline testing feature (v0.4.7, complete) lacks comprehensive user-facing documentation on ailang.sunholo.com.**

While the feature is fully implemented and working, users encounter friction when learning to use inline tests:

1. **Limited visibility on website** - Main guides don't prominently feature inline tests
2. **Syntax patterns scattered** - Users must search examples to find correct syntax
3. **Edge case gaps** - Nullary functions, multi-arg tuples, type-specific patterns not documented
4. **File-level tests confusing** - Users expect `test "name" { }` blocks to work but they're skipped
5. **No integration guide** - How inline tests fit into workflow/tooling not clear
6. **Example coverage sparse** - Few runnable examples demonstrating real-world patterns

**Impact**:
- Users attempting inline tests struggle with syntax (especially tuples for multi-arg functions)
- ecommerce demo revealed gaps after implementation ("had to search through repo to find syntax")
- Lost productivity while users decipher correct patterns
- Potential for feature underuse due to friction

**Current State**:
- ✅ `docs/guides/inline_tests.md` exists (comprehensive, 740 lines)
- ✅ `docs/docs/guides/testing.md` covers property tests extensively (600 lines)
- ⚠️ Website integration incomplete (guide not linked from main nav)
- ⚠️ No example files specifically for inline tests
- ⚠️ Limitations section missing from main testing guide
- ⚠️ Quick reference card missing

## Goals

**Primary Goal**: Make inline tests the first thing users reach when learning testing in AILANG.

**Success Metrics**:
1. Inline tests guide linked from main testing page and examples page
2. 8+ example files demonstrating inline test patterns (nullary, multi-arg, types, edge cases)
3. Quick reference card on website (1-page syntax guide)
4. Zero friction for new users learning inline test syntax
5. All examples runnable with `ailang check` and passing

## Solution Design

### Overview

Implement comprehensive, discovery-friendly documentation for inline tests:

1. **Website Integration** - Link from main pages, make discoverable
2. **Example Files** - Create runnable examples for each pattern
3. **Quick Reference** - 1-page syntax guide for quick lookup
4. **Update Main Guides** - Add inline tests to testing.md prominently

### Architecture

#### Component 1: Website Integration

**Update**: `docs/docs/guides/testing.md`
- Add "Inline Tests" as first section after Quick Start
- Link to detailed guide: `[Inline Tests Guide](/docs/guides/inline-tests)`
- Include 2-3 quick examples
- Note file-level tests are skipped (Pitfall 5 in detailed guide)

**Update**: `docs/docs/examples.mdx`
- Add "Inline Tests" section highlighting syntax
- Link to inline_tests.md detailed guide
- Show simple example with output

**Create**: Quick Reference Card
- 1-page syntax guide
- Patterns table: nullary, single-arg, multi-arg, 3+ args
- Common pitfalls table
- Downloadable as PDF

#### Component 2: Example Files

Create 8 example files demonstrating all patterns:

**File**: `examples/inline_tests_arithmetic.ail`
- Add/subtract/multiply/divide with multi-arg tests
- Edge cases: zero, negatives, division by small numbers
- ~50 LOC

**File**: `examples/inline_tests_recursive.ail`
- Factorial (0!, 1!, 5!, 6!)
- Fibonacci (0-10)
- Shows base cases and recursion
- ~40 LOC

**File**: `examples/inline_tests_types.ail`
- Integers, floats, strings, booleans
- Float precision warnings
- Type mismatch examples (marked as should-fail)
- ~60 LOC

**File**: `examples/inline_tests_adts.ail`
- Algebraic data types (Color, Option, Result)
- Pattern matching in function body
- ADT in test cases
- ~50 LOC

**File**: `examples/inline_tests_nullary.ail`
- Zero-argument functions
- Empty tuple format `((), expected)`
- Constants and generators
- ~30 LOC

**File**: `examples/inline_tests_complex.ail`
- Complex types: records, lists
- Records with named fields
- List operations with recursive functions
- ~70 LOC

**File**: `examples/inline_tests_edge_cases.ail`
- Boundary values
- Type edge cases (max int, min int, etc.)
- Float precision pitfalls with explanations
- ~60 LOC

**File**: `examples/inline_tests_best_practices.ail`
- Well-organized tests with comments
- Descriptive test names via comments
- Grouping by category
- Coverage guidelines (3+ tests per function)
- ~80 LOC

**Total**: ~440 LOC across 8 files

#### Component 3: Quick Reference Card

**Create**: `docs/static/inline_tests_quick_ref.md`

Content:
```
# Inline Tests Quick Reference

## Syntax
func name(args) -> Type
  tests [test_cases]
{
  body
}

## Test Case Formats

| Function Type | Format | Example |
|---|---|---|
| `f()` | `((), expected)` | `((), 42)` |
| `f(x)` | `(input, expected)` | `(5, 10)` |
| `f(x, y)` | `((x, y), expected)` | `((3, 5), 8)` |
| `f(x, y, z)` | `((x, y, z), expected)` | `((2, 3, 4), 14)` |

## Common Pitfalls

❌ Expression style: `= tests [...]`
✅ Block style: `{ tests [...] }`

❌ Single arg wrapped: `((5), 10)`
✅ Single arg bare: `(5, 10)`

❌ Nullary missing tuple: `(42)`
✅ Nullary with empty tuple: `((), 42)`

## Running Tests
```bash
ailang check file.ail                    # Tests run automatically
make verify-examples                     # Run all examples
```
```

#### Component 4: Update Main Testing Guide

**Update**: `docs/docs/guides/testing.md`

Add new section early in document:

```markdown
## Inline Tests

Inline tests are the lightweight, recommended approach for simple unit tests.

[See Inline Tests Guide](/docs/guides/inline-tests) for complete documentation.

### Quick Example

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

**Benefits:**
- Tests next to implementation
- Run automatically during compilation
- Perfect for training AI with examples

**Limitations:**
- Block-style functions only (`{ }`, not `=`)
- Pure functions only (no IO, FS effects)
- File-level `test "name" { }` blocks don't run

[Learn more →](/docs/guides/inline-tests)
```

### Implementation Plan

**Phase 1: Create Example Files** (~4-5 hours)
- [ ] Create all 8 example files in `examples/inline_tests_*.ail`
- [ ] Verify each file runs with `ailang check`
- [ ] Add comments explaining each pattern
- [ ] Verify no syntax errors or type mismatches
- [ ] Total: ~440 LOC

**Phase 2: Website Integration** (~3-4 hours)
- [ ] Create quick reference card: `docs/static/inline_tests_quick_ref.md`
- [ ] Update `docs/docs/guides/testing.md` with inline tests section
- [ ] Update `docs/docs/examples.mdx` with inline tests feature highlight
- [ ] Test links work on website
- [ ] Verify responsive design on mobile

**Phase 3: Docusaurus Build & Validation** (~2-3 hours)
- [ ] Build website locally: `cd docs && npm run build`
- [ ] Verify inline tests guide appears in sidebar
- [ ] Check search indexes inline test content
- [ ] Test code example imports work
- [ ] Mobile responsiveness check

**Phase 4: Testing & Cleanup** (~1-2 hours)
- [ ] All 8 example files pass `make verify-examples`
- [ ] No broken links in documentation
- [ ] Quick reference card renders correctly
- [ ] Examples display properly with syntax highlighting
- [ ] Version notes added to CHANGELOG.md

### Files to Modify/Create

**New files:**
- `examples/inline_tests_arithmetic.ail` (~50 LOC)
- `examples/inline_tests_recursive.ail` (~40 LOC)
- `examples/inline_tests_types.ail` (~60 LOC)
- `examples/inline_tests_adts.ail` (~50 LOC)
- `examples/inline_tests_nullary.ail` (~30 LOC)
- `examples/inline_tests_complex.ail` (~70 LOC)
- `examples/inline_tests_edge_cases.ail` (~60 LOC)
- `examples/inline_tests_best_practices.ail` (~80 LOC)
- `docs/static/inline_tests_quick_ref.md` (~80 LOC)

**Modified files:**
- `docs/docs/guides/testing.md` - Add inline tests section (~40 lines)
- `docs/docs/examples.mdx` - Add inline tests feature (~30 lines)
- `CHANGELOG.md` - Document documentation improvements (~5 lines)

**Total new content**: ~650 LOC

### Success Criteria

- [ ] All 8 example files created and passing `ailang check`
- [ ] Inline tests guide prominently linked from testing.md
- [ ] Examples linked from main examples page
- [ ] Quick reference card visible on website
- [ ] No broken internal links
- [ ] All example syntax highlighted correctly
- [ ] Mobile-responsive layout verified
- [ ] CHANGELOG.md updated
- [ ] Zero user friction for learning inline test syntax

## Examples

### Before (Current State)

User wants to learn inline test syntax:
1. Visit https://ailang.sunholo.com
2. Find "Testing Guide" (deep in nav)
3. Read property test section (extensive but for properties, not inline)
4. No inline test section visible
5. Frustrated, searches GitHub repo examples
6. Finds scattered examples, infers patterns
7. Struggles with tuple syntax for multi-arg functions

**Result**: 15-20 min to learn, multiple wrong attempts

### After (With Documentation)

User wants to learn inline test syntax:
1. Visit https://ailang.sunholo.com
2. Find "Testing Guide" (inline tests first)
3. Click "Inline Tests" section
4. See quick example with output
5. Find "Quick Reference" for syntax patterns
6. Browse 8 example files matching their use case
7. Copy pattern, test works immediately

**Result**: 3-5 min to learn, zero wrong attempts

## Timeline

**Day 1** (6-8 hours):
- Phase 1: Create all 8 example files
- Verify each passes `ailang check`

**Day 2** (5-6 hours):
- Phase 2: Website integration (guides, quick ref)
- Phase 3: Docusaurus build & validation

**Day 3** (2-3 hours):
- Phase 4: Testing, cleanup, CHANGELOG update

**Total: ~13-17 hours across 2-3 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| **Examples don't compile** | High | Verify each file with `ailang check` before committing. Run `make verify-examples` pre-commit. |
| **Website doesn't rebuild** | Medium | Build locally before pushing. Test `cd docs && npm run build`. |
| **Links break** | Medium | Manual testing of all internal links. Use link checker tool in CI. |
| **Code examples drift** | Medium | Use `!!raw-loader` to import examples from files (automatic sync). |
| **Float precision test failure** | Low | Document float precision pitfall with binary representation explanation. |
| **Incomplete example coverage** | Low | Pre-plan 8 patterns, verify against current LIMITATIONS.md. |

## Related Documents

- **M-TESTING-INLINE** (`design_docs/implemented/v0_4_7/m-testing-inline-core-evaluation.md`) - Implementation details
- **Testing Guide** (`docs/docs/guides/testing.md`) - Main testing documentation
- **LIMITATIONS.md** - Known limitations of inline tests
- **examples/** - Existing example files directory

## Future Work

**Post-v0.7.1:**
- **M-DX24: IDE Integration** - Inline test hints in syntax highlighting
- **M-DX25: Test Coverage Reporting** - Show which code paths tested
- **M-DX26: Incremental Test Caching** - Cache results for faster re-runs
- **M-DX27: Test Generation** - AI-assisted inline test case generation

---

**Document created**: 2026-01-27
**Last updated**: 2026-01-27
**Priority**: P1 - Blocks quality DX for new users learning to write tests

