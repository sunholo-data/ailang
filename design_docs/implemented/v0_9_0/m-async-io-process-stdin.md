# M-ASYNC-IO Phase 3: Subprocess Stdin Writing (ProcessHandle)

**Status**: IMPLEMENTED
**Target**: v0.9.0
**Priority**: P1 (Medium) — Completes bidirectional subprocess I/O for ambient assistant
**Estimated**: 1.5 days (~6 hours implementation + 3 hours testing + 2 hours docs)
**Dependencies**: M-ASYNC-IO Phase 2 (implemented), M-PROCESS (implemented)
**Source**: msg_20260308_210308_35dab31c (ambient-assistant-demo inbox)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Subprocess stdin timing is nondeterministic (same as existing Process effect). Write ordering within a single handler turn IS deterministic. |
| A2: Replayability | +1 | Every write traced with handle ID, byte count, and timestamp. Replay drives writes from recorded events. |
| A3: Effect Legibility | +1 | Requires `Process` capability. Stdin writing is a subprocess operation, not a stream source operation. |
| A4: Explicit Authority | +1 | Subprocess spawning gated on `--caps Process` with allowlist. No ambient access. |
| A5: Bounded Verification | 0 | No impact on local type checking |
| A6: Safe Concurrency | +1 | Writes are serialized through a buffered channel — no concurrent pipe access. Goroutine drains queue. |
| A7: Machines First | +1 | Opaque `ProcessHandle(int)` ADT — machine-manipulable, not human-oriented |
| A8: Minimal Syntax | +1 | No new syntax — three function calls using existing ADT pattern |
| A9: Cost Visibility | +1 | `Process` effect signals expensive operation. Write sizes are explicit (bytes parameter). |
| A10: Composability | +1 | Composes with `asyncExecProcess` — read stdout events in `selectEvents` while writing to a different process's stdin. |
| A11: Structured Failure | +1 | Returns `Result[(), string]` — typed success/failure. Closed pipe, broken process = `Err(reason)`. |
| A12: System Boundary | +1 | Explicit boundary: AILANG bytes → subprocess stdin pipe. ProcessHandle is the boundary marker. |

**Net Score: +10** → **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): Write ordering is deterministic within a handler turn; I/O timing accepted via Process effect
- [x] A3 (Effects): `Process` capability required for all operations
- [x] A4 (Authority): `--caps Process` required; allowlist applies to spawn
- [x] A7 (Machines First): Opaque handle ADT, not raw file descriptor

## Problem Statement

M-ASYNC-IO Phase 2 shipped `asyncExecProcess` which streams subprocess **stdout** as `SourceBytes` events into `selectEvents`. The ambient assistant demo can now capture microphone audio via `sox rec → SourceBytes`. But **playback** requires writing audio frames TO a subprocess's stdin — the missing counterpart.

**Current State:**
- `asyncExecProcess` reads subprocess stdout → `SourceBytes` events (Phase 2, implemented)
- `std/process.exec()` runs a subprocess synchronously and waits for completion — cannot stream stdin
- Ambient assistant accumulates ALL audio frames, writes to file, then blocks on `exec("afplay", [wavPath])`
- This accumulate-then-play pattern adds noticeable latency (~2-5s depending on response length)

**Impact:**
- Real-time audio playback impossible — must wait for full response before any audio plays
- Cannot stream PCM frames to `sox -d` as they arrive from Gemini
- Any use case requiring incremental writes to a long-running subprocess (audio, video encoding, data pipelines) is blocked

## Goals

**Primary Goal:** Enable AILANG programs to spawn a subprocess and write bytes to its stdin incrementally, completing the bidirectional CLI pipeline for the ambient assistant.

**Success Metrics:**
- `spawnProcess("cat", [])` spawns a subprocess and returns a `ProcessHandle`
- `writeProcessStdin(handle, bytes)` writes bytes to the subprocess's stdin pipe
- `closeProcessStdin(handle)` closes stdin (signals EOF to subprocess)
- Subprocess is killed when handle is closed or program exits
- Missing `--caps Process` returns capability error
- Process allowlist applies to `spawnProcess` commands
- No zombie processes after handle close

## Design Decision: ProcessHandle over StreamSource

The feature request proposed two API designs. We choose **Option B: ProcessHandle** for these reasons:

1. **Separation of concerns.** Writing to stdin is *output* to the process, not *input* to the event loop. `StreamSource` represents event ingress; `ProcessHandle` represents command egress.

2. **Effect clarity.** `StreamSource` operations require `{Stream}` — but stdin writing is purely a `{Process}` operation. Forcing `{Stream}` would be misleading.

