# M-R4: Recursion Support

**Status**: 📋 Planned
**Priority**: P0 (CRITICAL - MUST SHIP)
**Estimated**: 600 LOC (400 impl + 200 tests)
**Duration**: 3 days
**Dependencies**: None
**Blocks**: Real-world programs (factorial, fibonacci, quicksort, tree traversal)

## Problem Statement

**Current State**: Functions cannot call themselves. Recursive patterns fail with "undefined variable" errors.

```ailang
-- ❌ BROKEN in v0.2.0
export func factorial(n: int) -> int {
  if n <= 1 then 1 else n * factorial(n - 1)  -- ERROR: undefined variable 'factorial'
}
```

**Root Cause**:
- LetRec expressions parse and type-check correctly
- Runtime evaluator doesn't create self-referential closures
- Function bindings not available in their own scope during evaluation

**Impact**: Blocks fundamental programming patterns:
- Recursive algorithms (factorial, fibonacci, gcd)
- List processing (map, filter, fold implementations)
- Tree traversal (AST walking, search algorithms)
- Mutual recursion (isEven/isOdd, parser combinators)

## Goals

### Primary Goals (Must Achieve)
1. **Self-recursion works**: Functions can call themselves
2. **Mutual recursion works**: Multiple functions can reference each other
3. **Stack safety**: Graceful error on stack overflow (not panic)
4. **Examples pass**: factorial, fibonacci, quicksort examples work

### Secondary Goals (Nice to Have)
5. Tail-call optimization (deferred to v0.4.0)
6. Recursion depth limits (deferred to v0.4.0)

## Design

### Core Approach: Indirection Cells (RefCell) with Function-First Semantics

**Key Insight**: Use mutable indirection cells and treat function bindings specially for safe, predictable recursion (OCaml/Haskell style).

**Why RefCell over nil/ThunkValue?**
- Prevents nil lookup failures
- Gives precise diagnostic errors
- Makes cycles explicit
- Keeps semantics strict and predictable (no implicit laziness)

**Algorithm (3 Phases):**

**Phase 1: Pre-allocate Indirection Cells**
```go
recEnv := env.NewChildEnvironment()
cells := make(map[string]*RefCell, len(bindings))

for _, binding := range bindings {
    cell := &RefCell{}  // Uninitialized cell
    cells[binding.Name] = cell
    recEnv.Set(binding.Name, &IndirectValue{Cell: cell})
}
```

**Phase 2: Evaluate RHS (Function-First)**
```go
oldEnv := e.env
e.env = recEnv
defer func() { e.env = oldEnv }()

for _, binding := range bindings {
    // Lambda RHS: Build closure immediately (safe, body executes later)
    if lam, ok := isLambda(binding.Value); ok {
        fv := &FunctionValue{
            Params: lam.Params,
            Body:   lam.Body,
            Env:    recEnv,  // Captures recursive environment
        }
        cells[binding.Name].Val = fv
        cells[binding.Name].Init = true
        continue
    }

    // Non-lambda RHS: Strict evaluation (may error if reads self)
    cells[binding.Name].Visiting = true
    val, err := e.evalCore(binding.Value)
    cells[binding.Name].Visiting = false
    if err != nil { return nil, err }

    cells[binding.Name].Val = val
    cells[binding.Name].Init = true
}
```

**Phase 3: Evaluate Body**
```go
return e.evalCore(letrec.Body)  // Under recursive env
```

**Cycle Detection** (at IndirectValue.Force()):
```go
func (iv *IndirectValue) Force() (Value, error) {
    if !iv.Cell.Init {
        if iv.Cell.Visiting {
            return nil, newRecursiveValueError()  // "recursive value used before init"
        }
        return nil, newUninitializedRecError()  // Internal bug
    }
    return iv.Cell.Val, nil
}
```

### Mutual Recursion

**Challenge**: Multiple functions referencing each other.

```ailang
letrec
  isEven = \n. if n == 0 then true else isOdd(n - 1),
  isOdd = \n. if n == 0 then false else isEven(n - 1)
in
  isEven(42)
```

**Solution**: All names pre-bound with IndirectValue cells before ANY RHS evaluation.
- All names visible to all function bodies (via captured `recEnv`)
- Order of evaluation doesn't matter (deterministic)
- Works naturally with RefCell approach (no special handling needed)

### Data Structures

**RefCell** - Mutable indirection for recursion:
```go
type RefCell struct {
    Val      Value  // The actual value (once initialized)
    Init     bool   // Has the value been set?
    Visiting bool   // Currently being evaluated? (cycle detection)
}
```

