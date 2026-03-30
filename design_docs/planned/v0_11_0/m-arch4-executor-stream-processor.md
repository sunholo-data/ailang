# M-ARCH4: Executor Stream Processor

**Status**: Planned
**Target**: v0.6.5
**Priority**: P2 (Medium)
**Estimated**: 8-10 hours
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to determinism |
| A2: Replayability | +1 | Unified stream parsing enables consistent traces |
| A3: Effect Legibility | 0 | No change to effects |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Single parsing implementation to verify |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Reduces duplicate code for AI maintenance |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No cost changes |
| A10: Composability | +1 | Shared processor works with any JSON stream |
| A11: Structured Failure | +1 | Unified error handling for stream parsing |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Reduces code duplication

## Problem Statement

Both executor implementations (Claude Code CLI, Gemini CLI) have ~50 lines of identical JSON stream processing code. This duplication means bug fixes must be applied twice and behavior can drift.

**Current State:**
- `internal/executor/claude/claude.go:150-200` - JSON stream parsing
- `internal/executor/gemini/gemini.go:150-200` - Nearly identical JSON stream parsing

**Duplicated pattern:**
```go
scanner := bufio.NewScanner(stdout)
for scanner.Scan() {
    line := scanner.Bytes()
    var event map[string]interface{}
    if err := json.Unmarshal(line, &event); err != nil {
        continue // or handle error
    }
    // Process event type
    eventType, _ := event["type"].(string)
    switch eventType {
    case "assistant":
        // handle assistant message
    case "tool_use":
        // handle tool use
    case "result":
        // handle result
    }
    handler(event)
}
```

**Impact:**
- Bug fixes applied inconsistently
- 100+ lines of duplicated code
- Different error handling between executors
- New event types require changes in 2 files

## Goals

**Primary Goal:** Extract common JSON stream processing into shared `StreamProcessor` that both executors use.

**Success Metrics:**
- Single stream processing implementation
- Executor implementations reduced by ~40 lines each
- New event types require changes in 1 file
- All executor tests pass

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| JSON-lines only (no binary/protobuf streams) | Limits future executor integrations to JSON-lines output format; non-JSON CLIs would need adapters | human | design | med |
| Non-JSON lines silently skipped (not errored) | Critical for Gemini executor which emits non-JSON debug lines to stdout; changing to strict mode would break it | human | design | high |
| `EventHandler` is a callback function (not interface) | Simpler API but no lifecycle hooks (OnStart, OnEnd); interface would allow richer executor integration | human | design | med |
| 1MB max line buffer default | Affects ability to handle large tool outputs; too small truncates, too large wastes memory | human | design | low |
| Event struct uses `map[string]interface{}` for Content (not typed structs) | Keeps processor generic but pushes type assertions to executor-specific handlers | human | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Confirm that both Claude and Gemini CLI outputs are valid JSON-lines (one JSON object per line)
- [ ] Decide if `Event.Content` should be `map[string]interface{}` or `json.RawMessage` for downstream parsing
- [ ] Verify 1MB buffer is sufficient by checking max observed line length in production logs
- [ ] Determine whether context cancellation should drain remaining buffered lines or abort immediately
- [ ] Agree on error semantics: should handler errors stop processing or be collected?

## Solution Design

### Overview

Create `internal/executor/stream/processor.go` with a `StreamProcessor` that handles JSON-lines parsing, error handling, and event dispatching. Executors configure the processor with their specific event handlers.

### Architecture

```
internal/executor/
├── stream/
│   ├── processor.go      # StreamProcessor implementation (~150 LOC)
│   ├── events.go         # Event type definitions (~50 LOC)
│   └── processor_test.go # Tests (~200 LOC)
├── claude/
│   └── claude.go         # Uses StreamProcessor
└── gemini/
    └── gemini.go         # Uses StreamProcessor
```

**Components:**

1. **StreamProcessor**: Reads JSON lines from io.Reader, parses events, calls handlers
2. **Event Types**: Typed structs for common event types (assistant, tool_use, result)
3. **EventHandler**: Callback interface for executor-specific handling

### Implementation Plan

**Phase 1: Create StreamProcessor** (~4 hours)
- [ ] Create `internal/executor/stream/processor.go`
- [ ] Define `Event` struct and common event types
- [ ] Implement `Process(reader io.Reader, handler EventHandler) error`
- [ ] Add error handling with context
- [ ] Add unit tests with mock readers

**Phase 2: Migrate Claude Executor** (~2 hours)
- [ ] Refactor `claude/claude.go` to use StreamProcessor
- [ ] Remove duplicate stream parsing code
- [ ] Verify Claude executor tests pass

**Phase 3: Migrate Gemini Executor** (~2 hours)
- [ ] Refactor `gemini/gemini.go` to use StreamProcessor
- [ ] Remove duplicate stream parsing code
- [ ] Verify Gemini executor tests pass

### Files to Modify/Create

**New files:**
- `internal/executor/stream/processor.go` (~150 LOC)
- `internal/executor/stream/events.go` (~50 LOC)
- `internal/executor/stream/processor_test.go` (~200 LOC)

**Modified files:**
- `internal/executor/claude/claude.go` - Use StreamProcessor (~-45 LOC)
- `internal/executor/gemini/gemini.go` - Use StreamProcessor (~-45 LOC)

## Examples

### Example 1: StreamProcessor Interface

