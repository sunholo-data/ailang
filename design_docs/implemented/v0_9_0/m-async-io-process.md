# M-ASYNC-IO Phase 2: Subprocess Stdout as StreamSource (asyncExecProcess)

**Status**: Planned
**Target**: v0.9.0
**Priority**: P1 (Medium) — Completes the CLI ambient assistant pipeline
**Estimated**: 2 days (~8 hours implementation + 4 hours testing + 2 hours docs)
**Dependencies**: M-ASYNC-IO Phase 1 (implemented), M-PROCESS (implemented)
**Source**: msg_20260308_200203_2b080f07 (demos/ambient_assistant inbox)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Subprocess output timing is nondeterministic (same as existing Process effect). Deterministic within a recorded trace. No new category of nondeterminism beyond what `Process` + `Stream` already introduce. |
| A2: Replayability | +1 | Every chunk read from subprocess is traced with source tag, chunk size, and timestamp. Replay drives handler from recorded events. |
| A3: Effect Legibility | +1 | Requires **both** `Process` (to spawn subprocess) and `Stream` (to create StreamSource). Two effects, both explicit in signature. |
| A4: Explicit Authority | +1 | Subprocess spawning gated on `--caps Process` with allowlist. Source creation gated on `Stream` cap. No ambient access. |
| A5: Bounded Verification | 0 | No impact on local type checking |
| A6: Safe Concurrency | +1 | Same model as Phase 1: source goroutine writes to channel, `selectEventsLoop` reads. No shared mutable state. Subprocess killed on loop exit. |
| A7: Machines First | +1 | Structured `SourceBytes(name, bytes)` events — machine-parseable, not raw strings |
| A8: Minimal Syntax | +1 | No new syntax — function call returning `StreamSource`, uses existing ADT variants |
| A9: Cost Visibility | +1 | `chunkSize` parameter makes I/O granularity explicit. Process effect signals expensive operation. |
| A10: Composability | +1 | Composes with all existing sources in `selectEvents` — mic + stdin + WebSocket in one loop |
| A11: Structured Failure | +1 | Subprocess spawn failures use existing `ProcessError` ADT. Source close on process exit is clean. |
| A12: System Boundary | +1 | Explicit boundary: AILANG → subprocess → stdout bytes → `SourceBytes` event. Subprocess is a system boundary crossing. |

**Net Score: +10** → **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): Subprocess nondeterminism already accepted via Process effect; merge policy remains deterministic
- [x] A3 (Effects): Both `Process` and `Stream` effects declared in function signature
- [x] A4 (Authority): `--caps Process` required to spawn; `--caps Stream` to create source. Allowlist applies.
- [x] A7 (Machines First): Binary chunks delivered as `SourceBytes(name, bytes)` — structured, not raw

## Problem Statement

M-ASYNC-IO Phase 1 shipped `selectEvents` with `asyncReadStdinLines` (text) and `sourceOfConn` (WebSocket). The ambient assistant demo can now type commands while receiving WebSocket events. But the **real goal** is continuous audio/video streaming from system capture tools.

**Current State:**
- `std/process.exec()` runs a subprocess and waits for it to complete — **synchronous, blocks until exit**
- Cannot stream subprocess stdout into the `selectEvents` event loop
- Ambient assistant demo is text-only (no microphone, no screen capture)
- Browser demo handles full audio via MediaStream API, but CLI has no equivalent

**Impact:**
- CLI ambient assistant cannot use microphone (`rec`, `arecord`, `sox`)
- No screen capture for "what's on my screen?" queries (`ffmpeg -f avfoundation`)
- Any long-running subprocess that streams data (log tailing, sensor feeds) cannot be multiplexed with WebSocket

## Goals

**Primary Goal:** Enable AILANG programs to spawn a subprocess and receive its stdout as `SourceBytes` events in `selectEvents`, completing the multi-modal CLI pipeline.

**Success Metrics:**
- `asyncExecProcess("echo", ["hello"], "test", 5, 1024)` delivers `SourceBytes("test", <bytes>)` event
- `rec -q -r 16000 -c 1 -b 16 -t raw -` streams PCM audio chunks as `SourceBytes("mic", <4800 bytes>)`
- Subprocess is killed when `selectEvents` loop exits (handler returns false)
- Missing `--caps Process` returns capability error
- Process allowlist applies to subprocess commands
- All 3 source types compose in one `selectEvents` call (WebSocket + stdin + subprocess)

