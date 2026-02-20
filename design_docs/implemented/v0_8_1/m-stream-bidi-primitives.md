# M-STREAM-BIDI: Generic Bidirectional Streaming Primitives

**Status**: Planned
**Target**: v0.9.0+
**Priority**: P2 (Medium) - Enables real-time AI agent patterns
**Estimated**: 4-5 weeks (~160 hours)
**Dependencies**: None (callback-based Phase 1 has no dependency on M-CSP-SESSION-TYPES)
**GitHub Issue**: #136

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | `Stream` introduces nondeterminism consistent with `Net`: event arrival timing, chunking, coalescing, and disconnect boundaries are externally determined. Determinism is preserved for a fixed recorded event log (see Replay Contract below). No new *category* of nondeterminism beyond what `std/net` already introduces. |
| A2: Replayability | +1 | Explicit replay contract: `--trace=record` logs all stream events (raw frames, timestamps, connection lifecycle); `--trace=replay=<file>` drives `runEventLoop` from recorded log instead of network. Handler side effects re-execute deterministically given identical event sequence. See Replay Contract section. |
| A3: Effect Legibility | +1 | New `Stream` effect makes persistent connection lifecycle explicit; `! {Stream}` in signatures clearly marks streaming code. |
| A4: Explicit Authority | +1 | `--caps Stream` required; `StreamContext` enforces security (allowlists, connection limits, message size bounds). |
| A5: Bounded Verification | +1 | Connection count limits and per-operation budgets (`Stream.send @limit=N`, `Stream.recv @limit=N`) enable static resource reasoning. |
| A6: Safe Concurrency | 0 | Phase 1 (callback-based) is single-threaded; handler invocations are serialized by the dispatcher. No scheduling-dependent semantics. Phase 2 (channel-based, future) defers to M-CSP-SESSION-TYPES for safety guarantees. |
| A7: Machines First | +1 | Structured event types (ADTs, not raw strings); machine-parseable error hierarchy; JSON-lines wire format. |
| A8: Minimal Syntax | +1 | No new syntax needed. Uses existing function calls, ADTs, lambdas, and pattern matching. All streaming expressed through library functions. |
| A9: Cost Visibility | +1 | Per-operation budgets visible in types (`Stream.send @limit=N`, `Stream.recv @limit=N`); `StreamContext` tracks bytes/messages per connection; budget deltas emitted to traces. |
| A10: Composability | +1 | `std/stream` composes with `std/json` (encode/decode), `std/net` (auth tokens), existing ADTs; generic over any WebSocket/SSE endpoint. |
| A11: Structured Failure | +1 | `StreamError` ADT with typed variants (ConnectionFailed, MessageTooLarge, ProtocolError, Timeout, BudgetExhausted). Errors delivered as events (single error surface). |
| A12: System Boundary | +1 | `connect()` is explicit boundary crossing; `close()` is explicit teardown; protocol negotiation visible in config via `StreamProtocol` ADT. |

**Net Score: +10** -> **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): Nondeterminism is consistent with `Net` — external event arrival. Deterministic replay guaranteed for recorded event logs.
- [x] A3 (Effects): `Stream` effect explicitly declares persistent connection lifecycle
- [x] A4 (Authority): `--caps Stream` required; connection limits and domain allowlists enforced
- [x] A7 (Machines First): Structured event ADTs; no prose-formatted errors; machine-parseable protocol

## Problem Statement

AILANG has no way to implement bidirectional streaming patterns that are becoming essential for AI agent applications:

**Current State:**
- `std/net` provides only request-response HTTP (`httpGet`, `httpPost`, `httpRequest`)
- No persistent connection support (WebSocket, SSE, or similar)
- No way to receive push events from a server
- No streaming iteration over incoming data
- Bytes type exists (v0.5.11) but no binary streaming
- Concurrency keywords reserved (`spawn`, `channel`, `send`, `recv`) but unimplemented

**Impact:**
- Cannot build real-time voice/video AI agents (Google ADK BIDI, OpenAI Realtime API)
- Cannot implement Server-Sent Events for live dashboards
- Cannot subscribe to database change streams (Firestore, MongoDB)
- Cannot implement MQTT/IoT patterns
- Forces AILANG users to drop down to Go for any streaming use case

**Who is affected:**
- AI agent developers needing live API integration (Gemini Live, OpenAI Realtime, ElevenLabs)
- Applications requiring push notifications or real-time updates
- IoT and monitoring applications consuming event streams

**Systemic Analysis:**
This is NOT a one-off feature request. It addresses a fundamental gap — AILANG has no primitive for "persistent connection with bidirectional message flow." The existing `Net` effect covers request-response; `Stream` covers connection-lifecycle patterns. These are orthogonal concerns that compose cleanly.

## Goals

**Primary Goal:** Add generic, protocol-agnostic bidirectional streaming primitives to AILANG's standard library, using a new `Stream` effect with the same security model as `std/net`.

**Success Metrics:**
1. WebSocket connections work with echo server + one real API (Gemini Live or OpenAI Realtime)
2. SSE connections work against a local test server for unidirectional streaming
3. Binary data (PCM audio, images) can be sent/received via `bytes` type
4. Per-operation budgets (`Stream.send @limit=N`, `Stream.recv @limit=N`) correctly limit operations
5. Connection security (domain allowlist, TLS enforcement, connection limits) matches `Net` rigor
6. All streaming examples pass `make verify-examples`
7. 90%+ test coverage for new packages

## Solution Design

### Overview

Introduce a `std/stream` module backed by a new `Stream` effect, following the same architectural patterns as `std/net`:

1. **AILANG module** (`std/stream.ail`) — public API with typed functions
2. **Go builtins** (`internal/builtins/stream.go`) — builtin registration via `BuiltinSpec`
3. **Effect implementation** (`internal/effects/stream.go`) — Go runtime with `gorilla/websocket`
4. **Effect context** (`internal/effects/stream_context.go`) — security config and connection tracking
5. **Effect registration** — Add `"Stream"` to `effects.Registry`

### Concurrency Model Decision: Callback-Based (Option A from Issue #136)

**Chosen approach: Callback-based (Phase 1), channel-based (Phase 2, future)**

Rationale:
- **No new keywords needed** — uses existing lambdas for event handlers
- **No dependency on M-CSP-SESSION-TYPES** — can ship independently
- **Runtime manages event loop** — Go goroutines handle WebSocket I/O internally; AILANG sees synchronous callbacks
- **Composable with future channels** — `std/stream` can later be reimplemented on top of channels without API change
- **Axiom A8 compliance** — zero new syntax

