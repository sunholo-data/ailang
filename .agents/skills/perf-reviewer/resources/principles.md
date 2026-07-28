# Performance Optimization Principles

Comprehensive guide inspired by Abseil's performance tips, adapted for AILANG/Go development.

## Core Philosophy

**Balance simplicity with performance.** Knuth's "premature optimization" quote acknowledges that while 97% of code shouldn't be micro-optimized, the remaining 3% matters significantly. Write naturally performant code during development.

## Principle 1: Profile Before Optimizing

**Never optimize without data.** Use profiling tools to identify actual bottlenecks.

```bash
# Go profiling
go test -bench=. -cpuprofile=cpu.prof
go tool pprof cpu.prof

# AILANG phase timing
ailang check --debug-compile file.ail
```

**Key insight:** Optimization efforts should target proven hot paths. Many small 1% improvements can collectively yield 20%+ gains.

## Principle 2: Algorithmic Improvements First

**O(n) beats optimized O(n²).** Always seek better algorithms before micro-optimizing.

| Bad | Better |
|-----|--------|
| Nested loop search O(n²) | Hash table lookup O(1) |
| Bubble sort O(n²) | Quick/merge sort O(n log n) |
| String concatenation in loop | StringBuilder/buffer |
| Repeated linear search | Build index first |

```go
// BAD: O(n²) - searching list for each item
for _, item := range items {
    if contains(otherList, item) { // O(n) each time
        process(item)
    }
}

// GOOD: O(n) - build set first
otherSet := make(map[string]bool, len(otherList))
for _, item := range otherList {
    otherSet[item] = true
}
for _, item := range items {
    if otherSet[item] { // O(1) lookup
        process(item)
    }
}
```

## Principle 3: Batch Operations

**Amortize overhead by processing multiple items together.**

```go
// BAD: Lock per item
for _, item := range items {
    mutex.Lock()
    process(item)
    mutex.Unlock()
}

// GOOD: Lock once for batch
mutex.Lock()
for _, item := range items {
    process(item)
}
mutex.Unlock()
```

**Apply to:**
- Database operations (batch inserts)
- Network requests (HTTP/2 multiplexing)
- File I/O (buffered writes)
- API calls (bulk endpoints)

## Principle 4: Memory Layout Matters

**Cache-friendly data structures reduce memory bandwidth.**

**Latency hierarchy:**
| Operation | Time |
|-----------|------|
| L1 cache hit | 0.5 ns |
| L2 cache hit | 7 ns |
| Main memory | 100 ns |
| SSD read | 150 μs |
| Disk seek | 10 ms |

**Guidelines:**
- Colocate frequently-accessed fields
- Use arrays over linked lists when possible
- Prefer flat structures over pointer-heavy trees
- Consider struct-of-arrays vs array-of-structs

```go
// BAD: Pointer-heavy (cache unfriendly)
type Node struct {
    Value int
    Left  *Node
    Right *Node
}

// BETTER for iteration: Flat array
type FlatTree struct {
    Values []int
    Left   []int  // indices
    Right  []int  // indices
}
```

## Principle 5: Avoid Unnecessary Allocations

**Each allocation has overhead: allocator cost, initialization, new cache line.**

```go
// BAD: Allocation per iteration
for i := 0; i < n; i++ {
    buf := make([]byte, 1024)  // Allocates each time
    process(buf)
}

// GOOD: Reuse allocation
buf := make([]byte, 1024)
for i := 0; i < n; i++ {
    process(buf)
    clear(buf)  // Reset for reuse
}
```

**Techniques:**
- `sync.Pool` for frequently allocated objects
- Pre-allocate slices with known capacity: `make([]T, 0, expectedSize)`
- Use value types for small structs (avoid pointer indirection)
- String builders instead of concatenation

## Principle 6: Fast Paths for Common Cases

**Optimize the typical path; handle edge cases separately.**