## Solution Design

### Overview

Add `asyncExecProcess` to `std/stream` — spawns a subprocess via `os/exec.Command`, reads stdout in fixed-size chunks via a background goroutine, and delivers chunks as `SourceBytes` events through the existing `EventSource` interface into `selectEventsLoop`.

This bridges `std/process` (synchronous exec) and `std/stream` (async event sources) without modifying either. The new function is a **streaming variant** of `exec` that returns a `StreamSource` instead of waiting for completion.

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
        │  stdin    │  │ WebSocket│   │  subprocess  │  ← NEW
        │  reader   │  │ readLoop │   │  stdout      │
        │ goroutine │  │ (exists) │   │  goroutine   │
        └──────────┘  └──────────┘   └──────────────┘
             ↑              ↑               ↑
        os.Stdin      net.Conn       exec.Command.Stdout
```

**Components:**

1. **`processSource`** (`internal/effects/stream_process.go`): Implements `EventSource` interface. Spawns subprocess, reads stdout in `chunkSize` increments, sends `SourceBytes` events to channel.
2. **`StreamAsyncExecProcess` handler** (`internal/effects/stream_async_ops.go`): Effect operation handler — validates args, resolves command via ProcessContext allowlist, creates `processSource`, stores in `StreamContext`.
3. **Builtin registration** (`internal/builtins/stream.go`): Register `_stream_async_exec_process` with type signature.
4. **Stdlib wrapper** (`std/stream.ail`): Export `asyncExecProcess` function.

### Capability Design

**Requires BOTH capabilities:** `! {Process, Stream}`

- `Process` — authority to spawn external commands (reuses existing allowlist, timeout, output limits)
- `Stream` — authority to create event sources and run `selectEvents`

This is the correct design: spawning a subprocess is a `Process` authority. Wrapping its output as a `StreamSource` is a `Stream` authority. The function crosses both boundaries.

### Type Signature

```ailang
asyncExecProcess : (
  cmd: string,
  args: [string],
  name: string,
  priority: int,
  chunkSize: int
) -> StreamSource ! {Process, Stream}
```

**Parameters:**
| Parameter | Type | Purpose |
|-----------|------|---------|
| `cmd` | `string` | Command to execute (resolved via allowlist/PATH) |
| `args` | `[string]` | Arguments (no shell expansion — same security as `exec`) |
| `name` | `string` | Source name for `SourceBytes(name, data)` matching |
| `priority` | `int` | Dispatch priority in `selectEvents` (higher = checked first) |
| `chunkSize` | `int` | Bytes per `SourceBytes` event (determines latency) |

### Subprocess Lifecycle

```
asyncExecProcess("rec", [...], "mic", 8, 4800)
  │
  ├── 1. Validate command via ProcessContext allowlist (same as exec())
  ├── 2. exec.CommandContext(ctx, cmd, args...) — context tied to selectEvents lifecycle
  ├── 3. cmd.StdoutPipe() → io.ReadCloser
  ├── 4. cmd.Start() — subprocess begins
  ├── 5. Goroutine: loop { read(stdout, chunkSize) → channel ← SourceBytes event }
  │       ├── EOF → close channel → source removed from selectEvents
  │       └── error → StreamError event → close channel
  ├── 6. selectEvents dispatches SourceBytes events to handler
  └── 7. On loop exit (handler returns false / all sources closed):
          ├── ctx.Cancel() → sends SIGTERM to subprocess
          ├── Wait with 5s grace period
          └── SIGKILL if still running
