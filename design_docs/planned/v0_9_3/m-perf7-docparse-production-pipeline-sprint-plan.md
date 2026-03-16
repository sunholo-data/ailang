# M-PERF7 Sprint Plan: DocParse Production Pipeline

**Sprint ID**: M-PERF7
**Duration**: 2 days (estimated 10h implementation + 4h testing)
**Risk Level**: Low — follows established patterns (list_iterative.go for builtins, existing CLI flag infrastructure)
**Design Doc**: [m-perf7-docparse-production-pipeline.md](m-perf7-docparse-production-pipeline.md)

## Scope

**In scope (2 tracks):**
- **Track 2**: `foldChars` + `charAt` Go builtins for fast character-level parsing
- **Track 3 Option C**: `--batch` flag for `ailang run` (compile once, execute many)

**Out of scope:**
- Track 1 (cached type-checking) — separate design doc M-INCREMENTAL-TYPECHECK
- Track 3 Option A (coordinator ailang-script executor) — future work
- Track 3 Option B (persistent compiler daemon) — v0.10.0+

## Velocity Baseline

Recent sprints (last 7 days):
- M-DOCPARSE-DX: 4 milestones in ~2 days (~400 LOC)
- M-PERF6: 4 milestones in ~1.5 days (~350 LOC)
- M-ITERATIVE-LIST: 1 milestone in ~0.5 day (~220 LOC)

Average: ~200 LOC/day for builtins, ~150 LOC/day for CLI work. This sprint estimates ~300 LOC total across 2 days — well within velocity.

---

## Milestone 1: String Character Builtins (Day 1, ~5h)

**Goal**: Add `_str_foldChars` and `_str_charAt` Go builtins for character-level processing without list allocation overhead.

### Pattern

Follows `internal/builtins/list_iterative.go` exactly:
- `_str_foldChars` mirrors `_list_foldl`: uses `ctx.FnCallerN(fn, []eval.Value{acc, charStr})` per rune
- `_str_charAt` is a simple pure builtin (no closure invocation needed)
- `_str_len` already exists — no work needed

### Tasks

1. Create `internal/builtins/string_char.go` (~100 LOC)
   - `registerStrFoldChars()` — Type: `forall a. ((a, string) -> a, a, string) -> a`, uses `FnCallerN`
   - `registerStrCharAt()` — Type: `(string, int) -> string`, pure, rune-safe indexing
   - Follow `RegisterEffectBuiltin` + `BuiltinSpec` pattern from list_iterative.go
   - Include `BuiltinMetadata` with descriptions, params, tags

2. Create `internal/builtins/string_char_test.go` (~100 LOC)
   - `foldChars` tests: empty string, ASCII, Unicode (emoji 🎉, CJK 中文), accumulation (char count, string reversal)
   - `charAt` tests: first/last/middle, out-of-bounds error, multi-byte runes, empty string
   - Determinism: `-count=20`

3. Add wrappers to `std/string.ail` (~6 LOC)
   - `export pure func foldChars(f, acc, s) = _str_foldChars(f, acc, s)`
   - `export pure func charAt(s, i) = _str_charAt(s, i)`

4. Create `examples/string_chars.ail` (~20 LOC)
   - Character counting, string reversal via foldChars
   - charAt indexing example

5. Verify: `make test && make lint && make verify-examples`

### Acceptance Criteria

- `foldChars(\acc c -> acc ++ c, "", "hello")` returns `"hello"`
- `foldChars(\n _ -> n + 1, 0, "🎉中文")` returns `3` (correct rune count)
- `charAt("hello", 0)` returns `"h"`, `charAt("hello", 4)` returns `"o"`
- `charAt("🎉x", 1)` returns `"x"` (rune-safe, not byte-offset)
- `charAt("hello", 5)` returns error (out of bounds)
- Tests pass with `-count=20`
- `make test && make lint` pass

---

## Milestone 2: Batch CLI Mode (Day 1-2, ~5h)

**Goal**: Add `--batch` flag to `ailang run` that compiles modules once and executes the entrypoint once per input file, passing each as a program argument.

### Architecture

Currently `runFile()` does:
1. `os.ReadFile(filename)` — read source
2. `pipeline.RunWithContext(ctx, cfg, src)` — parse + typecheck + monomorph (~2s, 19 modules)
3. `runtime.NewModuleRuntime(".")` — create runtime
4. `executeModuleEntrypoint(rt, execParams)` — run entry function

**Batch mode**: Steps 1-3 happen once. Step 4 repeats per input file with fresh `programArgs`.

### Key Design Decision: Runtime Reset

Between executions, need to reset:
- `effCtx.ProgramArgs` — update to current file's args
- Effect budgets — reset per file (or accumulate — design choice)
- Trace collector — new trace per file (if enabled)
- **NOT** reset: compiled modules, evaluator, builtins

