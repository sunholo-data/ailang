# Case Study: How M-EVAL Helped Fix the Float Equality Bug

**Date**: 2025-10-10
**Version**: v0.3.2-17-g157f05d
**Impact**: 10% benchmark improvement (40% → 50% success rate)

## Summary

This case study demonstrates how the M-EVAL benchmark suite identified a critical bug in AILANG's type system that was causing runtime crashes, and how fixing it measurably improved the language's reliability.

## The Bug

**Symptom**: The `adt_option` benchmark was failing with a runtime error.

**Root Cause**: When float equality operations involved variables (e.g., `let b: float = 0.0; b == 0.0`), the OpLowering pass incorrectly called `eq_Int` instead of `eq_Float`, causing a type mismatch crash at runtime.

**Why It Happened**: The OpLowering pass used literal inspection heuristics instead of the type checker's resolved constraints:

```go
// BAD: Heuristic-based (only checks first argument)
typeSuffix := "Int"  // default
if lit, ok := args[0].(*core.Lit); ok && lit.Kind == core.FloatLit {
    typeSuffix = "Float"
}
```

This worked for literals (`0.0 == 0.0`) but failed for variables (`b == 0.0` where `b: float`).

## How M-EVAL Detected It

### 1. Benchmark Definition

The `adt_option.yml` benchmark tested a realistic use case:

```yaml
task_prompt: |
  Write a program that:
  1. Defines an Option type (Some/None)
  2. Implements a safe division function that returns Option[Float]
     - Returns Some(result) if divisor is non-zero
     - Returns None if divisor is zero
  3. Tests with: divide(10, 2) and divide(10, 0)
```

### 2. Generated Code

The AI correctly generated AILANG code that should work:

```ailang
export func divide(a: float, b: float) -> Option[float] {
  if b == 0.0  // ← This line crashed!
  then None
  else Some(a / b)
}
```

### 3. Error Captured

The eval harness captured the exact error:

```json
{
  "error_category": "runtime_error",
  "stderr": "builtin eq_Int expects int arguments, but received float"
}
```

### 4. Baseline Comparison

Running `ailang eval-compare` showed:

```
adt_option (ailang, claude-sonnet-4-5): runtime_error
Success rate: 4/10 (40%)
```

This made the bug **visible and measurable**.

## The Fix

### Phase 1: Investigation

Using the M-EVAL data, we traced the issue through the compilation pipeline:

1. **Type Checker**: ✅ Correctly assigns `TFloat` to variables
2. **Constraint Resolution**: ✅ Correctly creates `Eq[Float]` constraint
3. **OpLowering**: ❌ Ignores constraints, uses heuristics

### Phase 2: Implementation

Updated OpLowering to use resolved constraints:

```go
// GOOD: Constraint-based (uses type checker results)
if constraint, ok := l.resolvedConstraints[intrinsic.ID()]; ok {
    typeSuffix = getTypeSuffixFromType(constraint.Type)
    // constraint.Type = TFloat → typeSuffix = "Float"
}
```

**Files Changed**:
- `internal/pipeline/op_lowering.go` - Use resolved constraints
- `internal/pipeline/pipeline.go` - Wire constraints into OpLowering
- `internal/pipeline/op_lowering_test.go` - Add regression tests

### Phase 3: Validation

Re-ran the benchmarks:

```bash
ailang eval-validate adt_option
```

Result:

```
✓ FIX VALIDATED: Benchmark now passing!

Before: ✗ runtime_error
After:  ✓ PASSING

Output:
  Result: 5.0
  Error: Division by zero
```

## Secondary Fixes

While fixing the main bug, M-EVAL revealed two more issues:

### 1. Float Display Formatting

**Issue**: `show(5.0)` displayed as `"5"` instead of `"5.0"`, causing benchmark output mismatches.

**Fix**: Updated `showValue()` to ensure floats always show decimal points.

**Files Changed**:
- `internal/eval/value.go`
- `internal/eval/eval_simple.go`

### 2. Eval Harness Missing Output Data

**Issue**: JSON results had `stdout: null` making debugging difficult.

**Fix**: Added `stdout`, `stderr`, and `expected_stdout` to `RunMetrics`.

**Files Changed**:
- `internal/eval_harness/metrics.go`
- `internal/eval_harness/repair.go`

### 3. Prompt Version System

**Issue**: Benchmarks were using outdated v0.3.0 prompt, generating syntax errors.

**Fix**:
- Updated `getDefaultPrompt()` to use active prompt from registry
- Fixed `NewPromptLoader` path handling
- Changed active prompt to v0.3.2
- Implemented `"latest"` special value for automatic version selection

