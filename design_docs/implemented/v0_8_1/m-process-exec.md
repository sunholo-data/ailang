# M-PROCESS: std/process Module for External Command Execution

**Status**: Planned
**Target**: v0.9.0
**Priority**: P2 (Low — workaround exists via `io.writeBytes` piping)
**Estimated**: 3 days (6h implementation + 4h testing + 2h docs)
**Dependencies**: None (all effect infrastructure exists)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Process nondeterminism is *stronger* than FS: includes host state, PATH resolution, locale, available binaries, OS differences. Explicit via Process effect type. |
| A2: Replayability | +0.5 | Exec call + output are trace-recorded. Full replay requires `--replay-trace` mode that returns recorded outputs without spawning processes (not in v1, but structure supports it). |
| A3: Effect Legibility | +1 | Process effect is explicit in function signatures: `! {Process}` |
| A4: Explicit Authority | +1 | Requires `--caps Process`, no ambient access. CLI flags for allowlist and resource bounds. |
| A5: Bounded Verification | 0 | No impact on local type checking |
| A6: Safe Concurrency | 0 | No concurrency — exec is synchronous, blocks until child exits |
| A7: Machines First | +1 | Structured ADT error types (`ProcessError`), machine-parseable output fields, resolved paths in trace |
| A8: Minimal Syntax | +1 | No new syntax — uses existing function call, Result type, and effect declaration |
| A9: Cost Visibility | +0.5 | Mandatory timeout bounds wall-clock cost; Process effect signals expensive operation. Stronger with `Process @limit=N` budgeting (future). |
| A10: Composability | +1 | Composes with FS (write file then exec), IO (print result), Net (fetch then process) |
| A11: Structured Failure | +1 | Returns `Result[ProcessOutput, ProcessError]` — errors are typed ADTs, not strings |
| A12: System Boundary | +1 | Explicit transition from AILANG to host OS — the defining use case for A12. Structured trace event emitted for every exec call. |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): Nondeterminism is explicit via Process effect, not implicit
- [x] A3 (Effects): Process side effect is declared in type signature
- [x] A4 (Authority): Requires explicit `--caps Process` grant
- [x] A7 (Machines First): Returns structured ADTs, not raw strings

## Problem Statement

AILANG programs can read/write files (FS), talk to servers (Net), and handle console I/O (IO), but **cannot execute local commands**. This is the missing piece for host-tool integration.

**Current State:**
- The Gemini Live demo writes PCM audio to disk but cannot auto-play it
- No way to call `afplay`, `ffmpeg`, `sox`, or any host tool from AILANG
- Workaround: `io.writeBytes` piping works for stdout-based tools, but not for tools that need file arguments or produce files
- No way to invoke git, cloud CLIs, or any system utility

**Impact:**
- Streaming demos cannot complete the audio pipeline (record → save → play)
- AILANG programs cannot integrate with the host environment beyond file I/O
- Limits usefulness for automation, scripting, and tool orchestration

## Goals

**Primary Goal:** Enable AILANG programs to execute external commands with capability-based security and structured output.

**Success Metrics:**
- `exec("echo", ["hello"])` returns `Ok({stdout: ..., stderr: ..., exitCode: 0})`
- `exec("cat", ["/nonexistent"])` returns `Ok({..., exitCode: 1})` (completed but failed)
- `exec("nonexistent_cmd", [])` returns `Err(NotFound("nonexistent_cmd"))`
- Missing `--caps Process` returns `CapabilityError` (same pattern as FS/Net)
- Timeout enforcement: `Err(Timeout(30000))` for commands exceeding timeout
- No shell expansion: `exec("echo", ["$(whoami)"])` prints literal `$(whoami)`

## Solution Design

### Overview

Add a new `Process` effect with a single operation `exec` that runs an external command synchronously, captures stdout/stderr/exitCode, and returns a structured `Result` with typed errors.

### Architecture

**Components:**
1. **Effect handler** (`internal/effects/process.go`): Go implementation using `os/exec.Command`
2. **Builtin registration** (`internal/builtins/process.go`): `_process_exec` builtin spec
3. **Stdlib wrapper** (`std/process.ail`): `exec` function with clean AILANG API
4. **ADT types** (`std/process.ail`): `ProcessOutput` and `ProcessError` type definitions
5. **Security layer**: No shell expansion, allowlist with path resolution, mandatory timeout

### Completion Semantics (CRITICAL)

**`Ok` = process completed (regardless of exit code). `Err` = infra/spawn failure.**

