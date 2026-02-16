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
| A1: Determinism | 0 | Streaming is inherently event-driven but not nondeterministic per se; event order is preserved per connection. Network I/O nondeterminism already captured by `Net` effect. No new nondeterminism beyond what `std/net` already introduces. |
| A2: Replayability | +1 | All stream events are traceable via `EffContext.Trace`; serializable event logs enable replay of conversation history. |
| A3: Effect Legibility | +1 | New `Stream` effect makes persistent connection lifecycle explicit; `! {Stream}` in signatures clearly marks streaming code. |
| A4: Explicit Authority | +1 | `--caps Stream` required; `StreamContext` enforces security (allowlists, connection limits, message size bounds). |
| A5: Bounded Verification | +1 | Connection count limits and message budgets (`Stream @limit=N`) enable static resource reasoning. |
| A6: Safe Concurrency | 0 | Phase 1 (callback-based) is single-threaded; no scheduling-dependent semantics. Phase 2 (channel-based, future) defers to M-CSP-SESSION-TYPES for safety guarantees. |
| A7: Machines First | +1 | Structured event types (ADTs, not raw strings); machine-parseable error hierarchy; JSON-lines wire format. |
| A8: Minimal Syntax | +1 | No new syntax needed. Uses existing function calls, ADTs, lambdas, and pattern matching. All streaming expressed through library functions. |
| A9: Cost Visibility | +1 | `Stream @limit=N` budgets visible in types; `StreamContext` tracks bytes/messages per connection; budget deltas emitted to traces. |
| A10: Composability | +1 | `std/stream` composes with `std/json` (encode/decode), `std/net` (auth tokens), existing ADTs; generic over any WebSocket/SSE endpoint. |
| A11: Structured Failure | +1 | `StreamError` ADT with typed variants (ConnectionFailed, MessageTooLarge, ProtocolError, Timeout, BudgetExhausted). |
| A12: System Boundary | +1 | `connect()` is explicit boundary crossing; `close()` is explicit teardown; protocol negotiation visible in config. |

**Net Score: +10** -> **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism — event delivery order is preserved per connection; network timing nondeterminism already captured by effect system
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
1. WebSocket connections work with at least 3 real-world APIs (Google Gemini Live, OpenAI Realtime, generic echo server)
2. SSE connections work for unidirectional streaming
3. Binary data (PCM audio, images) can be sent/received via `bytes` type
4. `Stream @limit=N` budgets correctly limit message count
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

### Architecture

```
std/stream.ail                     # AILANG public API
    │
    ├── connect()      → _stream_connect        # Establish connection
    ├── send()         → _stream_send            # Send message/binary
    ├── onEvent()      → _stream_onEvent         # Register event handler
    ├── runEventLoop() → _stream_runEventLoop    # Block until close
    ├── close()        → _stream_close           # Graceful shutdown
    └── status()       → _stream_status          # Connection state query
         │
         ▼
internal/builtins/stream.go        # BuiltinSpec registration (6 builtins)
         │
         ▼
internal/effects/stream.go         # Go implementation
    │
    ├── StreamConnection struct    # Per-connection state
    │   ├── gorilla/websocket.Conn # WebSocket backend
    │   ├── eventHandlers          # Registered AILANG callbacks
    │   ├── messageBuffer          # Incoming message queue
    │   └── metrics                # Bytes/messages counters
    │
    └── StreamContext struct        # Security + connection tracking
        ├── MaxConnections int      # Default: 4
        ├── MaxMessageSize int64    # Default: 1MB
        ├── AllowedDomains []string # Domain allowlist
        ├── AllowedProtocols        # ["websocket", "sse"]
        ├── Timeout time.Duration   # Connection timeout: 5min
        ├── IdleTimeout             # No-message timeout: 60s
        └── connections map[id]*StreamConnection
```

### Type Definitions

