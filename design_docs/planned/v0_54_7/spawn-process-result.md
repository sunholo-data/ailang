# spawnProcessResult: a Result-returning process spawn

**Status**: Planned
**Target**: v0.54.7
**Priority**: P1 (Medium)
**Estimated**: 2 days (~6h implementation + 3h testing + 2h docs + buffer)
**Dependencies**: None (reuses the existing `ProcessError` ADT and the `Authorize` decision shared with `exec`)

## Problem Statement

`std/process.spawnProcess` returns a bare `ProcessHandle`. Every pre-spawn
failure — a command missing from `PATH`, a command outside
`--process-allowlist`, or an OS-level spawn failure — is returned to the
evaluator as a Go error, which aborts the entire run. The program cannot
observe the failure, so it cannot fall back.

**Current state (reproduced on v0.54.6, repo binary `71f4cf6`):**

```
-- capprobe.ail
export func main() -> () ! {IO,Process} {
  println("started");
  match cancelProcess(spawnProcess("crew-nonexistent-worker",["x"])) {_=>()};
  println("after")
}
```

```console
$ ailang run --quiet --caps IO,Process --process-allowlist crew-nonexistent-worker capprobe.ail
started
Error: execution failed: _process_spawn_process: command not found: crew-nonexistent-worker
```

`after` never prints. Same for the allowlist case: with `--process-allowlist cat`,
`spawnProcess("/bin/cat", [])` aborts the run with
`command not allowed: /bin/cat`.

**The asymmetry:** `std/process.exec` already returns
`Result[ProcessOutput, ProcessError]`, and `ProcessError` already has the
exact constructors this case needs: `NotAllowed(string)`, `NotFound(string)`,
`SpawnFailed(string)`. `spawnProcess` shares the *same* authorization decision
(`ProcessContext.Authorize`, `internal/effects/process_authorize.go:29` —
"the one authorizer shared with spawnProcess and asyncExecProcess";
`internal/effects/process_spawn.go:56` reuses it via `resolveCommand`) and
the same spawn helpers, but wraps the denial in a Go error
(`internal/effects/process_spawn.go:58`:
`fmt.Errorf("_process_spawn_process: %w", err)`) and throws the typed
value away.

**Impact:**
- Downstream (sunholo/crew_lab 0.6.0) works around this by scanning `$PATH`
  with `fileExists` before spawning — racy (PATH can change between check and
  spawn), duplicates PATH resolution, and still cannot handle the
  `NotAllowed` case at all (no way to probe the allowlist).
