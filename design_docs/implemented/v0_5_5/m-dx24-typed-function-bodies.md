# M-DX24: Typed Function Bodies

## Status: SUPERSEDED by M-DX26

**Note**: This approach was superseded by M-DX26 (Typed Wrapper Architecture), which solved
the problem differently using a dual-function pattern (_impl + typed wrapper) instead of
trying to generate typed code in function bodies directly. The wrapper pattern proved
simpler and more maintainable.

## Strategic Context

This is the final piece of the **Go backend correctness triangle**:

| Feature | Status |
|---------|--------|
| Typed function signatures | DONE (M-DX23) |
| Typed function bodies | THIS milestone |
| Typed local variables / ADT lowering | NEXT milestone (M-DX25+) |

The 3-phase structure aligns with mainstream compiler architecture patterns used in:
- Haskell: Core → STG → CMM
- Elm: JavaScript backend
- Swift: SIL → IRGen → LLVM
- OCaml: flambda backend

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

## Architectural Traps to Avoid

### Trap 1: Type Assertions in Middle of Expressions

Naively wrapping type assertions can produce invalid code.

```ailang
(a + b) * c
```

**WRONG** - assertion on composite expression:
```go
(AddInt(a, b) * c).(int64)  // Invalid Go!
```

**CORRECT** - assertion at leaf node:
```go
(AddInt(a, b).(int64) * c)  // Valid Go
```

**Rule:** Type assertions can only appear at:
- Return statements
- Variable initialization
- Argument positions
- **Never** on composite expressions

### Trap 2: Polymorphic vs Monomorphic Context Mismatch

```ailang
func f(x: a, y: int) -> int { y }
```

Go signature:
```go
func F(x interface{}, y int64) int64
```

**Rule:** Only assert when CoreTypeInfo gives a **concrete monotype**.
Never assert when the type contains a `TVar`.

### Trap 3: ADTs with Parametric Components

```ailang
type Box[a] = Box(a)
```

With return type `Box[int]`:
- Constructor returns `interface{}`
- Struct field is concrete `int64`

```go
return NewBox(x).(BoxInt)  // Must be supported
```

**Rule:** Constructor codegen must be type-aware before Phase 3.
(See M-DX25 for typed ADT lowering.)

## Proposed Solution

### Phase 1: Type Assertion Generation (Minimal Correctness Fix)

**Key insight:** Instead of wrapping expressions inside `codegen_expr.go`, apply the **expression boundary rule** - wrap only at top-level nodes:
- Return values
- Assigned variables
- Function arguments
- Struct fields
- List elements

**Implementation:**

1. Add `expectedType GoType` field to `generateExpr` context:
   ```go
   expr := g.generateExpr(node, expectedType)
   ```

2. Expression codegen decides:
   - If `expectedType` is concrete
   - And node produces `interface{}`
   - Emit `(expr).(expectedType)`

**Files to modify:**
- `internal/gen/golang/codegen_decl.go`: Pass return type context to body generation
- `internal/gen/golang/codegen_expr.go`: Add type assertion wrapping with boundary rule

**Example transformation:**

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

### Phase 2: Direct Typed Operations (Performance)

For maximum performance, skip runtime helpers entirely when types are known.

**Operator mapping:**

| AILANG op | Go op (int) | Go op (float) | Go op (string) |
|-----------|-------------|---------------|----------------|
| `+` | `+` | `+` | - |
| `-` | `-` | `-` | - |
| `*` | `*` | `*` | - |
| `/` | `/` | `/` | - |
| `%` | `%` | - | - |
| `++` | - | - | `+` |
| `>` `<` `>=` `<=` `==` `!=` | same | same | same |

**Implementation:**

1. In `generateBinaryOp`, check if both operands have known types from CoreTypeInfo
2. If both are `int64`, emit `a + b` instead of `AddInt(a, b)`
3. If both are `float64`, emit `a + b` instead of `AddFloat(a, b)`
4. If both are `string` and op is `++`, emit `a + b` instead of `ConcatString(a, b)`
5. **Critical:** Mixed-type operations must fallback to helpers or compile-time error
6. Fall back to runtime helpers for unknown/polymorphic types

**Files to modify:**
- `internal/gen/golang/codegen_ops.go`: Type-aware binary operation generation

**Example transformation:**

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

**Performance impact:** 10-40x speedup in tight loops. Critical for game engines.

### Phase 3: Typed Local Variables (Full Type Propagation)

This is where most compilers fail. Requires careful design.

**Implementation:**

1. Maintain a mapping during function generation:
   ```go
   localTypeEnv map[CoreVar]GoType
   ```

2. Populate from:
   - Parameters (from signature)
   - `core.Let` bindings (from CoreTypeInfo)
   - Lambda arguments (from CoreTypeInfo)
   - Match bindings (from pattern destructuring)

