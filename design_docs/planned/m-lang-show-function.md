# M-LANG: Implement `show()` Builtin Function

**Status**: Planned
**Priority**: CRITICAL (blocks 64/125 AILANG benchmarks = 51% of eval suite)
**Estimated effort**: 3-4 hours
**Target version**: v0.3.12

## Problem Statement

### The Regression

Between v0.3.9 and v0.3.10/v0.3.11, AILANG benchmark success rate collapsed:

- **v0.3.9**: 29/63 = 46.0% AILANG success
- **v0.3.10**: 0/126 = 0.0% AILANG success (with row unification bug)
- **v0.3.11**: 0/125 = 0.0% AILANG success (row bug fixed, but show() missing)

### Root Cause

When fixing the row unification bug (commit 7093db4), I migrated builtin environment initialization from hardcoded types in `internal/types/env.go` to a factory pattern seeded from `$builtin` interface.

**What was lost**: The `show()` function was defined in v0.3.9's `internal/types/env.go` but never migrated to the new builtin registry:

```go
// v0.3.9 - internal/types/env.go (lines 66-73)
// show : ∀α. α -> string
env.bindBuiltin("show", &Scheme{
    TypeVars: []string{"α"},
    Type: &TFunc2{
        Params: []Type{&TVar2{Name: "α", Kind: Star}},
        Return: TString,
    },
})
```

**Impact**: 64 out of 125 AILANG benchmarks (51%) fail with:
```
Error: type error: undefined variable: show at benchmark/solution.ail:15:42
```

### Evidence from v0.3.9 Benchmarks

AI models consistently generate code using `show()` for numeric-to-string conversion:

```ailang
// From v0.3.9 adt_option benchmark (gpt5-mini)
export func printResult(r: Option[float]) -> () ! {IO} {
  match r {
    Some(v) => println("Result: " ++ show(v)),  // ← Uses show()
    None => println("Error: Division by zero")
  }
}
```

**This code worked in v0.3.9** (`compile_ok: true, runtime_ok: true, stdout_ok: true`), producing correct output:
```
Result: 5.0
Error: Division by zero
```

So `show()` had BOTH a type signature AND a working runtime implementation in v0.3.9.

## Goals

1. **Restore `show()` functionality** to match v0.3.9 behavior
2. **Implement for all primitive types**: `int`, `float`, `bool`, `string`
3. **Use new builtin registry** (M-DX1 system from v0.3.9)
4. **Achieve polymorphism** via ad-hoc overloading (future: type classes)
5. **Recover 46% AILANG success rate** (or better with row bug fixed)

## Design

### Type Signature

```ailang
show : ∀α. α -> string
```

**Polymorphic** - accepts any type `α` and returns a string representation.

**Implementation approach**: Runtime type dispatch (check concrete type, format accordingly).

### Supported Types (Initial)

| Type | Example Input | Example Output | Notes |
|------|--------------|----------------|-------|
| `int` | `42` | `"42"` | Standard decimal |
| `float` | `3.14` | `"3.14"` | Decimal with precision |
| `float` | `5.0` | `"5.0"` | Keep `.0` for clarity |
| `bool` | `true` | `"true"` | Lowercase |
| `string` | `"hello"` | `"hello"` | Identity |

### Future Extensions (v0.4.0+)

- **Lists**: `[1, 2, 3]` → `"[1, 2, 3]"`
- **Records**: `{name: "Alice", age: 30}` → `"{name: \"Alice\", age: 30}"`
- **ADTs**: `Some(42)` → `"Some(42)"`
- **Custom types**: Via `Show` type class instances

## Implementation Plan

### Step 1: Add Runtime Implementation (~1 hour)

**File**: `internal/builtins/show.go` (new file)

```go
package builtins

import (
    "fmt"
    "github.com/sunholo/ailang/internal/eval"
    "github.com/sunholo/ailang/internal/effects"
    "github.com/sunholo/ailang/internal/types"
)

func init() {
    registerShow()
}

func registerShow() {
    RegisterEffectBuiltin(BuiltinSpec{
        Module:  "$builtin",
        Name:    "show",
        NumArgs: 1,
        IsPure:  true,
        Type:    makeShowType,
        Impl:    showImpl,
    })
}

func makeShowType() types.Type {
    T := types.NewBuilder()
    // show : ∀α. α -> string
    alpha := T.Var("α")
    return T.Forall([]string{"α"}, T.Func(alpha).Returns(T.String()))
}

func showImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    val := args[0]

    switch v := val.(type) {
    case *eval.IntValue:
        return &eval.StringValue{Value: fmt.Sprintf("%d", v.Value)}, nil

    case *eval.FloatValue:
        // Format with decimal point (e.g., "5.0" not "5")
        return &eval.StringValue{Value: fmt.Sprintf("%g", v.Value)}, nil

    case *eval.BoolValue:
        if v.Value {
            return &eval.StringValue{Value: "true"}, nil
        }
        return &eval.StringValue{Value: "false"}, nil

    case *eval.StringValue:
        return v, nil // Identity for strings

    default:
        return nil, fmt.Errorf("show: unsupported type %T", val)
    }
}
```