The Go runtime spawns internal goroutines for connection I/O, but these are invisible to AILANG code. From AILANG's perspective, `onEvent` registers a handler and `runEventLoop` blocks until the connection closes — deterministic, sequential semantics.

### Callback Dispatch Model

**Reentrancy and serialization guarantees:**

1. **Handler invocations are serialized.** A single dispatch goroutine reads from the internal event buffer and calls the AILANG handler sequentially. The next event is not dispatched until the handler returns.

2. **Backpressure policy: Block.** If the handler is slow, the internal read goroutine blocks when the bounded event buffer (capacity: 1000 events) is full. This applies backpressure to the network read, which is the simplest and most axiom-friendly policy (no dropped events, no reordering).

3. **`send()` is safe inside handlers.** The dispatcher does NOT hold the connection write lock during handler invocation. Calling `send(conn, msg)` from within a handler acquires the write lock independently and completes without deadlock.

4. **Handler panics are recovered.** If an AILANG handler panics, the dispatcher catches it via `recover()` and delivers an `Error(ProtocolError("handler panic: ..."))` event. The event loop continues unless the error handler also returns `false`.

5. **Handler errors.** If the handler function itself returns an error (via the effect system), it is treated as if the handler returned `false` — the event loop terminates and `runEventLoop` returns the error.

```
┌─────────────┐     ┌──────────────────┐     ┌───────────────┐
│ Network I/O │────▶│ Bounded Buffer   │────▶│ Dispatcher    │
│ (goroutine) │     │ (cap: 1000)      │     │ (sequential)  │
│             │     │                  │     │               │
│ Blocks when │◀────│ Back-pressure    │     │ Calls handler │
│ buffer full │     │ when full        │     │ one at a time │
└─────────────┘     └──────────────────┘     └───────────────┘
                                                    │
                                              ┌─────▼─────┐
                                              │ AILANG     │
                                              │ Handler    │
                                              │            │
                                              │ Can call   │
                                              │ send()     │
                                              └────────────┘
```

### Replay Contract

**Stream replay is a runtime mode, not a stdlib feature.**

When `--trace=record` is active:
- Every stream event (raw frames, parsed events, timestamps, connection open/close) is logged to the trace stream in order
- Each event record includes: `{timestamp, conn_id, event_type, payload_bytes, payload_text}`
- Connection metadata (URL, protocol, headers minus auth) is logged at open time

When `--trace=replay=<file>` is active:
- `_stream_connect` returns a synthetic `StreamConn` without touching the network
- `_stream_send` records the send but does not transmit (allows verifying send sequences)
- `_stream_runEventLoop` reads events from the trace file instead of the network
- Events are delivered to the handler in recorded order with original timing optionally preserved
- Handler side effects (IO, etc.) re-execute deterministically given identical event sequence

This keeps A2 honest: replay is a first-class runtime capability, not just "events are traceable."

### Budget Semantics

**Budget counters are split by semantic operation, not a single integer.**

Each operation type has its own counter tracked as sub-budget keys in `BudgetContext`:

| Budget Key | What it counts | CLI Syntax |
|------------|---------------|------------|
| `Stream.connect` | Connection establishments | `--caps "Stream.connect @limit=4"` |
| `Stream.send` | Messages sent (one per `send()` call) | `--caps "Stream.send @limit=100"` |
| `Stream.recv` | Events received (one per handler invocation) | `--caps "Stream.recv @limit=1000"` |

**Convenience shorthand:** `--caps "Stream @limit=N"` sets all three sub-budgets to N.

**Budget units:**
- For WebSocket: one message = one budget unit (regardless of frame fragmentation)
- For SSE: one event = one budget unit (an event is `data:` block terminated by blank line)

**Exhaustion behavior:** When a budget is exhausted, the runtime delivers an `Error(BudgetExhausted("Stream.send limit exceeded: 100/100"))` event to the handler. For `send()`, the call returns `Err(BudgetExhausted(...))`. For `recv`, the event loop terminates after delivering the error event.

### Error Surface Decision

**Errors are delivered as events (single error surface).**

There is ONE primary error delivery path: the `Error(StreamError)` variant in `StreamEvent`. This avoids double-signaling between event delivery and return values.

- `connect()` returns `Result[StreamConn, StreamError]` — this is the only function that returns errors via Result, because the connection must exist before events can flow.
- `send()` returns `Result[unit, StreamError]` — immediate failures (budget exhausted, connection closed) are returned here since they are synchronous.
- `runEventLoop()` returns `unit` — all errors during the event loop (timeout, protocol error, budget exhaustion) are delivered as `Error(...)` events to the handler. The function returns `unit` when the loop terminates (handler returned false, connection closed, or error delivered).
- `close()` returns `unit` — fire and forget; errors during close are logged to trace.

### Architecture

```
std/stream.ail                     # AILANG public API
    │
    ├── connect()      → _stream_connect        # Establish connection
    ├── send()         → _stream_send            # Send message/binary
    ├── onEvent()      → _stream_onEvent         # Register event handler
    ├── runEventLoop() → _stream_runEventLoop    # Block until close
    ├── close()        → _stream_close           # Graceful shutdown
    ├── withStream()   → connect+onEvent+run+close  # High-level helper
    └── status()       → _stream_status          # Connection state query
         │
         ▼
internal/builtins/stream.go        # BuiltinSpec registration (7 builtins)
         │
         ▼
internal/effects/stream.go         # Go implementation
    │
    ├── StreamConnection struct    # Per-connection state
    │   ├── gorilla/websocket.Conn # WebSocket backend
    │   ├── eventHandler           # Registered AILANG callback (single)
    │   ├── eventBuffer            # Bounded incoming event queue (cap: 1000)
    │   └── metrics                # Bytes/messages counters + sub-budgets
    │
    └── StreamContext struct        # Security + connection tracking
        ├── MaxConnections int      # Default: 4
        ├── MaxMessageSize int64    # Default: 1MB (per message, not per frame)
        ├── MaxFrameSize   int64    # Default: 64KB (gorilla read limit)
        ├── AllowedDomains []string # Domain allowlist
        ├── AllowedProtocols        # [WebSocket, SSE]
        ├── Timeout time.Duration   # Connection timeout: 5min
        ├── IdleTimeout             # No-message timeout: 60s
        ├── BlockPrivateIPs bool    # Default: true (RFC1918 + link-local)
        └── connections map[id]*StreamConnection
```