```ailang
-- std/stream.ail

module std/stream

import std/result (Result)

-- Stream connection handle (opaque)
export type StreamConn = StreamConn(int)  -- Internal connection ID

-- Connection configuration
export type StreamConfig = {
  protocol: string,                              -- "websocket" | "sse"
  headers: List[{name: string, value: string}],  -- Custom headers (auth, etc.)
  subprotocols: List[string]                     -- WebSocket subprotocols
}

-- Stream events (received from server)
export type StreamEvent =
  | Message(string)         -- Text message
  | Binary(bytes)           -- Binary data (audio, images)
  | Opened(string)          -- Connection opened (selected subprotocol)
  | Closed(int, string)     -- Connection closed (code, reason)
  | Error(StreamError)      -- Error occurred
  | Ping(bytes)             -- Ping frame (auto-ponged by runtime)

-- Stream errors (structured, typed)
export type StreamError =
  | ConnectionFailed(string)   -- Could not establish connection
  | MessageTooLarge(string)    -- Message exceeds size limit
  | ProtocolError(string)      -- WebSocket/SSE protocol violation
  | Timeout(string)            -- Connection or idle timeout
  | BudgetExhausted(string)    -- Stream @limit=N exceeded
  | DisallowedHost(string)     -- Domain not in allowlist
  | InvalidProtocol(string)    -- Unsupported protocol requested

-- Message types for sending
export type StreamMessage =
  | Text(string)      -- Send text frame
  | Bin(bytes)         -- Send binary frame
```

### Public API

```ailang
-- Connect to a streaming endpoint
-- Returns connection handle or error
export func connect(
  url: string,
  config: StreamConfig
) -> Result[StreamConn, StreamError] ! {Stream} =
  _stream_connect(url, config)

-- Send a message on an open connection
export func send(
  conn: StreamConn,
  msg: StreamMessage
) -> Result[unit, StreamError] ! {Stream} =
  _stream_send(conn, msg)

-- Register event handler for incoming events
-- Handler is called for each event; returns true to continue, false to stop
export func onEvent(
  conn: StreamConn,
  handler: StreamEvent -> bool
) -> unit ! {Stream} =
  _stream_onEvent(conn, handler)

-- Run the event loop (blocks until connection closes or handler returns false)
-- This is the main entry point for consuming events
export func runEventLoop(conn: StreamConn) -> Result[unit, StreamError] ! {Stream} =
  _stream_runEventLoop(conn)

-- Close a connection gracefully
export func close(conn: StreamConn) -> unit ! {Stream} =
  _stream_close(conn)

-- Query connection status
export func status(conn: StreamConn) -> string ! {Stream} =
  _stream_status(conn)
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
// ... similar for other 5 builtins
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
    MaxMessageSize int64         // Default: 1MB per message
    MaxTotalBytes  int64         // Default: 100MB lifetime per connection

    // Timeouts
    ConnectTimeout time.Duration // Default: 30s
    IdleTimeout    time.Duration // Default: 60s (close if no messages)
    MaxDuration    time.Duration // Default: 5min (hard ceiling)

    // Security (inherits from NetContext patterns)
    AllowHTTP      bool          // Default: false (wss:// only)
    AllowLocalhost bool          // Default: false
    AllowedDomains []string      // Domain allowlist (empty = all)

    // Protocol
    AllowedProtocols []string    // Default: ["websocket", "sse"]

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
    // 1. Parse URL and config
    // 2. Validate domain (allowlist), protocol (wss/https), IP (no localhost)
    // 3. Check connection count limit
    // 4. Establish WebSocket connection via gorilla/websocket
    // 5. Start internal read goroutine (buffers events)
    // 6. Return StreamConn(id) wrapped in Ok()
}

// StreamSend sends a text or binary message
func StreamSend(ctx *EffContext, args []eval.Value) (eval.Value, error) {
    // 1. Look up connection by ID
    // 2. Check budget (Stream @limit=N)
    // 3. Check message size limit
    // 4. Write to WebSocket (Text or Binary frame based on StreamMessage variant)
    // 5. Update metrics (bytes sent, messages sent)
    // 6. Return Ok(()) or Err(StreamError)
}

// StreamOnEvent registers an AILANG handler function for events
func StreamOnEvent(ctx *EffContext, args []eval.Value) (eval.Value, error) {
    // 1. Look up connection by ID
    // 2. Store handler function (eval.FunctionValue) in connection state
    // 3. Return unit
}

// StreamRunEventLoop blocks, dispatching events to registered handler
func StreamRunEventLoop(ctx *EffContext, args []eval.Value) (eval.Value, error) {
    // 1. Look up connection by ID
    // 2. Loop: read from internal message buffer
    // 3. For each event: call AILANG handler via eval engine
    // 4. If handler returns false → break
    // 5. If connection closed → break with Closed event
    // 6. If timeout → break with Timeout error
    // 7. Return Ok(()) or Err(StreamError)
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

# Budget limit
ailang run --caps "IO,Stream @limit=100" --entry main agent.ail
```

