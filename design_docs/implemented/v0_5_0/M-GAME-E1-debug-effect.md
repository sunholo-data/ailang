# M-GAME-E1: Debug Effect (Foundational Trace Substrate)

**Status**: Planned
**Target**: v0.5.1
**Priority**: P1 - High (Contract requirement)
**Estimated**: 4-5 days (~400 LOC)
**Dependencies**: M-GAME-C (Go codegen infrastructure)

## AI-First Alignment Check

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Structured debug output vs printf debugging |
| Preserve Semantic Clarity | + | +2 | Ghost effect - erasable, no semantic pollution |
| Increase Determinism | + | +2 | Host-controlled collection, no in-language branching |
| Lower Token Cost | 0 | 0 | Infrastructure feature |
| **Net Score** | | **+5** | **Decision: High priority** |

## Problem Statement

**Current State:**
- No `Debug` effect in AILANG
- No structured way to collect trace/debug output
- No `--release` mode to strip debug overhead
- No foundational substrate for testing/tracing infrastructure

**Impact:**
- Debugging requires println hacks
- No way to verify invariants
- Debug output mixed with program output
- Release builds still have debug overhead
- Testing infrastructure will need separate tracing (duplication)

## Design Philosophy

**This is NOT "just a game debug feature".**

Debug is designed as AILANG's **foundational tracing substrate** that will be reused by:
- Game engine debugging (tick-based collection)
- Inline test assertions (M-TESTING-INLINE)
- Property test shrinking (trace failing cases)
- CLI tooling (debugging scripts)
- Future browser REPL (structured output)

### Key Design Decisions

1. **Write-only effect from AILANG** - No `collect()` in language; prevents branching on debug state
2. **Ghost effect semantics** - Erasable from effect rows in release mode, not just no-op at runtime
3. **Hidden location arguments** - Source positions wired automatically, not passed by user
4. **Abstract logical time** - Host defines timestamp meaning (tick, test case index, etc.)
5. **Backend-agnostic** - Core semantics work for interpreter, Go, future JS/WASM

## Solution Design

### Effect Definition (Core)

```ailang
-- std/debug.ail
-- Debug is a GHOST EFFECT: erasable in release mode

effect Debug {
    assert(cond: bool, msg: string) -> ()
    log(msg: string) -> ()
    -- NOTE: No collect() in language - host-only operation
}
```

**Key semantic properties:**
- `assert` and `log` are **write-only** - they append to a host-managed buffer
- Source location is **automatically injected** by compiler (hidden argument)
- Effect row `! {Debug}` can be **erased** in release mode
- No way for AILANG code to observe its own debug output

### Output Types (Shared Schema)

```ailang
-- std/debug/types.ail
-- Shared schema for all backends and tools

type DebugOutput = {
    logs: [LogEntry],
    assertions: [AssertionResult]
}

type LogEntry = {
    message: string,
    location: string,      -- "file.ail:42" (auto-injected)
    timestamp: int         -- Logical step (host-defined meaning)
}

type AssertionResult = {
    passed: bool,
    message: string,
    location: string       -- "file.ail:42" (auto-injected)
}
```

**Timestamp semantics:**
- Host defines what timestamp means
- Game engine: frame/tick index
- Test runner: test case index
- CLI: 0 (or monotonic counter)
- AILANG code MUST NOT rely on timestamp meaning

### Usage in AILANG

```ailang
import std/debug (Debug)

-- Debug appears in effect row
func update_entity(e: Entity) -> Entity ! {Debug} {
    -- Location injected automatically by compiler
    Debug.assert(e.health >= 0, "health must be non-negative")
    Debug.assert(e.health <= 100, "health exceeds maximum")
    Debug.log("updating entity " ++ show(e.id))

    let new_health = e.health - damage
    Debug.log("new health: " ++ show(new_health))

    { ...e, health = new_health }
}

-- Multiple effects compose naturally
func step(world: World, input: Input) -> (World, Output) ! {RNG, Debug} {
    -- Debug writes accumulate during execution
    -- Host collects after step returns
    process(world, input)
}
```

### Host Contract (All Backends)

Every backend must implement this contract:

```
DebugContext Interface:
  - Log(msg string, location string)
  - Assert(cond bool, msg string, location string)
  - SetTimestamp(t int64)          -- Host sets logical time
  - Collect() -> DebugOutput       -- Host-only, returns accumulated data
  - Reset()                        -- Clear buffer for next step
```