### Type Definitions

```ailang
-- std/stream.ail

module std/stream

import std/result (Result)

-- Stream connection handle (opaque)
export type StreamConn = StreamConn(int)  -- Internal connection ID

-- Protocol selection (typed, not stringly)
export type StreamProtocol =
  | WebSocket      -- Bidirectional, full-duplex
  | SSE            -- Unidirectional (server → client only; send() is a no-op)

-- Connection configuration
export type StreamConfig = {
  protocol: StreamProtocol,                        -- WebSocket | SSE
  headers: List[{name: string, value: string}],    -- Custom headers (auth, etc.)
  subprotocols: List[string]                       -- WebSocket subprotocols (ignored for SSE)
}

-- Stream events (received from server)
-- This is the SINGLE error surface: all errors arrive as Error(StreamError) events
export type StreamEvent =
  | Message(string)         -- Text message (WS text frame / SSE data field)
  | Binary(bytes)           -- Binary data (WS binary frame; not applicable for SSE)
  | Opened(StreamOpenInfo)  -- Connection opened
  | Closed(int, string)     -- Connection closed (code, reason); code 1000 = clean close
  | Error(StreamError)      -- Error occurred (timeout, budget, protocol, handler panic)
  | Ping(bytes)             -- Ping frame (auto-ponged by runtime; informational only)

-- Connection open metadata
export type StreamOpenInfo = {
  protocol: StreamProtocol,      -- Negotiated protocol
  subprotocol: string            -- Selected WebSocket subprotocol ("" if none)
}

-- SSE-specific event (delivered as Message with structured content)
-- For SSE connections, Message(data) contains the SSE `data` field.
-- SSE metadata (event type, id, retry) is available via sseInfo():
export type SSEEventInfo = {
  eventType: string,   -- SSE `event:` field ("message" if omitted)
  id: string,          -- SSE `id:` field ("" if omitted)
  retry: int           -- SSE `retry:` field (0 if omitted)
}

-- Stream errors (structured, typed)
export type StreamError =
  | ConnectionFailed(string)   -- Could not establish connection
  | MessageTooLarge(string)    -- Message exceeds size limit
  | ProtocolError(string)      -- WebSocket/SSE protocol violation or handler panic
  | Timeout(string)            -- Connection or idle timeout
  | BudgetExhausted(string)    -- Sub-budget limit exceeded (includes which budget)
  | DisallowedHost(string)     -- Domain not in allowlist
  | InvalidProtocol(string)    -- Unsupported protocol requested

-- Message types for sending
export type StreamMessage =
  | Text(string)      -- Send text frame
  | Bin(bytes)         -- Send binary frame

-- Connection status (typed, not stringly)
export type StreamStatus =
  | Connecting         -- Connection in progress
  | Open               -- Connected and ready
  | Closing            -- Close handshake in progress
  | Closed             -- Connection terminated
```

### Public API

```ailang
-- Connect to a streaming endpoint
-- Returns connection handle or error
-- This is the ONLY function that returns Result — errors during event loop
-- are delivered as Error(StreamError) events to the handler.
export func connect(
  url: string,
  config: StreamConfig
) -> Result[StreamConn, StreamError] ! {Stream} =
  _stream_connect(url, config)

-- Send a message on an open connection
-- Returns Err for immediate failures (budget exhausted, connection closed)
-- For SSE connections, send() is a no-op and returns Ok(())
export func send(
  conn: StreamConn,
  msg: StreamMessage
) -> Result[unit, StreamError] ! {Stream} =
  _stream_send(conn, msg)

-- Register event handler for incoming events
-- Handler is called for each event; returns true to continue, false to stop
-- Handler MAY perform effects (IO, Net, etc.) — effect row is propagated
export func onEvent[e](
  conn: StreamConn,
  handler: StreamEvent -> bool ! {e}
) -> unit ! {Stream, e} =
  _stream_onEvent(conn, handler)

-- Run the event loop (blocks until connection closes or handler returns false)
-- Returns unit — all errors are delivered as Error(StreamError) events
export func runEventLoop(conn: StreamConn) -> unit ! {Stream} =
  _stream_runEventLoop(conn)

-- Close a connection gracefully
export func close(conn: StreamConn) -> unit ! {Stream} =
  _stream_close(conn)

-- Query connection status (returns typed ADT, not string)
export func status(conn: StreamConn) -> StreamStatus ! {Stream} =
  _stream_status(conn)

-- High-level helper: connect, register handler, run event loop, close
-- Ensures connection is always closed (defer-equivalent semantics)
-- Reduces boilerplate and prevents leaked connections
export func withStream[e](
  url: string,
  config: StreamConfig,
  handler: StreamConn -> StreamEvent -> bool ! {e}
) -> Result[unit, StreamError] ! {Stream, e} =
  match connect(url, config) {
    Ok(conn) => {
      onEvent(conn, handler(conn));
      runEventLoop(conn);
      close(conn);
      Ok(())
    },
    Err(e) => Err(e)
  }

-- Query SSE event metadata (only meaningful for SSE connections)
-- Returns None for WebSocket connections
export func sseInfo(conn: StreamConn) -> Result[SSEEventInfo, StreamError] ! {Stream} =
  _stream_sseInfo(conn)
```

### Builtin Registration Pattern

Following the established pattern from `internal/builtins/net.go`:

```go
// internal/builtins/stream.go
package builtins

func init() {
    registerStreamConnect()
    registerStreamSend()
    registerStreamOnEvent()
    registerStreamRunEventLoop()
    registerStreamClose()
    registerStreamStatus()
    registerStreamWithStream()
    registerStreamSSEInfo()
}

func registerStreamConnect() {
    err := RegisterEffectBuiltin(BuiltinSpec{
        Module:  "std/stream",
        Name:    "_stream_connect",
        NumArgs: 2,
        IsPure:  false,
        Effect:  "Stream",
        Type:    makeStreamConnectType,
        Impl:    effects.StreamConnect,
        Metadata: &BuiltinMetadata{
            Description: "Establish a persistent streaming connection (WebSocket/SSE)",
            Since:       "v0.9.0",
            Category:    "streaming",
        },
    })
    if err != nil {
        panic("failed to register _stream_connect: " + err.Error())
    }
}
// ... similar for other builtins
```

### Effect Registration

