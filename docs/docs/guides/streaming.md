---
sidebar_position: 6
title: Streaming & Real-Time I/O
description: WebSocket, SSE, and multi-source event multiplexing in AILANG
---

# Streaming & Real-Time I/O

AILANG's `Stream` effect provides event-driven I/O for real-time data: bidirectional WebSocket connections, Server-Sent Events (SSE), stdin line reading, and subprocess stdout streaming — all multiplexed through a single deterministic event loop.

:::tip See Streaming in Action
Browse the [Live Demos](https://www.sunholo.com/ailang-demos/) to see streaming effects in the browser:
- **[Claude Chat](https://www.sunholo.com/ailang-demos/streaming/claude_chat/)** — SSE streaming with Claude Messages API
- **[Gemini Live](https://www.sunholo.com/ailang-demos/streaming/gemini_live/)** — Bidirectional WebSocket audio with 30 voices
:::

## Quick Start

A minimal WebSocket echo client:

```typescript
module my/echo_client

import std/stream (connect, transmit, onEvent, runEventLoop, disconnect,
                   StreamEvent, Message, Closed, StreamError)
import std/result (Result, Ok, Err)

export func main() -> unit ! {Stream, IO} {
  match connect("wss://echo.websocket.org", {headers: []}) {
    Ok(conn) => {
      onEvent(conn, \event. match event {
        Message(msg) => { println("Received: " ++ msg); false },
        Closed(code, reason) => false,
        StreamError(err) => false,
        _ => true
      });
      transmit(conn, "Hello from AILANG!");
      runEventLoop(conn);
      disconnect(conn)
    },
    Err(e) => println("Connection failed")
  }
}
```

```bash
ailang run --caps Stream,IO --entry main echo_client.ail
```

## WebSocket API

Bidirectional WebSocket connections with typed events.

### Functions

| Function | Type | Description |
|----------|------|-------------|
| `connect` | `(string, StreamConfig) -> Result[StreamConn, StreamErrorKind] ! {Stream}` | Open WebSocket connection |
| `transmit` | `(StreamConn, string) -> Result[unit, StreamErrorKind] ! {Stream}` | Send text message |
| `transmitBinary` | `(StreamConn, bytes) -> Result[unit, StreamErrorKind] ! {Stream}` | Send binary data (no base64 overhead) |
| `onEvent` | `(StreamConn, (StreamEvent) -> bool) -> unit ! {Stream}` | Register event handler |
| `runEventLoop` | `(StreamConn) -> unit ! {Stream}` | Block until handler returns false or timeout |
| `disconnect` | `(StreamConn) -> unit ! {Stream}` | Close connection gracefully |
| `status` | `(StreamConn) -> StreamStatus ! {Stream}` | Get connection status |
| `withStream` | `(string, (StreamEvent) -> bool) -> Result[StreamConn, StreamErrorKind] ! {Stream}` | Connect + register + run + disconnect |

### Event Handler Pattern

The handler function receives `StreamEvent` and returns `bool`:
- Return `true` to continue receiving events
- Return `false` to exit the event loop

```typescript
func myHandler(event: StreamEvent) -> bool ! {IO} {
  match event {
    Message(text)      => { println("Text: " ++ text); true },
    Binary(data)       => { println("Binary frame received"); true },
    Opened(url)        => { println("Connected to " ++ url); true },
    Closed(code, reason) => { println("Closed: " ++ reason); false },
    StreamError(err)   => { println("Error"); false },
    Ping(data)         => true,
    _                  => true
  }
}
```

### Configuration

Custom headers for authentication:

```typescript
let config = {headers: [
  {name: "Authorization", value: "Bearer " ++ token},
  {name: "X-Custom", value: "value"}
]};
match connect("wss://api.example.com/ws", config) {
  Ok(conn) => { ... },
  Err(e) => println("Failed")
}
```

## SSE (Server-Sent Events)

Read-only HTTP streaming for consuming APIs.

### Functions

| Function | Type | Description |
|----------|------|-------------|
| `sseConnect` | `(string, StreamConfig) -> Result[StreamConn, StreamErrorKind] ! {Stream}` | Open SSE via HTTP GET |
| `ssePost` | `(string, string, StreamConfig) -> Result[StreamConn, StreamErrorKind] ! {Stream}` | Open SSE via HTTP POST (AI APIs) |
| `withSSE` | `(string, (StreamEvent) -> bool) -> Result[StreamConn, StreamErrorKind] ! {Stream}` | Connect + run + disconnect |

### SSE Events

SSE connections deliver events as `SSEData(eventType, data)`:

```typescript
import std/stream (ssePost, onEvent, runEventLoop, disconnect,
                   StreamEvent, SSEData, StreamError, Closed)

func handleSSE(event: StreamEvent) -> bool ! {IO} {
  match event {
    SSEData(eventType, data) => {
      println("[" ++ eventType ++ "] " ++ data);
      true
    },
    Closed(_, _) => false,
    StreamError(_) => false,
    _ => true
  }
}
```

### AI API Streaming

Claude, OpenAI, and Gemini all use POST+SSE:

```typescript
let body = "{\"model\": \"claude-sonnet-4-5-20250514\", \"messages\": [...]}";
let config = {headers: [
  {name: "Authorization", value: "Bearer " ++ apiKey},
  {name: "Content-Type", value: "application/json"},
  {name: "anthropic-version", value: "2023-06-01"}
]};

match ssePost("https://api.anthropic.com/v1/messages", body, config) {
  Ok(conn) => {
    onEvent(conn, handleSSE);
    runEventLoop(conn);
    disconnect(conn)
  },
  Err(e) => println("SSE connection failed")
}
```

## Multi-Source Event Multiplexing

*Added in v0.9.0 (M-ASYNC-IO)*

Read from multiple sources (WebSocket, stdin, subprocesses) in a single event loop with deterministic priority ordering.

### Functions

| Function | Type | Description |
|----------|------|-------------|
| `sourceOfConn` | `(StreamConn, string, int) -> StreamSource ! {Stream}` | Wrap a connection as named source |
| `asyncReadStdinLines` | `(string, int) -> StreamSource ! {Stream}` | Create stdin line reader source |
| `asyncExecProcess` | `(string, [string], string, int, int) -> StreamSource ! {Stream}` | Spawn subprocess, deliver stdout as chunks |
| `selectEvents` | `([StreamSource], (StreamEvent) -> bool) -> unit ! {Stream}` | Run priority-ordered event loop |

### How selectEvents Works

`selectEvents` dispatches events from multiple sources to a single handler:

1. **Priority-ordered check**: Higher-priority sources are checked first (deterministic)
2. **Round-robin within bands**: Same-priority sources rotate to prevent starvation
3. **Block when idle**: Uses `reflect.Select` when no events are ready
4. **Stops when**: Handler returns `false`, idle timeout, or max duration

### Source-Tagged Events

Multi-source events carry their source name:

| Event | Fields | Produced By |
|-------|--------|-------------|
| `SourceText(name, text)` | Source name + line text | `asyncReadStdinLines` |
| `SourceBytes(name, data)` | Source name + raw bytes | `asyncExecProcess`, `sourceOfConn` (binary) |
| `Message(text)` | Text content | `sourceOfConn` (WebSocket text frames) |

### Example: Stdin + WebSocket

```typescript
import std/stream (
  connect, sourceOfConn, asyncReadStdinLines, selectEvents,
  StreamEvent, Message, SourceText, SourceBytes, StreamError, Closed
)
import std/result (Ok, Err)

func handler(event: StreamEvent) -> bool ! {IO} {
  match event {
    SourceText(source, text) => { println("[" ++ source ++ "] " ++ text); true },
    Message(msg)             => { println("[ws] " ++ msg); true },
    Closed(_, _)             => false,
    StreamError(_)           => false,
    _                        => true
  }
}

export func main() -> unit ! {Stream, IO} {
  let stdin = asyncReadStdinLines("stdin", 10);
  selectEvents([stdin], handler)
}
```

### Subprocess Streaming (asyncExecProcess)

*Added in v0.9.0 (M-ASYNC-IO Phase 2)*

Spawn a subprocess and receive its stdout as fixed-size byte chunks via `SourceBytes` events. Useful for:
- **Audio capture**: `rec` / `arecord` producing PCM data
- **Screen capture**: `ffmpeg` producing video frames
- **Log tailing**: `tail -f` producing text lines
- **Any streaming subprocess**: Pipe output through `selectEvents`

```typescript
import std/stream (asyncExecProcess, asyncReadStdinLines, selectEvents,
                   StreamEvent, SourceText, SourceBytes)
import std/bytes (toString, toBase64)

func handler(event: StreamEvent) -> bool ! {IO} {
  match event {
    SourceBytes(source, data) => {
      println("[" ++ source ++ "] " ++ toString(data));
      true
    },
    SourceText(source, text) => {
      println("[" ++ source ++ "] " ++ text);
      false
    },
    _ => true
  }
}

export func main() -> unit ! {Stream, Process, IO} {
  -- Spawn subprocess: reads stdout in 4096-byte chunks
  let proc = asyncExecProcess("echo", ["hello", "world"], "echo", 5, 4096);
  let stdin = asyncReadStdinLines("stdin", 10);
  selectEvents([proc, stdin], handler)
}
```

**Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `cmd` | `string` | Command name (resolved via allowlist/PATH) |
| `args` | `[string]` | Arguments (no shell expansion) |
| `name` | `string` | Source name for `SourceBytes(name, data)` matching |
| `priority` | `int` | Dispatch priority (higher = checked first) |
| `chunkSize` | `int` | Bytes per event (e.g. 4096 for audio, 65536 for video) |

**Subprocess lifecycle:**
- Stdout is read in `chunkSize` chunks; final chunk may be smaller
- Subprocess is killed on `Close()`: SIGTERM, 5s grace period, then SIGKILL
- EOF from subprocess closes the source cleanly
- Stderr is discarded (not captured)

**Requires both capabilities:** `--caps Stream,Process`

### Subprocess Stdin Writing (spawnProcess)

*Added in v0.9.0 (M-ASYNC-IO Phase 3)*

Write data incrementally to a subprocess's stdin pipe. Complements `asyncExecProcess` (which reads stdout) — use `spawnProcess` when you need to **send data to** a subprocess rather than **read from** it.

Use cases:
- **Audio playback**: Pipe PCM bytes to `aplay` / `sox` / `ffplay`
- **Data processing**: Feed data to `jq`, `sed`, `awk` via stdin
- **Interprocess communication**: Write structured data to a child process

```typescript
import std/process (spawnProcess, writeProcessStdin, closeProcessStdin, ProcessHandle)
import std/bytes (fromString)
import std/result (Result, Ok, Err)

export func main() -> () ! {Process, IO} {
  -- Spawn cat with writable stdin (stdout goes to terminal)
  let handle = spawnProcess("cat", []);

  -- Write three lines to cat's stdin
  match writeProcessStdin(handle, fromString("hello from AILANG\n")) {
    Ok(_) => println("wrote line 1"),
    Err(e) => println("write error: " ++ e)
  };

  match writeProcessStdin(handle, fromString("streaming to subprocess\n")) {
    Ok(_) => println("wrote line 2"),
    Err(e) => println("write error: " ++ e)
  };

  -- Close stdin — cat will echo all lines and exit
  closeProcessStdin(handle)
}
```

```bash
ailang run --caps Process,IO --entry main examples/runnable/process_stdin_write.ail
```

**API:**

| Function | Type | Description |
|----------|------|-------------|
| `spawnProcess` | `(string, [string]) -> ProcessHandle ! {Process}` | Spawn subprocess with writable stdin pipe |
| `writeProcessStdin` | `(ProcessHandle, bytes) -> Result[(), string] ! {Process}` | Write bytes to subprocess stdin |
| `closeProcessStdin` | `(ProcessHandle) -> () ! {Process}` | Close stdin pipe (signals EOF) |

**ProcessHandle** is an opaque ADT: `ProcessHandle(int)`. Do not construct manually — always use `spawnProcess`.

**Write semantics:**
- Non-blocking: uses a 256-slot buffered write channel
- Returns `Ok(())` on success, `Err(reason)` if pipe closed or buffer full
- Subprocess receives bytes on its stdin as they are written

**Lifecycle:**
1. `spawnProcess(cmd, args)` — spawns subprocess, opens stdin pipe
2. `writeProcessStdin(handle, data)` — write bytes (repeatable)
3. `closeProcessStdin(handle)` — signals EOF, subprocess sees end-of-input
4. Subprocess cleanup: SIGTERM, 5s grace period, then SIGKILL if needed

**Requires:** `--caps Process` (does NOT require Stream capability)

:::tip asyncExecProcess vs spawnProcess
- **`asyncExecProcess`** = Read subprocess stdout via event loop (Stream + Process effects)
- **`spawnProcess`** = Write to subprocess stdin via ProcessHandle (Process effect only)

These are complementary — use `asyncExecProcess` to read, `spawnProcess` to write.
:::

### chunkSize Guidelines

| Use Case | Recommended chunkSize | Latency |
|----------|----------------------|---------|
| Audio (16kHz PCM) | 3200 (100ms frames) | ~100ms |
| Audio (44.1kHz PCM) | 4096 (~23ms frames) | ~23ms |
| Video frames | 65536 | frame-dependent |
| Text/log lines | 4096 | line-dependent |

## Types

### StreamConn

Opaque connection handle: `StreamConn(int)`

### StreamSource

Opaque event source handle: `StreamSource(int)`

### StreamEvent (ADT)

All events received by handlers:

| Variant | Fields | Description |
|---------|--------|-------------|
| `Message(string)` | Text content | WebSocket text frame |
| `Binary(string)` | Base64 data | WebSocket binary frame |
| `Opened(string)` | URL | Connection established |
| `Closed(int, string)` | Code, reason | Connection closed |
| `StreamError(StreamErrorKind)` | Error details | Error occurred |
| `Ping(string)` | Ping data | Keep-alive ping |
| `SSEData(string, string)` | Event type, data | Server-Sent Event |
| `SourceText(string, string)` | Source name, text | Line from stdin/text source |
| `SourceBytes(string, bytes)` | Source name, data | Chunk from subprocess/binary source |

### StreamErrorKind (ADT)

| Variant | Fields | Description |
|---------|--------|-------------|
| `ConnectionFailed(string)` | Reason | Could not connect |
| `Timeout(string)` | Details | Connection/idle timeout |
| `BudgetExhausted(string)` | Details | Capability budget exceeded |
| `ProtocolError(string)` | Details | WebSocket/SSE protocol error |
| `MessageTooLarge(string)` | Details | Message exceeds size limit |

### StreamConfig

Connection configuration: `{ headers: [StreamHeader] }`

### StreamHeader

Header entry: `{ name: string, value: string }`

### StreamStatus

Connection state: `Connecting | Open | Closing | StreamClosed`

## CLI Flags

```bash
# Grant Stream capability
ailang run --caps Stream --entry main module.ail

# Multiple capabilities (WebSocket + console output)
ailang run --caps Stream,IO --entry main module.ail

# Subprocess streaming (requires both Stream and Process)
ailang run --caps Stream,Process,IO --entry main module.ail

# With process allowlist (security)
ailang run --caps Stream,Process,IO --process-allowlist "echo,ffmpeg,rec" --entry main module.ail

# Stream timing configuration
ailang run --caps Stream,IO --stream-idle-timeout 30s --stream-max-duration 5m --entry main module.ail
```

## Related Resources

- [Effect System](/docs/reference/effects) — How effects work in AILANG
- [Process Effect](/docs/reference/effects#process-effect) — Command execution and security
- [Examples](/docs/examples) — Working AILANG programs
- [Capability Budgets](/docs/reference/capability-budgets) — Rate-limiting effects
