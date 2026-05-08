# M-FS-SANDBOX-DIAGNOSTICS: Sandbox path rejection diagnostics

**Status**: Implemented
**Target**: v0.19.0
**Priority**: P1 (High — repeated production gotcha; silent false on `exists` causes hard-to-diagnose fallback degradation)
**Estimated**: 1 day (~150 LOC across effects/, cmd/)
**Dependencies**: None

**Author**: Claude Sonnet 4.6 + Mark
**Created**: 2026-05-08

**Trigger**: The `resolve_profile_dir` function in motoko_agent `config.ail` silently degraded to empty config when `AILANG_FS_SANDBOX` was active and WORKDIR was set to a per-task scratch dir. `fileExists(path)` returned `false` (sandbox rejection silently swallowed) rather than an error, so the fallback chain ran without any indication that the sandbox was the cause. This has now occurred multiple times across different programs.

---

## Problem Statement

`AILANG_FS_SANDBOX` restricts all FS operations to a root directory. The current behaviour diverges by operation type:

| Operation | Path escapes sandbox | Current behaviour |
|---|---|---|
| `readFile`, `writeFile`, `appendFile`, `removeFile` | error returned | propagates as runtime panic |
| `exists`, `isDir`, `isFile` | error **swallowed** | silently returns `false` |

The silent-`false` behaviour in the second row is intentional for the common case (programs shouldn't crash just because a file doesn't exist). But it creates a category of hard-to-diagnose bugs:

```ailang
-- resolve_profile_dir (motoko config.ail)
let dir0_ok = fileExists(dir0_cfg);   -- returns false — is file missing, or is it a sandbox escape?
if dir0_ok then dir0
else if fileExists(flat_cfg) then flat_dir  -- also false (sandbox)
else
  match getEnv("MOTOKO_REPO") {        -- maybe not set
    Err(_) => dir0                     -- silently uses empty config dir
  }
```

The program degrades to defaults (`extensions.order = []`, `cost_rates = {}`) with no error, no warning, and no clue that the sandbox is involved. The only current debug strategy is manually adding `println` statements to AILANG code.

### Why it's hard to spot

