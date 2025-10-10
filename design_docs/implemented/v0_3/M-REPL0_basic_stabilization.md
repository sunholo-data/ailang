# M-REPL0: REPL Basic Stabilization

**Status**: ✅ IMPLEMENTED in v0.3.3
**Priority**: P1 (IMPORTANT)
**Actual Effort**: 200 LOC
**Duration**: 4 hours (2025-10-10)
**Shipped**: v0.3.3

## Summary

Fixed three critical REPL bugs that made basic operations fail:
1. ✅ Builtin resolver - Arithmetic operations now work
2. ✅ Persistent environment - Let bindings survive across inputs
3. ✅ Float equality - Direct literal comparisons work

## Problems Fixed

### Problem 1: No Builtin Resolver → Arithmetic Fails

**Before (v0.3.2)**:
```ailang
λ> 1 + 2
Runtime error: no resolver available to resolve global reference: $builtin.add_Int
```

**Root Cause**: REPL created bare `CoreEvaluator` without `GlobalResolver` for builtins.

**Fix**: Added `BuiltinOnlyResolver` to persistent evaluator ([internal/repl/repl.go:84-87](../../internal/repl/repl.go#L84-L87))

**After (v0.3.3)**:
```ailang
λ[IO]> 1 + 2
3 :: Int              ✅
```

### Problem 2: Let Bindings Don't Persist

**Before (v0.3.2)**:
```ailang
λ> let x = 42
() :: ()
λ> x
Runtime error: undefined variable: x
```

**Root Cause**: Each REPL input created new evaluator, environment was not shared.

**Fix**: Made evaluator persistent, shared environment across inputs ([internal/repl/repl.go:99](../../internal/repl/repl.go#L99))

**After (v0.3.3)**:
```ailang
λ[IO]> let x = 42
() :: ()
λ[IO]> x + 1
43 :: Int             ✅
```

### Problem 3: Float Equality Hits Wrong Implementation

**Before (v0.3.2)**:
```ailang
λ> 0.0 == 0.0
Runtime error: builtin eq_Int expects Int arguments, but received float
```

**Root Cause**: OpLowering pass chose `eq_Int` instead of `eq_Float` for float comparisons.

**Fix**: Enabled experimental binop shim for REPL ([internal/repl/repl.go:95](../../internal/repl/repl.go#L95))

**After (v0.3.3)**:
```ailang
λ[IO]> 0.0 == 0.0
true :: Bool          ✅

λ[IO]> 1.5 == 1.5
true :: Bool          ✅
```

## Implementation Details

### Files Modified

1. **`internal/repl/repl.go`** (~100 LOC)
   - Lines 84-95: Added persistent evaluator with builtin resolver and binop shim
   - Line 99: Shared evaluator environment for persistence
   - Lines 120-135: Added `getPrompt()` method to show active capabilities
   - Lines 632-660: Persist top-level let bindings in value and type environments
   - Lines 546-548: Skip type env update for Let (to avoid nested scope issues)

2. **`internal/types/env.go`** (~12 LOC)
   - Lines 154-166: Added `BindScheme()` and `BindType()` methods for REPL persistence

3. **`cmd/wasm/main.go`** (unchanged - WASM inherits REPL fixes)

### Code Changes

**Key change: Persistent evaluator with resolvers**
```go
// internal/repl/repl.go:82-95
evaluator := eval.NewCoreEvaluator()
builtinRegistry := runtime.NewBuiltinRegistry(evaluator)
builtinResolver := runtime.NewBuiltinOnlyResolver(builtinRegistry)
evaluator.SetGlobalResolver(builtinResolver)

effContext := effects.NewEffContext()
effContext.Grant(effects.NewCapability("IO"))
evaluator.SetEffContext(effContext)

evaluator.SetExperimentalBinopShim(true)  // For float equality
```

**Key change: Shared environment**
```go
// internal/repl/repl.go:99
r := &REPL{
    env:       evaluator.Env(), // Share the evaluator's environment
    // ...
}
```

**Key change: Persist let bindings**
```go
// internal/repl/repl.go:635-660
if letExpr, ok := elaboratedCore.(*core.Let); ok {
    val, err := r.evaluator.Eval(letExpr.Value)
    if err == nil {
        // Persist VALUE binding
        r.env.Set(letExpr.Name, val)

        // Persist TYPE binding
        if typedLet, ok := typedNode.(*typedast.TypedLet); ok {
            if typedLet.Scheme != nil {
                r.typeEnv.BindScheme(letExpr.Name, scheme)
            }
        }
    }
}
```

## Test Results

### Manual Testing
```bash
λ[IO]> 1 + 2
3 :: Int              ✅

λ[IO]> let x = 41
() :: ()              ✅

λ[IO]> x + 1
42 :: Int             ✅

λ[IO]> 0.0 == 0.0
true :: Bool          ✅

λ[IO]> { let y = 10; y * 5 }
50 :: Int             ✅
```

### Automated Tests
- ✅ All existing tests pass (`make test`)
- ✅ WASM build succeeds (`make build-wasm`)
- ✅ Browser playground functional

## Known Limitations

### Limitation 1: Type Annotations Lost

**Issue**: Type annotations from Surface AST are lost during elaboration.

```ailang
λ[IO]> let b: float = 0.0
() :: ()
λ[IO]> b
0.0 :: α1             # ❌ Type variable, not float!
λ[IO]> b == 0.0
Runtime error: builtin eq_Int expects Int arguments, but received float
```

**Root Cause**: Type annotations disappear in Surface → Core elaboration. Type checker only sees `let b = 0.0` (no annotation).

**Workaround**: Use direct literals instead of variables:
```ailang
λ[IO]> 0.0 == 0.0
true :: Bool          ✅ Works
```

**Fix Planned**: See [M-REPL1_persistent_bindings.md](../planned/M-REPL1_persistent_bindings.md)

### Limitation 2: Module Loading Not Supported

**Issue**: REPL can't load module files like `std/io`.

```ailang
λ[IO]> println("test")
Type error: undefined variable: println
```

**Root Cause**: `importModule()` only loads hardcoded type class instances, doesn't execute module files.

**Workaround**: None - `println` unavailable in REPL.

**Fix Planned**: See [M-REPL1_persistent_bindings.md](../planned/M-REPL1_persistent_bindings.md)

## Metrics

| Metric | Value |
|--------|-------|
| **Lines of code** | ~200 LOC |
| **Files modified** | 2 files |
| **Test coverage** | All existing tests pass |
| **Time to implement** | 4 hours |
| **Bugs fixed** | 3 critical |

## Impact

### Before vs After

| Operation | Before (v0.3.2) | After (v0.3.3) |
|-----------|-----------------|----------------|
| `1 + 2` | ❌ "no resolver" error | ✅ `3 :: Int` |
| `let x = 42; x` | ❌ "undefined variable" | ✅ `42 :: Int` |
| `0.0 == 0.0` | ❌ "eq_Int error" | ✅ `true :: Bool` |
| Prompt | `λ>` | `λ[IO]>` |

### User Experience
- ✅ REPL now suitable for basic demos and testing
- ✅ Arithmetic operations work as expected
- ✅ Let bindings persist across inputs
- ✅ Float comparisons work (with direct literals)
- ✅ Capability prompt shows active capabilities

## Next Steps

**Immediate (v0.3.4)**:
- Document REPL limitations in user guide
- Update REPL examples in README

**Future (v0.4.0)**:
- Implement M-REPL1 (type annotation persistence + module loading)
- Add REPL commands (`:clear`, `:caps`)
- Auto-import common modules

## Related Work

**Builds on**:
- Module execution runtime (v0.2.0)
- Effect system (v0.2.0)
- Type class system (v0.1.0)

**Enables**:
- Browser-based playground (WASM)
- Interactive AI code demos
- Quick prototyping

**Follow-up**:
- [M-REPL1: Type Bindings & Module Loading](../planned/M-REPL1_persistent_bindings.md)

## Lessons Learned

1. **Test REPL separately from module execution** - REPL had different code path without resolver
2. **Persistent state needs careful design** - Environment sharing was key to fixing bindings
3. **Experimental flags can unblock progress** - Binop shim fixed float equality temporarily
4. **Type annotations need pipeline support** - Can't fix in REPL alone, needs elaboration changes

## References

- Issue: Float equality bug (2025-10-10)
- User feedback: REPL arithmetic broken
- Design: Improve REPL usability for demos

---

**Document Version**: v1.0
**Created**: October 10, 2025
**Author**: AILANG Development Team
**Released**: v0.3.3
