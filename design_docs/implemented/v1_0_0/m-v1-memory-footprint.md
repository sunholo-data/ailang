# M-V1-MEMORY-FOOTPRINT: Memory efficiency audit and fixes for v1.0.0

**Status**: IMPLEMENTED
**Target**: v1.0.0
**Priority**: P1 — a 1 GiB container OOMs on an 8.7 MB workbook; "logging on" multiplies peak RSS 5–20×
**Estimated**: 4–5 days across three milestones in this repo, plus three downstream asks to ailang-parse
**Dependencies**: None. Complements (does not depend on) M-MEM-BUDGET-RUNTIME (a *bound*, not a *reduction*) and M-LIST-CONS-QUADRATIC (the structural list fix, parked).
**Lane**: AILANG fix (PROGRAM.md §4 — runtime substrate defects; no motoko surface).

Every claim below was re-derived on this machine at working-tree HEAD `bb1b45967` (dev) with the
installed `ailang v0.39.2-dirty`; measurements are peak RSS from `/usr/bin/time -l`. See the
[Verification Log](#verification-log).

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No evaluation-order or value change. Caps produce typed errors at deterministic byte counts; the cgroup-derived GC limit is a host-boundary control like `--timeout` (A12), not a semantic. |
| A2: Replayability | 0 | Trace events keep their schema; values were already truncated to 1 KB after rendering, so a bounded renderer yields byte-identical retained events. |
| A3: Effect Legibility | 0 | No new effects. `FS.readFile` gains a size cap on the existing effect. |
| A4: Explicit Authority | 0 | `--fs-max-bytes` and `--max-memory` are explicit, caller-set caps consistent with `Net.MaxBytes` (+). D-D as first drafted inferred a limit from the cgroup with only an opt-out — environment-detected, not caller-set (−); D-D is therefore recast as **opt-in** (`--max-memory cgroup` / `AILANG_MEMLIMIT=cgroup`), and the human freeze item decides whether the containers we own pass it. Net 0 until that ruling. |
| A5: Bounded Verification | 0 | No type-system change. |
| A6: Safe Concurrency | +1 | serve-api stops sharing one `DebugContext` across concurrent requests. |
| A7: Machines First | +1 | A run that dies by OS OOM leaves no structured error; capped reads and streamed logs give agents a coded failure and a complete log tail. |
| A8: Minimal Syntax | +1 | Zero new syntax. |
| A9: Cost Visibility | +1 | Trace retention accounting becomes honest (struct + header bytes counted), so the 256 MB cap means 256 MB. |
| A10: Composability | 0 | Orthogonal to budgets and effect rows. |
| A11: Structured Failure | +1 | `E_FS_BODY_TOO_LARGE`-style typed error replaces host death for oversize reads. |
| A12: System Boundary | 0 | The memory limit is documented as a host-boundary control. |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

## Problem Statement

Ahead of v1.0.0 the runtime's memory behaviour was audited end to end: value representation,
environments, recursion, tracing, effects, parser/loader, type checker, the serve-api upload path,
process-level GC controls, and the docparse/ailang-parse consumer that first exposed the problem.
The known symptom — "if logging is on, `ailang parse` on an Excel sheet blows up" — turned out
to be one of ten distinct causes, and the cheapest five live in this repo.

### What "logging on" actually is

The parse launcher and container both set `AILANG_NO_TRACE=1` for production
(`ailang-parse/bin/docparse:9`, `docparse/Dockerfile:110`). Neither parse repo calls `Debug.log`
at all (V13). So "logging" means the **in-process trace collector**, enabled by any of
`AILANG_NO_TRACE=0`, `AILANG_TRACE=standard|deep`, `--trace-tier`, `--emit-trace`, or an OTLP
endpoint being configured (`internal/runner/run.go:490-495` sets `emitTrace="auto"` when
telemetry is enabled — which it is on Cloud Run, where `GOOGLE_CLOUD_PROJECT` is set).

Three defects compound once the collector exists:

1. **Effect events render every argument and result to a full string before any check.**
   `internal/effects/ops.go:112-127` calls `String()` on all args and the result whenever
   `ctx.Trace.Enabled()`, at *every* tier, then `boundValues` truncates to 1 KB and discards.
   The function-call path was fixed for exactly this in M-TRACE-TIER-NOT-ENFORCED M1
   (`internal/eval/eval_operations.go:118-130` checks `RecordsFunctionCalls()` first); the
   effect path was not. An effect carrying a 45 MB worksheet string, a block tree or a list of
   zip entries allocates a full extra copy per call — at the default `standard` tier.
2. **The retention cap under-counts.** `eventSize` (`internal/trace/retention.go:128-146`)
   charges string bytes plus a flat 128 B per event. It ignores the `TraceEvent` struct, the
   heap `*FunctionEvent`/`*EffectEvent`, 16 B per string header and size-class rounding. The
   256 MB default (`retention.go:32`) is therefore 1–2 GB of real RSS before eviction fires.
3. **Peak is unbounded even where retention is bounded.** e4c42a7bf's own commit message says
   so: rendering materialises the whole value via `String()` before truncation, so deep tracing
   of a growing accumulator is O(n) allocation per call and O(n²) over a recursion. Measured
   there: n=800 retained ~40 MB of a 2061 MB peak.

### Measured on this machine (2026-09-16)

| Program | Peak RSS | Note |
|---|---|---|
| type-check only, GOGC=500 (`ailang run` default) | 78 MB | process floor with stdlib loaded |
| type-check only, GOGC=100 | 54 MB | the 6× heap headroom costs ~24 MB at the floor |
| int recursion, depth 9,000 | 113 MB | |
| int recursion, depth 90,000 (`--max-recursion-depth 100000`) | 671 MB | ≈ 6–7 KB of Go stack per AILANG frame |
| `n :: acc` recursion, depth 4,500 | 242 MB | |
| `n :: acc` recursion, depth 9,000 | 772 MB | 2× depth → 3.2× RSS: quadratic, and **live**, not garbage |
| same, `--max-memory 64MB` | 931 MB | 11 s of GC thrash, no reduction — the data is reachable |
| same at depth 20,000 | — | hits the 10,000 recursion cap (`RT_REC_003`) |
| `n :: acc` depth 9,000, `AILANG_TRACE=deep --emit-trace jsonl` | 2,282 MB | 3× the untraced peak |

Probe sources: [Verification Log V14](#verification-log). These reproduce the open upstream
reports #676 (6,400 conses = 2.6 GB) and the docparse xlsx analysis
(`docparse/design_docs/planned/xlsx_resource_tiers.md`: 8.68 MB workbook OOMs a 2 GiB container
at ~72 s; 11 sheets, 45 MB of sheet XML all resident at once).

### Findings catalogue

Severity **A** = superlinear or unbounded; **B** = linear but avoidable; **C** = fixed overhead.

| # | Finding | Where | Sev | Handled by |
|---|---|---|---|---|
| F1 | `::` copies the whole tail; no TCO in the tree walker; every recursion frame holds its own distinct accumulator copy → O(n²) **live** memory | `internal/builtins/list.go:87-106`; `eval_operations.go:55-60` (depth cap is the only backstop) | A | M-LIST-CONS-QUADRATIC (parked) — **not this doc** |
| F2 | Effect trace events rendered at every tier before truncation | `internal/effects/ops.go:112-127` | A | **M1** |
| F3 | `eventSize` under-counts; 256 MB default ≈ 1–2 GB real | `internal/trace/retention.go:32,128-146` | A | **M1** |
| F4 | Rendering is not early-terminating; `ListValue.String()`/`RecordValue.String()` serialise the whole structure | `internal/eval/value.go:89-100,154-178,255-266,363-378` | A | **M1** |
| F5 | `AILANG_TRACE_VALUES=off` applied after the render | `retention.go:104-107,205-217` | B | **M1** (falls out of F4) |
| F6 | `Debug.log` lines retained unbounded until flush; `--log-level` filters at flush; serve-api shares one `DebugContext` across concurrent requests | `internal/effects/debug.go:78-85`; `debug_sink.go:87-108`; `apiserver/server.go:313-323` | A | **M2** |
| F7 | `TypedEvaluator.trace.Entries` appends per call with no cap | `internal/eval/eval_typed_helpers.go:90` | A | **M2** |
| F8 | `FS.readFile` has no size cap and double-copies (`[]byte` → `string`); Net is capped at 5 MB | `internal/effects/fs.go:142-147,252,322` vs `context.go:195,231` | A | **M3** |
| F9 | serve-api multipart: `ParseMultipartForm(maxSize)` uses the 50 MB upload cap as the *in-memory* threshold, then `io.ReadAll` copies the part, then it is written to a temp file and re-read — three copies | `internal/apiserver/routes_dispatch.go:55-59,563-575,467-471` | A | **M3** |
| F10 | `_zip_readEntry`: `io.ReadAll` without pre-sizing to `UncompressedSize64`, then `string(data)` | `internal/builtins/zip.go:559-579,199` | B | **M3** |
| F11 | `--max-memory` exists but nothing sets it in containers; Go does not read the cgroup limit; `ailang run` sets GOGC=500 by default, so the heap may grow 6× inside a 1 GiB cgroup with no limit | `cmd/ailang/memory_limit.go`; `main_run_exec.go:30-35`; `ailang-parse/Dockerfile`, `docparse/Dockerfile:127` | A | **M4** |
| F12 | `AILANG_EVAL_MAX_RSS` watchdog covers only the eval harness | `internal/eval_harness/memlimit.go:106` | B | **M4** (documented, not extended) |
| F13 | String `++` is immutable concat; no builder type; docparse builds the whole document with a `foldl` string accumulator, which is also what deep tracing renders per call | `eval_operations.go:400-406`; `ailang-parse/docparse/services/markdown_writer.ail:46,273` | A | **Downstream ask D1**; builder type = Future Work |
| F14 | docparse xlsx: worksheets read whole (shared strings already stream), `sanitizeXml` second copy, `parseSheetsWithNames` non-tail so all sheets live at once, images base64-retained for the document lifetime | `xlsx_parser.ail:470-484,501`; `docx_parser.ail:825-832` | A | **Downstream asks D2, D3** |
| F15 | Strict `take(n, flatMap(...))` never bounds peak (#617); fused `_list_takeMap`/`_list_takeFlatMap` exist | `internal/builtins/list_bounded.go` | A | Non-goal here; prompt/stdlib guidance |
| F16 | Closures capture the whole `Environment` chain; a fresh `map` + `sync.RWMutex` per call; 4–5 `defer`s per call | `eval_expressions.go:178-185`; `env.go:25-30`; `eval_operations.go:60,143-153` | B | Future Work (measure first) |
| F17 | `MapValue.Insert` and record update each copy the whole map (O(n) per update, O(n²) to build incrementally). `Insert` pre-sizes (`make(..., len+1)`); record update does not (`make(map[string]Value)` with no hint, so it rehashes while copying) | `value.go:213-221`; `eval_expressions.go:411-425` | B | Future Work; the record-update capacity hint is a trivial M1 add-on |
| F18 | `LoadedModule` retains source text + surface AST + Core + Iface + CoreTI per module for the run | `internal/loader/loader.go:47-58` | B | Non-goal (intentional; error rendering + cache key) |

**Already in place and kept:** trace per-value cap 1 KB and retention eviction with a
`trace_truncated` marker; tier gating on the *function-call* render path; `Net.MaxBytes` 5 MB;
zip 100 MB/entry and 10,000 entries; recursion cap 10,000; `--max-memory`; `--memprofile` on
`ailang run`; `AILANG_EVAL_MAX_RSS`; iterative Go builtins for map/filter/foldl/reverse;
streaming `scanFold`/`scanFoldStep` zip-XML builtins; SSE/NDJSON streaming; `sync.Pool`s in the
type checker.

## Goals

**Primary Goal:** Tracing, logging and input handling must cost O(bounded) extra memory on top of
the program's own live data, and containers must run under a memory limit the Go runtime knows about.

**Success Metrics:**
- Deep trace of the depth-9,000 cons probe: peak RSS within **1.25×** of the untraced run (today 3×).
- Standard tier with `--emit-trace`: peak within **1.05×** of untraced (today: unmeasured regression on effect-heavy programs; the function-call path already meets this).
- `Debug.log` of 1,000,000 lines at `--log-level ERROR`: peak RSS flat (today: linear).
- A 50 MB multipart upload to serve-api: at most **one** in-memory copy of the body at any time (today: three).
- `ailang run --max-memory cgroup` inside a cgroup with `memory.max` set: `debug.SetMemoryLimit` is applied from it; the default stays unlimited.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D-A: Bound the *renderer*, not the record site — add `eval.ShowBounded(v, maxBytes)` (early-terminating writer) and make every trace render site use it | Fixes F2, F4, F5 in one mechanism and keeps `standard`-tier effect events (which `scorer`/`comparator` consume) byte-identical; gating effect events off at `standard` would change the trace artifact | agent | design | med |
| D-B: `DefaultMaxRetainedBytes` 256 MB → **32 MB** with honest `eventSize` | Changes the size of every `--emit-trace` artifact's in-memory tail; observers still see the complete stream, so exporters are unaffected, but anyone reading the retained tail after a long deep run sees less of it | **human** | design | low |
| D-C: `FS.readFile` cap — **default unbounded in the CLI** (no behaviour change), `--fs-max-bytes` flag, and serve-api sets it to its upload cap | A default cap would break existing programs silently; an unset cap leaves the server unprotected | **human** | design | low |
| D-D: `--max-memory cgroup` (and `AILANG_MEMLIMIT=cgroup`) derives the limit from Linux `memory.max` × 0.9 — **opt-in**, default unchanged (no limit) | Round-2 quorum (oc-glm-5-2) showed an opt-out default contradicts A4 ("caller-set"); opt-in keeps authority explicit at the cost of two downstream Dockerfile lines. This is best-effort GC tuning, **not** a request-failure mechanism (see M4) | **human** | design | low |
| D-E: `Debug.log` streams to the sink at log time when a sink is attached; `Collect()` semantics preserved for embedding hosts (game engine, WASM) | CLI users see log lines interleaved with program output instead of after it; `--log-level` moves to arrival time | **human** | design | med |
| D-F: Multipart — `ParseMultipartForm(4 MB)` + `http.MaxBytesReader` for the real cap; parts `io.Copy`'d to the temp file | Removes two of three copies; the on-disk temp file becomes the only full copy | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] D-B — retention default 32 MB (Mark 2026-09-16; exporters read the observer stream, which is unaffected)
- [x] D-C — FS cap: unbounded CLI, serve-api sets it to its upload cap (Mark 2026-09-16)
- [x] D-D — opt-in `--max-memory cgroup` (Mark 2026-09-16); the downstream Dockerfile lines go out as part of the D1–D3 messages
- [x] D-E — Debug.log streams at log time in CLI and serve-api; Collect semantics preserved for hosts (Mark 2026-09-16)

## Solution Design

### Overview

Three milestones in this repo, one process-level milestone, and three asks to ailang-parse.
Order is by RSS reduction per hour: M1 defuses the trace multiplier, M2 the unbounded
accumulators, M3 the input copies, M4 makes containers honest. Structural language changes
(cons cells, TCO, a string builder value) are explicitly out of scope and routed to their own
docs.

### Architecture

**M1 — bounded rendering and honest retention (`internal/eval`, `internal/effects`, `internal/trace`)**
- `eval.ShowBounded(v Value, maxBytes int) (s string, truncated bool)`: a `boundedWriter` that
  every container `String()` writes into via a new `WriteTo(w)`-style method; the writer returns
  `errBudgetExhausted` once `maxBytes` is exceeded and the traversal unwinds. Scalars unchanged.
- `Collector.ValueBudget() int` returns `maxValueBytes` (0 = redacted → render nothing).
  `internal/effects/ops.go` and both sites in `eval_operations.go` render through
  `ShowBounded(a, budget)`. `boundValues` stays as the retention-side guard (belt and braces).
- `eventSize` charges `unsafe.Sizeof(TraceEvent{})` + the payload struct + 16 B per string +
  string bytes. `DefaultMaxRetainedBytes` per D-B.
- Record update: `make(map[string]Value, len(baseRecord.Fields)+len(update.Updates))` at `eval_expressions.go:411` (F17 hint; `MapValue.Insert` already does this).

**M2 — unbounded accumulators (`internal/effects/debug.go`, `debug_sink.go`, `apiserver/server.go`, `eval_typed_helpers.go`)**
- `DebugContext` gains an optional `sink func(LogEntry)` and `minLevel`; `Log` filters by level
  on arrival and, when a sink is set, writes through instead of appending. `Collect()`/`Reset()`
  unchanged for hosts with no sink. `DebugSink.Flush` remains for the no-sink path.
- Structured-line severity parsed once (`json.Decoder` over a `strings.Reader`), not `json.Valid` + `json.Unmarshal`.
- serve-api: a `DebugContext` per request (constructed in `routes_dispatch.go`, flushed in the
  same handler), removing the shared `s.effCtx.Debug`.
- `TypedEvaluator.trace.Entries`: ring of `DefaultMaxTypedTraceEntries` (10,000) with a dropped counter.

**M3 — input copies (`internal/effects/fs.go`, `internal/builtins/zip.go`, `internal/apiserver/routes_dispatch.go`)**
- `EffContext.FS.MaxBytes int64` (0 = unbounded, per D-C); `readFile`/`readFileE`/`readBytes`
  stat first and return `E_FS_FILE_TOO_LARGE` with the size and the cap. Read via `os.Open` +
  `io.ReadAll(io.LimitReader(f, cap+1))` into a pre-sized buffer, **then check
  `len(data) > cap` and return `E_FS_FILE_TOO_LARGE`** — the stat is an early exit, not the
  guard: pseudo-files report size 0 and a file can grow between stat and read, and returning
  a silently truncated string would be the "no silent fallbacks" violation the round-2 quorum
  named. Same post-read check as `_zip_readEntry` already does at `zip.go:576`. `string()`
  conversion kept (Go strings are immutable; the alternative is `unsafe`, rejected).
- `_zip_readEntry`: `bytes.Buffer` grown to `UncompressedSize64` before `io.Copy`.
- Multipart per D-F.

**M4 — process controls (`cmd/ailang/memory_limit.go`, `main_run_exec.go`, `serve_api.go`)**
- `resolveMemoryLimit()`: `--max-memory <size>` as today; new literal `--max-memory cgroup`
  (or `AILANG_MEMLIMIT=cgroup`, registered in `internal/config`) reads Linux cgroup v2
  `/sys/fs/cgroup/memory.max` (v1 `memory.limit_in_bytes`) × 0.9; `max`/unreadable → no limit,
  logged. An explicit `GOMEMLIMIT` is left to Go and only logged. Nothing is applied unless asked
  (D-D, opt-in). Applied in both `run` and `serve-api`.
- **What the limit is and is not.** `SetMemoryLimit` is best-effort GC tuning: it makes the
  collector work harder as total runtime memory approaches the limit and can be exceeded by
  reachable data (V17: 772 MB live under a 64 MB limit). It is **not** a hard bound, not a
  request-cancellation mechanism and not an isolation boundary; FS/upload caps bound only their
  own inputs, not decompressed data, accumulators or concurrent requests. Structured failure for
  runtime memory exhaustion (`MEM001`) remains M-MEM-BUDGET-RUNTIME's scope.
- **GOGC and the limit interact as documented by Go, not as "the limit wins"** (V17): the GC
  trigger is the *smaller* of the GOGC-derived heap target and the limit-derived target, and the
  limit is soft. Below the limit GOGC=500 still lets the heap grow 6× live before a cycle; near
  the limit the runtime collects continuously and Go caps GC at ~50 % of CPU rather than
  guaranteeing the limit (measured: `--max-memory 64MB` on 772 MB live → 11 s CPU, no reduction).
  Consequences adopted: (a) GOGC=500 stays for the short-lived `run`/`exec` CLI path, where the
  25 % speed-up was measured; (b) `serve-api` keeps Go's default GOGC=100 — it already does (V10)
  — because a long-lived process at concurrency 80 that sits at min(6× live, limit) has no
  headroom for non-heap memory (goroutine stacks, the SQLite cache, cgo); (c) the cgroup fraction
  is 0.9 so that a true overrun reaches the kernel OOM killer after bounded GC pressure rather
  than thrashing indefinitely at the soft limit — the kernel still kills the instance in that
  case; this milestone reduces how often that happens, it does not make it impossible.
- `ailang doctor` prints the resolved limit and its source.
- Docs: `docs/docs/guides/debugging.md` gains a "Memory" section listing every control and the
  `/usr/bin/time -l` probe protocol.

**Downstream asks (sent as `ailang messages send` to the ailang-parse inbox, not implemented here)**
- D1: `markdown_writer.ail:46,273` `foldl` string accumulators → `join("", map(...))` (`std/string.join` verified V4). Removes the O(n²) and the deep-trace multiplier at once.
- D2: `xlsx_parser.ail:470-484` — parse each sheet inside a function the loop **calls** (`foldlE`/`mapE`), not a tail-recursive rewrite (V23: AILANG recursion retains every frame's bindings until it unwinds, so tail form frees nothing); route worksheets through `scanFoldStep`; drop the `sanitizeXml` copy.
- D3: images to temp files with a path in the block; base64 on demand; per-document image-bytes cap.
- Both Dockerfiles: add `--max-memory` (or rely on M4's cgroup detection once released).

### Implementation Plan

**M1: Bounded rendering + honest retention** (~1.5 days)
- [ ] `eval.ShowBounded` + `boundedWriter`; container `String()` methods delegate to a shared writer path
- [ ] `Collector.ValueBudget()`; three render sites switched; `RecordsFunctionCalls` gate kept
- [ ] `eventSize` honest; `DefaultMaxRetainedBytes` per D-B; update `TestRetentionCapKeepsTheTail` expectations
- [ ] Record-update capacity hint
- [ ] Fixtures: depth-9,000 cons probe under `deep` ≤ 1.25× untraced; effect-heavy probe under `standard --emit-trace` ≤ 1.05×
- [ ] Mutation test: revert the `ops.go` change and watch the effect-heavy fixture fail

**M2: Unbounded accumulators** (~1 day)
- [ ] `DebugContext` sink + arrival-time level filter; CLI and serve-api attach sinks; hosts unchanged
- [ ] Per-request `DebugContext` in serve-api; concurrency test with two overlapping requests asserting no cross-talk
- [ ] Typed-evaluator trace ring
- [ ] Fixture: 1,000,000 `Debug.log` lines at `--log-level ERROR`, RSS flat

**M3: Input copies** (~1 day)
- [ ] `FS.MaxBytes` + `--fs-max-bytes` + serve-api wiring per D-C; `E_FS_FILE_TOO_LARGE`
- [ ] Zip pre-size
- [ ] Multipart per D-F; fixture: 50 MB upload, one in-memory copy (assert via `runtime.MemStats` delta in the handler test)

**M4: Process controls** (~0.5 day)
- [ ] `resolveMemoryLimit()` with cgroup detection per D-D; `doctor` line; guide section
- [ ] Downstream messages D1–D3 sent with the measured numbers

### Files to Modify/Create

**New files:**
- `internal/eval/show_bounded.go` — bounded writer + `ShowBounded`, ~120 LOC
- `internal/eval/show_bounded_test.go` — budget exhaustion on list/record/map/ADT, ~120 LOC
- `internal/effects/fs_limits.go` — `FS.MaxBytes`, stat-first read, typed error, ~80 LOC
- `cmd/ailang/memory_limit_cgroup.go` — cgroup v1/v2 detection (Linux build tag) + stub, ~90 LOC
- `internal/trace/testdata/memprobe_cons.ail`, `memprobe_effects.ail`, `memprobe_debuglog.ail` — fixtures, ~30 LOC each
- `internal/trace/memprobe_test.go` — runs the fixtures via the built binary with an RSS ratio assertion (skipped on `-short`), ~150 LOC

**Modified files:**
- `internal/effects/ops.go` — render via `ShowBounded(a, ctx.Trace.ValueBudget())`, ~15 LOC
- `internal/eval/eval_operations.go` — same at both function sites, ~10 LOC
- `internal/eval/value.go` — container `String()` methods route through the writer, ~60 LOC
- `internal/eval/eval_expressions.go` — record-update capacity hint, ~2 LOC
- `internal/trace/retention.go` — honest `eventSize`, new default, ~25 LOC
- `internal/trace/collector.go` — `ValueBudget()`, ~10 LOC
- `internal/effects/debug.go`, `debug_sink.go` — sink + arrival filter + single-parse severity, ~60 LOC
- `internal/apiserver/server.go`, `routes_dispatch.go` — per-request debug ctx, multipart rewrite, ~70 LOC
- `internal/eval/eval_typed_helpers.go`, `eval_typed.go` — trace ring, ~25 LOC
- `internal/builtins/zip.go` — pre-sized read, ~10 LOC
- `internal/effects/context.go`, `internal/runner/run.go`, `cmd/ailang/main_run.go`, `cmd/ailang/serve_api.go` — FS cap plumbing + memory limit resolution, ~60 LOC
- `internal/config/*.go` — `AILANG_FS_MAX_BYTES`, `AILANG_NO_CGROUP_MEMLIMIT` in the Registry (forbidigo requires it), ~15 LOC
- `docs/docs/guides/debugging.md` — Memory section, ~60 lines
- `changelogs/v0.32-current.md` — entry

## Conflict Surface

The change touches `internal/eval`, `internal/effects` and `internal/trace`, so this section is mandatory.

1. **Positions extended.** Value rendering (`Value.String()`), the trace render sites, `DebugContext.Log`, FS read effects, the serve-api multipart handler, and process start-up in `run`/`serve-api`.
2. **What else lives there.**
   - `String()` is also used by `show`, `println`, error messages, `Debug.log` interpolation and the REPL. **Rule: `String()` keeps its unbounded semantics; only trace sites call `ShowBounded`.** A bounded `show` would change program output.
   - `internal/trace/scorer.go` and `comparator.go` read effect args/results from retained events; they already see 1 KB-truncated values (`boundValues`), so a bounded render that produces the same prefix is invisible to them. Acceptance test: `TestValueTruncationKeepsEveryEvent` and `TestTierGovernsWhatIsRecorded` unchanged and green (V6).
   - `Secret` results are replaced by `redactedSecretMarker` *after* render (`ops.go`); the bounded render must run before, and the marker after, exactly as today — the `string<secret>` canary from M-TRACE-TIER-NOT-ENFORCED (0 verbatim copies at `standard`) is the regression test.
   - `DebugContext.Collect()` is the embedding-host API (`internal/gen/golang/debug.go` mirrors it in generated Go). D-E keeps it byte-for-byte when no sink is attached.
   - `ParseMultipartForm`'s `maxMemory` also governs *form fields*, not just files; the 4 MB threshold must still admit the `@raw`/JSON limits at `routes_dispatch.go:43,83`.
   - `GOGC` handling: `config.GOGCSet()` already defers to the operator; the cgroup limit must likewise defer to an explicit `GOMEMLIMIT` (Go applies it natively; setting it again would override the operator).
3. **Disambiguation.** None syntactic; all changes are behind existing gates (`Trace.Enabled()`, sink presence, flag/env presence).
4. **Programs that MUST still work** (fixtures verified to exist, V6/V15): `internal/trace/retention_test.go` (5 tests), `internal/trace/tier_enforcement_test.go`, `internal/builtins/zip_memory_test.go`, `internal/effects` debug tests, the serve-api multipart tests in `internal/apiserver`, plus `make verify-examples`.
5. **Deliberate changes.** Retention default (D-B); `Debug.log` timing in CLI/serve-api (D-E); a typed error instead of success for FS reads above a configured cap (D-C, only when set).

## Examples

### Example 1: deep tracing a list-building recursion

**Before** (measured): depth 9,000, `AILANG_TRACE=deep --emit-trace jsonl` → 2,282 MB peak; every call renders the full accumulator, then truncates it to 1 KB.

**After**: the render stops at 1 KB; peak within 1.25× of the 772 MB untraced run. The retained events are byte-identical.

### Example 2: reading a large file

**Before:**
```
readFile("big.xml")            -- 45 MB: two full copies, no cap, no error
```

**After** (serve-api sets the cap to its upload limit; CLI unchanged unless `--fs-max-bytes` is given):
```
readFile("big.xml")
-- Err(E_FS_FILE_TOO_LARGE: big.xml is 47,185,920 bytes, cap 10,485,760)
```

### Example 3: a container

**Before:** `docker run -m 1g ailang serve-api …` — Go sizes its heap to the host, GOGC=500 lets it grow 6×, the cgroup OOM-kills the instance at ~72 s (measured in docparse).

**After** (with `--max-memory cgroup` in the Dockerfile): start-up logs `memory limit: 966 MB (cgroup memory.max × 0.9)`; the runtime collects hard as total memory nears the limit instead of growing 6× live first, so garbage-heavy requests that OOM'd today complete. A request whose *live* data exceeds the limit still dies by cgroup OOM — M1–M3 lower that live data (no full renders, no triple upload copies, capped reads); a typed per-request failure is M-MEM-BUDGET-RUNTIME.

## Success Criteria

- [ ] Deep-trace cons probe ≤ 1.25× untraced peak (fixture in `internal/trace/memprobe_test.go`)
- [ ] Standard-tier effect-heavy probe with `--emit-trace` ≤ 1.05× untraced
- [ ] Retained trace bytes measured with `runtime.MemStats` within 1.5× of `eventSize`'s total (today ≥ 4×)
- [ ] 1,000,000 `Debug.log` lines at `--log-level ERROR`: RSS flat within 10 MB of the empty-program floor
- [ ] Two concurrent serve-api requests with `Debug.log` never see each other's lines
- [ ] 50 MB multipart upload: handler `HeapAlloc` delta < 60 MB (one copy) — today three copies
- [ ] `ailang run --max-memory cgroup` under a cgroup limit applies `SetMemoryLimit`; without the flag nothing is applied; `ailang doctor` reports the resolved limit and its source
- [ ] `string<secret>` canary still 0 verbatim copies at `standard`
- [ ] Mutation tests: reverting `ops.go` fails the effect fixture; reverting `eventSize` fails the retention ratio test
- [ ] `make test-core`, `make ci`, `make simplicity-audit` green (two new env vars registered in `internal/config`)
- [ ] Documentation updated (`debugging.md` Memory section; changelog)
- [x] Downstream asks D1–D3 sent 2026-09-16 to `pkg:sunholo/ailang_parse` (canonical store): D1 `inbox_1789590140354_20306541`, D2 `inbox_1789590142122_e881c3aa`, D3 `inbox_1789590143755_f84d120c`

## Testing Strategy

**Unit tests:** `ShowBounded` on every container type at budgets 0, 1, 1 KB, unlimited; `eventSize` vs `unsafe.Sizeof`; `DebugContext` with and without a sink; cgroup file parsing (`max`, numeric, missing).

**Integration tests:** RSS-ratio fixtures run against the built binary, skipped under `-short`, using `/usr/bin/time -l` on macOS and `/usr/bin/time -v` on Linux (`internal/eval_harness/memlimit.go` already parses both).

**Manual testing:** `bin/docparse` on `xlsx_usda_food_atlas.xlsx` with `AILANG_NO_TRACE=0` before and after M1, peak RSS recorded in the implementation report.

## Deferred Decisions

- Exact `boundedWriter` API (io.Writer wrapper vs. visitor) — agent may choose.
- Whether `ShowBounded` reports `truncated` via a suffix (`…`) or a flag on the event — agent may choose; the retained-event prefix must be unchanged.
- Typed-trace ring size (10,000 proposed) — agent may choose within 1,000–100,000.
- cgroup fraction (0.9 proposed) — agent may choose within 0.8–0.95.

## Non-Goals

- **Cons cells / tail-call optimisation / arena-backed lists** (F1) — M-LIST-CONS-QUADRATIC and M-LIST-CONS-CELLS-DECOMPOSITION, parked pending human review. This doc's fixes are orthogonal and do not change value representation.
- **A string-builder value type** (F13) — language-surface change; Future Work.
- **Closure free-variable capture and per-call mutex removal** (F16) — needs a measurement first; Future Work.
- **Lazy evaluation or generators** for `take(n, flatMap(...))` (F15, #617) — language semantics; the fused bounded combinators and prompt guidance stand.
- **Dropping the surface AST after elaboration** (F18) — intentional retention.
- **A `MEM001` logical memory budget** — M-MEM-BUDGET-RUNTIME owns it; this doc lowers usage, that doc bounds it.
- **Changing GOGC=500 for the CLI** — kept for `run`/`exec` (short-lived; 25 % measured speed-up, ~24 MB at the floor). It is *not* harmless under a limit (V17); that is why `serve-api` does not set it.

## Timeline

**Days 1–2:** M1 (bounded renderer, honest retention, fixtures, mutation test)
**Day 3:** M2 (Debug.log sink, per-request context, typed-trace ring)
**Day 4:** M3 (FS cap, zip pre-size, multipart)
**Day 5 (half):** M4 (cgroup limit, doctor, docs), downstream messages, changelog

**Total: ~4.5 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| A bounded renderer changes the retained prefix of some value type (e.g. map key ordering) | Med — scorer/comparator diffs | Golden test: `ShowBounded(v, 1024)` == `truncateValue(v.String(), 1024)` for every fixture value |
| Streaming `Debug.log` reorders CLI output relative to stdout | Low | Both go to stderr/stdout as today; the change is *when*, gated by D-E |
| cgroup detection misreads a nested/limitless container | Med — GC thrash or no limit | `max` → no limit; unreadable → no limit; always log the source; opt-out env |
| Retention default change surprises a deep-trace consumer | Low | D-B freeze item; exporters unaffected by design (observers notified before retention) |
| FS cap default set too low by serve-api | Med — false `FILE_TOO_LARGE` | Cap = upload cap; temp files written by the server are by construction under it |

## Verification Log

| # | Claim | How verified | Result |
|---|---|---|---|
| V1 | No runtime memory bound / `MEM001` exists today | `grep -rn "MEM001\|max-mem\b\|MaxMem\b" internal cmd` | empty — M-MEM-BUDGET-RUNTIME is unimplemented |
| V2 | `RecordsEffectValues`/`RecordsValues`/`ValueBudget` do not exist (proposed name is free) | grep `internal/` | empty |
| V3 | FS has no byte cap; Net does | `grep MaxBytes internal/effects/context.go internal/effects/fs.go` | `Net.MaxBytes` at context.go:195,231 (5 MB); nothing in fs.go |
| V4 | `std/string.join(delimiter, xs)` and `concat(xs)` exist for D1 | `grep "^export pure func join\|concat" std/string.ail` | string.ail:106, :112 |
| V5 | Function-call render is tier-gated; effect render is not | read `eval_operations.go:118-130` and `ops.go:112-127` | confirmed |
| V6 | Cited retention tests exist | `grep "func Test" internal/trace/retention_test.go` | 5 tests incl. `TestValueTruncationKeepsEveryEvent`, `TestRetentionCapKeepsTheTail` |
| V7 | Harness RSS watchdog covers harness only | `internal/eval_harness/memlimit.go:106 waitWithGuards`; no caller in runner/apiserver | confirmed |
| V8 | `Debug.log` is collect-then-flush | `debug.go:78-85`, `runner/handlers.go:239`, `apiserver/server.go:317-322` | confirmed |
| V9 | `--max-memory` only applied when the flag is passed; neither Dockerfile passes it | `main_run.go:197`, `serve_api.go:55`; `grep max-memory` both Dockerfiles | confirmed, empty |
| V10 | GOGC=500 default in `run`/`exec`, not in `serve-api` | `main_run_exec.go:33-35`; grep serve_api.go | confirmed |
| V11 | Multipart uses upload cap as in-memory threshold; `ReadAll` of the part | `routes_dispatch.go:55-59,563-575` | confirmed |
| V12 | `eventSize` counts 128 B + string bytes only | `retention.go:128-146` | confirmed |
| V13 | Neither parse repo calls `Debug.log` | `grep -rl "Debug.log\|std/debug" --include=*.ail ailang-parse docparse` | empty (positive control: the same grep finds `std/debug.ail`, 4 lines) |
| V14 | Measurements | probes `stack.ail`, `acc.ail` (`build(n, acc) = if n == 0 then acc else build(n-1, n :: acc)`), `/usr/bin/time -l`, installed `ailang v0.39.2-dirty` | table above |
| V15 | Trace collector only constructed with `--emit-trace` or telemetry enabled | `runner/run.go:490-495` | confirmed — CLI runs without either pay nothing |
| V16 | M-TRACE-TIER-NOT-ENFORCED M2 landed despite the doc header saying outstanding | `git show e4c42a7bf` | landed; doc status stale |
| V17 | GOGC vs memory limit: trigger = min(GOGC target, limit target); limit is soft | Go docs `runtime/debug.SetMemoryLimit` ("GOGC still determines the trigger point when below the limit"; "soft limit … the runtime will not exceed ~50 % GC CPU"); measured `GODEBUG=gctrace=1` on a garbage-heavy probe (200k-string list mapped 40×): GOGC=100 → 7 cycles; GOGC=500 → 1 cycle; GOGC=500 + `GOMEMLIMIT=120MiB` → 7 cycles (total runtime memory ≈ 107 MB RSS approached the limit, so the limit-derived trigger took over); `--max-memory 64MB` on 772 MB live → 931 MB peak, 11.2 s user CPU | confirmed; M4 text rewritten accordingly |
| V18 | F4: `ListValue.String()` serialises every element into one builder; same shape for record/map/ADT | read `internal/eval/value.go:89-100` (`for _, elem := range l.Elements { b.WriteString(elem.String()) }`) | confirmed; no depth/size bound |
| V19 | F7: typed-evaluator trace has no cap | read `internal/eval/eval_typed_helpers.go:90` (`e.trace.Entries = append(e.trace.Entries, entry)`); `eval_typed.go:22,50,74` (`Entries []TraceEntry`) | confirmed; unbounded append, no eviction |
| V20 | F10: `_zip_readEntry` reads with `io.ReadAll` (doubling growth, no pre-size) then `string(data)` | read `internal/builtins/zip.go:559-579` (`io.ReadAll(limited)` at :573) and `:199-200` (`string(data)`) | confirmed; `f.UncompressedSize64` is checked at :561 but never used to size the buffer |
| V21 | F16: closures capture the whole `Environment`; each child env allocates a map and carries a `sync.RWMutex` | read `internal/eval/eval_expressions.go:178-185` (`Env: env`), `env.go:10-14` (struct: `mu sync.RWMutex; values map[string]Value; parent *Environment`), `env.go:24-30` (`NewChildEnvironment`: `make(map[string]Value)`, no hint) | confirmed; no free-variable analysis exists (`grep -rn "freeVars\|FreeVars" internal/eval` → empty) |
| V22 | F17: `MapValue.Insert` copies the whole map but pre-sizes; record update copies without a hint | read `internal/eval/value.go:213-221` (`make(map[string]*MapEntry, len(m.Entries)+1)`), `eval_expressions.go:411` (`make(map[string]Value)`) | confirmed; F17 wording corrected (Insert does hint) |
| V23 | **Found during M1 (2026-09-16):** a recursive loop keeps every frame's `let`/`match` bindings live until the whole recursion unwinds (no TCO), so per-item data accumulates even in "tail" form | 40 × `readFileResult` of a 20 MB file: hand-written recursive loop 870 MB peak (`--max-memory 64MB` no effect: live); the same via `foldlE(step, …)` with the read inside `step` 212 MB | confirmed; corrects downstream ask D2 (tail-recursion would not help) and is written into `debugging.md#memory` |

**Quorum triggers:** trigger 1 fires (four design-freeze items) and trigger 2 fires (D-B overrides
the shared trace retention default). Both rounds run; see the log below.

## Quorum log

Artifacts: `.ailang/state/mission-quorum/m-v1-memory-footprint-2026-09-16T19-03-39Z.json` (round 1) and the
`m-v1-memory-footprint-2026-09-16T19-06-27Z.json` (round 2). Controller (this session) passed both rounds; every reviewer
objection was accepted and applied — none was argued.

| Round | Reviewer | Verdict | Objection (abridged) | Applied as |
|---|---|---|---|---|
| 1 | gpt6-astra | absent (budget cap $0.10 < est. $0.14) | — | round 2 run at $0.30 |
| 1 | gemini-3-1-pro | reject | F4, F7, F10, F16, F17 cited with line numbers but no Verification Log rows | V18–V22 |
| 1 | oc-glm-5-2 | reject | "the runtime collects at the limit regardless of GOGC" is wrong: trigger = min(GOGC target, limit target); limit is soft; unverified | V17 (Go docs + gctrace), M4 rewritten, non-goal reworded |
| 2 | gpt6-astra | reject | M4(c) and Example 3 promised request-level structured failure that a soft limit cannot provide | M4 "what the limit is and is not"; Example 3 rewritten; `MEM001` stays with M-MEM-BUDGET-RUNTIME |
| 2 | gemini-3-1-pro | reject | FS cap had no post-read check → silent truncation on pseudo-files or growing files | M3 mandates `len(data) > cap` → `E_FS_FILE_TOO_LARGE` |
| 2 | oc-glm-5-2 | reject | D-D opt-out contradicts A4 "+1 caller-set" | D-D recast as opt-in `--max-memory cgroup`; A4 rescored 0; net +5 |

The re-quorum-once guardrail is spent. Per the design-doc-creator skill this parks the doc for a human
ratification of D-B..D-E plus the three round-2 fixes rather than grinding a third round.

## Related Documents

- [design_docs/planned/v0_31_0/m-mem-budget-runtime.md](../v0_31_0/m-mem-budget-runtime.md) (0.42) — a logical memory *budget* with a typed `MEM001`; unimplemented (V1). Distinct: that doc bounds usage, this one reduces it. M4's cgroup limit is the physical backstop that doc also describes.
- [design_docs/planned/v0_36_0/m-trace-tier-not-enforced.md](../v0_36_0/m-trace-tier-not-enforced.md) (0.40) — M1+M3 shipped, M2 shipped as e4c42a7bf; this doc is its "honest limit" follow-up (peak, not retention).
- [design_docs/planned/m-list-cons-quadratic.md](../m-list-cons-quadratic.md) — F1, parked; the structural fix.
- [design_docs/planned/m-list-repr-spike.md](../m-list-repr-spike.md) (0.37) — list representation spike feeding the above.
- [design_docs/implemented/v0_9_2/m-docparse-dx.md](../../implemented/v0_9_2/m-docparse-dx.md) — the original 4.8 GB XML-tree finding and the streaming `parseElements` fix.
- `docparse/design_docs/planned/xlsx_resource_tiers.md` (sibling repo) — the container-side measurements and the still-open parser fixes D2/D3.
- Issues: [#676](https://github.com/sunholo-data/ailang/issues/676) (quadratic cons), [#617](https://github.com/sunholo-data/ailang/issues/617) (strict `take`/`flatMap`).

## References

- [Design Axioms](/docs/references/axioms)
- `docs/docs/guides/debugging.md` — flags table (Memory section to be added by M4)

## Future Work

- A `StringBuilder`/rope value or compiler-recognised `foldl`-concat rewrite (F13).
- Closure free-variable capture and removal of the per-call `sync.RWMutex` (F16), after a probe shows the win.
- Persistent map for `MapValue`/records (F17).
- Reducing Go stack per AILANG frame (fewer `defer`s per call; measured ≈ 6–7 KB/frame) — or TCO, which M-LIST-CONS-QUADRATIC's successor may bring.
- Extend the RSS watchdog pattern to `serve-api` per request.

## Implementation Report (2026-09-16)

All four milestones landed on `dev` the same day, attended, in four commits
(`ca61a685d` M1, `2491ab45e` M2, `8c079b659` M3, `1eca5d4bd` M4). Every measurement below is
`/usr/bin/time -l` peak RSS on the rig; every ratio is pinned by a test that runs in `make test`.

| Goal | Before | After | Pinned by |
|---|---|---|---|
| Deep trace of cons recursion (depth 9,000) vs untraced | 2,759 MB vs 769 (3.6×) | 814 MB (1.06×) | `TestMemprobeDeepTraceIsBoundedOnConsRecursion` ≤ 1.25 |
| Standard tier + `--emit-trace`, 40 × `readFileResult` of 20 MB | 1.14× untraced | 1.00× | `TestMemprobeStandardTierEffectResultsAreBounded` ≤ 1.10 (mutation of `ops.go`: 1.39×) |
| 200k `Debug.log` lines at `--log-level error` (GOGC=100) | +64 MB retained | +11 MB transient | `TestMemprobeDebugLogDoesNotAccumulate` ≤ 30 |
| 2M `Debug.log` lines | 1.8 GB | 180 MB | (manual) |
| 48 MB multipart upload to a string param | 126 MB allocated | 16 MB, flat | `TestMultipartStringParamStreamsToTempFile` (mutation: 126 MB) |
| serve-api concurrent `Debug.log` | shared buffer | per-request | `TestServeAPI_ConcurrentRequestsKeepTheirOwnDebugLines` |
| `--max-memory cgroup` | — | opt-in, ×0.9, none on `max`/missing | `TestResolveMemoryLimitPrecedence` + cgroup file tests |

**Deviations from the plan.** (1) The memprobe tests gate on `testutil.SkipInFastLoop`, not
`testing.Short` — gatelint R1 forbids the latter as inert in CI. (2) `Severity` is now a single
`json.Decoder` pass; the sink resolves `os.Stderr` at write time so a host that swaps stderr
after start-up is honoured. (3) `applyMemoryLimit` was deleted (superseded by
`applyResolvedMemoryLimit`; its only test covered the parser, which stays). (4) The typed-trace
ring evicts half at the cap (amortised O(1)) rather than one per append.

**Found on the way (V23).** AILANG recursion keeps every frame's bindings live until the whole
recursion unwinds, so a "tail-recursive" per-item loop does not free per-item data. The fix is
structural in the program — do the per-item work in a function the loop calls (`foldlE`,
`mapE`) — and it corrected downstream ask D2. It is now in `debugging.md#memory`.

**Not done here, by design.** F1 (cons cells / TCO) stays with M-LIST-CONS-QUADRATIC; F13 (a
string builder value) and F16 (closure capture, per-call mutex) are Future Work; `MEM001` is
M-MEM-BUDGET-RUNTIME. The `simplicity-audit` `tracked_files` gate is red today (+141) from the
day's releases by other sessions; this sprint's share is 15 files, all tests, fixtures and two
source files.

---

**Document created**: 2026-09-16
**Last updated**: 2026-09-16
