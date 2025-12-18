# M-ENV-TYPE Sprint Plan: Fix EnvError Type Parameter Bug

**Sprint ID**: M-ENV-TYPE
**Design Doc**: [m-env-type-parameter-bug.md](m-env-type-parameter-bug.md)
**Duration**: 1-2 hours (single session)
**Risk Level**: Low

## Summary

**Goal**: Fix the type system bug where `EnvError` (non-parameterized ADT) is incorrectly treated as having 1 type parameter when used in `Result[string, EnvError]`.

**Root Cause Identified**: The bug is in `internal/builtins/env.go:30`:
```go
// BUG: EnvError has 0 type params, not 1
envErrorType := T.App("EnvError", T.String())  // WRONG
```

Should be:
```go
envErrorType := T.Con("EnvError")  // CORRECT
```

## Current Status Analysis

### Verified Bug
```bash
$ ./bin/ailang run --caps IO,Env --entry main examples/runnable/cli_args_demo.ail
Error: type error in std/env (decl 0): type unification failed at [return type annotation at
std/env.ail:29:8]: type EnvError expects 1 type argument(s), but got 0 (did you mean EnvError[string]?)
```

### Root Cause
In `internal/builtins/env.go`, the `_env_getEnv` builtin declares its return type as `Result[string, EnvError[string]]` when it should be `Result[string, EnvError]`.

The `EnvError` type in `std/env.ail` is:
```ailang
export type EnvError =
  | NotFound(string)    -- Constructor field types, NOT type parameters
  | NotAllowed(string)
```

This is a **non-parameterized ADT** (arity 0). The builtin incorrectly uses `T.App("EnvError", T.String())` which creates `EnvError[string]` (arity 1).

## Milestones

### M1: Fix Builtin Type Declaration (~10 LOC, 15 min)

**Task**: Change `T.App("EnvError", T.String())` to `T.Con("EnvError")`

**Files**:
- `internal/builtins/env.go` - Line 30

**Change**:
```go
// Before (line 30):
envErrorType := T.App("EnvError", T.String())

// After:
envErrorType := T.Con("EnvError")
```

**Acceptance Criteria**:
- [ ] `./bin/ailang check std/env.ail` passes
- [ ] `./bin/ailang run --caps IO,Env --entry main examples/runnable/cli_args_demo.ail` runs

### M2: Update Golden File (~5 LOC, 10 min)

**Task**: Update the golden file that tests builtin types

**Files**:
- `internal/pipeline/testdata/builtin_types.golden` - Line 29

**Change**:
```
# Before:
_env_getEnv : string -> Result[string, EnvError[string]] ! {Env}

# After:
_env_getEnv : string -> Result[string, EnvError] ! {Env}
```

**Acceptance Criteria**:
- [ ] `go test ./internal/pipeline/... -run TestBuiltinTypes` passes

### M3: Add Unit Test (~30 LOC, 20 min)

**Task**: Add explicit test for non-parameterized ADTs in Result position

**Files**:
- `internal/types/typechecker_test.go` - New test function

**Test Case**:
```go
func TestNonParameterizedADTInResult(t *testing.T) {
    // Test that EnvError (arity 0) works in Result[string, EnvError]
    // Minimal reproduction without full module loading
}
```

**Acceptance Criteria**:
- [ ] New test passes
- [ ] Test catches the original bug if reintroduced

### M4: Verify Full Test Suite (~10 min)

**Task**: Run full test suite to ensure no regressions

**Commands**:
```bash
make test
make verify-examples
./bin/ailang run --caps IO,Env --entry main examples/runnable/cli_args_demo.ail Alice
```

**Acceptance Criteria**:
- [ ] `make test` passes
- [ ] `make verify-examples` passes
- [ ] cli_args_demo.ail outputs "Hello, Alice!"

## Implementation Plan

### Day 1 (1-2 hours total)

| Time | Task | Details |
|------|------|---------|
| 0:00 | M1: Fix builtin | 1-line change in env.go |
| 0:15 | M2: Update golden | 1-line change in golden file |
| 0:25 | Rebuild & smoke test | `make build && ailang check std/env.ail` |
| 0:35 | M3: Add unit test | Write test for non-parameterized ADT in Result |
| 0:55 | M4: Full test suite | `make test && make verify-examples` |
| 1:15 | Manual verification | Run cli_args_demo.ail |
| 1:30 | Done | Commit and close |

## Success Metrics

- [ ] `std/env.ail` compiles without type errors
- [ ] Non-parameterized ADTs work correctly as type arguments
- [ ] `examples/runnable/cli_args_demo.ail` passes verification
- [ ] Unit test added for ADT type parameter arity
- [ ] `make test` passes (all ~500 tests)
- [ ] `make verify-examples` shows improved pass rate

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Fix breaks other builtins | Low | Medium | Run full test suite |
| Golden file conflicts | Low | Low | Update in same PR |
| Type inference regression | Low | High | Run full test suite |

## Dependencies

- None - this is a self-contained bug fix

## Velocity Analysis

Based on recent commits (last 7 days):
- M-LETREC-SCOPING fix: Similar complexity, completed in single session
- Average LOC/day: ~150-300 LOC for bug fixes
- This fix: ~45 LOC total (very small)

**Estimate confidence**: HIGH - root cause identified, fix is trivial

## Notes

- The design doc's "suspected causes" were slightly off - the issue is in builtin registration, not in type checker or exports
- This is a 1-line fix with golden file update and new test
- Should be completable in single session

---

**Created**: 2024-12-18
**Sprint Planner**: Claude Code