```go
// BAD: Always check expensive condition
func process(items []Item) {
    for _, item := range items {
        if expensiveValidation(item) {  // Called every time
            doWork(item)
        }
    }
}

// GOOD: Fast path for common case
func process(items []Item) {
    for _, item := range items {
        if item.IsPreValidated {  // Fast check
            doWork(item)
            continue
        }
        if expensiveValidation(item) {  // Rare path
            doWork(item)
        }
    }
}
```

## Principle 7: Precompute Expensive Information

**Calculate once, use many times.**

```go
// BAD: Recompute every call
func isExpensive(config Config) bool {
    return len(config.Features) > 100 &&
           computeComplexity(config) > threshold
}

// GOOD: Precompute at construction
type Config struct {
    Features   []Feature
    isExpensive bool  // Set once during init
}

func NewConfig(features []Feature) *Config {
    c := &Config{Features: features}
    c.isExpensive = len(features) > 100 &&
                    computeComplexity(c) > threshold
    return c
}
```

## Principle 8: Defer Expensive Work

**Move costly operations outside hot loops; lazy evaluation.**

```go
// BAD: Format string every iteration (even if not used)
for _, item := range items {
    log.Debug(fmt.Sprintf("Processing %v", item))  // Always formats
    if shouldProcess(item) {
        process(item)
    }
}

// GOOD: Defer formatting
for _, item := range items {
    if shouldProcess(item) {
        log.Debug("Processing %v", item)  // Only formats when needed
        process(item)
    }
}
```

## Principle 9: Right-Size Data Structures

**Choose containers appropriate to access patterns.**

| Use Case | Best Choice |
|----------|-------------|
| Small fixed set (<10) | Array/slice |
| Frequent lookups | map |
| Ordered iteration | sorted slice |
| Set membership | map[T]struct{} |
| Bit flags | uint64 or bitset |

```go
// BAD: Map for small known set
colors := map[string]int{"red": 0, "green": 1, "blue": 2}

// GOOD: Array lookup for small enum
const (
    Red = iota
    Green
    Blue
)
// Direct index access: O(1) with no hash overhead
```

## Principle 10: Help the Compiler

**Structure code to enable optimizations.**

```go
// BAD: Compiler can't prove slice bounds
func sum(data []int) int {
    total := 0
    for i := 0; i < len(data); i++ {
        total += data[i]  // Bounds check each iteration
    }
    return total
}

// GOOD: Range loop - compiler eliminates bounds checks
func sum(data []int) int {
    total := 0
    for _, v := range data {
        total += v  // No bounds check needed
    }
    return total
}
```

## Principle 11: Estimation via Back-of-Envelope

**Quantify before building.** Quick estimates prevent bad designs.

**Quick reference:**
- CPU cycle: ~0.3 ns
- Function call: ~1-5 ns
- Memory allocation: ~25 ns
- Mutex lock/unlock: ~25 ns
- System call: ~1000 ns
- SSD random read: ~150,000 ns
- Network round-trip (local): ~500,000 ns

**Example calculation:**
```
Processing 1M items with 1 allocation each:
1,000,000 × 25 ns = 25 ms just for allocations

With pooling (reuse 99%):
10,000 × 25 ns = 0.25 ms
Savings: 24.75 ms
```

## AILANG-Specific Considerations

### Interpreted vs Compiled

| Mode | Use When |
|------|----------|
| Interpreted | Prototyping, small scripts, REPL |
| Compiled Go | Production, performance-critical |

**Expected speedups (compiled vs interpreted):**
- Simple arithmetic: 50-100x
- Data transformation: 20-50x
- I/O bound: 1-2x (I/O dominates)

### Type System Overhead

- Monomorphization adds compile time but improves runtime
- Complex type inference can slow compilation
- Explicit type annotations help compiler

### Effect System

- Effect handlers have minimal runtime overhead
- Debug effect is zero-cost in release builds
- Capability checking is compile-time only

## Anti-Patterns to Avoid

1. **String formatting in hot loops** - Use structured logging
2. **Reflection in hot paths** - Generate code instead
3. **Interface{} everywhere** - Lose type info and optimizations
4. **Unbounded caching** - Memory leaks
5. **Premature abstraction** - Adds indirection cost
6. **Ignoring allocations** - GC pressure adds latency spikes