| Scenario | Return | Rationale |
|----------|--------|-----------|
| `echo hello` exits 0 | `Ok({exitCode: 0, ...})` | Completed successfully |
| `cat /nonexistent` exits 1 | `Ok({exitCode: 1, ...})` | Completed with non-zero exit — tool's error, not ours |
| `grep -q pattern file` exits 1 | `Ok({exitCode: 1, ...})` | Non-zero exit is semantic (no match), not failure |
| Command not found | `Err(NotFound("cmd"))` | Spawn failure |
| Timeout exceeded | `Err(Timeout(30000))` | Infra failure |
| Command not in allowlist | `Err(NotAllowed("cmd"))` | Policy failure |
| Output cap exceeded | `Err(OutputLimitExceeded(10485760))` | Resource failure — process is killed |
| Permission denied | `Err(PermissionDenied("cmd"))` | OS failure |
| Killed by signal | `Err(AbnormalExit(9, "SIGKILL"))` | Abnormal termination |

This is the most composable design: it avoids conflating "tool returned error" with "we failed to run it."

### Type Definitions

```ailang
-- Process output: always returned on successful completion
type ProcessOutput = {
  stdout: bytes,           -- raw bytes (use bytes.toString for UTF-8)
  stderr: bytes,           -- raw bytes
  exitCode: int,           -- 0 = success, non-zero = tool error
  truncated: bool,         -- true if output was capped
  resolvedPath: string     -- absolute path of binary that was executed
}

-- Process error: structured ADT (not string!)
type ProcessError =
  | NotAllowed(string)             -- blocked by allowlist
  | NotFound(string)               -- binary not on PATH
  | PermissionDenied(string)       -- OS permission error
  | Timeout(int)                   -- ms elapsed before kill
  | OutputLimitExceeded(int)       -- bytes captured before kill
  | SpawnFailed(string)            -- other spawn error
  | AbnormalExit(int, string)      -- signal number, signal name
```

### API Design

```ailang
import std/process (exec, ProcessOutput, ProcessError)
import std/bytes (toString)

-- Execute a command with arguments
-- Returns Ok on completion (even non-zero exit), Err on infra failure
let result = exec("echo", ["hello", "world"])
match result {
  Ok(output) => {
    println(toString(output.stdout));    -- "hello world\n"
    println(toString(output.stderr));    -- ""
    println(show(output.exitCode))       -- "0"
  },
  Err(NotFound(cmd)) => println("Not found: " ++ cmd),
  Err(Timeout(ms)) => println("Timed out after " ++ show(ms) ++ "ms"),
  Err(NotAllowed(cmd)) => println("Blocked: " ++ cmd),
  Err(_) => println("Other error")
}
```

**Type signature:**
```
exec : (string, [string]) -> Result[ProcessOutput, ProcessError] ! {Process}
```

**Convenience wrapper (stdlib):**
```ailang
-- execText: same as exec but decodes stdout/stderr as UTF-8
-- Invalid UTF-8 sequences replaced with U+FFFD (deterministic)
export func execText(cmd: string, args: [string])
  -> Result[{stdout: string, stderr: string, exitCode: int, truncated: bool}, ProcessError] ! {Process}
```

### Security Design

**1. No shell expansion (CRITICAL)**
```go
// ✅ CORRECT — exec.Command runs binary directly
cmd := exec.Command(name, args...)

// ❌ NEVER — no sh -c wrapper
cmd := exec.Command("sh", "-c", userInput)
```

**2. Mandatory timeout (CLI flag, not hidden env var)**
```bash
# CLI flags (preferred — explicit authority)
ailang run --caps Process --process-timeout 30s module.ail
ailang run --caps Process --process-timeout 5m module.ail

# Env var override (lower precedence)
AILANG_PROCESS_TIMEOUT=30s ailang run --caps Process module.ail
```

Precedence: CLI flag > env var > default (30s).

**3. Allowlist with path resolution**
```bash
# CLI flag (preferred)
ailang run --caps Process --process-allowlist "echo,cat,ffmpeg,afplay,sox" module.ail

# Env var override
AILANG_PROCESS_ALLOWLIST="echo,cat,ffmpeg" ailang run --caps Process module.ail
```

**Path resolution behavior:**
- If allowlist entry is a name (e.g., `ffmpeg`): resolve via `exec.LookPath` **once at startup**, pin resolved absolute path, record in trace
- If allowlist entry is an absolute path (e.g., `/usr/bin/ffmpeg`): use directly
- Runtime uses pinned absolute path, not re-resolving per call (prevents TOCTOU)
- Resolved paths are recorded in execution trace for audit

