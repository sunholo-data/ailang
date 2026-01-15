# M-EMBED: Generic AILANG Embedding API

**Status**: Implemented (v1)
**Target**: v0.6.4
**Priority**: P1 (Medium)
**Estimated**: 4 hours (actual: 3 hours)
**Dependencies**: Module runtime (v0.2.0), Pipeline (v0.3.0)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Embedded modules are pure; same inputs = same outputs |
| A2: Replayability | +1 | All conversions are reversible; traces can include Go↔AILANG boundaries |
| A3: Effect Legibility | +1 | Embedded modules must declare effects; no hidden side effects |
| A4: Explicit Authority | +1 | Capabilities must be passed explicitly at call site |
| A5: Bounded Verification | 0 | No change to type checking |
| A6: Safe Concurrency | 0 | Single-threaded embedding; mutex-protected engine |
| A7: Machines First | +1 | JSON-based interface enables easy AI agent integration |
| A8: Minimal Syntax | +1 | No new AILANG syntax; pure Go API |
| A9: Cost Visibility | 0 | Resource costs unchanged |
| A10: Composability | +1 | Works with existing module system |
| A11: Structured Failure | +1 | Errors return as Go errors; no panics |
| A12: System Boundary | +1 | Explicit boundary via FromGo/ToGo conversion |

**Net Score: +9** → **Decision: Move forward (implemented)**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects - modules declare all effects
- [x] A4 (Authority): No ambient access granted - capabilities explicit
- [x] A7 (Machines First): API designed for programmatic use

## Problem Statement

Go applications wanting to use AILANG for data transformation, configuration, or business logic have no supported way to embed the AILANG runtime.

**Current State:**
- Must shell out to `ailang run` subprocess
- No way to call AILANG functions from Go code
- No type-safe value conversion between Go and AILANG
- Module caching not available to external callers

