# M-PERF7: DocParse Production Pipeline Optimization

**Status**: Planned
**Target**: v0.9.3
**Priority**: P2 (Enhancement — DocParse functional, but CI overhead significant)
**Estimated**: 4-5 days (16h implementation + 8h testing + 4h docs)
**Dependencies**: M-INCREMENTAL-TYPECHECK (Phase 2), M-DOCPARSE-DX M3 (listDir)
**Milestone ID**: M-PERF7
**Created**: 2026-03-16
**Source**: DocParse agent message `839ae1dd` (performance profile, 2026-03-16)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Pure performance optimizations; cached compilation produces identical results |
| A2: Replayability | 0 | Traces unchanged |
| A3: Effect Legibility | 0 | No new effects; daemon mode reuses existing FS capability |
| A4: Explicit Authority | 0 | No new capabilities |
| A5: Bounded Verification | +1 | Cached type-checking is still local, just faster |
| A6: Safe Concurrency | 0 | Coordinator parallelism uses isolated worktrees, no shared mutable state |
| A7: Machines First | +2 | Eliminates 108s wasted startup in CI; enables batch processing; AI agents process documents faster |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | `--debug-compile` shows cache hits; batch mode reports per-file timing |
| A10: Composability | +1 | `foldChars` composes with existing HOF pattern; batch mode composes with coordinator |
| A11: Structured Failure | 0 | Error handling unchanged |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): Cached compilation produces bit-identical output; foldChars is pure
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Directly improves machine throughput

---

## Related Documents

**Directly relevant:**
- [m-incremental-typecheck.md](m-incremental-typecheck.md) — Phase 2 cached type-checking (types.Type JSON serialization). Track 1 of this doc depends on it.
- [m-perf5-data-intensive-workloads.md](../v0_9_2/m-perf5-data-intensive-workloads.md) — Bulk XML ops, string join. Already implemented.
- [m-docparse-dx.md](m-docparse-dx.md) — stdlib/DX improvements from DocParse feedback. M3 (listDir) enables Track 3.
- [m-perf6-compilation-performance.md](../../implemented/v0_9_3/m-perf6-compilation-performance.md) — Cache infrastructure (Phase 1). Content-addressed keys, manifest.

**Coordinator architecture:**
- [coordinator.md](../../../docs/docs/guides/coordinator.md) — Daemon architecture, worktree isolation, executor providers.

---

## Systemic Audit

**Is this a one-off or part of a pattern?**

Yes — this is the **convergence point** of three optimization tracks:

1. **M-PERF3→M-PERF5→M-PERF6**: Interpreter performance, bulk ops, compilation cache infrastructure
2. **M-DOCPARSE-DX**: stdlib gaps for real-world document processing
3. **Coordinator**: Task dispatch infrastructure with worktree isolation

All three were built independently. This design doc connects them into a production pipeline: cached compilation (fast startup) + batch/daemon mode (no startup) + coordinator dispatch (parallel processing).

**Pattern**: DocParse keeps hitting the same wall — per-invocation overhead dominates when processing many files. The systemic fix is to eliminate per-invocation cost entirely.

---

## Problem Statement

**DocParse performance profile (March 2026) — post M-PERF5 + iterative builtins:**

| File | Size | Time | Parsing work |
|------|------|------|-------------|
| tiny.csv | <1KB | 2.0s | ~0s (pure startup) |
| sample.pptx | 40KB | 1.5s | ~0s |
| poi_two_sheets.xlsx | 8KB | 1.7s | ~0s |
| sample.docx | 8KB | 3.2s | ~1.2s |
| test_code.md | 4KB | 3.4s | ~1.4s |
| docx-hdrftr.docx | 24KB | 3.8s | ~1.8s |
| moby_dick.epub | 800KB | 10.4s | ~8.4s |
| poi_many_merges.xlsx (5K rows) | 829KB | ~2min | ~1:58 |

**Three distinct bottlenecks:**

1. **Fixed startup cost: ~2s per invocation.** Type-check of 19 modules + effect-check + runtime init. For eval running 54 files, that's ~108s wasted (1.8 min). The cache infrastructure exists (M-PERF6) but can't skip compilation yet (blocked on `types.Type` JSON serialization).

