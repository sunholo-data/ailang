# M-DX24: Typed Function Bodies

## Status: PLANNED

## Problem

M-DX23 successfully implemented typed function signatures in Go codegen. Functions now generate with concrete Go types:

```go
// M-DX23 achievement - typed signature!
func Add(a int64, b int64) int64 {
    return AddInt(a, b)  // BUG: AddInt returns interface{}
}
```

However, the function **body** still uses `interface{}`-based runtime helpers, causing compilation errors:

```
./funcs.go:388:9: cannot use AddInt(a, b) (value of type interface{}) as int64
                  value in return statement: need type assertion
```

## Root Cause Analysis

The Go codegen has two separate concerns:

1. **Function signatures** (M-DX23 - COMPLETE): `func Add(a int64, b int64) int64`
2. **Function bodies** (this milestone): `return a + b` vs `return AddInt(a, b)`

Currently, `codegen_expr.go` generates expression code without considering the expected type context. It always emits calls to generic helpers like `AddInt`, `SubInt`, etc.

## Proposed Solution

### Phase 1: Type Assertion Generation (Quick Fix)

When generating a return statement and we know the function's return type, wrap `interface{}` expressions with type assertions:

```go
// Before (breaks)
func Add(a int64, b int64) int64 {
    return AddInt(a, b)
}

// After (works)
func Add(a int64, b int64) int64 {
    return AddInt(a, b).(int64)
}
```

**Implementation:**
1. Track expected return type in `Generator` during function generation
2. In `generateExpr`, when generating a return value:
   - If expected type is concrete (int64, string, bool, etc.)
   - And generated expression returns `interface{}`
   - Emit `.(expectedType)` assertion

**Files to modify:**
- `internal/gen/golang/codegen_decl.go`: Pass return type context to body generation
- `internal/gen/golang/codegen_expr.go`: Add type assertion wrapping

### Phase 2: Direct Typed Operations (Optimization)

For maximum performance, skip runtime helpers entirely when types are known:

```go
// Phase 1 (type assertion)
func Add(a int64, b int64) int64 {
    return AddInt(a, b).(int64)
}

// Phase 2 (direct operations)
func Add(a int64, b int64) int64 {
    return a + b
}
```

**Implementation:**
1. In `generateBinaryOp`, check if both operands have known types from CoreTypeInfo
2. If both are `int64`, emit `a + b` instead of `AddInt(a, b)`
3. If both are `float64`, emit `a + b` instead of `AddFloat(a, b)`
4. If both are `string` and op is `++`, emit `a + b` instead of `ConcatString(a, b)`
5. Fall back to runtime helpers for unknown/polymorphic types

**Files to modify:**
- `internal/gen/golang/codegen_ops.go`: Type-aware binary operation generation

### Phase 3: Typed Local Variables (Full Type Propagation)

Propagate types through the entire function body:

```go
// Ideal output
func Calculate(x int64, y int64) int64 {
    temp := x * 2    // Go infers: temp is int64
    return temp + y  // Direct int64 addition
}
```

**Implementation:**
1. Maintain a `localTypeEnv` during function generation
2. Track variable types from:
   - Parameter types (from signature)
   - Let bindings (from CoreTypeInfo)
   - Lambda bindings (from CoreTypeInfo)
3. Use typed variable declarations: `var temp int64 = ...`

## Test Cases

```ailang
-- Test 1: Basic arithmetic (Phase 1+2)
export pure func add(a: int, b: int) -> int { a + b }

-- Test 2: Comparison (Phase 1+2)
export pure func isPositive(x: int) -> bool { x > 0 }

-- Test 3: String concat (Phase 1+2)
export pure func greet(name: string) -> string { "Hello, " ++ name }

-- Test 4: Mixed operations (Phase 2)
export pure func calc(a: int, b: int) -> int { (a + b) * 2 }

-- Test 5: Local binding (Phase 3)
export pure func withLocal(x: int) -> int {
  let doubled = x * 2 in
  doubled + 1
}

-- Test 6: Polymorphic (should keep interface{})
export pure func identity(x) { x }
```

Expected generated code after all phases:

```go
func Add(a int64, b int64) int64 {
    return a + b
}

func IsPositive(x int64) bool {
    return x > 0
}

func Greet(name string) string {
    return "Hello, " + name
}

func Calc(a int64, b int64) int64 {
    return (a + b) * 2
}

func WithLocal(x int64) int64 {
    doubled := x * 2
    return doubled + 1
}

func Identity(x interface{}) interface{} {
    return x  // Polymorphic, keeps interface{}
}
```

## Dependencies

- M-DX23: Typed Function Signatures (COMPLETE)
- CoreTypeInfo population (WORKING - verified)

## Acceptance Criteria

### Phase 1 (Minimum Viable)
- [ ] Generated code compiles without manual modification
- [ ] Type assertions added where needed
- [ ] No performance regression vs current output

### Phase 2 (Performance)
- [ ] Direct Go operations for primitive types
- [ ] Reduced runtime helper calls
- [ ] Benchmark shows improvement

### Phase 3 (Full)
- [ ] Local variables are typed
- [ ] Let bindings preserve types
- [ ] Generated code reads like idiomatic Go

## Estimated Effort

- Phase 1: 2-4 hours (quick fix)
- Phase 2: 4-8 hours (operator rewrite)
- Phase 3: 8-16 hours (full type propagation)

## Notes

The current architecture supports this work well:
- CoreTypeInfo already maps all expressions to types
- `getTypedSignature` already extracts function types
- TypeMapper already converts AILANG types to Go types

The main work is threading type context through expression generation and conditionally emitting typed code vs generic code.
