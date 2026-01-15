# M-GAP2: Fix Path-Dependent Lambda Arity Bug

## Status
- **Status:** Planned
- **Target:** v0.6.4
- **Priority:** P0 (Critical)
- **Estimated:** 4-8 hours
- **Dependencies:** None

## Problem Statement

**CRITICAL BUG:** Lambda syntax with `foldl` works in `examples/runnable/` but fails with "arity mismatch: 2 vs 1" error when the identical code is placed in a different directory.

### Reproduction

**Works:**
```bash
ailang check examples/runnable/no_loops_fold.ail  # ✓ passes
```

**Fails (identical code):**
```bash
ailang check internal/dashboard_transforms/test_fold.ail  # ✗ fails
# Error: arity mismatch: 2 vs 1
```

### Symptoms
- Same lambda syntax `\acc x. ...` works in some locations but not others
- Error is "arity mismatch: 2 vs 1" - suggests type checker sees lambda as 1-arity instead of 2-arity
- Non-deterministic behavior based on file path

### Impact
- Users cannot reliably use lambda syntax with higher-order functions
- Forces use of verbose `func` syntax as workaround (GAP-3)
- Breaks AILANG's determinism guarantee (Axiom A1)

## Goals

**Primary Goal:** Lambda syntax behaves identically regardless of file location

**Success Metrics:**
- `\acc x. expr` parses as 2-arity lambda everywhere
- All `foldl` examples work in any directory
- No path-dependent type checking behavior

## Investigation Plan

### Hypothesis 1: Cached Type Information
The type checker may be caching results that differ by path.

**Investigation:**
```bash
# Clear any caches
rm -rf ~/.ailang/cache/
rm -rf /tmp/ailang*

# Test fresh
ailang check internal/dashboard_transforms/test_fold.ail
```

### Hypothesis 2: Module Path Resolution
The module system may influence type checking based on canonical path.

**Investigation:**
```bash
# Check if module declaration affects behavior
# Create test file with different module declarations
echo 'module test' > /tmp/test1.ail
echo 'module internal/test' > /tmp/test2.ail
# Add identical foldl code to both
```

### Hypothesis 3: Parser Token Position Bug
The lexer/parser may have path-dependent behavior affecting lambda parsing.

**Investigation:**
```bash
DEBUG_PARSER=1 ailang check examples/runnable/no_loops_fold.ail 2>&1 | tee working.log
DEBUG_PARSER=1 ailang check internal/dashboard_transforms/test_fold.ail 2>&1 | tee failing.log
diff working.log failing.log
```

### Hypothesis 4: stdlib Import Differences
Different paths may resolve stdlib imports differently, affecting `foldl` type.

**Investigation:**
```bash
# Check if foldl has same type in both contexts
ailang repl
:type foldl
# Compare with explicit import in test file
```

## Solution Design

### Phase 1: Diagnosis (~2-4 hours)

1. **Create minimal reproduction:**
   ```ailang
   -- test_lambda_arity.ail
   module test
   import std/list (foldl)

   let sum = foldl(\acc x. acc + x, 0, [1,2,3])
   ```

2. **Test in multiple locations:**
   - `examples/runnable/`
   - `internal/`
   - `/tmp/`
   - Project root

3. **Enable debug output:**
   ```bash
   DEBUG_STRICT=1 DEBUG_MONO_VERBOSE=1 ailang check test.ail
   ```

4. **Compare AST output:**
   ```bash
   ailang debug ast working.ail --show-types > working_ast.txt
   ailang debug ast failing.ail --show-types > failing_ast.txt
   diff working_ast.txt failing_ast.txt
   ```

### Phase 2: Fix (~2-4 hours)

Based on diagnosis, fix will be one of:

**If cache issue:**
- Clear path-dependent caching
- Ensure cache keys include full context

**If module resolution:**
- Normalize paths before module resolution
- Ensure canonical path handling is consistent

**If parser bug:**
- Fix token position tracking
- Ensure lambda parameter parsing is context-free

**If stdlib import:**
- Ensure `foldl` type is resolved identically
- Fix import path normalization

### Files Likely to Modify

| File | Possible Change |
|------|-----------------|
| `internal/types/unify.go` | Lambda arity unification |
| `internal/elaborate/elaborate.go` | Lambda elaboration |
| `internal/loader/loader.go` | Module path resolution |
| `internal/pipeline/pipeline.go` | Caching behavior |

## Testing

### Regression Test Suite
```bash
# Create test cases in multiple directories
mkdir -p tests/lambda_arity/{root,nested/deep,absolute}

# Each contains identical lambda test
for dir in tests/lambda_arity/*/; do
  cp test_lambda_arity.ail "$dir"
  ailang check "$dir/test_lambda_arity.ail" || echo "FAIL: $dir"
done
```

### Edge Cases
- [ ] Lambda in REPL vs file
- [ ] Lambda in imported module vs main module
- [ ] Lambda with 1, 2, 3+ parameters
- [ ] Nested lambdas `\x. \y. \z. ...`

## Success Criteria

- [ ] `\acc x. expr` works identically in all file locations
- [ ] No "arity mismatch" errors for correct code
- [ ] Deterministic behavior (Axiom A1 compliance)
- [ ] All existing tests pass
- [ ] New regression tests added

## Timeline

**Day 1:** Diagnosis and minimal reproduction (2-4 hours)
**Day 2:** Implement fix and test (2-4 hours)

## Axiom Alignment

| Axiom | Score | Rationale |
|-------|-------|-----------|
| A1: Determinism | +1 | Fixes non-deterministic path-dependent behavior |
| A5: Bounded Verification | +1 | Enables local reasoning about lambda types |
| A7: Machines First | +1 | AI-generated code will work reliably |

**Net Score:** +3 (Accept - Critical fix)

## Related Documents

- [GAPS_DISCOVERED.md](../../../internal/dashboard_transforms/GAPS_DISCOVERED.md) - Discovery context
- GAP-3 is the workaround for this bug (use inline `func` syntax)

## Notes

This is the root cause of GAP-3. Fixing this bug will eliminate the need for the verbose `func` workaround.