**IndirectValue** - Defers to cell at read-time:
```go
type IndirectValue struct {
    Cell *RefCell
}

func (iv *IndirectValue) Force() (Value, error) {
    if !iv.Cell.Init {
        if iv.Cell.Visiting {
            return nil, newRecursiveValueError()  // Cycle detected
        }
        return nil, newUninitializedRecError()  // Internal bug
    }
    return iv.Cell.Val, nil
}
```

### Stack Overflow Protection

**Problem**: Infinite recursion crashes interpreter with stack overflow.

**Solution**: Track recursion depth in function application, fail gracefully.

```go
type CoreEvaluator struct {
    recursionDepth    int
    maxRecursionDepth int  // Default: 10,000
}

func (e *CoreEvaluator) applyFunction(fn Value, args []Value) (Value, error) {
    e.recursionDepth++
    if e.recursionDepth > e.maxRecursionDepth {
        e.recursionDepth--
        return nil, newStackOverflowError(e.maxRecursionDepth)
    }
    defer func() { e.recursionDepth-- }()

    // ... actual function application
}
```

**Error Message**:
```
RT_REC_003: max recursion depth 10,000 exceeded
  Try smaller input, enable tail recursion, or increase with --max-recursion-depth
```

**CLI Flag**: `--max-recursion-depth=N` (default 10,000)

## Implementation Plan

### Day 1: Core LetRec Implementation (~250 LOC)

**Files to Modify**:
- `internal/eval/eval_core.go` (~150 LOC)
- `internal/eval/value.go` (~50 LOC)
- `internal/eval/env.go` (~50 LOC)

**Tasks**:
1. Add `PlaceholderValue` type for pre-binding
2. Implement `evalLetRec()` with pre-bind → evaluate → backpatch flow
3. Ensure closure environments capture self-references
4. Unit tests: simple recursion (factorial, fibonacci)

**Test Cases**:
```go
// internal/eval/recursion_test.go
func TestSimpleRecursion(t *testing.T) {
    tests := []struct {
        name string
        expr string
        want Value
    }{
        {"factorial_5", "letrec fac = \\n. if n <= 1 then 1 else n * fac(n-1) in fac(5)", IntValue(120)},
        {"fib_10", "letrec fib = \\n. if n <= 1 then n else fib(n-1) + fib(n-2) in fib(10)", IntValue(55)},
    }
    // ...
}
```

### Day 2: Mutual Recursion (~200 LOC)

**Files to Modify**:
- `internal/eval/eval_core.go` (~100 LOC)
- `internal/runtime/resolver.go` (~50 LOC)
- `internal/eval/recursion_test.go` (~50 LOC)

**Tasks**:
1. Extend `evalLetRec()` to handle multiple bindings
2. Pre-bind all names before evaluating any bodies
3. Add stack depth tracking with configurable limit
4. Unit tests: mutual recursion (isEven/isOdd)

**Test Cases**:
```go
func TestMutualRecursion(t *testing.T) {
    expr := `
        letrec
          isEven = \n. if n == 0 then true else isOdd(n - 1),
          isOdd = \n. if n == 0 then false else isEven(n - 1)
        in
          isEven(42)
    `
    got, err := evalString(expr)
    assert.NoError(t, err)
    assert.Equal(t, BoolValue(true), got)
}
```

### Day 3: Examples & Polish (~150 LOC)

**New Example Files** (`examples/`):
1. `recursive_factorial.ail` (~20 LOC)
2. `recursive_fibonacci.ail` (~20 LOC)
3. `recursive_quicksort.ail` (~40 LOC)
4. `recursive_mutual.ail` (~20 LOC)

**Example Content**:
```ailang
// examples/recursive_factorial.ail
module examples/recursive_factorial

export func factorial(n: int) -> int {
  if n <= 1 then 1 else n * factorial(n - 1)
}

export func main() -> int {
  factorial(5)  -- Returns: 120
}
```

```ailang
// examples/recursive_quicksort.ail
module examples/recursive_quicksort

import std/list (filter, concat)

export func quicksort(xs: [int]) -> [int] {
  match xs {
    [] => [],
    [pivot, ...rest] => {
      let smaller = filter(\x. x < pivot, rest);
      let larger = filter(\x. x >= pivot, rest);
      concat([quicksort(smaller), [pivot], quicksort(larger)])
    }
  }
}

export func main() -> [int] {
  quicksort([3, 1, 4, 1, 5, 9, 2, 6])
}
```