```go
// internal/effects/ops.go - Add to Registry
var Registry = map[string]map[string]EffOp{
    "IO":     {},
    "FS":     {},
    "Clock":  {},
    "Net":    {},
    "Debug":  {},
    "Stream": {},  // NEW
}
```

### StreamContext (Security Model)

Following the `NetContext` pattern from `internal/effects/context.go`:

```go
// internal/effects/stream_context.go
type StreamContext struct {
    // Connection limits
    MaxConnections int           // Default: 4 (prevent resource exhaustion)
    MaxMessageSize int64         // Default: 1MB per message (reassembled from frames)
    MaxFrameSize   int64         // Default: 64KB per frame (gorilla ReadLimit)

    // Per-operation budgets (sub-budget keys in BudgetContext)
    // These are tracked as "Stream.connect", "Stream.send", "Stream.recv"
    ConnectBudget  int           // Default: -1 (unlimited); set via --caps "Stream.connect @limit=N"
    SendBudget     int           // Default: -1 (unlimited); set via --caps "Stream.send @limit=N"
    RecvBudget     int           // Default: -1 (unlimited); set via --caps "Stream.recv @limit=N"

    // Timeouts
    ConnectTimeout time.Duration // Default: 30s
    IdleTimeout    time.Duration // Default: 60s (close if no messages; reset on any activity)
    MaxDuration    time.Duration // Default: 5min (hard ceiling)

    // Security (inherits from NetContext patterns)
    AllowHTTP       bool          // Default: false (wss:// only)
    AllowLocalhost  bool          // Default: false
    BlockPrivateIPs bool          // Default: true (RFC1918: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16 + link-local)
    AllowedDomains  []string      // Domain allowlist (empty = all allowed)

    // Protocol
    AllowedProtocols []StreamProtocol // Default: [WebSocket, SSE]

    // Event buffer
    EventBufferSize int           // Default: 1000 (bounded; backpressure when full)

    // Runtime state
    mu          sync.Mutex
    connections map[int]*StreamConnection
    nextID      int
}
```

### Go Runtime Implementation (Key Functions)

```go
// internal/effects/stream.go

// StreamConnect establishes a WebSocket or SSE connection
func StreamConnect(ctx *EffContext, args []eval.Value) (eval.Value, error) {
    // 1. Parse URL and StreamConfig (extract StreamProtocol ADT variant)
    // 2. Validate domain (allowlist), protocol (wss/https), IP (block private + localhost)
    // 3. Check connection count limit
    // 4. Decrement Stream.connect budget (return BudgetExhausted if exceeded)
    // 5. Establish WebSocket connection via gorilla/websocket (or HTTP GET for SSE)
    // 6. Start internal read goroutine → bounded event buffer (cap: EventBufferSize)
    // 7. If --trace=record: log connection metadata
    // 8. Return StreamConn(id) wrapped in Ok()
}

// StreamSend sends a text or binary message
func StreamSend(ctx *EffContext, args []eval.Value) (eval.Value, error) {
    // 1. Look up connection by ID
    // 2. If SSE: return Ok(()) (no-op, SSE is unidirectional)
    // 3. Decrement Stream.send budget (return Err(BudgetExhausted) if exceeded)
    // 4. Check message size limit
    // 5. Acquire write lock (independent of dispatcher — no deadlock with handler)
    // 6. Write to WebSocket (Text or Binary frame based on StreamMessage variant)
    // 7. Update metrics (bytes sent, messages sent)
    // 8. If --trace=record: log sent message
    // 9. Return Ok(()) or Err(StreamError)
}

// StreamOnEvent registers an AILANG handler function for events
func StreamOnEvent(ctx *EffContext, args []eval.Value) (eval.Value, error) {
    // 1. Look up connection by ID
    // 2. Store handler function (eval.FunctionValue) in connection state
    //    Handler type: StreamEvent -> bool ! {e} (effectful)
    // 3. Return unit
}

// StreamRunEventLoop blocks, dispatching events to registered handler
func StreamRunEventLoop(ctx *EffContext, args []eval.Value) (eval.Value, error) {
    // 1. Look up connection by ID
    // 2. Loop (single dispatcher goroutine, sequential):
    //    a. Read from bounded event buffer (blocks if empty)
    //    b. Decrement Stream.recv budget; if exhausted → deliver Error event, break
    //    c. If --trace=replay: read event from trace file instead of buffer
    //    d. Call AILANG handler via eval engine (with recover for panics)
    //    e. If handler returns false → break
    //    f. If handler panics → deliver Error(ProtocolError("handler panic: ..."))
    //    g. If connection closed → deliver Closed event, break
    //    h. If timeout → deliver Error(Timeout(...)), break
    // 3. If --trace=record: log all dispatched events
    // 4. Return unit (all errors delivered as events, not as return value)
}
```

### CLI Integration

```bash
# Run with Stream capability
ailang run --caps IO,Stream --entry main voice_agent.ail

# Allow WebSocket over HTTP (development)
ailang run --caps IO,Stream --stream-allow-http --entry main test.ail

# Domain allowlist
ailang run --caps IO,Stream --stream-allow-domains "generativelanguage.googleapis.com,api.openai.com" --entry main agent.ail

# Per-operation budgets (recommended)
ailang run --caps "IO,Stream.send @limit=100,Stream.recv @limit=1000" --entry main agent.ail

# Shorthand: set all sub-budgets to same value
ailang run --caps "IO,Stream @limit=100" --entry main agent.ail

# Record stream events for replay
ailang run --caps IO,Stream --trace=record --entry main agent.ail

# Replay recorded stream events (deterministic re-execution)
ailang run --caps IO,Stream --trace=replay=stream_events.jsonl --entry main agent.ail
```

### Implementation Plan

**Phase 1: Core Infrastructure** (~40 hours, Week 1-2)
- [ ] Add `"Stream"` to `effects.Registry` in `ops.go`
- [ ] Create `StreamContext` in `internal/effects/stream_context.go` (~250 LOC)
  - Sub-budget tracking (Stream.connect, Stream.send, Stream.recv)
  - Private IP blocking (RFC1918 + link-local)
  - Frame size vs message size limits
- [ ] Create `StreamConnection` struct in `internal/effects/stream.go` (~200 LOC)
  - Bounded event buffer (capacity configurable, default 1000)
  - Serialized dispatch goroutine
  - Write lock independent of dispatcher
