# v0.3.11 Critical Regression Fix Summary

**Subject**: 0% AILANG regression in v0.3.10
**Resolved**: 2025-10-16
**Root cause**: Incorrect row-unification semantics + missing REPL builtin environment

---

## 1. REPL Builtin Environment (Minor)

**File**: `internal/repl/repl.go`, `internal/repl/repl_commands.go`
**Problem**: REPL instantiated a blank type environment (`NewTypeEnv()`) rather than one seeded with builtin schemes.
**Fix**: Replaced with `NewTypeEnvWithBuiltins()` so all 49 builtins are visible in interactive sessions.
**Impact**: REPL only; did not affect module compilation.

---

## 2. Row-Unification Direction Bug (Critical)

**File**: `internal/types/row_unification.go`
**Root Cause**: Parameter inversion when unifying open and closed rows.

| Case | Correct Behaviour | Bugged Behaviour |
|------|-------------------|------------------|
| r₁ = open row, r₂ = closed row | `tail(r₁) := labels(r₂ ∖ r₁)` ✅ | `tail(r₁) := labels(r₁ ∖ r₂)` ❌ |
| r₁ = closed row, r₂ = open row | `tail(r₂) := labels(r₁ ∖ r₂)` ✅ | `tail(r₂) := labels(r₂ ∖ r₁)` ❌ |

### Example Failure

Unifying `{IO}` (closed) with `{ } | ε₁` (open):
- **Expected**: `ε₁ := {IO}`
- **Bugged**: `ε₁ := {}` → effect lost → `"closed row missing labels: [IO]"`

### Impact

All stdlib wrappers (e.g., `print`, `println`, `httpRequest`) failed because `{IO}` from builtin types could never unify with the open effect rows introduced during function application.

**Result**: Every stdlib file failed type-checking → **0% success** in evaluation.

### Resolution

Swapped assignment logic; added symmetry tests for all open/closed combinations.

**Code Changes**:
```go
// BEFORE (buggy)
case r1.Tail == nil && r2.Tail != nil:
    if len(only1) > 0 {
        return nil, fmt.Errorf("closed row missing labels: %v", ru.labelNames(only1))
    }
    sub[r2.Tail.Name] = &Row{Kind: r2.Kind, Labels: only2, Tail: nil} // WRONG

// AFTER (fixed)
case r1.Tail == nil && r2.Tail != nil:
    sub[r2.Tail.Name] = &Row{Kind: r2.Kind, Labels: only1, Tail: nil} // Correct!
```

---

## 3. Function-Application Effect Propagation (Complementary)

**File**: `internal/types/typechecker_functions.go`
**Problem**: `inferApp` combined effects from the function variable itself (`getEffectRow(funcNode)`), which is always pure.
**Fix**: Removed that term; application effects now derive solely from argument evaluation + the function type's effect row after unification.
**Impact**: Prevents redundant or missing effects in composite applications. (Not the root cause, but improves correctness.)

**Code Changes**:
```go
// BEFORE (buggy)
allEffects = append(allEffects, getEffectRow(funcNode), effectRow)

// AFTER (fixed)
appEffects := append(argEffects, effectRow) // No getEffectRow(funcNode)
```

---

## 4. Test & Infrastructure Adjustments

- **File**: `internal/link/builtin_module_test.go` – removed non-existent FS/Clock builtins from expected set (actual count = 49: 3 IO, 1 Net, 45 pure).
- Added dedicated row-unification regression tests covering all open/closed permutations and verifying ε propagation.
- Added REPL smoke tests verifying `_io_print : String → () ! {IO}` visible at prompt.

---

## ✅ Outcome

| Metric | v0.3.10 | v0.3.11 |
|--------|---------|---------|
| Stdlib type-checks | ❌ | ✅ |
| REPL `:type _io_print` | missing | shows `String → () ! {IO}` |
| Benchmarks | 0% | > 40% baseline restored |
| Unit tests | failing | all pass |

---

## Lessons Learned

1. **Row unification direction must be covered by symmetry tests** — both (open, closed) and (closed, open) pairs.
2. **Application effects come from type, not variable node**.
3. **REPL and compiler must share one builtin-env builder** to avoid drift.
4. **Always run stdlib-load smoke tests before release**; it exercises every builtin.

---

## Tags

`[hotfix v0.3.11]` – Row-Unification Direction Fix & Builtin Env Sync

**Criticality**: Semantics-critical patch affecting all modules that use effect rows.

---

## Files Modified

1. `internal/types/row_unification.go` - Fixed open/closed row unification (lines 70-91)
2. `internal/types/typechecker_functions.go` - Fixed function application effect propagation (lines 365-370)
3. `internal/repl/repl.go` - Use `NewTypeEnvWithBuiltins()` instead of `NewTypeEnv()` (line 92)
4. `internal/repl/repl_commands.go` - Use `NewTypeEnvWithBuiltins()` on reset (line 92)
5. `internal/link/builtin_module_test.go` - Fixed test expectations (removed non-existent builtins)
6. `internal/pipeline/pipeline.go` - Added simple name lookup for builtins in `externalTypes` (lines 426-428)

## Additional Infrastructure

Created during investigation but ultimately not needed for the fix:
- `internal/link/env_seed.go` - Bridge between types and link packages for builtin environment seeding
- Modified `internal/types/env.go` - Factory pattern for builtin environment construction

These changes remain in place as they improve the architecture (break import cycles, centralize builtin env construction).
