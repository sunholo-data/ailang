# Sprint Plan: M-PROCESS — std/process Module

**Sprint ID**: M-PROCESS
**Design Doc**: [design_docs/planned/m-process-exec.md](design_docs/planned/m-process-exec.md)
**Duration**: 3 days (~12 hours)
**Risk Level**: Low (well-scoped, all infrastructure exists)
**Target Version**: v0.9.0

## Summary

Implement `std/process` — a new Process effect module that enables AILANG programs to execute external commands with capability-based security, structured output, and typed errors. This is the final "host integration" effect alongside FS, Net, IO, and Env.

**Key deliverables:**
- `exec(cmd, args)` function returning `Result[ProcessOutput, ProcessError]`
- `ProcessError` ADT with 7 variants (NotAllowed, NotFound, PermissionDenied, Timeout, OutputLimitExceeded, SpawnFailed, AbnormalExit)
- `ProcessOutput` record with bytes stdout/stderr, exitCode, resolvedPath
- CLI flags: `--process-timeout`, `--process-allowlist`, `--process-max-output`
- Completion semantics: Ok for all completions, Err for infra failures only
- `execText` convenience wrapper for UTF-8 decoded output
- WASM build exclusion (build tag `!js`)

## Velocity Analysis

**Recent velocity (14 days):**
- M-STREAM-PHASE2-DX: ~280 LOC (1 day)
- M-STREAM-DX: 6 milestones across several days
- M-SMT-BOUNDED-RECURSION: ~1,380 LOC impl + ~960 LOC tests
- M-TRACE-EXPORT: ~690 LOC (2 phases)

**Observed rate:** ~200-300 LOC/day for implementation + tests
**This sprint:** ~420 LOC implementation + ~200 LOC tests = ~620 LOC total
**Confidence:** High — effect handler + builtin patterns are well-established (FS, Stream, AI all follow same pattern)

## Milestones

### M1: Process Effect Handler + ProcessError ADT (~3 hours, ~180 LOC)

**Goal:** Create the Go effect handler that actually executes commands, and register the Process effect operations.

**Files:**
- `internal/effects/process.go` — New file (~120 LOC)
  - `processExec` handler using `os/exec.Command`
  - No shell expansion (direct exec, no `sh -c`)
  - Timeout via `context.WithTimeout`
  - Output capture with limit enforcement (terminate on exceed)
  - Allowlist checking with LookPath resolution
  - Sandbox working directory support
  - Returns `RecordValue` for ProcessOutput, ADT for ProcessError
- `internal/effects/process_context.go` — New file (~60 LOC)
  - `ProcessContext` struct (timeout, allowlist map, maxOutput, resolved paths)
  - Initialization from CLI flags
  - LookPath resolution at startup (path pinning)
  - WASM build tag `//go:build !js`

**Acceptance criteria:**
- [ ] `processExec` handler compiles and registers via `RegisterOp("Process", "exec", processExec)`
- [ ] `ProcessContext` holds timeout, allowlist, maxOutput config
- [ ] Allowlist entries resolved via `exec.LookPath` at init, pinned paths stored
- [ ] Output capture reads stdout/stderr into buffers up to maxOutput
- [ ] On output limit exceeded: kills process, returns OutputLimitExceeded error
- [ ] On timeout: kills process, returns Timeout error
- [ ] Non-zero exit returns Ok (completion semantics)
- [ ] Signal kill returns AbnormalExit error

### M2: Builtin Registration + CLI Flags (~2 hours, ~120 LOC)

**Goal:** Register `_process_exec` builtin and add Process-specific CLI flags.

**Files:**
- `internal/builtins/process.go` — New file (~60 LOC)
  - `_process_exec` registration following FS/IO pattern
  - Type: `T.Func(T.String(), T.List(T.String())).Returns(T.Result(T.Record(...), T.ADT(...))).Effects("Process")`
  - Impl: `effects.Call(ctx, "Process", "exec", args)`
  - Metadata with docs, params, examples
- `cmd/ailang/main.go` — Add 3 CLI flags (~10 LOC)
  - `--process-timeout` (default: "30s", Go duration string)
  - `--process-allowlist` (comma-separated names or absolute paths)
  - `--process-max-output` (default: "10MB", size string)
- `cmd/ailang/run_helpers.go` — Add `setupProcessHandler` (~20 LOC)
  - Parse flags → create ProcessContext → store on EffContext
- `internal/effects/context.go` — Add `Process *ProcessContext` field (~5 LOC)
- `internal/pipeline/testdata/builtin_types.golden` — Update snapshot

**Dependencies:** M1 (needs ProcessContext type)

**Acceptance criteria:**
- [ ] `_process_exec` appears in `ailang doctor builtins`
- [ ] CLI flags parse correctly: `ailang run --help` shows process flags
- [ ] Golden snapshot passes with `UPDATE_GOLDEN=1`
- [ ] ProcessContext initialized from CLI flags in setupProcessHandler
- [ ] Process capability check: missing `--caps Process` → CapabilityError

