# M-ASYNC-IO: Multi-Source Event Multiplexing for std/stream

**Status**: IMPLEMENTED (Phase 1)
**Target**: v0.9.0
**Priority**: P1 (Medium) — Unlocks CLI ambient assistant mode
**Estimated**: 1-2 weeks (Phase 1: text-mode mux ~1 week; Phase 2: bytes/pipe ~1 week if Phase 1 solid)
**Dependencies**: M-STREAM-BIDI (v0.8.1, implemented), Effect system (v0.2.0, implemented)
**Source**: msg_20260308_164817_2c8b5818 (demos inbox — ambient assistant)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Event merge policy is deterministic (priority-ordered). External event arrival timing remains nondeterministic and is captured in the trace for replay. |
| A2: Replayability | +1 | All events logged to trace with source tag + timestamp. Replay requires recorded arrival order. |
| A3: Effect Legibility | +1 | All source constructors and `selectEvents` require `Stream` capability — no new capability needed |
| A4: Explicit Authority | +1 | No ambient access to stdin/pipes — must hold `Stream` cap to create sources |
| A5: Bounded Verification | +1 | Local verification: handler signature declares event types it handles |
| A6: Safe Concurrency | +1 | **Key design**: no shared mutable state — sources are read-only ingress channels |
| A7: Machines First | +1 | Machine-parseable structured event stream; merge semantics are decidable |
| A8: Minimal Syntax | 0 | No new syntax — uses existing function calls + ADTs |
| A9: Cost Visibility | +1 | Each source has explicit buffer limits and backpressure |
| A10: Composability | +1 | Sources compose via `selectEvents` — mix WebSocket + stdin + future timer/CSP sources |
| A11: Structured Failure | +1 | Stream errors use typed `StreamErrorKind` ADT (existing pattern) |
| A12: System Boundary | +1 | Each event source is an explicit boundary (stdin, pipe, WebSocket) |

**Net Score: +11** → **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): Merge policy is deterministic; nondeterministic arrival timing is traced, not hidden
- [x] A3 (Effects): `Stream` capability required for all source/mux operations
- [x] A4 (Authority): No ambient access — `Stream` cap gates all source creation
- [x] A7 (Machines First): Event stream is structured data, not human-oriented text

### Determinism Precision (A1/A2)

Event merging is deterministic **conditional on observed source arrival order**. External event timing (stdin keystrokes, network delivery, pipe writes, goroutine scheduling on source readers) remains nondeterministic and is captured in the execution trace for faithful replay. This is the same nondeterminism model as the existing WebSocket `runEventLoop` — we do not introduce new sources of nondeterminism, only new sources of events.

## Problem Statement

The ambient assistant demo needs continuous stdin input while simultaneously processing WebSocket events via `runEventLoop()`. Currently, `runEventLoop()` blocks the main thread in a `select` loop over the WebSocket event buffer, making it impossible to read from other sources concurrently.

**Current State:**
- `runEventLoop()` in `internal/effects/stream.go:409-507` blocks on `select { case evt <- conn.eventBuffer ... }`
- Only one event source (WebSocket) can drive the event loop
- CLI ambient mode limited to text-only (no concurrent stdin + WebSocket)
- Browser version works via JS event loop (different runtime, handles concurrency natively)

**Impact:**
- Any AILANG program needing multiple concurrent I/O sources must choose one
- Blocks the "interactive CLI + live WebSocket" pattern (ambient assistant, chat overlays, monitoring dashboards)

## Goals

**Primary Goal:** Enable AILANG programs to consume events from multiple concurrent sources (stdin, WebSocket, future: pipes/timers) in a single event loop, with deterministic merge semantics.

**Phase 1 Scope (v0.9.0):**
- `selectEvents` multiplexer with deterministic priority ordering
- stdin line source (`asyncReadStdinLines`)
- WebSocket connections as selectable sources (`sourceOfConn`)
- `runEventLoop` becomes sugar over `selectEvents`
- Trace integration for replay

