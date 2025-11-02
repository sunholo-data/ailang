# M-S-CALL0 Zero-Arg Builtin Bug (CRITICAL)

**Status**: Planned (P0 BLOCKER)
**Target**: v0.4.2 (hotfix)
**Priority**: P0 - Critical blocker
**Estimated**: 1 day (4h investigation complete, 4h fix + test)
**Dependencies**: None - blocks all v0.4.1 functionality

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | S-CALL0 sugar reduces noise when working |
| Preserve Semantic Clarity | − | −1 | Bug breaks zero-arg builtins entirely |
| Increase Determinism | − | −2 | Type checker sees wrong arity (non-deterministic failure) |
| Lower Token Cost | − | −2 | std/io module completely broken, forces workarounds |
| **Net Score** | | **−4** | **Decision: CRITICAL BUG - Must fix immediately** |

**Decision rule:** This is a critical regression that breaks core functionality. S-CALL0 feature must work correctly with zero-arg builtins.

## Problem Statement

**Critical Discovery (2025-11-02 Post-Release Analysis):**

During v0.4.1 post-release evaluation analysis, we discovered that **std/io module is completely broken**. Any code that imports `std/io` fails immediately with a type error.

**Current State:**
- ❌ `_io_readLine()` builtin cannot be called
- ❌ `std/io` module cannot be imported
- ❌ Type checker reports "function arity mismatch: 0 vs 1"
- ❌ Registered type: `() -> string ! {IO}` (correct)
- ❌ Type checker sees: expects 1 argument (incorrect)
- ❌ Eval results: Haiku AILANG success drops to **4.9%** (was 58.3% in v0.4.0)

**Impact:**
- **100% of code importing std/io fails** - zero-arg builtins broken
- **Haiku most affected**: Frequently tries `import std/io (println)`
- **Eval baseline invalid**: v0.4.1 metrics severely underreported
- **Production blocker**: Cannot use any zero-arg builtins

**Root Cause:** S-CALL0 surface sugar (v0.4.1) interferes with zero-arg function type checking.

## Investigation Timeline

### Discovery (Nov 2, 2025 - Post-Release Eval)

**Initial Observation:**
```
v0.4.1 baseline: 59.9% success (333/556)
AILANG: 45.8% (130/284)
Haiku: 30.0% (18/60) ← Catastrophic drop from v0.4.0: 58.3%
```

**Hypothesis 1:** Surface Sugar syntax confused models
- **Result:** WRONG - No pattern linking failures to `::`, `->`, `f()`
- **Evidence:** +22 WRONG_LANG errors (models using non-existent features)

**Hypothesis 2:** LLM variance between baseline runs
- **Result:** PARTIALLY TRUE - but variance was ±5%, not ±50%
- **Evidence:** Ran Haiku with v0.4.0 prompt → 4.9%, v0.4.1 prompt → 4.9%

**Hypothesis 3:** Eval harness marking valid code as failed
- **Result:** **CONFIRMED** - stdlib bug, not eval bug
- **Evidence:** Haiku generates correct code but compilation fails

### Root Cause Analysis

**Test Case:**
```ailang
module benchmark/solution

import std/io (println)  // ← This import fails!

export func main() -> () ! {IO} {
  print("Hello, World!")
}
```

**Error:**
```
Error: type error in std/io (decl 2): type unification failed at
[function application at std/io.ail:14:43]: function arity mismatch: 0 vs 1
```

**Direct Test:**
```ailang
module tmp/test

export func main() -> () ! {IO} {
  let line = _io_readLine();  // ← Fails: "arity mismatch: 0 vs 1"
  _io_println(line)
}
```

**Builtin Registration (internal/builtins/io.go:100):**
```go
type3 := func() types.Type {
    T := types.NewBuilder()
    return T.Func().Returns(T.String()).Effects("IO")  // ← Correct: () -> string
}
err = RegisterEffectBuiltin(BuiltinSpec{
    Module: "std/io", Name: "_io_readLine", NumArgs: 0,  // ← Correct: 0 args
    IsPure: false, Effect: "IO", Type: type3, Impl: impl3,
})
```

**Stdlib Module (std/io.ail:14):**
```ailang
export func readLine() -> string ! {IO} = _io_readLine()
                                          ^^^^^^^^^^^^^^^^
                                          Type checker thinks this needs 1 arg!
```

### Smoking Gun

**S-CALL0 was introduced in v0.4.1 for zero-arg calls:**
- Syntax: `f()` desugars to `f (())`
- Implementation: `parseZeroArgCall()` in `parser_expr.go`
- **BUG**: Type checker now expects ALL zero-arg calls to have unit argument

