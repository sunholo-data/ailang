# Example Promotion Opportunities

**Created**: 2025-10-21
**Status**: Analysis complete, action optional

## Executive Summary

After reorganizing examples in v0.3.14, we discovered that **most test examples actually work** and could be promoted to runnable examples with minimal fixes.

**Current state:**
- ✅ 21 runnable examples (100% pass rate)
- 📝 21 snippets (documentation, intentionally not runnable)
- 🧪 37 tests (31+ actually work!)
- 🔬 9 experimental (future features)

**Opportunity**: Could increase runnable examples from 21 → 52+ by promoting working tests.

---

## Analysis Results

### Tests That Already Work (31 files)

All `test_*.ail` files were tested and **all 31 passed**:

```bash
✅ test_effect_annotation.ail
✅ test_effect_capability.ail
✅ test_effect_fs.ail
✅ test_effect_io.ail
✅ test_effect_io_simple.ail
✅ test_exhaustive_bool_complete.ail
✅ test_exhaustive_bool_incomplete.ail
✅ test_exhaustive_wildcard.ail
✅ test_fizzbuzz.ail
✅ test_float_comparison.ail
✅ test_float_eq_works.ail
✅ test_float_modulo.ail
✅ test_guard_bool.ail
✅ test_guard_debug.ail
✅ test_guard_false.ail
✅ test_import_ctor.ail
✅ test_import_func.ail
✅ test_integral.ail
✅ test_invocation.ail
✅ test_io_builtins.ail
✅ test_m_r7_comprehensive.ail
✅ test_module_minimal.ail
✅ test_modulo_works.ail
✅ test_net_file_protocol.ail
✅ test_net_localhost.ail
✅ test_net_security.ail
✅ test_no_import.ail
✅ test_record_subsumption.ail
✅ test_single_guard.ail
✅ test_use_constructor.ail
✅ test_with_import.ail
```

### Tests Needing Module Path Fixes (5 files)

These work but need module declaration updates:

```bash
⚠️ bug_float_comparison.ail - needs module path fix
⚠️ bug_modulo_operator.ail - needs module path fix
⚠️ micro_clock_measure.ail - needs module path fix
⚠️ micro_net_fetch.ail - needs module path fix
⚠️ demos/effects_pure.ail - needs module path fix
```

### Tests That Should Stay as Tests (1 file)

```bash
🧪 recursion_error.ail - Tests error handling (intentionally fails)
```

---

## Recommendation Options

### Option 1: Promote Working Tests (RECOMMENDED)

**Benefit**: Showcase more AILANG capabilities
**Time**: ~2 hours

```bash
# Move 31 working test_*.ail to runnable/
git mv examples/tests/test_*.ail examples/runnable/

# Fix module paths for 5 remaining tests
# Move to runnable/

# Update README.md
# Result: 21 → 57 runnable examples (171% increase!)
```

**Pros:**
- Demonstrates more language features
- Better user experience (more working examples)
- Shows off effect system, pattern matching, modules

**Cons:**
- Some tests are very technical (not ideal for learning)
- May clutter runnable/ directory
- Tests serve documentation purpose where they are

### Option 2: Keep Current Organization

**Benefit**: Clear separation of concerns
**Time**: 0 hours

Keep tests in `tests/` directory because:
- They document test cases and edge cases
- They show expected behavior for specific features
- Not all are beginner-friendly examples
- Current 21 runnable examples are high-quality

**Pros:**
- No work required
- Clear organization (runnable vs tests)
- Tests serve documentation purpose

**Cons:**
- Users may not realize these work
- Missing opportunity to showcase features

### Option 3: Selective Promotion (BALANCED)

**Benefit**: Promote only user-friendly examples
**Time**: ~1 hour

