# Improve Module Loading Error Messages for Standalone/Scratch Files

**Status**: Planned
**Target**: v0.10.0
**Priority**: P1 - Medium (high impact on AI agent success rates in benchmarks)
**Estimated**: 4 hours
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Every feature must align with AILANG's 12 Design Axioms. Score each axiom and verify no hard violations.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to deterministic behavior; error messages are still deterministic |
| A2: Replayability | 0 | No impact on traces or replays |
| A3: Effect Legibility | 0 | No effect system changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | No verification changes |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Directly helps AI agents parse and recover from module errors |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No cost changes |
| A10: Composability | 0 | No composability changes |
| A11: Structured Failure | +1 | Error messages become actionable with structured fix suggestions |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +2** -> **Decision: Move forward**

### Hard Violation Check

**These axioms cannot have -1 scores (automatic rejection):**

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Actively improves machine analysis of errors

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| >= +2 | Proceed to implementation |
| 0 to +1 | Needs stronger justification |
| < 0 | Reject or redesign |
| Any -1 on A1/A3/A4/A7 | Automatic rejection |

## Problem Statement

During the ai-coding-lang-bench benchmark, AI agents (Claude Code, Gemini) repeatedly failed to write valid standalone AILANG files because module loading errors are confusing, internal-facing, and lack actionable guidance.

**Current State:**
- MOD010 errors in strict mode include suggestions, but the upstream `LoadAll` wraps them with internal search traces that obscure the fix: `"module loading error: failed to load /path/to/minigit.ail (search trace: [Loading module: ...])"`. The useful suggestion text from `validateMOD010` is buried inside nested `%w` wrapping.
- When a file is outside the project tree (e.g., `generated/`, `workspace/`, benchmark scratch dirs), `IsTempPath` returns false and strict MOD010 fires. The agent has no way to know it should use `--relax-modules` unless it has seen the prompt text.
- Stdlib version mismatch warnings (`Warning: stdlib version mismatch: expected v0.10.0, found v0.9.0 at ...`) appear even under `--relax-modules`, adding noise to stderr that AI agents parse as errors.
- The search trace (`[Loading module: benchmark/solution.ail  -> dependency: std/io Loading module: std/io]`) is internal debugging info formatted as a Go slice, not structured user-facing output.

**Impact:**
- AI agents writing standalone AILANG files in benchmark directories hit MOD010 errors on first attempt, wasting a repair turn (or failing outright)
- Benchmark pass rates for AILANG are lower than they should be due to tooling friction, not language capability
- New users writing scratch files outside a project tree get the same poor experience

## Goals

**Primary Goal:** Make module loading errors actionable so that AI agents (and humans) can self-recover on the first attempt without consulting documentation.

**Success Metrics:**
- MOD010 error message includes the exact CLI flag/env var to fix it, visible without unwrapping nested errors
- Files outside project tree auto-relax or get a one-line actionable error (no search trace noise)
- Stdlib version mismatch warning is suppressed (demoted to debug) when `--relax-modules` is active
- Benchmark re-run with same agent prompts shows fewer module-related failures

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Auto-relax for files outside project tree | Changes default strictness semantics; could mask real mismatches in non-temp dirs | human | design | med |
| Search trace visibility (remove vs demote) | Affects debugging of loader issues; removing entirely could hurt maintainers | agent | compile | low |
| Stdlib version warning suppression scope | Could hide real version problems if suppressed too broadly | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Auto-relax heuristic: expand `IsTempPath` to also detect "outside project tree" or add a separate `isStandaloneFile` check
- [ ] Whether auto-relax should be opt-out (always on for non-project files) or opt-in with better error message

## Solution Design

### Overview

Three changes to module loading:

1. **Improve MOD010 error formatting** -- ensure the actionable suggestion text is the first thing visible, not buried under search traces
2. **Expand auto-relax heuristic** -- detect files outside any AILANG project (no `ailang.toml` in parent dirs) and auto-relax with a demoted warning
3. **Suppress stdlib version mismatch under --relax-modules** -- demote to debug-level when relaxation is active

### Architecture

**Components:**
1. **Error formatter** (`internal/pipeline/pipeline_module.go`): Restructure the MOD010 error path so the suggestion appears first, search trace is demoted
2. **Project detection** (`internal/loader/loader.go`): New `IsOutsideProject(path)` function that walks parent directories looking for `ailang.toml` or `.ailang/`
3. **Stdlib warning gating** (`internal/loader/stdlib_resolver.go`): Accept a `relaxModules` flag and suppress version mismatch warnings when active

### Implementation Plan