3. Generate typed declarations:
   ```go
   var doubled int64 = x * 2
   ```

   Or for complex initializers:
   ```go
   var doubled int64
   doubled = x * 2
   ```

4. In match expressions:
   ```go
   switch v := scrutinee.(type) {
   case *ADT_Foo:
       a := v.A.(int64)   // Phase 1 assertion
       // Later removed by typed ADT lowering (M-DX25)
   }
   ```

**Files to modify:**
- `internal/gen/golang/codegen_expr.go`: Local type environment tracking
- `internal/gen/golang/codegen_match.go`: Typed pattern binding generation

**Example transformation:**

```go
// Ideal output
func Calculate(x int64, y int64) int64 {
    temp := x * 2    // Go infers: temp is int64
    return temp + y  // Direct int64 addition
}
```

## Test Cases

### Basic Tests (Original)

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

### Edge Case Tests (Critical)

```ailang
-- Test 7: Higher-order function with typed return
export pure func apply(f: int -> int, x: int) -> int { f(x) }
-- Must generate: func Apply(f func(int64) int64, x int64) int64 { return f(x) }

-- Test 8: Pattern match assigns local variables
type Point = Point(int, int)
export pure func sumPoint(p: Point) -> int {
  match p {
    Point(x, y) -> x + y
  }
}
-- x and y must be typed int64, constructor field extraction must be typed

-- Test 9: Let binding with tuple destructuring
export pure func sumPair(pair: (int, int)) -> int {
  let (a, b) = pair in a + b
}
-- Must generate valid Go (Go doesn't support native tuple destructure)
-- Generates: temp := pair; a := temp.A; b := temp.B; return a + b

-- Test 10: Return type interface{} but locals typed
export pure func wrap(x: int) { x }
-- Return type unknown → must NOT assert on return

-- Test 11: Mixed-type binary operations must forbid direct emission
-- let y = x + 3.2  -- Should force fallback to helpers or compile-time error
```

### Expected Generated Code (All Phases Complete)

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

func Apply(f func(int64) int64, x int64) int64 {
    return f(x)
}

func SumPoint(p *Point) int64 {
    x := p.Value0
    y := p.Value1
    return x + y
}

func SumPair(pair *Tuple2[int64, int64]) int64 {
    a := pair.A
    b := pair.B
    return a + b
}

func Wrap(x int64) interface{} {
    return x  // No assertion - return type is interface{}
}
```

## Dependencies

- M-DX23: Typed Function Signatures (COMPLETE)
- CoreTypeInfo population (WORKING - verified)

## Acceptance Criteria

### Phase 1 (Minimum Viable) - CRITICAL

- [ ] Generated code compiles without manual modification
- [ ] Type assertions added only at expression boundaries (return, assign, args)
- [ ] No assertions when type contains TVar (polymorphic)
- [ ] No assertions on composite expressions (only leaves)
- [ ] No runtime panics due to invalid assertions

### Phase 2 (Performance) - HIGH IMPACT

- [ ] Direct Go operations for primitive types (int64, float64, string, bool)
- [ ] Reduced runtime helper calls
- [ ] Mixed-type operations rejected or use fallback
- [ ] Benchmark shows >= 3x speedup on numeric-heavy loops

### Phase 3 (Full) - DEVELOPER EXPERIENCE

- [ ] Local variables are typed (no `interface{}` in locals)
- [ ] Let bindings preserve types
- [ ] Match bindings preserve types
- [ ] Generated code reads like idiomatic Go
- [ ] Near-zero allocation pressure in generated code

### Strict Completion Criteria

M-DX24 is complete ONLY when:

1. Generated Go code for any **pure, monomorphic** AILANG function contains:
   - No `interface{}` in locals
   - No generic helpers for arithmetic or equality
   - No runtime panics due to invalid assertions
   - Only direct Go operators

2. Generated ADT accessors contain:
   - No casting except for constructors
   - No `interface{}` except in polymorphic ADTs

3. Benchmarks show:
   - >= 3x speedup on numeric-heavy simulated tick loops
   - Near-zero allocation pressure in generated code

## Estimated Effort

| Phase | Effort | Importance | Notes |
|-------|--------|------------|-------|
| Phase 1 | 2-4 hours | **Critical** | Must ship ASAP - fixes broken codegen |
| Phase 2 | 4-8 hours | **Huge Impact** | Main perf win for Go backend |
| Phase 3 | 8-16 hours | Developer Experience | The "Go-native" milestone |

## Notes

The current architecture supports this work well:
- CoreTypeInfo already maps all expressions to types
- `getTypedSignature` already extracts function types
- TypeMapper already converts AILANG types to Go types

The main work is threading type context through expression generation and conditionally emitting typed code vs generic code.

## Future Work

- **M-DX25: Typed ADT Lowering** - Make constructor codegen type-aware, eliminate casting in ADT accessors
- **Go Backend Architecture Doc** - Document the complete Go backend strategy