- AI-generated programs (the language's primary audience) cannot express
  "try this worker, else fall back" — a normal, decidable control-flow shape.

**Two documentation gaps ride along (behavior is correct, docs are silent):**
1. `--process-allowlist` matches the command string **exactly as written at
   the call site** (`pc.Allowlist[cmdName]`, `internal/effects/process_authorize.go:46`).
   Allowlisting `name` and spawning `/abs/path/name` is `NotAllowed` — and
   vice versa. PATH pinning happens per allowlist entry at startup; the
   per-call string is the lookup key. Worth a sentence in the docs.
   (Verified: `--process-allowlist cat` + `spawnProcess("/bin/cat", [])` →
   `command not allowed: /bin/cat`.)
2. `cancelProcess` on a worker that already exited (auto-reaped by the
   release goroutine, `internal/effects/process_spawn.go:78`) returns
   `Err(WorkerHandleInvalid("process handle is stale, foreign or unknown"))`
   (`internal/effects/process_cancel.go:35`). This is the **normal reaped
   case**, not an error condition; docs should say so.
   (Verified: spawn `cat`, close stdin, `sleep(500)`, cancel → `handle invalid: process handle is stale, foreign or unknown`.)

### Systemic audit (same-gap check before designing)

The "bare handle, fatal pre-spawn errors" shape is not unique to
`spawnProcess`. All three Process entry points:

| Function | Module | Return | Pre-spawn failure handling |
|---|---|---|---|
| `exec` | `std/process` | `Result[ProcessOutput, ProcessError]` | Typed (`NotFound`/`NotAllowed`/`SpawnFailed`) — `internal/effects/process.go:142,153` |
| `spawnProcess` | `std/process` | `ProcessHandle` | **Fatal** — `internal/effects/process_spawn.go:58` |
| `asyncExecProcess` | `std/stream` | `StreamSource` | **Fatal** — `internal/effects/stream_async_process.go:63-72` |

This doc fixes `spawnProcess` (the reported case) with a mechanism
(a shared denial→typed-`Err` helper) that `asyncExecProcessResult` can reuse
verbatim if parity is wanted. Whether to ship the stream sibling now or in a
follow-up is a review decision — see High-Impact Decisions.

## Goals

**Primary Goal:** A program holding the `Process` capability can attempt a
subprocess spawn and fall back on failure:
`spawnProcessResult(cmd, args) -> Result[ProcessHandle, ProcessError] ! {Process}`,
with **zero behavior change** to existing `spawnProcess` callers.

**Success Metrics:**
- The repro program above prints `started` and `after` with an observable
  `Err(NotFound(...))` in between (today: fatal abort).
- `NotAllowed` (allowlist + confined mode) and `SpawnFailed` are observable
  and matchable in the same `match`.
- No existing `.ail` program changes meaning: `spawnProcess` keeps its
  signature and fail-fast semantics (regression fixtures below).
- Stdlib convention held: `…Result` suffix pairs a fail-fast function with
  a typed one (`readFileResult`, `callResult`, `sendMsgResult`).
- Docs state the allowlist string-match rule and the `cancelProcess` reaped
  case.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Add `spawnProcessResult` as a **new** function; `spawnProcess` unchanged | Changing `spawnProcess`'s return type breaks every existing caller (stdlib pair convention is fail-fast + `…Result`) | human | design | high |
| Reuse the existing `ProcessError` ADT (no new `SpawnError` type) | Callers match one error type across `exec`/`spawnProcessResult`; avoids a near-duplicate ADT in the stdlib surface | human | design | med |
| `spawnProcessResult` **never aborts the run after the capability gate**: every pre-spawn failure (resolution, allowlist, confinement, OS spawn, admission/shutdown) is a typed `Err`; no fatal escape of its own | A machine-decidable contract ("match on it, you will get a value") is the whole point; a residual fatal path would silently reintroduce the bug. The capability gate itself (`--caps Process` absent) aborts every Process effect including `exec` — that is run policy, unchanged (V13) | human | design | med |
| Error payloads: denial → existing `ProcessDenial{Ctor, Detail}` mapping; spawn/admission failure → `SpawnFailed(err.Error())` | `Err` payload shape is API surface; exec already renders `NotAllowed`/`NotFound` detail this way | agent | design | low |
| `asyncExecProcessResult` (std/stream) parity in the same sprint vs follow-up | Same mechanism, but doubles test/doc surface in this sprint | human (review) | design | med |
| Bytecode VM: append `__process_spawn_result` to `EffectBuiltinNames` (append-only) | Persisted bytecode uses explicit numeric indices; insertion would corrupt old images | compiler | compile | low |
| Docs-only items (allowlist string-match sentence; `cancelProcess` reaped case) ship in the same release | They are one-sentence additions to pages this feature already touches | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] New function name: `spawnProcessResult` (convention: `…Result` suffix; builtin `_process_spawn_result`; grep shows both unallocated)
- [x] Error type: existing `ProcessError` ADT — `NotAllowed`/`NotFound`/`SpawnFailed` payloads only (no `Timeout`/`OutputLimitExceeded`/`AbnormalExit` reachable pre-spawn)
- [x] `spawnProcess` signature and fail-fast semantics are untouched
- [x] Bytecode name appended, not inserted
- [x] `asyncExecProcessResult`: follow-up; no stream API is added in this sprint (approved sprint-plan scope).

## Solution Design

### Overview

Add a `Result`-returning sibling of `spawnProcess` to `std/process`, following
the established stdlib pair pattern (`readFile`/`readFileResult`). The new
effect op shares `spawnProcess`'s spawn path — same `Authorize` call, same
`NewManagedProcess`, same admission bookkeeping — but converts each
pre-spawn failure into a typed `ProcessError` value wrapped in
`Result[ProcessHandle, ProcessError]` instead of a Go error.

### API

```ailang
-- std/process (additive)
export func spawnProcessResult(
  cmd: string,
  args: [string]
) -> Result[ProcessHandle, ProcessError] ! {Process} =
  _process_spawn_result(cmd, args)
```