**Phase 1: Error message restructuring** (~1.5 hours)
- [ ] In `LoadAll` (`internal/loader/loader.go:464`), stop embedding the raw search trace in the user-facing error. Move search trace to a structured `ModuleError.Trace` field that is only printed when `--trace-loader` or `DEBUG_LOADER=1` is set.
- [ ] In `pipeline_module.go` `validateMOD010`, restructure the error to lead with the actionable fix:
  ```
  Error MOD010: module 'minigit/main' doesn't match file path 'workspace/solution'.
  Fix: Use --relax-modules flag or set AILANG_RELAX_MODULES=1
  Alt: Rename module declaration to: module workspace/solution
  ```
- [ ] In the `pipeline_module.go` `module loading error:` wrapper, preserve the inner error's message without adding search trace noise.

**Phase 2: Expand auto-relax heuristic** (~1.5 hours)
- [ ] Add `IsOutsideProject(filePath string) bool` to `internal/loader/loader.go`. Walks parent dirs of `filePath` looking for `ailang.toml`. If none found, returns true.
- [ ] In `pipeline_module.go` `validateMOD010`, add `IsOutsideProject` as a third relaxation condition alongside `cfg.RelaxModules` and `IsTempPath`.
- [ ] Add a new `warnMOD010Relaxed` reason: `"no-project"` with message: `"Auto-relaxed: no ailang.toml found in parent directories. For strict checking, create an ailang.toml in the project root."`
- [ ] Add tests for `IsOutsideProject` with various directory layouts.

**Phase 3: Stdlib version warning suppression** (~1 hour)
- [ ] Thread a `relaxModules bool` parameter through `StdlibResolver` (add field to struct, set during construction or via setter).
- [ ] In `ResolveStdlib` (line 197), when `relaxModules` is true, skip the version mismatch warning entirely (or only emit at debug level via `DEBUG_LOADER=1`).
- [ ] Wire the `RelaxModules` config through `NewModuleLoader` -> `StdlibResolver` in `pipeline_module.go`.

### Files to Modify/Create

**New files:**
- None

**Modified files:**
- `internal/loader/loader.go` - Add `IsOutsideProject()`, modify `LoadAll` error formatting (~30 LOC)
- `internal/pipeline/pipeline_module.go` - Restructure MOD010 error messages, add no-project auto-relax (~40 LOC)
- `internal/loader/stdlib_resolver.go` - Add `relaxModules` field, gate version warning (~15 LOC)
- `internal/loader/loader_test.go` - Tests for `IsOutsideProject` (~40 LOC)
- `internal/pipeline/pipeline_module_test.go` - Tests for restructured error messages (~30 LOC)

## Examples

### Example 1: MOD010 error for standalone file (strict mode)

**Before:**
```
Error: module loading error: failed to load /home/user/workspace/solution.ail (search trace: [Loading module: /home/user/workspace/solution.ail]): MOD010: module declaration 'minigit/main' doesn't match canonical path 'home/user/workspace/solution'
Suggestions:
  1. Rename module to: module home/user/workspace/solution
  2. Move file to: minigit/main.ail
  3. For temp/scratch files: use --relax-modules or AILANG_RELAX_MODULES=1
```

**After (auto-relaxed, no ailang.toml found):**
```
WARNING MOD010 (no-project): module 'minigit/main' does not match file path 'home/user/workspace/solution'
  Auto-relaxed: no ailang.toml in parent directories.
```
(Program continues to run successfully.)

### Example 2: MOD010 error when project exists (strict mode, actionable)

**Before:**
```
Error: module loading error: failed to load /project/src/foo.ail (search trace: [Loading module: /project/src/foo.ail]): MOD010: module declaration 'bar/baz' doesn't match canonical path 'src/foo'
```

**After:**
```
Error MOD010: module 'bar/baz' doesn't match file path 'src/foo'.
  Fix: use --relax-modules or AILANG_RELAX_MODULES=1
  Alt: rename module declaration to: module src/foo
  Alt: move file to: bar/baz.ail
```

### Example 3: Stdlib version mismatch under --relax-modules

**Before:**
```
Warning: stdlib version mismatch: expected v0.10.0, found v0.9.0 at /usr/local/share/ailang/std
42
```

**After (with --relax-modules):**
```
42
```
(Warning suppressed. Only visible with `DEBUG_LOADER=1`.)

## Success Criteria

- [ ] MOD010 error in strict mode leads with "Fix: use --relax-modules" on its own line, no search trace visible
- [ ] Files outside a project tree (no `ailang.toml` in ancestors) auto-relax with a single-line warning
- [ ] Stdlib version mismatch warning is not emitted when `--relax-modules` or auto-relax is active
- [ ] Search trace only visible with `--trace-loader` or `DEBUG_LOADER=1`
- [ ] All existing tests pass (`make test`)
- [ ] New unit tests for `IsOutsideProject` and restructured error format
- [ ] `make verify-examples` passes