**Deferred to Phase 2:**
- Raw byte stdin (`asyncReadStdinBytes(chunkSize)`)
- Named pipe sources
- Audio pipe examples
- User-configurable priorities
- Generic source taxonomy / `EventEnvelope`

## Solution Design

### Overview

Extend `std/stream` with **multi-source event multiplexing**. The core abstraction: a `StreamSource` is anything that produces `StreamEvent` values. `selectEvents` monitors N sources and dispatches to one handler.

AILANG already has the right architecture. `runEventLoop()` reads from a Go channel (`conn.eventBuffer`). We generalize to read from _multiple_ Go channels, each backed by a different I/O source goroutine.

### Architecture

```
                    ┌─────────────────────────┐
                    │   selectEvents(sources)  │  ← AILANG event loop
                    │   (deterministic merge)  │
                    └────────┬────────────────┘
                             │ unified StreamEvent
                ┌────────────┼────────────────┐
                ▼            ▼                ▼
        ┌──────────┐  ┌──────────┐   ┌──────────────┐
        │  stdin    │  │ WebSocket│   │  (future:    │
        │  reader   │  │ readLoop │   │   pipe/timer)│
        │ goroutine │  │ (exists) │   │              │
        └──────────┘  └──────────┘   └──────────────┘
             ↑              ↑
        os.Stdin      net.Conn
```

**Components:**

1. **StreamSource abstraction** (`internal/effects/stream_source.go`): Interface wrapping any Go channel into a tagged, prioritized event source
2. **selectEvents multiplexer** (`internal/effects/stream_mux.go`): Deterministic priority-ordered dispatch over N source channels
3. **Source constructors**: `asyncReadStdinLines()`, `sourceOfConn(conn)` — each returns a `StreamSource`
4. **Unified event model**: Source-tagged event variants (`SourceText`) alongside existing WebSocket variants

### Capability Design Decision

**No new capability.** All source constructors and `selectEvents` use the existing `Stream` capability.

**Rationale:** From the user's perspective, WebSocket events, stdin lines, and pipe bytes are all "event sources feeding an event loop." They are one coherent authority, not two. Introducing a separate `AsyncIO` cap would split one concept across two authorities without a clear security boundary justification.

If a future use case demands finer-grained control (e.g., "allow WebSocket but deny stdin"), that can be added later via sub-capabilities. For v0.9.0, `Stream` is sufficient.

### Event Model

**Two new generic event variants** replace the source-specific approach:

```ailang
type StreamEvent =
  -- Existing (unchanged)
  | Message(string)
  | Binary(bytes)
  | Opened(string)
  | Closed(int, string)
  | StreamError(StreamErrorKind)
  | Ping(bytes)
  | SSEData(string, string)
  -- New: source-tagged events
  | SourceText(string, string)     -- (sourceName, text)
  | SourceBytes(string, bytes)     -- (sourceName, raw bytes — NOT base64)

-- Event source handle (opaque, like StreamConn)
type StreamSource = StreamSource(int)
```

**Design notes:**
- `SourceText` / `SourceBytes` use a generic `sourceName` tag, not source-specific variants. This avoids an ever-growing ADT as source types are added.
- `SourceBytes` carries `bytes` directly, not base64-encoded strings. Base64 belongs in serialization layers (trace export, JSON), not the runtime event ADT.
- `Binary` and `Ping` also changed from `string` to `bytes` for consistency (breaking change for v0.9.0 — acceptable).
- WebSocket `Message` events retain their own variant for backward compatibility. They are not wrapped in `SourceText`.

**Source naming convention:**
- stdin → `"stdin"`
- WebSocket → `"ws:<url>"` (auto-generated from connection URL)
- Named pipe → `"pipe:<path>"` (Phase 2)

### API Design

**Constructors:**