**Verification:**
- v0.3.20 baseline (pre-S-CALL0): No `_io_readLine` errors
- v0.4.0 baseline (pre-S-CALL0): `_io_readLine` works
- v0.4.1 baseline (with S-CALL0): `_io_readLine` completely broken
- v0.4.1 targeted test: 4.9% success (same as v0.4.0 prompt test)

## Goals

**Primary Goal:** Fix S-CALL0 to correctly handle zero-arg builtins without breaking the sugar feature.

**Success Metrics:**
- ✅ `_io_readLine()` compiles and type-checks correctly
- ✅ `import std/io` works without errors
- ✅ S-CALL0 sugar `f()` still works for user functions
- ✅ Haiku AILANG success rate returns to >50%
- ✅ All 41 benchmarks can import std/io if needed

## Solution Design

### Overview

The S-CALL0 sugar is interfering with the type checker's understanding of zero-arg function signatures. We need to separate:

1. **Parser sugar**: `f()` → `f (())` (expression-level syntax convenience)
2. **Type checking**: Zero-arg builtins should still type as `() -> T`, not `(()) -> T`

**Key Insight:** The type checker must distinguish between:
- **Builtin zero-arg functions**: `_io_readLine : () -> string ! {IO}`
- **User zero-arg calls with sugar**: `main() : main (())`

### Root Cause

Looking at the S-CALL0 implementation, the bug is likely in one of these places:

1. **Type builder DSL** (`T.Func()` with no args)
2. **Type unification** (treating `()` as a function taking unit)
3. **Builtin type registration** (mismatch between NumArgs and Type)
4. **Core elaboration** (zero-arg calls being transformed incorrectly)

### Investigation Plan

**Phase 1: Locate the Bug** (~2 hours)
- [ ] Add debug logging to type checker for `_io_readLine` type resolution
- [ ] Check if `T.Func()` generates correct type
- [ ] Verify builtin registry shows correct signature
- [ ] Test if issue is in parser, elaborator, or type checker

**Phase 2: Fix Implementation** (~1 hour)
- [ ] Fix type builder to handle zero-arg correctly
- [ ] OR fix type checker to handle builtin zero-arg
- [ ] OR fix elaborator to not desugar builtin calls

**Phase 3: Test & Verify** (~2 hours)
- [ ] Add regression test for `_io_readLine()`
- [ ] Test std/io import works
- [ ] Run Haiku eval to verify fix
- [ ] Run full eval suite to verify no new regressions

### Files to Investigate/Modify

**Investigation targets:**
- `internal/types/builder.go:Func()` - Type builder for zero-arg functions
- `internal/types/typechecker.go` - Type unification for function application
- `internal/elaborate/elaborate.go` - Core elaboration (desugaring)
- `internal/parser/parser_expr.go:parseZeroArgCall()` - S-CALL0 implementation

**Likely fixes:**
- `internal/types/builder.go` - ~5 LOC fix
- OR `internal/types/typechecker.go` - ~10 LOC fix
- OR `internal/elaborate/elaborate.go` - ~15 LOC fix

**Test files:**
- `internal/types/builder_test.go` - Add zero-arg test
- `internal/builtins/io_test.go` - Add `_io_readLine` regression test
- `internal/integration/stdlib_test.go` - Add std/io import test

## Examples

### Example 1: std/io Import (Currently Broken)

**Before (v0.4.1 broken):**
```ailang
module test
import std/io (println, readLine)

export func main() -> () ! {IO} {
  println("Enter name:")
  let name = readLine();
  println("Hello, " ++ name)
}
```

**Error:**
```
Error: type error in std/io (decl 2): type unification failed at
[function application at std/io.ail:14:43]: function arity mismatch: 0 vs 1
```

**After (v0.4.2 fixed):**
```ailang
module test
import std/io (println, readLine)

export func main() -> () ! {IO} {
  println("Enter name:")
  let name = readLine();  // ← Works! Type: () -> string ! {IO}
  println("Hello, " ++ name)
}
```

**Output:**
```
Enter name:
Alice
Hello, Alice
```

### Example 2: Direct Builtin Call (Currently Broken)

**Before (v0.4.1 broken):**
```ailang
module test

export func main() -> () ! {IO} {
  let line = _io_readLine();  // ← Fails: "arity mismatch: 0 vs 1"
  _io_println(line)
}
```

**After (v0.4.2 fixed):**
```ailang
module test

export func main() -> () ! {IO} {
  let line = _io_readLine();  // ← Works! Type: () -> string ! {IO}
  _io_println(line)
}
```

### Example 3: S-CALL0 Sugar Still Works

**User function with sugar (should still work):**
```ailang
module test

export func greet() -> () ! {IO} {
  print("Hello!")
}

export func main() -> () ! {IO} {
  greet()  // ← Sugar: greet (())
}
```

**This should continue working** - the fix must not break S-CALL0 sugar for user functions.