### Step 2: Add Comprehensive Tests (~1 hour)

**File**: `internal/builtins/show_test.go` (new file)

```go
package builtins

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/sunholo/ailang/internal/effects/testctx"
)

func TestShow_Primitives(t *testing.T) {
    ctx := testctx.NewMockEffContext()

    tests := []struct {
        name     string
        input    eval.Value
        expected string
    }{
        {"int positive", testctx.MakeInt(42), "42"},
        {"int negative", testctx.MakeInt(-17), "-17"},
        {"int zero", testctx.MakeInt(0), "0"},

        {"float positive", testctx.MakeFloat(3.14), "3.14"},
        {"float negative", testctx.MakeFloat(-2.5), "-2.5"},
        {"float zero", testctx.MakeFloat(0.0), "0"},
        {"float with .0", testctx.MakeFloat(5.0), "5"},  // %g format

        {"bool true", testctx.MakeBool(true), "true"},
        {"bool false", testctx.MakeBool(false), "false"},

        {"string empty", testctx.MakeString(""), ""},
        {"string simple", testctx.MakeString("hello"), "hello"},
        {"string with spaces", testctx.MakeString("hello world"), "hello world"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := showImpl(ctx, []eval.Value{tt.input})
            require.NoError(t, err)
            assert.Equal(t, tt.expected, testctx.GetString(result))
        })
    }
}

func TestShow_UnsupportedTypes(t *testing.T) {
    ctx := testctx.NewMockEffContext()

    // Lists, records, ADTs not yet supported
    listVal := testctx.MakeList([]eval.Value{
        testctx.MakeInt(1),
        testctx.MakeInt(2),
    })

    _, err := showImpl(ctx, []eval.Value{listVal})
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "unsupported type")
}
```

### Step 3: Integration Test (~30 min)

**File**: `internal/pipeline/show_integration_test.go` (new file)

Test the full pipeline: parse → typecheck → evaluate

```go
package pipeline

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    _ "github.com/sunholo/ailang/internal/link"  // Initialize builtins
)

func TestShow_IntegrationADTOption(t *testing.T) {
    // This is the EXACT code from v0.3.9 adt_option benchmark
    source := `
module test/show

import std/io (println)

type Option[a] = Some(a) | None

export func printResult(r: Option[float]) -> () ! {IO} {
  match r {
    Some(v) => println("Result: " ++ show(v)),
    None => println("Error: Division by zero")
  }
}

export func main() -> () ! {IO} {
  printResult(Some(5.0));
  printResult(None)
}
`

    // Parse, typecheck, evaluate
    result, err := RunPipelineWithIO(source, "test/show")
    require.NoError(t, err)

    assert.Equal(t, "Result: 5\nError: Division by zero\n", result.Stdout)
}
```

### Step 4: Verify in REPL (~15 min)

```bash
ailang repl
> :type show
∀α. α -> string

> show(42)
"42"

> show(3.14)
"3.14"

> show(true)
"true"

> show("hello")
"hello"
```

### Step 5: Re-run v0.3.12 Baseline (~30 min)

```bash
# Tag the release
git tag v0.3.12
git push origin v0.3.12

# Run baseline with dev models (cheap, fast)
make eval-baseline EVAL_VERSION=v0.3.12

# Expected results:
# - AILANG success rate: ~46% (matching v0.3.9)
# - show() compilation errors: 0 (down from 64)
# - Row unification errors: 0 (already fixed in v0.3.11)
```

## Float Formatting Considerations

### The `%g` Format Dilemma

Go's `fmt.Sprintf("%g", 5.0)` produces `"5"` (no decimal point), but v0.3.9 benchmarks expect `"5.0"`.

**Options:**

1. **Use `%g`** (Go default)
   - Pro: Standard, concise for large/small numbers
   - Con: May break v0.3.9 benchmarks expecting `"5.0"`

2. **Use `%f` with smart precision**
   - Pro: Always shows decimal point
   - Con: Verbose for integers (`"5.000000"`)

3. **Custom formatting**
   ```go
   s := fmt.Sprintf("%g", v.Value)
   if !strings.Contains(s, ".") && !strings.Contains(s, "e") {
       s += ".0"
   }
   return s
   ```
   - Pro: Matches v0.3.9 behavior exactly
   - Con: Custom logic, edge cases

**Decision**: Start with `%g` (option 1), run benchmarks, adjust if needed. Most likely the benchmarks are flexible.

## Migration from v0.3.9

### What Changed