### M3: Stdlib Wrapper + ADT Types (~1 hour, ~50 LOC)

**Goal:** Create `std/process.ail` with AILANG-level types and wrapper functions.

**Files:**
- `std/process.ail` — New file (~50 LOC)
  - `module std/process`
  - `ProcessOutput` record type (stdout, stderr, exitCode, truncated, resolvedPath)
  - `ProcessError` ADT (7 variants)
  - `export func exec(...)` wrapper around `_process_exec`
  - `export func execText(...)` convenience wrapper (bytes → string via `bytes.toString`)

**Dependencies:** M2 (needs `_process_exec` registered)

**Acceptance criteria:**
- [ ] `std/process` imports successfully in AILANG programs
- [ ] `ProcessOutput` and `ProcessError` types accessible
- [ ] `exec("echo", ["hello"])` returns structured Result
- [ ] `execText` decodes stdout/stderr as UTF-8

### M4: Tests (~3 hours, ~200 LOC)

**Goal:** Comprehensive test suite covering all completion semantics and security features.

**Files:**
- `internal/effects/process_test.go` — New file (~200 LOC)
  - Happy path: `exec("echo", ["hello"])` → Ok with stdout
  - Completion semantics: `exec("cat", ["/nonexistent"])` → Ok with non-zero exitCode
  - Command not found: `exec("nonexistent_xyz", [])` → Err(NotFound)
  - Timeout: `exec("sleep", ["60"])` with 1s timeout → Err(Timeout)
  - Output limit: command producing large output → Err(OutputLimitExceeded)
  - Allowlist: blocked command → Err(NotAllowed)
  - Allowlist: name resolution → resolvedPath populated
  - Capability check: no Process cap → CapabilityError
  - No shell expansion: `exec("echo", ["$(whoami)"])` → literal `$(whoami)`
  - Bytes output: binary data preserved (not UTF-8 mangled)

**Dependencies:** M1, M2, M3

**Acceptance criteria:**
- [ ] All test cases pass
- [ ] `make test` passes (no regressions)
- [ ] `make lint` clean
- [ ] Completion semantics verified (Ok for all exits, Err for infra)

### M5: Documentation + Example (~2 hours, ~80 LOC)

**Goal:** Teaching prompt update, example file, help text.

**Files:**
- `examples/runnable/process_demo.ail` — New file (~30 LOC)
  - Simple exec examples (echo, exit codes, error handling)
  - Pattern match on ProcessError variants
- `examples/manifest.json` — Update with new example
- CHANGELOG.md — Add M-PROCESS entry
- `cmd/ailang/help.go` — Update caps list to include Process
- Teaching prompt — Add std/process section (deferred to prompt manager)

**Dependencies:** M1-M4 (all prior milestones)

**Acceptance criteria:**
- [ ] Example runs successfully: `ailang run --caps Process,IO --entry main examples/runnable/process_demo.ail`
- [ ] `make verify-examples` passes
- [ ] CHANGELOG updated with M-PROCESS entry
- [ ] Help text shows Process in capability list

## Dependency Graph

```
M1 (Effect Handler) ─┐
                      ├─→ M3 (Stdlib) ─→ M4 (Tests) ─→ M5 (Docs)
M2 (Builtin + CLI)  ─┘
```

M1 and M2 can be done in parallel (different files), but M3 needs both. M4 needs all three. M5 is final.

**Execution plan:** M1 → M2 → M3 → M4 → M5 (sequential — files overlap between milestones)

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|-----------|
| `os/exec` API differences across OS | Low | Low | Standard Go stdlib, well-tested |
| ProcessError ADT registration complexity | Medium | Low | Follow existing Result/Option pattern |
| Golden snapshot conflicts | Low | Low | `UPDATE_GOLDEN=1` regenerates |
| WASM build breakage | Low | Medium | Build tag `//go:build !js` on process files |
| Long-running test commands (sleep) | Low | Low | Use short timeouts (100ms) in tests |

## Success Metrics

- [ ] `exec("echo", ["hello"])` returns Ok with stdout bytes
- [ ] `exec("nonexistent", [])` returns Err(NotFound)
- [ ] Non-zero exit → Ok (not Err)
- [ ] Missing `--caps Process` → CapabilityError
- [ ] All 10+ test cases pass
- [ ] `make test && make lint` clean
- [ ] Example file runs successfully
- [ ] WASM build not broken
- [ ] Golden snapshot updated

## Estimated LOC

| Component | Implementation | Tests | Total |
|-----------|---------------|-------|-------|
| M1: Effect handler + context | 180 | — | 180 |
| M2: Builtin + CLI flags | 95 | — | 95 |
| M3: Stdlib wrapper | 50 | — | 50 |
| M4: Test suite | — | 200 | 200 |
| M5: Docs + examples | 80 | — | 80 |
| **Total** | **405** | **200** | **605** |

---

**Created**: 2026-02-18
**Sprint Planner**: Claude Opus 4.6