**New shared processor:**
```go
package stream

import (
    "bufio"
    "context"
    "encoding/json"
    "io"
)

// Event represents a parsed JSON event from CLI output
type Event struct {
    Type    string                 `json:"type"`
    Content map[string]interface{} `json:"-"` // Full event data
}

// EventHandler processes events from the stream
type EventHandler func(event Event) error

// StreamProcessor handles JSON-lines stream processing
type StreamProcessor struct {
    maxLineSize int
}

func NewStreamProcessor() *StreamProcessor {
    return &StreamProcessor{
        maxLineSize: 1024 * 1024, // 1MB max line
    }
}

// Process reads JSON lines from reader and calls handler for each event
func (p *StreamProcessor) Process(ctx context.Context, reader io.Reader, handler EventHandler) error {
    scanner := bufio.NewScanner(reader)
    scanner.Buffer(make([]byte, p.maxLineSize), p.maxLineSize)

    for scanner.Scan() {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }

        line := scanner.Bytes()
        if len(line) == 0 {
            continue
        }

        var event Event
        if err := json.Unmarshal(line, &event); err != nil {
            // Log but continue - some lines may not be JSON
            continue
        }

        // Store full content for handler access
        if err := json.Unmarshal(line, &event.Content); err == nil {
            if err := handler(event); err != nil {
                return err
            }
        }
    }

    return scanner.Err()
}
```

### Example 2: Claude Executor Using StreamProcessor

**Before (claude/claude.go ~50 lines of stream handling):**
```go
func (e *Executor) executeStreaming(ctx context.Context, cmd *exec.Cmd, handler ExecutionHandler) error {
    stdout, _ := cmd.StdoutPipe()
    cmd.Start()

    scanner := bufio.NewScanner(stdout)
    for scanner.Scan() {
        line := scanner.Bytes()
        var event map[string]interface{}
        if err := json.Unmarshal(line, &event); err != nil {
            continue
        }
        eventType, _ := event["type"].(string)
        switch eventType {
        case "assistant":
            if msg, ok := event["message"].(map[string]interface{}); ok {
                handler.OnAssistant(msg)
            }
        case "tool_use":
            handler.OnToolUse(event)
        case "result":
            handler.OnResult(event)
        }
    }
    return scanner.Err()
}
```

**After (claude/claude.go ~15 lines):**
```go
func (e *Executor) executeStreaming(ctx context.Context, cmd *exec.Cmd, handler ExecutionHandler) error {
    stdout, _ := cmd.StdoutPipe()
    cmd.Start()

    processor := stream.NewStreamProcessor()
    return processor.Process(ctx, stdout, func(event stream.Event) error {
        switch event.Type {
        case "assistant":
            handler.OnAssistant(event.Content)
        case "tool_use":
            handler.OnToolUse(event.Content)
        case "result":
            handler.OnResult(event.Content)
        }
        return nil
    })
}
```

### Example 3: Testing Stream Processing

```go
func TestStreamProcessor_ParsesEvents(t *testing.T) {
    input := `{"type":"assistant","message":"Hello"}
{"type":"tool_use","tool":"bash","input":"ls"}
{"type":"result","success":true}`

    processor := stream.NewStreamProcessor()
    var events []stream.Event

    err := processor.Process(context.Background(), strings.NewReader(input),
        func(e stream.Event) error {
            events = append(events, e)
            return nil
        })

    assert.NoError(t, err)
    assert.Len(t, events, 3)
    assert.Equal(t, "assistant", events[0].Type)
    assert.Equal(t, "tool_use", events[1].Type)
    assert.Equal(t, "result", events[2].Type)
}
```

## Success Criteria

- [ ] StreamProcessor handles JSON-lines parsing
- [ ] Both executors use StreamProcessor
- [ ] Stream parsing code removed from executor files
- [ ] Context cancellation works correctly
- [ ] Error handling is consistent
- [ ] All executor tests pass
- [ ] Documentation added

## Testing Strategy

**Unit tests:**
- Test valid JSON parsing
- Test invalid JSON handling (skip, don't fail)
- Test empty lines handling
- Test context cancellation
- Test large lines (buffer sizing)

**Integration tests:**
- Claude executor integration tests
- Gemini executor integration tests

**Manual testing:**
- Run actual CLI commands and verify stream handling

## Deferred Decisions

The following are intentionally left open for the implementer:

- Whether to use `bufio.Scanner` or `bufio.Reader` for line reading — [agent may resolve]
- Exact error wrapping format for stream parse failures — [agent may resolve]
- Whether `StreamProcessor` should be a struct with options or a plain function — [agent may resolve]
- Binary/protobuf stream support — will be needed when non-JSON executors are added — [human may resolve]
- Event filtering by type (skip certain event types before handler) — [human may resolve]
- Event buffering/batching for high-throughput scenarios — [human may resolve]

## Non-Goals

**Not in this feature:**
- Adding new event types - Focus on extraction
- Changing event handling behavior - Same functionality

## Timeline

**Day 1** (4 hours):
- Create StreamProcessor package with tests

**Day 2** (4 hours):
- Migrate both executors
- Final testing

**Total: ~8 hours across 2 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Breaking executor behavior | High | Add golden tests for stream parsing first |
| Performance regression | Low | Benchmark before/after |
| Buffer sizing issues | Medium | Make buffer size configurable |

## Related Documents

**Implemented (may inform design):**
- [design_docs/implemented/v0_6_2/m-coordinator-feedback-loop.md](design_docs/implemented/v0_6_2/m-coordinator-feedback-loop.md) (0.44)

## References

- [Design Axioms](/docs/references/axioms)
- `internal/executor/claude/claude.go` - Current implementation
- `internal/executor/gemini/gemini.go` - Current implementation

## Future Work

- Support for binary event streams
- Event filtering by type
- Event buffering/batching

---

**Document created**: 2026-01-05
**Last updated**: 2026-01-05