**Lifecycle (host-controlled):**
1. Host creates/resets DebugContext
2. Host sets timestamp (if relevant)
3. Run AILANG code (may call Debug.log, Debug.assert)
4. Host calls Collect() to get DebugOutput
5. Host decides what to do with output (display, store, discard)

### Go Codegen Output

```go
// debug_types.go (generated)
package game

// DebugOutput matches std/debug/types.ail schema
type DebugOutput struct {
    Logs       []LogEntry
    Assertions []AssertionResult
}

type LogEntry struct {
    Message   string
    Location  string
    Timestamp int64
}

type AssertionResult struct {
    Passed   bool
    Message  string
    Location string
}

// DebugContext is the host-side handler
type DebugContext struct {
    logs       []LogEntry
    assertions []AssertionResult
    timestamp  int64
}

func NewDebugContext() *DebugContext {
    return &DebugContext{}
}

func (d *DebugContext) SetTimestamp(t int64) {
    d.timestamp = t
}

func (d *DebugContext) Log(msg, location string) {
    d.logs = append(d.logs, LogEntry{
        Message:   msg,
        Location:  location,
        Timestamp: d.timestamp,
    })
}

func (d *DebugContext) Assert(cond bool, msg, location string) {
    d.assertions = append(d.assertions, AssertionResult{
        Passed:   cond,
        Message:  msg,
        Location: location,
    })
}

// Collect returns accumulated data (HOST-ONLY - not callable from AILANG)
func (d *DebugContext) Collect() DebugOutput {
    return DebugOutput{
        Logs:       d.logs,
        Assertions: d.assertions,
    }
}

// Reset clears buffer for next step
func (d *DebugContext) Reset() {
    d.logs = d.logs[:0]
    d.assertions = d.assertions[:0]
}
```

### Game Engine Integration Example

```go
// cmd/game/main.go
func main() {
    debugCtx := game.NewDebugContext()
    rngCtx := game.NewRNGContext(42)

    world := game.InitWorld(42, rngCtx)

    for tick := 0; tick < 1000; tick++ {
        // Host controls lifecycle
        debugCtx.Reset()
        debugCtx.SetTimestamp(int64(tick))

        input := captureInput()

        // Run AILANG code - Debug writes accumulate
        world, output, err := game.Step(world, input, debugCtx, rngCtx)
        if err != nil {
            log.Fatal(err)
        }

        // Host collects after step
        debugData := debugCtx.Collect()

        // Host decides what to do with debug output
        if len(debugData.Assertions) > 0 {
            for _, a := range debugData.Assertions {
                if !a.Passed {
                    log.Printf("ASSERTION FAILED at %s: %s", a.Location, a.Message)
                }
            }
        }

        render(output)
    }
}
```

### Ghost Effect Semantics (Release Mode)

**Debug is an erasable ("ghost") effect.**

In release mode, the compiler:
1. **Removes Debug from effect rows** - `! {RNG, Debug}` becomes `! {RNG}`
2. **Rewrites Debug calls to unit** - `Debug.log(msg)` becomes `()`
3. **Validates no dependencies** - Error if code depends on Debug return value

```bash
# Debug mode (default) - Debug in effect rows, calls execute
ailang compile --emit-go world.ail

# Release mode - Debug erased from types AND calls
ailang compile --emit-go --release world.ail
```

**Type-level erasure matters because:**
- "Pure except for Debug" functions become truly pure in release
- Effect-polymorphic code that forbids effects can accept Debug-only code
- Optimizer can reason about purity correctly

**Go implementation:**
```go
//go:build !release

func (d *DebugContext) Log(msg, location string) {
    d.logs = append(d.logs, LogEntry{...})
}
```

```go
//go:build release

func (d *DebugContext) Log(msg, location string) {
    // Erased - zero cost
}
```

### Location Injection (Hidden Arguments)

AILANG code does NOT pass location strings - they're injected by the compiler.

**Surface syntax:**
```ailang
Debug.log("message")
Debug.assert(x > 0, "x must be positive")
```

**After elaboration (internal representation):**
```
Debug.log("message", __LOC__("module.ail", 42, 5))
Debug.assert(x > 0, "x must be positive", __LOC__("module.ail", 43, 5))
```

**Implementation:**
- Elaborator attaches SourcePos to Debug effect calls
- Codegen converts to "file.ail:line" string
- User never sees or passes location

### Implementation Plan

