# M-DX-TEST-HARNESS-NIL: Inline Tests Fail with Stdlib Imports

**Status**: Planned
**Priority**: Medium
**Source**: DX feedback from docparse-demo (Feb 2026)
**Milestone**: v0.8.0

## Problem

Inline tests on pure functions that call stdlib imports (`split`, `last`, `find`, etc.) fail with `"cannot apply non-function value: nil"` even though:
- Type-checking passes
- Runtime evaluation works (`ailang run`)
- Only the test harness fails

### Minimal Reproduction

```ailang
import std/list (split)

pure func chunkList(n: int, xs: [int]) -> [[int]]
  tests [
    (2, [1,2,3,4], [[1,2],[3,4]])
  ]
{ split(n, xs) }
```

Running `ailang test file.ail` produces:
```
cannot apply non-function value: nil
```

But `ailang run --entry main file.ail` with equivalent code works fine.

## Root Cause Analysis

The bug is in `internal/testing/executor.go`, specifically in the two-pass module binding injection system (M-DX25).

### The Two-Pass System (lines 748-809)

```
Pass 1: Collect lambda bindings and inject non-lambda values (including VarGlobal)
Pass 2: Create FunctionValues with fully-populated environment
```

### Three Interacting Issues

**Issue 1: VarGlobal Silent Failures** (lines 775-781)

When handling re-exported functions (`VarGlobal`), the code calls `evaluator.Eval()` in Pass 1. If evaluation fails (because dependencies aren't bound yet), the binding is **silently skipped**:

```go
} else if _, ok := d.Value.(*core.VarGlobal); ok {
    val, err := evaluator.Eval(d.Value)
    if err == nil && val != nil {  // Silent skip on failure!
        env.Set(d.Name, val)
    }
}
```

This violates the "no silent fallbacks" principle.

**Issue 2: VarGlobal Dependency Ordering**

Some `VarGlobal` values reference other functions that haven't been bound yet in Pass 1. For example, `std/list.split` might reference `_core_split_builtin` which is in a `LetRec` that hasn't been processed.

**Issue 3: Cluster Path Missing Module Injection**

`EvaluateInlineTestsWithCluster()` (line 570) uses `runtime.NewBuiltinOnlyResolver()` with **no module binding injection at all**, so any test using stdlib functions through this path will always fail.

### Error Origin

The actual error message comes from `internal/eval/eval_operations.go:128`:
```go
// evalCoreApp() - when the function value is nil
return nil, fmt.Errorf("cannot apply non-function value: nil")
```

## Proposed Fix

### Option A: Three-Pass System (Recommended)

Add a third pass that retries failed VarGlobal evaluations after all base bindings exist:

```
Pass 1: Inject non-lambda, non-VarGlobal values (constants, constructors)
Pass 2: Create FunctionValues for lambdas (with full environment)
Pass 3: Evaluate VarGlobal references (all dependencies now available)
```

### Option B: Lazy VarGlobal Evaluation

Wrap VarGlobal bindings in a lazy thunk that evaluates on first call:

```go
env.Set(d.Name, &LazyValue{
    eval: func() (eval.Value, error) {
        return evaluator.Eval(d.Value)
    },
})
```

### Option C: Make Failures Loud + Retry

At minimum, log VarGlobal evaluation failures instead of silently skipping:

```go
val, err := evaluator.Eval(d.Value)
if err != nil {
    failedVarGlobals = append(failedVarGlobals, d) // retry later
} else if val != nil {
    env.Set(d.Name, val)
}
```

### Additional Fix: Cluster Path

Apply module binding injection to `EvaluateInlineTestsWithCluster()` as well, matching the existing logic in `EvaluateInlineTestsWithHarness()`.

## Implementation Plan

1. Write regression test: inline test calling `std/list.split`
2. Make VarGlobal failures loud (log warning, collect for retry)
3. Add Pass 3: retry failed VarGlobals after full environment is built
4. Apply module injection to `EvaluateInlineTestsWithCluster()` path
5. Run full test suite + inline test examples

## Files to Modify

| File | Change |
|------|--------|
| `internal/testing/executor.go` | Three-pass injection, cluster path fix |
| `internal/testing/executor_test.go` | Regression tests with stdlib imports |
| `examples/tests/test_stdlib_inline.ail` | Example file demonstrating fix |

## Risk Assessment

- **Medium risk**: Changes to binding injection order could affect existing tests
- **Testing**: Existing inline test suite + new stdlib-specific tests
- **Backward compat**: No API changes, purely internal fix