If allowlist is set and command is not in it: return `Err(NotAllowed("cmd"))`.
If allowlist is not set: any command is permitted (with Process capability).

**4. Output capture limits**
```bash
# CLI flag
ailang run --caps Process --process-max-output 10MB module.ail

# Default: 10MB
```

When cap is hit: **terminate process and return `Err(OutputLimitExceeded(bytesRead))`**.
Termination is safer and more predictable than truncation. The `truncated` field in `ProcessOutput` handles the edge case where output is within limit but was close.

**5. Working directory**
- Default: current working directory of the AILANG process
- If `AILANG_FS_SANDBOX` is set: working directory is set to sandbox path
- **Important**: sandbox WD is a convenience default, NOT a security boundary. The child process can still access any file the OS allows. True containment requires OS-level sandboxing (namespaces/seatbelt), which is out of scope.
- No `chdir` parameter in v1 (future enhancement)

**6. Environment**
- Child process inherits parent's environment (same as `os/exec` default)
- No env passthrough parameter in v1 (future enhancement)

**7. Trace event emission**
Every exec call emits a structured trace event:
```json
{
  "command": "/usr/local/bin/ffmpeg",
  "args": ["-i", "input.pcm", "output.wav"],
  "resolved_path": "/usr/local/bin/ffmpeg",
  "timeout_ms": 30000,
  "max_output_bytes": 10485760,
  "exit_code": 0,
  "stdout_bytes": 0,
  "stderr_bytes": 1234,
  "duration_ms": 542,
  "truncated": false
}
```

### Implementation Plan

**Phase 1: Types + Effect handler + builtin** (~6 hours)
- [ ] Define `ProcessOutput` record type and `ProcessError` ADT in `std/process.ail`
- [ ] Create `internal/effects/process.go` with `processExec` handler
- [ ] Register `ProcessExec` in effects `init()`
- [ ] Create `internal/builtins/process.go` with `_process_exec` builtin
- [ ] Add `exec`, `execText` stdlib wrappers in `std/process.ail`
- [ ] Add CLI flags: `--process-timeout`, `--process-allowlist`, `--process-max-output`
- [ ] Update golden snapshot

**Phase 2: Testing** (~4 hours)
- [ ] Unit tests for exec handler (echo, cat, nonexistent command, timeout)
- [ ] Completion semantics test: non-zero exit → Ok with exitCode
- [ ] Spawn failure test: missing command → Err(NotFound)
- [ ] Capability check test (missing --caps Process)
- [ ] Allowlist tests (name + absolute path + LookPath resolution)
- [ ] Output limit test
- [ ] Trace event emission test

**Phase 3: Documentation** (~2 hours)
- [ ] Update teaching prompt with std/process section
- [ ] Add example file `examples/runnable/process_demo.ail`
- [ ] Document CLI flags in help text

### Files to Modify/Create

**New files:**
- `internal/effects/process.go` - Process effect handler (~120 LOC)
- `internal/builtins/process.go` - Builtin registration (~60 LOC)
- `std/process.ail` - Stdlib types + wrappers (~40 LOC)
- `internal/effects/process_test.go` - Tests (~200 LOC)

**Modified files:**
- `internal/pipeline/testdata/builtin_types.golden` - Add `_process_exec` entry
- `cmd/ailang/main.go` (or CLI flag registration) - Add --process-* flags
- `prompts/vX.X.md` - Add std/process documentation

## Examples

### Example 1: Audio Playback (Primary Use Case)

```ailang
module demos/audio_player

import std/process (exec, ProcessOutput, ProcessError)
import std/result (Result, Ok, Err)
import std/bytes (toString)

func playAudio(path: string) -> Result[string, ProcessError] ! {Process} {
  match exec("afplay", ["-f", "LEI16", "-r", "24000", "-c", "1", path]) {
    Ok(out) => {
      match out.exitCode {
        0 => Ok("played successfully"),
        n => Err(SpawnFailed("afplay exited with " ++ show(n)))
      }
    },
    Err(e) => Err(e)
  }
}
```

### Example 2: Checking Exit Code

```ailang
module demos/grep_check

import std/process (exec)
import std/bytes (toString)

-- grep returns exitCode 0 = found, 1 = not found, 2 = error
-- All three are Ok — completed processes, not infra failures
func fileContains(path: string, pattern: string) -> bool ! {Process} {
  match exec("grep", ["-q", pattern, path]) {
    Ok(out) => out.exitCode == 0,
    Err(_) => false  -- grep binary missing or blocked
  }
}
```

