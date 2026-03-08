# Sprint Plan: M-ASYNC-IO Phase 3 — Subprocess Stdin Writing (ProcessHandle)

## Summary

Add `spawnProcess`, `writeProcessStdin`, and `closeProcessStdin` builtins to `std/process`, enabling AILANG programs to write bytes incrementally to a subprocess's stdin pipe. Completes the bidirectional subprocess I/O story for the ambient assistant's real-time audio playback.

**Duration:** 1.5 days (~11 hours)
**Dependencies:** M-ASYNC-IO Phase 2 (implemented), M-PROCESS (implemented)
**Risk Level:** Low — follows established patterns from Phase 2 (`processSource`) and `std/process` (`exec`)
**Design Doc:** [m-async-io-process-stdin.md](m-async-io-process-stdin.md)

## Current Status Analysis

### Completed Recently
- M-ASYNC-IO Phase 1+2: ~1900 LOC in ~2 days (selectEventsLoop, stdin sources, processSource, mux)
- M-CLOUD-WEBHOOK + M-CLOUD-ENDPOINT-AUTH: ~500 LOC in 1 day
- M-CLOUD-DISPATCH: ~600 LOC in 1 day

### Velocity
- Recent average: ~600-900 LOC/day (high velocity sprint period)
- Conservative estimate for this feature: ~500 LOC/day (new abstraction, careful testing)
- Estimated capacity: ~630 LOC over 1.5 days

### Existing Infrastructure (reuse targets)
- `ProcessContext` (`process_context.go`, 25 LOC) — needs extension for managed processes
- `process.go` (289 LOC) — existing exec handler, command resolution logic to reuse
- `builtins/process.go` (91 LOC) — registration pattern established
- `std/process.ail` (69 LOC) — exports ProcessError, ProcessOutput, exec
- `stream_async_ops.go` (257 LOC) — command resolution via allowlist (copy pattern)
- `stream_process.go` (154 LOC) — process lifecycle (SIGTERM → SIGKILL pattern to reuse)

## Proposed Milestones

### Milestone 1: managedProcess Core + ProcessContext Extension
**Goal:** Create the `managedProcess` struct with write channel, write loop goroutine, and clean shutdown. Extend `ProcessContext` with managed process tracking.
**Estimated:** ~130 LOC implementation + ~40 LOC context extension = ~170 LOC
**Duration:** ~3 hours

**Tasks:**
1. Create `internal/effects/process_managed.go`:
   - `managedProcess` struct: cmd, stdin pipe, writeCh (cap 256), done channel, sync.Once
   - `NewManagedProcess(ctx, cmdPath, args)` — start cmd, open stdin pipe, launch writeLoop
   - `writeLoop()` goroutine — drain writeCh, write to stdin pipe, handle errors
   - `Write(data []byte) error` — non-blocking send to writeCh with backpressure
   - `CloseStdin()` — close channel, drain remaining, close pipe, wait for exit
   - `Close()` — idempotent full cleanup: SIGTERM → 5s grace → SIGKILL (reuse pattern from `processSource`)
2. Extend `internal/effects/process_context.go`:
   - Add `mu sync.Mutex`, `managed map[int]*managedProcess`, `nextManagedID int`
   - `AcquireManagedProcess(mp) int` — register, return ID
   - `GetManagedProcess(id) (*managedProcess, bool)` — lookup
   - `ReleaseManagedProcess(id)` — remove from map
   - `CloseAllManaged()` — kill all tracked processes (teardown)

**Acceptance Criteria:**
- [ ] `NewManagedProcess` starts subprocess, opens stdin pipe
- [ ] `Write()` sends data through write channel to subprocess stdin
- [ ] `CloseStdin()` signals EOF, subprocess exits cleanly
- [ ] `Close()` kills subprocess (SIGTERM → SIGKILL), idempotent
- [ ] `CloseAllManaged()` kills all tracked processes
- [ ] No zombie processes after Close()

**Risks:**
- Write channel backpressure design — Mitigation: return error string, don't block

### Milestone 2: Effect Handlers (spawn, write, close)
**Goal:** Create the three Process effect operation handlers that bridge AILANG values to the managedProcess API.
**Estimated:** ~150 LOC
**Duration:** ~2 hours

**Tasks:**
1. Create `internal/effects/process_spawn.go`:
   - `ProcessSpawn(ctx, args)` — extract cmd/args strings, resolve via ProcessContext allowlist (copy pattern from `StreamAsyncExecProcess`), create managedProcess, register in context, return `ProcessHandle(id)` ADT
   - `ProcessWriteStdin(ctx, args)` — extract handle ID + bytes, lookup managedProcess, call Write, return `Ok(())` or `Err(reason)`
   - `ProcessCloseStdin(ctx, args)` — extract handle ID, lookup, call CloseStdin, release from context, return unit
2. Register operations in `init()`:
   - `RegisterOp("Process", "spawnProcess", ProcessSpawn)`
   - `RegisterOp("Process", "writeProcessStdin", ProcessWriteStdin)`
   - `RegisterOp("Process", "closeProcessStdin", ProcessCloseStdin)`
3. Add helper: `makeProcessHandle(id int) eval.Value` — creates `ProcessHandle(id)` ADT
4. Add helper: `extractProcessHandleID(v eval.Value) (int, error)` — extracts int from `ProcessHandle(int)`