- [ ] Implement WebSocket connect/close in Go (~250 LOC)
- [ ] Add `gorilla/websocket` dependency
- [ ] Wire `StreamContext` into `EffContext` (modify `context.go`)
- [ ] Add `--caps Stream` CLI flag parsing (with sub-budget syntax)
- [ ] Add `--stream-allow-http`, `--stream-allow-domains` CLI flags
- [ ] Unit tests for StreamContext, budgets, and connection lifecycle (~250 LOC)

**Phase 2: Builtins & AILANG Module** (~40 hours, Week 2-3)
- [ ] Register 8 builtins in `internal/builtins/stream.go` (~450 LOC)
- [ ] Implement `_stream_connect` with full security validation (~150 LOC)
- [ ] Implement `_stream_send` with per-operation budget tracking (~120 LOC)
- [ ] Implement `_stream_onEvent` handler registration (effectful handler type) (~80 LOC)
- [ ] Implement `_stream_runEventLoop` with serialized dispatch + panic recovery (~250 LOC)
- [ ] Implement `_stream_close` graceful shutdown (~50 LOC)
- [ ] Implement `_stream_status` returning `StreamStatus` ADT (~40 LOC)
- [ ] Implement `_stream_withStream` high-level helper (~60 LOC)
- [ ] Implement `_stream_sseInfo` SSE metadata query (~40 LOC)
- [ ] Create `std/stream.ail` module with types and exports (~150 LOC)
- [ ] Register types (StreamConn, StreamProtocol, StreamConfig, StreamEvent, StreamOpenInfo, SSEEventInfo, StreamError, StreamMessage, StreamStatus)
- [ ] Unit tests for each builtin (~350 LOC)

**Phase 3: SSE Support** (~20 hours, Week 3)
- [ ] Add SSE (Server-Sent Events) connection backend (~200 LOC)
  - Unidirectional: `send()` is a no-op for SSE connections
  - SSE event parsing (event, data, id, retry fields) → `SSEEventInfo`
  - Multi-line data concatenation
- [ ] Protocol detection from `StreamProtocol` ADT variant
- [ ] SSE unit tests (~150 LOC)

**Phase 4: Replay, Integration, Examples & Docs** (~30 hours, Week 4)
- [ ] Implement trace record mode for stream events (~150 LOC)
- [ ] Implement trace replay mode (`--trace=replay=<file>`) (~200 LOC)
- [ ] Create `examples/stream_websocket.ail` — basic WebSocket echo
- [ ] Create `examples/stream_sse.ail` — SSE event consumer
- [ ] Create `examples/stream_voice_agent.ail` — Gemini Live API pattern
- [ ] Integration tests with test WebSocket server (~200 LOC)
- [ ] Per-operation budget integration tests (~150 LOC)
- [ ] Replay integration tests (record → replay → verify) (~150 LOC)
- [ ] Trace integration tests (verify events emitted) (~100 LOC)
- [ ] Update `docs/LIMITATIONS.md` with streaming caveats
- [ ] Update `CHANGELOG.md`
- [ ] Update `README.md` with streaming capability

**Phase 5: Hardening** (~30 hours, Week 5)
- [ ] Connection cleanup on panic/timeout
- [ ] Graceful shutdown propagation (SIGTERM → close all connections)
- [ ] Memory leak prevention (connection registry cleanup)
- [ ] Stress testing (many connections, large messages, rapid open/close)
- [ ] Security audit (DNS rebinding, origin validation, private IP blocking, header exfiltration)
- [ ] `ailang doctor stream` validation command
- [ ] Performance benchmarks

### Files to Modify/Create

**New files:**
| File | LOC | Purpose |
|------|-----|---------|
| `std/stream.ail` | ~150 | AILANG module (types + exports) |
| `internal/effects/stream.go` | ~600 | WebSocket/SSE Go implementation + dispatch |
| `internal/effects/stream_context.go` | ~250 | Security config + connection tracking + sub-budgets |
| `internal/effects/stream_replay.go` | ~200 | Trace record/replay for stream events |
| `internal/effects/stream_test.go` | ~450 | Unit tests |
| `internal/builtins/stream.go` | ~450 | Builtin registration (8 builtins) |
| `internal/builtins/stream_test.go` | ~350 | Builtin tests |
| `examples/stream_websocket.ail` | ~30 | WebSocket echo example (using withStream) |
| `examples/stream_sse.ail` | ~30 | SSE consumer example |
| `examples/stream_voice_agent.ail` | ~60 | Gemini Live API pattern |

**Modified files:**
| File | Changes | Purpose |
|------|---------|---------|
| `internal/effects/ops.go` | +1 line | Add `"Stream": {}` to Registry |
| `internal/effects/context.go` | +5 lines | Add `Stream *StreamContext` field |
| `cmd/ailang/main.go` | ~25 lines | CLI flags for Stream caps + sub-budgets |
| `internal/runtime/config.go` | ~15 lines | Stream runtime config + replay mode |
| `go.mod` | +1 dep | `gorilla/websocket` |
| `CHANGELOG.md` | ~25 lines | Document new feature |
| `README.md` | ~10 lines | Update implementation status |

**Total new code:** ~2,500 LOC (implementation) + ~1,200 LOC (tests)

## Examples

### Example 1: WebSocket Echo Client (using withStream)

```ailang
module examples/stream_websocket

import std/stream (withStream, send, StreamConfig, StreamConn, StreamEvent, StreamMessage, StreamError, WebSocket)
import std/result (Result)
import std/io (println)

func main() -> unit ! {IO, Stream} {
  let config = {
    protocol: WebSocket,
    headers: [],
    subprotocols: []
  };

  -- withStream handles connect/run/close lifecycle automatically
  match withStream("wss://echo.websocket.events", config, \conn. \event.
    match event {
      Opened(info)         => { println("Connected! Subprotocol: " ++ info.subprotocol); send(conn, Text("Hello from AILANG!")); true },
      Message(data)        => { println("Received: " ++ data); false },
      Binary(data)         => { println("Binary received"); true },
      Closed(code, reason) => { println("Closed: " ++ intToString(code)); false },
      Error(err)           => { println("Error occurred"); false },
      Ping(_)              => true
    }
  ) {
    Ok(()) => (),
    Err(ConnectionFailed(msg)) => println("Failed to connect: " ++ msg),
    Err(DisallowedHost(host))  => println("Host blocked: " ++ host),
    Err(_)                     => println("Connection error")
  }
}
```

### Example 2: Gemini Live Voice Agent