- `Ok(handle)` — the handle is a normal `ProcessHandle(int)`, usable with
  `writeProcessStdin`, `closeProcessStdin`, `cancelProcess` unchanged. The
  capability gate (`--caps Process` absent) aborts before this function is
  reached, exactly as it does for `exec` — run policy, unchanged (V13).
- `Err(NotAllowed(detail))` — command not in the allowlist (detail carries
  the command and, for narrowed entries, the subcommand words tried), or
  confined Process (spawn is exec-only under confinement — this becomes
  observable instead of fatal).
- `Err(NotFound(cmd))` — command not resolvable via `PATH`.
- `Err(SpawnFailed(reason))` — OS spawn failure or admission failure
  (execution owner shutting down): no usable handle is returned. Admission completion can fail after a child starts; that child is stopped and reaped before returning.

Type-checking of the exact surface (function returning
`Result[ProcessHandle, ProcessError]`, constructing and matching
`NotFound`/`NotAllowed`/`SpawnFailed`) was verified live with `ailang check`
today — see Verification Log. Only the *fallible op* is new.

### Architecture

**Components:**
1. **`std/process.ail`** — `spawnProcessResult` wrapper delegating to the
   `_process_spawn_result` builtin (same delegation shape as `exec` →
   `_process_exec`).
2. **`internal/effects/process_spawn.go`** — new op
   `RegisterOp("Process", "spawnProcessResult", ProcessSpawnResult)`.
   Refactor the shared spawn sequence out of `ProcessSpawn` (argument
   extraction → `resolveCommand` → `BeginWorker` → `NewManagedProcess` →
   `AcquireManagedProcess`/`Complete` → release goroutine) into one
   internal `spawnManaged(ctx, cmd, args) (eval.Value, *ProcessDenial, error)`
   helper; `ProcessSpawn` keeps today's fatal wrapping,
   `ProcessSpawnResult` converts:
   - denial → `makeProcessResultErr(denial.Ctor, denial.Detail)` (helper
     already exists in this package: `internal/effects/process.go:320`;
     `exec` uses it identically at `process.go:142,153`),
   - spawn/admission failure → `makeProcessResultErr("SpawnFailed", err.Error())`,
   - success → Ok-wrapped `makeProcessHandle(handleID)`.
   The helper carries the fixed Go-error prefix `_process_spawn_process:`
   only for the fatal variant, so error text of existing failures does not
   change.
3. **`internal/builtins/process.go`** — `registerProcessSpawnResult`:
   `Module: "std/process", Name: "_process_spawn_result", NumArgs: 2,
   Effect: "Process"`, type via Type Builder
   `T.App("Result", T.Con("ProcessHandle"), T.Con("ProcessError"))`
   (mirrors `makeProcessExecType`'s `T.App("Result", …)` at
   `internal/builtins/process.go:261`), metadata with
   `Since: "unreleased"` (matches `registerProcessCancel` in
   `process_cancel.go:27`).
4. **`internal/bytecode/effect_names.go`** — append `"__process_spawn_result"`
   to `EffectBuiltinNames` (append-only; the compiler's `effectBuiltinIndex`
   at `internal/bytecode/compiler/builtins.go:126` and the VM's
   `callEffectBuiltin` at `internal/vm/effects.go:25` then dispatch it with
   no further VM changes).
5. **Docs** (see Files to Modify).

No parser, typechecker, elaborator, or eval changes: the elaborator already
registers every builtin into `globalEnv` as `$builtin.<name>`
(`internal/elaborate/core.go:150`), so the new name resolves exactly like
`_process_exec` does today.

### Implementation Plan

**Phase 1: Effect op + builtin** (~3h)
- [ ] Extract shared spawn sequence in `internal/effects/process_spawn.go`; keep `ProcessSpawn` output byte-identical
- [ ] Add `ProcessSpawnResult` op with typed conversions
- [ ] Register `_process_spawn_result` in `internal/builtins/process.go` with Type Builder + metadata
- [ ] Append `__process_spawn_result` to `EffectBuiltinNames`
- [ ] `std/process.ail`: `spawnProcessResult` wrapper + comment block mirroring `exec`'s docs

