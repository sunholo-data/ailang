# M-EXIT-CODE: Process Exit with Explicit Exit Code

**Status**: Planned
**Target**: v0.10.0
**Priority**: P1 — Blocks CLI tool development and ai-coding-lang-bench Track A
**Estimated**: 0.5 days (~4 hours)
**Dependencies**: None (builtin infrastructure, IO effect system, and runtime all exist)
**Motivation**: ai-coding-lang-bench minigit benchmark — Claude Code could not implement minigit because AILANG has no way to exit with a non-zero exit code

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | exit(code) is deterministic — same code always produces same OS exit |
| A2: Replayability | 0 | Exit code is captured in traces like any other effect call |
| A3: Effect Legibility | +1 | Requires IO capability — side effect is explicit in the type signature |
| A4: Explicit Authority | +1 | Gated behind IO capability — cannot call without `--caps IO` |
| A5: Bounded Verification | 0 | No impact on type checking |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Enables AI agents to write CLI tools that conform to Unix exit code conventions |
| A8: Minimal Syntax | 0 | No new syntax — uses existing function call syntax |
| A9: Cost Visibility | 0 | Immediate, obvious cost (process termination) |
| A10: Composability | 0 | Terminal operation — does not compose (by design) |
| A11: Structured Failure | 0 | Not an error mechanism — explicit program termination |
| A12: System Boundary | +1 | Makes the OS process boundary crossing explicit and controlled |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): Fully deterministic — `exit(0)` always exits 0
- [x] A3 (Effects): IO effect is required in the type signature
- [x] A4 (Authority): Gated behind IO capability
- [x] A7 (Machines First): Directly enables AI agents to build Unix-conforming CLI tools

## Problem Statement

AILANG programs cannot control their process exit code. The runtime always exits 0 on success or 1 on unhandled error. This makes it impossible to write CLI tools that communicate status via exit codes — a fundamental Unix convention.

**Current State:**
- `std/process` has `exec()` which returns `ProcessOutput` with `exitCode` from child processes
- But the AILANG process itself has no way to exit with a specific code
- `cmd/ailang/main_run.go` calls `os.Exit(1)` on error, `os.Exit(0)` implicitly on success
- No `Never`/bottom type exists in the type system (not needed for this feature)

**Discovery:**
- Found during the ai-coding-lang-bench benchmark when Claude Code tried to implement minigit in AILANG
- The benchmark test script (`test-v1.sh`) checks exit codes from `./minigit` commands
- Every other language in the benchmark suite supports program exit with a code (Go: `os.Exit`, Python: `sys.exit`, Haskell: `exitWith`, etc.)

**Impact:**
- Blocks all CLI tool development where exit codes matter (which is most CLI tools)
- Blocks AILANG participation in Track A of ai-coding-lang-bench
- Every serious programming language has this primitive

## Goals

**Primary Goal:** Allow AILANG programs to terminate with a caller-specified exit code.

**Success Metrics:**
- `exit(0)` terminates with code 0
- `exit(1)` terminates with code 1
- `exit(42)` terminates with code 42
- Requires IO capability — compile error without `--caps IO`
- ai-coding-lang-bench minigit can use `exit(1)` for error paths

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Panic-based vs return-based exit | Determines whether `os.Exit()` is called directly or the runtime catches a sentinel | agent | design | med |
| Module placement: `std/io` vs `std/sys` | Affects import paths in all user code | human | design | med |
| Return type: `Unit` vs introducing `Never` | Affects type system and future `Never` type work | human | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] **Panic-based exit** (see Solution Design below)
- [x] **Module: `std/io`** — exit is an IO operation; a new `std/sys` module is premature for one function
- [x] **Return type: `Unit`** — defer `Never` type to future work; `exit()` practically never returns but the type system does not need to know that yet

## Solution Design

### Overview