2. **Character-level parsing: markdown_parser is 1.4s for 4KB.** Uses `foldl` state machine over character list. Converting string→[char] then iterating creates N list nodes. Need Go-native `foldChars(f, acc, s)` or `charAt(s, i)` to bypass list overhead.

3. **Repeated invocations: 54 files × 2s = 108s startup overhead.** DocParse eval spawns one `ailang` process per file because there's no `listDir` (M-DOCPARSE-DX M3) and no batch mode.

**Impact:**
- DocParse CI takes ~4 minutes for 54 files (108s startup + actual parsing)
- Small files (<50KB) are 80-95% startup overhead
- Character-level parsers (markdown, eventually others) hit interpreter overhead wall
- No way to amortize startup across multiple files

---

## Goals

**Primary Goal:** Reduce DocParse full eval time from ~4 minutes to <1 minute.

**Success Metrics:**
- Small file (<50KB) processing: <0.5s (from 2-4s) — via cached type-checking or daemon mode
- Full 54-file eval: <60s total (from ~240s) — via batch mode eliminating startup overhead
- Markdown 4KB parsing: <0.5s (from 3.4s) — via `foldChars` Go builtin
- Optional: parallel batch processing 3x throughput via coordinator dispatch

---

## Solution Design

### Track 1: Cached Type-Checking Integration (Depends on M-INCREMENTAL-TYPECHECK)

**This track is blocked on M-INCREMENTAL-TYPECHECK Phase 2** (types.Type JSON serialization). When that lands:

- Cache already detects 4/4 HIT on repeat runs of docparse modules
- Pipeline skip logic (~20 LOC) loads `CompileUnit` from disk instead of recompiling
- Expected: warm-cache startup drops from ~2s to ~0.4s (Go runtime + cache load)
- 54-file eval: 108s startup → ~22s startup (5x improvement)

**No new work needed here** — M-INCREMENTAL-TYPECHECK covers this. Listed for completeness.

### Track 2: String Character Builtins (`foldChars`, `charAt`)

**Problem:** DocParse's markdown_parser uses `foldl` over character lists:
```ailang
-- Current: string → [char] → foldl → slow
let chars = toChars(text)  -- allocates N list nodes
let result = foldl(\acc c -> parseChar(acc, c), initState, chars)
```

For a 4KB markdown file (~4000 chars), this creates 4000 `CharValue` nodes, 4000 cons cells, then iterates them with full evaluator overhead per step.

**Solution:** Two Go builtins that iterate over string bytes/runes in Go:

```go
// _str_foldChars(f: (a, string) -> a, acc: a, s: string) -> a
// Iterates over string runes in Go, calling AILANG closure per rune
func strFoldCharsImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    fn := args[0]   // AILANG closure: (acc, char_string) -> acc
    acc := args[1]   // Initial accumulator
    s := args[2].(*eval.StringValue).Value

    for _, r := range s {
        charStr := &eval.StringValue{Value: string(r)}
        var err error
        acc, err = applyFunction(ctx, fn, []eval.Value{acc, charStr})
        if err != nil { return nil, err }
    }
    return acc, nil
}

// _str_charAt(s: string, i: int) -> string
// O(1) byte access (O(n) for rune — document this)
func strCharAtImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
    s := args[0].(*eval.StringValue).Value
    i := args[1].(*eval.IntValue).Value
    runes := []rune(s)
    if i < 0 || i >= int64(len(runes)) {
        return nil, fmt.Errorf("charAt: index %d out of range [0, %d)", i, len(runes))
    }
    return &eval.StringValue{Value: string(runes[i])}, nil
}
```

**AILANG wrappers:**
```ailang
-- std/string.ail additions
export pure func foldChars(f: (a, string) -> a, acc: a, s: string) -> a =
    _str_foldChars(f, acc, s)

export pure func charAt(s: string, i: int) -> string =
    _str_charAt(s, i)

export pure func stringLength(s: string) -> int =
    _str_length(s)
```

**Expected impact:** Eliminates N list node allocations + list cons/pattern matching. For 4KB markdown: ~1.4s → ~0.3s (closure call overhead per char remains, but list overhead gone).

**Note:** `_str_length` is needed for `charAt`-based loops. Check if it already exists as a builtin.

### Track 3: Batch/Daemon Mode via Coordinator

**Key insight:** The coordinator daemon is already running with worktree isolation, task queuing, and executor dispatch. We just need a new executor type.