## Testing Strategy

**Unit tests:**
- `IsOutsideProject` with: dir containing `ailang.toml`, dir without, nested subdir of project, `/tmp/` path
- MOD010 error string format: verify "Fix:" line appears, search trace absent
- Stdlib resolver with `relaxModules=true`: verify no stderr warning on version mismatch

**Integration tests:**
- Run `ailang run` on a file in `/tmp/` -- should auto-relax (existing behavior, regression test)
- Run `ailang run` on a file in a non-project directory -- should auto-relax (new behavior)
- Run `ailang run --relax-modules` with mismatched stdlib version -- should not emit version warning

**Manual testing:**
- Run ai-coding-lang-bench minigit benchmark with an agent and verify the module error is either auto-relaxed or the error message is clear enough for the agent to self-correct

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact wording of error messages -- agent may choose, but must include `--relax-modules` and `AILANG_RELAX_MODULES=1` verbatim
- Whether `IsOutsideProject` caches results -- agent may choose (performance is not critical since it runs once per file)
- Whether to add `DEBUG_LOADER=1` as a new debug flag or reuse `--trace-loader` for search trace visibility -- agent may choose

## Non-Goals

**Not attempted in this feature:**
- Changing the module system semantics (path resolution, canonical ID computation) -- too high risk, separate design needed
- Making `--relax-modules` the default for all commands -- would break strict checking for real projects
- Structured JSON error output for MOD010 -- already handled by `--json` flag, orthogonal to this work
- Improving error messages for import resolution failures (LDR001) -- separate issue, though the search trace cleanup helps

## Timeline

**Day 1** (4 hours):
- Phase 1: Error message restructuring (1.5h)
- Phase 2: Auto-relax heuristic (1.5h)
- Phase 3: Stdlib warning suppression (1h)

**Total: ~4 hours in 1 day**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Auto-relax hides real module path bugs in non-project dirs | Med | Warning still emitted; only error->warning, not silent. Real projects have ailang.toml. |
| `IsOutsideProject` parent-dir walk is slow on deep paths | Low | Cap walk depth at 20 levels; `/` always returns true (no ailang.toml at root). |
| Suppressing stdlib version warning masks real incompatibility | Low | Only suppressed under explicit --relax-modules or auto-relax; strict mode unaffected. |

## Related Documents

<!-- Auto-populated by Ollama neural search on "module error messages" -->

**Implemented (may inform design):**
- [design_docs/implemented/v0_5_10/m-codegen-cross-module-impl.md](design_docs/implemented/v0_5_10/m-codegen-cross-module-impl.md) (0.40)
- [design_docs/implemented/v0_9_0/m-module-scope.md](design_docs/implemented/v0_9_0/m-module-scope.md) (0.38)
- [design_docs/implemented/v0_2_0/m_r1_module_execution.md](design_docs/implemented/v0_2_0/m_r1_module_execution.md) (0.37)

**Planned (check for overlap):**
- [design_docs/planned/v0_11_0/m-arch5-error-handling-strategy.md](design_docs/planned/v0_11_0/m-arch5-error-handling-strategy.md) (0.41)
- [design_docs/planned/v0_11_0/m-codegen-multimodule-bugs-sprint-plan.md](design_docs/planned/v0_11_0/m-codegen-multimodule-bugs-sprint-plan.md) (0.32)
- [design_docs/planned/v0_11_0/m-typeenv-sub-fix.md](design_docs/planned/v0_11_0/m-typeenv-sub-fix.md) (0.31)

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- `internal/pipeline/pipeline_module.go` - MOD010 validation and warning logic
- `internal/loader/loader.go` - `LoadAll`, `IsTempPath`, `CanonicalModuleID`
- `internal/loader/stdlib_resolver.go` - Stdlib resolution and version checking
- `internal/eval_harness/runner.go:327` - Eval harness already uses `--relax-modules`
- `eval_results/race_condition_test/` - Example of search trace in real eval stderr
- ai-coding-lang-bench repository - Benchmark that discovered these issues

## Future Work

- Structured error catalog: assign error codes to all loader errors (not just MOD010) with machine-readable fix suggestions
- LSP integration: show MOD010 as a diagnostic with quick-fix action to add `--relax-modules` to launch config
- Auto-infer module name from file path when no `module` declaration is present (would eliminate the error entirely for scratch files, but requires parser changes)

---

**Document created**: 2026-03-30
**Last updated**: 2026-03-30