**Error Messages** (`internal/eval/errors.go`):
```go
func newRecursiveValueError() error {
    return fmt.Errorf("RT_REC_001: recursive value used before initialization (non-function RHS). Consider making it a function or introducing laziness")
}

func newUninitializedRecError() error {
    return fmt.Errorf("RT_REC_002: uninitialized recursive binding; this indicates an internal ordering bug")
}

func newStackOverflowError(maxDepth int) error {
    return fmt.Errorf("RT_REC_003: max recursion depth %d exceeded. Try smaller input, enable tail recursion, or increase with --max-recursion-depth", maxDepth)
}
```

**Error Taxonomy**:
- `RT_REC_001` - Recursive value used before initialization (user error)
- `RT_REC_002` - Uninitialized binding (internal bug)
- `RT_REC_003` - Stack overflow (too deep, may need TCO)

## Acceptance Criteria

### Functional Requirements
- [ ] `factorial(5)` returns 120
- [ ] `fibonacci(10)` returns 55
- [ ] `quicksort([3,1,4,1,5,9])` returns sorted list
- [ ] Mutual recursion (isEven/isOdd) works correctly
- [ ] Stack overflow gives friendly error (not panic)

### Code Quality
- [ ] 100% test coverage for recursion paths
- [ ] No regressions in existing tests
- [ ] Clean error messages with suggestions
- [ ] Examples documented and passing

### Performance
- [ ] Factorial(100) completes in <10ms
- [ ] Fibonacci(20) completes in <100ms (no memoization, exponential is expected)
- [ ] Stack depth tracking adds <5% overhead

## Risks & Mitigations

| Risk | Severity | Likelihood | Mitigation |
|------|----------|------------|------------|
| **Closure environment bugs** | High | Medium | Extensive unit testing, start with simple cases |
| **Performance overhead** | Medium | Low | Benchmark before/after, optimize if >5% regression |
| **Stack overflow handling** | Low | Low | Use deferred cleanup, never panic |
| **Mutual recursion edge cases** | Medium | Medium | Test all permutations of binding order |

## Testing Strategy

### Unit Tests (~200 LOC)
- `internal/eval/recursion_test.go`
  - Simple self-recursion (factorial, fibonacci, gcd)
  - Mutual recursion (isEven/isOdd, parser combinators)
  - Stack overflow (depth limit exceeded)
  - Edge cases (empty LetRec, single binding, shadowing)

### Integration Tests
- `examples/recursive_factorial.ail` - Basic recursion
- `examples/recursive_fibonacci.ail` - Exponential recursion
- `examples/recursive_quicksort.ail` - Real algorithm
- `examples/recursive_mutual.ail` - Mutual recursion

### Negative Tests
- Infinite recursion triggers graceful error
- Missing base case detected
- Large recursion depth (10k+ calls) handled

## Success Metrics

| Metric | Target |
|--------|--------|
| **Examples fixed** | +4 (factorial, fib, quicksort, mutual) |
| **Test coverage** | 100% for recursion paths |
| **Performance** | <5% overhead for non-recursive code |
| **Error quality** | Clear stack overflow messages |

## Future Work (Deferred)

**v0.4.0 - Tail Call Optimization**:
- Detect tail-recursive patterns
- Transform to iterative loops
- Enable unbounded recursion for tail calls

**v0.4.0 - Recursion Depth Limits**:
- CLI flag: `--max-recursion-depth=N`
- Per-module limits
- Budget tracking for cloud functions

**v0.5.0 - Trampolining**:
- Convert deep recursion to iterative bouncing
- Enable stack-safe recursion
- Preserve semantics for mutual recursion

## Edge Cases & Gotchas

### 1. **Mutual Recursion Across Modules**
- ❌ Out of scope for v0.3.0
- Recursion is per-LetRec group within a module
- Cross-module cycles still error at loader/link time (existing behavior)

### 2. **Effects Inside Recursive Functions**
- ✅ Works fine; recursion semantics are independent
- Depth guard still applies
- Example: `letrec loop = λn. println(n); loop(n+1) in loop(0)`

### 3. **Pattern Matching Guards**
- ✅ No interaction needed
- Body evaluation occurs under recursive env
- Guards evaluate normally

### 4. **Tail Recursion**
- ❌ No TCO yet (deferred to v0.3.1)
- Depth guard + helpful error suffice for v0.3.0
- Future: Detect tail position, convert to loop

### 5. **Non-Function Recursive Values**
- ❌ Errors with RT_REC_001 (by design)
- `letrec x = x in x` → "recursive value used before initialization"
- Future: Allow with explicit `lazy` annotation

## References

- **Design Doc**: `design_docs/planned/v0_3_0_implementation_plan.md`
- **Related Issues**: Recursion blocker (multiple user reports)
- **Prior Art**: OCaml let rec (function-first), Haskell recursive bindings, Scheme letrec*