```ailang
module examples/stream_voice_agent

import std/stream (connect, send, onEvent, runEventLoop, close, StreamConfig, StreamEvent, StreamMessage, WebSocket)
import std/json (encode, decode)
import std/net (httpRequest)
import std/io (println)
import std/result (Result)

-- Get OAuth2 token (simplified)
func getToken(projectId: string) -> string ! {Net} =
  match httpRequest("POST", "https://oauth2.googleapis.com/token", [], "") {
    Ok(resp) => resp.body,
    Err(_)   => ""
  }

func main() -> unit ! {IO, Net, Stream} {
  let token = getToken("my-project");
  let url = "wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1beta.GenerativeService.BidiGenerateContent";
  let config = {
    protocol: WebSocket,
    headers: [{name: "Authorization", value: "Bearer " ++ token}],
    subprotocols: []
  };

  match connect(url, config) {
    Ok(conn) => {
      -- Send setup message
      send(conn, Text(encode({
        setup: {
          model: "gemini-2.5-flash",
          generationConfig: {responseModalities: ["AUDIO"]}
        }
      })));

      -- Handle streaming events
      onEvent(conn, \event. match event {
        Message(data) => {
          println("Server: " ++ data);
          true
        },
        Binary(audioBytes) => {
          -- Process audio chunk (PCM 24kHz)
          println("Audio chunk received");
          true
        },
        Error(Timeout(msg)) => { println("Timeout: " ++ msg); false },
        Error(BudgetExhausted(msg)) => { println("Budget: " ++ msg); false },
        Error(_) => false,
        Closed(_, _) => false,
        _ => true
      });

      runEventLoop(conn);
      close(conn)
    },
    Err(err) => println("Connection failed")
  }
}
```

### Example 3: Server-Sent Events Consumer

```ailang
module examples/stream_sse

import std/stream (withStream, StreamConfig, StreamEvent, SSE)
import std/io (println)

func main() -> unit ! {IO, Stream} {
  let config = {
    protocol: SSE,
    headers: [{name: "Accept", value: "text/event-stream"}],
    subprotocols: []
  };

  -- SSE is unidirectional: send() inside handler would be a no-op
  match withStream("https://example.com/events", config, \_conn. \event.
    match event {
      Opened(_)    => { println("SSE connected"); true },
      Message(data) => {
        println("Event: " ++ data);
        true  -- Continue listening
      },
      Closed(_, _) => false,
      Error(_)     => false,
      _            => true
    }
  ) {
    Ok(()) => (),
    Err(_) => println("Failed to connect to SSE endpoint")
  }
}
```

### Example 4: Budget-Limited Streaming

```ailang
module examples/stream_budget

import std/stream (connect, send, onEvent, runEventLoop, close, StreamConfig, StreamEvent, StreamMessage, WebSocket)
import std/io (println)

-- Run with per-operation budgets:
-- ailang run --caps "IO,Stream.send @limit=5,Stream.recv @limit=10" --entry main examples/stream_budget.ail
--
-- Stream.send limits outgoing messages (5 sends allowed)
-- Stream.recv limits incoming events (10 events allowed)
-- Exceeding either delivers Error(BudgetExhausted(...)) event

func main() -> unit ! {IO, Stream} {
  let config = {protocol: WebSocket, headers: [], subprotocols: []};

  match connect("wss://echo.websocket.events", config) {
    Ok(conn) => {
      -- Each send counts against Stream.send budget
      send(conn, Text("Message 1"));
      send(conn, Text("Message 2"));
      send(conn, Text("Message 3"));

      onEvent(conn, \event. match event {
        Message(data) => { println(data); true },
        Error(BudgetExhausted(msg)) => { println("Budget hit: " ++ msg); false },
        Closed(_, _) => false,
        _ => true
      });

      runEventLoop(conn);
      close(conn)
    },
    Err(_) => println("Connection failed")
  }
}
```

## Security Considerations

### Network Security

| Concern | Mitigation |
|---------|-----------|
| DNS rebinding | Validate `Host` header matches connection URL; re-resolve DNS before data exchange |
| Private IP access | `BlockPrivateIPs` (default: true) blocks RFC1918 (10/8, 172.16/12, 192.168/16) + link-local (169.254/16, fe80::/10) |
| TLS enforcement | Default: `wss://` and `https://` only; `ws://` requires `--stream-allow-http` flag |
| Domain allowlist | `--stream-allow-domains` restricts connections to listed hosts |
| IP literals | Blocked by default (treated as "no domain" → fails allowlist if set) |

### Resource Limits

| Concern | Mitigation |
|---------|-----------|
| Connection exhaustion | `MaxConnections` (default: 4 concurrent) |
| Large messages | `MaxMessageSize` (default: 1MB) per reassembled message |
| Large frames | `MaxFrameSize` (default: 64KB) passed to gorilla `ReadLimit` |
| Memory from buffered events | Bounded event buffer (default: 1000); backpressure blocks network read |
| Runaway connections | `MaxDuration` (default: 5min hard ceiling); `IdleTimeout` (default: 60s) |

### Header Security

- Authentication headers (`Authorization`, `Cookie`) are logged in traces with values redacted (`Bearer ***`)
- `--stream-allow-domains` must include the auth token endpoint if using token refresh

### Non-Goals (Security)

- **TLS certificate pinning** — not in Phase 1; can be added via custom TLS config later
- **Per-header allowlist** — considered but adds complexity; rely on domain allowlist instead
- **WebSocket Origin validation** — server-side concern; this is a client library

## Success Criteria

- [ ] `_stream_connect` establishes WebSocket connections with full security validation
- [ ] `_stream_send` sends text and binary messages with per-operation budget tracking
- [ ] `_stream_onEvent` registers effectful AILANG handler functions
- [ ] `_stream_runEventLoop` dispatches events to handler sequentially with panic recovery
- [ ] `_stream_close` performs graceful WebSocket close handshake
- [ ] `Stream` effect enforced via `--caps Stream`
- [ ] Per-operation budgets work: `Stream.connect`, `Stream.send`, `Stream.recv`
- [ ] Domain allowlist (`--stream-allow-domains`) blocks unauthorized hosts
- [ ] TLS enforcement (`wss://` by default, `--stream-allow-http` for `ws://`)
- [ ] Private IP blocking (RFC1918 + link-local) enabled by default
- [ ] SSE protocol supported via `StreamProtocol::SSE` (unidirectional; `send()` is no-op)
- [ ] Connection limits enforced (default: 4 concurrent)
- [ ] Message size limits enforced (default: 1MB message, 64KB frame)
- [ ] Idle timeout closes stale connections (reset on any activity)
- [ ] `withStream` helper correctly handles connect/run/close lifecycle
- [ ] Trace record mode logs all stream events
- [ ] Trace replay mode drives event loop from recorded log
- [ ] `ailang doctor stream` validates all stream builtins
- [ ] All example files pass `make verify-examples`
- [ ] 90%+ test coverage for `internal/effects/stream*.go`
- [ ] 90%+ test coverage for `internal/builtins/stream*.go`
- [ ] Documentation updated (CHANGELOG, README, LIMITATIONS)
- [ ] All existing tests pass (`make test`)

