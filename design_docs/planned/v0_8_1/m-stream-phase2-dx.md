# M-STREAM-PHASE2-DX: Stream Typed ADTs & DX Improvements

**Status**: Phase 1 bug fixes shipped, Phase 2 (typed ADTs) planned
**Target**: v0.8.1
**Priority**: P0 (Blocker) - 6 demos, 12 files blocked on missing type exports
**Estimated**: 1 day (bug fixes done), 1 day (typed ADTs)
**Parent**: M-STREAM-BIDI (m-stream-bidi-primitives.md)
**Bug Reports**: msg_cbe8c22b (7 bugs), msg_5ba52e2d (transmit), msg_3d826344 (ADT types blocker)

## Problem Statement

The v0.8.0 `std/stream` module shipped with all types as `string` (Phase 1
simplification). This means:

1. **No pattern matching on events** — AILANG's core strength is unusable
2. **No type-checked function signatures** — wrong args silently fail at runtime
3. **6 demos blocked** — 12 files fail IMP010 trying to import StreamConn/StreamEvent/Message
4. **AI code generation broken** — LLMs naturally write `match event { Message(msg) => ... }` which fails

The infrastructure for exporting ADT types from stdlib modules **already exists**:
- `std/option.ail` exports `Option[a] = Some(a) | None` — works end-to-end
- `std/result.ail` exports `Result[a, e] = Ok(a) | Err(e)` — works end-to-end
- `std/net.ail` exports `NetError` and `HttpResponse` — works with builtins
- Pattern: define type in `.ail`, reference via `T.Con("TypeName")` in Go builtins

## Implemented Fixes (Phase 1 — done)

### Fix 1: extractConnID auto-unwrap (Bugs 1, 2, 3)
3-layer `extractConnID()`: StreamConn(int) | Ok(StreamConn(int)) | Err(...)
**File**: `internal/effects/stream.go`

### Fix 2: transmit auto-wrap (Bug 8)
Auto-wrap plain `StringValue` as `Text(string)` in `StreamSend`.
**File**: `internal/effects/stream.go`

### Fix 3: show(unit) rendering
Added `*eval.UnitValue` case returning `"()"`.
**File**: `internal/builtins/show.go`

### Fix 4: Documentation (Bugs 5, 6, 7)
Updated `std/stream.ail` header comments with all ADT types.
**File**: `std/stream.ail`

## Phase 2: Export Typed ADTs (THE MAIN DELIVERABLE)

### 2a: Add type declarations to `std/stream.ail`

Following the `std/net.ail` and `std/option.ail` patterns:

```ailang
import std/result (Result, Ok, Err)

-- Connection handle (opaque, holds integer ID)
export type StreamConn = StreamConn(int)

-- Stream event union — received by event handlers
export type StreamEvent =
  | Message(string)
  | Binary(string)
  | Opened(string)
  | Closed(int, string)
  | StreamError(StreamErrorKind)
  | Ping(string)
  | SSEData(string, string)

-- Stream error variants
export type StreamErrorKind =
  | ConnectionFailed(string)
  | Timeout(string)
  | BudgetExhausted(string)
  | ProtocolError(string)
  | MessageTooLarge(string)

-- Message type for transmit
export type StreamMessage = Text(string) | Bin(string)

-- Connection status
export type StreamStatus = Connecting | Open | Closing | Closed
```

**Notes on type simplifications for Phase 2:**
- `Binary(string)` not `Binary(bytes)` — bytes type may not unify across modules yet
- `Opened(string)` not `Opened({protocol, subprotocol})` — record types in ADTs are complex
- Can refine to richer types in Phase 3 once proven working

**Import from user code:**
```ailang
import std/stream (connect, withStream, StreamEvent, Message, Closed, StreamError)
import std/result (Result, Ok, Err)
```

### 2b: Update builtin type signatures in `internal/builtins/stream.go`

Change from `T.String()` to proper ADT references:

| Function | Current Return | New Return |
|----------|---------------|------------|
| `_stream_connect` | `string` | `Result[StreamConn, StreamErrorKind]` |
| `_stream_sse_connect` | `string` | `Result[StreamConn, StreamErrorKind]` |
| `_stream_send` | `string` | `Result[unit, StreamErrorKind]` |
| `_stream_status` | `string` | `StreamStatus` |
| `_stream_close` | `unit` | `unit` (unchanged) |
| `_stream_onEvent` | `unit` | `unit` (unchanged) |
| `_stream_runEventLoop` | `unit` | `unit` (unchanged) |

Handler type changes:
| Function | Current Handler | New Handler |
|----------|----------------|-------------|
| `_stream_onEvent` | `(string) -> bool` | `(StreamEvent) -> bool` |

