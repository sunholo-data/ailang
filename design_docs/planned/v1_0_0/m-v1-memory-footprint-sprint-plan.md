# Sprint Plan: M-V1-MEMORY-FOOTPRINT

**Design doc**: [m-v1-memory-footprint.md](m-v1-memory-footprint.md) (ratified 2026-09-16)
**Sprint ID**: M-V1-MEMORY-FOOTPRINT
**Duration**: 4 milestones, ~4.5 days planned; executed attended in one session
**Risk**: medium — M1 touches every container's rendering path; M2 changes when `Debug.log` lines appear
**Branch**: `dev` (attended; commit per milestone)

## Summary

Reduce the runtime's memory overhead on top of the program's own live data: bounded trace
rendering with honest retention accounting (M1), no unbounded log/trace accumulators (M2),
one in-memory copy of inputs instead of three (M3), and an opt-in cgroup-derived memory limit
with documentation (M4). The structural list fix stays with M-LIST-CONS-QUADRATIC.

## Velocity

Last 7 days on dev: two design-doc sprints landed at roughly 400–600 LOC/day of Go including
tests. This sprint is ~1,300 LOC implementation + tests, all in files already read during the
audit, so the 4.5-day estimate carries a 30 % buffer.

## Milestones

### M1 — Bounded rendering + honest retention (~500 LOC, 1.5 days)

**Mechanism.** A new `eval.ShowBounded(v Value, maxBytes int) string` renders into a
`boundedWriter` that stops writing once the budget is spent and then *counts* the remainder
(no allocation for containers; scalars format transiently) so the output is byte-identical to
`truncateValue(v.String(), maxBytes)`: `prefix[:max] + "…(+N bytes elided)"`. Budget ≤ 0 means
unbounded (delegates to `String()`). `eval.RenderedLen(v)` is the counting pass alone, used for
the redacted descriptor. Container `String()` methods keep their unbounded semantics — `show`,
`println` and error messages are untouched.

**Collector API.** `Collector.ValueBudget() (maxBytes int, redacted bool)`. `truncateValue` and
`redactValue` become idempotent by a position-anchored suffix check (`s[max:]` starts with
`…(+` and ends with ` bytes elided)`; `<redacted:N bytes>`), so a pre-bounded string passing
through `record()` is not truncated twice.

**Sites.** `internal/effects/ops.go:112-127` (effect args + result), `eval_operations.go:124-131`
and `:218-226` (function enter/exit, still tier-gated), `eval_typed_helpers.go:boundedShow`
(delegates to `ShowBounded`). `EffContext` gains `RenderTraceValue(v) string` on the
`TraceRecorder` interface so the eval sites do not import trace.

**Retention.** `eventSize` charges `unsafe.Sizeof(TraceEvent{})` + payload struct size + 16 B per
string header + string bytes + slice header. `DefaultMaxRetainedBytes = 32 << 20`.

**Record update.** `make(map[string]Value, len(base)+len(updates))` at `eval_expressions.go:411`.

**Tests.** `show_bounded_test.go`: golden equality against `truncateValue(v.String(), n)` for
list/array/map/tuple/record/ADT/nested/bytes/indirect at budgets 0, 1, 16, 1024, unlimited; a
1M-element list renders at budget 64 with `testing.AllocsPerRun` bounded. `retention_test.go`:
idempotence of both markers; `eventSize` ≥ `unsafe.Sizeof` floor. `memprobe_test.go` (skipped
under `-short`): builds the binary, runs `testdata/memprobe_cons.ail` untraced and with
`AILANG_TRACE=deep --emit-trace jsonl`, asserts peak RSS ratio ≤ 1.25 via `/usr/bin/time`.
Mutation: revert the `ops.go` site and assert the effect fixture's ratio test fails.