### Implementation Plan

**Phase 1: Core Infrastructure** (~40 hours, Week 1-2)
- [ ] Add `"Stream"` to `effects.Registry` in `ops.go`
- [ ] Create `StreamContext` in `internal/effects/stream_context.go` (~200 LOC)
- [ ] Create `StreamConnection` struct in `internal/effects/stream.go` (~150 LOC)
- [ ] Implement WebSocket connect/close in Go (~250 LOC)
- [ ] Add `gorilla/websocket` dependency
- [ ] Wire `StreamContext` into `EffContext` (modify `context.go`)
- [ ] Add `--caps Stream` CLI flag parsing
- [ ] Add `--stream-allow-http`, `--stream-allow-domains` CLI flags
- [ ] Unit tests for StreamContext and connection lifecycle (~200 LOC)

**Phase 2: Builtins & AILANG Module** (~40 hours, Week 2-3)
- [ ] Register 6 builtins in `internal/builtins/stream.go` (~400 LOC)
- [ ] Implement `_stream_connect` with full security validation (~150 LOC)
- [ ] Implement `_stream_send` with budget tracking (~100 LOC)
- [ ] Implement `_stream_onEvent` handler registration (~80 LOC)
- [ ] Implement `_stream_runEventLoop` with event dispatching (~200 LOC)
- [ ] Implement `_stream_close` graceful shutdown (~50 LOC)
- [ ] Implement `_stream_status` connection query (~30 LOC)
- [ ] Create `std/stream.ail` module with types and exports (~120 LOC)
- [ ] Register types (StreamConn, StreamConfig, StreamEvent, StreamError, StreamMessage)
- [ ] Unit tests for each builtin (~300 LOC)

**Phase 3: SSE Support** (~20 hours, Week 3)
- [ ] Add SSE (Server-Sent Events) connection backend (~200 LOC)
- [ ] SSE-specific event parsing (event, data, id, retry fields)
- [ ] Protocol detection from config (`protocol: "sse"`)
- [ ] SSE unit tests (~150 LOC)

**Phase 4: Integration, Examples & Docs** (~30 hours, Week 4)
- [ ] Create `examples/stream_websocket.ail` — basic WebSocket echo
- [ ] Create `examples/stream_sse.ail` — SSE event consumer
- [ ] Create `examples/stream_voice_agent.ail` — Google ADK BIDI pattern
- [ ] Create `examples/stream_openai_realtime.ail` — OpenAI Realtime API pattern
- [ ] Integration tests with test WebSocket server (~200 LOC)
- [ ] Budget integration tests (`Stream @limit=N`) (~100 LOC)
- [ ] Trace integration tests (verify events emitted) (~100 LOC)
- [ ] Update `docs/LIMITATIONS.md` with streaming caveats
- [ ] Update `CHANGELOG.md`
- [ ] Update `README.md` with streaming capability

