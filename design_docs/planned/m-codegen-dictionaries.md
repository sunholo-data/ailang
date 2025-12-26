# M-CODEGEN-DICTIONARIES: Generate Type Class Dictionary Implementations for Go

**Status:** Planned
**Priority:** High (blocks contract integration tests)
**Discovered:** 2025-12-26 during M-DX19 implementation
**Affects:** Go code generation (`ailang compile --emit-go`)

## Problem

The Go code generator emits references to type class dictionaries but never generates the dictionary struct definitions:

```go
// Generated code references these but they're never defined:
dict_Num_Int.Add(x, y)
dict_Num_Int.Div(dividend, divisor)
dict_Ord_Int.Gt(tmp1, max)
dict_Ord_Int.Lt(value, minVal)
```

This causes compilation failures:
```
./basic.go:26:9: undefined: dict_Num_Int
./basic.go:44:25: undefined: dict_Ord_Int
```

## Root Cause

In `internal/gen/golang/codegen_expr.go:74-77`:
```go
case *core.DictRef:
    // Dictionary references are runtime-resolved
    g.writef("dict_%s_%s", e.ClassName, e.TypeName)
    return nil
```

The code outputs dictionary variable references but no code generates the actual dictionary implementations.

## Impact

- `TestContractViolation_Integration` in `internal/gen/golang/contracts_integration_test.go` fails
- Any AILANG program using operators (`+`, `-`, `*`, `/`, `<`, `>`, etc.) on Int/Float fails to compile to Go
- The `ailang compile --emit-go` feature is broken for most real programs

## Solution

### Phase 1: Generate Built-in Dictionaries (Priority: High)

Generate dictionary struct definitions for the standard type classes and types:

**File: `dictionaries.go` (new)**

```go
// dict_Num_Int provides Num type class methods for int64
var dict_Num_Int = struct {
    Add func(interface{}, interface{}) interface{}
    Sub func(interface{}, interface{}) interface{}
    Mul func(interface{}, interface{}) interface{}
    Div func(interface{}, interface{}) interface{}
    Neg func(interface{}) interface{}
    Abs func(interface{}) interface{}
}{
    Add: func(x, y interface{}) interface{} { return x.(int64) + y.(int64) },
    Sub: func(x, y interface{}) interface{} { return x.(int64) - y.(int64) },
    Mul: func(x, y interface{}) interface{} { return x.(int64) * y.(int64) },
    Div: func(x, y interface{}) interface{} { return x.(int64) / y.(int64) },
    Neg: func(x interface{}) interface{} { return -x.(int64) },
    Abs: func(x interface{}) interface{} {
        v := x.(int64)
        if v < 0 { return -v }
        return v
    },
}

var dict_Ord_Int = struct {
    Lt  func(interface{}, interface{}) interface{}
    Gt  func(interface{}, interface{}) interface{}
    Lte func(interface{}, interface{}) interface{}
    Gte func(interface{}, interface{}) interface{}
}{
    Lt:  func(x, y interface{}) interface{} { return x.(int64) < y.(int64) },
    Gt:  func(x, y interface{}) interface{} { return x.(int64) > y.(int64) },
    Lte: func(x, y interface{}) interface{} { return x.(int64) <= y.(int64) },
    Gte: func(x, y interface{}) interface{} { return x.(int64) >= y.(int64) },
}

var dict_Eq_Int = struct {
    Eq  func(interface{}, interface{}) interface{}
    Neq func(interface{}, interface{}) interface{}
}{
    Eq:  func(x, y interface{}) interface{} { return x.(int64) == y.(int64) },
    Neq: func(x, y interface{}) interface{} { return x.(int64) != y.(int64) },
}
```

**Required dictionaries:**
- `dict_Num_Int`, `dict_Num_Float` (arithmetic)
- `dict_Ord_Int`, `dict_Ord_Float`, `dict_Ord_String` (comparison)
- `dict_Eq_Int`, `dict_Eq_Float`, `dict_Eq_String`, `dict_Eq_Bool` (equality)

### Phase 2: Generate Derived Eq Dictionaries (M-DX19)

For ADT types with `deriving (Eq)`, generate structural equality:

```go
// Generated for: type Color = Red | Green | Blue deriving (Eq)
var dict_Eq_Color = struct {
    Eq  func(interface{}, interface{}) interface{}
    Neq func(interface{}, interface{}) interface{}
}{
    Eq: func(x, y interface{}) interface{} {
        a, b := x.(Color), y.(Color)
        return a.Tag == b.Tag && reflect.DeepEqual(a.Fields, b.Fields)
    },
    Neq: func(x, y interface{}) interface{} {
        return !dict_Eq_Color.Eq(x, y).(bool)
    },
}
```

### Phase 3: Track Required Dictionaries

Instead of generating all possible dictionaries, track which ones are actually used:

1. During code generation, collect `DictRef` nodes
2. Build set of required `(ClassName, TypeName)` pairs
3. Generate only needed dictionaries

## Implementation Plan

### Files to Modify

1. **`internal/gen/golang/codegen_dictionaries.go`** (NEW)
   - `generateDictionaries()` - emit dictionary definitions
   - `collectRequiredDictionaries()` - analyze which dictionaries are needed
   - Built-in implementations for Num, Ord, Eq on primitives

2. **`internal/gen/golang/codegen.go`**
   - Call `generateDictionaries()` after generating functions
   - Pass collected dictionary requirements

3. **`internal/gen/golang/codegen_expr.go`**
   - Track `DictRef` usage during generation

### Testing

1. Fix `TestContractViolation_Integration`
2. Add unit tests for dictionary generation
3. Add test for derived Eq dictionaries (M-DX19)

## Risks

1. **Type mismatch**: Dictionary methods use `interface{}` but might need proper type handling
2. **Performance**: Using `interface{}` adds boxing overhead (acceptable for now)
3. **Derived types**: Need to handle generic ADT types carefully

## Alternatives Considered

1. **Inline operations**: Instead of `dict_Num_Int.Add(x, y)`, generate `x.(int64) + y.(int64)`
   - Pro: Simpler, no dictionary generation
   - Con: Loses the dictionary-passing design, harder to extend

2. **Runtime dictionary lookup**: Generate code that looks up dictionaries at runtime
   - Pro: More flexible
   - Con: More runtime overhead, harder to compile

## Decision

Generate static dictionary implementations for:
1. All built-in type class/type combinations (Num/Int, Num/Float, Ord/*, Eq/*)
2. Derived Eq instances for ADT types with `deriving (Eq)`

This matches the current codegen design pattern and is the minimal fix to unblock the integration tests.

## References

- `internal/gen/golang/codegen_expr.go:74-77` - Current DictRef generation
- `internal/gen/golang/contracts_integration_test.go:242` - Failing test
- M-DX19 (this session) - Auto-derive Eq for ADT types