#### Option A: `ailang-script` Executor (Recommended — small scope)

Add a new executor provider alongside `claude` and `gemini` that runs `.ail` files:

```go
// internal/executor/ailang/ailang.go
package ailang

import (
    "github.com/sunholo/ailang/internal/executor"
)

type AilangExecutor struct {
    workspace string
}

func (e *AilangExecutor) Execute(ctx context.Context, task executor.Task) (executor.Result, error) {
    // Parse task content for: file path, arguments, stdin
    // Run: ailang run <file> [args...]
    // Capture stdout/stderr
    // Return structured result
}

func Register() {
    executor.GlobalFactory().Register("ailang", func(cfg executor.Config) (executor.Executor, error) {
        return &AilangExecutor{workspace: cfg.Workspace}, nil
    })
}

func init() { Register() }
```

Auto-import in `provider_executor.go`:
```go
_ "github.com/sunholo/ailang/internal/executor/ailang"
```

**Usage via coordinator:**
```bash
# Send batch processing task
ailang messages send coordinator "Process documents in golden/" \
    --title "DocParse batch: 54 files" --from "docparse-ci"

# Coordinator dispatches as ailang-script task:
# - Creates worktree
# - Runs: ailang run eval.ail -- golden/file1.docx golden/file2.xlsx ...
# - Collects results
# - Reports back via message
```

**With `MaxWorktrees: 3`**, the coordinator can process 3 batches in parallel. For 54 files split into 3 batches of 18: 3x throughput.

**Limitation:** Each executor invocation still spawns a new `ailang` process. The startup overhead remains unless cached type-checking lands first.

#### Option B: Persistent Compiler Daemon (larger scope, future)

```bash
# Start AILANG compiler daemon (keeps modules loaded in memory)
ailang daemon --socket /tmp/ailang.sock

# Client sends run requests
ailang run --daemon eval.ail -- golden/file1.docx
```

The daemon loads and type-checks modules once, then serves compilation requests via Unix socket. Each request skips module loading entirely.

**Expected:** <100ms per small file (vs 2s currently). This is the ultimate fix but requires:
- Module hot-reload on source change
- Connection pooling / request queuing
- Process lifecycle management (pid file, health check, auto-restart)
- The existing `ailang serve-api` server provides a starting point

**Recommendation:** Defer Option B to v0.10.0+. Option A + cached type-checking covers 80% of the need.

#### Option C: Batch CLI Mode (simplest, immediate)

No coordinator involvement. Just a `--batch` flag:

```bash
# Process multiple files in one invocation
ailang run eval.ail --batch golden/*.docx golden/*.xlsx

# Or via stdin
find golden/ -name "*.docx" | ailang run eval.ail --batch --stdin
```

The runtime loads modules once, then runs the entry function once per input file. Startup amortized across all files.

**Implementation:** ~50 LOC in `cmd/ailang/run.go`:
```go
if batchMode {
    // Load and compile modules once
    program := compile(file)
    for _, input := range batchInputs {
        // Reset runtime state, run with new input
        result := execute(program, input)
        fmt.Println(result)
    }
}
```

**Expected:** 54 files × 2s = 108s → 2s + 54 × (parse time) ≈ 30-40s. No coordinator needed.

**Recommendation:** Implement Option C first (simplest, highest ROI), then Option A for integration with coordinator workflows.

---

## Implementation Plan

### Day 1: Track 2 — foldChars + charAt Builtins ✅ COMPLETE (2026-03-16)

- [x] Add `_str_foldChars` Go builtin with closure invocation — uses `ctx.FnCallerN` (same as `_list_foldl`)
- [x] Add `_str_charAt` Go builtin (rune-safe indexing with bounds checking)
- [x] `_str_length` not needed — `_str_len` already exists in `string.go`
- [x] Add `foldChars`/`charAt` wrappers to `std/string.ail`
- [x] 14 unit tests: empty string, ASCII, Unicode (emoji 🎉), 10K char stress, error propagation, type validation
- [x] Tests pass with `-count=20` — fully deterministic
- [x] Benchmark: 10K chars folded in ~515µs (0.5ms) — well within <0.5s target

**Files created:**
- `internal/builtins/string_char.go` (144 LOC) — `_str_foldChars`, `_str_charAt` implementations + registration
- `internal/builtins/string_char_test.go` (246 LOC) — 14 tests + 1 benchmark

