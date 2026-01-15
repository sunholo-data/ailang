# M-DASHBOARD-DOGFOODING: AILANG Dashboard Integration

**Status:** Implemented
**Version:** v0.6.6
**Completed:** 2026-01-15
**Duration:** ~6 hours (across multiple sessions)

## Summary

Used AILANG for dashboard data transformation to discover language gaps, validate the embed API, and measure real-world performance. Successfully integrated AILANG into the Collaboration Hub dashboard with feature flag control.

## Problem Statement

AILANG needed real-world dogfooding beyond synthetic benchmarks to:
1. Discover missing stdlib functions and syntax pain points
2. Validate the embed API for Go↔AILANG integration
3. Measure performance characteristics in a production-like context
4. Prove AILANG is usable for practical data transformation

## Solution Implemented

### Architecture: Embedded Engine (Option A)

```
HTTP Request → Go Handler → AILANG Bridge → AILANG Module → Go Response
```

The Go server embeds the AILANG runtime and calls modules for specific transformations, controlled by feature flag `AILANG_DASHBOARD=1`.

### Components Built

#### 1. Embed API Fixes (`internal/embed/embed.go`)

**Problem:** Original embed API bypassed OpLowering pass, causing runtime errors.

**Fix:** Added `compileModule()` that runs pipeline with `ModeCheck` before loading:

```go
func (e *Engine) compileModule(modulePath string) error {
    cfg := pipeline.Config{
        Mode:         pipeline.ModeCheck,
        RelaxModules: true,
    }
    result, err := pipeline.RunWithContext(ctx, cfg, src)
    // Preload compiled modules to runtime
    for path, loaded := range result.Modules {
        e.runtime.PreloadModule(path, loaded)
    }
}
```

**Additional fixes:**
- Set `AILANG_STDLIB_PATH` environment variable for stdlib resolution
- Use absolute paths for module resolution
- Preload with both canonical and user-requested paths
- RWMutex optimization for read-heavy paths

#### 2. AILANG Bridge (`internal/server/ailang_bridge.go`)

Singleton bridge with lazy initialization and graceful fallback:

```go
type AILANGBridge struct {
    engine  *embed.Engine
    enabled bool
    mu      sync.RWMutex
}

func (b *AILANGBridge) SummarizeEvents(events []*coordinator.TaskEventRecord) string {
    if !b.IsEnabled() {
        return coordinator.SummarizeEvents(events)  // Go fallback
    }
    result, err := b.engine.Call("internal/dashboard_transforms/event_formatter",
                                  "summarizeEvents", convertEventsForAILANG(events))
    if err != nil {
        return coordinator.SummarizeEvents(events)  // Graceful fallback
    }
    return embed.ToString(result)
}
```

#### 3. Dashboard Integration (`internal/server/handlers_coordinator.go`)

Wired AILANG bridge into event endpoints:

```go
case "summary":
    bridge := GetAILANGBridge()
    summary := bridge.SummarizeEvents(events)
    resp := map[string]interface{}{
        "content":       summary,
        "ailang_active": bridge.IsEnabled(),
    }
```

#### 4. Event Formatter Module (`internal/dashboard_transforms/event_formatter.ail`)

145-line AILANG module with pure functions:

```ailang
module internal/dashboard_transforms/event_formatter

import std/list (filter, foldl, length)
import std/string (length as strlen, substring, intToStr)

export pure func countTurns(events: [{turnNum: int, ...}]) -> int =
  foldl(\acc e. if e.turnNum > acc then e.turnNum else acc, 0, events)

export pure func summarizeEvents(events: [{turnNum: int, streamType: string, text: string, ...}]) -> string {
  let turns = countTurns(events);
  let toolCount = length(filter(\e. e.streamType == "tool_use", events));
  let textLen = foldl(\acc e. if e.streamType == "text" then acc + strlen(e.text) else acc, 0, events);
  intToStr(turns) ++ " turns, " ++ intToStr(toolCount) ++ " tool calls, " ++ intToStr(textLen) ++ " chars of text"
}
```

### Gaps Discovered and Fixed