1. `fileExists` returning `false` is normal and expected — the fallback logic is correct
2. The sandbox rejection happens inside the Go runtime, invisible to AILANG code
3. The error message from `resolveSandboxPath` (`path %q escapes sandbox %q`) is discarded at [internal/effects/fs.go:378](internal/effects/fs.go#L378)
4. AILANG's trace system records effect invocations but not the resolved-vs-attempted path delta

---

## Goals

- **G1**: Make sandbox path rejections visible without requiring AILANG code changes
- **G2**: Zero cost in production (no I/O overhead when diagnostics are off)
- **G3**: Minimal API surface — use existing env var patterns, no new CLI flags

## Non-Goals

- Changing the `false`-return semantics of `exists/isDir/isFile` (that's the correct public contract)
- Adding policy enforcement (strict/allowlist) — that's a separate security feature
- Logging every successful path resolution (too noisy)

---

## Design

### M1 — `AILANG_FS_SANDBOX_DEBUG=1`: emit rejection warnings to stderr

When `AILANG_FS_SANDBOX_DEBUG=1` is set alongside `AILANG_FS_SANDBOX`, every sandbox path rejection emits a structured line to stderr:

```
[ailang/sandbox] REJECT exists("/path/to/config.json") → escapes sandbox "/tmp/task-abc" (returns false)
[ailang/sandbox] REJECT exists("/path/to/flat.json") → escapes sandbox "/tmp/task-abc" (returns false)
[ailang/sandbox] REJECT isDir("/path/to/dir") → escapes sandbox "/tmp/task-abc" (returns false)
```

Format: `[ailang/sandbox] REJECT <op>(<attempted_path>) → <reason> (returns <value>)`

This is deliberately stderr-only — it doesn't pollute AILANG program output and doesn't require `--caps IO`.

**Implementation**: add a `logSandboxReject` helper in `internal/effects/fs.go` called from the three silent-false sites:

```go
func logSandboxReject(op, attemptedPath, sandbox, returnVal string) {
    if os.Getenv("AILANG_FS_SANDBOX_DEBUG") != "1" {
        return
    }
    fmt.Fprintf(os.Stderr, "[ailang/sandbox] REJECT %s(%q) → escapes sandbox %q (returns %s)\n",
        op, attemptedPath, sandbox, returnVal)
}
```

Called at lines [378](internal/effects/fs.go#L378), [567](internal/effects/fs.go#L567), [597](internal/effects/fs.go#L597):

```go
// exists — before:
if sandboxErr != nil {
    return &eval.BoolValue{Value: false}, nil
}
// after:
if sandboxErr != nil {
    logSandboxReject("exists", pathVal.Value, ctx.Env.Sandbox, "false")
    return &eval.BoolValue{Value: false}, nil
}
```

Same pattern for `isDir` and `isFile`.

### M2 — Record rejections as OTEL trace events

When `AILANG_TRACE=deep` is active, record sandbox rejections as span events on the current FS span:

```
span: FS.exists
  event: sandbox.reject
    attempted_path: /path/to/config.json
    sandbox: /tmp/task-abc
    result: false
```

This lets `ailang trace list` + trace-debugger identify sandbox-related `false` returns in recorded traces without any extra env vars.

**Implementation**: in `logSandboxReject`, check `ctx.Trace` and emit a `RecordEffect` event if present. Since `logSandboxReject` needs access to `ctx`, pass `ctx *EffContext` as first arg.

### M3 — `ailang sandbox-check <path>` CLI subcommand

A diagnostic CLI command that prints what would happen to a given path under the current sandbox configuration:

```
$ AILANG_FS_SANDBOX=/tmp/task-abc ailang sandbox-check /home/mark/.motoko/config/default/config.json
sandbox:  /tmp/task-abc
path:     /home/mark/.motoko/config/default/config.json (absolute)
result:   REJECT — escapes sandbox
          exists/isDir/isFile would return false
          readFile/writeFile/etc would return error

$ AILANG_FS_SANDBOX=/tmp/task-abc ailang sandbox-check /tmp/task-abc/config.json
sandbox:  /tmp/task-abc
path:     /tmp/task-abc/config.json (absolute, within sandbox)
result:   ALLOW → /tmp/task-abc/config.json

$ AILANG_FS_SANDBOX=/tmp/task-abc ailang sandbox-check config.json
sandbox:  /tmp/task-abc
path:     config.json (relative)
result:   ALLOW → /tmp/task-abc/config.json
```

Exit code 0 = ALLOW, 1 = REJECT. Scriptable for use in shell-level debugging.

**Implementation**: add `cmd/ailang/sandbox_check.go`, wire into the cobra command tree as `ailang sandbox-check`.

---

## Acceptance Criteria

- [ ] **M1**: `AILANG_FS_SANDBOX_DEBUG=1` causes rejected `exists`, `isDir`, `isFile` calls to emit `[ailang/sandbox] REJECT ...` to stderr
- [ ] **M1**: No output when `AILANG_FS_SANDBOX_DEBUG` is unset (zero-cost in production)
- [ ] **M1**: Log line includes: op name, attempted path, sandbox root, return value
- [ ] **M2**: `AILANG_TRACE=deep` records sandbox rejections as trace events on the FS span (skips if no trace collector)
- [ ] **M3**: `ailang sandbox-check <path>` prints ALLOW/REJECT with resolved path and exits 0/1
- [ ] **M3**: `ailang sandbox-check` with no `AILANG_FS_SANDBOX` set prints a clear "no sandbox configured" message
- [ ] Unit tests for `logSandboxReject` output (capture stderr in test)
- [ ] `make ci` passes

---

## Usage Guide (for motoko / eval harness debugging)

When a motoko task silently loads empty config, run:

```bash
# Step 1 — identify what paths are being rejected
AILANG_FS_SANDBOX=/tmp/task-abc AILANG_FS_SANDBOX_DEBUG=1 ailang run motoko_agent/src/core/config.ail ...
# stderr output:
# [ailang/sandbox] REJECT exists("/home/mark/.motoko/config/default/config.json") → escapes sandbox "/tmp/task-abc" (returns false)

# Step 2 — confirm what would be allowed
ailang sandbox-check /home/mark/.motoko/config/default/config.json
# REJECT — escapes sandbox /tmp/task-abc

# Step 3 — fix by setting MOTOKO_REPO so resolve_profile_dir finds the fallback path within sandbox
MOTOKO_REPO=/tmp/task-abc AILANG_FS_SANDBOX=/tmp/task-abc ailang run ...
```

---

## Files Changed (estimated)

| File | Change |
|---|---|
| `internal/effects/fs.go` | Add `logSandboxReject(ctx, op, path, result)`, call from 3 silent-false sites |
| `internal/effects/context.go` | No change (logSandboxReject reads `ctx.Env.Sandbox` and `ctx.Trace` directly) |
| `cmd/ailang/sandbox_check.go` | New — `sandbox-check` subcommand (~60 LOC) |
| `cmd/ailang/main.go` | Wire new subcommand |
| `internal/effects/fs_test.go` | Tests for rejection logging (capture stderr) |
| `docs/docs/guides/debugging.md` | Add "Sandbox debugging" section with the 3-step recipe above |
| `changelogs/v0.10-current.md` | v0.19.0 DX entry |
