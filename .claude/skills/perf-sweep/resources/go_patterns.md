# Go-Specific Performance Patterns

Optimization patterns for Go code, especially relevant for AILANG's compiled output.

## Memory Management

### Use sync.Pool for Frequently Allocated Objects

```go
var bufPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 4096)
    },
}

func process() {
    buf := bufPool.Get().([]byte)
    defer bufPool.Put(buf)
    // use buf...
}
```

### Pre-allocate Slices

```go
// BAD: Grows and reallocates
result := []int{}
for _, v := range input {
    result = append(result, transform(v))
}

// GOOD: Single allocation
result := make([]int, 0, len(input))
for _, v := range input {
    result = append(result, transform(v))
}
```

### Avoid Escape to Heap

```go
// BAD: Escapes to heap (pointer returned)
func newBuffer() *Buffer {
    b := Buffer{}  // Allocated on heap
    return &b
}

// GOOD: Stays on stack (caller provides)
func initBuffer(b *Buffer) {
    b.Reset()
}

// Usage:
var b Buffer  // Stack allocated
initBuffer(&b)
```

## String Handling

### Use strings.Builder

```go
// BAD: O(n²) allocations
result := ""
for _, s := range parts {
    result += s
}

// GOOD: O(n) single allocation
var b strings.Builder
b.Grow(estimatedSize)
for _, s := range parts {
    b.WriteString(s)
}
result := b.String()
```

### Avoid Unnecessary Conversions

```go
// BAD: Allocates new string
if string(someBytes) == "expected" {
    // ...
}

// GOOD: Compare bytes directly
if bytes.Equal(someBytes, []byte("expected")) {
    // ...
}
```

## Concurrency

### Use Buffered Channels Appropriately

```go
// BAD: Unbuffered causes blocking
ch := make(chan Result)

// GOOD: Buffer to reduce synchronization
ch := make(chan Result, 100)
```

### Avoid Lock Contention

```go
// BAD: Single lock for all operations
type Cache struct {
    mu    sync.Mutex
    items map[string]Item
}

// GOOD: Sharded locks
type Cache struct {
    shards [256]struct {
        mu    sync.Mutex
        items map[string]Item
    }
}

func (c *Cache) getShard(key string) *shard {
    h := fnv.New32a()
    h.Write([]byte(key))
    return &c.shards[h.Sum32()%256]
}
```

## Struct Layout

### Order Fields by Size

```go
// BAD: Poor alignment (40 bytes with padding)
type Bad struct {
    a bool    // 1 byte + 7 padding
    b int64   // 8 bytes
    c bool    // 1 byte + 7 padding
    d int64   // 8 bytes
    e bool    // 1 byte + 7 padding
}

// GOOD: Optimal alignment (32 bytes)
type Good struct {
    b int64   // 8 bytes
    d int64   // 8 bytes
    a bool    // 1 byte
    c bool    // 1 byte
    e bool    // 1 byte + 5 padding
}
```

### Use Value Receivers for Small Structs

```go
// For structs <= 2 words, value receivers avoid indirection
type Point struct {
    X, Y float64
}

func (p Point) Distance(other Point) float64 {
    dx := p.X - other.X
    dy := p.Y - other.Y
    return math.Sqrt(dx*dx + dy*dy)
}
```

## Interface Optimization

### Avoid Interface{} in Hot Paths

```go
// BAD: Type assertions have cost
func process(items []interface{}) {
    for _, item := range items {
        if s, ok := item.(string); ok {
            handleString(s)
        }
    }
}

// GOOD: Type-specific function
func processStrings(items []string) {
    for _, item := range items {
        handleString(item)
    }
}
```

### Use Concrete Types When Possible

```go
// BAD: Interface adds indirection
func Sum(nums []fmt.Stringer) int { ... }

// GOOD: Concrete type
func Sum(nums []int) int { ... }
```

## Compiler Hints

### Use //go:noinline Sparingly

```go
// Only for debugging/profiling
//go:noinline
func expensiveOperation() { ... }
```

### Bounds Check Elimination

```go
// Help compiler eliminate bounds checks
func process(data []int) {
    if len(data) < 4 {
        return
    }
    _ = data[3]  // Proves bounds to compiler

    // Now these won't have bounds checks
    a := data[0]
    b := data[1]
    c := data[2]
    d := data[3]
}
```

## Profiling Commands

```bash
# CPU profiling
go test -bench=. -cpuprofile=cpu.prof
go tool pprof -http=:8080 cpu.prof

# Memory profiling
go test -bench=. -memprofile=mem.prof
go tool pprof -http=:8080 mem.prof

# Trace
go test -bench=. -trace=trace.out
go tool trace trace.out

# Escape analysis
go build -gcflags='-m -m' 2>&1 | grep 'escapes to heap'

# Assembly output
go build -gcflags='-S' 2>&1 | less
```

## Benchmarking Best Practices

```go
func BenchmarkProcess(b *testing.B) {
    // Setup outside the loop
    data := generateTestData()

    b.ResetTimer()  // Don't count setup
    b.ReportAllocs()  // Track allocations

    for i := 0; i < b.N; i++ {
        result := process(data)
        // Prevent compiler from optimizing away
        _ = result
    }
}
```