**Milestone E1.1: Effect System (1.5 days)**
- [ ] Define `Debug` effect in `internal/effects/debug.go`
- [ ] Mark Debug as ghost effect (erasable)
- [ ] Add `DebugOutput`, `LogEntry`, `AssertionResult` types to `std/debug/types.ail`
- [ ] Register builtins: `_debug_assert`, `_debug_log` (no collect!)
- [ ] Implement location injection in elaborator
- [ ] Implement interpreter handler with DebugContext

**Milestone E1.2: Go Codegen (1.5 days)**
- [ ] Generate `debug_types.go` with shared schema
- [ ] Generate `DebugContext` handler
- [ ] Thread `DebugContext` through generated functions
- [ ] Generate location strings from AST positions at call sites

**Milestone E1.3: Release Mode & Erasure (1 day)**
- [ ] Add `--release` flag to compile command
- [ ] Implement Debug erasure pass (remove from effect rows)
- [ ] Rewrite Debug calls to unit in release mode
- [ ] Generate build-tagged Go files
- [ ] Test debug vs release output

**Milestone E1.4: Documentation & Testing Alignment (0.5 days)**
- [ ] Update go-interop.md with host contract
- [ ] Add examples to sim_stub
- [ ] Document JSON schema for DebugOutput (for future tooling)
- [ ] Plan alignment with M-TESTING-INLINE (shared trace format)

### Files to Modify/Create

**New files:**
- `internal/effects/debug.go` - Ghost effect definition (~80 LOC)
- `internal/pipeline/debug_erasure.go` - Release mode erasure pass (~100 LOC)
- `internal/gen/golang/debug.go` - Go codegen for debug (~150 LOC)
- `std/debug.ail` - Effect definition (~10 LOC)
- `std/debug/types.ail` - Shared output types (~25 LOC)

**Modified files:**
- `internal/builtins/spec.go` - Register debug builtins
- `internal/elaborate/elaborate.go` - Location injection
- `cmd/ailang/compile.go` - Add `--release` flag
- `examples/sim_stub/world.ail` - Add debug usage example
- `examples/sim_stub/main.go` - Add DebugContext usage

### Testing Strategy

**Unit tests:**
- Assert collection (passed and failed)
- Log collection with auto-injected locations
- DebugContext lifecycle (reset, collect)
- Empty context returns empty output

**Integration tests:**
- Debug effect in interpreter
- Go codegen produces valid code
- Release mode erases Debug from effect rows
- Release mode no-ops Debug calls
- sim_stub example works with debug

**Cross-backend tests:**
- Same DebugOutput schema from interpreter and Go
- JSON serialization matches expected format

### Success Criteria

- [ ] `Debug.assert` and `Debug.log` work (write-only)
- [ ] No `collect()` callable from AILANG
- [ ] Locations auto-injected by compiler
- [ ] Debug is ghost effect (erasable in release)
- [ ] `--release` mode erases from effect rows AND calls
- [ ] Works in interpreter
- [ ] Generates valid Go code
- [ ] Host contract documented
- [ ] JSON schema documented for tooling
- [ ] Consumer contract updated to ✅

## Future Enhancements (Planned Alignment)

### Testing Infrastructure Reuse
- Inline test assertions emit to same DebugOutput
- Property test shrinking attaches DebugOutput to failing cases
- Shared JSON format for all observability

### Extended Trace Data
- Log levels (trace, debug, info, warn, error)
- Log categories/tags for filtering
- Structured metadata: `Debug.logWithMeta(msg, {entity: id, phase: "update"})`

### Backend Extensions
- Browser REPL: DebugOutput → structured console
- WASM: DebugOutput → host callback
- Future tracing integration (OpenTelemetry spans)

## Security Considerations

- Debug data may contain sensitive information
- Host controls what happens with DebugOutput
- Release mode ensures no debug overhead in production
- No way for AILANG code to exfiltrate data via Debug

---

**Document created**: 2025-12-02
**Last updated**: 2025-12-02 (incorporated architectural feedback)

## Appendix: Why No collect() in Language?

**The Footgun:**
```ailang
-- BAD: Branching on debug state
func weird(x: int) -> int ! {Debug} {
    Debug.log("checking x")
    let logs = Debug.collect()  -- If this existed...
    if length(logs.logs) > 10 then
        0  -- "Too much logging, bail out"
    else
        compute(x)
}
```

This creates:
1. **Non-determinism** - Behavior depends on how many logs were emitted
2. **Optimization barriers** - Can't inline/optimize Debug calls
3. **Semantic pollution** - "Pure except for Debug" is no longer pure

**The Solution:**
- Debug is write-only from AILANG perspective
- Only host can observe debug output
- AILANG code cannot branch on its own trace