**Phase 2: Tests** (~3h)
- [ ] `internal/effects/process_spawn_result_test.go`: NotFound, NotAllowed (incl. confined), SpawnFailed, Ok-handle-is-usable (write/close), `spawnProcess` unchanged (fatal path still returns Go error with same text)
- [ ] End-to-end: the capprobe repro as a runnable example — asserts `after` prints
- [ ] `internal/vm/worker_effect_test.go`-style dispatch check for the new effect name

**Phase 3: Docs + examples** (~2h)
- [ ] `docs/docs/reference/effects.md`: Process table row; allowlist string-match sentence (§Process flags); `cancelProcess` reaped-case sentence (§Worker lifetime)
- [ ] `docs/docs/guides/streaming.md`: `spawnProcessResult` subsection with fallback example + API table row
- [ ] `examples/runnable/process_spawn_result.ail` + `examples/manifest.json` entry
- [ ] CHANGELOG entry under v0.54.7

### Files to Modify/Create

**New files:**
- `internal/effects/process_spawn_result_test.go` (~130 LOC) — hermetic denial tests + real-spawn success path
- `examples/runnable/process_spawn_result.ail` (~35 LOC) — the capprobe repro, asserting `after` prints

**Modified files:**
- `std/process.ail` (+~25 LOC) — wrapper + docs
- `internal/effects/process_spawn.go` (+~70/−15 LOC) — shared spawn helper + `ProcessSpawnResult`
- `internal/builtins/process.go` (+~70 LOC) — registration + `makeProcessSpawnResultType`
- `internal/bytecode/effect_names.go` (+1 LOC)
- `docs/docs/reference/effects.md` (+~10 LOC), `docs/docs/guides/streaming.md` (+~25 LOC)
- `examples/manifest.json` (+1 entry), `changelogs/v0.32-current.md` (v0.54.7 entry)

## Examples

### Example 1: Fall back when the worker command is missing

**Before (aborts the run):**
```ailang
export func main() -> () ! {IO,Process} {
  println("started");
  match cancelProcess(spawnProcess("crew-nonexistent-worker",["x"])) {_=>()};
  println("after")   -- never reached
}
```

**After (decidable fallback):**
```ailang
import std/process (spawnProcessResult, cancelProcess, ProcessError)
import std/result (Result, Ok, Err)

export func main() -> () ! {IO,Process} {
  println("started");
  match spawnProcessResult("crew-nonexistent-worker", ["x"]) {
    Ok(handle) => match cancelProcess(handle) { _ => () },
    Err(NotFound(cmd)) => println("worker missing: ${cmd}"),
    Err(NotAllowed(cmd)) => println("worker not allowlisted: ${cmd}"),
    Err(SpawnFailed(reason)) => println("spawn failed: ${reason}"),
    Err(_) => ()
  };
  println("after")   -- always reached
}
```

### Example 2: The allowlist string-match rule (docs item)

```console
$ ailang run --caps IO,Process --process-allowlist cat use_cat.ail
# spawnProcess("cat", [])          → OK (allowlist key "cat" matches)
# spawnProcess("/bin/cat", [])     → NotAllowed — the lookup key is the
#                                     literal string; allowlist "cat" does
#                                     not authorize "/bin/cat"
```

Rule to document: **pass the command exactly as it appears in
`--process-allowlist`** (or add the absolute path as its own entry — PATH
pinning happens per entry at startup).

## Success Criteria

- [ ] Repro program prints `started` and `after` with a matchable `Err(NotFound(...))`; exits cleanly
- [ ] `NotAllowed` (allowlist miss, confined mode) and `SpawnFailed` reachable and matchable; `Timeout`/`OutputLimitExceeded`/`AbnormalExit` are NOT reachable from this function (pre-spawn only)
- [ ] `Ok` handle composes with `writeProcessStdin`/`closeProcessStdin`/`cancelProcess` unchanged
- [ ] `spawnProcess` behavior byte-identical (fail-fast, same error text)
- [ ] `ailang check` passes for the new wrapper from an importing module (verified pattern in Verification Log)
- [ ] VM path works (`EffectBuiltinNames` appended; `ailang run` tree-walk unaffected)
- [ ] `make test` green; `make test-core` (<15s inner loop) green; simplicity gates not regressed (`make simplicity-audit` — new builtin adds one stdlib surface point, within budget)
- [ ] Docs updated (effects.md, streaming.md); runnable example + manifest entry; CHANGELOG entry

