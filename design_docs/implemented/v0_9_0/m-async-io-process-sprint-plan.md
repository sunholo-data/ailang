# Sprint Plan: M-ASYNC-IO Phase 2 — asyncExecProcess

## Summary

Add `asyncExecProcess` to `std/stream` — spawns a subprocess and delivers its stdout as `SourceBytes` events into `selectEvents`. Enables mic capture, screen capture, and any streaming subprocess in the ambient assistant CLI.

**Duration:** 1 day (~6 hours)
**Dependencies:** M-ASYNC-IO Phase 1 (completed), M-PROCESS (completed)
**Risk Level:** Low (proven patterns from Phase 1 + existing ProcessContext)
**Design Doc:** [m-async-io-process.md](m-async-io-process.md)

## Velocity

- M-ASYNC-IO Phase 1: 850 LOC in ~2 hours (4 milestones)
- This sprint: ~450 LOC estimated, simpler scope (one new source type)
- Stream subsystem patterns are mature — implementation is fast

## Proposed Milestones

### Milestone 1: processSource EventSource Implementation
**Goal:** Create `processSource` struct implementing `EventSource`, with subprocess lifecycle management.
**Estimated:** ~170 LOC (100 impl + 70 tests)

**Tasks:**
1. Create `internal/effects/stream_process.go`:
   - `processSource` struct: `cmd *exec.Cmd`, `stdout io.ReadCloser`, `events chan streamEvent`, `done chan struct{}`, `cancel context.CancelFunc`
   - `NewProcessSource(ctx, cmd, args, name, priority, chunkSize, processCtx)` — validates via ProcessContext, spawns subprocess, starts reader goroutine
   - Reader goroutine: `io.ReadFull(stdout, buf[:chunkSize])` loop → `SourceBytes` events
   - `Close()`: cancel context → SIGTERM → 5s grace → SIGKILL → wait for goroutine
   - Implements `EventSource` interface (Name, Priority, Events, Close)
2. Unit tests in `stream_process_test.go`:
   - `echo "hello"` delivers SourceBytes with correct content
   - EOF closes source cleanly
   - Close() kills subprocess (no zombie)
   - chunkSize respected for multi-chunk output

**Acceptance Criteria:**
- [ ] `processSource` implements `EventSource` interface
- [ ] Subprocess stdout read in `chunkSize` chunks as `SourceBytes` events
- [ ] Subprocess killed on `Close()` (SIGTERM → SIGKILL)
- [ ] EOF from subprocess → clean source close
- [ ] No zombie processes
- [ ] Tests pass, lint clean

---

### Milestone 2: Handler + Builtin + Stdlib Wiring
**Goal:** Wire `asyncExecProcess` into effects registry, register builtin, update stdlib.
**Estimated:** ~130 LOC (80 impl + 50 tests)

**Tasks:**
1. Add `StreamAsyncExecProcess` handler to `stream_async_ops.go`:
   - Extract args (cmd, args, name, priority, chunkSize)
   - Get ProcessContext from EffContext (reuse allowlist validation)
   - Create `processSource`, store in `StreamContext.sources`
   - Return `StreamSource(id)`
2. Register `"asyncExecProcess"` op in `stream.go` init()
3. Register `_stream_async_exec_process` builtin in `builtins/stream.go`
4. Update `std/stream.ail` with `asyncExecProcess` export
5. Update golden snapshot
6. Tests: allowlist enforcement, capability check

**Acceptance Criteria:**
- [ ] `asyncExecProcess` callable from AILANG
- [ ] ProcessContext allowlist applies
- [ ] Builtin registered (183 total)
- [ ] `std/stream.ail` exports `asyncExecProcess`
- [ ] Golden snapshot updated
- [ ] All existing tests pass

---

### Milestone 3: Example + Integration Test + Docs
**Goal:** Working example, integration test, changelog update.
**Estimated:** ~150 LOC (40 example + 60 integration test + 50 docs)

**Tasks:**
1. Create `examples/runnable/stream_process_source.ail`
2. Integration test: subprocess + stdin in `selectEvents`
3. Update `examples/manifest.json`
4. Update `changelogs/v0.9-current.md`
5. Run `make test` + `make lint`

**Acceptance Criteria:**
- [ ] Example compiles and runs with piped input
- [ ] Integration test verifies subprocess + stdin multiplexing
- [ ] `make test` passes, `make lint` clean
- [ ] CHANGELOG updated

## Success Metrics

| Metric | Target |
|--------|--------|
| New LOC | ~450 total |
| New tests | 10+ test cases |
| Builtins | 183 (was 182) |
| Existing tests | All passing |
| Linting | Clean |

---

**Sprint created**: 2026-03-08
**Design doc**: [m-async-io-process.md](m-async-io-process.md)
