# Common API Patterns

**Quick reference for AILANG internal APIs learned from M-TESTING and other sprints.**

## Key Principle

**⚠️ ALWAYS check `make doc PKG=<package>` before grepping or guessing APIs!**

API discovery with `make doc` is 80% faster than manual searching (5-10 min → 30 sec per lookup).

## Quick API Lookup

```bash
# Find constructor signatures
make doc PKG=internal/testing | grep "NewCollector"
# Output: func NewCollector(modulePath string) *Collector

# Find struct fields
make doc PKG=internal/ast | grep -A 20 "type FuncDecl"
# Shows: Tests []*TestCase, Properties []*Property
```

## Common Constructors

| Package | Constructor | Signature | Notes |
|---------|-------------|-----------|-------|
| `internal/testing` | `NewCollector(path)` | Takes module path | M-TESTING |
| `internal/elaborate` | `NewElaborator()` | No arguments | Surface → Core |
| `internal/types` | `NewTypeChecker(core, imports)` | Takes Core prog + imports | Type inference |
| `internal/link` | `NewLinker()` | No arguments | Dictionary linking |
| `internal/parser` | `New(lexer)` | Takes lexer instance | Parser |
| `internal/eval` | `NewEvaluator(ctx)` | Takes EffContext | Core evaluator |

## Common API Mistakes

### Test Collection (M-TESTING)

```go
// ✅ CORRECT
collector := testing.NewCollector("module/path")
suite := collector.Collect(file)
for _, test := range suite.Tests { ... }  // Tests is the slice!

// ❌ WRONG
collector := testing.NewCollector(file, modulePath)  // Wrong arg order!
for _, test := range suite.Tests.Cases { ... }      // No .Cases field!
```

### String Formatting

```go
// ✅ CORRECT
name := fmt.Sprintf("test_%d", i+1)

// ❌ WRONG - Produces "\x01" not "1"!
name := "test_" + string(rune(i+1))  // BUG!
```

### Field Access

```go
// ✅ CORRECT
funcDecl.Tests        // []*ast.TestCase
funcDecl.Properties   // []*ast.Property

// ❌ WRONG
funcDecl.InlineTests  // Doesn't exist! Use .Tests
```

## API Discovery Workflow

1. **`make doc PKG=<package>`** (~30 sec) ← Start here!
2. Check source file if you know location (`grep "^func New" file.go`)
3. Check test files for usage examples (`grep "NewCollector" *_test.go`)
4. Read [docs/guides/](../../../docs/guides/) for complex workflows

**Time savings**: 80% reduction (5-10 min → 30 sec per lookup)

## M-DX10: Unit-Argument Model for "Nullary" Builtins

**AILANG has no true nullary functions.** All functions that appear to take no arguments actually take a `unit` parameter.

### The Rule

| Surface syntax | Desugars to | Builtin registration |
|----------------|-------------|---------------------|
| `f()` | `f(())` | `NumArgs: 1`, type `() -> T` |
| `keys()` | `keys(())` | `NumArgs: 1`, type `() -> list[string]` |

### Implementing "Zero-Arg" Builtins

```go
// ✅ CORRECT - Unit-argument model
RegisterEffectBuiltin(BuiltinSpec{
    Name:    "_sharedmem_keys",
    NumArgs: 1,  // Takes unit parameter!
    Type:    func() types.Type {
        T := types.NewBuilder()
        return T.Func(T.Unit()).Returns(T.List(T.String())).Effects("SharedMem")
    },
    Impl: func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
        // args[0] is unit, ignored (but validates arity)
        // ... implementation
    },
})

// ❌ WRONG - True nullary (doesn't work!)
RegisterEffectBuiltin(BuiltinSpec{
    Name:    "_sharedmem_keys",
    NumArgs: 0,  // BUG: Will cause "arity mismatch: 0 vs 1"
    // ...
})
```

### Stdlib Wrappers

```ailang
-- ✅ CORRECT - Call with ()
export func keys(u: unit) -> list[string] ! {SharedMem} {
    _sharedmem_keys(u)
}
-- Or using expression body:
export func now() -> int ! {Clock} = _clock_now()

-- ❌ WRONG - Missing ()
export func now() -> int ! {Clock} = _clock_now  -- Returns function object!
```

### Go Tests for "Zero-Arg" Builtins

```go
// ✅ CORRECT - Pass unit value
result, err := sharedMemKeysImpl(ctx, []eval.Value{&eval.UnitValue{}})

// ❌ WRONG - Empty slice
result, err := sharedMemKeysImpl(ctx, []eval.Value{})  // Arity mismatch!
```

### Why This Model?

- **ML tradition**: Follows OCaml/SML where `()` is the canonical "no value"
- **Uniform semantics**: `f x` works for all functions; `f ()` is just applying unit
- **Higher-order friendly**: `let f = now` has type `() -> int ! {Clock}`
- **S-CALL0 sugar**: Parser automatically desugars `f()` → `f(())`

**Reference**: [M-DX10 design doc](../../../../design_docs/implemented/v0_4_6/m-dx10-nullary-function-calls.md)

## Full Reference

See CLAUDE.md "Common API Patterns" section for additional patterns and examples.