**Files modified:**
- `std/string.ail` — added `foldChars`, `charAt` exports

**Design note:** Created `string_char.go` as a new file because `string.go` was at 1163 lines (near 1200-line limit).

### Day 2: Track 3 Option C — Batch CLI Mode

- [ ] Add `--batch` flag to `ailang run` command
- [ ] Implement compile-once, execute-many loop in runner
- [ ] Handle per-file error reporting (don't abort on single failure)
- [ ] Add `--batch --stdin` for pipe-based input
- [ ] Test: `ailang run eval.ail --batch test1.docx test2.xlsx`
- [ ] Benchmark: 10-file batch vs 10 separate invocations

### Day 3: Track 3 Option A — ailang-script Executor

- [ ] Create `internal/executor/ailang/ailang.go` with `AilangExecutor`
- [ ] Implement `Execute()`: parse task content, run `ailang run --batch`, collect results
- [ ] Add `init()` registration, auto-import in `provider_executor.go`
- [ ] Test: coordinator dispatches ailang-script task, receives result
- [ ] Test: parallel dispatch of 3 ailang-script tasks to 3 worktrees

### Day 4: Integration Testing + Documentation

- [ ] End-to-end: DocParse eval.ail with `--batch` flag across all 54 golden files
- [ ] End-to-end: coordinator dispatch of batched document processing
- [ ] Verify `foldChars` in markdown_parser (if DocParse updates their parser)
- [ ] CHANGELOG.md updated
- [ ] Example: `examples/batch_processing.ail`
- [ ] Send response to docparse inbox with migration guide

---

## Files to Modify/Create

**New files:**
- `internal/builtins/string_char.go` (144 LOC) — foldChars, charAt builtins ✅
- `internal/builtins/string_char_test.go` (246 LOC) — 14 tests + benchmark ✅
- `internal/executor/ailang/ailang.go` (~100 LOC) — ailang-script executor
- `internal/executor/ailang/ailang_test.go` (~80 LOC) — executor tests
- `examples/batch_processing.ail` (~30 LOC) — batch mode example

**Modified files:**
- `cmd/ailang/run.go` (~50 LOC) — `--batch` flag and compile-once loop
- `std/string.ail` (~10 LOC) — foldChars, charAt wrappers ✅ (stringLength not needed — `length` already wraps `_str_len`)
- `internal/coordinator/provider_executor.go` (~2 LOC) — auto-import ailang executor

---

## Examples

### Example 1: foldChars for markdown parsing

**Before (slow — 1.4s for 4KB):**
```ailang
import std/string (toChars)
import std/list (foldl)

let parseMarkdown = \text ->
    let chars = toChars(text)          -- allocates 4000 list nodes
    foldl(parseChar, initState, chars)  -- 4000 recursive calls
```

**After (fast — ~0.3s for 4KB):**
```ailang
import std/string (foldChars)

let parseMarkdown = \text ->
    foldChars(parseChar, initState, text)  -- iterates in Go, calls closure per rune
```

### Example 2: Batch mode for CI eval

**Before (slow — 108s startup overhead):**
```bash
# eval.sh — spawns 54 separate ailang processes
for file in golden/*; do
    ailang run eval.ail -- "$file"    # 2s startup each!
done
```

**After (fast — 2s startup total):**
```bash
ailang run eval.ail --batch golden/*
# Compiles once, runs 54 times. ~30s total.
```

### Example 3: Coordinator parallel batch

```bash
# Split 54 files into 3 batches, dispatch to coordinator
ailang messages send coordinator \
    "ailang run eval.ail --batch golden/batch1/*" \
    --title "DocParse batch 1/3" --from "docparse-ci"

# Coordinator runs 3 batches in parallel (3 worktrees)
# Total: ~15s (3x throughput) + coordinator overhead
```

---

## Success Criteria

- [x] `foldChars(\acc c -> acc ++ c, "", "hello")` returns `"hello"` — verified in unit test
- [x] `charAt("hello", 2)` returns `"l"` — verified in unit test
- [ ] Markdown 4KB parsing: <0.5s (from 3.4s) with foldChars — pending DocParse parser update
- [ ] `ailang run eval.ail --batch file1 file2 file3` processes all files in one invocation
- [ ] Batch mode startup: ~2s total (not 2s × N)
- [ ] Coordinator dispatches ailang-script tasks and returns results
- [ ] All tests passing (`make test`)
- [ ] Examples passing (`make verify-examples`)
- [ ] CHANGELOG.md updated
- [ ] Response sent to docparse inbox

---

## Testing Strategy

**Unit tests:**
- `foldChars`: empty string, ASCII, Unicode (emoji, CJK), large strings, closure that errors mid-iteration
- `charAt`: bounds checking, negative index, multi-byte runes, empty string
- Batch mode: single file, multiple files, file that errors (other files continue), no files (error)
- Executor: task parsing, process spawning, result collection, timeout handling

**Integration tests:**
- Batch mode with actual DocParse eval.ail on test files
- Coordinator + ailang-script executor end-to-end
- foldChars in a realistic parser (markdown or similar)

**Benchmarks:**
- foldChars vs toChars+foldl on 4KB, 40KB, 400KB strings
- Batch mode: 10 files batch vs 10 separate invocations (wall time comparison)

---

## Non-Goals

- **Cached type-checking implementation** — Covered by M-INCREMENTAL-TYPECHECK. This doc assumes it will land and describes the integration.
- **Persistent compiler daemon** — Option B is deferred to v0.10.0+. Too much scope for this sprint.
- **Parallel compilation within a single invocation** — Separate concern (concurrent module compilation).
- **AILANG-level concurrency primitives** — Determinism axiom (A1) prevents implicit parallelism. Coordinator provides external parallelism.
- **Regex engine** — foldChars covers character-level parsing; full regex is v0.10.0+.

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `foldChars` needs evaluator access (like `_list_map`) | Med | Follow M-ITERATIVE-LIST pattern — already solved for list builtins |
| Batch mode state leaks between runs | High | Reset runtime environment fully between each file execution |
| Coordinator ailang-script tasks hang | Med | Enforce task timeout (default 5min); coordinator already has timeout infra |
| `charAt` O(n) for rune conversion on each call | Low | Document as O(n); recommend `foldChars` for sequential access |
| Batch mode changes CLI contract | Low | `--batch` is opt-in; existing single-file behavior unchanged |

---

## Messages Addressed

| Message ID | Title | Tracks |
|------------|-------|--------|
| `839ae1dd` | Performance profile: startup 2s fixed cost, markdown slow | All 3 tracks |
| `a761543f` | Iterative builtins: 47% speedup, xlsxMaxRows 2K→5K | Context (already shipped) |
| `3fcc9756` | New: parseElements streaming XML parser | Context (already shipped) |
| `13ae1597` | M-ITERATIVE-LIST: Iterative list builtins shipped | Context (already shipped) |

---

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- [M-INCREMENTAL-TYPECHECK](m-incremental-typecheck.md) — Cached compilation (dependency)
- [M-PERF5: Data-Intensive Workloads](../v0_9_2/m-perf5-data-intensive-workloads.md) — Bulk XML ops
- [M-PERF6: Compilation Performance](../../implemented/v0_9_3/m-perf6-compilation-performance.md) — Cache infrastructure
- [M-DOCPARSE-DX](m-docparse-dx.md) — stdlib gaps including listDir
- [Coordinator Guide](../../../docs/docs/guides/coordinator.md) — Daemon architecture
- `internal/executor/factory.go` — Executor registration pattern
- `internal/coordinator/daemon_tasks_exec.go` — Task execution (Limit: 1)
- `internal/coordinator/worktree.go` — MaxWorktrees: 3

## Future Work

- **Persistent compiler daemon** (Option B): Keep modules loaded in memory, serve via Unix socket. <100ms per file. Requires hot-reload, connection pooling.
- **Coordinator cloud dispatch**: Cloud Run Jobs already support batch parallelism via Pub/Sub. ailang-script executor + cloud mode = serverless document processing at scale.
- **AILANG-level parallel map**: If determinism can be preserved (pure functions over independent data), a `pmap` builtin could parallelize within a single process. Requires careful axiom analysis.
- **Shared stdlib compilation cache**: All projects share cached stdlib modules. Further reduces warm-cache startup.

---

**Document created**: 2026-03-16
**Last updated**: 2026-03-16 (Track 2 implemented)