3. **Simpler lifecycle.** A `ProcessHandle` is fire-and-forget on the write side — write bytes, close when done. No event loop integration needed. The subprocess reads stdin at its own pace.

4. **Future composability.** A bidirectional variant (`asyncExecProcessBidi → (StreamSource, ProcessHandle)`) can layer both abstractions cleanly. Neither needs to know about the other.

5. **No stdout capture.** The primary use case (audio playback via `sox -d`) doesn't need stdout events. A lightweight handle that just opens a stdin pipe is sufficient.

## Solution Design

### Overview

Add three new builtins to `std/process` that manage a long-running subprocess with a writable stdin pipe:

```ailang
spawnProcess(cmd, args) -> ProcessHandle ! {Process}
writeProcessStdin(handle, data) -> Result[(), string] ! {Process}
closeProcessStdin(handle) -> () ! {Process}
```

The subprocess is spawned immediately, its stdin pipe is held open, and writes are buffered through a Go channel drained by a background goroutine. The subprocess is killed on `closeProcessStdin` or program exit.

### Architecture

```
                        AILANG program
                             │
          ┌──────────────────┼──────────────────────┐
          ▼                  ▼                       ▼
   asyncExecProcess    spawnProcess           selectEvents
   (stdout → events)  (stdin ← writes)       (event loop)
          │                  │                       │
    ┌─────┴─────┐     ┌─────┴──────┐          ┌─────┴─────┐
    │ process   │     │ managed    │          │  mux      │
    │ Source    │     │ Process    │          │  loop     │
    │ (stdout)  │     │ (stdin)    │          │           │
    └─────┬─────┘     └─────┬──────┘          └───────────┘
          │                 │
    exec.Cmd.Stdout   exec.Cmd.Stdin
          │                 │
       ┌──┴─────────────────┴──┐
       │      subprocess       │
       │   (e.g. sox -d)       │
       └───────────────────────┘
```

**Components:**

1. **`managedProcess`** (`internal/effects/process_managed.go`): Holds `exec.Cmd`, stdin pipe, write channel, goroutine. Implements write buffering and clean shutdown.
2. **`ProcessSpawn` handler** (`internal/effects/process_spawn.go`): Effect operation — validates args, resolves via allowlist, creates `managedProcess`, stores in `ProcessContext`.
3. **`ProcessWriteStdin` handler**: Writes bytes to the managed process's stdin channel.
4. **`ProcessCloseStdin` handler**: Closes stdin pipe, waits for subprocess exit.
5. **Builtin registration** (`internal/builtins/process.go`): Register three new builtins.
6. **Stdlib wrapper** (`std/process.ail`): Export three functions + `ProcessHandle` type.

### Type Signatures

```ailang
-- ADT for opaque process handle
export type ProcessHandle = ProcessHandle(int)

-- Spawn a subprocess with writable stdin (stdout/stderr discarded)
export func spawnProcess(cmd: string, args: [string]) -> ProcessHandle ! {Process}

-- Write bytes to subprocess stdin (buffered, non-blocking)
export func writeProcessStdin(handle: ProcessHandle, data: bytes) -> Result[(), string] ! {Process}

-- Close stdin pipe (signals EOF to subprocess) and wait for exit
export func closeProcessStdin(handle: ProcessHandle) -> () ! {Process}
```

**Parameters:**

| Function | Parameter | Type | Purpose |
|----------|-----------|------|---------|
| `spawnProcess` | `cmd` | `string` | Command to execute (resolved via allowlist/PATH) |
| `spawnProcess` | `args` | `[string]` | Arguments (no shell expansion) |
| `writeProcessStdin` | `handle` | `ProcessHandle` | Opaque handle from `spawnProcess` |
| `writeProcessStdin` | `data` | `bytes` | Raw bytes to write to stdin pipe |
| `closeProcessStdin` | `handle` | `ProcessHandle` | Handle to close |

### Why `Result[(), string]` for writes

Writes can fail if:
- The subprocess has exited (broken pipe)
- The stdin pipe was already closed
- The write buffer is full and subprocess is not consuming (backpressure)

Returning `Result` lets the caller decide: ignore failures (fire-and-forget audio) or handle them (data pipeline integrity).

### managedProcess Lifecycle