```ailang
-- Create a line-buffered stdin source
asyncReadStdinLines : () -> StreamSource ! {Stream}

-- Wrap an existing WebSocket connection as a selectable source
sourceOfConn : (conn: StreamConn) -> StreamSource ! {Stream}

-- Multiplex N sources into one event loop
selectEvents : (sources: [StreamSource], handler: StreamEvent -> bool) -> unit ! {Stream}
```

**Unified event loop model:** `selectEvents` is the primary event driver. `runEventLoop` becomes a thin wrapper:

```ailang
-- runEventLoop(conn, handler) is now equivalent to:
-- selectEvents([sourceOfConn(conn)], handler)
```

This eliminates the awkward overlap between `onEvent`/`runEventLoop` and `selectEvents`. One model, not two.

### Example: Concurrent Stdin + WebSocket

```ailang
module examples/stdin_websocket

import std/stream (
  connect, disconnect, transmit,
  asyncReadStdinLines, sourceOfConn, selectEvents,
  StreamEvent, StreamSource, StreamConn, defaultConfig
)

export func main() -> unit ! {Stream, IO} {
  let wsResult = connect("wss://echo.websocket.events", defaultConfig(()));
  match wsResult {
    Ok(conn) => {
      let wsSrc   = sourceOfConn(conn);
      let stdinSrc = asyncReadStdinLines();
      selectEvents([wsSrc, stdinSrc], handler);
      disconnect(conn)
    },
    Err(e) => _io_println("connection failed")
  }
}

export func handler(event: StreamEvent) -> bool {
  match event {
    Message(msg)          => { _io_println(string_concat("ws: ", msg)); true },
    SourceText(src, line) => {
      if string_eq(line, "quit") then false
      else {
        _io_println(string_concat(string_concat(src, ": "), line));
        true
      }
    },
    Closed(_, _)    => false,
    StreamError(_)  => false,
    _               => true
  }
}
```

### Deterministic Multiplexing

**Critical design decision:** Go's `select` on multiple channels is nondeterministic when multiple are ready. This violates Axiom A1.

**Solution — Priority-ordered polling with bounded fairness:**

```go
// In stream_mux.go
// This defines LANGUAGE SEMANTICS, not just implementation.
// Go's reflect.Select is an implementation detail, not the semantic basis.

func selectEventsLoop(sources []eventSource, handler handlerFn) {
    for {
        // Phase 1: Check sources in priority order (highest first)
        // Non-blocking drain of the highest-priority ready source
        for _, src := range sources {  // sorted by priority descending
            select {
            case evt, ok := <-src.ch:
                if !ok { markClosed(src); continue }
                if !handler(tagEvent(src.name, evt)) { return }
                goto nextRound
            default:
                continue
            }
        }
        // Phase 2: No events ready — block until any source delivers
        // Implementation may use reflect.Select internally
        idx, evt := blockOnAny(sources)
        if !handler(tagEvent(sources[idx].name, evt)) { return }
    nextRound:
    }
}
```

**Formal merge rules (v0.9.0):**

1. Sources are ordered by **priority** (integer, higher = checked first). Source list order is the default priority.
2. At each round, choose the **highest-priority source with a ready event** (non-blocking check).
3. Within the **same priority band**, use **round-robin** to prevent starvation.
4. If **no source is ready**, block until any source delivers. (Which source unblocks first is nondeterministic — this is the only nondeterministic point, and it is traced.)
5. **Lower-priority sources may starve** if higher-priority sources are continuously ready. This is documented behavior, not a bug.
6. All merge decisions are **logged to the execution trace** (source name, priority, round number) for replay verification.

**Starvation note:** Bounded fairness (e.g., "serve at most K events from priority band N before checking lower bands") is deferred. For v0.9.0, the simple priority model is sufficient. If starvation becomes a real issue, add a `maxBurst` parameter per source in a later version.

### Implementation Plan