Add `exit(code: int) -> () ! {IO}` as a builtin in `std/io`. The Go implementation uses a sentinel panic (`EvalExitCode`) that propagates up the call stack. The runtime (`main_run.go`) catches this panic and calls `os.Exit(code)`.

**Why panic-based, not `os.Exit()` directly:**
- Calling `os.Exit()` inside the evaluator would skip deferred cleanup (trace flushing, budget reports, debug output)
- A sentinel panic lets the runtime unwind cleanly, flush telemetry, then exit
- This pattern is already used in Go's `testing` package (`runtime.Goexit`)

### Architecture

**Components:**
1. **Sentinel type** (`internal/eval/exit.go`): `EvalExitCode` struct with `Code int`
2. **Effect operation** (`internal/effects/io.go`): `ioExit` function that panics with `EvalExitCode`
3. **Builtin spec** (`internal/builtins/io.go`): Register `_io_exit` with BuiltinSpec
4. **AILANG wrapper** (`std/io.ail`): `export func exit(code: int) -> () ! {IO} = _io_exit(code)`
5. **Runtime catch** (`cmd/ailang/main_run.go`): `recover()` in `runCommand` that catches `EvalExitCode` and calls `os.Exit(code)`

### Implementation Plan

**Phase 1: Core implementation** (~2 hours)
- [ ] Add `EvalExitCode` sentinel type to `internal/eval/exit.go`
- [ ] Add `ioExit` operation to `internal/effects/io.go`
- [ ] Register `_io_exit` builtin in `internal/builtins/io.go`
- [ ] Add `exit` wrapper to `std/io.ail`

**Phase 2: Runtime integration** (~1 hour)
- [ ] Add `recover()` handler in `cmd/ailang/main_run.go` to catch `EvalExitCode`
- [ ] Ensure trace flushing and cleanup happen before `os.Exit()`
- [ ] Handle edge case: `exit()` inside batch mode

**Phase 3: Testing and examples** (~1 hour)
- [ ] Unit test for `ioExit` effect operation
- [ ] Integration test: `exit(0)` produces exit code 0
- [ ] Integration test: `exit(1)` produces exit code 1
- [ ] Integration test: `exit(42)` produces exit code 42
- [ ] Integration test: compile error without IO capability
- [ ] Add `examples/exit_code.ail` example

### Files to Modify/Create

**New files:**
- `internal/eval/exit.go` - EvalExitCode sentinel type (~10 LOC)

**Modified files:**
- `internal/effects/io.go` - Add `ioExit` operation and register it (~20 LOC)
- `internal/builtins/io.go` - Register `_io_exit` BuiltinSpec (~30 LOC)
- `std/io.ail` - Add `exit` wrapper function (~3 LOC)
- `cmd/ailang/main_run.go` - Add `recover()` handler for EvalExitCode (~15 LOC)

## Examples

### Example 1: CLI tool with error exit

**Before (impossible):**
```ailang
-- No way to signal failure to the calling process
export func main() -> () ! {IO} {
  println("error: invalid argument")
  -- program exits 0 even though we want to signal failure
}
```

**After:**
```ailang
module myapp

import std/io (println, exit)

export func main() -> () ! {IO} {
  let valid = false
  if valid then
    println("ok")
  else {
    println("error: invalid argument")
    exit(1)
  }
}
```

### Example 2: Minigit-style CLI

```ailang
module minigit

import std/io (println, exit)
import std/process (exec)

export func main() -> () ! {IO, Process} {
  let cmd = "init"
  match cmd {
    "init" => {
      println("Initialized empty repository")
      exit(0)
    },
    _ => {
      println("Unknown command: " ++ cmd)
      exit(1)
    }
  }
}
```

## Success Criteria

- [ ] `exit(0)` causes the AILANG process to exit with code 0
- [ ] `exit(1)` causes the AILANG process to exit with code 1
- [ ] `exit(code)` works for any int value (clamped to 0-255 by OS)
- [ ] IO capability is required — missing `--caps IO` produces a compile error
- [ ] Telemetry/traces are flushed before exit
- [ ] All existing tests still pass
- [ ] Example file added: `examples/exit_code.ail`