## Testing Strategy

### Unit Tests

**StreamContext tests** (`internal/effects/stream_context_test.go`):
- Connection limit enforcement (4 connections → 5th rejected)
- Domain allowlist validation (exact match, wildcard)
- Protocol validation (wss allowed, ws blocked by default)
- Private IP blocking (10.0.0.1, 172.16.0.1, 192.168.1.1, 169.254.x.x all blocked)
- Message size limit (1MB default) vs frame size limit (64KB default)
- Per-operation budget tracking (connect, send, recv independently)
- Timeout configuration
- Event buffer capacity enforcement

**Builtin tests** (`internal/builtins/stream_test.go`):
- Type signature validation for all 8 builtins
- Argument count validation
- Effect tag verification (`[stream]`)
- Effectful handler type verification (handler carries effect row)
- Using `MockEffContext` for hermetic testing

**Event handling tests** (`internal/effects/stream_test.go`):
- Text message send/receive round-trip
- Binary message send/receive round-trip
- Handler registration and invocation (effectful handler)
- Handler returning false stops event loop
- Connection close propagates Closed event
- Error delivery as events (timeout, budget, protocol error)
- Handler panic recovery → Error(ProtocolError("handler panic: ..."))
- send() from within handler (no deadlock)
- Serialized dispatch (events processed one at a time)
- Backpressure when event buffer full

**Replay tests** (`internal/effects/stream_replay_test.go`):
- Record mode captures all events with timestamps
- Replay mode delivers events in recorded order
- Replay with synthetic StreamConn (no network)
- Round-trip: record → replay → verify identical handler calls

### Integration Tests

**WebSocket integration** (test server in Go):
- Full connect → send → receive → close lifecycle
- Multiple concurrent connections (within limit)
- Binary frame handling (bytes round-trip)
- Subprotocol negotiation
- `withStream` lifecycle (auto-close on handler return false)

**SSE integration** (test server in Go):
- SSE event parsing (event, data, id, retry)
- Multi-line data events
- `send()` is no-op for SSE connections
- SSEEventInfo metadata query

**Budget integration**:
- `Stream.send @limit=5` → 6th send returns Err(BudgetExhausted)
- `Stream.recv @limit=10` → 11th event delivers Error(BudgetExhausted)
- `Stream.connect @limit=2` → 3rd connect returns Err(BudgetExhausted)
- Shorthand `Stream @limit=N` sets all three sub-budgets
- Budget trace events emitted

### Manual Testing

- [ ] Connect to `wss://echo.websocket.events` (public WebSocket echo)
- [ ] Connect to Gemini Live API or OpenAI Realtime (requires auth)
- [ ] Connect to SSE test server (local; not a public endpoint for CI stability)
- [ ] Verify connection cleanup on program exit (SIGTERM)
- [ ] Verify timeout behavior (idle connection closes)
- [ ] Verify trace record/replay round-trip

## Design Decisions

**Resolved decisions that shaped the final design:**

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Error surface | Events (single surface) | Avoids double-signaling; `Error(StreamError)` events are consistent with "stream = events" model. Only `connect()` and `send()` return `Result` for immediate/synchronous failures. |
| Handler effects | Effectful (`StreamEvent -> bool ! {e}`) | Real handlers do IO, Net, etc. Pure handler type would force awkward wrappers or fail to typecheck. Effect row `{e}` propagates through `onEvent`. |
| Protocol selection | `StreamProtocol` ADT | Typed discriminant avoids stringly-typed bugs; `WebSocket \| SSE` is exhaustive in match. |
| Budget granularity | Per-operation sub-budgets | Single integer budget is uninterpretable; developers cannot reason about "100 operations" across connect+send+recv. Per-operation budgets are independently reasoned about. |
| Backpressure | Block (bounded buffer) | Simplest policy; no dropped events; no reordering; most axiom-friendly. Alternative policies (DropOldest, DropNewest) change semantics and violate A1 for replay. |
| Replay | Runtime mode, not stdlib | `--trace=record` / `--trace=replay=<file>` keeps replay orthogonal to user code. Handler side effects re-execute deterministically given identical event sequence. |
| SSE directionality | Acknowledged as unidirectional | `send()` is a no-op for SSE; no fake bidirectionality. SSE metadata via `sseInfo()`. |
| send() in handler | Allowed, no deadlock | Dispatcher releases connection lock before calling handler; `send()` acquires write lock independently. |

**Open questions for implementation (non-blocking):**

1. **Should replay preserve original timing?** Option A: deliver events as fast as handler processes them (faster testing). Option B: sleep to match recorded timestamps (fidelity). *Recommendation: Option A by default, `--trace=replay-realtime=<file>` for Option B.*

2. **Should `withStream` accept a `StreamConn -> StreamEvent -> bool` or just `StreamEvent -> bool`?** The former allows `send()` in the handler via the conn argument. *Decision: `StreamConn -> StreamEvent -> bool ! {e}` (curried, conn available).*

3. **SSE auto-reconnect.** Phase 1: no auto-reconnect (Axiom A1). Document manual reconnection pattern in examples. Phase 2: optional `reconnect: true` in config with explicit backoff.

## Non-Goals

**Not in this feature (Phase 1):**
- **Channel-based concurrency** — Deferred to M-CSP-SESSION-TYPES (v0.8.1+). Phase 1 uses callbacks.
- **`spawn` keyword integration** — No concurrent AILANG tasks. Go goroutines are internal only.
- **Server-side WebSocket** — `std/stream` is client-only. Server endpoints are a separate feature.
- **MQTT/AMQP/gRPC** — Focus on WebSocket and SSE. Other protocols can be added later as additional backends.
- **Automatic reconnection** — Must be explicit in AILANG code. No hidden retry logic (Axiom A1).
- **Session types for streaming protocols** — Deferred to M-CSP-SESSION-TYPES.
- **Stream combinators** (`map`, `filter`, `merge`) — Future work after channels exist.
- **WebRTC** — Different protocol category; separate design doc if needed.
- **TLS certificate pinning** — Can be added via custom TLS config later.
- **DropOldest/DropNewest backpressure** — Block is the only policy for Phase 1; alternatives change semantics.