### Example 3: Command Not Found

```ailang
let result = exec("nonexistent_command", []);
-- Returns: Err(NotFound("nonexistent_command"))
-- NOT Err("some string") — structured ADT!
```

### Example 4: Format Conversion with Error Handling

```ailang
module demos/convert

import std/process (exec, ProcessError)
import std/bytes (toString)

func pcmToWav(pcmPath: string, wavPath: string) -> () ! {Process, IO} {
  match exec("ffmpeg", ["-f", "s16le", "-ar", "24000", "-ac", "1", "-i", pcmPath, wavPath]) {
    Ok(out) => {
      match out.exitCode {
        0 => println("Converted to WAV"),
        _ => println("ffmpeg error: " ++ toString(out.stderr))
      }
    },
    Err(NotFound(_)) => println("ffmpeg not installed"),
    Err(Timeout(ms)) => println("ffmpeg timed out after " ++ show(ms) ++ "ms"),
    Err(_) => println("exec failed")
  }
}
```

## Success Criteria

- [ ] `exec("echo", ["hello"])` returns `Ok({stdout: bytes, exitCode: 0, ...})`
- [ ] `exec("cat", ["/nonexistent"])` returns `Ok({exitCode: 1, ...})` — NOT Err
- [ ] `exec("nonexistent", [])` returns `Err(NotFound("nonexistent"))`
- [ ] Missing `--caps Process` produces `CapabilityError`
- [ ] Commands exceeding `--process-timeout` return `Err(Timeout(ms))`
- [ ] Commands exceeding `--process-max-output` return `Err(OutputLimitExceeded(bytes))`
- [ ] No shell expansion: arguments are passed literally
- [ ] Allowlist with LookPath resolution works (name → pinned absolute path)
- [ ] `resolvedPath` field populated in ProcessOutput
- [ ] Trace event emitted for every exec call
- [ ] WASM build excludes process module (build tag `!js`)
- [ ] All tests passing
- [ ] Documentation updated
- [ ] Example added

## Testing Strategy

**Unit tests:**
- `exec("echo", ["hello"])` → Ok with stdout capture
- `exec("cat", ["/nonexistent"])` → Ok with non-zero exitCode (completion, not failure)
- `exec("nonexistent_cmd", [])` → Err(NotFound)
- `exec("sleep", ["60"])` with 1s timeout → Err(Timeout)
- Output limit: command producing >10MB → Err(OutputLimitExceeded)
- Capability check: no Process cap → CapabilityError
- Allowlist: name-based resolution, absolute path, blocked command

**Integration tests:**
- Sandbox working directory: exec with AILANG_FS_SANDBOX set
- CLI flag precedence: --process-timeout overrides env var
- Trace event contents: resolved path, args, timing, output sizes

**Manual testing:**
- Audio playback demo with afplay
- Pipeline: write PCM → exec ffmpeg → verify WAV output

## Non-Goals

**Not in this feature:**
- **Shell expressions** (`sh -c "..."`) — Security risk, violates determinism. Use exec with explicit binary + args.
- **Background/async execution** — Violates A6 (Safe Concurrency). Future task graph system.
- **Stdin piping to child** — Complex, deferred. API designed to not block future `execWithInput(cmd, args, stdin: bytes)`.
- **Custom environment variables** — Deferred. Child inherits parent env.
- **Custom working directory** — Deferred. Uses sandbox or CWD.
- **OS-level sandboxing** — Namespace/seatbelt isolation is out of scope. Sandbox WD is convenience, not security.
- **WASM support** — Process execution is inherently host-only. WASM build should exclude this module.
- **AI agent orchestration** — See separate design doc: `m-agent-orchestration.md`

## Design Decisions & Rationale

### Why bytes instead of strings for stdout/stderr?

External commands may produce binary output (ffmpeg, imagemagick, audio tools). Defining output as `bytes` is the correct long-term choice:
- Binary-safe by default
- `execText` convenience wrapper handles UTF-8 decoding with deterministic replacement (U+FFFD)
- Doesn't paint us into a corner when stdin piping arrives (also bytes)
- Avoids lossy encoding on every exec call

### Why Ok for non-zero exit codes?

