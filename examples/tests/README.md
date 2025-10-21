# Test Examples

Test cases for AILANG features. These files test specific language behavior, edge cases, and regression scenarios.

## Purpose

- **Integration testing**: Verify features work end-to-end
- **Regression prevention**: Catch bugs that were previously fixed
- **Edge case coverage**: Test boundary conditions
- **Documentation**: Show how specific features should behave

## Categories

### Effect System Tests
- `test_effect_io.ail` - I/O effect with capability grant
- `test_effect_io_simple.ail` - Simple I/O test
- `test_effect_fs.ail` - File system effects
- `test_effect_annotation.ail` - Effect annotations
- `test_effect_capability.ail` - Capability checking

### Network Effects
- `test_net_localhost.ail` - HTTP calls to localhost
- `test_net_file_protocol.ail` - File protocol URLs
- `test_net_security.ail` - Network security tests
- `micro_net_fetch.ail` - Simple HTTP fetch

### Pattern Matching & Guards
- `test_guard_bool.ail` - Boolean guards
- `test_guard_false.ail` - False guard handling
- `test_guard_debug.ail` - Guard debugging
- `test_single_guard.ail` - Single guard pattern
- `test_exhaustive_bool_complete.ail` - Complete bool matching
- `test_exhaustive_bool_incomplete.ail` - Incomplete matching (should error)
- `test_exhaustive_wildcard.ail` - Wildcard patterns

### Type System
- `test_integral.ail` - Integer type class
- `test_float_eq_works.ail` - Float equality
- `test_float_comparison.ail` - Float comparison ops
- `test_record_subsumption.ail` - Record type subsumption

### Module System
- `test_module_minimal.ail` - Minimal module
- `test_import_func.ail` - Function imports
- `test_import_ctor.ail` - Constructor imports
- `test_use_constructor.ail` - Using imported constructors
- `test_with_import.ail` - Module with imports
- `test_no_import.ail` - Module without imports

### Builtins
- `test_io_builtins.ail` - I/O builtin functions
- `test_modulo_works.ail` - Modulo operator
- `test_float_modulo.ail` - Float modulo (should error)
- `test_fizzbuzz.ail` - FizzBuzz implementation
- `test_m_r7_comprehensive.ail` - Comprehensive feature test
- `micro_clock_measure.ail` - Clock capability

### Bug Regression Tests
- `bug_float_comparison.ail` - Float comparison bug (fixed)
- `bug_modulo_operator.ail` - Modulo operator bug (fixed)
- `recursion_error.ail` - Recursion error handling

### Effect Purity
- `demos/effects_pure.ail` - Pure vs effectful separation

### Invocation
- `test_invocation.ail` - Function invocation tests

## Running Tests

```bash
# Run a single test
ailang run --caps IO,FS,Net,Clock examples/tests/test_effect_io.ail --entry main

# Run all tests (some may intentionally fail to test error handling)
for f in examples/tests/*.ail; do
  echo "Testing $f..."
  ailang run --caps IO,FS,Net,Clock "$f" --entry main || echo "  (expected failure or missing capability)"
done
```

## Expected Behavior

Some tests are **expected to fail** (testing error conditions):
- `test_exhaustive_bool_incomplete.ail` - Should error with "non-exhaustive pattern"
- `test_float_modulo.ail` - Should error with "modulo not defined for Float"
- `recursion_error.ail` - Should error with stack overflow or recursion limit

Tests that **should pass**:
- All `test_effect_*.ail` files (with correct capabilities granted)
- All `test_guard_*.ail` files
- All `test_import_*.ail` files
- All `micro_*.ail` files

## CI Verification

Tests are **not** automatically verified by `make verify-examples` because:
1. Some intentionally fail (testing error handling)
2. Some require external resources (network, files)
3. Some test specific capability grant scenarios

For manual testing, use the test runner above.

## Contributing

When adding new tests:
1. Use descriptive filenames: `test_<feature>_<scenario>.ail`
2. Add comments explaining what's being tested
3. Document expected behavior (pass/fail, error message)
4. Include test in appropriate category above