```
spawnProcess("sox", ["-t", "raw", ..., "-", "-d"])
  │
  ├── 1. Validate command via ProcessContext allowlist (same as exec())
  ├── 2. exec.Command(cmd, args...)
  ├── 3. cmd.StdinPipe() → io.WriteCloser
  ├── 4. cmd.Stdout = nil, cmd.Stderr = nil (discarded)
  ├── 5. cmd.Start() — subprocess begins, reads stdin
  ├── 6. Start writeLoop goroutine (drains write channel → stdin pipe)
  ├── 7. Store in ProcessContext.managed map, return ProcessHandle(id)
  │
  ▼ (in selectEvents handler, when audio frame arrives)
  │
writeProcessStdin(handle, pcmData)
  ├── 1. Lookup managedProcess by handle ID
  ├── 2. Send pcmData to write channel (buffered, non-blocking with backpressure)
  ├── 3. writeLoop goroutine writes to stdin pipe
  └── 4. Return Ok(()) or Err("pipe closed") / Err("process exited")
  │
  ▼ (on turn complete or session end)
  │
closeProcessStdin(handle)
  ├── 1. Close write channel (signals writeLoop to stop)
  ├── 2. writeLoop drains remaining buffered writes
  ├── 3. Close stdin pipe (subprocess sees EOF)
  ├── 4. Wait for subprocess exit (5s grace → SIGKILL)
  └── 5. Remove from ProcessContext.managed map
```

### Write Buffering Design

```go
type managedProcess struct {
    id       int
    cmd      *exec.Cmd
    stdin    io.WriteCloser
    writeCh  chan []byte        // Buffered write channel (capacity: 256)
    done     chan struct{}      // Signals subprocess exited
    once     sync.Once         // Idempotent close
    cancel   context.CancelFunc
    mu       sync.Mutex
    closed   bool              // Stdin pipe closed
    exitErr  error             // Subprocess exit error (if any)
}
```

**Write channel capacity: 256 buffers.**

For audio at 24kHz 16-bit mono with 150ms chunks (7200 bytes each):
- 256 × 7200 = ~1.8 MB buffered
- At 48 KB/s audio rate, that's ~38 seconds of buffering
- More than enough to absorb Gemini response bursts

**Backpressure:** If the channel is full (subprocess not consuming), `writeProcessStdin` returns `Err("write buffer full — subprocess may be stalled")`. This is a signal to the caller, not a silent drop.

### Subprocess stdout/stderr

**Both discarded.** The `spawnProcess` use case is write-only (audio playback, data feeding). If you need stdout, use `asyncExecProcess` (Phase 2). If you need both directions on the same process, that's a future Phase 4 (`asyncExecProcessBidi`).

### ProcessContext Extension

```go
// Added to ProcessContext:
type ProcessContext struct {
    // ... existing fields ...

    // Managed process tracking (M-ASYNC-IO Phase 3)
    mu             sync.Mutex
    managed        map[int]*managedProcess
    nextManagedID  int
}
```

New methods:
- `AcquireManagedProcess(mp *managedProcess) int` — register, return ID
- `GetManagedProcess(id int) (*managedProcess, bool)` — lookup by ID
- `ReleaseManagedProcess(id int)` — remove from tracking
- `CloseAllManaged()` — kill all managed processes (graceful shutdown)

### Cleanup on Program Exit

`CloseAllManaged()` is called during effect context teardown (same path as `StreamContext.CloseAll()`). This ensures no zombie processes if the AILANG program exits without calling `closeProcessStdin`.

### Implementation Plan

**Phase 1: managedProcess + handlers** (~4 hours)
- [ ] Create `internal/effects/process_managed.go`:
  - `managedProcess` struct with write channel, writeLoop goroutine
  - `NewManagedProcess(ctx, cmdPath, args)` constructor
  - `Write(data []byte) error` — send to write channel
  - `CloseStdin()` — close channel, drain, close pipe, wait
  - `Close()` — idempotent full cleanup (SIGTERM → SIGKILL)
- [ ] Add `ProcessContext` extensions:
  - `managed` map, `AcquireManagedProcess`, `GetManagedProcess`, `ReleaseManagedProcess`, `CloseAllManaged`
- [ ] Create `internal/effects/process_spawn.go`:
  - `ProcessSpawn(ctx, args)` — validates, resolves command, creates managedProcess
  - `ProcessWriteStdin(ctx, args)` — writes bytes to managed process
  - `ProcessCloseStdin(ctx, args)` — closes stdin, waits for exit
- [ ] Register operations in effect dispatcher

**Phase 2: Builtin + stdlib + tests** (~3 hours)
- [ ] Register builtins in `internal/builtins/process.go`:
  - `_process_spawn_process`
  - `_process_write_stdin`
  - `_process_close_stdin`