## Verification Log

Every "does/doesn't" claim above was checked live (repo binary v0.54.6,
`71f4cf6`) unless it cites a file:line:

| # | Claim | Verification | Result |
|---|-------|--------------|--------|
| V1 | `spawnProcess` aborts the run on a missing command; `after` never prints | `ailang run --quiet --caps IO,Process --process-allowlist crew-nonexistent-worker capprobe.ail` | `Error: execution failed: _process_spawn_process: command not found: crew-nonexistent-worker`; `after` not printed ✗→ bug confirmed |
| V2 | Allowlist matches the command string as written | same setup, `spawnProcess("/bin/cat", [])`, `--process-allowlist cat` | `Error: execution failed: _process_spawn_process: command not allowed: /bin/cat` |
| V3 | `Result[ProcessHandle, ProcessError]` with constructed/matched `NotFound`/`NotAllowed`/`SpawnFailed` is expressible today | `ailang check` + `ailang run` on a probe module wrapping `spawnProcess`'s `Ok` and `Err(NotFound("missing"))` | `✓ No errors found!`; probe printed `not found: missing` |
| V4 | `cancelProcess` on an auto-reaped worker returns `WorkerHandleInvalid` | spawn `cat`, write+close stdin, `sleep(500)`, then `cancelProcess` | `handle invalid: process handle is stale, foreign or unknown` |
| V5 | `…Result` suffix is the stdlib pair convention | `grep 'export func.*Result(' std/*.ail` | 20+ hits: `readFileResult` (std/fs.ail:125), `writeFileResult` (std/fs.ail:131), `callResult` (std/ai.ail:142), `sendMsgResult` (std/cognition.ail:47) |
| V6 | Builtin name `_process_spawn_result` and stdlib name `spawnProcessResult` are unallocated | `grep -rn '_process_spawn_result\|spawnProcessResult' internal/ cmd/ std/ docs/` | no hits |
| V7 | Shared denial→`Err` helper already exists in the spawn op's package | `internal/effects/process.go:320` (`makeProcessResultErr`); used by exec at `process.go:142,153` | same package `effects`; no new helper needed |
| V8 | `asyncExecProcess` has the same gap | `internal/effects/stream_async_process.go:63-72` — denials/spawn failures return Go errors | confirmed fatal (audit row) |
| V9 | `EffectBuiltinNames` is append-only | `internal/bytecode/effect_names.go:14` comment + persisted-bytecode index usage (`compiler/builtins.go:126`, `vm/effects.go:25`) | append confirmed required |
| V10 | js/wasm has no spawn op registered (new op must match) | `process_spawn.go` carries `//go:build !js`; `ls internal/effects/*spawn*` → only `process_spawn.go` | on js builds the op lookup fails ("unknown operation") for both variants — unchanged behavior |
| V11 | Elaborator auto-registers builtins (`$builtin` module) — no parser/typecheck change needed | `internal/elaborate/core.go:144-154` (`AddBuiltinsToGlobalEnv`) | new name resolves like `_process_exec` |
| V12 | Regression fixtures exist | `ls examples/runnable/process_stdin_write.ail examples/runnable/process_cancel.ail examples/runnable/process_policy_cancel.ail cmd/ailang/run_policy_owned_test.go internal/effects/process_confined_test.go` | all present |
| V13 | Capability gate aborts (not typed) for every Process effect, incl. Result-returning ones | `ailang run --quiet --caps IO nocap.ail` (calls `spawnProcess`) | `Error: execution failed: effect 'Process' requires capability, but none provided` — gate is run policy, stays fatal; contract begins after the gate |
| V14 | Example 1's syntax (nested `Ok(handle) => match …`, ctor-pattern `Err(NotFound(cmd))` arms) type-checks today with a simulated source | `ailang check` + `ailang run` on a probe `spawnres2.ail` (`attempt` returns `Result[ProcessHandle, ProcessError]`) | `✓ No errors found!`; prints `started` / `worker missing: missing` / `after` |

## Conflict Surface

No parser, lexer, grammar, or effect-row changes — the mandatory
parser-oriented conflict analysis does not apply. The touched positions and
their existing claimers:

### Name-collision surface