**Acceptance.** Deep-trace cons probe ≤ 1.25× untraced; standard-tier effect probe ≤ 1.05×;
retained bytes within 1.5× of `eventSize` sum; `string<secret>` canary still 0 verbatim copies;
`TestValueTruncationKeepsEveryEvent`, `TestRetentionCapKeepsTheTail`, `TestTierGovernsWhatIsRecorded` green.

### M2 — Unbounded accumulators (~300 LOC, 1 day) ✅ 2026-09-16

- `DebugContext` gains `sink func(LogEntry)` and `minLevel int` via `SetSink(sink, minLevel)`.
  `Log` filters structured lines below `minLevel` on arrival; with a sink set it writes through
  and does **not** append. `Check` unchanged (assertions are few). `Collect`/`Reset` unchanged
  when no sink is set (embedding hosts, generated Go in `internal/gen/golang/debug.go` untouched).
- `DebugSink` gains `LineWriter() func(LogEntry)` so `FlushDebugOutput` (runner) and
  `flushDebugOutput` (apiserver) attach it before execution; `Flush` still drains assertions.
  Severity parsed once with a `json.Decoder` over a `strings.Reader`.
- serve-api: `routes_dispatch.go` builds a per-request `DebugContext` on the forked effect
  context (verify `engine.Call` → `runtime/entrypoint.go:96 Fork()` clones `EffContext`; if the
  clone shares `Debug`, give the clone its own). Concurrency test: two overlapping requests with
  `Debug.log`, no cross-talk.
- `TypedEvaluator.TraceCollector`: ring of `DefaultMaxTypedTraceEntries = 10000` with a `Dropped` counter.
- Fixture `memprobe_debuglog.ail`: 1,000,000 `Debug.log` lines at `--log-level ERROR`; RSS within
  10 MB of the empty-program floor.

### M3 — Input copies (~250 LOC, 1 day)

- `EffEnv.FSMaxBytes int64` (0 = unbounded). New `readCapped(path, cap)` in `fs_limits.go`:
  stat early-exit, `os.Open` + `io.ReadAll(io.LimitReader(f, cap+1))` into a buffer pre-sized to
  `min(stat.Size, cap)+1`, **post-read `len > cap` → `E_FS_FILE_TOO_LARGE`** (typed `Err` on the
  Result-returning variants, Go error on `readFile`). Used at `fs.go:142`, `:252`, `:322`.
- `--fs-max-bytes <size>` on `ailang run` (registry var `AILANG_FS_MAX_BYTES`), serve-api sets
  `FSMaxBytes = maxUploadSize`.
- `_zip_readEntry`: `bytes.Buffer` grown to `UncompressedSize64` before `io.Copy`.
- Multipart: `r.Body = http.MaxBytesReader(w, r.Body, maxSize)`; `ParseMultipartForm(4 << 20)`;
  `readMultipartFile` returns an `io.ReadCloser` and the temp-file path is written with `io.Copy`;
  the `[]byte` path (bytes-typed params) keeps `ReadAll` (that copy IS the value). Handler test:
  50 MB upload, `HeapAlloc` delta < 60 MB.

### M4 — Process controls + downstream (~150 LOC, 0.5 day)

- `--max-memory cgroup` literal and `AILANG_MEMLIMIT=cgroup` (registry): Linux cgroup v2
  `memory.max` (v1 `memory.limit_in_bytes`) × 0.9; `max`/unreadable → no limit, logged. Applied
  in `run` and `serve-api`. Non-Linux build: the literal resolves to "no limit" with a log line.
- `ailang doctor` prints the resolved limit and source.
- `docs/docs/guides/debugging.md`: Memory section (controls table + probe protocol).
- Changelog entry. Design doc → implemented report. Downstream asks D1–D3 sent to ailang-parse
  with message IDs recorded in the design doc.

## Files

See the design doc's "Files to Modify/Create"; M1–M4 above name the exact sites.

## Success Metrics

All acceptance boxes in the design doc's Success Criteria; `make test-core`, `make ci`,
`make simplicity-audit` green; two new env vars registered in `internal/config`.