- [ ] Add `ProcessHandle` ADT and exports to `std/process.ail`
- [ ] Update golden snapshot
- [ ] Unit tests in `process_managed_test.go`:
  - spawnProcess + writeProcessStdin + closeProcessStdin basic flow
  - Write to closed pipe returns Err
  - Process killed on close
  - Allowlist enforcement
  - Buffer full backpressure
  - CloseAllManaged on teardown

**Phase 3: Example + docs** (~2 hours)
- [ ] Create `examples/runnable/process_stdin_write.ail` — demo with `cat` echo
- [ ] Update CHANGELOG
- [ ] Update design doc status
- [ ] Update CLAUDE.md if needed

### Files to Modify/Create

**New files:**
- `internal/effects/process_managed.go` — `managedProcess` + writeLoop (~130 LOC)
- `internal/effects/process_spawn.go` — Effect handlers (~150 LOC)
- `internal/effects/process_managed_test.go` — Tests (~200 LOC)
- `examples/runnable/process_stdin_write.ail` — Example (~30 LOC)

**Modified files:**
- `internal/effects/process_context.go` — Add managed process tracking (~40 LOC)
- `internal/builtins/process.go` — Register 3 new builtins (~60 LOC)
- `std/process.ail` — Export ProcessHandle + 3 functions (~15 LOC)
- `internal/effects/context.go` — CloseAllManaged in teardown (~3 LOC)
- `internal/pipeline/testdata/builtin_types.golden` — Update snapshot

**Estimated total:** ~630 LOC (implementation + tests)

## Examples

### Example 1: Streaming Audio Playback (Ambient Assistant)

```ailang
module demos/ambient_speaker

import std/stream (
  connect, sourceOfConn, asyncExecProcess, selectEvents,
  StreamSource, StreamEvent, Message, SourceBytes, Closed
)
import std/process (spawnProcess, writeProcessStdin, closeProcessStdin, ProcessHandle)
import std/bytes (fromBase64)
import std/json (decode)
import std/result (Result, Ok, Err)

-- Spawn sox for real-time PCM playback (24kHz 16-bit mono)
func startSpeaker() -> ProcessHandle ! {Process} {
  spawnProcess("sox", [
    "-t", "raw", "-r", "24000", "-b", "16", "-c", "1",
    "-e", "signed-integer", "-", "-d"
  ])
}

export func main() -> unit ! {Stream, Process, IO} {
  let config = defaultConfig(());
  match connect("wss://generativelanguage.googleapis.com/...", config) {
    Ok(conn) => {
      let wsSrc = sourceOfConn(conn, "gemini", 10);
      let speaker = startSpeaker();

      selectEvents([wsSrc], \event.
        match event {
          Message(msg) => {
            -- Parse Gemini response for audio chunks
            match extractAudioData(msg) {
              Ok(b64) => {
                let pcm = fromBase64(b64);
                -- Stream audio frame directly to sox — plays immediately!
                writeProcessStdin(speaker, pcm);
                true
              },
              Err(_) => { _io_println("gemini: " ++ msg); true }
            }
          },
          Closed(_, _) => {
            closeProcessStdin(speaker);
            false
          },
          _ => true
        });
      disconnect(conn)
    },
    Err(e) => _io_println("connection failed: " ++ e)
  }
}
```

### Example 2: Simple stdin/stdout pipe (test with cat)

```ailang
module examples/process_stdin_write

import std/process (spawnProcess, writeProcessStdin, closeProcessStdin, ProcessHandle)
import std/bytes (fromString)
import std/result (Result, Ok, Err)

export func main() -> unit ! {Process, IO} {
  let handle = spawnProcess("cat", []);

  -- Write three lines to cat's stdin
  writeProcessStdin(handle, fromString("hello\n"));
  writeProcessStdin(handle, fromString("world\n"));
  writeProcessStdin(handle, fromString("done\n"));

  -- Close stdin — cat will echo all lines and exit
  closeProcessStdin(handle)
}
```

### Example 3: Data pipeline (feed to jq)

```ailang
module examples/process_jq_pipe

import std/process (spawnProcess, writeProcessStdin, closeProcessStdin)
import std/bytes (fromString)

export func main() -> unit ! {Process, IO} {
  -- Spawn jq to pretty-print JSON from stdin
  let jq = spawnProcess("jq", ["."]);

  writeProcessStdin(jq, fromString("{\"name\":\"ailang\",\"version\":\"0.9.0\"}"));
  closeProcessStdin(jq)
}
```

## Success Criteria