## Timeline

**Week 1** (40 hours):
- Phase 1: Core infrastructure (StreamContext, sub-budgets, connection management, Go WebSocket)
- Add `gorilla/websocket` dependency
- Wire Stream effect into CLI (including sub-budget syntax parsing)

**Week 2** (40 hours):
- Phase 2: Builtins and AILANG module
- Register all 8 builtins (including withStream, sseInfo)
- Create `std/stream.ail` with all type definitions

**Week 3** (20 hours):
- Phase 3: SSE support (unidirectional backend)
- SSE event parsing and `SSEEventInfo`
- Protocol detection from `StreamProtocol` ADT

**Week 4** (30 hours):
- Phase 4: Replay contract, integration tests, examples, documentation
- Trace record/replay implementation
- 3 example files (echo, SSE, Gemini Live)
- CHANGELOG/README updates

**Week 5** (30 hours):
- Phase 5: Hardening
- Security audit (DNS rebinding, private IP, origin validation)
- Stress testing, connection cleanup, SIGTERM handling
- `ailang doctor stream` command

**Total: ~160 hours across 5 weeks**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Go goroutine leak (connections not cleaned up) | High | Connection registry with finalizer; `withStream` ensures cleanup; SIGTERM handler |
| WebSocket library compatibility | Medium | Use well-maintained `gorilla/websocket`; integration tests against real endpoints |
| Callback handler panics | High | `recover()` in dispatcher; propagate as `Error(ProtocolError("..."))` event |
| Per-operation budget tracking complexity | Medium | Three independent counters in `StreamContext`; unit tests for each budget independently |
| Idle timeout races with legitimate slow streams | Medium | Configurable idle timeout; reset on ANY activity (send or recv) |
| SSE send() confusion | Low | `send()` returns `Ok(())` silently for SSE; `status()` returns protocol info for runtime checks |
| Replay fidelity | Medium | Record raw frames + timestamps; replay test suite verifies identical handler call sequences |
| Memory pressure from buffered events | Medium | Bounded event buffer (1000 events); Block backpressure policy; configurable via `EventBufferSize` |
| Handler deadlock if send() holds locks | High | Dispatcher releases ALL locks before calling handler; send() acquires write lock independently |
| Voice agent example complexity | Medium | Scope to connection + setup message only; actual audio processing is application-level |

## Relationship to Existing Features

| Feature | Relationship |
|---------|-------------|
| `std/net` | Complementary — `std/net` for request-response, `std/stream` for persistent connections. Both use domain allowlists. `std/net` auth tokens compose with `StreamConfig.headers`. |
| `std/json` | Composes — `encode`/`decode` used for structured WebSocket messages |
| `std/io` | Composes — `println` for logging stream events (effectful handlers enable this) |
| Bytes type (v0.5.11) | Required — `Binary(bytes)` events use existing bytes infrastructure |
| Capability budgets (v0.6.2) | Integrates — Sub-budgets use existing `BudgetContext` mechanism with keyed counters |
| M-CSP-SESSION-TYPES (planned) | Future integration — `std/stream` Phase 2 can use channels internally |
| M-ARCH4 Executor Stream (planned) | Different layer — M-ARCH4 is internal tooling JSON parsing; this is language-level |
| `ailang serve-api` | Future — could later support WebSocket endpoints backed by `std/stream` |
| WASM target (future) | Compatible — browser WebSocket API could back `std/stream` in WASM |
| Trace system | Integrates — stream events recorded via `EffContext.Trace`; replay mode drives event loop from trace |

## Related Documents

**Implemented (informs design):**
- [design_docs/implemented/v0_6_2/m-capability-budgets.md](../../implemented/v0_6_2/m-capability-budgets.md) — Resource-bounded effects pattern (sub-budget keys)
- [design_docs/implemented/v0_0_3/initial_design.md](../../implemented/v0_0_3/initial_design.md) — Original effect system design (IO, FS, Clock, Net)

**Planned (related work):**
- [design_docs/planned/v0_8_1/m-csp-session-types.md](m-csp-session-types.md) — CSP channels (Phase 2 foundation)
- [design_docs/planned/v0_8_1/m-arch4-executor-stream-processor.md](m-arch4-executor-stream-processor.md) — Internal JSON stream processing (different layer)

**Axiom References:**
- [Design Axioms](/docs/references/axioms) — A1 (Determinism + replay), A3 (Effect Legibility), A4 (Explicit Authority), A8 (Minimal Syntax)

**Implementation References:**
- `std/net.ail` — Pattern for stdlib module structure
- `internal/effects/net.go` — Pattern for Go effect implementation
- `internal/builtins/net.go` — Pattern for builtin registration
- `internal/effects/context.go` — Pattern for effect context (StreamContext follows NetContext)

## Future Work

**Phase 2: Channel-Based Streaming** (after M-CSP-SESSION-TYPES ships)
- Reimplement `std/stream` internals using AILANG channels
- Add `toChannel(conn) -> Chan[StreamEvent]` bridge function
- Enable `select` over multiple stream connections
- Session types for streaming protocols

**Phase 3: Stream Combinators**
- `map(conn, f)` — transform events
- `filter(conn, pred)` — filter events
- `merge(conn1, conn2)` — combine multiple streams
- `buffer(conn, n)` — windowed buffering
- `timeout(conn, duration)` — per-event timeout

**Phase 4: Server-Side Streaming**
- `std/stream.listen(port, handler)` — WebSocket server
- Integration with `ailang serve-api` for WebSocket endpoints
- Load balancing and connection management

**Phase 5: Additional Protocols**
- MQTT client (IoT patterns)
- gRPC streaming (bidirectional RPC)
- Database change streams (Firestore, MongoDB)
- Custom protocol backends via plugin interface

---

**Document created**: 2026-02-16
**Last updated**: 2026-02-16
**Revision**: 2 — Incorporated feedback on determinism framing, replay contract, budget semantics, callback dispatch model, effectful handlers, single error surface, typed protocol ADT, withStream helper, and security considerations.