| Position | Existing claimers | Disambiguation |
|---|---|---|
| Builtin registry name `_process_spawn_result` | `_process_spawn_process`, `_process_write_stdin`, `_process_close_stdin`, `_process_cancel` (all `std/process`) | Exact-name map in `specRegistry`; duplicate registration errors at startup (`internal/builtins/spec.go:105`) and `process.go` panics on it. V6 confirms unallocated |
| Stdlib export `std/process.spawnProcessResult` | `spawnProcess`, `exec`, `writeProcessStdin`, `closeProcessStdin`, `cancelProcess` | Module-scoped export table; no overloading in AILANG |
| Bytecode IR name `__process_spawn_result` | `__process_spawn_process` etc. (`internal/bytecode/effect_names.go`) | Appended index (V9); no reordering |
| Go-codegen stub table (`internal/builtins/registry_codegen_effects.go:197`) | `_process_spawn` stub maps `std/process.spawnProcess` | Effects panic in compiled mode by design; see Deferred Decisions |
A user module naming a local `_process_spawn_result` shadows exactly as any
existing builtin does (globalEnv `$builtin` entry vs local binding) — no new
shadowing class is introduced; behavior matches `_fs_readFileResult` today.

### Programs that MUST still work (regression fixtures)

- `examples/runnable/process_stdin_write.ail` (spawn → write → close)
- `examples/runnable/process_cancel.ail` (spawn → cancel, `WorkerCancelError` match)
- `examples/runnable/process_policy_cancel.ail` (+ its restricted companion fixture)
- `internal/effects/process_subcmd_test.go` (Authorize/subcommand narrowing)
- `cmd/ailang/run_policy_owned_test.go`, `cmd/ailang/run_policy_nested_test.go` (policy + spawn interplay)

### What deliberately changes

- Nothing observable for existing programs. One behavioral note: under
  **confined Process** (`resolveCommand`, `internal/effects/process_spawn.go:149-155`),
  `spawnProcess` today aborts with a fatal `NotAllowed`; the new function
  returns `Err(NotAllowed(...))` — typed observability is the point. The
  fail-fast variant is untouched.

## Testing Strategy

**Unit tests (`internal/effects/process_spawn_result_test.go`, `!js` build tag):**
- Missing command → `Err(NotFound("…"))`, no process spawned
- Allowlist miss → `Err(NotAllowed("…"))` with the same detail string exec produces
- Confined context → `Err(NotAllowed(... spawnProcess is not available ...))`
- Ungranted capability → aborts at the capability gate (`effect 'Process' requires capability`) — identical to `exec` today (V13); the gate is run policy and stays fatal for every Process effect. Below the gate, the `ctx.Process == nil` defense-in-depth check routes through `Authorize`'s `NotAllowed` denial instead of the bare `E_PROCESS_NO_CONTEXT` fatal the fail-fast variant keeps
- OS spawn failure (e.g. non-executable resolved path) → `Err(SpawnFailed(reason))`
- Success → `Ok(ProcessHandle(n))`; write/close round-trip; handle released after exit
- `ProcessSpawn` (fail-fast) still returns a Go error with the exact old text

**Integration:**
- `examples/runnable/process_spawn_result.ail` — asserts `started`/`after` both print under `--process-allowlist crew-nonexistent-worker`
- VM: new name dispatches through `callEffectBuiltin` (index/table integrity)

**Manual:**
- `ailang check` on the example from importing-module perspective (done at design time — V3)

## Deferred Decisions

- Exact comment phrasing in `std/process.ail` — agent may choose
- Whether to add a Go-codegen stub entry for `_process_spawn_result` in `registry_codegen_effects.go` (effects panic in compiled mode by design; `_process_spawn` stub exists as precedent) — agent may choose
- Test file organization (one test file vs extending `process_subcmd_test.go`) — agent may choose
- `asyncExecProcessResult` (std/stream parity) — human at review; default follow-up doc
- Where in `docs/docs/guides/streaming.md` the fallback example sits — agent may choose

## Non-Goals

**Not in this feature:**
- Changing `spawnProcess`'s signature — breaking; the pair convention exists precisely to avoid it
- A `waitProcess`/exit-status API for spawned children (stdout/stderr are discarded by design; reaping stays implicit) — out of scope
- Changing allowlist *matching* semantics (absolute-path normalization etc.) — the string-match behavior is correct and pinned at startup; this is a docs item only
- Removing the user-side PATH-scan workaround from crew_lab — downstream's job once this ships
- `Timeout`/`OutputLimitExceeded`/`AbnormalExit` payloads from `spawnProcessResult` — unreachable pre-spawn; they remain `exec`-only