- [ ] `spawnProcess("cat", [])` returns a `ProcessHandle`
- [ ] `writeProcessStdin(handle, bytes)` delivers bytes to subprocess stdin
- [ ] `closeProcessStdin(handle)` closes pipe, subprocess sees EOF and exits
- [ ] Write to exited process returns `Err("pipe closed")`
- [ ] Missing `--caps Process` produces capability error
- [ ] Process allowlist applies to `spawnProcess` commands
- [ ] No zombie processes after handle close or program exit
- [ ] `CloseAllManaged()` kills all managed processes on teardown
- [ ] Composes with `asyncExecProcess` — can read one process's stdout while writing to another's stdin
- [ ] All tests passing
- [ ] Example works with `echo "test" | ailang run --caps Process,IO --entry main examples/runnable/process_stdin_write.ail`

## Testing Strategy

**Unit tests:**
- `spawnProcess("cat", [])` + `writeProcessStdin` + `closeProcessStdin` — data echoed
- Write to closed handle → `Err("pipe closed")`
- Write after `closeProcessStdin` → `Err("stdin already closed")`
- `closeProcessStdin` idempotent (safe to call twice)
- Subprocess killed on `Close()` (verify process no longer running)
- Command not found → error (ProcessContext validation)
- Allowlist blocking → error
- Buffer full backpressure → `Err("write buffer full")`
- `CloseAllManaged` kills all tracked processes

**Integration tests:**
- `asyncExecProcess` (read stdout) + `spawnProcess` (write stdin) on different processes — both work
- Program exit without `closeProcessStdin` — subprocess killed via `CloseAllManaged`

**Manual testing:**
- `ailang run --caps Process,IO --entry main examples/runnable/process_stdin_write.ail`
- Verify audio playback with `sox` (macOS: `brew install sox`)

## Non-Goals

**Not in this feature:**
- **Stdout/stderr capture from spawned process** — Use `asyncExecProcess` for stdout. This feature is write-only.
- **Bidirectional I/O on same process** — Future Phase 4: `asyncExecProcessBidi → (StreamSource, ProcessHandle)`.
- **Interactive TTY** — No PTY allocation. Stdin is a raw pipe. Interactive programs (vim, less) won't work.
- **Shell expansion** — Same as `exec()`: no `sh -c`, args are literal.
- **Windows support** — Same Unix lifecycle model as Phase 2.
- **Write timeout** — Writes to the channel are non-blocking (return Err if full). Subprocess-side blocking is bounded by the pipe buffer.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Zombie processes on unclean exit | High | `CloseAllManaged()` in effect context teardown. Context cancellation propagates. |
| Subprocess ignores SIGTERM (doesn't read stdin) | Medium | 5s grace → SIGKILL, same as Phase 2. |
| Write buffer fills up (slow consumer) | Medium | `writeProcessStdin` returns `Err("write buffer full")`. Caller can log/skip. |
| Broken pipe race (write after subprocess exit) | Medium | writeLoop detects pipe error, sets `closed` flag. Next write returns `Err`. |
| Audio glitches from write batching | Low | 256-slot buffer provides ~38s of audio buffering. Goroutine drains continuously. |
| Platform-specific audio tools missing | Low | Allowlist fails at startup. Example uses cross-platform `cat` for CI. |

## Related Documents

**Implemented (informs design):**
- [M-ASYNC-IO Phase 1](design_docs/implemented/v0_9_0/m-async-io-stream.md) — EventSource interface, selectEventsLoop, asyncReadStdinLines (v0.9.0)
- [M-ASYNC-IO Phase 2](design_docs/planned/v0_9_0/m-async-io-process.md) — asyncExecProcess, processSource, subprocess stdout streaming (v0.9.0)
- [M-PROCESS](design_docs/implemented/v0_8_1/m-process-exec.md) — Synchronous exec, ProcessContext, allowlist, ProcessError ADT (v0.8.1)

**Planned (check for overlap):**
- [M-CSP-SESSION-TYPES](design_docs/planned/v1_0_0/m-csp-session-types.md) — Full CSP concurrency (v1.0.0)

## Future Work

- **Phase 4: Bidirectional subprocess I/O**: `asyncExecProcessBidi(cmd, args, name, priority, chunkSize) -> (StreamSource, ProcessHandle) ! {Process, Stream}` — read stdout events AND write to stdin on the same process.
- **Stderr forwarding**: `spawnProcessWithStderr(cmd, args) -> (ProcessHandle, StreamSource)` — stdin writable + stderr as events.
- **Restart policy**: Auto-restart subprocess on exit with backoff.
- **Process budget integration**: `Process @limit=N` to bound total managed process count.

---

**Document created**: 2026-03-08
**Last updated**: 2026-03-08