**Acceptance Criteria:**
- [ ] `ProcessSpawn` resolves commands via allowlist
- [ ] `ProcessSpawn` returns `ProcessHandle(int)` ADT value
- [ ] `ProcessWriteStdin` writes bytes to correct managed process
- [ ] `ProcessWriteStdin` returns `Result` ADT (Ok/Err)
- [ ] `ProcessCloseStdin` closes stdin and waits for subprocess exit
- [ ] Operations registered in Process effect registry
- [ ] Allowlist enforcement works (blocked command → error)

**Risks:**
- Command resolution duplication with `StreamAsyncExecProcess` — Mitigation: Extract shared helper if >10 lines duplicated

### Milestone 3: Builtin Registration + stdlib
**Goal:** Wire the effect handlers into the builtin system and expose via `std/process.ail`.
**Estimated:** ~75 LOC (builtins) + ~15 LOC (stdlib) = ~90 LOC
**Duration:** ~1.5 hours

**Tasks:**
1. Add to `internal/builtins/process.go`:
   - `registerProcessSpawn()` — `_process_spawn_process` builtin with type: `(String, List[String]) -> ProcessHandle ! {Process}`
   - `registerProcessWriteStdin()` — `_process_write_stdin` builtin with type: `(ProcessHandle, Bytes) -> Result[(), String] ! {Process}`
   - `registerProcessCloseStdin()` — `_process_close_stdin` builtin with type: `(ProcessHandle) -> () ! {Process}`
   - Call all three from `init()`
2. Update `std/process.ail`:
   - Add `export type ProcessHandle = ProcessHandle(int)`
   - Add `export func spawnProcess(cmd, args)` wrapper
   - Add `export func writeProcessStdin(handle, data)` wrapper
   - Add `export func closeProcessStdin(handle)` wrapper
3. Update `internal/pipeline/testdata/builtin_types.golden` (regenerate snapshot)

**Acceptance Criteria:**
- [ ] `ailang doctor builtins` passes with new builtins
- [ ] Golden snapshot updated (should show 3 new builtins: 190→193 lines)
- [ ] `std/process.ail` exports ProcessHandle type and 3 functions
- [ ] `make test` passes
- [ ] `make lint` passes

**Risks:**
- Type builder DSL for ProcessHandle ADT — Mitigation: follow `StreamSource` pattern exactly

### Milestone 4: Tests + Example + Docs
**Goal:** Comprehensive test coverage, working example, documentation updates.
**Estimated:** ~200 LOC (tests) + ~30 LOC (example) + docs = ~230 LOC
**Duration:** ~3 hours

**Tasks:**
1. Create `internal/effects/process_managed_test.go`:
   - Test spawn + write + close basic flow (using `cat`)
   - Test write to closed handle → Err
   - Test write after closeProcessStdin → Err
   - Test closeProcessStdin idempotent
   - Test subprocess killed on Close()
   - Test command not found → error
   - Test allowlist blocking → error
   - Test buffer full backpressure (if feasible)
   - Test CloseAllManaged kills tracked processes
2. Create `examples/runnable/process_stdin_write.ail`:
   - Spawn `cat`, write 3 lines, close stdin
   - Simple demo that works with `ailang run --caps Process,IO --entry main`
3. Update CHANGELOG.md with M-ASYNC-IO Phase 3 entry
4. Move design doc status from Planned to reflect implementation
5. Update prompts if needed (std/process gained new exports)

**Acceptance Criteria:**
- [ ] All unit tests passing (≥9 test cases)
- [ ] Example runs successfully: `ailang run --caps Process,IO --entry main examples/runnable/process_stdin_write.ail`
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] `make verify-examples` passes
- [ ] CHANGELOG updated

**Risks:**
- `cat` behavior differences across platforms — Mitigation: use simple echo pattern, skip on Windows

## Day-by-Day Breakdown

### Day 1 (~7 hours)
- **Morning** (~3h): Milestone 1 — managedProcess core + ProcessContext extension
- **Midday** (~2h): Milestone 2 — Effect handlers (spawn, write, close)
- **Afternoon** (~2h): Milestone 3 — Builtin registration + stdlib + golden snapshot

### Day 2 (~4 hours)
- **Morning** (~3h): Milestone 4 — Tests (9+ test cases) + example file
- **Wrap-up** (~1h): CHANGELOG, docs, final `make ci` verification

## Success Metrics
- All tests passing: `make test`
- Linting clean: `make lint`
- New builtins validated: `ailang doctor builtins`
- Example working: `make verify-examples` includes process_stdin_write.ail
- Golden snapshot updated (193 builtins)
- CHANGELOG updated with M-ASYNC-IO Phase 3 entry

## Dependencies
- M-ASYNC-IO Phase 2 (implemented) — `processSource` pattern reused
- M-PROCESS (implemented) — `ProcessContext`, allowlist, exec handler
- `std/result` — Result[(), string] ADT for write errors

## Open Questions
None — design is well-constrained by existing patterns. All decisions documented in design doc.

## Notes
- The `managedProcess` lifecycle mirrors `processSource` (Phase 2) but reversed: write-only instead of read-only
- Command resolution logic is duplicated from `StreamAsyncExecProcess` — consider extracting a shared `resolveCommand` helper if this grows further
- `CloseAllManaged()` is called from effect context teardown but currently no central teardown path exists in main.go — this is an existing gap (same for `StreamContext.CloseAll()`) that should be addressed separately
- `ProcessHandle` is a new ADT in `std/process`, separate from `StreamSource` in `std/stream` — deliberately different types for different concerns