### Tasks

1. Add `--batch` flag to `runCommand()` in `cmd/ailang/main.go` (~5 LOC)
   - `batchFlag := fs.Bool("batch", false, "Process multiple inputs: compile once, run entrypoint per input")`
   - Pass to `runFile()` (or new `runBatch()` function)

2. Extract compile step into reusable function (~30 LOC refactor)
   - `compileFile(filename, cfg) → (result, evaluator, builtins, error)`
   - Both single-file and batch modes call this

3. Implement `runBatch()` in `cmd/ailang/main.go` (~80 LOC)
   - Compile once via `compileFile()`
   - For each remaining arg (batch inputs):
     - Create fresh `effCtx` with `programArgs = [inputFile]`
     - Wire capabilities, handlers (reuse setup functions)
     - Call `executeModuleEntrypoint()`
     - Print separator between outputs (e.g., `--- file.docx ---`)
     - On error: print error, continue to next file (don't abort batch)
   - Report summary: `N files processed, M errors`

4. Create `internal/builtins/string_char_test.go` integration tests (~40 LOC)
   - Batch of 3 test files, verify all produce output
   - Batch with 1 bad file, verify others still run
   - Batch with no files → error message

5. Create `examples/batch_processing.ail` (~15 LOC)
   - Simple entrypoint that reads `getArgs()` and processes input

6. Update CLI help text in `cmd/ailang/main.go`

7. Verify: `make test && make lint && make verify-examples`

### Acceptance Criteria

- `ailang run eval.ail --batch file1.docx file2.xlsx file3.csv` compiles once and runs 3 times
- Compilation happens once (verify via `--debug-compile` showing cache stats once)
- Each file gets its own `getArgs()` value
- Error in one file doesn't abort others
- `--batch` with zero files prints usage error
- `--batch` with one file works (same as non-batch)
- All existing `ailang run` tests still pass (no regression)
- `make test && make lint` pass

---

## Milestone 3: Benchmarks + CHANGELOG (Day 2, ~2h)

**Goal**: Verify performance improvements and document results.

### Tasks

1. Benchmark `foldChars` vs `toChars` + `foldl` on representative strings
   - 100 chars, 1K chars, 10K chars
   - Measure time + allocations
   - Document in design doc "Implementation Results" section

2. Benchmark batch mode
   - 5 identical small files: batch vs 5 separate invocations
   - Measure wall time difference
   - Calculate startup amortization

3. Update CHANGELOG.md
   - New builtins: `foldChars`, `charAt`
   - New CLI feature: `--batch` flag
   - Performance metrics

4. Update design doc with "Implementation Results" section

5. Send response to docparse inbox with migration guide

### Acceptance Criteria

- `foldChars` at least 3x faster than `toChars` + `foldl` for 1K+ chars
- Batch mode at least 3x faster than N separate invocations for 5+ files
- CHANGELOG.md updated
- Design doc updated with benchmark results
- DocParse inbox notified

---

## Files Summary

**New files (~280 LOC):**
| File | LOC | Purpose |
|------|-----|---------|
| `internal/builtins/string_char.go` | ~100 | foldChars + charAt builtins |
| `internal/builtins/string_char_test.go` | ~100 | Unit tests |
| `examples/string_chars.ail` | ~20 | Character processing examples |
| `examples/batch_processing.ail` | ~15 | Batch mode example |

**Modified files (~120 LOC):**
| File | LOC | Purpose |
|------|-----|---------|
| `cmd/ailang/main.go` | ~100 | --batch flag + runBatch() + refactor |
| `std/string.ail` | ~6 | foldChars + charAt wrappers |
| `CHANGELOG.md` | ~15 | Release notes |

**Total: ~400 LOC** (within 2-day velocity at 200 LOC/day)

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| `FnCallerN` not available in string builtin context | Low | High | Proven pattern from list_iterative.go — same wiring |
| Batch mode leaks state between runs | Med | High | Fresh `effCtx` per file; test with side-effecting programs |
| `runFile()` signature too long for refactor | Low | Low | Extract config struct if needed |
| `charAt` O(n) rune conversion | Low | Low | Document; recommend `foldChars` for sequential access |

---

## Day-by-Day Schedule

### Day 1
- **Morning**: M1 — Implement `foldChars` + `charAt` builtins, tests, wrappers
- **Afternoon**: M2 — Add `--batch` flag, refactor compile step, implement `runBatch()`

### Day 2
- **Morning**: M2 — Integration tests for batch mode, examples
- **Afternoon**: M3 — Benchmarks, CHANGELOG, design doc update, docparse notification

---

**Sprint plan created**: 2026-03-16
