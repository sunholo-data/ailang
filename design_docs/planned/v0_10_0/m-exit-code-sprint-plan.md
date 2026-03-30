# Sprint Plan: M-EXIT-CODE

## Summary
Add `exit(code: int)` builtin to AILANG via a sentinel panic pattern, enabling CLI tools to terminate with explicit exit codes. Unblocks ai-coding-lang-bench Track A.

**Duration:** 1 day (~4 hours)
**Dependencies:** None
**Risk Level:** Low

## Current Status Analysis

### Completed Recently
- ✅ Fix pkg/ route canonicalization: ~80 LOC
- ✅ Add charCode builtin + FNV-1a hash: ~120 LOC
- ✅ Add bitwise operators: ~200 LOC
- ✅ Fix serve-api @route dispatch: ~150 LOC

### Velocity
- Recent average: ~200-400 LOC/day (small focused features)
- This sprint: ~80 LOC implementation + tests — well within capacity

### Remaining from Design Doc
- ⏳ Phase 1: Core implementation (sentinel, effect op, builtin, std wrapper)
- ⏳ Phase 2: Runtime integration (recover handler in main_run.go)
- ⏳ Phase 3: Testing and example file

## Proposed Milestones

### Milestone 1: M1_CORE_EXIT — Sentinel + Effect + Builtin
**Goal:** Create EvalExitCode sentinel, ioExit effect operation, _io_exit builtin spec, and std/io.ail wrapper
**Estimated:** ~40 LOC implementation + ~20 LOC tests = ~60 LOC
**Duration:** ~1.5 hours

**Tasks:**
1. Create `internal/eval/exit.go` — EvalExitCode sentinel struct (~10 LOC)
2. Add `ioExit` operation to `internal/effects/io.go` — panics with EvalExitCode (~15 LOC)
3. Register `_io_exit` builtin in `internal/builtins/io.go` — BuiltinSpec with IO effect (~25 LOC)
4. Add `exit` wrapper to `std/io.ail` (~2 LOC)

**Acceptance Criteria:**
- [ ] EvalExitCode sentinel type exists with Code int field
- [ ] ioExit registered as IO effect operation
- [ ] _io_exit builtin registered with type `int -> () ! {IO}`
- [ ] `exit` exported from std/io.ail
- [ ] `make build` succeeds
- [ ] `make lint` clean

**Risks:**
- None significant — follows established builtin patterns exactly

### Milestone 2: M2_RUNTIME_CATCH — Runtime Integration
**Goal:** Add recover() handler in main_run.go to catch EvalExitCode and call os.Exit(code)
**Estimated:** ~15 LOC implementation + ~5 LOC cleanup = ~20 LOC
**Duration:** ~30 minutes

**Tasks:**
1. Add recover() handler in executeModuleEntrypoint or runFile that catches EvalExitCode
2. Ensure trace flushing and telemetry cleanup before os.Exit()
3. Handle batch mode edge case (exit terminates whole batch)

**Acceptance Criteria:**
- [ ] EvalExitCode panic is caught and converted to os.Exit(code)
- [ ] Non-EvalExitCode panics still propagate normally
- [ ] Telemetry/traces flushed before exit
- [ ] All existing tests still pass

**Risks:**
- Panic-based approach could interfere with existing error handling — mitigated by unique sentinel type

### Milestone 3: M3_TESTS_EXAMPLE — Tests and Example
**Goal:** Integration tests verifying exit codes and example file
**Estimated:** ~40 LOC tests + ~15 LOC example = ~55 LOC
**Duration:** ~1 hour

**Tasks:**
1. Integration test: exit(0) produces exit code 0
2. Integration test: exit(1) produces exit code 1
3. Integration test: exit(42) produces exit code 42
4. Integration test: compile error without --caps IO
5. Create examples/exit_code.ail

**Acceptance Criteria:**
- [ ] All integration tests pass
- [ ] Missing IO capability produces compile error
- [ ] examples/exit_code.ail runs correctly
- [ ] `make test` passes
- [ ] `make verify-examples` passes

**Risks:**
- None — straightforward test file creation

## Success Metrics
- All tests passing: `make test`
- Linting clean: `make lint`
- Example verified: `make verify-examples`
- exit(0), exit(1), exit(42) all produce correct OS exit codes
- Missing --caps IO produces compile error

## Dependencies
- None

## Open Questions
- None — all design decisions frozen in design doc

## Notes
- Uses IO effect (not Process) per design doc decision
- Return type is Unit (not Never) — defer Never type to future work
- Negative exit codes: OS typically takes code & 0xFF, no special handling needed