Non-zero exit is a *semantic* signal from the tool, not an infrastructure failure:
- `grep -q` returns 1 for "no match" — that's information, not an error
- `diff` returns 1 for "files differ" — useful data
- Conflating tool exit codes with spawn failures makes composition harder
- Users can trivially check `exitCode` and convert to Err if desired

### Why terminate on output limit instead of truncate?

- Termination is deterministic: you always get `Err(OutputLimitExceeded(n))`
- Truncation is ambiguous: partial stdout may be malformed JSON, broken UTF-8, etc.
- Safer default: user knows they need to increase limit or pipe differently
- `truncated: bool` field in ProcessOutput handles the case where output was within limit

### Why CLI flags instead of env vars for configuration?

- Env vars are hidden configuration — breaks A4 (Explicit Authority)
- CLI flags are visible in command invocation and help text
- Env vars are supported as lower-precedence overrides for CI/scripting
- Precedence: CLI flag > env var > default

## Timeline

**Day 1** (6 hours):
- Phase 1: Types, effect handler, builtin, stdlib wrapper, CLI flags
- Golden snapshot update

**Day 2** (4 hours):
- Phase 2: Tests (unit + integration + trace events)

**Day 3** (2 hours):
- Phase 3: Documentation, examples, teaching prompt

**Total: ~12 hours across 3 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Command injection via crafted args | High | No shell expansion — `exec.Command` directly, no `sh -c` |
| Hanging child process | Medium | Mandatory timeout (default 30s), `context.WithTimeout` |
| Large stdout/stderr consuming memory | Medium | Hard cap (default 10MB), terminate on exceed |
| PATH hijacking with name-based allowlist | Medium | LookPath at startup, pin resolved path, record in trace |
| Platform differences (Windows vs Unix) | Low | Use `exec.LookPath` for command resolution |
| WASM build breakage | Low | Build tag `//go:build !js` on process.go files |
| Signal handling differences across OS | Low | AbnormalExit ADT captures signal number + name |

## Related Documents

**Implemented (informs design):**
- [design_docs/implemented/v0_6_2/m-otel-integration.md](design_docs/implemented/v0_6_2/m-otel-integration.md) — Effect system patterns
- [design_docs/implemented/v0_7_2/m-wasm-stdlib.md](design_docs/implemented/v0_7_2/m-wasm-stdlib.md) — WASM exclusion patterns

**Planned (check for overlap):**
- [design_docs/planned/v0_8_1/m-arch4-executor-stream-processor.md](design_docs/planned/v0_8_1/m-arch4-executor-stream-processor.md) — Related: executor runs external processes
- [design_docs/planned/m-agent-orchestration.md](design_docs/planned/m-agent-orchestration.md) — Higher-level Agent effect (separate scope)

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- [Go os/exec package](https://pkg.go.dev/os/exec) — Implementation foundation
- Feature request: `msg_20260218_082635_ddd3e3ee` from demos-streaming

## Follow-Up Questions (Resolved)

| Question | Decision | Rationale |
|----------|----------|-----------|
| Non-zero exit codes: Ok or Err? | **Ok** | Tool semantics vs infra failure — composability wins |
| stdout/stderr type: bytes or string? | **bytes** | Binary-safe by default, `execText` for convenience |
| Primary use case: dev tooling or production? | **Dev/demo tooling** | P2 priority, allowlist + timeout sufficient for now |

## Future Work

- **Stdin piping**: `execWithInput(cmd, args, stdin: bytes) -> Result[ProcessOutput, ProcessError]`
- **Custom environment**: `execWithEnv(cmd, args, env: [{key: string, value: string}]) -> Result[...]`
- **Custom working directory**: `execInDir(cmd, args, dir: string) -> Result[...]`
- **Streaming output**: Callback-based output capture for long-running commands
- **Process budget**: `Process @limit=N` for bounding total exec calls (like IO @limit)
- **Replay mode**: `--replay-trace` runtime option that returns recorded outputs without spawning processes (strengthens A2)
- **`which` builtin**: `which(cmd: string) -> Option[string] ! {Process}` to help debug missing binaries
- **`ProcessConfig` record**: Optional config parameter for per-call timeout, maxOutput, cwd, env overrides
- **Agent orchestration**: See `m-agent-orchestration.md` for AI-specific execution with streaming, sessions, etc.

---

**Document created**: 2026-02-18
**Last updated**: 2026-02-18 (v2 — incorporated reviewer feedback: structured ProcessError ADT, completion semantics, bytes output, CLI flags, path-pinned allowlist, output limit termination, trace events)