Connection param changes:
| Function | Current Conn | New Conn |
|----------|-------------|----------|
| `_stream_send` | `string` | `StreamConn` |
| `_stream_onEvent` | `string` | `StreamConn` |
| `_stream_runEventLoop` | `string` | `StreamConn` |
| `_stream_close` | `string` | `StreamConn` |
| `_stream_status` | `string` | `StreamConn` |

**Implementation pattern** (from `std/net` builtins):
```go
func makeStreamConnectType() types.Type {
    T := types.NewBuilder()
    return T.Func(
        T.String(),            // url
        T.String(),            // config (still string for Phase 2)
    ).Returns(
        T.App("Result", T.Con("StreamConn"), T.Con("StreamErrorKind")),
    ).Effects("Stream")
}
```

### 2c: Update function signatures in `std/stream.ail`

```ailang
import std/result (Result, Ok, Err)

export func connect(url: string, config: string) -> Result[StreamConn, StreamErrorKind] ! {Stream}
export func transmit(conn: StreamConn, msg: string) -> Result[unit, StreamErrorKind] ! {Stream}
export func onEvent(conn: StreamConn, handler: (StreamEvent) -> bool) -> unit ! {Stream}
export func runEventLoop(conn: StreamConn) -> unit ! {Stream}
export func disconnect(conn: StreamConn) -> unit ! {Stream}
export func status(conn: StreamConn) -> StreamStatus ! {Stream}
export func withStream(url: string, handler: (StreamEvent) -> bool) -> Result[StreamConn, StreamErrorKind] ! {Stream}
export func withSSE(url: string, handler: (StreamEvent) -> bool) -> Result[StreamConn, StreamErrorKind] ! {Stream}
```

### 2d: Update Go runtime to match ADT names

The Go `eventToADT()` function in `internal/effects/stream.go` creates TaggedValues
with constructor names like `"Message"`, `"Opened"`, `"Error"` etc. These MUST match
the constructor names in the AILANG type declaration exactly.

Current `eventToADT` uses `"Error"` for the error case — rename to `"StreamError"`
to match the `StreamEvent` type declaration and avoid clashing with any future
stdlib `Error` type.

### 2e: Backward compatibility in extractConnID

The existing `extractConnID` auto-unwrap logic remains. With typed ADTs, the
type checker will catch misuse at compile time, but the runtime unwrap is still
needed for:
- `withStream`/`withSSE` which pass connect result directly
- Any user code that stores the Result and passes it later

## Phase 3: Future DX (nice-to-have)

### 3a: foldEvents — Stateful Event Accumulation

```ailang
export func foldEvents[a](conn: StreamConn, init: a, folder: (a, StreamEvent) -> {acc: a, continue: bool}) -> a ! {Stream}
```

New builtin `_stream_foldEvents` wrapping the event loop with Go-side accumulator.
~160 LOC total. Deferred until typed ADTs are proven working.

### 3b: Typed Config Records

Replace `defaultConfig` string with proper record type once ADT types are stable.

## Files to Modify (Phase 2)

| File | Change |
|------|--------|
| `std/stream.ail` | Add type declarations, update function signatures, import std/result |
| `internal/builtins/stream.go` | Update 7 type builder functions to use `T.Con()`/`T.App()` |
| `internal/effects/stream.go` | Rename `"Error"` to `"StreamError"` in `eventToADT` |
| `internal/effects/stream_test.go` | Update test assertions for renamed constructor |
| `internal/effects/stream_integration_test.go` | Update integration tests |
| `examples/runnable/stream_websocket.ail` | Update to use typed imports |
| `examples/runnable/stream_sse.ail` | Update to use typed imports |

## Verification

```bash
# Unit tests
go test ./internal/effects/ -run TestStream -v
go test ./internal/builtins/ -run TestStream -v

# Type checking examples
ailang check examples/runnable/stream_websocket.ail
ailang check examples/runnable/stream_sse.ail

# Full test suite
make test
make lint

# Verify imports work
echo 'import std/stream (StreamEvent, Message, Closed)' | ailang check --stdin
```

## Test Coverage (Phase 1 — done)

| Test | Status |
|------|--------|
| `TestExtractConnID` (Ok-wrapped) | PASS |
| `TestExtractConnID` (Err descriptive) | PASS |
| `TestExtractConnID` (Ok wrong inner/ctor) | PASS |
| `TestStreamIntegration_RawResultPassthrough` | PASS |
| `TestStreamSend_TextMessage` | PASS |
| All effects tests | PASS |
| show(unit) | PASS |