**Phase 1: Text-Mode Multiplexer** (~1 week)
- [ ] Define `EventSource` interface in `internal/effects/stream_source.go`
- [ ] Implement `sourceOfConn()` — adapt existing `streamConn` to `EventSource`
- [ ] Implement `asyncReadStdinLines()` — spawns goroutine with `bufio.Scanner`, writes `SourceText("stdin", line)` to channel
- [ ] Implement `selectEventsLoop()` in `internal/effects/stream_mux.go` — priority-ordered polling as specified above
- [ ] Register builtins: `asyncReadStdinLines`, `sourceOfConn`, `selectEvents`
- [ ] Extend `StreamEvent` ADT in `std/stream.ail` with `SourceText`, `SourceBytes`
- [ ] Make `runEventLoop(conn, handler)` delegate to `selectEvents([sourceOfConn(conn)], handler)`
- [ ] Source lifecycle: cleanup goroutines on handler return / source close
- [ ] Trace integration: log source tag + priority + round to execution trace
- [ ] Unit tests: priority ordering, fairness, source close, timeout
- [ ] Integration test: stdin + WebSocket concurrent reading

**Phase 2: Bytes & Pipes** (~1 week, deferred)
- [ ] `asyncReadStdinBytes(chunkSize: int) -> StreamSource`
- [ ] `asyncReadPipeLines(path: string) -> StreamSource`
- [ ] `asyncReadPipeBytes(path: string, chunkSize: int) -> StreamSource`
- [ ] Cross-platform stdin buffering tests (macOS, Linux)
- [ ] Audio pipe example with `sox`

### Files to Modify/Create

**New files:**
- `internal/effects/stream_source.go` — `EventSource` interface + `sourceOfConn` adapter (~100 LOC)
- `internal/effects/stream_mux.go` — Priority-ordered multiplexer (~200 LOC)
- `internal/effects/stream_stdin.go` — Stdin line reader goroutine (~80 LOC)
- `internal/effects/stream_mux_test.go` — Multiplexer tests (~300 LOC)
- `examples/runnable/stdin_websocket.ail` — Example (~30 LOC)

**Modified files:**
- `std/stream.ail` — Add `SourceText`, `SourceBytes`, `StreamSource` type + new exports (~+25 LOC)
- `internal/builtins/spec.go` — Register `asyncReadStdinLines`, `sourceOfConn`, `selectEvents` (~+25 LOC)
- `internal/effects/stream.go` — Extract channel interface from `streamConn`; make `runEventLoop` delegate to `selectEvents` (~+30 LOC, refactor)

## Success Criteria

- [ ] `selectEvents` multiplexes 2+ sources with deterministic priority ordering
- [ ] `asyncReadStdinLines()` reads line-buffered stdin concurrently with WebSocket
- [ ] `sourceOfConn(conn)` adapts existing WebSocket connections to `StreamSource`
- [ ] `runEventLoop(conn, handler)` is equivalent to `selectEvents([sourceOfConn(conn)], handler)`
- [ ] All operations gated on `Stream` capability (no new capability)
- [ ] Priority ordering verified: when both sources have data, higher priority wins
- [ ] Event trace includes source tags for replay
- [ ] All existing stream tests still pass (backward compatible except `Binary`/`Ping` type change)
- [ ] Documentation updated (prompt, stdlib docs)
- [ ] Example added and verified

## Testing Strategy

**Unit tests:**
- Multiplexer priority ordering (mock channels, verify event order is deterministic)
- Round-robin fairness within same priority band
- Source close handling (one source closes, others continue)
- All sources closed → loop exits
- Timeout/idle behavior with multiple sources

**Integration tests:**
- Stdin reader goroutine with simulated line input (pipe into test process)
- WebSocket + stdin concurrent reading (echo server + piped stdin)
- `Stream` capability enforcement (missing cap → error)
- `runEventLoop` backward compatibility (single-source, existing tests pass)

**Manual testing:**
- `echo "hello\nworld\nquit" | ailang run --caps Stream,IO examples/runnable/stdin_websocket.ail`
- Interactive stdin + WebSocket echo test