```

**Critical: subprocess cleanup.** The `processSource.Close()` method must:
1. Cancel the context (sends SIGTERM to subprocess)
2. Wait up to 5 seconds for clean exit
3. Force-kill (SIGKILL) if subprocess is still running
4. Close the stdout pipe
5. Wait for goroutine to exit

This prevents zombie processes when the handler returns `false` (e.g., user typed "quit").

### Stderr Handling

**Design decision: stderr is discarded (not captured) in streaming mode.**

Rationale:
- `asyncExecProcess` is for **continuous stdout streaming** (audio PCM, video frames, log lines)
- Capturing stderr as a separate source adds complexity with little value for the primary use case
- If stderr matters, use synchronous `exec()` which captures both
- Audio tools like `rec`/`ffmpeg` produce diagnostic output on stderr that is noise for the streaming pipeline

If needed in the future, add `asyncExecProcessWithStderr` that returns two sources.

### chunkSize Design

The `chunkSize` parameter directly controls **latency** for real-time streaming:

| Use Case | chunkSize | Latency | Calculation |
|----------|-----------|---------|-------------|
| Mic audio (16kHz 16-bit mono) | 4800 bytes | 150ms | 16000 Hz × 2 bytes × 0.15s |
| Mic audio (lower latency) | 1600 bytes | 50ms | 16000 Hz × 2 bytes × 0.05s |
| JPEG screen capture | 65536 bytes | N/A | One frame per read (frames are discrete) |
| Log tailing | 4096 bytes | N/A | Buffer accumulation |

**Partial reads:** The goroutine uses `io.ReadFull` semantics — it blocks until `chunkSize` bytes are available or the pipe reaches EOF. Partial final chunks (less than `chunkSize`) are delivered as-is.

### Implementation Plan

**Phase 1: processSource + handler** (~4 hours)
- [ ] Create `internal/effects/stream_process.go`:
  - `processSource` struct implementing `EventSource`
  - `NewProcessSource(ctx, cmd, args, name, priority, chunkSize)` constructor
  - Background goroutine: `io.ReadFull(stdout, buf)` → `SourceBytes` events
  - `Close()` method with SIGTERM → grace period → SIGKILL
- [ ] Add `StreamAsyncExecProcess` handler to `stream_async_ops.go`:
  - Validate args (cmd, args, name, priority, chunkSize)
  - Check ProcessContext allowlist (reuse existing `ResolveCommand` logic)
  - Create `processSource`, store in `StreamContext.sources`
  - Return `StreamSource(id)`
- [ ] Register `"asyncExecProcess"` operation in `stream.go` init()

**Phase 2: Builtin + stdlib + tests** (~4 hours)
- [ ] Register `_stream_async_exec_process` builtin in `internal/builtins/stream.go`
- [ ] Update `std/stream.ail` with `asyncExecProcess` export
- [ ] Update golden snapshot
- [ ] Unit tests in `stream_process_test.go`:
  - processSource reads echo output as SourceBytes
  - chunkSize respected (partial final chunk)
  - Process killed on source Close()
  - Allowlist enforcement
  - EOF → clean source close → selectEvents continues with other sources
  - Error propagation (command not found)

**Phase 3: Example + integration + docs** (~3 hours)
- [ ] Create `examples/runnable/stream_process_source.ail` — demo with `echo` or `cat`
- [ ] Integration test: subprocess source + stdin source in `selectEvents`
- [ ] Update CHANGELOG
- [ ] Update design doc status

### Files to Modify/Create

**New files:**
- `internal/effects/stream_process.go` — `processSource` EventSource (~120 LOC)
- `internal/effects/stream_process_test.go` — Tests (~180 LOC)
- `examples/runnable/stream_process_source.ail` — Example (~40 LOC)

**Modified files:**
- `internal/effects/stream_async_ops.go` — Add `StreamAsyncExecProcess` handler (~50 LOC)
- `internal/effects/stream.go` — Add `RegisterOp` for asyncExecProcess (~1 line)
- `internal/builtins/stream.go` — Register `_stream_async_exec_process` (~50 LOC)
- `std/stream.ail` — Export `asyncExecProcess` function (~5 LOC)
- `internal/pipeline/testdata/builtin_types.golden` — Update (183 builtins)

**Estimated total:** ~450 LOC (implementation + tests)

## Examples

### Example 1: Mic Capture → Gemini Live

```ailang
module demos/ambient_mic

import std/stream (
  connect, transmit, sourceOfConn, asyncReadStdinLines,
  asyncExecProcess, selectEvents,
  StreamConn, StreamSource, StreamEvent,
  Message, SourceText, SourceBytes, Closed
)
import std/bytes (toBase64)
import std/result (Result, Ok, Err)