**Impact:**
- Dashboard transforms written in Go instead of AILANG (can't dogfood)
- External projects can't use AILANG as extension language
- Forces duplicating logic in both Go and AILANG

## Goals

**Primary Goal:** Enable Go applications to embed AILANG runtime and call AILANG functions directly.

**Success Metrics:**
- Go programs can load AILANG modules in <100ms
- Type-safe conversion for all primitive types + collections
- JSON round-trip preserves data integrity
- Module caching provides O(1) subsequent calls

## Solution Design

### Overview

Create `internal/embed/` package with:
1. `Engine` - Manages module loading and caching
2. `FromGo` / `ToGo` - Type conversion functions
3. Type-safe extractors - `ToInt`, `ToString`, `ToBool`, etc.
4. JSON helpers - `FromJSON`, `ToJSON`

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Go Application                          │
├─────────────────────────────────────────────────────────────┤
│  engine := embed.New(basePath)                              │
│  result, _ := engine.Call("module", "func", arg1, arg2)     │
│  value, _ := embed.ToInt(result)                            │
├─────────────────────────────────────────────────────────────┤
│                   internal/embed/                           │
│  ┌─────────────┐  ┌─────────────┐  ┌────────────────────┐  │
│  │   Engine    │  │  FromGo     │  │   Type Extractors  │  │
│  │  - Load     │  │  ToGo       │  │  ToInt, ToString   │  │
│  │  - Call     │  │  FromJSON   │  │  ToBool, ToBytes   │  │
│  │  - Eval     │  │  ToJSON     │  │  ToList, ToRecord  │  │
│  └──────┬──────┘  └──────┬──────┘  └────────────────────┘  │
│         │                │                                   │
├─────────┴────────────────┴───────────────────────────────────┤
│              internal/runtime/ModuleRuntime                  │
│              internal/pipeline/RunWithContext                │
│              internal/eval/Value                             │
└─────────────────────────────────────────────────────────────┘
```

**Components:**

1. **Engine**: Wraps ModuleRuntime with mutex protection and caching
2. **Converters**: Reflection-based Go↔AILANG value translation
3. **Extractors**: Type-safe helpers to unwrap AILANG values

### Implementation Plan

**Phase 1: Core Engine** (~2 hours) ✅
- [x] Create Engine struct with mutex and runtime
- [x] Implement Load, Call, CallJSON methods
- [x] Implement Eval for expressions
- [x] Add ListExports, HasExport for introspection

**Phase 2: Value Conversion** (~1.5 hours) ✅
- [x] FromGo: nil, bool, int*, uint*, float*, string, []byte
- [x] FromGo: slices, arrays, maps, structs
- [x] ToGo: All eval.Value types to Go
- [x] Handle pointers and interfaces

**Phase 3: Testing** (~0.5 hours) ✅
- [x] Unit tests for FromGo/ToGo
- [x] Round-trip tests (Go→AILANG→Go)
- [x] JSON conversion tests
- [ ] Engine.Call integration tests (requires module fixtures)

### Files Created

**New files:**
- `internal/embed/embed.go` - Engine API (~220 LOC)
- `internal/embed/convert.go` - Value conversion (~330 LOC)
- `internal/embed/embed_test.go` - Test suite (~270 LOC)

## Examples

### Example 1: Simple Expression Evaluation

```go
package main

import "github.com/sunholo/ailang/internal/embed"

func main() {
    engine := embed.New(".")
    defer engine.Close()

    // Evaluate simple expression
    result, _ := engine.Eval("1 + 2 * 3")
    value, _ := embed.ToInt(result)
    fmt.Println(value) // 7
}
```

### Example 2: Calling Module Function

```go
// AILANG module: transforms/event_formatter.ail
// export pure func truncate(text: string, maxLen: int) -> string

engine := embed.New("/path/to/project")
defer engine.Close()

result, err := engine.Call(
    "transforms/event_formatter",
    "truncate",
    "Hello, World! This is a long string.",
    10,
)
if err != nil {
    log.Fatal(err)
}

truncated, _ := embed.ToString(result)
fmt.Println(truncated) // "Hello, Wor..."
```

### Example 3: JSON API

```go
// For language-agnostic integrations
inputJSON := []byte(`{"events": [...], "maxLen": 200}`)
outputJSON, err := engine.CallJSON(
    "transforms/event_formatter",
    "formatEvents",
    inputJSON,
)
// outputJSON is ready for HTTP response
```

### Example 4: Value Conversion

```go
// Go struct → AILANG Record
type Event struct {
    TurnNum    int    `json:"turnNum"`
    StreamType string `json:"streamType"`
    Text       string `json:"text"`
}

event := Event{TurnNum: 1, StreamType: "text", Text: "Hello"}
ailangVal, _ := embed.FromGo(event)

// AILANG Record → Go map
goVal, _ := embed.ToGo(ailangVal)
m := goVal.(map[string]interface{})
// m["turnNum"] == 1
```

## Success Criteria

- [x] Engine.New() creates runtime without error
- [x] FromGo handles all primitive types
- [x] ToGo handles all eval.Value types
- [x] Round-trip conversion preserves data
- [x] JSON conversion works correctly
- [x] All conversion tests passing (14 tests)
- [ ] Module Call tests passing (blocked by pipeline issue)
- [ ] Documentation in docs/guides/

## Testing Strategy

**Unit tests:**
- FromGo for each supported type
- ToGo for each Value variant
- Round-trip preservation tests
- Error cases (unsupported types)

**Integration tests:**
- Load and call event_formatter.ail
- Compare AILANG vs Go implementation outputs

**Manual testing:**
- Build and run dashboard with embedded transforms

## Non-Goals

**Not in this feature:**
- Async/concurrent calls - Use separate engines (mutex inside)
- Effect handling - Embedded modules should be pure
- Hot reloading - Restart engine to reload modules
- Performance optimization - Caching is sufficient for v1

## Timeline

**Session 1** (3 hours): ✅
- Phase 1: Core Engine
- Phase 2: Value Conversion
- Phase 3: Basic tests

**Session 2** (1 hour): In Progress
- Fix Eval tests (pipeline empty program issue)
- Integration tests with event_formatter.ail
- Documentation

**Total: ~4 hours across 2 sessions**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Pipeline changes break embedding | Medium | Pin to specific pipeline interface |
| Module resolution differences | Low | Use same ModuleRuntime as CLI |
| Memory leaks from cached modules | Low | Engine.Close() clears runtime |

## Related Documents

**Implemented (informing design):**
- [Module Runtime (v0.2.0)](../../../design_docs/implemented/v0_2_0/) - Underlying module loading
- [Pipeline](../../../internal/pipeline/) - Compilation pipeline

**Planned (related work):**
- [Dashboard AILANG Transforms](../../../internal/dashboard_transforms/) - First dogfooding use case

## Gaps Discovered During Implementation

The embedding work revealed several AILANG language gaps:

| Gap | Severity | Status |
|-----|----------|--------|
| GAP-1: Teaching prompt wrong about foldl lambda | High | Fixed in v0.6.5 prompt |
| GAP-2: Path-dependent type checking | Critical | Needs investigation |
| GAP-3: Lambda syntax unreliable with foldl | Medium | Workaround: use inline func |
| GAP-4: No record width subtyping | High | Future: row polymorphism |

See [GAPS_DISCOVERED.md](../../../internal/dashboard_transforms/GAPS_DISCOVERED.md) for details.

## Future Work

- `CallWithCaps` - Allow specifying capabilities for effectful functions
- Module hot-reloading - Watch for changes and recompile
- Parallel execution - Pool of engines for concurrent calls
- WASM embedding - Compile AILANG to WASM for browser use
- Python bindings - Expose embed API via cgo

---

**Document created**: 2026-01-15
**Last updated**: 2026-01-15