## Non-Goals

**Not in Phase 1:**
- Raw byte stdin / binary pipe reading (Phase 2)
- Audio device access from AILANG (use external tools, pipe to stdin)
- Named pipe sources (Phase 2)
- User-configurable priorities / `maxBurst` fairness bounds
- `EventEnvelope` wrapper type with metadata (source, timestamp)
- Full CSP channels (deferred to M-CSP-SESSION-TYPES v1.0.0)
- WASM/browser async I/O (different runtime — browser's event loop handles this natively)
- `async`/`await` syntax — this is a library-level extension, no new keywords

## Timeline

**Week 1** (~20 hours):
- Phase 1: Core multiplexer + stdin line reader + sourceOfConn + selectEvents
- Trace integration + tests

**Week 2** (~4 hours, if Phase 1 is solid):
- Examples, docs, prompt update
- Integration testing edge cases

**Phase 2 (separate milestone, if needed):**
- Bytes mode, pipe sources, audio examples

**Total Phase 1: ~24 hours across 1-2 weeks**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `reflect.Select` performance with many sources | Medium | Benchmark; cap at 8 sources initially; `reflect.Select` is implementation detail, can be replaced |
| stdin buffering semantics differ across OS | Medium | Use `bufio.Scanner` (line-buffered); Phase 2 adds raw mode |
| Priority starvation of low-priority sources | Medium | Document as expected behavior; add `maxBurst` in future version if needed |
| Backpressure: fast stdin floods slow handler | Medium | Per-source buffer limits (configurable, default 1000) |
| `Binary`/`Ping` type change breaks existing code | Low | v0.9.0 is a minor version bump; document in CHANGELOG |
| Source goroutine leaks on unclean exit | Medium | Cleanup on handler return; `context.Context` cancellation |

## Related Documents

**Implemented (informs design):**
- [M-STREAM-BIDI](design_docs/implemented/v0_8_1/m-wasm-stream-bridge-sprint-plan.md) — WebSocket streaming architecture (v0.8.1)
- [M-STREAM-DX](design_docs/implemented/v0_8_1/m-stream-dx-improvements.md) — Stream developer experience (v0.8.1)
- [M-R2 Effect System](design_docs/implemented/v0_2_0/m_r2_effect_system.md) — Capability-based effects (v0.2.0)

**Planned (check for overlap):**
- [M-CSP-SESSION-TYPES](design_docs/planned/v1_0_0/m-csp-session-types.md) — Full CSP concurrency (v1.0.0, this feature is a stepping stone)
- [Execution Profiles](design_docs/planned/v0_9_0/execution-profiles.md) — May define how async I/O fits different profiles
- [M-ARCH4 Executor Stream Processor](design_docs/planned/v0_9_0/m-arch4-executor-stream-processor.md) — Related stream processing architecture

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- [Philosophical Foundations](/docs/references/philosophical-foundations) — Block-universe determinism
- Source message: msg_20260308_164817_2c8b5818 (demos inbox)
- Existing stream runtime: `internal/effects/stream.go`

## Future Work

- **Phase 2 — Bytes & Pipes**: `asyncReadStdinBytes(chunkSize)`, `asyncReadPipeLines(path)`, `asyncReadPipeBytes(path, chunkSize)` for binary ingestion (audio PCM, etc.)
- **EventEnvelope**: Richer handler context `{ source: string, event: StreamEvent, timestamp: int }` — deferred until 3+ source types exist
- **Bounded fairness**: `maxBurst` parameter per source to prevent starvation
- **Named source metadata**: Handler access to source priority, buffer fill level
- **Backpressure signaling**: Source-aware flow control (slow consumer pauses fast producer)
- **CSP integration**: When M-CSP ships, `selectEvents` could accept CSP channels as sources
- **WASM async**: Bridge to browser's async I/O via JS interop (separate design doc)

---

**Document created**: 2026-03-08
**Last updated**: 2026-03-08
