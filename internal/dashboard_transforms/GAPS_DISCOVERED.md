# AILANG Dogfooding: Gaps Discovered

This document tracks gaps discovered while porting `internal/coordinator/event_formatter.go` to AILANG.

**Last Updated:** 2026-01-15

---

## All Major Gaps Fixed! 🎉

### GAP-1: Teaching Prompt Wrong About foldl Lambda Syntax ✅ FIXED

**Status:** Fixed in prompts/v0.6.5.md

The teaching prompt incorrectly showed tuple syntax `\(acc, x).` instead of curried `\acc x.`.

---

### GAP-2: Multi-Param Lambda Type Inference Bug ✅ FIXED

**Status:** Fixed in v0.6.3

**Original issue:** Multi-param lambda type inference failed when the module's final expression was a bare identifier.

```ailang
-- NOW WORKS:
let sum = foldl(\acc x. acc + x, 0, [1,2,3])
sum  -- ✅ Compiles correctly
```

---

### GAP-3: Lambda Syntax with foldl ✅ FIXED

**Status:** Fixed (was dependent on GAP-2)

```ailang
-- NOW WORKS:
let max = foldl(\acc x. if x > acc then x else acc, 0, xs)
```

---

### GAP-4: No Record Width Subtyping ✅ FIXED (v0.6.4)

**Status:** Fixed with row polymorphism syntax!

**New syntax:** Use `{field: T | r}` or `{field: T, ...}` for open records:

```ailang
-- EXACT record (default): only accepts {name: string}
pure func getNameExact(p: {name: string}) -> string = p.name

-- OPEN record with | r: accepts extra fields
pure func getName(p: {name: string | r}) -> string = p.name

-- OPEN record with ... sugar (equivalent to | r)
pure func getEmail(u: {email: string, ...}) -> string = u.email

-- Usage: all these work!
getName({name: "Alice"})                          -- ✅
getName({name: "Bob", age: 30})                   -- ✅
getName({name: "Charlie", age: 25, city: "NYC"}) -- ✅
```

**Example file:** `examples/runnable/record_width_subtyping.ail`

---

### Missing `repeat` in stdlib ✅ FIXED

**Status:** Added to std/string in v0.6.3

```ailang
import std/string (repeat)
let dashes = repeat("-", 50)
```

---

### Missing `maximum` in stdlib ✅ FIXED

**Status:** Added `maximumInt`, `minimumInt`, etc. to std/list in v0.6.3

```ailang
import std/list (maximumInt)
let maxVal = maximumInt([1, 5, 3, 9, 2])  -- 9
```

---

## Remaining Low-Priority Issue

### GAP-5: Pipeline Cannot Evaluate Standalone Expressions

**Severity:** Low (edge case for embed API)

The `Engine.Eval("1 + 2")` method doesn't work for standalone expressions without module context.

**Workaround:** Use `Engine.Call()` to call module functions:
```go
result, err := engine.Call("my/module", "myFunction", arg1, arg2)
```

---

## Summary

| Gap | Status |
|-----|--------|
| GAP-1: Teaching prompt wrong | ✅ FIXED |
| GAP-2: Multi-param lambda bug | ✅ FIXED |
| GAP-3: Lambda syntax with foldl | ✅ FIXED |
| GAP-4: No record subtyping | ✅ FIXED (row polymorphism) |
| GAP-5: Standalone expression eval | ⚠️ Low priority |
| Missing `repeat` | ✅ FIXED |
| Missing `maximum` | ✅ FIXED |

---

## Dogfooding Outcome

**6 of 7 gaps fixed!** (GAP-5 is a minor edge case)

The event_formatter.ail module compiles and can be used via the embed API:

```go
engine := embed.New(".")
result, _ := engine.Call(
    "internal/dashboard_transforms/event_formatter",
    "summarizeEvents",
    events,
)
```

**Dogfooding was successful!** AILANG is now usable for real dashboard data transformation with:
- Concise lambda syntax with foldl
- Row polymorphism for flexible record types
- Complete stdlib for string/list operations

---

## Performance Benchmarks (v0.6.6)

**Go vs AILANG comparison** (Apple M2):

| Operation | Go | AILANG | Slowdown | AILANG Allocs |
|-----------|-----|--------|----------|---------------|
| truncate (1 call) | 25 ns | 26 µs | ~1,000x | 153 |
| countTurns (10 events) | 4 ns | 215 µs | ~50,000x | 1,475 |
| countTurns (100 events) | 31 ns | 2.1 ms | ~68,000x | 14,268 |
| summarize (10 events) | 162 ns | 426 µs | ~2,600x | 3,371 |
| summarize (100 events) | 407 ns | 4.1 ms | ~10,000x | 30,605 |

**Performance Analysis:**
- **Function call overhead:** ~26µs base cost per call (even for trivial operations)
- **Scaling:** Linear with data size (~10x data → ~10x time)
- **Allocation overhead:** ~150 allocations per call + ~140 per event

**Root Causes:**
1. Interpreted execution (no JIT or compilation to native code)
2. Value boxing (every primitive wrapped in interface{})
3. Reflection-based Go↔AILANG conversion

**Optimization Attempts:**
- RWMutex for read-heavy paths: ~100ns improvement (minimal)
- Further optimization requires compiler work (JIT, AOT)

**Dashboard Use Case Assessment:**
- ✅ **4ms latency acceptable** for human-refreshed data (100 events)
- ✅ **Good for <100 events** per call, <10 calls/second
- ⚠️ **Not suitable** for hot paths or high-frequency calls
- ⚠️ **Memory pressure** may matter for very large event lists

**Live Dashboard Integration:**
```bash
# Enable AILANG for dashboard transforms
AILANG_DASHBOARD=1 AILANG_PROJECT_ROOT=/path/to/ailang ailang serve
```

The dashboard API returns `"ailang_active": true` when AILANG is processing events.

**Benchmark file:** `internal/dashboard_transforms/benchmark_test.go`

Run benchmarks:
```bash
go test -bench=. -benchmem ./internal/dashboard_transforms/...
```