-- Build Gemini realtimeInput audio chunk
func buildAudioChunk(b64: string) -> string {
  "{\"realtimeInput\":{\"mediaChunks\":[{\"mimeType\":\"audio/pcm;rate=16000\",\"data\":\"" ++ b64 ++ "\"}]}}"
}

export func main() -> unit ! {Stream, Process, IO} {
  let config = defaultConfig(());
  match connect("wss://generativelanguage.googleapis.com/...", config) {
    Ok(conn) => {
      let wsSrc = sourceOfConn(conn, "gemini", 10);
      let stdinSrc = asyncReadStdinLines("stdin", 5);
      -- Mic: 16kHz 16-bit mono PCM, 150ms chunks
      let micSrc = asyncExecProcess(
        "rec", ["-q", "-r", "16000", "-c", "1", "-b", "16", "-t", "raw", "-"],
        "mic", 8, 4800
      );
      selectEvents([wsSrc, stdinSrc, micSrc], \event.
        match event {
          SourceBytes("mic", pcm) => {
            transmit(conn, buildAudioChunk(toBase64(pcm)));
            true
          },
          SourceText("stdin", "quit") => false,
          SourceText("stdin", line) => {
            transmit(conn, "{\"clientContent\":{\"turns\":[{\"parts\":[{\"text\":\"" ++ line ++ "\"}]}]}}");
            true
          },
          Message(msg) => { _io_println("gemini: " ++ msg); true },
          Closed(_, _) => false,
          _ => true
        });
      disconnect(conn)
    },
    Err(_) => _io_println("connection failed")
  }
}
```

### Example 2: Screen Capture

```ailang
-- Capture screen at 0.5 FPS as JPEG frames
let screenSrc = asyncExecProcess(
  "ffmpeg",
  ["-f", "avfoundation", "-i", "1", "-r", "0.5", "-f", "image2pipe", "-vcodec", "mjpeg", "-"],
  "screen", 6, 65536
);
```

### Example 3: Log Tailing

```ailang
-- Tail a log file as a streaming source
let logSrc = asyncExecProcess("tail", ["-f", "/var/log/system.log"], "syslog", 3, 4096);

selectEvents([logSrc, stdinSrc], \event.
  match event {
    SourceBytes("syslog", chunk) => {
      _io_println("log: " ++ bytesToString(chunk));
      true
    },
    SourceText("stdin", "quit") => false,
    _ => true
  })