## Timeline

**Day 1** (~6h): Phase 1 (op + builtin + wrapper) 3h; Phase 2 starts 3h
**Day 2** (~5h): finish tests 1h; docs + example + manifest + CHANGELOG 3h; full `make test` + simplicity audit 1h

**Total: ~11h across 2 days** (2× the naive 5h estimate, per repo convention)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Refactoring `ProcessSpawn` changes fatal error text used by tests/tools | Medium | Extract-shared-helper keeps the `_process_spawn_process: %w` wrapping in the fatal variant; byte-identical text asserted by a unit test |
| js/wasm builds diverge (op unregistered there) | Low | Registration stays in the `!js` file; js behavior ("unknown operation") unchanged — V10; no js stub is implied |
| `ProcessError` payloads grow ambiguous (e.g. `NotAllowed` detail vs bare cmd) | Low | Reuse `ProcessDenial.Detail` verbatim — the same strings exec already surfaces; no new payload shapes |
| Handle leak on typed failure paths (admission bookkeeping) | Medium | Failures happen before `AcquireManagedProcess`; `admission.Abort()` on every early return, mirroring `ProcessSpawn` |
| Scope creep into stream parity | Medium | Explicit review decision (default: follow-up doc) |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No nondeterminism introduced; removes a racy user-side PATH scan as the *only* fallback |
| A2: Replayability | 0 | Trace/replay unaffected |
| A3: Effect Legibility | +1 | Failure of a `! {Process}` operation becomes a typed, effect-visible value instead of a runtime abort |
| A4: Explicit Authority | 0 | Allowlist authority unchanged; denials enforced identically, now observable |
| A5: Bounded Verification | 0 | No verification surface change |
| A6: Safe Concurrency | 0 | No concurrency change (spawn sequence reused as-is) |
| A7: Machines First | +1 | AI programs can match a typed `ProcessError` instead of parsing fatal stderr |
| A8: Minimal Syntax | 0 | No new syntax; one stdlib function |
| A9: Cost Visibility | 0 | No resource-cost change |
| A10: Composability | +1 | `Result`/`match` composition, `ProcessHandle` unchanged |
| A11: Structured Failure | +1 | Pre-spawn failures structured into the existing `ProcessError` ADT |
| A12: System Boundary | 0 | OS boundary unchanged |

**Net Score: +4** → **Decision: Move forward** ✅

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted (denials still enforced)
- [x] A7 (Machines First): Typed fallback serves machine decidability, not human convenience

## Related Documents

**Implemented (may inform design):**
- `design_docs/implemented/v0_9_0/m-async-io-process-stdin.md` — original `spawnProcess` design (M-ASYNC-IO Phase 3)
- `design_docs/implemented/v0_38_0/m-process-subcmd-allowlist.md` — the shared `Authorize` decision this feature reuses
- `design_docs/implemented/v0_54_4/m-background-worker-lifecycle.md` — worker admission/lifecycle (`BeginWorker`/release goroutine)
- `design_docs/implemented/v0_54_5/m-worker-policy-and-lazy-ai.md` — `cancelProcess` policy semantics

**Planned (check for overlap):**
- None covering a `Result`-returning spawn (auto-search: no matches; V6)

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- `docs/docs/reference/effects.md` §Process — current spawn/exec surface
- `internal/effects/process_authorize.go` — `Authorize` (the shared decision)
- `internal/effects/process.go:142,153,320` — exec's denial→`Err` conversion (the pattern copied)
- Reported from: sunholo/crew_lab 0.6.0 (PATH-scan workaround)

## Future Work

- `asyncExecProcessResult: (cmd, args, name, priority, chunkSize) -> Result[StreamSource, ProcessError] ! {Stream, Process}` — same helper, stream sibling (V8 audit row; default follow-up)
- A `waitProcess`/exit-status surface for spawned children, if a use case that needs stdout-discipline changes appears
- Allowlist *probe* (`allowlisted(cmd) -> bool`) — rejected for now: probing invites TOCTOU-style drift between check and spawn; the `…Result` pattern avoids it by construction