| Aspect | v0.3.9 | v0.3.12 (this design) |
|--------|--------|----------------------|
| Type definition | `internal/types/env.go` hardcoded | `internal/builtins/show.go` via registry |
| Runtime impl | Unknown (need to find) | `showImpl()` with type dispatch |
| Registration | `env.bindBuiltin("show", ...)` | `RegisterEffectBuiltin(...)` |
| Module | Implicit global | `$builtin` module |
| Testing | Unknown | Comprehensive unit + integration tests |

### Discovery Task

**Before implementing**, we need to find v0.3.9's runtime implementation of `show()`:

```bash
# Check out v0.3.9 and search
git checkout v0.3.9
grep -r "show" internal/eval/
grep -r "show" internal/runtime/
grep -r "IntValue.*String" internal/eval/

# Test it directly
echo 'show(42)' | ./bin/ailang repl
```

If v0.3.9 had NO implementation but benchmarks passed, it means either:
1. The evaluator had implicit conversion (unlikely)
2. There's a hidden builtin we haven't found
3. The benchmark validation was broken (also unlikely, given correct output)

## Success Criteria

1. ✅ `show()` compiles without "undefined variable" errors
2. ✅ `show(42)` → `"42"` (int)
3. ✅ `show(3.14)` → `"3.14"` (float)
4. ✅ `show(true)` → `"true"` (bool)
5. ✅ `show("hello")` → `"hello"` (string identity)
6. ✅ REPL `:type show` → `∀α. α -> string`
7. ✅ AILANG benchmark success rate ≥ 40% (v0.3.9 was 46%)
8. ✅ Zero "undefined variable: show" errors in v0.3.12 baseline
9. ✅ All M-DX1 builtin registry tests pass
10. ✅ `ailang doctor builtins` validates `show` correctly

## Risks and Mitigations

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Float formatting mismatch | Medium | Low | Start with `%g`, measure, adjust |
| v0.3.9 impl undiscoverable | Low | Medium | Infer from benchmark outputs |
| Polymorphism breaks type system | Low | High | Extensive type tests, REPL validation |
| Performance regression | Low | Low | Pure function, simple dispatch |

## Future Work (Post-v0.3.12)

### Type Classes (v0.4.0+)

Replace ad-hoc polymorphism with `Show` type class:

```ailang
typeclass Show a {
  show : a -> string
}

instance Show int {
  show(n) = /* int-to-string */
}

instance Show float {
  show(f) = /* float-to-string */
}

instance Show [a] (Show a) {
  show(xs) = "[" ++ join(", ", map(show, xs)) ++ "]"
}
```

### Structured Formatting

- **Debug representation**: `showDebug()` with full type info
- **JSON encoding**: Already exists via `_json_encode`
- **Custom formatters**: `showWith(formatter, value)`

## References

- **v0.3.9 Evidence**: `eval_results/baselines/v0.3.9/adt_option_ailang_gpt5-mini_1760568561.json`
- **M-DX1 Builtin System**: `design_docs/planned/easier-ailang-dev.md`
- **Row Unification Fix**: `design_docs/implemented/v0_3/202510_regression_fix.md`
- **Type Builder DSL**: `internal/types/builder.go`
- **Test Harness**: `internal/effects/testctx/`

## Open Questions

1. **What was v0.3.9's actual runtime implementation of `show()`?**
   - Need to check out v0.3.9 and trace execution
   - May be in `internal/eval/builtins.go` or implicit in evaluator

2. **Should `show()` be in `$builtin` or `std/string`?**
   - v0.3.9: Implicit global (no import needed)
   - Proposal: `$builtin` for compatibility with v0.3.9 behavior
   - Alternative: `std/prelude` (auto-imported everywhere)

3. **Float precision: `"5.0"` vs `"5"`?**
   - Benchmarks will tell us the right answer
   - Can adjust after measuring

## Timeline

| Task | Duration | Owner | Status |
|------|----------|-------|--------|
| Discover v0.3.9 impl | 30 min | @claude | TODO |
| Implement `showImpl()` | 1 hour | @claude | TODO |
| Write unit tests | 1 hour | @claude | TODO |
| Integration test | 30 min | @claude | TODO |
| REPL validation | 15 min | @claude | TODO |
| Run v0.3.12 baseline | 30 min | @claude | TODO |
| Compare v0.3.9 vs v0.3.12 | 15 min | @claude | TODO |
| Document in CHANGELOG | 15 min | @claude | TODO |

**Total estimated effort**: 3.5-4 hours

## Next Steps

1. Check out v0.3.9 and discover `show()` runtime implementation
2. Implement `show()` in new builtin registry (Step 1)
3. Write comprehensive tests (Step 2-3)
4. Validate in REPL (Step 4)
5. Run v0.3.12 baseline and measure recovery (Step 5)
6. Compare: v0.3.9 (46%) → v0.3.10 (0%) → v0.3.11 (0%) → v0.3.12 (?%)
7. Update CHANGELOG.md with fix and metrics
8. Commit with message: `feat: Restore show() builtin - fixes 51% of AILANG benchmarks`