```

## Success Criteria

- [ ] `asyncExecProcess("echo", ["hello"], "test", 5, 1024)` delivers `SourceBytes("test", <bytes>)` in selectEvents
- [ ] Subprocess stdout is read in `chunkSize` increments
- [ ] Partial final chunk delivered (less than `chunkSize` at EOF)
- [ ] Subprocess killed (SIGTERM → SIGKILL) when selectEvents loop exits
- [ ] Missing `--caps Process` produces capability error
- [ ] Process allowlist applies to `asyncExecProcess` commands
- [ ] Composes with `asyncReadStdinLines` + `sourceOfConn` in same `selectEvents` call
- [ ] No zombie processes after loop exit
- [ ] All tests passing
- [ ] Documentation updated
- [ ] Example added

## Testing Strategy

**Unit tests:**
- `processSource` with `echo "hello"` → delivers SourceBytes with correct data
- `chunkSize` enforcement: large output split into correct chunk sizes
- Partial final chunk: output smaller than chunkSize delivered as-is
- `Close()` kills subprocess (verify process no longer running)
- Command not found → error (ProcessContext validation)
- Allowlist blocking → error
- EOF from subprocess → source closes cleanly → selectEvents continues with remaining sources

**Integration tests:**
- Subprocess source + stdin source in `selectEvents` — both event types arrive
- Priority ordering: high-priority subprocess over low-priority stdin
- Handler returning false → subprocess killed, no zombie

**Manual testing:**
- `echo "test data" | ailang run --caps Stream,Process,IO --entry main examples/runnable/stream_process_source.ail`
- Verify mic capture with `rec` (macOS: `brew install sox`)

## Non-Goals

**Not in this feature:**
- **Stderr capture as a source** — Adds complexity; use synchronous `exec()` if stderr matters. Future: `asyncExecProcessWithStderr` returning two sources.
- **Stdin piping to subprocess** — Subprocess receives no input on stdin (pipe is closed). Future: `asyncExecProcessWithInput(cmd, args, inputSource)`.
- **Restart on exit** — Subprocess exits once, source closes. Reconnection/restart is application logic.
- **Windows support** — `SIGTERM`/`SIGKILL` lifecycle is Unix-specific. Windows would need `taskkill`. Deferred.
- **Shell expansion** — Same security model as `exec()`: no `sh -c`, args are passed literally.
- **Custom environment/working directory** — Inherits parent process environment, uses ProcessContext sandbox if set.

## Timeline

**Day 1** (~6 hours):
- Phase 1: `processSource` struct, goroutine, cleanup
- Phase 2: Builtin registration, stdlib update

**Day 2** (~5 hours):
- Phase 2 continued: Tests
- Phase 3: Example, integration test, docs

**Total: ~11 hours across 2 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Zombie processes on unclean exit | High | Context cancellation + SIGTERM + 5s grace + SIGKILL. `Close()` always called by `selectEventsLoop` cleanup. |
| Subprocess blocks on stderr (pipe full) | Medium | Stderr is not captured (redirected to /dev/null). No pipe to fill. |
| `io.ReadFull` blocks indefinitely on slow subprocess | Medium | Context cancellation from `selectEventsLoop` timeout/exit propagates to read via pipe close. |
| Audio latency too high with large chunkSize | Low | User controls chunkSize. Document recommended values per use case. |
| Platform-specific audio tools missing | Low | `--process-allowlist` fails at startup with clear error. Example uses cross-platform `echo` for CI. |
| WASM build breakage | Low | `processSource` file gets `//go:build !js` tag (same as `process.go`). |

## Related Documents

**Implemented (informs design):**
- [M-ASYNC-IO Phase 1](design_docs/implemented/v0_9_0/m-async-io-stream.md) — EventSource interface, selectEventsLoop, asyncReadStdinLines (v0.9.0)
- [M-PROCESS](design_docs/implemented/v0_8_1/m-process-exec.md) — Synchronous exec, ProcessContext, allowlist, ProcessError ADT (v0.8.1)
- [M-STREAM-BIDI](design_docs/implemented/v0_8_1/m-stream-bidi-primitives.md) — WebSocket streaming, StreamConnection, eventBuffer pattern (v0.8.1)

**Planned (check for overlap):**
- [M-CSP-SESSION-TYPES](design_docs/planned/v1_0_0/m-csp-session-types.md) — Full CSP concurrency (v1.0.0, `selectEvents` is a stepping stone)
- [M-ARCH4 Executor Stream Processor](design_docs/planned/v0_9_0/m-arch4-executor-stream-processor.md) — Related stream processing architecture

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- Source message: msg_20260308_200203_2b080f07 (demos/ambient_assistant inbox)
- `std/bytes.toBase64` — Already exists, no new builtin needed
- `std/process.exec` — Synchronous exec (ProcessContext, allowlist reused)
- `internal/effects/stream_source.go` — EventSource interface (Phase 1)
- `internal/effects/stream_mux.go` — selectEventsLoop (Phase 1)

## Future Work

- **Stdin piping to subprocess**: `asyncExecProcessWithInput(cmd, args, inputSource)` — enables bidirectional subprocess I/O
- **Stderr as separate source**: `asyncExecProcessFull(cmd, args, ...) -> (StreamSource, StreamSource)` — stdout and stderr as independent sources
- **Restart policy**: Auto-restart subprocess on exit with backoff (for persistent capture sessions)
- **Process budget integration**: `Process @limit=N` to bound total subprocess count
- **Timer source**: `asyncTimer(interval, name, priority)` — periodic tick events for heartbeats/polling
- **File watcher source**: `asyncWatchFile(path, name, priority)` — inotify/FSEvents as StreamSource

---

**Document created**: 2026-03-08
**Last updated**: 2026-03-08