**Phase 5: Hardening** (~30 hours, Week 5)
- [ ] Connection cleanup on panic/timeout
- [ ] Graceful shutdown propagation (SIGTERM → close all connections)
- [ ] Memory leak prevention (connection registry cleanup)
- [ ] Stress testing (many connections, large messages, rapid open/close)
- [ ] Security audit (DNS rebinding for WebSocket, origin validation)
- [ ] `ailang doctor stream` validation command
- [ ] Performance benchmarks

### Files to Modify/Create

**New files:**
| File | LOC | Purpose |
|------|-----|---------|
| `std/stream.ail` | ~120 | AILANG module (types + exports) |
| `internal/effects/stream.go` | ~500 | WebSocket/SSE Go implementation |
| `internal/effects/stream_context.go` | ~200 | Security config + connection tracking |
| `internal/effects/stream_test.go` | ~400 | Unit tests |
| `internal/builtins/stream.go` | ~400 | Builtin registration (6 builtins) |
| `internal/builtins/stream_test.go` | ~300 | Builtin tests |
| `examples/stream_websocket.ail` | ~40 | WebSocket echo example |
| `examples/stream_sse.ail` | ~30 | SSE consumer example |
| `examples/stream_voice_agent.ail` | ~60 | Google ADK BIDI pattern |
| `examples/stream_openai_realtime.ail` | ~50 | OpenAI Realtime pattern |

**Modified files:**
| File | Changes | Purpose |
|------|---------|---------|
| `internal/effects/ops.go` | +1 line | Add `"Stream": {}` to Registry |
| `internal/effects/context.go` | +5 lines | Add `Stream *StreamContext` field |
| `cmd/ailang/main.go` | ~20 lines | CLI flags for Stream caps |
| `internal/runtime/config.go` | ~10 lines | Stream runtime config |
| `go.mod` | +1 dep | `gorilla/websocket` |
| `CHANGELOG.md` | ~20 lines | Document new feature |
| `README.md` | ~10 lines | Update implementation status |

**Total new code:** ~2,100 LOC (implementation) + ~1,000 LOC (tests)

## Examples

### Example 1: WebSocket Echo Client

```ailang
module examples/stream_websocket

import std/stream (connect, send, onEvent, runEventLoop, close, StreamConfig, StreamConn, StreamEvent, StreamMessage, StreamError)
import std/result (Result)
import std/io (println)

func main() -> unit ! {IO, Stream} {
  let config = {
    protocol: "websocket",
    headers: [],
    subprotocols: []
  };

  match connect("wss://echo.websocket.events", config) {
    Ok(conn) => {
      -- Send a message
      send(conn, Text("Hello from AILANG!"));

      -- Register event handler
      onEvent(conn, \event. match event {
        Opened(proto)       => { println("Connected! Protocol: " ++ proto); true },
        Message(data)       => { println("Received: " ++ data); false },
        Binary(data)        => { println("Binary received"); true },
        Closed(code, reason) => { println("Closed: " ++ intToString(code)); false },
        Error(err)          => { println("Error occurred"); false },
        Ping(_)             => true
      });

      -- Run event loop (blocks until handler returns false)
      runEventLoop(conn);
      close(conn)
    },
    Err(err) => match err {
      ConnectionFailed(msg) => println("Failed to connect: " ++ msg),
      DisallowedHost(host)  => println("Host blocked: " ++ host),
      _                     => println("Connection error")
    }
  }
}
```

### Example 2: Google ADK BIDI Voice Agent

