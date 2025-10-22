# IO Output Not Appearing in Module Execution

**Status**: 🐛 BUG - Needs Investigation
**Priority**: P1 (MEDIUM - Testing/Debugging)
**Estimated Effort**: 4-8 hours debugging
**Target Release**: v0.3.6
**Created**: 2025-10-13

---

## Problem Statement

When running modules with IO effects, `println()` and `print()` produce no output, even though the code type-checks and runs without errors.

### Reproduction

```ailang
module tmp/test
import std/io (println)

func main() -> () ! {IO} = println("TEST OUTPUT")
```

```bash
$ ailang run --caps IO --entry main test.ail
→ Type checking...
→ Effect checking...
✓ Running test.ail
# ❌ No output appears - should print "TEST OUTPUT"
# ✅ Exit code: 0 (success)
```

---

## Current Findings

### What We Know

1. **Effect Handlers Exist** (`internal/effects/io.go:74`):
   ```go
   func ioPrintln(ctx *EffContext, args []eval.Value) (eval.Value, error) {
       // ...
       fmt.Println(str.Value)  // ← This SHOULD print to stdout
       return &eval.UnitValue{}, nil
   }
   ```

2. **Code Type-Checks**: IO effect is recognized, capability granted
   - `--caps IO` flag works
   - No type errors
   - Effect checking passes

3. **Code Runs**: Exit code 0, no runtime errors
   - Function executes
   - Returns UnitValue
   - No panics or crashes

4. **But**: No output appears on stdout/stderr

### What This Suggests

The most likely cause: **Effect handler is not being called**

Possible reasons:
1. Stdlib wrapper (`std/io.ail`) doesn't route through effect system
2. Module execution uses different builtin resolution path
3. EffContext not properly initialized in module runtime
4. Output buffering issue (unlikely - `fmt.Println` auto-flushes)

---

## Investigation Plan

### Step 1: Verify Effect Handler Is Registered (5 min)
```go
// Add debug logging to internal/effects/io.go:
func ioPrintln(ctx *EffContext, args []eval.Value) (eval.Value, error) {
    fmt.Fprintln(os.Stderr, "DEBUG: ioPrintln called!")  // ← Add this
    // ... existing code
}
```

Run test again - if we see "DEBUG: ioPrintln called!" then handler IS running.

### Step 2: Check Builtin Resolution Path (30 min)

Two builtin implementations exist:
1. `internal/eval/builtins.go:768` - Direct `fmt.Println()`
2. `internal/runtime/builtins.go:99` - Routes through effect system

Which one is used in module execution?

```go
// Check: internal/runtime/builtins.go
func (br *BuiltinRegistry) registerEffectBuiltins() {
    br.builtins["_io_println"] = &eval.BuiltinFunction{
        Fn: func(args []eval.Value) (eval.Value, error) {
            ctx := br.getEffContext()  // ← Is this returning nil?
            if ctx == nil {
                return nil, fmt.Errorf("_io_println: no effect context available")
            }
            return effects.Call(ctx, "IO", "println", args)
        },
    }
}
```

Add debug logging to check if `ctx` is nil.

### Step 3: Trace Stdlib Wrapper (15 min)

Check `stdlib/std/io.ail`:
```ailang
export func println(s: string) -> () ! {IO} = _io_println(s)
```

Is `_io_println` resolving correctly?

Add debug logging to:
- `internal/runtime/resolver.go` - Global resolution
- `internal/eval/eval_core.go:evalCoreVarGlobal()` - Variable evaluation

### Step 4: Check Module Runtime Initialization (30 min)

File: `internal/runtime/module_runtime.go`

Does module execution initialize EffContext properly?

```go
// Expected initialization:
effCtx := effects.NewEffContext()
effCtx.Grant("IO")  // Grant IO capability
// ... pass effCtx to evaluator
```

Compare with REPL initialization to see what's different.

### Step 5: Compare REPL vs Module Execution (30 min)

REPL might work differently:
- Test if REPL IO works (it might not - needs import support)
- Compare initialization code paths
- Identify where they diverge

---

## Quick Test Cases

### Test 1: Direct Builtin Call
```ailang
func main() -> () ! {IO} = _io_println("Direct builtin")
```
If this works, problem is in stdlib wrapper.

### Test 2: Simple Expression
```ailang
func main() -> int = 42
```
Verify basic execution works (it should).

### Test 3: Effect-Free IO
```go
// Bypass effect system - add to builtins:
Builtins["_debug_print"] = &BuiltinFunc{
    Impl: func(s *StringValue) (*UnitValue, error) {
        fmt.Fprintln(os.Stderr, "DEBUG:", s.Value)
        return &UnitValue{}, nil
    },
}
```

Use `_debug_print("test")` - if this works, problem is effect system.

---

## Debugging Checklist

- [ ] Add debug logging to `ioPrintln()` handler
- [ ] Check which builtin implementation is used
- [ ] Verify EffContext is not nil
- [ ] Trace `println` resolution path
- [ ] Compare module vs REPL initialization
- [ ] Test direct `_io_println` call
- [ ] Test effect-free debug print
- [ ] Check for output buffering issues
- [ ] Verify stdout isn't being redirected

---

## Expected Resolution

### If Handler Not Called

**Root Cause**: Wrong builtin resolution path or missing EffContext

**Fix**: Ensure module runtime uses effect-based builtins
- File: `internal/runtime/module_runtime.go`
- Initialize EffContext
- Wire it to builtin registry
- ~50-100 LOC fix

### If Handler Called But No Output

**Root Cause**: Output redirection or buffering

**Fix**: Ensure stdout is properly configured
- Check terminal/stdio setup
- Flush output buffers explicitly
- ~10-20 LOC fix

### If Stdlib Wrapper Issue

**Root Cause**: `_io_println` not resolving to effect handler

**Fix**: Debug global variable resolution
- File: `internal/runtime/resolver.go`
- Ensure builtins with `_` prefix route correctly
- ~20-30 LOC fix

---

## Impact

### Current Workarounds

1. **Can't test IO in modules** - No way to verify output
2. **Can't debug programs** - Printf debugging doesn't work
3. **M-EVAL validation harder** - Need to trust type checking only

### Once Fixed

1. ✅ Printf debugging works
2. ✅ Can write integration tests with IO
3. ✅ Better development experience
4. ✅ M-EVAL validation more reliable

---

## Related Issues

- REPL import support (needs `import std/io` to work)
- Effect system wiring in different execution contexts
- Builtin resolution consistency

---

## References

- `internal/effects/io.go:74` - IO effect handlers
- `internal/runtime/builtins.go:99` - Effect builtin registration
- `internal/eval/builtins.go:768` - Direct builtin implementation
- `stdlib/std/io.ail` - Stdlib wrapper functions

---

## Next Steps

1. **Immediate**: Add debug logging to narrow down issue
2. **Short-term**: Fix identified root cause
3. **Long-term**: Ensure consistency across execution contexts (module, REPL, tests)