## Testing Strategy

**Unit tests:**
- `ioExit` panics with `EvalExitCode{Code: N}` for various N values
- `_io_exit` builtin spec validates correctly (arity, type, effect)

**Integration tests:**
- Run `ailang run --caps IO exit_code.ail` and verify process exit code matches
- Run without `--caps IO` and verify compile error about missing IO capability
- Verify exit inside nested function calls still works (panic propagates)

**Manual testing:**
- `echo $?` after running an AILANG program that calls `exit(42)` should print `42`

## Deferred Decisions

The following are intentionally left open for the implementer:

- **Negative exit codes**: Whether to clamp, error, or pass through — agent may choose (OS typically takes `code & 0xFF` anyway)
- **Exit inside `batch` mode**: Whether exit terminates the whole batch or just the current iteration — agent may choose, document the behavior

## Non-Goals

**Not attempted in this feature:**
- **`Never`/bottom type**: Would be the ideal return type for `exit()`, but introducing a new type into the Hindley-Milner system is a separate, larger effort. Using `Unit` return type is pragmatically correct — code after `exit()` is unreachable but the type checker does not need to enforce that yet.
- **`std/sys` module**: A dedicated system module could hold `exit`, `getpid`, `signal` etc. Premature for one function — revisit if more system primitives are needed.
- **Structured exit / `atexit` hooks**: No cleanup callback mechanism. The panic-based approach naturally lets Go `defer` statements run, which handles trace flushing.
- **Exit code from `main` return value**: Some languages use `main() -> int` where the return value is the exit code. This is a larger design change to the entry point convention.

## Timeline

**Day 1** (~4 hours):
- Phase 1: Core implementation (sentinel, effect, builtin, std wrapper)
- Phase 2: Runtime integration (recover handler, cleanup)
- Phase 3: Tests and example file

**Total: ~4 hours in 1 day**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Panic-based approach interferes with existing error handling | Med | Use a unique sentinel type (`EvalExitCode`) that cannot be confused with other panics. Only catch this specific type in `recover()`. |
| `os.Exit()` skips deferred cleanup | Low | The `recover()` handler runs cleanup explicitly before calling `os.Exit()`. |
| Code after `exit()` is unreachable but type checker allows it | Low | Document this as a known limitation. Defer to future `Never` type work for enforcement. |

## Related Documents

**Implemented (may inform design):**
- `std/process.ail` — existing process execution with exit code capture
- `internal/builtins/io.go` — existing IO builtins (print, println, readLine, writeBytes)
- `internal/effects/io.go` — existing IO effect operations

**Planned (check for overlap):**
- `m-eval-cross-language-benchmark.md` — the benchmark that discovered this gap

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [ai-coding-lang-bench](https://github.com/anthropics/ai-coding-lang-bench) - Benchmark that exposed the gap
- [Go os.Exit](https://pkg.go.dev/os#Exit) - Target implementation primitive
- [Haskell exitWith](https://hackage.haskell.org/package/base/docs/System-Exit.html) - Prior art
- [Python sys.exit](https://docs.python.org/3/library/sys.html#sys.exit) - Prior art (uses SystemExit exception — same panic pattern)

## Future Work

- **`Never` type**: Introduce a bottom type so `exit()` has type `int -> never ! {IO}`, enabling dead code detection after `exit()` calls
- **`std/sys` module**: If more system primitives are added (signals, PID, etc.), migrate `exit` to `std/sys` with a re-export from `std/io` for backward compatibility
- **`main() -> int` convention**: Allow entry points to return an int that becomes the exit code, as an alternative to calling `exit()` explicitly

---

**Document created**: 2026-03-30
**Last updated**: 2026-03-30
