# M-CODEGEN-BOOL-ASSERTIONS: Type assertions for boolean contexts in Go codegen

**Status**: PLANNED
**Priority**: High (blocks `deriving (Eq)` in Go codegen)
**Estimated LOC**: ~50-100
**Related**: M-CODEGEN-DICTIONARIES, M-DX19

## Problem Statement

When compiling AILANG code to Go, expressions that return `interface{}` (such as dictionary method calls) are used directly in boolean contexts without type assertions. This causes Go compilation errors.

### Example

**AILANG code** (`examples/deriving_eq.ail`):
```ailang
type Color = Red | Green | Blue deriving (Eq)

let colorTest: bool =
  let c1 = Red in
  let c2 = Red in
  let sameColor = c1 == c2 in     -- Returns bool via dict_Eq_Color.Eq
  let diffColor = c1 == Blue in
  sameColor && (diffColor == false)  -- Uses sameColor in boolean context
```

**Generated Go code** (incorrect):
```go
var colorTest = func() bool {
    var c1 interface{} = NewColorRed()
    var c2 interface{} = NewColorRed()
    var sameColor interface{} = dict_Eq_Color.Eq(c1, c2)  // Returns interface{}
    var diffColor interface{} = dict_Eq_Color.Eq(c1, c3)
    var tmp7 interface{} = dict_Eq_Bool.Eq(diffColor, false)
    return func() bool {
        if sameColor {  // ERROR: non-boolean condition in if statement
            return tmp7  // ERROR: cannot use interface{} as bool
        }
        return false
    }()
}()
```

**Go compilation errors**:
```
./deriving_eq.go:18:6: non-boolean condition in if statement
./deriving_eq.go:19:11: cannot use tmp7 (variable of type interface{}) as bool value in return statement: need type assertion
```

## Root Cause Analysis

The Go code generator (`internal/gen/golang/`) generates dictionary method calls that return `interface{}`:

```go
// Dictionary methods return interface{} for polymorphism
var dict_Eq_Color = struct {
    Eq  func(interface{}, interface{}) interface{}
    Neq func(interface{}, interface{}) interface{}
}{...}
```

When these results are used in:
1. **If conditions**: `if sameColor { ... }` - Go requires `bool`
2. **Return statements with bool type**: `return tmp7` - Go requires `bool`
3. **Logical operators**: `sameColor && otherBool` - Go requires `bool` operands

The codegen doesn't add the necessary type assertions: `sameColor.(bool)`.

## Affected Code Paths

1. **`codegen_expr_control.go`** - `generateIf()`: If condition expressions
2. **`codegen_decl.go`** - Function return statements
3. **`codegen_ops.go`** - Logical operators (`&&`, `||`)

## Proposed Solution

### Option A: Context-aware type assertions (Recommended)

Add type assertions when an `interface{}` expression is used in a boolean context.

**Key insight**: We can detect boolean context at generation time:
- If conditions always expect `bool`
- Logical operators (`&&`, `||`) always expect `bool` operands
- Return statements in functions with `bool` return type

**Implementation**:

1. In `generateIf()`:
```go
// Before generating condition, check if it's interface{} typed
// and wrap with .(bool) assertion
g.writef("if ")
if g.isInterfaceTyped(e.Cond) {
    g.generateExpr(e.Cond)
    g.write(".(bool)")
} else {
    g.generateExpr(e.Cond)
}
```

2. Add helper method `isInterfaceTyped(expr core.CoreExpr) bool`:
   - Check if expression is a `DictApp` (dictionary method call)
   - Check if expression is a `Var` bound to an `interface{}` value
   - Use `CoreTypeInfo` to check if expression type is polymorphic

3. In logical operator generation (`&&`, `||`):
```go
// Both operands of && and || need bool assertions if interface{}
g.generateExprWithBoolAssertion(e.Left)
g.write(" && ")
g.generateExprWithBoolAssertion(e.Right)
```

### Option B: Always generate bool for Eq/Ord dictionaries

Change dictionary return types from `interface{}` to `bool`:

```go
var dict_Eq_Color = struct {
    Eq  func(interface{}, interface{}) bool  // Return bool directly
    Neq func(interface{}, interface{}) bool
}{...}
```

**Pros**: Simpler, no runtime assertions needed
**Cons**: Breaks polymorphic dictionary abstraction, special-cases Eq/Ord

### Option C: Typed wrapper functions

Generate type-safe wrapper functions:

```go
func eqColor(x, y *Color) bool {
    return dict_Eq_Color.Eq(x, y).(bool)
}
```

**Pros**: Type-safe at call sites
**Cons**: More generated code, function call overhead

## Recommended Approach

**Option A** is recommended because:
1. Maintains the polymorphic dictionary abstraction
2. Minimal code changes (localized to codegen)
3. Assertions are only added where needed
4. Works with existing dictionary infrastructure

## Implementation Plan

### M1: Add boolean context detection (~30 LOC)

**File**: `internal/gen/golang/codegen_expr.go`

Add helper methods:
- `isInterfaceTyped(expr core.CoreExpr) bool`
- `generateExprWithBoolAssertion(expr core.CoreExpr) error`

### M2: Fix If conditions (~10 LOC)

**File**: `internal/gen/golang/codegen_expr_control.go`

Update `generateIf()` to use boolean assertions for conditions.

### M3: Fix logical operators (~15 LOC)

**File**: `internal/gen/golang/codegen_ops.go`

Update `generateBinOp()` for `&&` and `||` operators.

### M4: Fix return statements (~20 LOC)

**File**: `internal/gen/golang/codegen_decl.go`

Update return statement generation when return type is `bool`.

### M5: Add tests (~50 LOC)

**File**: `internal/gen/golang/codegen_bool_test.go`

Test cases:
- Dictionary results in if conditions
- Dictionary results in logical operators
- Dictionary results in bool return statements
- Nested boolean expressions

## Test Cases

```ailang
-- Test 1: Dictionary result in if condition
type Color = Red | Blue deriving (Eq)
let test1 = if Red == Red then 1 else 0  -- Should compile

-- Test 2: Dictionary result in && operator
let test2 = (Red == Red) && (Blue == Blue)  -- Should compile

-- Test 3: Dictionary result in function return
func checkColor(c: Color) -> bool = c == Red  -- Should compile

-- Test 4: Nested boolean expressions
let test4 = if (Red == Red) && true then 1 else 0  -- Should compile
```

## Acceptance Criteria

1. `examples/deriving_eq.ail` compiles to valid Go code
2. Generated Go code compiles without errors
3. All existing codegen tests pass
4. New test cases for boolean assertions pass

## Dependencies

- M-CODEGEN-DICTIONARIES (completed) - Dictionary generation infrastructure
- M-DX19 (completed) - `deriving (Eq)` support in interpreter

## Risk Assessment

**Low risk**: Changes are localized to codegen output formatting, not affecting the compilation pipeline or type system.

## References

- Go spec on boolean expressions: https://go.dev/ref/spec#Boolean_types
- Related issue: Dictionary results used in conditionals without type assertions
- Files to modify:
  - `internal/gen/golang/codegen_expr_control.go`
  - `internal/gen/golang/codegen_ops.go`
  - `internal/gen/golang/codegen_decl.go`