| Gap | Description | Status |
|-----|-------------|--------|
| GAP-1 | Teaching prompt wrong about foldl lambda syntax | ✅ Fixed in prompts/v0.6.5.md |
| GAP-2 | Multi-param lambda type inference bug | ✅ Fixed in v0.6.3 |
| GAP-3 | Lambda syntax with foldl | ✅ Fixed (dependent on GAP-2) |
| GAP-4 | No record width subtyping | ✅ Fixed with row polymorphism in v0.6.4 |
| GAP-5 | Pipeline can't eval standalone expressions | ⚠️ Low priority (workaround exists) |
| Missing `repeat` | String repetition function | ✅ Added to std/string in v0.6.3 |
| Missing `maximum` | List maximum function | ✅ Added `maximumInt` to std/list in v0.6.3 |

### Performance Results

**Go vs AILANG comparison** (Apple M2):

| Operation | Go | AILANG | Slowdown | Allocations |
|-----------|-----|--------|----------|-------------|
| truncate | 25 ns | 26 µs | ~1,000x | 153 |
| countTurns (10) | 4 ns | 215 µs | ~50,000x | 1,475 |
| countTurns (100) | 31 ns | 2.1 ms | ~68,000x | 14,268 |
| summarize (100) | 407 ns | 4.1 ms | ~10,000x | 30,605 |

**Assessment:**
- ✅ 4ms latency acceptable for human-refreshed data
- ✅ Good for <100 events per call, <10 calls/second
- ⚠️ Not suitable for hot paths or high-frequency calls

**Root causes of slowdown:**
1. Interpreted execution (no JIT)
2. Value boxing (every primitive wrapped in interface{})
3. Reflection-based Go↔AILANG conversion

## Files Created/Modified

| File | Action | LOC |
|------|--------|-----|
| `internal/embed/embed.go` | Modified - pipeline compilation, RWMutex | +80 |
| `internal/embed/convert.go` | New - Go↔AILANG conversion | 328 |
| `internal/server/ailang_bridge.go` | New - dashboard bridge | 170 |
| `internal/server/handlers_coordinator.go` | Modified - use bridge | +20 |
| `internal/dashboard_transforms/event_formatter.ail` | New - AILANG port | 145 |
| `internal/dashboard_transforms/benchmark_test.go` | New - comparison tests | 270 |
| `internal/dashboard_transforms/GAPS_DISCOVERED.md` | New - gap tracking | 189 |

**Total:** ~1,200 LOC

## Usage

```bash
# Enable AILANG for dashboard transforms
AILANG_DASHBOARD=1 AILANG_PROJECT_ROOT=/path/to/ailang ailang serve

# Verify AILANG is active
curl "http://localhost:1957/api/coordinator/tasks/{id}/events?format=summary"
# Response includes: "ailang_active": true
```

## Success Criteria Met

- [x] `event_formatter.ail` compiles without errors
- [x] Comparison tests pass (Go == AILANG output)
- [x] Performance <10ms for typical workloads (4ms for 100 events)
- [x] GAPS_DISCOVERED.md documents all findings
- [x] Feature flag allows toggling between Go/AILANG

## Lessons Learned

1. **Embed API must use pipeline:** Direct runtime loading bypasses critical passes (OpLowering)
2. **Path resolution is tricky:** Must handle absolute vs relative, canonical vs declared paths
3. **Performance is inherent:** Interpretation overhead is fundamental, not easily optimized
4. **Row polymorphism was key:** `{field: T, ...}` syntax enabled flexible record matching
5. **Stdlib gaps found early:** Dogfooding revealed missing `repeat` and `maximum` functions

## Future Work

1. **JIT compilation** - Would reduce ~1000x overhead significantly
2. **Ahead-of-time compilation** - Generate Go code for hot paths
3. **Batch API** - Process multiple items in single Call to amortize overhead
4. **Additional transforms** - Heatmap generator, cost breakdown calculator

## Related Documents

- [GAPS_DISCOVERED.md](./GAPS_DISCOVERED.md) - Full gap tracking (co-located)
- [m-pipeline-standalone-expr.md](../../planned/v0_7_0/m-pipeline-standalone-expr.md) - GAP-5 design doc
- [benchmark_test.go](../../../internal/dashboard_transforms/benchmark_test.go) - Performance tests