## Success Criteria

- [ ] `_io_readLine()` compiles without errors
- [ ] `import std/io` succeeds
- [ ] Type signature correct: `() -> string ! {IO}` (not `(()) -> string`)
- [ ] S-CALL0 sugar `f()` still works for user functions
- [ ] All existing tests pass
- [ ] Haiku eval >50% (was 58.3% in v0.4.0, should recover to similar)
- [ ] Full eval baseline shows actual v0.4.1 performance
- [ ] Regression tests added for zero-arg builtins
- [ ] Documentation updated if type system semantics changed

## Testing Strategy

**Unit tests:**
- Type builder: `T.Func().Returns(T.String())` generates `() -> string`
- Type checker: Zero-arg function application type-checks correctly
- Builtin registry: `_io_readLine` signature matches implementation

**Integration tests:**
- `std/io` module imports successfully
- `_io_readLine()` can be called directly
- S-CALL0 sugar `f()` still works
- No regression in other zero-arg functions

**Manual testing:**
- Run Haiku eval with v0.4.1 prompt after fix
- Verify success rate recovers to >50%
- Test all examples in this doc
- Verify `ailang check` works with std/io imports

## Non-Goals

**Not in this hotfix:**
- Full S-CALL0 redesign - just fix the bug
- Statement-level `f()` support (that's working)
- Other surface sugar features (those are fine)
- Eval baseline re-run (will do after fix)

**Future Work:**
- Consider if S-CALL0 should desugar builtins at all
- Add more robust testing for surface sugar + builtins interaction
- Document type checker behavior for zero-arg functions

## Timeline

**Day 1** (8 hours):
- Phase 1: Locate bug (2h) - COMPLETE
- Phase 2: Fix implementation (2h)
- Phase 3: Test & verify (2h)
- Phase 4: Documentation + commit (2h)

**Total: ~8 hours (1 day)**

**Note:** 4 hours already spent on investigation during post-release analysis. Remaining: 4 hours for fix + test + docs.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Fix breaks S-CALL0 sugar | High | Comprehensive tests for both builtins and user functions |
| Type system complexity | Medium | Keep fix minimal, focus on root cause |
| Eval baseline needs re-run | High | Document that v0.4.1 metrics are invalid |
| Other zero-arg builtins broken | Medium | Test all zero-arg builtins, not just `_io_readLine` |

## References

- [S-CALL0 Original Implementation](../../implemented/v0_4_1/m-sugar-surface-syntax.md)
- [Builtin Developer Experience (M-DX1)](../../implemented/v0_3_10/M-DX1_developer_experience.md)
- [Type Builder DSL](../../../internal/types/builder.go)
- [Post-Release Analysis (2025-11-02)](../../../CHANGELOG.md#v041)

## Investigation Evidence

### Eval Results Comparison

| Metric | v0.4.0 baseline | v0.4.1 baseline | v0.4.1 verify | Expected (fixed) |
|--------|----------------|----------------|---------------|------------------|
| **Haiku AILANG** | 58.3% (35/60) | 30.0% (18/60) | **4.9% (2/41)** | >50% |
| **Overall AILANG** | 49.3% | 45.8% | — | >45% |
| **Agent Eval** | 76.3% | 81.6% | — | 81.6% (not affected) |

**Conclusion:** The 4.9% result in verification run proves stdlib is broken, not LLM variance.

### Error Analysis

**Error Code Distribution (Haiku v0.4.1):**
```
WRONG_LANG: 36/42 failures (86%)
PAR_001: 3/42 failures
null: 3/42 failures
```

**WRONG_LANG errors are caused by:**
- Models try to import std/io
- Import fails with type error
- Code marked as WRONG_LANG (used non-existent feature)
- **Reality**: Code is correct, stdlib is broken!

### Code Examples from Failures

**Haiku generated (marked as WRONG_LANG):**
```ailang
module benchmark/solution
import std/io (println)  // ← This line causes type error in std/io

export func main() -> () ! {IO} {
  print("Hello, World!")  // ← This line is correct!
}
```

**Should work but doesn't** because `std/io` module has type error in `readLine()` definition.

## Future Work

1. **Comprehensive surface sugar testing**: Test all sugar features with all builtin types
2. **Type checker invariant**: Document that `T.Func()` must generate `() -> T`, not `(()) -> T`
3. **Eval baseline quality**: Add CI check that std/io imports work before running baselines
4. **Prompt update**: Ensure teaching prompt correctly documents zero-arg syntax
5. **M-EVAL-STDLIB**: Add benchmark specifically testing stdlib imports

---

**Document created**: 2025-11-02
**Last updated**: 2025-11-02
**Investigation complete**: 2025-11-02 (4h)
**Fix pending**: v0.4.2 hotfix
