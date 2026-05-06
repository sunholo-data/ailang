# M-AILANG-FS-RESULT — Result-returning std/fs builtins

**Status**: Implemented (2026-05-06)
**Target**: v0.16.0
**Priority**: P1 (medium — agent-runtime safety)
**Estimated**: 0.5 day (actual: ~30 min single session)
**Dependencies**: None
**Implementation commits**:
- AILANG: `90cc85ad` (5 builtins + Go handlers + std/fs exports + 11 Go tests)
- motoko_agent: `be10cab` (tool_runtime.ail dispatcher migration + regression smoke)

**Bug Report**: Surfaced during M-MOTOKO-RPC-LOOP-FULL-MIGRATION live testing
([motoko_agent commit 06f74e5](https://github.com/sunholo-data/motoko_agent/commit/06f74e5))
— a missing-parent-dir on `WriteFile` killed the entire agent process
because `std/fs.writeFile` returns `()` and panics through the effect
system on syscall failure. The agent's tool dispatcher had no way to
catch the error and wrap it as a `ToolErrorResult`.

---

## Problem Statement

`std/fs` ships two error-handling regimes side by side:

| Builtin            | Return type                  | On syscall failure                  |
|--------------------|------------------------------|-------------------------------------|
| `readFileBytes`    | `Result[string, string]`     | Returns `Err(message)` ✅            |
| `readFile`         | `string`                     | **Panics through effect system** ❌  |
| `writeFile`        | `()`                         | **Panics through effect system** ❌  |
| `appendFile`       | `()`                         | **Panics through effect system** ❌  |
| `removeFile`       | `()`                         | **Panics through effect system** ❌  |
| `mkdir` / `mkdirAll` | `()`                       | **Panics through effect system** ❌  |

Agent runtimes (motoko, future agent harnesses) need a way to attempt
fs operations that may fail in user-supplied paths and react to the
failure without crashing the entire agent process. The legacy motoko
dispatcher tried to use `writeFile` directly and crashed when the
model emitted a path whose parent directory didn't exist:

```
Error: execution failed: writeFile: open
  /Users/mark/dev/sunholo/ailang/test_tools/hello.txt:
  no such file or directory
```

This terminated the user's TUI session — they had to restart motoko
to recover.

The mkdirAll workaround motoko shipped covers the most common case
(missing parent), but other failure modes — permission denied, disk
full, file vanished mid-read, sandbox violation, etc. — still escape
as fatal panics.

## Goals

**Primary**: agent dispatchers can call `std/fs` operations against
user-supplied paths and reliably recover from any syscall failure.

**Success metrics**:
- New `readFileResult`, `writeFileResult`, `appendFileResult`,
  `removeFileResult`, `mkdirAllResult` builtins exist and follow the
  `Result[T, string]` shape established by `readFileBytes`.
- Each new builtin has a Go-side test pinning at least one success
  path and one failure path (file-not-found, permission-denied, or
  sandbox-violation depending on the operation).
- motoko's `run_read_file` / `run_write_file` migrate to these
  builtins so an unexpected fs failure produces a `ToolErrorResult`
  instead of crashing the agent.
- A live test of the original failure path
  (`WriteFile path="missing/dir/foo.txt"` with no mkdirAll) returns
  a structured `Err` instead of panicking.

## Solution Design

### New std/fs builtins

Five new exports, each `Result[T, string] ! {FS}`:

```ailang
-- Result variants of the existing void-returning fs operations.
export func readFileResult(path: string)
  -> Result[string, string] ! {FS} = _fs_readFileResult(path)

export func writeFileResult(path: string, content: string)
  -> Result[(), string] ! {FS} = _fs_writeFileResult(path, content)

export func appendFileResult(path: string, content: string)
  -> Result[(), string] ! {FS} = _fs_appendFileResult(path, content)

export func removeFileResult(path: string)
  -> Result[(), string] ! {FS} = _fs_removeFileResult(path)

export func mkdirAllResult(path: string)
  -> Result[(), string] ! {FS} = _fs_mkdirAllResult(path)
```

### Go-side handlers

Each new handler mirrors the `fsReadFileBytes` pattern: return
`(eval.Value, error)` where the inner Value is `Ok(...)` or
`Err(message)`, and the outer Go error is reserved only for
type-mismatch arity errors (which AILANG's typechecker prevents at
compile time anyway).

```go
// internal/effects/fs.go
func fsReadFileResult(ctx *EffContext, args []eval.Value) (eval.Value, error) {
    // arg validation as today
    path := applySandbox(ctx, pathVal.Value)
    content, err := os.ReadFile(path)
    if err != nil {
        return fsMakeErr(fmt.Sprintf("cannot read file: %v", err)), nil
    }
    return fsMakeOk(&eval.StringValue{Value: string(content)}), nil
}
```

`fsWriteFileResult`, `fsAppendFileResult`, `fsRemoveFileResult`,
`fsMakeDirAllResult` follow the same pattern but `Ok(())` returns
`fsMakeOk(&eval.UnitValue{})`.

### Conflict surface

The new builtins are **purely additive** — no existing API is
changed, no existing example breaks. The only "conflict" is naming
overlap with the panicking variants: `readFile` vs `readFileResult`.
Following the precedent set by `readFileBytes` (Result-shaped) vs a
hypothetical panicking `readFileBytesUnsafe`, we keep both regimes.
Future code SHOULD prefer `*Result` for any user-supplied path; the
panicking variants stay valid for known-good paths (e.g. embedded
resources) where a panic is the right failure mode.

No parser, typechecker, or codegen changes — these are plain effect
builtins registered through the same path as `_fs_readFileBytes`.

### Out of scope

- Changing existing `readFile` / `writeFile` to return Result
  (breaking change; defer to a major version bump if ever needed).
- Wider Result-ification of `std/process`, `std/net`, etc. (each
  has its own design considerations; this doc is fs-only).
- Changing motoko's WriteFile contract (still tool-role envelope;
  this just lets motoko *catch* fs failures it currently can't).

## Implementation Plan

| Milestone | Description                                                              | Est LOC |
|-----------|--------------------------------------------------------------------------|---------|
| M1        | `fsReadFileResult` + `_fs_readFileResult` + `readFileResult` + Go test   | ~120    |
| M2        | `fsWriteFileResult` + `_fs_writeFileResult` + `writeFileResult` + test   | ~120    |
| M3        | `appendFileResult`, `removeFileResult`, `mkdirAllResult` (same pattern)  | ~150    |
| M4        | Update motoko's `run_read_file` / `run_write_file` to use the new
            builtins so fs panics are caught at the dispatch boundary.
            Add a regression smoke (`smoke_v2_writefile_missing_parent.ail`). | ~50     |

Total: ~440 LOC (+ ~100 LOC test).

## Examples

### Caller migration in motoko_agent

Before (the line that crashed the user's TUI):
```ailang
let _ = writeFile(path, content);  -- panics on missing parent dir
```

After:
```ailang
match writeFileResult(path, content) {
  Ok(_)    => WriteFileResult({ id, path, bytes_written: ..., ... }),
  Err(msg) => ToolErrorResult({ id: id, tool: "WriteFile", message: msg })
}
```

### Direct user code

```ailang
import std/fs (writeFileResult)
import std/result (Result, Ok, Err)
import std/io (println)

func save_config(path: string, content: string) -> () ! {FS, IO} {
  match writeFileResult(path, content) {
    Ok(_)    => println("✓ saved ${path}"),
    Err(msg) => println("✗ save failed: ${msg}")
  }
}
```

## Success Criteria

- [ ] All 5 new builtins registered and discoverable via `ailang builtins list | grep Result`
- [ ] All 5 std/fs exports added with proper docstrings
- [ ] Go tests cover both Ok and Err paths for each
- [ ] motoko's `tool_runtime.ail` migrated to use them
- [ ] Live regression: motoko WriteFile to a missing-parent path returns
      a structured tool-role error, agent process keeps running
- [ ] CHANGELOG entry under v0.16.0
- [ ] Document moved from `planned/v0_16_0/` to `implemented/v0_16_0/` on completion

## Timeline

Single-session sprint: ~0.5 day total
- M1+M2 (read/write Result variants): 2 hours
- M3 (append/remove/mkdirAll): 1 hour
- M4 (motoko migration + regression smoke): 1 hour
- Docs + CHANGELOG: 30 min

## Cross-references

- Bug surfaced in: [motoko_agent commit 06f74e5 — fix(M10): mkdirAll parent dir before WriteFile](https://github.com/sunholo-data/motoko_agent/blob/ailang-tool-loop-migration/CHANGELOG.md)
- Related: M-MOTOKO-RPC-LOOP-FULL-MIGRATION (the migration that made this gap visible)
- Related: AG-UI follow-up — when motoko adopts AG-UI, the dispatch boundary still needs a way to catch fs errors and emit them as `TOOL_CALL_RESULT` events with structured error payloads. This work is a prerequisite.