```ailang
module examples/stream_voice_agent

import std/stream (connect, send, onEvent, runEventLoop, close, StreamConfig, StreamEvent, StreamMessage)
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
    protocol: "websocket",
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
          let parsed = decode(data);
          println("Server: " ++ data);
          true
        },
        Binary(audioBytes) => {
          -- Process audio chunk (PCM 24kHz)
          println("Audio chunk received");
          true
        },
        Closed(_, _) => false,
        Error(_)     => false,
        _            => true
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

import std/stream (connect, onEvent, runEventLoop, close, StreamConfig, StreamEvent)
import std/io (println)

func main() -> unit ! {IO, Stream} {
  let config = {
    protocol: "sse",
    headers: [{name: "Accept", value: "text/event-stream"}],
    subprotocols: []
  };

  match connect("https://example.com/events", config) {
    Ok(conn) => {
      let count = 0;
      onEvent(conn, \event. match event {
        Message(data) => {
          println("Event: " ++ data);
          true  -- Continue listening
        },
        Closed(_, _) => false,
        Error(_) => false,
        _ => true
      });
      runEventLoop(conn);
      close(conn)
    },
    Err(_) => println("Failed to connect to SSE endpoint")
  }
}
```

### Example 4: Budget-Limited Streaming

```ailang
module examples/stream_budget

import std/stream (connect, send, onEvent, runEventLoop, close, StreamConfig, StreamEvent, StreamMessage)
import std/io (println)

-- Run with: ailang run --caps "IO,Stream @limit=10" --entry main examples/stream_budget.ail
-- The Stream budget limits total stream operations (connect + send + receive)
-- After 10 operations, further calls return BudgetExhausted error

func main() -> unit ! {IO, Stream} {
  let config = {protocol: "websocket", headers: [], subprotocols: []};

  match connect("wss://echo.websocket.events", config) {
    Ok(conn) => {
      -- Each send counts against the budget
      send(conn, Text("Message 1"));
      send(conn, Text("Message 2"));
      send(conn, Text("Message 3"));

      onEvent(conn, \event. match event {
        Message(data)  => { println(data); true },
        Error(BudgetExhausted(msg)) => { println("Budget hit: " ++ msg); false },
        Closed(_, _)   => false,
        _              => true
      });

      runEventLoop(conn);
      close(conn)
    },
    Err(_) => println("Connection failed")
  }
}
```

## Success Criteria

- [ ] `_stream_connect` establishes WebSocket connections with full security validation
- [ ] `_stream_send` sends text and binary messages
- [ ] `_stream_onEvent` registers AILANG handler functions
- [ ] `_stream_runEventLoop` dispatches events to handlers correctly
- [ ] `_stream_close` performs graceful WebSocket close handshake
- [ ] `Stream` effect enforced via `--caps Stream`
- [ ] `Stream @limit=N` budget tracking works
- [ ] Domain allowlist (`--stream-allow-domains`) blocks unauthorized hosts
- [ ] TLS enforcement (`wss://` by default, `--stream-allow-http` for `ws://`)
- [ ] SSE protocol supported via `{protocol: "sse"}`
- [ ] Connection limits enforced (default: 4 concurrent)
- [ ] Message size limits enforced (default: 1MB)
- [ ] Idle timeout closes stale connections
- [ ] `ailang doctor stream` validates all stream builtins
- [ ] All 4 example files pass `make verify-examples`
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
- Message size limit (1MB default)
- Timeout configuration

**Builtin tests** (`internal/builtins/stream_test.go`):
- Type signature validation for all 6 builtins
- Argument count validation
- Effect tag verification (`[stream]`)
- Using `MockEffContext` for hermetic testing

**Event handling tests** (`internal/effects/stream_test.go`):
- Text message send/receive round-trip
- Binary message send/receive round-trip
- Handler registration and invocation
- Handler returning false stops event loop
- Connection close propagates Closed event
- Error propagation (timeout, budget, protocol)

### Integration Tests

**WebSocket integration** (test server in Go):
- Full connect → send → receive → close lifecycle
- Multiple concurrent connections (within limit)
- Binary frame handling (bytes round-trip)
- Subprotocol negotiation
- Reconnection after close

**SSE integration** (test server in Go):
- SSE event parsing (event, data, id, retry)
- Multi-line data events
- Automatic reconnection on connection drop

**Budget integration**:
- `Stream @limit=10` → 11th operation returns BudgetExhausted
- Budget shared across send and receive
- Budget trace events emitted

### Manual Testing