**Files Changed**:
- `internal/eval_harness/spec.go`
- `internal/eval_harness/prompt_loader.go`
- `cmd/ailang/eval_suite.go`
- `prompts/versions.json`

## Measured Impact

### Before Fix (v0.3.0-40-ga7be6e9)

```
Total benchmarks: 10
Passing:          4  (40%)
Failing:          6  (60%)

adt_option: ✗ runtime_error
```

### After Fix (v0.3.2-17-g157f05d)

```
Total benchmarks: 10
Passing:          5  (50%)
Failing:          5  (50%)

adt_option: ✓ PASSING

Improvement: +10.0%
```

### Breakdown by Benchmark

| Benchmark            | Before          | After           | Change    |
|---------------------|-----------------|-----------------|-----------|
| adt_option          | ✗ runtime_error | ✓ PASSING       | **FIXED** |
| records_person      | ✓ PASSING       | ✓ PASSING       | -         |
| fizzbuzz            | ✓ PASSING       | ✓ PASSING       | -         |
| recursion_factorial | ✓ PASSING       | ✓ PASSING       | -         |
| recursion_fibonacci | ✓ PASSING       | ✓ PASSING       | -         |
| numeric_modulo      | ✗ compile_error | ✗ logic_error   | Different |
| json_parse          | ✗ compile_error | ✗ compile_error | -         |
| cli_args            | ✗ compile_error | ✗ compile_error | -         |
| pipeline            | ✗ compile_error | ✗ compile_error | -         |
| float_eq            | ✗ compile_error | ✗ compile_error | -         |

**Key Win**: `adt_option` changed from **crash** → **passing**, demonstrating that the fix resolved a critical runtime bug.

## Lessons Learned

### 1. Benchmarks Catch Real Bugs

The `adt_option` benchmark tested a realistic pattern (safe division with Option types) that exposed a critical bug in the type system. Without M-EVAL, this bug might have been discovered much later by users.

### 2. Baselines Enable Progress Tracking

Having baseline results for v0.3.0 allowed us to:
- Measure the exact impact of the fix (+10%)
- Detect regressions immediately
- Build confidence in the language's reliability

### 3. Multiple Issues Often Cluster

Fixing one bug (OpLowering) revealed related issues (float display, eval harness, prompt versioning). The eval system made all of these visible and measurable.

### 4. Automated Validation Is Critical

Running `ailang eval-validate adt_option` after each change provided immediate feedback:
- ✗ "Still failing" → keep debugging
- ✓ "Fix validated" → move on

This tight feedback loop accelerated development.

## Workflow Enabled by M-EVAL

The bug fix followed this workflow:

```
1. Run Baseline
   ↓
   make eval-baseline
   → Discover: adt_option failing (runtime_error)

2. Investigate
   ↓
   Check JSON: eval_results/baselines/.../adt_option_*.json
   → Root cause: eq_Int called on floats

3. Implement Fix
   ↓
   Update OpLowering to use resolved constraints
   → Add regression tests

4. Validate
   ↓
   ailang eval-validate adt_option
   → ✓ FIX VALIDATED: Benchmark now passing!

5. Measure Impact
   ↓
   ailang eval-compare baselines/old baselines/new
   → +10% improvement (40% → 50%)

6. Store Baseline
   ↓
   Commit: eval_results/baselines/v0.3.2-17-g157f05d/
   → Future releases can compare against this
```

## Tools Used

- `make eval-baseline` - Capture baseline performance
- `ailang eval-validate` - Test specific fix
- `ailang eval-compare` - Compare baselines
- `ailang eval-matrix` - Generate performance reports
- `ailang eval-summary` - Export to JSONL

## Conclusion

The M-EVAL benchmark suite:

1. **Detected** a critical runtime bug that caused crashes
2. **Diagnosed** the root cause through structured error reporting
3. **Validated** the fix with automated testing
4. **Measured** the improvement quantitatively (+10%)
5. **Documented** the fix for future reference

This demonstrates that **investment in evaluation infrastructure pays dividends** by making bugs visible, fixes measurable, and progress trackable.

### Quote

> "You can't improve what you can't measure."
> — M-EVAL made language reliability measurable, enabling systematic improvement.

---

**Next Steps**:
- Add more benchmarks for coverage of language features
- Implement automated regression detection in CI
- Track benchmark success rates over time
- Use eval results to prioritize bug fixes

**Related Documentation**:
- [M-EVAL-LOOP Design Doc](../../../design_docs/implemented/M-EVAL-LOOP_self_improving_feedback.md)
- [Eval Loop Guide](./eval-loop.md)
- [Go Implementation Guide](./go-implementation.md)
