# M-STD-MATH-TRIG: Trigonometric Functions for std/math

**Status**: Planned
**Target**: v0.5.10
**Priority**: P1 (Blocking external project - stapledons_voyage)
**Estimated**: 0.5 days
**Dependencies**: None
**Reporter**: stapledons_voyage via agent messaging (GitHub Issue #19)

---

## TL;DR

**Problem**: AILANG has no trigonometric functions. Projects requiring angle calculations must fall back to Go, violating language boundaries.

**Solution**: Add sin, cos, atan2, sqrt, and PI constant to std/math module.

**Risk**: Minimal - pure wrappers around Go's math package.

---

## Background

### Use Case

stapledons_voyage is building a game that requires positioning elements around an arc (dome struts with parallax effect). This requires trigonometric functions:

```ailang
import std/math (sin, cos, PI)

pure func makeArcStrut(index: int, total: int, centerX: float, centerY: float, radius: float) -> Position {
    let angle = intToFloat(index) / intToFloat(total) * PI;
    let x = centerX + radius * cos(angle);
    let y = centerY - radius * sin(angle);
    { x = x, y = y }
}
```

### Current State

`std/math` (v0.5.9) has 12 builtins - all basic arithmetic:

| Category | Functions |
|----------|-----------|
| Integer | add_Int, sub_Int, mul_Int, div_Int, mod_Int, neg_Int |
| Float | add_Float, sub_Float, mul_Float, div_Float, mod_Float, neg_Float |

**Missing**: Any mathematical functions beyond basic arithmetic.

### Impact

- **Blocking**: stapledons_voyage cannot implement arc positioning in AILANG
- **Workaround**: None - must use Go, violating project architecture
- **Users affected**: Any project requiring angle/distance calculations

---

## Requirements

### Must Have (P0)

| Function | Signature | Description |
|----------|-----------|-------------|
| `sin` | `float -> float` | Sine (radians) |
| `cos` | `float -> float` | Cosine (radians) |
| `sqrt` | `float -> float` | Square root |
| `PI` | `float` constant | Mathematical constant π |

### Should Have (P1)

| Function | Signature | Description |
|----------|-----------|-------------|
| `atan2` | `(float, float) -> float` | Two-argument arctangent |
| `abs_Float` | `float -> float` | Absolute value (float) |
| `abs_Int` | `int -> int` | Absolute value (int) |

### Nice to Have (P2 - Future)

| Function | Signature | Description |
|----------|-----------|-------------|
| `tan` | `float -> float` | Tangent |
| `asin` | `float -> float` | Arcsine |
| `acos` | `float -> float` | Arccosine |
| `atan` | `float -> float` | Arctangent (single arg) |
| `pow` | `(float, float) -> float` | Power |
| `exp` | `float -> float` | Exponential |
| `log` | `float -> float` | Natural logarithm |
| `floor` | `float -> float` | Floor |
| `ceil` | `float -> float` | Ceiling |
| `round` | `float -> float` | Round to nearest |
| `E` | `float` constant | Euler's number e |

---

## Solution Design

### Implementation Location

Add to `internal/builtins/math.go` following existing patterns.

### Registration Pattern

```go
func init() {
    registerArithmetic()
    registerComparisons()
    registerLogic()
    registerConversions()
    registerTrigonometry()  // NEW
    registerConstants()      // NEW
}
```

### Trigonometry Implementation

```go
func registerTrigonometry() {
    // sin: float -> float
    registerMathFunc("sin", 1, math.Sin,
        "Compute sine of angle in radians",
        []string{"math", "trigonometry", "sin"})

    // cos: float -> float
    registerMathFunc("cos", 1, math.Cos,
        "Compute cosine of angle in radians",
        []string{"math", "trigonometry", "cos"})

    // sqrt: float -> float
    registerMathFunc("sqrt", 1, math.Sqrt,
        "Compute square root",
        []string{"math", "sqrt", "root"})

    // atan2: (float, float) -> float
    registerMathFunc2("atan2", math.Atan2,
        "Compute two-argument arctangent (y, x)",
        []string{"math", "trigonometry", "atan2", "angle"})

    // abs_Float: float -> float
    registerMathFunc("abs_Float", 1, math.Abs,
        "Compute absolute value of float",
        []string{"math", "abs", "absolute"})

    // abs_Int: int -> int
    registerBuiltinWithMeta("abs_Int", 1, true, intToInt(func(a int) int {
        if a < 0 {
            return -a
        }
        return a
    }), "Compute absolute value of integer", []string{"math", "abs", "absolute"})
}

// Helper for unary float functions
func registerMathFunc(name string, arity int, fn func(float64) float64, desc string, tags []string) {
    impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
        a := args[0].(*eval.FloatValue)
        return &eval.FloatValue{Value: fn(a.Value)}, nil
    }
    typeFunc := func() types.Type {
        T := types.NewBuilder()
        return T.Func(T.Float()).Returns(T.Float()).Build()
    }
    err := RegisterEffectBuiltin(BuiltinSpec{
        Module:  "std/math",
        Name:    name,
        NumArgs: arity,
        IsPure:  true,
        Type:    typeFunc,
        Impl:    impl,
        Metadata: &BuiltinMetadata{
            Description: desc,
            Params:      []ParamDoc{{Name: "x", Description: "Input value (radians for trig functions)"}},
            Returns:     "Computed result",
            Since:       "v0.5.10",
            Stability:   StabilityStable,
            Tags:        tags,
            Category:    "math",
        },
    })
    if err != nil {
        panic(fmt.Sprintf("failed to register %s: %v", name, err))
    }
}

// Helper for binary float functions
func registerMathFunc2(name string, fn func(float64, float64) float64, desc string, tags []string) {
    impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
        a := args[0].(*eval.FloatValue)
        b := args[1].(*eval.FloatValue)
        return &eval.FloatValue{Value: fn(a.Value, b.Value)}, nil
    }
    typeFunc := func() types.Type {
        T := types.NewBuilder()
        return T.Func(T.Float(), T.Float()).Returns(T.Float()).Build()
    }
    err := RegisterEffectBuiltin(BuiltinSpec{
        Module:  "std/math",
        Name:    name,
        NumArgs: 2,
        IsPure:  true,
        Type:    typeFunc,
        Impl:    impl,
        Metadata: &BuiltinMetadata{
            Description: desc,
            Params: []ParamDoc{
                {Name: "y", Description: "Y coordinate"},
                {Name: "x", Description: "X coordinate"},
            },
            Returns:     "Angle in radians",
            Since:       "v0.5.10",
            Stability:   StabilityStable,
            Tags:        tags,
            Category:    "math",
        },
    })
    if err != nil {
        panic(fmt.Sprintf("failed to register %s: %v", name, err))
    }
}
```

### Constants Implementation

Constants require special handling since they're not functions.

**Option A: Zero-arity Functions (Recommended)**

```go
func registerConstants() {
    // PI as nullary function (consistent with other constants)
    impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
        return &eval.FloatValue{Value: math.Pi}, nil
    }
    typeFunc := func() types.Type {
        T := types.NewBuilder()
        return T.Float()  // Just returns float, no function type
    }
    // Note: Constants will need special handling in the module system
    // For now, PI can be a 0-arg function: PI() -> float
    // Or we defer constants to v0.6.0 with proper module constant support
}
```

**Option B: Workaround via Module Definition**

Create `std/math.ail` that exposes PI:

```ailang
module std/math

-- Mathematical constant PI
let PI: float = 3.14159265358979323846
```

**Decision**: Use Option A (0-arg function) for consistency. User calls `PI()` to get the value. True constants can be added in v0.6.0 when module constants are supported.

---

## Test Plan

### Unit Tests

Add to `internal/builtins/math_test.go`:

```go
func TestSin(t *testing.T) {
    tests := []struct {
        input    float64
        expected float64
    }{
        {0.0, 0.0},
        {math.Pi / 2, 1.0},
        {math.Pi, 0.0},  // within tolerance
        {-math.Pi / 2, -1.0},
    }
    for _, tc := range tests {
        result := callBuiltin("sin", tc.input)
        if math.Abs(result-tc.expected) > 1e-10 {
            t.Errorf("sin(%v) = %v, expected %v", tc.input, result, tc.expected)
        }
    }
}

func TestCos(t *testing.T) { /* similar */ }
func TestSqrt(t *testing.T) { /* similar */ }
func TestAtan2(t *testing.T) { /* similar */ }
func TestAbs(t *testing.T) { /* similar */ }
```

### Integration Tests

Add to `examples/`:

```ailang
-- examples/runnable/math_trig.ail
module examples/runnable/math_trig

import std/io (println)
import std/math (sin, cos, sqrt, atan2, PI)

func main() ! IO =
    let angle = PI() / 4.0;
    let s = sin(angle);
    let c = cos(angle);
    let _ = println("sin(PI/4) = " ++ show(s));
    let _ = println("cos(PI/4) = " ++ show(c));
    let _ = println("sqrt(2) = " ++ show(sqrt(2.0)));
    let dist = sqrt(3.0 * 3.0 + 4.0 * 4.0);
    let _ = println("distance(3,4) = " ++ show(dist));
    let angle2 = atan2(1.0, 1.0);
    println("atan2(1,1) = " ++ show(angle2))
```

### Validation on stapledons_voyage

After implementation, verify:

1. `import std/math (sin, cos)` resolves
2. Arc positioning code compiles
3. Struts render at correct positions

---

## Implementation Plan

### Phase 1: Core Trig Functions (2 hours)

1. Add `registerTrigonometry()` to `internal/builtins/math.go`
2. Implement: sin, cos, sqrt
3. Add unit tests
4. Run `make test && make lint`

### Phase 2: Extended Functions (1 hour)

1. Add: atan2, abs_Float, abs_Int
2. Add PI as 0-arg function
3. Add unit tests
4. Validate with `ailang doctor builtins`

### Phase 3: Integration & Validation (1 hour)

1. Create example file `examples/runnable/math_trig.ail`
2. Run `make verify-examples`
3. Test with stapledons_voyage
4. Update documentation

---

## Success Criteria

### Hard Line

1. **sin, cos, sqrt work** - basic trig functions available
2. **PI accessible** - as function or constant
3. **stapledons_voyage unblocked** - arc positioning works in AILANG
4. **All tests pass** - no regressions

### Stretch Goals

1. All P1 functions (atan2, abs) implemented
2. Teaching prompt updated with math examples
3. Website docs updated

---

## Non-Goals

- Degree variants (sinDeg, cosDeg) - users can convert
- Complex numbers - separate proposal needed
- High-precision arithmetic - standard IEEE 754 is sufficient
- Matrix operations - separate math/linear module

---

## Alternatives Considered

### 1. Pure AILANG Implementation

Could implement using Taylor series:

```ailang
pure func sin(x: float) -> float =
    let x2 = x * x;
    x - (x * x2 / 6.0) + (x * x2 * x2 / 120.0) - ...
```

**Rejected**: Less accurate, much slower than native Go functions.

### 2. FFI to Go math Package

Expose Go's math package directly via FFI.

**Rejected**: AILANG doesn't have general FFI yet. Builtins are the correct approach.

### 3. External Library

Use a third-party math library.

**Rejected**: Unnecessary dependency. Go's math package is comprehensive and well-tested.

---

## References

- [GitHub Issue #19](https://github.com/sunholo-data/ailang/issues/19) - Original feature request
- [Go math package](https://pkg.go.dev/math) - Implementation source
- [internal/builtins/math.go](../../../internal/builtins/math.go) - Existing implementation
- [stapledons_voyage agent messages](../../) - Use case details

## Changelog

| Date | Change |
|------|--------|
| 2025-12-10 | Initial design document from GitHub Issue #19 |
