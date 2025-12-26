# M-CODEGEN-BOOL-ASSERTIONS: Type assertions for boolean contexts in Go codegen

**Status**: IMPLEMENTED
**Priority**: High (blocks `deriving (Eq)` in Go codegen)
**Actual LOC**: ~100
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

**Generated Go code** (BEFORE fix - incorrect):
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

**Generated Go code** (AFTER fix - correct):
```go
var colorTest = func() bool {
    var c1 interface{} = NewColorRed()
    var c2 interface{} = NewColorRed()
    var sameColor interface{} = dict_Eq_Color.Eq(c1, c2)
    var diffColor interface{} = dict_Eq_Color.Eq(c1, c3)
    var tmp7 interface{} = dict_Eq_Bool.Eq(diffColor, false)
    return func() bool {
        if sameColor.(bool) {  // FIXED: Added .(bool) assertion
            return tmp7.(bool)  // FIXED: Added .(bool) assertion
        }
        return false
    }()
}()
```

## Implementation

### M1: Boolean Context Detection Helpers (codegen_expr.go)

Added two helper functions:

1. `needsBoolAssertion(expr core.CoreExpr) bool` - Detects expressions that need `.(bool)`:
   - `DictApp` with comparison methods (eq, neq, lt, gt, lte, gte)
   - `Var` not in typedLocalVars and not a typed function parameter
   - Conservative: returns true for interface{}-typed variables

2. `generateExprWithBoolAssertion(expr core.CoreExpr) error` - Wrapper that:
   - Generates the expression
   - Appends `.(bool)` if `needsBoolAssertion()` returns true

### M2: Fix If Conditions (codegen_expr_control.go)

Updated `generateIf()` and `generateIfChain()` to use `generateExprWithBoolAssertion()` for if condition expressions.

### M3: Fix Logical Operators (codegen_ops.go)

Added special case in `generateBinOp()` for `&&` and `||` operators:
- Both operands are generated with `generateExprWithBoolAssertion()`
- Ensures Go receives `bool` operands, not `interface{}`

### M4: Fix Return Statements (codegen_expr_control.go)

Updated return statement type assertion checks to also call `needsBoolAssertion()`:
- Original check: `exprProducesInterface(expr)`
- New check: `exprProducesInterface(expr) || needsBoolAssertion(expr)`

### M5: Tests (codegen_bool_test.go)

Added comprehensive test suite:
- `TestNeedsBoolAssertion`: Tests detection of various expression types
- `TestGenerateExprWithBoolAssertion`: Tests assertion wrapper
- `TestLogicalOperatorsBoolAssertion`: Tests && and || handling
- `TestLogicalOperatorsNoAssertionForLiterals`: Tests that literals don't get assertions

## Files Modified

- `internal/gen/golang/codegen_expr.go` - Added helper functions
- `internal/gen/golang/codegen_expr_control.go` - Fixed if conditions and return statements
- `internal/gen/golang/codegen_ops.go` - Fixed logical operators
- `internal/gen/golang/codegen_bool_test.go` - Added tests (NEW)

## Acceptance Criteria

1. ✅ `examples/deriving_eq.ail` compiles to valid Go code
2. ✅ Generated Go code compiles without errors
3. ✅ All existing codegen tests pass
4. ✅ New test cases for boolean assertions pass

## Dependencies

- M-CODEGEN-DICTIONARIES (completed) - Dictionary generation infrastructure
- M-DX19 (completed) - `deriving (Eq)` support in interpreter