- [ ] Connect to `wss://echo.websocket.events` (public WebSocket echo)
- [ ] Connect to Google Gemini Live API (requires auth)
- [ ] Connect to SSE endpoint (httpbin or similar)
- [ ] Verify connection cleanup on program exit
- [ ] Verify timeout behavior (idle connection closes)

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

## Timeline

**Week 1** (40 hours):
- Phase 1: Core infrastructure (StreamContext, connection management, Go WebSocket)
- Add `gorilla/websocket` dependency
- Wire Stream effect into CLI

**Week 2** (40 hours):
- Phase 2: Builtins and AILANG module
- Register all 6 builtins
- Create `std/stream.ail`

**Week 3** (20 hours):
- Phase 3: SSE support
- Event parsing and protocol detection

**Week 4** (30 hours):
- Phase 4: Integration tests, examples, documentation
- 4 example files
- CHANGELOG/README updates

**Week 5** (30 hours):
- Phase 5: Hardening
- Security audit, stress testing, connection cleanup
- `ailang doctor stream` command

**Total: ~160 hours across 5 weeks**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Go goroutine leak (connections not cleaned up) | High | Connection registry with finalizer; `defer close()` pattern in examples; SIGTERM handler |
| WebSocket library compatibility | Medium | Use well-maintained `gorilla/websocket`; integration tests against real endpoints |
| Callback handler panics | High | Recover in event dispatch; propagate as `StreamError` to AILANG |
| Budget tracking across send+receive | Medium | Centralized counter in `StreamConnection`; test thoroughly |
| Idle timeout races with legitimate slow streams | Medium | Configurable idle timeout; reset on any activity |
| SSE reconnection complexity | Low | Phase 1: no auto-reconnect; document manual pattern |
| Binary data encoding over JSON setup messages | Low | `StreamMessage` ADT separates Text/Binary explicitly |
| Memory pressure from buffered events | Medium | Bounded event buffer (1000 events); back-pressure on slow handlers |

## Relationship to Existing Features

| Feature | Relationship |
|---------|-------------|
| `std/net` | Complementary — `std/net` for request-response, `std/stream` for persistent connections. Both use domain allowlists. |
| `std/json` | Composes — `encode`/`decode` used for structured WebSocket messages |
| `std/io` | Composes — `println` for logging stream events |
| Bytes type (v0.5.11) | Required — `Binary(bytes)` events use existing bytes infrastructure |
| Capability budgets (v0.6.2) | Integrates — `Stream @limit=N` uses existing budget mechanism |
| M-CSP-SESSION-TYPES (planned) | Future integration — `std/stream` Phase 2 can use channels internally |
| M-ARCH4 Executor Stream (planned) | Different layer — M-ARCH4 is internal tooling JSON parsing; this is language-level |
| `ailang serve-api` | Future — could later support WebSocket endpoints backed by `std/stream` |
| WASM target (future) | Compatible — browser WebSocket API could back `std/stream` in WASM |
| Trace system | Integrates — stream events recorded via `EffContext.Trace` |

## Related Documents

**Implemented (informs design):**
- [design_docs/implemented/v0_6_2/m-capability-budgets.md](../../implemented/v0_6_2/m-capability-budgets.md) — Resource-bounded effects pattern (`Stream @limit=N`)
- [design_docs/implemented/v0_0_3/initial_design.md](../../implemented/v0_0_3/initial_design.md) — Original effect system design (IO, FS, Clock, Net)

**Planned (related work):**
- [design_docs/planned/v0_8_1/m-csp-session-types.md](m-csp-session-types.md) — CSP channels (Phase 2 foundation)
- [design_docs/planned/v0_8_1/m-arch4-executor-stream-processor.md](m-arch4-executor-stream-processor.md) — Internal JSON stream processing (different layer)

**Axiom References:**
- [Design Axioms](/docs/references/axioms) — A3 (Effect Legibility), A4 (Explicit Authority), A8 (Minimal Syntax)

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