Move these **10 high-quality tests** to runnable/:
```bash
test_fizzbuzz.ail            # Classic algorithm
test_effect_io_simple.ail    # Simple I/O demo
test_guard_bool.ail          # Guard patterns
test_import_func.ail         # Module imports
test_module_minimal.ail      # Minimal module
test_io_builtins.ail         # I/O builtin showcase
test_m_r7_comprehensive.ail  # Comprehensive feature demo
micro_clock_measure.ail      # Clock capability
micro_net_fetch.ail          # Network fetch
bug_float_comparison.ail     # Float comparison (fixed bug demo)
```

Leave technical tests in `tests/`:
```bash
test_exhaustive_bool_incomplete.ail  # Error condition
test_float_modulo.ail                # Error condition
test_net_security.ail                # Security test
test_record_subsumption.ail          # Type system detail
# ... etc
```

**Result:** 21 → 31 runnable examples (48% increase, high quality)

---

## Impact Analysis

### Current (21 runnable):
```
Categories covered:
- ADTs (2)
- Recursion (4)
- Effects (2)
- Blocks (3)
- Data structures (3)
- Simple programs (7)
```

### With Option 3 (+10 examples):
```
NEW categories covered:
- FizzBuzz algorithm
- Module imports/exports
- Guard patterns
- Network operations
- Clock capability
- Float comparisons
- Comprehensive feature showcase

Total: 31 examples across 12+ categories
```

---

## Decision Matrix

| Criteria | Option 1 (All) | Option 2 (None) | Option 3 (Selective) |
|----------|---------------|-----------------|----------------------|
| Example count | 57 | 21 | 31 |
| User value | High | Medium | High |
| Time cost | 2h | 0h | 1h |
| Organization clarity | Lower | Highest | High |
| Feature showcase | Best | Good | Better |
| Learning curve | Steeper | Gentle | Balanced |

---

## Recommendation

**Use Option 3 (Selective Promotion)** because:

1. **Balanced approach**: Keep tests that test edge cases, promote examples that teach
2. **Manageable increase**: 21 → 31 (not overwhelming)
3. **Quality over quantity**: Each promoted example is beginner-friendly
4. **Time efficient**: ~1 hour work for 48% more examples

---

## Implementation Steps (If Pursuing Option 3)

```bash
# 1. Create promotion script
cat > scripts/promote_tests.sh <<'EOF'
#!/bin/bash
# Promote high-quality tests to runnable examples

PROMOTE=(
  "test_fizzbuzz.ail"
  "test_effect_io_simple.ail"
  "test_guard_bool.ail"
  "test_import_func.ail"
  "test_module_minimal.ail"
  "test_io_builtins.ail"
  "test_m_r7_comprehensive.ail"
  "micro_clock_measure.ail"
  "micro_net_fetch.ail"
  "bug_float_comparison.ail"
)

for f in "${PROMOTE[@]}"; do
  git mv "examples/tests/$f" "examples/runnable/$f"
  # Fix module path
  sed -i.bak "s|examples/|examples/runnable/|" "examples/runnable/$f"
  rm -f "examples/runnable/$f.bak"
done
EOF

# 2. Run promotion
chmod +x scripts/promote_tests.sh
./scripts/promote_tests.sh

# 3. Verify all still pass
make verify-examples
# Expected: 31 passed, 0 failed

# 4. Update README.md files
# Update counts: 21 → 31
# Add new categories to index

# 5. Commit
git add examples/
git commit -m "Promote 10 high-quality test examples to runnable/"
```

---

## Deferred to v0.3.15 or Later

If not done now, this is a **great v0.3.15 task** because:
- Low risk (examples already work)
- High user value (more learning materials)
- Demonstrates v0.3.14 features better
- ~1 hour of work

Could be combined with:
- Documentation improvements
- Tutorial updates
- Website example gallery

---

## Conclusion

We have a **wealth of working examples** hidden in `tests/`. Promoting just 10 high-quality ones would significantly improve user experience with minimal effort.

**Status**: Optional for v0.3.14, recommended for v0.3.15
