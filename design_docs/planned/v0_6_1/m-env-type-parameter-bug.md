# M-ENV-TYPE: Fix std/env EnvError Type Parameter Bug

**Status**: Planned
**Target**: v0.6.1
**Priority**: P1 (High - breaks stdlib module)
**Estimated**: 2-4 hours
**Dependencies**: None

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | Bug fix - no change |
| Preserve Semantic Clarity | + | +1 | Fixes incorrect type interpretation |
| Increase Determinism | + | +1 | Type system should be consistent |
| Lower Token Cost | 0 | 0 | Bug fix - no change |
| **Net Score** | | **+2** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward

## Problem Statement

**Type System Bug**: Non-parameterized ADT `EnvError` is incorrectly treated as having 1 type parameter.

**Symptom:**
```
$ ./bin/ailang run --caps IO,Env --entry main examples/runnable/cli_args_demo.ail
Error: type error in std/env (decl 0): type unification failed at [return type annotation at
/Users/mark/dev/sunholo/ailang/std/env.ail:29:8]: failed to unify type argument 1:
type EnvError expects 1 type argument(s), but got 0 (did you mean EnvError[string]?)
```

**The Bug Location:**

In `std/env.ail`:
```ailang
-- Line 18-21: EnvError is a simple sum type with NO type parameters
export type EnvError =
  | NotFound(string)
  | NotAllowed(string)

-- Line 29: Using EnvError in Result fails
export func getEnv(name: string) -> Result[string, EnvError] ! {Env} = _env_getEnv(name)
```

**The type checker incorrectly believes `EnvError` needs a type argument, but it's defined without parameters.**

**Current State:**
- `std/env.ail` compiles when loaded alone
- Fails when used as import dependency
- The type `EnvError` is being confused with a parameterized type
- Possibly related to how types are exported/imported across modules

**Impact:**
- **Who**: All users of `std/env` module
- **Severity**: High - entire module unusable
- **Affected example**: `examples/runnable/cli_args_demo.ail`

## Goals

**Primary Goal:** Fix type system to correctly handle non-parameterized ADTs in cross-module imports.

**Success Metrics:**
- `std/env.ail` compiles and type-checks correctly
- `examples/runnable/cli_args_demo.ail` passes
- Non-parameterized types work correctly in `Result[T, E]` position
- No regressions in other type parameter handling

## Root Cause Analysis

**Suspected issue:**

When `Result[string, EnvError]` is type-checked:
1. `Result` is parameterized as `Result[a, e]`
2. `string` is substituted for `a` ✓
3. `EnvError` is substituted for `e` ✗
4. Type checker may be confusing `EnvError` type with a type **constructor** that needs arguments

**Possible causes:**

1. **Import Type Registration**
   - When `EnvError` is exported and imported, its arity may be incorrectly recorded
   - Check `internal/module/exports.go` or similar

2. **Type Constructor vs Type Confusion**
   - Type checker may treat all ADT type names as constructors needing args
   - Check `internal/types/typechecker.go` - how are imported types resolved?

3. **Result Type Instantiation**
   - The `e` parameter in `Result[a, e]` may expect a type constructor
   - Check how type parameters are unified with concrete types

**Investigation steps:**
```bash
# Debug type checking
DEBUG_TYPES=1 ./bin/ailang check std/env.ail

# Check if EnvError alone works
echo 'type E = A(int) | B(string)' | ./bin/ailang repl
# Then try using it in Result

# Check Result type definition
cat std/result.ail
```

## Solution Design

### Overview

Fix the type system to correctly identify `EnvError` as a fully-applied type (arity 0) when used in type expressions.

### Suspected Code Flow

**Current (buggy):**
1. Parse `Result[string, EnvError]`
2. Look up `EnvError` - finds ADT definition
3. **BUG**: Records arity as 1 (counts variants?) instead of 0
4. Fails: "expects 1 type argument"

**Correct:**
1. Parse `Result[string, EnvError]`
2. Look up `EnvError` - finds ADT with arity 0
3. Unify successfully - `EnvError` is a concrete type

### Implementation Plan

**Phase 1: Diagnosis** (~1 hour)
- [ ] Add debug logging to type parameter unification
- [ ] Trace where `EnvError` arity is computed
- [ ] Check export/import path for type definitions
- [ ] Identify exact line where arity mismatch occurs

**Phase 2: Fix** (~1-2 hours)
- [ ] Fix arity computation for non-parameterized ADTs
- [ ] Ensure exports preserve correct arity information
- [ ] Test with simple reproduction case first

**Phase 3: Testing** (~1 hour)
- [ ] Verify `std/env.ail` type-checks
- [ ] Verify `examples/runnable/cli_args_demo.ail` runs
- [ ] Add unit test for non-parameterized ADTs in Result
- [ ] Run full test suite

### Files to Modify

**Primary suspects:**
- `internal/types/typechecker.go` - Type parameter unification (~20 LOC)
- `internal/module/exports.go` - Type export handling (~10 LOC)
- `internal/types/env.go` - Type environment (~10 LOC)

**Testing:**
- `internal/types/typechecker_test.go` - Add ADT arity test (~30 LOC)

## Examples

### Example 1: Minimal Reproduction

**This should work:**
```ailang
-- error_type.ail
module test/error_type
import std/result (Result, Ok, Err)

export type MyError = NotFound(string) | Invalid(int)

export func test() -> Result[int, MyError] = Ok(42)
```

**Currently fails with:** "type MyError expects 1 type argument(s)"

### Example 2: cli_args_demo.ail

**Expected behavior:**
```
$ ./bin/ailang run --caps IO,Env --entry main examples/runnable/cli_args_demo.ail Alice
Received 1 argument:
Hello, Alice!
```

## Success Criteria

- [ ] `std/env.ail` compiles without type errors
- [ ] Non-parameterized ADTs work correctly as type arguments
- [ ] `examples/runnable/cli_args_demo.ail` passes verification
- [ ] Unit test added for ADT type parameter arity
- [ ] `make test` passes
- [ ] `make verify-examples` shows improved pass rate

## Testing Strategy

**Unit tests:**
- `TestADTArityZero` - Non-parameterized ADT as type argument
- `TestADTArityInResult` - ADT used in `Result[T, E]` position
- `TestImportedADTArity` - Cross-module ADT type checking

**Integration tests:**
- Compile and run `examples/runnable/cli_args_demo.ail`
- Compile `std/env.ail` as dependency

**Manual testing:**
```bash
# Quick smoke test
./bin/ailang check std/env.ail
./bin/ailang run --caps IO,Env --entry main examples/runnable/cli_args_demo.ail test
```

## Non-Goals

**Not in this fix:**
- Higher-kinded types - out of scope
- Generic ADT parameters - separate feature
- Better error messages for type arity - separate issue

## Timeline

**Day 1** (2-4 hours):
- Diagnosis and root cause identification
- Implement fix
- Test with examples
- Run full test suite
- Document fix

**Total: ~2-4 hours (single session fix)**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Fix breaks parameterized ADTs | High | Test both parameterized and non-parameterized |
| Type inference regression | High | Run full test suite |
| Cross-module imports break | Medium | Test multi-module scenarios |

## References

- `std/env.ail` - Affected stdlib module
- `std/result.ail` - Result type definition
- Example file: `examples/runnable/cli_args_demo.ail`

## Future Work

- Better error messages for type arity mismatches
- Type-level documentation for ADTs
- IDE hover showing type parameters

---

**Document created**: 2025-12-17
**Last updated**: 2025-12-17
